package class

// A graphNode is a thing drawn: a namespace, a class, a note or an interface. Pele's GraphNode.
type graphNode struct {
	ID      string
	Label   string
	IsGroup bool
	Shape   string // "rect" for a namespace, "classBox", "note" or "interface"
	// ParentID is the namespace it is drawn in, or "", and Parent its index among the nodes,
	// or -1.
	ParentID string
	Parent   int
	Class    *Class
	// ColorIndex is its position among the classes, in declaration order; -1 for namespaces
	// and notes, which have none.
	ColorIndex int
}

// A graphEdge is a line drawn: a relation, or a note's link to its class. Pele's GraphEdge.
type graphEdge struct {
	ID         string
	Start, End string
	// Source and Target are indexes into the nodes, or -1. Ids alone are ambiguous: a class
	// may be named `note0` or `interface0`.
	Source, Target  int
	Label           string
	HasLabel        bool
	ArrowTypeStart  string // "aggregation", "extension", "composition", "dependency", "lollipop" or "none"
	ArrowTypeEnd    string
	StartLabelRight string
	EndLabelLeft    string
	Pattern         string // "solid", "dashed" or "dotted"
	Classes         string
}

// A Graph is what is drawn of a class diagram. Pele's ClassGraph.
type graph struct {
	Nodes []*graphNode
	Edges []*graphEdge
}

// markerNames are the markers of the ends of a relation, by End: Pele's MARKERS.
var markerNames = [...]string{"aggregation", "extension", "composition", "dependency", "lollipop"}

// marker is the name of the marker at an end of a relation, "none" for NoEnd, as Pele's
// marker gives it.
func marker(e End) string {
	if e >= 0 && int(e) < len(markerNames) {
		return markerNames[e]
	}
	return "none"
}

// buildGraph resolves a parsed class diagram into the nodes and edges that are drawn, as
// Pele's buildClassGraph does. With hierarchical off, only namespaces the text declares are
// kept, and each is labeled with its full name.
func buildGraph(db *DB, hierarchical bool) *graph {
	g := &graph{}
	namespaceAt := map[string]int{}
	classAt := map[string]int{}
	interfaceAt := map[string]int{}

	declared := map[string]string{}
	// declaredAncestor is the nearest namespace the text declares that id is, or is in, or ""
	// for none, as the declaredAncestor of Pele's buildClassGraph finds it.
	declaredAncestor := func(id string) string {
		if hierarchical || id == "" {
			return id
		}
		var chain []string
		current := id
		found := ""
		for current != "" {
			if d, ok := declared[current]; ok {
				found = d
				break
			}
			ns := db.namespaces[current]
			if ns == nil {
				break
			}
			if ns.Explicit {
				found = current
				break
			}
			chain = append(chain, current)
			current = ns.Parent
		}
		for _, visited := range chain {
			declared[visited] = found
		}
		return found
	}
	// group is the index among the nodes of the namespace of id, or -1, as the group of Pele's
	// buildClassGraph gives it.
	group := func(id string) int {
		if id == "" {
			return -1
		}
		if i, ok := namespaceAt[id]; ok {
			return i
		}
		return -1
	}

	for _, ns := range db.Namespaces {
		if !hierarchical && !ns.Explicit {
			continue
		}
		namespaceAt[ns.ID] = len(g.Nodes)
		n := &graphNode{ID: ns.ID, Label: ns.ID, IsGroup: true, Shape: "rect", Parent: -1, ColorIndex: -1}
		if hierarchical {
			n.Label, n.ParentID = ns.Label, ns.Parent
		}
		g.Nodes = append(g.Nodes, n)
	}
	for _, n := range g.Nodes {
		n.Parent = group(n.ParentID)
	}

	for i, c := range db.Classes {
		parentID := declaredAncestor(c.Parent)
		classAt[c.ID] = len(g.Nodes)
		g.Nodes = append(g.Nodes, &graphNode{ID: c.ID, Label: c.Label, Shape: "classBox", ParentID: parentID,
			Parent: group(parentID), Class: c, ColorIndex: i})
	}

	for _, note := range db.Notes {
		parentID := declaredAncestor(note.Parent)
		if note.Class != "" {
			if target, ok := classAt[note.Class]; ok {
				g.Edges = append(g.Edges, &graphEdge{
					ID: "edgeNote" + itoa(note.Index), Start: note.ID, End: g.Nodes[target].ID,
					Source: len(g.Nodes), Target: target, ArrowTypeStart: "none", ArrowTypeEnd: "none",
					Pattern: "dotted", Classes: "relation",
				})
			}
		}
		g.Nodes = append(g.Nodes, &graphNode{ID: note.ID, Label: note.Text, Shape: "note", ParentID: parentID,
			Parent: group(parentID), ColorIndex: -1})
	}

	// Mermaid leaves every lollipop interface outside the namespaces. Here it stays beside
	// its class.
	for _, item := range db.Interfaces {
		parent := -1
		if owner, ok := classAt[item.ClassID]; ok {
			parent = g.Nodes[owner].Parent
		}
		interfaceAt[item.ID] = len(g.Nodes)
		g.Nodes = append(g.Nodes, &graphNode{ID: item.ID, Label: item.Label, Shape: "interface", Parent: parent,
			ColorIndex: -1})
	}

	for k, r := range db.Relations {
		// addRelation swaps in an interface for the lollipop end of a relation with no other
		// marker.
		startAt, endAt := classAt, classAt
		if r.Type1 == Lollipop && r.Type2 == NoEnd {
			startAt = interfaceAt
		} else if r.Type2 == Lollipop && r.Type1 == NoEnd {
			endAt = interfaceAt
		}
		e := &graphEdge{
			ID: "id_" + r.ID1 + "_" + r.ID2 + "_" + itoa(k+1), Start: r.ID1, End: r.ID2,
			Source: index(startAt, r.ID1), Target: index(endAt, r.ID2), Label: r.Title, HasLabel: r.HasTitle,
			ArrowTypeStart: marker(r.Type1), ArrowTypeEnd: marker(r.Type2), Pattern: "solid", Classes: "relation",
		}
		if r.RelationTitle1 != "none" {
			e.StartLabelRight = r.RelationTitle1
		}
		if r.RelationTitle2 != "none" {
			e.EndLabelLeft = r.RelationTitle2
		}
		if r.LineType == LineDotted {
			e.Pattern = "dashed"
		}
		g.Edges = append(g.Edges, e)
	}
	return g
}

// index is the index m holds for id, or -1: the `get(id) ?? -1` of Pele's buildClassGraph.
func index(m map[string]int, id string) int {
	if i, ok := m[id]; ok {
		return i
	}
	return -1
}
