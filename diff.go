package rdx

import (
	"bytes"
	"errors"
	"io"
	"slices"
)

var ErrBadOp = errors.New("bad operation for the context")

const (
	Keep    = '='
	Insert  = '+'
	Remove  = '-'
	Replace = '*'
	Update  = '/'
	Over    = '.'
)

type DiffProgress struct {
	prev, prnt int
	eq, in, rm int
	told, tneu int
	plex       byte
	act        byte
}

func (a *DiffProgress) Less(b *DiffProgress) bool {
	return (a.in + a.rm) < (b.in + b.rm)
}

func (a *DiffProgress) OldPos() int {
	return a.eq + a.rm
}

func (a *DiffProgress) NeuPos() int {
	return a.eq + a.in
}

type Diff struct {
	Orig []byte
	Old  []byte
	Neu  []byte
	log  []DiffProgress
	heap []int
	path []byte
	adv  map[int]int
}

func (h Diff) Len() int {
	return len(h.heap)
}
func (h Diff) Less(a, b int) bool {
	aa := h.heap[a]
	bb := h.heap[b]
	return h.log[aa].Less(&h.log[bb])
}
func (h Diff) Swap(a, b int) {
	h.heap[a], h.heap[b] = h.heap[b], h.heap[a]
}
func (h *Diff) Push(progress DiffProgress) {
	l := len(h.log)
	h.log = append(h.log, progress)
	h.heap = append(h.heap, l)
	HeapUp(h)
}
func (h *Diff) Pop() (ret int) {
	ret = h.heap[0]
	last := len(h.heap) - 1
	h.heap[0] = h.heap[last]
	h.heap = h.heap[:last]
	HeapDown(h)
	return
}

func (d *Diff) Step(at int) (err error) {
	t := d.log[at].plex
	switch t {
	case Tuple:
		return d.TupleStep(at)
	case Linear:
		return d.LinearStep(at)
	case Euler:
		return d.EulerStep(at)
	case Multix:
		return d.MultixStep(at)
	default:
		panic("bad RDX type")
	}
}

func (d *Diff) PushParent(at int) (err error) {
	a := &d.log[at]
	if a.prnt == -1 {
		panic("can't be here")
		return nil
	}
	p := &d.log[a.prnt]
	d.Push(DiffProgress{
		prnt: p.prnt,
		prev: at,
		eq:   a.eq,
		in:   a.in,
		rm:   a.rm,
		told: p.told,
		tneu: p.tneu,
		plex: p.plex,
		act:  Over,
	})
	return nil
}

func (d *Diff) PushInto(at int, old, neu *Iter) (err error) {
	a := &d.log[at]
	oldhead := len(old.Last) - len(old.Value)
	neuhead := len(neu.Last) - len(neu.Value)
	d.Push(DiffProgress{
		prnt: at,
		prev: at,
		eq:   a.eq + 1,
		in:   a.in + neuhead - 1,
		rm:   a.rm + oldhead - 1,
		told: a.OldPos() + len(old.Last),
		tneu: a.NeuPos() + len(neu.Last),
		plex: old.Lit(),
		act:  Update,
	})
	return nil
}

func (d *Diff) PushKeep(at int, old, neu *Iter) (err error) {
	a := &d.log[at]
	l := len(old.Last)
	d.Push(DiffProgress{
		prnt: a.prnt,
		prev: at,
		eq:   a.eq + l,
		in:   a.in,
		rm:   a.rm,
		told: a.told,
		tneu: a.tneu,
		plex: a.plex,
		act:  Keep,
	})
	return nil
}

func (d *Diff) PushReplace(at int, old, neu *Iter) (err error) {
	a := &d.log[at]
	lold := len(old.Last)
	lneu := len(neu.Last)
	d.Push(DiffProgress{
		prnt: a.prnt,
		prev: at,
		eq:   a.eq,
		in:   a.in + lneu,
		rm:   a.rm + lold,
		told: a.told,
		tneu: a.tneu,
		plex: a.plex,
		act:  Replace,
	})
	return nil
}

func (d *Diff) PushInsert(at int, old, neu *Iter) (err error) {
	a := &d.log[at]
	lneu := len(neu.Last)
	d.Push(DiffProgress{
		prnt: a.prnt,
		prev: at,
		eq:   a.eq,
		in:   a.in + lneu,
		rm:   a.rm,
		told: a.told,
		tneu: a.tneu,
		plex: a.plex,
		act:  Insert,
	})
	return nil
}

func (d *Diff) PushRemove(at int, old, neu *Iter) (err error) {
	a := &d.log[at]
	lold := len(old.Last)
	d.Push(DiffProgress{
		prnt: a.prnt,
		prev: at,
		eq:   a.eq,
		in:   a.in,
		rm:   a.rm + lold,
		told: a.told,
		tneu: a.tneu,
		plex: a.plex,
		act:  Remove,
	})
	return nil
}

func (d *Diff) iters(at int) (old, neu Iter, err error) {
	p := &d.log[at]
	old.Rest = d.Old[p.OldPos():p.told]
	if len(old.Rest) > 0 {
		err = old.Next()
		if err != nil {
			return
		}
	}
	neu.Rest = d.Neu[p.NeuPos():p.tneu]
	if len(neu.Rest) > 0 {
		err = neu.Next()
		if err != nil {
			return
		}
	}
	return
}

func (d *Diff) TupleStep(at int) (err error) {
	old, neu, e := d.iters(at)
	if e != nil {
		return e
	}
	if len(old.Last) == 0 && len(neu.Last) == 0 {
		return d.PushParent(at)
	}
	if len(old.Last) == 0 {
		return d.PushInsert(at, &old, &neu)
	}
	if len(neu.Last) == 0 {
		return d.PushRemove(at, &old, &neu)
	}
	if bytes.Equal(old.Last, neu.Last) {
		return d.PushKeep(at, &old, &neu)
	}
	err = d.PushReplace(at, &old, &neu)
	if err == nil && neu.Lit() == old.Lit() && IsPLEX(old.Lit()) {
		err = d.PushInto(at, &old, &neu)
	}
	return
}

func (d *Diff) LinearStep(at int) (err error) {
	old, neu, e := d.iters(at)
	if e != nil {
		return e
	}
	if len(old.Last) == 0 && len(neu.Last) == 0 {
		return d.PushParent(at)
	}
	if bytes.Equal(old.Last, neu.Last) {
		return d.PushKeep(at, &old, &neu)
	}
	if len(old.Last) > 0 {
		err = d.PushRemove(at, &old, &neu)
	}
	if err == nil && len(neu.Last) > 0 {
		err = d.PushInsert(at, &old, &neu)
	}
	if err == nil && neu.Lit() == old.Lit() && IsPLEX(old.Lit()) {
		err = d.PushInto(at, &old, &neu)
	}
	return
}

func (d *Diff) EulerStep(at int) (err error) {
	old, neu, e := d.iters(at)
	if e != nil {
		return e
	}
	if len(old.Last) == 0 {
		return d.PushInsert(at, &old, &neu)
	} else if len(neu.Last) == 0 {
		return d.PushRemove(at, &old, &neu)
	}
	z := CompareEuler(&old, &neu)
	if z > Eq {
		err = d.PushInsert(at, &old, &neu)
	} else if z < Eq {
		err = d.PushRemove(at, &old, &neu)
	} else {
		if bytes.Compare(old.Last, neu.Last) == 0 {
			err = d.PushKeep(at, &old, &neu)
		} else {
			err = d.PushReplace(at, &old, &neu)
			if err == nil && neu.Lit() == old.Lit() && IsPLEX(old.Lit()) {
				err = d.PushInto(at, &old, &neu)
			}
		}
	}
	return
}

func (d *Diff) MultixStep(at int) (err error) {
	old, neu, e := d.iters(at)
	if e != nil {
		return e
	}
	if len(old.Last) == 0 {
		return d.PushInsert(at, &old, &neu)
	} else if len(neu.Last) == 0 {
		return d.PushRemove(at, &old, &neu)
	}
	z := CompareMultix(&old, &neu)
	if z > Eq {
		err = d.PushInsert(at, &old, &neu)
	} else if z < Eq {
		err = d.PushRemove(at, &old, &neu)
	} else { // todo into
		if bytes.Compare(old.Last, neu.Last) == 0 {
			err = d.PushKeep(at, &old, &neu)
		} else {
			err = d.PushReplace(at, &old, &neu)
		}
	}
	return
}

func (d *Diff) Solve() (err error) {
	// init
	d.Push(DiffProgress{
		prnt: -1,
		prev: -1,
		told: len(d.Old),
		tneu: len(d.Neu),
		plex: Tuple,
	})
	for err == nil && len(d.heap) > 0 {
		pop := d.heap[0]
		diff := &d.log[pop]
		if diff.eq+diff.rm >= len(d.Old) &&
			diff.eq+diff.in >= len(d.Neu) {
			break
		}
		err = d.Step(d.Pop())
	}
	d.path = make([]byte, 0, 1024)
	for i := d.heap[0]; i > 0; i = d.log[i].prev {
		d.path = append(d.path, d.log[i].act)
	}
	d.heap = nil
	d.log = nil
	slices.Reverse(d.path)
	return
}

var KeepId = []byte{}
var RemoveId = []byte{1}
var InsertId = []byte{2}
var ReplaceId = []byte{3}
var UpdateId = []byte{4}

func (d *Diff) hili(data, old, neu []byte, p int, stack *Marks) (np int, out []byte, err error) {
	out = data
	o := Iter{Rest: old}
	n := Iter{Rest: neu}
	np = p
	for err == nil && np < len(d.path) {
		act := d.path[np]
		switch act {
		case Keep:
			_ = o.Next()
			out = append(out, o.Last...)
			_ = n.Next()
		case Insert:
			_ = n.Next()
			out = WriteTLKV(out, n.Lit(), InsertId, n.Value)
		case Remove:
			_ = o.Next()
			out = WriteTLKV(out, o.Lit(), RemoveId, o.Value)
		case Replace: // todo nicer replace hili
			_ = o.Next()
			out = WriteTLKV(out, o.Lit(), RemoveId, o.Value)
			_ = n.Next()
			out = WriteTLKV(out, n.Lit(), InsertId, n.Value)
		case Update:
			_ = o.Next()
			_ = n.Next()
			out = OpenTLV(out, o.Lit(), stack)
			out = append(out, byte(len(UpdateId)))
			out = append(out, UpdateId...)
			np, out, err = d.hili(out, o.Value, n.Value, np+1, stack)
			np -= 1
			out, _ = CloseTLV(out, o.Lit(), stack)
		case Over:
			err = io.EOF
		default:
			panic("bad action")
		}
		np++
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func (d *Diff) Hili() (out []byte, err error) {
	var stack Marks
	_, out, err = d.hili(nil, d.Old, d.Neu, 0, &stack)
	return
}

var EmptyTuple = []byte{'p', 1, 0}

func (d *Diff) diffP(data, old, neu []byte, p int, stack *Marks) (np int, out []byte, err error) {
	out = data
	o := Iter{Rest: old}
	n := Iter{Rest: neu}
	np = p
	for err == nil && np < len(d.path) {
		act := d.path[np]
		switch act {
		case Keep:
			_ = o.Next()
			_ = n.Next()
			if len(out) == len(data) && len(*stack) > 1 && (*stack)[len(*stack)-2].Lit == Euler {
				out = append(out, o.Last...)
			} else {
				out = append(out, EmptyTuple...)
			}
		case Insert:
			err = n.Next()
			if err != nil {
				return
			}
			out = WriteRDX(out, n.Lit(), n.Id, n.Value)
		case Remove:
			_ = o.Next()
			_ = n.Next()
			id := ID{Seq: o.Id.Seq | 1}
			out = WriteRDX(out, Tuple, id, nil)
		case Replace:
			_ = o.Next()
			_ = n.Next()
			id := ID{Seq: (o.Id.Seq | 1) + 1}
			out = WriteRDX(out, n.Lit(), id, n.Value)
		case Update:
			_ = o.Next()
			_ = n.Next()
			out = OpenTLV(out, o.Lit(), stack)
			id := ZipID(o.Id)
			out = append(out, byte(len(id)))
			out = append(out, id...)
			np, out, err = d.diff(out, o.Lit(), o.Value, n.Value, np+1, stack) // todo bad signature
			np -= 1
			out, _ = CloseTLV(out, o.Lit(), stack)
		case Over:
			err = io.EOF
		default:
			panic("bad action")
		}
		np++
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func fitSeq(a, b uint64) (ret uint64) {
	if b == 0 {
		ret = (a &^ 1) + (1 << 12)
	} else if b >= a+(1<<12) {
		ret = (a &^ 1) + (1 << 7)
	} else if b >= a+(1<<9) {
		ret = (a &^ 1) + (1 << 7)
	} else {
		ret = 1 << 12
	}
	return
}

func (d *Diff) diffL(data, old, neu []byte, p int, stack *Marks) (np int, out []byte, err error) {
	out = data
	oldit := Iter{Rest: old}
	neuit := Iter{Rest: neu}
	nextit := oldit
	_ = nextit.Next()
	heads := make([]Iter, 0, 32)
	np = p
	var oldseq uint64
	lastold := -1
	for err == nil && np < len(d.path) {
		act := d.path[np]
		if act != Insert {
			oldit = nextit
			_ = nextit.Next()
			for !oldit.IsLive() {
				oldit = nextit
				_ = nextit.Next()
			}
			oldseq = oldit.Id.Seq
			if nextit.Id.Compare(oldit.Id) < Eq {
				for len(heads) > 0 && heads[len(heads)-1].Id.Compare(nextit.Id) < Eq {
					heads = heads[:len(heads)-1]
				}
				heads = append(heads, nextit)
			}
		}
		if act != Remove {
			_ = neuit.Next()
		}
		if act != Keep && len(heads) > 0 {
			for _, h := range heads {
				out = append(out, h.Last...)
				lastold = len(h.Rest)
			}
			heads = heads[:0]
		}
		switch act {
		case Keep:
			if oldit.Id.IsZero() {
				out = append(out, oldit.Last...)
				lastold = len(oldit.Rest)
			}
		case Insert:
			newid := ID{Seq: fitSeq(oldseq, nextit.Id.Seq)}
			if newid.Seq < oldseq {
				if lastold < len(oldit.Rest) {
					out = append(out, oldit.Last...)
					lastold = len(oldit.Rest)
				}
			}
			oldseq = newid.Seq
			out = WriteRDX(out, neuit.Lit(), newid, neuit.Value)
		case Remove:
			id := oldit.Id
			id.Seq |= 1
			val := oldit.Value
			if IsPLEX(oldit.Lit()) {
				val = nil
			}
			out = WriteRDX(out, oldit.Lit(), id, val)
			lastold = len(oldit.Rest)
		case Replace:
			id := ID{Seq: (oldit.Id.Seq | 1) + 1}
			out = WriteRDX(out, neuit.Lit(), id, neuit.Value)
		case Update:
			out = OpenTLV(out, oldit.Lit(), stack)
			id := ZipID(oldit.Id)
			out = append(out, byte(len(id)))
			out = append(out, id...)
			np, out, err = d.diff(out, oldit.Lit(), oldit.Value, neuit.Value, np+1, stack) // todo bad signature
			np -= 1
			out, _ = CloseTLV(out, oldit.Lit(), stack)
			lastold = len(oldit.Rest)
		case Over:
			err = io.EOF
		default:
			panic("bad action")
		}
		np++
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func (d *Diff) diffE(data, old, neu []byte, p int, stack *Marks) (np int, out []byte, err error) {
	out = data
	o := Iter{Rest: old}
	n := Iter{Rest: neu}
	np = p
	for err == nil && np < len(d.path) {
		act := d.path[np]
		switch act {
		case Keep:
			_ = o.Next()
			_ = n.Next()
		case Insert:
			err = n.Next()
			if err != nil {
				return
			}
			out = append(out, n.Last...)
		case Remove:
			_ = o.Next()
			id := o.Id
			id.Seq |= 1
			if o.Lit() == Tuple && len(o.Value) > 0 {
				key := Iter{Rest: o.Value}
				_ = key.Next()
				out = WriteRDX(out, key.Lit(), id, key.Value)
			} else { // todo plex shell
				out = WriteRDX(out, o.Lit(), id, o.Value)
			}
		case Replace:
			_ = o.Next()
			_ = n.Next()
			id := ID{Seq: (o.Id.Seq | 1) + 1}
			out = WriteRDX(out, n.Lit(), id, n.Value)
		case Update:
			_ = o.Next()
			_ = n.Next()
			out = OpenTLV(out, o.Lit(), stack)
			id := ZipID(o.Id)
			out = append(out, byte(len(id)))
			out = append(out, id...)
			np, out, err = d.diff(out, o.Lit(), o.Value, n.Value, np+1, stack) // todo bad signature
			np -= 1
			out, _ = CloseTLV(out, o.Lit(), stack)
		case Over:
			err = io.EOF
		default:
			panic("bad action")
		}
		np++
	}
	if err == io.EOF {
		err = nil
	}
	return
}

func (d *Diff) diff(data []byte, t byte, old, neu []byte, p int, stack *Marks) (np int, out []byte, err error) {
	out = data
	switch t {
	case Tuple:
		return d.diffP(data, old, neu, p, stack)
	case Euler:
		return d.diffE(data, old, neu, p, stack)
	case Linear:
		return d.diffL(data, old, neu, p, stack)
	default:
		return
	}
}

func (d *Diff) Diff() (diff []byte, err error) {
	stack := make(Marks, 0, MaxNesting)
	_, diff, err = d.diffP(nil, d.Orig, d.Neu, 0, &stack)
	return
}
