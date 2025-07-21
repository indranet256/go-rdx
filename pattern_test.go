package rdx

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type match struct {
	pattern string
	jdr     string
	len     int
}

func TestMatchRDX(t *testing.T) {
	cases := []match{
		match{"ist", "1 \"one\" one", 3},
		match{"(ttt)", "(one two three)", 5},
		match{"(ttt)[ii]", "(one two three)[1 2]", 9},
		match{"(ttt)[ii]", "(one two three)[no no]", 0},
		match{"{(is)(is)}", "{1:\"one\", 2:\"two\"}", 10},
	}
	for _, c := range cases {
		matches, _ := MatchJDR(c.pattern, []byte(c.jdr))
		assert.Equal(t, c.len, len(matches))
	}
}
