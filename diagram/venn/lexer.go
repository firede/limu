package venn

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The tokens of Mermaid's venn grammar, as Pele's T numbers them.
const (
	tEnd = iota
	tError
	tEOF
	tNewline
	tVenn
	tTitle
	tSet
	tUnion
	tText
	tIndentText
	tStyle
	tBracketLabel
	tNumeric
	tHexColor
	tRGBColor
	tRGBAColor
	tIdentifier
	tString
	tComma
	tColon
)

// tokenNames are the tokens' names, as Pele's TOKEN_NAMES and Mermaid's errors give them.
var tokenNames = [...]string{
	"$end", "INVALID", "EOF", "NEWLINE", "VENN", "TITLE", "SET", "UNION", "TEXT", "INDENT_TEXT", "STYLE",
	"BRACKET_LABEL", "NUMERIC", "HEXCOLOR", "RGBCOLOR", "RGBACOLOR", "IDENTIFIER", "STRING", "COMMA", "COLON",
}

// indentHost is the part of the model the lexer talks to, Pele's IndentHost: an indented
// `text` line is a different token after a set or union than anywhere else, and the parser is
// what knows which came before.
type indentHost interface {
	getIndentMode() bool
	setIndentMode(enabled bool)
}

var keywords = []struct {
	word string
	typ  int
}{
	{"venn-beta", tVenn},
	{"set", tSet},
	{"union", tUnion},
	{"text", tText},
	{"style", tStyle},
}

// lexer reads tokens one at a time, as Pele's VennLexer does after Mermaid's Jison lexer for
// venn diagrams: every rule ignores case, the first rule that matches wins, and a keyword must
// end at a word boundary.
type lexer struct {
	// The text and position, in bytes, of the token next returned last.
	text      string
	start     int
	src       string
	host      indentHost
	at        int
	lineStart bool
	ended     bool
}

func newLexer(src string, host indentHost) *lexer { return &lexer{src: src, host: host} }

// unit is the code unit JavaScript's charCodeAt reads at byte i, -1 past the end.
func (l *lexer) unit(i int) int { return chars.Unit(l.src, i) }

// word reports whether the source has word at byte at, its ASCII letters in either case.
func (l *lexer) word(at int, word string) bool {
	for k := 0; k < len(word); k++ {
		c := l.unit(at + k)
		w := int(word[k])
		if c != w && (c+32 != w || c < 65 || c > 90) {
			return false
		}
	}
	return true
}

func (l *lexer) keyword(at int, word string) bool {
	return l.word(at, word) && !chars.IsWord(l.unit(at+len(word)))
}

// color is the end of `rgb(1, 2, 3)` or `rgba(1, 2, 3, 0.5)` whose parenthesis opens at at,
// or -1.
func (l *lexer) color(at, parts int) int {
	i := at + 1
	space := func() {
		for {
			c, size := chars.UnitAt(l.src, i)
			if !chars.IsSpace(c) {
				return
			}
			i += size
		}
	}
	for part := 0; part < parts; part++ {
		space()
		from := i
		for chars.IsDigit(l.unit(i)) || l.unit(i) == 46 {
			i++
		}
		if i == from {
			return -1
		}
		space()
		want := 44
		if part == parts-1 {
			want = 41
		}
		if l.unit(i) != want {
			return -1
		}
		i++
	}
	return i
}

// afterLineFeed reports whether only spaces and tabs stand between a line feed and byte at,
// as Pele's afterLineFeed does.
func (l *lexer) afterLineFeed(at int) bool {
	k := at - 1
	for l.unit(k) == 32 || l.unit(k) == 9 {
		k--
	}
	return l.unit(k) == 10
}

func (l *lexer) lineEnd(from int) int {
	if end := strings.IndexByte(l.src[from:], '\n'); end >= 0 {
		return from + end
	}
	return len(l.src)
}

// next reads the next token, as Pele's next does.
func (l *lexer) next() int {
	src := l.src
	n := len(src)
	for {
		i := l.at
		l.start = i
		l.text = ""
		if i >= n {
			if l.ended {
				return tEnd
			}
			l.ended = true
			return tEOF
		}
		c := l.unit(i)
		if c == 37 && l.unit(i+1) == 37 && l.unit(i+2) != 123 {
			l.at = l.lineEnd(i + 2)
			continue
		}
		if l.lineStart && l.word(i, "title") && l.afterLineFeed(i) && chars.Trim(src[i+5:l.lineEnd(i+5)]) == "" {
			// A bare title is skipped as a comment is, so that it does not end indented text.
			j := l.lineEnd(i + 5)
			for l.unit(j) == 10 || l.unit(j) == 13 {
				j++
			}
			l.at = j
			continue
		}
		if l.lineStart {
			if c == 32 || c == 9 {
				j := i + 1
				for l.unit(j) == 32 || l.unit(j) == 9 {
					j++
				}
				l.at = j
				if l.keyword(j, "text") && l.host.getIndentMode() {
					// The word `text` itself is dropped; the indentation stands for it.
					l.at = j + 4
					l.lineStart = false
					l.text = src[i:j]
					return tIndentText
				}
				continue
			}
			if c != 10 && c != 13 {
				l.host.setIndentMode(false)
				l.lineStart = false
			}
		}
		if c == 10 || c == 13 {
			i++
			for l.unit(i) == 10 || l.unit(i) == 13 {
				i++
			}
			l.lineStart = true
			return l.token(tNewline, i)
		}
		if c == 37 && l.unit(i+1) == 37 {
			l.at = l.lineEnd(i)
			continue
		}
		if c == 32 || c == 9 {
			i++
			for l.unit(i) == 32 || l.unit(i) == 9 {
				i++
			}
			l.at = i
			continue
		}

		if (c|32) >= 97 && (c|32) <= 122 || c == 95 {
			// The text of a title is on its line: a line break after the word is no space.
			if l.word(i, "title") {
				if s, ssize := chars.UnitAt(src, i+5); chars.IsSpace(s) && s != 10 && s != 13 {
					from := i + 5 + ssize
					end := from
					for end < n {
						ch := src[end]
						if ch == 35 || ch == 10 || ch == 59 {
							break
						}
						end++
					}
					if end > from {
						return l.token(tTitle, end)
					}
				}
			}
			for _, k := range keywords {
				// A hyphen continues a name, as in `set-theory`.
				if l.keyword(i, k.word) && l.unit(i+len(k.word)) != 45 {
					return l.token(k.typ, i+len(k.word))
				}
			}
			if l.word(i, "rgb") {
				alpha := l.unit(i+3)|32 == 97
				open := i + 3
				parts := 3
				typ := tRGBColor
				if alpha {
					open, parts, typ = i+4, 4, tRGBAColor
				}
				end := -1
				if l.unit(open) == 40 {
					end = l.color(open, parts)
				}
				if end != -1 {
					return l.token(typ, end)
				}
			}
			i++
			for chars.IsWord(l.unit(i)) || l.unit(i) == 45 {
				i++
			}
			return l.token(tIdentifier, i)
		}
		if c == 91 {
			if l.unit(i+1) == 34 {
				quote := strings.IndexByte(src[i+2:], '"')
				if quote == -1 || l.unit(i+2+quote+1) != 93 {
					return tError
				}
				quote += i + 2
				l.at = quote + 2
				l.text = src[i+2 : quote]
				return tBracketLabel
			}
			end := i + 1
			for end < n && src[end] != 93 && src[end] != 34 {
				end++
			}
			if end == i+1 || l.unit(end) != 93 {
				return tError
			}
			l.at = end + 1
			l.text = chars.Trim(src[i+1 : end])
			return tBracketLabel
		}
		if chars.IsDigit(c) || c == 46 || c == 43 || c == 45 {
			if c == 43 || c == 45 {
				i++
			}
			from := i
			for chars.IsDigit(l.unit(i)) {
				i++
			}
			if i > from {
				if l.unit(i) == 46 && chars.IsDigit(l.unit(i+1)) {
					i += 2
					for chars.IsDigit(l.unit(i)) {
						i++
					}
				}
			} else if l.unit(i) == 46 && chars.IsDigit(l.unit(i+1)) {
				i += 2
				for chars.IsDigit(l.unit(i)) {
					i++
				}
			} else {
				return tError
			}
			return l.token(tNumeric, i)
		}
		if c == 35 {
			end := i + 1
			for end < i+9 && chars.IsHex(l.unit(end)) {
				end++
			}
			if end >= i+4 {
				return l.token(tHexColor, end)
			}
			return tError
		}
		if c == 34 {
			quote := strings.IndexByte(src[i+1:], '"')
			if quote == -1 {
				return tError
			}
			return l.token(tString, i+1+quote+1)
		}
		if c == 44 {
			return l.token(tComma, i+1)
		}
		if c == 58 {
			return l.token(tColon, i+1)
		}
		return tError
	}
}

func (l *lexer) token(typ, end int) int {
	l.text = l.src[l.at:end]
	l.at = end
	return typ
}
