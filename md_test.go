package rdx

import (
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
