package journey

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The types of token, in the order of Pele's T.
const (
	tEnd = iota
	tEOF
	tInvalid
	tNewline
	tJourney
	tTitle
	tAccTitle
	tAccTitleValue
	tAccDescr
	tAccDescrValue
	tAccDescrMultilineValue
	tSection
	tTaskName
	tTaskData
	tColon
)

// tokenNames are the names Pele's TOKEN_NAMES gives the types, as its errors write them.
var tokenNames = []string{
	"$end",
	"EOF",
	"INVALID",
	"NEWLINE",
	"journey",
	"title",
	"acc_title",
	"acc_title_value",
	"acc_descr",
	"acc_descr_value",
	"acc_descr_multiline_value",
	"section",
	"taskName",
	"taskData",
	":",
}

// tokens are what the lexer makes of a source, Pele's Tokens: the type of each token, its
// text, and the byte it starts at.
type tokens struct {
	types  []int
	texts  []string
	starts []int
}

func (t *tokens) emit(typ int, src string, start, end int) {
	t.types = append(t.types, typ)
	t.texts = append(t.texts, src[start:end])
	t.starts = append(t.starts, start)
}

// The lexer rules that Mermaid's journey and timeline grammars have in common, as Pele's
// diagrams/common/scan.ts has them, ported from the compiled Jison rules: comments, white
// space, and the accTitle and accDescr statements.

// lineEnd is where the line that from is on ends, as Pele's lineEnd finds it.
func lineEnd(src string, from int) int {
	if at := strings.IndexByte(src[from:], '\n'); at >= 0 {
		return from + at
	}
	return len(src)
}

// keywordAt reports whether word, in lowercase ASCII letters, is at byte p in any case, as
// Pele's keywordAt does.
func keywordAt(src string, p int, word string) bool {
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

// nextUnit is the code unit JavaScript reads after the character at byte p, and the byte it
// is at: after a character above the Basic Multilingual Plane, it reads the low surrogate,
// which is no character the lexer looks for.
func nextUnit(src string, p int) (c, at int) {
	_, size := chars.UnitAt(src, p)
	if size == 4 {
		return 0xdc00, p + size
	}
	return chars.Unit(src, p+size), p + size
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
		if c == '%' && chars.Unit(src, p+1) != '{' {
			p = lineEnd(src, p)
		} else if c2, at := nextUnit(src, p); c != '}' && c2 == '%' && chars.Unit(src, at+1) == '%' {
			p = lineEnd(src, at+2)
		} else if c == '\n' {
			return p
		} else if chars.IsSpace(c) {
			p = chars.SpaceRun(src, p+size)
		} else if c == '#' {
			p = lineEnd(src, p)
		} else {
			return p
		}
	}
	return p
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
	q := chars.SpaceRun(src, p+8)
	c := chars.Unit(src, q)
	if c == ':' {
		q = chars.SpaceRun(src, q+1)
		end := lineEnd(src, q)
		if descr {
			out.emit(first+2, src, p, q)
			out.emit(first+3, src, q, end)
		} else {
			out.emit(first, src, p, q)
			out.emit(first+1, src, q, end)
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
		out.emit(first+4, src, q, end)
		if q == n {
			return n
		}
		q = end
	}
	return q + 1
}

// textEnd is where text from from ends, as Pele's textEnd finds it: `#`, a line break and
// `;` end every kind of text, and a colon also ends a task name or a section.
func textEnd(src string, from int, colon bool) int {
	e := from
	for ; e < len(src); e++ {
		c := src[e]
		if c == '#' || c == '\n' || c == ';' || colon && c == ':' {
			break
		}
	}
	return e
}

// tokenize lexes a journey diagram, as Pele's tokenize does.
func tokenize(src string) *tokens {
	n := len(src)
	out := &tokens{}
	p := 0
	emit := func(typ, e int) {
		out.emit(typ, src, p, e)
		p = e
	}
	for {
		p = skipTrivia(src, p)
		if p >= n {
			break
		}
		c := src[p]
		if c == '\n' {
			e := p + 1
			for e < n && src[e] == '\n' {
				e++
			}
			emit(tNewline, e)
			continue
		}
		if keywordAt(src, p, "journey") && !chars.IsWord(chars.Unit(src, p+7)) {
			emit(tJourney, p+7)
			continue
		}
		// \s after a keyword also takes a line break, so `title` alone on a line takes the
		// next line as its text.
		if keywordAt(src, p, "title") {
			if space, size := chars.UnitAt(src, p+5); chars.IsSpace(space) {
				from := p + 5 + size
				if e := textEnd(src, from, false); e > from {
					emit(tTitle, e)
					continue
				}
			}
		}
		if e := accessibility(src, p, out, tAccTitle); e != p {
			p = e
			continue
		}
		if keywordAt(src, p, "section") {
			if space, size := chars.UnitAt(src, p+7); chars.IsSpace(space) {
				from := p + 7 + size
				if e := textEnd(src, from, true); e > from {
					emit(tSection, e)
					continue
				}
			}
		}
		switch c {
		case ':':
			if e := textEnd(src, p+1, false); e > p+1 {
				emit(tTaskData, e)
			} else {
				emit(tColon, p+1)
			}
		case ';':
			emit(tInvalid, p+1)
		default:
			_, size := chars.UnitAt(src, p)
			emit(tTaskName, textEnd(src, p+size, true))
		}
	}

	// Jison marks its lexer done when a rule runs on empty input, so an accessibility value
	// that matches nothing at the very end of the text is followed by no EOF token.
	if len(out.starts) == 0 || out.starts[len(out.starts)-1] < n {
		emit(tEOF, n)
	}
	emit(tEnd, n)
	return out
}
