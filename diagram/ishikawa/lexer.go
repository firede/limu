package ishikawa

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The types of token, as Pele's lexer numbers them.
const (
	tokEnd = iota
	tokEOF
	tokSpaceLine
	tokNL
	tokIshikawa
	tokSpaceList
	tokText
)

// tokenNames are the names Jison gives the types of token in its errors, Pele's TOKEN_NAMES.
var tokenNames = [...]string{"$end", "EOF", "SPACELINE", "NL", "ISHIKAWA", "SPACELIST", "TEXT"}

// tokens are the tokens of a text: the type of each, and the bytes it starts and ends at.
type tokens struct {
	types, starts, ends []int
}

// keyword is the length of the keyword at byte at of src, or 0 if there is none, as Pele's
// keyword reads it: `ishikawa-beta`, or failing that `ishikawa`, in any case, either one
// ending at a word boundary.
func keyword(src string, at int) int {
	const word = "ishikawa-beta"
	k := 0
	for k < 13 {
		c := chars.Unit(src, at+k)
		w := int(word[k])
		if c != w && (c+32 != w || c < 65 || c > 90) {
			break
		}
		k++
	}
	if k == 13 && !chars.IsWord(chars.Unit(src, at+13)) {
		return 13
	}
	if k >= 8 && !chars.IsWord(chars.Unit(src, at+8)) {
		return 8
	}
	return 0
}

// tokenize makes the tokens of Mermaid's Jison lexer for Ishikawa diagrams, as Pele's
// tokenize does, where the first rule that matches wins: white space up to a comment or up to
// the last line break in it is a blank line, a single line break before text is a new line,
// other white space is indentation, and the rest of a line is text.
func tokenize(src string) tokens {
	var t tokens
	n := len(src)
	i := 0
	for i < n {
		j := i
		lastBreak := -1
		for j < n {
			c, size := chars.UnitAt(src, j)
			if !chars.IsSpace(c) {
				break
			}
			if c == '\n' {
				lastBreak = j
			}
			j += size
		}
		var typ, end int
		switch {
		case chars.Unit(src, j) == '%' && chars.Unit(src, j+1) == '%':
			typ = tokSpaceLine
			end = j + 2
			for end < n {
				c, size := chars.UnitAt(src, end)
				if chars.IsLineEnd(c) {
					break
				}
				end += size
			}
		case j == i:
			if length := keyword(src, i); length > 0 {
				typ = tokIshikawa
				end = i + length
			} else {
				typ = tokText
				end = n
				if at := strings.IndexByte(src[i:], '\n'); at >= 0 {
					end = i + at
				}
			}
		case lastBreak > i:
			typ = tokSpaceLine
			end = lastBreak + 1
		case lastBreak == i:
			typ = tokNL
			end = i + 1
		default:
			typ = tokSpaceList
			end = j
		}
		t.types = append(t.types, typ)
		t.starts = append(t.starts, i)
		t.ends = append(t.ends, end)
		i = end
	}
	t.types = append(t.types, tokEOF, tokEnd)
	t.starts = append(t.starts, n, n)
	t.ends = append(t.ends, n, n)
	return t
}
