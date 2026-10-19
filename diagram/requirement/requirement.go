// Package requirement parses and lays out Mermaid's requirement diagrams, as Pele's
// src/diagrams/requirement does: a lexer and a recursive-descent parser that agree with
// Mermaid's requirementDiagram.jison, the DB they fill, BuildGraph, which resolves the DB
// into the nodes and edges that are drawn, and Layout, which lays them out as a scene.
package requirement

import "github.com/firede/limu/diagram"

// Parse parses a requirement diagram's text, as the parse of Pele's requirement diagram
// does: source is what diagram.Prepared.Source gives, and title the title its front matter
// gives, "" for none. An error is a *diagram.Error.
func Parse(source, title string) (*DB, error) {
	db := NewDB()
	if title != "" {
		db.Title = title
	}
	if err := ParseInto(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared requirement diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return Parse(p.Source(false), p.Title)
}
