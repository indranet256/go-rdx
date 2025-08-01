package main

import (
	"errors"
	"fmt"
	"github.com/gritzko/rdx"
)

// brix:new({@author-seq field:"value"})
func CmdBrixNew(ctx *Context, args []byte) (out []byte, err error) {
	w := rdx.Brik{}
	err = w.Create([]rdx.Sha256{})
	_, err = w.WriteAll(args)
	if err != nil {
		_ = w.Unlink()
		return
	}
	err = w.Seal()
	if err != nil {
		return
	}
	err = w.Close()
	if err == nil {
		out = rdx.AppendTerm(nil, []byte(w.Hash7574.String()))
	}
	return
}

var ErrNoVariableName = errors.New("variable name not specified")

func openBrixBySpec(brix rdx.Brix, spec []byte) (more rdx.Brix, err error) {
	more = brix
	for len(spec) > 0 && err == nil {
		var lit byte
		var val []byte
		lit, _, val, spec, err = rdx.ReadRDX(spec)
		switch lit {
		case rdx.Integer:
			str := fmt.Sprintf("%d", rdx.UnzipInt64(val))
			more, err = more.OpenByHashlet(str)
		case rdx.Term:
			fallthrough
		case rdx.String:
			more, err = more.OpenByHashlet(string(val))
		case rdx.Tuple:
			fallthrough
		case rdx.Linear:
			fallthrough
		case rdx.Euler:
			more, err = openBrixBySpec(more, val)
		default:
			return nil, ErrBadArguments
		}
	}
	return
}

// brix:open(var f7b055)
// brix:open(var f7b055)
// brix:open(var "f7b055")
// brix:open(var 524564)
func CmdBrixOpen(ctx *Context, args []byte) (out []byte, err error) {
	if len(args) == 0 || rdx.Peek(args) != rdx.Term {
		return nil, ErrNoVariableName
	}
	var name, rest []byte
	_, _, name, rest, err = rdx.ReadRDX(args)
	if err != nil {
		return
	}
	if len(rest) == 0 {
		rest = args
	}
	var brix rdx.Brix
	brix, err = openBrixBySpec(nil, rest)
	if err == nil {
		ctx.names[string(name)] = brix
	}
	return
}

var ErrNameNotFound = errors.New("name not found")

// brix:info(var)
func CmdBrixInfo(ctx *Context, args []byte) (out []byte, err error) {
	brix := ctx.resolve(args)
	if brix == nil {
		return nil, ErrNameNotFound
	}
	switch brix.(type) {
	case rdx.Brix:
		b := brix.(rdx.Brix)
		for n, b := range b {
			fmt.Printf("%d. %s (%d bytes, %d pages)\n",
				n+1, b.Hash7574.String(), b.Header.DataLen, b.Header.IndexLen/32)
		}
	default:
		return nil, ErrUnexpectedNameType
	}
	return
}

func CmdBrixFind(ctx *Context, args []byte) (out []byte, err error) {
	var lit byte
	var val []byte
	lit, _, val, args, err = rdx.ReadRDX(args)
	var sha rdx.Sha256
	switch lit {
	case rdx.Integer:
		str := fmt.Sprintf("%d", rdx.UnzipInt64(val))
		sha, err = rdx.FindByHashlet(str)
	case rdx.Term:
		fallthrough
	case rdx.String:
		sha, err = rdx.FindByHashlet(string(val))
	}
	if err == nil {
		out = rdx.AppendTerm(out, []byte(sha.String()))
	}
	return
}

func CmdBrixClose(ctx *Context, args []byte) (out []byte, err error) {
	brix := ctx.resolve(args)
	if brix == nil {
		return nil, ErrNameNotFound
	}
	switch brix.(type) {
	case rdx.Brix:
		b := brix.(rdx.Brix)
		err = b.Close()
		err = ctx.set(args, nil)
	default:
		return nil, ErrUnexpectedNameType
	}
	return
}

func CmdBrixGet(ctx *Context, args []byte) (out []byte, err error) {
	if rdx.Peek(args) != rdx.Term {
		return nil, ErrBadArguments
	}
	var id rdx.ID
	var hashlet []byte
	hashlet, _, args, err = rdx.ReadTerm(args)
	if err != nil {
		return nil, err
	}
	var brix rdx.Brix
	brix, err = brix.OpenByHashlet(string(hashlet))
	if err != nil {
		return
	}
	id, _, args, err = rdx.ReadID(args)
	out, err = brix.Get(nil, id)
	return
}

// brix:add (3c0dce, {@alice-345 5:"five"})
func CmdBrixAdd(ctx *Context, args []byte) (out []byte, err error) {
	w := rdx.Brik{}
	var hashlet []byte
	hashlet, _, args, err = rdx.ReadTerm(args)
	if err != nil {
		return nil, err
	}
	deps := make([]rdx.Sha256, 0, 1)
	var hash rdx.Sha256
	if len(hashlet) == rdx.Sha256Bytes*2 {
		hash, err = rdx.ParseSha256(hashlet)
	} else if len(hashlet) < rdx.Sha256Bytes*2 {
		hash, err = rdx.FindByHashlet(string(hashlet))
	} else {
		return nil, ErrBadArguments
	}
	if err != nil {
		return
	}
	deps = append(deps, hash)
	err = w.Create(deps)
	_, err = w.WriteAll(args)
	if err != nil {
		_ = w.Unlink()
		return
	}
	err = w.Seal()
	if err != nil {
		return
	}
	err = w.Close()
	if err == nil {
		out = rdx.WriteRDX(nil, rdx.Term, rdx.ID{}, []byte(w.Hash7574.String()))
	}
	return
}

func CmdBrixDel(ctx *Context, args []byte) (out []byte, err error) {
	return
}
func CmdBrixHas(ctx *Context, args []byte) (out []byte, err error) {
	return
}

// evaluate for every record in a range
func CmdBrixScan(ctx *Context, args []byte) (out []byte, err error) {
	brix := ctx.resolve(args)
	if brix == nil {
		return nil, ErrNameNotFound
	}
	var it rdx.BrixReader
	switch brix.(type) {
	case rdx.Brix:
		b := brix.(rdx.Brix)
		it, err = b.Iterator()
	default:
		return nil, ErrUnexpectedNameType
	}
	var jdr []byte
	for it.Read() {
		if err == nil {
			jdr, err = rdx.WriteAllJDR(jdr, it.Record(), 0)
			fmt.Println(string(jdr))
		}
		jdr = jdr[:0]
	}
	return
}

func CmdBrixSeek(ctx *Context, args []byte) (out []byte, err error) {
	return
}
func CmdBrixNext(ctx *Context, args []byte) (out []byte, err error) {
	return
}
func CmdBrixOver(ctx *Context, args []byte) (out []byte, err error) {
	return
}

func CmdBrixBase(ctx *Context, args []byte) (out []byte, err error) {
	return
}

func CmdBrixKind(ctx *Context, args []byte) (out []byte, err error) {
	return
}

func CmdBrixPack(ctx *Context, args []byte) (out []byte, err error) {
	return
}
