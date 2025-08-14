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
	//@deprecated
	Handle uint64
	Len    int
	Keys   KeyPair
}

func (branch *Branch) IsWritable() bool {
	return branch.Clock.Src != 0
}

type KeyPair struct {
	Pub ed25519.PublicKey
	Sec ed25519.PrivateKey
}

func (pair *KeyPair) HasPrivateKey() bool {
	return len(pair.Sec) == ed25519.PrivateKeySize
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

var ErrAlreadyOpen = errors.New("the branch is already open")

func (branch *Branch) Join(id ID) (err error) {
	if id.Seq != 0 {
		return ErrNotImplementedYet
	}
	path := TipPath(id.Src)
	var tip Brik
	err = tip.OpenByPath(path)
	if err == nil && len(branch.Tip.Meta) > 0 {
		branch.Brix, err = branch.Brix.OpenByHash(tip.Meta[0])
	}
	_ = tip.Close()
	return
}

func (branch *Branch) Open(id ID) (err error) {
	if len(branch.Brix) != 0 {
		return ErrAlreadyOpen
	}
	if id.Src == 0 && id.Seq != 0 { // notational: branch, not branch-0
		id.Src, id.Seq = id.Seq, 0
	}
	if id.Seq != 0 {
		return ErrNotImplementedYet
	}
	path := TipPath(id.Src)
	err = branch.Tip.OpenByPath(path)
	if err != nil {
		return
	}

	reflen := len(branch.Tip.Meta)
	if reflen > 2 {
		return ErrBadTipFormat
	}
	if reflen > 0 {
		branch.Brix, err = branch.Brix.OpenByHash(branch.Tip.Meta[0])
		if err != nil { // FIXME
			return
		}
	}
	branch.Stage = make(Stage)
	if reflen == 2 {
		var stage Brik
		err = stage.OpenByHash(branch.Tip.Meta[1])
		if err != nil {
			return
		}
		_ = stage.ToStage(branch.Stage)
		_ = stage.Close()
	}
	branch.Len = len(branch.Brix)
	branch.Clock = ID{id.Src, Timestamp()} // FIXME
	return
}

func (branch *Branch) Info() (info Stream, err error) {
	id := ID{branch.Clock.Src, 0}
	return branch.Brix.Get(nil, id)
}

// sealed branch
func (branch *Branch) OpenSealed(id ID, hash Sha256) (err error) {
	if !hash.IsEmpty() {
		branch.Brix, err = branch.Brix.OpenByHash(hash)
	}
	if err == nil {
		branch.Stage = make(Stage)
		branch.Clock = id
	}
	return
}

var ErrBadStagedBrik = errors.New("bad staged brik format")

// non-empty Staged
func (branch *Branch) OpenSaved(id ID, path string) (err error) {
	var staged Brik
	err = staged.OpenByPath(path)
	if err != nil {
		return
	}
	if len(staged.Meta) != 1 {
		_ = staged.Close()
		return ErrBadStagedBrik
	}
	branch.Brix, err = branch.Brix.OpenByHash(staged.Meta[0])
	if err == nil {
		err = staged.ToStage(branch.Stage)
	}
	_ = staged.Close()
	return
}

func (branch *Branch) Tick() ID {
	branch.Clock.Seq = (branch.Clock.Seq & SeqMask) + 64
	return branch.Clock
}

// Adds a record change.
func (branch *Branch) Add(delta Stream) (err error) {
	// FIXME here and in other places: normalize
	it := NewIter(delta)
	if !it.Read() {
		return ErrBadRecord
	}
	id := it.ID()
	base := id.Base()
	pre, found := branch.Stage[base]
	if found {
		inputs := [][]byte{pre, it.Record()}
		var merged Stream
		merged, err = Merge(nil, inputs)
		branch.Stage[base] = merged
	} else {
		branch.Stage[base] = it.Record()
	}
	return
}

func (branch *Branch) Get(id ID) (rec Stream, err error) {
	id.Seq &= SeqMask
	staged, hasChanges := branch.Stage[id.Stem()]
	stored, err := branch.Brix.Get(nil, id)
	if err == nil {
		if hasChanges {
			rec, err = Merge(nil, [][]byte{staged, stored})
		} else {
			rec = stored
		}
	} else if err == ErrRecordNotFound {
		err = nil
		rec = staged
	} else {
		return
	}
	it := NewIter(rec)
	if it.Read() && (it.ID().Seq&63) == 63 {
		rec = nil
	}
	return
}

var ErrNoClock = errors.New("no clock set")

// Put creates a record with the content provided;
// must be one Stream element, preferably PLEX.
func (branch *Branch) Put(elem Stream) (id ID, err error) {
	if branch.Clock.Src == 0 {
		err = ErrNoClock
		return
	}
	id = branch.Tick()
	it := NewIter(elem)
	if !it.Read() {
		err = ErrBadRecord
		return
	}
	rec := WriteRDX(nil, it.Lit(), id, it.Value())
	branch.Stage[id] = rec
	return
}

func (branch *Branch) Set(elem Stream) error {
	return nil
}

// Drops the staged changes
func (branch *Branch) Drop() (err error) {
	branch.Stage = make(Stage)
	return
}

var ErrHasStagedChanges = errors.New("contains staged changes (stash or drop)")
var ErrNoStagedChanges = errors.New("no staged changes")
var ErrNoJoinedChanges = errors.New("no joined changes")
var ErrNoChanges = errors.New("no joined or staged changes")

// Saves the current staged state into a new brik.
// Returns the hash.
func (branch *Branch) Save() (sha Sha256, err error) {
	if len(branch.Stage) == 0 {
		err = ErrNoStagedChanges
		return
	}
	deps := []Sha256{branch.Brix.Hash7574()}
	sha, err = MakeBrik(deps, branch.Stage)
	if err == nil {
		branch.Brix, err = branch.Brix.OpenByHash(sha)
	}
	if err == nil {
		branch.Stage = make(Stage)
	}
	return
}

// FIXME move to Brix
func (branch *Branch) Compact(saveLen int) (sha Sha256, err error) {
	joined := branch.Brix[saveLen:]
	base := Sha256{}
	if len(joined) == 0 {
		err = ErrNoJoinedChanges
		return
	}
	if saveLen > 0 {
		base = branch.Brix[saveLen-1].Hash7574
	}
	var brik Brik
	sha, err = joined.merge(base)
	if err != nil {
		return
	}
	err = brik.OpenByHash(sha)
	if err != nil {
		return
	}
	branch.Brix = branch.Brix[:saveLen]
	branch.Brix = append(branch.Brix, &brik)
	return
}

// Takes any joined changes, merges those into a new brik.
// Saves that brik, replaces the joined briks.
// Returns the hash.
func (branch *Branch) Merge() (sha Sha256, err error) {
	sha, err = branch.Compact(branch.Len)
	branch.Len++
	err = branch.retip([]Sha256{sha})
	return
}

func (branch *Branch) makeTip(deps []Sha256, private Stage) (err error) {
	var tipsha Sha256
	tipsha, err = MakeBrik(deps, private)
	if err != nil {
		return
	}
	hashfn := BrikPath(tipsha)
	handfn := TipPath(branch.Clock.Src)
	err = os.Rename(hashfn, handfn)
	if err == nil {
		_ = branch.Tip.Close()
		err = branch.Tip.OpenByPath(handfn)
	}
	return
}

func (branch *Branch) retip(deps []Sha256) (err error) {
	if branch.Clock.Src == 0 {
		return errors.New("the branch is not writable")
	}
	metaId := ID{branch.Clock.Src, 0}
	tipStage := make(Stage)
	err = branch.Tip.ToStage(tipStage)
	if err != nil {
		return
	}
	edit := X(metaId, P(metaId, R0(branch.Clock)))
	_ = tipStage.Add(edit)
	return branch.makeTip(deps, tipStage)
}

func (branch *Branch) Hash7574() Sha256 {
	if branch.Len == 0 {
		return Sha256{}
	}
	return branch.Brix[branch.Len-1].Hash7574
}

// Merges the joined briks, if any.
// Saves the staged changes if any.
// Makes the tip mention the stash.
// The branch must be writable.
func (branch *Branch) Stash() (err error) {
	old := branch.Hash7574()
	if len(branch.Brix) > branch.Len {
		_, err = branch.Merge()
	}
	if err == nil && len(branch.Stage) > 0 {
		_, err = branch.Save()
	}
	sha := branch.Brix.Hash7574()
	err = branch.retip([]Sha256{old, sha})
	return
}

var ErrNoKeysNotWritable = errors.New("no private key, branch is not writable")

// Merges the joined briks, if any.
// Saves the staged changes if any.
// Signs the brik: (hash, key, signature)
// Moves the tip to point to the resulting version.
// The branch must be writable.
func (branch *Branch) Seal() (err error) {
	if !branch.Keys.HasPrivateKey() {
		return ErrNoKeysNotWritable
	}
	old := branch.Hash7574()
	if len(branch.Brix) > branch.Len {
		_, err = branch.Merge()
	}
	if err == nil && len(branch.Stage) > 0 {
		_, err = branch.Save()
	}
	top := branch.Brix[len(branch.Brix)-1]
	sha := top.Hash7574
	if old.Equal(sha) {
		err = ErrNoChanges
		return
	}

	var file *os.File
	file, err = os.OpenFile(top.File.Name(), os.O_WRONLY|os.O_APPEND, 0o777)
	if err == nil {
		sign := ed25519.Sign(branch.Keys.Sec, top.Hash7574[:])
		err = writeAll(file, top.Hash7574[:], branch.Keys.Pub, sign)
		_ = file.Close()
	}
	if err == nil {
		err = branch.retip([]Sha256{sha})
	}
	if err == nil {
		branch.Len++
	}
	return
}

func (branch *Branch) Fork() (err error) { // todo legend
	//if len(branch.Stage) != 0 {  FIXME check if the changes were saved
	//	return ErrHasStagedChanges
	//}
	branch.Stage = make(Stage)
	err = branch.CloseJoined()
	if err != nil {
		return
	}
	branch.Keys = MakeKeypair()
	branch.Clock.Src = branch.Keys.KeyLet()
	private := make(Stage)
	secHex := hex.EncodeToString(branch.Keys.Sec)
	_ = private.Add(E(ID{branch.Clock.Src, 0}, P0(
		R0(branch.Clock),    //  clocks
		S0("some branch"),   // description
		S(ID{0, 1}, secHex), // private key
	)))
	return branch.makeTip([]Sha256{branch.Brix.Hash7574()}, private)
}

func (branch *Branch) CloseJoined() error {
	joined := branch.Brix[branch.Len:]
	branch.Brix = branch.Brix[:branch.Len]
	return joined.Close()
}

func (branch *Branch) Close() (err error) {
	err = branch.Brix.Close()
	_ = branch.Tip.Close()
	*branch = Branch{}
	return
}

func (branch *Branch) IsOpen() bool {
	return len(branch.Brix) > 0 || !branch.Clock.IsZero()
}

type BranchReader struct {
	branch *Branch
	reader BrixReader
}

func writeAll(file *os.File, data ...[]byte) (err error) {
	for err == nil && len(data) > 0 {
		next := data[0]
		data = data[1:]
		n := 0
		for err == nil && len(next) > 0 {
			next = next[n:]
			n, err = file.Write(next)
		}
	}
	return
}
