package quadrant

import (
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// flags is a set of token types, as Pele's flags makes one.
func flags(tokens ...int) [tEOF + 1]bool {
	var out [tEOF + 1]bool
	for _, t := range tokens {
		out[t] = true
	}
	return out
}

// The tokens of text, of a class's name, of a style, and of the end of a statement, as
// Pele's WORD, ID, STYLE and EOL.
var (
	word = flags(tPunctuation, tAmp, tNum, tAlpha, tComma, tPlus, tEquals, tMult, tDot, tBrkt, tUnderscore,
		tUnicodeText)
	id     = flags(tAlpha, tNum, tMinus, tComma, tColon, tAmp, tBrkt, tMult, tUnicodeText)
	styleT = flags(tAlpha, tNum, tColon, tSpace, tBrkt, tMinus)
	eol    = flags(tNewline, tSemi, tEOF)
)

// textTokens is what may start a text, as Mermaid's grammar lists it in its errors.
const textTokens = "'STR', 'MD_STR', 'ALPHA', 'NUM', 'UNICODE_TEXT', 'PUNCTUATION'"

// parseError carries an error out of the parser's helpers.
type parseError struct{ err error }

// parse parses a quadrant chart into db, as Pele's parseQuadrant does. The model's setters
// are called as each statement is read, so that an error in a style stops the parse where
// it is, as Pele's thrown error does.
func parse(src string, db *db) (err error) {
	t := tokenize(src)
	types, texts := t.Types, t.Texts
	i := 0

	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(parseError)
			if !ok {
				panic(r)
			}
			err = e.err
		}
	}()
	throw := func(e error) {
		if e != nil {
			panic(parseError{e})
		}
	}
	fail := func(expected string) {
		typ := types[i]
		if typ == tError {
			throw(common.SyntaxError("quadrantChart", src, t.Starts[i], "", true))
		}
		throw(common.SyntaxError("quadrantChart", src, t.Starts[i], "Expecting "+expected+", got '"+tokenNames[typ]+"'", false))
	}
	expect := func(typ int) string {
		if types[i] != typ {
			fail("'" + tokenNames[typ] + "'")
		}
		i++
		return texts[i-1]
	}
	isText := func(typ int) bool { return typ == tStr || typ == tMdStr || word[typ] }
	text := func() Text {
		first := types[i]
		if !isText(first) {
			fail(textTokens)
		}
		s := texts[i]
		i++
		for word[types[i]] || types[i] == tSpace || types[i] == tMinus {
			s += texts[i]
			i++
		}
		return Text{Text: s, Markdown: first == tMdStr}
	}
	styles := func() []string {
		var out []string
		for {
			if !styleT[types[i]] {
				fail("'ALPHA', 'NUM', 'COLON', 'SPACE', 'BRKT', 'MINUS'")
			}
			s := texts[i]
			i++
			for styleT[types[i]] {
				s += texts[i]
				i++
			}
			out = append(out, chars.Trim(s))
			if types[i] != tComma {
				return out
			}
			i++
		}
	}

	for eol[types[i]] || types[i] == tSpace {
		i++
	}
	expect(tQuadrant)

	for types[i] != tEnd {
		for types[i] == tSpace {
			i++
		}
		switch typ := types[i]; typ {
		case tClassDef:
			i++
			expect(tSpace)
			if !id[types[i]] {
				fail("'ALPHA', 'NUM', 'UNICODE_TEXT'")
			}
			name := texts[i]
			i++
			for id[types[i]] {
				name += texts[i]
				i++
			}
			expect(tSpace)
			throw(db.addClass(name, styles()))
		case tXAxis, tYAxis:
			i++
			from := text()
			var to *Text
			if types[i] == tDelimiter {
				i++
				if isText(types[i]) {
					t := text()
					to = &t
				} else {
					from.Text += " ⟶ "
				}
			}
			if typ == tXAxis {
				db.setXAxisLeftText(from)
				if to != nil {
					db.setXAxisRightText(*to)
				}
			} else {
				db.setYAxisBottomText(from)
				if to != nil {
					db.setYAxisTopText(*to)
				}
			}
		case tQuadrant1, tQuadrant2, tQuadrant3, tQuadrant4:
			i++
			db.setQuadrantText(typ-tQuadrant1, text())
		case tTitle:
			i++
			db.setDiagramTitle(chars.Trim(expect(tTitleValue)))
		case tAccTitle:
			i++
			db.setAccTitle(chars.Trim(expect(tAccTitleValue)))
		case tAccDescr:
			i++
			db.setAccDescription(chars.Trim(expect(tAccDescrValue)))
		case tAccDescrMultilineValue:
			db.setAccDescription(chars.Trim(texts[i]))
			i++
		default:
			if isText(typ) {
				label := text()
				className := ""
				if types[i] == tClassName {
					className = texts[i]
					i++
				}
				expect(tPointStart)
				x := expect(tPointX)
				y := expect(tPointY)
				var st []string
				if styleT[types[i]] {
					st = styles()
				}
				throw(db.addPoint(label, className, x, y, st))
			}
		}
		if !eol[types[i]] {
			fail("'NEWLINE', 'SEMI', 'EOF'")
		}
		i++
	}
	return nil
}
