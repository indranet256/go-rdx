package rdx

import (
	"fmt"
	"os"
)

type Tester func(rdx []byte) error

func ProcessTestFile(path string, tester Tester) (err error) {
	jdr, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	rdx, err := ParseJDR(jdr)
	if err != nil {
		return err
	}
	rest := rdx
	state := 0
	start := len(rest)
	finish := len(rest)
	var cases [][]byte
	var legend []byte
	var leg []byte
	var legends []string
	for len(rest) > 0 && err == nil {
		var lit byte
		var val []byte
		pre := len(rest)
		lit, _, val, rest, err = ReadRDX(rest)
		switch state {
		case 0:
			if lit == String && len(val) == 0 {
				state++
				finish = pre
			}
		case 1:
			if lit == String {
				state++
				leg = val
			} else {
				state = 0
			}
		case 2:
			if lit == String && len(val) == 0 {
				if finish < start {
					cases = append(cases, rdx[len(rdx)-start:len(rdx)-finish])
					legends = append(legends, string(legend))
				}
				start = len(rest)
				finish = start
				legend = leg
			}
			state = 0
		}
	}
	finish = 0
	if finish < start {
		cases = append(cases, rdx[len(rdx)-start:len(rdx)-finish])
		legends = append(legends, string(legend))
	}

	for n, c := range cases {
		err = tester(c)
		fmt.Print(legends[n])
		if err != nil {
			fmt.Printf("Test case %d fails: %s\n", n, c)
		}
	}

	return
}
