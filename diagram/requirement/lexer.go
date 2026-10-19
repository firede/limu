package requirement

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
	sStyle
	sAccTitle
	sAccDescr
	sAccDescrMultiline
)

// longestKeyword is the length of the longest keyword: a longer word is no keyword.
const longestKeyword = 22

// stop holds the characters that end an unquoted string: `:,{<>-=` and line breaks.
var stop = func() (t [128]bool) {
	for _, c := range ":,\r\n{<>-=" {
		t[c] = true
	}
	return
}()

// The regular expressions of Pele's lexer, written out. Each is sticky: it matches at q or
// not at all, and gives the end of its match or -1.

// keywordSep matches /word\s*<sep>\s*/iy.
func keywordSep(s string, q int, word string, sep byte) int {
	if !chars.HasPrefixFold(s[q:], word) {
		return -1
	}
	j := chars.SpaceRun(s, q+len(word))
	if j >= len(s) || s[j] != sep {
		return -1
	}
	return chars.SpaceRun(s, j+1)
}

// directionAt matches /direction\s+(?:(TB)|(BT)|(RL)|LR)/i at q: the kind of direction, 0 to
// 3 for TB, BT, RL and LR, and the end of the match, or -1.
func directionAt(s string, q int) (kind, end int) {
	if !chars.HasPrefixFold(s[q:], "direction") {
		return -1, -1
	}
	j := chars.SpaceRun(s, q+len("direction"))
	if j == q+len("direction") || j+2 > len(s) {
		return -1, -1
	}
	for k, d := range [...]string{"TB", "BT", "RL", "LR"} {
		if chars.EqualFoldASCII(s[j:j+2], d) {
			return k, j + 2
		}
	}
	return -1, -1
}

// lineBreak is the first line feed at or after q, or the end.
func lineBreak(s string, q int) int {
	if e := strings.IndexByte(s[q:], '\n'); e >= 0 {
		return q + e
	}
	return len(s)
}

// tokenize produces the token stream of Mermaid's requirement lexer, as Pele's tokenize
// does. Its rules are tried in order and the first match wins, so a line that holds a
// direction statement anywhere is one token. A text no rule matches ends the stream with an
// ERROR token where it starts.
func tokenize(src string) []tok {
	n := len(src)
	var toks []tok
	var stack []int
	state := sINITIAL
	p := 0

	// Where each kind of direction statement starts and where its keyword ends, found in one
	// pass, as RE_DIRECTION finds them: each search starts where the last match ended.
	var dirStarts, dirEnds [4][]int
	var dirSeen [4]int
	anyDir := false
	lineEnd := -1
	for i := 0; i < n; {
		if c := src[i]; c == 'd' || c == 'D' {
			if k, e := directionAt(src, i); k >= 0 {
				dirStarts[k] = append(dirStarts[k], i)
				dirEnds[k] = append(dirEnds[k], e)
				anyDir = true
				i = e
				continue
			}
		}
		i++
	}

	emit := func(typ token, s, e int) {
		toks = append(toks, tok{typ, src[s:e], s})
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

	// `.*direction\s+XX[^\n]*`: the last such statement of a kind that starts on this line,
	// the kinds tried in their order, decides the token.
	direction := func() bool {
		if lineEnd < p {
			lineEnd = n
			for q := p; q < n; {
				c, size := chars.UnitAt(src, q)
				if c == 10 || c == 13 || c == 0x2028 || c == 0x2029 {
					lineEnd = q
					break
				}
				q += size
			}
			for k := range 4 {
				list := dirStarts[k]
				j := dirSeen[k]
				for j < len(list) && list[j] <= lineEnd {
					j++
				}
				dirSeen[k] = j
			}
		}
		for k := range 4 {
			j := dirSeen[k] - 1
			if j >= 0 && dirStarts[k][j] >= p {
				emit(t_direction_tb+token(k), p, lineBreak(src, dirEnds[k][j]))
				return true
			}
		}
		return false
	}

	initial := func(c, size int) bool {
		lower := c | 32
		if lower == 't' && chars.HasPrefixFold(src[p+1:], "itle") {
			if d, dsize := chars.UnitAt(src, p+5); chars.IsSpace(d) {
				from := p + 5 + dsize
				q := from
				for q < n {
					if d := src[q]; d == '#' || d == '\n' || d == ';' {
						break
					}
					q++
				}
				if q > from {
					emit(t_title, p, q)
					return true
				}
			}
		}
		if lower == 'a' && chars.Unit(src, p+1)|32 == 'c' && chars.Unit(src, p+2)|32 == 'c' {
			if e := keywordSep(src, p, "accTitle", ':'); e >= 0 {
				push(sAccTitle)
				emit(t_acc_title, p, e)
				return true
			}
			if e := keywordSep(src, p, "accDescr", ':'); e >= 0 {
				push(sAccDescr)
				emit(t_acc_descr, p, e)
				return true
			}
			if e := keywordSep(src, p, "accDescr", '{'); e >= 0 {
				push(sAccDescrMultiline)
				p = e
				return true
			}
		}
		if anyDir && !chars.IsLineEnd(c) && direction() {
			return true
		}

		if c == '\n' || c == '\r' && chars.Unit(src, p+1) == '\n' {
			q := p
			for q < n {
				if d := src[q]; d == '\n' {
					q++
				} else if d == '\r' && q+1 < n && src[q+1] == '\n' {
					q += 2
				} else {
					break
				}
			}
			emit(t_NEWLINE, p, q)
			return true
		}
		if chars.IsSpace(c) {
			p = chars.SpaceRun(src, p+size)
			return true
		}
		if c == '#' || c == '%' {
			p = lineBreak(src, p)
			return true
		}

		if chars.IsWord(c) {
			q := p + 1
			for q < n && chars.IsWord(int(src[q])) {
				q++
			}
			if q-p <= longestKeyword {
				if keyword, ok := keywords[strings.ToLower(src[p:q])]; ok {
					if keyword == t_STYLE || keyword == t_CLASSDEF || keyword == t_CLASS {
						push(sStyle)
					}
					emit(keyword, p, q)
					return true
				}
			}
			for q < n {
				if d := src[q]; d < 128 && stop[d] {
					break
				}
				q++
			}
			toks = append(toks, tok{t_unqString, chars.Trim(src[p:q]), p})
			p = q
			return true
		}

		switch c {
		case '{':
			emit(t_STRUCT_START, p, p+1)
			return true
		case '}':
			emit(t_STRUCT_STOP, p, p+1)
			return true
		case ':':
			if strings.HasPrefix(src[p:], ":::") {
				emit(t_STYLE_SEPARATOR, p, p+3)
			} else {
				emit(t_COLONSEP, p, p+1)
			}
			return true
		case '<':
			if chars.Unit(src, p+1) != '-' {
				return false
			}
			emit(t_END_ARROW_L, p, p+2)
			return true
		case '-':
			if chars.Unit(src, p+1) == '>' {
				emit(t_END_ARROW_R, p, p+2)
			} else {
				emit(t_LINE, p, p+1)
			}
			return true
		case '"':
			push(sString)
			p++
			return true
		case ',':
			emit(t_COMMA, p, p+1)
			return true
		}
		return false
	}

	style := func(c int) bool {
		if chars.IsWord(c) {
			q := p + 1
			for q < n && chars.IsWord(int(src[q])) {
				q++
			}
			emit(t_ALPHA, p, q)
			return true
		}
		switch c {
		case ':':
			emit(t_COLON, p, p+1)
			return true
		case ';':
			emit(t_SEMICOLON, p, p+1)
			return true
		case '%':
			emit(t_PERCENT, p, p+1)
			return true
		case '-':
			emit(t_MINUS, p, p+1)
			return true
		case '#':
			emit(t_BRKT, p, p+1)
			return true
		case ' ':
			p++
			return true
		case '"':
			push(sString)
			p++
			return true
		case '\n':
			pop()
			p++
			return true
		case ',':
			emit(t_COMMA, p, p+1)
			return true
		}
		return false
	}

	upTo := func(q int, b byte) int {
		if e := strings.IndexByte(src[q:], b); e >= 0 {
			return q + e
		}
		return n
	}

	for p < n {
		c, size := chars.UnitAt(src, p)
		ok := true
		switch state {
		case sINITIAL:
			ok = initial(c, size)
		case sStyle:
			ok = style(c)
		case sString:
			if c == '"' {
				pop()
				p++
			} else {
				emit(t_qString, p, upTo(p, '"'))
			}
		case sAccTitle:
			pop()
			emit(t_acc_title_value, p, lineBreak(src, p))
		case sAccDescr:
			pop()
			emit(t_acc_descr_value, p, lineBreak(src, p))
		case sAccDescrMultiline:
			if c == '}' {
				pop()
				p++
			} else {
				emit(t_acc_descr_multiline_value, p, upTo(p, '}'))
			}
		}
		if !ok {
			return append(toks, tok{t_ERROR, "", p})
		}
	}

	// At the end of input Jison still tries the rules that can match nothing, once.
	switch state {
	case sINITIAL:
		emit(t_EOF, n, n)
	case sString:
		emit(t_qString, n, n)
	case sAccTitle:
		emit(t_acc_title_value, n, n)
	case sAccDescr:
		emit(t_acc_descr_value, n, n)
	case sAccDescrMultiline:
		emit(t_acc_descr_multiline_value, n, n)
	}
	emit(t_END, n, n)
	return toks
}
