package rdx

import "errors"

var ErrPatternMismatch = errors.New("pattern mismatch")

func MatchJDR(pattern string, jdr []byte) (matches [][]byte, err error) {
	var rdx []byte
	rdx, err = ParseJDR(jdr)
	if err == nil {
		matches, err = MatchRDX(pattern, rdx)
	}
	return
}

func MatchRDX(pattern string, rdx []byte) (matches [][]byte, err error) {
	pat := make([]byte, 0, len(pattern))
	for _, p := range pattern {
		if p != ' ' && p != '\n' && p != '\t' {
			pat = append(pat, byte(p))
		}
	}
	r := rdx
	err = matchRDX(&pat, &r, &matches)
	return
}

func matchRDX(pattern *[]byte, rdxes *[]byte, match *[][]byte) (err error) {
	pat := *pattern
	rdx := *rdxes
	mat := *match
	back := false
	for len(pat) > 0 && !back {
		var lit byte
		var val []byte
		lit, _, val, rdx, err = ReadRDX(rdx)
		p := pat[0]
		pat = pat[1:]
		switch p {
		case 'f':
			if lit != Float {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
		case 'i':
			if lit != Integer {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
		case 'r':
			if lit != Reference {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
		case 's':
			if lit != String {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
		case 't':
			if lit != Term {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
		case '(':
			if lit != Tuple {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
			err = matchRDX(&pat, &val, &mat)
			if err != nil {
				return
			}
		case ')':
			if len(rdx) > 0 {
				return ErrPatternMismatch
			}
			mat = append(mat, nil)
			back = true
		case '[':
			if lit != Linear {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
			err = matchRDX(&pat, &val, &mat)
			if err != nil {
				return
			}
		case ']':
			if len(rdx) > 0 {
				return ErrPatternMismatch
			}
			mat = append(mat, nil)
			back = true
		case '{':
			if lit != Euler {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
			err = matchRDX(&pat, &val, &mat)
			if err != nil {
				return
			}
		case '}':
			if len(rdx) > 0 {
				return ErrPatternMismatch
			}
			mat = append(mat, nil)
			back = true
		case '<':
			if lit != Multix {
				return ErrPatternMismatch
			}
			mat = append(mat, val)
			err = matchRDX(&pat, &val, &mat)
			if err != nil {
				return
			}
		case '>':
			if len(rdx) > 0 {
				return ErrPatternMismatch
			}
			mat = append(mat, nil)
			back = true
		}
	}
	*match = mat
	*rdxes = rdx
	*pattern = pat
	return
}
