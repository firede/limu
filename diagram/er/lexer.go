package er

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// tok is a token: its kind, its text and the byte of the source it starts at.
type tok struct {
	typ   token
	text  string
	start int
}

// The states of the lexer, Pele's S.
const (
	sINITIAL = iota
	sBlock
	sBlockBq
	sAccTitle
	sAccDescr
	sAccDescrMultiline
	sStyle
)

// The classes of an ASCII character: what it may be part of.
const (
	cText      = 1 // UNICODE_TEXT, outside a block
	cStyleText = 2 // STYLE_TEXT
	cAttrStart = 4 // the first character of an ATTRIBUTE_WORD
	cAttrRest  = 8 // the others
)

// class is the class of each ASCII character, as Pele's CLASS.
var class = func() (t [128]uint8) {
	for c := 0; c < 128; c++ {
		word := c >= 48 && c <= 57 || c >= 65 && c <= 90 || c >= 97 && c <= 122 || c == 95
		letter := word && (c < 48 || c > 57)
		if word {
			t[c] |= cText | cStyleText | cAttrRest
		}
		if letter {
			t[c] |= cAttrStart
		}
	}
	for _, c := range "-*" {
		t[c] |= cText | cStyleText | cAttrRest
	}
	t['.'] |= cText | cAttrRest
	t['*'] |= cAttrStart
	for _, c := range "[]()," {
		t[c] |= cAttrRest
	}
	return
}()

// isAttr reports whether code unit c is of the attribute class kind. Mermaid's attribute
// pattern is case-insensitive, which lets the micro sign into its À-￿ range. It is Pele's
// isAttr.
func isAttr(c, kind int) bool {
	if c < 0 {
		return false
	}
	if c < 128 {
		return int(class[c])&kind != 0
	}
	return c >= 0xc0 || c == 0xb5
}

// isClass reports whether c is of the class kind, any character past ASCII being of it. It is
// the test `c >= 128 || (CLASS[c] & kind) !== 0` that Pele's tokenize writes out in place.
func isClass(c, kind int) bool {
	return c >= 128 || c >= 0 && int(class[c])&kind != 0
}

// directionNames are the names after `direction` in Pele's RE_DIRECTION, in its order.
var directionNames = [4]string{"TB", "BT", "RL", "LR"}

// lexer is the state of Pele's tokenize.
type lexer struct {
	src   string
	n     int
	toks  []tok
	state int
	p     int
	// The caches that keep the lexer linear: where the next `direction` is, the end of the line
	// it failed on, the end of the line p is on, and for generic types the next tilde, the end
	// of its line and the last tilde before it, and the next white space.
	dirNext, dirFailed, lineEnd     int
	tildeNext, tildeLine, tildeLast int
	wsNext                          int
}

// tokenize splits an ER diagram's text into tokens, as Pele's tokenize does, with Jison's
// end of input after them. Positions are bytes; where Pele steps one code unit, this steps
// one character.
func tokenize(src string) []tok {
	l := &lexer{src: src, n: len(src), dirNext: -2, lineEnd: -1, tildeNext: -2, tildeLine: -1, tildeLast: -1, wsNext: -1}
	for l.p < l.n {
		c, size := chars.UnitAt(src, l.p)
		switch l.state {
		case sINITIAL:
			l.initial(c, size)
		case sBlock:
			l.block(c, size)
		case sBlockBq:
			if c == '`' {
				l.state = sBlock
				l.p++
			} else {
				l.emit(t_ATTRIBUTE_WORD, l.p, l.indexOr(l.p, '`', l.n))
			}
		case sAccTitle, sAccDescr:
			typ := t_acc_title_value
			if l.state == sAccDescr {
				typ = t_acc_descr_value
			}
			l.emit(typ, l.p, l.indexOr(l.p, '\n', l.n))
			l.state = sINITIAL
		case sAccDescrMultiline:
			if c == '}' {
				l.state = sINITIAL
				l.p++
			} else {
				l.emit(t_acc_descr_multiline_value, l.p, l.indexOr(l.p, '}', l.n))
			}
		case sStyle:
			switch {
			case c == '\n':
				e := l.p + 1
				for e < l.n && src[e] == '\n' {
					e++
				}
				l.state = sINITIAL
				l.emit(t_NEWLINE, l.p, e)
			case chars.IsSpace(c):
				l.p = chars.SpaceRun(src, l.p)
			case c == ':':
				l.emit(t_COLON, l.p, l.p+1)
			case c == ',':
				l.emit(t_COMMA, l.p, l.p+1)
			case c == '#':
				l.emit(t_BRKT, l.p, l.p+1)
			case c == ';':
				l.emit(t_SEMI, l.p, l.p+1)
			case isClass(c, cStyleText):
				l.emit(t_STYLE_TEXT, l.p, l.run(l.p+size, cStyleText))
			default:
				l.toks = append(l.toks, tok{t_ERROR, "", l.p})
				return l.toks
			}
		}
	}
	// At the end of input Jison's INITIAL state yields EOF, the title and description states
	// yield an empty value, and the block and style states yield nothing.
	last := -1
	switch l.state {
	case sINITIAL:
		last = int(t_EOF)
	case sAccTitle:
		last = int(t_acc_title_value)
	case sAccDescr:
		last = int(t_acc_descr_value)
	case sAccDescrMultiline:
		last = int(t_acc_descr_multiline_value)
	}
	if last != -1 {
		l.toks = append(l.toks, tok{token(last), "", l.n})
	}
	return append(l.toks, tok{t_END, "", l.n})
}

// emit appends a token of type typ for the text from s to e and moves the lexer to e, as the
// emit of Pele's tokenize does.
func (l *lexer) emit(typ token, s, e int) {
	l.toks = append(l.toks, tok{typ, l.src[s:e], s})
	l.p = e
}

// unit is the code unit at byte q, or -1 past the end: Pele's src.charCodeAt(q), whose NaN
// past the end matches nothing.
func (l *lexer) unit(q int) int { return chars.Unit(l.src, q) }

// indexOr is the first b at or after q, or or: Pele's src.indexOf(b, q), with what Pele puts
// in place of -1.
func (l *lexer) indexOr(q int, b byte, or int) int {
	if i := strings.IndexByte(l.src[q:], b); i >= 0 {
		return q + i
	}
	return or
}

// run is the end of the run of characters of class kind that starts at q, as the loops of
// Pele's tokenize that read UNICODE_TEXT and STYLE_TEXT find it.
func (l *lexer) run(q, kind int) int {
	for q < l.n {
		c, size := chars.UnitAt(l.src, q)
		if !isClass(c, kind) {
			break
		}
		q += size
	}
	return q
}

// kw matches a lower-case keyword at p, ignoring case, as the kw of Pele's tokenize does.
func (l *lexer) kw(word string) bool {
	if l.p+len(word) > l.n {
		return false
	}
	for i := 0; i < len(word); i++ {
		k, c := word[i], l.src[l.p+i]
		if c != k && (k < 'a' || k > 'z' || c|32 != k) {
			return false
		}
	}
	return true
}

// kwb matches a keyword at p that no word character follows, as the kwb of Pele's tokenize
// does.
func (l *lexer) kwb(word string) bool {
	return l.kw(word) && !chars.IsWord(l.unit(l.p+len(word)))
}

// lower is the code unit at q, lowered if it is an ASCII capital, as the lower of Pele's
// tokenize gives it.
func (l *lexer) lower(q int) int {
	c := l.unit(q)
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// nextLineEnd is the first line terminator at or after q, or the end, as the nextLineEnd of
// Pele's tokenize finds it with RE_LINE_END.
func (l *lexer) nextLineEnd(q int) int {
	for q < l.n {
		c, size := chars.UnitAt(l.src, q)
		if chars.IsLineEnd(c) {
			return q
		}
		q += size
	}
	return l.n
}

// digits is the end of the run of ASCII digits from q, as the digits of Pele's tokenize gives
// it.
func (l *lexer) digits(q int) int {
	for q < l.n && chars.IsDigit(int(l.src[q])) {
		q++
	}
	return q
}

// indexFold is the first "direction" at or after q, its case ignored: /direction/gi, Pele's
// RE_DIR_WORD, as the direction of Pele's tokenize searches with it.
func (l *lexer) indexFold(q int) int {
	for ; q+9 <= l.n; q++ {
		if l.src[q]|32 == 'd' && chars.HasPrefixFold(l.src[q:], "direction") {
			return q
		}
	}
	return -1
}

// wordFold matches /<word>\s*:\s*/iy at q, or with open for the colon; it gives the end of
// the match or -1. It reads Pele's RE_ACC_TITLE, RE_ACC_DESCR and RE_ACC_DESCR_ML, as the at
// of Pele's tokenize matches them.
func (l *lexer) wordFold(q int, word string, sep byte) int {
	if !chars.HasPrefixFold(l.src[q:], word) {
		return -1
	}
	j := chars.SpaceRun(l.src, q+len(word))
	if j >= l.n || l.src[j] != sep {
		return -1
	}
	return chars.SpaceRun(l.src, j+1)
}

// directionAt matches /.*direction\s+<name>[^\n]*/iy at q: the last `direction` on the line
// that white space and the name follow, and the rest of the line its match ends on. The
// white space may run onto further lines. It is one of Pele's RE_DIRECTION, written out with
// the backtracking it does.
func (l *lexer) directionAt(q int, name string) int {
	end := l.nextLineEnd(q)
	for x := end - 1; x >= q; x-- {
		if l.src[x]|32 != 'd' || !chars.HasPrefixFold(l.src[x:], "direction") {
			continue
		}
		j := chars.SpaceRun(l.src, x+9)
		if j == x+9 || !chars.HasPrefixFold(l.src[j:], name) {
			continue
		}
		return l.indexOr(j+2, '\n', l.n)
	}
	return -1
}

// direction lexes a direction statement, which swallows its whole line from wherever the
// lexer stands on it, as the direction of Pele's tokenize does.
func (l *lexer) direction() bool {
	if l.dirNext == -1 {
		return false
	}
	if l.dirNext < l.p {
		l.dirNext = l.indexFold(l.p)
		if l.dirNext == -1 {
			return false
		}
	}
	if l.p < l.dirFailed {
		return false
	}
	if l.lineEnd < l.p {
		l.lineEnd = l.nextLineEnd(l.p)
	}
	if l.dirNext >= l.lineEnd {
		return false
	}
	for i, name := range directionNames {
		if e := l.directionAt(l.p, name); e >= 0 {
			l.emit(t_direction_tb+token(i), l.p, e)
			return true
		}
	}
	l.dirFailed = l.lineEnd
	return false
}

// quoted lexes a double quote at p, as the quoted of Pele's tokenize does: an ENTITY_NAME
// through the next quote when the text between is not empty and holds none of % \ \r \n \v
// \b, a WORD through it when it does, and a CHAR when no quote closes it.
func (l *lexer) quoted() {
	p := l.p
	e := strings.IndexByte(l.src[p+1:], '"')
	if e == -1 {
		l.emit(t_CHAR, p, p+1)
		return
	}
	e += p + 1
	name := e > p+1
	for q := p + 1; name && q < e; q++ {
		switch l.src[q] {
		case '%', '\\', '\r', '\n', '\v', '\b':
			name = false
		}
	}
	if name {
		l.emit(t_ENTITY_NAME, p, e+1)
	} else {
		l.emit(t_WORD, p, e+1)
	}
}

// number lexes what starts with the digit c at p, as the number of Pele's tokenize does: the
// cardinalities 1+ and 0+, a DECIMAL_NUM, a 1 that is a cardinality, or a NUM.
func (l *lexer) number(c int) {
	p := l.p
	d, dsize := chars.UnitAt(l.src, p+1)
	if d == '+' && (c == '1' || c == '0') {
		if c == '1' {
			l.emit(t_ONE_OR_MORE, p, p+2)
		} else {
			l.emit(t_ZERO_OR_MORE, p, p+2)
		}
		return
	}
	q := l.digits(p + 1)
	if l.unit(q) == '.' && chars.IsDigit(l.unit(q+1)) {
		l.emit(t_DECIMAL_NUM, p, l.digits(q+2))
		return
	}
	if c == '1' {
		if chars.IsSpace(d) {
			r := chars.SpaceRun(l.src, p+1+dsize)
			if f := l.unit(r); chars.IsWord(f) || f == '"' || f == '\'' {
				l.emit(t_ONLY_ONE, p, p+1)
				return
			}
		} else if d == '-' || d == '.' {
			if f := l.unit(p + 2); f == '-' || f == '.' {
				l.emit(t_ONLY_ONE, p, p+1)
				return
			}
		}
		if !chars.IsWord(d) {
			l.emit(t_ENTITY_ONE, p, p+1)
			return
		}
	}
	l.emit(t_NUM, p, q)
}

// keyword lexes a keyword, a cardinality or a relationship mark at p, whose first code unit
// lowered is lc, and reports whether it found one, as the keyword of Pele's tokenize does.
func (l *lexer) keyword(lc int) bool {
	p := l.p
	d := l.unit(p + 1)
	switch lc {
	case 'e':
		if l.kwb("erdiagram") {
			l.emit(t_ER_DIAGRAM, p, p+9)
		} else if l.kwb("end") {
			l.emit(t_END_KW, p, chars.SpaceRun(l.src, p+3))
		} else {
			return false
		}
	case 's':
		if l.kwb("style") {
			l.state = sStyle
			l.emit(t_STYLE, p, p+5)
		} else if l.kwb("subgraph") {
			l.emit(t_SUBGRAPH, p, p+8)
		} else {
			return false
		}
	case 'c':
		if l.kwb("classdef") {
			l.state = sStyle
			l.emit(t_CLASSDEF, p, p+8)
		} else if l.kwb("class") {
			l.emit(t_CLASS, p, p+5)
		} else {
			return false
		}
	case 'o':
		switch {
		case l.kwb("one or zero"):
			l.emit(t_ZERO_OR_ONE, p, p+11)
		case l.kwb("one or more") || l.kwb("one or many"):
			l.emit(t_ONE_OR_MORE, p, p+11)
		case l.kwb("one"):
			l.emit(t_ONLY_ONE, p, p+3)
		case l.kwb("only one"):
			l.emit(t_ONLY_ONE, p, p+8)
		case d == '|':
			l.emit(t_ZERO_OR_ONE, p, p+2)
		case d == '{':
			l.emit(t_ZERO_OR_MORE, p, p+2)
		case l.kwb("optionally to"):
			l.emit(t_NON_IDENTIFYING, p, p+13)
		default:
			return false
		}
	case 'z':
		switch {
		case l.kwb("zero or one"):
			l.emit(t_ZERO_OR_ONE, p, p+11)
		case l.kwb("zero or more") || l.kwb("zero or many"):
			l.emit(t_ZERO_OR_MORE, p, p+12)
		default:
			return false
		}
	case 'm':
		switch {
		case l.kw("many(0)"):
			l.emit(t_ZERO_OR_MORE, p, p+7)
		case l.kw("many(1)"):
			l.emit(t_ONE_OR_MORE, p, p+7)
		case l.kwb("many"):
			l.emit(t_ZERO_OR_MORE, p, p+4)
		default:
			return false
		}
	case 't':
		if !l.kwb("to") {
			return false
		}
		l.emit(t_IDENTIFYING, p, p+2)
	case 'u':
		if d != '.' && d != '-' && d != '|' {
			return false
		}
		l.emit(t_MD_PARENT, p, p+1)
	case '|':
		switch {
		case l.lower(p+1) == 'o' && !chars.IsWord(l.unit(p+2)):
			l.emit(t_ZERO_OR_ONE, p, p+2)
		case d == '|':
			l.emit(t_ONLY_ONE, p, p+2)
		case d == '{':
			l.emit(t_ONE_OR_MORE, p, p+2)
		default:
			return false
		}
	case '}':
		switch {
		case l.lower(p+1) == 'o' && !chars.IsWord(l.unit(p+2)):
			l.emit(t_ZERO_OR_MORE, p, p+2)
		case d == '|':
			l.emit(t_ONE_OR_MORE, p, p+2)
		default:
			return false
		}
	case '.':
		if d != '.' && d != '-' {
			return false
		}
		l.emit(t_NON_IDENTIFYING, p, p+2)
	case '-':
		switch d {
		case '-':
			l.emit(t_IDENTIFYING, p, p+2)
		case '.':
			l.emit(t_NON_IDENTIFYING, p, p+2)
		default:
			return false
		}
	default:
		return false
	}
	return true
}

// initial lexes one token in the INITIAL state from code unit c, size bytes long, at p, as
// the initial of Pele's tokenize does.
func (l *lexer) initial(c, size int) {
	p := l.p
	if (c == 'a' || c == 'A') && l.kw("acc") {
		if e := l.wordFold(p, "accTitle", ':'); e >= 0 {
			l.state = sAccTitle
			l.emit(t_acc_title, p, e)
			return
		}
		if e := l.wordFold(p, "accDescr", ':'); e >= 0 {
			l.state = sAccDescr
			l.emit(t_acc_descr, p, e)
			return
		}
		if e := l.wordFold(p, "accDescr", '{'); e >= 0 {
			l.state = sAccDescrMultiline
			l.p = e
			return
		}
	}
	if l.direction() {
		return
	}
	switch c {
	case ' ', '\t', '\r':
		e := p + 1
		for e < l.n && (l.src[e] == ' ' || l.src[e] == '\t' || l.src[e] == '\r') {
			e++
		}
		l.p = e
		return
	case '\n':
		e := p + 1
		for e < l.n && l.src[e] == '\n' {
			e++
		}
		l.emit(t_NEWLINE, p, e)
		return
	case '"':
		l.quoted()
		return
	case '{':
		l.state = sBlock
		l.emit(t_BLOCK_START, p, p+1)
		return
	case '#':
		l.emit(t_BRKT, p, p+1)
		return
	case ',':
		l.emit(t_COMMA, p, p+1)
		return
	case ':':
		if strings.HasPrefix(l.src[p:], ":::") {
			l.emit(t_STYLE_SEPARATOR, p, p+3)
		} else {
			l.emit(t_COLON, p, p+1)
		}
		return
	case '[':
		l.emit(t_SQS, p, p+1)
		return
	case ']':
		l.emit(t_SQE, p, p+1)
		return
	}
	if chars.IsDigit(c) {
		l.number(c)
		return
	}
	lc := c
	if c >= 'A' && c <= 'Z' {
		lc += 32
	}
	if l.keyword(lc) {
		return
	}
	if isClass(c, cText) {
		l.emit(t_UNICODE_TEXT, p, l.run(p+size, cText))
		return
	}
	l.emit(t_CHAR, p, p+size)
}

// generic lexes Mermaid's generic type rule: a run of non-space characters holding a `~`,
// through the last `~` on the line, as the generic of Pele's tokenize does.
func (l *lexer) generic() bool {
	if l.tildeNext == -1 {
		return false
	}
	if l.tildeNext < l.p {
		l.tildeNext = l.indexOr(l.p, '~', -1)
		if l.tildeNext == -1 {
			return false
		}
	}
	if l.wsNext < l.p {
		l.wsNext = l.spaceNext(l.p)
	}
	if l.tildeNext >= l.wsNext {
		return false
	}
	if l.tildeLine < l.p {
		l.tildeLine = l.nextLineEnd(l.p)
		l.tildeLast = strings.LastIndexByte(l.src[:l.tildeLine], '~')
	}
	if l.tildeLast <= l.tildeNext {
		return false
	}
	l.emit(t_ATTRIBUTE_WORD, l.p, l.spaceNext(l.tildeLast+1))
	return true
}

// spaceNext is the first white space at or after q, or the end, as the loops in the generic
// of Pele's tokenize find it.
func (l *lexer) spaceNext(q int) int {
	for q < l.n {
		c, size := chars.UnitAt(l.src, q)
		if chars.IsSpace(c) {
			break
		}
		q += size
	}
	return q
}

// block lexes one token in an entity's block of attributes from code unit c, size bytes long,
// at p, as the block of Pele's tokenize does.
func (l *lexer) block(c, size int) {
	p := l.p
	if chars.IsSpace(c) {
		l.p = chars.SpaceRun(l.src, p)
		return
	}
	if lc := c | 32; (lc == 'p' || lc == 'f' || lc == 'u') && l.unit(p+1)|32 == 'k' && !chars.IsWord(l.unit(p+2)) {
		l.emit(t_ATTRIBUTE_KEY, p, p+2)
		return
	}
	if l.generic() {
		return
	}
	if isAttr(c, cAttrStart) {
		e := p + size
		for e < l.n {
			d, dsize := chars.UnitAt(l.src, e)
			if !isAttr(d, cAttrRest) {
				break
			}
			e += dsize
		}
		l.emit(t_ATTRIBUTE_WORD, p, e)
		return
	}
	switch c {
	case '`':
		l.state = sBlockBq
		l.p++
		return
	case '"':
		if e := l.indexOr(p+1, '"', -1); e != -1 {
			l.emit(t_COMMENT, p, e+1)
			return
		}
	case '}':
		l.state = sINITIAL
		l.emit(t_BLOCK_STOP, p, p+1)
		return
	}
	l.emit(t_CHAR, p, p+size)
}
