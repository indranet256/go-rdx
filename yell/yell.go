package main

import (
	"errors"
	"fmt"
	"github.com/gritzko/rdx"
	"os"
	"strings"
)

var ErrBadArguments = errors.New("bad arguments")

var TopContext = Context{
	names: map[string]any{
		"__version": rdx.WriteRDX(nil, rdx.String, rdx.ID{}, []byte("RDXLisp v0.0.1")),
		"if":        Control(CmdIf),
		"eq":        Command(CmdEq),
		"echo":      Command(CmdEcho),
		"join":      Command(CmdJoin),
		"load":      Command(CmdLoad),
		"eval":      Command(CmdEval),
		"rdx": &Context{
			names: map[string]any{
				"idint":     Command(CmdIDInts),
				"fitid":     Command(CmdFitID),
				"merge":     Command(CmdMerge),
				"normalize": Command(CmdNormalize),
				"flat":      Command(CmdRdxFlatten),
			},
		},
		"crypto": &Context{
			names: map[string]any{
				"sha256": Command(CmdHash),
			},
		},
		"brix": &Context{
			names: map[string]any{
				"new": Command(CmdBrixNew),
				"get": Command(CmdBrixGet),
				"add": Command(CmdBrixAdd),
			},
		},
		"test": &Context{
			names: map[string]any{
				"eq": Command(CmdTestEq),
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

	var out []byte
	if err == nil {
		out, err = TopContext.Evaluate(nil, cmds)
	}
	if err != nil {
		fmt.Printf("bad command: %s\n", err.Error())
		os.Exit(-1)
	}
	_ = out
	// todo repl mode
	//jdr, err := rdx.WriteAllJDR(nil, out, 0)
	//fmt.Print(string(jdr))
	return
}
