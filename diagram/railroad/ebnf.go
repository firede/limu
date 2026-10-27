package railroad

import (
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// railroad-ebnf-beta: EBNF in the W3C and ISO 14977 styles, as Pele's railroad/ebnf.ts reads
// it.

// The kinds of token, by index in ebnfTokens, as Pele's T.
const (
	ebnfKeyword = iota
	ebnfDefines
	ebnfEquals
	ebnfSemicolon
	ebnfBar
	ebnfComma
	ebnfGroup
	ebnfGroupEnd
	ebnfOption
	ebnfOptionEnd
	ebnfRepeat
	ebnfRepeatEnd
	ebnfQuestion
	ebnfStar
	ebnfPlus
	ebnfMinus
	ebnfTitle
	ebnfAccTitle
	ebnfAccDescr
	ebnfID
	ebnfString
	ebnfSpecial
	// The hidden types after them, which a literal names as longer.
	ebnfYAML       = ebnfSpecial + 2
	ebnfISOComment = ebnfSpecial + 6
)

// specialSequence is Pele's specialSequence: a question mark, text with something other than
// blanks in it and no ? or ;, and a question mark. Mermaid's pattern,
// /\?(?=[^?;]*[^?\s;][^?;]*\?)[^?;]*\?/y, looks ahead with two nested runs, which rescans;
// this reads the text once.
func specialSequence(src string, at int) int {
	if src[at] != '?' {
		return -1
	}
	solid := false
	for i := at + 1; i < len(src); {
		c, size := chars.UnitAt(src, i)
		if c == '?' {
			if solid {
				return i + 1
			}
			return -1
		}
		if c == ';' {
			return -1
		}
		solid = solid || !isSpace(c)
		i += size
	}
	return -1
}

// ebnfTokens is Pele's EBNF_TOKENS.
var ebnfTokens = append(append([]common.TokenType{
	opening("railroad-ebnf-beta", ebnfID),
	literal("::="),
	literal("="),
	literal(";"),
	literal("|"),
	literal(","),
	literal("(", ebnfISOComment),
	literal(")"),
	literal("["),
	literal("]"),
	literal("{"),
	literal("}"),
	literal("?", ebnfSpecial),
	literal("*"),
	literal("+"),
	literal("-", ebnfYAML),
	title(wordOrDash),
	accTitleToken,
	accDescrToken,
	identifier("EBNF_ID"),
	quoted("EBNF_STRING"),
	{Name: "EBNF_SPECIAL_SEQUENCE", First: "?", Scanner: func() func(string, int) int { return specialSequence }},
}, hiddenTokens("EBNF")...),
	blockComment("EBNF_BLOCK_COMMENT", "/*", "*/"),
	blockComment("EBNF_ISO_COMMENT", "(*", "*)"),
)

var ebnfTable = common.Prepare(ebnfTokens)

// The syntax tree of Pele's ebnf.ts. ebnfPart is any node of it.
type ebnfPart interface{ parts() []ebnfPart }

// ebnfPrimary is Pele's EbnfPrimary: a terminal's value, a non-terminal's name or a special's
// text, or a bracket's choice.
type ebnfPrimary struct {
	typ     string
	value   string
	element *ebnfChoice
}

// ebnfPostfix is Pele's EbnfPostfix: an operator, or an exception's right side.
type ebnfPostfix struct {
	typ      string
	operator string
	except   *ebnfPrimary
}

type ebnfTerm struct {
	base      *ebnfPrimary
	postfixes []*ebnfPostfix
}

type ebnfSequence struct{ elements []*ebnfTerm }

type ebnfChoice struct{ alternatives []*ebnfSequence }

// The parts of each node, as Pele's parts gives them: a term's are its base and the right
// sides of its exceptions.
func (p *ebnfPrimary) parts() []ebnfPart {
	if p.element == nil {
		return nil
	}
	return []ebnfPart{p.element}
}

func (t *ebnfTerm) parts() []ebnfPart {
	list := []ebnfPart{t.base}
	for _, postfix := range t.postfixes {
		if postfix.except != nil {
			list = append(list, postfix.except)
		}
	}
	return list
}

func (s *ebnfSequence) parts() []ebnfPart {
	list := make([]ebnfPart, len(s.elements))
	for i, e := range s.elements {
		list[i] = e
	}
	return list
}

func (c *ebnfChoice) parts() []ebnfPart {
	list := make([]ebnfPart, len(c.alternatives))
	for i, a := range c.alternatives {
		list[i] = a
	}
	return list
}

type ebnfAst = grammar[rule[ebnfPart]]

// ebnfFrame is one open bracket, or the rule itself at the bottom of the stack, as Pele's
// Frame: the bracket's node, its closing token, and whether it is the right side of an
// exception.
type ebnfFrame struct {
	choice   *ebnfChoice
	sequence *ebnfSequence
	term     *ebnfTerm
	node     *ebnfPrimary
	close    int
	except   bool
}

func newEbnfFrame(node *ebnfPrimary, choice *ebnfChoice, close int, except bool) *ebnfFrame {
	sequence := &ebnfSequence{}
	choice.alternatives = append(choice.alternatives, sequence)
	return &ebnfFrame{choice: choice, sequence: sequence, node: node, close: close, except: except}
}

var ebnfBrackets = [...]string{"EbnfGroup", "EbnfOptional", "EbnfRepetition"}

// ebnfDefinition reads a rule's definition, as Pele's definition does, without recursion: the
// brackets still open are on a stack.
func ebnfDefinition(r *common.Reader) *ebnfChoice {
	var stack []*ebnfFrame
	top := newEbnfFrame(nil, &ebnfChoice{}, -2, false)
	except := false
	for {
		var primary *ebnfPrimary
		switch kind := r.Kind(); kind {
		case ebnfString:
			primary = &ebnfPrimary{typ: "EbnfTerminal", value: unescape(r.Take())}
		case ebnfID:
			primary = &ebnfPrimary{typ: "EbnfNonTerminal", value: r.Take()}
		case ebnfSpecial:
			primary = &ebnfPrimary{typ: "EbnfSpecial", value: chars.Trim(slice(r.Take()))}
		case ebnfGroup, ebnfOption, ebnfRepeat:
			r.I++
			element := &ebnfChoice{}
			stack = append(stack, top)
			top = newEbnfFrame(&ebnfPrimary{typ: ebnfBrackets[(kind-ebnfGroup)>>1], element: element}, element, kind+1, except)
			except = false
			continue
		default:
			r.Fail("a terminal, a name, or an opening bracket")
		}
		for {
			if except {
				top.term.postfixes = append(top.term.postfixes, &ebnfPostfix{typ: "EbnfExceptionPostfix", except: primary})
				except = false
			} else {
				top.term = &ebnfTerm{base: primary}
				top.sequence.elements = append(top.sequence.elements, top.term)
			}
			term := top.term
			for {
				var typ string
				switch r.Kind() {
				case ebnfQuestion:
					typ = "EbnfOptionalPostfix"
				case ebnfStar:
					typ = "EbnfZeroOrMorePostfix"
				case ebnfPlus:
					typ = "EbnfOneOrMorePostfix"
				}
				if typ == "" {
					break
				}
				term.postfixes = append(term.postfixes, &ebnfPostfix{typ: typ, operator: r.Take()})
			}
			k := r.Kind()
			if k == top.close {
				r.I++
				primary = top.node
				except = top.except
				top = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				continue
			}
			switch k {
			case ebnfMinus:
				r.I++
				except = true
			case ebnfComma:
				r.I++
			case ebnfBar:
				r.I++
				top.sequence = &ebnfSequence{}
				top.choice.alternatives = append(top.choice.alternatives, top.sequence)
			case ebnfString, ebnfID, ebnfSpecial, ebnfGroup, ebnfOption, ebnfRepeat:
			default:
				if len(stack) == 0 {
					return top.choice
				}
				r.Fail("token of type '" + ebnfTokens[top.close].Name + "'")
			}
			break
		}
	}
}

// parseEbnf parses a railroad-ebnf-beta text into its tree, as Pele's parseEbnf does.
func parseEbnf(src string) (*ebnfAst, error) {
	return parseGrammar(src, ebnfTable, "RailroadEbnf", ebnfTitle, unescape, func(r *common.Reader) rule[ebnfPart] {
		name := r.Expect(ebnfID)
		if !r.Accept(ebnfEquals) {
			r.Expect(ebnfDefines)
		}
		rule := rule[ebnfPart]{typ: "EbnfRule", name: name, definition: ebnfDefinition(r)}
		r.Expect(ebnfSemicolon)
		return rule
	})
}

// ebnfNode builds a node of the model from a part of the tree and its parts, as Pele's node
// does.
func ebnfNode(part ebnfPart, kids []*Node) *Node {
	switch p := part.(type) {
	case *ebnfChoice:
		return choice(kids)
	case *ebnfSequence:
		return sequence(kids)
	case *ebnfTerm:
		// Each postfix operator applies to everything before it in the term.
		result := kids[0]
		next := 1
		for _, postfix := range p.postfixes {
			switch postfix.typ {
			case "EbnfOptionalPostfix":
				result = optional(result)
			case "EbnfExceptionPostfix":
				result = &Node{Type: Sequence, Kids: []*Node{result, terminal("-"), kids[next]}}
				next++
			case "EbnfOneOrMorePostfix":
				result = repetition(result, 1, infinite)
			default:
				result = repetition(result, 0, infinite)
			}
		}
		return result
	case *ebnfPrimary:
		switch p.typ {
		case "EbnfTerminal":
			return terminal(p.value)
		case "EbnfNonTerminal":
			return nonterminal(p.value)
		case "EbnfSpecial":
			return special(p.value)
		case "EbnfGroup":
			return kids[0]
		case "EbnfOptional":
			return optional(kids[0])
		}
		return repetition(kids[0], 0, infinite)
	}
	panic("railroad: no such EBNF part")
}

// ebnfRules are the rules of a railroad-ebnf-beta tree, as Pele's ebnfRules gives them.
func ebnfRules(ast *ebnfAst) []Rule { return toRules(ast.rules, ebnfNode) }
