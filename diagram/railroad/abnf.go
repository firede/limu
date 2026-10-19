package railroad

import (
	"strings"

	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// railroad-abnf-beta: ABNF as in RFC 5234, with each rule closed by a semicolon, as Pele's
// railroad/abnf.ts reads it.

// The kinds of token, by index in abnfTokens, as Pele's T.
const (
	abnfKeyword = iota
	abnfEquals
	abnfSemicolon
	abnfSlash
	abnfGroup
	abnfGroupEnd
	abnfOption
	abnfOptionEnd
	abnfTitle
	abnfAccTitle
	abnfAccDescr
	abnfName
	abnfString
	abnfNumber
	abnfRepeat
	abnfExact
	// The comment, after the hidden types, which a semicolon names as longer.
	abnfComment = abnfExact + 5
)

// digits is the run of decimal digits from byte i of s, [0-9]*.
func digits(s string, i int) int {
	for i < len(s) && chars.IsDigit(int(s[i])) {
		i++
	}
	return i
}

// hexDigits is the run of hexadecimal digits from byte i of s, [0-9A-Fa-f]*.
func hexDigits(s string, i int) int {
	for i < len(s) && chars.IsHex(int(s[i])) {
		i++
	}
	return i
}

// abnfTokens is Pele's ABNF_TOKENS.
var abnfTokens = append(append([]common.TokenType{
	opening("railroad-abnf-beta", abnfName),
	literal("="),
	// A semicolon closes a rule only at the end of its line; with anything after it, it
	// starts a comment.
	literal(";", abnfComment),
	literal("/"),
	literal("("),
	literal(")"),
	literal("["),
	literal("]"),
	titleToken,
	accTitleToken,
	accDescrToken,
	// /[A-Za-z][A-Za-z0-9-]*/y
	{Name: "ABNF_RULENAME", First: letters, Match: func(s string, at int) int {
		if !chars.IsLetter(int(s[at])) {
			return -1
		}
		i := at + 1
		for i < len(s) && (chars.IsLetter(int(s[i])) || chars.IsDigit(int(s[i])) || s[i] == '-') {
			i++
		}
		return i
	}},
	// /"[^"]*"/y
	{Name: "ABNF_STRING", First: `"`, Match: func(s string, at int) int {
		if s[at] != '"' {
			return -1
		}
		end := strings.IndexByte(s[at+1:], '"')
		if end < 0 {
			return -1
		}
		return at + 1 + end + 1
	}},
	// /%[xXdDbB][0-9A-Fa-f]+(?:-[0-9A-Fa-f]+|\.[0-9A-Fa-f]+)*/y: each group needs a digit
	// after its - or ., and gives way where none is.
	{Name: "ABNF_NUMVAL", First: "%", Match: func(s string, at int) int {
		if s[at] != '%' || at+1 >= len(s) || !strings.ContainsRune("xXdDbB", rune(s[at+1])) {
			return -1
		}
		i := hexDigits(s, at+2)
		if i == at+2 {
			return -1
		}
		for i < len(s) && (s[i] == '-' || s[i] == '.') {
			j := hexDigits(s, i+1)
			if j == i+1 {
				break
			}
			i = j
		}
		return i
	}},
	// /[0-9]*\*[0-9]*/y
	{Name: "ABNF_REPEAT", First: "0123456789*", Match: func(s string, at int) int {
		i := digits(s, at)
		if i >= len(s) || s[i] != '*' {
			return -1
		}
		return digits(s, i+1)
	}},
	// /[0-9]+/y
	{Name: "ABNF_EXACT_REPEAT", First: "0123456789", Match: func(s string, at int) int {
		if i := digits(s, at); i > at {
			return i
		}
		return -1
	}},
}, hiddenTokens("ABNF")...),
	// /;[^\n\r]*/y
	common.TokenType{Name: "ABNF_COMMENT", First: ";", Hidden: true, Match: func(s string, at int) int {
		if s[at] != ';' {
			return -1
		}
		i := at + 1
		for i < len(s) && s[i] != '\n' && s[i] != '\r' {
			i++
		}
		return i
	}},
)

var abnfTable = common.Prepare(abnfTokens)

// The syntax tree of Pele's abnf.ts. abnfPart is any node of it.
type abnfPart interface{ parts() []abnfPart }

// abnfPrimary is Pele's AbnfPrimary: a string's or a number's value, a rule's name, or a
// group's alternation.
type abnfPrimary struct {
	typ     string
	value   string
	element *abnfAlternation
}

// abnfElement is Pele's AbnfElement, with the repeat written before it, if hasRepeat.
type abnfElement struct {
	repeat    string
	hasRepeat bool
	primary   *abnfPrimary
}

type abnfConcatenation struct{ elements []*abnfElement }

type abnfAlternation struct{ alternatives []*abnfConcatenation }

func (p *abnfPrimary) parts() []abnfPart {
	if p.element == nil {
		return nil
	}
	return []abnfPart{p.element}
}

func (e *abnfElement) parts() []abnfPart { return []abnfPart{e.primary} }

func (c *abnfConcatenation) parts() []abnfPart {
	list := make([]abnfPart, len(c.elements))
	for i, e := range c.elements {
		list[i] = e
	}
	return list
}

func (a *abnfAlternation) parts() []abnfPart {
	list := make([]abnfPart, len(a.alternatives))
	for i, c := range a.alternatives {
		list[i] = c
	}
	return list
}

type abnfAst = grammar[rule[abnfPart]]

// abnfFrame is one open bracket, or the rule itself at the bottom of the stack, as Pele's
// Frame: the bracket's node, its closing token, and the repeat written before it.
type abnfFrame struct {
	alternation   *abnfAlternation
	concatenation *abnfConcatenation
	node          *abnfPrimary
	close         int
	repeat        string
	hasRepeat     bool
}

func newAbnfFrame(node *abnfPrimary, alternation *abnfAlternation, close int, repeat string, hasRepeat bool) *abnfFrame {
	concatenation := &abnfConcatenation{}
	alternation.alternatives = append(alternation.alternatives, concatenation)
	return &abnfFrame{alternation, concatenation, node, close, repeat, hasRepeat}
}

// abnfStarts reports whether a token of a kind starts an element.
func abnfStarts(k int) bool {
	switch k {
	case abnfString, abnfNumber, abnfName, abnfGroup, abnfOption, abnfRepeat, abnfExact:
		return true
	}
	return false
}

// abnfAlternationOf reads a rule's definition, as Pele's alternation does, without
// recursion: the brackets still open are on a stack.
func abnfAlternationOf(r *common.Reader) *abnfAlternation {
	var stack []*abnfFrame
	top := newAbnfFrame(nil, &abnfAlternation{}, -2, "", false)
	for {
		repeat, hasRepeat := "", false
		if r.Kind() == abnfRepeat || r.Kind() == abnfExact {
			repeat, hasRepeat = r.Take(), true
		}
		var primary *abnfPrimary
		switch kind := r.Kind(); kind {
		case abnfString:
			primary = &abnfPrimary{typ: "AbnfStringLiteral", value: slice(r.Take())}
		case abnfNumber:
			primary = &abnfPrimary{typ: "AbnfNumVal", value: r.Take()}
		case abnfName:
			primary = &abnfPrimary{typ: "AbnfRuleName", value: r.Take()}
		case abnfGroup, abnfOption:
			r.I++
			element := &abnfAlternation{}
			stack = append(stack, top)
			typ := "AbnfOptionalGroup"
			if kind == abnfGroup {
				typ = "AbnfGroup"
			}
			top = newAbnfFrame(&abnfPrimary{typ: typ, element: element}, element, kind+1, repeat, hasRepeat)
			continue
		default:
			r.Fail("a string, a number, a name, or an opening bracket")
		}
		for {
			top.concatenation.elements = append(top.concatenation.elements, &abnfElement{repeat, hasRepeat, primary})
			k := r.Kind()
			if k == top.close {
				r.I++
				primary = top.node
				repeat, hasRepeat = top.repeat, top.hasRepeat
				top = stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				continue
			}
			if k == abnfSlash {
				r.I++
				top.concatenation = &abnfConcatenation{}
				top.alternation.alternatives = append(top.alternation.alternatives, top.concatenation)
			} else if !abnfStarts(k) {
				if len(stack) == 0 {
					return top.alternation
				}
				r.Fail("token of type '" + abnfTokens[top.close].Name + "'")
			}
			break
		}
	}
}

// parseAbnf parses a railroad-abnf-beta text into its tree, as Pele's parseAbnf does. Its
// title loses its quotes but keeps its backslashes.
func parseAbnf(src string) (*abnfAst, error) {
	return parseGrammar(src, abnfTable, "RailroadAbnf", abnfTitle, slice, func(r *common.Reader) rule[abnfPart] {
		name := r.Expect(abnfName)
		r.Expect(abnfEquals)
		definition := abnfAlternationOf(r)
		r.Expect(abnfSemicolon)
		return rule[abnfPart]{typ: "AbnfRule", name: name, definition: definition}
	})
}

// abnfNode builds a node of the model from a part of the tree and its parts, as Pele's node
// does. A repeat is read as parseInt reads it, so that a count of too many digits is
// infinite.
func abnfNode(part abnfPart, kids []*Node) *Node {
	switch p := part.(type) {
	case *abnfAlternation:
		return choice(kids)
	case *abnfConcatenation:
		return sequence(kids)
	case *abnfElement:
		repeat := p.repeat
		if repeat == "" {
			return kids[0]
		}
		star := strings.IndexByte(repeat, '*')
		exact := star == -1
		least := 0.0
		if star > 0 || exact {
			least = chars.ParseInt(repeat, 10)
		}
		most := infinite
		switch {
		case exact:
			most = least
		case star < len(repeat)-1:
			most = chars.ParseInt(repeat[star+1:], 10)
		}
		if least == 0 && most == 1 {
			return optional(kids[0])
		}
		return repetition(kids[0], least, most)
	case *abnfPrimary:
		switch p.typ {
		case "AbnfStringLiteral", "AbnfNumVal":
			return terminal(p.value)
		case "AbnfRuleName":
			return nonterminal(p.value)
		case "AbnfGroup":
			return kids[0]
		}
		return optional(kids[0])
	}
	panic("railroad: no such ABNF part")
}

// abnfRules are the rules of a railroad-abnf-beta tree, as Pele's abnfRules gives them.
func abnfRules(ast *abnfAst) []Rule { return toRules(ast.rules, abnfNode) }
