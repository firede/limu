package cynefin

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// The kinds of token, by index in cynefinTokens, as Pele's T, and those of the hidden types
// after them.
const (
	tKeywordColon = iota
	tKeyword
	tArrow
	tColon
	tDomain
	tAccDescr
	tAccTitle
	tTitle
	tString
	tNewline
)

// domainNames are the names DOMAIN_NAME matches, in the order its alternation tries them.
var domainNames = []string{"complex", "complicated", "clear", "chaotic", "confusion"}

// domainToken is Pele's DOMAIN_NAME, /complex|complicated|clear|chaotic|confusion/y: the
// first of the names the text starts with, with no word boundary after it, so that
// `complexity` starts with `complex`.
var domainToken = common.TokenType{Name: "DOMAIN_NAME", First: "c", Match: func(s string, at int) int {
	for _, name := range domainNames {
		if strings.HasPrefix(s[at:], name) {
			return at + len(name)
		}
	}
	return -1
}}

// cynefinTokens is Pele's CYNEFIN_TOKENS.
var cynefinTokens = []common.TokenType{
	{Name: "cynefin-beta:", Literal: "cynefin-beta:"},
	common.Keyword("cynefin-beta"),
	{Name: "-->", Literal: "-->"},
	{Name: ":", Literal: ":"},
	domainToken,
	common.AccDescrToken,
	common.AccTitleToken,
	common.TitleToken,
	common.StringToken,
	common.NewlineToken,
	common.WhitespaceToken,
	common.YAMLToken,
	common.DirectiveToken,
	common.CommentToken,
}

var cynefinTable = common.Prepare(cynefinTokens)

// blockAst is Pele's DomainBlockAst.
type blockAst struct {
	domain string
	items  []string
}

// transitionAst is Pele's TransitionAst; a label Pele leaves undefined has hasLabel unset.
type transitionAst struct {
	from, to string
	label    string
	hasLabel bool
}

// ast is Pele's CynefinAst.
type ast struct {
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	domains                            []blockAst
	transitions                        []transitionAst
}

// parseCynefin parses a Cynefin diagram's text into its tree, as Pele's parseCynefin does.
func parseCynefin(src string) (tree *ast, err error) {
	toks, err := common.Tokenize(src, cynefinTable, "cynefin")
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := recover(); e != nil {
			de, ok := e.(*diagram.Error)
			if !ok {
				panic(e)
			}
			tree, err = nil, de
		}
	}()
	r := &common.Reader{Tokens: toks, Types: cynefinTokens, Diagram: "cynefin"}
	tree = &ast{}

	skipLines := func() {
		for r.Kind() == tNewline {
			r.I++
		}
	}
	endOfLine := func() {
		if r.Kind() == -1 {
			return
		}
		if r.Kind() != tNewline {
			r.Fail("a line break or the end of input")
		}
		skipLines()
	}

	skipLines()
	if r.Kind() != tKeyword && r.Kind() != tKeywordColon {
		r.Fail("'cynefin-beta'")
	}
	r.I++
	for r.Kind() != -1 {
		switch r.Kind() {
		case tNewline:
			r.I++
		case tAccDescr:
			tree.accDescr, tree.hasAccDescr = common.AccDescrValue(r.Take()), true
			endOfLine()
		case tAccTitle:
			tree.accTitle, tree.hasAccTitle = common.AccTitleValue(r.Take()), true
			endOfLine()
		case tTitle:
			tree.title, tree.hasTitle = common.TitleValue(r.Take()), true
			endOfLine()
		case tDomain:
			name := r.Take()
			if r.Accept(tArrow) {
				t := transitionAst{from: name, to: r.Expect(tDomain)}
				if r.Accept(tColon) {
					t.label, t.hasLabel = common.StringValue(r.Expect(tString)), true
				}
				endOfLine()
				tree.transitions = append(tree.transitions, t)
				break
			}
			block := blockAst{domain: name}
			skipLines()
			for r.Kind() == tString {
				block.items = append(block.items, common.StringValue(r.Take()))
				skipLines()
			}
			tree.domains = append(tree.domains, block)
		default:
			r.Fail("a domain, a transition, a title, or a line break")
		}
	}
	return tree, nil
}
