package defaults

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"unsafe"
)

var (
	ErrDivideByZero = errors.New("goctl/defaults: divide by zero")

	priorities = map[string]uint8{
		"||": 1,
		"&&": 1,
		">":  2,
		"<":  2,
		">=": 2,
		"<=": 2,
		"==": 2,
		"!=": 2,
		"+":  3,
		"-":  3,
		"%":  4,
		"*":  4,
		"/":  4,
	}

	epool = sync.Pool{
		New: func() interface{} {
			return &evaluator{}
		},
	}
)

type ErrInvalidExpression struct {
	raw string
	pos int
}

func (e ErrInvalidExpression) Error() string {
	return fmt.Sprintf("goctl/defaults: '%s': invalid expression around %d", e.raw, e.pos)
}

func isSperator(b byte) bool {
	switch b {
	case ' ', '|', '&', '>', '<', '=', '!', '+', '-', '%', '*', '/', '(', ')':
		return true
	}
	return false
}

func evalBool(pattern string, arg float64) bool {
	e := epool.Get().(*evaluator)
	defer epool.Put(e)
	e.reset(pattern, arg)
	ok, err := e.eval()
	if err != nil {
		panic(err)
	}
	return ok
}

type val struct {
	isbool bool
	num    float64
	b      bool
}

type evaluator struct {
	pos   int
	raw   string
	value float64

	vals []val
	nval int
	ops  []string
	nop  int
	buf  *bytes.Buffer // Every reset for strings.Builder will allocates new memory, so..
}

func (e *evaluator) reset(raw string, value float64) {
	e.pos = 0
	e.raw = raw
	e.value = value
	if e.vals != nil {
		e.vals = e.vals[:0]
	}
	e.nval = 0
	if e.ops != nil {
		e.ops = e.ops[:0]
	}
	e.nop = 0
	if e.buf == nil {
		e.buf = new(bytes.Buffer)
	}
}

func (e *evaluator) checkBoundary(n int) (err error) {
	if e.pos+1 >= n {
		err = ErrInvalidExpression{raw: e.raw, pos: e.pos}
	}
	return
}

func (e *evaluator) eval() (res bool, err error) {
	for n := len(e.raw); e.pos < n; e.pos++ {
		ch := e.raw[e.pos]
		switch ch {
		case ' ':
		case '*', '/', '%', '+', '-', '(', ')':
			e.newOp(string(ch))
		case '!':
			if err = e.checkBoundary(n); err != nil {
				return
			}
			next := e.raw[e.pos+1]
			if next == '=' {
				e.pos++
				e.newOp(b2a([]byte{ch, next}))
			} else if next != '(' {
				err = ErrInvalidExpression{raw: e.raw, pos: e.pos}
				return
			} else {
				e.newOp(string(ch))
			}
		case '>', '<':
			if err = e.checkBoundary(n); err != nil {
				return
			}
			op := []byte{ch}
			if e.raw[e.pos+1] == '=' {
				op = append(op, '=')
				e.pos++
			}
			e.newOp(b2a(op))
		case '|', '&', '=':
			if err = e.checkBoundary(n); err != nil {
				return
			}
			next := e.raw[e.pos+1]
			if next != ch {
				err = ErrInvalidExpression{raw: e.raw, pos: e.pos}
				return
			}
			e.pos++
			e.newOp(b2a([]byte{ch, next}))
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			e.buf.Reset()
			for e.pos < n {
				ch = e.raw[e.pos]
				if isSperator(ch) { // FIXME(damnever): ..
					break
				}
				e.buf.WriteByte(ch)
				e.pos++
			}
			e.pos--
			num, err0 := strconv.ParseFloat(b2a(e.buf.Bytes()), 64)
			if err0 != nil {
				err = ErrInvalidExpression{raw: e.raw, pos: e.pos}
				return
			}
			e.pushVal(val{num: num})
		case 'N':
			e.pushVal(val{num: e.value})
		default:
			err = ErrInvalidExpression{raw: e.raw, pos: e.pos}
			return
		}
	}

	for e.nop > 0 {
		if err = e.calc(); err != nil {
			return
		}
	}
	if e.nval != 1 || !e.vals[0].isbool {
		err = ErrInvalidExpression{raw: e.raw, pos: e.pos}
		return
	}
	res = e.vals[0].b
	return
}

func (e *evaluator) newOp(op string) error {
	if op == ")" {
		for e.nop > 0 && e.ops[e.nop-1] != "(" {
			if err := e.calc(); err != nil {
				return err
			}
		}
		if e.nop == 0 || e.ops[e.nop-1] != "(" {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.nop--
		if e.nop > 0 && e.ops[e.nop-1] == "!" {
			if err := e.calc(); err != nil {
				return err
			}
		}
		return nil
	}
	if op != "(" {
		priority := priorities[op]
		for e.nop > 0 && priorities[e.ops[e.nop-1]] >= priority {
			if err := e.calc(); err != nil {
				return err
			}
		}
	}
	e.ops = append(e.ops[:e.nop], op)
	e.nop++
	return nil
}

func (e *evaluator) pushVal(v val) {
	e.vals = append(e.vals[:e.nval], v)
	e.nval++
}

func (e *evaluator) popTwoNum() (v1, v2 float64, ok bool) {
	if e.nval < 2 {
		return
	}
	v := e.vals[e.nval-1]
	if v.isbool {
		return
	}
	v2 = v.num
	v = e.vals[e.nval-2]
	if v.isbool {
		return
	}
	v1 = v.num
	ok = true
	e.nval -= 2
	return
}

func (e *evaluator) popTwoBool() (v1, v2, ok bool) {
	if e.nval < 2 {
		return
	}
	v := e.vals[e.nval-1]
	if !v.isbool {
		return
	}
	v2 = v.b
	v = e.vals[e.nval-2]
	if !v.isbool {
		return
	}
	v1 = v.b
	ok = true
	e.nval -= 2
	return
}

func (e *evaluator) popBool() (v, ok bool) {
	if e.nval < 1 {
		return
	}
	vv := e.vals[e.nval-1]
	if !vv.isbool {
		return
	}
	v = vv.b
	e.nval -= 1
	ok = true
	return
}

func (e *evaluator) calc() error {
	e.nop--
	op := e.ops[e.nop]
	switch op {
	case "+":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{num: num1 + num2})
	case "-":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{num: num1 - num2})
	case "*":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{num: num1 * num2})
	case "/":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		if num2 == 0 {
			return ErrDivideByZero
		}
		e.pushVal(val{num: num1 / num2})
	case "%":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{num: math.Mod(num1, num2)})
	case "<":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{isbool: true, b: num1 < num2})
	case ">":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{isbool: true, b: num1 > num2})
	case "<=":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{isbool: true, b: num1 <= num2})
	case ">=":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{isbool: true, b: num1 >= num2})
	case "==":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			b1, b2, ok := e.popTwoBool()
			if !ok {
				return ErrInvalidExpression{raw: e.raw, pos: e.pos}
			}
			e.pushVal(val{isbool: true, b: b1 == b2})
		} else {
			e.pushVal(val{isbool: true, b: num1 == num2})
		}
	case "!=":
		num1, num2, ok := e.popTwoNum()
		if !ok {
			b1, b2, ok := e.popTwoBool()
			if !ok {
				return ErrInvalidExpression{raw: e.raw, pos: e.pos}
			}
			e.pushVal(val{isbool: true, b: b1 != b2})
		} else {
			e.pushVal(val{isbool: true, b: num1 != num2})
		}
	case "||":
		num1, num2, ok := e.popTwoBool()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{isbool: true, b: num1 || num2})
	case "&&":
		num1, num2, ok := e.popTwoBool()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{isbool: true, b: num1 && num2})
	case "!":
		v, ok := e.popBool()
		if !ok {
			return ErrInvalidExpression{raw: e.raw, pos: e.pos}
		}
		e.pushVal(val{isbool: true, b: !v})
	default:
		return ErrInvalidExpression{raw: e.raw, pos: e.pos}
	}
	return nil
}

func b2a(p []byte) string {
	return *(*string)(unsafe.Pointer(&p))
}
