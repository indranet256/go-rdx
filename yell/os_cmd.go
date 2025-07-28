package main

import (
	"github.com/gritzko/rdx"
	"os"
)

func CmdOsUnlink(ctx *Context, args rdx.Iter) (out []byte, err error) {
	for args.Read() && err == nil {
		err = os.Remove(args.String())
	}
	return
}

func CmdOsMkTmpDir(ctx *Context, args rdx.Iter) (out []byte, err error) {
	var path string
	pattern := "test"
	if args.Read() && args.Lit() == rdx.String {
		pattern = string(args.Value())
	}
	path, err = os.MkdirTemp("", pattern)
	if err == nil {
		out = rdx.AppendString(out, []byte(path))
	}
	return
}

func CmdOsPwd(ctx *Context, args rdx.Iter) (out []byte, err error) {
	var path string
	path, err = os.Getwd()
	if err == nil {
		out = rdx.AppendString(out, []byte(path))
	}
	return
}

func CmdOsMkDir(ctx *Context, args rdx.Iter) (out []byte, err error) {
	for args.Read() && err == nil {
		err = os.Mkdir(string(args.String()), 0777)
	}
	return
}

func CmdOsChDir(ctx *Context, args rdx.Iter) (out []byte, err error) {
	for args.Read() && err == nil {
		err = os.Chdir(args.String())
	}
	return
}

func CmdOsLsDir(ctx *Context, args rdx.Iter) (out []byte, err error) {
	if len(args.Rest()) == 0 {
		args = rdx.NewIter([]byte{'s', 2, 0, '.'})
	}
	var marks rdx.Marks
	for err == nil && args.Read() {
		var de []os.DirEntry
		de, err = os.ReadDir(args.String())
		if err != nil {
			break
		}
		out = rdx.OpenTLV(out, rdx.Euler, &marks)
		out = append(out, 0)
		for _, e := range de {
			out = rdx.AppendString(out, []byte(e.Name()))
		}
		out, err = rdx.CloseTLV(out, rdx.Euler, &marks)
	}
	return
}
