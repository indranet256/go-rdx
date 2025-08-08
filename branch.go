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
	Local Brik
	Stage Stage
}

type KeyPair struct {
	Pub ed25519.PublicKey
	Sec ed25519.PrivateKey
}

func (pair *KeyPair) KeyLet() uint64 {
	return binary.LittleEndian.Uint64(pair.Pub) & Mask60bit
}

var ID0 = ID{}

func MakeBranch(handle, title string,
	recs Stage, keys *KeyPair) (sha Sha256, err error) {
	id := ID{keys.KeyLet(), 0}
	pub := hex.EncodeToString(keys.Pub)
	err = recs.Add(MakeEulerOf(id, []RDX{
		MakeTuple(ID0, MakeTerm("ed25519pub").AppendString(pub)),
		MakeTuple(ID0, MakeTerm("id").AppendString(handle)),
		MakeTuple(ID0, MakeTerm("title").AppendString(title)),
	}))
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
	err = private.Add(MakeEulerOf(id, []RDX{
		MakeTuple(ID0, MakeTerm("ed25519sec").AppendString(sec)),
	}))
	tipsha, err = MakeBrik([]Sha256{sha}, private)
	if err != nil {
		return
	}
	hashfn := BrixPath + tipsha.String() + BrixFileExt
	handfn := BrixPath + handle + BrixFileExt
	err = os.Rename(hashfn, handfn)
	return
}

var ErrNotImplementedYet = errors.New("not implemented yet")

// read-only branch
func (b *Branch) Open(id ID) (err error) {
	if id.Seq != 0 {
		return ErrNotImplementedYet
	}
	path := BrixPath + string(RON64String(id.Src)) + BrixFileExt
	b.Brix, err = b.Brix.OpenByPath(path)
	if err == nil {
		b.Clock = id // TODO recover
	}
	b.Stage = make(Stage)
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

func (b *Branch) Get(id ID) RDX {
	id.Seq &= SeqMask
	stage, _ := b.Stage[id]
	return stage
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
func (b *Branch) Save(filename string) (err error) {
	return nil
}

// Commits the staged part
func (b *Branch) Seal() (err error) {
	return nil
}

func (b *Branch) Compact(newHeight int) (err error) {
	return nil
}

func (b *Branch) Close() error {
	b.Clock = ID{}
	_ = b.Local.Close()
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
