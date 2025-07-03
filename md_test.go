package rdx

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func testValueOrder(rdx []byte) error {
	i := Iter{Rest: rdx}
	n := 0
	for len(i.Rest) > 0 {
		prev := i
		n++
		err := i.Next()
		if err != nil {
			return err
		}
		z := CompareEuler(&prev, &i)
		if z > 0 {
			pj, _, _ := WriteJDR(nil, prev.Last, 0)
			tj, _, _ := WriteJDR(nil, i.Last, 0)
			fmt.Printf(Red+"Bad order %d and %d"+Reset+"\n\t%s\n\t%s", n, n+1, pj, tj)
			return errors.New("bad order")
		}
	}
	return nil
}

func TestValueOrder(t *testing.T) {
	err := ProcessTestFile("z.md", testValueOrder)
	if err != nil {
		t.Fatal(err)
	}
}

var Tilde = []byte{'~'}

func mergeTest(correct []byte, inputs [][]byte) (err error) {
	var merged []byte
	merged, err = Merge(nil, inputs)
	if err != nil {
		return err
	}
	if !bytes.Equal(correct, merged) {
		pj, _, _ := WriteJDR(nil, correct, 0)
		tj, _, _ := WriteJDR(nil, merged, 0)
		fmt.Printf(Red+"Bad merge\n"+Green+"\t%s\n"+Red+"\t%s"+Reset, pj, tj)
		return errors.New("bad merge")
	}
	return nil
}

func testMerge(rdx []byte) (err error) {
	i := Iter{Rest: rdx}
	inputs := make([][]byte, 0, 32)
	for len(i.Rest) > 0 && err == nil {
		err = i.Next()
		if err != nil {
			break
		}
		if i.Lit() == Term && bytes.Equal(i.Value, Tilde) {
			err = i.Next()
			if err != nil {
				break
			}
			err = mergeTest(i.Last, inputs)
			inputs = inputs[:0]
		} else {
			j := i
			inputs = append(inputs, j.Last)
		}
	}
	return err
}

func TestOneSpecialCase(t *testing.T) {
	inputs := [][]byte{[]byte("(@1 two:2)"),
		[]byte("(@2 four:4)")}
	rdxins := [][]byte{}
	for _, in := range inputs {
		rdx, err := ParseJDR(in)
		if err != nil {
			t.Fatal(err)
		}
		rdxins = append(rdxins, rdx)
	}
	out, err := Merge(nil, rdxins)
	if err != nil {
		t.Fatal(err)
	}
	jdr, _, err := WriteJDR(nil, out, 0)
	assert.Equal(t, "(@2 four:4)", string(jdr))
}

func TestFirstMerge(t *testing.T) {
	err := ProcessTestFile("y.FIRST.md", testMerge)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTupleMerge(t *testing.T) {
	err := ProcessTestFile("y.P.md", testMerge)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLinearMerge(t *testing.T) {
	err := ProcessTestFile("y.L.md", testMerge)
	if err != nil {
		t.Fatal(err)
	}
}

func TestEulerMerge(t *testing.T) {
	err := ProcessTestFile("y.E.md", testMerge)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMultixMerge(t *testing.T) {
	err := ProcessTestFile("y.X.md", testMerge)
	if err != nil {
		t.Fatal(err)
	}
}
