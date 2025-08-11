package main

import (
	"encoding/hex"
	"errors"

	"github.com/gritzko/rdx"
)

// make-branch -> s4a35Rlh6N
func CmdMakeBranch(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !repl.space.IsOpen() {
		return nil, ErrNoSpaceOpen
	}
	handle := ""
	if args.Peek() == rdx.Term && args.Read() {
		handle = string(args.Value())
	}
	title := "just a branch"
	if args.Peek() == rdx.String && args.Read() {
		title = string(args.Value())
	}
	recs := make(rdx.Stage)
	if repl.branch.Stage != nil {
		recs, repl.branch.Stage = repl.branch.Stage, recs
	}
	keys := rdx.MakeKeypair()
	if len(handle) == 0 {
		i := keys.KeyLet()
		handle = string(rdx.RON64String(i & rdx.Mask60bit))
	}
	var sha rdx.Sha256
	sha, err = rdx.MakeBranch(handle, title, recs, &keys)
	if err != nil {
		return
	}

	spaceId := rdx.ID{repl.space.Clock.Src, 0}
	branchId := rdx.ID{keys.KeyLet(), 0}
	err = repl.space.Add(
		rdx.E(spaceId,
			rdx.P0(rdx.T0("branches"), rdx.X0(rdx.S(branchId, title))),
		))
	if err != nil {
		return
	}
	_, err = repl.space.Seal()
	if err != nil {
		return
	}

	out = rdx.AppendTerm(out, []byte(hex.EncodeToString(keys.Pub)))
	out = rdx.AppendTerm(out, []byte(hex.EncodeToString(sha[:])))
	return
}

// list-branches
func CmdListBranches(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !repl.space.IsOpen() {
		return nil, ErrNoSpaceOpen
	}

	return
}

// fork -> s4a35Rlh6N
// fork-branch(orig-1234) -> s4a35Rlh6N
func CmdFork(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

// open(space, branch)
// open Branch
// open Branch-234
// open e5f379
func CmdOpen(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var it rdx.Iter
	var id rdx.ID
	it, err = repl.evalArgs(args)
	if err != nil {
		return
	}
	if !it.Read() {
		err = ErrNoArgument
	}
	id, err = pickId(it)
	if err != nil {
		return
	}
	err = repl.space.Open(id)
	if err != nil {
		return
	}
	if it.Read() {
		id, err = pickId(it)
		if err != nil {
			return
		}
		err = repl.branch.Open(id)
	}
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
	var eval rdx.Iter
	eval, err = repl.evalArgs(args)
	if err != nil {
		return
	}
	if !eval.Read() {
		return nil, ErrNoArgument
	}
	var added []byte
	var id rdx.ID
	if rdx.IsPLEX(eval.Lit()) {
		id = eval.ID()
		added = eval.Record()
	} else {
		id, err = pickId(eval)
		if !eval.Read() {
			return nil, ErrNoArgument
		}
		added = rdx.WriteRDX(nil, eval.Lit(), id, eval.Value())
	}
	err = repl.branch.Add(added)
	if err == nil {
		out = rdx.AppendReference(out, id)
	}
	return
}

var ErrNoArgument = errors.New("no argument provided")

// put {key:"value"} -> Alice-4450
func CmdPut(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !args.Read() {
		return nil, ErrNoArgument
	}
	var eval []byte
	if args.Lit() == rdx.Tuple {
		i := rdx.NewIter(args.Value())
		if !i.Read() {
			return nil, ErrNoArgument
		}
		eval, err = repl.Eval(&i)
	} else {
		eval, err = repl.Eval(args)
	}
	var id rdx.ID
	if err == nil {
		id, err = repl.branch.Put(eval)
	}
	if err == nil {
		out = rdx.AppendReference(out, id)
	}
	return
}

// set {@Alice-234 key:"value"} -> Alice-236
func CmdSet(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

func (repl *REPL) evalArgs(args *rdx.Iter) (eval rdx.Iter, err error) {
	if !args.Read() {
		err = ErrNoArgument
		return
	}
	var e []byte
	e, err = repl.Eval(args)
	if err != nil {
		return
	}
	eval = rdx.NewIter(e)
	if !eval.Read() {
		err = ErrNoArgument
	} else if eval.Lit() == rdx.Tuple {
		eval = rdx.NewIter(eval.Value())
	}
	return
}

func (repl *REPL) pickEvalId(args *rdx.Iter) (id rdx.ID, rest rdx.RDX, err error) {
	var eval rdx.Iter
	eval, err = repl.evalArgs(args)
	if err != nil {
		return
	}
	if !eval.Read() {
		err = ErrNoArgument
		return
	}
	if eval.Lit() == rdx.Reference {
		id = eval.Reference()
		rest = eval.Rest()
	} else { // todo
		err = ErrBadArgumentType
	}
	return
}

// get Alice-1230 -> {@Alice-1232 key:"value"}
func CmdGet(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var id rdx.ID
	id, _, err = repl.pickEvalId(args)
	if err == nil {
		out, err = repl.branch.Get(id)
	}
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
