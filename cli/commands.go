package main

import (
	"fmt"
	"github.com/gritzko/rdx"
)

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
