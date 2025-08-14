package main

import "github.com/gritzko/rdx"

func CmdFlatRDX(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return rdx.Flatten(nil, args.Rest())
}

func CmdNormalRDX(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return rdx.Normalize(args.Rest())
}

func CmdBareRDX(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if args.Read() {
		out = args.Value()
	}
	return
}
