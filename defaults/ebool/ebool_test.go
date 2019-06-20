package ebool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func zeroNodesPos(nodes []node) []node {
	for i, node := range nodes {
		node.pos = 0
		nodes[i] = node
	}
	return nodes
}

func TestParseOK(t *testing.T) {
	for _, ds := range []struct {
		input string
		nodes []node
	}{
		{
			input: "N+2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "+"},
			},
		},
		{
			input: "N-2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "-"},
			},
		},
		{
			input: "N*2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "*"},
			},
		},
		{
			input: "N/2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "/"},
			},
		},
		{
			input: "N>2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: ">"},
			},
		},
		{
			input: "N<2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "<"},
			},
		},
		{
			input: "N>=2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: ">="},
			},
		},
		{
			input: "N<=2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "<="},
			},
		},
		{
			input: "N==2",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "=="},
			},
		},
		{
			input: "N!=1e3",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 1e3},
				{typ: ntOperator, op: "!="},
			},
		},
		{
			input: "!(N==5)",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 5},
				{typ: ntOperator, op: "=="},
				{typ: ntOperator, op: "!"},
			},
		},
		{
			input: "(N==2)||(N!=3)",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "=="},
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 3},
				{typ: ntOperator, op: "!="},
				{typ: ntOperator, op: "||"},
			},
		},
		{
			input: "!((N!=2)&&(N>=3))",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "!="},
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 3},
				{typ: ntOperator, op: ">="},
				{typ: ntOperator, op: "&&"},
				{typ: ntOperator, op: "!"},
			},
		},
		{
			input: "N%(2+3)",
			nodes: []node{
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntNumber, num: 3},
				{typ: ntOperator, op: "+"},
				{typ: ntOperator, op: "%"},
			},
		},
		{
			input: "5+N* 2-3",
			nodes: []node{
				{typ: ntNumber, num: 5},
				{typ: ntPlaceHolder},
				{typ: ntNumber, num: 2},
				{typ: ntOperator, op: "*"},
				{typ: ntOperator, op: "+"},
				{typ: ntNumber, num: 3},
				{typ: ntOperator, op: "-"},
			},
		},
	} {
		e, err := New(ds.input)
		assert.Nil(t, err)
		assert.Equal(t, ds.nodes, zeroNodesPos(e.nodes))
	}
}

func TestParseErr(t *testing.T) {
	for _, input := range []string{
		"(N>1))",
		"((N==2)",
		"!!123.345.3",
		"!!",
		">",
		"<",
		"|",
		"&",
		"=",
		"1==N|N<8",
	} {
		_, err := New(input)
		assert.NotNil(t, err, input)
	}
}

func TestCalculating(t *testing.T) {
	for _, ds := range []struct {
		input  string
		arg    float64
		expect bool
		haserr bool
	}{
		{input: "(N*(N-3)>=10)&&(N<7)&&(N>5)", arg: 6, expect: true},
		{input: "(N*(N-3)>=10)&&(N<7)&&(N>5)", arg: 4, expect: false},
		{input: "!(N>3)", arg: 4, expect: false},
		{input: "!(N>3)", arg: 3, expect: true},
		{input: "!(N+N>3)", arg: 3, expect: false},
		{input: "!(N+N!=4)", arg: 2, expect: true},
		{input: "(N%2==0)||(N<5)", arg: 8, expect: true},
		{input: "(N%2==2)||(N<5)", arg: 3, expect: true},
		{input: "(N/100>0.3)&&(N/100<=0.8)", arg: 40, expect: true},
		{input: "(N/100>0.3)&&(N/100<=0.8)", arg: 29, expect: false},
		{input: "!((N*2>20)||(N<=8&&N%2==0))", arg: 6, expect: false},
		{input: "!((N*2>20)||(N<=8&&N%2==0))", arg: 9, expect: true},
		{input: "!((N*2>20)||(N<=&&N%2==0))", arg: 1, haserr: true},
		{input: "!(>1)&&(N%3)!=5||N>=9", arg: 1, haserr: true},
		{input: "!(N>1)&&(N%)!=5||N>=9", arg: 1, haserr: true},
		{input: "!(N>1)&&(N%3)!=||N>=9", arg: 1, haserr: true},
		{input: "!(N>1)&&(N%3)!=5||N>=", arg: 1, haserr: true},
	} {
		e, err := New(ds.input)
		assert.Nil(t, err)
		res, err := e.Eval(ds.arg)
		if ds.haserr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
			assert.Equal(t, ds.expect, res, ds.input)
		}
	}
}
