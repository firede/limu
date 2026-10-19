package state

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

// tokens is what the lexer makes of a text, Pele's Tokens: the tokens, and the message of the
// grammar's own error where lexing stopped on one rather than on text no rule matches.
type tokens struct {
	toks     []tok
	error    string
	hasError bool
}

// The states of the lexer, Pele's S.
const (
	sINITIAL = iota
	sStruct
	sSTATE
	sSTATE_STRING
	sSTATE_ID
	sSCALE
	sAccTitle
	sAccDescr
	sAccDescrMultiline
	sCLASSDEF
	sCLASSDEFID
	sCLASS
	sCLASS_STYLE
	sSTYLE
	sSTYLEDEF_STYLES
	sNOTE
	sNOTE_ID
	sNOTE_TEXT
	sFLOATING_NOTE
	sFLOATING_NOTE_ID
)

// The patterns Pele's lexer searches for, written out: each gives the first byte at or after q
// where it matches, or -1.

// searchDirection is /direction/gi.
func searchDirection(s string, q int) int {
	for i := q; i+9 <= len(s); i++ {
		if s[i]|32 == 'd' && chars.EqualFoldASCII(s[i:i+9], "direction") {
			return i
		}
	}
	return -1
}

// searchStereotype is /<<(?:fork|join|choice)>>|\[\[(?:fork|join|choice)\]\]/gi.
func searchStereotype(s string, q int) int {
	for i := q; i+8 <= len(s); i++ {
		var end string
		switch {
		case s[i] == '<' && s[i+1] == '<':
			end = ">>"
		case s[i] == '[' && s[i+1] == '[':
			end = "]]"
		default:
			continue
		}
		for _, name := range [...]string{"fork", "join", "choice"} {
			if chars.HasPrefixFold(s[i+2:], name) && strings.HasPrefix(s[i+2+len(name):], end) {
				return i
			}
		}
	}
	return -1
}

// searchLineEnd is /[\n\r  ]/g.
func searchLineEnd(s string, q int) int {
	for i := q; i < len(s); i++ {
		switch s[i] {
		case '\n', '\r':
			return i
		case 0xe2:
			if i+2 < len(s) && s[i+1] == 0x80 && (s[i+2] == 0xa8 || s[i+2] == 0xa9) {
				return i
			}
		}
	}
	return -1
}

// searchPercent is /%%/g.
func searchPercent(s string, q int) int {
	if q > len(s) {
		return -1
	}
	if i := strings.Index(s[q:], "%%"); i >= 0 {
		return q + i
	}
	return -1
}

// searchBrace is /\{/g.
func searchBrace(s string, q int) int {
	if q > len(s) {
		return -1
	}
	if i := strings.IndexByte(s[q:], '{'); i >= 0 {
		return q + i
	}
	return -1
}

// finder finds the next match of a pattern at or after a position, remembering the answer so
// that asking at each token costs one scan of the text in total, as Pele's finder does.
func finder(search func(s string, q int) int, src string) func(q int) int {
	from, next := 0, -2
	return func(q int) int {
		if next == -2 || q < from || (next != -1 && q > next) {
			next = search(src, q)
			from = q
		}
		return next
	}
}

// tokenize reads a state diagram's text into tokens, as Pele's tokenize does: the lexer of
// Mermaid's stateDiagram.jison, written out by hand.
func tokenize(src string) tokens {
	n := len(src)
	var out tokens
	var stack []int
	state := sINITIAL
	p := 0

	nextDirection := finder(searchDirection, src)
	nextStereotype := finder(searchStereotype, src)
	nextLineEnd := finder(searchLineEnd, src)
	nextPercent := finder(searchPercent, src)
	nextBrace := finder(searchBrace, src)

	// The last `direction <dir>` of each kind on the line being lexed, and where its keyword
	// ends.
	dirFrom, dirTo := -1, -1
	dirAt := [4]int{-1, -1, -1, -1}
	var dirEnd [4]int
	// The same for <<fork>>, <<join>>, <<choice>> and their [[ ]] spellings.
	stereoFrom, stereoTo := -1, -1
	stereoAt := [6]int{-1, -1, -1, -1, -1, -1}

	// at is the byte at i, which is the code unit there for ASCII, and -1 past the end.
	at := func(i int) int {
		if i < 0 || i >= n {
			return -1
		}
		return int(src[i])
	}
	emit := func(typ token, s, e int) {
		out.toks = append(out.toks, tok{typ, src[s:e], s})
		p = e
	}
	emitText := func(typ token, text string, s, e int) {
		out.toks = append(out.toks, tok{typ, text, s})
		p = e
	}
	push := func(s int) {
		stack = append(stack, state)
		state = s
	}
	pop := func() {
		if len(stack) > 0 {
			state = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		}
	}
	// word is a case-insensitive match of a lowercase ASCII word, as the grammar's /i flag
	// gives.
	word := func(w string, q int) bool {
		for i := 0; i < len(w); i++ {
			a := w[i]
			if q+i >= n {
				return false
			}
			c := src[q+i]
			if c != a && !(a >= 'a' && a <= 'z' && c == a-32) {
				return false
			}
		}
		return true
	}
	wsEnd := func(q int) int { return chars.SpaceRun(src, q) }
	wordEnd := func(q int) int {
		j := q
		for j < n && chars.IsWord(int(src[j])) {
			j++
		}
		return j
	}
	eol := func(q int) int {
		if k := strings.IndexByte(src[q:], '\n'); k >= 0 {
			return q + k
		}
		return n
	}
	lineEnd := func(q int) int {
		if k := nextLineEnd(q); k != -1 {
			return k
		}
		return n
	}
	// blankRun is the end of the run of white space other than line feeds that starts at q.
	blankRun := func(q int) int {
		j := q
		for j < n {
			c, size := chars.UnitAt(src, j)
			if c == 10 || !chars.IsSpace(c) {
				break
			}
			j += size
		}
		return j
	}

	// direction is the rules `.*direction\s+TB[^\n]*` and their three siblings: the first
	// kind, in grammar order, found between q and the end of its line.
	direction := func(q int) int {
		if q < dirFrom || q > dirTo {
			first := nextDirection(q)
			end := n
			if first != -1 {
				end = lineEnd(q)
			}
			dirFrom, dirTo = q, end
			dirAt = [4]int{-1, -1, -1, -1}
			for d := first; d != -1 && d < end; d = nextDirection(d + 1) {
				w := wsEnd(d + 9)
				if w == d+9 {
					continue
				}
				kind := -1
				switch {
				case word("tb", w):
					kind = 0
				case word("bt", w):
					kind = 1
				case word("rl", w):
					kind = 2
				case word("lr", w):
					kind = 3
				}
				if kind == -1 {
					continue
				}
				dirAt[kind] = d
				dirEnd[kind] = w + 2
			}
		}
		for kind := 0; kind < 4; kind++ {
			if dirAt[kind] >= q {
				return kind
			}
		}
		return -1
	}

	stereotype := func(q int) int {
		if q < stereoFrom || q > stereoTo {
			first := nextStereotype(q)
			end := n
			if first != -1 {
				end = lineEnd(q)
			}
			stereoFrom, stereoTo = q, end
			stereoAt = [6]int{-1, -1, -1, -1, -1, -1}
			for d := first; d != -1 && d < end; d = nextStereotype(d + 1) {
				letter := src[d+2] | 32
				k := 3
				if src[d] == '<' {
					k = 0
				}
				switch letter {
				case 'f':
				case 'j':
					k++
				default:
					k += 2
				}
				stereoAt[k] = d
			}
		}
		for kind := 0; kind < 6; kind++ {
			if stereoAt[kind] >= q {
				return kind
			}
		}
		return -1
	}

	// emitID is the grammar's processId(): an id ends at an inline `%%`, and a text that
	// starts with one is dropped.
	emitID := func(s, e int) bool {
		at := nextPercent(s)
		if at == -1 || at+2 > e {
			emit(t_ID, s, e)
			return true
		}
		if at == s {
			p = e
			return false
		}
		emit(t_ID, s, at)
		return true
	}

	finish := func(typ token) tokens {
		out.toks = append(out.toks, tok{typ, "", p})
		return out
	}

	for {
		if p >= n {
			// Jison tries the current state's rules once more on the empty remainder.
			switch state {
			case sINITIAL:
				emit(t_NL, n, n)
			case sAccTitle:
				emit(t_acc_title_value, n, n)
			case sAccDescr:
				emit(t_acc_descr_value, n, n)
			case sAccDescrMultiline:
				emit(t_acc_descr_multiline_value, n, n)
			case sCLASSDEFID:
				emit(t_CLASSDEF_STYLEOPTS, n, n)
			case sCLASS_STYLE:
				emit(t_STYLECLASS, n, n)
			case sSTYLEDEF_STYLES:
				emit(t_STYLEDEF_STYLEOPTS, n, n)
			case sSTATE_STRING:
				emit(t_STATE_DESCR, n, n)
			case sFLOATING_NOTE:
				emit(t_NOTE_TEXT, n, n)
			case sSTATE_ID, sFLOATING_NOTE_ID:
				emit(t_ID, n, n)
			}
			return finish(t_END)
		}

		c, size := chars.UnitAt(src, p)
		switch state {
		case sINITIAL, sStruct:
			root := state == sINITIAL
			lc := c | 32
			if root {
				if lc == 'c' && word("click", p) && !chars.IsWord(at(p+5)) {
					emit(t_CLICK, p, p+5)
					continue
				}
				if lc == 'h' && word("href", p) && !chars.IsWord(at(p+4)) {
					emit(t_HREF, p, p+4)
					continue
				}
				if c == '"' {
					if close := strings.IndexByte(src[p+1:], '"'); close != -1 {
						emit(t_STRING, p, p+1+close+1)
						continue
					}
				}
				if lc == 'd' && word("default", p) && !chars.IsWord(at(p+7)) {
					emit(t_DEFAULT, p, p+7)
					continue
				}
				if kind := direction(p); kind != -1 {
					emit(t_direction_tb+token(kind), p, eol(dirEnd[kind]))
					continue
				}
				if c == 10 {
					j := p + 1
					for at(j) == 10 {
						j++
					}
					emit(t_NL, p, j)
					continue
				}
				if chars.IsSpace(c) {
					p = wsEnd(p)
					continue
				}
			} else if c != 10 && chars.IsSpace(c) {
				p = blankRun(p + size)
				continue
			}
			if c == '#' || (c == '%' && at(p+1) == '%' && at(p+2) != '{') {
				p = eol(p)
				continue
			}
			switch {
			case lc == 's':
				if root && word("scale", p) {
					if e := wsEnd(p + 5); e > p+5 {
						emit(t_scale, p, e)
						push(sSCALE)
						continue
					}
				}
				if word("style", p) {
					if e := wsEnd(p + 5); e > p+5 {
						emit(t_style, p, e)
						push(sSTYLE)
						continue
					}
				}
				if word("state", p) {
					if e := wsEnd(p + 5); e > p+5 {
						p = e
						push(sSTATE)
						continue
					}
					if root && word("diagram", p+5) {
						from := p + 12
						if word("-v2", p+12) {
							from = p + 15
						}
						if e := wsEnd(from); e > from {
							emit(t_SD, p, e)
							continue
						}
					}
				}
			case lc == 'c':
				if word("class", p) {
					def := word("def", p+5)
					from := p + 5
					if def {
						from = p + 8
					}
					if e := wsEnd(from); e > from {
						if def {
							emit(t_classDef, p, e)
							push(sCLASSDEF)
						} else {
							emit(t_class, p, e)
							push(sCLASS)
						}
						continue
					}
				}
			case lc == 'a' && root:
				if word("acctitle", p) || word("accdescr", p) {
					title := src[p+3]|32 == 't'
					q := wsEnd(p + 8)
					d := at(q)
					if d == ':' {
						if title {
							emit(t_acc_title, p, wsEnd(q+1))
							push(sAccTitle)
						} else {
							emit(t_acc_descr, p, wsEnd(q+1))
							push(sAccDescr)
						}
						continue
					}
					if d == '{' && !title {
						p = wsEnd(q + 1)
						push(sAccDescrMultiline)
						continue
					}
				}
			}
			if root {
				if c == '{' {
					emit(t_STRUCT_START, p, p+1)
					push(sStruct)
					continue
				}
			} else {
				if kind := direction(p); kind != -1 {
					emit(t_direction_tb+token(kind), p, eol(dirEnd[kind]))
					continue
				}
				if c == '}' {
					emit(t_STRUCT_STOP, p, p+1)
					pop()
					continue
				}
				if c == 10 {
					p++
					continue
				}
			}
			if lc == 'n' && word("note", p) {
				if e := wsEnd(p + 4); e > p+4 {
					emit(t_note, p, e)
					push(sNOTE)
					continue
				}
			}
			if root && lc == 'h' && word("hide empty description", p) && !chars.IsWord(at(p+22)) {
				emit(t_HIDE_EMPTY, p, p+22)
				continue
			}
			if c == '[' && at(p+1) == '*' && at(p+2) == ']' {
				emit(t_EDGE_STATE, p, p+3)
				continue
			}
			j := p
			for j < n {
				d, size := chars.UnitAt(src, j)
				if d == ':' || d == '-' || d == '{' || chars.IsSpace(d) {
					break
				}
				j += size
			}
			if j > p {
				emitID(p, j)
				continue
			}
			if c == ':' {
				j = p + 1
				for j < n {
					d := at(j)
					if d == 10 || d == ';' {
						break
					}
					if d == ':' {
						after := at(j + 1)
						if j+1 >= n || after == ':' || after == 10 || after == ';' {
							break
						}
						j++
					}
					j++
				}
				if j > p+1 {
					emitText(t_DESCR, chars.Trim(src[p:j]), p, j)
					continue
				}
				if at(p+1) == ':' && at(p+2) == ':' {
					emit(t_STYLE_SEPARATOR, p, p+3)
					continue
				}
			} else if c == '-' && at(p+1) == '-' {
				if at(p+2) == '>' {
					emit(t_ARROW, p, p+3)
					continue
				}
				if !root {
					emit(t_CONCURRENT, p, p+2)
					continue
				}
			}
			if !root {
				return finish(t_ERROR)
			}
			emit(t_INVALID, p, p+1)
			continue

		case sSTATE:
			if c != 10 && chars.IsSpace(c) {
				p = blankRun(p + size)
				continue
			}
			if c == '#' || (c == '%' && at(p+1) == '%' && at(p+2) != '{') {
				p = eol(p)
				continue
			}
			if kind := stereotype(p); kind != -1 {
				sAt := stereoAt[kind]
				choice := kind == 2 || kind == 5
				typ, length := t_JOIN, 8
				switch {
				case choice:
					typ, length = t_CHOICE, 10
				case kind%3 == 0:
					typ = t_FORK
				}
				emitText(typ, chars.Trim(src[p:sAt]), p, sAt+length)
				pop()
				continue
			}
			if c == '"' {
				p++
				push(sSTATE_STRING)
				continue
			}
			q := p
			if c == 10 {
				q = wsEnd(p)
			}
			if word("as", q) {
				if e := wsEnd(q + 2); e > q+2 {
					emit(t_AS, p, e)
					push(sSTATE_ID)
					continue
				}
			}
			if c == 10 {
				p++
				pop()
				continue
			}
			if c == '{' {
				emit(t_STRUCT_START, p, p+1)
				pop()
				push(sStruct)
				continue
			}
			if chars.IsWord(c) {
				first := wordEnd(p)
				second := wsEnd(first)
				if second > first && chars.IsWord(at(second)) {
					if brace := nextBrace(second + 1); brace != -1 && brace < lineEnd(second) {
						out.error = `Error: State name must be a single word. Found: "` + chars.Trim(src[p:brace+1]) + `"`
						out.hasError = true
						return finish(t_ERROR)
					}
				}
			}
			j := p + size
			for j < n {
				d, size := chars.UnitAt(src, j)
				if d == '{' || chars.IsSpace(d) {
					break
				}
				j += size
			}
			emit(t_COMPOSIT_STATE, p, j)

		case sSTATE_STRING:
			if c == '"' {
				p++
				pop()
				continue
			}
			end := n
			if close := strings.IndexByte(src[p:], '"'); close != -1 {
				end = p + close
			}
			emit(t_STATE_DESCR, p, end)

		case sSTATE_ID:
			j := p
			for j < n && src[j] != '\n' && src[j] != '{' {
				j++
			}
			if emitID(p, j) {
				pop()
			}

		case sSCALE:
			if chars.IsDigit(c) {
				j := p + 1
				for j < n && chars.IsDigit(int(src[j])) {
					j++
				}
				emit(t_WIDTH, p, j)
				continue
			}
			if q := wsEnd(p); q > p && word("width", q) && !chars.IsWord(at(q+5)) {
				p = q + 5
				pop()
				continue
			}
			return finish(t_ERROR)

		case sAccTitle:
			emit(t_acc_title_value, p, eol(p))
			pop()

		case sAccDescr:
			emit(t_acc_descr_value, p, eol(p))
			pop()

		case sAccDescrMultiline:
			if c == '}' {
				p++
				pop()
				continue
			}
			end := n
			if close := strings.IndexByte(src[p:], '}'); close != -1 {
				end = p + close
			}
			emit(t_acc_descr_multiline_value, p, end)

		case sCLASSDEF:
			isDefault := word("default", p) && wsEnd(p+7) > p+7
			j := wordEnd(p)
			if isDefault {
				j = p + 7
			}
			e := wsEnd(j)
			if j == p || e == j {
				return finish(t_ERROR)
			}
			if isDefault {
				emit(t_DEFAULT_CLASSDEF_ID, p, e)
			} else {
				emit(t_CLASSDEF_ID, p, e)
			}
			pop()
			push(sCLASSDEFID)

		case sCLASSDEFID:
			emit(t_CLASSDEF_STYLEOPTS, p, eol(p))
			pop()

		case sCLASS:
			j := wordEnd(p)
			if j == p {
				return finish(t_ERROR)
			}
			for at(j) == ',' {
				k := wsEnd(j + 1)
				e := wordEnd(k)
				if e == k {
					break
				}
				j = e
			}
			emit(t_CLASSENTITY_IDS, p, j)
			pop()
			push(sCLASS_STYLE)

		case sCLASS_STYLE:
			emit(t_STYLECLASS, p, eol(p))
			pop()

		case sSTYLE:
			j := p
			for j < n && (chars.IsWord(int(src[j])) || src[j] == ',') {
				j++
			}
			e := wsEnd(j)
			if j == p || e == j {
				return finish(t_ERROR)
			}
			emit(t_STYLE_IDS, p, e)
			pop()
			push(sSTYLEDEF_STYLES)

		case sSTYLEDEF_STYLES:
			emit(t_STYLEDEF_STYLEOPTS, p, eol(p))
			pop()

		case sNOTE:
			if word("left of", p) && !chars.IsWord(at(p+7)) {
				emit(t_left_of, p, p+7)
				pop()
				push(sNOTE_ID)
				continue
			}
			if word("right of", p) && !chars.IsWord(at(p+8)) {
				emit(t_right_of, p, p+8)
				pop()
				push(sNOTE_ID)
				continue
			}
			if c == '"' {
				p++
				pop()
				push(sFLOATING_NOTE)
				continue
			}
			return finish(t_ERROR)

		case sNOTE_ID:
			q := wsEnd(p)
			j := q
			for j < n {
				d, size := chars.UnitAt(src, j)
				if d == ':' || d == '-' || chars.IsSpace(d) {
					break
				}
				j += size
			}
			if j == q {
				return finish(t_ERROR)
			}
			if emitID(p, j) {
				pop()
				push(sNOTE_TEXT)
			}

		case sNOTE_TEXT:
			q := wsEnd(p)
			if at(q) == ':' {
				j := q + 1
				for j < n {
					d := src[j]
					if d == ':' || d == '\n' || d == ';' {
						break
					}
					j++
				}
				if j > q+1 {
					// Mermaid drops two characters here, expecting " :"; with no space before
					// the colon the first character of the note goes too.
					emitText(t_NOTE_TEXT, chars.Trim(dropTwoUnits(src[p:j])), p, j)
					pop()
					continue
				}
			}
			k := strings.IndexByte(src[p:], '\n')
			if k != -1 {
				k += p
			}
			e := -1
			for k != -1 {
				e = wsEnd(k)
				if word("end note", e) && !chars.IsWord(at(e+8)) {
					break
				}
				k = strings.IndexByte(src[e:], '\n')
				if k != -1 {
					k += e
				}
			}
			if k == -1 {
				return finish(t_ERROR)
			}
			emitText(t_NOTE_TEXT, chars.Trim(src[p:e]), p, e+8)
			pop()

		case sFLOATING_NOTE:
			if q := wsEnd(p); word("as", q) {
				emit(t_AS, p, wsEnd(q+2))
				pop()
				push(sFLOATING_NOTE_ID)
				continue
			}
			if c == '"' {
				p++
				continue
			}
			end := n
			if close := strings.IndexByte(src[p:], '"'); close != -1 {
				end = p + close
			}
			emit(t_NOTE_TEXT, p, end)

		case sFLOATING_NOTE_ID:
			if emitID(p, eol(p)) {
				pop()
			}
		}
	}
}

// dropTwoUnits is s.slice(2): s without its first two UTF-16 code units. Where the second is
// the first half of a character outside the Basic Multilingual Plane, the half left over is
// what a Go string cannot hold, and becomes U+FFFD, as JSON gives it to Go.
func dropTwoUnits(s string) string {
	i := 0
	for units := 0; units < 2 && i < len(s); {
		c, size := chars.UnitAt(s, i)
		if c >= 0xd800 && c <= 0xdbff {
			if units == 1 {
				return "�" + s[i+size:]
			}
			units += 2
		} else {
			units++
		}
		i += size
	}
	return s[i:]
}
