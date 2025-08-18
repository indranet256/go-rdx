package main

import (
	"crypto/ed25519"
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

func CmdMakeBranch(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !repl.space.IsOpen() {
		return nil, ErrNoSpaceOpen
	}
	pub, sec, _ := ed25519.GenerateKey(nil)
	meta := rdx.BranchInfo{
		Title: "(a branch)",
		Key:   sec,
	}
	if args.Read() {
		meta.Title, err = pickString(*args)
		if err != nil {
			return
		}
	}
	err = repl.branch.Fork(&meta)
	if err != nil {
		return
	}
	id0 := rdx.ID{meta.Clock.Src, 0}
	meta.Key = pub
	pubRecord := meta.SaveRDX()
	err = repl.space.Add(pubRecord)
	if err != nil {
		return
	}
	err = repl.space.Seal()
	if err != nil {
		return
	}
	_ = repl.branch.Close()
	err = repl.branch.Open(id0)

	return
}

// list-branches
func CmdListBranches(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	if !repl.space.IsOpen() {
		return nil, ErrNoSpaceOpen
	}

	return
}

// fork -> s4a35Rlh6N-0
func CmdFork(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	meta := rdx.BranchInfo{Title: "(a branch)"}
	if args.Read() {
		if args.Lit() == rdx.Term || args.Lit() == rdx.String {
			meta.Title = string(args.Value())
		}
	}
	err = repl.branch.Fork(&meta)
	if err == nil && repl.space.IsOpen() {
		meta.Key = meta.PublicKey()
		rec := meta.SaveRDX()
		err = repl.space.Add(rec)
	}
	if err == nil {
		out = rdx.R0(repl.branch.Clock)
	}
	return
}

// open (branch)
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
		return nil, ErrNoArgument
	}
	id, err = pickID(it)
	if err != nil {
		return
	}
	_ = repl.branch.Close()
	id.Seq = 0 // TODO
	err = repl.branch.Open(id)
	return
}

// join Branch
// join Branch-234
// join f2ae63
func CmdJoin(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var it rdx.Iter
	var id rdx.ID
	it, err = repl.evalArgs(args)
	if err != nil {
		return
	}
	if !it.Read() {
		return nil, ErrNoArgument
	}
	id, err = pickID(it)
	if err != nil {
		return
	}
	id.Seq = 0 // FIXME
	err = repl.branch.Join(id)
	return
}

func (repl *REPL) PickNameValue(args *rdx.Iter) (handle rdx.ID, value []byte, err error) {
	if !args.Read() {
		return
	}
	if args.Lit() == rdx.Tuple && args.ID().IsZero() {
		inner := rdx.NewIter(args.Value())
		if !inner.Read() {
			return
		}
		if inner.Lit() == rdx.Term && len(inner.Value()) <= 10 {
			handle, _ = rdx.ParseID(inner.Value())
			inner.Read()
		}
		value, err = repl.Eval(&inner)
		if err == nil && len(inner.Rest()) > 0 {
			err = errors.New("extra arguments provided")
		}
	} else {
		if args.Lit() == rdx.Term && len(args.Value()) <= 10 {
			handle, _ = rdx.ParseID(args.Value())
			if !args.Read() {
				return
			}
		}
		value, err = repl.Eval(args)
	}
	return
}

func CmdTime(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	t := rdx.Timestamp()
	out = rdx.R0(rdx.ID{0, t})
	return
}

func CmdTry(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	handle := rdx.ID{}
	handle, out, err = repl.PickNameValue(args)
	if err != nil {
		if !handle.IsZero() {
			repl.vals[handle] = rdx.S0(err.Error())
		}
		err = nil
	}
	return
}

func CmdStamp(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var handle rdx.ID
	var value rdx.Stream
	handle, value, err = repl.PickNameValue(args)
	if err == nil && handle.Src == 0 {
		handle, err = repl.ResolveHandle(handle)
	}
	if err != nil {
		return
	}
	valit := rdx.NewIter(value)
	if !valit.Read() {
		return nil, ErrNoArgument
	}
	out = rdx.WriteRDX(nil, valit.Lit(), handle, valit.Value())
	return
}

func (repl *REPL) ResolveStream(handle rdx.ID) (solved rdx.Stream, err error) {
	a, found := repl.vals[handle]
	if !found {
		return nil, errors.New("handle unknown: " + string(handle.String()))
	}
	switch a.(type) {
	case []byte:
		return a.([]byte), nil
	case rdx.Stream:
		return a.(rdx.Stream), nil
	default:
		return nil, ErrBadVariableType
	}
}

func (repl *REPL) ResolveHandle(handle rdx.ID) (solved rdx.ID, err error) {
	idxx, found := repl.vals[handle]
	if !found {
		return rdx.ID0, errors.New("handle unknown: " + string(handle.String()))
	}
	idx, ok := idxx.(rdx.Stream)
	if !ok {
		return rdx.ID0, errors.New("handle resolves to a non-RDX value: " + string(handle.String()))
	}
	idit := rdx.NewIter(idx)
	if idit.Read() && len(idit.Rest()) == 0 && idit.Lit() == rdx.Reference {
		solved = idit.Reference()
	} else {
		return rdx.ID0, errors.New("can not resolve handle " + string(handle.String()))
	}
	return
}

func CmdDel(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var handle rdx.ID
	var value rdx.Stream
	handle, value, err = repl.PickNameValue(args)
	if err != nil {
		return
	}
	if handle.Src == 0 && handle.Seq != 0 {
		value, err = repl.ResolveStream(handle)
		if err != nil {
			return
		}
		handle = rdx.ID{}
	}
	if handle.IsZero() {
		valit := rdx.NewIter(value)
		if !valit.Read() {
			return nil, ErrNoArgument
		}
		if !valit.ID().IsZero() {
			handle = valit.ID()
		} else if valit.Lit() == rdx.Reference {
			handle = valit.Reference()
		} else {
			return nil, errors.New("no id argument provided")
		}
	}
	bustId := rdx.ID{handle.Src, handle.Seq | 63}
	rec := rdx.P(bustId)
	err = repl.branch.Add(rec)
	return
}

// add {@Alice-1232 key:"value"}, add(x {...}), add(alice-132, {...})
func CmdAdd(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var handle rdx.ID
	var value rdx.Stream
	handle, value, err = repl.PickNameValue(args)
	if err != nil {
		return
	}
	if len(value) == 0 && !handle.IsZero() { // add(A) where A is {@a-b c d e}
		value, err = repl.ResolveStream(handle)
		if err != nil {
			return
		}
		handle = rdx.ID{}
	}
	valit := rdx.NewIter(value)
	if !valit.Read() {
		return nil, ErrNoArgument
	}
	if handle.IsZero() {
		if valit.ID().IsZero() {
			return nil, errors.New("no id argument provided")
		}
		handle = valit.ID()
	} else {
		if handle.Src == 0 {
			handle, err = repl.ResolveHandle(handle)
			if err != nil {
				return
			}
		}
		if valit.ID().IsZero() {
			value = rdx.WriteRDX(nil, valit.Lit(), handle, valit.Value())
		} else if handle.Compare(valit.ID()) != rdx.Eq {
			return nil, errors.New("conflicting ids")
		}
	}
	err = repl.branch.Add(value)
	if err == nil {
		out = rdx.AppendReference(out, handle)
	}
	return
}

var ErrNoArgument = errors.New("no argument provided")

// put {key:"value"} -> Alice-4450
func CmdPut(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var handle rdx.ID
	var value rdx.Stream
	handle, value, err = repl.PickNameValue(args)
	if len(value) == 0 {
		value, err = repl.ResolveStream(handle)
		if err != nil {
			return
		} else {
			handle = rdx.ID{}
		}
	}
	var id rdx.ID
	id, err = repl.branch.Put(value)
	if err == nil {
		if !handle.IsZero() {
			repl.vals[handle] = id
		}
		out = rdx.AppendReference(out, id)
	}
	return
}

// set {@Alice-234 key:"value"} -> Alice-236
func CmdSet(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var handle rdx.ID
	var value rdx.Stream
	handle, value, err = repl.PickNameValue(args)
	if err != nil {
		return
	}
	if handle.Src == 0 {
		handle, err = repl.ResolveHandle(handle)
		if err != nil {
			return
		}
	}
	valit := rdx.NewIter(value)
	if !valit.Read() {
		return nil, ErrNoArgument
	}
	if handle.IsZero() {
		if valit.ID().IsZero() {
			return nil, errors.New("no id argument provided")
		}
		handle = valit.ID()
	}
	var pre rdx.Stream
	pre, err = repl.branch.Get(handle)
	if err != nil {
		return
	}
	pit := rdx.NewIter(pre)
	pit.Read()
	newid := pit.ID()
	rev := newid.Seq & 63
	if rev == 63 {
		return nil, errors.New("revision limit exceeded")
	}
	newid.Seq = (newid.Seq & ^uint64(63)) | ((rev &^ uint64(1)) + 2)
	value = rdx.WriteRDX(nil, valit.Lit(), newid, valit.Value())
	err = repl.branch.Add(value)
	if err == nil {
		out = rdx.AppendReference(out, handle)
	}
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
		eval = rdx.NewIter(eval.Value()) // fixme mistake
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
	if id.Src == 0 {
		local, has := repl.vals[id]
		if has {
			r, ok := local.(rdx.Stream)
			if ok {
				rit := rdx.NewIter(r)
				if rit.Read() && rit.Lit() == rdx.Reference {
					id = rit.Reference()
				}
			}
		}
	}
	if err == nil {
		out, err = repl.branch.Get(id)
	}
	return
}

// Drops any staged or joined changes
func CmdDrop(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	// FIXME erase from the tip
	repl.branch.Stage = make(rdx.Stage)
	return
}

// commit -> branch-345
// save -> f2ae63
func CmdSeal(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	comment := "(a commit)"
	err = repl.branch.Seal()
	hash := repl.branch.Brix.Hash7574()
	if err == nil && repl.space.IsOpen() {
		meta := rdx.BranchInfo{
			Title: comment,
			Key:   hash.Bytes(),
		}
		rec := meta.SaveRDX()
		err = repl.space.Add(rec)
		if err == nil {
			err = repl.space.Seal()
		}
	}
	if err == nil {
		out = rdx.R0(repl.branch.Clock)
	}
	return
}

// Saves the staged changes
func CmdSave(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	err = repl.branch.Stash()
	if err == nil {
		out = rdx.S0(repl.branch.Brix.Hash7574().String())
	}
	return
}

// Merges the joined changes
func CmdMerge(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	var sha rdx.Sha256
	sha, err = repl.branch.Merge()
	if err == nil {
		out = rdx.S0(sha.String())
	}
	return
}

func CmdStash(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	err = repl.branch.Stash()
	if err == nil {
		out = rdx.S0(repl.branch.Brix.Hash7574().String())
	}
	return
}

// Seals the joined and staged changes.
// Adds a workspace commit record to reflect that.
func CmdCommit(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	return
}

func CmdListBranch(repl *REPL, args *rdx.Iter) (out []byte, err error) {
	/*vid := rdx.ID{0, rand.Uint64() & rdx.Mask60bit}
	id := rdx.ID{repl.space.Clock.Src, rdx.IdCommitList}
	var list rdx.Stream
	list, err = repl.space.Get(id)
	if err == nil {
		var it rdx.Iter
		it = rdx.NewIter(list)
		repl.vals[vid] = &it
		out = rdx.R0(vid)
	}*/
	return
}
