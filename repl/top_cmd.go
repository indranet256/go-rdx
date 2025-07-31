package main

import "github.com/gritzko/rdx"

func CmdExit(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return nil, Errturn
}

func CmdString(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	stack := make(rdx.Marks, 2)
	tmp := rdx.OpenShortTLV(nil, rdx.String, &stack)
	tmp = append(tmp, 0)
	for args.Read() {
		tmp = append(tmp, args.String()...)
	}
	out, err = rdx.CloseTLV(tmp, rdx.String, &stack)
	return
}

func CmdList(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	for args.Read() {
		if rdx.IsPLEX(args.Lit()) {
			out = append(out, args.Value()...)
		} else {
			out = append(out, args.Record()...)
		}
	}
	return
}
