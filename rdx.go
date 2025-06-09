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
	if len(i.Rest) == 0 {
		i.Lit = 0
		i.Id = ID{}
		i.Value = nil
		i.Rest = nil
		i.Last = nil
		return ErrEoF
	}
	rest := i.Rest
	i.Lit, i.Id, i.Value, i.Rest, err = ReadRDX(rest)
	i.Last = rest[:len(rest)-len(i.Rest)]
	return err
}

type Heap []*Iter

func Heapize(rdx [][]byte, z Compare) (heap Heap, err error) {
	heap = make(Heap, 0, len(rdx))
	for _, r := range rdx {
		if len(r) == 0 {
			continue
		}
		i := Iter{Rest: r}
		err = i.Next()
		if err != nil {
			break
		}
		heap = append(heap, &i)
		heap.Up(len(heap)-1, z)
	}
	return
}

func Iterize(rdx [][]byte) (heap Heap, err error) {
	heap = make(Heap, 0, len(rdx))
	for _, r := range rdx {
		if len(r) == 0 {
			continue
		}
		i := Iter{Rest: r}
		err = i.Next()
		if err != nil {
			break
		}
		heap = append(heap, &i)
	}
	return
}

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

func (ih Heap) EqUp(z Compare) (eqs int) {
	if len(ih) < 2 {
		return len(ih)
	}
	q := make([]int, 0, MaxInputs)
	q = append(q, 1, 2)
	eqs = 1
	for len(q) > 0 && q[0] < len(ih) {
		n := q[0]
		if Eq == z(ih[0], ih[n]) {
			j1 := 2*n + 1
			q = append(q, j1, j1+1)
			ih[eqs], ih[n] = ih[n], ih[eqs]
			eqs++
		}
		q = q[1:]
	}
	return
}

func (ih Heap) Remove(i int) Heap {
	l := len(ih) - 1
	ih[l], ih[i] = ih[i], ih[l]
	return ih[:l]
}

func (ih Heap) NextK(k int, z Compare) (nh Heap, err error) {
	for i := k - 1; i >= 0; i-- {
		if len(ih[i].Rest) == 0 {
			ih = ih.Remove(i)
			if i < len(ih) {
				ih.Down(i, z)
			}
		} else {
			err = ih[i].Next()
			if err != nil {
				break
			}
			ih.Down(i, z)
		}
	}
	return ih, err
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
	ret = data
	its, err := Iterize(bare)
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
		} // TODO 1
		ret, err = mergeElementsSame(ret, its[:eqs])
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

func mergeElementsL(data []byte, bare [][]byte) ([]byte, error) {
	return data, nil
}

func mergeElementsE(data []byte, bare [][]byte) ([]byte, error) {
	return heapMerge(data, bare, CompareEuler)
}

func mergeElementsX(data []byte, bare [][]byte) ([]byte, error) {
	return heapMerge(data, bare, CompareMultix)
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

func CompareLinear(a *Iter, b *Iter) int {
	return Eq
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
	if a.Id.src < b.Id.src {
		return Less
	} else if a.Id.src < b.Id.src {
		return Grtr
	}
	return Eq
}

func heapTopIDs(heap Heap) Heap {
	if len(heap) == 0 {
		return heap
	}
	eqs := 1
	for i := 1; i < len(heap); i++ {
		z := heap[i].Id.Compare(heap[0].Id)
		if z > Eq || (z == Eq && CompareType(heap[i], heap[0]) > Eq) {
			heap[0], heap[i] = heap[i], heap[0]
			eqs = 1
		} else if z == Eq {
			heap[eqs], heap[i] = heap[i], heap[eqs]
			eqs++
		}
	}
	return heap[:eqs]
}

func heapMergeNext(data []byte, heap Heap, Z Compare) ([]byte, Heap, error) {
	var err error = nil

	eqlen := heap.EqUp(Z)
	if eqlen == 1 {
		data = append(data, heap[0].Last...)
	} else {
		eqs := heap[:eqlen]
		tops := heapTopIDs(eqs)
		data, err = mergeElementsSame(data, tops)
	}
	if err == nil {
		heap, err = heap.NextK(eqlen, Z)
	}

	return data, heap, err
}

func heapMerge(data []byte, inputs [][]byte, Z Compare) (res []byte, err error) {
	var heap Heap
	heap, err = Heapize(inputs, Z)
	res = data
	for len(heap) > 0 && err == nil {
		res, heap, err = heapMergeNext(res, heap, Z)
	}
	return
}
