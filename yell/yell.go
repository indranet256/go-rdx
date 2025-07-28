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
		"for":       Operator(CmdFor),
		"map":       Operator(CmdFor),
		"echo":      Function(CmdEcho),
		"set":       Function(CmdSet),
		"join":      Command(CmdJoin),
		"load":      Command(CmdLoad),
		"eval":      Command(CmdEval),
		"list":      Function(CmdList),
		"seq":       Function(CmdSeq),
		"read":      Operator(CmdRead),
		//		"over":      Control2(CmdOver),
		"exit": Command(CmdExit),
		"rdx": &Context{
			names: map[string]any{
				"idint":     Command(CmdRdxIDInts),
				"fitid":     Command(CmdRdxFitID),
				"fit":       Command(CmdRdxFitID),
				"merge":     Function(CmdRdxMerge),
				"y":         Function(CmdRdxMerge),
				"normalize": Function(CmdRdxNormalize),
				"norm":      Function(CmdRdxNormalize),
				"normal":    Function(CmdRdxNormalize),
				"flat":      Function(CmdRdxFlatten),
				"flatten":   Function(CmdRdxFlatten),
				"diffhili":  Command(CmdRdxDiffHili),
				"diff":      Function(CmdRdxDiff),
			},
		},
		"crypto": &Context{
			names: map[string]any{
				"sha256": Function(CmdCryptoHash),
				"sha":    Function(CmdCryptoHash),
				"hash":   Function(CmdCryptoHash),
			},
		},
		"brix": &Context{
			names: map[string]any{
				"new":   Function(CmdBrixNew),
				"open":  Function(CmdBrixOpen),
				"info":  Command(CmdBrixInfo),
				"id":    Function(CmdBrixId),
				"find":  Command(CmdBrixFind),
				"close": Command(CmdBrixClose),

				"merge": Function(CmdBrixMerge),

				"prev": Command(CmdBrixBase),
				"base": Command(CmdBrixBase),
				"kind": Command(CmdBrixKind),

				"get": Function(CmdBrixGet),
				"add": Function(CmdBrixAdd),
				"has": Command(CmdBrixHas),
				"del": Command(CmdBrixDel),

				"list": Function(CmdBrixList),
				"seek": Command(CmdBrixSeek),

				"read": Command(CmdBrixRead),
				"over": Command(CmdBrixOver),
			},
		},
		"test": &Context{
			names: map[string]any{
				"eq":    Operator(CmdTestEq),
				"equal": Operator(CmdTestEq),
			},
		},
		"os": &Context{
			names: map[string]any{
				"ls":       Function(CmdOsLsDir),
				"lsdir":    Function(CmdOsLsDir),
				"chdir":    Function(CmdOsChDir),
				"mkdir":    Function(CmdOsMkDir),
				"mktmpdir": Function(CmdOsMkTmpDir),
				"mktmp":    Function(CmdOsMkTmpDir),
				"pwd":      Function(CmdOsPwd),
				"unlink":   Function(CmdOsUnlink),
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
	it := rdx.NewIter(cmds)
	if err == nil {
		out, err = TopContext.Eval(&it)
	}
	if err != nil && err != ErrNormalExit {
		fmt.Printf("%sbad command:%s %s\n", TermEsc(LIGHT_RED), TermEsc(0), err.Error())
		os.Exit(-1)
	}
	_ = out
	// todo repl mode
	//jdr, _ := rdx.WriteAllJDR(nil, out, 0)
	//fmt.Println(string(jdr))
	return
}
