package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"os"

	"github.com/gritzko/rdx"
)

var SpaceExt = ".space"

const IdEd25519SecKeySeq = 1152919967043440713
const IdSha256SumSeq = 1150584288716750449
const IdEd25519SignSeq = 1152823100548846199

// branch-seq -> {title:"some commit",sha:ae26b48..., ~:aaa}
// branch-0 -> {title:"somebranch", key:5b93... commits:[], ~:aaa}
// space-0 -> {title:"somespace", type:space, braches:{}, ~:aaa}
// HANDLE.space  space-0 -> {peers:{}...} // local info
// BRANCH.branch branch-0 -> {~~~ed25519:bbb} // author's metainfo

// space-new mybranch "here I try things" -> pubkey
func CmdMakeSpace(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	handle := ""
	if args.Peek() == rdx.Term && args.Read() {
		handle = string(args.Value())
	}
	title := "just a branch"
	if args.Peek() == rdx.String && args.Read() {
		title = string(args.Value())
	}
	var stat os.FileInfo
	stat, err = os.Stat(rdx.BrixPath)
	if err != nil {
		err = os.Mkdir(rdx.BrixPath, 0777)
		if err != nil {
			return
		}
	} else if !stat.IsDir() {
		return nil, rdx.ErrBadFile
	}
	recs := make(rdx.Stage) // todo supply
	var keys rdx.KeyPair
	keys.Pub, keys.Sec, err = ed25519.GenerateKey(nil)
	if err != nil {
		return
	}
	if len(handle) == 0 {
		i := keys.KeyLet()
		handle = string(rdx.RON64String(i & rdx.Mask60bit))
	}
	sha, err := rdx.MakeSpace(handle, title, recs, &keys)
	out = rdx.AppendTerm(out, []byte(hex.EncodeToString(keys.Pub)))
	out = rdx.AppendTerm(out, []byte(hex.EncodeToString(sha[:])))
	return
}

func CmdOpenSpace(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoArgument
	}
	if repl.space.IsOpen() {
		_ = repl.space.Close()
		_ = repl.branch.Close()
	}
	var id rdx.ID
	id, err = pickId(*args)
	if id.Src == 0 {
		id.Src, id.Seq = id.Seq, id.Src
	}
	if err == nil {
		err = repl.space.Open(id)
	}
	return
}

// id (branch, commit...)
func CmdShowSpace(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !repl.space.IsOpen() {
		return nil, ErrNoSpaceOpen
	}
	out, err = repl.space.Info()
	return
}

func CmdSpacePeer(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

func CmdSpacePush(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

func CmdSpacePull(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}
