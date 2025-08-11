package main

import "github.com/gritzko/rdx"

func CmdSumInt(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var sum int64
	for args.Read() {
		sum += args.Integer()
	}
	return rdx.I0(sum), nil
}
