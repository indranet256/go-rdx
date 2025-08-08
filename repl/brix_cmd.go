package main

import (
	"fmt"
	"github.com/gritzko/rdx"
)

func guessHash(at rdx.Iter) (sha rdx.Sha256, err error) {
	switch at.Lit() {
	case rdx.Integer:
		str := fmt.Sprintf("%d", rdx.UnzipInt64(at.Value()))
		sha, err = rdx.FindByHashlet(str)
	case rdx.Term:
		fallthrough
	case rdx.String:
		sha, err = rdx.FindByHashlet(string(at.Value()))
	default:
		err = ErrBadArguments
	}
	return
}

// brik-list("b" fa428e)
func CmdBrikList(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() || args.Lit() != rdx.String {
		return nil, ErrBadArguments
	}
	handle, rest := rdx.ParseID(args.Value())
	if len(rest) > 0 {
		return nil, ErrBadArguments
	}
	var sha rdx.Sha256
	if !args.Read() {
		err = ErrNoArgument
	} else {
		sha, err = guessHash(*args)
	}
	if err != nil {
		return
	}
	var brik rdx.Brik
	err = brik.OpenByHash(sha)
	if err != nil {
		return
	}
	var it rdx.BrikReader
	it, err = brik.Seek(rdx.ID{})
	if err != nil {
		return
	}
	repl.vals[handle] = it
	return
}

// brix-list fa428e
func CmdBrixList(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}
