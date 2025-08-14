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

func pickID(at rdx.Iter) (id rdx.ID, err error) {
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

func CmdReturn(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if args.Read() {
		out, err = repl.Eval(args)
	}
	if err == nil {
		err = Errturn
	}
	return
}

func CmdExit(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return nil, Errturn
}

func CmdLen(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoArgument
	}
	var l int64
	var a = *args
	if a.Lit() == rdx.Tuple {
		a = rdx.NewIter(args.Value())
		a.Read()
	}
	if !a.HasData() {
		l = 0
	} else if !rdx.IsPLEX(a.Lit()) {
		l = 1
	} else {
		it := rdx.NewIter(a.Value())
		for it.Read() {
			l++
		}
	}
	out = rdx.I0(l)
	return
}

func CmdVar(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNotAVariable
	}
	var n rdx.ID
	n, err = pickID(*args)
	if err != nil {
		return
	}
	if !args.Read() {
		repl.vals[n] = rdx.Stream{}
		return
	}
	var eval rdx.Stream
	eval, err = repl.Eval(args)
	if err == nil {
		repl.vals[n] = eval
	}
	return
}

var ErrNoProcedureName = errors.New("no procedure name")
var ErrNoProcedureParams = errors.New("no procedure name")
var ErrNoProcedureBody = errors.New("no procedure body")

// proc Fn(p1 p2 p3) [ ... ]
func CmdProc(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoProcedureName
	}
	var name rdx.ID
	name, err = pickID(*args)
	if err != nil {
		return
	}
	if !args.Read() || args.Lit() != rdx.Tuple {
		return nil, ErrNoProcedureParams
	}
	var pro Proc
	pro.params = args.Value()
	if !args.Read() || !rdx.IsPLEX(args.Lit()) {
		return nil, ErrNoProcedureBody
	}
	pro.body = args.Value()
	repl.pros[name] = pro
	//return rdx.R0(name), nil
	return
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

func CmdVerbatim(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if args.Read() {
		out = append(out, args.Record()...)
	}
	return
}

func CmdMute(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

const elseId = 10948073

func CmdIf(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	// todo
	return
}

func CmdPick(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var params rdx.Iter
	params, err = repl.evalArgs(args)
	if err != nil {
		return
	}
	if !params.Read() {
		return nil, ErrNoArgument
	}
	key := params.Record()
	if !params.Read() {
		return nil, ErrNoArgument
	}
	if !rdx.IsPLEX(params.Lit()) {
		return nil, ErrBadArgumentType
	}
	plex := params.Record()
	return rdx.Pick(key, plex)
}

func CmdPrint(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var eval []byte
	if !args.Read() {
		under, gotunder := repl.vals[UnderscoreId]
		if !gotunder {
			return nil, ErrNoArgument
		}
		b, ok := under.(rdx.Stream)
		if !ok {
			b, ok = under.([]byte)
			if !ok {
				return nil, ErrNotAVariable
			}
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
	id, eval, err = repl.pickIdEval(args)
	if err != nil {
		return
	}
	if id.IsZero() {
		repl.vinc++
		id = rdx.ID{0, repl.vinc}
	}
	it := rdx.NewIter(eval)
	if rdx.IsPLEX(it.Peek()) {
		j := it
		j.Read()
		if len(j.Rest()) == 0 {
			it = rdx.NewIter(j.Value())
		}
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
	var loopVar, spec rdx.ID

	var eval []byte
	loopVar, eval, err = repl.pickIdEval(args)
	if loopVar.IsZero() {
		loopVar = UnderscoreId
	}
	eit := rdx.NewIter(eval)
	if !eit.Read() {
		return nil, errors.New("nothing to iterate")
	}
	if eit.Lit() == rdx.Reference && len(eit.Rest()) == 0 {
		spec = eit.Reference()
	} else {
		return nil, errors.New("for(x plex) NIY")
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
		varId, err = pickID(it)
	}
	if err == nil {
		readerId, err = pickID(it)
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
		id, err = pickID(inner)
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
func (sr *SeqReader) Record() rdx.Stream {
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
		err = repl.Close()
		return
	}
	it := *args
	if it.Lit() == rdx.Tuple {
		it = rdx.NewIter(it.Value())
		if !it.Read() {
			return nil, ErrNoArgument
		}
	}
	id, err = pickID(it)

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

func CmdIdNow(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var id rdx.ID
	if args.Read() {
		id, _ = pickID(*args)
		if id.Src == 0 {
			id.Src, id.Seq = id.Seq, id.Src
		}
	}
	id.Seq = rdx.Timestamp()
	out = rdx.AppendReference(nil, id)
	return
}
