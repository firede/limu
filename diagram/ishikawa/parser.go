package ishikawa

import (
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

const stopNames = "'SPACELINE', 'NL', 'EOF'"

// parseIshikawa follows the parser Jison builds from Mermaid's grammar, as Pele's
// parseIshikawa does: blank lines and comments may come first, then the keyword and at most
// one line break, then statements. A statement is a line of text, with or without
// indentation, or a blank line; each is ended by a blank line, a line break or the end of the
// text, and any number of line breaks after that.
func parseIshikawa(src string, db *db) error {
	t := tokenize(src)
	types, starts, ends := t.types, t.starts, t.ends
	i := 0

	fail := func(expected string) error {
		return common.SyntaxError("ishikawa", src, starts[i], "Expecting "+expected+", got '"+tokenNames[types[i]]+"'", false)
	}
	stops := func(typ int) bool { return typ == tokSpaceLine || typ == tokNL || typ == tokEOF }

	if types[i] == tokSpaceLine {
		i++
		for types[i] == tokSpaceLine || types[i] == tokNL {
			i++
		}
		if types[i] != tokIshikawa {
			return fail("'SPACELINE', 'NL', 'ISHIKAWA'")
		}
	} else if types[i] != tokIshikawa {
		return fail("'SPACELINE', 'ISHIKAWA'")
	}
	i++
	if types[i] == tokNL {
		i++
	}

	for {
		typ := types[i]
		switch {
		case typ == tokText || typ == tokSpaceList && types[i+1] == tokText:
			text := i
			if typ != tokText {
				text = i + 1
			}
			i = text + 1
			if !stops(types[i]) {
				return fail(stopNames)
			}
			level := 0
			if typ != tokText {
				level = chars.Len16(src[starts[text-1]:ends[text-1]])
			}
			db.addNode(level, chars.Trim(src[starts[text]:ends[text]]))
		case typ == tokSpaceLine || typ == tokSpaceList:
			i++
			if !stops(types[i]) {
				if typ == tokSpaceList {
					return fail(stopNames + ", 'TEXT'")
				}
				return fail(stopNames)
			}
		default:
			return fail("'SPACELINE', 'SPACELIST', 'TEXT'")
		}
		i++
		for types[i] == tokNL || types[i] == tokEOF {
			i++
		}
		if types[i] == tokEnd {
			return nil
		}
	}
}
