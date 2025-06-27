package main

import (
	"fmt"
	"github.com/gritzko/rdx"
	"os"
	"strings"
)

var RDXCommands = rdx.Command{
	Subs: []rdx.Command{
		rdx.Command{
			"help",
			CmdHelp,
			"show help on commands",
			[]string{},
			nil,
		},
		rdx.Command{
			"RDX",
			nil,
			"RDX related commands",
			[]string{"rdx", "x"},
			[]rdx.Command{
				rdx.Command{
					"Linear",
					nil,
					"RDX Linear actions",
					[]string{"l", "L"},
					[]rdx.Command{
						rdx.Command{"ID",
							CmdLinearID,
							"l:id:A1ice-fr0m:B0b-ti11:32  calculate L IDs for an insertion",
							[]string{"id"},
							nil,
						},
					},
				},
			},
		},
		rdx.Command{
			"BRIX",
			nil,
			"BRIX store actions",
			[]string{"brix"},
			[]rdx.Command{
				rdx.Command{"New",
					CmdBrixNew,
					"brix:new:{@bob-123 Name:\"Robert A. Moore\"}  make a BRIX file for the objects listed",
					[]string{"n", "new"},
					nil,
				},
			},
		},
	},
}

func main() {
	concat := strings.Join(os.Args[1:], " ")

	cmds, err := rdx.ParseJDR([]byte(concat))
	var out []byte
	if err == nil {
		out, err = rdx.Execute(cmds, &RDXCommands)
	}
	if err != nil {
		fmt.Printf("bad command: %s\n", err.Error())
		os.Exit(-1)
	}
	jdr, err := rdx.WriteAllJDR(nil, out, 0)
	fmt.Println(string(jdr))
	return
}
