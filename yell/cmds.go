package main

import (
	"bytes"
	"fmt"
	"github.com/gritzko/rdx"
	"os"
	"strconv"
)

func CmdEcho(ctx *Context, arg []byte) (out []byte, err error) {
	var jdr, eval []byte
	eval, err = ctx.Evaluate(nil, arg)
	if err != nil {
		return
	}
	jdr, err = rdx.WriteAllJDR(nil, eval, 0)
	if err != nil {
		return
	}
	fmt.Println(string(jdr))
	return nil, nil
}

func flatten(ctx *Context, arg, j []byte) (out []byte, err error) {
	for len(arg) > 0 && err == nil {
		var lit byte
		var val, rest []byte
		lit, _, val, rest, err = rdx.ReadTLKV(arg)
		switch lit {
		case rdx.String:
			out = append(out, val...)
		case rdx.Term:
			v, ok := ctx.names[string(val)]
			if ok {
				switch v.(type) {
				case []byte:
					var tmp []byte
					tmp, err = flatten(ctx, v.([]byte), j)
					out = append(out, tmp...)
				default:
					return nil, ErrUnexpectedNameType
				}
			} else {
				out = append(out, val...)
			}
		case rdx.Float:
			f := rdx.UnzipFloat64(val)
			out = strconv.AppendFloat(out, f, 'e', -1, 64)
		case rdx.Integer:
			i := rdx.UnzipInt64(val)
			out = strconv.AppendInt(out, i, 10)
		default:

		}
		if len(rest) > 0 {
			out = append(out, j...)
		}
		arg = rest
	}
	return
}

func CmdJoin(ctx *Context, arg []byte) (ret []byte, err error) {
	if len(arg) == 0 {
		return
	}
	j := []byte{}
	if rdx.Peek(arg) == rdx.String {
		_, _, j, arg, err = rdx.ReadTLKV(arg)
	}
	var out []byte
	out, err = flatten(ctx, arg, j)
	if err == nil {
		ret = rdx.WriteRDX(nil, rdx.String, rdx.ID{}, out)
	}
	return
}

func CmdLoad(ctx *Context, args []byte) (out []byte, err error) {
	path := ""
	if len(args) > 0 && rdx.Peek(args) == rdx.String {
		path, _, args, err = rdx.ReadString(args)
		return LoadJDR(path)
	} else {
		return nil, ErrBadArguments
	}
}

func CmdEval(ctx *Context, args []byte) (out []byte, err error) {
	path := ""
	if len(args) == 0 || rdx.Peek(args) != rdx.String {
		return nil, ErrBadArguments
	}
	path, _, args, err = rdx.ReadString(args)
	var code []byte
	if err == nil {
		code, err = LoadJDR(path)
	}
	if err == nil {
		out, err = ctx.Evaluate(nil, code)
	}
	return
}

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

func CmdExit(ctx *Context, args []byte) (out []byte, err error) {
	return nil, ErrNormalExit
}

func CmdSet(ctx *Context, args []byte) (out []byte, err error) {
	if len(args) == 0 || rdx.Peek(args) != rdx.Term {
		return nil, ErrBadArguments
	}
	var name []byte
	_, _, name, args, err = rdx.ReadRDX(args)
	namestr := string(name)
	nameval, ok := ctx.names[namestr]
	if !ok {
		ctx.names[namestr] = args
		return nil, err
	}
	switch nameval.(type) {
	case []byte:
		ctx.names[namestr] = args
		return nil, err
	default:
		return nil, ErrUnexpectedNameType
	}
}
