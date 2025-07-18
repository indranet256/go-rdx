package main

import "github.com/gritzko/rdx"

func CmdRdxDiffHili(ctx *Context, args []byte) (out []byte, err error) {
	var one, two, rest, rest2 []byte
	_, _, _, rest, err = rdx.ReadRDX(args)
	if err != nil {
		return
	}
	one = args[:len(args)-len(rest)]
	_, _, _, rest2, err = rdx.ReadRDX(rest)
	if err != nil {
		return
	}
	two = args[len(args)-len(rest) : len(args)-len(rest2)]
	diff := rdx.Diff{
		Old: one,
		Neu: two,
	}
	err = diff.Solve()
	if err != nil {
		return
	}
	return diff.Hili()
}

func CmdRdxDiff(ctx *Context, args []byte) (out []byte, err error) {
	var one, two, rest, rest2 []byte
	_, _, _, rest, err = rdx.ReadRDX(args)
	if err != nil {
		return
	}
	one = args[:len(args)-len(rest)]
	_, _, _, rest2, err = rdx.ReadRDX(rest)
	if err != nil {
		return
	}
	two = args[len(args)-len(rest) : len(args)-len(rest2)]
	flat, _ := rdx.Flatten(nil, one)
	diff := rdx.Diff{
		Orig: one,
		Old:  flat,
		Neu:  two,
	}
	err = diff.Solve()
	if err != nil {
		return
	}
	return diff.Diff()
}
