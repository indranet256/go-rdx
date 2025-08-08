package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/gritzko/rdx"
)

func LoadJDR(path string) (cmds []byte, err error) {
	var file *os.File
	file, err = os.Open(path)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "IO error: %s\n", err.Error())
		return
	}
	stat, _ := file.Stat()
	todo := stat.Size()
	code := make([]byte, todo)
	rest := code
	for len(rest) > 0 && err == nil {
		var n int
		n, err = file.Read(rest)
		rest = rest[n:]
	}
	if err == nil && len(code) > 0 && code[0] == '#' {
		i := bytes.IndexByte(code, '\n')
		if i > 0 {
			code = code[i:]
		}
	}
	if err == nil {
		cmds, err = rdx.ParseJDR(code)
	}
	return
}

func EvalArgs(repl *REPL) (err error) {
	var code, cmds, out []byte
	if len(os.Args) == 2 && strings.HasSuffix(os.Args[1], ".jdr") {
		cmds, err = LoadJDR(os.Args[1])
	} else {
		code = []byte(strings.Join(os.Args[1:], " "))
		cmds, err = rdx.ParseJDR(code)
	}
	if err != nil {
		return
	}

	repl.InitTerm()

	out, err = repl.Evaluate(cmds)
	jdr, _ := rdx.WriteAllJDR(nil, out, 0)
	if len(jdr) > 0 {
		fmt.Println(string(jdr))
	}

	return
}

func main() {
	var err error
	repl := NewREPL(Yell, nil)

	if len(os.Args) > 1 {
		err = EvalArgs(repl)
	} else {
		for err == nil {
			err = repl.Loop(os.Stdin, os.Stdout)
		}
	}
	if err == Errturn {
		err = nil
	}

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%sbad command:%s %s\n", TermEsc(LIGHT_RED), TermEsc(0), err.Error())
		os.Exit(-1)
	}
}
