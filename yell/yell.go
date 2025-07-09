package main

import (
	"bytes"
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
		"rdx": &Context{
			names: map[string]any{
				"idint":     Command(CmdIDInts),
				"fitid":     Command(CmdFitID),
				"merge":     Command(CmdMerge),
				"normalize": Command(CmdNormalize),
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
		var file *os.File
		file, err = os.Open(os.Args[1])
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "IO error: %s\n", err.Error())
			return
		}
		stat, _ := file.Stat()
		todo := stat.Size()
		code = make([]byte, todo)
		rest := code
		for len(rest) > 0 && err == nil {
			var n int
			n, err = file.Read(rest)
			rest = rest[n:]
		}
		if len(code) > 0 && code[0] == '#' {
			i := bytes.IndexByte(code, '\n')
			if i > 0 {
				code = code[i:]
			}
		}
	} else {
		code = []byte(strings.Join(os.Args[1:], " "))
	}

	if err == nil {
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
