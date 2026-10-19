package venn

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/diagram"
)

// The tokens a statement and a style's value may start with, as Mermaid's grammar lists them
// in its errors.
const (
	statementTokens = "'EOF', 'NEWLINE', 'TITLE', 'SET', 'UNION', 'TEXT', 'INDENT_TEXT', 'STYLE'"
	valueTokens     = "'NUMERIC', 'IDENTIFIER', 'STRING', 'HEXCOLOR', 'RGBCOLOR', 'RGBACOLOR'"
)

// parseError carries an error out of the parser's helpers.
type parseError struct{ err error }

// parseVenn parses a venn diagram into db, as Pele's parseVenn ports Mermaid's venn grammar. A
// token is read only when the grammar has to look at one to decide, because the lexer asks
// the model whether an indented `text` continues a set, and the statements are what tell the
// model so.
func parseVenn(src string, db *db) (err error) {
	lex := newLexer(src, db)
	typ := -1

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
	peek := func() int {
		if typ == -1 {
			typ = lex.next()
		}
		return typ
	}
	fail := func(expected string) {
		if peek() == tError {
			throw(diagram.SyntaxError("venn", src, lex.start, "", true))
		}
		throw(diagram.SyntaxError("venn", src, lex.start, "Expecting "+expected+", got '"+tokenNames[typ]+"'", false))
	}
	take := func() string {
		typ = -1
		return lex.text
	}
	expect := func(wanted int) string {
		if peek() != wanted {
			fail("'" + tokenNames[wanted] + "'")
		}
		return take()
	}
	identifier := func() string {
		if peek() != tIdentifier && typ != tString {
			fail("'IDENTIFIER', 'STRING'")
		}
		return take()
	}
	identifiers := func() []string {
		list := []string{identifier()}
		for peek() == tComma {
			take()
			list = append(list, identifier())
		}
		return list
	}

	subset := func(ids []string, union bool) {
		var label *string
		if peek() == tBracketLabel {
			s := take()
			label = &s
		}
		var size *float64
		if peek() == tColon {
			take()
			f := parseFloat(expect(tNumeric))
			size = &f
		}
		if union {
			if len(ids) < 2 {
				throw(diagram.NewError("union requires multiple identifiers", diagram.Semantic, "venn"))
			}
			throw(db.validateUnionIdentifiers(ids))
		}
		db.addSubsetData(ids, label, size)
		db.setIndentMode(true)
	}

	// The id of a text, and the label in brackets that an id which is not a number may carry.
	text := func(sets func() []string) {
		kind := peek()
		if kind != tIdentifier && kind != tString && kind != tNumeric {
			fail("'NUMERIC', 'IDENTIFIER', 'STRING'")
		}
		id := take()
		var label *string
		if kind != tNumeric && peek() == tBracketLabel {
			s := take()
			label = &s
		}
		db.addTextData(sets(), id, label)
	}

	styleStatement := func() {
		ids := identifiers()
		var styles []Declaration
		for {
			key := expect(tIdentifier)
			expect(tColon)
			var value string
			if peek() == tString {
				value = take()
			} else {
				var parts []string
				for peek() == tIdentifier || typ >= tNumeric && typ <= tRGBAColor {
					parts = append(parts, take())
				}
				if len(parts) == 0 {
					fail(valueTokens)
				}
				value = strings.Join(parts, " ")
			}
			styles = append(styles, Declaration{key, value})
			if peek() != tComma {
				break
			}
			take()
		}
		db.addStyleData(ids, styles)
	}

	for peek() == tNewline {
		take()
	}
	expect(tVenn)
	for {
		statement := peek()
		if statement == tEOF {
			return nil
		}
		if statement < tNewline || statement > tStyle || statement == tVenn {
			fail(statementTokens)
		}
		take()
		switch statement {
		case tTitle:
			// The token without `title` and the one white space after it.
			_, size := utf8.DecodeRuneInString(lex.text[5:])
			db.setDiagramTitle(lex.text[5+size:])
		case tSet:
			subset([]string{identifier()}, false)
		case tUnion:
			subset(identifiers(), true)
		case tText:
			ids := identifiers()
			text(func() []string { return ids })
		case tIndentText:
			text(func() []string {
				current := db.getCurrentSets()
				if current == nil {
					throw(diagram.NewError("text requires set", diagram.Semantic, "venn"))
				}
				return current
			})
		case tStyle:
			styleStatement()
		}
	}
}

// parseFloat is JavaScript's parseFloat of a NUMERIC token: an optional sign, digits and a
// fraction, which Go reads to the same number, out to ±Infinity.
func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
