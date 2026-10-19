package agentflow

import (
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/yaml"
)

// GraphNode is a node or a flow as it is drawn, Pele's GraphNode. A string Pele may leave
// undefined is empty where it is, but for Parent, whose HasParent says.
type GraphNode struct {
	ID        string
	Label     string
	LabelType LabelType
	// Shape is the shape it is drawn with: one of the shapes agent flows draw, flowGroup for
	// a flow, or collapsedGroup for a collapsed one.
	Shape   string
	IsGroup bool
	// Parent is the id of the flow it is in, if HasParent; a flow's id may be "".
	Parent            string
	HasParent         bool
	CSSStyles         []string
	CSSCompiledStyles []string
	CSSClasses        string
	Dir               string
	Link              string
	LinkTarget        string
	Tooltip           string
	// ColorIndex is the palette slot of a flow, collapsed or not, and -1 for a node.
	ColorIndex int
	// Kind is what a node is, which picks its palette slot: "tool", "task", "decision",
	// "input", "refdoc", "connector" or "action"; "" for a flow.
	Kind     string
	Metadata *yaml.Map
}

// GraphEdge is an edge as it is drawn, Pele's GraphEdge.
type GraphEdge struct {
	ID              string
	IsUserDefinedID bool
	Start           string
	End             string
	Type            string
	Label           string
	LabelType       LabelType
	Thickness       string
	Pattern         string
	// MinLen is the edge's length in ranks, 0 where it has none.
	MinLen            int
	ArrowTypeStart    string
	ArrowTypeEnd      string
	CSSCompiledStyles []string
	Style             []string
	Classes           string
	Animate           any
	Animation         any
	// Curve is the curve the edge is drawn with: its own, the diagram's default, or the
	// default the drawing is given, and nil where there is none. It is a string but where the
	// edge's metadata gives its curve another value.
	Curve    any
	Metadata *yaml.Map
}

// Graph is what an agent flow draws, Pele's AgentGraph.
type Graph struct {
	Nodes []*GraphNode
	Edges []*GraphEdge
}

// Palette is the number of series colors, the palette Mermaid's slot rules are applied to.
const Palette = 8

// kindSlot is the palette slot of each kind of node: fixed, so that editing a diagram
// around a node never recolors it.
var kindSlot = map[string]int{"tool": 0, "task": 1, "decision": 2, "input": 3, "refdoc": 4, "connector": 5, "action": 6}

const kindCount = 7

// containerSlot is the palette slot of the nth container, counted through the slots above the
// kinds, as Pele's containerSlot gives it.
func containerSlot(n, paletteLength int) int {
	count := max(1, paletteLength-kindCount)
	slot := kindCount + n%count
	if paletteLength > 0 {
		return slot % paletteLength
	}
	return slot
}

func trimAll(s []string) []string {
	for i := range s {
		s[i] = chars.Trim(s[i])
	}
	return s
}

func collapsedView(m *yaml.Map) bool {
	v, _ := m.Value("view").(string)
	return v == "collapsed"
}

// orderedSet is a JavaScript Set of strings: it iterates in the order its keys were added.
type orderedSet struct {
	keys []string
	has  map[string]bool
}

func (s *orderedSet) add(k string) {
	if s.has == nil {
		s.has = map[string]bool{}
	}
	if !s.has[k] {
		s.has[k] = true
		s.keys = append(s.keys, k)
	}
}

// Graph is the graph the diagram draws, built once, as Pele's agentGraph builds it: building
// it records the model's shape and containment diagnostics. defaultCurve is the flowchart
// config's curve, "" where it has none, and is read only the first time.
func (d *DB) Graph(defaultCurve string) *Graph {
	if d.graph == nil {
		d.graph = BuildGraph(d, Palette, defaultCurve)
	}
	return d.graph
}

// BuildGraph resolves a parsed agent flow into the nodes and edges that are drawn, as Pele's
// buildAgentGraph does: flow membership, collapsed flows, shapes reduced to the ones agent
// flows have, class styles and palette slots. Shape and containment problems are recorded on
// the model as diagnostics.
func BuildGraph(d *DB, paletteLength int, defaultCurve string) *Graph {
	g := &Graph{Nodes: []*GraphNode{}, Edges: []*GraphEdge{}}
	subgraphs := d.Subgraphs

	compiled := func(classNames []string) []string {
		out := []string{}
		for _, name := range classNames {
			def := d.classes[name]
			if def == nil {
				continue
			}
			out = trimAll(append(out, def.Styles...))
			out = trimAll(append(out, def.TextStyles...))
		}
		return out
	}

	// Everything inside a collapsed flow is hidden, and stands for its outermost collapsed
	// flow. Mermaid walks the whole inside of each collapsed flow, in the order the flows
	// closed, and the last walk to reach a thing wins. A thing has one container, so a later
	// walk can only enter an earlier one through its root: it stops there, and the root's
	// owner is followed afterwards.
	var hidden orderedSet
	owner := map[string]string{}
	walked := map[string]bool{}
	for _, sg := range subgraphs {
		if !collapsedView(sg.Metadata) || hidden.has[sg.ID] {
			continue
		}
		pending := []string{sg.ID}
		for len(pending) > 0 {
			top := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			inner := d.subgraphLookup[top]
			if inner == nil {
				continue
			}
			for _, childID := range inner.Nodes {
				if childID == sg.ID || hidden.has[childID] {
					continue
				}
				hidden.add(childID)
				owner[childID] = sg.ID
				if !walked[childID] {
					pending = append(pending, childID)
				}
			}
		}
		walked[sg.ID] = true
	}
	collapsedInto := map[string]string{}
	for _, id := range hidden.keys {
		path := []string{id}
		top := owner[id]
		for {
			known, isKnown := collapsedInto[top]
			up, ok := known, isKnown
			if !isKnown {
				up, ok = owner[top]
			}
			if !ok {
				break
			}
			if !isKnown {
				path = append(path, top)
			}
			top = up
			if isKnown {
				break
			}
		}
		for _, p := range path {
			collapsedInto[p] = top
		}
	}

	position := map[string]int{}
	for i, sg := range subgraphs {
		position[sg.ID] = i
	}

	// Two flows can each name the other. The nesting that would close the loop is dropped.
	parentOf := map[string]string{}
	closesLoop := func(childID, parentID string, at int) bool {
		// Only a flow handled earlier in this pass can already be above the parent.
		childAt, ok := position[childID]
		if !ok || childAt < at {
			return false
		}
		current, has := parentID, true
		for hops := 0; has && hops <= len(subgraphs); hops++ {
			if current == childID {
				return true
			}
			current, has = parentOf[current]
		}
		return false
	}
	for i := len(subgraphs) - 1; i >= 0; i-- {
		sg := subgraphs[i]
		if hidden.has[sg.ID] {
			continue
		}
		for _, id := range sg.Nodes {
			if closesLoop(id, sg.ID, i) {
				d.EmitWarning(ContainmentViolation, `Container "`+sg.ID+`" cannot contain "`+id+`" because "`+id+
					`" already contains it. The nesting that would close the loop is dropped.`, id)
				continue
			}
			parentOf[id] = sg.ID
		}
	}

	byID := map[string]*GraphNode{}
	for i := len(subgraphs) - 1; i >= 0; i-- {
		sg := subgraphs[i]
		if hidden.has[sg.ID] {
			continue
		}
		collapsed := collapsedView(sg.Metadata)
		node := &GraphNode{
			ID:                sg.ID,
			Label:             sg.Title,
			LabelType:         sg.LabelType,
			CSSStyles:         []string{},
			CSSCompiledStyles: compiled(sg.Classes),
			CSSClasses:        strings.Join(sg.Classes, " "),
			Shape:             "flowGroup",
			Dir:               sg.Dir,
			IsGroup:           !collapsed,
			Metadata:          sg.Metadata,
			ColorIndex:        -1,
		}
		node.Parent, node.HasParent = parentOf[sg.ID]
		if collapsed {
			node.Shape = "collapsedGroup"
			node.Metadata = sg.Metadata.Spread(nil)
			node.Metadata.Set("containerType", sg.Type)
		}
		g.Nodes = append(g.Nodes, node)
		byID[sg.ID] = node
	}

	for _, vertex := range d.Nodes {
		if hidden.has[vertex.ID] {
			continue
		}
		if existing := byID[vertex.ID]; existing != nil {
			existing.CSSStyles = vertex.Styles
			existing.CSSCompiledStyles = compiled(vertex.Classes)
			existing.CSSClasses = strings.Join(vertex.Classes, " ")
			continue
		}
		typ := ResolveShapeAlias(vertex.Type)
		if typ == "square" || typ == "" {
			typ = "squareRect"
		}
		tooltip, _ := d.Tooltip(vertex.ID)
		node := &GraphNode{
			ID:                vertex.ID,
			Label:             vertex.Text,
			LabelType:         vertex.LabelType,
			CSSStyles:         vertex.Styles,
			CSSCompiledStyles: compiled(append([]string{"default", "node"}, vertex.Classes...)),
			CSSClasses:        "default " + strings.Join(vertex.Classes, " "),
			Dir:               vertex.Dir,
			Link:              vertex.Link,
			LinkTarget:        vertex.LinkTarget,
			Tooltip:           tooltip,
			Metadata:          vertex.Metadata,
			Shape:             typ,
			ColorIndex:        -1,
		}
		node.Parent, node.HasParent = parentOf[vertex.ID]
		g.Nodes = append(g.Nodes, node)
		byID[vertex.ID] = node
	}

	for index, raw := range d.Edges {
		start, end := raw.Start, raw.End
		if s, ok := collapsedInto[start]; ok {
			start = s
		}
		if e, ok := collapsedInto[end]; ok {
			end = e
		}
		// An edge between two things inside one collapsed flow has nothing left to connect.
		if start == end && raw.Start != raw.End {
			continue
		}
		styles := append([]string{}, d.DefaultEdgeStyle...)
		styles = append(styles, raw.Style...)
		arrow := "arrow_point"
		switch raw.Type {
		case "arrow_open":
			arrow = "none"
		case "arrow_cross", "arrow_point":
			arrow = raw.Type
		}
		id := raw.ID
		if id == "" {
			id = "L_" + start + "_" + end + "_" + strconv.Itoa(index)
		}
		typ := raw.Type
		if typ == "" {
			typ = "normal"
		}
		var curve any
		switch {
		case yaml.Truthy(raw.Interpolate):
			curve = raw.Interpolate
		case d.DefaultInterpolate != "":
			curve = d.DefaultInterpolate
		case defaultCurve != "":
			curve = defaultCurve
		}
		g.Edges = append(g.Edges, &GraphEdge{
			ID:                id,
			IsUserDefinedID:   raw.IsUserDefinedID,
			Start:             start,
			End:               end,
			Type:              typ,
			Label:             raw.Text,
			LabelType:         raw.LabelType,
			Thickness:         raw.Stroke,
			Pattern:           raw.Stroke,
			MinLen:            raw.Length,
			Classes:           "edge-thickness-normal edge-pattern-solid flowchart-link",
			ArrowTypeStart:    "none",
			ArrowTypeEnd:      arrow,
			CSSCompiledStyles: compiled(raw.Classes),
			Style:             styles,
			Animate:           raw.Animate,
			Animation:         raw.Animation,
			Curve:             curve,
			Metadata:          raw.Metadata,
		})
	}

	normaliseNodeShapes(g.Nodes, d)

	// Containers are numbered in the order they were written: a walk of the nesting, parents
	// first. Subgraphs itself is in the order the blocks closed, with a nested flow before its
	// parent.
	children := map[string][]string{}
	for _, sg := range subgraphs {
		if parent, ok := parentOf[sg.ID]; ok {
			children[parent] = append(children[parent], sg.ID)
		}
	}
	order := map[string]int{}
	for _, sg := range subgraphs {
		if _, ok := parentOf[sg.ID]; ok {
			continue
		}
		pending := []string{sg.ID}
		for len(pending) > 0 {
			id := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if _, ok := order[id]; ok {
				continue
			}
			order[id] = len(order)
			kids := children[id]
			for i := len(kids) - 1; i >= 0; i-- {
				pending = append(pending, kids[i])
			}
		}
	}

	// assignColorSlots: a node takes the color of its kind, from a fixed slot; containers
	// count through the slots above the kinds, in the order they were declared. A collapsed
	// container is drawn as a node but keeps its slot, so collapsing one does not recolor the
	// containers after it.
	fallback := len(order)
	for _, node := range g.Nodes {
		if node.IsGroup || node.Shape == "collapsedGroup" {
			n, ok := order[node.ID]
			if !ok {
				n = fallback
				fallback++
			}
			node.ColorIndex = containerSlot(n, paletteLength)
			continue
		}
		kind := ""
		if d.connectors[node.ID] != nil {
			kind = "connector"
		} else if v := d.Node(node.ID); v != nil {
			kind = d.VertexKind(v)
		}
		if _, ok := kindSlot[kind]; ok {
			node.Kind = kind
		}
	}
	return g
}
