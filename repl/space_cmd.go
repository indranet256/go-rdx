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
	if !args.Read() {
		return nil, ErrNoArgument
	}
	var handle uint64
	handle, err = repl.pickHandle(*args)
	if err != nil {
		return
	}
	legend := "some space"
	if args.Read() {
		legend, err = pickString(*args)
		if err != nil {
			return
		}
	}
	//todo args.Rest()
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

	_, err = rdx.MakeSpace(handle, legend, recs, &keys)
	spaceId := rdx.ID{handle, 0}
	if err == nil {
		err = repl.space.Open(spaceId)
	}
	if err == nil {
		err = repl.space.LoadCreds(handle)
		out = rdx.R0(spaceId)
	}

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
	if err == nil {
		err = repl.space.LoadCreds(id.Src)
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
