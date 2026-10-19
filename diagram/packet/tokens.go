package packet

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// The tokenizer of the diagram types Mermaid defines with Langium grammars, as Pele's
// diagrams/common/tokens.ts has it: token types are tried in order and the first that matches
// wins, as in Chevrotain, but for the longer alternatives a type names. Packet diagrams are
// the first of those types limu reads, so it is kept here; the regular expressions of Pele's
// table are written out by hand below, with the backtracking they do.

// A tokenType is Pele's TokenType. A type matches its literal, or, without one, by match,
// which gives the end of the match at a byte of the text, or -1.
type tokenType struct {
	name    string
	literal string
	match   func(src string, at int) int
	// opener reports whether the start of a construct that needs a closer is at a byte. When
	// it is but the whole type does not match, no closer is in the rest of the text, and the
	// type is not tried again.
	opener func(src string, at int) bool
	// first is every ASCII character a match can start with, when anyFirst is not set.
	first    string
	anyFirst bool
	hidden   bool
	// longer are Chevrotain's LONGER_ALT: when this type matches, these are tried at the same
	// place, and the first that matches more text becomes the token. By index in the table.
	longer []int
}

// tokens are Pele's Tokens: the kind of each token, by index in the table, and the bytes it
// starts and ends at, with a last of kind -1 at the end of the text.
type tokens struct {
	kinds, starts, ends []int
	src                 string
}

// table is the types of a table that can start with each ASCII character, as Pele's prepare
// gives them.
type table struct {
	starting [128][]int
	every    []int
}

func prepare(types []tokenType) *table {
	t := &table{}
	for k := range types {
		t.every = append(t.every, k)
	}
	for c := range 128 {
		for k, typ := range types {
			if typ.literal != "" && typ.literal[0] == byte(c) ||
				typ.literal == "" && (typ.anyFirst || strings.IndexByte(typ.first, byte(c)) >= 0) {
				t.starting[c] = append(t.starting[c], k)
			}
		}
	}
	return t
}

// tokenize splits src into tokens, as Pele's tokenize does. Text no type matches is a lexical
// error.
func tokenize(src string, types []tokenType, tab *table, typ string) (*tokens, error) {
	out := &tokens{src: src}
	dead := make([]bool, len(types))
	match := func(k, p int) int {
		if dead[k] {
			return -1
		}
		t := &types[k]
		if t.literal != "" {
			if strings.HasPrefix(src[p:], t.literal) {
				return p + len(t.literal)
			}
			return -1
		}
		end := t.match(src, p)
		if end < 0 && t.opener != nil && t.opener(src, p) {
			dead[k] = true
		}
		return end
	}
	n := len(src)
	p := 0
scan:
	for p < n {
		candidates := tab.every
		if c := src[p]; c < 128 {
			candidates = tab.starting[c]
		}
		for _, k := range candidates {
			end := match(k, p)
			if end <= p {
				continue
			}
			kind := k
			for _, alt := range types[k].longer {
				if altEnd := match(alt, p); altEnd > end {
					kind, end = alt, altEnd
					break
				}
			}
			if !types[kind].hidden {
				out.kinds = append(out.kinds, kind)
				out.starts = append(out.starts, p)
				out.ends = append(out.ends, end)
			}
			p = end
			continue scan
		}
		return nil, diagram.SyntaxError(typ, src, p, "", true)
	}
	out.kinds = append(out.kinds, -1)
	out.starts = append(out.starts, n)
	out.ends = append(out.ends, n)
	return out, nil
}

// blanks is the end of a run of tabs and spaces from i, [\t ]*.
func blanks(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

// spaceAt reports whether JavaScript's \s matches at byte i of s.
func spaceAt(s string, i int) bool { return chars.IsSpace(chars.Unit(s, i)) }

// lineRest matches [^\n\r]*?(?=%%)|[^\n\r]* from i: the text up to the first %% on the line,
// or to the end of the line.
func lineRest(s string, i int) int {
	for k := i; k < len(s); k++ {
		if s[k] == '\n' || s[k] == '\r' {
			return k
		}
		if strings.HasPrefix(s[k:], "%%") {
			return k
		}
	}
	return len(s)
}

// closed matches (?:\r?\n|(?!\S)) at i: a line break, or white space or the end of the text
// after it, which it does not take.
func closed(s string, i int) int {
	switch {
	case strings.HasPrefix(s[i:], "\r\n"):
		return i + 2
	case strings.HasPrefix(s[i:], "\n"):
		return i + 1
	case i == len(s) || spaceAt(s, i):
		return i
	}
	return -1
}

// lineBreak matches \r?\n at i.
func lineBreak(s string, i int) int {
	switch {
	case strings.HasPrefix(s[i:], "\r\n"):
		return i + 2
	case strings.HasPrefix(s[i:], "\n"):
		return i + 1
	}
	return -1
}

// keyword is Pele's keyword: the word, ended only by white space, a comment or the end of
// the text, (?:(?=%%)|(?!\S)).
func keyword(word string) tokenType {
	return tokenType{name: word, first: word[:1], match: func(s string, at int) int {
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

// accDescrToken is Pele's ACC_DESCR,
// /[\t ]*accDescr(?:[\t ]*:([^\n\r]*?(?=%%)|[^\n\r]*)|\s*{([^}]*)})/y.
var accDescrToken = tokenType{name: "ACC_DESCR", first: "\t a", match: func(s string, at int) int {
	i := blanks(s, at)
	if !strings.HasPrefix(s[i:], "accDescr") {
		return -1
	}
	i += len("accDescr")
	if j := blanks(s, i); j < len(s) && s[j] == ':' {
		return lineRest(s, j+1)
	}
	for i < len(s) && spaceAt(s, i) {
		_, size := chars.UnitAt(s, i)
		i += size
	}
	if i == len(s) || s[i] != '{' {
		return -1
	}
	end := strings.IndexByte(s[i+1:], '}')
	if end < 0 {
		return -1
	}
	return i + 1 + end + 1
}}

// accTitleToken is Pele's ACC_TITLE, /[\t ]*accTitle[\t ]*:(?:[^\n\r]*?(?=%%)|[^\n\r]*)/y.
var accTitleToken = tokenType{name: "ACC_TITLE", first: "\t a", match: func(s string, at int) int {
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

// titleToken is Pele's TITLE, /[\t ]*title(?:[\t ][^\n\r]*?(?=%%)|[\t ][^\n\r]*|)/y.
var titleToken = tokenType{name: "TITLE", first: "\t t", match: func(s string, at int) int {
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

// stringToken is Pele's STRING, /"([^"\\]|\\.)*"|'([^'\\]|\\.)*'/y. Each character has one
// way to be taken, so the first quote that ends it is its end; a backslash before a line
// terminator, which . does not take, ends no string.
var stringToken = tokenType{name: "STRING", first: `"'`, match: func(s string, at int) int {
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

var newlineToken = tokenType{name: "NEWLINE", first: "\r\n", match: lineBreak}

var whitespaceToken = tokenType{name: "WHITESPACE", first: "\t ", hidden: true, match: func(s string, at int) int {
	if end := blanks(s, at); end > at {
		return end
	}
	return -1
}}

// yamlOpener is /---[\t ]*\r?\n/y, the end of the opener, or -1.
func yamlOpener(s string, at int) int {
	if !strings.HasPrefix(s[at:], "---") {
		return -1
	}
	return lineBreak(s, blanks(s, at+3))
}

// yamlToken is Pele's YAML, /---[\t ]*\r?\n(?:[\S\s]*?\r?\n)?---(?:\r?\n|(?!\S))/y. The group
// is tried first, so the closing dashes are the first after a line break, past any that
// follow the opener at once.
var yamlToken = tokenType{
	name: "YAML", first: "-", hidden: true,
	opener: func(s string, at int) bool { return yamlOpener(s, at) >= 0 },
	match: func(s string, at int) int {
		q := yamlOpener(s, at)
		if q < 0 {
			return -1
		}
		for k := q; k < len(s); k++ {
			j := lineBreak(s, k)
			if j < 0 || !strings.HasPrefix(s[j:], "---") {
				continue
			}
			if end := closed(s, j+3); end >= 0 {
				return end
			}
		}
		if strings.HasPrefix(s[q:], "---") {
			return closed(s, q+3)
		}
		return -1
	},
}

// directiveOpener is /[\t ]*%%{/y, the end of the opener, or -1.
func directiveOpener(s string, at int) int {
	i := blanks(s, at)
	if !strings.HasPrefix(s[i:], "%%{") {
		return -1
	}
	return i + 3
}

// directiveToken is Pele's DIRECTIVE, /[\t ]*%%{[\S\s]*?}%%(?:\r?\n|(?!\S))/y.
var directiveToken = tokenType{
	name: "DIRECTIVE", first: "\t %", hidden: true,
	opener: func(s string, at int) bool { return directiveOpener(s, at) >= 0 },
	match: func(s string, at int) int {
		q := directiveOpener(s, at)
		if q < 0 {
			return -1
		}
		for k := q; k < len(s); k++ {
			if !strings.HasPrefix(s[k:], "}%%") {
				continue
			}
			if end := closed(s, k+3); end >= 0 {
				return end
			}
		}
		return -1
	},
}

// commentToken is Pele's SINGLE_LINE_COMMENT, /[\t ]*%%[^\n\r]*/y.
var commentToken = tokenType{name: "SINGLE_LINE_COMMENT", first: "\t %", hidden: true, match: func(s string, at int) int {
	i := blanks(s, at)
	if !strings.HasPrefix(s[i:], "%%") {
		return -1
	}
	for i += 2; i < len(s) && s[i] != '\n' && s[i] != '\r'; i++ {
	}
	return i
}}

// single is Pele's single: the text trimmed, with each run of tabs and spaces made one space.
func single(text string) string {
	text = chars.Trim(text)
	var b strings.Builder
	for i := 0; i < len(text); {
		j := blanks(text, i)
		if j-i >= 2 {
			b.WriteByte(' ')
			i = j
			continue
		}
		b.WriteByte(text[i])
		i++
	}
	return b.String()
}

// titleValue is the title of a TITLE token, as Pele's titleValue reads it with
// /title([\t ][^\n\r]*|)/: what follows the keyword, which the token holds to one line.
func titleValue(text string) string {
	i := strings.Index(text, "title")
	return single(text[i+len("title"):])
}

// accTitleValue is the title of an ACC_TITLE token, read with /accTitle[\t ]*:([^\n\r]*)/.
func accTitleValue(text string) string {
	i := strings.Index(text, "accTitle")
	i = blanks(text, i+len("accTitle"))
	return single(text[i+1:])
}

// accDescrValue is the description of an ACC_DESCR token, read with
// /accDescr(?:[\t ]*:([^\n\r]*)|\s*{([^}]*)})/: a line made single, or a block with its lines'
// leading and trailing white space taken off, runs of tabs and spaces made one space and runs
// of line breaks one line feed.
func accDescrValue(text string) string {
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

// lineStart reports whether ^ matches at byte i of s in a multiline regular expression.
func lineStart(s string, i int) bool { return i == 0 || chars.IsLineEnd(chars.UnitBefore(s, i)) }

// spaceEnd is the end of the run of white space, \s*, from byte i.
func spaceEnd(s string, i int) int {
	for i < len(s) {
		c, size := chars.UnitAt(s, i)
		if !chars.IsSpace(c) {
			break
		}
		i += size
	}
	return i
}

// dropLeading is s.replace(/^\s*/gm, ”): the white space at the start of each line, with
// the line breaks in it, taken off.
func dropLeading(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if lineStart(s, i) {
			if j := spaceEnd(s, i); j > i {
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// dropTrailing is s.replace(/\s+$/gm, ”): a run of white space is taken off up to the last
// line terminator in it after its first character, or whole at the end of the text.
func dropTrailing(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		j := spaceEnd(s, i)
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

// stringValue is the text of a STRING token, as Pele's stringValue reads it: the quotes taken
// off, and each backslash with the character after it made that character, or the control
// character \b, \f, \n, \r, \t, \v or \0 names.
func stringValue(text string) string {
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
			b.WriteByte(next)
		}
	}
	return b.String()
}

// reader reads tokens for a hand-written parser, as Pele's Reader does, with Chevrotain's
// messages on failure. Its failures panic with a *diagram.Error, which the parser recovers.
type reader struct {
	i     int
	toks  *tokens
	types []tokenType
	typ   string
}

func (r *reader) kind() int { return r.toks.kinds[r.i] }

func (r *reader) text(i int) string { return r.toks.src[r.toks.starts[i]:r.toks.ends[i]] }

func (r *reader) take() string {
	r.i++
	return r.text(r.i - 1)
}

func (r *reader) accept(kind int) bool {
	if r.toks.kinds[r.i] != kind {
		return false
	}
	r.i++
	return true
}

func (r *reader) expect(kind int) string {
	if r.toks.kinds[r.i] != kind {
		r.fail("token of type '" + r.types[kind].name + "'")
	}
	return r.take()
}

func (r *reader) fail(expected string) {
	found := "end of input"
	if r.kind() != -1 {
		found = "`" + r.text(r.i) + "`"
	}
	panic(diagram.SyntaxError(r.typ, r.toks.src, r.toks.starts[r.i], "Expecting "+expected+" but found "+found+".", false))
}
