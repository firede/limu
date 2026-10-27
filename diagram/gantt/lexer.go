package gantt

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
	sAccTitle
	sAccDescr
	sAccDescrMultiline
	sHref
	sCallbackname
	sCallbackargs
	sClick
)

var dayNames = [...]string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}

// unitAt is the code unit JavaScript's charCodeAt reads at byte i of s, -1 past the end, as
// Pele's tokenize reads src.charCodeAt.
func unitAt(s string, i int) int {
	c, _ := chars.UnitAt(s, i)
	return c
}

// isSpaceAt reports whether the character at byte i of s is \s, as Pele's isSpace of its
// charCodeAt does.
func isSpaceAt(s string, i int) bool { return chars.IsSpace(unitAt(s, i)) }

// spaceEnd is the end of a run of \s from byte i, as the loops of Pele's tokenize over
// isSpace find it.
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

// foldAt reports whether word, in ASCII letters, is at byte i of s but for case, as the i flag
// of Pele's patterns matches it.
func foldAt(s string, i int, word string) bool { return chars.HasPrefixFold(s[i:], word) }

// isWordByte is \w, as the \b of Pele's patterns tests the byte after a word.
func isWordByte(s string, i int) bool { return i < len(s) && chars.IsWord(int(s[i])) }

// The patterns the lexer matches where it stands, as Pele's sticky regular expressions, each
// written out: the end of the match, or -1.

// matchAcc is /word\s*sep\s*/iy, for accTitle and accDescr with ':' or '{': Pele's
// RE_ACC_TITLE, RE_ACC_DESCR and RE_ACC_DESCR_ML.
func matchAcc(s string, p int, word string, sep byte) int {
	if !foldAt(s, p, word) {
		return -1
	}
	e := spaceEnd(s, p+len(word))
	if e >= len(s) || s[e] != sep {
		return -1
	}
	return spaceEnd(s, e+1)
}

// matchSpaced is /word\s+/iy, and with quote set /word\s+"/iy: Pele's RE_CALL, RE_CLICK and
// RE_HREF.
func matchSpaced(s string, p int, word string, quote bool) int {
	if !foldAt(s, p, word) {
		return -1
	}
	e := spaceEnd(s, p+len(word))
	if e == p+len(word) {
		return -1
	}
	if quote {
		if e >= len(s) || s[e] != '"' {
			return -1
		}
		e++
	}
	return e
}

// matchWord is /word\b/iy, as the keywords of Pele's KEYWORDS that take nothing after them.
func matchWord(s string, p int, word string) int {
	if !foldAt(s, p, word) || isWordByte(s, p+len(word)) {
		return -1
	}
	return p + len(word)
}

// matchRest is /word\s[^stop]+/iy: the keyword, one white space character, and then all up to
// the first of the stop bytes, of which there must be one or more. It is the keywords of
// Pele's KEYWORDS that take a value.
func matchRest(s string, p int, word, stop string) int {
	if !foldAt(s, p, word) {
		return -1
	}
	q := p + len(word)
	c, size := chars.UnitAt(s, q)
	if !chars.IsSpace(c) {
		return -1
	}
	q += size
	e := q
	for e < len(s) && strings.IndexByte(stop, s[e]) < 0 {
		e++
	}
	if e == q {
		return -1
	}
	return e
}

// matchBareTitle is /title[^\S\r\n]*(?=[\r\n]|$)/iy, Pele's RE_BARE_TITLE: the word alone on
// the rest of its line, but for white space.
func matchBareTitle(s string, p int) int {
	if !foldAt(s, p, "title") {
		return -1
	}
	e := p + len("title")
	for {
		c, size := chars.UnitAt(s, e)
		if c == '\n' || c == '\r' || !chars.IsSpace(c) {
			break
		}
		e += size
	}
	if e < len(s) && s[e] != '\n' && s[e] != '\r' {
		return -1
	}
	return e
}

// matchTitle is /title[^\S\r\n][^\n]+/iy: the word, a white space character that breaks no
// line, and the rest of the line, of which there must be some.
func matchTitle(s string, p int) int {
	if c := chars.Unit(s, p+len("title")); c == '\n' || c == '\r' {
		return -1
	}
	return matchRest(s, p, "title", "\n")
}

// matchDays is /prefix\s+(name|…)\b/iy: the end and the index of the name. It is Pele's
// RE_WEEKDAY and RE_WEEKEND.
func matchDays(s string, p int, prefix string, names []string) (int, int) {
	e := matchSpaced(s, p, prefix, false)
	if e < 0 {
		return -1, -1
	}
	for k, name := range names {
		if foldAt(s, e, name) && !isWordByte(s, e+len(name)) {
			return e + len(name), k
		}
	}
	return -1, -1
}

// matchDate is /\d\d\d\d-\d\d-\d\d\b/y, Pele's RE_DATE.
func matchDate(s string, p int) bool {
	if p+10 > len(s) {
		return false
	}
	for i, c := range []byte(s[p : p+10]) {
		if i == 4 || i == 7 {
			if c != '-' {
				return false
			}
		} else if c < '0' || c > '9' {
			return false
		}
	}
	return !isWordByte(s, p+10)
}

// rule is a statement that starts with a keyword: its pattern and its token.
type rule struct {
	match func(s string, p int) int
	typ   token
}

// rest is the pattern of a keyword of Pele's KEYWORDS that takes a value up to a stop byte.
func rest(word, stop string) func(string, int) int {
	return func(s string, p int) int { return matchRest(s, p, word, stop) }
}

// bounded is the pattern of a keyword of Pele's KEYWORDS that ends at a word boundary.
func bounded(word string) func(string, int) int {
	return func(s string, p int) int { return matchWord(s, p, word) }
}

// keywordRules are Pele's KEYWORDS: statements that start with a keyword, by the keyword's
// first letter and in the grammar's rule order.
var keywordRules = map[int][]rule{
	'a': {{rest("axisFormat", "#\n;"), tAxisFormat}, {rest("accDescription", "#\n;"), tAccDescription}},
	'g': {{bounded("gantt"), tGantt}},
	'd': {{rest("dateFormat", "#\n;"), tDateFormat}},
	'i': {{bounded("inclusiveEndDates"), tInclusiveEndDates}, {rest("includes", "#\n;"), tIncludes}},
	't': {
		{bounded("topAxis"), tTopAxis},
		{rest("tickInterval", "#\n;"), tTickInterval},
		{rest("todayMarker", "\n;"), tTodayMarker},
		{matchTitle, tTitle},
	},
	'e': {{rest("excludes", "#\n;"), tExcludes}},
	's': {{rest("section", "\n"), tSection}},
}

// tokenize is Pele's tokenize: the tokens of a gantt chart, ending with tEND.
func tokenize(src string) []tok {
	n := len(src)
	var toks []tok
	state := sINITIAL
	p := 0

	// emit, until and keyword are Pele's closures of the same names; the at of Pele's tokenize
	// is the match functions above.
	emit := func(typ token, s, e int) {
		toks = append(toks, tok{typ, src[s:e], s})
		p = e
	}
	until := func(ch byte, from int) int {
		if from > n {
			return n
		}
		e := strings.IndexByte(src[from:], ch)
		if e == -1 {
			return n
		}
		return from + e
	}

	keyword := func(c int) bool {
		var e int
		switch c | 32 {
		case 'a':
			if e = matchAcc(src, p, "accTitle", ':'); e >= 0 {
				state = sAccTitle
				emit(tAccTitle, p, e)
				return true
			}
			if e = matchAcc(src, p, "accDescr", ':'); e >= 0 {
				state = sAccDescr
				emit(tAccDescr, p, e)
				return true
			}
			if e = matchAcc(src, p, "accDescr", '{'); e >= 0 {
				state = sAccDescrMultiline
				p = e
				return true
			}
		case 'h':
			if e = matchSpaced(src, p, "href", true); e >= 0 {
				state = sHref
				p = e
				return true
			}
		case 'c':
			if e = matchSpaced(src, p, "call", false); e >= 0 {
				state = sCallbackname
				p = e
				return true
			}
			if e = matchSpaced(src, p, "click", false); e >= 0 {
				state = sClick
				p = e
				return true
			}
		case 't':
			// A bare title is skipped, as no title.
			if e = matchBareTitle(src, p); e >= 0 {
				p = e
				return true
			}
		case 'w':
			if e, k := matchDays(src, p, "weekday", dayNames[:]); e >= 0 {
				emit(tWeekdayMonday+token(k), p, e)
				return true
			}
			if e, k := matchDays(src, p, "weekend", []string{"friday", "saturday"}); e >= 0 {
				if k == 0 {
					emit(tWeekendFriday, p, e)
				} else {
					emit(tWeekendSaturday, p, e)
				}
				return true
			}
		}
		for _, r := range keywordRules[c|32] {
			if e = r.match(src, p); e >= 0 {
				emit(r.typ, p, e)
				return true
			}
		}
		return false
	}

	for {
		if p >= n {
			typ := tEOF
			switch state {
			case sAccTitle:
				typ = tAccTitleValue
			case sAccDescr:
				typ = tAccDescrValue
			case sAccDescrMultiline:
				typ = tAccDescrMultilineValue
			case sHref:
				typ = tHref
			case sCallbackname:
				typ = tCallbackname
			case sCallbackargs:
				typ = tCallbackargs
			case sClick:
				typ = tClick
			}
			emit(typ, n, n)
			break
		}
		c, size := chars.UnitAt(src, p)

		switch state {
		case sINITIAL:
			// The code unit after c: a character above the Basic Multilingual Plane is followed
			// by its low surrogate, which is no character looked for here.
			d := -1
			if size < 4 {
				d = unitAt(src, p+size)
			}
			switch {
			case c == '%':
				if d != '%' {
					e := until('\n', p+1)
					for e < n && src[e] == '\n' {
						e++
					}
					p = e
				} else if unitAt(src, p+2) == '{' {
					// The grammar enters a lexer state it never defines, so nothing can follow.
					emit(tOpenDirective, p, p+3)
					emit(tERROR, p, p)
					return append(toks, tok{tEND, "", n})
				} else {
					p = until('\n', p+2)
				}
			case c == '\n':
				e := p + 1
				for e < n && src[e] == '\n' {
					e++
				}
				emit(tNL, p, e)
			case chars.IsSpace(c):
				p = spaceEnd(src, p)
			case keyword(c):
			case c >= '0' && c <= '9' && matchDate(src, p):
				emit(tDate, p, p+10)
			case c != ':':
				e := p + 1
				for e < n && src[e] != ':' && src[e] != '\n' {
					e++
				}
				emit(tTaskTxt, p, e)
			default:
				e := p + 1
				for e < n && src[e] != '#' && src[e] != '\n' && src[e] != ';' {
					e++
				}
				if e > p+1 {
					emit(tTaskData, p, e)
				} else {
					emit(tCOLON, p, e)
				}
			}

		case sAccTitle, sAccDescr:
			typ := tAccTitleValue
			if state == sAccDescr {
				typ = tAccDescrValue
			}
			state = sINITIAL
			emit(typ, p, until('\n', p))

		case sAccDescrMultiline:
			if c == '}' {
				state = sINITIAL
				p++
			} else {
				emit(tAccDescrMultilineValue, p, until('}', p))
			}

		case sHref:
			if c == '"' {
				state = sINITIAL
				p++
			} else {
				emit(tHref, p, until('"', p))
			}

		case sCallbackname:
			if c == '(' {
				if e := spaceEnd(src, p+1); e < n && src[e] == ')' {
					state = sINITIAL
					p = e + 1
				} else {
					state = sCallbackargs
					p++
				}
			} else {
				emit(tCallbackname, p, until('(', p))
			}

		case sCallbackargs:
			if c == ')' {
				state = sINITIAL
				p++
			} else {
				emit(tCallbackargs, p, until(')', p))
			}

		case sClick:
			if chars.IsSpace(c) {
				state = sINITIAL
				p += size
			} else {
				e := p + size
				for e < n && !isSpaceAt(src, e) {
					_, sz := chars.UnitAt(src, e)
					e += sz
				}
				emit(tClick, p, e)
			}
		}
	}
	return append(toks, tok{tEND, "", n})
}
