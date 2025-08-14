package main

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gritzko/rdx"
)

var ErrBadTestEqArgs = errors.New("test-eq(comment, correct, eval)")

func report(comment, correct, expr, fact rdx.Stream) string {
	text := make([]byte, 0, 256)
	text = appendTermEsc(text, DARK_BLUE)
	text = append(text, comment...)
	text = append(text, '\t')
	if !bytes.Equal(fact, correct) {
		text = appendTermEsc(text, DARK_RED)
		text = append(text, "FAIL"...)
		text = append(text, '\n')
		text = appendTermEsc(text, LIGHT_GRAY)
		text = append(text, "want\t"...)
		text = appendTermEsc(text, DARK_GREEN)
		jdrc, _ := rdx.WriteAllJDR(nil, correct, 0)
		text = append(text, jdrc...)
		text = append(text, '\n')
		text = appendTermEsc(text, LIGHT_GRAY)
		text = append(text, "have\t"...)
		text = appendTermEsc(text, LIGHT_RED)
		jdrv, _ := rdx.WriteAllJDR(nil, fact, 0)
		text = append(text, jdrv...)
		text = append(text, '\n')
		text = appendTermEsc(text, LIGHT_GRAY)
		text = append(text, "eval\t"...)
		text = appendTermEsc(text, 0)
		jdrev, _ := rdx.WriteAllJDR(nil, expr, 0)
		text = append(text, jdrev...)
	} else {
		text = appendTermEsc(text, LIGHT_GREEN)
		text = append(text, "OK"...)
		text = appendTermEsc(text, 0)
	}
	return string(text)
}

func CmdTestEq(ctx *REPL, arg *rdx.Iter) (ret []byte, err error) {
	var comment, correct, expr, fact []byte
	if !arg.Read() {
		return nil, ErrBadTestEqArgs
	}
	if arg.Lit() == rdx.String {
		comment = arg.Value()
		if !arg.Read() {
			return
		}
	} else {
		comment = []byte("unnamed test")
	}
	correct, err = ctx.Eval(arg)
	fact, err = ctx.evaluate(arg.Rest())
	fmt.Println(report(comment, correct, expr, fact))
	return nil, nil
}

func CmdTestNil(ctx *REPL, eit *rdx.Iter) (ret []byte, err error) {
	var comment []byte
	if eit.Peek() == rdx.String {
		if !eit.Read() {
			return
		}
		comment = eit.Value()
	} else {
		comment = []byte("unnamed test")
	}
	fmt.Println(report(comment, nil, nil, eit.Rest()))
	return
}

func CmdTestAll(ctx *REPL, eit *rdx.Iter) (ret []byte, err error) {
	var eval, comment, correct, expr, fact []byte
	if eit.Peek() == rdx.String {
		if !eit.Read() {
			return
		}
		comment = eit.Value()
	} else {
		comment = []byte("unnamed test")
	}
	eval, err = ctx.evaluate(eit.Rest())
	res := rdx.NewIter(eval)
	if !res.Read() {
		return nil, ErrNoProcedureParams
	}
	correct = res.Record()
	for res.Read() {
		fact = res.Record()
		if !bytes.Equal(correct, fact) {
			fmt.Println(report(comment, correct, expr, fact))
			return nil, nil
		}
	}
	fmt.Println(report(comment, correct, expr, fact))
	return nil, nil
}
