package main

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/gritzko/rdx"
)

var ErrBadVariableType = errors.New("bad variable type")

func pickString(at rdx.Iter) (term string, err error) {
	switch at.Lit() {
	case rdx.Integer:
		term = fmt.Sprintf("%d", rdx.UnzipInt64(at.Value()))
	case rdx.Term:
		fallthrough
	case rdx.String:
		term = string(at.Value())
	default:
		return "", ErrBadVariableType
	}
	return
}

func pickHash(at rdx.Iter) (sha rdx.Sha256, err error) {
	switch at.Lit() {
	case rdx.Integer:
		str := fmt.Sprintf("%d", rdx.UnzipInt64(at.Value()))
		sha, err = rdx.FindByHashlet(str)
	case rdx.Term:
		fallthrough
	case rdx.String:
		sha, err = rdx.FindByHashlet(string(at.Value()))
	default:
		err = ErrBadVariableType
	}
	return
}

func (repl *REPL) pickBrik(at rdx.Iter) (brik *rdx.Brik, err error) {
	brik = &rdx.Brik{}
	
	if at.Lit() == rdx.Reference {
		id := at.Reference()
		pre, has := repl.vals[id]
		if has {
			prebrik, ok := pre.(*rdx.Brik)
			if ok {
				return prebrik, nil
			} else {
				return nil, ErrBadVariableType
			}
		}
		path := rdx.TipPath(id.Src)
		err = brik.OpenByPath(path)
		if err == nil {
			repl.vals[id] = brik
		}
		return brik, err
	}

	var hashlet string
	hashlet, err = pickString(at)
	id, _ := rdx.ParseID([]byte(hashlet))
	ex, ok := repl.vals[id]
	if ok {
		brik, ok = ex.(*rdx.Brik)
		if !ok {
			return nil, ErrBadArguments
		}
	} else {
		var sha rdx.Sha256
		sha, err = rdx.FindByHashlet(hashlet)
		if err == nil {
			err = brik.OpenByHash(sha)
		}
		if err == nil {
			repl.vals[id] = brik
		}
	}
	return
}

func pickBrix(at rdx.Iter) (brik *rdx.Brik, err error) {
	return
}

// brik-list("b" fa428e)
// brik-list fa428e
func CmdListBrik(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoArgument
	}
	readerId, err := pickStringID(args)
	var brik *rdx.Brik
	brik, err = repl.pickBrik(*args)
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

func pickStringID(args *rdx.Iter) (id rdx.ID, err error) {
	if args.Lit() != rdx.String {
		id = rdx.ID{0, rand.Uint64() & rdx.Mask60bit}
	} else {
		var rest []byte
		id, rest = rdx.ParseID(args.Value())
		if len(rest) > 0 {
			err = ErrBadArguments
		} else {
			args.Read()
		}
	}
	return
}

// brix-list fa428e
func CmdListBrix(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoArgument
	}
	var readerId rdx.ID
	readerId, err = pickStringID(args)
	if err != nil {
		return
	}
	var sha rdx.Sha256
	sha, err = pickHash(*args)
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
