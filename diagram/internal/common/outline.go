package common

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// The indented outline grammar that Mermaid's mindmap and kanban diagrams share, as Pele's
// diagrams/common/outline.ts has it. Kanban adds `@{ ... }` metadata after a node and keeps
// `@` out of node ids.

// The outline's tokens, Pele's T.
const (
	TokEnd = iota
	TokError
	TokSpaceLine
	TokNL
	TokEOF
	TokKeyword
	TokSpaceList
	TokIcon
	TokClass
	TokNodeDStart
	TokNodeID
	TokNodeDescr
	TokNodeDEnd
	TokShapeData
)

// tokenNames are the tokens as the generated parser prints them. It has no name for the end
// of input and prints its number.
var tokenNames = [...]string{
	"1", "INVALID", "SPACELINE", "NL", "EOF", "", "SPACELIST", "ICON", "CLASS", "NODE_DSTART",
	"NODE_ID", "NODE_DESCR", "NODE_DEND", "SHAPE_DATA",
}

// outlineTokenName is a token as the generated parser prints it, as Pele's tokenName does.
func outlineTokenName(typ int, kanban bool) string {
	if typ == TokKeyword {
		if kanban {
			return "KANBAN"
		}
		return "MINDMAP"
	}
	return tokenNames[typ]
}

// OutlineTokens are an outline's tokens, Pele's Tokens: their types, their texts, and the byte
// offsets they start at.
type OutlineTokens struct {
	Types  []int
	Texts  []string
	Starts []int
}

// The shapes of a node, Pele's NODE_TYPE.
const (
	NodeDefault     = 0
	NodeRoundedRect = 1
	NodeRect        = 2
	NodeCircle      = 3
	NodeCloud       = 4
	NodeBang        = 5
	NodeHexagon     = 6
)

// OutlineType is the shape a node's delimiters give it, as Pele's getType does: any opening
// delimiter pairs with any closing one, as in Mermaid.
func OutlineType(start, end string) int {
	switch start {
	case "[":
		return NodeRect
	case "(":
		if end == ")" {
			return NodeRoundedRect
		}
		return NodeCloud
	case "((":
		return NodeCircle
	case ")":
		return NodeCloud
	case "))":
		return NodeBang
	case "{{":
		return NodeHexagon
	}
	return NodeDefault
}

// OutlineDB is what the outline's parser builds, Pele's OutlineDb. shapeData is the text of a
// metadata block, if hasData.
type OutlineDB interface {
	GetType(start, end string) int
	AddNode(level int, id, descr string, typ int, shapeData string, hasData bool) error
	// DecorateNode is given an icon or, when class is set, a class.
	DecorateNode(text string, class bool)
}

// The lexer's states, Pele's S.
const (
	sInitial = iota
	sClass
	sIcon
	sNode
	sNStr
	sNStr2
	sData
	sDataStr
)

// letters reports whether word, in lowercase ASCII, is at byte p of src, its letters in either
// case: Jison's case-insensitive flag folds ASCII letters only.
func letters(src string, p int, word string) bool {
	for k := 0; k < len(word); k++ {
		if p+k >= len(src) || src[p+k]|32 != word[k] {
			return false
		}
	}
	return true
}

// unit is the code unit at byte i of src, -1 past its end.
func unit(src string, i int) int { return chars.Unit(src, i) }

// lineRun is the end of /.+/y at byte i: the run of characters that do not end a line, or -1
// where there is none.
func lineRun(src string, i int) int {
	j := i
	for j < len(src) {
		c, size := chars.UnitAt(src, j)
		if chars.IsLineEnd(c) {
			break
		}
		j += size
	}
	if j == i {
		return -1
	}
	return j
}

// runNot is the end of a regular expression's run of bytes none of which is in set, starting
// at byte i, or -1 where there is none. Every byte in set is ASCII, so the run steps over
// whole characters.
func runNot(src string, i int, set string) int {
	j := i
	for j < len(src) && strings.IndexByte(set, src[j]) < 0 {
		j++
	}
	if j == i {
		return -1
	}
	return j
}

// TokenizeOutline reads an outline's tokens, as Pele's tokenize does.
func TokenizeOutline(src string, kanban bool) OutlineTokens {
	n := len(src)
	var t OutlineTokens
	keyword := "mindmap"
	idSet := "([\n){}"
	if kanban {
		keyword = "kanban"
		idSet = "([\n){}@"
	}
	state := sInitial
	p := 0
	nextBreak := -1

	put := func(typ int, text string, end int) {
		t.Types = append(t.Types, typ)
		t.Texts = append(t.Texts, text)
		t.Starts = append(t.Starts, p)
		p = end
	}
	emit := func(typ, end, next int) {
		put(typ, src[p:end], end)
		state = next
	}
	upTo := func(ch byte) int {
		if e := strings.IndexByte(src[p:], ch); e >= 0 {
			return p + e
		}
		return n
	}

loop:
	for p < n {
		c := unit(src, p)
		d := unit(src, p+1)
		var e int
		switch state {
		case sInitial:
			w := p
			if c <= 32 || c >= 160 {
				w = chars.SpaceRun(src, p)
			}
			switch {
			case kanban && c == '@' && d == '{':
				put(TokShapeData, "", p+2)
				state = sData
			case unit(src, w) == '%' && unit(src, w+1) == '%':
				emit(TokSpaceLine, lineRun(src, w), sInitial)
			case w > p:
				if nextBreak < p {
					nextBreak = upTo('\n')
				}
				if nextBreak >= w {
					emit(TokSpaceList, w, sInitial)
				} else if e = strings.LastIndexByte(src[:w], '\n'); e > p {
					emit(TokSpaceLine, e+1, sInitial)
				} else {
					emit(TokNL, p+1, sInitial)
				}
			case letters(src, p, keyword) && !chars.IsWord(unit(src, p+len(keyword))):
				emit(TokKeyword, p+len(keyword), sInitial)
			case c == ':' && d == ':' && unit(src, p+2) == ':':
				p += 3
				state = sClass
			case c == ':' && d == ':' && letters(src, p+2, "icon") && unit(src, p+6) == '(':
				p += 7
				state = sIcon
			case c == '-' && d == ')' || c == '(' && (d == '-' || d == '(') || (c == ')' || c == '{') && d == c:
				emit(TokNodeDStart, p+2, sNode)
			case c == ')' || c == '(' || c == '[':
				emit(TokNodeDStart, p+1, sNode)
			default:
				if e = runNot(src, p, idSet); e != -1 {
					emit(TokNodeID, e, sInitial)
				} else {
					break loop
				}
			}
		case sNode:
			switch {
			case c == '"':
				if d == '`' {
					state = sNStr2
					p += 2
				} else {
					state = sNStr
					p++
				}
			case c == ')' || c == '(':
				end := p + 1
				if d == c || c == '(' && d == '-' {
					end = p + 2
				}
				emit(TokNodeDEnd, end, sInitial)
			case c == ']':
				emit(TokNodeDEnd, p+1, sInitial)
			case c == '}' && d == '}' || c == '-' && d == ')':
				emit(TokNodeDEnd, p+2, sInitial)
			case c == '}':
				// A lone closing brace takes the rest of its line as text.
				emit(TokNodeDescr, lineRun(src, p), sNode)
			default:
				emit(TokNodeDescr, runNot(src, p, ")](}"), sNode)
			}
		case sNStr:
			if c == '"' {
				p++
				state = sNode
			} else {
				emit(TokNodeDescr, upTo('"'), sNStr)
			}
		case sNStr2:
			if c == '`' && d == '"' {
				p += 2
				state = sNode
			} else if e = runNot(src, p, "`\""); e != -1 {
				emit(TokNodeDescr, e, sNStr2)
			} else {
				break loop
			}
		case sClass:
			if e = lineRun(src, p); e != -1 {
				emit(TokClass, e, sInitial)
			} else if c == '\n' {
				p++
				state = sInitial
			} else {
				break loop
			}
		case sIcon:
			if c == ')' {
				p++
				state = sInitial
			} else {
				emit(TokIcon, upTo(')'), sIcon)
			}
		case sData:
			if c == '"' {
				emit(TokShapeData, p+1, sDataStr)
			} else if c == '}' {
				p++
				state = sInitial
			} else if e = runNot(src, p, "}^\""); e != -1 {
				emit(TokShapeData, e, sData)
			} else {
				break loop
			}
		default:
			if c == '"' {
				emit(TokShapeData, p+1, sData)
			} else {
				e = upTo('"')
				put(TokShapeData, breaks(src[p:e]), e)
			}
		}
	}

	if p < n {
		put(TokError, "", p)
	} else {
		if state == sInitial {
			put(TokEOF, "", n)
		}
		put(TokEnd, "", n)
	}
	return t
}

// breaks writes each line feed of s, with the white space after it, as <br/>, as Pele's
// replace of /\n\s*/g does.
func breaks(s string) string {
	if !strings.Contains(s, "\n") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\n' {
			b.WriteString("<br/>")
			i = chars.SpaceRun(s, i+1)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

func isStop(typ int) bool { return typ == TokSpaceLine || typ == TokNL || typ == TokEOF }

// startsStatement are the tokens a statement may start with, Pele's START.
func startsStatement(typ int) bool {
	switch typ {
	case TokSpaceLine, TokSpaceList, TokIcon, TokClass, TokNodeDStart, TokNodeID:
		return true
	}
	return false
}

const stopNames = "'SPACELINE', 'NL', 'EOF'"

// ParseOutline makes the calls on db that the grammar's actions make, in the same order, and
// reports errors in the state where the generated parser would, as Pele's parseOutline does.
func ParseOutline(src string, db OutlineDB, kanban bool) error {
	typ := diagram.Mindmap
	data := ""
	if kanban {
		typ = diagram.Kanban
		data = ", 'SHAPE_DATA'"
	}
	tok := TokenizeOutline(src, kanban)
	types, texts, starts := tok.Types, tok.Texts, tok.Starts
	i := 0

	fail := func(expected string) error {
		t := types[i]
		if t == TokError {
			return diagram.SyntaxError(typ, src, starts[i], "", true)
		}
		return diagram.SyntaxError(typ, src, starts[i], "Expecting "+expected+", got '"+outlineTokenName(t, kanban)+"'", false)
	}

	keyword := "'" + outlineTokenName(TokKeyword, kanban) + "'"
	if types[i] == TokSpaceLine {
		i++
		for types[i] == TokSpaceLine || types[i] == TokNL {
			i++
		}
		if types[i] != TokKeyword {
			return fail("'SPACELINE', 'NL', " + keyword)
		}
	} else if types[i] != TokKeyword {
		return fail("'SPACELINE', " + keyword)
	}
	i++
	first := "'SPACELINE', 'NL', 'SPACELIST', 'ICON', 'CLASS', 'NODE_DSTART', 'NODE_ID'"
	if types[i] == TokNL {
		i++
		first = "'SPACELINE', 'SPACELIST', 'ICON', 'CLASS', 'NODE_DSTART', 'NODE_ID'"
	}

	for {
		t := types[i]
		level := 0
		indented := t == TokSpaceList
		if indented {
			level = chars.Len16(texts[i])
			i++
			t = types[i]
		}
		switch {
		case t == TokNodeID || t == TokNodeDStart:
			var id, descr string
			hasID := false
			typ := 0
			if t == TokNodeID {
				id, hasID = texts[i], true
				i++
			}
			if types[i] == TokNodeDStart {
				start := texts[i]
				i++
				if types[i] != TokNodeDescr {
					return fail("'NODE_DESCR'")
				}
				descr = texts[i]
				i++
				if types[i] != TokNodeDEnd {
					return fail("'NODE_DEND'")
				}
				end := texts[i]
				i++
				if !isStop(types[i]) && types[i] != TokShapeData {
					return fail(stopNames + data)
				}
				typ = db.GetType(start, end)
			} else {
				if !isStop(types[i]) && types[i] != TokShapeData {
					return fail(stopNames + ", 'NODE_DSTART'" + data)
				}
				descr = id
			}
			if !hasID {
				id = descr
			}
			if types[i] == TokShapeData {
				shapeData := texts[i]
				i++
				for types[i] == TokShapeData {
					shapeData += texts[i]
					i++
				}
				if !isStop(types[i]) {
					return fail(stopNames + data)
				}
				if err := db.AddNode(level, id, descr, typ, shapeData, true); err != nil {
					return err
				}
			} else if err := db.AddNode(level, id, descr, typ, "", false); err != nil {
				return err
			}
		case t == TokIcon || t == TokClass:
			text := texts[i]
			i++
			if !isStop(types[i]) {
				return fail(stopNames)
			}
			db.DecorateNode(text, t == TokClass)
		case indented:
			if !isStop(t) {
				return fail(stopNames + ", 'ICON', 'CLASS', 'NODE_DSTART', 'NODE_ID'")
			}
		case t == TokSpaceLine:
			i++
			if !isStop(types[i]) {
				return fail(stopNames)
			}
		default:
			return fail(first)
		}

		i++
		for types[i] == TokNL || types[i] == TokEOF {
			i++
		}
		if types[i] == TokEnd {
			return nil
		}
		if !startsStatement(types[i]) {
			return fail(stopNames + ", 'SPACELIST', 'ICON', 'CLASS', 'NODE_DSTART', 'NODE_ID'")
		}
	}
}
