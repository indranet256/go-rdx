package rdx

import (
	"crypto/ed25519"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"os"
)

type Branch struct {
	Brix   Brix
	Clock  ID
	Tip    Brik
	Stage  Stage
	Handle uint64
}

func (b *Branch) IsWritable() bool {
	return b.Clock.Src != 0
}

type KeyPair struct {
	Pub ed25519.PublicKey
	Sec ed25519.PrivateKey
}

func MakeKeypair() (keys KeyPair) {
	keys.Pub, keys.Sec, _ = ed25519.GenerateKey(nil)
	return
}

func (pair *KeyPair) PubRDX() Stream {
	return P0(S0(hex.EncodeToString(pair.Pub)))
}

func (pair *KeyPair) RDX() Stream {
	return P0(S0(hex.EncodeToString(pair.Pub)), S0(hex.EncodeToString(pair.Sec)))
}

func (pair *KeyPair) KeyLet() uint64 {
	return binary.LittleEndian.Uint64(pair.Pub) & Mask60bit
}

var ID0 = ID{}

func TipPath(src uint64) string {
	return BrixPath + string(RON64String(src)) + BrixFileExt
}

var ErrNotImplementedYet = errors.New("not implemented yet")
var ErrBadTipFormat = errors.New("bad branch tip format")
var ClockID = ID{0, 667105775}

func pickClock(src uint64, clock Stream) (c ID) {
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

func (b *Branch) Info() (info Stream, err error) {
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
func (b *Branch) Add(delta Stream) (err error) {
	// FIXME here and in other places: normalize
	it := NewIter(delta)
	if !it.Read() {
		return ErrBadRecord
	}
	id := it.ID()
	base := id.Base()
	pre, found := b.Stage[base]
	if found {
		inputs := [][]byte{pre, it.Record()}
		var merged Stream
		merged, err = Merge(nil, inputs)
		b.Stage[base] = merged
	} else {
		b.Stage[base] = it.Record()
	}
	return
}

func (b *Branch) Get(id ID) (rec Stream, err error) {
	id.Seq &= SeqMask
	stage, _ := b.Stage[id.Base()]
	// todo merge
	it := NewIter(stage)
	if it.Read() && (it.ID().Seq&63) == 63 {
		stage = nil
	}
	return stage, nil
}

var ErrNoClock = errors.New("no clock set")

// Put creates a record with the content provided;
// must be one Stream element, preferably PLEX.
func (b *Branch) Put(elem Stream) (id ID, err error) {
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

func (b *Branch) Set(elem Stream) error {
	return nil
}

// Saves the current staged state
func (b *Branch) Stash() (err error) {
	if len(b.Stage) == 0 {
		return errors.New("no new data")
	}
	deps := []Sha256{b.Brix.Hash7574()}
	var hash Sha256
	hash, err = MakeBrik(deps, b.Stage)
	if err == nil {
		b.Brix, err = b.Brix.OpenByHash(hash)
	}
	if b.IsWritable() {
		// todo change the tip
	}
	return
}

// Commits the staged part
func (b *Branch) Seal() (sha Sha256, err error) {
	if b.Clock.Src == 0 {
		return Sha256{}, errors.New("the branch is not writable")
	}
	b.Clock.Seq++ // todo
	if len(b.Stage) == 0 {
		err = errors.New("no staged changes")
		return
	}
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
	// todo clock
	metaId := ID{b.Clock.Src, 0}
	var meta Stream
	meta, err = b.Tip.Get(metaId)
	if err != nil || !IsPLEX(Peek(meta)) {
		err = errors.New("the meta record is missing")
		return
	}
	var edited Stream
	edit := X(metaId, P(metaId, R0(b.Clock)))
	edited, err = Merge(nil, [][]byte{meta, edit})
	tipdeps := []Sha256{sha}
	_ = tipStage.Add(edited)
	tipSha, err = MakeBrik(tipdeps, tipStage)
	if err != nil {
		return
	}
	err = os.Rename(BrikPath(tipSha), TipPath(b.Handle))
	if err != nil {
		return
	}
	b.Brix, err = b.Brix.OpenByHash(sha)
	if err != nil {
		return
	}
	_ = b.Tip.Close()
	err = b.Tip.OpenByPath(TipPath(b.Handle))
	// TODO sign it
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
