package rdx

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
)

type Branch struct {
	Brix  Brix
	Clock ID
	Tip   Brik
	Stage Stage
}

type KeyPair struct {
	Pub ed25519.PublicKey
	Sec ed25519.PrivateKey
}

func MakeKeypair() (keys KeyPair) {
	keys.Pub, keys.Sec, _ = ed25519.GenerateKey(nil)
	return
}

func (pair *KeyPair) KeyLet() uint64 {
	return binary.LittleEndian.Uint64(pair.Pub) & Mask60bit
}

var ID0 = ID{}

func MakeBranch(handle, title string,
	recs Stage, keys *KeyPair) (sha Sha256, err error) {
	id := ID{keys.KeyLet(), 0}
	pub := hex.EncodeToString(keys.Pub)
	err = recs.Add(E(id,
		P0(T0("ed25519pub"), S0(pub)),
		P0(T0("id"), S0(handle)),
		P0(T0("title"), S0(title)),
		// todo clock type, other things
	))
	if err != nil {
		return
	}
	var tipsha Sha256
	sha, err = MakeBrik([]Sha256{}, recs)
	if err != nil {
		return
	}
	private := make(Stage)
	sec := hex.EncodeToString(keys.Sec)
	err = private.Add(E(id,
		P0(T0("ed25519sec"), S0(sec)),
	))
	tipsha, err = MakeBrik([]Sha256{sha}, private)
	if err != nil {
		return
	}
	hashfn := BrikPath(tipsha)
	handfn := TipPath(keys.KeyLet())
	err = os.Rename(hashfn, handfn)
	return
}

func TipPath(src uint64) string {
	return BrixPath + string(RON64String(src)) + BrixFileExt
}

var ErrNotImplementedYet = errors.New("not implemented yet")
var ErrBadTipFormat = errors.New("bad branch tip format")
var ClockID = ID{0, 667105775}

func pickClock(src uint64, clock RDX) (c ID) {
	cit := NewIter(clock)
	if cit.Read() && cit.Lit() == Multix {
		cint := NewIter((cit.Value()))
		for cint.Read() && cint.ID().Src != src {
		}
		if cint.HasData() {
			c = cint.ID()
		}
	}
	return
}

// read-only branch
func (b *Branch) Open(id ID) (err error) {
	if id.Src == 0 && id.Seq != 0 { // notational: branch, not branch-0
		id.Src, id.Seq = id.Seq, 0
	}
	if id.Seq != 0 {
		return ErrNotImplementedYet
	}
	path := TipPath(id.Src)
	err = b.Tip.OpenByPath(path)
	if err != nil {
		return
	}
	var clock RDX
	clock, _ = b.Tip.Get(ClockID)
	if len(clock) > 0 {
		b.Clock = pickClock(id.Src, clock)
	}
	reflen := len(b.Tip.Meta)
	if reflen > 2 {
		return ErrBadTipFormat
	}
	if reflen > 0 {
		b.Brix, err = b.Brix.OpenByHash(b.Tip.Meta[0])
		if err != nil { // FIXME
			return
		}
	}
	b.Stage = make(Stage)
	if reflen == 2 {
		var stage Brik
		err = stage.OpenByHash(b.Tip.Meta[1])
		if err != nil {
			return
		}
		_ = stage.ToStage(b.Stage)
		_ = stage.Close()
	}
	return
}

func (b *Branch) Info() (info RDX, err error) {
	id := ID{b.Clock.Src, 0}
	return b.Brix.Get(nil, id)
}

// sealed branch
func (b *Branch) OpenSealed(id ID, hash Sha256) (err error) {
	if !hash.IsEmpty() {
		b.Brix, err = b.Brix.OpenByHash(hash)
	}
	if err == nil {
		b.Stage = make(Stage)
		b.Clock = id
	}
	return
}

var ErrBadStagedBrik = errors.New("bad staged brik format")

// non-empty Staged
func (b *Branch) OpenSaved(id ID, path string) (err error) {
	var staged Brik
	err = staged.OpenByPath(path)
	if err != nil {
		return
	}
	if len(staged.Meta) != 1 {
		_ = staged.Close()
		return ErrBadStagedBrik
	}
	b.Brix, err = b.Brix.OpenByHash(staged.Meta[0])
	if err == nil {
		err = staged.ToStage(b.Stage)
	}
	_ = staged.Close()
	return
}

func (b *Branch) Tick() ID {
	b.Clock.Seq = (b.Clock.Seq & SeqMask) + 64
	return b.Clock
}

// Adds a record change.
func (b *Branch) Add(delta RDX) (err error) {
	// FIXME here and in other places: normalize
	it := NewIter(delta)
	if !it.Read() {
		return ErrBadRecord
	}
	id := it.ID()
	pre, found := b.Stage[id]
	if found {
		inputs := [][]byte{pre, it.Record()}
		var merged RDX
		merged, err = Merge(nil, inputs)
		b.Stage[id] = merged
	} else {
		b.Stage[id] = it.Record()
	}
	return
}

func (b *Branch) Get(id ID) (rec RDX, err error) {
	id.Seq &= SeqMask
	stage, _ := b.Stage[id]
	return stage, nil
}

var ErrNoClock = errors.New("no clock set")

// Put creates a record with the content provided;
// must be one RDX element, preferably PLEX.
func (b *Branch) Put(elem RDX) (id ID, err error) {
	if b.Clock.Src == 0 {
		err = ErrNoClock
		return
	}
	id = b.Tick()
	it := NewIter(elem)
	if !it.Read() {
		err = ErrBadRecord
		return
	}
	rec := WriteRDX(nil, it.Lit(), id, it.Value())
	b.Stage[id] = rec
	return
}

func (b *Branch) Set(elem RDX) error {
	return nil
}

// Saves the current staged state
func (b *Branch) Stash(filename string) (err error) {
	return nil
}

// Commits the staged part
func (b *Branch) Seal() (sha Sha256, err error) {
	tipStage := make(Stage)
	err = b.Tip.ToStage(tipStage)
	if err != nil {
		return
	}
	var tipSha Sha256
	deps := []Sha256{b.Brix.Hash7574()}
	sha, err = MakeBrik(deps, b.Stage)
	if err != nil {
		return
	}
	// fixme clock
	tipdeps := []Sha256{sha}
	_ = tipStage.Add(X(ClockID, T(b.Clock, "sealed")))
	tipSha, err = MakeBrik(tipdeps, tipStage)
	if err != nil {
		return
	}
	err = os.Rename(BrikPath(tipSha), TipPath(b.Clock.Src))
	if err != nil {
		return
	}
	b.Brix, err = b.Brix.OpenByHash(sha)
	if err != nil {
		return
	}
	_ = b.Tip.Close()
	err = b.Tip.OpenByHash(tipSha)
	return
}

func (b *Branch) Compact(newHeight int) (err error) {
	return nil
}

func (b *Branch) Close() error {
	b.Clock = ID{}
	_ = b.Tip.Close()
	b.Stage = nil
	return b.Brix.Close()
}

func (b *Branch) IsOpen() bool {
	return len(b.Brix) > 0 || !b.Clock.IsZero()
}

type BranchReader struct {
	branch *Branch
	reader BrixReader
}
