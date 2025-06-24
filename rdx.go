package rdx

import (
	"bytes"
	"errors"
	"math/bits"
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
	ErrBadRDXRecord = errors.New("bad RDX record format")
	ErrBadUtf8      = errors.New("bad UTF8 codepoint")
	ErrBadState     = errors.New("bad state")
	ErrBadOrder     = errors.New("bad RDX order")
	ErrEoF          = errors.New("end of file")
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
	id.Seq, id.Src = UnzipUint64Pair(pair)
	return
}

func WriteRDX(data []byte, lit byte, id ID, value []byte) []byte {
	pair := ZipUint64Pair(id.Seq, id.Src)
	return WriteTLKV(data, lit, pair, value)
}

type Merger func(data []byte, bare Heap) ([]byte, error)

func mergeValuesF(data []byte, bare [][]byte) ([]byte, error) {
	var max float64
	var win []byte
	for i, b := range bare {
		n := UnzipFloat64(b)
		if i == 0 || n > max {
			max = n
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
	return data, nil
}

func mergeElementsE(data []byte, bare [][]byte) ([]byte, error) {
	return HeapMerge(data, bare, CompareEuler)
}

func mergeElementsX(data []byte, bare [][]byte) ([]byte, error) {
	return HeapMerge(data, bare, CompareMultix)
}

func mergeElementsSame(data []byte, heap Heap) (ret []byte, err error) {
	vals := make([][]byte, 0, MaxInputs)
	stack := make(Marks, 0, 16)
	lit := heap[0].Lit
	id := heap[0].Id
	ret = OpenTLV(data, lit, &stack)
	key := ZipID(id)
	ret = append(ret, byte(len(key)))
	ret = append(ret, key...) // TODO
	for _, val := range heap {
		vals = append(vals, val.Value)
	}
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
		z = CompareValue(a, b)
	}
	return z
}

func CompareFloat(a *Iter, b *Iter) int {
	af := UnzipFloat64(a.Value)
	bf := UnzipFloat64(b.Value)
	if af == bf {
		return Eq
	} else if af < bf {
		return Less
	} else {
		return Grtr
	}
}

func CompareInteger(a *Iter, b *Iter) int {
	af := UnzipInt64(a.Value)
	bf := UnzipInt64(b.Value)
	if af == bf {
		return Eq
	} else if af < bf {
		return Less
	} else {
		return Grtr
	}
}

func CompareReference(a *Iter, b *Iter) int {
	aid := UnzipID(a.Value)
	bid := UnzipID(b.Value)
	return aid.Compare(bid)
}

func CompareString(a *Iter, b *Iter) int {
	return bytes.Compare(a.Value, b.Value) * 2
}

func CompareTerm(a *Iter, b *Iter) int {
	return CompareString(a, b)
}

func UnwrapTuple(a *Iter) *Iter {
	b := Iter{Rest: a.Value}
	b.Next()
	return &b
}

func CompareTuple(a *Iter, b *Iter) int {
	return Eq
}

func CompareLinear(a *Iter, b *Iter) int {
	an := bits.ReverseBytes64(a.Id.Seq & ^uint64(0xff))
	bn := bits.ReverseBytes64(b.Id.Seq & ^uint64(0xff))
	if an == bn {
		if a.Id.Src < b.Id.Src {
			return Less
		} else if a.Id.Src > b.Id.Src {
			return Grtr
		} else {
			return Eq
		}
	} else if an < bn {
		return Less
	} else {
		return Grtr
	}
}

func CompareType(a *Iter, b *Iter) int {
	if a.Lit == b.Lit {
		return Eq
	}
	ap := IsPLEX(a.Lit)
	bp := IsPLEX(b.Lit)
	if ap != bp {
		if ap {
			return Grtr
		} else {
			return Less
		}
	}
	if a.Lit < b.Lit {
		return Less
	} else {
		return Grtr
	}
}

func CompareID(a *Iter, b *Iter) int {
	return a.Id.Compare(b.Id)
}

func CompareValue(a *Iter, b *Iter) int {
	if a.Lit == Tuple {
		a = UnwrapTuple(a)
	}
	if b.Lit == Tuple {
		b = UnwrapTuple(b)
	}
	if a.Lit != b.Lit {
		return CompareType(a, b)
	}
	switch a.Lit {
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
	if a.Id.Src < b.Id.Src {
		return Less
	} else if a.Id.Src < b.Id.Src {
		return Grtr
	}
	return Eq
}

func LowDiffBit(a, b uint64) uint64 {
	l := ((a ^ b) &^ 0xff)
	return l & ((l - 1) << 1)
}

func HiDiffBit(a, b uint64) uint64 {
	return uint64(1) << (63 - bits.LeadingZeros64(a^b))
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
