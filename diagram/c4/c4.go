// Package c4 parses Mermaid's C4 diagrams and lays them out, as Pele's src/diagrams/c4 does: a
// lexer and a parser that read what Mermaid's c4Diagram.jison reads, the model they build
// (C4Db), the grid of shapes and boundaries Mermaid lays them out in (layout.ts), the routes
// of the relations between them (route.ts), and Layout, which draws them as a scene
// (render.ts).
package c4

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// Parse parses a C4 diagram's text, as the parse of Pele's C4 diagram does: source is what
// diagram.Prepared.Source gives, and title the title its front matter gives, "" for none. An
// error is a *diagram.Error.
func Parse(source, title string) (*DB, error) {
	db := newDB()
	if title != "" {
		db.setTitle(title)
	}
	if err := parseC4(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared C4 diagram. It is limu's own: it hands Parse the source and
// the title that Pele's caller hands the parse of its C4 diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return common.ParsePrepared(p, false, func(src string) (*DB, error) {
		return Parse(src, p.Title)
	})
}
