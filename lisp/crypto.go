package main

import "github.com/gritzko/rdx"
import "encoding/hex"

func Hash(ctx *Context, arg []byte) (ret []byte, err error) {
	sha := rdx.Sha256Of(arg)
	hx := make([]byte, rdx.Sha256Bytes*2)
	hex.Encode(hx, sha[:])
	ret = rdx.WriteRDX(nil, rdx.Term, rdx.ID{}, hx)
	return
}
