package main

import (
	"bytes"
	"errors"
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

func readerVar(ctx *Context, args []byte) (r rdx.Reader, rest []byte, err error) {
	it := rdx.NewIter(args)
	if !it.Read() {
		err = ErrBadName
		return
	}
	b := ctx.resolve(it.Record())
	if b == nil {
		err = ErrNameNotFound
	} else {
		switch b.(type) {
		case rdx.Reader:
			r = b.(rdx.Reader)
			rest = it.Rest()
		default:
			err = ErrUnexpectedNameType
		}
	}
	return
}

func CmdScan(ctx *Context, args []byte) (ret []byte, err error) {
	it := rdx.NewIter(args)
	if !it.Read() {
		return nil, ErrBadArguments
	}
	if it.Lit() != rdx.Tuple && it.Lit() != rdx.Term {
		return nil, ErrNameNotFound
	}
	err = ctx.set(it.Record(), &it)
	return
}

func CmdRead(ctx *Context, args []byte) (ret []byte, err error) {
	var it rdx.Reader
	it, args, err = readerVar(ctx, args)
	if err != nil {
		return
	}
	if !it.Read() {
		return nil, nil
	} else {
		return it.Record(), nil
	}
}

var ErrNoLoopVariable = errors.New("no loop variable specified")

// for(i (1 2 3 4 5)) [ echo i ]
func CmdFor(ctx *Context, args []byte, rest *[]byte) (ret []byte, err error) {
	if rdx.Peek(args) != rdx.Term {
		return nil, ErrNoLoopVariable
	}
	var name []byte
	_, _, name, args, err = rdx.ReadRDX(args)
	if len(args) == 0 {
		return nil, nil
	}
	oldValue, hadOldValue := ctx.names[string(name)]

	var rem, code []byte
	_, _, _, rem, err = rdx.ReadRDX(*rest)
	code = (*rest)[:len(*rest)-len(rem)]
	*rest = rem

	var list []byte
	_, _, list, args, err = rdx.ReadRDX(args)
	for len(list) > 0 && err == nil {
		var re []byte
		_, _, _, re, err = rdx.ReadRDX(list)
		if err != nil {
			break
		}
		one := list[:len(list)-len(re)] // TODO iter
		ctx.names[string(name)] = one
		ret, err = ctx.Evaluate(ret, code)
		list = re
	}

	if hadOldValue {
		ctx.names[string(name)] = oldValue
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
