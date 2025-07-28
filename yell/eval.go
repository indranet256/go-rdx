package main

import (
	"errors"
	rdx "github.com/gritzko/rdx"
)

type Context struct {
	names   map[string]any
	unnamed any
	stack   rdx.Marks
}

type Function func(ctx *Context, args rdx.Iter) (out []byte, err error)
type Operator func(ctx *Context, args *rdx.Iter) (out []byte, err error)

type Command func(ctx *Context, args []byte) (out []byte, err error)
type Control func(ctx *Context, args []byte, rest *[]byte) (out []byte, err error)
type Control2 func(ctx *Context, args rdx.Iter, rest *rdx.Iter) (out []byte, err error)
type Call func(ctx *Context, path, args []byte) (out []byte, err error)

var ErrNotACall = errors.New("not a function call")
var ErrSkip = errors.New("skip an element")
var ErrRepeat = errors.New("repeat an element")
var ErrUnexpectedNameType = errors.New("the name is associated to a value of a different type")

func (ctx *Context) Get(code *rdx.Iter) any {
	c := ctx
	var a any
	path := *code
	if path.Lit() == rdx.Term {
		a = c.names[string(path.Value())]
	} else if path.Lit() == rdx.Tuple {
		nested := rdx.NewIter(path.Value())
		for nested.Read(); len(nested.Rest()) > 0; nested.Read() {
			if nested.Lit() != rdx.Term {
				return nil
			}
			a = c.names[string(nested.Value())]
			switch a.(type) {
			case *Context:
				c = a.(*Context)
			default:
				return nil
			}
		}
		a = c.names[string(nested.Value())]
	}
	*code = path
	return a
}

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

var ErrOverwriteForbidden = errors.New("overwrite of built-in values is forbidden")

func (ctx *Context) set(path []byte, v any) (err error) {
	if len(path) == 0 || rdx.Peek(path) != rdx.Term {
		return ErrBadArguments
	}
	var val []byte
	_, _, val, path, err = rdx.ReadTLKV(path)
	if err != nil {
		return
	}
	f, ok := ctx.names[string(val)]
	if !ok && len(path) == 0 {
		ctx.names[string(val)] = v
		return
	}
	switch f.(type) {
	case *Context:
		if len(path) == 0 {
			return ErrOverwriteForbidden
		} else {
			err = f.(*Context).set(path, v)
		}
	case Command:
		return ErrOverwriteForbidden
	case Control:
		return ErrOverwriteForbidden
	case Call:
		return ErrOverwriteForbidden
	default:
		if len(path) == 0 {
			ctx.names[string(val)] = v
		} else {
			err = ErrUnexpectedNameType
		}
	}
	return
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
		a = ctx.resolve(val)
	}
	if a != nil {
		switch a.(type) {
		case []byte:
			out = append(out, a.([]byte)...)
		case Call:
			if len(next) == 0 || rdx.Peek(next) != rdx.Tuple {
				return ErrBadArguments
			}
			var cargs, eargs, path, rargs, res []byte
			_, _, cargs, next, err = rdx.ReadTLKV(next)
			if len(cargs) == 0 || (rdx.Peek(cargs) != rdx.Tuple && rdx.Peek(cargs) != rdx.Term) {
				return ErrBadArguments
			}
			_, _, _, rargs, err = rdx.ReadTLKV(cargs)
			path = cargs[:len(cargs)-len(rargs)]
			if err == nil {
				eargs, err = ctx.Evaluate(nil, rargs)
			}
			if err != nil {
				return
			}
			res, err = a.(Call)(ctx, path, eargs)
			if err != nil {
				return
			}
			out = append(out, res...)
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
		case Control2:
			var res []byte
			it := rdx.NewIter(next)
			//res, err = a.(Control2)(ctx, &it)
			if err != nil {
				return
			}
			next = it.Rest()
			out = append(out, res...)
		case rdx.Reader:
			out = append(out, a.(rdx.Reader).Record()...)
		default:
			out = append(out, whole...)
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

func (ctx *Context) Eval1(code *rdx.Iter) (out []byte, err error) {
	var a any
	switch code.Lit() {
	case rdx.Float:
		fallthrough
	case rdx.Integer:
		fallthrough
	case rdx.Reference:
		fallthrough
	case rdx.String:
		out = append(out, code.Record()...)
		return
	case rdx.Term:
		a = ctx.Get(code)
		if a == nil {
			out = append(out, code.Record()...)
			return
		}
	case rdx.Tuple:
		a = ctx.Get(code)
		if a != nil {
			break
		}
		fallthrough
	case rdx.Euler:
		fallthrough
	case rdx.Multix:
		fallthrough
	case rdx.Linear:
		out = rdx.OpenTLV(out, code.Lit(), &ctx.stack)
		id := rdx.ZipID(code.ID())
		out = append(out, byte(len(id)))
		out = append(out, id...)
		it := rdx.NewIter(code.Value())
		var ev []byte
		ev, err = ctx.Eval(&it)
		out = append(out, ev...)
		out, _ = rdx.CloseTLV(out, code.Lit(), &ctx.stack)
		return
	}
	switch a.(type) {
	case []byte:
		out = append(out, a.([]byte)...)
	case rdx.Reader:
		out = append(out, a.(rdx.Reader).Record()...)
	case Function:
		var eval, expr []byte
		if code.Peek() == rdx.Tuple {
			_ = code.Read()
			expr = code.Value()
			args := rdx.NewIter(code.Value())
			eval, err = ctx.Eval(&args)
		} else if code.Read() {
			expr = code.Record()
			eval, err = ctx.Eval1(code)
		}
		if err == nil {
			args := rdx.NewIter(eval)
			var res []byte
			res, err = a.(Function)(ctx, args)
			if err != nil {
				jdr, _ := rdx.WriteAllJDR(nil, expr, 0)
				err = errors.New(err.Error() + " in " + string(jdr))
			}
			out = append(out, res...)
		}
	case Operator:
		var res []byte
		res, err = a.(Operator)(ctx, code)
		out = append(out, res...)
	default:
		out = append(out, code.Record()...)
	}
	return
}

func (ctx *Context) Eval(code *rdx.Iter) (out []byte, err error) {
	for err == nil && code.Read() {
		var one []byte
		one, err = ctx.Eval1(code)
		out = append(out, one...)
	}
	return
}
