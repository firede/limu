package agentflow

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The token names and kinds agree in number.
var _ [len(tokenNames)]struct{} = [t_EOF + 1]struct{}{}

// tok is a token: its kind, its text, the byte of the source it starts at, and its span as
// Jison counts it: first line, first column, last line, last column, lines from 1 and
// columns from 0, in UTF-16 code units.
type tok struct {
	typ   token
	text  string
	start int
	loc   [4]int
}

// The states of the lexer, Pele's S.
const (
	sINITIAL = iota
	sString
	sMdString
	sAccTitle
	sAccDescr
	sAccDescrMultiline
	sDir
	sText
	sEllipseText
	sTrapText
	sEdgeText
	sClick
	sCallbackname
	sCallbackargs
	sShapeData
	sShapeDataStr
)

// nodeChar holds the ASCII characters a NODE_STRING is made of, Pele's NODE_CHAR.
var nodeChar = func() (t [128]bool) {
	for _, c := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!\"#$%&'*+.`?\\_/" {
		t[c] = true
	}
	return
}()

// The regular expressions of Pele's lexer, written out. Each is sticky: it matches at q or
// not at all, and gives the end of its match or -1.

// space matches /\s*/y, the run of white space Pele's patterns begin and end with.
func space(s string, q int) int { return chars.SpaceRun(s, q) }

// keywordColon matches /word\s*<sep>\s*/y.
// It is Pele's RE_ACC_TITLE, RE_ACC_DESCR and RE_ACC_DESCR_ML.
func keywordColon(s string, q int, word string, sep byte) int {
	if !strings.HasPrefix(s[q:], word) {
		return -1
	}
	j := space(s, q+len(word))
	if j >= len(s) || s[j] != sep {
		return -1
	}
	return space(s, j+1)
}

// keywordSpace matches /word\s+/y.
// It is Pele's RE_CALL and RE_CLICK.
func keywordSpace(s string, q int, word string) int {
	if !strings.HasPrefix(s[q:], word) {
		return -1
	}
	j := space(s, q+len(word))
	if j == q+len(word) {
		return -1
	}
	return j
}

// nodir matches /(?:\r?\n)*\s*\n/y: white space up to the last line feed in it.
// It is Pele's RE_NODIR.
func nodir(s string, q int) int {
	end := space(s, q)
	if i := strings.LastIndexByte(s[q:end], '\n'); i >= 0 {
		return q + i + 1
	}
	return -1
}

// wordAt reports whether the code unit at i is a word character, which \b tests in Pele's
// RE_DIR and RE_DIR_V; past the end of s it is not.
func wordAt(s string, i int) bool { return chars.IsWord(chars.Unit(s, i)) }

// dir matches /\s*(?:LR|RL|TB|BT|TD|BR)\b/y.
// It is Pele's RE_DIR.
func dir(s string, q int) int {
	j := space(s, q)
	if j+2 > len(s) {
		return -1
	}
	switch s[j : j+2] {
	case "LR", "RL", "TB", "BT", "TD", "BR":
		if !wordAt(s, j+2) {
			return j + 2
		}
	}
	return -1
}

// dirSym matches /\s*[<>^]/y.
// It is Pele's RE_DIR_SYM.
func dirSym(s string, q int) int {
	j := space(s, q)
	if j < len(s) && (s[j] == '<' || s[j] == '>' || s[j] == '^') {
		return j + 1
	}
	return -1
}

// dirV matches /\s*v\b/y.
// It is Pele's RE_DIR_V.
func dirV(s string, q int) int {
	j := space(s, q)
	if j < len(s) && s[j] == 'v' && !wordAt(s, j+1) {
		return j + 1
	}
	return -1
}

// lineEndFrom is the first line terminator at or after q, or the end.
// It is the search with RE_LINE_END that Pele's initial makes before it tries RE_DIRECTION.
func lineEndFrom(s string, q int) int {
	for q < len(s) {
		c, size := chars.UnitAt(s, q)
		if chars.IsLineEnd(c) {
			return q
		}
		q += size
	}
	return len(s)
}

// directionNames are the names Pele's RE_DIRECTION are written for, in their order.
var directionNames = [5]string{"TB", "BT", "RL", "LR", "TD"}

// direction matches /.*direction\s+<name>[^\n]*/y: the last `direction` on the line that is
// followed by white space and the name, and the rest of the line its match ends on.
// It is one of Pele's RE_DIRECTION.
func direction(s string, q int, name string) int {
	end := lineEndFrom(s, q)
	for x := end - 1; x >= q; x-- {
		if s[x] != 'd' || !strings.HasPrefix(s[x:], "direction") {
			continue
		}
		j := space(s, x+9)
		if j == x+9 || !strings.HasPrefix(s[j:], name) {
			continue
		}
		if nl := strings.IndexByte(s[j+2:], '\n'); nl >= 0 {
			return j + 2 + nl
		}
		return len(s)
	}
	return -1
}

// emptyCall matches /\(\s*\)/y.
// It is Pele's RE_EMPTY_CALL.
func emptyCall(s string, q int) int {
	if q >= len(s) || s[q] != '(' {
		return -1
	}
	j := space(s, q+1)
	if j < len(s) && s[j] == ')' {
		return j + 1
	}
	return -1
}

// breaks is s.replace(/\n\s*/g, '<br/>').
// It is Pele's replace with RE_BR, which a metadata string's line breaks go through.
func breaks(s string) string {
	if !strings.Contains(s, "\n") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\n' {
			b.WriteString("<br/>")
			i = space(s, i+1)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// index is s.indexOf(sub, from).
// It is Pele's src.indexOf, whose start may be past the end.
func index(s, sub string, from int) int {
	if from > len(s) {
		return -1
	}
	i := strings.Index(s[from:], sub)
	if i == -1 {
		return -1
	}
	return from + i
}

// separatorAt reports whether U+2028 or U+2029 starts at byte i of s.
// It is the test of Pele's advance for where a column stops.
func separatorAt(s string, i int) bool {
	return i+2 < len(s) && s[i] == 0xe2 && s[i+1] == 0x80 && (s[i+2] == 0xa8 || s[i+2] == 0xa9)
}

// lexer holds the variables of Pele's tokenize, which its closures share; here they are the
// lexer's fields, and the closures its methods.
type lexer struct {
	src        string
	n          int
	toks       []tok
	stack      []int
	state      int
	p          int
	line, col  int
	firstGraph bool
	dirNext    int
	atNext     int
	lineEnd    int
	dirFailed  int
	runEnd     int
	runAt      int
	wsFrom     int
	wsTo       int
	blankFrom  int
	blankTo    int
	noLink     int
	noArrow    int
}

// tokenize reads src into tokens, as Pele's tokenize does. The last token is t_END, or t_ERROR
// where no token starts.
func tokenize(src string) []tok {
	l := &lexer{
		src:        src,
		n:          len(src),
		line:       1,
		firstGraph: true,
		dirNext:    strings.Index(src, "direction"),
		atNext:     strings.Index(src, "@"),
		lineEnd:    -1,
		runAt:      -1,
		wsFrom:     -1,
		wsTo:       -1,
		blankFrom:  -1,
		blankTo:    -1,
		noLink:     -1,
		noArrow:    -1,
	}
	return l.run()
}

// at is the code unit at i, -1 past the end.
// It is Pele's src.charCodeAt(i), but -1 where that gives NaN. Pele's own at, which runs a
// pattern at q, is written here as a call of the pattern's function.
func (l *lexer) at(i int) int { return chars.Unit(l.src, i) }

// advance moves over the source up to e. Jison counts \r\n, \r and \n as one line break each,
// and after a break its column stops at the first U+2028 or U+2029.
// It is Pele's advance.
func (l *lexer) advance(e int) {
	brk := -1
	for k := l.p; k < e; k++ {
		switch l.src[k] {
		case '\n':
			l.line++
			brk = k
		case '\r':
			if k+1 < e && l.src[k+1] == '\n' {
				k++
			}
			l.line++
			brk = k
		}
	}
	if brk < 0 {
		l.col += chars.Len16(l.src[l.p:e])
	} else {
		k := brk + 1
		for k < e && !separatorAt(l.src, k) {
			k++
		}
		l.col = chars.Len16(l.src[brk+1 : k])
	}
	l.p = e
}

// emit adds a token of the source up to e.
// It is Pele's emit.
func (l *lexer) emit(typ token, e int) { l.emitText(typ, l.src[l.p:e], e) }

// emitText adds a token of the source up to e, with a text of its own.
// It is Pele's emitText.
func (l *lexer) emitText(typ token, text string, e int) {
	t := tok{typ: typ, text: text, start: l.p}
	t.loc[0], t.loc[1] = l.line, l.col
	l.advance(e)
	t.loc[2], t.loc[3] = l.line, l.col
	l.toks = append(l.toks, t)
}

// emitFlat adds a token that cannot hold a line break.
// It is Pele's emitFlat.
func (l *lexer) emitFlat(typ token, e int) {
	t := tok{typ: typ, text: l.src[l.p:e], start: l.p}
	t.loc[0], t.loc[1] = l.line, l.col
	l.col += chars.Len16(l.src[l.p:e])
	l.p = e
	t.loc[2], t.loc[3] = l.line, l.col
	l.toks = append(l.toks, t)
}

// emitChars adds a run that Jison matches one character at a time, where \r\n counts as two
// line breaks.
// It is Pele's emitChars, which counts a character above U+FFFF as the two code units it
// reads it as.
func (l *lexer) emitChars(typ token, e int) {
	t := tok{typ: typ, text: l.src[l.p:e], start: l.p}
	t.loc[0], t.loc[1] = l.line, l.col
	for _, r := range l.src[l.p:e] {
		switch {
		case r == '\n' || r == '\r':
			l.line++
			l.col = 0
		case r > 0xffff:
			l.col += 2
		default:
			l.col++
		}
	}
	l.p = e
	t.loc[2], t.loc[3] = l.line, l.col
	l.toks = append(l.toks, t)
}

// push enters state s, keeping the state it leaves, as Pele's push does.
func (l *lexer) push(s int) {
	l.stack = append(l.stack, l.state)
	l.state = s
}

// pop goes back to the state push left, if there is one, as Pele's pop does.
func (l *lexer) pop() {
	if len(l.stack) > 0 {
		l.state = l.stack[len(l.stack)-1]
		l.stack = l.stack[:len(l.stack)-1]
	}
}

// kw reports whether the source has the keyword word at the current position, not followed
// by a word character, as Pele's kw does.
func (l *lexer) kw(word string) bool {
	return strings.HasPrefix(l.src[l.p:], word) && !chars.IsWord(l.at(l.p+len(word)))
}

// wsEnd is the end of the run of white space starting at q. The last run is kept, so asking
// again from inside it costs nothing.
// It is Pele's wsEnd.
func (l *lexer) wsEnd(q int) int {
	if q >= l.wsFrom && q <= l.wsTo {
		return l.wsTo
	}
	j := space(l.src, q)
	l.wsFrom = q
	l.wsTo = j
	return j
}

// blankEnd is the end of the run of spaces and tabs starting at q, kept as wsEnd keeps its.
// It is Pele's blankEnd.
func (l *lexer) blankEnd(q int) int {
	if q >= l.blankFrom && q <= l.blankTo {
		return l.blankTo
	}
	j := blanks(l.src, q)
	l.blankFrom = q
	l.blankTo = j
	return j
}

// blanks is the end of the run of spaces and tabs from q.
// It is Pele's blanks.
func blanks(s string, q int) int {
	for q < len(s) && (s[q] == ' ' || s[q] == '\t') {
		q++
	}
	return q
}

// arrowEnd is the end of `\s*--+[x>]\s*` starting at q, or -1.
// It is Pele's arrowEnd: a place it fails at is kept, so it fails there at once again.
func (l *lexer) arrowEnd(q int) int {
	j := l.wsEnd(q)
	if j == l.noArrow || l.at(j) != '-' || l.at(j+1) != '-' {
		return -1
	}
	k := j + 2
	for l.at(k) == '-' {
		k++
	}
	if t := l.at(k); t == 'x' || t == '>' {
		return l.wsEnd(k + 1)
	}
	l.noArrow = j
	return -1
}

// link reads a link in the initial state, a whole one or the start of one with a label in it,
// as Pele's link does: a place it fails at is kept, so it fails there at once again.
func (l *lexer) link() bool {
	j := l.wsEnd(l.p)
	if j == l.noLink || l.at(j) != '-' {
		return false
	}
	d := l.at(j + 1)
	k := j + 2
	if d == '-' {
		for l.at(k) == '-' {
			k++
		}
		if t := l.at(k); t == 'x' || t == '>' {
			l.emit(t_LINK, l.wsEnd(k+1))
		} else {
			l.push(sEdgeText)
			l.emit(t_START_LINK, l.wsEnd(j+2))
		}
		return true
	}
	if d == '.' {
		for l.at(k) == '.' {
			k++
		}
		if l.at(k) == '-' {
			l.emit(t_LINK, l.wsEnd(k+1))
			return true
		}
	}
	l.noLink = j
	return false
}

// commentEnd is the end of `%%` not followed by `{`, through the end of its line, or -1.
// It is Pele's commentEnd.
func (l *lexer) commentEnd(q int) int {
	if l.at(q) != '%' || l.at(q+1) != '%' || l.at(q+2) == '{' {
		return -1
	}
	if e := index(l.src, "\n", q+2); e != -1 {
		return e
	}
	return l.n
}

// quote enters a string or a Markdown string at a double quote, as Pele's quote does.
func (l *lexer) quote() {
	if l.at(l.p+1) == '`' {
		l.push(sMdString)
		l.advance(l.p + 2)
	} else {
		l.push(sString)
		l.advance(l.p + 1)
	}
}

// opener reads the token that opens a node's shape or a link's label at character c, and
// enters the state its text is read in, as Pele's opener does. It reports whether c opens one.
func (l *lexer) opener(c int) bool {
	p := l.p
	d := l.at(p + 1)
	switch c {
	case '(':
		switch {
		case d == '-':
			l.push(sEllipseText)
			l.emitFlat(t_ELLIPSE_START, p+2)
		case d == '[':
			l.push(sText)
			l.emitFlat(t_STADIUMSTART, p+2)
		case d == '(' && l.at(p+2) == '(':
			l.push(sText)
			l.emitFlat(t_DOUBLECIRCLESTART, p+3)
		default:
			l.push(sText)
			l.emitFlat(t_PS, p+1)
		}
		return true
	case '[':
		switch d {
		case '[':
			l.push(sText)
			l.emitFlat(t_SUBROUTINESTART, p+2)
		case '(':
			l.push(sText)
			l.emitFlat(t_CYLINDERSTART, p+2)
		case '/':
			l.push(sTrapText)
			l.emitFlat(t_TRAPSTART, p+2)
		case '\\':
			l.push(sTrapText)
			l.emitFlat(t_INVTRAPSTART, p+2)
		default:
			l.push(sText)
			l.emitFlat(t_SQS, p+1)
		}
		return true
	case '|':
		l.push(sText)
		l.emitFlat(t_PIPE, p+1)
		return true
	case '{':
		l.push(sText)
		l.emitFlat(t_DIAMOND_START, p+1)
		return true
	}
	return false
}

// nodeString reads a NODE_STRING, which stops before a dash that starts a link, as Pele's
// nodeString does. It reports whether there was one.
func (l *lexer) nodeString() bool {
	q := l.p
	for q < l.n {
		ch := l.src[q]
		if ch < 128 && nodeChar[ch] {
			q++
		} else if ch == '-' && q+1 < l.n {
			d := l.src[q+1]
			if d == '>' || d == '-' || d == '.' {
				break
			}
			q++
		} else {
			break
		}
	}
	if q == l.p {
		return false
	}
	l.emitFlat(t_NODE_STRING, q)
	return true
}

// initial reads a token in the initial state, at character c, as Pele's initial does. It
// reports whether one starts there.
func (l *lexer) initial(c int) bool {
	p := l.p
	src := l.src
	var e int
	switch c {
	case 'a':
		if strings.HasPrefix(src[p:], "acc") {
			if e = keywordColon(src, p, "accTitle", ':'); e >= 0 {
				l.push(sAccTitle)
				l.emit(t_acc_title, e)
				return true
			}
			if e = keywordColon(src, p, "accDescr", ':'); e >= 0 {
				l.push(sAccDescr)
				l.emit(t_acc_descr, e)
				return true
			}
			if e = keywordColon(src, p, "accDescr", '{'); e >= 0 {
				l.push(sAccDescrMultiline)
				l.advance(e)
				return true
			}
		}
		if l.kw("agentflow-beta") {
			if l.firstGraph {
				l.firstGraph = false
				l.push(sDir)
			}
			l.emitFlat(t_GRAPH, p+14)
			return true
		}
	case '@':
		if l.at(p+1) == '{' {
			l.push(sShapeData)
			l.emitText(t_SHAPE_DATA, "", p+2)
			return true
		}
	case 'c':
		if e = keywordSpace(src, p, "call"); e >= 0 {
			l.push(sCallbackname)
			l.advance(e)
			return true
		}
		if l.kw("classDef") {
			l.emitFlat(t_CLASSDEF, p+8)
			return true
		}
		if l.kw("class") {
			l.emitFlat(t_CLASS, p+5)
			return true
		}
		if e = keywordSpace(src, p, "click"); e >= 0 {
			l.push(sClick)
			l.advance(e)
			return true
		}
		if l.kw("connector") {
			l.emitFlat(t_connector, p+9)
			return true
		}
	case '"':
		l.quote()
		return true
	case 's':
		if l.kw("style") {
			l.emitFlat(t_STYLE, p+5)
			return true
		}
	case 'd':
		if l.kw("default") {
			l.emitFlat(t_DEFAULT, p+7)
			return true
		}
	case 'l':
		if l.kw("linkStyle") {
			l.emitFlat(t_LINKSTYLE, p+9)
			return true
		}
	case 'i':
		if l.kw("interpolate") {
			l.emitFlat(t_INTERPOLATE, p+11)
			return true
		}
	case 'h':
		if strings.HasPrefix(src[p:], "href") {
			if d, size := chars.UnitAt(src, p+4); chars.IsSpace(d) {
				l.emit(t_HREF, p+4+size)
				return true
			}
		}
	case 'f':
		if l.kw("flow") {
			l.emitFlat(t_flow, p+4)
			return true
		}
	case 'g':
		if l.kw("global") {
			l.emitFlat(t_global, blanks(src, p+6))
			return true
		}
	case 'e':
		if l.kw("end") {
			l.emitFlat(t_end, blanks(src, p+3))
			return true
		}
	case '_':
		for _, w := range []string{"_self", "_blank", "_parent", "_top"} {
			if l.kw(w) {
				l.emitFlat(t_LINK_TARGET, p+len(w))
				return true
			}
		}
	}

	if l.dirNext != -1 && p >= l.dirFailed {
		if l.dirNext < p {
			l.dirNext = index(src, "direction", p)
		}
		if l.dirNext != -1 {
			if l.lineEnd < p {
				l.lineEnd = lineEndFrom(src, p)
			}
			if l.dirNext < l.lineEnd {
				for i, name := range directionNames {
					if e = direction(src, p, name); e >= 0 {
						l.emit(t_direction_tb+token(i), e)
						return true
					}
				}
				// No direction statement starts anywhere in the rest of this line either.
				l.dirFailed = l.lineEnd
			}
		}
	}

	// A link id is a run of non-space characters up to its last `@` that is not followed by `{` or `"`.
	if l.atNext != -1 && c != '"' && !chars.IsSpace(c) {
		if p >= l.runEnd {
			if l.atNext < p {
				l.atNext = index(src, "@", p)
			}
			q := p
			for q < l.n {
				ch, size := chars.UnitAt(src, q)
				if ch == '"' || chars.IsSpace(ch) {
					break
				}
				q += size
			}
			l.runEnd = q
			l.runAt = -1
			if l.atNext != -1 && l.atNext < q {
				// From the character before the run's end, or, when the run reaches the end of
				// the source, from the one before its last: Pele's Math.min(q, n - 1) - 1. Only
				// an @ is looked for, so bytes stand for code units here.
				from := q - 1
				if q >= l.n {
					from = l.n - 2
				}
				for k := from; k >= l.atNext; k-- {
					if src[k] != '@' {
						continue
					}
					if after := l.at(k + 1); after != '{' && after != '"' {
						l.runAt = k
						break
					}
				}
			}
		}
		if l.runAt > p {
			l.emitFlat(t_LINK_ID, l.runAt+1)
			return true
		}
	}

	if c >= '0' && c <= '9' {
		q := p + 1
		for q < l.n && src[q] >= '0' && src[q] <= '9' {
			q++
		}
		l.emitFlat(t_NUM, q)
		return true
	}

	switch c {
	case '#':
		l.emitFlat(t_BRKT, p+1)
		return true
	case ':':
		if l.at(p+1) == ':' && l.at(p+2) == ':' {
			l.emitFlat(t_STYLE_SEPARATOR, p+3)
		} else {
			l.emitFlat(t_COLON, p+1)
		}
		return true
	case '&':
		l.emitFlat(t_AMP, p+1)
		return true
	case ';':
		l.emitFlat(t_SEMI, p+1)
		return true
	case ',':
		l.emitFlat(t_COMMA, p+1)
		return true
	case '*':
		l.emitFlat(t_MULT, p+1)
		return true
	case '<':
		l.emitFlat(t_TAGSTART, p+1)
		return true
	case '-':
		if l.link() || l.nodeString() {
			return true
		}
		l.emitFlat(t_MINUS, p+1)
		return true
	case '(':
		return l.opener(c)
	case '[':
		if l.at(p+1) == '|' {
			l.emitFlat(t_VERTEX_WITH_PROPS_START, p+2)
			return true
		}
		return l.opener(c)
	case '|', '{':
		return l.opener(c)
	case '>':
		l.push(sText)
		l.emitFlat(t_TAGEND, p+1)
		return true
	case '^':
		l.emitFlat(t_UP, p+1)
		return true
	case '\\':
		if l.at(p+1) == '|' {
			l.emitFlat(t_SEP, p+2)
			return true
		}
		return l.nodeString()
	case 'v':
		if !chars.IsWord(l.at(p + 1)) {
			l.emitFlat(t_DOWN, p+1)
			return true
		}
		return l.nodeString()
	case '%':
		if e = l.commentEnd(p); e >= 0 {
			l.emit(t_COMMENT, e)
			return true
		}
		return l.nodeString()
	}

	if chars.IsSpace(c) {
		if l.link() {
			return true
		}
		if c == ' ' || c == '\t' {
			if e = l.commentEnd(l.blankEnd(p)); e >= 0 {
				l.emit(t_COMMENT, e)
				return true
			}
		}
		e = p
		for {
			if l.at(e) == '\n' {
				e++
			} else if l.at(e) == '\r' && l.at(e+1) == '\n' {
				e += 2
			} else {
				break
			}
		}
		if e > p {
			l.emit(t_NEWLINE, e)
		} else {
			_, size := chars.UnitAt(src, p)
			l.emit(t_SPACE, p+size)
		}
		return true
	}

	if c < 128 {
		return l.nodeString()
	}
	if chars.IsUnicodeLetter(c) {
		_, size := chars.UnitAt(src, p)
		l.emitFlat(t_UNICODE_TEXT, p+size)
		return true
	}
	return false
}

// run reads the tokens of each state until the end of the source or an error, as the loop of
// Pele's tokenize does after its closures.
func (l *lexer) run() []tok {
	src := l.src
	for {
		if l.p >= l.n {
			typ := token(255)
			switch l.state {
			case sINITIAL:
				typ = t_EOF
			case sAccTitle:
				typ = t_acc_title_value
			case sAccDescr:
				typ = t_acc_descr_value
			case sAccDescrMultiline:
				typ = t_acc_descr_multiline_value
			case sClick:
				typ = t_CLICK
			case sCallbackname:
				typ = t_CALLBACKNAME
			case sCallbackargs:
				typ = t_CALLBACKARGS
			}
			if typ != 255 {
				l.emitFlat(typ, l.n)
			}
			break
		}

		p := l.p
		c, size := chars.UnitAt(src, p)
		ok := true

		switch l.state {
		case sINITIAL:
			ok = l.initial(c)

		case sString:
			if c == '"' {
				if l.at(p+1) == '`' {
					l.push(sMdString)
					l.advance(p + 2)
				} else {
					l.pop()
					l.advance(p + 1)
				}
			} else {
				e := index(src, "\"", p)
				if e == -1 {
					e = l.n
				}
				l.emit(t_STR, e)
			}

		case sMdString:
			if c == '`' {
				if l.at(p+1) == '"' {
					l.pop()
					l.advance(p + 2)
				} else {
					ok = false
				}
			} else if c == '"' {
				l.quote()
			} else {
				q := p + size
				for q < l.n && src[q] != '`' && src[q] != '"' {
					q++
				}
				l.emit(t_MD_STR, q)
			}

		case sAccTitle, sAccDescr:
			e := index(src, "\n", p)
			if e == -1 {
				e = l.n
			}
			typ := t_acc_title_value
			if l.state == sAccDescr {
				typ = t_acc_descr_value
			}
			l.pop()
			l.emit(typ, e)

		case sAccDescrMultiline:
			if c == '}' {
				l.pop()
				l.advance(p + 1)
			} else {
				e := index(src, "}", p)
				if e == -1 {
					e = l.n
				}
				l.emit(t_acc_descr_multiline_value, e)
			}

		case sDir:
			var e int
			if c == '"' {
				l.quote()
			} else if e = nodir(src, p); e >= 0 {
				l.pop()
				l.emit(t_NODIR, e)
			} else if e = dir(src, p); e >= 0 {
				l.pop()
				l.emit(t_DIR, e)
			} else if e = dirSym(src, p); e >= 0 {
				l.pop()
				l.emit(t_DIR, e)
			} else if e = dirV(src, p); e >= 0 {
				l.pop()
				l.emit(t_DIR, e)
			} else {
				ok = l.opener(c)
			}

		case sText:
			d := l.at(p + 1)
			switch c {
			case '"':
				l.quote()
			case '(', '[', '{':
				l.opener(c)
			case ']':
				l.pop()
				if d == ')' {
					l.emitFlat(t_STADIUMEND, p+2)
				} else if d == ']' {
					l.emitFlat(t_SUBROUTINEEND, p+2)
				} else {
					l.emitFlat(t_SQE, p+1)
				}
			case ')':
				l.pop()
				if d == ']' {
					l.emitFlat(t_CYLINDEREND, p+2)
				} else if d == ')' && l.at(p+2) == ')' {
					l.emitFlat(t_DOUBLECIRCLEEND, p+3)
				} else {
					l.emitFlat(t_PE, p+1)
				}
			case '|':
				l.pop()
				l.emitFlat(t_PIPE, p+1)
			case '}':
				l.pop()
				l.emitFlat(t_DIAMOND_STOP, p+1)
			case '%':
				// Jison drops the lone `%` from Mermaid's text pattern, so only a comment may
				// start with one.
				if e := l.commentEnd(p); e >= 0 {
					l.advance(e)
				} else {
					ok = false
				}
			default:
				q := p + size
				for q < l.n && strings.IndexByte("[]()}{|\"%", src[q]) < 0 {
					q++
				}
				l.emit(t_TEXT, q)
			}

		case sEllipseText:
			d := l.at(p + 1)
			var e int
			if c == '"' {
				l.quote()
			} else if c == '%' && func() bool { e = l.commentEnd(p); return e >= 0 }() {
				l.advance(e)
			} else if (c == '-' || c == '/' || c == ')') && d == ')' {
				l.pop()
				l.emitFlat(t_ELLIPSE_END, p+2)
			} else if c == '(' || c == '[' || c == '{' {
				l.opener(c)
			} else if c == ')' || c == ']' || c == '}' {
				ok = false
			} else {
				q := p + size
				for q < l.n {
					ch := src[q]
					if strings.IndexByte("\"()[]{}", ch) >= 0 {
						break
					}
					if (ch == '-' || ch == '/') && l.at(q+1) == ')' {
						break
					}
					if ch == '%' && l.at(q+1) == '%' && l.at(q+2) != '{' {
						break
					}
					q++
				}
				l.emitChars(t_TEXT, q)
			}

		case sTrapText:
			d := l.at(p + 1)
			if c == '"' {
				l.quote()
				break
			}
			switch c {
			case '(':
				if d == '-' || d == '[' || d == '(' && l.at(p+2) == '(' {
					l.opener(c)
				} else if d == ']' {
					l.pop()
					l.emitFlat(t_TRAPEND, p+2)
				} else {
					l.opener(c)
				}
			case '[', '{':
				l.opener(c)
			case '\\':
				if d == ']' {
					l.pop()
					l.emitFlat(t_TRAPEND, p+2)
				} else {
					l.emitFlat(t_TEXT, p+1)
				}
			case '/':
				if d == ']' {
					l.pop()
					l.emitFlat(t_INVTRAPEND, p+2)
				} else {
					l.emitFlat(t_TEXT, p+1)
				}
			case ']', ')':
				if d == ']' {
					l.pop()
					l.emitFlat(t_TRAPEND, p+2)
				} else {
					ok = false
				}
			case '}':
				ok = false
			case '%':
				if e := l.commentEnd(p); e >= 0 {
					l.advance(e)
				} else {
					ok = false
				}
			default:
				if (c == '?' || c == '=') && d == ']' {
					l.pop()
					l.emitFlat(t_TRAPEND, p+2)
					break
				}
				q := p + size
				for q < l.n && strings.IndexByte("\\[]()}{/%", src[q]) < 0 {
					q++
				}
				l.emit(t_TEXT, q)
			}

		case sEdgeText:
			if c == '"' {
				l.quote()
				break
			}
			if e := l.arrowEnd(p); e >= 0 {
				l.pop()
				l.emit(t_LINK, e)
				break
			}
			if c == '-' && l.at(p+1) == '-' {
				ok = false
				break
			}
			q := p + size
			for q < l.n {
				ch, size := chars.UnitAt(src, q)
				if ch == '"' {
					break
				}
				if ch == '-' {
					if l.at(q+1) == '-' {
						break
					}
				} else if chars.IsSpace(ch) && l.arrowEnd(q) >= 0 {
					break
				}
				q += size
			}
			l.emitChars(t_EDGE_TEXT, q)

		case sClick:
			if c == '"' {
				l.quote()
			} else if chars.IsSpace(c) {
				l.pop()
				l.advance(p + size)
			} else {
				q := p + size
				for q < l.n {
					ch, size := chars.UnitAt(src, q)
					if chars.IsSpace(ch) {
						break
					}
					q += size
				}
				l.emitFlat(t_CLICK, q)
			}

		case sCallbackname:
			if c == '(' {
				e := emptyCall(src, p)
				l.pop()
				if e >= 0 {
					l.advance(e)
				} else {
					l.push(sCallbackargs)
					l.advance(p + 1)
				}
			} else {
				e := index(src, "(", p)
				if e == -1 {
					e = l.n
				}
				l.emit(t_CALLBACKNAME, e)
			}

		case sCallbackargs:
			if c == ')' {
				l.pop()
				l.advance(p + 1)
			} else {
				e := index(src, ")", p)
				if e == -1 {
					e = l.n
				}
				l.emit(t_CALLBACKARGS, e)
			}

		case sShapeData:
			if c == '"' {
				l.push(sShapeDataStr)
				l.emitFlat(t_SHAPE_DATA, p+1)
			} else if c == '}' {
				l.pop()
				l.advance(p + 1)
			} else if c == '^' {
				ok = false
			} else {
				q := p + size
				for q < l.n && src[q] != '}' && src[q] != '^' && src[q] != '"' {
					q++
				}
				l.emit(t_SHAPE_DATA, q)
			}

		case sShapeDataStr:
			if c == '"' {
				l.pop()
				l.emitFlat(t_SHAPE_DATA, p+1)
			} else {
				e := index(src, "\"", p)
				if e == -1 {
					e = l.n
				}
				l.emitText(t_SHAPE_DATA, breaks(src[p:e]), e)
			}
		}

		if !ok {
			l.toks = append(l.toks, tok{typ: t_ERROR, start: l.p, loc: [4]int{l.line, l.col, l.line, l.col}})
			return l.toks
		}
	}

	l.toks = append(l.toks, tok{typ: t_END, start: l.n, loc: [4]int{l.line, l.col, l.line, l.col}})
	return l.toks
}
