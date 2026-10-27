// Package usecase parses Mermaid's use case diagrams and lays them out, as Pele's
// src/diagrams/usecase does: a lexer and a parser that make the decisions Mermaid's Chevrotain
// parser makes, the resolution of what they read into a Model, buildGraph, which turns the
// model into the nodes and edges that are drawn, and Layout, which lays them out as a scene.
//
// Offsets into the source, in a Span, are bytes of the Go string; the offsets and columns in
// errors are UTF-16 code units, as Pele's are.
package usecase

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// LabelType is how a label's text is read: "text", or "markdown" for a label written in
// backquotes.
type LabelType string

// The types of label.
const (
	LabelText     LabelType = "text"
	LabelMarkdown LabelType = "markdown"
)

// Span is where a part of a statement is in the source: the byte it starts at and the byte
// after its end.
type Span [2]int

// Actor is an actor, Pele's Actor. A string Pele may leave undefined is empty here where it is.
type Actor struct {
	ID        string
	Label     string
	LabelType LabelType
	// Type is "normal", "hollow", "awesome" or "icon".
	Type       string
	Icon       string
	Business   bool
	Stereotype string
	ParentID   string
	Classes    []string
	Styles     []string
}

// UseCase is a use case, Pele's UseCase.
type UseCase struct {
	ID        string
	Label     string
	LabelType LabelType
	// Shape is "ellipse" or "rect".
	Shape      string
	Business   bool
	Stereotype string
	ParentID   string
	Classes    []string
	Styles     []string
}

// SystemBoundary is a system boundary, Pele's SystemBoundary.
type SystemBoundary struct {
	ID        string
	Label     string
	LabelType LabelType
	// Type is "rect" or "package".
	Type    string
	Members []string
	Classes []string
	Styles  []string
}

// The arrow types of an association, in the order of Pele's Arrow.
const (
	SolidArrow = iota
	BackArrow
	LineSolid
	CircleArrow
	CrossArrow
	CircleArrowReversed
	CrossArrowReversed
)

// Relationship is a relationship between two elements, Pele's Relationship.
type Relationship struct {
	ID         string
	ExplicitID bool
	Source     string
	Target     string
	// Type is "association", "include", "extend" or "generalization".
	Type      string
	ArrowType int
	// Label and LabelType are the relationship's label, if LabelType is set.
	Label     string
	LabelType LabelType
	MinLen    int
	Classes   []string
	Styles    []string
	Animate   bool
	// Animation is "fast" or "slow", or "" where none is given.
	Animation string
}

// Note is a note about an actor or a use case, Pele's UsecaseNote.
type Note struct {
	ID        string
	Target    string
	Label     string
	LabelType LabelType
}

// JSONNode is a table of JSON, Pele's UsecaseJsonNode.
type JSONNode struct {
	ID    string
	Value *Object
	// PropertyOrder is the keys of each object in the order they were written, by JSON
	// Pointer, in the order the objects were met.
	PropertyOrder *KeyOrder
	Classes       []string
	Styles        []string
}

// ClassDef is a class defined with classDef, Pele's ClassDef.
type ClassDef struct {
	ID     string
	Styles []string
}

// MetadataOccurrence is where a property of a metadata block is, Pele's MetadataOccurrence.
type MetadataOccurrence struct {
	Key                      string
	Span, KeySpan, ValueSpan Span
}

// NodeOccurrence is where an element is mentioned in a statement, Pele's NodeOccurrence. A
// span Pele may leave undefined is nil, and so are Metadata and ClassSpans.
type NodeOccurrence struct {
	ID             string
	Span           Span
	IDSpan         Span
	LabelSpan      *Span
	Defines        bool
	StereotypeSpan *Span
	Metadata       []MetadataOccurrence
	ClassSpans     []Span
}

// EdgeOccurrence is where a relationship is in a statement, Pele's EdgeOccurrence.
type EdgeOccurrence struct {
	ID        string
	Span      Span
	LabelSpan *Span
	IDSpan    *Span
	Metadata  []MetadataOccurrence
}

// Statement is one statement of the source with the positions of its parts, as in Mermaid's
// use case AST: Pele's GraphStatement. What Pele leaves undefined is nil or empty.
type Statement struct {
	// Kind is "node", "edge", "group", "note", "json", "metadata", "edgeMetadata",
	// "classDef", "classAssign", "style", "direction", "accTitle", "accDescr", "comment" or
	// "blank".
	Kind       string
	Span       Span
	Nodes      []NodeOccurrence
	Edges      []EdgeOccurrence
	Group      string
	IDSpan     *Span
	TitleSpan  *Span
	EndSpan    *Span
	Ref        string
	RefSpan    *Span
	Children   []*Statement
	Metadata   []MetadataOccurrence
	ClassSpans []Span
	hasGroup   bool
}

// HasGroup reports whether the statement is of a system boundary, whose id may be "": whether
// Pele's GraphStatement has its group set.
func (s *Statement) HasGroup() bool { return s.hasGroup }

// Model is a parsed use case diagram, Pele's UsecaseModel. Each list is in the order Pele's
// map of it iterates.
type Model struct {
	Actors           []*Actor
	UseCases         []*UseCase
	SystemBoundaries []*SystemBoundary
	Relationships    []*Relationship
	Notes            []*Note
	JSONNodes        []*JSONNode
	ClassDefs        []*ClassDef
	// Direction is "TB", "BT", "LR" or "RL".
	Direction string
	Title     string
	AccTitle  string
	AccDescr  string
	// AccDescription is the accessible description as written, before the indentation of
	// its lines is dropped.
	AccDescription string
	Source         string
	HeaderSpan     Span
	Statements     []*Statement
}

// Parse parses a use case diagram's text, as the parse of Pele's use case diagram does: source
// is what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	return parse(source, title)
}

// ParsePrepared parses a prepared use case diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return common.ParsePrepared(p, false, func(src string) (*Model, error) {
		return Parse(src, p.Title)
	})
}
