package flowchart

import (
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/yaml"
)

// GraphNode is a node or a group as it is drawn, Pele's GraphNode. A string Pele may leave
// undefined is empty where it is, but for Parent, whose HasParent says.
type GraphNode struct {
	ID        string
	Label     string
	LabelType LabelType
	// Shape is the shape it is drawn with: a name canonicalShape knows, or one of
	// squareRect, roundedRect, imageSquare, icon, iconCircle, iconSquare, iconRounded and
	// collapsedGroup.
	Shape   string
	IsGroup bool
	// Parent is the id of the group it is in, if HasParent; a subgraph's id may be "".
	Parent            string
	HasParent         bool
	CSSStyles         []string
	CSSCompiledStyles []string
	CSSClasses        string
	Dir               string
	Link              string
	LinkTarget        string
	Tooltip           string
	Icon              string
	Pos               string
	Img               string
	AssetWidth        *float64
	AssetHeight       *float64
	Constraint        string
	// ColorIndex is the place of a group in the order its subgraphs are declared, outer
	// before inner, and -1 for a node.
	ColorIndex int
	Metadata   *yaml.Map
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
	// Curve is the curve the edge is drawn with: its own, the flowchart's default, or the
	// default the drawing is given, and nil where there is none. It is a string but where the
	// edge's metadata gives its curve another value.
	Curve any
}

// Graph is what a flowchart draws, Pele's FlowGraph.
type Graph struct {
	Nodes []*GraphNode
	Edges []*GraphEdge
}

// shapeOf is the shape a node that is not a group is drawn with: an image's or an icon's, or
// the one its brackets give, as Pele's shapeOf gives it.
func shapeOf(v *Node) string {
	if v.Img != "" {
		return "imageSquare"
	}
	if v.Icon != "" {
		switch v.Form {
		case "circle":
			return "iconCircle"
		case "square":
			return "iconSquare"
		case "rounded":
			return "iconRounded"
		}
		return "icon"
	}
	switch v.Type {
	case "square", "":
		return "squareRect"
	case "round":
		return "roundedRect"
	}
	return v.Type
}

// arrows are the markers at the start and at the end of an edge of type typ, as Pele's arrows
// gives them.
func arrows(typ string) (string, string) {
	switch typ {
	case "arrow_point", "arrow_circle", "arrow_cross":
		return "none", typ
	case "double_arrow_point", "double_arrow_circle", "double_arrow_cross":
		single := strings.Replace(typ, "double_", "", 1)
		return single, single
	}
	return "none", "arrow_point"
}

// collapsed reports whether a subgraph's metadata collapses it, the test of Pele's
// isCollapsed and of the loops of buildFlowGraph that read metadata.view.
func collapsed(sg *Subgraph) bool {
	if sg == nil {
		return false
	}
	v, _ := sg.Metadata.Value("view").(string)
	return v == "collapsed"
}

// buildGraph resolves a parsed flowchart into the nodes and edges that are drawn, as Pele's
// buildFlowGraph does: group membership, collapsed subgraphs, class styles and edge defaults.
// defaultCurve is the flowchart config's curve, "" where it has none.
func buildGraph(model *DB, defaultCurve string) *Graph {
	g := &Graph{Nodes: []*GraphNode{}, Edges: []*GraphEdge{}}
	subgraphs := model.Subgraphs

	// compiled is the styles of the classes, trimmed, as Pele's compiled gives them.
	compiled := func(classNames []string) []string {
		out := []string{}
		for _, name := range classNames {
			def := model.classes[name]
			if def == nil {
				continue
			}
			out = trimAll(append(out, def.Styles...))
			out = trimAll(append(out, def.TextStyles...))
		}
		return out
	}

	subgraphParent := map[string]string{}
	for _, sg := range subgraphs {
		for _, child := range sg.Nodes {
			if model.Subgraph(child) != nil {
				subgraphParent[child] = sg.ID
			}
		}
	}

	declarationIndex := map[string]int{}
	childrenOf := map[string][]string{}
	for _, sg := range subgraphs {
		if parent, ok := subgraphParent[sg.ID]; ok {
			childrenOf[parent] = append(childrenOf[parent], sg.ID)
		}
	}
	next := 0
	for _, sg := range subgraphs {
		if _, ok := subgraphParent[sg.ID]; ok {
			continue
		}
		pending := []string{sg.ID}
		for len(pending) > 0 {
			id := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if _, ok := declarationIndex[id]; ok {
				continue
			}
			declarationIndex[id] = next
			next++
			kids := childrenOf[id]
			for i := len(kids) - 1; i >= 0; i-- {
				pending = append(pending, kids[i])
			}
		}
	}

	anyCollapsed := false
	for _, sg := range subgraphs {
		if collapsed(sg) {
			anyCollapsed = true
		}
	}
	// The outermost collapsed subgraph around each subgraph, resolved once per chain of parents.
	type result struct {
		id string
		ok bool
	}
	outermost := map[string]result{}
	outermostCollapsed := func(id string) result {
		var chain []string
		seen := map[string]bool{}
		current, has := id, true
		for has {
			if _, done := outermost[current]; done || seen[current] {
				break
			}
			seen[current] = true
			chain = append(chain, current)
			current, has = subgraphParent[current]
		}
		var r result
		if has {
			r = outermost[current]
		}
		for i := len(chain) - 1; i >= 0; i-- {
			if !r.ok && collapsed(model.Subgraph(chain[i])) {
				r = result{chain[i], true}
			}
			outermost[chain[i]] = r
		}
		return r
	}

	hidden := map[string]bool{}
	collapsedInto := map[string]string{}
	if anyCollapsed {
		for _, sg := range subgraphs {
			ancestor := outermostCollapsed(sg.ID)
			if !ancestor.ok {
				continue
			}
			if sg.ID != ancestor.id {
				hidden[sg.ID] = true
				collapsedInto[sg.ID] = ancestor.id
			}
			for _, child := range sg.Nodes {
				if child == ancestor.id {
					continue
				}
				hidden[child] = true
				collapsedInto[child] = ancestor.id
			}
		}
	}

	parentOf := map[string]string{}
	groups := map[string]bool{}
	for i := len(subgraphs) - 1; i >= 0; i-- {
		sg := subgraphs[i]
		if hidden[sg.ID] {
			continue
		}
		if len(sg.Nodes) > 0 {
			groups[sg.ID] = true
		}
		for _, id := range sg.Nodes {
			parentOf[id] = sg.ID
		}
	}

	byID := map[string]*GraphNode{}
	for i := len(subgraphs) - 1; i >= 0; i-- {
		sg := subgraphs[i]
		if hidden[sg.ID] {
			continue
		}
		isCollapsed := collapsed(sg)
		parent, hasParent := parentOf[sg.ID]
		colorIndex, ok := declarationIndex[sg.ID]
		if !ok {
			colorIndex = -1
		}
		node := &GraphNode{
			ID:                sg.ID,
			Label:             sg.Title,
			LabelType:         sg.LabelType,
			Parent:            parent,
			HasParent:         hasParent,
			CSSStyles:         []string{},
			CSSCompiledStyles: compiled(sg.Classes),
			CSSClasses:        strings.Join(sg.Classes, " "),
			Shape:             "rect",
			Dir:               sg.Dir,
			IsGroup:           !isCollapsed,
			ColorIndex:        colorIndex,
		}
		if isCollapsed {
			node.Shape = "collapsedGroup"
		} else {
			node.Metadata = sg.Metadata
		}
		g.Nodes = append(g.Nodes, node)
		byID[sg.ID] = node
	}

	for _, v := range model.Nodes {
		if hidden[v.ID] {
			continue
		}
		if existing := byID[v.ID]; existing != nil {
			existing.CSSStyles = v.Styles
			existing.CSSCompiledStyles = compiled(v.Classes)
			existing.CSSClasses = strings.Join(v.Classes, " ")
			continue
		}
		isGroup := groups[v.ID]
		parent, hasParent := parentOf[v.ID]
		tooltip, _ := model.Tooltip(v.ID)
		node := &GraphNode{
			ID:                v.ID,
			Label:             v.Text,
			LabelType:         v.LabelType,
			Parent:            parent,
			HasParent:         hasParent,
			CSSStyles:         v.Styles,
			CSSCompiledStyles: compiled(append([]string{"default", "node"}, v.Classes...)),
			CSSClasses:        "default " + strings.Join(v.Classes, " "),
			Dir:               v.Dir,
			Link:              v.Link,
			LinkTarget:        v.LinkTarget,
			Tooltip:           tooltip,
			Icon:              v.Icon,
			Pos:               v.Pos,
			Img:               v.Img,
			AssetWidth:        v.AssetWidth,
			AssetHeight:       v.AssetHeight,
			Constraint:        v.Constraint,
			IsGroup:           isGroup,
			Shape:             "rect",
			ColorIndex:        -1,
		}
		if !isGroup {
			node.Shape = shapeOf(v)
		}
		g.Nodes = append(g.Nodes, node)
		byID[v.ID] = node
	}

	for index, raw := range model.Edges {
		start, startHidden := collapsedInto[raw.Start]
		if !startHidden {
			start = raw.Start
		}
		end, endHidden := collapsedInto[raw.End]
		if !endHidden {
			end = raw.End
		}
		if start == end && (startHidden || endHidden) {
			continue
		}

		arrowStart, arrowEnd := arrows(raw.Type)
		styles := append([]string{}, model.DefaultEdgeStyle...)
		styles = append(styles, raw.Style...)
		plain := raw.Stroke == "invisible" || raw.Type == "arrow_open"
		id := raw.ID
		if id == "" {
			id = "L_" + start + "_" + end + "_" + strconv.Itoa(index)
		}
		typ := raw.Type
		if typ == "" {
			typ = "normal"
		}
		classes := "edge-thickness-normal edge-pattern-solid flowchart-link"
		if raw.Stroke == "invisible" {
			classes = ""
		}
		if plain {
			arrowStart, arrowEnd = "none", "none"
		}
		var curve any
		switch {
		case yaml.Truthy(raw.Interpolate):
			curve = raw.Interpolate
		case model.DefaultInterpolate != "":
			curve = model.DefaultInterpolate
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
			Classes:           classes,
			ArrowTypeStart:    arrowStart,
			ArrowTypeEnd:      arrowEnd,
			CSSCompiledStyles: compiled(raw.Classes),
			Style:             styles,
			Animate:           raw.Animate,
			Animation:         raw.Animation,
			Curve:             curve,
		})
	}
	return g
}

// trimAll trims each of the strings in place, the map((s) => s.trim()) of Pele's compiled.
func trimAll(s []string) []string {
	for i := range s {
		s[i] = chars.Trim(s[i])
	}
	return s
}
