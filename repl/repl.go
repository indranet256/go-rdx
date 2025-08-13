package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/gritzko/rdx"
)

var Errturn = errors.New("everything's gonna be allright")
var ErrNoSpaceOpen = errors.New("no space open")
var ErrNoBranchOpen = errors.New("no branch open")

type Command func(repl *REPL, args *rdx.Iter) (out []byte, err error)

var Yell = map[rdx.ID]Command{
	rdx.ID{0, 199282}:          CmdLen,      // len [1 2 3] -> 3
	rdx.ID{0, 239990}:          CmdVar,      // var i 0
	rdx.ID{0, 58684841394}:     CmdReturn,   // return true
	rdx.ID{0x0, 0xa7cb78}:      CmdExit,     // exit
	rdx.ID{0, 60009667755}:     CmdString,   // string
	rdx.ID{0, 12770808}:        CmdList,     // list({val}) list(val1 val2 val3)
	rdx.ID{0, 178808}:          CmdGet,      // get(id)
	rdx.ID{0, 216696}:          CmdPut,      // put(val)
	rdx.ID{0, 227960}:          CmdSet,      // set({@id}) set(id val)
	rdx.ID{0, 154152}:          CmdAdd,      // add({@id}) add(id {val})
	rdx.ID{0, 13585010}:        CmdOpen,     // open(space branch)
	rdx.ID{0, 13818351}:        CmdPick,     // pick(a {a:1 b:2}) -> a:1
	rdx.ID{0, 13855975}:        CmdProc,     // proc Fn(p1 p2 p3) [ ...code...]
	rdx.ID{0, 2922}:            CmdIf,       // if eq(a b) [...] else [...]
	rdx.ID{0, 63}:              CmdVerbatim, // ~{a:b}
	rdx.ID{0, 257962825714545}: CmdVerbatim, // verbatim {a:b}

	rdx.ID{0, 14326120}:  CmdRead,  // read(rdr)
	rdx.ID{0, 175350}:    CmdFor,   // for(rdr)[code]
	rdx.ID{0, 203124}:    CmdFor,   // map(rdr)[code]
	rdx.ID{0, 227957}:    CmdSeq,   // seq(var from to step)
	rdx.ID{0, 886758584}: CmdPrint, // print(...)
	rdx.ID{0, 667106793}: CmdClose, // close(closer) close()

	rdx.ID{228977, 187576}: CmdSumInt, // sum-int(1 2 3 4)

	rdx.ID{11209080, 223804}:    CmdFlatRDX,   // flat-rdx {@a-2 b@1 c d@e-3}
	rdx.ID{54557088112, 223804}: CmdNormalRDX, // normal-rdx { a a a}

	rdx.ID{14851576, 2677}:   CmdTestEq,  // test-eq("comment" correct eval)
	rdx.ID{14851576, 154672}: CmdTestAll, // test-all("comment" correct eval)

	rdx.ID{12770808, 10185583}:  CmdListBrik, // list-brik
	rdx.ID{667106793, 10185583}: CmdClose,    // close-brik
	rdx.ID{12770808, 10185596}:  CmdListBrix, // list-brix
	rdx.ID{667106793, 10185596}: CmdClose,    // close-brix

	rdx.ID{12999657, 41718065644}: CmdMakeBranch, // make-branch(handle "mission")

	rdx.ID{12999657, 2860483104841}:    CmdCryptoKeyGen, // make-ed25519
	rdx.ID{14605042, 2860483104841}:    CmdCryptoSign,   // sign-ed25519
	rdx.ID{62979234493, 2860483104841}: CmdCryptoVerify, // verify-ed25519

	rdx.ID{12999657, 936532457}: CmdMakeSpace, // make-space(handle "description")
	rdx.ID{13585010, 936532457}: CmdOpenSpace, // open-space(handle)
	rdx.ID{0, 936532457}:        CmdOpenSpace, // space handle
	rdx.ID{14601467, 936532457}: CmdShowSpace, // show-space

	rdx.ID{0xb68, 0x2dcb8}: CmdIdInt, // id-int
	rdx.ID{0xb68, 208123}:  CmdIdNow, // id-now

	rdx.ID{228977, 59803705670}: CmdSumSha256, // sum-sha256

	rdx.ID{12794216, 11197481}:  CmdLoadFile,  // load-file "path.jdr"
	rdx.ID{14573225, 11197481}:  CmdSaveFile,  // save-file ("path.jdr" {something})
	rdx.ID{3319, 62054915247}:   CmdOsUnlink,  // os-unlink
	rdx.ID{3505, 166774}:        CmdRmDir,     // rm-dir
	rdx.ID{178808, 166774}:      CmdGetDir,    // get-dir
	rdx.ID{12999657, 166774}:    CmdMakeDir,   // make-dir
	rdx.ID{12770808, 166774}:    CmdListDir,   // list-dir
	rdx.ID{42624035561, 166774}: CmdChangeDir, // change-dir

}

type REPL struct {
	space  rdx.Branch
	branch rdx.Branch
	cmds   map[rdx.ID]Command
	vals   map[rdx.ID]any
	pros   map[rdx.ID]Proc
	vinc   uint64
}

type Proc struct {
	params, body rdx.RDX
}

func NewREPL(cmds map[rdx.ID]Command, vals map[rdx.ID]any) *REPL {
	if vals == nil {
		vals = make(map[rdx.ID]any)
	}
	return &REPL{cmds: cmds, vals: vals, pros: make(map[rdx.ID]Proc)}
}

func (repl *REPL) Close() (err error) {
	err = repl.space.Close()
	if repl.branch.IsOpen() {
		_ = repl.branch.Close()
	}
	repl.vals = nil
	return
}

func (repl *REPL) EvalCommand(code *rdx.Iter, cmd Command) (out []byte, err error) {
	var eval []byte
	c := *code
	var params []byte
	if !code.Read() {
	} else if code.Lit() == rdx.Tuple {
		params = code.Value()
	} else {
		params = code.Record()
	}
	if params != nil {
		eval, err = repl.evaluate(params)
	}
	if err == nil {
		it := rdx.NewIter(eval)
		out, err = cmd(repl, &it)
	}
	if err != nil {
		jdr, _ := rdx.WriteAllJDR(nil, params, 0)
		err = errors.New("error in " + c.String() + "(" + string(jdr) + "): " + err.Error())
	}
	return
}

func (repl *REPL) Eval(code *rdx.Iter) (out rdx.RDX, err error) {
	switch code.Lit() {
	case rdx.Reference:
		ref := code.Reference()
		cmd, okcmd := repl.cmds[ref]
		if okcmd {
			return repl.EvalCommand(code, cmd)
		}
		local, oklocal := repl.vals[ref]
		if oklocal {
			switch local.(type) {
			case []byte:
				return local.([]byte), nil
			case rdx.Reader:
				return local.(rdx.Reader).Record(), nil
			default:
				return code.Record(), nil
			}
		}
		stored, _ := repl.branch.Get(ref)
		if stored != nil {
			return stored, nil
		}
		fallthrough
	case rdx.Float:
		fallthrough
	case rdx.Integer:
		fallthrough
	case rdx.String:
		out = append(out, code.Record()...)
		return
	case rdx.Term:
		if len(code.Value()) > 10 {
			out = append(out, code.Record()...)
		}
		seq, _ := rdx.ParseRON64(code.Value())
		ref := rdx.ID{0, seq}
		cmd, okcmd := repl.cmds[ref]
		if okcmd { // controls evaluate stuff themselves
			return cmd(repl, code)
		}
		local, oklocal := repl.vals[ref]
		if oklocal {
			switch local.(type) {
			case rdx.RDX:
				return local.(rdx.RDX), nil
			case []byte:
				return local.([]byte), nil
			//case rdx.Reader:  for rdr [ print ]
			//	return local.(rdx.Reader).Record(), nil
			default:
				return code.Record(), nil
			}
		}
		proc, okproc := repl.pros[ref]
		if okproc {
			return repl.Call(proc, code)
		}
		ref.Src = repl.branch.Clock.Src
		stored, _ := repl.branch.Get(ref)
		if stored != nil {
			return stored, nil
		}
		out = append(out, code.Record()...)
		return
	case rdx.Tuple:
		fallthrough
	case rdx.Euler:
		fallthrough
	case rdx.Multix:
		fallthrough
	case rdx.Linear:
		var ev []byte
		ev, err = repl.evaluate(code.Value())
		if err == nil && ev != nil {
			out = rdx.WriteRDX(out, code.Lit(), code.ID(), ev)
		}
	}
	return
}

type oldVar struct {
	nm  rdx.ID
	val any
}

func (repl *REPL) Call(proc Proc, args *rdx.Iter) (out []byte, err error) {
	var eval rdx.Iter
	eval, err = repl.evalArgs(args)
	if err != nil {
		return
	}
	var olds []oldVar
	parit := rdx.NewIter(proc.params)
	for parit.Read() {
		var pn rdx.ID
		pn, err = pickID(parit)
		if err != nil {
			return
		}
		if !eval.Read() {
			return nil, errors.New("argument is missing: " + string(pn.String()))
		}
		olds = append(olds, oldVar{pn, repl.vals[pn]})
		repl.vals[pn] = eval.Record()
	}
	out, err = repl.evaluate(proc.body)
	for _, old := range olds {
		if old.val == nil {
			delete(repl.vals, old.nm)
		} else {
			repl.vals[old.nm] = old.val
		}
	}
	return
}

func (repl *REPL) evaluate(code []byte) (out []byte, err error) {
	it := rdx.NewIter(code)
	for err == nil && it.Read() {
		var one []byte
		one, err = repl.Eval(&it)
		if err == Errturn {
			out = one
		} else {
			out = append(out, one...)
		}
	}
	return
}

func (repl *REPL) Evaluate(code []byte) (out []byte, err error) {
	var norm rdx.RDX
	norm, err = rdx.Normalize(code)
	if err != nil {
		return
	}
	return repl.evaluate(norm)
}

var ReplGreeting = "$ "

func (repl *REPL) Loop(reader io.Reader, writer io.Writer) (err error) {
	var command, code, out, jdr []byte
	n, m := 0, 0
	_, err = writer.Write([]byte(ReplGreeting))
	for err == nil {
		if n == len(command) {
			nc := make([]byte, len(command)*2+(1<<12))
			copy(nc, command)
			command = nc
		}
		m, err = reader.Read(command[n:])
		if err != nil {
			break
		}
		n += m
		code, err = rdx.ParseJDR(command[:n])
		if err == rdx.ErrIncomplete {
			err = nil
			continue
		} else if err != nil {
			break
		} else {
			n = 0
		}
		out, err = repl.Evaluate(code)
		if err == nil {
			jdr, err = rdx.WriteAllJDR(nil, out, 0)
			jdr = append(jdr, '\n')
			jdr = append(jdr, ReplGreeting...)
			for len(jdr) > 0 && err == nil {
				l := 0
				l, err = writer.Write(jdr)
				jdr = jdr[l:]
			}
		}
	}
	return
}

func appendTermEsc(data []byte, code int) []byte {
	return append(data, []byte(fmt.Sprintf("\x1b[%dm", code))...)
}

func TermEsc(code int) []byte {
	ret := make([]byte, 0, 16)
	return appendTermEsc(ret, code)
}

func (repl *REPL) InitTerm() {
	repl.vals[rdx.ParseIDString("RESET")] = rdx.AppendString(nil, TermEsc(0))
	repl.vals[rdx.ParseIDString("BOLD")] = rdx.AppendString(nil, TermEsc(1))
	repl.vals[rdx.ParseIDString("WEAK")] = rdx.AppendString(nil, TermEsc(2))
	repl.vals[rdx.ParseIDString("HIGHLIGHT")] = rdx.AppendString(nil, TermEsc(3))
	repl.vals[rdx.ParseIDString("UNDERLINE")] = rdx.AppendString(nil, TermEsc(4))
	repl.vals[rdx.ParseIDString("BLACK")] = rdx.AppendString(nil, TermEsc(30))
	repl.vals[rdx.ParseIDString("DARK_RED")] = rdx.AppendString(nil, TermEsc(31))
	repl.vals[rdx.ParseIDString("DARK_GREEN")] = rdx.AppendString(nil, TermEsc(32))
	repl.vals[rdx.ParseIDString("DARK_YELLOW")] = rdx.AppendString(nil, TermEsc(33))
	repl.vals[rdx.ParseIDString("DARK_BLUE")] = rdx.AppendString(nil, TermEsc(34))
	repl.vals[rdx.ParseIDString("DARK_PINK")] = rdx.AppendString(nil, TermEsc(35))
	repl.vals[rdx.ParseIDString("DARK_CYAN")] = rdx.AppendString(nil, TermEsc(36))
	repl.vals[rdx.ParseIDString("BLACK_BG")] = rdx.AppendString(nil, TermEsc(40))
	repl.vals[rdx.ParseIDString("DARK_RED_BG")] = rdx.AppendString(nil, TermEsc(41))
	repl.vals[rdx.ParseIDString("DARK_GREEN_BG")] = rdx.AppendString(nil, TermEsc(42))
	repl.vals[rdx.ParseIDString("DARK_YELLOW_BG")] = rdx.AppendString(nil, TermEsc(43))
	repl.vals[rdx.ParseIDString("DARK_BLUE_BG")] = rdx.AppendString(nil, TermEsc(44))
	repl.vals[rdx.ParseIDString("DARK_PINK_BG")] = rdx.AppendString(nil, TermEsc(45))
	repl.vals[rdx.ParseIDString("DARK_CYAN_BG")] = rdx.AppendString(nil, TermEsc(46))
	repl.vals[rdx.ParseIDString("GRAY")] = rdx.AppendString(nil, TermEsc(90))
	repl.vals[rdx.ParseIDString("LIGHT_RED")] = rdx.AppendString(nil, TermEsc(91))
	repl.vals[rdx.ParseIDString("LIGHT_GREEN")] = rdx.AppendString(nil, TermEsc(92))
	repl.vals[rdx.ParseIDString("LIGHT_YELLOW")] = rdx.AppendString(nil, TermEsc(93))
	repl.vals[rdx.ParseIDString("LIGHT_BLUE")] = rdx.AppendString(nil, TermEsc(94))
	repl.vals[rdx.ParseIDString("LIGHT_PINK")] = rdx.AppendString(nil, TermEsc(95))
	repl.vals[rdx.ParseIDString("LIGHT_CYAN")] = rdx.AppendString(nil, TermEsc(96))
	repl.vals[rdx.ParseIDString("LIGHT_GRAY")] = rdx.AppendString(nil, TermEsc(97))
	repl.vals[rdx.ParseIDString("GRAY_BG")] = rdx.AppendString(nil, TermEsc(100))
	repl.vals[rdx.ParseIDString("LIGHT_RED_BG")] = rdx.AppendString(nil, TermEsc(101))
	repl.vals[rdx.ParseIDString("LIGHT_GREEN_BG")] = rdx.AppendString(nil, TermEsc(102))
	repl.vals[rdx.ParseIDString("LIGHT_YELLOW_BG")] = rdx.AppendString(nil, TermEsc(103))
	repl.vals[rdx.ParseIDString("LIGHT_BLUE_BG")] = rdx.AppendString(nil, TermEsc(104))
	repl.vals[rdx.ParseIDString("LIGHT_PINK_BG")] = rdx.AppendString(nil, TermEsc(105))
	repl.vals[rdx.ParseIDString("LIGHT_CYAN_BG")] = rdx.AppendString(nil, TermEsc(106))
	repl.vals[rdx.ParseIDString("LIGHT_GRAY_BG")] = rdx.AppendString(nil, TermEsc(107))

	repl.vals[rdx.ParseIDString("__version")] = rdx.S0("RDXLisp v0.0.1")

	return
}

const (
	BOLD            = 1
	WEAK            = 2
	HIGHLIGHT       = 3
	UNDERLINE       = 4
	BLACK           = 30
	DARK_RED        = 31
	DARK_GREEN      = 32
	DARK_YELLOW     = 33
	DARK_BLUE       = 34
	DARK_PINK       = 35
	DARK_CYAN       = 36
	BLACK_BG        = 40
	DARK_RED_BG     = 41
	DARK_GREEN_BG   = 42
	DARK_YELLOW_BG  = 43
	DARK_BLUE_BG    = 44
	DARK_PINK_BG    = 45
	DARK_CYAN_BG    = 46
	GRAY            = 90
	LIGHT_RED       = 91
	LIGHT_GREEN     = 92
	LIGHT_YELLOW    = 93
	LIGHT_BLUE      = 94
	LIGHT_PINK      = 95
	LIGHT_CYAN      = 96
	LIGHT_GRAY      = 97
	GRAY_BG         = 100
	LIGHT_RED_BG    = 101
	LIGHT_GREEN_BG  = 102
	LIGHT_YELLOW_BG = 103
	LIGHT_BLUE_BG   = 104
	LIGHT_PINK_BG   = 105
	LIGHT_CYAN_BG   = 106
	LIGHT_GRAY_BG   = 107
)
