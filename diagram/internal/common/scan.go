package common

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The lexer rules that Mermaid's journey and timeline grammars have in common, as Pele's
// diagrams/common/scan.ts has them, ported from the compiled Jison rules: comments, white
// space, and the accTitle and accDescr statements. Positions are bytes; where Pele reads a
// code unit with charCodeAt, chars.UnitAt reads the same one, and a character outside ASCII
// is stepped over whole.

// JisonTokens are the tokens of a lexer ported from Jison, the Tokens of Pele's scan.ts: the
// type of each token, its text, and the byte it starts at.
type JisonTokens struct {
	Types  []int
	Texts  []string
	Starts []int
}

// Emit adds a token of type typ that runs from byte start to byte end of src.
func (t *JisonTokens) Emit(typ int, src string, start, end int) {
	t.Types = append(t.Types, typ)
	t.Texts = append(t.Texts, src[start:end])
	t.Starts = append(t.Starts, start)
}

// LineEnd is where the line that byte from is on ends, as Pele's lineEnd finds it.
func LineEnd(src string, from int) int {
	if at := strings.IndexByte(src[from:], '\n'); at >= 0 {
		return from + at
	}
	return len(src)
}

// KeywordAt reports whether word, in lowercase ASCII letters, is at byte p in any case, as
// Pele's keywordAt does: the grammars are case-insensitive.
func KeywordAt(src string, p int, word string) bool {
	if p+len(word) > len(src) {
		return false
	}
	for i := 0; i < len(word); i++ {
		if src[p+i]|32 != word[i] {
			return false
		}
	}
	return true
}

// SkipTrivia skips comments and white space, as Pele's skipTrivia does. It stops at a line
// break that starts a token, which is a NEWLINE. Jison compiles `\%%` to `%`, so a single `%`
// starts a comment.
func SkipTrivia(src string, from int) int {
	n := len(src)
	p := from
	for p < n {
		c, size := chars.UnitAt(src, p)
		switch {
		case c == '%' && chars.Unit(src, p+1) != '{':
			p = LineEnd(src, p)
		case c == '\n':
			return p
		case chars.IsSpace(c):
			p = chars.SpaceRun(src, p+size)
		case c == '#':
			p = LineEnd(src, p)
		default:
			return p
		}
	}
	return p
}

// Accessibility lexes an accTitle or accDescr statement at byte p and returns where it ends,
// or p if there is none, as Pele's accessibility does. The token types acc_title,
// acc_title_value, acc_descr, acc_descr_value and acc_descr_multiline_value are numbered
// consecutively from first.
func Accessibility(src string, p int, out *JisonTokens, first int) int {
	descr := KeywordAt(src, p, "accdescr")
	if !descr && !KeywordAt(src, p, "acctitle") {
		return p
	}
	n := len(src)
	q := chars.SpaceRun(src, p+8)
	c := chars.Unit(src, q)
	if c == ':' {
		q = chars.SpaceRun(src, q+1)
		end := LineEnd(src, q)
		if descr {
			out.Emit(first+2, src, p, q)
			out.Emit(first+3, src, q, end)
		} else {
			out.Emit(first, src, p, q)
			out.Emit(first+1, src, q, end)
		}
		return end
	}
	if !descr || c != '{' {
		return p
	}
	q = chars.SpaceRun(src, q+1)
	for chars.Unit(src, q) != '}' {
		end := n
		if at := strings.IndexByte(src[q:], '}'); at >= 0 {
			end = q + at
		}
		out.Emit(first+4, src, q, end)
		if q == n {
			return n
		}
		q = end
	}
	return q + 1
}

// WordAt reports whether word, written in lower case, is at byte at of src, as Pele's wordAt
// (common/caseless.ts) does: only ASCII letters match in either case, as with the i flag on
// a regular expression without the u flag. KeywordAt folds any character with |32, which
// would take a carriage return for the dash of x-axis.
func WordAt(src string, at int, word string) bool {
	if at+len(word) > len(src) {
		return false
	}
	for k := 0; k < len(word); k++ {
		c, w := src[at+k], word[k]
		if c != w && !(w >= 'a' && w <= 'z' && c == w-32) {
			return false
		}
	}
	return true
}
