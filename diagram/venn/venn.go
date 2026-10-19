// Package venn parses Mermaid's venn diagrams and lays them out, as Pele's src/diagrams/venn
// does: a lexer and a parser that make the decisions Mermaid's Jison grammar makes, the Model
// they fill, Pele's VennDb, and Layout, which places a circle for each set, with areas in
// proportion to the sizes of the sets and each pair as far apart as its overlap asks, and
// writes each region's label and texts where the region has the most room.
//
// Offsets into the source are bytes of the Go string; the columns in errors are UTF-16 code
// units, as Pele's are.
package venn

import (
	"github.com/firede/limu/diagram"
)

// Subset is a set (one id) or an overlap of sets (two or more ids, sorted), with its relative
// area, Pele's VennSubset. Label is nil where Pele leaves it undefined.
type Subset struct {
	Sets  []string
	Size  float64
	Label *string
}

// Text is a line of text shown inside the region of the given sets, Pele's VennText. Label
// is nil where Pele leaves it undefined.
type Text struct {
	Sets  []string
	ID    string
	Label *string
}

// Declaration is a property of a style and its value.
type Declaration struct {
	Property, Value string
}

// Style is a style statement, Pele's VennStyle: its targets are the sorted ids of a set or an
// overlap, or the id of a text, and its declarations each property once, in the order the
// properties first came, with the last value given it, as a Map keeps them.
type Style struct {
	Targets []string
	Styles  []Declaration
}

// Model is a parsed venn diagram, Pele's VennModel. Title is "" where Pele leaves it
// undefined.
type Model struct {
	Title     string
	Subsets   []Subset
	TextNodes []Text
	Styles    []Style
}

// Parse parses a venn diagram's text, as the parse of Pele's venn diagram does: source is what
// diagram.Prepared.Source gives, and title the title its front matter gives, "" for none. An
// error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	db := &db{}
	if title != "" {
		db.Title = title
	}
	if err := parseVenn(source, db); err != nil {
		return nil, err
	}
	return &db.Model, nil
}

// ParsePrepared parses a prepared venn diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}
