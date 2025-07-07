package main

import (
	"bytes"
	"github.com/gritzko/rdx"
)

// if () {}
func CmdIf(ctx *Context, args []byte, rest *[]byte) (ret []byte, err error) {
	var res []byte
	res, err = ctx.Evaluate(nil, args)
	if len(res) == 0 {
		var lit byte
		var val []byte
		var a any
		lit, _, val, *rest, _ = rdx.ReadTLKV(*rest)
		if lit == rdx.Term {
			a = ctx.resolve(val)
		} else if lit == rdx.Tuple && rdx.IsAllTerm(val) {
			_, _, path, _, _ := rdx.ReadTLKV(val)
			a = ctx.resolve(path)
		} else {
			return nil, nil
		}
		if a == nil {
			return nil, nil
		}
		switch a.(type) {
		case Command:
			lit, _, _, *rest, err = rdx.ReadTLKV(*rest)
		case Control:
			lit, _, _, *rest, err = rdx.ReadTLKV(*rest)
		default:
			lit = rdx.Tuple
		}
		if err != nil {
			return nil, rdx.ErrBadCommand
		}
		if lit != rdx.Tuple {
			return nil, ErrBadArguments
		}
		return nil, nil
	} else {
		err = ctx.Evaluate1(&ret, rest)
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
