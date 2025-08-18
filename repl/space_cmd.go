package main

import (
	"crypto/ed25519"
	"os"

	"github.com/gritzko/rdx"
)

var SpaceExt = ".space"

const IdEd25519SecKeySeq = 1152919967043440713
const IdSha256SumSeq = 1150584288716750449
const IdEd25519SignSeq = 1152823100548846199

// space: < (@bE4Kc2Ofc-23b2 "crypto" "Changes to the yell crypto API" pubkey), ...>
// branch: { (@bE4Kc2Ofc-23bd "Author B" "Ed25519 extended" hash) }
// make-space(handle "description")
func CmdMakeSpace(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if repl.space.IsOpen() {
		return nil, rdx.ErrAlreadyOpen
	}
	if !args.Read() || (args.Lit() != rdx.Tuple && args.Lit() != rdx.String) {
		return nil, ErrNoArgument
	}
	//_ = repl.branch.Close()
	//_ = repl.space.Close()
	var stat os.FileInfo
	stat, err = os.Stat(rdx.BrixPath)
	if err != nil {
		err = os.Mkdir(rdx.BrixPath, 0755)
		if err != nil {
			return
		}
	} else if !stat.IsDir() {
		return nil, rdx.ErrBadFile
	}
	pub, sec, _ := ed25519.GenerateKey(nil)
	meta := rdx.BranchInfo{
		Title: args.String(),
		Key:   sec,
		Clock: rdx.ID{rdx.KeyLet(pub), rdx.Timestamp()},
	}
	err = repl.space.Fork(&meta)
	if err != nil {
		return
	}
	meta.Key = pub
	record := meta.SaveRDX()
	err = repl.space.Add(record)
	if err != nil {
		return
	}
	err = repl.space.Seal()
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
	id, err = pickID(*args)
	if id.Src == 0 {
		id.Src, id.Seq = id.Seq, id.Src
	}
	if err == nil {
		err = repl.space.Open(id)
	}
	return
}

func CmdListSpace(repl *REPL, args *rdx.Iter) (out []byte, err error) {
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
