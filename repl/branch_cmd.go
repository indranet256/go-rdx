package main

import (
	"encoding/hex"
	"errors"

	"github.com/gritzko/rdx"
)

func (repl *REPL) pickHandle(args rdx.Iter) (handle uint64, err error) {
	var id rdx.ID
	id, err = pickID(args)
	if err != nil || id.Src != 0 || id.Seq == 0 {
		err = ErrBadArguments
	} else {
		handle = id.Seq
	}
	return
}

// space: < (@bE4Kc2Ofc-23b2 crypto "Changes to the yell crypto API" pubkey), ...>
// branch: { (@bE4Kc2Ofc-23bd tag "Ed25519 extended" hash) }
// make-branch(handle mission)
func CmdMakeBranch(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !repl.space.IsOpen() {
		return nil, ErrNoSpaceOpen
	}
	if !args.Read() {
		return nil, ErrNoArgument
	}
	var handle uint64
	handle, err = repl.pickHandle(*args)
	if err != nil {
		return
	}
	legend := "some branch"
	if args.Read() {
		legend, err = pickString(*args)
		if err != nil {
			return
		}
	}
	recs := make(rdx.Stage)
	if repl.branch.Stage != nil {
		recs, repl.branch.Stage = repl.branch.Stage, recs
	}
	keys := rdx.MakeKeypair()
	/*if len(handle) == 0 {
		i := keys.KeyLet()
		handle = string(rdx.RON64String(i & rdx.Mask60bit))
	}*/
	_, err = rdx.MakeBranch(handle, legend, recs, &keys, false)
	if err != nil {
		return
	}

	spaceId := rdx.ID{repl.space.Clock.Src, 0}
	branchId := rdx.ID{Src: keys.KeyLet()}
	err = repl.space.Add(
		rdx.X(spaceId,
			rdx.P(branchId,
				rdx.R0(rdx.ID{handle, 0}), rdx.S0(legend), rdx.S0(hex.EncodeToString(keys.Pub)),
			),
		))
	if err == nil {
		_, err = repl.space.Seal()
	}
	if err != nil {
		return
	}

	out = rdx.R0(branchId)
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
	id, err = pickID(it)
	if err != nil {
		return
	}
	err = repl.space.Open(id)
	if err != nil {
		return
	}
	if it.Read() {
		id, err = pickID(it)
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
	var added []byte
	var id rdx.ID
	if rdx.IsPLEX(eval.Lit()) {
		id = eval.ID()
		added = eval.Record()
	} else {
		id, err = pickID(eval)
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

// todo is it OK that the returned iterator is positioned?
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

func (repl *REPL) pickIdEval(args *rdx.Iter) (id rdx.ID, rest rdx.Stream, err error) {
	if !args.Read() {
		err = ErrNoArgument
		return
	}
	if args.Lit() != rdx.Tuple {
		rest, err = repl.Eval(args)
		return
	}
	it := rdx.NewIter(args.Value())
	{
		t := it
		t.Read()
		i, e := pickID(t)
		if e == nil && repl.cmds[i] == nil && len(t.Rest()) > 0 {
			id = i
			it = t
		}
	}

	rest, err = repl.evaluate(it.Rest())

	return
}

func (repl *REPL) pickEvalId(args *rdx.Iter) (id rdx.ID, rest rdx.Stream, err error) {
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
	if !args.Read() {
		return nil, ErrNoArgument
	}
	var id rdx.ID
	id, err = pickID(*args)
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

func CmdStash(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	err = repl.branch.Stash()
	if err == nil {
		out = rdx.S0(repl.branch.Brix.Hash7574().String())
	}
	return
}
