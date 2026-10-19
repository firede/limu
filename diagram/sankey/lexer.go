package sankey

import (
	"unicode/utf8"

	"github.com/firede/limu/internal/chars"
)

// The tokens of Pele's sankey lexer, in its order, which is Jison's.
const (
	tEND = iota
	// tSTUCK is not a Jison token: the reference lexer would return the same empty token
	// forever from here.
	tSTUCK
	tSANKEY
	tNEWLINE
	tCOMMA
	tDQUOTE
	tNON_ESCAPED_TEXT
	tESCAPED_TEXT
	tEOF
)

var tokenNames = [...]string{
	"$end",
	"STUCK",
	"SANKEY",
	"NEWLINE",
	"COMMA",
	"DQUOTE",
	"NON_ESCAPED_TEXT",
	"ESCAPED_TEXT",
	"EOF",
}

// tokens are the tokens of a text, as Pele's Tokens: each one's type, its text, and the
// byte it starts at.
type tokens struct {
	Types  []int
	Texts  []string
	Starts []int
}

// The lexer's start conditions.
const (
	sINITIAL = iota
	sCsv
	sEscapedText
)

// isTextData reports whether c is RFC 4180's TEXTDATA: printable ASCII but for the double
// quote and the comma.
func isTextData(c int) bool {
	return c >= 32 && c <= 126 && c != 34 && c != 44
}

// keyword reports whether word, in lowercase ASCII, is at byte p of src in any case, and not
// followed by a letter, a digit or an underscore, as Pele's lexer's keyword does: the next
// len(word) code units, lowered as toLowerCase lowers them, are the word. The Kelvin sign
// lowers to a k, so it can stand for one. It gives the byte the word ends at.
func keyword(src string, p int, word string) (int, bool) {
	end, units := p, 0
	for end < len(src) && units < len(word) {
		r, size := utf8.DecodeRuneInString(src[end:])
		end += size
		units++
		if r > 0xffff {
			units++
		}
	}
	if units != len(word) || chars.ToLower(src[p:end]) != word || chars.IsWord(chars.Unit(src, end)) {
		return 0, false
	}
	return end, true
}

// tokenize splits a text into tokens, as Pele's tokenize does.
func tokenize(src string) tokens {
	n := len(src)
	var t tokens
	var stack []int
	state := sINITIAL
	p := 0

	emit := func(typ, s, e int) {
		t.Types = append(t.Types, typ)
		t.Texts = append(t.Texts, src[s:e])
		t.Starts = append(t.Starts, s)
		p = e
	}

	for p < n {
		c := int(src[p])
		start := p
		q := p
		if state == sEscapedText {
			if c == 34 && chars.Unit(src, p+1) != 34 {
				if len(stack) > 0 {
					state = stack[len(stack)-1]
					stack = stack[:len(stack)-1]
				}
				emit(tDQUOTE, p, p+1)
				continue
			}
			for q < n {
				d := int(src[q])
				if isTextData(d) || d == 44 || d == 13 || d == 10 {
					q++
				} else if d == 34 && chars.Unit(src, q+1) == 34 {
					q += 2
				} else {
					break
				}
			}
			emit(tESCAPED_TEXT, p, q)
		} else {
			if state == sINITIAL && c|32 == 115 {
				end, ok := keyword(src, p, "sankey-beta")
				if !ok {
					end, ok = keyword(src, p, "sankey")
				}
				if ok {
					stack = append(stack, state)
					state = sCsv
					emit(tSANKEY, p, end)
					continue
				}
			}
			if c == 10 || c == 13 && chars.Unit(src, p+1) == 10 {
				if c == 10 {
					emit(tNEWLINE, p, p+1)
				} else {
					emit(tNEWLINE, p, p+2)
				}
				continue
			}
			if c == 44 {
				emit(tCOMMA, p, p+1)
				continue
			}
			if c == 34 {
				stack = append(stack, state)
				state = sEscapedText
				emit(tDQUOTE, p, p+1)
				continue
			}
			for q < n && isTextData(int(src[q])) {
				q++
			}
			emit(tNON_ESCAPED_TEXT, p, q)
		}
		// A character outside the CSV alphabet matches as empty text and is never consumed.
		if q == start {
			emit(tSTUCK, p, p)
			return t
		}
	}

	if state == sEscapedText {
		emit(tESCAPED_TEXT, n, n)
	} else {
		emit(tEOF, n, n)
	}
	emit(tEND, n, n)
	return t
}
