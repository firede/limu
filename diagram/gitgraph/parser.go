package gitgraph

import (
	"strconv"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// parseGit parses a git graph's text into its tree, as Pele's parseGit does.
func parseGit(src string) (ast *AST, err error) {
	t, err := common.Tokenize(src, gitTable, "gitGraph")
	if err != nil {
		return nil, err
	}
	defer func() {
		if v := recover(); v != nil {
			e, ok := v.(*diagram.Error)
			if !ok {
				panic(v)
			}
			ast, err = nil, e
		}
	}()
	r := &common.Reader{Tokens: t, Types: gitTokens, Diagram: "gitGraph"}
	ast = &AST{Type: "GitGraph", Statements: []Statement{}}

	endOfLine := func() {
		if r.Kind() == tEnd {
			return
		}
		if r.Kind() != tNewline {
			r.Fail("a line break or the end of input")
		}
		for r.Kind() == tNewline {
			r.I++
		}
	}
	name := func() string {
		switch r.Kind() {
		case tReference:
			return r.Take()
		case tString:
			return common.StringValue(r.Take())
		}
		r.Fail("a branch name")
		return ""
	}
	str := func() *string {
		s := common.StringValue(r.Expect(tString))
		return &s
	}
	commitType := func() *string {
		if k := r.Kind(); k != tNormal && k != tReverse && k != tHighlight {
			r.Fail("NORMAL, REVERSE, or HIGHLIGHT")
		}
		s := r.Take()
		return &s
	}

	for r.Kind() == tNewline {
		r.I++
	}
	if !r.Accept(tGitGraphColon) {
		r.Expect(tGitGraph)
		// The kind of the token after this one, or tEnd past the last, as JavaScript reads
		// undefined there.
		next := tEnd
		if r.I+1 < len(t.Kinds) {
			next = t.Kinds[r.I+1]
		}
		if k := r.Kind(); (k == tLR || k == tTB || k == tBT) && next == tColon {
			ast.Type = "Direction"
			ast.Dir = r.Take()
			r.I++
		} else {
			r.Accept(tColon)
		}
	}

	for r.Kind() != tEnd {
		switch r.Kind() {
		case tNewline:
			r.I++
		case tAccDescr:
			s := common.AccDescrValue(r.Take())
			ast.AccDescr = &s
			endOfLine()
		case tAccTitle:
			s := common.AccTitleValue(r.Take())
			ast.AccTitle = &s
			endOfLine()
		case tTitle:
			s := common.TitleValue(r.Take())
			ast.Title = &s
			endOfLine()
		case tCommit:
			r.I++
			node := Statement{Kind: "Commit", Tags: []string{}}
			for {
				if r.Accept(tID) {
					node.ID = str()
				} else if r.Accept(tTag) {
					node.Tags = append(node.Tags, *str())
				} else if r.Accept(tType) {
					node.Type = commitType()
				} else if r.Accept(tMsg) || r.Kind() == tString {
					node.Message = str()
				} else {
					break
				}
			}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		case tBranch:
			r.I++
			node := Statement{Kind: "Branch", Name: name()}
			if r.Accept(tOrder) {
				// parseInt of a run of digits: the nearest number, Infinity past the
				// largest.
				v, _ := strconv.ParseFloat(r.Expect(tInt), 64)
				node.Order = &v
			}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		case tMerge:
			r.I++
			node := Statement{Kind: "Merge", Branch: name(), Tags: []string{}}
			for {
				if r.Accept(tID) {
					node.ID = str()
				} else if r.Accept(tTag) {
					node.Tags = append(node.Tags, *str())
				} else if r.Accept(tType) {
					node.Type = commitType()
				} else {
					break
				}
			}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		case tCheckout, tSwitch:
			r.I++
			node := Statement{Kind: "Checkout", Branch: name()}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		case tCherryPick:
			r.I++
			node := Statement{Kind: "CherryPicking", Tags: []string{}}
			for {
				if r.Accept(tID) {
					node.ID = str()
				} else if r.Accept(tTag) {
					node.Tags = append(node.Tags, *str())
				} else if r.Accept(tParent) {
					node.Parent = str()
				} else {
					break
				}
			}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		default:
			r.Fail("a statement, a title, or a line break")
		}
	}
	return ast, nil
}
