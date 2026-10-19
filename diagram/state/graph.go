package state

import (
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// GraphNode is a node as it is drawn, Pele's StateNode. A string Pele may leave undefined is
// empty here where it is.
type GraphNode struct {
	ID string
	// Shape is rect, rectWithTitle, stateStart, stateEnd, choice, fork, join, divider,
	// roundedWithTitle, note or noteGroup; "" for a note's state, which has none.
	Shape       string
	Label       string
	Description []string
	CSSClasses  string
	// CSSCompiledStyles are the styles of its classes; CSSStyles those of style statements.
	CSSCompiledStyles []string
	CSSStyles         []string
	// Type is "group", "node" or "".
	Type    string
	IsGroup bool
	// Parent is the id of the group it is in, where HasParent is set.
	Parent    string
	HasParent bool
	// Dir is the direction inside a composite: its last `direction` statement, or TB.
	Dir      string
	Position string
	NoteFor  string
	// ColorIndex is the slot of the palette a composite takes, -1 for none.
	ColorIndex int
}

// GraphEdge is an edge as it is drawn, Pele's StateEdge. Labels are markdown throughout, as in
// Mermaid.
type GraphEdge struct {
	ID      string
	Start   string
	End     string
	Label   string
	Classes string
	Pattern string // "dashed" for the line from a note to its state, "" otherwise
}

// Graph is the nodes and edges of a state diagram, Pele's StateGraph.
type Graph struct {
	Nodes []*GraphNode
	Edges []*GraphEdge
}

// The kinds of node buildGraph makes, which differ in the fields they carry, so that one that
// is merged into a node of the same id leaves the fields it does not carry as they were.
const (
	kindState = iota
	kindNoteGroup
	kindNote
)

// draft is a node as buildGraph gathers it across the statements that name it.
type draft struct {
	shape string
	// desc is the description: one string, or a list where descList is set.
	desc       []string
	descList   bool
	cssClasses string
	cssStyles  []string
	typ        string
	dir        string
	colorIndex int
}

// descLength is the length of a draft's description: of the list, or of the string in UTF-16
// code units.
func (d *draft) descLength() int {
	if d.descList {
		return len(d.desc)
	}
	return chars.Len16(d.desc[0])
}

type frame struct {
	item *StateStmt
	doc  []Stmt
	at   int
	alt  bool
}

// docDirection is the direction of a document: its last `direction` statement, or TB.
func docDirection(doc []Stmt) string {
	dir := "TB"
	for _, item := range doc {
		if s, ok := item.(*DirectionStmt); ok {
			dir = s.Value
		}
	}
	return dir
}

// buildGraph walks the documents and resolves them into the nodes and edges that are drawn, as
// Pele's buildStateGraph does: Mermaid's dataFetcher.
func buildGraph(rootDoc []Stmt, states map[string]*StateStmt, classes map[string]*ClassDef) (*Graph, error) {
	g := &Graph{}
	byID := map[string]*GraphNode{}
	drafts := map[string]*draft{}
	slots := map[string]int{}
	counter := 0
	nextSlot := 0

	insert := func(node *GraphNode, kind int) {
		// The two literal ids are what DOMPurify leaves of `<<fork>>` and `<<choice>>` in
		// Mermaid; kept for parity.
		if node.ID == "" || node.ID == "</join></fork>" || node.ID == "</choice>" {
			return
		}
		if node.CSSClasses != "" {
			// One copy, however many classes: a state named in a class statement many times has
			// a long list.
			var compiled []string
			copied := false
			for _, name := range strings.Split(node.CSSClasses, " ") {
				def := classes[name]
				if def == nil {
					continue
				}
				if !copied {
					compiled = append([]string{}, node.CSSCompiledStyles...)
					copied = true
				}
				compiled = append(compiled, def.Styles...)
			}
			if copied {
				node.CSSCompiledStyles = compiled
			}
		}
		existing := byID[node.ID]
		if existing == nil {
			byID[node.ID] = node
			g.Nodes = append(g.Nodes, node)
			return
		}
		parent, hasParent := node.Parent, node.HasParent
		if !hasParent {
			parent, hasParent = existing.Parent, existing.HasParent
		}
		// Object.assign: each field the new node carries replaces the existing one's.
		existing.ID = node.ID
		existing.Shape = node.Shape
		existing.Label = node.Label
		existing.Description = node.Description
		existing.CSSClasses = node.CSSClasses
		existing.CSSCompiledStyles = node.CSSCompiledStyles
		existing.CSSStyles = node.CSSStyles
		existing.Type = node.Type
		existing.IsGroup = node.IsGroup
		existing.Dir = node.Dir
		switch kind {
		case kindState:
			existing.ColorIndex = node.ColorIndex
		case kindNoteGroup:
			existing.Position = node.Position
		case kindNote:
			existing.Position = node.Position
			existing.NoteFor = node.NoteFor
		}
		if hasParent {
			existing.Parent, existing.HasParent = parent, true
		}
	}

	visit := func(parent, item *StateStmt, alt bool) {
		id := item.ID
		if id == "root" {
			return
		}
		known := states[id]
		classStr := ""
		styles := []string{}
		if known != nil {
			classStr = strings.Join(known.Classes, " ")
			if known.Styles != nil {
				styles = known.Styles
			}
		}

		dr := drafts[id]
		if dr == nil {
			shape := "rect"
			if item.Start != nil {
				if *item.Start {
					shape = "stateStart"
				} else {
					shape = "stateEnd"
				}
			}
			if item.Type != "default" {
				shape = item.Type
			}
			dr = &draft{shape: shape, desc: []string{id}, cssClasses: classStr + " statediagram-state",
				cssStyles: styles, colorIndex: -1}
			drafts[id] = dr
		}

		if item.Description != nil && (item.DescriptionList || item.Description[0] != "") {
			switch {
			case dr.descList:
				dr.shape = "rectWithTitle"
				dr.desc = append(append([]string{}, dr.desc...), item.Description...)
			case dr.desc[0] != "":
				dr.shape = "rectWithTitle"
				if dr.desc[0] == id {
					dr.desc = append([]string{}, item.Description...)
				} else {
					dr.desc = append([]string{dr.desc[0]}, item.Description...)
				}
				dr.descList = true
			default:
				dr.shape = "rect"
				dr.desc = append([]string{}, item.Description...)
				dr.descList = item.DescriptionList
			}
		}
		if dr.descLength() == 1 && dr.shape == "rectWithTitle" {
			if dr.typ == "group" {
				dr.shape = "roundedWithTitle"
			} else {
				dr.shape = "rect"
			}
		}

		if dr.typ == "" && item.Doc != nil {
			dr.typ = "group"
			dr.dir = docDirection(item.Doc)
			if item.Type == "divider" {
				dr.shape = "divider"
			} else {
				dr.shape = "roundedWithTitle"
			}
			// A container takes the next palette slot unless it is a region, which shares its
			// composite's.
			slot, shared := 0, false
			if dr.shape == "divider" && parent != nil {
				slot, shared = slots[parent.ID]
			}
			if shared {
				dr.colorIndex = slot
			} else {
				slot := nextSlot
				nextSlot++
				if chars.Trim(classStr) != "" || len(styles) > 0 {
					dr.colorIndex = -1
				} else {
					dr.colorIndex = slot
				}
			}
			slots[id] = dr.colorIndex
			alternate := ""
			if alt {
				alternate = "statediagram-cluster-alt"
			}
			dr.cssClasses = dr.cssClasses + " statediagram-cluster " + alternate
		}

		node := &GraphNode{
			ID:          id,
			Shape:       dr.shape,
			Description: []string{},
			CSSClasses:  dr.cssClasses,
			CSSStyles:   dr.cssStyles,
			Type:        dr.typ,
			IsGroup:     dr.typ == "group",
			Dir:         dr.dir,
			ColorIndex:  dr.colorIndex,
		}
		node.CSSCompiledStyles = []string{}
		if dr.descList {
			node.Label = dr.desc[0]
			node.Description = append([]string{}, dr.desc[1:]...)
		} else {
			node.Label = dr.desc[0]
		}
		if node.Shape == "divider" {
			node.Label = ""
		}
		if parent != nil && parent.ID != "root" {
			node.Parent, node.HasParent = parent.ID, true
		}

		note := item.Note
		if note == nil {
			insert(node, kindState)
			return
		}
		// Mermaid puts each note in a group of its own, named after the state. Pele keeps that
		// group in the model and lays the note out beside its state instead.
		aside := func(asideID, shape, cssClasses string, isGroup bool) *GraphNode {
			typ := "node"
			if isGroup {
				typ = "group"
			}
			return &GraphNode{ID: asideID, Shape: shape, Label: note.Text, Description: []string{},
				CSSClasses: cssClasses, CSSCompiledStyles: []string{}, CSSStyles: []string{}, Type: typ,
				IsGroup: isGroup, Position: note.Position, ColorIndex: -1}
		}
		noteNode := aside(id+"----note-"+strconv.Itoa(counter), "note", "statediagram-note", false)
		counter++
		noteNode.Parent, noteNode.HasParent = id+"----parent", true
		noteNode.NoteFor = id
		insert(aside(noteNode.Parent, "noteGroup", dr.cssClasses, true), kindNoteGroup)
		insert(noteNode, kindNote)
		insert(node, kindState)
		from, to := id, noteNode.ID
		if note.Position == "left of" {
			from, to = noteNode.ID, id
		}
		g.Edges = append(g.Edges, &GraphEdge{ID: from + "-" + to, Start: from, End: to, Label: "",
			Classes: "transition note-edge", Pattern: "dashed"})
	}

	stack := []*frame{{doc: rootDoc, alt: true}}
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		if f.at >= len(f.doc) {
			stack = stack[:len(stack)-1]
			continue
		}
		item := f.doc[f.at]
		f.at++
		switch s := item.(type) {
		case *StateStmt:
			visit(f.item, s, f.alt)
			if s.Doc != nil {
				stack = append(stack, &frame{item: s, doc: s.Doc, alt: !f.alt})
			}
		case *RelationStmt:
			visit(f.item, s.State1, f.alt)
			visit(f.item, s.State2, f.alt)
			g.Edges = append(g.Edges, &GraphEdge{ID: "edge" + strconv.Itoa(counter), Start: s.State1.ID,
				End: s.State2.ID, Label: s.Description, Classes: "transition"})
			counter++
		}
	}

	for _, node := range g.Nodes {
		if node.IsGroup && len(node.Description) > 0 {
			return nil, diagram.NewError("Group nodes can only have label. Remove the additional description for node ["+
				node.ID+"]", diagram.Semantic, "state")
		}
	}
	return g, nil
}
