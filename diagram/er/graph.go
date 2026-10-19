package er

import (
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
)

// graphNode is an entity or a subgraph as it is drawn, Pele's ErGraphNode.
type graphNode struct {
	ID         string
	Label      string
	Alias      string
	Attributes []Attribute
	IsGroup    bool
	// Parent is the id of the subgraph it is in, if HasParent; a subgraph's id may be "".
	Parent            string
	HasParent         bool
	CSSStyles         []string
	CSSCompiledStyles []string
	CSSClasses        string
	// Dir is the direction a subgraph sets, or "".
	Dir       string
	LabelType string
}

// graphEdge is a relationship as it is drawn, Pele's ErGraphEdge: the marker types at its
// ends are Mermaid's cardinalities in lower case, and its pattern "solid" or "dashed".
type graphEdge struct {
	ID             string
	Start          string
	End            string
	Label          string
	ArrowTypeStart string
	ArrowTypeEnd   string
	Pattern        string
}

// graph is what an ER diagram draws, Pele's ErGraph.
type graph struct {
	Nodes []*graphNode
	Edges []*graphEdge
}

// buildGraph resolves a parsed ER diagram into what is drawn, as Pele's buildErGraph does:
// subgraphs as groups, entities as nodes with their class styles, and relationships as
// edges with a marker type at each end.
func buildGraph(db *DB) *graph {
	g := &graph{}
	subgraphs := db.Subgraphs

	// compiled is the styles of the classes named, each trimmed, the text styles of each
	// after its styles, as the compiled of Pele's buildErGraph gives them.
	compiled := func(classNames []string) []string {
		out := []string{}
		for _, name := range classNames {
			def := db.classes[name]
			if def == nil {
				continue
			}
			for _, s := range def.Styles {
				out = append(out, chars.Trim(s))
			}
			for _, s := range def.TextStyles {
				out = append(out, chars.Trim(s))
			}
		}
		return out
	}

	parentOf := map[string]string{}
	subgraphIDs := map[string]bool{}
	for i := len(subgraphs) - 1; i >= 0; i-- {
		s := subgraphs[i]
		subgraphIDs[s.ID] = true
		for _, id := range s.Nodes {
			parentOf[id] = s.ID
		}
	}

	for i := len(subgraphs) - 1; i >= 0; i-- {
		s := subgraphs[i]
		parent, has := parentOf[s.ID]
		g.Nodes = append(g.Nodes, &graphNode{
			ID:                s.ID,
			Label:             s.Title,
			IsGroup:           true,
			Parent:            parent,
			HasParent:         has,
			CSSStyles:         s.CSSStyles,
			CSSCompiledStyles: compiled(s.Classes),
			CSSClasses:        strings.Join(s.Classes, " "),
			Dir:               s.Dir,
			LabelType:         s.LabelType,
		})
	}

	// A name that is also a subgraph id stands for the subgraph. Mermaid drops such an entity
	// and leaves relationships written before the subgraph pointing at nothing; here they
	// follow it.
	moved := map[string]string{}
	for _, e := range db.Entities {
		if subgraphIDs[e.Label] {
			moved[e.ID] = e.Label
			continue
		}
		parent, has := parentOf[e.Label]
		g.Nodes = append(g.Nodes, &graphNode{
			ID:                e.ID,
			Label:             e.Label,
			Alias:             e.Alias,
			Attributes:        e.Attributes,
			Parent:            parent,
			HasParent:         has,
			CSSStyles:         e.CSSStyles,
			CSSCompiledStyles: compiled(strings.Split(e.CSSClasses, " ")),
			CSSClasses:        e.CSSClasses,
			LabelType:         "markdown",
		})
	}

	for i, r := range db.Relationships {
		start, ok := moved[r.EntityA]
		if !ok {
			start = r.EntityA
		}
		end, ok := moved[r.EntityB]
		if !ok {
			end = r.EntityB
		}
		pattern := "dashed"
		if r.RelSpec.RelType == Identifying {
			pattern = "solid"
		}
		g.Edges = append(g.Edges, &graphEdge{
			ID:             "id_" + r.EntityA + "_" + r.EntityB + "_" + strconv.Itoa(i),
			Start:          start,
			End:            end,
			Label:          r.RoleA,
			ArrowTypeStart: strings.ToLower(r.RelSpec.CardB),
			ArrowTypeEnd:   strings.ToLower(r.RelSpec.CardA),
			Pattern:        pattern,
		})
	}
	return g
}
