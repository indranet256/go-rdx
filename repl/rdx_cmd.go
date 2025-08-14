package main

import "github.com/gritzko/rdx"

func CmdFlatRDX(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return rdx.Flatten(nil, args.Rest())
}

func CmdNormalRDX(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return rdx.Normalize(args.Rest())
}

func CmdMergeRDX(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	inputs := [][]byte{}
	for args.Read() {
		inputs = append(inputs, args.Record())
	}
	out, err = rdx.Merge(nil, inputs)
	return
}

func CmdBareRDX(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if args.Read() {
		out = args.Value()
	}
	return
}

func CmdDiffRDX(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrBadArguments
	}
	v1 := args.Record()
	if !args.Read() {
		return nil, ErrBadArguments
	}
	v2 := args.Record()
	flat, _ := rdx.Flatten(nil, v1)
	diff := rdx.Diff{
		Orig: v1,
		Old:  flat,
		Neu:  v2,
	}
	err = diff.Solve()
	if err != nil {
		return
	}
	return diff.Diff()
}
