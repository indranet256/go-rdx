package main

import (
	"os"

	"github.com/gritzko/rdx"
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

func CmdRmFile(ctx *REPL, args *rdx.Iter) (out []byte, err error) {
	return osScan(args, func(data []byte, path string) (out []byte, err error) {
		return nil, os.Remove(path)
	})
}

func CmdRmDir(ctx *REPL, args *rdx.Iter) (out []byte, err error) {
	var path string
	path, err = pickString(*args)
	if err != nil {
		err = os.RemoveAll(path)
	}
	return
}

func CmdGetDir(ctx *REPL, args *rdx.Iter) (out []byte, err error) {
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

func CmdMakeDir(ctx *REPL, arg *rdx.Iter) (out []byte, err error) {
	var fact string
	if !arg.Read() {
		fact, err = os.MkdirTemp("", "yell*")
	} else {
		fact, err = pickString(*arg)
		err = os.Mkdir(fact, 0777)
	}
	if err == nil {
		out = rdx.AppendString(out, []byte(fact))
	}
	return
}

func CmdChangeDir(ctx *REPL, arg *rdx.Iter) (out []byte, err error) {
	out, err = osScan(arg, func(data []byte, path string) (out []byte, err error) {
		return nil, os.Chdir(path)
	})
	var path string
	path, _ = os.Getwd()
	out = rdx.AppendString(out, []byte(path))
	return
}

func CmdListDir(ctx *REPL, arg *rdx.Iter) (out []byte, err error) {
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

func CmdLoadFile(ctx *REPL, args *rdx.Iter) (out []byte, err error) {
	path := ""
	if !args.Read() {
		return nil, ErrNoArgument
	}
	path, err = pickString(*args)
	if err != nil {
		return
	}
	return LoadJDR(path)
}

func CmdSaveFile(ctx *REPL, args *rdx.Iter) (out []byte, err error) {
	path := ""
	if !args.Read() {
		return nil, ErrNoArgument
	}
	path, err = pickString(*args)
	if err != nil {
		return
	}
	var jdr []byte
	jdr, err = rdx.WriteAllJDR(nil, args.Rest(), 0)
	if err == nil {
		err = os.WriteFile(path, jdr, 777)
	}
	return
}
