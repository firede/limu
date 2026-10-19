package flowchart

import (
	"math"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/draw"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The spacing of a flowchart, as Pele's render.ts has it.
const (
	nodeSep  = 40
	edgeSep  = 16
	rankSep  = 48
	groupPad = 20
	loop     = 26
	padding  = 8
)

// A Variant is a diagram type that is a flowchart laid out its own way, which may draw its
// top-level groups itself, as Pele's FlowVariant is.
type Variant struct {
	// Type is the diagram's type: the scene's, and the section of the configuration read in
	// place of the flowchart's when it is one.
	Type string
	// Prepare changes the graph before it is laid out.
	Prepare func(g *Graph)
	// Layout lays the graph out in place of layout.Compound; tight is set when the diagram is
	// laid out tightly to fit the host's width.
	Layout func(nodes []layout.CNode, edges []layout.CEdge, dir layout.Dir, opt layout.LayeredOptions, tight bool) layout.CompoundResult
	// DrawGroup, if set, draws a top-level group in place of a cluster: c is the group as it
	// was laid out, label its title, author what its author set for it, and dir the
	// direction the diagram was laid out in.
	DrawGroup func(c *layout.CNode, label *text.Label, author style.Author, classes []string, id string, dir layout.Dir) scene.Item
}

// Layout lays out a parsed flowchart as a scene, as Pele's renderFlowchart draws it: across
// and loosely, and, when that does not fit o.MaxWidth, tightly, or down instead of across,
// whichever is narrower.
func Layout(db *DB, config diagram.Config, o *layout.Options) *scene.Scene {
	return LayoutVariant(db, config, o, nil)
}

// LayoutVariant lays out a parsed flowchart as a scene, as Pele's renderFlowchart draws it
// with a variant, or as Layout does when v is nil.
func LayoutVariant(db *DB, config diagram.Config, o *layout.Options, v *Variant) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	fitted := func(down bool) *scene.Scene {
		return common.Tighten(o, func(tight bool) *scene.Scene { return lay(db, config, o, v, down, tight) })
	}
	if common.RunsAcross(db.Direction) {
		return common.TurnToFit(o, fitted)
	}
	return fitted(false)
}

// numberOption is a number of a flowchart's config, if it is one above 0, at most 2000.
func numberOption(section diagram.Config, key string, fallback float64) float64 {
	if v, ok := section[key].(float64); ok && v > 0 {
		return min(v, 2000)
	}
	return fallback
}

// assetSize is the size an image or icon is given, if it is a sensible one.
func assetSize(v *float64, fallback float64) float64 {
	if v != nil && *v > 0 && *v <= 4000 {
		return *v
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
}

// lay lays the flowchart out once, as Pele's draw does, with the variant v if it is not nil.
// down lays out a flowchart that runs across as one that runs down; tight sets nodes closer
// together.
func lay(db *DB, config diagram.Config, o *layout.Options, variant *Variant, down, tight bool) *scene.Scene {
	gpad := float64(groupPad)
	if tight {
		gpad = 12
	}
	flow, _ := config["flowchart"].(diagram.Config)
	if variant != nil {
		if own, ok := config[variant.Type].(diagram.Config); ok {
			flow = own
		}
	}
	curve, _ := flow["curve"].(string)
	switch o.Routing {
	case layout.Orthogonal:
		curve = "step"
	case layout.Straight:
		curve = "linear"
	}
	graph := BuildGraph(db, curve)
	if variant != nil {
		variant.Prepare(graph)
	}
	size := o.Size()
	edgeSize := jsmath.Round(float64(size * 0.875))
	measure := o.Measure()
	wrapWidth := numberOption(flow, "wrappingWidth", 200)
	groupMax := 4000.0
	if variant != nil && tight {
		wrapWidth = min(wrapWidth, 100)
		groupMax = max(80, common.TitleRoom(o)/2)
	}
	labelOf := func(raw string, markdown bool, size, max float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: face, Markdown: markdown, Icons: o.Icons})
	}

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
		if node.IsGroup {
			label := labelOf(node.Label, markdown, fontSize, groupMax, face)
			v = view{node: node, author: author, label: label, shape: "group"}
			c = layout.NewCNode(0, 0, -1)
			c.IsGroup = true
			if node.Dir != "" {
				c.Dir = layout.Direction(node.Dir)
			}
			c.PadX = gpad
			if label.Height > 0 {
				c.PadTop = label.Height + 16
			} else {
				c.PadTop = groupPad
			}
			c.PadBottom = gpad
			c.MinW = label.Width + 2*gpad - 12
		} else {
			shape := CanonicalShape(node.Shape)
			var raw string
			if node.Img != "" || node.Icon != "" || draw.ShapeHasLabel(shape) {
				raw = node.Label
			}
			label := labelOf(raw, markdown, fontSize, wrapWidth, face)
			var w, h, dy float64
			if node.Img != "" || node.Icon != "" {
				box := 48.0
				if node.Img != "" {
					box = 80
				}
				aw, ah := assetSize(node.AssetWidth, box), assetSize(node.AssetHeight, box)
				w = max(aw, label.Width+8)
				h = ah
				if label.Height > 0 {
					h += label.Height + 6
				}
				if node.Pos == "t" {
					dy = -ah / 2
				} else {
					dy = ah / 2
				}
				if label.Height == 0 {
					dy = 0
				}
			} else {
				s := draw.ShapeSize(shape, label.Width, label.Height)
				w, h, dy = s.W, s.H, s.DY
			}
			v = view{node: node, author: author, label: label, shape: shape, w: w, h: h, dy: dy}
			c = layout.NewCNode(w, h, -1)
		}
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
	flowsSideways := func(i int) bool {
		var dir layout.Dir
		for p, hops := cnodes[i].Parent, 0; p >= 0 && dir == "" && hops < 10000; p, hops = cnodes[p].Parent, hops+1 {
			dir = cnodes[p].Dir
		}
		if dir == "" {
			dir = rootDir
		}
		return dir == layout.LR || dir == layout.RL
	}
	// The edges on one side of a node spread along it; a picture or an icon keeps them at
	// its middle.
	for i, v := range views {
		if v.node.IsGroup {
			cnodes[i].Span = math.Inf(1)
		} else if v.node.Img == "" && v.node.Icon == "" {
			cnodes[i].Span = draw.ShapeSpan(v.shape, v.w, v.h, flowsSideways(i))
		}
	}

	var cedges []layout.CEdge
	var edgeLabels []text.Label
	var drawn []*GraphEdge
	loops := make([]int, len(views))
	for _, edge := range graph.Edges {
		src, ok1 := index[edge.Start]
		dst, ok2 := index[edge.End]
		if !ok1 || !ok2 {
			continue
		}
		label := labelOf(edge.Label, edge.LabelType == LabelMarkdown, edgeSize, wrapWidth, 0)
		if src == dst {
			// A self-loop sits beside the node, across the flow, and the node's box grows to
			// hold it.
			loops[src]++
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
		NodeSep:  numberOption(flow, "nodeSpacing", nodeSep),
		EdgeSep:  edgeSep,
		RankSep:  numberOption(flow, "rankSpacing", rankSep),
		PortSep:  20,
		Tracks:   strings.HasPrefix(curve, "step"),
		Straight: curve == "linear",
		HeadRoom: scene.MarkerType("arrow_point").Trim(),
	}
	if tight {
		opt.NodeSep *= 0.5
		opt.RankSep *= 0.75
	}
	var result layout.CompoundResult
	if variant != nil {
		result = variant.Layout(cnodes, cedges, rootDir, opt, tight)
	} else {
		result = layout.Compound(cnodes, cedges, rootDir, opt)
	}

	titleMax := 4000.0
	if tight {
		titleMax = common.TitleRoom(o)
	}
	title := labelOf(db.Title, false, size, titleMax, text.Bold)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	inner := max(result.Width, title.Width)
	ox := padding + (inner-result.Width)/2
	oy := padding + titleHeight
	layout.Shift(cnodes, cedges, ox, oy)
	sc := &scene.Scene{
		Type:     "flowchart",
		Width:    math.Ceil(inner + 2*padding),
		Height:   math.Ceil(result.Height + titleHeight + 2*padding),
		FontSize: size,
		Title:    db.AccTitle,
		Desc:     db.AccDescr,
	}
	if variant != nil {
		sc.Type = variant.Type
	}
	node := func(i int) draw.Node {
		c := &cnodes[i]
		return draw.Node{X: c.X, Y: c.Y, W: c.W, H: c.H, Group: c.IsGroup}
	}

	crossings := draw.Crossings{}
	for _, c := range cnodes {
		if c.IsGroup {
			for _, e := range cedges {
				draw.MarkCrossings(crossings, e.Route)
			}
			break
		}
	}

	var titles, clusters, struck, nodes, edges, labels []scene.Item
	if !title.Empty() {
		titles = append(titles, scene.Item{
			Kind: scene.Text, Role: "flowchart.title", Label: &title,
			At: scene.Point{X: sc.Width / 2, Y: padding + title.Height/2},
		})
	}
	for i := range views {
		v := &views[i]
		c := &cnodes[i]
		n := v.node
		x, y := c.X, c.Y
		classes := common.ClassNames(common.TrimDefault(n.CSSClasses))
		if c.IsGroup && variant != nil && variant.DrawGroup != nil && c.Parent < 0 {
			clusters = append(clusters, variant.DrawGroup(c, &v.label, v.author, classes, n.ID, rootDir))
			continue
		}
		if c.IsGroup {
			titleX := draw.ClusterTitleX(node(i), v.label.Width, crossings, gpad)
			titleY := y - c.H/2 + 8 + v.label.Height/2
			box := scene.Rect{X: x - c.W/2, Y: y - c.H/2, W: c.W, H: c.H}
			cluster := scene.Item{
				Kind: scene.Group, Role: "flowchart.cluster", ID: n.ID, Class: classes, Box: box,
				Parts:  []scene.Part{{Box: box, Rounded: true}},
				Style:  style.Style{Fill: style.SurfaceAlt.Alpha(0.5), Stroke: style.Border, Text: style.Muted},
				Author: v.author.Style,
			}
			label := &v.label
			if !label.Empty() {
				at := scene.Point{X: titleX, Y: titleY}
				if draw.TitleCrossed(node(i), label.Width, titleX, crossings) {
					depth := draw.StruckDepth(func(k int) int { return cnodes[k].Parent }, i)
					struck = append(struck, scene.Item{
						Kind: scene.Text, Role: "flowchart.cluster-title", ID: n.ID, Label: label, At: at,
						Style: style.Style{
							Text:      style.Muted,
							Halo:      style.SurfaceAlt.Alpha(draw.StruckTint(depth) / 100).Over(style.Bg),
							HaloWidth: style.N(draw.StruckHalo),
						},
						Author: style.Style{Text: v.author.Style.Text},
					})
				} else {
					cluster.Label, cluster.At = label, at
				}
			}
			clusters = append(clusters, cluster)
			continue
		}

		w, h := v.w, v.h
		it := scene.Item{
			Kind: scene.Node, Role: "flowchart.node", Shape: scene.Shape(v.shape), ID: n.ID, Class: classes,
			Box:    scene.Rect{X: x - w/2, Y: y - h/2, W: w, H: h},
			Style:  style.Style{Fill: style.Surface, Stroke: style.Border},
			Author: v.author.Style,
			Label:  &v.label, At: scene.Point{X: x, Y: y + v.dy},
			Tooltip: n.Tooltip,
		}
		switch {
		case n.Img != "":
			aw, ah := assetSize(n.AssetWidth, 80), assetSize(n.AssetHeight, 80)
			top := -ah / 2
			if v.label.Height > 0 {
				if n.Pos == "t" {
					top = h/2 - ah
				} else {
					top = -h / 2
				}
			}
			if src, ok := url.ImageURL(url.SafeURL(n.Img), !o.NoImages, o.ImageSchemes); ok {
				it.Picture = &scene.Picture{Image: src, Box: scene.Rect{X: x - aw/2, Y: y + top, W: aw, H: ah}, Fit: n.Constraint == "on"}
			}
		case n.Icon != "":
			box := assetSize(n.AssetHeight, 48)
			top := -box / 2
			if v.label.Height > 0 {
				if n.Pos == "t" {
					top = h/2 - box
				} else {
					top = -h / 2
				}
			}
			switch n.Shape {
			case "icon":
			case "iconCircle":
				it.Parts = []scene.Part{{Path: new(scene.Path).Ellipse(x, y+top+box/2, box/2, box/2)}}
			case "iconRounded":
				it.Parts = []scene.Part{{Path: new(scene.Path).RoundRect(scene.Rect{X: x - box/2, Y: y + top, W: box, H: box}, 8)}}
			default:
				r := scene.Rect{X: x - box/2, Y: y + top, W: box, H: box}
				it.Parts = []scene.Part{{Box: r}}
			}
			glyph := box * 0.6
			it.Picture = &scene.Picture{Icon: n.Icon, Box: scene.Rect{X: x - glyph/2, Y: y + top + (box-glyph)/2, W: glyph, H: glyph}}
		default:
			it.Parts = draw.Outline(v.shape, x, y, w, h)
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
	}

	var reach draw.SideReach
	for i, ce := range cedges {
		if ce.Src == ce.Dst || drawn[i].Thickness == "invisible" || len(ce.Route) < 6 {
			continue
		}
		reach.Add(ce.Src, draw.EndSide(ce.Route, 0, node(ce.Src)), scene.MarkerType(drawn[i].ArrowTypeStart))
		reach.Add(ce.Dst, draw.EndSide(ce.Route, len(ce.Route)-3, node(ce.Dst)), scene.MarkerType(drawn[i].ArrowTypeEnd))
	}

	loopSeen := make([]int, len(views))
	for i := range cedges {
		ce := &cedges[i]
		edge := drawn[i]
		label := &edgeLabels[i]
		var author style.Author
		if len(edge.Style) > 0 || len(edge.CSSCompiledStyles) > 0 {
			author = style.Declarations(append(append([]string(nil), edge.CSSCompiledStyles...), edge.Style...))
			// What styles an edge's line styles its markers, and fills nothing.
			author.Style.Fill = style.Paint{}
		}
		if edge.Thickness != "invisible" {
			start, end := scene.MarkerType(edge.ArrowTypeStart), scene.MarkerType(edge.ArrowTypeEnd)
			var path draw.EdgePath
			var route []float64
			if ce.Src == ce.Dst {
				v := &views[ce.Src]
				k := loopSeen[ce.Src]
				loopSeen[ce.Src]++
				var at scene.Point
				c := &cnodes[ce.Src]
				path, at = draw.LoopPath(c.X, c.Y, v.w, v.h, k, flowsSideways(ce.Src), loop, end.Trim(), ce.LabelW, ce.LabelH)
				ce.LabelX, ce.LabelY = at.X, at.Y
			} else {
				route = append([]float64(nil), ce.Route...)
				leaves := reach.Of(ce.Src, draw.EndSide(route, 0, node(ce.Src)))
				arrives := reach.Of(ce.Dst, draw.EndSide(route, len(route)-3, node(ce.Dst)))
				sv, dv := &views[ce.Src], &views[ce.Dst]
				draw.InsetRoute(route, 0, sv.shape, sv.w, sv.h, node(ce.Src))
				draw.InsetRoute(route, len(route)-3, dv.shape, dv.w, dv.h, node(ce.Dst))
				edgeCurve, _ := edge.Curve.(string)
				if o.Routing != layout.Auto {
					edgeCurve = curve
				}
				path = draw.RoutePath(route, edgeCurve, start.Trim(), end.Trim(), leaves, arrives)
			}
			// Pele draws the edges in a group with round caps; what the author sets on the
			// edge's own group is laid over them.
			own := style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound}
			switch edge.Thickness {
			case "thick":
				own.StrokeWidth = style.N(2.5)
			case "dotted":
				own.Dash = style.Dashes(3, 4)
			}
			it := scene.Item{
				Kind: scene.Edge, Role: "flowchart.edge", ID: edge.ID,
				Class:  common.ClassNames(edgeClassNoise.Replace(edge.Classes)),
				Box:    path.Path.Bounds(),
				Parts:  []scene.Part{{Path: path.Path, Paint: scene.StrokeOnly}},
				Ends:   path.Ends(start, end),
				Style:  own,
				Author: author.Style,
			}
			for k := 0; k+1 < len(route); k += 3 {
				it.Route = append(it.Route, scene.Point{X: route[k], Y: route[k+1]})
			}
			edges = append(edges, it)
		}
		if label.Width > 0 {
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: "flowchart.edge-label", ID: edge.ID,
				Box:    scene.Rect{X: ce.LabelX - (label.Width+8)/2, Y: ce.LabelY - label.Height/2, W: label.Width + 8, H: label.Height},
				Parts:  []scene.Part{draw.LabelPatch(label, ce.LabelX, ce.LabelY)},
				Label:  label,
				At:     scene.Point{X: ce.LabelX, Y: ce.LabelY},
				Style:  style.Style{Fill: style.Bg, Stroke: style.None},
				Author: style.Style{Text: author.Style.Text, Opacity: author.Style.Opacity},
			})
		}
	}

	sc.Items = make([]scene.Item, 0, len(titles)+len(clusters)+len(edges)+len(struck)+len(labels)+len(nodes))
	for _, layer := range [][]scene.Item{titles, clusters, edges, struck, labels, nodes} {
		sc.Items = append(sc.Items, layer...)
	}
	return sc
}

// The classes Mermaid gives every edge, which say nothing.
var edgeClassNoise = strings.NewReplacer("edge-thickness-normal", "", "edge-pattern-solid", "", "flowchart-link", "")

func hasClass(classes []string, name string) bool {
	for _, c := range classes {
		if c == name {
			return true
		}
	}
	return false
}
