// Package er parses and lays out Mermaid's entity relationship diagrams, as Pele's
// src/diagrams/er does: a lexer and a parser that agree with Mermaid's erDiagram.jison, the
// DB they fill, BuildGraph, which resolves the DB into the entities, subgraphs and
// relationships that are drawn, and Layout, which lays them out as a scene.
package er

import "github.com/firede/limu/diagram"

// Parse parses an ER diagram's text, as the parse of Pele's er diagram does: source is what
// diagram.Prepared.Source gives, and title the title its front matter gives, "" for none. An
// error is a *diagram.Error.
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

// ParsePrepared parses a prepared ER diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return Parse(p.Source(false), p.Title)
}
