package railroad

import "github.com/firede/limu/diagram/internal/common"

// railroad-beta: Mermaid's own notation, one constructor call per node, as Pele's
// railroad/ir.ts reads it.

// The kinds of token, by index in irTokens, as Pele's T.
const (
	irKeyword = iota
	irNonterminal
	irZeroOrMore
	irOneOrMore
	irSequence
	irOptional
	irTerminal
	irSpecial
	irChoice
	irEquals
	irSemicolon
	irOpen
	irComma
	irClose
	irTitle
	irAccTitle
	irAccDescr
	irID
	irString
)

// irWord is a word of the notation, which a longer name wins over.
func irWord(name string) common.TokenType { return literal(name, irID) }

// irTokens is Pele's IR_TOKENS.
var irTokens = append(append([]common.TokenType{
	opening("railroad-beta", irID),
	irWord("nonterminal"),
	irWord("zeroOrMore"),
	irWord("oneOrMore"),
	irWord("sequence"),
	irWord("optional"),
	irWord("terminal"),
	irWord("special"),
	irWord("choice"),
	literal("="),
	literal(";"),
	literal("("),
	literal(","),
	literal(")"),
	title(wordOrDash),
	accTitleToken,
	accDescrToken,
	identifier("RR_ID"),
	quoted("RR_STRING"),
}, hiddenTokens("RR")...), blockComment("RR_BLOCK_COMMENT", "/*", "*/"))

var irTable = common.Prepare(irTokens)

// irExpression is Pele's IrExpression: Type is its $type; Value a terminal's value, a
// non-terminal's name or a special's text; List a sequence's elements or a choice's
// alternatives; and Element the element of the three wrappers.
type irExpression struct {
	typ     string
	value   string
	list    []*irExpression
	element *irExpression
}

// container reports whether the expression holds others, as Pele's test of elements,
// alternatives or element in it does.
func (e *irExpression) container() bool {
	switch e.typ {
	case "RailroadSequenceExpr", "RailroadChoiceExpr":
		return true
	}
	return e.wrapper()
}

func (e *irExpression) wrapper() bool {
	switch e.typ {
	case "RailroadOptionalExpr", "RailroadOneOrMoreExpr", "RailroadZeroOrMoreExpr":
		return true
	}
	return false
}

// parts are the expression's parts, as Pele's parts gives them.
func (e *irExpression) parts() []*irExpression {
	if e.wrapper() {
		return []*irExpression{e.element}
	}
	return e.list
}

type irAst = grammar[rule[*irExpression]]

// irArgument reads a constructor's one string argument, as Pele's argument does.
func irArgument(r *common.Reader) string {
	r.I++
	r.Expect(irOpen)
	value := unescape(r.Expect(irString))
	r.Expect(irClose)
	return value
}

// irExpr reads an expression, as Pele's expression does, without recursion: the constructors
// still open are on a stack.
func irExpr(r *common.Reader) *irExpression {
	var open []*irExpression
outer:
	for {
		var value *irExpression
		switch r.Kind() {
		case irTerminal:
			value = &irExpression{typ: "RailroadTerminalExpr", value: irArgument(r)}
		case irNonterminal:
			value = &irExpression{typ: "RailroadNonTerminalExpr", value: irArgument(r)}
		case irSpecial:
			value = &irExpression{typ: "RailroadSpecialExpr", value: irArgument(r)}
		case irSequence:
			value = &irExpression{typ: "RailroadSequenceExpr", list: []*irExpression{}}
		case irChoice:
			value = &irExpression{typ: "RailroadChoiceExpr", list: []*irExpression{}}
		case irOptional:
			value = &irExpression{typ: "RailroadOptionalExpr"}
		case irOneOrMore:
			value = &irExpression{typ: "RailroadOneOrMoreExpr"}
		case irZeroOrMore:
			value = &irExpression{typ: "RailroadZeroOrMoreExpr"}
		default:
			r.Fail("an expression")
		}
		if value.container() {
			r.I++
			r.Expect(irOpen)
			open = append(open, value)
			continue
		}
		for {
			if len(open) == 0 {
				return value
			}
			parent := open[len(open)-1]
			if parent.wrapper() {
				parent.element = value
			} else {
				parent.list = append(parent.list, value)
				if r.Accept(irComma) {
					continue outer
				}
			}
			r.Expect(irClose)
			open = open[:len(open)-1]
			value = parent
		}
	}
}

// parseIr parses a railroad-beta text into its tree, as Pele's parseIr does.
func parseIr(src string) (*irAst, error) {
	return parseGrammar(src, irTable, "Railroad", irTitle, unescape, func(r *common.Reader) rule[*irExpression] {
		name := r.Expect(irID)
		r.Expect(irEquals)
		definition := irExpr(r)
		r.Expect(irSemicolon)
		return rule[*irExpression]{typ: "RailroadRule", name: name, definition: definition}
	})
}

// irNode builds a node of the model from an expression and its parts, as Pele's node does.
func irNode(e *irExpression, kids []*Node) *Node {
	switch e.typ {
	case "RailroadTerminalExpr":
		return terminal(e.value)
	case "RailroadNonTerminalExpr":
		return nonterminal(e.value)
	case "RailroadSpecialExpr":
		return special(e.value)
	case "RailroadSequenceExpr":
		return sequence(kids)
	case "RailroadChoiceExpr":
		return choice(kids)
	case "RailroadOptionalExpr":
		return optional(kids[0])
	}
	least := 0.0
	if e.typ == "RailroadOneOrMoreExpr" {
		least = 1
	}
	return repetition(kids[0], least, infinite)
}

// irRules are the rules of a railroad-beta tree, as Pele's irRules gives them.
func irRules(ast *irAst) []Rule { return toRules(ast.rules, irNode) }
