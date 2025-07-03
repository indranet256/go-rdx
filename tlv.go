package rdx

import (
	"encoding/binary"
	"errors"
)

var (
	ErrIncomplete = errors.New("incomplete data")
	ErrBadRecord  = errors.New("bad TLV record format")
	ErrBadHeader  = errors.New("bad BRIX file header")
	ErrBadNesting = errors.New("bad TLV nesting")
)

const CaseBit uint8 = 'a' - 'A'

const MaxRecLen = 0x7fffffff

func ReadTLV(data []byte) (lit byte, value, rest []byte, err error) {
	if len(data) == 0 {
		return 0, nil, nil, nil
	}
	dlit := data[0]
	if dlit >= 'a' && dlit <= 'z' { // short
		lit = dlit - CaseBit
		if len(data) < 2 || len(data) < 2+int(data[1]) {
			rest = data
			err = ErrIncomplete
			return
		}
		value = data[2 : 2+data[1]]
		rest = data[2+data[1]:]
	} else if dlit >= 'A' && dlit <= 'Z' { // long
		if len(data) < 5 {
			err = ErrIncomplete
			return
		}
		bl := binary.LittleEndian.Uint32(data[1:5])
		if bl > MaxRecLen || int(bl) > len(data)+5 {
			if bl > MaxRecLen {
				err = ErrBadRecord
			} else {
				err = ErrIncomplete
			}
			return
		}
		lit = dlit
		value = data[5 : 5+bl]
		rest = data[5+bl:]
	} else {
		err = ErrBadRecord
	}
	return
}

func ReadTLKV(data []byte) (lit byte, key, value, rest []byte, err error) {
	lit, value, rest, err = ReadTLV(data)
	if err == nil && len(value) > 0 {
		if len(value) < int(value[0])+1 {
			err = ErrBadRecord
		} else {
			key = value[1 : 1+value[0]]
			value = value[1+value[0]:]
		}
	}
	return
}

func WriteTLV(data []byte, lit byte, value []byte) []byte {
	ret := data
	if len(value) > 0xff {
		ret = append(ret, lit)
		ret = binary.LittleEndian.AppendUint32(ret, uint32(len(value)))
	} else {
		ret = append(ret, lit|CaseBit)
		ret = append(ret, byte(len(value)))
	}
	ret = append(ret, value...)
	return ret
}

func WriteTLKV(data []byte, lit byte, key, value []byte) []byte {
	ret := data
	if len(key) > 0xff { // todo
		key = key[:0xff]
	}
	l := len(key) + len(value) + 1
	if l > 0xff {
		ret = append(ret, lit)
		ret = binary.LittleEndian.AppendUint32(ret, uint32(l))
	} else {
		ret = append(ret, lit|CaseBit)
		ret = append(ret, byte(l))
	}
	ret = append(ret, byte(len(key)))
	ret = append(ret, key...)
	ret = append(ret, value...)
	return ret
}

type Mark struct {
	Pos  int
	Mark int
	Lit  byte
}

type Marks []Mark

func (stack *Marks) Top() *Mark {
	if len(*stack) == 0 {
		return nil
	}
	return &(*stack)[len(*stack)-1]
}

func (stack *Marks) TopLit() byte {
	if len(*stack) == 0 {
		return 0
	}
	return stack.Top().Lit
}

func (stack Marks) Len() int {
	return len(stack)
}

func SpliceTLV(data []byte, lit byte, pos int) []byte {
	l := len(data) - pos + 1
	if l <= 0xff {
		data = append(data, 0, 0, 0)
		copy(data[pos+3:], data[pos:])
		data[pos] = lit | CaseBit
		data[pos+1] = byte(l)
	} else {
		data = append(data, 0, 0, 0, 0, 0, 0)
		copy(data[pos+6:], data[pos:])
		data[pos] = lit &^ CaseBit
		binary.LittleEndian.PutUint32(data[pos+1:pos+5], uint32(l))
	}
	return data
}

func OpenTLV(data []byte, lit byte, stack *Marks) []byte {
	if lit < 'A' || lit > 'Z' {
		panic("bad Lit")
	}
	*stack = append(*stack, Mark{len(data), 0, lit})
	data = append(data, lit, 0, 0, 0, 0)
	return data
}

func OpenShortTLV(data []byte, lit byte, stack *Marks) []byte {
	if lit < 'A' || lit > 'Z' {
		panic("bad Lit")
	}
	lit |= CaseBit
	*stack = append(*stack, Mark{len(data), 0, lit})
	data = append(data, lit, 0)
	return data
}

func upper(lit byte) byte {
	return lit &^ CaseBit
}

func CloseTLV(data []byte, lit byte, stack *Marks) (ret []byte, err error) {
	if len(*stack) == 0 {
		return nil, ErrBadNesting
	}
	nl := len(*stack) - 1
	last := (*stack)[nl]
	*stack = (*stack)[:nl]
	if upper(last.Lit) != lit || last.Pos+2 > len(data) || data[last.Pos]&^CaseBit != lit {
		return nil, ErrBadNesting
	}
	fact := len(data) - last.Pos
	if 0 == (data[last.Pos] & CaseBit) { // A
		fact -= 5
		if fact < 0 {
			return nil, ErrBadNesting
		}
		if fact < 0x100 {
			copy(data[last.Pos+2:len(data)-3], data[last.Pos+5:len(data)])
			data = data[:len(data)-3]
			data[last.Pos] |= CaseBit
		}
	} else { // a
		fact -= 2
		if fact < 0 {
			return nil, ErrBadNesting
		}
		if fact >= 0x100 {
			l := len(data)
			data = append(data, 0, 0, 0)
			copy(data[last.Pos+5:len(data)], data[last.Pos+2:l])
			data[last.Pos] &= ^CaseBit
		}
	}
	if fact < 0x100 {
		data[last.Pos+1] = byte(fact)
	} else {
		binary.LittleEndian.PutUint32(data[last.Pos+1:], uint32(fact))
	}
	return data, nil
}
