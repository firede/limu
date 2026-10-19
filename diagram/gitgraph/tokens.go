package gitgraph

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// The token types of Pele's GIT_TOKENS, in its order, which is Mermaid's: where two match at
// the same place, the first one listed wins.
const (
	tCherryPick = iota
	tGitGraphColon
	tHighlight
	tGitGraph
	tCheckout
	tReverse
	tParent
	tCommit
	tNormal
	tBranch
	tOrder
	tSwitch
	tType
	tMerge
	tMsg
	tTag
	tID
	tLR
	tTB
	tBT
	tColon
	tAccDescr
	tAccTitle
	tTitle
	tInt
	tString
	tNewline
	tWhitespace
	tYAML
	tDirective
	tComment
	tReference
	nTypes

	// tEnd is the kind of the token after the last, at the end of the text.
	tEnd = -1
)

// tokenNames are the names of the token types, as Pele's messages give them.
var tokenNames = [nTypes]string{
	"cherry-pick", "gitGraph:", "HIGHLIGHT", "gitGraph", "checkout", "REVERSE", "parent:", "commit",
	"NORMAL", "branch", "order:", "switch", "type:", "merge", "msg:", "tag:", "id:", "LR", "TB", "BT",
	":", "ACC_DESCR", "ACC_TITLE", "TITLE", "INT", "STRING", "NEWLINE", "WHITESPACE", "YAML",
	"DIRECTIVE", "SINGLE_LINE_COMMENT", "REFERENCE",
}

// literals are the token types whose pattern is a string, by type.
var literals = map[int]string{
	tCherryPick: "cherry-pick", tGitGraphColon: "gitGraph:", tHighlight: "HIGHLIGHT",
	tCheckout: "checkout", tReverse: "REVERSE", tParent: "parent:", tCommit: "commit",
	tNormal: "NORMAL", tBranch: "branch", tOrder: "order:", tSwitch: "switch", tType: "type:",
	tMerge: "merge", tMsg: "msg:", tTag: "tag:", tID: "id:", tLR: "LR", tTB: "TB", tBT: "BT",
	tColon: ":",
}

// longer reports whether a token type has REFERENCE as its longer alternative, Chevrotain's
// LONGER_ALT: a keyword that starts a longer name is part of that name, so `commits` is a
// branch.
func longer(k int) bool {
	switch k {
	case tCherryPick, tHighlight, tGitGraph, tCheckout, tReverse, tCommit, tNormal, tBranch,
		tSwitch, tMerge, tLR, tTB, tBT:
		return true
	}
	return false
}

// hidden reports whether tokens of a type are dropped, as white space and comments are.
func hidden(k int) bool {
	return k == tWhitespace || k == tYAML || k == tDirective || k == tComment
}

// tokens is what the lexer makes of a text, Pele's Tokens: the kinds of the tokens, and the
// bytes each starts and ends at. The last is of kind tEnd, at the end of the text.
type tokens struct {
	kinds  []int
	starts []int
	ends   []int
	src    string
}

// tokenize splits src into tokens, as Pele's tokenize does with GIT_TOKENS: the types are
// tried in order, the first that matches wins, and a type with a longer alternative gives way
// to it where it matches more. Text no type matches is a lexical error.
func tokenize(src string) (*tokens, error) {
	t := &tokens{src: src}
	l := lexer{src: src}
	p := 0
scan:
	for p < len(src) {
		for k := 0; k < nTypes; k++ {
			end := l.match(k, p)
			if end <= p {
				continue
			}
			kind := k
			if longer(k) {
				if alt := l.match(tReference, p); alt > end {
					kind, end = tReference, alt
				}
			}
			if !hidden(kind) {
				t.kinds = append(t.kinds, kind)
				t.starts = append(t.starts, p)
				t.ends = append(t.ends, end)
			}
			p = end
			continue scan
		}
		return nil, diagram.SyntaxError("gitGraph", src, p, "", true)
	}
	t.kinds = append(t.kinds, tEnd)
	t.starts = append(t.starts, len(src))
	t.ends = append(t.ends, len(src))
	return t, nil
}

// lexer matches the token types at places of a text. The regular expressions of Pele's
// patterns are written out by hand, with the backtracking they would do.
type lexer struct {
	src string
	// dead marks a type whose opener matched where its whole pattern did not: no closer is
	// left in the text, so the type is not tried again, as in Pele's tokenize.
	dead [nTypes]bool
}

// match is the end of a token of type k that starts at byte p, or -1.
func (l *lexer) match(k, p int) int {
	if l.dead[k] {
		return -1
	}
	s := l.src
	if lit, ok := literals[k]; ok {
		if strings.HasPrefix(s[p:], lit) {
			return p + len(lit)
		}
		return -1
	}
	switch k {
	case tGitGraph:
		return matchKeyword(s, p, "gitGraph")
	case tAccDescr:
		end, opened := matchAccDescr(s, p)
		if end < 0 && opened {
			l.dead[k] = true
		}
		return end
	case tAccTitle:
		return matchAccTitle(s, p)
	case tTitle:
		return matchTitle(s, p)
	case tInt:
		return matchInt(s, p)
	case tString:
		return matchString(s, p)
	case tNewline:
		return newlineAt(s, p)
	case tWhitespace:
		if e := blanks(s, p); e > p {
			return e
		}
		return -1
	case tYAML:
		end, opened := matchYAML(s, p)
		if end < 0 && opened {
			l.dead[k] = true
		}
		return end
	case tDirective:
		end, opened := matchDirective(s, p)
		if end < 0 && opened {
			l.dead[k] = true
		}
		return end
	case tComment:
		i := blanks(s, p)
		if !strings.HasPrefix(s[i:], "%%") {
			return -1
		}
		return lineEnd(s, i+2)
	case tReference:
		return matchReference(s, p)
	}
	return -1
}

// blanks is the end of the run of tabs and spaces at byte i: /[\t ]*/.
func blanks(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return i
}

// lineEnd is the end of the run of characters at byte i that are not \n or \r: /[^\n\r]*/.
func lineEnd(s string, i int) int {
	for i < len(s) && s[i] != '\n' && s[i] != '\r' {
		i++
	}
	return i
}

// newlineAt is the end of /\r?\n/ at byte i, or -1.
func newlineAt(s string, i int) int {
	if strings.HasPrefix(s[i:], "\r\n") {
		return i + 2
	}
	if i < len(s) && s[i] == '\n' {
		return i + 1
	}
	return -1
}

// beforePercents is the end of /[^\n\r]*?(?=%%)|[^\n\r]*/ at byte i: up to the first %% on
// the line, or to its end where it has none.
func beforePercents(s string, i int) int {
	end := lineEnd(s, i)
	if at := strings.Index(s[i:end], "%%"); at >= 0 {
		return i + at
	}
	return end
}

// boundary is the end of /(?:\r?\n|(?!\S))/ at byte i: a line break, or nothing before white
// space or the end of the text; -1 where neither is.
func boundary(s string, i int) int {
	if e := newlineAt(s, i); e >= 0 {
		return e
	}
	if i >= len(s) || chars.IsSpace(chars.Unit(s, i)) {
		return i
	}
	return -1
}

// matchKeyword matches Pele's keyword(word): /word(?:(?=%%)|(?!\S))/, a word that ends only
// at white space, a comment or the end of the text.
func matchKeyword(s string, p int, word string) int {
	if !strings.HasPrefix(s[p:], word) {
		return -1
	}
	q := p + len(word)
	if strings.HasPrefix(s[q:], "%%") || q >= len(s) || chars.IsSpace(chars.Unit(s, q)) {
		return q
	}
	return -1
}

// matchAccDescr matches /[\t ]*accDescr(?:[\t ]*:([^\n\r]*?(?=%%)|[^\n\r]*)|\s*{([^}]*)})/,
// and reports whether its opener, /[\t ]*accDescr\s*{/, matched.
func matchAccDescr(s string, p int) (end int, opened bool) {
	i := blanks(s, p)
	if !strings.HasPrefix(s[i:], "accDescr") {
		return -1, false
	}
	a := i + len("accDescr")
	if j := blanks(s, a); j < len(s) && s[j] == ':' {
		return beforePercents(s, j+1), false
	}
	j := chars.SpaceRun(s, a)
	if j >= len(s) || s[j] != '{' {
		return -1, false
	}
	if close := strings.IndexByte(s[j+1:], '}'); close >= 0 {
		return j + 1 + close + 1, true
	}
	return -1, true
}

// matchAccTitle matches /[\t ]*accTitle[\t ]*:(?:[^\n\r]*?(?=%%)|[^\n\r]*)/.
func matchAccTitle(s string, p int) int {
	i := blanks(s, p)
	if !strings.HasPrefix(s[i:], "accTitle") {
		return -1
	}
	j := blanks(s, i+len("accTitle"))
	if j >= len(s) || s[j] != ':' {
		return -1
	}
	return beforePercents(s, j+1)
}

// matchTitle matches /[\t ]*title(?:[\t ][^\n\r]*?(?=%%)|[\t ][^\n\r]*|)/: the word title
// alone matches too, so `title1` starts with a title.
func matchTitle(s string, p int) int {
	i := blanks(s, p)
	if !strings.HasPrefix(s[i:], "title") {
		return -1
	}
	a := i + len("title")
	if a < len(s) && (s[a] == ' ' || s[a] == '\t') {
		return beforePercents(s, a+1)
	}
	return a
}

// matchInt matches /0|[1-9][0-9]*(?!\.)/: a run of digits that a dot does not follow gives
// up its last digit to one that is not a dot, as the regular expression backtracks.
func matchInt(s string, p int) int {
	if p >= len(s) {
		return -1
	}
	if s[p] == '0' {
		return p + 1
	}
	if s[p] < '1' || s[p] > '9' {
		return -1
	}
	e := p + 1
	for e < len(s) && chars.IsDigit(int(s[e])) {
		e++
	}
	if e >= len(s) || s[e] != '.' {
		return e
	}
	if e-p >= 2 {
		return e - 1
	}
	return -1
}

// matchString matches /"([^"\\]|\\.)*"|'([^'\\]|\\.)*'/, in which a backslash escapes any
// character but one that ends a line.
func matchString(s string, p int) int {
	if p >= len(s) || s[p] != '"' && s[p] != '\'' {
		return -1
	}
	q := s[p]
	for i := p + 1; i < len(s); {
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
}

// matchYAML matches /---[\t ]*\r?\n(?:[\S\s]*?\r?\n)?---(?:\r?\n|(?!\S))/: front matter, up
// to the first fence that closes it. It reports whether its opener, /---[\t ]*\r?\n/,
// matched.
func matchYAML(s string, p int) (end int, opened bool) {
	if !strings.HasPrefix(s[p:], "---") {
		return -1, false
	}
	a := newlineAt(s, blanks(s, p+3))
	if a < 0 {
		return -1, false
	}
	closes := func(at int) int {
		if !strings.HasPrefix(s[at:], "---") {
			return -1
		}
		return boundary(s, at+3)
	}
	for j := a; j < len(s); j++ {
		n := newlineAt(s, j)
		if n < 0 {
			continue
		}
		if e := closes(n); e >= 0 {
			return e, true
		}
	}
	if e := closes(a); e >= 0 {
		return e, true
	}
	return -1, true
}

// matchDirective matches /[\t ]*%%{[\S\s]*?}%%(?:\r?\n|(?!\S))/, and reports whether its
// opener, /[\t ]*%%{/, matched.
func matchDirective(s string, p int) (end int, opened bool) {
	i := blanks(s, p)
	if !strings.HasPrefix(s[i:], "%%{") {
		return -1, false
	}
	for j := i + 3; ; {
		at := strings.Index(s[j:], "}%%")
		if at < 0 {
			return -1, true
		}
		if e := boundary(s, j+at+3); e >= 0 {
			return e, true
		}
		j += at + 1
	}
}

// matchReference matches /\w([-\./\w]*[-\w])?/: a name, which may hold dots and slashes but
// does not end in one.
func matchReference(s string, p int) int {
	if p >= len(s) || !chars.IsWord(int(s[p])) {
		return -1
	}
	e := p + 1
	for e < len(s) && (chars.IsWord(int(s[e])) || s[e] == '-' || s[e] == '.' || s[e] == '/') {
		e++
	}
	for e > p+1 && (s[e-1] == '.' || s[e-1] == '/') {
		e--
	}
	return e
}

// The values of the tokens of titles and descriptions, as Pele's titleValue, accTitleValue
// and accDescrValue give them.

// single is text.trim().replace(/[\t ]{2,}/gm, ' ').
func single(text string) string {
	return squeezeBlanks(chars.Trim(text))
}

// squeezeBlanks is text.replace(/[\t ]{2,}/gm, ' ').
func squeezeBlanks(text string) string {
	var b strings.Builder
	for i := 0; i < len(text); {
		if e := blanks(text, i); e-i >= 2 {
			b.WriteByte(' ')
			i = e
			continue
		}
		b.WriteByte(text[i])
		i++
	}
	return b.String()
}

// titleValue is the title a TITLE token gives: what follows the word, as
// /title([\t ][^\n\r]*|)/ finds it.
func titleValue(text string) string {
	i := strings.Index(text, "title")
	return single(text[i+len("title"):])
}

// accTitleValue is the title an ACC_TITLE token gives: what follows its colon, as
// /accTitle[\t ]*:([^\n\r]*)/ finds it.
func accTitleValue(text string) string {
	i := strings.Index(text, "accTitle")
	j := blanks(text, i+len("accTitle"))
	return single(text[j+1:])
}

// accDescrValue is the description an ACC_DESCR token gives: the line after its colon, made
// single, or what is between its braces with the white space about each line taken off, runs
// of blanks made one, and runs of line breaks made one.
func accDescrValue(text string) string {
	i := strings.Index(text, "accDescr")
	a := i + len("accDescr")
	if j := blanks(text, a); j < len(text) && text[j] == ':' {
		return single(text[j+1:])
	}
	j := chars.SpaceRun(text, a)
	close := strings.IndexByte(text[j+1:], '}')
	body := text[j+1 : j+1+close]
	return squeezeBreaks(squeezeBlanks(trimLineEnds(trimLineStarts(body))))
}

// isLineStart reports whether a multiline ^ matches at byte i of s: at its start, or after a
// line terminator.
func isLineStart(s string, i int) bool {
	c := chars.UnitBefore(s, i)
	return i == 0 || chars.IsLineEnd(c)
}

// trimLineStarts is text.replace(/^\s*/gm, ""). Where a line starts, the white space after
// it goes, line breaks with it, and the search goes on from where that ends; a match that
// takes nothing steps over one character.
func trimLineStarts(s string) string {
	var b strings.Builder
	for i := 0; i <= len(s); {
		if isLineStart(s, i) {
			e := chars.SpaceRun(s, i)
			if e > i {
				i = e
				continue
			}
		}
		if i == len(s) {
			break
		}
		_, size := chars.UnitAt(s, i)
		b.WriteString(s[i : i+size])
		i += size
	}
	return b.String()
}

// trimLineEnds is text.replace(/\s+$/gm, ""): a run of white space that a line terminator or
// the end of the text follows goes. From where it starts, the regular expression takes the
// whole run and backs off to where $ matches; where it matches nowhere in the run, it matches
// from no later start in the run either.
func trimLineEnds(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		e := chars.SpaceRun(s, i)
		if e == i {
			b.WriteByte(s[i])
			i++
			continue
		}
		k := e
		for k > i && k < len(s) && !chars.IsLineEnd(chars.Unit(s, k)) {
			k = prevChar(s, k)
		}
		if k > i {
			i = k
			continue
		}
		b.WriteString(s[i:e])
		i = e
	}
	return b.String()
}

// prevChar is the byte the character before byte i starts at.
func prevChar(s string, i int) int {
	i--
	for i > 0 && s[i]&0xc0 == 0x80 {
		i--
	}
	return i
}

// squeezeBreaks is text.replace(/[\n\r]{2,}/gm, '\n').
func squeezeBreaks(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		e := i
		for e < len(s) && (s[e] == '\n' || s[e] == '\r') {
			e++
		}
		if e-i >= 2 {
			b.WriteByte('\n')
			i = e
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// stringValue is the text of a STRING token without its quotes, its escapes read as
// Pele's stringValue reads them.
func stringValue(text string) string {
	if !strings.Contains(text, `\`) {
		return text[1 : len(text)-1]
	}
	var b strings.Builder
	body := text[1 : len(text)-1]
	for i := 0; i < len(body); {
		c := body[i]
		if c != '\\' {
			b.WriteByte(c)
			i++
			continue
		}
		_, size := chars.UnitAt(body, i+1)
		next := body[i+1 : i+1+size]
		switch next {
		case "b":
			next = "\b"
		case "f":
			next = "\f"
		case "n":
			next = "\n"
		case "r":
			next = "\r"
		case "t":
			next = "\t"
		case "v":
			next = "\v"
		case "0":
			next = "\x00"
		}
		b.WriteString(next)
		i += 1 + size
	}
	return b.String()
}
