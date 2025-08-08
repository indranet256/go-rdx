package main

import (
	"errors"

	"github.com/gritzko/rdx"
)

// make-branch -> s4a35Rlh6N
func CmdMakeBranch(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if repl.spaceId.IsZero() {
		return nil, ErrNoSpaceOpen
	}

	return
}

// list-branches
func CmdListBranches(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if repl.spaceId.IsZero() {
		return nil, ErrNoSpaceOpen
	}

	return
}

// fork -> s4a35Rlh6N
// fork-branch(orig-1234) -> s4a35Rlh6N
func CmdFork(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

// open-branch Branch
// open-branch Branch-234
// open-branch e5f379
func CmdOpen(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

// join Branch
// join Branch-234
// join f2ae63
func CmdJoin(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

// add {@Alice-1232 key:"value"}
func CmdAdd(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

var ErrNoArgument = errors.New("no argument provided")

// put {key:"value"} -> Alice-4450
func CmdPut(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoArgument
	}
	var id rdx.ID
	id, err = repl.branch.Put(args.Record())
	if err == nil {
		out = rdx.AppendReference(out, id)
	}
	return
}

// set {@Alice-234 key:"value"} -> Alice-236
func CmdSet(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

// get Alice-1230 -> {@Alice-1232 key:"value"}
func CmdGet(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

// rollback
// back
func CmdRollback(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

// commit -> branch-345
// save -> f2ae63
func CmdSeal(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

func CmdSave(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}
