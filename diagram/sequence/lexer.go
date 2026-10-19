package sequence

import (
	"unicode/utf16"

	"github.com/firede/limu/internal/chars"
)

// A token: its type, its text, and the byte of the source it starts at.
type token struct {
	typ   int
	text  string
	start int
}

// The states of the lexer, Pele's S.
const (
	sInitial = iota
	sID
	sAlias
	sLine
	sConfig
	sAccTitle
	sAccDescr
	sAccDescrMultiline
)

// The character classes of Mermaid's lexer rules, Pele's C.
const (
	cActorStart = 1
	cActorRest  = 2
	cID         = 4
	cIDWide     = 8
	cLine       = 16
	cMenu       = 32
)

// classOf holds the classes of each ASCII character, as Pele's CLASS. Other characters are
// tested by chars.IsSpace.
var classOf = func() (out [128]uint8) {
	in := func(c byte, set string) bool {
		for i := 0; i < len(set); i++ {
			if set[i] == c {
				return true
			}
		}
		return false
	}
	for c := 0; c < 128; c++ {
		ch := byte(c)
		ws := c == 32 || c >= 9 && c <= 13
		var bits uint8
		if !in(ch, "/\\+()<->:\n,;") {
			bits |= cActorStart
		}
		if !in(ch, "+<->:\n,;") {
			bits |= cActorRest
		}
		if !in(ch, "<>:\n,;@") {
			bits |= cIDWide
			if !ws {
				bits |= cID
			}
		}
		if !in(ch, "#\n;") {
			bits |= cLine
		}
		if !in(ch, "/\\+()<>:\n,;-") && !ws {
			bits |= cMenu
		}
		out[c] = bits
	}
	return
}()

// lexer reads the source as JavaScript does, by UTF-16 code units: u holds them, and at the
// byte of the source each starts at, so that tokens are cut from the Go string. No token ends
// between the two surrogates of a character.
type lexer struct {
	src    string
	u      []uint16
	at     []int
	n      int
	toks   []token
	stack  []int
	state  int
	p      int
	nextNl int
}

// newLexer makes a lexer at the start of src, splitting it into the UTF-16 code units Pele's
// tokenize reads with charCodeAt. Pele has no such function: the state it sets up is the local
// variables of tokenize.
func newLexer(src string) *lexer {
	l := &lexer{src: src, nextNl: -1}
	// A byte that is not UTF-8 reads as U+FFFD, as range gives it.
	for i, r := range src {
		if r > 0xffff {
			hi, lo := utf16.EncodeRune(r)
			l.u = append(l.u, uint16(hi), uint16(lo))
			l.at = append(l.at, i, i)
		} else {
			l.u = append(l.u, uint16(r))
			l.at = append(l.at, i)
		}
	}
	l.at = append(l.at, len(src))
	l.n = len(l.u)
	return l
}

// code is charCodeAt: the code unit at q, or -1 past the end, for NaN. It stands for
// src.charCodeAt in Pele's tokenize.
func (l *lexer) code(q int) int {
	if q < 0 || q >= l.n {
		return -1
	}
	return int(l.u[q])
}

// slice is src.slice(s, e), in code units. It stands for src.slice in Pele's tokenize.
func (l *lexer) slice(s, e int) string { return l.src[l.at[s]:l.at[e]] }

// emitText records a token of a type with the text given and moves past it to e, as the
// emitText of Pele's tokenize does.
func (l *lexer) emitText(typ, s, e int, text string) {
	l.toks = append(l.toks, token{typ, text, l.at[s]})
	l.p = e
}

// emit records a token of a type with the text from s to e and moves past it, as the emit of
// Pele's tokenize does.
func (l *lexer) emit(typ, s, e int) { l.emitText(typ, s, e, l.slice(s, e)) }

// push enters a state, keeping the current one to return to, as the push of Pele's tokenize
// does.
func (l *lexer) push(s int) {
	l.stack = append(l.stack, l.state)
	l.state = s
}

// pop returns to the state before the last push, and keeps the state when there is none, as
// the pop of Pele's tokenize does.
func (l *lexer) pop() {
	if n := len(l.stack); n > 0 {
		l.state = l.stack[n-1]
		l.stack = l.stack[:n-1]
	}
}

// indexOf is src.indexOf(String.fromCharCode(c), q), or -1. It stands for src.indexOf in
// Pele's tokenize.
func (l *lexer) indexOf(c uint16, q int) int {
	for ; q < l.n; q++ {
		if l.u[q] == c {
			return q
		}
	}
	return -1
}

// eol is the position of the first newline from q, or the end of the source, as the eol of
// Pele's tokenize gives it. The newline found is kept, so that a line is searched once however
// many times its end is asked for.
func (l *lexer) eol(q int) int {
	if l.nextNl < q {
		l.nextNl = l.indexOf('\n', q)
		if l.nextNl == -1 {
			l.nextNl = l.n
		}
	}
	return l.nextNl
}

// has reports whether the code unit at q is in a character class, as the has of Pele's
// tokenize does: a character past ASCII is in every class but those of names, which take any
// character but a space.
func (l *lexer) has(q int, bit uint8) bool {
	c := l.code(q)
	if c < 0 {
		// NaN past the end, which is no ASCII character and no space.
		return true
	}
	if c < 128 {
		return classOf[c]&bit != 0
	}
	if bit == cID || bit == cMenu {
		return !chars.IsSpace(c)
	}
	return true
}

// ci matches a lowercase literal without regard to the case of the source. It is the ci of
// Pele's tokenize.
func (l *lexer) ci(q int, word string) bool {
	for k := 0; k < len(word); k++ {
		w := int(word[k])
		c := l.code(q + k)
		if w >= 97 && w <= 122 {
			if c < 0 || c|32 != w {
				return false
			}
		} else if c != w {
			return false
		}
	}
	return true
}

// word reports whether a keyword, in any case, is at the current position and no word
// character follows it, as the word of Pele's tokenize does.
func (l *lexer) word(lit string) bool {
	return l.ci(l.p, lit) && !chars.IsWord(l.code(l.p+len(lit)))
}

// skipWs is the position of the first character from q that is not a space, as the skipWs of
// Pele's tokenize gives it.
func (l *lexer) skipWs(q int) int {
	for q < l.n && chars.IsSpace(l.code(q)) {
		q++
	}
	return q
}

// skipSameLine moves past the spaces at the current position that are not newlines, reporting
// whether there were any, as the skipSameLine of Pele's tokenize does.
func (l *lexer) skipSameLine() bool {
	q := l.p
	for q < l.n {
		c := l.code(q)
		if c == 10 || !chars.IsSpace(c) {
			break
		}
		q++
	}
	if q == l.p {
		return false
	}
	l.p = q
	return true
}

// lineRun is the end of the run of characters from q up to a newline, a semicolon or a hash,
// as the lineRun of Pele's tokenize gives it.
func (l *lexer) lineRun(q int) int {
	for q < l.n && l.has(q, cLine) {
		q++
	}
	return q
}

// begin records a token of a length at the current position and enters a state, as the begin
// of Pele's tokenize does.
func (l *lexer) begin(typ, length, next int) {
	l.emit(typ, l.p, l.p+length)
	l.push(next)
}

// keyword records a keyword's token when the keyword is at the current position, entering a
// state after it when next is not -1, as the keyword of Pele's tokenize does.
func (l *lexer) keyword(lit string, typ, next int) bool {
	if !l.word(lit) {
		return false
	}
	l.emit(typ, l.p, l.p+len(lit))
	if next >= 0 {
		l.push(next)
	}
	return true
}

// menu matches an actor menu keyword, which counts only when spaces and the start of a name
// follow. It is the menu of Pele's tokenize.
func (l *lexer) menu(lit string, typ int) bool {
	if !l.ci(l.p, lit) {
		return false
	}
	q := l.p + len(lit)
	c := l.code(q)
	if c != 32 && c != 9 {
		return false
	}
	for c == 32 || c == 9 {
		q++
		c = l.code(q)
	}
	if q >= l.n || !l.has(q, cMenu) {
		return false
	}
	l.emit(typ, l.p, l.p+len(lit))
	return true
}

// arrowAhead reports whether an arrow starts at q, where it ends a name, as the arrowAhead of
// Pele's tokenize does.
func (l *lexer) arrowAhead(q int) bool {
	c := l.code(q)
	d := l.code(q + 1)
	if c == 45 {
		if d == 120 || d == 88 || d == 41 || d == 92 || d == 47 || d == 45 {
			return true
		}
		e := l.code(q + 2)
		return d == 124 && (e == 92 || e == 47)
	}
	if c == 47 || c == 92 {
		return (d == 124 || d == c) && l.code(q+2) == 45
	}
	return c == 40 && d == 41
}

// actor records the name of an actor at the current position, which runs up to an arrow or a
// character no name holds, as the actor of Pele's tokenize does.
func (l *lexer) actor() {
	q := l.p + 1
	for q < l.n && l.has(q, cActorStart) {
		q++
	}
	for q < l.n && !l.arrowAhead(q) {
		r := q
		for l.code(r) == 45 {
			r++
		}
		if r >= l.n || !l.has(r, cActorRest) {
			break
		}
		r++
		for r < l.n && l.has(r, cActorRest) {
			r++
		}
		q = r
	}
	l.emitText(tACTOR, l.p, q, chars.Trim(l.slice(l.p, q)))
}

// number records a number at the current position, which a space or a newline must follow, as
// the number of Pele's tokenize does, and reports whether there was one.
func (l *lexer) number() bool {
	p := l.p
	q := p
	for chars.IsDigit(l.code(q)) {
		q++
	}
	whole := q > p
	if l.code(q) == 46 && chars.IsDigit(l.code(q+1)) {
		r := q + 2
		if chars.IsDigit(l.code(q + 2)) {
			r = q + 3
		}
		if after := l.code(r); after == 32 || after == 10 {
			l.emit(tNUM, p, r)
			return true
		}
	}
	if after := l.code(q); whole && (after == 32 || after == 10) {
		l.emit(tNUM, p, q)
		return true
	}
	return false
}

// dash records the arrow that starts with the dash at the current position, or the dash alone,
// as the dash of Pele's tokenize does.
func (l *lexer) dash() {
	p := l.p
	d := l.code(p + 1)
	e := l.code(p + 2)
	switch {
	case d == 62:
		if e == 62 {
			l.emit(tSOLID_ARROW, p, p+3)
		} else {
			l.emit(tSOLID_OPEN_ARROW, p, p+2)
		}
	case d == 120 || d == 88:
		l.emit(tSOLID_CROSS, p, p+2)
	case d == 41:
		l.emit(tSOLID_POINT, p, p+2)
	case d == 124 && e == 92:
		l.emit(tSOLID_ARROW_TOP, p, p+3)
	case d == 124 && e == 47:
		l.emit(tSOLID_ARROW_BOTTOM, p, p+3)
	case d == 92 && e == 92:
		l.emit(tSTICK_ARROW_TOP, p, p+3)
	case d == 47 && e == 47:
		l.emit(tSTICK_ARROW_BOTTOM, p, p+3)
	case d == 45:
		f := l.code(p + 3)
		switch {
		case e == 62:
			if f == 62 {
				l.emit(tDOTTED_ARROW, p, p+4)
			} else {
				l.emit(tDOTTED_OPEN_ARROW, p, p+3)
			}
		case e == 120 || e == 88:
			l.emit(tDOTTED_CROSS, p, p+3)
		case e == 41:
			l.emit(tDOTTED_POINT, p, p+3)
		case e == 124 && f == 92:
			l.emit(tSOLID_ARROW_TOP_DOTTED, p, p+4)
		case e == 124 && f == 47:
			l.emit(tSOLID_ARROW_BOTTOM_DOTTED, p, p+4)
		case e == 92 && f == 92:
			l.emit(tSTICK_ARROW_TOP_DOTTED, p, p+4)
		case e == 47 && f == 47:
			l.emit(tSTICK_ARROW_BOTTOM_DOTTED, p, p+4)
		default:
			l.emit(tMINUS, p, p+1)
		}
	default:
		l.emit(tMINUS, p, p+1)
	}
}

// reverse records a reversed half arrow, which starts with the slash or backslash c, as the
// reverse of Pele's tokenize does, and reports whether there was one.
func (l *lexer) reverse(c int) bool {
	p := l.p
	d := l.code(p + 1)
	if d != 124 && d != c || l.code(p+2) != 45 {
		return false
	}
	dotted := l.code(p+3) == 45
	top := c == 47
	var typ int
	switch {
	case d == 124 && dotted && top:
		typ = tSOLID_ARROW_TOP_REVERSE_DOTTED
	case d == 124 && dotted:
		typ = tSOLID_ARROW_BOTTOM_REVERSE_DOTTED
	case d == 124 && top:
		typ = tSOLID_ARROW_TOP_REVERSE
	case d == 124:
		typ = tSOLID_ARROW_BOTTOM_REVERSE
	case dotted && top:
		typ = tSTICK_ARROW_TOP_REVERSE_DOTTED
	case dotted:
		typ = tSTICK_ARROW_BOTTOM_REVERSE_DOTTED
	case top:
		typ = tSTICK_ARROW_TOP_REVERSE
	default:
		typ = tSTICK_ARROW_BOTTOM_REVERSE
	}
	if dotted {
		l.emit(typ, p, p+4)
	} else {
		l.emit(typ, p, p+3)
	}
	return true
}

// title records a title statement, with or without the legacy colon, up to the end of its
// line, as the title of Pele's tokenize does, and reports whether there was one.
func (l *lexer) title() bool {
	if !l.ci(l.p, "title") {
		return false
	}
	q := l.p + 5
	legacy := l.code(q) == 58
	if legacy {
		q++
	}
	if !chars.IsSpace(l.code(q)) || q+1 >= l.n || !l.has(q+1, cLine) {
		return false
	}
	typ := tTitle
	if legacy {
		typ = tLegacyTitle
	}
	l.emit(typ, l.p, l.lineRun(q+1))
	return true
}

// acc reads the start of an accTitle or accDescr statement and enters the state that reads its
// value, as the acc of Pele's tokenize does, and reports whether there was one.
func (l *lexer) acc() bool {
	descr := l.ci(l.p, "accdescr")
	if !descr && !l.ci(l.p, "acctitle") {
		return false
	}
	q := l.skipWs(l.p + 8)
	c := l.code(q)
	if c == 58 {
		if descr {
			l.emit(tAccDescr, l.p, l.skipWs(q+1))
			l.push(sAccDescr)
		} else {
			l.emit(tAccTitle, l.p, l.skipWs(q+1))
			l.push(sAccTitle)
		}
		return true
	}
	if descr && c == 123 {
		l.p = l.skipWs(q + 1)
		l.push(sAccDescrMultiline)
		return true
	}
	return false
}

// keywords records the keyword at the current position, choosing among those that start with
// the letter c, as the keywords of Pele's tokenize does, and reports whether there was one.
func (l *lexer) keywords(c int) bool {
	switch c | 32 {
	case 97:
		return l.keyword("actor", tParticipantActor, sID) || l.keyword("alt", tAlt, sLine) ||
			l.keyword("and", tAnd, sLine) || l.keyword("activate", tActivate, sID) || l.acc() ||
			l.keyword("autonumber", tAutonumber, -1)
	case 98:
		return l.keyword("box", tBox, sLine) || l.keyword("break", tBreak, sLine)
	case 99:
		return l.keyword("create", tCreate, -1) || l.keyword("critical", tCritical, sLine)
	case 100:
		return l.keyword("destroy", tDestroy, sID) || l.menu("details", tDetails) ||
			l.keyword("deactivate", tDeactivate, sID)
	case 101:
		return l.keyword("else", tElse, sLine) || l.keyword("end", tEnd, -1)
	case 108:
		return l.keyword("loop", tLoop, sLine) || l.keyword("left of", tLeftOf, -1) ||
			l.menu("links", tLinks) || l.menu("link", tLink)
	case 110:
		return l.keyword("note", tNote, -1)
	case 111:
		return l.keyword("opt", tOpt, sLine) || l.keyword("option", tOption, sLine) ||
			l.keyword("over", tOver, -1) || l.keyword("off", tOff, -1)
	case 112:
		return l.keyword("participant", tParticipant, sID) || l.keyword("par", tPar, sLine) ||
			l.keyword("par_over", tParOver, sLine) || l.menu("properties", tProperties)
	case 114:
		return l.keyword("rect", tRect, sLine) || l.keyword("right of", tRightOf, -1)
	case 115:
		return l.keyword("sequencediagram", tSD, -1)
	case 116:
		return l.title()
	}
	return false
}

// startsWith reports whether the ASCII text lit is at q. It stands for src.startsWith in the
// initial of Pele's tokenize.
func (l *lexer) startsWith(lit string, q int) bool {
	for k := 0; k < len(lit); k++ {
		if l.code(q+k) != int(lit[k]) {
			return false
		}
	}
	return true
}

// initial reads a token in the initial state, or skips spaces and comments, as the initial of
// Pele's tokenize does.
func (l *lexer) initial() {
	p := l.p
	c := l.code(p)
	if c == 10 {
		q := p + 1
		for l.code(q) == 10 {
			q++
		}
		l.emit(tNEWLINE, p, q)
		return
	}
	if chars.IsSpace(c) {
		l.p = l.skipWs(p + 1)
		return
	}
	if c == 35 || c == 37 && l.code(p+1) != 123 {
		l.p = l.eol(p)
		return
	}
	if c != 125 && l.code(p+1) == 37 && l.code(p+2) == 37 {
		l.p = l.eol(p)
		return
	}
	if (chars.IsDigit(c) || c == 46) && l.number() {
		return
	}
	if c < 128 && c|32 >= 97 && c|32 <= 122 && l.keywords(c) {
		return
	}
	switch {
	case c == 44:
		l.emit(tCOMMA, p, p+1)
	case c == 59:
		l.emit(tNEWLINE, p, p+1)
	case l.has(p, cActorStart):
		l.actor()
	case c == 45:
		l.dash()
	case c == 60 && l.startsWith("<<->>", p):
		l.emit(tBIDIRECTIONAL_SOLID_ARROW, p, p+5)
	case c == 60 && l.startsWith("<<-->>", p):
		l.emit(tBIDIRECTIONAL_DOTTED_ARROW, p, p+6)
	case (c == 47 || c == 92) && l.reverse(c):
	case c == 58:
		l.emit(tTXT, p, l.lineRun(p+1))
	case c == 43:
		l.emit(tPLUS, p, p+1)
	case c == 40 && l.code(p+1) == 41:
		l.emit(tCENTRAL, p, p+2)
	default:
		l.emit(tINVALID, p, p+1)
	}
}

// id reads a participant name, up to an alias, a config object, or the end of the statement.
// It is the id of Pele's tokenize.
func (l *lexer) id() bool {
	if l.skipSameLine() {
		return true
	}
	p := l.p
	c := l.code(p)
	if c == 35 {
		l.p = l.eol(p)
		return true
	}
	if c == 64 && l.code(p+1) == 123 {
		l.begin(tCONFIG_START, 2, sConfig)
		return true
	}
	a := p
	for a < l.n && l.has(a, cID) {
		a++
	}
	if a > p {
		w := l.skipWs(a)
		if l.code(w) == 64 && l.code(w+1) == 123 {
			l.emit(tACTOR, p, a)
			return true
		}
		if w > a && l.ci(w, "as") && chars.IsSpace(l.code(w+2)) {
			l.emit(tACTOR, p, a)
			l.push(sAlias)
			return true
		}
	}
	b := a
	for b < l.n && l.has(b, cIDWide) {
		b++
	}
	if b > p {
		end := l.code(b)
		// Before any other delimiter the name must be followed by a comment, so it ends at
		// the last one.
		cut := b
		if b < l.n && end != 10 && end != 59 {
			cut--
			for cut > p && l.code(cut) != 35 {
				cut--
			}
		}
		if cut > p {
			l.emitText(tACTOR, p, cut, chars.Trim(l.slice(p, cut)))
			l.pop()
			return true
		}
	}
	if l.code(b) == 60 {
		l.emit(tINVALID, p, l.eol(p))
		l.pop()
		return true
	}
	if c != 10 {
		e := l.eol(p)
		l.emitText(tINVALID, p, e, chars.Trim(l.slice(p, e)))
		l.pop()
		return true
	}
	return false
}

// step reads a token, or skips what is not one, in the current state, as the step of Pele's
// tokenize does, and reports false where no rule matched.
func (l *lexer) step() bool {
	switch l.state {
	case sInitial:
		l.initial()
		return true
	case sID:
		return l.id()
	case sAlias:
		if l.skipSameLine() {
			return true
		}
		if l.code(l.p) == 35 {
			l.p = l.eol(l.p)
			return true
		}
		l.pop()
		l.pop()
		if l.word("as") {
			l.begin(tAS, 2, sLine)
		} else {
			l.emit(tNEWLINE, l.p, l.p)
		}
		return true
	case sLine:
		if l.skipSameLine() {
			return true
		}
		if l.code(l.p) == 35 {
			l.p = l.eol(l.p)
			return true
		}
		l.emit(tRestOfLine, l.p, l.lineRun(l.p))
		l.pop()
		return true
	case sConfig:
		if l.code(l.p) != 125 {
			q := l.indexOf('}', l.p)
			if q == -1 {
				q = l.n
			}
			l.emit(tCONFIG_CONTENT, l.p, q)
			return true
		}
		w := l.skipWs(l.p + 1)
		alias := w > l.p+1 && l.ci(w, "as") && chars.IsSpace(l.code(w+2))
		l.emit(tCONFIG_END, l.p, l.p+1)
		l.pop()
		if alias {
			l.push(sAlias)
		} else {
			l.pop()
		}
		return true
	case sAccTitle, sAccDescr:
		typ := tAccDescrValue
		if l.state == sAccTitle {
			typ = tAccTitleValue
		}
		l.emit(typ, l.p, l.eol(l.p))
		l.pop()
		return true
	default:
		if l.code(l.p) == 125 {
			l.p++
			l.pop()
		} else {
			q := l.indexOf('}', l.p)
			if q == -1 {
				q = l.n
			}
			l.emit(tAccDescrMultilineValue, l.p, q)
		}
		return true
	}
}

// tokenize reads the tokens of a sequence diagram, as Pele's tokenize does. A token of type
// tERROR is where no rule matched, and ends the list.
func tokenize(src string) []token {
	l := newLexer(src)
	for l.p < l.n {
		if !l.step() {
			l.toks = append(l.toks, token{tERROR, "", l.at[l.p]})
			return l.toks
		}
	}
	// Mermaid's lexer tries its rules once more on the empty remainder before it reports the
	// end.
	n := l.n
	switch l.state {
	case sInitial, sAlias:
		l.emit(tNEWLINE, n, n)
	case sLine:
		l.emit(tRestOfLine, n, n)
	case sAccTitle:
		l.emit(tAccTitleValue, n, n)
	case sAccDescr:
		l.emit(tAccDescrValue, n, n)
	case sAccDescrMultiline:
		l.emit(tAccDescrMultilineValue, n, n)
	}
	l.toks = append(l.toks, token{tEND, "", len(src)})
	return l.toks
}
