package ebool

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unsafe"
)

// TODO(damnever): check all syntax error in parsing stage.

var (
	ErrDivideByZero = errors.New("goctl.ebool: divide by zero")

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
)

type ErrInvalidSyntax struct {
	raw string
	pos int
}

func (e ErrInvalidSyntax) Error() string {
	return fmt.Sprintf("goctl.rpn: parsing '%s': invalid syntax around %d", e.raw, e.pos)
}

type ErrInvalidExpression ErrInvalidSyntax

func (e ErrInvalidExpression) Error() string {
	return fmt.Sprintf("goctl.rpn: '%s': invalid expression around %d", e.raw, e.pos)
}

func isSperator(b byte) bool {
	switch b {
	case ' ', '|', '&', '>', '<', '=', '!', '+', '-', '%', '*', '/', '(', ')':
		return true
	}
	return false
}

type nodeType uint8

const (
	ntPlaceHolder nodeType = iota
	ntOperator
	ntNumber
)

type node struct {
	pos int
	typ nodeType
	op  string
	num float64
}

type EBool struct {
	raw   string
	nodes []node
	stack []interface{}
}

func New(s string) (*EBool, error) {
	nop, n := 0, len(s)
	operators := make([]string, n, n)
	nodes := []node{}

	// FIXME(damnever): dirty code..
	i := 0
	popPushOp := func(op string) {
		priority := priorities[op]
		for nop > 0 && priorities[operators[nop-1]] >= priority {
			nop--
			nodes = append(nodes, node{typ: ntOperator, pos: i, op: operators[nop]})
		}
		operators[nop] = op
		nop++
		i++
	}

	for i < n {
		c := s[i]
		switch c {
		case ' ':
			i++
		case ')':
			for nop > 0 && operators[nop-1] != "(" {
				nop--
				nodes = append(nodes, node{typ: ntOperator, pos: i, op: operators[nop]})
			}
			if nop == 0 || operators[nop-1] != "(" {
				return nil, ErrInvalidSyntax{raw: s, pos: i}
			}
			nop--
			if nop > 0 && operators[nop-1] == "!" {
				nodes = append(nodes, node{typ: ntOperator, pos: i, op: "!"})
				nop--
			}
			i++
		case '(':
			operators[nop] = "("
			nop++
			i++
		case '*', '/', '%', '+', '-':
			popPushOp(string(c))
		case '!':
			if i+1 >= n {
				return nil, ErrInvalidSyntax{raw: s, pos: i}
			}
			next := s[i+1]
			if next == '(' {
				operators[nop] = string(c)
				nop++
				i++
			} else if next == '=' {
				i++
				popPushOp(b2a([]byte{c, next}))
			} else {
				return nil, ErrInvalidSyntax{raw: s, pos: i}
			}
		case '>', '<':
			if i+1 >= n {
				return nil, ErrInvalidSyntax{raw: s, pos: i}
			}
			op := []byte{c}
			if s[i+1] == '=' {
				op = append(op, '=')
				i++
			}
			popPushOp(b2a(op))
		case '|', '&', '=':
			if i+1 >= n {
				return nil, ErrInvalidSyntax{raw: s, pos: i}
			}
			next := s[i+1]
			if next != c {
				return nil, ErrInvalidSyntax{raw: s, pos: i}
			}
			i++
			popPushOp(b2a([]byte{c, next}))
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			numstr := new(strings.Builder)
			for i < n {
				c = s[i]
				if isSperator(c) { // FIXME(damnever): ..
					break
				}
				numstr.WriteByte(c)
				i++
			}
			num, err := strconv.ParseFloat(numstr.String(), 64)
			if err != nil {
				return nil, ErrInvalidSyntax{raw: s, pos: i}
			}
			nodes = append(nodes, node{typ: ntNumber, pos: i, num: num})
		case 'N':
			nodes = append(nodes, node{typ: ntPlaceHolder, pos: i})
			i++
		default:
			return nil, ErrInvalidSyntax{raw: s, pos: i}
		}
	}

	for nop > 0 {
		nop--
		op := operators[nop]
		if op == "(" {
			return nil, ErrInvalidSyntax{raw: s, pos: len(s)}
		}
		nodes = append(nodes, node{typ: ntOperator, pos: i, op: op})
	}

	return &EBool{
		raw:   s,
		nodes: nodes,
		stack: make([]interface{}, len(nodes), len(nodes)),
	}, nil
}

func (e *EBool) Eval(value float64) (bool, error) {
	stack := e.stack
	top := -1
	push := func(v interface{}) {
		top++
		stack[top] = v
	}
	popTwoNum := func() (v1, v2 float64, ok bool) {
		if top < 1 {
			return
		}
		if v2, ok = stack[top].(float64); !ok {
			return
		}
		top--
		v1, ok = stack[top].(float64)
		top--
		return
	}
	popBool := func() (v, ok bool) {
		if top < 0 {
			return
		}
		v, ok = stack[top].(bool)
		top--
		return
	}
	popTwoBool := func() (v1, v2, ok bool) {
		if top < 1 {
			return
		}
		if v2, ok = stack[top].(bool); !ok {
			return
		}
		top--
		v1, ok = stack[top].(bool)
		top--
		return
	}

	for _, node := range e.nodes {
		switch node.typ {
		case ntPlaceHolder:
			push(value)
		case ntOperator:
			switch node.op {
			case "+":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 + num2)
			case "-":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 - num2)
			case "*":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 * num2)
			case "/":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				if num2 == 0 {
					return false, ErrDivideByZero
				}
				push(num1 / num2)
			case "%":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(math.Mod(num1, num2))
			case "<":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 < num2)
			case ">":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 > num2)
			case "<=":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 <= num2)
			case ">=":
				num1, num2, ok := popTwoNum()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 >= num2)
			case "==":
				num1, num2, ok := popTwoNum()
				if !ok {
					b1, b2, ok := popTwoBool()
					if !ok {
						return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
					}
					push(b1 == b2)
				} else {
					push(num1 == num2)
				}
			case "!=":
				num1, num2, ok := popTwoNum()
				if !ok {
					b1, b2, ok := popTwoBool()
					if !ok {
						return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
					}
					push(b1 != b2)
				} else {
					push(num1 != num2)
				}
			case "||":
				num1, num2, ok := popTwoBool()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 || num2)
			case "&&":
				num1, num2, ok := popTwoBool()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(num1 && num2)
			case "!":
				v, ok := popBool()
				if !ok {
					return false, ErrInvalidExpression{raw: e.raw, pos: node.pos}
				}
				push(!v)
			}
		case ntNumber:
			push(node.num)
		default:
		}
	}

	if top != 0 {
		return false, ErrInvalidExpression{raw: e.raw, pos: len(e.raw)}
	}
	if val, ok := stack[top].(bool); ok {
		return val, nil
	}
	return false, ErrInvalidExpression{raw: e.raw, pos: len(e.raw)}
}

func b2a(p []byte) string {
	return *(*string)(unsafe.Pointer(&p))
}
