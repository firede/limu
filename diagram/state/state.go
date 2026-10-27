// Package state parses Mermaid's state diagrams and lays them out, as Pele's
// src/diagrams/state does: a lexer and a parser that agree with Mermaid's stateDiagram.jison,
// the DB they fill, the graph it resolves into, and the layout of that graph as a scene.
//
// The scene's items have the roles state.state (a state, with shape rect, or rectWithTitle for
// one with descriptions under a rule), state.start, state.end, state.choice, state.fork,
// state.join, state.note, state.composite (a group, shape roundedWithTitle), state.divider (the
// dashed line between the regions of a composite), state.transition, state.note-edge (the
// dashed line from a note to its state), state.edge-label, state.description (a line under a
// state's name) and state.title.
package state

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// Parse parses a state diagram's text, as the parse of Pele's state diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for none.
// An error is a *diagram.Error.
func Parse(source string, title string) (*DB, error) {
	db := newDB()
	if title != "" {
		db.Title = title
	}
	if err := parseInto(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared state diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return common.ParsePrepared(p, false, func(src string) (*DB, error) {
		return Parse(src, p.Title)
	})
}
