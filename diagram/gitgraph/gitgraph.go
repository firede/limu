// Package gitgraph parses Mermaid's git graphs and lays them out, as Pele's src/diagrams/git
// does: a lexer and a parser that make the decisions Mermaid's Langium grammar makes, the
// replay of the statements into a Model of commits and branches, and Layout, which lays the
// commits out along the lanes of their branches as a scene.
//
// Byte offsets into the source are those of the Go string; the columns in errors are UTF-16
// code units, as Pele's are.
package gitgraph

import "github.com/firede/limu/diagram"

// AST is a git graph's text as the parser reads it, Pele's GitGraphAst. A string Pele may
// leave undefined is nil where it does.
type AST struct {
	// Type is "GitGraph", or "Direction" when the header gives a direction, as Langium names
	// the root.
	Type       string
	Dir        string // "LR", "TB", "BT", or "" where the header gives none
	Title      *string
	AccTitle   *string
	AccDescr   *string
	Statements []Statement
}

// Statement is a statement of a git graph, Pele's StatementAst: a commit, a branch, a merge,
// a checkout or a cherry-pick, by Kind, with the fields of its kind.
type Statement struct {
	// Kind is "Commit", "Branch", "Merge", "Checkout" or "CherryPicking".
	Kind string
	// ID, Message and Type are a commit's; ID and Type a merge's; ID a cherry-pick's. Type is
	// "NORMAL", "REVERSE" or "HIGHLIGHT".
	ID      *string
	Message *string
	Type    *string
	// Tags are a commit's, a merge's or a cherry-pick's.
	Tags []string
	// Name and Order are a branch's.
	Name  string
	Order *float64
	// Branch is the branch a merge or a checkout names.
	Branch string
	// Parent is the parent a cherry-pick names.
	Parent *string
}

// The types of commit, Pele's commitType.
const (
	Normal = iota
	Reverse
	Highlight
	MergeType
	CherryPick
)

// Commit is a commit, Pele's GitCommit.
type Commit struct {
	ID      string
	Message string
	Seq     int
	Type    int
	Tags    []string
	Parents []string
	Branch  string
	// CustomType is the type a merge is drawn as, where its statement gives one.
	CustomType *int
	// CustomID reports whether a merge's statement gave its id; nil for other commits.
	CustomID *bool
}

// Branch is a branch and the id of its latest commit, "" while HasHead is false.
type Branch struct {
	Name    string
	Head    string
	HasHead bool
}

// Model is a git graph, Pele's GitModel.
type Model struct {
	Title    string
	AccTitle string
	AccDescr string
	// Direction is "LR", "TB" or "BT".
	Direction string
	// Commits are keyed by id, in the order the ids first appeared; a commit given an id
	// another had takes its place.
	Commits []*Commit
	// Branches are in the order they were made, the main branch first.
	Branches []Branch
	// Lanes are the names of the branches in the order they are drawn: by order, then as
	// they were made.
	Lanes         []string
	CurrentBranch string
	Head          *Commit
	Warnings      []string
}

// Parse parses a git graph's text, as the parse of Pele's git graph does: source is what
// diagram.Prepared.Source gives, config the diagram's configuration and title the title its
// front matter gives, "" for none. An error is a *diagram.Error.
func Parse(source string, config diagram.Config, title string) (*Model, error) {
	ast, err := parseGit(source)
	if err != nil {
		return nil, err
	}
	return buildGit(ast, config, title)
}

// ParsePrepared parses a prepared git graph.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Config, p.Title)
}
