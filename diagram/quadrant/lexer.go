package quadrant

import (
	"strings"

	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The types of token, in the order of Pele's T.
const (
	tEnd = iota
	tError
	tNewline
	tTitle
	tTitleValue
	tAccTitle
	tAccTitleValue
	tAccDescr
	tAccDescrValue
	tAccDescrMultilineValue
	tXAxis
	tYAxis
	tDelimiter
	tQuadrant1
	tQuadrant2
	tQuadrant3
	tQuadrant4
	tClassDef
	tMdStr
	tStr
	tClassName
	tPointStart
	tPointX
	tPointY
	tQuadrant
	tAlpha
	tUnicodeText
	tColon
	tPlus
	tComma
	tEquals
	tMult
	tBrkt
	tUnderscore
	tDot
	tAmp
	tMinus
	tNum
	tSpace
	tSemi
	tPunctuation
	tEOF
)

// tokenNames are the names Pele's TOKEN_NAMES gives the types, as its errors write them.
var tokenNames = []string{
	"$end",
	"INVALID",
	"NEWLINE",
	"title",
	"title_value",
	"acc_title",
	"acc_title_value",
	"acc_descr",
	"acc_descr_value",
	"acc_descr_multiline_value",
	"X-AXIS",
	"Y-AXIS",
	"AXIS-TEXT-DELIMITER",
	"QUADRANT_1",
	"QUADRANT_2",
	"QUADRANT_3",
	"QUADRANT_4",
	"CLASSDEF",
	"MD_STR",
	"STR",
	"class_name",
	"point_start",
	"point_x",
	"point_y",
	"QUADRANT",
	"ALPHA",
	"UNICODE_TEXT",
	"COLON",
	"PLUS",
	"COMMA",
	"EQUALS",
	"MULT",
	"BRKT",
	"UNDERSCORE",
	"DOT",
	"AMP",
	"MINUS",
	"NUM",
	"SPACE",
	"SEMI",
	"PUNCTUATION",
	"EOF",
}

// The states of the lexer, as Pele's S.
const (
	sInitial = iota
	sTitle
	sAccTitle
	sAccDescr
	sAccDescrMultiline
	sString
	sMdString
	sClassName
	sPointStart
	sPointX
	sPointY
)

// atEnd is the token each state gives for the empty rest of the input, where it gives one,
// as Pele's AT_END.
var atEnd = [...]int{tEOF, tTitleValue, tAccTitleValue, tAccDescrValue, tAccDescrMultilineValue, tStr}

// single is the type of the tokens of one ASCII character, as Pele's SINGLE.
var single [128]uint8

func init() {
	for chars, typ := range map[string]int{
		":": tColon, "+": tPlus, ",": tComma, "=": tEquals, "*": tMult, "#": tBrkt, "_": tUnderscore,
		".": tDot, "&": tAmp, "-": tMinus, " \t\v\f": tSpace, ";": tSemi, "!$%'`?\\/": tPunctuation,
	} {
		for i := 0; i < len(chars); i++ {
			single[chars[i]] = uint8(typ)
		}
	}
}

// tokenize lexes a quadrant chart, as Pele's tokenize does. It reads by code units where
// Pele does with charCodeAt: chars.UnitAt reads the one JavaScript would at a byte, and a
// character above the Basic Multilingual Plane is followed by its low surrogate, which is no
// character the lexer looks for.
func tokenize(src string) *common.JisonTokens {
	n := len(src)
	out := &common.JisonTokens{}
	var stack []int
	state := sInitial
	p := 0
	// The last run scanned of each kind, so that a long run is not scanned again from each
	// of its characters.
	var spaceFrom, spaceTo, dashFrom, dashTo, wsFrom, wsTo, gapFrom, gapTo int

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
	byteAt := func(q int) int {
		if q < 0 || q >= n {
			return -1
		}
		return int(src[q])
	}
	lineEnd := func(q int) int { return common.LineEnd(src, q) }
	spaces := func(q int) int {
		for byteAt(q) == ' ' {
			q++
		}
		return q
	}
	run := func(q int) int { return chars.SpaceRun(src, q) }
	// Mermaid's `(1)|(0(.\d+)?)`, where the dot is any character within a line.
	coordinate := func(q int) int {
		c := byteAt(q)
		if c == '1' {
			return q + 1
		}
		if c != '0' {
			return -1
		}
		e := q + 1
		if u, size := chars.UnitAt(src, e); e < n && !chars.IsLineEnd(u) && size < 4 && chars.IsDigit(byteAt(e+size)) {
			e += size + 1
			for chars.IsDigit(byteAt(e)) {
				e++
			}
		}
		return e
	}

lexing:
	for p < n {
		c, size := chars.UnitAt(src, p)
		// The code unit after c; after a character above the Basic Multilingual Plane, its
		// low surrogate, which no rule looks for.
		next := -2
		if size < 4 {
			next = chars.Unit(src, p+size)
		}

		switch state {
		case sTitle, sAccTitle, sAccDescr:
			emit(atEnd[state], p, lineEnd(p))
			pop()
			continue

		case sAccDescrMultiline:
			if c == '}' {
				p++
				pop()
			} else {
				e := n
				if at := strings.IndexByte(src[p:], '}'); at >= 0 {
					e = p + at
				}
				emit(tAccDescrMultilineValue, p, e)
			}
			continue

		case sString:
			if c == '"' {
				p++
				pop()
			} else {
				e := n
				if at := strings.IndexByte(src[p:], '"'); at >= 0 {
					e = p + at
				}
				emit(tStr, p, e)
			}
			continue

		case sMdString:
			if c == '`' && next == '"' {
				p += 2
				pop()
				continue
			}
			if c != '`' && c != '"' {
				e := n
				if at := strings.IndexAny(src[p+size:], "`\""); at >= 0 {
					e = p + size + at
				}
				emit(tMdStr, p, e)
				continue
			}
			break lexing

		case sClassName:
			if chars.IsWord(c) {
				e := p + 1
				for chars.IsWord(byteAt(e)) {
					e++
				}
				emit(tClassName, p, e)
				pop()
				continue
			}
			break lexing

		case sPointStart:
			if e := coordinate(p); e != -1 {
				emit(tPointX, p, e)
				push(sPointX)
				continue
			}
			if k := run(p); byteAt(k) == ']' {
				p = spaces(k + 1)
				pop()
				continue
			}
			break lexing

		case sPointX:
			if k := run(p); byteAt(k) == ',' {
				p = run(k + 1)
				state = sPointY
				continue
			}
			break lexing

		case sPointY:
			if e := coordinate(p); e != -1 {
				emit(tPointY, p, e)
				pop()
				continue
			}
			break lexing
		}

		if c == '%' && next == '%' {
			p = lineEnd(p + 2)
			continue
		}
		if c == '\n' || c == '\r' {
			e := p + 1
			for d := byteAt(e); d == '\n' || d == '\r'; d = byteAt(e) {
				e++
			}
			emit(tNewline, p, e)
			continue
		}

		lower := c | 32
		if lower == 't' && common.WordAt(src, p, "title") && !chars.IsWord(byteAt(p+5)) {
			emit(tTitle, p, p+5)
			push(sTitle)
			continue
		}
		if lower == 'a' && common.WordAt(src, p, "acc") {
			descr := common.WordAt(src, p+3, "descr")
			if descr || common.WordAt(src, p+3, "title") {
				k := run(p + 8)
				d := byteAt(k)
				if d == ':' {
					if descr {
						emit(tAccDescr, p, run(k+1))
						push(sAccDescr)
					} else {
						emit(tAccTitle, p, run(k+1))
						push(sAccTitle)
					}
					continue
				}
				if d == '{' && descr {
					p = run(k + 1)
					push(sAccDescrMultiline)
					continue
				}
			}
		}

		// Axis, quadrant and chart keywords and the axis arrow take the spaces around them.
		j := p
		if c == ' ' {
			if p < spaceFrom || p >= spaceTo {
				spaceFrom = p
				spaceTo = spaces(p)
			}
			j = spaceTo
		}
		d := byteAt(j)
		dl := d | 32
		typ, end := 0, 0
		switch {
		case dl == 'x' && common.WordAt(src, j, "x-axis"):
			typ, end = tXAxis, j+6
		case dl == 'y' && common.WordAt(src, j, "y-axis"):
			typ, end = tYAxis, j+6
		case d == '-':
			if j < dashFrom || j >= dashTo {
				dashFrom = j
				dashTo = j
				for byteAt(dashTo) == '-' {
					dashTo++
				}
			}
			if dashTo-j >= 2 && byteAt(dashTo) == '>' {
				typ, end = tDelimiter, dashTo+1
			}
		case dl == 'q' && common.WordAt(src, j, "quadrant"):
			if q := byteAt(j + 9); byteAt(j+8) == '-' && q >= '1' && q <= '4' {
				typ, end = tQuadrant1+q-'1', j+10
			} else if common.WordAt(src, j+8, "chart") {
				typ, end = tQuadrant, j+13
			}
		}
		if typ != 0 {
			emit(typ, p, spaces(end))
			continue
		}

		if lower == 'c' && common.WordAt(src, p, "classdef") && !chars.IsWord(byteAt(p+8)) {
			emit(tClassDef, p, p+8)
			continue
		}
		if c == '"' {
			if next == '`' {
				p += 2
				push(sMdString)
			} else {
				p++
				push(sString)
			}
			continue
		}
		if c == ':' && next == ':' && byteAt(p+2) == ':' {
			p += 3
			push(sClassName)
			continue
		}
		// A point's `: [`, with any white space before, between and after, line breaks
		// included.
		if c == ':' || chars.IsSpace(c) {
			k := p
			if c != ':' {
				if p < wsFrom || p >= wsTo {
					wsFrom = p
					wsTo = run(p)
				}
				k = wsTo
			}
			if byteAt(k) == ':' {
				if k+1 < gapFrom || k+1 >= gapTo {
					gapFrom = k + 1
					gapTo = run(k + 1)
				}
				if byteAt(gapTo) == '[' {
					emit(tPointStart, p, run(gapTo+1))
					push(sPointStart)
					continue
				}
			}
		}

		if chars.IsLetter(c) {
			e := p + 1
			for chars.IsLetter(byteAt(e)) {
				e++
			}
			emit(tAlpha, p, e)
			continue
		}
		if c >= 128 {
			e := p + 1
			for e < n && src[e] >= 0x80 {
				e++
			}
			emit(tUnicodeText, p, e)
			continue
		}
		if chars.IsDigit(c) {
			e := p + 1
			for chars.IsDigit(byteAt(e)) {
				e++
			}
			emit(tNum, p, e)
			continue
		}
		if single[c] != 0 {
			emit(int(single[c]), p, p+1)
			continue
		}
		break
	}

	if p < n {
		out.Types = append(out.Types, tError)
		out.Texts = append(out.Texts, "")
		out.Starts = append(out.Starts, p)
		return out
	}
	// Jison tries the rules of the current state once more against the empty rest of the
	// input.
	if state < len(atEnd) {
		emit(atEnd[state], n, n)
	}
	emit(tEnd, n, n)
	return out
}
