// Package block parses Mermaid's block diagrams and lays them out, as Pele's
// src/diagrams/block does: a lexer and a parser that make the decisions Mermaid's Jison parser
// makes, the model they fill, Pele's BlockDb, and Layout, which places the blocks in the
// columns of their composites, draws arrow blocks and the edges between blocks, and gives a
// scene.
//
// Offsets into the source are bytes of the Go string; the columns in errors are UTF-16 code
// units, as Pele's are.
package block

import "github.com/firede/limu/diagram"

// A Block is a block, an edge, or a statement that sets columns, classes or styles, in one
// shape, as Pele's Block is, after Mermaid. A field Pele may leave undefined is nil here, or
// says so with a flag beside it.
type Block struct {
	// ID is "" only for a composite whose header was a statement with links, until the model
	// gives it one.
	ID   string
	Type string
	// Label is the block's label, where HasLabel says it has one.
	Label    string
	HasLabel bool
	// Children are the blocks a composite holds, in order; nil for an edge or a statement.
	Children []*Block
	// Columns is the number of columns of a composite, -1 for as many as it has blocks; nil
	// where it was not set.
	Columns *float64
	// WidthInColumns is the number of columns a block asked to span.
	WidthInColumns *float64
	// Width is the number of cells a `space` statement asked for.
	Width      *float64
	Directions []string
	Classes    []string
	Styles     []string
	ColorIndex *int
	// What an edge has: the ids of the blocks it joins, its markers, and how its line is
	// drawn.
	Start, End                   string
	ArrowTypeStart, ArrowTypeEnd string
	Thickness, Pattern           string
	// What the statements that set classes and styles have.
	CSS, StyleClass, StylesStr string

	// statements are what a composite holds as the parser leaves it, before the model reads
	// them into Children.
	statements []statement
}

// A ClassDef is a class of styles a classDef statement defines, Pele's BlockClassDef.
type ClassDef struct {
	ID         string
	Styles     []string
	TextStyles []string
}

// Parse parses a block diagram's text, as the parse of Pele's block diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. An error is a *diagram.Error.
func Parse(source, title string) (*DB, error) {
	db := NewDB()
	db.Title = title
	if err := parseBlock(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared block diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return Parse(p.Source(false), p.Title)
}
