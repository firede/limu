package xychart

import (
	"strings"

	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The types of token, in the order of Pele's T.
const (
	tEND = iota
	tERROR
	tNEWLINE
	tTitle
	tAccTitle
	tAccTitleValue
	tAccDescr
	tAccDescrValue
	tAccDescrMultilineValue
	tXYCHART
	tCHART_ORIENTATION
	tX_AXIS
	tY_AXIS
	tARROW_DELIMITER
	tLINE
	tBAR
	tNUMBER_WITH_DECIMAL
	tSQUARE_BRACES_START
	tSQUARE_BRACES_END
	tSTR
	tALPHA
	tCOLON
	tPLUS
	tCOMMA
	tEQUALS
	tMULT
	tBRKT
	tUNDERSCORE
	tDOT
	tAMP
	tMINUS
	tNUM
	tSEMI
	tEOF
)

// tokenNames are the names of the types of token, as Mermaid's errors give them: Pele's
// TOKEN_NAMES.
var tokenNames = [...]string{
	"$end", "INVALID", "NEWLINE", "title", "acc_title", "acc_title_value", "acc_descr", "acc_descr_value",
	"acc_descr_multiline_value", "XYCHART", "CHART_ORIENTATION", "X_AXIS", "Y_AXIS", "ARROW_DELIMITER", "LINE", "BAR",
	"NUMBER_WITH_DECIMAL", "SQUARE_BRACES_START", "SQUARE_BRACES_END", "STR", "ALPHA", "COLON", "PLUS", "COMMA",
	"EQUALS", "MULT", "BRKT", "UNDERSCORE", "DOT", "AMP", "MINUS", "NUM", "SEMI", "EOF",
}

// The states of the lexer, as Pele's S.
const (
	sINITIAL = iota
	sAxisData
	sAxisBandData
	sData
	sDataInner
	sString
	sAccTitle
	sAccDescr
	sAccDescrMultiline
)

// atEnd is the token each state gives for the empty rest of the input, Pele's AT_END.
var atEnd = [...]int{tEOF, tEOF, tEOF, tEOF, tEOF, tSTR, tAccTitleValue, tAccDescrValue, tAccDescrMultilineValue}

// brokenRule is the text that Mermaid's rule for a markdown string matches, Pele's
// BROKEN_RULE. The grammar means to open a markdown string at `"` followed by a backtick, but
// its quoting turns the rule into one that matches this text and leaves the current state, so
// markdown strings are never read.
var brokenRule = "`)" + strings.Repeat(" ", 36) + "{ this.pushstate(md_string); }\n<md_string>(?:(?!`\").)+" +
	strings.Repeat(" ", 18) + "{ return md_str; }\n<md_string>(?:`"

// single is the token of each character that is a token on its own, Pele's SINGLE.
var single = [128]uint8{
	':': tCOLON, '+': tPLUS, ',': tCOMMA, '=': tEQUALS, '*': tMULT, '#': tBRKT, '_': tUNDERSCORE, '.': tDOT,
	'&': tAMP, '-': tMINUS, ';': tSEMI,
}

// tokenize splits an XY chart's text into tokens, as Pele's tokenize does. Where Pele reads
// a code unit with charCodeAt, chars.UnitAt reads the same one; a character outside ASCII is
// white space, which is stepped over whole, or ends the tokens.
func tokenize(src string) *common.JisonTokens {
	n := len(src)
	out := &common.JisonTokens{}
	var stack []int
	state := sINITIAL
	p := 0

	emit := func(typ, s, e int) {
		out.Emit(typ, src, s, e)
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
	indexFrom := func(c byte, q int) int {
		if at := strings.IndexByte(src[q:], c); at >= 0 {
			return q + at
		}
		return n
	}
	digits := func(q int) int {
		for q < n && chars.IsDigit(int(src[q])) {
			q++
		}
		return q
	}
	keyword := func(word string) bool {
		return common.WordAt(src, p, word) && !chars.IsWord(chars.Unit(src, p+len(word)))
	}

	for p < n {
		c, size := chars.UnitAt(src, p)

		if state == sString {
			if c == '"' {
				p++
				pop()
			} else {
				emit(tSTR, p, indexFrom('"', p))
			}
			continue
		}
		if state == sAccTitle || state == sAccDescr {
			emit(atEnd[state], p, common.LineEnd(src, p))
			pop()
			continue
		}
		if state == sAccDescrMultiline {
			if c == '}' {
				p++
				pop()
			} else {
				emit(tAccDescrMultilineValue, p, indexFrom('}', p))
			}
			continue
		}

		// The code unit after c; after a character above the Basic Multilingual Plane it is
		// a low surrogate, which is none of the characters looked for.
		next := -1
		if size < 4 {
			next = chars.Unit(src, p+size)
		}
		if c == '%' && next == '%' {
			p = common.LineEnd(src, p+2)
			continue
		}
		// Mermaid's second comment rule takes any character before `%%` with it, a line break
		// included.
		if c != '}' && next == '%' && chars.Unit(src, p+size+1) == '%' {
			p = common.LineEnd(src, p+size+2)
			continue
		}
		if c == '\n' || c == '\r' {
			// Only a line break that directly follows an axis or a series ends it; one after
			// spaces is skipped with them, and the state carries on to the next line.
			if (state == sAxisData || state == sData) && (c == '\n' || next == '\n') {
				if c == '\n' {
					emit(tNEWLINE, p, p+1)
				} else {
					emit(tNEWLINE, p, p+2)
				}
				pop()
				continue
			}
			e := p + 1
			for e < n && (src[e] == '\n' || src[e] == '\r') {
				e++
			}
			emit(tNEWLINE, p, e)
			continue
		}

		switch c | 32 {
		case 't':
			if keyword("title") {
				emit(tTitle, p, p+5)
				continue
			}
		case 'a':
			if common.WordAt(src, p, "acc") {
				descr := common.WordAt(src, p+3, "descr")
				if descr || common.WordAt(src, p+3, "title") {
					k := chars.SpaceRun(src, p+8)
					d := chars.Unit(src, k)
					if d == ':' {
						if descr {
							emit(tAccDescr, p, chars.SpaceRun(src, k+1))
							push(sAccDescr)
						} else {
							emit(tAccTitle, p, chars.SpaceRun(src, k+1))
							push(sAccTitle)
						}
						continue
					}
					if d == '{' && descr {
						p = chars.SpaceRun(src, k+1)
						push(sAccDescrMultiline)
						continue
					}
				}
			}
		case 'x':
			if keyword("xychart-beta") {
				emit(tXYCHART, p, p+12)
				continue
			}
			if keyword("xychart") {
				emit(tXYCHART, p, p+7)
				continue
			}
			if keyword("x-axis") {
				emit(tX_AXIS, p, p+6)
				push(sAxisData)
				continue
			}
		case 'y':
			if keyword("y-axis") {
				emit(tY_AXIS, p, p+6)
				push(sAxisData)
				continue
			}
		case 'v':
			if common.WordAt(src, p, "vertical") {
				emit(tCHART_ORIENTATION, p, p+8)
				continue
			}
		case 'h':
			if common.WordAt(src, p, "horizontal") {
				emit(tCHART_ORIENTATION, p, p+10)
				continue
			}
		case 'l':
			if keyword("line") {
				emit(tLINE, p, p+4)
				push(sData)
				continue
			}
		case 'b':
			if keyword("bar") {
				emit(tBAR, p, p+3)
				push(sData)
				continue
			}
		}

		if c == '[' {
			if state == sAxisData {
				push(sAxisBandData)
			} else if state == sData {
				push(sDataInner)
			}
			emit(tSQUARE_BRACES_START, p, p+1)
			continue
		}
		if c == ']' {
			if state == sDataInner || state == sAxisBandData {
				pop()
			}
			emit(tSQUARE_BRACES_END, p, p+1)
			continue
		}
		if state == sAxisData && c == '-' && next == '-' && chars.Unit(src, p+2) == '>' {
			emit(tARROW_DELIMITER, p, p+3)
			continue
		}
		if state == sAxisData || state == sDataInner {
			q := p
			if c == '+' || c == '-' {
				q = p + 1
			}
			d := chars.Unit(src, q)
			e := -1
			if chars.IsDigit(d) {
				e = digits(q + 1)
				if chars.Unit(src, e) == '.' && chars.IsDigit(chars.Unit(src, e+1)) {
					e = digits(e + 2)
				}
			} else if d == '.' && chars.IsDigit(chars.Unit(src, q+1)) {
				e = digits(q + 2)
			}
			if e != -1 {
				emit(tNUMBER_WITH_DECIMAL, p, e)
				continue
			}
		}
		if c == '`' && common.WordAt(src, p, brokenRule) {
			p += len(brokenRule)
			pop()
			continue
		}
		if c == '"' {
			p++
			push(sString)
			continue
		}
		if chars.IsLetter(c) {
			e := p + 1
			for e < n && chars.IsLetter(int(src[e])) {
				e++
			}
			emit(tALPHA, p, e)
			continue
		}
		if chars.IsDigit(c) {
			emit(tNUM, p, digits(p+1))
			continue
		}
		if chars.IsSpace(c) {
			p = chars.SpaceRun(src, p+size)
			continue
		}
		if c < 128 && single[c] != 0 {
			emit(int(single[c]), p, p+1)
			continue
		}
		break
	}

	if p < n {
		out.Types = append(out.Types, tERROR)
		out.Texts = append(out.Texts, "")
		out.Starts = append(out.Starts, p)
		return out
	}
	// Jison tries the rules of the current state once more against the empty rest of the
	// input.
	emit(atEnd[state], n, n)
	emit(tEND, n, n)
	return out
}
