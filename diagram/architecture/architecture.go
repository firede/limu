// Package architecture parses Mermaid's architecture diagrams and lays them out, as Pele's
// src/diagrams/architecture does: a tokenizer and a parser that read what Mermaid's Langium
// grammar reads, the db that checks what the text declares, placeOnGrid, which puts the
// services, junctions and groups on a grid of cells as the edges' sides ask, and Layout,
// which sizes the grid's rows and columns and lays the diagram out as a scene.
package architecture

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// A Node is a service or a junction, Pele's ArchNode. A string Pele leaves undefined is
// empty.
type Node struct {
	ID       string
	Junction bool
	// Icon is the name of a service's icon, and IconText the text it shows in its place.
	Icon, IconText string
	Title          string
	// In is the id of the group the node is in.
	In string
}

// A Group is Pele's ArchGroup.
type Group struct {
	ID, Icon, Title, In string
}

// An Edge is Pele's ArchEdge: from the side LhsDir of LhsID to the side RhsDir of RhsID.
// Into is set at an end with an arrowhead, and Group where the edge leaves the node's
// group at its border.
type Edge struct {
	LhsID             string
	LhsDir            Side
	LhsInto, LhsGroup bool
	RhsID             string
	RhsDir            Side
	RhsInto, RhsGroup bool
	Title             string
}

// A LayoutHint is an align directive, Pele's ArchLayoutHint: its members in a row or a
// column, in order.
type LayoutHint struct {
	Direction string // "row" or "column"
	Members   []string
}

// Model is a parsed architecture diagram, Pele's ArchitectureModel. Its nodes and groups are
// in the order they were declared, as Pele's Maps keep them.
type Model struct {
	Title, AccTitle, AccDescr string
	Nodes                     []*Node
	Groups                    []*Group
	Edges                     []Edge
	LayoutHints               []LayoutHint
}

// db is Pele's ArchitectureDb.
type db struct {
	m          *Model
	nodes      map[string]*Node
	groups     map[string]bool
	registered map[string]string // "node" or "group", by id
}

func fail(message string) {
	panic(common.NewError(message, diagram.Semantic, "architecture"))
}

func (d *db) register(kind, id, parent string) {
	if was, ok := d.registered[id]; ok {
		fail("The " + kind + " id [" + id + "] is already in use by another " + was)
	}
	if parent != "" {
		if id == parent {
			fail("The " + kind + " [" + id + "] cannot be placed within itself")
		}
		was, ok := d.registered[parent]
		if !ok {
			fail("The " + kind + " [" + id + "]'s parent does not exist. Please make sure the parent is created before this " + kind)
		}
		if was == "node" {
			fail("The " + kind + " [" + id + "]'s parent is not a group")
		}
	}
	if kind == "group" {
		d.registered[id] = "group"
	} else {
		d.registered[id] = "node"
	}
}

func (d *db) addService(s serviceAst) {
	d.register("service", s.id, s.in)
	n := &Node{ID: s.id, Icon: s.icon, IconText: s.iconText, Title: s.title, In: s.in}
	d.nodes[s.id] = n
	d.m.Nodes = append(d.m.Nodes, n)
}

func (d *db) addJunction(j junctionAst) {
	d.register("junction", j.id, j.in)
	n := &Node{ID: j.id, Junction: true, In: j.in}
	d.nodes[j.id] = n
	d.m.Nodes = append(d.m.Nodes, n)
}

func (d *db) addGroup(g groupAst) {
	d.register("group", g.id, g.in)
	d.groups[g.id] = true
	d.m.Groups = append(d.m.Groups, &Group{ID: g.id, Icon: g.icon, Title: g.title, In: g.in})
}

func (d *db) addEdge(e edgeAst) {
	type end struct {
		side, id string
		crosses  bool
	}
	ends := [2]end{{"left", e.lhsID, e.lhsGroup}, {"right", e.rhsID, e.rhsGroup}}
	for _, x := range ends {
		if _, ok := d.registered[x.id]; !ok {
			fail("The " + x.side + "-hand id [" + x.id + "] does not yet exist. Please create the service/group before declaring an edge to it.")
		}
	}
	// Mermaid lets a group id through the check above and then fails with a TypeError.
	for _, x := range ends {
		if d.nodes[x.id] == nil {
			fail("The " + x.side + "-hand id [" + x.id + "] is a group. Add {group} to a service inside it instead.")
		}
	}
	if parent := d.nodes[e.lhsID].In; parent != "" && parent == d.nodes[e.rhsID].In {
		for _, x := range ends {
			if x.crosses {
				fail("The " + x.side + "-hand id [" + x.id + "] is modified to traverse the group boundary, but the edge does not pass through two groups.")
			}
		}
	}
	d.m.Edges = append(d.m.Edges, Edge{
		LhsID: e.lhsID, LhsDir: e.lhsDir, LhsInto: e.lhsInto, LhsGroup: e.lhsGroup,
		RhsID: e.rhsID, RhsDir: e.rhsDir, RhsInto: e.rhsInto, RhsGroup: e.rhsGroup,
		Title: e.title,
	})
}

func (d *db) addLayoutHint(h LayoutHint) {
	if len(h.Members) < 2 {
		fail("An align directive requires at least two members; got " + chars.NumberString(float64(len(h.Members))))
	}
	seen := map[string]bool{}
	for _, id := range h.Members {
		if d.registered[id] != "node" {
			fail("align " + h.Direction + " references [" + id + "], which is not a service or junction")
		}
		if seen[id] {
			fail("align " + h.Direction + " lists [" + id + "] more than once")
		}
		seen[id] = true
	}
	d.m.LayoutHints = append(d.m.LayoutHints, h)
}

// populate fills the db from the tree, as Pele's populate does after Mermaid's populateDb:
// every group, then services, junctions, edges, and alignments.
func populate(tree *ast, d *db) {
	if tree.accDescr != "" {
		d.m.AccDescr = chars.DropIndents(tree.accDescr)
	}
	if tree.accTitle != "" {
		d.m.AccTitle = chars.TrimStart(tree.accTitle)
	}
	if tree.title != "" {
		d.m.Title = tree.title
	}
	for _, g := range tree.groups {
		d.addGroup(g)
	}
	for _, s := range tree.services {
		d.addService(s)
	}
	for _, j := range tree.junctions {
		d.addJunction(j)
	}
	for _, e := range tree.edges {
		d.addEdge(e)
	}
	for _, a := range tree.alignments {
		d.addLayoutHint(LayoutHint{Direction: a.direction, Members: append([]string(nil), a.members...)})
	}
}

// Parse parses an architecture diagram's text, as the parse of Pele's architecture diagram
// does: source is what diagram.Prepared.Source gives, and title the title its front matter
// gives, "" for none. An error is a *diagram.Error.
func Parse(source, title string) (m *Model, err error) {
	tree, err := parseArchitecture(source)
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := recover(); e != nil {
			de, ok := e.(*diagram.Error)
			if !ok {
				panic(e)
			}
			m, err = nil, de
		}
	}()
	m = &Model{Title: title}
	populate(tree, &db{m: m, nodes: map[string]*Node{}, groups: map[string]bool{}, registered: map[string]string{}})
	return m, nil
}

// ParsePrepared parses a prepared architecture diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return common.ParsePrepared(p, false, func(src string) (*Model, error) {
		return Parse(src, p.Title)
	})
}
