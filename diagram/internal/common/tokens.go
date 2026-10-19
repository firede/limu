package common

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// The tokenizer of the diagram types Mermaid defines with Langium grammars, as Pele's
// diagrams/common/tokens.ts has it: token types are tried in order and the first that matches
// wins, as in Chevrotain, but for the longer alternatives a type names. The regular
// expressions of Pele's patterns are written out by hand, with the backtracking they would
// do. Positions are bytes.

// A TokenType is Pele's TokenType. A type matches its Literal, or, without one, by Match,
// which gives the end of the match at a byte of the text, or -1: Pele's pattern, when that is
// a regular expression, or its match.
type TokenType struct {
	Name    string
	Literal string
	Match   func(src string, at int) int
	// Scanner makes a matcher that answers as Match would, for a pattern that could rescan
	// the text, as Pele's scanner does: it is called once for each text tokenized, so the
	// matcher may remember what it has scanned. A type with a Scanner needs no Match.
	Scanner func() func(src string, at int) int
	// Opener reports whether the start of a construct that needs a closer is at a byte. When
	// it is but the whole type does not match, no closer is in the rest of the text, and the
	// type is not tried again.
	Opener func(src string, at int) bool
	// First is every ASCII character a match can start with, or "" when that is not known.
	// A type with a First is tried only at those characters and at characters outside ASCII.
	First  string
	Hidden bool
	// Longer are Chevrotain's LONGER_ALT, by index in the table: when this type matches,
	// these are tried at the same place, and the first that matches more text becomes the
	// token.
	Longer []int
}

// Tokens are Pele's Tokens: the kind of each token, by index in the table, and the bytes it
// starts and ends at, with a last of kind -1 at the end of the text.
type Tokens struct {
	Kinds, Starts, Ends []int
	Src                 string
}

// A Table is a table of token types made ready to tokenize with, as Pele's prepare makes it:
// the types, and those that can start with each ASCII character, in matching order.
type Table struct {
	Types    []TokenType
	starting [128][]int
	every    []int
}

// Prepare makes a table of token types ready to tokenize with, as Pele's prepare does.
func Prepare(types []TokenType) *Table {
	t := &Table{Types: types}
	for k := range types {
		t.every = append(t.every, k)
	}
	for c := range 128 {
		for k, typ := range types {
			if typ.Literal != "" && typ.Literal[0] == byte(c) ||
				typ.Literal == "" && (typ.First == "" || strings.IndexByte(typ.First, byte(c)) >= 0) {
				t.starting[c] = append(t.starting[c], k)
			}
		}
	}
	return t
}

// Tokenize splits src into tokens with the types of a table, as Pele's tokenize does. Text
// no type matches is a lexical error of the diagram type named.
func Tokenize(src string, table *Table, typ diagram.Type) (*Tokens, error) {
	types := table.Types
	out := &Tokens{Src: src}
	dead := make([]bool, len(types))
	scanners := make([]func(string, int) int, len(types))
	match := func(k, p int) int {
		if dead[k] {
			return -1
		}
		t := &types[k]
		if t.Scanner != nil {
			if scanners[k] == nil {
				scanners[k] = t.Scanner()
			}
			return scanners[k](src, p)
		}
		if t.Literal != "" {
			if strings.HasPrefix(src[p:], t.Literal) {
				return p + len(t.Literal)
			}
			return -1
		}
		end := t.Match(src, p)
		if end < 0 && t.Opener != nil && t.Opener(src, p) {
			dead[k] = true
		}
		return end
	}
	n := len(src)
	p := 0
scan:
	for p < n {
		candidates := table.every
		if c := src[p]; c < 128 {
			candidates = table.starting[c]
		}
		for _, k := range candidates {
			end := match(k, p)
			if end <= p {
				continue
			}
			kind := k
			for _, alt := range types[k].Longer {
				if altEnd := match(alt, p); altEnd > end {
					kind, end = alt, altEnd
					break
				}
			}
			if !types[kind].Hidden {
				out.Kinds = append(out.Kinds, kind)
				out.Starts = append(out.Starts, p)
				out.Ends = append(out.Ends, end)
			}
			p = end
			continue scan
		}
		return nil, diagram.SyntaxError(typ, src, p, "", true)
	}
	out.Kinds = append(out.Kinds, -1)
	out.Starts = append(out.Starts, n)
	out.Ends = append(out.Ends, n)
	return out, nil
}

// blanks is the end of the run of tabs and spaces from byte i, [\t ]*.
func blanks(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

// spaceAt reports whether JavaScript's \s matches at byte i of s.
func spaceAt(s string, i int) bool { return chars.IsSpace(chars.Unit(s, i)) }

// lineRest matches [^\n\r]*?(?=%%)|[^\n\r]* from byte i: the text up to the first %% on the
// line, or to the end of the line.
func lineRest(s string, i int) int {
	for k := i; k < len(s); k++ {
		if s[k] == '\n' || s[k] == '\r' || strings.HasPrefix(s[k:], "%%") {
			return k
		}
	}
	return len(s)
}

// LineBreak matches \r?\n at byte i, or gives -1.
func LineBreak(s string, i int) int {
	switch {
	case strings.HasPrefix(s[i:], "\r\n"):
		return i + 2
	case strings.HasPrefix(s[i:], "\n"):
		return i + 1
	}
	return -1
}

// Closed matches (?:\r?\n|(?!\S)) at byte i: a line break, or nothing before white space or
// the end of the text; -1 where neither is.
func Closed(s string, i int) int {
	if end := LineBreak(s, i); end >= 0 {
		return end
	}
	if i == len(s) || spaceAt(s, i) {
		return i
	}
	return -1
}

// AccDescrToken is Pele's ACC_DESCR,
// /[\t ]*accDescr(?:[\t ]*:([^\n\r]*?(?=%%)|[^\n\r]*)|\s*{([^}]*)})/y.
var AccDescrToken = TokenType{Name: "ACC_DESCR", First: "\t a", Match: func(s string, at int) int {
	i := blanks(s, at)
	if !strings.HasPrefix(s[i:], "accDescr") {
		return -1
	}
	i += len("accDescr")
	if j := blanks(s, i); j < len(s) && s[j] == ':' {
		return lineRest(s, j+1)
	}
	i = chars.SpaceRun(s, i)
	if i == len(s) || s[i] != '{' {
		return -1
	}
	end := strings.IndexByte(s[i+1:], '}')
	if end < 0 {
		return -1
	}
	return i + 1 + end + 1
}}

// AccDescr is Pele's accDescr(name) (common/accDescr.ts): ACC_DESCR under the name given,
// for grammars where lexing goes on after an `accDescr {` that never closes, because
// accDescr is also a name there. It matches what AccDescrToken does, but once the braced form
// finds no closing brace there is none further on, so later braced descriptions fail
// without scanning the rest of the text again.
func AccDescr(name string) TokenType {
	return TokenType{Name: name, First: AccDescrToken.First, Scanner: accDescrScanner}
}

// accDescrScanner is the scanner of Pele's accDescr, which matches
// /[\t ]*accDescr(?:[\t ]*:(?:[^\n\r]*?(?=%%)|[^\n\r]*)|\s*({))/y and then, for the braced
// form, looks for the closing brace unless it has found none before.
func accDescrScanner() func(string, int) int {
	unclosed := false
	return func(s string, at int) int {
		i := blanks(s, at)
		if !strings.HasPrefix(s[i:], "accDescr") {
			return -1
		}
		i += len("accDescr")
		if j := blanks(s, i); j < len(s) && s[j] == ':' {
			return lineRest(s, j+1)
		}
		i = chars.SpaceRun(s, i)
		if i == len(s) || s[i] != '{' || unclosed {
			return -1
		}
		end := strings.IndexByte(s[i+1:], '}')
		unclosed = end < 0
		if end < 0 {
			return -1
		}
		return i + 1 + end + 1
	}
}

// AccTitleToken is Pele's ACC_TITLE, /[\t ]*accTitle[\t ]*:(?:[^\n\r]*?(?=%%)|[^\n\r]*)/y.
var AccTitleToken = TokenType{Name: "ACC_TITLE", First: "\t a", Match: func(s string, at int) int {
	i := blanks(s, at)
	if !strings.HasPrefix(s[i:], "accTitle") {
		return -1
	}
	i = blanks(s, i+len("accTitle"))
	if i == len(s) || s[i] != ':' {
		return -1
	}
	return lineRest(s, i+1)
}}

// TitleToken is Pele's TITLE, /[\t ]*title(?:[\t ][^\n\r]*?(?=%%)|[\t ][^\n\r]*|)/y: the word
// alone matches too, so `title1` starts with a title.
var TitleToken = TokenType{Name: "TITLE", First: "\t t", Match: func(s string, at int) int {
	i := blanks(s, at)
	if !strings.HasPrefix(s[i:], "title") {
		return -1
	}
	i += len("title")
	if i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		return lineRest(s, i+1)
	}
	return i
}}

// StringToken is Pele's STRING, /"([^"\\]|\\.)*"|'([^'\\]|\\.)*'/y. Each character has one
// way to be taken, so the first quote that ends it is its end; a backslash before a line
// terminator, which . does not take, ends no string.
var StringToken = TokenType{Name: "STRING", First: `"'`, Match: func(s string, at int) int {
	q := s[at]
	if q != '"' && q != '\'' {
		return -1
	}
	for i := at + 1; i < len(s); {
		switch s[i] {
		case q:
			return i + 1
		case '\\':
			c, size := chars.UnitAt(s, i+1)
			if c < 0 || chars.IsLineEnd(c) {
				return -1
			}
			i += 1 + size
		default:
			i++
		}
	}
	return -1
}}

// NewlineToken is Pele's NEWLINE, /\r?\n/y.
var NewlineToken = TokenType{Name: "NEWLINE", First: "\r\n", Match: LineBreak}

// WhitespaceToken is Pele's WHITESPACE, /[\t ]+/y, which is hidden.
var WhitespaceToken = TokenType{Name: "WHITESPACE", First: "\t ", Hidden: true, Match: func(s string, at int) int {
	if end := blanks(s, at); end > at {
		return end
	}
	return -1
}}

// yamlOpener matches /---[\t ]*\r?\n/y at byte at, or gives -1.
func yamlOpener(s string, at int) int {
	if !strings.HasPrefix(s[at:], "---") {
		return -1
	}
	return LineBreak(s, blanks(s, at+3))
}

// YAMLToken is Pele's YAML, /---[\t ]*\r?\n(?:[\S\s]*?\r?\n)?---(?:\r?\n|(?!\S))/y, front
// matter, which is hidden. The group is tried first, so the closing dashes are the first
// after a line break, past any that follow the opener at once.
var YAMLToken = TokenType{
	Name: "YAML", First: "-", Hidden: true,
	Opener: func(s string, at int) bool { return yamlOpener(s, at) >= 0 },
	Match: func(s string, at int) int {
		q := yamlOpener(s, at)
		if q < 0 {
			return -1
		}
		for k := q; k < len(s); k++ {
			j := LineBreak(s, k)
			if j < 0 || !strings.HasPrefix(s[j:], "---") {
				continue
			}
			if end := Closed(s, j+3); end >= 0 {
				return end
			}
		}
		if strings.HasPrefix(s[q:], "---") {
			return Closed(s, q+3)
		}
		return -1
	},
}

// directiveOpener matches /[\t ]*%%{/y at byte at, or gives -1.
func directiveOpener(s string, at int) int {
	i := blanks(s, at)
	if !strings.HasPrefix(s[i:], "%%{") {
		return -1
	}
	return i + 3
}

// DirectiveToken is Pele's DIRECTIVE, /[\t ]*%%{[\S\s]*?}%%(?:\r?\n|(?!\S))/y, which is
// hidden.
var DirectiveToken = TokenType{
	Name: "DIRECTIVE", First: "\t %", Hidden: true,
	Opener: func(s string, at int) bool { return directiveOpener(s, at) >= 0 },
	Match: func(s string, at int) int {
		q := directiveOpener(s, at)
		if q < 0 {
			return -1
		}
		for k := q; k < len(s); k++ {
			if !strings.HasPrefix(s[k:], "}%%") {
				continue
			}
			if end := Closed(s, k+3); end >= 0 {
				return end
			}
		}
		return -1
	},
}

// CommentToken is Pele's SINGLE_LINE_COMMENT, /[\t ]*%%[^\n\r]*/y, which is hidden.
var CommentToken = TokenType{Name: "SINGLE_LINE_COMMENT", First: "\t %", Hidden: true, Match: func(s string, at int) int {
	i := blanks(s, at)
	if !strings.HasPrefix(s[i:], "%%") {
		return -1
	}
	for i += 2; i < len(s) && s[i] != '\n' && s[i] != '\r'; i++ {
	}
	return i
}}

// IntToken is the INT that packet and git diagrams define alike, /0|[1-9][0-9]*(?!\.)/y: a
// lone 0, or digits not followed by a dot, which give back the last digit rather than fail
// when they are. Pele writes it in each diagram's table.
var IntToken = TokenType{Name: "INT", Match: func(s string, at int) int {
	switch {
	case s[at] == '0':
		return at + 1
	case s[at] < '1' || s[at] > '9':
		return -1
	}
	end := at + 1
	for end < len(s) && chars.IsDigit(int(s[end])) {
		end++
	}
	if end < len(s) && s[end] == '.' {
		if end-1 > at {
			return end - 1
		}
		return -1
	}
	return end
}}

// Keyword is Pele's keyword: the word, ended only by white space, a comment or the end of
// the text, (?:(?=%%)|(?!\S)).
func Keyword(word string) TokenType {
	return TokenType{Name: word, First: word[:1], Match: func(s string, at int) int {
		if !strings.HasPrefix(s[at:], word) {
			return -1
		}
		end := at + len(word)
		if strings.HasPrefix(s[end:], "%%") || end == len(s) || spaceAt(s, end) {
			return end
		}
		return -1
	}}
}

// single is Pele's single, text.trim().replace(/[\t ]{2,}/gm, ' '): the text trimmed, with
// each run of two or more tabs and spaces made one space.
func single(text string) string {
	return squeeze(chars.Trim(text), func(b byte) bool { return b == ' ' || b == '\t' }, " ")
}

// TitleValue is the title of a TITLE token, as Pele's titleValue reads it with
// /title([\t ][^\n\r]*|)/: what follows the word, which the token holds to one line.
func TitleValue(text string) string {
	i := strings.Index(text, "title")
	return single(text[i+len("title"):])
}

// AccTitleValue is the title of an ACC_TITLE token, as Pele's accTitleValue reads it with
// /accTitle[\t ]*:([^\n\r]*)/.
func AccTitleValue(text string) string {
	i := strings.Index(text, "accTitle")
	i = blanks(text, i+len("accTitle"))
	return single(text[i+1:])
}

// AccDescrValue is the description of an ACC_DESCR token, as Pele's accDescrValue reads it
// with /accDescr(?:[\t ]*:([^\n\r]*)|\s*{([^}]*)})/: a line made single, or a block with its
// lines' leading and trailing white space taken off, runs of tabs and spaces made one space
// and runs of line breaks one line feed.
func AccDescrValue(text string) string {
	i := strings.Index(text, "accDescr") + len("accDescr")
	if j := blanks(text, i); j < len(text) && text[j] == ':' {
		return single(text[j+1:])
	}
	open := strings.IndexByte(text[i:], '{') + i
	body := text[open+1 : len(text)-1]
	body = dropLeading(body)
	body = dropTrailing(body)
	body = squeeze(body, func(b byte) bool { return b == ' ' || b == '\t' }, " ")
	return squeeze(body, func(b byte) bool { return b == '\n' || b == '\r' }, "\n")
}

// lineStart reports whether ^ matches at byte i of s in a multiline regular expression: at
// its start, or after a line terminator.
func lineStart(s string, i int) bool { return i == 0 || chars.IsLineEnd(chars.UnitBefore(s, i)) }

// dropLeading is s.replace(/^\s*/gm, ""): the white space at the start of each line, with
// the line breaks in it, taken off.
func dropLeading(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if lineStart(s, i) {
			if j := chars.SpaceRun(s, i); j > i {
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// dropTrailing is s.replace(/\s+$/gm, ""): a run of white space is taken off up to the last
// line terminator in it after its first character, or whole at the end of the text.
func dropTrailing(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		j := chars.SpaceRun(s, i)
		if j == i {
			b.WriteByte(s[i])
			i++
			continue
		}
		if j == len(s) {
			break
		}
		last := -1
		_, size := chars.UnitAt(s, i)
		for k := i + size; k < j; {
			c, size := chars.UnitAt(s, k)
			if chars.IsLineEnd(c) {
				last = k
			}
			k += size
		}
		if last >= 0 {
			i = last
		}
		b.WriteString(s[i:j])
		i = j
	}
	return b.String()
}

// squeeze replaces each run of two or more bytes in a set with with.
func squeeze(s string, in func(byte) bool, with string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		j := i
		for j < len(s) && in(s[j]) {
			j++
		}
		switch {
		case j-i >= 2:
			b.WriteString(with)
			i = j
		case j > i:
			b.WriteString(s[i:j])
			i = j
		default:
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// StringValue is the text of a STRING token, as Pele's stringValue reads it: the quotes
// taken off, and each backslash with the character after it made that character, or the
// control character \b, \f, \n, \r, \t, \v or \0 names.
func StringValue(text string) string {
	if !strings.Contains(text, `\`) {
		return text[1 : len(text)-1]
	}
	var b strings.Builder
	for i := 1; i < len(text)-1; i++ {
		c := text[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		switch next := text[i]; next {
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'v':
			b.WriteByte('\v')
		case '0':
			b.WriteByte(0)
		default:
			// The first byte of the character escaped; the loop writes the rest.
			b.WriteByte(next)
		}
	}
	return b.String()
}

// A Reader reads tokens for a hand-written parser, as Pele's Reader does, with Chevrotain's
// messages on failure. Its failures panic with a *diagram.Error, which the parser recovers.
type Reader struct {
	I       int
	Tokens  *Tokens
	Types   []TokenType
	Diagram diagram.Type
}

// Kind is the kind of the token being read.
func (r *Reader) Kind() int { return r.Tokens.Kinds[r.I] }

// Text is the text of token i.
func (r *Reader) Text(i int) string { return r.Tokens.Src[r.Tokens.Starts[i]:r.Tokens.Ends[i]] }

// Take reads the token and gives its text.
func (r *Reader) Take() string {
	r.I++
	return r.Text(r.I - 1)
}

// Accept reads the token if it is of a kind, and reports whether it was.
func (r *Reader) Accept(kind int) bool {
	if r.Tokens.Kinds[r.I] != kind {
		return false
	}
	r.I++
	return true
}

// Expect reads the token, which must be of a kind, and gives its text.
func (r *Reader) Expect(kind int) string {
	if r.Tokens.Kinds[r.I] != kind {
		r.Fail("token of type '" + r.Types[kind].Name + "'")
	}
	return r.Take()
}

// Fail panics with a syntax error at the token: what was expected, and what was found.
func (r *Reader) Fail(expected string) {
	found := "end of input"
	if r.Kind() != -1 {
		found = "`" + r.Text(r.I) + "`"
	}
	panic(diagram.SyntaxError(r.Diagram, r.Tokens.Src, r.Tokens.Starts[r.I], "Expecting "+expected+" but found "+found+".", false))
}
