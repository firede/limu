package class

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
	sBqstring
	sGeneric
	sCallbackName
	sCallbackArgs
	sAccTitle
	sAccDescr
	sAccDescrMultiline
	sCls
	sClassBody
	sNamespace
	sNamespaceBody
)

// atEnd is the token each state gives at the end of input, before the parser sees the end
// itself: Pele's AT_END.
var atEnd = [...]token{
	t_EOF,
	t_STR,
	t_EOF,
	t_GENERICTYPE,
	t_CALLBACK_NAME,
	t_CALLBACK_ARGS,
	t_acc_title_value,
	t_acc_descr_value,
	t_acc_descr_multiline_value,
	t_EOF,
	t_EOF_IN_STRUCT,
	t_EOF,
	t_EOF_IN_STRUCT,
}

// directionNames are the directions Pele's RE_DIRECTION matches, in its order.
var directionNames = [4]string{"TB", "BT", "RL", "LR"}

// lexer is Pele's tokenize, its closures as methods.
type lexer struct {
	src       string
	n         int
	toks      []tok
	stack     []int
	state     int
	p         int
	dirNext   int
	lineEnd   int
	dirFailed int
}

// tokenize splits a class diagram's text into tokens, as Pele's tokenize does. The last
// token is t_END, or t_ERROR where no token starts.
func tokenize(src string) []tok {
	l := &lexer{src: src, n: len(src), dirNext: strings.Index(src, "direction"), lineEnd: -1}
	return l.run()
}

// emit adds a token of the source from s to e, and reads on from e, as the emit of Pele's
// tokenize does. It reports true, so that a match can return it.
func (l *lexer) emit(typ token, s, e int) bool {
	l.toks = append(l.toks, tok{typ, l.src[s:e], s})
	l.p = e
	return true
}

// push enters state s, keeping the one it leaves, as the push of Pele's tokenize does.
func (l *lexer) push(s int) {
	l.stack = append(l.stack, l.state)
	l.state = s
}

// pop goes back to the state kept last, if any, as the pop of Pele's tokenize does.
func (l *lexer) pop() {
	if len(l.stack) > 0 {
		l.state = l.stack[len(l.stack)-1]
		l.stack = l.stack[:len(l.stack)-1]
	}
}

// unit is the code unit JavaScript reads at byte i, or -1 past the end: the charCodeAt Pele's
// tokenize calls.
func (l *lexer) unit(i int) int { return chars.Unit(l.src, i) }

// kw reports whether word starts at p and no word character follows it, as the kw of Pele's
// tokenize does.
func (l *lexer) kw(word string) bool {
	return strings.HasPrefix(l.src[l.p:], word) && !chars.IsWord(l.unit(l.p+len(word)))
}

// skipWs is the end of the run of white space from q, as the skipWs of Pele's tokenize gives
// it.
func (l *lexer) skipWs(q int) int { return chars.SpaceRun(l.src, q) }

// until is where the next ch is from p, or the end, as the until of Pele's tokenize gives it.
func (l *lexer) until(ch byte) int {
	if e := strings.IndexByte(l.src[l.p:], ch); e >= 0 {
		return l.p + e
	}
	return l.n
}

// space reads a run of white space: a NEWLINE up to its last line feed, and the rest is
// skipped. leaves leaves the state before the NEWLINE. It is the space of Pele's tokenize.
func (l *lexer) space(leaves bool) {
	q := l.p
	k := -1
	for q < l.n {
		c, size := chars.UnitAt(l.src, q)
		if !chars.IsSpace(c) {
			break
		}
		if c == 10 {
			k = q
		}
		q += size
	}
	if k < 0 {
		l.p = q
		return
	}
	if leaves {
		l.pop()
	}
	l.emit(t_NEWLINE, l.p, k+1)
}

// keyword reads the keywords that every state recognizes, even inside generics and backtick
// names, as the keyword of Pele's tokenize does.
func (l *lexer) keyword(c int) bool {
	p := l.p
	switch c {
	case 99:
		if l.kw("cssClass") {
			return l.emit(t_CSSCLASS, p, p+8)
		}
		if l.kw("callback") {
			return l.emit(t_CALLBACK, p, p+8)
		}
		if l.kw("click") {
			return l.emit(t_CLICK, p, p+5)
		}
	case 108:
		if l.kw("link") {
			return l.emit(t_LINK, p, p+4)
		}
	case 110:
		if l.kw("note for") {
			return l.emit(t_NOTE_FOR, p, p+8)
		}
		if l.kw("note") {
			return l.emit(t_NOTE, p, p+4)
		}
	case 60:
		if l.unit(p+1) == 60 {
			return l.emit(t_ANNOTATION_START, p, p+2)
		}
	case 62:
		if l.unit(p+1) == 62 {
			return l.emit(t_ANNOTATION_END, p, p+2)
		}
	case 104:
		if l.kw("href") {
			return l.emit(t_HREF, p, p+4)
		}
	}
	return false
}

// common reads what the states that read statements share, as the common of Pele's tokenize
// does.
func (l *lexer) common(c, size int) bool {
	if l.keyword(c) {
		return true
	}
	p := l.p
	d := l.unit(p + 1)
	switch c {
	case 34:
		l.push(sString)
		l.p++
		return true
	case 126:
		l.push(sGeneric)
		l.p++
		return true
	case 96:
		l.push(sBqstring)
		l.p++
		return true
	case 95:
		for _, target := range [...]string{"_self", "_blank", "_parent", "_top"} {
			if l.kw(target) {
				return l.emit(t_LINK_TARGET, p, p+len(target))
			}
		}
	case 60:
		if d == 124 {
			return l.emit(t_EXTENSION, p, p+2)
		}
		return l.emit(t_DEPENDENCY, p, p+1)
	case 124:
		return d == 62 && l.emit(t_EXTENSION, p, p+2)
	case 62:
		return l.emit(t_DEPENDENCY, p, p+1)
	case 42:
		return l.emit(t_COMPOSITION, p, p+1)
	case 111:
		if !chars.IsWord(d) {
			return l.emit(t_AGGREGATION, p, p+1)
		}
	case 40:
		return d == 41 && l.emit(t_LOLLIPOP, p, p+2)
	case 45:
		if d == 45 {
			return l.emit(t_LINE, p, p+2)
		}
		return l.emit(t_MINUS, p, p+1)
	case 46:
		if d == 46 {
			return l.emit(t_DOTTED_LINE, p, p+2)
		}
		return l.emit(t_DOT, p, p+1)
	case 58:
		q := p + 1
		for q < l.n {
			ch := l.src[q]
			if ch == ':' || ch == '\n' || ch == ';' {
				break
			}
			q++
		}
		if q > p+1 {
			return l.emit(t_LABEL, p, q)
		}
		return d == 58 && l.unit(p+2) == 58 && l.emit(t_STYLE_SEPARATOR, p, p+3)
	case 43:
		return l.emit(t_PLUS, p, p+1)
	case 37:
		return l.emit(t_PCT, p, p+1)
	case 61:
		return l.emit(t_EQUALS, p, p+1)
	case 91:
		return l.emit(t_SQS, p, p+1)
	case 93:
		return l.emit(t_SQE, p, p+1)
	case 33, 35, 36, 38, 39, 44, 63, 92, 47:
		return l.emit(t_PUNCTUATION, p, p+1)
	}
	if chars.IsWord(c) {
		q := p + 1
		for chars.IsWord(l.unit(q)) {
			q++
		}
		return l.emit(t_ALPHA, p, q)
	}
	return chars.IsUnicodeLetter(c) && l.emit(t_UNICODE_TEXT, p, p+size)
}

// directionEnd matches /.*direction\s+<name>[^\n]*/y at q, on a line that ends at lineEnd:
// the last `direction` on the line that is followed by white space and the name, and the
// rest of the text up to a line feed. It is -1 where none matches. It reads one of Pele's
// RE_DIRECTION by hand.
func directionEnd(s string, q, lineEnd int, name string) int {
	for x := lineEnd - 1; x >= q; x-- {
		if s[x] != 'd' || !strings.HasPrefix(s[x:], "direction") {
			continue
		}
		j := chars.SpaceRun(s, x+9)
		if j == x+9 || !strings.HasPrefix(s[j:], name) {
			continue
		}
		if e := strings.IndexByte(s[j+2:], '\n'); e >= 0 {
			return j + 2 + e
		}
		return len(s)
	}
	return -1
}

// direction reads a direction statement that starts anywhere on the line, as the direction
// of Pele's tokenize does.
func (l *lexer) direction() bool {
	if l.dirNext == -1 || l.p < l.dirFailed {
		return false
	}
	if l.dirNext < l.p {
		if i := strings.Index(l.src[l.p:], "direction"); i >= 0 {
			l.dirNext = l.p + i
		} else {
			l.dirNext = -1
		}
	}
	if l.dirNext == -1 {
		return false
	}
	if l.lineEnd < l.p {
		l.lineEnd = l.n
		for q := l.p; q < l.n; {
			c, size := chars.UnitAt(l.src, q)
			if chars.IsLineEnd(c) {
				l.lineEnd = q
				break
			}
			q += size
		}
	}
	if l.dirNext >= l.lineEnd {
		return false
	}
	for i, name := range directionNames {
		if e := directionEnd(l.src, l.p, l.lineEnd, name); e >= 0 {
			return l.emit(t_direction_tb+token(i), l.p, e)
		}
	}
	// No direction statement starts anywhere in the rest of this line either.
	l.dirFailed = l.lineEnd
	return false
}

// initial reads a token in the INITIAL state, as the initial of Pele's tokenize does.
func (l *lexer) initial(c, size int) bool {
	if l.direction() {
		return true
	}
	if chars.IsSpace(c) {
		l.space(false)
		return true
	}
	p := l.p
	switch c {
	case 37:
		if l.unit(p+1) == 37 {
			e := l.until('\n')
			for e < l.n && (l.src[e] == '\n' || l.src[e] == '\r') {
				e++
			}
			l.p = e
			return true
		}
	case 97:
		title := strings.HasPrefix(l.src[p:], "accTitle")
		if !title && !strings.HasPrefix(l.src[p:], "accDescr") {
			break
		}
		q := l.skipWs(p + 8)
		d := l.unit(q)
		if d == 58 {
			if title {
				l.emit(t_acc_title, p, l.skipWs(q+1))
				l.push(sAccTitle)
			} else {
				l.emit(t_acc_descr, p, l.skipWs(q+1))
				l.push(sAccDescr)
			}
			return true
		}
		if d == 123 && !title {
			l.p = l.skipWs(q + 1)
			l.push(sAccDescrMultiline)
			return true
		}
	case 99:
		if l.kw("classDiagram-v2") {
			return l.emit(t_CLASS_DIAGRAM, p, p+15)
		}
		if l.kw("classDiagram") {
			return l.emit(t_CLASS_DIAGRAM, p, p+12)
		}
		if strings.HasPrefix(l.src[p:], "call") && chars.IsSpace(l.unit(p+4)) {
			l.p = l.skipWs(p + 4)
			l.push(sCallbackName)
			return true
		}
		if l.kw("classDef") {
			return l.emit(t_CLASSDEF, p, p+8)
		}
		if l.kw("class") {
			l.push(sCls)
			return l.emit(t_CLASS, p, p+5)
		}
	case 91:
		if strings.HasPrefix(l.src[p+1:], "*]") {
			return l.emit(t_EDGE_STATE, p, p+3)
		}
	case 115:
		if l.kw("style") {
			return l.emit(t_STYLE, p, p+5)
		}
	case 110:
		if l.kw("namespace") {
			l.push(sNamespace)
			return l.emit(t_NAMESPACE, p, p+9)
		}
	case 44:
		return l.emit(t_COMMA, p, p+1)
	case 35:
		return l.emit(t_BRKT, p, p+1)
	}
	return l.common(c, size) || c == 58 && l.emit(t_COLON, p, p+1)
}

// run reads tokens until the end of the source or one that cannot be read: the loop of Pele's
// tokenize, which reads each state's tokens.
func (l *lexer) run() []tok {
	for {
		if l.p >= l.n {
			l.toks = append(l.toks, tok{atEnd[l.state], "", l.n})
			break
		}
		p := l.p
		c, size := chars.UnitAt(l.src, p)
		ok := true
		switch l.state {
		case sINITIAL:
			ok = l.initial(c, size)

		case sCls:
			switch {
			case chars.IsSpace(c):
				l.space(true)
			case c == 125:
				l.pop()
				l.pop()
				l.emit(t_STRUCT_STOP, p, p+1)
			case c == 123:
				l.push(sClassBody)
				l.emit(t_STRUCT_START, p, p+1)
			default:
				ok = l.common(c, size)
			}

		case sClassBody:
			switch {
			case c == 34:
				l.push(sString)
				l.p++
			case c == 125:
				l.pop()
				l.emit(t_STRUCT_STOP, p, p+1)
			case c == 91 && strings.HasPrefix(l.src[p+1:], "*]"):
				l.emit(t_EDGE_STATE, p, p+3)
			case c == 123:
				l.emit(t_OPEN_IN_STRUCT, p, p+1)
			case c == 10:
				l.p++
			default:
				q := p + 1
				for q < l.n {
					ch := l.src[q]
					if ch == '{' || ch == '}' || ch == '\n' {
						break
					}
					q++
				}
				l.emit(t_MEMBER, p, q)
			}

		case sNamespace:
			switch {
			case chars.IsSpace(c):
				l.space(true)
			case c == 123:
				l.push(sNamespaceBody)
				l.emit(t_STRUCT_START, p, p+1)
			case c == 125:
				// The brace is read again in the state the namespace was opened from.
				l.pop()
			case c == 110 && l.kw("namespace"):
				l.push(sNamespace)
				l.emit(t_NAMESPACE, p, p+9)
			default:
				ok = l.common(c, size)
			}

		case sNamespaceBody:
			switch {
			case chars.IsSpace(c):
				l.space(false)
			case c == 125:
				l.pop()
				l.emit(t_STRUCT_STOP, p, p+1)
			case c == 110 && l.kw("namespace"):
				l.push(sNamespace)
				l.emit(t_NAMESPACE, p, p+9)
			case c == 91 && strings.HasPrefix(l.src[p+1:], "*]"):
				l.emit(t_EDGE_STATE, p, p+3)
			case c == 99 && l.kw("class"):
				l.push(sCls)
				l.emit(t_CLASS, p, p+5)
			default:
				ok = l.common(c, size)
			}

		case sString:
			if c == 34 {
				l.pop()
				l.p++
			} else {
				l.emit(t_STR, p, l.until('"'))
			}

		case sGeneric:
			switch {
			case c == 34:
				l.push(sString)
				l.p++
			case l.keyword(c):
			case c == 126:
				l.pop()
				l.p++
			default:
				l.emit(t_GENERICTYPE, p, l.until('~'))
			}

		case sBqstring:
			switch {
			case c == 34:
				l.push(sString)
				l.p++
			case l.keyword(c):
			case c == 126:
				l.push(sGeneric)
				l.p++
			case c == 96:
				l.pop()
				l.p++
			default:
				l.emit(t_BQUOTE_STR, p, l.until('`'))
			}

		case sCallbackName:
			if c == 40 {
				q := l.skipWs(p + 1)
				l.pop()
				if l.unit(q) == 41 {
					l.p = q + 1
				} else {
					l.push(sCallbackArgs)
					l.p++
				}
			} else {
				l.emit(t_CALLBACK_NAME, p, l.until('('))
			}

		case sCallbackArgs:
			if c == 41 {
				l.pop()
				l.p++
			} else {
				l.emit(t_CALLBACK_ARGS, p, l.until(')'))
			}

		case sAccTitle, sAccDescr:
			typ := t_acc_descr_value
			if l.state == sAccTitle {
				typ = t_acc_title_value
			}
			l.emit(typ, p, l.until('\n'))
			l.pop()

		case sAccDescrMultiline:
			if c == 125 {
				l.pop()
				l.p++
			} else {
				l.emit(t_acc_descr_multiline_value, p, l.until('}'))
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
