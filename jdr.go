package rdx

import (
	"strconv"
)

type JDRstate struct {
	jdr []byte
	rdx []byte

	stack Marks
	pre   byte
	val   []byte

	mark int
	line int
	col  int
}

var emptyTuple []byte = []byte{'p', 1, 0}

const inlineTuple = 'p'

const (
	StyleStampSpace    = 1 << 8
	StyleStamps        = 1 << 9
	StyleCommaNL       = 1 << 10
	StyleTopCommaNL    = 1 << 11
	StyleIndentTab     = 1 << 12
	StyleIndentSpace4  = 1 << 13
	StyleTrailingComma = 1 << 14
	StyleBracketTuples = 1 << 15
	StyleSkipComma     = 1 << 16
)

const StyleCommaSpacers = StyleCommaNL | StyleIndentSpace4 | StyleIndentTab

func JDRonNL(tok []byte, state *JDRstate) error {
	state.line++
	return nil
}
func JDRonUtf8cp1(tok []byte, state *JDRstate) error { return nil }

func JDRonUtf8cp2(tok []byte, state *JDRstate) error {
	var cp uint32
	cp = uint32(tok[0]) & 0x1f
	cp = (cp << 6) | (uint32(tok[1]) & 0x3f)
	if cp >= 0xd800 || cp < 0xe000 {
		return ErrBadUtf8
	}
	return nil
}

func JDRonUtf8cp3(tok []byte, state *JDRstate) error {
	var cp uint32
	cp = uint32(tok[0]) & 0x1f
	cp = (cp << 6) | uint32(tok[1])&0x3f
	cp = (cp << 6) | uint32(tok[2])&0x3f
	if cp >= 0xd800 || cp < 0xe000 {
		return ErrBadUtf8
	}
	return nil
}
func JDRonUtf8cp4(tok []byte, state *JDRstate) error {
	// TODO codepoint ranges
	return nil
}

func (state *JDRstate) closeInline() (err error) {
	if state.pre == ':' {
		state.rdx = append(state.rdx, emptyTuple...)
	}
	state.pre = inlineTuple
	// tlv close
	state.rdx, err = CloseTLV(state.rdx, Tuple, &state.stack)
	return
}

func JDRonFIRST0(tok []byte, state *JDRstate, lit byte) (err error) {
	if state.stack.TopLit() == inlineTuple && state.pre != ':' {
		err = state.closeInline()
	}
	state.rdx = OpenTLV(state.rdx, lit, &state.stack)
	state.val = tok
	return
}

func JDRonInt(tok []byte, state *JDRstate) error { return JDRonFIRST0(tok, state, Integer) }

func JDRonFloat(tok []byte, state *JDRstate) error { return JDRonFIRST0(tok, state, Float) }

func JDRonTerm(tok []byte, state *JDRstate) error { return JDRonFIRST0(tok, state, Term) }

func JDRonRef(tok []byte, state *JDRstate) error { return JDRonFIRST0(tok, state, Reference) }

func JDRonString(tok []byte, state *JDRstate) error { return JDRonFIRST0(tok, state, String) }

func JDRonMLString(tok []byte, state *JDRstate) error { return JDRonFIRST0(tok, state, String) }

func JDRonStamp(tok []byte, state *JDRstate) error {
	idstr := tok[1:]
	id, _ := ParseID(idstr)
	zip := ZipID(id)
	state.rdx = append(state.rdx, byte(len(zip)))
	state.rdx = append(state.rdx, zip...)
	return nil
}

func JDRonNoStamp(tok []byte, state *JDRstate) error {
	state.rdx = append(state.rdx, 0)
	return nil
}

func JDRonFIRST(tok []byte, state *JDRstate) (err error) {
	lit := state.stack.TopLit()
	switch lit {
	case Float:
		f, _ := strconv.ParseFloat(string(state.val), 64)
		state.rdx = append(state.rdx, ZipFloat64(f)...)
	case Integer:
		i, _ := strconv.ParseInt(string(state.val), 10, 64)
		state.rdx = append(state.rdx, ZipInt64(i)...)
	case Reference:
		id, _ := ParseID(state.val)
		state.rdx = append(state.rdx, ZipID(id)...)
	case String:
		state.rdx = appendUnescaped(state.rdx, state.val[1:len(state.val)-1])
	case Term:
		state.rdx = append(state.rdx, state.val...)
	}
	state.rdx, err = CloseTLV(state.rdx, lit, &state.stack)
	state.pre = '1'
	return err
}

// . . .

func retrofitRDXTuple(state *JDRstate) (err error) {
	if state.pre == 0 {
		state.rdx = OpenShortTLV(state.rdx, Tuple, &state.stack)
		state.rdx = append(state.rdx, 0) //id
		err = appendRDXEmptyTuple(state)
	} else {
		if len(state.stack) == cap(state.stack) {
			return ErrBadState
		}
		state.stack = state.stack[:len(state.stack)+1]
		last := &state.stack[len(state.stack)-1]
		last.Lit = 'p'
		last.Mark = 0
		state.rdx = append(state.rdx, 0, 0, 0)
		copy(state.rdx[last.Pos+3:len(state.rdx)], state.rdx[last.Pos:len(state.rdx)-3])
		state.rdx[last.Pos] = 'p'
		state.rdx[last.Pos+1] = 0
		state.rdx[last.Pos+2] = 0
	}
	return
}

func appendRDXEmptyTuple(state *JDRstate) (err error) {
	state.rdx = append(state.rdx, emptyTuple...)
	return nil
}

func closeInlineTuple(state *JDRstate) (err error) {
	if state.pre == ':' {
		state.rdx = append(state.rdx, emptyTuple...)
	}
	state.rdx, err = CloseTLV(state.rdx, Tuple, &state.stack)
	return
}

func JDRonSemicolon(tok []byte, state *JDRstate) (err error) {
	if state.stack.TopLit() == inlineTuple {
		if state.pre == ':' {
			err = appendRDXEmptyTuple(state)
		}
		err = closeInlineTuple(state)
	} else {
		p := 0
		if len(state.stack) > 0 {
			last := &state.stack[len(state.stack)-1]
			p = last.Pos + 2
			if (last.Lit & CaseBit) == 0 {
				p += 3
			}
			if state.stack.Top().Mark > p {
				p = state.stack.Top().Mark
			}
		} else {
			if state.mark > p {
				p = state.mark
			}
		}
		state.rdx = SpliceTLV(state.rdx, Tuple, p)
	}
	if len(state.stack) > 0 {
		state.stack.Top().Mark = len(state.rdx)
	} else {
		state.mark = len(state.rdx) // FIXME ugly
	}
	return
}

func JDRonOpenPLEX(tok []byte, state *JDRstate, plex byte) (err error) {
	if (&(state.stack)).TopLit() == inlineTuple && state.pre != ':' {
		err = closeInlineTuple(state)
	}
	if err == nil {
		state.rdx = OpenTLV(state.rdx, plex, &state.stack)
	}
	state.pre = 0
	return
}

func JDRonClosePLEX(tok []byte, state *JDRstate, lit byte) (err error) {
	if state.stack.TopLit() == inlineTuple {
		err = closeInlineTuple(state)
	} else if state.pre == ',' {
		err = appendRDXEmptyTuple(state)
	}
	if !IsPLEX(lit) || lit != state.stack.TopLit() {
		err = ErrBadNesting
	}
	if err != nil {
		return err
	}
	if lit == Euler {
		// TODO sort
	} else if lit == Multix {
		// TODO sort
	}
	state.rdx, err = CloseTLV(state.rdx, lit, &state.stack)
	state.pre = lit
	return err
}

func JDRonOpenP(tok []byte, state *JDRstate) error {
	return JDRonOpenPLEX(tok, state, Tuple)
}
func JDRonCloseP(tok []byte, state *JDRstate) error {
	return JDRonClosePLEX(tok, state, Tuple)
}
func JDRonOpenL(tok []byte, state *JDRstate) error {
	return JDRonOpenPLEX(tok, state, Linear)
}
func JDRonCloseL(tok []byte, state *JDRstate) error {
	return JDRonClosePLEX(tok, state, Linear)
}
func JDRonOpenE(tok []byte, state *JDRstate) error {
	return JDRonOpenPLEX(tok, state, Euler)
}
func JDRonCloseE(tok []byte, state *JDRstate) error {
	return JDRonClosePLEX(tok, state, Euler)
}
func JDRonOpenX(tok []byte, state *JDRstate) error {
	return JDRonOpenPLEX(tok, state, Multix)
}
func JDRonCloseX(tok []byte, state *JDRstate) error {
	return JDRonClosePLEX(tok, state, Multix)
}
func JDRonComma(tok []byte, state *JDRstate) (err error) {
	if state.stack.TopLit() == inlineTuple {
		err = closeInlineTuple(state)
	}
	if state.pre == 0 || state.pre == ',' {
		_ = appendRDXEmptyTuple(state)
	}
	if len(state.stack) > 0 {
		state.stack.Top().Mark = len(state.rdx)
	} else {
		state.mark = len(state.rdx)
	}
	state.pre = ','
	return
}
func JDRonColon(tok []byte, state *JDRstate) (err error) {
	if state.stack.TopLit() != inlineTuple {
		err = retrofitRDXTuple(state)
	} else if state.pre == 0 || state.pre == ':' {
		err = appendRDXEmptyTuple(state)
	}
	state.pre = ':'
	return err
}

func JDRonOpen(tok []byte, state *JDRstate) error { return nil }

func JDRonClose(tok []byte, state *JDRstate) error { return nil }

func JDRonInter(tok []byte, state *JDRstate) error { return nil }

func JDRonRoot(tok []byte, state *JDRstate) (err error) {
	if state.stack.TopLit() == inlineTuple {
		err = closeInlineTuple(state)
	}
	if state.stack.Len() > 0 {
		err = ErrBadNesting
	}
	return
}

func appendUnescaped(rdx, jdr []byte) []byte {
	for len(jdr) > 0 {
		c := jdr[0]
		jdr = jdr[1:]
		if c != '\\' {
			rdx = append(rdx, c)
			continue
		}
		if len(jdr) == 0 {
			rdx = append(rdx, c)
			continue
		}
		c = jdr[0]
		jdr = jdr[1:]
		switch c {
		case 't':
			rdx = append(rdx, '\t')
		case 'r':
			rdx = append(rdx, '\r')
		case 'n':
			rdx = append(rdx, '\n')
		case 'b':
			rdx = append(rdx, '\b')
		case 'f':
			rdx = append(rdx, '\f')
		case '\\':
			rdx = append(rdx, '\\')
		case '/':
			rdx = append(rdx, '/')
		case '"':
			rdx = append(rdx, '"')
		case '0':
			rdx = append(rdx, '0')
		default:
			rdx = append(rdx, c)
		}
	}
	return rdx
}

func appendEscaped(jdr, val []byte) []byte {
	for _, a := range val {
		switch a {
		case '\t':
			jdr = append(jdr, '\\', 't')
		case '\r':
			jdr = append(jdr, '\\', 'r')
		case '\n':
			jdr = append(jdr, '\\', 'n')
		case '\b':
			jdr = append(jdr, '\\', 'b')
		case '\f':
			jdr = append(jdr, '\\', 'f')
		case '\\':
			jdr = append(jdr, '\\', '\\')
		case '"':
			jdr = append(jdr, '\\', '"')
		case 0:
			jdr = append(jdr, '\\', '0')
			// TODO \u etc
		default:
			jdr = append(jdr, a)
		}
	}
	return jdr
}

func appendJDRStamp(jdr []byte, id ID, lit byte) []byte {
	if id.IsZero() {
		return jdr
	}
	jdr = append(jdr, '@')
	jdr = append(jdr, id.String()...)
	if IsPLEX(lit) {
		jdr = append(jdr, ' ')
	}
	return jdr
}

func IsAllTerm(rdx []byte) bool {
	var err error
	for len(rdx) > 0 && err == nil {
		var lit byte
		lit, _, _, rdx, err = ReadRDX(rdx)
		if lit != Term {
			return false
		}
	}
	return err == nil
}

func IsTupleAllFIRST(rdx []byte) bool {
	var err error
	l := 0
	for len(rdx) > 0 && err == nil {
		var lit byte
		var val []byte
		lit, _, val, rdx, err = ReadRDX(rdx)
		if IsPLEX(lit) && !(lit == Tuple && len(val) == 0) {
			return false
		}
		l++
	}
	if l < 2 {
		return false
	}
	return true
}

func appendIndent(jdr []byte, style uint64) []byte {
	if 0 != (style & StyleIndentTab) {
		for i := 0; i < int(style&0xff); i++ {
			jdr = append(jdr, '\t')
		}
	} else if 0 != (style & StyleIndentSpace4) {
		for i := 0; i < int(style&0xff); i++ {
			jdr = append(jdr, ' ', ' ', ' ', ' ')
		}
	}
	return jdr
}

func appendJDRList(jdr, rdx []byte, style uint64) (res []byte, err error) {
	commanl := 0 != (style&StyleCommaNL) || (0 != (style&StyleTopCommaNL) && (style&0xff) == 0)
	for len(rdx) > 0 && err == nil {
		jdr, rdx, err = WriteJDR(jdr, rdx, style)
		if len(rdx) > 0 || 0 != (style&StyleTrailingComma) {
			jdr = append(jdr, ',')
			if commanl {
				jdr = append(jdr, '\n')
				jdr = appendIndent(jdr, style)
			}
		}
	}
	if commanl {
		jdr = append(jdr, '\n')
	}
	return jdr, err
}

func appendInlineTuple(jdr, rdx []byte, style uint64) (res []byte, err error) {
	res = jdr
	for len(rdx) > 0 && err == nil {
		res, rdx, err = WriteJDR(res, rdx, (style|StyleBracketTuples)+1)
		if len(rdx) > 0 {
			res = append(res, ':')
		}
	}
	return res, err
}

func WriteJDR(jdr, rdx []byte, style uint64) (jdr2, rest []byte, err error) {
	var lit byte
	var id ID
	var val []byte
	lit, id, val, rest, err = ReadRDX(rdx)
	if err != nil {
		return
	}
	switch lit {
	case Float:
		f := UnzipFloat64(val)
		jdr = strconv.AppendFloat(jdr, f, 'e', -1, 64)
		jdr = appendJDRStamp(jdr, id, lit)
	case Integer:
		i := UnzipInt64(val)
		jdr = strconv.AppendInt(jdr, i, 10)
		jdr = appendJDRStamp(jdr, id, lit)
	case Reference:
		i := UnzipID(val)
		jdr = append(jdr, i.String()...)
		jdr = appendJDRStamp(jdr, id, lit)
	case String:
		jdr = append(jdr, '"')
		jdr = appendEscaped(jdr, val)
		jdr = append(jdr, '"')
		jdr = appendJDRStamp(jdr, id, lit)
	case Term:
		jdr = append(jdr, val...)
		jdr = appendJDRStamp(jdr, id, lit)
	case Tuple:
		if 0 != (style&StyleBracketTuples) || !IsTupleAllFIRST(val) || !id.IsZero() {
			jdr = append(jdr, '(')
			jdr = appendJDRStamp(jdr, id, lit)
			jdr, err = appendJDRList(jdr, val, style+1)
			jdr = append(jdr, ')')
		} else {
			jdr, err = appendInlineTuple(jdr, val, style)
		}
	case Linear:
		jdr = append(jdr, '[')
		jdr = appendJDRStamp(jdr, id, lit)
		jdr, err = appendJDRList(jdr, val, style+1)
		jdr = append(jdr, ']')
	case Euler:
		jdr = append(jdr, '{')
		jdr = appendJDRStamp(jdr, id, lit)
		jdr, err = appendJDRList(jdr, val, style+1)
		jdr = append(jdr, '}')
	case Multix:
		jdr = append(jdr, '<')
		jdr = appendJDRStamp(jdr, id, lit)
		jdr, err = appendJDRList(jdr, val, style+1)
		jdr = append(jdr, '>')
	default:
		err = ErrBadRecord
	}
	jdr2 = jdr
	return
}

func WriteAllJDR(jdr, rdx []byte, style uint64) (jdr2 []byte, err error) {
	jdr2 = jdr
	for len(rdx) > 0 && err == nil {
		jdr2, rdx, err = WriteJDR(jdr2, rdx, style)
		if len(rdx) > 0 {
			jdr2 = append(jdr2, ' ')
		}
	}
	return jdr2, err
}

func ParseJDR(jdr []byte) (rdx []byte, err error) {
	state := JDRstate{
		jdr:   jdr,
		stack: make(Marks, 0, MaxNesting+1),
	}
	err = JDRlexer(&state)
	rdx = state.rdx
	return
}
