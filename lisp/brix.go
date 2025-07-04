package main

import (
	"github.com/gritzko/rdx"
)

func CmdBrixNew(args, pre []byte) (out []byte, err error) {
	w := rdx.Brik{}
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
	brix, err = brix.OpenByHashlet(string(hashlet))
	if err != nil {
		return
	}
	id, _, args, err = rdx.ReadID(args)
	out, err = brix.Get(nil, id)
	return
}

// brix:add (3c0dce, {@alice-345 5:"five"})
func CmdBrixAdd(args, pre []byte) (out []byte, err error) {
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
