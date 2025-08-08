package main

import (
	"fmt"
	"math/rand"

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
// brik-list fa428e
func CmdBrikList(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	readerId := rdx.ID{0, rand.Uint64() & rdx.Mask60bit}
	if !args.Read() {
		return nil, ErrNoArgument
	}
	if args.Lit() == rdx.String {
		var rest []byte
		readerId, rest = rdx.ParseID(args.Value())
		if len(rest) > 0 {
			return nil, ErrBadArguments
		}
		args.Read()
	}
	var sha rdx.Sha256
	sha, err = guessHash(*args)
	if err != nil {
		return
	}
	var brik rdx.Brik
	err = brik.OpenByHash(sha)
	if err != nil {
		return
	}
	var it rdx.BrikReader
	it, err = brik.Reader()
	if err != nil {
		return
	}
	repl.vals[readerId] = &it
	out = rdx.AppendReference(nil, readerId)
	return
}

// brix-list fa428e
func CmdBrixList(repl *REPL, args *rdx.Iter) (out []byte, err error) {

	readerId := rdx.ID{0, rand.Uint64() & rdx.Mask60bit}
	if !args.Read() {
		return nil, ErrNoArgument
	}
	if args.Lit() == rdx.String {
		var rest []byte
		readerId, rest = rdx.ParseID(args.Value())
		if len(rest) > 0 {
			return nil, ErrBadArguments
		}
		args.Read()
	}
	var sha rdx.Sha256
	sha, err = guessHash(*args)
	if err != nil {
		return
	}
	var brix rdx.Brix
	brix, err = brix.OpenByHash(sha)
	if err != nil {
		return
	}
	var it rdx.BrixReader
	it, err = brix.Reader()
	if err != nil {
		return
	}
	repl.vals[readerId] = &it
	out = rdx.AppendReference(nil, readerId)
	return
}
