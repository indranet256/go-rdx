package main

import (
	"errors"
	"fmt"
	"github.com/gritzko/rdx"
	"os"
	"strings"
)

var ErrBadArguments = errors.New("bad arguments")
var ErrNormalExit = errors.New("all OK")

var TopContext = Context{
	names: map[string]any{
		"__version": rdx.WriteRDX(nil, rdx.String, rdx.ID{}, []byte("RDXLisp v0.0.1")),
		"if":        Control(CmdIf),
		"eq":        Command(CmdEq),
		"for":       Control(CmdFor),
		"echo":      Command(CmdEcho),
		"set":       Command(CmdSet),
		"join":      Command(CmdJoin),
		"load":      Command(CmdLoad),
		"eval":      Command(CmdEval),
		"exit":      Command(CmdExit),
		"rdx": &Context{
			names: map[string]any{
				"idint":     Command(CmdRdxIDInts),
				"fitid":     Command(CmdRdxFitID),
				"fit":       Command(CmdRdxFitID),
				"merge":     Command(CmdRdxMerge),
				"y":         Command(CmdRdxMerge),
				"normalize": Command(CmdRdxNormalize),
				"norm":      Command(CmdRdxNormalize),
				"normal":    Command(CmdRdxNormalize),
				"flat":      Command(CmdRdxFlatten),
				"flatten":   Command(CmdRdxFlatten),
				"diffhili":  Command(CmdRdxDiffHili),
				"diff":      Command(CmdRdxDiff),
			},
		},
		"crypto": &Context{
			names: map[string]any{
				"sha256": Command(CmdCryptoHash),
				"sha":    Command(CmdCryptoHash),
				"hash":   Command(CmdCryptoHash),
			},
		},
		"brix": &Context{
			names: map[string]any{
				"new":   Command(CmdBrixNew),
				"open":  Command(CmdBrixOpen),
				"info":  Command(CmdBrixInfo),
				"find":  Command(CmdBrixFind),
				"close": Command(CmdBrixClose),

				"pack": Command(CmdBrixPack),

				"prev": Command(CmdBrixBase),
				"base": Command(CmdBrixBase),
				"kind": Command(CmdBrixKind),

				"get": Command(CmdBrixGet),
				"add": Command(CmdBrixAdd),
				"has": Command(CmdBrixHas),
				"del": Command(CmdBrixDel),

				"scan": Command(CmdBrixSeek),

				"seek": Command(CmdBrixSeek),
				"next": Command(CmdBrixNext),
				"over": Command(CmdBrixOver),
			},
		},
		"test": &Context{
			names: map[string]any{
				"eq":    Command(CmdTestEq),
				"equal": Command(CmdTestEq),
			},
		},
	},
}

func main() {
	var code, cmds []byte
	var err error
	if len(os.Args) == 2 && strings.HasSuffix(os.Args[1], ".jdr") {
		cmds, err = LoadJDR(os.Args[1])
	} else {
		code = []byte(strings.Join(os.Args[1:], " "))
		cmds, err = rdx.ParseJDR(code)
	}
	InitTerm()

	var out []byte
	if err == nil {
		out, err = TopContext.Evaluate(nil, cmds)
	}
	if err != nil && err != ErrNormalExit {
		fmt.Printf("bad command: %s\n", err.Error())
		os.Exit(-1)
	}
	_ = out
	// todo repl mode
	return
}
