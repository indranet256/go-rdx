package rdx

import "strconv"

type JDRstate struct {
	jdr []byte
	rdx []byte

	stack Marks
	pre   byte
	val   []byte
	line  int
	col   int
}

var emptyTuple []byte = []byte{'p', 1, 0}

const inlineTuple = 'p'

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
	if state.stack.Top() == inlineTuple && state.pre != ':' {
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
	lit := state.stack.Top()
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

func insertInlineTuple(state *JDRstate) (err error) {
	// TODO HORROR
	return nil
}

func insertEmptyTuple(state *JDRstate) (err error) {
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

func JDRonOpenPLEX(tok []byte, state *JDRstate, plex byte) (err error) {
	if (&(state.stack)).Top() == inlineTuple && state.pre != ':' {
		err = closeInlineTuple(state)
	}
	if err == nil {
		state.rdx = OpenTLV(state.rdx, plex, &state.stack)
	}
	return
}

func JDRonClosePLEX(tok []byte, state *JDRstate, plex byte) (err error) {
	lit := state.pre
	if lit != state.stack.Top() {
		return ErrBadNesting
	}
	if !IsPLEX(lit) {
		return ErrBadNesting
	}
	if lit == Euler {
		// TODO sort
	} else if lit == Multix {
		// TODO sort
	}
	state.rdx, err = CloseTLV(state.rdx, lit, &state.stack)
	return
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
	if state.stack.Top() == inlineTuple {
		err = closeInlineTuple(state)
	}
	if state.pre == 0 || state.pre == ',' {
		insertEmptyTuple(state)
	}
	return
}
func JDRonColon(tok []byte, state *JDRstate) (err error) {
	if state.stack.Top() != inlineTuple {
		err = insertInlineTuple(state)
	} else if state.pre == 0 || state.pre == ':' {
		err = insertEmptyTuple(state)
	}
	state.pre = ':'
	return err
}

func JDRonOpen(tok []byte, state *JDRstate) error { return nil }

func JDRonClose(tok []byte, state *JDRstate) error { return nil }

func JDRonInter(tok []byte, state *JDRstate) error { return nil }

func JDRonRoot(tok []byte, state *JDRstate) (err error) {
	if state.stack.Top() == inlineTuple {
		err = closeInlineTuple(state)
	}
	if state.stack.Len() > 0 {
		err = ErrBadNesting
	}
	return
}

// .. .. .. ..

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
		case '/':
			jdr = append(jdr, '\\', '/')
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

func appendJDRStamp(jdr []byte, id ID) []byte {
	if id.IsZero() {
		return jdr
	}
	jdr = append(jdr, '@')
	jdr = append(jdr, id.String()...)
	jdr = append(jdr, ' ')
	return jdr
}

func WriteJDR(jdr, rdx []byte, style uint64) (jdr2, rest []byte, err error) {
	var lit byte
	var key ID
	var val []byte
	lit, key, val, rest, err = ReadRDX(rdx)
	if err != nil {
		return
	}
	switch lit {
	case Float:
		f := UnzipFloat64(val)
		jdr = strconv.AppendFloat(jdr, f, 'e', -1, 64)
		jdr = appendJDRStamp(jdr, key)
	case Integer:
		i := UnzipInt64(val)
		jdr = strconv.AppendInt(jdr, i, 10)
		jdr = appendJDRStamp(jdr, key)
	case Reference:
		id := UnzipID(val)
		jdr = append(jdr, id.String()...)
		jdr = appendJDRStamp(jdr, key)
	case String:
		jdr = append(jdr, '"')
		jdr = appendEscaped(jdr, val)
		jdr = append(jdr, '"')
	case Term:
		jdr = append(jdr, val...)
	case Tuple:
	case Linear:
	case Euler:
	case Multix:
	default:
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
