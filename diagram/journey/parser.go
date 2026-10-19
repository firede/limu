package journey

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// statement is what may start a statement, as Mermaid's grammar lists it in its errors.
const statement = "'EOF', 'SPACE', 'NEWLINE', 'title', 'acc_title', 'acc_descr', 'acc_descr_multiline_value', 'section', 'taskName'"

// parse parses a journey diagram into db, as Pele's parseJourney does.
func parse(src string, db *db) error {
	t := tokenize(src)
	fail := func(i int, expected string) error {
		got := "1"
		if t.types[i] != tEnd {
			got = tokenNames[t.types[i]]
		}
		return diagram.SyntaxError("journey", src, t.starts[i], "Expecting "+expected+", got '"+got+"'", false)
	}

	if t.types[0] != tJourney {
		return fail(0, "'journey'")
	}
	for i := 1; ; i++ {
		text := t.texts[i]
		switch t.types[i] {
		case tNewline:
		case tTitle:
			db.setDiagramTitle(chars.DropUnits(text, 6))
		case tAccTitle:
			i++
			db.setAccTitle(chars.Trim(t.texts[i]))
		case tAccDescr:
			i++
			db.setAccDescription(chars.Trim(t.texts[i]))
		case tAccDescrMultilineValue:
			db.setAccDescription(chars.Trim(text))
		case tSection:
			db.addSection(chars.DropUnits(text, 8))
		case tTaskName:
			i++
			if t.types[i] != tTaskData {
				return fail(i, "'taskData'")
			}
			db.addTask(text, t.texts[i])
		case tEOF:
			return nil
		default:
			return fail(i, statement)
		}
	}
}
