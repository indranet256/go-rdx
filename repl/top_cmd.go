package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"

	"github.com/gritzko/rdx"
)

var ErrNotAVariable = errors.New("the argument is not a variable")

func pickVar(at rdx.Iter) (id rdx.ID, err error) {
	switch at.Lit() {
	case rdx.Integer:
		str := fmt.Sprintf("%d", rdx.UnzipInt64(at.Value()))
		id, _ = rdx.ParseID([]byte(str))
	case rdx.Term:
		fallthrough
	case rdx.String:
		id, _ = rdx.ParseID(at.Value())
	case rdx.Reference:
		id = rdx.UnzipID(at.Value())
	default:
		err = ErrNotAVariable
	}
	return
}

func CmdExit(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return nil, Errturn
}

func CmdString(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	stack := make(rdx.Marks, 2)
	tmp := rdx.OpenShortTLV(nil, rdx.String, &stack)
	tmp = append(tmp, 0)
	for args.Read() {
		tmp = append(tmp, args.String()...)
	}
	out, err = rdx.CloseTLV(tmp, rdx.String, &stack)
	return
}

func CmdPrint(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var eval []byte
	if !args.Read() {
		under, gotunder := repl.vals[UnderscoreId]
		if !gotunder {
			return nil, ErrNoArgument
		}
		b, ok := under.([]byte)
		if !ok {
			return nil, ErrNotAVariable
		}
		eval = b
	} else {
		eval, err = repl.Eval(args)
	}
	if err != nil {
		return
	}
	jdr, _ := rdx.WriteAllJDR(nil, eval, 0)
	fmt.Println(string(jdr))
	return nil, nil
}

// list("l" (1 2 3))
// list(1 2 3)
// list(eset)
func CmdList(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var id rdx.ID
	var eval []byte
	if !args.Read() {
		return nil, ErrNoArgument
	}
	id, err = pickStringID(args)
	if err != nil {
		return
	}
	eval, err = repl.Eval(args)
	if err != nil {
		return
	}
	it := rdx.NewIter(eval)
	if !it.Read() {
		return
	}
	if rdx.IsPLEX(it.Lit()) && len(it.Rest()) == 0 {
		it = rdx.NewIter(it.Value())
	}

	repl.vals[id] = rdx.Reader(&it)

	out = rdx.AppendReference(nil, id)

	return
}

var ErrNotAReader = errors.New("the var is not castable to a reader")
var ErrNotImplementedYet = errors.New("not implemented yet")
var ErrNoLoopVariable = errors.New("no loop variable specified")
var ErrNoLoopBody = errors.New("no loop body specified")
var ReaderType = reflect.TypeOf((*rdx.Reader)(nil))

// for brix-open f452 print
// for f452 print
// for seq(1 100) [ print ]
// for readerAB print;
func CmdFor(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	isMap := bytes.Equal(args.Record(), []byte{'t', 4, 0, 'm', 'a', 'p'})
	var rdr rdx.Reader
	if !args.Read() {
		return nil, ErrNoLoopVariable
	}
	var loopVar, spec rdx.ID
	loopVar = UnderscoreId
	var eval []byte

	// TODO pickStringID(), pickReader() !!!

	eval, err = repl.Eval(args)

	ait := rdx.NewIter(eval)
	if !ait.Read() {
		return nil, ErrNoLoopVariable
	}
	if ait.Lit() == rdx.Tuple {
		ait = rdx.NewIter(ait.Value())
		if !ait.Read() {
			return nil, ErrNoLoopVariable
		}
	}
	spec, err = pickVar(ait)
	if err == nil && ait.Read() {
		loopVar = spec
		spec, err = pickVar(ait)
	}
	if err != nil {
		return
	}

	local, oklocal := repl.vals[spec]
	if !oklocal {
		return nil, ErrNoLoopVariable
	}

	var ok bool
	rdr, ok = local.(rdx.Reader)
	if !ok {
		return nil, ErrNotAReader
	}

	if !args.Read() {
		return nil, ErrNoLoopBody
	}
	code := *args

	old, hadOld := repl.vals[loopVar]

	for rdr.Read() {
		repl.vals[loopVar] = rdr.Record()
		code = *args
		var tmp []byte
		tmp, err = repl.Eval(&code)
		if isMap {
			out = append(out, tmp...)
		}
	}

	if hadOld {
		repl.vals[loopVar] = old
	}
	*args = code

	return
}

var UnderscoreId = rdx.ID{0, 36}

// brix-open 5a4e; read (rec 5a4e); print rec; brix-close 5a4e;
// read reader
// read list abc
// read (i list abc)
func CmdRead(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoArgument
	}
	var eval []byte
	eval, err = repl.Eval(args)
	it := rdx.NewIter(eval)
	if !it.Read() {
		return nil, ErrNoArgument
	}
	if it.Lit() == rdx.Tuple {
		it = rdx.NewIter(it.Value())
		if !it.Read() {
			return nil, ErrNoArgument
		}
	}
	varId := rdx.ID0
	var readerId rdx.ID
	if len(it.Rest()) != 0 {
		varId, err = pickVar(it)
	}
	if err == nil {
		readerId, err = pickVar(it)
	}
	if err != nil {
		return
	}

	rdrany, okrdr := repl.vals[readerId]
	if !okrdr {
		return nil, ErrNotAVariable
	}
	rdr, oktype := rdrany.(rdx.Reader)
	if !oktype {
		return nil, ErrNotAReader
	}

	if !rdr.Read() {
		// todo Close
		delete(repl.vals, readerId)
		err = rdr.Error()
	} else {
		if !varId.IsZero() {
			repl.vals[varId] = rdr.Record()
		}
		out = rdr.Record()
	}
	return
}

// seq("i" 1 100) -> 0-i
// seq(1 100) -> tmp-aK4b
// seq 100 -> tmp-aK4b
// seq(a b) -> tmp-xyz
func CmdSeq(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	seq := SeqReader{0, 0}
	id := rdx.ID{232564, rand.Uint64() & rdx.Mask60bit}

	if !args.Read() {
		return nil, ErrNoArgument
	}
	var inner rdx.Iter
	if args.Lit() == rdx.Integer {
		inner = rdx.NewIter(args.Record())
	} else if args.Lit() == rdx.Tuple {
		inner = rdx.NewIter(args.Value())
	}

	if !inner.Read() {
		return nil, ErrNoArgument
	}
	if inner.Lit() != rdx.Integer {
		id, err = pickVar(inner)
		if err != nil {
			return
		}
		_ = inner.Read()
	}
	if inner.Lit() == rdx.Integer {
		seq.e = rdx.UnzipInt64(inner.Value())
		if inner.Read() && inner.Lit() == rdx.Integer {
			seq.i = seq.e - 1
			seq.e = rdx.UnzipInt64(inner.Value())
			_ = inner.Read()
		}
	}
	if inner.HasData() {
		return nil, ErrBadArguments
	}

	repl.vals[id] = rdx.Reader(&seq)
	out = rdx.AppendReference(nil, id)

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

// close(var)
func CmdClose(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var id rdx.ID
	if !args.Read() {
		return nil, ErrNoArgument
	}
	it := *args
	if it.Lit() == rdx.Tuple {
		it = rdx.NewIter(it.Value())
		if !it.Read() {
			return nil, ErrNoArgument
		}
	}
	id, err = pickVar(it)

	a, ok := repl.vals[id]
	if !ok {
		return nil, ErrNotAVariable
	}
	c, tok := a.(io.Closer)
	if !tok {
		return nil, ErrBadVariableType
	}
	err = c.Close()

	return
}
