// Package quadrant parses Mermaid's quadrant charts and lays them out, as Pele's
// src/diagrams/quadrant does: a lexer and a parser that make the decisions Mermaid's Jison
// grammar makes, the Model they fill, and Layout, which lays a model out as a scene.
//
// Offsets into the source are bytes of the Go string; the offsets and columns in errors are
// UTF-16 code units, as Pele's are.
package quadrant

import (
	"github.com/firede/limu/diagram"
)

// Text is a label of the chart, Pele's QuadrantText: plain text, or markdown when it was
// written in backquotes inside its quotes.
type Text struct {
	Text     string
	Markdown bool
}

// PointStyle is what a point's own styles or its class set, Pele's PointStyle. A field is
// nil where it was not set.
type PointStyle struct {
	// Radius is read with parseInt, so that a radius of very many digits is +Inf.
	Radius *float64
	// Color and StrokeColor are hex codes, with or without their #.
	Color       *string
	StrokeColor *string
	// StrokeWidth is a whole number of pixels, as "5px".
	StrokeWidth *string
}

// Point is a point of the chart, Pele's QuadrantPoint.
type Point struct {
	PointStyle
	// X and Y are what Number makes of the coordinates written: from 0 to 1 when they are
	// numbers, and NaN or a number outside that when they are not, as `0x5` is 5.
	X, Y  float64
	Text  Text
	Class string // "" for none
}

// Model is a parsed quadrant chart, Pele's QuadrantModel. A string Pele may leave undefined
// is empty here where it is; a label it may leave undefined is nil.
type Model struct {
	Title    string
	AccTitle string
	AccDescr string
	// Quadrants are the labels of quadrants 1 to 4: top right, top left, bottom left, bottom
	// right.
	Quadrants   [4]*Text
	XAxisLeft   *Text
	XAxisRight  *Text
	YAxisBottom *Text
	YAxisTop    *Text
	Points      []Point
	// Classes are the styles of each class, by its name; ClassOrder is the names in the order
	// they were first defined, as a Map iterates them.
	Classes    map[string]PointStyle
	ClassOrder []string
}

// Parse parses a quadrant chart's text, as the parse of Pele's quadrant chart does: source
// is what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	db := newDB()
	db.Title = title
	if err := parse(source, db); err != nil {
		return nil, err
	}
	return &db.Model, nil
}

// ParsePrepared parses a prepared quadrant chart.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}
