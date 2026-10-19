// Package info reads Mermaid's info diagrams and lays them out, as Pele's src/diagrams/info
// does: a tokenizer and a parser that read what Mermaid's Langium grammar reads, buildInfo,
// which makes the model of their tree, and Layout, which draws the renderer's name and
// version under the title, if there is one.
package info

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The kinds of token, by index in infoTokens, as Pele's T, and those of the hidden types
// after them.
const (
	tShowInfo = iota
	tInfo
	tAccDescr
	tAccTitle
	tTitle
	tNewline
	tWhitespace
	tYAML
	tDirective
	tComment
)

// infoTokens is Pele's INFO_TOKENS.
var infoTokens = []common.TokenType{
	common.Keyword("showInfo"),
	common.Keyword("info"),
	common.AccDescrToken,
	common.AccTitleToken,
	common.TitleToken,
	common.NewlineToken,
	common.WhitespaceToken,
	common.YAMLToken,
	common.DirectiveToken,
	common.CommentToken,
}

var infoTable = common.Prepare(infoTokens)

// ast is Pele's InfoAst; a string Pele leaves undefined has its has field unset.
type ast struct {
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
}

// parseInfo parses an info diagram's text into its tree, as Pele's parseInfo does.
func parseInfo(src string) (tree *ast, err error) {
	toks, err := common.Tokenize(src, infoTable, "info")
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
	r := &common.Reader{Tokens: toks, Types: infoTokens, Diagram: "info"}
	tree = &ast{}

	newlines := func() {
		for r.Kind() == tNewline {
			r.I++
		}
	}
	newlines()
	r.Expect(tInfo)
	newlines()
	if r.Accept(tShowInfo) {
		newlines()
	}
	for r.Kind() != -1 {
		switch r.Kind() {
		case tAccDescr:
			tree.accDescr, tree.hasAccDescr = common.AccDescrValue(r.Take()), true
		case tAccTitle:
			tree.accTitle, tree.hasAccTitle = common.AccTitleValue(r.Take()), true
		case tTitle:
			tree.title, tree.hasTitle = common.TitleValue(r.Take()), true
		default:
			r.Fail("a title or the end of input")
		}
		if r.Kind() == -1 {
			break
		}
		if r.Kind() != tNewline {
			r.Fail("a line break or the end of input")
		}
		newlines()
	}
	return tree, nil
}

// Model is a parsed info diagram, Pele's InfoModel. A string Pele leaves undefined is empty.
type Model struct {
	Title, AccTitle, AccDescr string
}

// buildInfo makes the model of an info diagram's tree, as Pele's buildInfo does.
func buildInfo(tree *ast, title string) *Model {
	m := &Model{Title: tree.title}
	if m.Title == "" {
		m.Title = title
	}
	if tree.accTitle != "" {
		m.AccTitle = chars.TrimStart(tree.accTitle)
	}
	if tree.accDescr != "" {
		m.AccDescr = chars.DropIndents(tree.accDescr)
	}
	return m
}

// Parse parses an info diagram's text, as the parse of Pele's info diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	tree, err := parseInfo(source)
	if err != nil {
		return nil, err
	}
	return buildInfo(tree, title), nil
}

// ParsePrepared parses a prepared info diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}
