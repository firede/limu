package railroad

import "github.com/firede/limu/diagram/internal/common"

// railroad-peg-beta: parsing expression grammars, as Pele's railroad/peg.ts reads them.

// The kinds of token, by index in pegTokens, as Pele's T.
const (
	pegKeyword = iota
	pegArrow
	pegSemicolon
	pegSlash
	pegAnd
	pegNot
	pegQuestion
	pegStar
	pegPlus
	pegGroup
	pegGroupEnd
	pegDot
	pegTitle
	pegAccTitle
	pegAccDescr
	pegID
	pegString
)

// pegTokens is Pele's PEG_TOKENS.
var pegTokens = append(append([]common.TokenType{
	opening("railroad-peg-beta", pegID),
	literal("<-"),
	literal(";"),
	literal("/"),
	literal("&"),
	literal("!"),
	literal("?"),
	literal("*"),
	literal("+"),
	literal("("),
	literal(")"),
	literal("."),
	titleToken,
	accTitleToken,
	accDescrToken,
	identifier("PEG_ID"),
	quoted("PEG_STRING"),
}, hiddenTokens("PEG")...),
	// /#[^\n\r]*/y
	common.TokenType{Name: "PEG_LINE_COMMENT", First: "#", Hidden: true, Match: func(s string, at int) int {
		if s[at] != '#' {
			return -1
		}
		i := at + 1
		for i < len(s) && s[i] != '\n' && s[i] != '\r' {
			i++
		}
		return i
	}},
)

var pegTable = common.Prepare(pegTokens)

// The syntax tree of Pele's peg.ts. pegPart is any node of it.
type pegPart interface{ parts() []pegPart }

// pegPrimary is Pele's PegPrimary: a literal's value, an identifier's name, the dot of any
// character, or a group's choice.
type pegPrimary struct {
	typ     string
	value   string
	element *pegChoice
}

// pegSuffix is Pele's PegSuffix, with its operator, if any.
type pegSuffix struct {
	primary  *pegPrimary
	operator string
}

// pegPrefix is Pele's PegPrefix, with its predicate, if any.
type pegPrefix struct {
	operator string
	suffix   *pegSuffix
}

type pegSequence struct{ elements []*pegPrefix }

type pegChoice struct{ alternatives []*pegSequence }

func (p *pegPrimary) parts() []pegPart {
	if p.element == nil {
		return nil
	}
	return []pegPart{p.element}
}

func (s *pegSuffix) parts() []pegPart { return []pegPart{s.primary} }

func (p *pegPrefix) parts() []pegPart { return []pegPart{p.suffix} }

func (s *pegSequence) parts() []pegPart {
	list := make([]pegPart, len(s.elements))
	for i, e := range s.elements {
		list[i] = e
	}
	return list
}

func (c *pegChoice) parts() []pegPart {
	list := make([]pegPart, len(c.alternatives))
	for i, a := range c.alternatives {
		list[i] = a
	}
	return list
}

type pegAst = grammar[rule[pegPart]]

// pegFrame is one open group, or the rule itself at the bottom of the stack, as Pele's
// Frame: the group's node and the predicate written before it.
type pegFrame struct {
	choice   *pegChoice
	sequence *pegSequence
	node     *pegPrimary
	operator string
}

func newPegFrame(node *pegPrimary, choice *pegChoice, operator string) *pegFrame {
	sequence := &pegSequence{}
	choice.alternatives = append(choice.alternatives, sequence)
	return &pegFrame{choice, sequence, node, operator}
}

// pegDefinition reads a rule's definition, as Pele's definition does, without recursion: the
// groups still open are on a stack.
func pegDefinition(r *common.Reader) *pegChoice {
	var stack []*pegFrame
	top := newPegFrame(nil, &pegChoice{}, "")
	for {
		operator := ""
		if r.Kind() == pegAnd || r.Kind() == pegNot {
			operator = r.Take()
		}
		var primary *pegPrimary
		switch r.Kind() {
		case pegString:
			primary = &pegPrimary{typ: "PegLiteral", value: unescape(r.Take())}
		case pegID:
			primary = &pegPrimary{typ: "PegIdentifier", value: r.Take()}
		case pegDot:
			primary = &pegPrimary{typ: "PegAny", value: r.Take()}
		case pegGroup:
			r.I++
			element := &pegChoice{}
			stack = append(stack, top)
			top = newPegFrame(&pegPrimary{typ: "PegGroup", element: element}, element, operator)
			continue
		default:
			r.Fail("a literal, a name, a dot, or an opening parenthesis")
		}
		for {
			suffix := &pegSuffix{primary: primary}
			if k := r.Kind(); k == pegQuestion || k == pegStar || k == pegPlus {
				suffix.operator = r.Take()
			}
			top.sequence.elements = append(top.sequence.elements, &pegPrefix{operator, suffix})
			k := r.Kind()
			if k == pegGroupEnd && len(stack) > 0 {
				r.I++
				primary = top.node
				operator = top.operator
				top = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				continue
			}
			switch k {
			case pegSlash:
				r.I++
				top.sequence = &pegSequence{}
				top.choice.alternatives = append(top.choice.alternatives, top.sequence)
			case pegAnd, pegNot, pegString, pegID, pegDot, pegGroup:
			default:
				if len(stack) == 0 {
					return top.choice
				}
				r.Fail("token of type ')'")
			}
			break
		}
	}
}

// parsePeg parses a railroad-peg-beta text into its tree, as Pele's parsePeg does.
func parsePeg(src string) (*pegAst, error) {
	return parseGrammar(src, pegTable, "RailroadPeg", pegTitle, unescape, func(r *common.Reader) rule[pegPart] {
		name := r.Expect(pegID)
		r.Expect(pegArrow)
		rule := rule[pegPart]{typ: "PegRule", name: name, definition: pegDefinition(r)}
		r.Expect(pegSemicolon)
		return rule
	})
}

// pegLabel is what a predicate shows of the expression it looks ahead for, as Pele's label
// gives it.
func pegLabel(node *Node) string {
	switch node.Type {
	case Terminal:
		return `"` + node.Text + `"`
	case Nonterminal, Special:
		return node.Text
	}
	return "(...)"
}

// pegNode builds a node of the model from a part of the tree and its parts, as Pele's node
// does.
func pegNode(part pegPart, kids []*Node) *Node {
	var kid *Node
	if len(kids) > 0 {
		kid = kids[0]
	}
	switch p := part.(type) {
	case *pegChoice:
		return choice(kids)
	case *pegSequence:
		return sequence(kids)
	case *pegPrefix:
		if p.operator != "" {
			return special(p.operator + pegLabel(kid))
		}
		return kid
	case *pegSuffix:
		switch p.operator {
		case "":
			return kid
		case "?":
			return optional(kid)
		case "+":
			return repetition(kid, 1, infinite)
		}
		return repetition(kid, 0, infinite)
	case *pegPrimary:
		switch p.typ {
		case "PegLiteral":
			return terminal(p.value)
		case "PegIdentifier":
			return nonterminal(p.value)
		case "PegAny":
			return special(p.value)
		}
		return kid
	}
	panic("railroad: no such PEG part")
}

// pegRules are the rules of a railroad-peg-beta tree, as Pele's pegRules gives them.
func pegRules(ast *pegAst) []Rule { return toRules(ast.rules, pegNode) }
