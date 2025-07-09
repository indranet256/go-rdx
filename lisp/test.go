package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/gritzko/rdx"
)

var ErrBadTestEqArgs = errors.New("test.eq(comment, eval, correct)")

func CmdTestEq(ctx *Context, arg []byte) (ret []byte, err error) {
	var comment, correct, val, ev []byte
	if rdx.Peek(arg) == rdx.String {
		_, _, comment, arg, err = rdx.ReadTLKV(arg)
	} else {
		comment = []byte("unnamed test")
	}
	_, _, _, ev, err = rdx.ReadTLKV(arg)
	correct = arg[:len(arg)-len(ev)]
	val, err = ctx.Evaluate(nil, ev)
	text := make([]byte, 0, 256)
	text = AppendTermEsc(text, DARK_BLUE)
	text = append(text, comment...)
	text = append(text, '\t')
	if bytes.Compare(val, correct) != 0 {
		text = AppendTermEsc(text, DARK_RED)
		text = append(text, "FAIL"...)
		text = append(text, '\n')
		text = AppendTermEsc(text, DARK_GREEN)
		jdrc, _ := rdx.WriteAllJDR(nil, ev, 0)
		text = append(text, jdrc...)
		text = append(text, '\n')
		text = AppendTermEsc(text, LIGHT_RED)
		jdrv, _ := rdx.WriteAllJDR(nil, val, 0)
		text = append(text, jdrv...)
	} else {
		text = AppendTermEsc(text, LIGHT_GREEN)
		text = append(text, "OK"...)
	}
	text = AppendTermEsc(text, 0)
	fmt.Println(string(text))
	return nil, nil
}

const (
	BOLD            = 1
	WEAK            = 2
	HIGHLIGHT       = 3
	UNDERLINE       = 4
	BLACK           = 30
	DARK_RED        = 31
	DARK_GREEN      = 32
	DARK_YELLOW     = 33
	DARK_BLUE       = 34
	DARK_PINK       = 35
	DARK_CYAN       = 36
	BLACK_BG        = 40
	DARK_RED_BG     = 41
	DARK_GREEN_BG   = 42
	DARK_YELLOW_BG  = 43
	DARK_BLUE_BG    = 44
	DARK_PINK_BG    = 45
	DARK_CYAN_BG    = 46
	GRAY            = 90
	LIGHT_RED       = 91
	LIGHT_GREEN     = 92
	LIGHT_YELLOW    = 93
	LIGHT_BLUE      = 94
	LIGHT_PINK      = 95
	LIGHT_CYAN      = 96
	LIGHT_GRAY      = 97
	GRAY_BG         = 100
	LIGHT_RED_BG    = 101
	LIGHT_GREEN_BG  = 102
	LIGHT_YELLOW_BG = 103
	LIGHT_BLUE_BG   = 104
	LIGHT_PINK_BG   = 105
	LIGHT_CYAN_BG   = 106
	LIGHT_GRAY_BG   = 107
)

func AppendTermEsc(data []byte, code int) []byte {
	return append(data, []byte(fmt.Sprintf("\x1b[%dm", code))...)
}

func TermEsc(code int) []byte {
	ret := make([]byte, 0, 16)
	return AppendTermEsc(ret, code)
}
