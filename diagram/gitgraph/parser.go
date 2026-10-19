package gitgraph

import (
	"strconv"

	"github.com/firede/limu/diagram"
)

// reader reads tokens for the parser, Pele's Reader, with Chevrotain's messages on failure.
// A failure panics with a *diagram.Error, which parseGit recovers.
type reader struct {
	t *tokens
	i int
}

func (r *reader) kind() int { return r.t.kinds[r.i] }

// kindAt is the kind of token i, or tEnd past the last, as JavaScript reads undefined there.
func (r *reader) kindAt(i int) int {
	if i < len(r.t.kinds) {
		return r.t.kinds[i]
	}
	return tEnd
}

func (r *reader) text(i int) string { return r.t.src[r.t.starts[i]:r.t.ends[i]] }

func (r *reader) take() string {
	s := r.text(r.i)
	r.i++
	return s
}

func (r *reader) accept(k int) bool {
	if r.kind() != k {
		return false
	}
	r.i++
	return true
}

func (r *reader) expect(k int) string {
	if r.kind() != k {
		r.fail("token of type '" + tokenNames[k] + "'")
	}
	return r.take()
}

func (r *reader) fail(expected string) {
	found := "end of input"
	if r.kind() != tEnd {
		found = "`" + r.text(r.i) + "`"
	}
	panic(diagram.SyntaxError("gitGraph", r.t.src, r.t.starts[r.i], "Expecting "+expected+" but found "+found+".", false))
}

// parseGit parses a git graph's text into its tree, as Pele's parseGit does.
func parseGit(src string) (ast *AST, err error) {
	t, err := tokenize(src)
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
	r := &reader{t: t}
	ast = &AST{Type: "GitGraph", Statements: []Statement{}}

	endOfLine := func() {
		if r.kind() == tEnd {
			return
		}
		if r.kind() != tNewline {
			r.fail("a line break or the end of input")
		}
		for r.kind() == tNewline {
			r.i++
		}
	}
	name := func() string {
		switch r.kind() {
		case tReference:
			return r.take()
		case tString:
			return stringValue(r.take())
		}
		r.fail("a branch name")
		return ""
	}
	str := func() *string {
		s := stringValue(r.expect(tString))
		return &s
	}
	commitType := func() *string {
		if k := r.kind(); k != tNormal && k != tReverse && k != tHighlight {
			r.fail("NORMAL, REVERSE, or HIGHLIGHT")
		}
		s := r.take()
		return &s
	}

	for r.kind() == tNewline {
		r.i++
	}
	if !r.accept(tGitGraphColon) {
		r.expect(tGitGraph)
		next := r.kindAt(r.i + 1)
		if k := r.kind(); (k == tLR || k == tTB || k == tBT) && next == tColon {
			ast.Type = "Direction"
			ast.Dir = r.take()
			r.i++
		} else {
			r.accept(tColon)
		}
	}

	for r.kind() != tEnd {
		switch r.kind() {
		case tNewline:
			r.i++
		case tAccDescr:
			s := accDescrValue(r.take())
			ast.AccDescr = &s
			endOfLine()
		case tAccTitle:
			s := accTitleValue(r.take())
			ast.AccTitle = &s
			endOfLine()
		case tTitle:
			s := titleValue(r.take())
			ast.Title = &s
			endOfLine()
		case tCommit:
			r.i++
			node := Statement{Kind: "Commit", Tags: []string{}}
			for {
				if r.accept(tID) {
					node.ID = str()
				} else if r.accept(tTag) {
					node.Tags = append(node.Tags, *str())
				} else if r.accept(tType) {
					node.Type = commitType()
				} else if r.accept(tMsg) || r.kind() == tString {
					node.Message = str()
				} else {
					break
				}
			}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		case tBranch:
			r.i++
			node := Statement{Kind: "Branch", Name: name()}
			if r.accept(tOrder) {
				// parseInt of a run of digits: the nearest number, Infinity past the
				// largest.
				v, _ := strconv.ParseFloat(r.expect(tInt), 64)
				node.Order = &v
			}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		case tMerge:
			r.i++
			node := Statement{Kind: "Merge", Branch: name(), Tags: []string{}}
			for {
				if r.accept(tID) {
					node.ID = str()
				} else if r.accept(tTag) {
					node.Tags = append(node.Tags, *str())
				} else if r.accept(tType) {
					node.Type = commitType()
				} else {
					break
				}
			}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		case tCheckout, tSwitch:
			r.i++
			node := Statement{Kind: "Checkout", Branch: name()}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		case tCherryPick:
			r.i++
			node := Statement{Kind: "CherryPicking", Tags: []string{}}
			for {
				if r.accept(tID) {
					node.ID = str()
				} else if r.accept(tTag) {
					node.Tags = append(node.Tags, *str())
				} else if r.accept(tParent) {
					node.Parent = str()
				} else {
					break
				}
			}
			endOfLine()
			ast.Statements = append(ast.Statements, node)
		default:
			r.fail("a statement, a title, or a line break")
		}
	}
	return ast, nil
}
