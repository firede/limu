// Package ishikawa parses Mermaid's Ishikawa (fishbone) diagrams and lays them out, as Pele's
// src/diagrams/ishikawa does: a lexer and a parser that make the decisions Mermaid's Jison
// parser makes, the model they fill, Pele's IshikawaDb, which reads the causes as a tree by
// their indentation, and Layout, which draws the effect at the head of a spine and the causes
// as bones off it, or, in a narrow host, as an outline that runs down.
//
// Offsets into the source are bytes of the Go string; levels of indentation and the columns
// in errors are UTF-16 code units, as Pele's are.
package ishikawa

import "github.com/firede/limu/diagram"

// A Node is the effect or a cause, Pele's IshikawaNode.
type Node struct {
	Text     string
	Children []*Node
}

// Model is a parsed Ishikawa diagram, Pele's IshikawaModel. The root is the effect; its
// children are the categories of causes, theirs the causes, and so on. Root is nil for a
// diagram with no lines after its keyword.
type Model struct {
	// Title is the title the front matter gives, if HasTitle.
	Title    string
	HasTitle bool
	Root     *Node
}

// Parse parses an Ishikawa diagram's text, as the parse of Pele's ishikawa diagram does:
// source is what diagram.Prepared.Source gives, and title the title its front matter gives,
// if hasTitle. An error is a *diagram.Error.
func Parse(source, title string, hasTitle bool) (*Model, error) {
	db := newDB()
	db.model.Title, db.model.HasTitle = title, hasTitle
	if err := parseIshikawa(source, db); err != nil {
		return nil, err
	}
	return db.model, nil
}

// ParsePrepared parses a prepared Ishikawa diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title, p.HasTitle)
}
