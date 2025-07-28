package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gritzko/rdx"
	"math/rand"
	"reflect"
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

func IsPath(path []byte) bool {
	it := rdx.NewIter(path)
	if !it.Read() || len(it.Rest()) > 0 {
		return false
	}
	if it.Lit() == rdx.Term {
		return true
	}
	if it.Lit() != rdx.Tuple {
		return false
	}
	return rdx.IsAllTerm(it.Value())
}

func randomPath() []byte {
	random := []byte(fmt.Sprintf("%x", rand.Uint32()))
	return rdx.AppendTerm(nil, random)
}

// FIXME a path is a T sequence!!!
func readPath(args *rdx.Iter) (path []byte) {
	var err error
	a := *args
	if !a.Read() {
		return nil
	}
	switch a.Lit() {
	case rdx.String:
		path, err = rdx.ParseJDR(a.Value())
		if err != nil || !IsPath(path) {
			return nil
		}
	case rdx.Term:
		path = a.Record()
	case rdx.Tuple:
		path = a.Value()
		if !rdx.IsAllTerm(path) {
			return nil
		}
	}
	if path != nil {
		*args = a
	}
	return path
}

func CmdList(ctx *Context, args rdx.Iter) (ret []byte, err error) {
	path := readPath(&args)
	if path == nil {
		path = randomPath()
	}
	if rdx.IsPLEX(args.Peek()) {
		a := args
		if !a.Read() {
			return nil, ErrBadArguments
		}
		if len(a.Rest()) == 0 {
			args = rdx.NewIter(a.Value())
		}
	}
	err = ctx.set(path, &args)
	ret = path
	return
}

type SeqReader struct {
	i int64
	e int64
}

func (sr *SeqReader) Read() bool {
	if sr.i < sr.e {
		sr.i++
		return true
	}
	return false
}
func (sr *SeqReader) Record() []byte {
	return rdx.AppendInteger(nil, sr.i)
}
func (sr *SeqReader) Parsed() (lit byte, id rdx.ID, value []byte) {
	return rdx.Integer, rdx.ID{}, rdx.ZipInt64(sr.i)
}
func (sr *SeqReader) Error() error {
	return nil
}

func CmdSeq(ctx *Context, args rdx.Iter) (ret []byte, err error) {
	var path []byte
	path = readPath(&args)
	if path == nil {
		path = randomPath()
	}
	seq := SeqReader{0, 0}
	if args.Read() && args.Lit() == rdx.Integer {
		seq.e = rdx.UnzipInt64(args.Value())
	}
	if args.Read() && args.Lit() == rdx.Integer {
		seq.i = seq.e - 1
		seq.e = rdx.UnzipInt64(args.Value())
	}
	err = ctx.set(path, &seq)
	ret = path
	return
}

func CmdRead(ctx *Context, args *rdx.Iter) (ret []byte, err error) {
	if !args.Read() {
		return nil, ErrBadArguments
	}
	if !IsPath(args.Record()) {
		return nil, ErrBadArguments
	}
	reader := ctx.Get(args)
	if reader == nil {
		return nil, ErrNameNotFound
	}
	rdr, ok := reader.(rdx.Reader)
	if !ok {
		return nil, ErrUnexpectedNameType
	}
	if !rdr.Read() {
		return nil, nil
	} else {
		return rdr.Record(), nil
	}
}

func isPath(it *rdx.Iter) bool {
	return it.HasData() && !it.HasFailed() &&
		(it.Lit() == rdx.Term || (it.Lit() == rdx.Tuple && rdx.IsAllTerm(it.Record())))
}

func CmdOver(ctx *Context, rest *rdx.Iter) (ret []byte, err error) {
	if !rest.Read() || !isPath(rest) {
		return nil, ErrBadArguments
	}
	err = ctx.set(rest.Record(), nil)
	return
}

var ErrNoLoopVariable = errors.New("no loop variable specified")
var ReaderType = reflect.TypeOf((*rdx.Reader)(nil))

type Unwrapper struct {
	it []rdx.Iter
}

func (un *Unwrapper) Read() bool {
	if len(un.it) == 0 {
		return false
	}
	last := un.it[len(un.it)-1]
	if !last.Read() {
		return false
	}
	if rdx.IsPLEX(last.Lit()) {
		un.it = append(un.it, rdx.NewIter(last.Value()))
		return un.Read()
	}
	return true
}

var ErrNoCodeBlock = errors.New("no code block specified")
var TermUnderscore []byte = []byte{'t', 2, 0, '_'}

func CmdFor(ctx *Context, args *rdx.Iter) (ret []byte, err error) {
	isMap := bytes.Equal(args.Record(), []byte{'t', 4, 0, 'm', 'a', 'p'})
	if !args.Read() {
		return nil, ErrNoLoopVariable
	}
	var readerPath []byte
	var loopVarPath []byte = []byte{'t', 2, 0, '_'}
	var params []byte
	params, err = ctx.Eval1(args)
	parit := rdx.NewIter(params)
	switch parit.Peek() {
	case rdx.String:
		fallthrough
	case rdx.Term:
		readerPath = readPath(&parit)
	case rdx.Tuple:
		parit.Read()
		parit = rdx.NewIter(parit.Value())
		loopVarPath = readPath(&parit)
		readerPath = readPath(&parit)
		if readerPath == nil {
			readerPath = loopVarPath
			loopVarPath = TermUnderscore
		}
	}
	a := ctx.resolve(readerPath)
	if a == nil {
		return nil, ErrNameNotFound
	}
	rdr, ok := a.(rdx.Reader)
	if !ok {
		return nil, ErrUnexpectedNameType
	}
	if !args.Read() {
		return nil, ErrNoCodeBlock
	}
	var code rdx.Iter
	old := ctx.resolve(loopVarPath)
	for rdr.Read() {
		_ = ctx.set(loopVarPath, rdr.Record())
		code = *args
		var out []byte
		out, err = ctx.Eval1(&code)
		if isMap {
			ret = append(ret, out...)
		}
	}
	_ = ctx.set(loopVarPath, old)
	*args = code
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
