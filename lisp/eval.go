package main

import (
	"errors"
	rdx "github.com/gritzko/rdx"
)

type Context struct {
	vars  map[string][]byte
	funs  map[string]Command
	subs  map[string]*Context
	ptrs  map[string]any
	stack rdx.Marks
}

type Command func(ctx *Context, rdx []byte) (out []byte, err error)

var ErrNotACall = errors.New("not a function call")

func (ctx *Context) resolve(path []byte) (c *Context, fn Command, va []byte) {
	if len(path) == 0 || rdx.Peek(path) != rdx.Term {
		return
	}
	var err error
	var val []byte
	_, _, val, path, err = rdx.ReadTLKV(path)
	if err != nil {
		return
	} else if len(path) == 0 {
		f := ctx.funs[string(val)]
		if f != nil {
			return ctx, f, nil
		}
		v := ctx.vars[string(val)]
		if v != nil {
			return ctx, nil, v
		}
	} else {
		sub := ctx.subs[string(val)]
		if sub != nil {
			c, fn, va = sub.resolve(path)
		}
	}
	return
}

func (ctx *Context) Evaluate(pre, args []byte) (out []byte, err error) {
	out = pre
	for len(args) > 0 && err == nil {
		var lit byte
		var id, val, rest []byte
		lit, id, val, rest, err = rdx.ReadTLKV(args)
		if err != nil {
			break
		}
		whole := args[:len(args)-len(rest)]

		if lit == rdx.Term || lit == rdx.Tuple {
			path := whole
			if lit == rdx.Tuple {
				path = val
			}
			c, fn, va := ctx.resolve(path)
			if va != nil {
				out = append(out, va...)
				args = rest
				continue
			} else if fn != nil {
				var fnargs []byte
				if len(rest) > 0 && rdx.Peek(rest) == rdx.Tuple {
					_, _, fnargs, rest, err = rdx.ReadTLKV(rest)
				}
				var res []byte
				res, err = fn(c, fnargs)
				out = append(out, res...)
				args = rest
				continue
			}
		}

		if rdx.IsFIRST(lit) {
			out = append(out, whole...)
		} else {
			out = rdx.OpenTLV(out, lit, &ctx.stack)
			out = append(out, byte(len(id)))
			out = append(out, id...)
			ol := len(out)
			out, err = ctx.Evaluate(out, val)
			if ol == len(out) {
				out, err = rdx.CancelTLV(out, lit, &ctx.stack)
			} else {
				out, err = rdx.CloseTLV(out, lit, &ctx.stack)
			}
		}
		args = rest
	}
	return
}
