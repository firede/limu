// Package kanban reads Mermaid's kanban boards and lays them out, as Pele's
// src/diagrams/kanban does: an outline of columns, each with its cards, and the cards'
// metadata of tickets, assignees and priorities.
package kanban

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// Tokenize reads a kanban board's tokens, as Pele's tokenize does.
func Tokenize(src string) common.OutlineTokens {
	return common.TokenizeOutline(src, true)
}

// Parse parses a kanban board's text, as the parse of Pele's kanban diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for none.
// An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	m := &Model{Title: title}
	if err := common.ParseOutline(source, &db{model: m}, true); err != nil {
		return nil, err
	}
	return m, nil
}

// ParsePrepared parses a prepared kanban board.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}
