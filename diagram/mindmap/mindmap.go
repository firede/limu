// Package mindmap reads Mermaid's mindmaps and lays them out, as Pele's src/diagrams/mindmap
// does: an indented outline read with the grammar mindmaps share with kanban boards, made
// into a tree of nodes, and laid out with the first-level branches spread to both sides of
// the root, or, in a narrow host, as an outline running down.
package mindmap

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// The shapes of a node, Pele's NODE_TYPE, by the delimiters about its text.
const (
	TypeDefault     = common.NodeDefault
	TypeRoundedRect = common.NodeRoundedRect
	TypeRect        = common.NodeRect
	TypeCircle      = common.NodeCircle
	TypeCloud       = common.NodeCloud
	TypeBang        = common.NodeBang
	TypeHexagon     = common.NodeHexagon
)

// A Node is a node of the map, Pele's MindmapNode.
type Node struct {
	ID       int    // its index in Model.Nodes
	NodeID   string // the id its text gave it, or its text
	Level    int    // its indentation less the root's, in UTF-16 code units
	Descr    string // its text
	Type     int    // its shape, one of the Type constants
	Children []*Node
	IsRoot   bool
	// Section is the index of the first-level branch the node is in, from 0 and wrapping
	// after eleven, as in Mermaid. The root is in none; its Section is 0.
	Section int
	// Class and Icon are what `:::` and `::icon()` lines gave the node, "" for none.
	Class string
	Icon  string
}

// Model is a parsed mindmap, Pele's MindmapModel.
type Model struct {
	Title string // from front matter, "" for none
	// Nodes are in source order, which is depth first. The first node is the root.
	Nodes []*Node
}

// sections are the branches before their colors start again, Pele's SECTIONS.
const sections = 11

// db builds a Model as Pele's MindmapDb does.
type db struct {
	model     *Model
	baseLevel int
	// open is the chain of nodes that can still take children, with levels strictly rising.
	open []*Node
}

func (d *db) GetType(start, end string) int { return common.OutlineType(start, end) }

// AddNode adds a node under the last node with a smaller indentation, wherever that is in
// the tree, as Pele's addNode does after Mermaid.
func (d *db) AddNode(level int, id, descr string, typ int, _ string, _ bool) error {
	nodes := d.model.Nodes
	isRoot := len(nodes) == 0
	if isRoot {
		d.baseLevel = level
	}
	level -= d.baseLevel

	depth := len(d.open)
	for depth > 0 && d.open[depth-1].Level >= level {
		depth--
	}
	var parent *Node
	if depth > 0 {
		parent = d.open[depth-1]
	}
	if parent == nil && !isRoot {
		return diagram.NewError(`There can be only one root. No parent could be found for ("`+descr+`")`, diagram.Semantic, "mindmap")
	}
	node := &Node{ID: len(nodes), NodeID: id, Level: level, Descr: descr, Type: typ, Children: []*Node{}, IsRoot: isRoot}
	if parent != nil {
		if parent.IsRoot {
			node.Section = len(parent.Children) % sections
		} else {
			node.Section = parent.Section
		}
		parent.Children = append(parent.Children, node)
	}
	d.open = append(d.open[:depth], node)
	d.model.Nodes = append(nodes, node)
	return nil
}

// DecorateNode gives the last node an icon or a class, as Pele's decorateNode does. Mermaid
// throws a TypeError when a decoration comes before any node. Here it is ignored.
func (d *db) DecorateNode(text string, class bool) {
	nodes := d.model.Nodes
	if len(nodes) == 0 || text == "" {
		return
	}
	node := nodes[len(nodes)-1]
	if class {
		node.Class = text
	} else {
		node.Icon = text
	}
}

// Tokenize reads a mindmap's tokens, as Pele's tokenize does.
func Tokenize(src string) common.OutlineTokens {
	return common.TokenizeOutline(src, false)
}

// Parse parses a mindmap's text, as the parse of Pele's mindmap diagram does: source is what
// diagram.Prepared.Source gives, and title the title its front matter gives, "" for none. An
// error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	m := &Model{Title: title, Nodes: []*Node{}}
	if err := common.ParseOutline(source, &db{model: m}, false); err != nil {
		return nil, err
	}
	return m, nil
}

// ParsePrepared parses a prepared mindmap.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}
