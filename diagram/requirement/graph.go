package requirement

import (
	"strconv"
	"strings"
)

// A GraphNode is a requirement or an element to lay out, as Pele's GraphNode. Exactly one of
// Requirement and Element is set.
type GraphNode struct {
	ID          string
	Requirement *Requirement
	Element     *Element
	CSSStyles   []string
	CSSClasses  string
}

// A GraphEdge is a relationship to lay out, as Pele's GraphEdge. HasStart and HasEnd say
// whether Start and End name a requirement or an element.
type GraphEdge struct {
	ID               string
	Start, End       string
	HasStart, HasEnd bool
	Type             string
	Contains         bool
}

// Graph is what BuildGraph makes of a DB, as Pele's RequirementGraph.
type Graph struct {
	Nodes     []GraphNode
	Edges     []GraphEdge
	Direction string
}

// BuildGraph is the nodes and edges to lay out, as Pele's buildRequirementGraph makes them:
// requirements first, then elements, as Mermaid orders them.
func BuildGraph(db *DB) *Graph {
	g := &Graph{Direction: db.Direction}
	for _, r := range db.Requirements {
		g.Nodes = append(g.Nodes, GraphNode{ID: r.Name, Requirement: r, CSSStyles: r.CSSStyles, CSSClasses: strings.Join(r.Classes, " ")})
	}
	for _, e := range db.Elements {
		g.Nodes = append(g.Nodes, GraphNode{ID: e.Name, Element: e, CSSStyles: e.CSSStyles, CSSClasses: strings.Join(e.Classes, " ")})
	}
	known := func(id string) bool { return db.requirements[id] != nil || db.elements[id] != nil }
	for i, r := range db.Relations {
		g.Edges = append(g.Edges, GraphEdge{
			ID:       r.Src + "-" + r.Dst + "-" + strconv.Itoa(i),
			Start:    r.Src,
			End:      r.Dst,
			HasStart: known(r.Src),
			HasEnd:   known(r.Dst),
			Type:     r.Type,
			Contains: r.Type == Contains,
		})
	}
	return g
}
