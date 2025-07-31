package main

import (
	"errors"
	"github.com/gritzko/rdx"
)

var ErrBadArguments = errors.New("bad command arguments")

func CmdIdInt(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrBadArguments
	}
	var it rdx.Iter
	if rdx.IsPLEX(args.Lit()) {
		it = rdx.NewIter(args.Value())
	} else {
		it = rdx.NewIter(args.Record())
	}
	for it.Read() {
		switch it.Lit() {
		case rdx.Reference:
			id := it.Reference()
			out = rdx.AppendInteger(out, int64(id.Src))
			out = rdx.AppendInteger(out, int64(id.Seq))
		case rdx.Integer:
			id := rdx.ID{}
			if it.Peek() == rdx.Integer {
				id.Src = uint64(it.Integer())
				it.Read()
				id.Seq = uint64(it.Integer())
			} else {
				id.Seq = uint64(it.Integer())
			}
			out = rdx.AppendReference(out, id)
		case rdx.Term:
		default:
			return nil, ErrBadArguments
		}
	}
	return
}
