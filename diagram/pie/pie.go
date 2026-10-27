// Package pie parses Mermaid's pie charts and lays them out, as Pele's src/diagrams/pie does:
// a tokenizer and a parser that read what Mermaid's Langium grammar reads, buildPie, which
// makes the model of their tree, and Layout, which lays out the slices, their percentages and
// a legend as a scene.
//
// Values are numbers as JavaScript's are, float64, as a value may be written with more
// digits than any float64 holds.
package pie

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// Section is a slice of the pie, with its label and value.
type Section struct {
	Label string
	Value float64
}

// Model is a parsed pie chart, Pele's PieModel. A string Pele leaves undefined is empty.
// Sections are in the order their labels first appear; a label written again keeps its first
// value, as Pele's Map of sections does.
type Model struct {
	Title, AccTitle, AccDescr string
	ShowData                  bool
	Sections                  []Section
}

// buildPie makes the model of a pie chart's tree, as Pele's buildPie does: a negative value
// is an error, and a label written again is left out.
func buildPie(tree *ast, title string) (*Model, error) {
	m := &Model{ShowData: tree.showData}
	seen := map[string]bool{}
	for _, s := range tree.sections {
		if s.value < 0 {
			return nil, common.NewError(`"`+s.label+`" has invalid value: `+chars.NumberString(s.value)+
				`. Negative values are not allowed in pie charts. All slice values must be >= 0.`, diagram.Semantic, "pie")
		}
		if !seen[s.label] {
			seen[s.label] = true
			m.Sections = append(m.Sections, Section{s.label, s.value})
		}
	}
	m.Title = tree.title
	if m.Title == "" {
		m.Title = title
	}
	if tree.accTitle != "" {
		m.AccTitle = chars.TrimStart(tree.accTitle)
	}
	if tree.accDescr != "" {
		m.AccDescr = chars.DropIndents(tree.accDescr)
	}
	return m, nil
}

// Parse parses a pie chart's text, as the parse of Pele's pie diagram does: source is what
// diagram.Prepared.Source gives, and title the title its front matter gives, "" for none. An
// error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	tree, err := parsePie(source)
	if err != nil {
		return nil, err
	}
	return buildPie(tree, title)
}

// ParsePrepared parses a prepared pie chart.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return common.ParsePrepared(p, false, func(src string) (*Model, error) {
		return Parse(src, p.Title)
	})
}
