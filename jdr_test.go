package rdx

import (
	"testing"
)

func TestReadWriteJDR(t *testing.T) {
	cases := []string{
		"1",
		"1 2 3",
		"1.2e+03",
		"12-3",
		"\"one\\ttwo three\"",
		"one two three",
		"():()",
		"()",
		"(())",
		"(1)",
		"1:2:3",
		"1:2.3e+04:56-78",
		"<1@alice-1,2@bob-2>",
		"{\"one\",\"two\",\"three\"}",
	}
	for _, c := range cases {
		state := JDRstate{jdr: []byte(c)}
		err := JDRlexer(&state)
		if err != nil {
			t.Fatal(err)
		}
		j2, err := WriteAllJDR(nil, state.rdx, 0)
		jdr2 := string(j2)
		if jdr2 != c {
			t.Error("'" + jdr2 + "' != '" + c + "'")
		}
	}
}

func TestTuples(t *testing.T) {
	cases := []string{
		"(1 2 3)(4 5 6)()",
		"1:2:3 4:5:6 ()",
		"1 2 3; 4:5:6; ;",
	}
	correct := "1:2:3 4:5:6 ()"
	for _, c := range cases {
		state := JDRstate{jdr: []byte(c)}
		err := JDRlexer(&state)
		if err != nil {
			t.Fatal(err)
		}
		j2, err := WriteAllJDR(nil, state.rdx, 0)
		jdr2 := string(j2)
		if jdr2 != correct {
			t.Error("'" + jdr2 + "' != '" + correct + "' for '" + c + "'")
		}
	}
}
