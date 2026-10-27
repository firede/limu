// Package sankey parses Mermaid's sankey diagrams and lays them out, as Pele's
// src/diagrams/sankey does: a lexer and a parser of the CSV that Mermaid's Jison grammar
// reads, the model they fill, Pele's SankeyDb, and Layout, which places the nodes in columns
// by their flow and draws the bands between them, with a label beside each node, as a scene.
//
// Offsets into the source are bytes of the Go string; the columns in errors are UTF-16 code
// units, as Pele's are.
package sankey

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// Node is a node of a diagram, Pele's SankeyNode: its id as written, with the entities of the
// text encoded as Pele's preprocessing leaves them, and its place in Model.Nodes.
type Node struct {
	ID    string
	Index int
}

// Link is a band of flow from one node to another, Pele's SankeyLink, with its nodes by
// their place in Model.Nodes. Value is what parseFloat reads of the third field, which is
// always finite, as a row whose value is not is an error.
type Link struct {
	Source, Target int
	Value          float64
}

// Model is a parsed sankey diagram, Pele's SankeyModel and the SankeyDb that fills it. Its
// grammar has no accessible title or description, so AccTitle and AccDescr stay empty, as
// Pele's stay undefined.
type Model struct {
	Nodes    []Node
	Links    []Link
	Title    string
	AccTitle string
	AccDescr string

	lookup map[string]int
}

// newModel is a model with nothing in it, as Pele's SankeyDb starts.
func newModel() *Model {
	return &Model{Nodes: []Node{}, Links: []Link{}, lookup: map[string]int{}}
}

// findOrCreateNode is the node of an id, added at the end of the nodes the first time the id
// is seen, as Pele's findOrCreateNode does.
func (m *Model) findOrCreateNode(id string) int {
	if i, ok := m.lookup[id]; ok {
		return i
	}
	i := len(m.Nodes)
	m.lookup[id] = i
	m.Nodes = append(m.Nodes, Node{ID: id, Index: i})
	return i
}

// addLink adds a link, as Pele's addLink does.
func (m *Model) addLink(source, target int, value float64) {
	m.Links = append(m.Links, Link{Source: source, Target: target, Value: value})
}

// Parse parses a sankey diagram's text, as the parse of Pele's sankey diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. The text is prepared as Pele's prepareTextForParsing prepares it before it is
// parsed, and an error is placed in the text before it was prepared. An error is a
// *diagram.Error.
func Parse(source, title string) (*Model, error) {
	m := newModel()
	m.Title = title
	if err := parseSankey(prepareTextForParsing(source, nil), m, source); err != nil {
		return nil, err
	}
	return m, nil
}

// ParsePrepared parses a prepared sankey diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return common.ParsePrepared(p, false, func(src string) (*Model, error) {
		return Parse(src, p.Title)
	})
}
