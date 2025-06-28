package main

import (
	"errors"
	"fmt"
	"github.com/gritzko/rdx"
)

var ErrBadArguments = errors.New("bad arguments")

func CmdHelp(args, pre []byte) (out []byte, err error) {
	return
}

func CmdLinearID(args, pre []byte) (out []byte, err error) {
	a := rdx.ID{0, 0}
	b := rdx.ID{0, 0xffffffffffffffff}
	n := int64(1)
	if rdx.Peek(args) == rdx.Reference {
		a, _, args, err = rdx.ReadID(args)
	}
	if rdx.Peek(args) == rdx.Reference {
		b, _, args, err = rdx.ReadID(args)
	}
	if rdx.Peek(args) == rdx.Integer {
		n, _, args, err = rdx.ReadInteger(args)
	}
	c := a
	for ; n > 0; n-- {
		c.Seq = rdx.LBetween(c.Seq, b.Seq)
		fmt.Printf("%s\n", c.String())
	}
	return
}

func CmdBrixNew(args, pre []byte) (out []byte, err error) {
	w := rdx.Brix{}
	err = w.Create([]rdx.Sha256{})
	if len(args) == 0 {
		args = pre
	}
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

func CmdBrixGet(args, pre []byte) (out []byte, err error) {
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
	err = brix.OpenByHashlet(string(hashlet))
	if err != nil {
		return
	}
	id, _, args, err = rdx.ReadID(args)
	out, err = brix.Get(nil, id)
	return
}

// brix:add (3c0dce, {@alice-345 5:"five"})
func CmdBrixAdd(args, pre []byte) (out []byte, err error) {
	w := rdx.Brix{}
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
	if len(args) == 0 {
		args = pre
	}
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
