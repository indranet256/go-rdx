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
	w := rdx.BrixWriter{
		//Compress: rdx.CompressLZ4
	}
	err = w.Open()
	prev := rdx.ID{}
	if len(args) == 0 {
		args = pre
	}
	for len(args) > 0 && err == nil {
		_, id, _, rest, _ := rdx.ReadRDX(args)
		if prev.Compare(id) != rdx.Less {
			return nil, rdx.ErrBadOrder
		}
		_, err = w.Write(args[:len(args)-len(rest)])
		args = rest
	}
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
	var brix rdx.BrixReader
	err = brix.OpenByHashlet(string(hashlet))
	if err != nil {
		return
	}
	id, _, args, err = rdx.ReadID(args)
	out, err = brix.Get(nil, id)
	return
}
