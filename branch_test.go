package rdx

import (
	"crypto/ed25519"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBranchInfoSerialization(t *testing.T) {
	// Create test keypair
	pub, sec, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate keypair: %v", err)
	}

	// Create original BranchInfo
	src := KeyLet(pub)
	original := BranchInfo{
		Clock: ID{src, Timestamp()},
		Title: "Branch for small experiments",
		Key:   sec, // private key
		Head:  ID{src, Timestamp() + 64},
	}

	// Test SaveRDX
	record := original.SaveRDX()
	if len(record) == 0 {
		t.Fatal("SaveRDX returned empty record")
	}

	// Verify record structure by parsing manually
	it := NewIter(record)
	if !it.Read() || it.Lit() != Euler {
		t.Fatal("Record should be an Euler (PLEX) type")
	}

	// Verify ID matches keylet-0 pattern
	expectedID := ID{KeyLet(pub), 0}
	if it.ID() != expectedID {
		t.Errorf("Expected ID %v, got %v", expectedID, it.ID())
	}

	// Test LoadRDX
	loaded := BranchInfo{}
	err = loaded.LoadRDX(record)
	if err != nil {
		t.Fatalf("LoadRDX failed: %v", err)
	}

	// Verify all fields match
	if loaded.Clock != original.Clock {
		t.Errorf("Clock mismatch: expected %v, got %v", original.Clock, loaded.Clock)
	}

	if loaded.Title != original.Title {
		t.Errorf("Title mismatch: expected %q, got %q", original.Title, loaded.Title)
	}

	if loaded.Head != original.Head {
		t.Errorf("Head mismatch: expected %v, got %v", original.Head, loaded.Head)
	}

	// Note: LoadRDX only loads public key, not private key
	if len(loaded.Key) != ed25519.PublicKeySize {
		t.Errorf("Expected public key size %d, got %d", ed25519.PublicKeySize, len(loaded.Key))
	}

	expectedPubKey := pub
	if string(loaded.Key) != string(expectedPubKey) {
		t.Errorf("Public key mismatch")
	}

	// Test round-trip with public key only
	pubOnlyInfo := BranchInfo{
		Clock: original.Clock,
		Title: original.Title,
		Key:   pub, // public key only
		Head:  original.Head,
	}

	record2 := pubOnlyInfo.SaveRDX()
	loaded2 := BranchInfo{}
	err = loaded2.LoadRDX(record2)
	if err != nil {
		t.Fatalf("Second LoadRDX failed: %v", err)
	}

	if loaded2.Clock != pubOnlyInfo.Clock {
		t.Errorf("Second round: Clock mismatch")
	}
	if loaded2.Title != pubOnlyInfo.Title {
		t.Errorf("Second round: Title mismatch")
	}
	if loaded2.Head != pubOnlyInfo.Head {
		t.Errorf("Second round: Head mismatch")
	}
	if string(loaded2.Key) != string(pub) {
		t.Errorf("Second round: Key mismatch")
	}
}

func TestBranchInfoSerializationEdgeCases(t *testing.T) {
	// Test with special characters Value title
	_, sec, _ := ed25519.GenerateKey(nil)
	special := BranchInfo{
		Clock: ID{KeyLet(sec), 123},
		Title: "Branch with \"quotes\" and 🚀 unicode",
		Key:   sec,
		Head:  ID{KeyLet(sec), 456},
	}

	record := special.SaveRDX()
	var loaded BranchInfo
	err := loaded.LoadRDX(record)
	if err != nil {
		t.Fatalf("Loading special title failed: %v", err)
	}

	if loaded.Title != special.Title {
		t.Errorf("Special title mismatch: expected %q, got %q", special.Title, loaded.Title)
	}
}

func TestBranchInfoInvalidData(t *testing.T) {
	// Test loading invalid ed25519 key
	invalidKeyInfo := BranchInfo{
		Clock: ID{123, 456},
		Title: "Test",
		Key:   []byte("invalid-key-too-short"),
		Head:  ID{123, 789},
	}

	record := invalidKeyInfo.SaveRDX()

	loaded := BranchInfo{}
	// Should handle gracefully - the actual error handling depends on hex.DecodeString
	err := loaded.LoadRDX(record)
	assert.NotNil(t, err)

	// Test loading malformed record
	malformed := []byte{0x01, 0x02, 0x03} // Not valid RDX
	err = loaded.LoadRDX(malformed)
	if err == nil {
		t.Error("Expected error when loading malformed record")
	}
}

func TestObliterate(t *testing.T) {
	object := E0(
		P0(S0("int"), I0(1)),
		P0(S0("ref"), R0(ID{12, 34})),
		P0(S0("str"), S0("abc")),
	)
	oread, err := NewObjectReader(object)
	assert.NotNil(t, oread)
	assert.Nil(t, err)
	count := 0
	for oread.Read() {
		switch oread.Key {
		case "int":
			assert.Equal(t, oread.Value.Integer(), int64(1))
			count++
		case "ref":
			assert.Equal(t, oread.Value.Reference(), ID{12, 34})
			count++
		case "str":
			assert.Equal(t, oread.Value.String(), "abc")
			count++
		}
	}
}
