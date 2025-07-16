package rdx

type Iter struct {
	Id    ID
	Value []byte
	Rest  []byte
	Last  []byte
}

func (i *Iter) Lit() byte {
	if len(i.Last) == 0 {
		return 0
	}
	return i.Last[0] & ^CaseBit
}

func (i Iter) NextStep(j *Iter) (err error) {
	if len(i.Rest) == 0 {
		err = ErrEoF
	} else {
		rest := i.Rest
		_, j.Id, j.Value, j.Rest, err = ReadRDX(rest)
		j.Last = rest[:len(rest)-len(j.Rest)]
	}
	return
}

func (i *Iter) Next() (err error) {
	err = i.NextStep(i)
	return
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
		heap.LastUp(z)
	}
	return
}

func (heap *Heap) LastUp(z Compare) {
	heap.Up(len(*heap)-1, z)
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

func (heap *Heap) MergeNext(data []byte, Z Compare) ([]byte, error) {
	var err error = nil
	h := *heap

	eqlen := heap.EqUp(Z)
	if eqlen == 1 {
		data = append(data, h[0].Last...)
	} else {
		eqs := h[:eqlen]
		data, err = mergeSameSpotElements(data, eqs)
	}
	if err == nil {
		h, err = h.NextK(eqlen, Z)
	}
	*heap = h
	return data, err
}

func HeapMerge(data []byte, inputs [][]byte, Z Compare) (res []byte, err error) {
	var heap Heap
	heap, err = Heapize(inputs, Z)
	res = data
	for len(heap) > 0 && err == nil {
		res, err = heap.MergeNext(res, Z)
	}
	return
}
