package timeline

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The lexer rules that Mermaid's journey and timeline grammars have in common, as Pele's
// diagrams/common/scan.ts has them, ported from the compiled Jison rules: comments, white
// space, and the accTitle and accDescr statements. Positions are bytes; where Pele reads a
// code unit with charCodeAt, chars.UnitAt reads the same one, and a character outside ASCII
// is stepped over whole.

// tokens are a lexer's tokens, Pele's Tokens: the type, the text and the byte it starts at of
// each.
type tokens struct {
	types  []int
	texts  []string
	starts []int
}

// lineEnd is where the line that from is on ends, as Pele's lineEnd finds it.
func lineEnd(src string, from int) int {
	if at := strings.IndexByte(src[from:], '\n'); at != -1 {
		return from + at
	}
	return len(src)
}

// keywordAt reports whether word, in lowercase ASCII letters, is at p in any case, as Pele's
// keywordAt does.
func keywordAt(src string, p int, word string) bool {
	for i := 0; i < len(word); i++ {
		if p+i >= len(src) || src[p+i]|32 != word[i] {
			return false
		}
	}
	return true
}

// skipTrivia skips comments and white space, as Pele's skipTrivia does. It stops at a line
// break that starts a token, which is a NEWLINE. Jison compiles `\%%` to a single `%`, so one
// percent sign starts a comment, and the second comment rule takes any character before `%%`
// with it, a line break included.
func skipTrivia(src string, from int) int {
	n := len(src)
	p := from
	for p < n {
		c, size := chars.UnitAt(src, p)
		// The code unit after c: past a character above the Basic Multilingual Plane it is
		// the low surrogate, which is no percent sign.
		next := -1
		if size < 4 {
			next = chars.Unit(src, p+size)
		}
		switch {
		case c == 37 && chars.Unit(src, p+1) != 123:
			p = lineEnd(src, p)
		case c != 125 && next == 37 && chars.Unit(src, p+size+1) == 37:
			p = lineEnd(src, p+size+2)
		case c == 10:
			return p
		case chars.IsSpace(c):
			for p < n {
				c, size := chars.UnitAt(src, p)
				if !chars.IsSpace(c) {
					break
				}
				p += size
			}
		case c == 35:
			p = lineEnd(src, p)
		default:
			return p
		}
	}
	return p
}

// skipSpace skips white space, line breaks included.
func skipSpace(src string, from int) int {
	p := from
	for {
		c, size := chars.UnitAt(src, p)
		if !chars.IsSpace(c) {
			return p
		}
		p += size
	}
}

// accessibility lexes an accTitle or accDescr statement at p and returns where it ends, or p
// if there is none, as Pele's accessibility does. The token types acc_title,
// acc_title_value, acc_descr, acc_descr_value and acc_descr_multiline_value are numbered
// consecutively from first.
func accessibility(src string, p int, out *tokens, first int) int {
	descr := keywordAt(src, p, "accdescr")
	if !descr && !keywordAt(src, p, "acctitle") {
		return p
	}
	n := len(src)
	q := skipSpace(src, p+8)
	c := chars.Unit(src, q)
	emit := func(typ, s, e int) {
		out.types = append(out.types, typ)
		out.texts = append(out.texts, src[s:e])
		out.starts = append(out.starts, s)
	}
	if c == 58 {
		q = skipSpace(src, q+1)
		end := lineEnd(src, q)
		if descr {
			emit(first+2, p, q)
			emit(first+3, q, end)
		} else {
			emit(first, p, q)
			emit(first+1, q, end)
		}
		return end
	}
	if !descr || c != 123 {
		return p
	}
	q = skipSpace(src, q+1)
	for chars.Unit(src, q) != 125 {
		end := strings.IndexByte(src[q:], '}')
		if end == -1 {
			end = n
		} else {
			end += q
		}
		emit(first+4, q, end)
		if q == n {
			return n
		}
		q = end
	}
	return q + 1
}
