package block

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The types of token, in the order of Pele's T.
const (
	tEND = iota
	tERROR
	tKEY
	tID_BLOCK
	tCOLUMNS
	tMD_STR
	tSTR
	tSPACE_BLOCK
	tDEFAULT
	tLINKSTYLE
	tINTERPOLATE
	tCLASSDEF
	tDEFAULT_CLASSDEF_ID
	tCLASSDEF_ID
	tCLASSDEF_STYLEOPTS
	tCLASS
	tCLASSENTITY_IDS
	tSTYLECLASS
	tSTYLE
	tSTYLE_ENTITY_IDS
	tSTYLE_DEFINITION_DATA
	tAccTitle
	tAccTitleValue
	tAccDescr
	tAccDescrValue
	tAccDescrMultilineValue
	tEnd
	tNODE_DSTART
	tNODE_DEND
	tBLOCK_ARROW_START
	tBLOCK_ARROW_END
	tDIR
	tNODE_ID
	tLINK
	tSTART_LINK
	tLINK_LABEL
	tSIZE
	tEOF
)

// tokenNames are the names of the types of token, as Mermaid's errors give them: Pele's
// TOKEN_NAMES.
var tokenNames = [...]string{
	"$end", "INVALID", "BLOCK_DIAGRAM_KEY", "id-block", "COLUMNS", "MD_STR", "STR", "SPACE_BLOCK", "DEFAULT",
	"LINKSTYLE", "INTERPOLATE", "classDef", "DEFAULT_CLASSDEF_ID", "CLASSDEF_ID", "CLASSDEF_STYLEOPTS", "class",
	"CLASSENTITY_IDS", "STYLECLASS", "style", "STYLE_ENTITY_IDS", "STYLE_DEFINITION_DATA", "acc_title",
	"acc_title_value", "acc_descr", "acc_descr_value", "acc_descr_multiline_value", "end", "NODE_DSTART", "NODE_DEND",
	"BLOCK_ARROW_START", "BLOCK_ARROW_END", "DIR", "NODE_ID", "LINK", "START_LINK", "LINK_LABEL", "SIZE", "EOF",
}

// tokens are the tokens of a text, Pele's Tokens: their types, their texts and where they
// start, as byte offsets.
type tokens struct {
	Types  []int
	Texts  []string
	Starts []int
}

// The states of the lexer, as Pele's S.
const (
	sINITIAL = iota
	sString
	sMdString
	sNODE
	sBLOCK_ARROW
	sARROW_DIR
	sLLABEL
	sCLASSDEF
	sCLASSDEFID
	sCLASS
	sCLASS_STYLE
	sSTYLE_STMNT
	sSTYLE_DEFINITION
	sAccTitle
	sAccDescr
	sAccDescrMultiline
)

// directions are the directions an arrow block may point, Pele's DIRECTIONS, in the order
// they are tried.
var directions = [...]string{"right", "left", "x", "y", "up", "down"}

// idStop are the characters that end a node id, Pele's ID_STOP: ( ) [ { } < > - : = and
// white space.
var idStop [128]bool

func init() {
	for _, c := range []int{9, 10, 11, 12, 13, 32, 40, 41, 45, 58, 60, 61, 62, 91, 123, 125} {
		idStop[c] = true
	}
}

// tokenize reads a block diagram's text into tokens, as Pele's tokenize does: with the
// rules and the states of Mermaid's block.jison, written out by hand. On text no rule
// matches it stops with an ERROR token.
func tokenize(src string) tokens {
	n := len(src)
	var out tokens
	var stack []int
	state := sINITIAL
	p := 0

	at := func(q int) int { return chars.Unit(src, q) }
	put := func(typ int, text string, end int) bool {
		out.Types = append(out.Types, typ)
		out.Texts = append(out.Texts, text)
		out.Starts = append(out.Starts, p)
		p = end
		return true
	}
	emit := func(typ, end int) bool { return put(typ, src[p:end], end) }
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
	ws := func(q int) int {
		for q < n {
			c, size := chars.UnitAt(src, q)
			if !chars.IsSpace(c) {
				break
			}
			q += size
		}
		return q
	}
	digits := func(q int) int {
		for q < n && chars.IsDigit(int(src[q])) {
			q++
		}
		return q
	}
	word := func(q int) int {
		for q < n && chars.IsWord(int(src[q])) {
			q++
		}
		return q
	}
	startsWith := func(text string, q int) bool { return strings.HasPrefix(src[min(q, n):], text) }
	keyword := func(text string) bool { return startsWith(text, p) && !chars.IsWord(at(p+len(text))) }

	quote := func() bool {
		if at(p+1) == 96 {
			push(sMdString)
			p += 2
		} else {
			push(sString)
			p++
		}
		return true
	}

	// dotted is the end of `.+-[xo>]?` at q, or -1.
	dotted := func(q int) int {
		if at(q) != 46 {
			return -1
		}
		for {
			q++
			if at(q) != 46 {
				break
			}
		}
		if at(q) != 45 {
			return -1
		}
		if c := at(q + 1); c == 120 || c == 111 || c == 62 {
			return q + 2
		}
		return q + 1
	}

	// linkEnd is the end of a complete link at q, before its trailing white space, or -1.
	linkEnd := func(q int) int {
		if c := at(q); c == 120 || c == 111 || c == 60 {
			q++
		}
		line := at(q)
		if line == 46 {
			return dotted(q)
		}
		if line != 45 && line != 61 {
			return -1
		}
		k := q + 1
		for at(k) == line {
			k++
		}
		if k-q == 1 {
			if line == 45 {
				return dotted(k)
			}
			return -1
		}
		if head := at(k); head == 120 || head == 111 || head == 62 {
			return k + 1
		}
		if k-q >= 3 {
			return k
		}
		return -1
	}

	link := func() bool {
		if e := linkEnd(p); e >= 0 {
			return emit(tLINK, ws(e))
		}
		q := p
		if at(p) == 60 {
			q = p + 1
		}
		a, b := at(q), at(q+1)
		if a == 45 && (b == 45 || b == 46) || a == 61 && b == 61 {
			push(sLLABEL)
			return emit(tSTART_LINK, ws(q+2))
		}
		return false
	}

	lineState := func(typ int) bool {
		e := strings.IndexByte(src[p:], '\n')
		if e == -1 {
			e = n
		} else {
			e += p
		}
		pop()
		return emit(typ, e)
	}

	// idList is `\w+(,\s*\w+)*`.
	idList := func(typ, next int) bool {
		k := word(p)
		if k == p {
			return false
		}
		for at(k) == 44 {
			from := ws(k + 1)
			to := word(from)
			if to == from {
				break
			}
			k = to
		}
		state = next
		return emit(typ, k)
	}

	initial := func(c int) bool {
		if chars.IsSpace(c) {
			_, size := chars.UnitAt(src, p)
			p = ws(p + size)
			return true
		}
		var q int
		switch c {
		case 98:
			if startsWith("block", p) {
				if keyword("block-beta") {
					return emit(tKEY, p+10)
				}
				if at(p+5) == 58 {
					return emit(tID_BLOCK, p+6)
				}
				if !chars.IsWord(at(p + 5)) {
					return emit(tKEY, p+5)
				}
			}
		case 99:
			if startsWith("columns", p) {
				q = ws(p + 7)
				if q > p+7 {
					if startsWith("auto", q) && !chars.IsWord(at(q+4)) {
						return put(tCOLUMNS, "-1", q+4)
					}
					if e := digits(q); e > q {
						return put(tCOLUMNS, src[q:e], e)
					}
				}
			} else if startsWith("class", p) {
				def := startsWith("Def", p+5)
				from := p + 5
				if def {
					from = p + 8
				}
				q = ws(from)
				if q > from {
					if def {
						push(sCLASSDEF)
						return emit(tCLASSDEF, q)
					}
					push(sCLASS)
					return emit(tCLASS, q)
				}
			}
		case 34:
			return quote()
		case 115:
			if startsWith("space", p) {
				if at(p+5) == 58 {
					if e := digits(p + 6); e > p+6 {
						return put(tSPACE_BLOCK, src[p+6:e], e)
					}
				}
				if !chars.IsWord(at(p + 5)) {
					return put(tSPACE_BLOCK, "1", p+5)
				}
			} else if startsWith("style", p) {
				q = ws(p + 5)
				if q > p+5 {
					push(sSTYLE_STMNT)
					return emit(tSTYLE, q)
				}
			}
		case 100:
			if keyword("default") {
				return emit(tDEFAULT, p+7)
			}
		case 108:
			if keyword("linkStyle") {
				return emit(tLINKSTYLE, p+9)
			}
		case 105:
			if keyword("interpolate") {
				return emit(tINTERPOLATE, p+11)
			}
		case 97:
			if startsWith("accTitle", p) {
				q = ws(p + 8)
				if at(q) == 58 {
					push(sAccTitle)
					return emit(tAccTitle, ws(q+1))
				}
			} else if startsWith("accDescr", p) {
				q = ws(p + 8)
				switch at(q) {
				case 58:
					push(sAccDescr)
					return emit(tAccDescr, ws(q+1))
				case 123:
					push(sAccDescrMultiline)
					p = ws(q + 1)
					return true
				}
			}
		case 101:
			if keyword("end") {
				return emit(tEnd, ws(p+3))
			}
		case 45:
			if at(p+1) == 41 {
				push(sNODE)
				return emit(tNODE_DSTART, p+2)
			}
			return link()
		case 61:
			return link()
		case 60:
			if at(p+1) == 91 {
				push(sBLOCK_ARROW)
				return emit(tBLOCK_ARROW_START, p+2)
			}
			return link()
		case 40:
			length := 1
			switch d := at(p + 1); {
			case d == 45 || d == 91:
				length = 2
			case d == 40:
				length = 2
				if at(p+2) == 40 {
					length = 3
				}
			}
			push(sNODE)
			return emit(tNODE_DSTART, p+length)
		case 41, 123:
			push(sNODE)
			if at(p+1) == c {
				return emit(tNODE_DSTART, p+2)
			}
			return emit(tNODE_DSTART, p+1)
		case 62:
			push(sNODE)
			return emit(tNODE_DSTART, p+1)
		case 91:
			push(sNODE)
			if d := at(p + 1); d == 91 || d == 124 || d == 40 || d == 92 || d == 47 {
				return emit(tNODE_DSTART, p+2)
			}
			return emit(tNODE_DSTART, p+1)
		case 58:
			if e := digits(p + 1); e > p+1 {
				return put(tSIZE, src[p+1:e], e)
			}
			return false
		}

		q = p
		for q < n {
			ch, size := chars.UnitAt(src, q)
			if ch < 128 && idStop[ch] || ch >= 128 && chars.IsSpace(ch) {
				break
			}
			q += size
		}
		return q > p && emit(tNODE_ID, q)
	}

	nodeEnd := func(c int) bool {
		d := at(p + 1)
		length := 0
		switch c {
		case 34:
			return quote()
		case 40:
			switch {
			case d == 45:
				length = 2
			case d == 40:
				length = 2
				if at(p+2) == 40 {
					length = 3
				}
			default:
				length = 1
			}
		case 41:
			switch {
			case d == 41:
				length = 2
				if at(p+2) == 41 {
					length = 3
				}
			case d == 93:
				length = 2
			default:
				length = 1
			}
		case 125:
			length = 1
			if d == 125 {
				length = 2
			}
		case 45:
			if d == 41 {
				length = 2
			}
		case 93:
			length = 1
			if d == 93 || d == 41 || d == 62 {
				length = 2
			}
		case 92, 47:
			if d == 93 {
				length = 2
			}
		}
		if length == 0 {
			return false
		}
		pop()
		return emit(tNODE_DEND, p+length)
	}

	arrowDir := func(c int) bool {
		if c == 41 {
			pop()
			pop()
			return put(tBLOCK_ARROW_END, "]>", ws(p+1))
		}
		from := p
		if c == 44 {
			from = p + 1
		}
		from = ws(from)
		for _, dir := range directions {
			if startsWith(dir, from) {
				e := ws(from + len(dir))
				if c == 44 {
					return put(tDIR, src[from:e], e)
				}
				return put(tDIR, src[p:e], e)
			}
		}
		return false
	}

	for {
		if p >= n {
			typ := -1
			switch state {
			case sINITIAL:
				typ = tEOF
			case sString:
				typ = tSTR
			case sCLASSDEFID:
				typ = tCLASSDEF_STYLEOPTS
			case sCLASS_STYLE:
				typ = tSTYLECLASS
			case sSTYLE_DEFINITION:
				typ = tSTYLE_DEFINITION_DATA
			case sAccTitle:
				typ = tAccTitleValue
			case sAccDescr:
				typ = tAccDescrValue
			case sAccDescrMultiline:
				typ = tAccDescrMultilineValue
			}
			p = n
			if typ != -1 {
				put(typ, "", n)
			}
			break
		}

		c := at(p)
		ok := true

		switch state {
		case sINITIAL:
			ok = initial(c)

		case sString:
			if c == 34 {
				pop()
				p++
			} else {
				e := strings.IndexByte(src[p:], '"')
				if e == -1 {
					e = n
				} else {
					e += p
				}
				emit(tSTR, e)
			}

		case sMdString:
			switch c {
			case 96:
				ok = at(p+1) == 34
				if ok {
					pop()
					p += 2
				}
			case 34:
				ok = false
			default:
				_, size := chars.UnitAt(src, p)
				q := p + size
				for q < n && src[q] != '`' && src[q] != '"' {
					q++
				}
				emit(tMD_STR, q)
			}

		case sNODE:
			ok = nodeEnd(c)

		case sBLOCK_ARROW:
			switch {
			case c == 34:
				quote()
			case c == 93 && at(p+1) == 62:
				q := ws(p + 2)
				ok = at(q) == 40
				if ok {
					push(sARROW_DIR)
					p = q + 1
				}
			default:
				ok = false
			}

		case sARROW_DIR:
			ok = arrowDir(c)

		case sLLABEL:
			if c == 34 {
				if at(p+1) == 96 {
					quote()
				} else {
					emit(tLINK_LABEL, p+1)
					push(sString)
				}
			} else {
				e := linkEnd(ws(p))
				ok = e >= 0
				if ok {
					pop()
					emit(tLINK, ws(e))
				}
			}

		case sCLASSDEF:
			typ := tDEFAULT_CLASSDEF_ID
			k := p
			if startsWith("DEFAULT", p) {
				k = p + 7
			}
			q := ws(k)
			if k == p || q == k {
				typ = tCLASSDEF_ID
				k = word(p)
				q = ws(k)
			}
			ok = k > p && q > k
			if ok {
				state = sCLASSDEFID
				emit(typ, q)
			}

		case sCLASSDEFID:
			lineState(tCLASSDEF_STYLEOPTS)

		case sCLASS:
			ok = idList(tCLASSENTITY_IDS, sCLASS_STYLE)

		case sCLASS_STYLE:
			lineState(tSTYLECLASS)

		case sSTYLE_STMNT:
			ok = idList(tSTYLE_ENTITY_IDS, sSTYLE_DEFINITION)

		case sSTYLE_DEFINITION:
			lineState(tSTYLE_DEFINITION_DATA)

		case sAccTitle:
			lineState(tAccTitleValue)

		case sAccDescr:
			lineState(tAccDescrValue)

		case sAccDescrMultiline:
			if c == 125 {
				pop()
				p++
			} else {
				e := strings.IndexByte(src[p:], '}')
				if e == -1 {
					e = n
				} else {
					e += p
				}
				emit(tAccDescrMultilineValue, e)
			}
		}

		if !ok {
			out.Types = append(out.Types, tERROR)
			out.Texts = append(out.Texts, "")
			out.Starts = append(out.Starts, p)
			return out
		}
	}

	out.Types = append(out.Types, tEND)
	out.Texts = append(out.Texts, "")
	out.Starts = append(out.Starts, n)
	return out
}
