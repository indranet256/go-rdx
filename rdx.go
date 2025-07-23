package rdx

import (
	"bytes"
	"errors"
	"math"
	"math/bits"
	"unicode/utf8"
)

const (
	Float     = 'F'
	Integer   = 'I'
	Reference = 'R'
	String    = 'S'
	Term      = 'T'

	Tuple  = 'P'
	Linear = 'L'
	Euler  = 'E'
	Multix = 'X'
)

const MaxInputs = 64
const MaxNesting = 255

var (
	ErrBadRDXRecord       = errors.New("bad RDX record format")
	ErrWrongRDXRecordType = errors.New("wrong RDX record type")
	ErrBadUtf8            = errors.New("bad UTF8 codepoint")
	ErrBadState           = errors.New("bad state")
	ErrBadOrder           = errors.New("bad RDX order")
	ErrEoF                = errors.New("end of file")
)

func IsPLEX(lit byte) bool {
	return lit == Tuple || lit == Linear || lit == Euler || lit == Multix
}

func IsFIRST(lit byte) bool {
	return lit == Float || lit == Integer || lit == Reference || lit == String || lit == Term
}

func ReadRDX(data []byte) (lit byte, id ID, value, rest []byte, err error) {
	var pair []byte
	lit, pair, value, rest, err = ReadTLKV(data)
	if err == nil {
		id.Seq, id.Src = UnzipUint64Pair(pair)
	}
	return
}

func WriteRDX(data []byte, lit byte, id ID, value []byte) []byte {
	pair := ZipUint64Pair(id.Seq, id.Src)
	return WriteTLKV(data, lit, pair, value)
}

type Merger func(data []byte, bare Heap) ([]byte, error)

func mergeValuesF(data []byte, bare [][]byte) ([]byte, error) {
	var mx float64
	var win []byte
	for i, b := range bare {
		n := UnzipFloat64(b)
		if i == 0 || n > mx {
			mx = n
			win = b
		}
	}
	data = append(data, win...)
	return data, nil
}

func mergeValuesI(data []byte, bare [][]byte) ([]byte, error) {
	var max int64
	var win []byte
	for i, b := range bare {
		n := UnzipInt64(b)
		if i == 0 || n > max {
			max = n
			win = b
		}
	}
	data = append(data, win...)
	return data, nil
}

func mergeValuesR(data []byte, bare [][]byte) ([]byte, error) {
	var max ID
	var win []byte
	for i, b := range bare {
		n := UnzipID(b)
		if i == 0 || max.Compare(n) < 0 {
			max = n
			win = b
		}
	}
	data = append(data, win...)
	return data, nil
}

func mergeValuesS(data []byte, bare [][]byte) ([]byte, error) {
	var win []byte
	for i, b := range bare {
		if i == 0 || bytes.Compare(win, b) < 0 {
			win = b
		}
	}
	data = append(data, win...)
	return data, nil
}

func mergeValuesT(data []byte, bare [][]byte) ([]byte, error) {
	return mergeValuesS(data, bare)
}

func Merge(data []byte, bare [][]byte) (ret []byte, err error) {
	return mergeElementsP(data, bare)
}

func mergeElementsP(data []byte, bare [][]byte) (ret []byte, err error) {
	return HeapMerge(data, bare, CompareTuple)
}

func mergeElementsL(data []byte, bare [][]byte) ([]byte, error) {
	return HeapMerge(data, bare, CompareLinear)
}

func mergeElementsE(data []byte, bare [][]byte) ([]byte, error) {
	return HeapMerge(data, bare, CompareEuler)
}

func mergeElementsX(data []byte, bare [][]byte) ([]byte, error) {
	return HeapMerge(data, bare, CompareMultix)
}

func mergeSameSpotElements(data []byte, heap Heap) (ret []byte, err error) {
	eq := 1
	for i := 1; i < len(heap); i++ {
		z := CompareLWW(&heap[0], &heap[i])
		if z < Eq {
			heap[0], heap[i] = heap[i], heap[0]
			eq = 1
		} else if z > Eq {
			pl := len(heap) - 1
			heap[pl], heap[i] = heap[i], heap[pl]
			heap = heap[:pl]
			i--
		} else {
			heap[eq], heap[i] = heap[i], heap[eq]
			eq++
		}
	}
	eqs := heap[:eq]
	lit := eqs[0].Lit()
	vals := make([][]byte, 0, MaxInputs)
	stack := make(Marks, 0, 16)
	id := heap[0].ID()
	ret = OpenTLV(data, lit, &stack)
	key := ZipID(id)
	ret = append(ret, byte(len(key)))
	ret = append(ret, key...) // TODO
	for _, val := range eqs {
		vals = append(vals, val.Value())
	} // FIXME 1
	switch lit {
	case Float:
		ret, err = mergeValuesF(ret, vals)
	case Integer:
		ret, err = mergeValuesI(ret, vals)
	case Reference:
		ret, err = mergeValuesR(ret, vals)
	case String:
		ret, err = mergeValuesS(ret, vals)
	case Term:
		ret, err = mergeValuesT(ret, vals)
	case Tuple:
		ret, err = mergeElementsP(ret, vals)
	case Linear:
		ret, err = mergeElementsL(ret, vals)
	case Euler:
		ret, err = mergeElementsE(ret, vals)
	case Multix:
		ret, err = mergeElementsX(ret, vals)
	default:
		ret, err = nil, ErrBadRDXRecord
	}
	if err == nil {
		ret, err = CloseTLV(ret, lit, &stack)
	}
	return
}

const (
	Less = -2
	LEq  = -1
	Eq   = 0
	GrEq = 1
	Grtr = 2
)

type Compare func(a *Iter, b *Iter) int

func CompareLWW(a *Iter, b *Iter) int {
	z := CompareID(a, b)
	if z == Eq {
		z = CompareType(a, b)
	}
	return z
}

func CompareFloat(a *Iter, b *Iter) int {
	af := UnzipFloat64(a.Value())
	bf := UnzipFloat64(b.Value())
	if af == bf {
		return Eq
	} else if af < bf {
		return Less
	} else {
		return Grtr
	}
}

func CompareInteger(a *Iter, b *Iter) int {
	af := UnzipInt64(a.Value())
	bf := UnzipInt64(b.Value())
	if af == bf {
		return Eq
	} else if af < bf {
		return Less
	} else {
		return Grtr
	}
}

func CompareReference(a *Iter, b *Iter) int {
	aid := UnzipID(a.Value())
	bid := UnzipID(b.Value())
	return aid.Compare(bid)
}

func CompareString(a *Iter, b *Iter) int {
	return bytes.Compare(a.Value(), b.Value()) * 2
}

func CompareTerm(a *Iter, b *Iter) int {
	return CompareString(a, b)
}

func UnwrapTuple(a *Iter) *Iter {
	b := NewIter(a.Value())
	b.Next()
	return &b
}

func CompareTuple(a *Iter, b *Iter) int {
	return Eq
}

func CompareLinear(a *Iter, b *Iter) int {
	aa := Revert64(a.ID().Seq >> 6)
	bb := Revert64(b.ID().Seq >> 6)
	if aa < bb {
		return Less
	} else if aa > bb {
		return Grtr
	}
	if a.ID().Src < b.ID().Src {
		return Less
	} else if a.ID().Src > b.ID().Src {
		return Grtr
	} else {
		return Eq
	}
}

func CompareType(a *Iter, b *Iter) int {
	al := a.Lit()
	bl := b.Lit()
	if al == bl {
		return Eq
	}
	ap := IsPLEX(al)
	bp := IsPLEX(bl)
	if ap != bp {
		if ap {
			return Grtr
		} else {
			return Less
		}
	}
	if al < bl {
		return Less
	} else {
		return Grtr
	}
}

func CompareID(a *Iter, b *Iter) int {
	return a.ID().Compare(b.ID())
}

func CompareValue(a *Iter, b *Iter) int {
	al := a.Lit()
	bl := b.Lit()
	if al == Tuple {
		a = UnwrapTuple(a)
		al = a.Lit()
	}
	if bl == Tuple {
		b = UnwrapTuple(b)
		bl = b.Lit()
	}
	if al != bl {
		return CompareType(a, b)
	}
	switch al {
	case Float:
		return CompareFloat(a, b)
	case Integer:
		return CompareInteger(a, b)
	case Reference:
		return CompareReference(a, b)
	case String:
		return CompareString(a, b)
	case Term:
		return CompareTerm(a, b)
	case Tuple:
		return CompareID(a, b)
	case Linear:
		return CompareID(a, b)
	case Euler:
		return CompareID(a, b)
	case Multix:
		return CompareID(a, b)
	default:
		return Eq
	}
}

func CompareEuler(a *Iter, b *Iter) int {
	return CompareValue(a, b)
}

func CompareMultix(a *Iter, b *Iter) int {
	if a.ID().Src < b.ID().Src {
		return Less
	} else if a.ID().Src < b.ID().Src {
		return Grtr
	}
	return Eq
}

func ReadTerm(rdx []byte) (val []byte, id ID, rest []byte, err error) {
	var lit byte
	lit, id, val, rest, err = ReadRDX(rdx)
	if err == nil && lit != Term {
		err = ErrWrongRDXRecordType
	}
	return
}

func ReadString(rdx []byte) (val string, id ID, rest []byte, err error) {
	var lit byte
	var v []byte
	lit, id, v, rest, err = ReadRDX(rdx)
	if err == nil && lit != String {
		err = ErrWrongRDXRecordType
	} else {
		val = string(v)
	}
	return
}

func ReadID(rdx []byte) (val, id ID, rest []byte, err error) {
	var v []byte
	var lit byte
	lit, id, v, rest, err = ReadRDX(rdx)
	if err != nil {
		return
	}
	if lit != Reference || len(v) > 16 {
		err = ErrBadRecord
		return
	}
	val = UnzipID(v)
	return
}

func ReadInteger(rdx []byte) (val int64, id ID, rest []byte, err error) {
	var v []byte
	var lit byte
	lit, id, v, rest, err = ReadRDX(rdx)
	if err != nil {
		return
	}
	if lit != Integer || len(v) > 8 {
		err = ErrBadRecord
		return
	}
	val = UnzipInt64(v)
	return
}

func TopBit(v uint64) uint64 {
	l := bits.LeadingZeros64(v)
	return uint64(1) << (63 - l)
}

// L-lexicographically in-between values
func LBetween(a, b uint64) (ret uint64) {
	aa := Revert64(a >> 6)
	bb := Revert64(b >> 6)
	if aa < bb {
		top := TopBit(bb - aa)
		ret = aa + (top >> 6)
	}
	if ret >= bb || ret == aa {
		ret = 1
	}
	return Revert64(ret) << 6
}

func AppendInteger(data []byte, val int64) []byte {
	b := ZipInt64(val)
	return WriteTLKV(data, Integer, nil, b)
}

func AppendString(data []byte, val []byte) []byte {
	return WriteTLKV(data, String, nil, val)
}

func AppendTerm(data []byte, val []byte) []byte {
	return WriteTLKV(data, Term, nil, val)
}

var ErrBadFloatRecord = errors.New("bad Float record format")
var ErrBadIntegerRecord = errors.New("bad Integer record format")
var ErrBadReferenceRecord = errors.New("bad Reference record format")
var ErrBadStringRecord = errors.New("bad String record format")
var ErrBadTermRecord = errors.New("bad Term record format")

func Normalize(rdx []byte) (norm []byte, err error) {
	data := make([]byte, 0, len(rdx))
	stack := Marks{}
	return normalize(data, rdx, CompareTuple, &stack)
}

func normalize(data, rdx []byte, z Compare, stack *Marks) (norm []byte, err error) {
	if len(rdx) == 0 {
		return
	}
	norm = data
	chunks := [][]byte{}
	at := NewIter(rdx)
	at.Next()
	next := at
	oc := len(norm)
	for at.HasData() && err == nil {
		norm, err = appendNorm(norm, at, stack)
		next.Next()
		if err == nil && next.HasData() && z != nil && z(&at, &next) != Less {
			chunks = append(chunks, norm[oc:])
			oc = len(norm)
		}
		at = next
	}
	if at.HasFailed() {
		err = next.Error()
	}
	if len(chunks) > 0 && err == nil {
		chunks = append(chunks, norm[oc:])
		sorted := make([]byte, 0, len(norm)-len(data))
		sorted, err = HeapMerge(sorted, chunks, z)
		norm = append(data, sorted...)
	}
	return
}

func appendNorm(to []byte, it Iter, stack *Marks) (norm []byte, err error) {
	val := it.Value()
	lit := it.Lit()
	idbytes := ZipID(it.ID())
	norm = to
	switch lit {
	case Float:
		if len(val) > 8 {
			return nil, ErrBadFloatRecord
		}
		f := UnzipFloat64(val)
		if math.IsNaN(f) {
			return nil, ErrBadFloatRecord
		}
		val = ZipFloat64(f)
		norm = WriteTLKV(norm, lit, idbytes, val)
	case Integer:
		if len(val) > 8 {
			return nil, ErrBadIntegerRecord
		}
		f := UnzipInt64(val)
		val = ZipInt64(f)
		norm = WriteTLKV(norm, lit, idbytes, val)
	case Reference:
		if len(val) > 16 { // todo bad sizes
			return nil, ErrBadReferenceRecord
		}
		i := UnzipID(val)
		val = ZipID(i)
		norm = WriteTLKV(norm, lit, idbytes, val)
	case String:
		if !utf8.Valid(val) {
			return nil, ErrBadStringRecord
		}
		norm = WriteTLKV(norm, lit, idbytes, val)
	case Term:
		for _, c := range val {
			if RON64REV[c] == 0xff {
				return nil, ErrBadTermRecord
			}
		}
		norm = WriteTLKV(norm, lit, idbytes, val)
	case Tuple:
		norm = OpenTLV(norm, Tuple, stack)
		norm = append(norm, byte(len(idbytes)))
		norm = append(norm, idbytes...)
		norm, err = normalize(norm, val, nil, stack)
		norm, err = CloseTLV(norm, Tuple, stack)
	case Linear:
		norm = OpenTLV(norm, Linear, stack)
		norm = append(norm, byte(len(idbytes)))
		norm = append(norm, idbytes...)
		norm, err = normalize(norm, val, nil, stack)
		norm, err = CloseTLV(norm, Linear, stack)
	case Euler:
		norm = OpenTLV(norm, Euler, stack)
		norm = append(norm, byte(len(idbytes)))
		norm = append(norm, idbytes...)
		norm, err = normalize(norm, val, CompareEuler, stack)
		norm, err = CloseTLV(norm, Euler, stack)
	case Multix:
		norm = OpenTLV(norm, Multix, stack)
		norm = append(norm, byte(len(idbytes)))
		norm = append(norm, idbytes...)
		norm, err = normalize(norm, val, CompareMultix, stack)
		norm, err = CloseTLV(norm, Multix, stack)
	}
	return
}

func flaten(data, rdx []byte, stack *Marks) (flat []byte, err error) {
	flat = data
	for len(rdx) > 0 && err == nil {
		var lit byte
		var id ID
		var val, rest []byte
		lit, id, val, rest, err = ReadRDX(rdx)
		if err != nil {
			break
		}
		if (id.Seq & 1) != 0 {
			if len(*stack) > 0 && (*stack)[len(*stack)-1].Lit == Tuple {
				flat = append(flat, Tuple|CaseBit, 1, 0)
			}
		} else if IsFIRST(lit) {
			flat = WriteTLKV(flat, lit, nil, val)
		} else {
			flat = OpenTLV(flat, lit, stack)
			flat = append(flat, 0)
			flat, err = flaten(flat, val, stack)
			if err == nil {
				flat, err = CloseTLV(flat, lit, stack)
			}
		}
		rdx = rest
	}
	return
}

func Flatten(data, rdx []byte) (flat []byte, err error) {
	stack := make(Marks, 0, 32)
	return flaten(data, rdx, &stack)
}
