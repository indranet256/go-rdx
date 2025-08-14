package rdx

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
)

type Space Branch

func MakeSpace(handle uint64, legend string, misc Stage, key *KeyPair) (sha Sha256, err error) {
	return MakeBranch(handle, legend, misc, key, true)
}

// space: < (@bE4Kc2Ofc-23b2 "crypto" "Changes to the yell crypto API" pubkey), ...>
// branch: { (@bE4Kc2Ofc-23bd tag "Ed25519 extended" hash) }
// FIXME make this a member!!!
func MakeBranch(handle uint64, legend string, misc Stage, key *KeyPair, isSpace bool) (sha Sha256, err error) {
	id := ID{key.KeyLet(), 0}
	lnid := ID{handle, 0}
	pubHex := hex.EncodeToString(key.Pub)
	_ = misc.Add(E(id, P0(
		R0(id),              //  clocks
		R0(ID{handle, 0}),   // handle
		S0(legend),          // description
		S(ID{0, 2}, pubHex), // pub key
	)))
	_ = misc.Add(R(lnid, id))
	var tipsha Sha256
	sha, err = MakeBrik([]Sha256{}, misc)
	if err != nil {
		return
	}
	private := make(Stage)
	secHex := hex.EncodeToString(key.Sec)
	_ = private.Add(E(id, P0(
		R0(id),              //  clocks
		R0(ID{handle, 0}),   // handle
		S0(legend),          // description
		S(ID{0, 1}, secHex), // private key
	)))
	_ = private.Add(R(lnid, id))
	tipsha, err = MakeBrik([]Sha256{sha}, private)
	if err != nil {
		return
	}
	hashfn := BrikPath(tipsha)
	handfn := TipPath(handle)
	err = os.Rename(hashfn, handfn)

	return
}

// makes the space writable
func (b *Branch) LoadCreds(handle uint64) (err error) {
	var meta Stream
	metaId := ID{handle, 0}
	meta, err = b.Tip.Get(metaId)
	if err != nil {
		return errors.New("no such space found")
	}
	mit := NewIter(meta)
	if !mit.Read() {
		return mit.Error()
	}
	keylet := handle
	if mit.Lit() == Reference { // it is a handle, not a key prefix
		keylet = mit.Reference().Src
		id0 := ID{keylet, 0}
		meta, err = b.Tip.Get(id0)
		if err != nil {
			return errors.New("space meta record not found for " + string(id0.String()))
		}
		mit = NewIter(meta)
		if !mit.Read() {
			return mit.Error()
		}
	}
	if !IsPLEX(mit.Lit()) {
		return errors.New(string(mit.ID().String()) + " is not a space")
	}
	var self Stream
	self, err = Pick(P0(R0(ID{keylet, 0})), mit.Record())
	if err != nil {
		jdr, _ := WriteAllJDR(nil, mit.Record(), 0)
		fmt.Println(string(jdr))
		return errors.New("space meta self-record not found")
	}
	sit := NewIter(self)
	if sit.Read() && sit.Lit() == Tuple && sit.Into() && sit.Read() && sit.Lit() == Reference {
		clock := sit.Reference()
		b.Clock = ID{keylet, clock.Seq}
		b.Handle = clock.Src
	}
	return
}
