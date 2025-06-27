package rdx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/pierrec/lz4/v4"
	"io"
	"math/bits"
	"os"
	"sort"
)

var ErrBadFile = errors.New("not a valid BrixReader file")
var ErrRecordNotFound = errors.New("no such record")
var ErrorBlockNotSupported = errors.New("block type not supported")

type ReaderAt interface {
	io.ReaderAt
	io.Closer
}

const (
	CompressNot = iota
	CompressLZ4
)

type IndexEntry struct {
	From ID
	// upper 8 bits: compression
	// next 8 bits: log2(len(unpacked))
	// lower 48 bits: position
	pos   uint64
	Bloom uint64
}

func (ie IndexEntry) MayHaveID(id ID) bool {
	bloom := ie.Bloom
	bit := uint64(1) << (id.Xor() & 63)
	return 0 != (bit & bloom)
}

func (ie IndexEntry) AppendBinary(to []byte) ([]byte, error) {
	to = binary.LittleEndian.AppendUint64(to, ie.From.Seq)
	to = binary.LittleEndian.AppendUint64(to, ie.From.Src)
	to = binary.LittleEndian.AppendUint64(to, ie.pos)
	to = binary.LittleEndian.AppendUint64(to, ie.Bloom)
	return to, nil
}

func (ie IndexEntry) MarshalBinary() (into []byte, err error) {
	return ie.AppendBinary(nil)
}

func (ie IndexEntry) UnmarshalBinary(from []byte) error {
	if len(from) < BrixIndexEntryLen {
		return ErrBadRecord
	}
	ie.From.Seq = binary.LittleEndian.Uint64(from[0:8])
	ie.From.Src = binary.LittleEndian.Uint64(from[8:16])
	ie.pos = binary.LittleEndian.Uint64(from[16:24])
	ie.Bloom = binary.LittleEndian.Uint64(from[24:BrixIndexEntryLen])
	return nil
}

func (ie IndexEntry) Compression() int {
	return int(ie.pos >> 56)
}

func (ie IndexEntry) UncompressedLength() int {
	return 1 << (0xff & (ie.pos >> 48))
}

const mask48 = (uint64(1) << 48) - 1

func (ie IndexEntry) Position() uint64 {
	return ie.pos & mask48
}

type BrixHeader struct {
	Magic    [8]byte
	MetaLen  uint64
	DataLen  uint64
	IndexLen uint64
}

const BrixIndexEntryLen = 32
const BrixHeaderLen = 32
const BrixMagic = "BRIX    "

func (hdr BrixHeader) AppendBinary(to []byte) ([]byte, error) {
	to = append(to, BrixMagic...)
	to = binary.LittleEndian.AppendUint64(to, hdr.MetaLen)
	to = binary.LittleEndian.AppendUint64(to, hdr.DataLen)
	to = binary.LittleEndian.AppendUint64(to, hdr.IndexLen)
	return to, nil
}

func (hdr BrixHeader) MarshalBinary() (into []byte, err error) {
	return hdr.AppendBinary(nil)
}

func (hdr BrixHeader) UnmarshalBinary(from []byte) error {
	if len(from) < BrixHeaderLen {
		return ErrBadHeader
	}
	copy(hdr.Magic[:], from[:8])
	if !bytes.Equal([]byte(BrixMagic), hdr.Magic[:]) {
		return ErrBadHeader
	}
	hdr.MetaLen = binary.LittleEndian.Uint64(from[8:16])
	hdr.DataLen = binary.LittleEndian.Uint64(from[16:24])
	hdr.IndexLen = binary.LittleEndian.Uint64(from[24:BrixHeaderLen])
	if hdr.IndexLen%BrixIndexEntryLen != 0 || hdr.MetaLen%32 != 0 {
		return ErrBadHeader
	}
	return nil
}

type BrixReader struct {
	Base   *BrixReader
	Reader ReaderAt
	Header BrixHeader
	AtLen  uint64
	Hashes []Sha256
	// there are two types of blocks, my friend
	// the ones that fit in 4K
	// and the ones that only have one record
	Index []IndexEntry
	Pad   []byte
}

type BrixWriter struct {
	Writer   *os.File
	Header   BrixHeader
	Index    []IndexEntry
	Hashes   Sha256Merkle7574
	Block    []byte
	Hash7574 Sha256
	Compress int
}

//func (hdr *BrixHeader)

func (brix *BrixReader) Open(reader ReaderAt) (err error) {
	var head [8 * 4]byte
	n := 0
	n, err = reader.ReadAt(head[:], 0)
	if err != nil {
		return err
	}
	if n < len(head) {
		return ErrBadFile
	}

	brix.Reader = reader
	err = brix.Header.UnmarshalBinary(head[:])
	if err != nil {
		return
	}

	meta := make([]byte, brix.Header.MetaLen)
	n, err = reader.ReadAt(meta, 8*4)
	if err != nil {
		return err
	}
	if n != int(brix.Header.MetaLen) {
		return ErrBadFile
	}
	for m := meta[:]; len(m) > 0; m = m[32:] { // TODO better
		brix.Hashes = append(brix.Hashes, Sha256(m[:32]))
	}
	// TODO Index

	return nil
}

func (brix *BrixReader) OpenByPath(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	} else {
		return brix.Open(file)
	}
}

const BrixFileExt = ".brix"

func (brix *BrixReader) OpenByHash(hash Sha256) error {
	name := make([]byte, 0, 32+16)
	name = append(name, hash.String()...)
	name = append(name, BrixFileExt...)
	return brix.OpenByPath(string(name))
}

func (brix *BrixReader) Hash() Sha256 {
	// todo full rescan
	return Sha256{}
}

func (brix *BrixReader) ReadPage(id ID) (page []byte, err error) {
	i := sort.Search(len(brix.Index), func(ndx int) bool {
		return brix.Index[ndx].From.Compare(id) >= Eq
	})
	if i == len(brix.Index) || !brix.Index[i].MayHaveID(id) {
		return nil, ErrRecordNotFound
	}
	from := brix.Index[i].Position()
	till := brix.Header.DataLen
	if i+1 < len(brix.Index) {
		till = brix.Index[i+1].Position()
	}
	n, e := brix.Reader.ReadAt(brix.Pad[:till-from], int64(from))
	if e != nil {
		return nil, e
	}
	zipped := brix.Pad[:n]
	switch brix.Index[i].Compression() {
	case CompressNot:
		return zipped, nil
	case CompressLZ4:
		l := 0
		l, err = lz4.UncompressBlock(zipped, brix.Pad[n:])
		return brix.Pad[n : n+l], err // l is 0 on error
	default:
		return nil, ErrorBlockNotSupported
	}
}

func (brix *BrixReader) ReadRecord(id ID) (record []byte, err error) {
	var page []byte
	page, err = brix.ReadPage(id)
	if err != nil {
		return
	}
	p := page
	pos := 0
	i := ID{}
	_, i, _, p, err = ReadRDX(p)
	for i != id && len(p) > 0 && err == nil {
		pos = len(page) - len(p)
		_, i, _, p, err = ReadRDX(p)
	}
	if i != id && err == nil {
		err = ErrRecordNotFound
	}
	record = page[pos : len(page)-len(p)]
	return
}

func (brix *BrixReader) Get(pad []byte, id ID) (rec []byte, err error) {
	var inputs = make([][]byte, 0, 64)
	for c := brix; c != nil && err == nil; c = c.Base {
		var in []byte
		in, e := brix.ReadRecord(id)
		if e == ErrRecordNotFound {
			continue
		} else if e == nil {
			inputs = append(inputs, in)
		} else {
			err = e
		}
	}
	if err == nil {
		rec, err = Merge(pad, inputs)
	}
	return
}

func (brix *BrixReader) Close() error {
	err := brix.Reader.Close()
	brix.Reader = nil
	return err
}

func (brix *BrixWriter) Open() (err error) {
	brix.Writer, err = os.CreateTemp(".", ".tmp.*.brix")
	if err != nil {
		return
	}
	brix.Index = append(brix.Index, IndexEntry{})
	hdr := BrixHeader{}
	h, _ := hdr.MarshalBinary()
	_, err = brix.Writer.Write(h)
	return
}

func (brix *BrixWriter) OpenMerge(inputs []Sha256) (err error) {
	err = brix.Open()
	// TODO
	return
}

const PageLen = 1 << 12

func (brix *BrixWriter) flushBlock() (err error) {
	idx := &brix.Index[len(brix.Index)-1]
	var factlen int
	switch brix.Compress {
	case CompressLZ4:
		var ZPad = make([]byte, PageLen)
		factlen, err = lz4.CompressBlock(brix.Block, ZPad, nil)
		if err != nil {
			return
		}
		factlen, err = brix.Writer.Write(ZPad[:factlen])
		idx.pos |= uint64(CompressLZ4) << 56
	default:
		factlen, err = brix.Writer.Write(brix.Block)
	}
	if err != nil {
		return
	}
	brix.Header.DataLen += uint64(factlen)
	idx.pos |= uint64(bits.LeadingZeros(uint(len(brix.Block)))) << 48
	brix.Index = append(brix.Index, IndexEntry{
		pos: brix.Header.DataLen,
	})
	idx = &brix.Index[len(brix.Index)-1]
	hash := Sha256Of(brix.Block)
	_ = brix.Hashes.Append(hash)
	brix.Block = brix.Block[:0]
	if cap(brix.Block) > (1 << 20) {
		brix.Block = make([]byte, 0, 1<<12)
	}
	return
}

func (brix *BrixWriter) WriteAll(rec []byte) (n int, err error) {
	for len(rec) > 0 && err == nil {
		p := 0
		p, err = brix.Write(rec)
		rec = rec[p:]
		n += p
	}
	return
}

func (brix *BrixWriter) Write(rec []byte) (n int, err error) {
	if len(brix.Block)+len(rec) > PageLen {
		err = brix.flushBlock()
		if err != nil {
			return
		}
	}
	idx := &brix.Index[len(brix.Index)-1]
	var id ID
	var rest []byte
	_, id, _, rest, err = ReadRDX(rec)
	n = len(rec) - len(rest)
	if err != nil || len(rest) > 0 {
		return 0, ErrBadRecord
	}
	if idx.From.Compare(id) != Less {
		return 0, ErrBadOrder
	}
	if idx.From.IsZero() {
		if id.IsZero() {
			return 0, ErrBadRecord
		}
		idx.From = id
	}
	idx.Bloom |= uint64(1) << (id.Xor() & 63)
	brix.Block = append(brix.Block, rec[:n]...)
	return
}

func (brix *BrixWriter) Close() (err error) {
	if len(brix.Block) != 0 {
		err = brix.flushBlock()
		if err != nil {
			return
		}
	}
	brix.Index = brix.Index[:len(brix.Index)-1]
	idx := make([]byte, 0, len(brix.Index)*32)
	for _, i := range brix.Index {
		idx, _ = i.AppendBinary(idx)
	}
	var idxlen int
	idxlen, err = brix.Writer.Write(idx)
	if err != nil {
		return
	}
	brix.Header.IndexLen = uint64(idxlen)
	tmppath := brix.Writer.Name()
	err = brix.Writer.Close()
	if err != nil {
		return
	}

	brix.Hash7574 = brix.Hashes.Sum()
	newpath := brix.Hash7574.String() + ".brix"
	err = os.Rename(tmppath, newpath)
	var rew *os.File
	rew, err = os.OpenFile(newpath, os.O_RDWR, 0)
	if err != nil {
		return
	}
	header := make([]byte, 0, 32)
	header, _ = brix.Header.AppendBinary(header)
	_, err = rew.Write(header)
	if err != nil {
		return
	}
	err = rew.Close()

	return
}
