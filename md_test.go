package rdx

import (
	"bytes"
	"errors"
	"fmt"
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
			return errors.New(fmt.Sprintf("bad order %d and %d\n%s\n%s", n, n+1, pj, tj))
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
	merged, err = MergeP(nil, inputs)
	if err != nil {
		return err
	}
	if !bytes.Equal(correct, merged) {
		pj, _, _ := WriteJDR(nil, correct, 0)
		tj, _, _ := WriteJDR(nil, merged, 0)
		return errors.New(fmt.Sprintf("bad merge\n%s\n%s", pj, tj))
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
		if i.Lit == Term && bytes.Equal(i.Value, Tilde) {
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

func TestFirstMerge(t *testing.T) {
	err := ProcessTestFile("y.FIRST.md", testMerge)
	if err != nil {
		t.Fatal(err)
	}
}
