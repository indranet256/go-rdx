package main

import (
	"bytes"
	"github.com/gritzko/rdx"
)

// if () {}
func CmdIf(ctx *Context, arg []byte) (ret []byte, err error) {
	var res []byte
	res, err = ctx.Evaluate(nil, arg)
	if len(res) == 0 {
		return nil, ErrSkip
	}
	return
}

func CmdEq(ctx *Context, arg []byte) (ret []byte, err error) {
	var rest, one, eq []byte
	for len(arg) > 0 {
		_, _, _, rest, err = rdx.ReadTLKV(arg)
		if err != nil {
			break
		}
		_one := arg[:len(arg)-len(rest)]
		one, err = ctx.Evaluate(nil, _one) // todo evaluate 1
		if err != nil {
			break
		}
		if eq == nil {
			eq = one
		} else if bytes.Compare(eq, one) != 0 {
			return nil, nil
		}
		arg = rest
	}
	return eq, nil
}
