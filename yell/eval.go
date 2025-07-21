package main

import (
	"errors"
	rdx "github.com/gritzko/rdx"
)

type Context struct {
	names map[string]any
	stack rdx.Marks
}

type Command func(ctx *Context, args []byte) (out []byte, err error)
type Control func(ctx *Context, args []byte, rest *[]byte) (out []byte, err error)

var ErrNotACall = errors.New("not a function call")
var ErrSkip = errors.New("skip an element")
var ErrRepeat = errors.New("repeat an element")
var ErrUnexpectedNameType = errors.New("the name is associated to a value of a different type")

func (ctx *Context) resolve(path []byte) any {
	if len(path) == 0 || rdx.Peek(path) != rdx.Term {
		return nil
	}
	var err error
	var val []byte
	_, _, val, path, err = rdx.ReadTLKV(path)
	if err != nil {
		return nil
	}
	f, ok := ctx.names[string(val)]
	if !ok {
		return nil
	}
	switch f.(type) {
	case *Context:
		if len(path) > 0 {
			return f.(*Context).resolve(path)
		} else {
			return f
		}
	default:
		if len(path) > 0 {
			return nil
		} else {
			return f
		}
	}
}

func (ctx *Context) Evaluate1(data, code *[]byte) (err error) {
	out := *data
	var lit byte
	var id, val, next []byte
	lit, id, val, next, err = rdx.ReadTLKV(*code)
	if err != nil {
		return
	}
	whole := (*code)[:len(*code)-len(next)]
	var a any
	if lit == rdx.Term {
		a = ctx.resolve(whole)
	} else if lit == rdx.Tuple && rdx.IsAllTerm(val) {
		_, _, path, _, _ := rdx.ReadTLKV(whole) // unwrap
		a = ctx.resolve(path)
	}
	if a != nil {
		switch a.(type) {
		case []byte:
			out = append(out, a.([]byte)...)
		case Command:
			if len(next) == 0 || rdx.Peek(next) != rdx.Tuple {
				return ErrBadArguments
			}
			var cmdargs, eargs, res []byte
			_, _, cmdargs, next, err = rdx.ReadTLKV(next)
			if err == nil {
				eargs, err = ctx.Evaluate(nil, cmdargs)
			}
			if err != nil {
				return
			}
			res, err = a.(Command)(ctx, eargs)
			if err != nil {
				return
			}
			out = append(out, res...)
		case Control:
			if len(next) == 0 || rdx.Peek(next) != rdx.Tuple {
				return ErrBadArguments
			}
			var cmdargs, res []byte
			_, _, cmdargs, next, err = rdx.ReadTLKV(next)
			res, err = a.(Control)(ctx, cmdargs, &next)
			if err != nil {
				return
			}
			out = append(out, res...)
		default:
			return ErrUnexpectedNameType
		}
	} else if rdx.IsFIRST(lit) {
		out = append(out, whole...)
	} else {
		out = rdx.OpenTLV(out, lit, &ctx.stack)
		out = append(out, byte(len(id)))
		out = append(out, id...)
		out, err = ctx.Evaluate(out, val)
		if err != nil {
			return
		}
		out, err = rdx.CloseTLV(out, lit, &ctx.stack)
	}
	*code = next
	*data = out
	return nil
}

func (ctx *Context) Evaluate(data, code []byte) (out []byte, err error) {
	out = data
	for len(code) > 0 && err == nil {
		err = ctx.Evaluate1(&out, &code)
	}
	return
}
