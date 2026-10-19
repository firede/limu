package railroad

import (
	"math"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// What the four railroad grammars share, as Pele's railroad/common.ts has it: the title and
// accessibility tokens (which, unlike the ones in Mermaid's common grammar, do not take
// leading blanks), the hidden tokens, the statements before the first rule, string values,
// and the walk from a syntax tree to the rule model.

// isSpace is Pele's isSpace: whether a code unit is white space as JavaScript's \s has it.
func isSpace(c int) bool { return chars.IsSpace(c) }

// accDescrScanner is Pele's accDescrScanner, which matches as
// /accDescr(?:[\t ]*:([^\n\r]*?(?=%%)|[^\n\r]*)|\s*{([^}]*)})/y does. The braced form scans
// to the next closing brace. Once none is found, none exists further on either, so later
// braced descriptions fail without scanning again.
func accDescrScanner() func(string, int) int {
	unclosed := math.MaxInt
	return func(src string, at int) int {
		if !strings.HasPrefix(src[at:], "accDescr") {
			return -1
		}
		n := len(src)
		i := at + len("accDescr")
		for i < n && (src[i] == ' ' || src[i] == '\t') {
			i++
		}
		if i < n && src[i] == ':' {
			for i++; i < n; i++ {
				c := src[i]
				if c == '\n' || c == '\r' || c == '%' && i+1 < n && src[i+1] == '%' {
					break
				}
			}
			return i
		}
		i = chars.SpaceRun(src, i)
		if i >= n || src[i] != '{' || i >= unclosed {
			return -1
		}
		end := strings.IndexByte(src[i+1:], '}')
		if end < 0 {
			unclosed = i
			return -1
		}
		return i + 1 + end + 1
	}
}

// letters are Pele's LETTERS.
const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// titleToken is Pele's TITLE of the railroad grammars,
// /title(?:[\t ][^\n\r]*?(?=%%)|[\t ][^\n\r]*|)/y: common's, tried only where a t is.
var titleToken = common.TokenType{Name: "TITLE", First: "t", Match: common.TitleToken.Match}

// accTitleToken is Pele's ACC_TITLE, /accTitle[\t ]*:(?:[^\n\r]*?(?=%%)|[^\n\r]*)/y.
var accTitleToken = common.TokenType{Name: "ACC_TITLE", First: "a", Match: common.AccTitleToken.Match}

// accDescrToken is Pele's ACC_DESCR.
var accDescrToken = common.TokenType{Name: "ACC_DESCR", First: "a", Scanner: accDescrScanner}

// literal is Pele's literal: a token of the text of its name, and the types that win over it
// where they match more, by index in the table.
func literal(name string, longer ...int) common.TokenType {
	return common.TokenType{Name: name, Literal: name, Longer: longer}
}

// opening is Pele's opening: the diagram's keyword. A name that starts with it and goes on
// is a name.
func opening(word string, name int) common.TokenType {
	t := common.Keyword(word)
	t.Longer = []int{name}
	return t
}

// identifier is Pele's identifier, /[A-Z_a-z][\w-]*/y.
func identifier(name string) common.TokenType {
	return common.TokenType{Name: name, First: letters + "_", Match: func(s string, at int) int {
		if c := s[at]; !chars.IsLetter(int(c)) && c != '_' {
			return -1
		}
		i := at + 1
		for i < len(s) && (chars.IsWord(int(s[i])) || s[i] == '-') {
			i++
		}
		return i
	}}
}

// quoted is Pele's quoted, /"([^"\\]|\\.)*"|'([^'\\]|\\.)*'/y, which is common's STRING.
func quoted(name string) common.TokenType {
	t := common.StringToken
	t.Name = name
	return t
}

// renamed is a token type under another name, as Pele's { ...TYPE, name, first } makes it.
func renamed(t common.TokenType, name, first string) common.TokenType {
	t.Name = name
	t.First = first
	return t
}

// hiddenTokens are Pele's hiddenTokens: whitespace, front matter, directive, and %% comment,
// in the order every railroad grammar lists them.
func hiddenTokens(prefix string) []common.TokenType {
	return []common.TokenType{
		{Name: prefix + "_WHITESPACE", First: "\t \r\n", Hidden: true, Match: func(s string, at int) int {
			i := at
			for i < len(s) && (s[i] == '\t' || s[i] == ' ' || s[i] == '\r' || s[i] == '\n') {
				i++
			}
			if i == at {
				return -1
			}
			return i
		}},
		renamed(common.YAMLToken, prefix+"_YAML", "-"),
		renamed(common.DirectiveToken, prefix+"_DIRECTIVE", "\t %"),
		renamed(common.CommentToken, prefix+"_SINGLE_LINE_COMMENT", "\t %"),
	}
}

// blockComment is a hidden comment from open to the first close after it, as Pele's
// /\/\*[\s\S]*?\*\//y and /\(\*[\s\S]*?\*\)/y match one, with open as its opener.
func blockComment(name, open, close string) common.TokenType {
	return common.TokenType{
		Name: name, First: open[:1], Hidden: true,
		Opener: func(s string, at int) bool { return strings.HasPrefix(s[at:], open) },
		Match: func(s string, at int) int {
			if !strings.HasPrefix(s[at:], open) {
				return -1
			}
			end := strings.Index(s[at+len(open):], close)
			if end < 0 {
				return -1
			}
			return at + len(open) + end + len(close)
		},
	}
}

// slice is JavaScript's text.slice(1, -1): the text without its first and last characters,
// or nothing when it has fewer than two.
func slice(text string) string {
	if len(text) < 2 {
		return ""
	}
	return text[1 : len(text)-1]
}

// unescape is Pele's unescape: the quotes taken off a string. Mermaid's railroad value
// converters know three escapes; any other escaped character stands for itself.
func unescape(text string) string {
	if !strings.Contains(text, `\`) {
		return slice(text)
	}
	end := len(text) - 1
	var out strings.Builder
	for i := 1; i < end; i++ {
		c := text[i]
		if c == '\\' && i+1 < end {
			i++
			switch c = text[i]; c {
			case 'n':
				out.WriteByte('\n')
			case 'r':
				out.WriteByte('\r')
			case 't':
				out.WriteByte('\t')
			default:
				// The first byte of the character escaped; the loop writes the rest.
				out.WriteByte(c)
			}
		} else {
			out.WriteByte(c)
		}
	}
	return out.String()
}

// grammar is Pele's Grammar: a syntax tree's statements and its rules. A string Pele leaves
// undefined has its Has field unset.
type grammar[R any] struct {
	typ                                string
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	rules                              []R
}

func (g *grammar[R]) parsed() parsed { return parsed{g.title, g.accTitle, g.accDescr} }

// rule is a rule of a syntax tree: its name and its definition, which is a part of the tree.
type rule[P any] struct {
	typ        string
	name       string
	definition P
}

// parseGrammar parses the keyword, then any titles and accessibility statements, then rules
// to the end of the text, as Pele's parseGrammar does. Every table starts with its keyword and
// has TITLE, ACC_TITLE, and ACC_DESCR in a row from meta.
func parseGrammar[R any](src string, table *common.Table, typ string, meta int, unquote func(string) string,
	rule func(r *common.Reader) R) (g *grammar[R], err error) {
	toks, err := common.Tokenize(src, table, "railroad")
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := recover(); e != nil {
			de, ok := e.(*diagram.Error)
			if !ok {
				panic(e)
			}
			g, err = nil, de
		}
	}()
	r := &common.Reader{Tokens: toks, Types: table.Types, Diagram: "railroad"}
	g = &grammar[R]{typ: typ}
	r.Expect(0)
meta:
	for {
		switch r.Kind() {
		case meta:
			title := common.TitleValue(r.Take())
			if title != "" && (title[0] == '"' || title[0] == '\'') && strings.HasSuffix(title, title[:1]) {
				title = unquote(title)
			}
			g.title, g.hasTitle = title, true
		case meta + 1:
			g.accTitle, g.hasAccTitle = common.AccTitleValue(r.Take()), true
		case meta + 2:
			g.accDescr, g.hasAccDescr = common.AccDescrValue(r.Take()), true
		default:
			break meta
		}
	}
	for r.Kind() != -1 {
		g.rules = append(g.rules, rule(r))
	}
	return g, nil
}

func terminal(value string) *Node   { return &Node{Type: Terminal, Text: value} }
func nonterminal(name string) *Node { return &Node{Type: Nonterminal, Text: name} }
func special(text string) *Node     { return &Node{Type: Special, Text: text} }
func optional(element *Node) *Node  { return &Node{Type: Optional, Kids: []*Node{element}} }

// repetition is Pele's repetition, whose max is Infinity when it is not given.
func repetition(element *Node, min, max float64) *Node {
	return &Node{Type: Repetition, Kids: []*Node{element}, Min: min, Max: max}
}

// sequence is Pele's sequence: a sequence of one is that one, in every notation.
func sequence(elements []*Node) *Node {
	if len(elements) == 1 {
		return elements[0]
	}
	return &Node{Type: Sequence, Kids: elements}
}

// choice is Pele's choice: a choice of one is that one, in every notation.
func choice(alternatives []*Node) *Node {
	if len(alternatives) == 1 {
		return alternatives[0]
	}
	return &Node{Type: Choice, Kids: alternatives}
}

// toRules turns each rule's syntax tree into the common model, given how one node is built
// from its parts, as Pele's toRules does.
func toRules[P interface{ parts() []P }](rules []rule[P], build func(part P, kids []*Node) *Node) []Rule {
	out := make([]Rule, len(rules))
	for i, r := range rules {
		out[i] = Rule{Name: r.name, Definition: fold(r.definition, P.parts, build)}
	}
	return out
}

// fold rebuilds a tree bottom-up without recursion, so that nesting depth is bounded by
// memory, not by the stack, as Pele's fold does.
func fold[T, R any](root T, children func(T) []T, build func(node T, results []R) R) R {
	nodes := []T{root}
	lists := [][]T{children(root)}
	done := [][]R{nil}
	for {
		top := len(nodes) - 1
		results := done[top]
		if len(results) < len(lists[top]) {
			child := lists[top][len(results)]
			nodes = append(nodes, child)
			lists = append(lists, children(child))
			done = append(done, nil)
			continue
		}
		value := build(nodes[top], results)
		if top == 0 {
			return value
		}
		nodes = nodes[:top]
		lists = lists[:top]
		done = done[:top]
		done[top-1] = append(done[top-1], value)
	}
}
