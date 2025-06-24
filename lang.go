package rdx

import "errors"

var ErrBadCommand = errors.New("bad command syntax")

type CmdFunc func(args, pre []byte) (out []byte, err error)

type Command struct {
	Name    string
	Func    CmdFunc
	Help    string
	Aliases []string
	Subs    []Command
}

// group:command(many arguments)
// command argument
// command(many arguments)
func ReadCommand(rdx []byte) (cmd, args, rest []byte, err error) {
	var lit byte
	orig := rdx
	lit, _, _, rdx, err = ReadRDX(rdx)
	if err == nil && lit != String && lit != Term && lit != Tuple {
		err = ErrBadCommand
	}
	cmd = orig[:len(orig)-len(rdx)]
	if err != nil {
		return
	}
	orig = rdx
	lit, _, args, rest, err = ReadRDX(rdx)
	if IsFIRST(lit) {
		args = orig[:len(orig)-len(rest)]
	}
	return
}

func Peek(rdx []byte) byte {
	if len(rdx) == 0 {
		return 0
	}
	return rdx[0] & ^CaseBit
}

func findCommand(cmd string, funs *Command) *Command {
	for _, c := range funs.Subs {
		if c.Name == cmd {
			return &c
		}
		for _, a := range c.Aliases {
			if a == cmd {
				return &c
			}
		}
	}
	return nil
}

func ExecuteCommand(cmd, args, pre []byte, root *Command) (out []byte, err error) {
	cc := root
	lit, _, val, cmd, err := ReadRDX(cmd)
	if len(cmd) > 0 || err != nil {
		return nil, ErrBadCommand
	}
	switch lit {
	case Term:
		cc = findCommand(string(val), cc)
	case String:
		cc = findCommand(string(val), cc)
	case Tuple:
		for len(val) > 0 && err == nil && cc != nil {
			lit, _, cmd, val, _ = ReadRDX(val)
			if lit != Term && lit != String {
				err = ErrBadCommand
			} else {
				cc = findCommand(string(cmd), cc)
			}
		}
	default:
	}
	if cc == nil {
		err = ErrBadCommand
	} else if cc.Func != nil {
		out, err = cc.Func(args, pre)
	} else {
		err = ErrBadCommand
	}
	return
}

func Execute(cmds []byte, root *Command) (out []byte, err error) {
	for len(cmds) > 0 && err == nil {
		var cmd, args []byte
		cmd, args, cmds, err = ReadCommand(cmds)
		if err == nil {
			out, err = ExecuteCommand(cmd, args, out, root)
		}
	}
	return
}
