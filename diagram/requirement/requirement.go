// Package requirement parses and lays out Mermaid's requirement diagrams, as Pele's
// src/diagrams/requirement does: a lexer and a recursive-descent parser that agree with
// Mermaid's requirementDiagram.jison, the DB they fill, buildGraph, which resolves the DB
// into the nodes and edges that are drawn, and Layout, which lays them out as a scene.
package requirement

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// Parse parses a requirement diagram's text, as the parse of Pele's requirement diagram
// does: source is what diagram.Prepared.Source gives, and title the title its front matter
// gives, "" for none. An error is a *diagram.Error.
func Parse(source, title string) (*DB, error) {
	db := newDB()
	if title != "" {
		db.Title = title
	}
	if err := parseInto(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared requirement diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return common.ParsePrepared(p, false, func(src string) (*DB, error) {
		return Parse(src, p.Title)
	})
}
