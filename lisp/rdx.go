package main

import (
	"fmt"
	"github.com/gritzko/rdx"
)

func CmdIDInts(ctx *Context, args []byte) (out []byte, err error) {
	stack := rdx.Marks{}
	for rdx.Peek(args) == rdx.Reference {
		var a rdx.ID
		a, _, args, err = rdx.ReadID(args)
		if err != nil {
			break
		}
		out = rdx.OpenShortTLV(out, rdx.Tuple, &stack)
		out = append(out, 0)
		out = rdx.AppendInteger(out, int64(a.Src))
		out = rdx.AppendInteger(out, int64(a.Seq))
		out, _ = rdx.CloseTLV(out, rdx.Tuple, &stack)
	}
	return
}

func CmdMerge(ctx *Context, args []byte) (out []byte, err error) {
	inputs := make([][]byte, 0, 10)
	for len(args) > 0 {
		_, _, _, rest, e := rdx.ReadRDX(args)
		if e != nil {
			err = e
			return
		}
		inputs = append(inputs, args[:len(args)-len(rest)])
		args = rest
	}
	out, err = rdx.Merge(nil, inputs)
	return
}

func CmdFitID(ctx *Context, args []byte) (out []byte, err error) {
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

func CmdNormalize(ctx *Context, args []byte) (out []byte, err error) {
	return rdx.Normalize(args)
}
