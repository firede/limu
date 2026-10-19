package c4

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// kinds are Mermaid's typeC4Shape for each shape keyword, from tPerson on, Pele's KINDS.
var kinds = [...]string{
	"person",
	"external_person",
	"system",
	"system_db",
	"system_queue",
	"external_system",
	"external_system_db",
	"external_system_queue",
	"container",
	"container_db",
	"container_queue",
	"external_container",
	"external_container_db",
	"external_container_queue",
	"component",
	"component_db",
	"component_queue",
	"external_component",
	"external_component_db",
	"external_component_queue",
}

var (
	relTypes  = [...]string{"rel", "birel", "rel_u", "rel_d", "rel_l", "rel_r", "rel_b", "rel"}
	nodeTypes = [...]string{"node", "nodeL", "nodeR"}
)

const (
	headers        = "'direction_tb', 'direction_bt', 'direction_rl', 'direction_lr', 'C4_CONTEXT', 'C4_CONTAINER', 'C4_COMPONENT', 'C4_DYNAMIC', 'C4_DEPLOYMENT'"
	attributeNames = "'STR', 'STR_KEY', 'ATTRIBUTE', 'ATTRIBUTE_EMPTY'"
	statementNames = "'title', 'accDescription', 'acc_title', 'acc_descr', 'acc_descr_multiline_value', 'PERSON', 'REL'"
)

// The states of the parser's loop, Pele's P.
const (
	// A statement is required.
	pItem = iota
	// Only a diagram statement may come next: the first one in a boundary, or one written
	// straight after a title.
	pDiagram
	pItemOrEnd
	pAfterOther
	pAfterDiagram
)

// isOther reports whether a token begins a statement other than a diagram's: a title or an
// accessible title or description, as Pele's isOther does.
func isOther(t int) bool {
	return t == tTitle || t == tAccDescription || t == tAccTitle || t == tAccDescr || t == tAccDescrMultilineValue
}

// isDiagram reports whether a token begins a diagram statement, a shape, a boundary, a
// relation or a style, as Pele's isDiagram does.
func isDiagram(t int) bool { return t >= tPerson && t <= tUpdateLayoutConfig }

// failure is a syntax error, carried out of the parser by a panic, as Pele throws it.
type failure struct{ err *diagram.Error }

// parseC4 parses a text into db, as Pele's parseC4 does: it accepts what the parser Jison
// generates from c4Diagram.jison accepts, and calls the model's builder in the order its
// semantic actions do. Boundaries nest through a counter, not recursion.
func parseC4(src string, db *DB) (err error) {
	defer func() {
		if r := recover(); r != nil {
			f, ok := r.(failure)
			if !ok {
				panic(r)
			}
			err = f.err
		}
	}()
	tk := tokenize(src)
	types, starts, ends := tk.types, tk.starts, tk.ends
	i := 0

	// text is the text of the token at index at, as Pele's text gives it.
	text := func(at int) string { return src[starts[at]:ends[at]] }
	// fail throws the syntax error at the token at i, as Pele's fail does.
	fail := func(expected string) {
		t := types[i]
		if t == tError {
			panic(failure{common.SyntaxError("c4", src, starts[i], "", true)})
		}
		panic(failure{common.SyntaxError("c4", src, starts[i], "Expecting "+expected+", got '"+tokenNames[t]+"'", false)})
	}
	// expect steps past a token of typ, or fails, as Pele's expect does.
	expect := func(typ int) {
		if types[i] != typ {
			fail("'" + tokenNames[typ] + "'")
		}
		i++
	}

	// attributes reads a statement's arguments, at least one, as Pele's attributes does.
	attributes := func() []attr {
		var attrs []attr
	loop:
		for {
			switch types[i] {
			case tStr:
				attrs = append(attrs, attr{value: chars.Trim(text(i))})
				i++
			case tAttributeEmpty:
				attrs = append(attrs, attr{})
				i++
			case tStrKey:
				i++
				if types[i] != tStrValue {
					fail("'STR_VALUE'")
				}
				attrs = append(attrs, attr{named: true, key: chars.Trim(text(i - 1)), value: chars.Trim(text(i))})
				i++
			default:
				break loop
			}
		}
		if len(attrs) == 0 {
			fail(attributeNames)
		}
		return attrs
	}

	// other reads a title or an accessible title or description, as Pele's other does.
	other := func(t int) {
		switch t {
		case tTitle:
			db.setTitle(afterSpace(text(i), len("title")))
			i++
		case tAccDescription:
			db.setAccDescription(afterSpace(text(i), len("accDescription")))
			i++
		case tAccDescrMultilineValue:
			db.setAccDescription(chars.Trim(text(i)))
			i++
		default:
			i++
			if t == tAccTitle {
				if types[i] != tAccTitleValue {
					fail("'acc_title_value'")
				}
				// The grammar hands `accTitle:` to setTitle, so it becomes the visible title.
				db.setTitle(chars.Trim(text(i)))
				i++
			} else {
				if types[i] != tAccDescrValue {
					fail("'acc_descr_value'")
				}
				db.setAccDescription(chars.Trim(text(i)))
				i++
			}
		}
	}

	// diagram reads a diagram statement, and reports whether it opens a boundary, as Pele's
	// diagram does.
	diagramStatement := func(t int) bool {
		i++
		attrs := attributes()
		switch {
		case t <= tSystemExtQueue:
			db.addPersonOrSystem(kinds[t-tPerson], attrs)
		case t <= tContainerExtQueue:
			db.addContainer(kinds[t-tPerson], attrs)
		case t <= tComponentExtQueue:
			db.addComponent(kinds[t-tPerson], attrs)
		case t <= tNodeR:
			switch t {
			case tEnterpriseBoundary:
				attrs = insert(attrs, 2, attr{value: "ENTERPRISE"})
			case tSystemBoundary:
				attrs = insert(attrs, 2, attr{value: "SYSTEM"})
			case tContainerBoundary:
				attrs = insert(attrs, 2, attr{value: "CONTAINER"})
			}
			switch {
			case t >= tNode:
				db.addDeploymentNode(nodeTypes[t-tNode], attrs)
			case t == tContainerBoundary:
				db.addContainerBoundary(attrs)
			default:
				db.addPersonOrSystemBoundary(attrs, "system")
			}
			switch types[i] {
			case tLBrace:
				i++
				expect(tNewline)
			case tNewline:
				i++
				expect(tLBrace)
				if types[i] == tNewline {
					i++
				}
			default:
				fail("'NEWLINE', 'LBRACE'")
			}
			return true
		case t <= tRelIndex:
			// RelIndex takes a sequence number first, which Mermaid ignores.
			if t == tRelIndex {
				attrs = attrs[1:]
			}
			db.addRel(relTypes[t-tRel], attrs)
		case t == tUpdateElStyle:
			db.updateElStyle(attrs)
		case t == tUpdateRelStyle:
			db.updateRelStyle(attrs)
		default:
			db.updateLayoutConfig(attrs)
		}
		return false
	}

	if header := types[i]; header < tC4Context || header > tC4Deployment {
		fail(headers)
	}
	i++
	expect(tNewline)

	depth := 0
	state := pItem
	for {
		t := types[i]
		if state == pAfterOther || state == pAfterDiagram {
			if t == tNewline {
				i++
				state = pItemOrEnd
				continue
			}
			if state == pAfterOther && isDiagram(t) {
				state = pDiagram
				continue
			}
		} else if isDiagram(t) {
			if diagramStatement(t) {
				depth++
				state = pDiagram
			} else {
				state = pAfterDiagram
			}
			continue
		} else if isOther(t) && state != pDiagram {
			other(t)
			state = pAfterOther
			continue
		} else if state != pItemOrEnd {
			fail(statementNames)
		}

		if depth > 0 {
			if t != tRBrace {
				fail("'NEWLINE', 'RBRACE'")
			}
			i++
			db.popBoundaryParseStack()
			depth--
			state = pAfterDiagram
		} else {
			if t != tEOF {
				fail("'NEWLINE', 'EOF'")
			}
			i++
			expect(tEnd)
			db.C4Type = text(0)
			return nil
		}
	}
}

// insert is attrs with a put at index at, as JavaScript's splice(at, 0, a) puts it: at the end
// when attrs is shorter. Pele's diagram calls splice so.
func insert(attrs []attr, at int, a attr) []attr {
	at = min(at, len(attrs))
	out := make([]attr, 0, len(attrs)+1)
	out = append(out, attrs[:at]...)
	out = append(out, a)
	return append(out, attrs[at:]...)
}

// afterSpace is s past a word of n bytes and the one white space character after it, as
// substring(n + 1) takes them: \s matches characters of one code unit only. It is the
// substring that Pele's other takes of a title and an accDescription.
func afterSpace(s string, n int) string {
	_, size := chars.UnitAt(s, n)
	return s[n+size:]
}
