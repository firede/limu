// Package timeline parses Mermaid's timeline diagrams and lays them out, as Pele's
// src/diagrams/timeline does: a lexer and a parser that make the decisions Mermaid's Jison
// parser makes, the model they fill, Pele's TimelineDb, and Layout, which lays the periods
// out along an axis, across or down, as a scene.
//
// Offsets into the source are bytes of the Go string; the columns in errors are UTF-16 code
// units, as Pele's are.
package timeline

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// Period is a time period and its events, Pele's TimelinePeriod.
type Period struct {
	Text   string
	Events []string
	// Section is the name of the section in force when the period was declared, "" before
	// the first one.
	Section string
	// SectionIndex is the index into Model.Sections of that section, -1 before the first one.
	SectionIndex int
}

// Model is a parsed timeline diagram, Pele's TimelineModel. A string Pele may leave
// undefined is empty here where it is.
type Model struct {
	Title    string
	AccTitle string
	AccDescr string
	// Direction is "LR" or "TD".
	Direction string
	Sections  []string
	Periods   []*Period

	currentSection string
}

// Parse parses a timeline diagram's text, as the parse of Pele's timeline diagram does:
// source is what diagram.Prepared.Source gives, and title the title its front matter gives,
// "" for none. An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	m := &Model{Direction: "LR", Title: title}
	if err := parseTimeline(source, m); err != nil {
		return nil, err
	}
	return m, nil
}

// ParsePrepared parses a prepared timeline diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return common.ParsePrepared(p, false, func(src string) (*Model, error) {
		return Parse(src, p.Title)
	})
}
