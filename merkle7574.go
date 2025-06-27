package rdx

import (
	"bytes"
	"crypto"
	"encoding/hex"
	"errors"
)

type SHA256 [32]byte
type Sha256Merkle7574 []SHA256

var Sha256Zero = SHA256{}

func (sha *SHA256) Clear() {
	copy(sha[:], Sha256Zero[:])
}

func (sha SHA256) String() string {
	return hex.EncodeToString(sha[:])
}

func (sha SHA256) IsEmpty() bool {
	return bytes.Equal(sha[:], Sha256Zero[:])
}

func SHA256Of(data []byte) (ret SHA256) {
	hash := crypto.SHA256.New()
	hash.Write(data[:])
	hash.Sum(ret[:0])
	return
}

func (sha SHA256) Merkle2(b SHA256) (sum SHA256) {
	hash := crypto.SHA256.New()
	hash.Write(sha[:])
	hash.Write(b[:])
	hash.Sum(sum[:0])
	return
}

var ErrOutOfRange = errors.New("out of hash tree range")

func (line Sha256Merkle7574) Append(next SHA256) error {
	p := next
	i := 0
	for ; !line[i].IsEmpty() && i < len(line); i++ {
		p = line[i].Merkle2(p)
		line[i] = Sha256Zero
	}
	if i == len(line) {
		return ErrOutOfRange
	}
	line[i] = p
	return nil
}

func (line Sha256Merkle7574) Sum() (sum SHA256) {
	hash := crypto.SHA256.New()
	for i := 0; i < len(line); i++ {
		hash.Write(line[i][:])
	}
	b := sum[:0]
	hash.Sum(b)
	return
}
