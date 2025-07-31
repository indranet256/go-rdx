package main

import "github.com/gritzko/rdx"
import "encoding/hex"

func CmdCryptoHash(ctx *REPL, arg *rdx.Iter) (ret []byte, err error) {
	if !arg.Read() {
		return nil, ErrBadArguments
	}
	subj := arg.Record()
	if arg.Lit() == rdx.Tuple {
		subj = arg.Value()
	}
	sha := rdx.Sha256Of(subj)
	hx := make([]byte, rdx.Sha256Bytes*2)
	hex.Encode(hx, sha[:])
	ret = rdx.WriteRDX(nil, rdx.Term, rdx.ID{}, hx)
	return
}
