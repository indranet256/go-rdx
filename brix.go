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
	"strings"
)

var ErrBadFile = errors.New("not a valid BrixReader file")
var ErrRecordNotFound = errors.New("no such record")
var ErrorBlockNotSupported = errors.New("block type not supported")
var ErrReadOnly = errors.New("read only mode")
var ErrNotOpen = errors.New("file is not open")

type ReaderAt interface {
	io.ReaderAt
	io.Closer
}

const (
	CompressNot = iota
	CompressLZ4
)

const BrixPageLen = 1 << 12

type IndexEntry struct {
	From ID
	// upper 8 bits: compression
	// next 8 bits: log2(len(unpacked))
	// lower 48 bits: position
	pos   uint64
	Bloom uint64
}

func (ie *IndexEntry) MayHaveID(id ID) bool {
	bloom := ie.Bloom
	bit := uint64(1) << (id.Xor() & 63)
	return 0 != (bit & bloom)
}

func (ie *IndexEntry) AppendBinary(to []byte) ([]byte, error) {
	to = binary.LittleEndian.AppendUint64(to, ie.From.Seq)
	to = binary.LittleEndian.AppendUint64(to, ie.From.Src)
	to = binary.LittleEndian.AppendUint64(to, ie.pos)
	to = binary.LittleEndian.AppendUint64(to, ie.Bloom)
	return to, nil
}

func (ie *IndexEntry) MarshalBinary() (into []byte, err error) {
	return ie.AppendBinary(nil)
}

func (ie *IndexEntry) UnmarshalBinary(from []byte) error {
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

type BrikHeader struct {
	Magic    [8]byte
	MetaLen  uint64
	DataLen  uint64
	IndexLen uint64
}

const BrixIndexEntryLen = 32
const BrixHeaderLen = 32
const BrixMagic = "BRIX    "

func (hdr BrikHeader) AppendBinary(to []byte) ([]byte, error) {
	to = append(to, BrixMagic...)
	to = binary.LittleEndian.AppendUint64(to, hdr.MetaLen)
	to = binary.LittleEndian.AppendUint64(to, hdr.DataLen)
	to = binary.LittleEndian.AppendUint64(to, hdr.IndexLen)
	return to, nil
}

func (hdr *BrikHeader) MarshalBinary() (into []byte, err error) {
	return hdr.AppendBinary(nil)
}

func (hdr *BrikHeader) UnmarshalBinary(from []byte) error {
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

type BrikIterator struct {
	Host *Brik
	// -1 for writes
	PageNo   int
	Off, Len int
	Id       ID
	Page     []byte
}

type Brik struct {
	// The underlying file
	File   *os.File
	Reader ReaderAt
	// File header, section lengths
	Header BrikHeader
	// Brik position in the Wall, expressed as hashes
	// 0. base brick hash
	// 1. orig brick hash
	// *. merged hashes
	Meta []Sha256
	// there are two types of blocks, my friend
	// the ones that fit in 4K
	// and the ones that only have one record
	Index []IndexEntry
	// RFC 7574 peak hashes
	Hash7574 Sha256
	Merkle   *Sha256Merkle7574
	// The default iterator
	At *BrikIterator
}

type Brix []*Brik

func (brix *Brik) Open(reader ReaderAt) (err error) {
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

	if brix.Header.MetaLen > 0 {
		err = brix.loadHashes()
	}
	if err == nil {
		err = brix.loadIndex()
	}
	return
}

func (brix *Brik) loadHashes() (err error) {
	meta := make([]byte, brix.Header.MetaLen)
	brix.Meta = make([]Sha256, 0, brix.Header.MetaLen/Sha256Bytes)
	n := 0
	n, err = brix.Reader.ReadAt(meta, BrixHeaderLen)
	if err != nil {
		return err
	}
	if n != int(brix.Header.MetaLen) {
		return ErrBadFile
	}
	for m := meta[:]; len(m) > 0; m = m[32:] { // TODO better
		brix.Meta = append(brix.Meta, Sha256(m[:32]))
	}
	return
}

func (brix *Brik) loadIndex() (err error) {
	off := int64(BrixHeaderLen + brix.Header.MetaLen + brix.Header.DataLen)
	todo := brix.Header.IndexLen
	brix.Index = make([]IndexEntry, 0, brix.Header.IndexLen/BrixIndexEntryLen)
	for todo > 0 {
		var e [BrixIndexEntryLen]byte
		_, err = brix.Reader.ReadAt(e[:], off)
		if err != nil {
			break
		}
		off += BrixIndexEntryLen
		todo -= BrixIndexEntryLen
		var entry IndexEntry
		err = entry.UnmarshalBinary(e[:])
		if err != nil {
			break
		}
		brix.Index = append(brix.Index, entry)
	}
	return
}

func (brix *Brik) OpenByPath(path string) (err error) {
	brix.File, err = os.Open(path)
	if err == nil {
		err = brix.Open(brix.File)
	}
	return
}

const BrixFileExt = ".brix"

func (brix *Brik) OpenByHash(hash Sha256) error {
	name := make([]byte, 0, 32+16)
	name = append(name, hash.String()...)
	name = append(name, BrixFileExt...)
	brix.Hash7574 = hash
	return brix.OpenByPath(string(name))
}

func FindByHashlet(hashlet string) (sha Sha256, err error) {
	var list []os.DirEntry
	list, err = os.ReadDir(".")
	var nm string
	for _, l := range list {
		if l.IsDir() {
			continue
		}
		nm = l.Name()
		if !strings.HasSuffix(nm, BrixFileExt) || len(nm) != Sha256Bytes*2+len(BrixFileExt) {
			continue
		}
		if strings.HasPrefix(nm, hashlet) {
			sha, err = ParseSha256([]byte(nm)[:Sha256Bytes*2])
			return
		}
	}
	err = os.ErrNotExist
	return

}

func (brix *Brik) findPage(id ID) int {
	return sort.Search(len(brix.Index), func(ndx int) bool {
		return brix.Index[ndx].From.Compare(id) >= Eq
	})
}

func (brix *Brik) loadPage(i int) (iter *BrikIterator, err error) {
	iter = &BrikIterator{}
	return iter, brix.loadPage2(iter, i)
}

func (brix *Brik) loadPage2(iter *BrikIterator, i int) (err error) {
	from := brix.Index[i].Position()
	till := brix.Header.DataLen
	if i+1 < len(brix.Index) {
		till = brix.Index[i+1].Position()
	}
	start := BrixHeaderLen + brix.Header.MetaLen
	pad := make([]byte, till-from)
	_, e := brix.Reader.ReadAt(pad, int64(start+from))
	if e != nil {
		return e
	}
	iter.PageNo = i
	iter.Host = brix
	iter.Id = brix.Index[i].From
	iter.Len, iter.Off = 0, 0
	switch brix.Index[i].Compression() {
	case CompressNot:
		iter.Page = pad
	case CompressLZ4:
		iter.Page = make([]byte, 0, brix.Index[i].UncompressedLength())
		l := 0
		l, err = lz4.UncompressBlock(pad, iter.Page)
		iter.Page = iter.Page[0:l]
	default:
		err = ErrorBlockNotSupported
		iter = nil
	}
	return
}

func (it *BrikIterator) NextPage() (err error) {
	if it.PageNo+1 >= len(it.Host.Index) {
		return io.EOF
	}
	return it.Host.loadPage2(it, it.PageNo+1)
}

func (it *BrikIterator) Next() (record []byte, err error) {
	p := it.Page[it.Off+it.Len:]
	if len(p) == 0 {
		err = it.NextPage()
		if err != nil {
			return
		}
	}
	var rest []byte
	_, it.Id, _, rest, err = ReadRDX(p)
	if err == nil {
		it.Off += it.Len
		it.Len = len(p) - len(rest)
		record = it.Page[it.Off : it.Off+it.Len]
	}
	return
}

func (it *BrikIterator) ScanTo(id ID) (record []byte, err error) {
	z := it.Id.Compare(id)
	if z == Grtr {
		it.Id = it.Host.Index[it.PageNo].From
		it.Off = 0
		it.Len = 0
	} else if z == Eq {
		p := it.Page[it.Off:]
		var rest []byte
		_, it.Id, _, rest, err = ReadRDX(p)
		it.Len = len(p) - len(rest)
		record = p[it.Off : it.Off+it.Len]
	} else {
		for z == Less && err == nil {
			record, err = it.Next()
			z = it.Id.Compare(id)
		}
	}
	return
}

func (brix *Brik) ReadRecord(id ID) (record []byte, err error) {
	i := brix.findPage(id)
	if i == len(brix.Index) || !brix.Index[i].MayHaveID(id) {
		return nil, ErrRecordNotFound
	}
	if brix.At == nil || i != brix.At.PageNo || brix.At.Host == nil {
		brix.At, err = brix.loadPage(i)
		if err != nil {
			return
		}
	}
	return brix.At.ScanTo(id)
}

func (brix *Brik) Close() (err error) {
	if brix.File != nil {
		err = brix.File.Close()
	} else if brix.Reader != nil {
		err = brix.Reader.Close()
	}
	brix.Reader = nil
	brix.File = nil
	return
}

func (brix *Brik) Create(meta []Sha256) (err error) {
	brix.File, err = os.CreateTemp(".", ".tmp.*.brix")
	if err != nil {
		return
	}
	brix.Reader = brix.File
	brix.Meta = append(brix.Meta, meta...)
	brix.Index = append(brix.Index, IndexEntry{})
	brix.Header.MetaLen = uint64(len(meta) * Sha256Bytes)
	h, _ := brix.Header.MarshalBinary()
	_, err = brix.File.Write(h)
	for i := 0; i < len(meta) && err == nil; i++ {
		_, err = brix.File.Write(meta[i][:])
	}
	brix.At = &BrikIterator{
		Host:   brix,
		PageNo: -1,
		Page:   make([]byte, 0, BrixPageLen),
	}
	brix.Merkle = &Sha256Merkle7574{}
	return
}

func (brix *Brik) flushBlock() (err error) {
	if brix.At.PageNo != -1 {
		return ErrReadOnly
	}
	block := brix.At.Page
	idx := &brix.Index[len(brix.Index)-1]
	var factlen int
	var ZPad = make([]byte, len(block)+8)
	factlen, err = lz4.CompressBlock(block, ZPad, nil)
	if err != nil {
		return
	}
	if factlen != 0 && factlen < len(block)*2/3 {
		idx.pos |= uint64(CompressLZ4) << 56
		block = ZPad[:factlen]
	}
	idx.pos |= uint64(bits.LeadingZeros(uint(len(block)))) << 48
	factlen, err = brix.File.Write(block)
	if err != nil {
		return
	}
	brix.Header.DataLen += uint64(len(block))
	brix.Index = append(brix.Index, IndexEntry{
		pos: brix.Header.DataLen,
	})
	idx = &brix.Index[len(brix.Index)-1]
	hash := Sha256Of(block)
	_ = brix.Merkle.Append(hash)
	brix.At.Page = brix.At.Page[:0]
	return
}

func (brix *Brik) WriteAll(rec []byte) (n int, err error) {
	for len(rec) > 0 && err == nil {
		p := 0
		p, err = brix.Write(rec)
		rec = rec[p:]
		n += p
	}
	return
}

func (brix *Brik) Unlink() error {
	if brix.File == nil {
		return ErrNotOpen
	}
	return os.Remove(brix.File.Name())
}

func (brix *Brik) Write(rec []byte) (n int, err error) {
	idx := &brix.Index[len(brix.Index)-1]
	var id ID
	var rest []byte
	_, id, _, rest, err = ReadRDX(rec)
	if brix.At.Id.Compare(id) != Less {
		return 0, ErrBadOrder
	}
	n = len(rec) - len(rest)
	if len(brix.At.Page)+n > BrixPageLen {
		err = brix.flushBlock()
		if err != nil {
			return
		}
		idx = &brix.Index[len(brix.Index)-1]
	}
	if idx.From.IsZero() {
		if id.IsZero() {
			return 0, ErrBadRecord
		}
		idx.From = id
	}
	idx.Bloom |= uint64(1) << (id.Xor() & 63)
	brix.At.Page = append(brix.At.Page, rec[:n]...)
	brix.At.Id = id
	// TODO don't copy larger records (1M?)
	return
}

func (brix *Brik) IsWritable() bool {
	return brix.File != nil && brix.At.PageNo == -1
}

func (brix *Brik) Seal() (err error) {
	if len(brix.At.Page) != 0 {
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
	idxlen, err = brix.File.Write(idx)
	if err != nil {
		return
	}
	brix.Header.IndexLen = uint64(idxlen)
	tmppath := brix.File.Name()
	err = brix.File.Close()
	if err != nil {
		return
	}

	brix.Hash7574 = brix.Merkle.Sum()
	newpath := brix.Hash7574.String() + ".brix"
	err = os.Rename(tmppath, newpath)
	brix.File, err = os.OpenFile(newpath, os.O_RDWR, 0)
	if err != nil {
		return
	}
	header := make([]byte, 0, 32)
	header, _ = brix.Header.AppendBinary(header)
	_, err = brix.File.Write(header)
	if err != nil {
		return
	}
	err = brix.File.Close()
	brix.File = nil

	if err == nil {
		err = brix.OpenByHash(brix.Hash7574)
	}

	return
}

func (brix Brix) OpenByHash(hash Sha256) (more Brix, err error) {
	for _, b := range brix {
		if b.Hash7574.Equal(hash) {
			return
		}
	}
	more = brix
	b := &Brik{}
	err = b.OpenByHash(hash)
	if len(b.Meta) > 0 {
		more, err = brix.OpenByHash(b.Meta[0])
	}
	if err == nil {
		more = append(brix, b)
	} else {
		_ = b.Close()
	}
	return
}

func (brix Brix) Close() (err error) {
	for _, b := range brix {
		e := b.Close()
		if err == nil {
			err = e
		}
	}
	return
}

func (brix Brix) Clone() Brix {
	return append(Brix(nil), brix...)
}

func (brix Brix) OpenByHashlet(hashlet string) (more Brix, err error) {
	var sha Sha256
	sha, err = FindByHashlet(hashlet)
	if err == nil {
		more, err = brix.OpenByHash(sha)
	}
	return
}

func (brix Brix) Get(pad []byte, id ID) (rec []byte, err error) {
	var inputs = make([][]byte, 0, len(brix))
	for _, b := range brix {
		in, e := b.ReadRecord(id)
		if e == ErrRecordNotFound {
			continue
		} else if e == nil {
			inputs = append(inputs, in)
		} else {
			return nil, e
		}
	}
	if len(inputs) == 1 {
		rec = append(pad, inputs[0]...)
	} else if len(inputs) == 0 {
		err = ErrRecordNotFound
	} else {
		rec, err = Merge(pad, inputs)
	}
	return
}

func (it *BrikIterator) Close() error {
	it.Id = ID{}
	it.Host = nil
	it.Page = nil
	it.PageNo, it.Len, it.Off = 0, 0, 0
	return nil
}

type BrixIterator struct { // BIG FIXME same ID different type
	pages []*BrikIterator
	iters []Iter
	heap  Heap
}

func (bit *BrixIterator) IsEmpty() bool {
	return len(bit.heap) == 0
}

func (bit *BrixIterator) Close() error {
	for _, i := range bit.pages {
		_ = i.Close()
	}
	return nil
}

func (bit *BrixIterator) nextPage() (err error) {
	for k := len(bit.heap); k < cap(bit.heap) && bit.heap[k] != nil && err != nil; k++ {
		it := bit.heap[k]
		ndx := 0
		for ndx < len(bit.iters) && &bit.iters[ndx] != it {
			ndx++
		}
		page := bit.pages[ndx]
		if len(page.Host.Index) > page.PageNo {
			bit.heap[k] = nil
		} else {
			bit.pages[ndx], err = page.Host.loadPage(page.PageNo + 1) // FIXME NextPage()
			bit.heap = append(bit.heap, &Iter{Rest: bit.pages[ndx].Page})
			bit.heap.LastUp(CompareID)
		}
	}
	return
}

func (brix Brix) Iterator() (bit *BrixIterator, err error) {
	it := BrixIterator{
		pages: make([]*BrikIterator, len(brix)),
		iters: make([]Iter, len(brix)),
		heap:  make([]*Iter, len(brix)),
	}

	for n, b := range brix {
		it.pages[n], err = b.loadPage(0)
		if err != nil {
			return nil, err
		}
		it.iters[n].Rest = it.pages[n].Page
		it.heap = append(it.heap, &it.iters[n])
		it.heap.LastUp(CompareID)
	}

	return &it, nil
}

func (bit *BrixIterator) Next(data []byte) (rec []byte, err error) {
	ol := len(bit.heap)
	rec, err = bit.heap.MergeNext(data, CompareID)
	if err == nil && len(bit.heap) != ol {
		err = bit.nextPage()
	}
	return
}

func (brix Brix) join() (joined *Brik, err error) {
	deps := make([]Sha256, 0, len(brix))
	for _, b := range brix {
		deps = append(deps, b.Hash7574)
	}
	joined = &Brik{}
	err = joined.Create(deps)
	res := make([]byte, 0, BrixPageLen)
	var it *BrixIterator
	it, err = brix.Iterator()

	for !it.IsEmpty() && err == nil {
		res, err = it.Next(res)
		_, err = joined.Write(res)
		res = res[:0]
	}

	if err == nil {
		err = joined.Seal()
	}
	if err == nil {
		err = joined.Close()
	}
	if err != nil {
		_ = joined.Unlink()
	}
	return
}

func (brix Brix) Merge1() (merged Brix, err error) {
	return
}

func (brix Brix) Merge8() (merged Brix, err error) {
	return
}

func (brix Brix) Merge() (merged Brix, err error) {
	return
}
