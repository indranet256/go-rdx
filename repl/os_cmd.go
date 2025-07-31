package main

import (
	"github.com/gritzko/rdx"
	"os"
)

type OsCallBack func(data []byte, path string) (out []byte, err error)

func osScan(arg *rdx.Iter, cb OsCallBack) (out []byte, err error) {
	if !arg.Read() {
		return cb(out, "")
	}
	var args rdx.Iter
	switch arg.Lit() {
	case rdx.String:
		fallthrough
	case rdx.Term:
		args = rdx.NewIter(arg.Record())
	case rdx.Tuple:
		fallthrough
	case rdx.Euler:
		fallthrough
	case rdx.Linear:
		if len(arg.Value()) == 0 {
			return cb(out, "")
		} else {
			args = rdx.NewIter(arg.Value())
		}
	default:
		return nil, ErrBadArguments
	}
	for args.Read() && err == nil {
		out, err = cb(out, args.String())
	}
	return
}

func CmdOsUnlink(ctx *REPL, args *rdx.Iter) (out []byte, err error) {
	return osScan(args, func(data []byte, path string) (out []byte, err error) {
		return nil, os.Remove(path)
	})
}

func CmdOsMkTmpDir(ctx *REPL, args *rdx.Iter) (out []byte, err error) {
	return osScan(args, func(data []byte, path string) (out []byte, err error) {
		var fact string
		fact, err = os.MkdirTemp("", path)
		out = rdx.AppendString(data, []byte(fact))
		return
	})
}

func CmdOsPwd(ctx *REPL, args *rdx.Iter) (out []byte, err error) {
	if args.Peek() == rdx.Tuple {
		args.Read()
	}
	var path string
	path, err = os.Getwd()
	if err == nil {
		out = rdx.AppendString(out, []byte(path))
	}
	return
}

func CmdOsMkDir(ctx *REPL, arg *rdx.Iter) (out []byte, err error) {
	return osScan(arg, func(data []byte, path string) (out []byte, err error) {
		return nil, os.Mkdir(path, 0777)
	})
}

func CmdOsChDir(ctx *REPL, arg *rdx.Iter) (out []byte, err error) {
	return osScan(arg, func(data []byte, path string) (out []byte, err error) {
		return nil, os.Chdir(path)
	})
}

func CmdOsLsDir(ctx *REPL, arg *rdx.Iter) (out []byte, err error) {
	return osScan(arg, func(data []byte, path string) (out []byte, err error) {
		if path == "" {
			path = "."
		}
		marks := make(rdx.Marks, 2)
		var de []os.DirEntry
		de, err = os.ReadDir(path)
		if err != nil {
			return
		}
		out = rdx.OpenTLV(out, rdx.Euler, &marks)
		out = append(out, 0)
		for _, e := range de {
			out = rdx.AppendString(out, []byte(e.Name()))
		}
		out, err = rdx.CloseTLV(out, rdx.Euler, &marks)
		return
	})
}
