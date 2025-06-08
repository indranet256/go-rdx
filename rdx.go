package rdx

import (
	"bytes"
	"errors"
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
	id.seq, id.src = UnzipUint64Pair(pair)
	return
}

func WriteRDX(data []byte, lit byte, id ID, value []byte) []byte {
	pair := ZipUint64Pair(id.seq, id.src)
	return WriteTLKV(data, lit, pair, value)
}

type Iter struct {
	Lit   byte
	Id    ID
	Value []byte
	Rest  []byte
	Last  []byte
}

func (i *Iter) Next() (err error) {
	rest := i.Rest
	i.Lit, i.Id, i.Value, i.Rest, err = ReadRDX(rest)
	i.Last = rest[:len(rest)-len(i.Rest)]
	return err
}

type Heap []*Iter

func (ih Heap) Up(a int, z Compare) {
	for {
		b := (a - 1) / 2 // parent
		if b == a || z(ih[a], ih[b]) >= Eq {
			break
		}
		ih[b], ih[a] = ih[a], ih[b]
		a = b
	}
}

func (ih Heap) Down(i0 int, z Compare) bool {
	n := len(ih)
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && z(ih[j2], ih[j1]) < Eq {
			j = j2 // = 2*i + 2  // right child
		}
		if z(ih[j], ih[i]) >= Eq {
			break
		}
		ih[i], ih[j] = ih[j], ih[i]
		i = j
	}
	return i > i0
}

type Merge func(data []byte, bare Heap) ([]byte, error)

func MergeF(data []byte, bare Heap) ([]byte, error) {
	var max float64
	var win []byte
	for i, b := range bare {
		n := UnzipFloat64(b.Value)
		if i == 0 || n > max {
			max = n
			win = b.Value
		}
	}
	data = append(data, win...)
	return data, nil
}

func MergeI(data []byte, bare Heap) ([]byte, error) {
	var max int64
	var win []byte
	for i, b := range bare {
		n := UnzipInt64(b.Value)
		if i == 0 || n > max {
			max = n
			win = b.Value
		}
	}
	data = append(data, win...)
	return data, nil
}

func MergeR(data []byte, bare Heap) ([]byte, error) {
	var max ID
	var win []byte
	for i, b := range bare {
		n := UnzipID(b.Value)
		if i == 0 || max.Compare(n) < 0 {
			max = n
			win = b.Value
		}
	}
	data = append(data, win...)
	return data, nil
}

func MergeS(data []byte, bare Heap) ([]byte, error) {
	var win []byte
	for i, b := range bare {
		if i == 0 || bytes.Compare(win, b.Value) < 0 {
			win = b.Value
		}
	}
	data = append(data, win...)
	return data, nil
}

func MergeT(data []byte, bare Heap) ([]byte, error) {
	return MergeS(data, bare)
}

func MergeP(data []byte, its Heap) (ret []byte, err error) {
	ret = data
	for err == nil && len(its) > 0 {
		eqs := 1
		for i := 1; i < len(its); i++ {
			z := CompareLWW(its[0], its[i])
			if z < Eq {
				its[i], its[0] = its[0], its[i]
				eqs = 1
			} else if z == Eq {
				its[i], its[eqs] = its[eqs], its[i]
				eqs++
			}
		}
		ret, err = MergeX(ret, its[:eqs])
		for i := 0; i < len(its) && err == nil; i++ {
			if len(its[i].Rest) == 0 {
				its[i] = its[len(its)-1]
				its = its[:len(its)-1]
				i--
			} else {
				err = its[i].Next()
			}
		}
	}
	return
}

func MergeL(data []byte, bare Heap) ([]byte, error) {
	return data, nil
}

func MergeE(data []byte, bare Heap) ([]byte, error) {
	return data, nil
}

func MergeX(data []byte, bare Heap) ([]byte, error) {
	return data, nil
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
	return Eq
}

func CompareFloat(a *Iter, b *Iter) int {
	return Eq
}

func CompareInteger(a *Iter, b *Iter) int {
	return Eq
}

func CompareReference(a *Iter, b *Iter) int {
	return Eq
}

func CompareString(a *Iter, b *Iter) int {
	return Eq
}

func CompareTerm(a *Iter, b *Iter) int {
	return CompareString(a, b)
}

func CompareLinear(a *Iter, b *Iter) int {
	return Eq
}

func CompareEuler(a *Iter, b *Iter) int {
	return Eq
}

func CompareMultix(a *Iter, b *Iter) int {
	return Eq
}

func mergeNext(data []byte, heap Heap, Z Compare) ([]byte, Heap, error) {
	var _ins [MaxInputs][]byte
	ins := _ins[:]
	z := Eq
	var cur *Iter

	for Less < z {
		if z != Eq {
			ins = _ins[:]
		}
		ins = append(ins, heap[0].Value)
		cur = heap[0]
		if len(heap[0].Rest) == 0 {
			l := len(heap) - 1
			heap[0], heap[l] = heap[l], heap[0]
			heap = heap[:l]
			if len(heap) == 0 {
				break
			}
		}
		heap[0].Next()
		heap.Down(0, Z)
		z = Z(cur, heap[0])
	}

	var err error = nil
	if len(ins) == 1 {
		data = append(data, ins[0]...)
	} else {
		data, err = merge(data, ins, Z) // FIXME
	}
	return data, heap, err
}

func merge(data []byte, inputs [][]byte, Z Compare) (res []byte, err error) {
	heap := make(Heap, 0, len(inputs))
	res = data
	for _, i := range inputs {
		it := Iter{Rest: i}
		it.Next()
		heap = append(heap, &it)
		heap.Up(len(heap)-1, Z)
	}
	for len(heap) > 0 && err == nil {
		res, heap, err = mergeNext(res, heap, Z)
	}
	return
}

func mergeX(data []byte, heap Heap) (ret []byte, err error) {
	var vals Heap
	ret = data
	// FIXME open
	switch heap[0].Lit {
	case Float:
		ret, err = MergeF(ret, vals)
	case Integer:
		ret, err = MergeI(ret, vals)
	case Reference:
		ret, err = MergeR(ret, vals)
	case String:
		ret, err = MergeS(ret, vals)
	case Term:
		ret, err = MergeT(ret, vals)
	case Tuple:
		ret, err = MergeP(ret, vals)
	case Linear:
		ret, err = MergeL(ret, vals)
	case Euler:
		ret, err = MergeE(ret, vals)
	case Multix:
		ret, err = MergeX(ret, vals)
	default:
		ret, err = nil, ErrBadRDXRecord
	}
	// FIXME close
	return
}
