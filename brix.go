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
	// there are two types of blocks, my friend,
	// the ones that fit in 4K
	// and the ones that only have one record
	Index []IndexEntry
	// RFC 7574 peak hashes
	Hash7574 Sha256
	Merkle   *Sha256Merkle7574
	// The default iterator
	At    *BrikReader
	block []byte
}

func (brik *Brik) Open(reader ReaderAt) (err error) {
	var head [8 * 4]byte
	n := 0
	n, err = reader.ReadAt(head[:], 0)
	if err != nil {
		return err
	}
	if n < len(head) {
		return ErrBadFile
	}

	brik.Reader = reader
	err = brik.Header.UnmarshalBinary(head[:])
	if err != nil {
		return
	}

	if brik.Header.MetaLen > 0 {
		err = brik.loadHashes()
	}
	if err == nil {
		err = brik.loadIndex()
	}
	return
}

func (brik *Brik) loadHashes() (err error) {
	meta := make([]byte, brik.Header.MetaLen)
	brik.Meta = make([]Sha256, 0, brik.Header.MetaLen/Sha256Bytes)
	n := 0
	n, err = brik.Reader.ReadAt(meta, BrixHeaderLen)
	if err != nil {
		return err
	}
	if n != int(brik.Header.MetaLen) {
		return ErrBadFile
	}
	for m := meta[:]; len(m) > 0; m = m[32:] { // TODO better
		brik.Meta = append(brik.Meta, Sha256(m[:32]))
	}
	return
}

func (brik *Brik) loadIndex() (err error) {
	off := int64(BrixHeaderLen + brik.Header.MetaLen + brik.Header.DataLen)
	todo := brik.Header.IndexLen
	brik.Index = make([]IndexEntry, 0, brik.Header.IndexLen/BrixIndexEntryLen)
	for todo > 0 {
		var e [BrixIndexEntryLen]byte
		_, err = brik.Reader.ReadAt(e[:], off)
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
		brik.Index = append(brik.Index, entry)
	}
	return
}

func (brik *Brik) OpenByPath(path string) (err error) {
	brik.File, err = os.Open(path)
	if err == nil {
		err = brik.Open(brik.File)
	}
	return
}

const BrixFileExt = ".brix"

func (brik *Brik) OpenByHash(hash Sha256) error {
	name := make([]byte, 0, 32+16)
	name = append(name, hash.String()...)
	name = append(name, BrixFileExt...)
	brik.Hash7574 = hash
	return brik.OpenByPath(string(name))
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

func (brik *Brik) findPage(id ID) int {
	return sort.Search(len(brik.Index), func(ndx int) bool {
		return brik.Index[ndx].From.Compare(id) >= Eq
	})
}

func (brik *Brik) loadPage(i int) (page []byte, err error) {
	from := brik.Index[i].Position()
	till := brik.Header.DataLen
	if i+1 < len(brik.Index) {
		till = brik.Index[i+1].Position()
	}
	start := BrixHeaderLen + brik.Header.MetaLen
	pad := make([]byte, till-from)
	_, err = brik.Reader.ReadAt(pad, int64(start+from))
	if err != nil {
		return
	}
	switch brik.Index[i].Compression() {
	case CompressNot:
		page = pad
	case CompressLZ4:
		page = make([]byte, 0, brik.Index[i].UncompressedLength())
		l := 0
		l, err = lz4.UncompressBlock(pad, page)
		page = page[0:l]
	default:
		err = ErrorBlockNotSupported
	}
	return
}

func (brik *Brik) LoadPage(ndx int) (err error) {
	brik.block, err = brik.loadPage(ndx)
	if err != nil {
		return
	}
	brik.At = &BrikReader{
		iter:    NewIter(brik.block),
		pagendx: ndx,
		host:    brik,
	}
	return
}

func (brik *Brik) ReadRecord(id ID) (record []byte, err error) {
	i := brik.findPage(id)
	if i == len(brik.Index) || !brik.Index[i].MayHaveID(id) {
		return nil, ErrRecordNotFound
	}
	if brik.At == nil || i != brik.At.pagendx || brik.At.host == nil {
		err = brik.LoadPage(i)
		if err != nil {
			return
		}
	} else if brik.At.ID().Compare(id) == Grtr {
		brik.At = &BrikReader{
			iter:    NewIter(brik.block),
			pagendx: i,
			host:    brik,
		}
	}
	if !brik.At.Seek(id) {
		return nil, ErrRecordNotFound
	}
	return brik.At.Record(), nil
}

func (brik *Brik) Close() (err error) {
	if brik.File != nil {
		err = brik.File.Close()
	} else if brik.Reader != nil {
		err = brik.Reader.Close()
	}
	brik.Reader = nil
	brik.File = nil
	return
}

func (brik *Brik) Create(meta []Sha256) (err error) {
	brik.File, err = os.CreateTemp(".", ".tmp.*.brik")
	if err != nil {
		return
	}
	brik.Reader = brik.File
	brik.Meta = append(brik.Meta, meta...)
	brik.Index = append(brik.Index, IndexEntry{})
	brik.Header.MetaLen = uint64(len(meta) * Sha256Bytes)
	h, _ := brik.Header.MarshalBinary()
	_, err = brik.File.Write(h)
	for i := 0; i < len(meta) && err == nil; i++ {
		_, err = brik.File.Write(meta[i][:])
	}
	brik.block = make([]byte, 0, BrixPageLen)
	brik.Merkle = &Sha256Merkle7574{}
	return
}

func (brik *Brik) flushBlock() (err error) {
	if brik.At.pagendx != -1 {
		return ErrReadOnly
	}
	idx := &brik.Index[len(brik.Index)-1]
	block := brik.block
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
	factlen, err = brik.File.Write(block)
	if err != nil {
		return
	}
	brik.Header.DataLen += uint64(len(block))
	brik.Index = append(brik.Index, IndexEntry{
		pos: brik.Header.DataLen,
	})
	idx = &brik.Index[len(brik.Index)-1]
	hash := Sha256Of(block)
	_ = brik.Merkle.Append(hash)
	brik.block = brik.block[:0]
	return
}

func (brik *Brik) WriteAll(rec []byte) (n int, err error) {
	for len(rec) > 0 && err == nil {
		p := 0
		p, err = brik.Write(rec)
		rec = rec[p:]
		n += p
	}
	return
}

func (brik *Brik) Unlink() error {
	if brik.File == nil {
		return ErrNotOpen
	}
	return os.Remove(brik.File.Name())
}

func (brik *Brik) Write(rec []byte) (n int, err error) {
	idx := &brik.Index[len(brik.Index)-1]
	var id ID
	var rest []byte
	_, id, _, rest, err = ReadRDX(rec)
	/*if brik.At.Id.Compare(id) != Less {
		return 0, ErrBadOrder
	}*/
	n = len(rec) - len(rest)
	if len(brik.block)+n > BrixPageLen {
		err = brik.flushBlock()
		if err != nil {
			return
		}
		idx = &brik.Index[len(brik.Index)-1]
	}
	if idx.From.IsZero() {
		if id.IsZero() {
			return 0, ErrBadRecord
		}
		idx.From = id
	}
	idx.Bloom |= uint64(1) << (id.Xor() & 63)
	brik.block = append(brik.block, rec[:n]...)
	//brik.At.Id = id
	// TODO don't copy larger records (1M?)
	return
}

func (brik *Brik) IsWritable() bool {
	return brik.File != nil && brik.At.pagendx == -1
}

func (brik *Brik) Seal() (err error) {
	if len(brik.block) != 0 {
		err = brik.flushBlock()
		if err != nil {
			return
		}
	}
	brik.Index = brik.Index[:len(brik.Index)-1]
	idx := make([]byte, 0, len(brik.Index)*32)
	for _, i := range brik.Index {
		idx, _ = i.AppendBinary(idx)
	}
	var idxlen int
	idxlen, err = brik.File.Write(idx)
	if err != nil {
		return
	}
	brik.Header.IndexLen = uint64(idxlen)
	tmppath := brik.File.Name()
	err = brik.File.Close()
	if err != nil {
		return
	}

	brik.Hash7574 = brik.Merkle.Sum()
	newpath := brik.Hash7574.String() + ".brik"
	err = os.Rename(tmppath, newpath)
	brik.File, err = os.OpenFile(newpath, os.O_RDWR, 0)
	if err != nil {
		return
	}
	header := make([]byte, 0, 32)
	header, _ = brik.Header.AppendBinary(header)
	_, err = brik.File.Write(header)
	if err != nil {
		return
	}
	err = brik.File.Close()
	brik.File = nil

	if err == nil {
		err = brik.OpenByHash(brik.Hash7574)
	}

	return
}

// BrikReader iterates over one sorted record file (a brik).
type BrikReader struct {
	host *Brik
	// -1 for writes
	pagendx int
	iter    Iter
}

func (bit *BrikReader) Read() bool {
	if len(bit.iter.Rest()) != 0 {
		return bit.iter.Read()
	} else if bit.pagendx+1 >= len(bit.host.Index) {
		bit.iter = Iter{errndx: 4}
		return false
	} else {
		bit.pagendx++
		page, err := bit.host.loadPage(bit.pagendx)
		if err != nil {
			bit.iter = Iter{errndx: 3}
			return false
		}
		bit.iter = NewIter(page)
		return bit.Read()
	}
}

func (bit *BrikReader) Record() []byte {
	return bit.iter.Record()
}
func (bit *BrikReader) ID() ID {
	return bit.iter.ID()
}
func (bit *BrikReader) Value() []byte {
	return bit.iter.Value()
}
func (bit *BrikReader) Error() error {
	return bit.iter.Error()
}

func (bit *BrikReader) Seek(id ID) bool {
	bit.pagendx = bit.host.findPage(id)
	if bit.pagendx >= len(bit.host.Index) {
		return false
	}
	page, err := bit.host.loadPage(bit.pagendx)
	if err != nil {
		bit.iter = Iter{errndx: 3}
		return false
	}
	bit.iter = NewIter(page)
	return bit.iter.Seek(id)
}

func (bit *BrikReader) Close() error {
	*bit = BrikReader{}
	return nil
}

type Brix []*Brik

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

func (brix Brix) Iterator() (xit BrixReader, err error) {
	if len(brix) > MaxBrixLen {
		err = ErrTooManyBrix
		return
	}
	xit.host = brix
	xit.pages = make([]int, len(brix))
	xit.heap = make(Heap, 0, len(brix))
	for n, b := range brix {
		var page []byte
		page, err = b.loadPage(0)
		if err != nil {
			return
		}
		xit.heap = append(xit.heap, Iter{data: page, errndx: int8(-n)})
		xit.heap.LastUp(CompareID)
	}
	return
}

type BrixReader struct { // BIG FIXME same ID different type
	host  Brix
	pages []int
	heap  Heap
	win   Iter
	data  []byte
}

func (xit *BrixReader) IsEmpty() bool {
	return len(xit.heap) == 0
}

func (xit *BrixReader) Close() error {
	for _, i := range xit.heap {
		_ = i.Close()
	}
	return nil
}

func (xit *BrixReader) nextPage(empty []Iter) (err error) {
	for _, e := range empty {
		if e.errndx > 0 {
			return iterr[e.errndx]
		}
		ndx := -e.errndx
		brik := xit.host[ndx]
		if len(brik.Index) > xit.pages[ndx]+1 {
			var page []byte
			page, err = brik.loadPage(xit.pages[ndx] + 1)
			if err != nil {
				return
			}
			it := Iter{data: page, errndx: -ndx}
			xit.heap = append(xit.heap, it)
			xit.heap.LastUp(CompareID)
		}
	}
	return
}

var ErrTooManyBrix = errors.New("too many bricks")

const MaxBrixLen = 0xff

func (xit *BrixReader) Read() bool {
	ol := len(xit.heap)
	if ol == 0 {
		return false
	}
	var err error
	eqlen := xit.heap.EqUp(CompareID)
	if eqlen == 1 {
		xit.win = xit.heap[0]
	} else {
		eqs := xit.heap[:eqlen]
		xit.data = xit.data[:0]
		xit.data, err = mergeSameSpotElements(xit.data, eqs)
		xit.win = Iter{data: xit.data}
	}
	if err == nil {
		err = xit.heap.NextK(eqlen, CompareID) // FIXME signature
	}
	if err == nil && len(xit.heap) != ol {
		err = xit.nextPage(xit.heap[len(xit.heap):ol])
	}
	if err != nil {
		xit.win.errndx = 3
		return false
	} else {
		return xit.win.Read()
	}
}

func (xit *BrixReader) Seek(id ID) int {
	return Less //???
}

func (xit *BrixReader) Record() []byte {
	return xit.win.Record()
}

func (xit *BrixReader) ID() ID {
	return xit.win.ID()
}

func (xit *BrixReader) Value() []byte {
	return xit.win.Value()
}

func (xit *BrixReader) Error() error {
	return xit.win.Error()
}

func (brix Brix) join() (joined *Brik, err error) {
	deps := make([]Sha256, 0, len(brix))
	for _, b := range brix {
		deps = append(deps, b.Hash7574)
	}
	joined = &Brik{}
	err = joined.Create(deps)
	if err != nil {
		return
	}
	var it BrixReader
	it, err = brix.Iterator()
	for err == nil && it.Read() {
		_, err = joined.Write(it.Record())
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
