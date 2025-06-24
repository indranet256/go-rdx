package main

import (
	"fmt"
	"github.com/gritzko/rdx"
	"math/bits"
)

func TopBit(v uint64) uint64 {
	l := bits.LeadingZeros64(v)
	return uint64(1) << (63 - l)
}

func LBetween(a, b uint64) (ret uint64) {
	mask := ^uint64(0x3f)
	aa := bits.ReverseBytes64(a & mask)
	bb := bits.ReverseBytes64(b & mask)
	if bb > aa {
		d := bb - aa
		dtop := TopBit(d)
		if dtop >= 64 {
			ret = aa + (dtop >> 6)
		} else {
			d = TopBit(aa) >> 6
		}
	} else {
		panic("todo")
	}
	ret = bits.ReverseBytes64(ret)
	return
}

func CmdHelp(args, pre []byte) (out []byte, err error) {
	return
}

func CmdLinearID(args, pre []byte) (out []byte, err error) {
	var a, b rdx.ID
	a, _, args, err = rdx.ReadID(args)
	if err != nil {
		return
	}
	b, _, args, err = rdx.ReadID(args)
	if err != nil {
		return
	}
	seq := LBetween(a.Seq, b.Seq)
	cc := rdx.ID{0, seq}
	fmt.Printf("%s\n", cc.String())
	return
}
