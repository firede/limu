package agentflow

import (
	"math"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/flowchart"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/draw"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The spacing of an agent flow, as Pele's render.ts has it.
const (
	nodeSep  = 40
	edgeSep  = 16
	rankSep  = 48
	groupPad = 20
	loop     = 26
	dots     = 14
)

// collapsed is the shape a collapsed flow is drawn with: a box with a dashed rule and three
// dots under its label.
const collapsed = "collapsed"

// flowEnd is the side a flow runs toward, with sides numbered clockwise from the top, as
// Pele's FLOW_END has it.
var flowEnd = map[layout.Dir]int{layout.TB: 2, layout.RL: 3, layout.BT: 0, layout.LR: 1}

// Layout lays out a parsed agent flow as a scene, as Pele's renderAgentflow draws it: in its
// own direction, or, when one that runs across does not fit o.MaxWidth, down instead if
// that is narrower.
//
// The scene's roles are "agentflow.title"; "agentflow.flow" for a flow's box with its title;
// a node by its kind, "agentflow.tool", "agentflow.task", "agentflow.decision",
// "agentflow.input", "agentflow.refdoc", "agentflow.connector" or "agentflow.action";
// "agentflow.collapsed" for a collapsed flow, with "agentflow.collapsed-rule" and
// "agentflow.collapsed-dots" for the dashed rule and the dots drawn under its label; an edge
// by its kind, "agentflow.sequence", "agentflow.reference" (a dotted edge) or
// "agentflow.failure" (one that ends in a cross); and "agentflow.edge-label".
func Layout(db *DB, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	if !common.RunsAcross(db.Direction) {
		return lay(db, config, o, false)
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return lay(db, config, o, down) })
}

// numberOption is a number of an agent flow's config, if it is one above 0, at most 2000, as
// Pele's numberOption reads it.
func numberOption(config diagram.Config, key string, fallback float64) float64 {
	section, _ := config["agentflow"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v > 0 {
		return min(v, 2000)
	}
	return fallback
}

// A view is a node as it is drawn: Pele's NodeView.
type view struct {
	node   *GraphNode
	author style.Author
	label  text.Label
	shape  string
	w, h   float64
	dy     float64
	loops  int
	// filled is whether the author's style declares a fill, which replaces the wash of a tint.
	filled bool
}

// slot is the palette slot of a flow, collapsed or not: its own, or the one past the kinds.
// It is Pele's node.colorIndex ?? KIND_SLOT.size, in its draw.
func slot(n *GraphNode) int {
	if n.ColorIndex >= 0 {
		return n.ColorIndex
	}
	return len(kindSlot)
}

// tint is a palette color as an outline with a light wash of the same color inside, as Pele's
// tint draws it: the wash a fill opacity, which Pele writes only where the author declares no
// fill, so that a fill the author sets is opaque, and a fill opacity the author sets with no
// fill washes the palette color at it.
func tint(slot int, wash float64, filled bool) style.Style {
	s := style.Style{Fill: style.Series(slot), Stroke: style.Series(slot)}
	if !filled {
		s.FillOpacity = style.N(wash)
	}
	return s
}

// A held edge runs between a node and a flow around it, which the layout has no place for.
type held struct {
	edge     *GraphEdge
	label    text.Label
	src, dst int
}

// lay lays the agent flow out once, as Pele's draw does. down lays out one that runs across
// as one that runs down.
func lay(db *DB, config diagram.Config, o *layout.Options, down bool) *scene.Scene {
	padding := o.Pad()
	graph := db.Graph(DefaultCurve(config))
	size := o.Size()
	edgeSize := jsmath.Round(float64(size * 0.875))
	measure := o.Measure()
	wrapWidth := numberOption(config, "wrappingWidth", 120)
	minWidth := numberOption(config, "minNodeWidth", 120)
	labelOf := func(raw string, markdown bool, size, max float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: face, Markdown: markdown, Icons: o.Icons})
	}
	// The routing the options ask for, which Pele has no option for, overrides the curves.
	curve := common.RoutingCurve(o.Routing)

	index := make(map[string]int, len(graph.Nodes))
	for i, n := range graph.Nodes {
		index[n.ID] = i
	}
	views := make([]view, 0, len(graph.Nodes))
	cnodes := make([]layout.CNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		decls := node.CSSCompiledStyles
		if len(node.CSSStyles) > 0 {
			decls = append(append([]string(nil), node.CSSCompiledStyles...), node.CSSStyles...)
		}
		author := style.Declarations(decls)
		var face text.Face
		if author.Bold {
			face |= text.Bold
		}
		if author.Italic {
			face |= text.Italic
		}
		markdown := node.LabelType == LabelMarkdown
		fontSize := size
		if author.FontSize > 0 {
			fontSize = author.FontSize
		}
		var v view
		var c layout.CNode
		switch {
		case node.IsGroup:
			label := labelOf(node.Label, markdown, fontSize, 4000, face)
			v = view{node: node, author: author, label: label, shape: "group"}
			c = layout.NewCNode(0, 0, -1)
			c.IsGroup = true
			if node.Dir != "" {
				c.Dir = layout.Direction(node.Dir)
			}
			c.PadX = groupPad
			if label.Height > 0 {
				c.PadTop = label.Height + 16
			} else {
				c.PadTop = groupPad
			}
			c.PadBottom = groupPad
			c.MinW = label.Width + 2*groupPad - 12
		case node.Shape == "collapsedGroup":
			label := labelOf(node.Label, markdown, fontSize, wrapWidth, face)
			w := max(label.Width, minWidth) + 2*draw.PadX
			h := label.Height + 2*draw.PadY + dots
			v = view{node: node, author: author, label: label, shape: collapsed, w: w, h: h, dy: -dots / 2}
			c = layout.NewCNode(w, h, -1)
		default:
			shape := flowchart.CanonicalShape(node.Shape)
			label := labelOf(node.Label, markdown, fontSize, wrapWidth, face)
			// A diamond grows twice as fast as its label, so it keeps the label's own width.
			tw := max(label.Width, minWidth)
			if shape == "diam" {
				tw = label.Width
			}
			s := draw.ShapeSize(shape, tw, label.Height)
			v = view{node: node, author: author, label: label, shape: shape, w: s.W, h: s.H, dy: s.DY}
			c = layout.NewCNode(s.W, s.H, -1)
		}
		v.filled = style.FillDeclared(decls)
		c.Seq = float64(len(views))
		views = append(views, v)
		cnodes = append(cnodes, c)
	}
	for i, node := range graph.Nodes {
		if node.HasParent {
			if p, ok := index[node.Parent]; ok {
				cnodes[i].Parent = p
			}
		}
	}

	rootDir := layout.Direction(db.Direction)
	if down {
		rootDir = layout.TB
	}
	// The direction of the level a node sits in.
	flowOf := func(i int) layout.Dir {
		var dir layout.Dir
		for p, hops := cnodes[i].Parent, 0; p >= 0 && dir == "" && hops < 10000; p, hops = cnodes[p].Parent, hops+1 {
			dir = cnodes[p].Dir
		}
		if dir == "" {
			return rootDir
		}
		return dir
	}
	// flowsSideways reports whether the level a node sits in runs across, as Pele's
	// flowsSideways does.
	flowsSideways := func(i int) bool {
		dir := flowOf(i)
		return dir == layout.LR || dir == layout.RL
	}

	depth := make([]int, len(cnodes))
	for i := range depth {
		depth[i] = -1
	}
	for i := range cnodes {
		var chain []int
		j := i
		for j >= 0 && depth[j] < 0 && len(chain) <= len(cnodes) {
			chain = append(chain, j)
			j = cnodes[j].Parent
		}
		d := 0
		if j >= 0 {
			d = depth[j] + 1
		}
		for k := len(chain) - 1; k >= 0; k-- {
			depth[chain[k]] = d
			d++
		}
	}
	// contains reports whether node i is inside group, at any depth, as Pele's contains does.
	contains := func(group, i int) bool {
		j := i
		for d := depth[i]; d > depth[group] && j >= 0; d-- {
			j = cnodes[j].Parent
		}
		return j == group && i != group
	}

	var cedges []layout.CEdge
	var edgeLabels []text.Label
	var drawn []*GraphEdge
	// Edges between a node and a flow around it. The layout has no place for them, so they
	// are drawn afterwards, between the node and the flow's border.
	var holds []held
	for _, edge := range graph.Edges {
		src, ok1 := index[edge.Start]
		dst, ok2 := index[edge.End]
		if !ok1 || !ok2 {
			continue
		}
		label := labelOf(edge.Label, edge.LabelType == LabelMarkdown, edgeSize, wrapWidth, 0)
		if contains(src, dst) || contains(dst, src) {
			holds = append(holds, held{edge, label, src, dst})
			continue
		}
		if src == dst {
			// A self-loop sits beside the node, across the flow, and the node's box grows to
			// hold it.
			views[src].loops++
			if flowsSideways(src) {
				room := 0.0
				if label.Height > 0 {
					room = label.Height + 8
				}
				cnodes[src].H += 2 * (loop + room)
			} else {
				room := 0.0
				if label.Width > 0 {
					room = label.Width + 12
				}
				cnodes[src].W += 2 * (loop + room)
			}
		}
		drawn = append(drawn, edge)
		edgeLabels = append(edgeLabels, label)
		minlen := float64(edge.MinLen)
		if edge.MinLen == 0 {
			minlen = 1
		}
		ce := layout.CEdge{Src: src, Dst: dst, Minlen: minlen}
		if label.Width > 0 {
			ce.LabelW = label.Width + 12
		}
		if label.Height > 0 {
			ce.LabelH = label.Height + 4
		}
		cedges = append(cedges, ce)
	}

	opt := layout.LayeredOptions{
		NodeSep:  numberOption(config, "nodeSpacing", nodeSep),
		EdgeSep:  edgeSep,
		RankSep:  numberOption(config, "rankSpacing", rankSep),
		PortSep:  20,
		Tracks:   curve == "step",
		Straight: curve == "linear",
	}
	result := layout.Compound(cnodes, cedges, rootDir, opt)

	title := labelOf(db.Title, false, size, 4000, text.Bold)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	inner := max(result.Width, title.Width)
	ox := padding + (inner-result.Width)/2
	oy := padding + titleHeight
	layout.Shift(cnodes, cedges, ox, oy)
	sc := &scene.Scene{
		Type:     "agentflow",
		Width:    math.Ceil(inner + 2*padding),
		Height:   math.Ceil(result.Height + titleHeight + 2*padding),
		FontSize: size,
		Title:    db.AccTitle,
		Desc:     db.AccDescr,
	}
	node := func(i int) draw.Node {
		c := &cnodes[i]
		return draw.Node{X: c.X, Y: c.Y, W: c.W, H: c.H, Group: c.IsGroup}
	}

	// An edge between a node and a flow around it runs straight between the node and the
	// flow's border: out along the flow, or in against it. If another node is in the way it
	// takes a side instead, and failing that the far end.
	heldPaths := make([]draw.EdgePath, 0, len(holds))
	heldLabels := make([]float64, 0, 2*len(holds))
	heldRoutes := make([][]float64, 0, len(holds))
	heldDrawn := make([][]float64, 0, len(holds))
	careful := len(holds)*len(cnodes) < 2e6
	for _, h := range holds {
		out := contains(h.dst, h.src)
		outer, i := h.src, h.dst
		if out {
			outer, i = h.dst, h.src
		}
		g := &cnodes[outer]
		n := &cnodes[i]
		dir := g.Dir
		if dir == "" {
			dir = flowOf(outer)
		}
		end := flowEnd[dir]
		first := (end + 2) & 3
		if out {
			first = end
		}
		// toward is the straight route between the node and the side of its flow, as Pele's
		// toward gives it.
		toward := func(side int) []float64 {
			upright := side&1 == 0
			sign := 1.0
			if side == 0 || side == 3 {
				sign = -1
			}
			var nx, ny, bx, by, axis float64
			if upright {
				nx, ny = n.X, n.Y+(sign*n.H)/2
				bx, by = n.X, g.Y+(sign*g.H)/2
			} else {
				nx, ny = n.X+(sign*n.W)/2, n.Y
				bx, by = g.X+(sign*g.W)/2, n.Y
				axis = 1
			}
			if out {
				return []float64{nx, ny, axis, bx, by, axis}
			}
			return []float64{bx, by, axis, nx, ny, axis}
		}
		// blocked reports whether another node that is not a flow lies across route r, as
		// Pele's blocked does.
		blocked := func(r []float64) bool {
			for k := range cnodes {
				c := &cnodes[k]
				if k != i && !c.IsGroup &&
					c.X+c.W/2 > min(r[0], r[3])-1 &&
					c.X-c.W/2 < max(r[0], r[3])+1 &&
					c.Y+c.H/2 > min(r[1], r[4])-1 &&
					c.Y-c.H/2 < max(r[1], r[4])+1 {
					return true
				}
			}
			return false
		}
		route := toward(first)
		if careful && blocked(route) {
			for _, side := range [3]int{(first + 1) & 3, (first + 3) & 3, (first + 2) & 3} {
				if other := toward(side); !blocked(other) {
					route = other
					break
				}
			}
		}
		heldRoutes = append(heldRoutes, route)
		heldLabels = append(heldLabels, (route[0]+route[3])/2, (route[1]+route[4])/2)
		drawnRoute := append([]float64(nil), route...)
		at := 3
		if out {
			at = 0
		}
		draw.InsetRoute(drawnRoute, at, views[i].shape, views[i].w, views[i].h, node(i))
		trim := scene.MarkerType(h.edge.ArrowTypeEnd).Trim()
		heldPaths = append(heldPaths, draw.RoutePath(drawnRoute, "linear", 0, trim, 0, draw.OwnReach(trim)))
		heldDrawn = append(heldDrawn, drawnRoute)
	}

	crossings := draw.Crossings{}
	for _, c := range cnodes {
		if c.IsGroup {
			for _, e := range cedges {
				draw.MarkCrossings(crossings, e.Route)
			}
			for _, route := range heldRoutes {
				draw.MarkCrossings(crossings, route)
			}
			break
		}
	}

	var titles, clusters, nodes, edges, labels []scene.Item
	if !title.Empty() {
		titles = append(titles, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At: scene.Point{X: sc.Width / 2, Y: padding + title.Height/2},
		})
	}
	for i := range views {
		v := &views[i]
		c := &cnodes[i]
		n := v.node
		x, y := c.X, c.Y
		classes, tags := common.Clickable(common.ClassNames(common.TrimDefault(n.CSSClasses)))
		if c.IsGroup {
			titleX := draw.ClusterTitleX(node(i), v.label.Width, crossings, groupPad)
			box := scene.Rect{X: x - c.W/2, Y: y - c.H/2, W: c.W, H: c.H}
			own := tint(slot(n), 0.06, v.filled)
			own.Text = style.Muted
			cluster := scene.Item{
				Kind: scene.Group, Role: RoleFlow, ID: n.ID, Class: classes, Tags: tags, Box: box,
				Parts:  []scene.Part{{Box: box, Rounded: true}},
				Style:  own,
				Author: v.author.Style,
			}
			if !v.label.Empty() {
				cluster.Label = &v.label
				cluster.At = scene.Point{X: titleX, Y: y - c.H/2 + 8 + v.label.Height/2}
			}
			clusters = append(clusters, cluster)
			continue
		}

		w, h := v.w, v.h
		box := scene.Rect{X: x - w/2, Y: y - h/2, W: w, H: h}
		it := scene.Item{
			Kind: scene.Node, Shape: scene.Shape(v.shape), ID: n.ID, Class: classes, Tags: tags, Box: box,
			Author: v.author.Style,
			Label:  &v.label, At: scene.Point{X: x, Y: y + v.dy},
			Tooltip: n.Tooltip,
		}
		var marks []scene.Item
		if v.shape == collapsed {
			s := slot(n)
			it.Role = RoleCollapsed
			it.Parts = []scene.Part{{Box: box, Rounded: true}}
			it.Style = tint(s, 0.06, v.filled)
			// What the author set for lines styles the rule, and fills nothing.
			line := v.author.Style
			line.Fill = style.Paint{}
			rule := h/2 - dots - 2
			mid := rule + (dots+2)/2
			marks = []scene.Item{{
				Kind: scene.Mark, Role: RoleCollapsedRule, ID: n.ID,
				Box: scene.Rect{X: x - w/2 + 10, Y: y + rule, W: w - 20},
				Parts: []scene.Part{{
					Path:  new(scene.Path).MoveTo(x+(-w/2+10), y+rule).LineTo(x+(w/2-10), y+rule),
					Paint: scene.StrokeOnly,
				}},
				Style:  style.Style{Stroke: style.Series(s), Fill: style.None, Dash: style.Dashes(3, 3)},
				Author: line,
			}, {
				Kind: scene.Mark, Role: RoleCollapsedDots, ID: n.ID,
				Box:   scene.Rect{X: x - 9 - 1.75, Y: y + mid - 1.75, W: 18 + 3.5, H: 3.5},
				Parts: []scene.Part{{Path: dot(dot(dot(new(scene.Path), x, y, -9, mid), x, y, 0, mid), x, y, 9, mid)}},
				Style: style.Style{Fill: style.Series(s), Stroke: style.None},
			}}
		} else {
			kind := n.Kind
			if kind == "" {
				kind = "task"
			}
			s, ok := kindSlot[kind]
			if !ok {
				s = 1
			}
			it.Role = scene.Role("agentflow." + kind)
			it.Parts = draw.Outline(v.shape, x, y, w, h)
			it.Style = tint(s, 0.14, v.filled)
		}
		if hasClass(classes, "internal-link") && n.Label != "" && !o.NoLinks {
			// The label is a note's name, but it ends up a link, so it is checked as an address.
			name := text.DecodeEntities(n.Label)
			if _, ok := url.LinkURL(url.SanitizeURL(name), true, o.LinkSchemes); ok {
				it.Link = name
			}
		}
		if n.Link != "" {
			if href, ok := url.LinkURL(n.Link, !o.NoLinks, o.LinkSchemes); ok {
				it.Link = href
			}
		}
		nodes = append(nodes, it)
		nodes = append(nodes, marks...)
	}

	// put adds an edge and its label, as Pele's put draws them; route, which Pele does not
	// have, is the route the edge was laid out along.
	put := func(edge *GraphEdge, label *text.Label, path draw.EdgePath, route []float64, x, y float64) {
		var author style.Author
		if len(edge.Style) > 0 || len(edge.CSSCompiledStyles) > 0 {
			author = style.Declarations(append(append([]string(nil), edge.CSSCompiledStyles...), edge.Style...))
			// What styles an edge's line styles its marker, and fills nothing.
			author.Style.Fill = style.Paint{}
		}
		kind := "sequence"
		// Pele draws the edges in a group with round caps; what the author sets on the
		// edge's own group is laid over them.
		own := style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound}
		switch {
		case edge.Thickness == "dotted":
			kind = "reference"
			own.Dash = style.Dashes(3, 4)
		case edge.ArrowTypeEnd == "arrow_cross":
			kind = "failure"
		}
		it := scene.Item{
			Kind: scene.Edge, Role: scene.Role("agentflow." + kind), ID: edge.ID,
			Class:  common.ClassNames(edgeClassNoise.Replace(edge.Classes)),
			Box:    path.Path.Bounds(),
			Parts:  []scene.Part{{Path: path.Path, Paint: scene.StrokeOnly}},
			Ends:   path.Ends("", scene.MarkerType(edge.ArrowTypeEnd)),
			Style:  own,
			Author: author.Style,
		}
		for k := 0; k+1 < len(route); k += 3 {
			it.Route = append(it.Route, scene.Point{X: route[k], Y: route[k+1]})
		}
		if len(it.Route) == 0 {
			// A self-loop, which the layout gives no route: the corners its curve is drawn about.
			it.Route = it.Parts[0].Path.Corners()
		}
		edges = append(edges, it)
		if label.Width > 0 {
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: RoleEdgeLabel, ID: edge.ID,
				Box:    scene.Rect{X: x - (label.Width+8)/2, Y: y - label.Height/2, W: label.Width + 8, H: label.Height},
				Parts:  []scene.Part{draw.LabelPatch(label, x, y)},
				Label:  label,
				At:     scene.Point{X: x, Y: y},
				Style:  style.Style{Fill: style.Bg, Stroke: style.None},
				Author: style.Style{Text: author.Style.Text, Opacity: author.Style.Opacity},
			})
		}
	}

	loopSeen := make([]int, len(views))
	for i := range cedges {
		ce := &cedges[i]
		edge := drawn[i]
		trim := scene.MarkerType(edge.ArrowTypeEnd).Trim()
		var path draw.EdgePath
		var route []float64
		if ce.Src == ce.Dst {
			v := &views[ce.Src]
			k := loopSeen[ce.Src]
			loopSeen[ce.Src]++
			c := &cnodes[ce.Src]
			var at scene.Point
			path, at = draw.LoopPath(c.X, c.Y, v.w, v.h, k, flowsSideways(ce.Src), loop, trim, ce.LabelW, ce.LabelH)
			ce.LabelX, ce.LabelY = at.X, at.Y
		} else {
			route = append([]float64(nil), ce.Route...)
			sv, dv := &views[ce.Src], &views[ce.Dst]
			draw.InsetRoute(route, 0, sv.shape, sv.w, sv.h, node(ce.Src))
			draw.InsetRoute(route, len(route)-3, dv.shape, dv.w, dv.h, node(ce.Dst))
			edgeCurve, _ := edge.Curve.(string)
			if o.Routing != layout.Auto {
				edgeCurve = curve
			}
			path = draw.RoutePath(route, edgeCurve, 0, trim, 0, draw.OwnReach(trim))
		}
		put(edge, &edgeLabels[i], path, route, ce.LabelX, ce.LabelY)
	}
	for i := range holds {
		put(holds[i].edge, &holds[i].label, heldPaths[i], heldDrawn[i], heldLabels[2*i], heldLabels[2*i+1])
	}

	sc.Items = make([]scene.Item, 0, len(titles)+len(clusters)+len(edges)+len(labels)+len(nodes))
	for _, layer := range [][]scene.Item{titles, clusters, edges, labels, nodes} {
		sc.Items = append(sc.Items, layer...)
	}
	return sc
}

// dot adds a dot 3.5 across to a path, at dx, dy from a collapsed flow's center x, y, as
// Pele's dot draws it: two half circles from its left to its right and back.
func dot(p *scene.Path, x, y, dx, dy float64) *scene.Path {
	return p.MoveTo(x+(dx-1.75), y+dy).
		ArcTo(1.75, 1.75, 0, true, false, x+(dx-1.75)+3.5, y+dy).
		ArcTo(1.75, 1.75, 0, true, false, x+(dx-1.75), y+dy).
		Close()
}

// The classes Mermaid gives every edge, which say nothing.
var edgeClassNoise = strings.NewReplacer("edge-thickness-normal", "", "edge-pattern-solid", "", "flowchart-link", "")

// hasClass reports whether name is one of classes, the test Pele's draw makes for
// internal-link with / internal-link(?: |$)/ on the joined class names.
func hasClass(classes []string, name string) bool {
	for _, c := range classes {
		if c == name {
			return true
		}
	}
	return false
}
