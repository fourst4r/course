package course

import (
	"io/ioutil"
	"reflect"
	"runtime/debug"
	"testing"
)

func TestLevel(t *testing.T) {
	c := Default()
	eq(t, c.Blocks, Layer{
		XY{12390, 10060}: []interface{}{111},
		XY{12420, 10060}: []interface{}{112},
		XY{12450, 10060}: []interface{}{113},
		XY{12480, 10060}: []interface{}{114},
	})

	getBlockStack := func(x, y int) []interface{} {
		s, _ := c.Blocks.Get(x, y)
		return s
	}

	peekBlock := func(x, y int) interface{} {
		s, _ := c.Blocks.Peek(x, y)
		return s
	}

	eq(t, getBlockStack(12390, 10060), []interface{}{111})
	eq(t, getBlockStack(12420, 10060), []interface{}{112})
	eq(t, getBlockStack(12450, 10060), []interface{}{113})
	eq(t, getBlockStack(12480, 10060), []interface{}{114})

	c.Blocks.Push(12390, 10060, 0)
	eq(t, getBlockStack(12390, 10060), []interface{}{111, 0})
	c.Blocks.Pop(12390, 10060)
	eq(t, getBlockStack(12390, 10060), []interface{}{111})
	c.Blocks.Pop(12390, 10060)
	eq(t, getBlockStack(12390, 10060), []interface{}(nil))

	bytes, err := ioutil.ReadFile("levels/5911080.txt")
	if err != nil {
		panic(err)
	}
	c, err = Parse(string(bytes))
	eq(t, err, error(nil))
	eq(t, peekBlock(413*30, 312*30), 11)

	bytes, err = ioutil.ReadFile("levels/magic.txt")
	if err != nil {
		panic(err)
	}
	c, err = Parse(string(bytes))
}

func eq(t *testing.T, x, y interface{}) {
	if !reflect.DeepEqual(x, y) {
		t.Errorf("want: %v, got: %v\n%s", y, x, string(debug.Stack()))
	}
}

func neq(t *testing.T, x, y interface{}) {
	if reflect.DeepEqual(x, y) {
		t.Errorf("want: %v, got: %v\n%s", y, x, string(debug.Stack()))
	}
}
