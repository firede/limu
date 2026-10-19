package flowchart

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
	sThickEdgeText
	sDottedEdgeText
	sClick
	sCallbackname
	sCallbackargs
	sShapeData
	sShapeDataStr
)

var nodeChar = func() (t [128]bool) {
	for _, c := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!\"#$%&'*+.`?\\_/" {
		t[c] = true
	}
	return
}()

// The regular expressions of Pele's lexer, written out. Each is sticky: it matches at q or
// not at all, and gives the end of its match or -1.

func space(s string, q int) int { return chars.SpaceRun(s, q) }

// keywordColon matches /word\s*<sep>\s*/y.
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
func nodir(s string, q int) int {
	end := space(s, q)
	if i := strings.LastIndexByte(s[q:end], '\n'); i >= 0 {
		return q + i + 1
	}
	return -1
}

func wordAt(s string, i int) bool { return chars.IsWord(chars.Unit(s, i)) }

// dir matches /\s*(?:LR|RL|TB|BT|TD|BR)\b/y.
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
func dirSym(s string, q int) int {
	j := space(s, q)
	if j < len(s) && (s[j] == '<' || s[j] == '>' || s[j] == '^') {
		return j + 1
	}
	return -1
}

// dirV matches /\s*v\b/y.
func dirV(s string, q int) int {
	j := space(s, q)
	if j < len(s) && s[j] == 'v' && !wordAt(s, j+1) {
		return j + 1
	}
	return -1
}

// lineEndFrom is the first line terminator at or after q, or the end.
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

var directionNames = [5]string{"TB", "BT", "RL", "LR", "TD"}

// direction matches /.*direction\s+<name>[^\n]*/y: the last `direction` on the line that is
// followed by white space and the name, and the rest of the line its match ends on.
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

func isOneOf(s string, i int, set string) bool {
	return i < len(s) && strings.IndexByte(set, s[i]) >= 0
}

// solidLink matches /\s*[xo<]?<c><c>+[<c>xo>]\s*/y for c of - or =.
func solidLink(s string, q int, c byte) int {
	j := space(s, q)
	if isOneOf(s, j, "xo<") {
		j++
	}
	k := j
	for k < len(s) && s[k] == c {
		k++
	}
	m := k - j
	switch {
	case m >= 2 && isOneOf(s, k, "xo>"):
		k++
	case m >= 3:
	default:
		return -1
	}
	return space(s, k)
}

// startLink matches /\s*[xo<]?<c><c>\s*/y.
func startLink(s string, q int, c byte) int {
	j := space(s, q)
	if isOneOf(s, j, "xo<") {
		j++
	}
	if j+1 < len(s) && s[j] == c && s[j+1] == c {
		return space(s, j+2)
	}
	return -1
}

// dottedLink matches /\s*[xo<]?-?\.+-[xo>]?\s*/y.
func dottedLink(s string, q int) int {
	j := space(s, q)
	if isOneOf(s, j, "xo<") {
		j++
	}
	if j < len(s) && s[j] == '-' {
		j++
	}
	k := j
	for k < len(s) && s[k] == '.' {
		k++
	}
	if k == j || k >= len(s) || s[k] != '-' {
		return -1
	}
	k++
	if isOneOf(s, k, "xo>") {
		k++
	}
	return space(s, k)
}

// dottedStart matches /\s*[xo<]?-\.\s*/y.
func dottedStart(s string, q int) int {
	j := space(s, q)
	if isOneOf(s, j, "xo<") {
		j++
	}
	if j+1 < len(s) && s[j] == '-' && s[j+1] == '.' {
		return space(s, j+2)
	}
	return -1
}

// invisible matches /\s*~~~+\s*/y.
func invisible(s string, q int) int {
	j := space(s, q)
	k := j
	for k < len(s) && s[k] == '~' {
		k++
	}
	if k-j < 3 {
		return -1
	}
	return space(s, k)
}

// newline matches /(?:\r?\n)+/y.
func newline(s string, q int) int {
	j := q
	for {
		if strings.HasPrefix(s[j:], "\r\n") {
			j += 2
		} else if j < len(s) && s[j] == '\n' {
			j++
		} else {
			break
		}
	}
	if j == q {
		return -1
	}
	return j
}

// blanks matches /[^\S\n\r]+/y.
func blanks(s string, q int) int {
	j := q
	for j < len(s) {
		c, size := chars.UnitAt(s, j)
		if !chars.IsSpace(c) || c == '\n' || c == '\r' {
			break
		}
		j += size
	}
	if j == q {
		return -1
	}
	return j
}

// emptyCall matches /\(\s*\)/y.
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

type lexer struct {
	src        string
	n          int
	toks       []tok
	stack      []int
	state      int
	p          int
	firstGraph bool
	dirNext    int
	atNext     int
	lineEnd    int
	dirFailed  int
	runEnd     int
	runAt      int
	wsFrom     int
	wsTo       int
}

// tokenize reads src into tokens, as Pele's tokenize does. The last token is t_END, or t_ERROR
// where no token starts.
func tokenize(src string) []tok {
	l := &lexer{
		src:        src,
		n:          len(src),
		firstGraph: true,
		dirNext:    strings.Index(src, "direction"),
		atNext:     strings.Index(src, "@"),
		lineEnd:    -1,
		runAt:      -1,
		wsFrom:     -1,
		wsTo:       -1,
	}
	return l.run()
}

// at is the code unit at i, -1 past the end.
func (l *lexer) at(i int) int { return chars.Unit(l.src, i) }

func (l *lexer) emit(typ token, s, e int) {
	l.toks = append(l.toks, tok{typ, l.src[s:e], s})
	l.p = e
}

func (l *lexer) push(s int) {
	l.stack = append(l.stack, l.state)
	l.state = s
}

func (l *lexer) pop() {
	if len(l.stack) > 0 {
		l.state = l.stack[len(l.stack)-1]
		l.stack = l.stack[:len(l.stack)-1]
	}
}

func (l *lexer) kw(word string) bool {
	return strings.HasPrefix(l.src[l.p:], word) && !chars.IsWord(l.at(l.p+len(word)))
}

// spaceEnd is the end of the white space from q, kept for the next call from within it.
func (l *lexer) spaceEnd(q int) int {
	if q >= l.wsFrom && q <= l.wsTo {
		return l.wsTo
	}
	if chars.IsSpace(l.at(q)) {
		j := space(l.src, q)
		l.wsFrom = q
		l.wsTo = j
		return j
	}
	return q
}

// linkChar is the first character after any run of white space starting at q, skipping one
// optional x, o or <.
func (l *lexer) linkChar(q int) int {
	j := l.spaceEnd(q)
	d := l.at(j)
	if d == 'x' || d == 'o' || d == '<' {
		return l.at(j + 1)
	}
	return d
}

func (l *lexer) invisAhead(q int) bool { return l.at(l.spaceEnd(q)) == '~' }

func (l *lexer) quote() {
	if l.at(l.p+1) == '`' {
		l.push(sMdString)
		l.p += 2
	} else {
		l.push(sString)
		l.p++
	}
}

func (l *lexer) invisLink() bool {
	if !l.invisAhead(l.p) {
		return false
	}
	e := invisible(l.src, l.p)
	if e < 0 {
		return false
	}
	l.emit(t_LINK, l.p, e)
	return true
}

func (l *lexer) opener(c int) bool {
	p := l.p
	d := l.at(p + 1)
	switch c {
	case '(':
		switch {
		case d == '-':
			l.push(sEllipseText)
			l.emit(t_ELLIPSE_START, p, p+2)
		case d == '[':
			l.push(sText)
			l.emit(t_STADIUMSTART, p, p+2)
		case d == '(' && l.at(p+2) == '(':
			l.push(sText)
			l.emit(t_DOUBLECIRCLESTART, p, p+3)
		default:
			l.push(sText)
			l.emit(t_PS, p, p+1)
		}
		return true
	case '[':
		switch d {
		case '[':
			l.push(sText)
			l.emit(t_SUBROUTINESTART, p, p+2)
		case '(':
			l.push(sText)
			l.emit(t_CYLINDERSTART, p, p+2)
		case '/':
			l.push(sTrapText)
			l.emit(t_TRAPSTART, p, p+2)
		case '\\':
			l.push(sTrapText)
			l.emit(t_INVTRAPSTART, p, p+2)
		default:
			l.push(sText)
			l.emit(t_SQS, p, p+1)
		}
		return true
	case '|':
		l.push(sText)
		l.emit(t_PIPE, p, p+1)
		return true
	case '{':
		l.push(sText)
		l.emit(t_DIAMOND_START, p, p+1)
		return true
	}
	return false
}

func (l *lexer) graph(n int) {
	if l.firstGraph {
		l.firstGraph = false
		l.push(sDir)
	}
	l.emit(t_GRAPH, l.p, l.p+n)
}

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
	l.emit(t_NODE_STRING, l.p, q)
	return true
}

func (l *lexer) link(c int) bool {
	p := l.p
	d := l.linkChar(p)
	var e int
	switch d {
	case '-':
		if e = solidLink(l.src, p, '-'); e >= 0 {
			l.emit(t_LINK, p, e)
			return true
		}
		if e = startLink(l.src, p, '-'); e >= 0 {
			l.push(sEdgeText)
			l.emit(t_START_LINK, p, e)
			return true
		}
		if e = dottedLink(l.src, p); e >= 0 {
			l.emit(t_LINK, p, e)
			return true
		}
		if e = dottedStart(l.src, p); e >= 0 {
			l.push(sDottedEdgeText)
			l.emit(t_START_LINK, p, e)
			return true
		}
	case '=':
		if e = solidLink(l.src, p, '='); e >= 0 {
			l.emit(t_LINK, p, e)
			return true
		}
		if e = startLink(l.src, p, '='); e >= 0 {
			l.push(sThickEdgeText)
			l.emit(t_START_LINK, p, e)
			return true
		}
	case '.':
		if e = dottedLink(l.src, p); e >= 0 {
			l.emit(t_LINK, p, e)
			return true
		}
	}
	return c != '-' && c != '=' && c != '.' && l.invisLink()
}

func (l *lexer) initial(c int) bool {
	p := l.p
	src := l.src
	var e int
	switch c {
	case 'a':
		if strings.HasPrefix(src[p:], "acc") {
			if e = keywordColon(src, p, "accTitle", ':'); e >= 0 {
				l.push(sAccTitle)
				l.emit(t_acc_title, p, e)
				return true
			}
			if e = keywordColon(src, p, "accDescr", ':'); e >= 0 {
				l.push(sAccDescr)
				l.emit(t_acc_descr, p, e)
				return true
			}
			if e = keywordColon(src, p, "accDescr", '{'); e >= 0 {
				l.push(sAccDescrMultiline)
				l.p = e
				return true
			}
		}
	case '@':
		if l.at(p+1) == '{' {
			l.push(sShapeData)
			l.toks = append(l.toks, tok{t_SHAPE_DATA, "", p})
			l.p += 2
			return true
		}
	case 'c':
		if e = keywordSpace(src, p, "call"); e >= 0 {
			l.push(sCallbackname)
			l.p = e
			return true
		}
		if l.kw("classDef") {
			l.emit(t_CLASSDEF, p, p+8)
			return true
		}
		if l.kw("class") {
			l.emit(t_CLASS, p, p+5)
			return true
		}
		if e = keywordSpace(src, p, "click"); e >= 0 {
			l.push(sClick)
			l.p = e
			return true
		}
	case '"':
		l.quote()
		return true
	case 's':
		if l.kw("style") {
			l.emit(t_STYLE, p, p+5)
			return true
		}
		if l.kw("swimlane-beta") {
			l.graph(13)
			return true
		}
		if l.kw("subgraph") {
			l.emit(t_subgraph, p, p+8)
			return true
		}
	case 'd':
		if l.kw("default") {
			l.emit(t_DEFAULT, p, p+7)
			return true
		}
	case 'l':
		if l.kw("linkStyle") {
			l.emit(t_LINKSTYLE, p, p+9)
			return true
		}
	case 'i':
		if l.kw("interpolate") {
			l.emit(t_INTERPOLATE, p, p+11)
			return true
		}
	case 'h':
		if strings.HasPrefix(src[p:], "href") {
			if d, size := chars.UnitAt(src, p+4); chars.IsSpace(d) {
				l.emit(t_HREF, p, p+4+size)
				return true
			}
		}
	case 'f':
		if l.kw("flowchart-elk") {
			l.graph(13)
			return true
		}
		if l.kw("flowchart") {
			l.graph(9)
			return true
		}
	case 'g':
		if l.kw("graph") {
			l.graph(5)
			return true
		}
	case 'e':
		if l.kw("end") {
			l.emit(t_end, p, space(src, p+3))
			return true
		}
	case '_':
		for _, w := range []string{"_self", "_blank", "_parent", "_top"} {
			if l.kw(w) {
				l.emit(t_LINK_TARGET, p, p+len(w))
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
						l.emit(t_direction_tb+token(i), p, e)
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
			l.emit(t_LINK_ID, p, l.runAt+1)
			return true
		}
	}

	if c >= '0' && c <= '9' {
		q := p + 1
		for q < l.n && src[q] >= '0' && src[q] <= '9' {
			q++
		}
		l.emit(t_NUM, p, q)
		return true
	}

	switch c {
	case '#':
		l.emit(t_BRKT, p, p+1)
		return true
	case ':':
		if l.at(p+1) == ':' && l.at(p+2) == ':' {
			l.emit(t_STYLE_SEPARATOR, p, p+3)
		} else {
			l.emit(t_COLON, p, p+1)
		}
		return true
	case '&':
		l.emit(t_AMP, p, p+1)
		return true
	case ';':
		l.emit(t_SEMI, p, p+1)
		return true
	case ',':
		l.emit(t_COMMA, p, p+1)
		return true
	case '*':
		l.emit(t_MULT, p, p+1)
		return true
	case 'x', 'o':
		return l.link(c) || l.nodeString()
	case '<':
		if l.link(c) {
			return true
		}
		l.emit(t_TAGSTART, p, p+1)
		return true
	case '-':
		if l.link(c) || l.nodeString() {
			return true
		}
		l.emit(t_MINUS, p, p+1)
		return true
	case '=':
		return l.link(c)
	case '.':
		return l.link(c) || l.nodeString()
	case '~':
		return l.invisLink()
	case '(':
		return l.opener(c)
	case '[':
		if l.at(p+1) == '|' {
			l.emit(t_VERTEX_WITH_PROPS_START, p, p+2)
			return true
		}
		return l.opener(c)
	case '|', '{':
		return l.opener(c)
	case '>':
		l.push(sText)
		l.emit(t_TAGEND, p, p+1)
		return true
	case '^':
		l.emit(t_UP, p, p+1)
		return true
	case '\\':
		if l.at(p+1) == '|' {
			l.emit(t_SEP, p, p+2)
			return true
		}
		return l.nodeString()
	case 'v':
		if !chars.IsWord(l.at(p + 1)) {
			l.emit(t_DOWN, p, p+1)
			return true
		}
		return l.nodeString()
	}

	if chars.IsSpace(c) {
		if l.link(c) {
			return true
		}
		if e = newline(src, p); e >= 0 {
			l.emit(t_NEWLINE, p, e)
			return true
		}
		if e = blanks(src, p); e >= 0 {
			l.emit(t_SPACE, p, e)
			return true
		}
		return false
	}

	if c < 128 {
		return l.nodeString()
	}
	if chars.IsUnicodeLetter(c) {
		_, size := chars.UnitAt(src, p)
		l.emit(t_UNICODE_TEXT, p, p+size)
		return true
	}
	return false
}

func (l *lexer) edgeText(c int, linkRe func(string, int) int, a, b int) bool {
	p := l.p
	if c == '"' {
		l.quote()
		return true
	}
	d := l.linkChar(p)
	if d == a || d == b {
		if e := linkRe(l.src, p); e >= 0 {
			l.pop()
			l.emit(t_LINK, p, e)
			return true
		}
	}
	if c == a && (a != '-' || l.at(p+1) == '-') {
		return false
	}
	_, size := chars.UnitAt(l.src, p)
	q := p + size
	for q < l.n {
		ch, size := chars.UnitAt(l.src, q)
		if ch == '"' {
			break
		}
		if ch == a {
			if a != '-' || l.at(q+1) == '-' {
				break
			}
		} else if ch == 'x' || ch == 'o' || ch == '<' || ch == b || chars.IsSpace(ch) {
			dd := l.linkChar(q)
			if (dd == a || dd == b) && linkRe(l.src, q) >= 0 {
				break
			}
		}
		q += size
	}
	l.emit(t_EDGE_TEXT, p, q)
	return true
}

func solidDash(s string, q int) int  { return solidLink(s, q, '-') }
func solidThick(s string, q int) int { return solidLink(s, q, '=') }

// index is s.indexOf(sub, from).
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
				l.emit(typ, l.n, l.n)
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
					l.p += 2
				} else {
					l.pop()
					l.p++
				}
			} else {
				e := index(src, "\"", p)
				if e == -1 {
					e = l.n
				}
				l.emit(t_STR, p, e)
			}

		case sMdString:
			if c == '`' {
				if l.at(p+1) == '"' {
					l.pop()
					l.p += 2
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
				l.emit(t_MD_STR, p, q)
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
			l.emit(typ, p, e)

		case sAccDescrMultiline:
			if c == '}' {
				l.pop()
				l.p++
			} else {
				e := index(src, "}", p)
				if e == -1 {
					e = l.n
				}
				l.emit(t_acc_descr_multiline_value, p, e)
			}

		case sDir:
			var e int
			if c == '"' {
				l.quote()
			} else if e = nodir(src, p); e >= 0 {
				l.pop()
				l.emit(t_NODIR, p, e)
			} else if e = dir(src, p); e >= 0 {
				l.pop()
				l.emit(t_DIR, p, e)
			} else if e = dirSym(src, p); e >= 0 {
				l.pop()
				l.emit(t_DIR, p, e)
			} else if e = dirV(src, p); e >= 0 {
				l.pop()
				l.emit(t_DIR, p, e)
			} else {
				ok = l.invisLink() || l.opener(c)
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
					l.emit(t_STADIUMEND, p, p+2)
				} else if d == ']' {
					l.emit(t_SUBROUTINEEND, p, p+2)
				} else {
					l.emit(t_SQE, p, p+1)
				}
			case ')':
				l.pop()
				if d == ']' {
					l.emit(t_CYLINDEREND, p, p+2)
				} else if d == ')' && l.at(p+2) == ')' {
					l.emit(t_DOUBLECIRCLEEND, p, p+3)
				} else {
					l.emit(t_PE, p, p+1)
				}
			case '|':
				l.pop()
				l.emit(t_PIPE, p, p+1)
			case '}':
				l.pop()
				l.emit(t_DIAMOND_STOP, p, p+1)
			default:
				if (c == '~' || chars.IsSpace(c)) && l.invisLink() {
					break
				}
				q := p + size
				for q < l.n && strings.IndexByte("[]()}{|\"", src[q]) < 0 {
					q++
				}
				l.emit(t_TEXT, p, q)
			}

		case sEllipseText:
			d := l.at(p + 1)
			if c == '"' {
				l.quote()
			} else if (c == '~' || chars.IsSpace(c)) && l.invisLink() {
				break
			} else if (c == '-' || c == '/' || c == ')') && d == ')' {
				l.pop()
				l.emit(t_ELLIPSE_END, p, p+2)
			} else if c == '(' || c == '[' || c == '{' {
				l.opener(c)
			} else if c == ')' || c == ']' || c == '}' {
				ok = false
			} else {
				q := p + size
				for q < l.n {
					ch, size := chars.UnitAt(src, q)
					if strings.IndexByte("\"()[]{}", byte(ch)) >= 0 && ch < 128 {
						break
					}
					if (ch == '-' || ch == '/') && l.at(q+1) == ')' {
						break
					}
					if (ch == '~' || chars.IsSpace(ch)) && l.invisAhead(q) && invisible(src, q) >= 0 {
						break
					}
					q += size
				}
				l.emit(t_TEXT, p, q)
			}

		case sTrapText:
			d := l.at(p + 1)
			if c == '"' {
				l.quote()
				break
			}
			if (c == '~' || chars.IsSpace(c)) && l.invisLink() {
				break
			}
			switch c {
			case '(':
				if d == '-' || d == '[' || d == '(' && l.at(p+2) == '(' {
					l.opener(c)
				} else if d == ']' {
					l.pop()
					l.emit(t_TRAPEND, p, p+2)
				} else {
					l.opener(c)
				}
			case '[', '{':
				l.opener(c)
			case '\\':
				if d == ']' {
					l.pop()
					l.emit(t_TRAPEND, p, p+2)
				} else {
					l.emit(t_TEXT, p, p+1)
				}
			case '/':
				if d == ']' {
					l.pop()
					l.emit(t_INVTRAPEND, p, p+2)
				} else {
					l.emit(t_TEXT, p, p+1)
				}
			case ']', ')':
				if d == ']' {
					l.pop()
					l.emit(t_TRAPEND, p, p+2)
				} else {
					ok = false
				}
			case '}':
				ok = false
			default:
				if (c == '?' || c == '=') && d == ']' {
					l.pop()
					l.emit(t_TRAPEND, p, p+2)
					break
				}
				q := p + size
				for q < l.n && strings.IndexByte("\\[]()}{/", src[q]) < 0 {
					q++
				}
				l.emit(t_TEXT, p, q)
			}

		case sEdgeText:
			ok = l.edgeText(c, solidDash, '-', -1)
		case sThickEdgeText:
			ok = l.edgeText(c, solidThick, '=', -1)
		case sDottedEdgeText:
			ok = l.edgeText(c, dottedLink, '.', '-')

		case sClick:
			if c == '"' {
				l.quote()
			} else if chars.IsSpace(c) {
				l.pop()
				l.p += size
			} else {
				q := p + size
				for q < l.n {
					ch, size := chars.UnitAt(src, q)
					if chars.IsSpace(ch) {
						break
					}
					q += size
				}
				l.emit(t_CLICK, p, q)
			}

		case sCallbackname:
			if c == '(' {
				e := emptyCall(src, p)
				l.pop()
				if e >= 0 {
					l.p = e
				} else {
					l.push(sCallbackargs)
					l.p++
				}
			} else {
				e := index(src, "(", p)
				if e == -1 {
					e = l.n
				}
				l.emit(t_CALLBACKNAME, p, e)
			}

		case sCallbackargs:
			if c == ')' {
				l.pop()
				l.p++
			} else {
				e := index(src, ")", p)
				if e == -1 {
					e = l.n
				}
				l.emit(t_CALLBACKARGS, p, e)
			}

		case sShapeData:
			if c == '"' {
				l.push(sShapeDataStr)
				l.emit(t_SHAPE_DATA, p, p+1)
			} else if c == '}' {
				l.pop()
				l.p++
			} else if c == '^' {
				ok = false
			} else {
				q := p + size
				for q < l.n && src[q] != '}' && src[q] != '^' && src[q] != '"' {
					q++
				}
				l.emit(t_SHAPE_DATA, p, q)
			}

		case sShapeDataStr:
			if c == '"' {
				l.pop()
				l.emit(t_SHAPE_DATA, p, p+1)
			} else {
				e := index(src, "\"", p)
				if e == -1 {
					e = l.n
				}
				l.toks = append(l.toks, tok{t_SHAPE_DATA, breaks(src[p:e]), p})
				l.p = e
			}
		}

		if !ok {
			l.toks = append(l.toks, tok{t_ERROR, "", l.p})
			return l.toks
		}
	}

	l.toks = append(l.toks, tok{t_END, "", l.n})
	return l.toks
}
