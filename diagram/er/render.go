package er

import (
	"math"
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/diagram/internal/draw"
	"github.com/firede/limu/diagram/internal/graphs"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of an ER diagram, as Pele's render.ts has them.
const (
	nodeSep     float64 = 48
	edgeSep     float64 = 20
	rankSep     float64 = 64
	groupPad    float64 = 20
	loopLength  float64 = 28 // how far a self-relationship loops out beyond its markers
	loopBand    float64 = 38 // the least stretch of a side a self-relationship takes
	minWidth    float64 = 96
	boxPadX     float64 = 18
	boxPadY     float64 = 12
	headPadY    float64 = 9
	cellPadX    float64 = 12
	cellGap     float64 = 16
	rowPad      float64 = 9
	commentWrap float64 = 260
	noWrap      float64 = 1e6
	portGap     float64 = 18
	portMargin  float64 = 14
	// markerLength is how far a marker reaches back along its line from the box it touches,
	// Pele's MARKER_LENGTH.
	markerLength float64 = 18
)

// The roles of the parts of an ER diagram.
const (
	RoleTitle             scene.Role = "er.title"
	RoleCluster           scene.Role = "er.cluster"
	RoleRelationship      scene.Role = "er.relationship"
	RoleRelationshipLabel scene.Role = "er.relationship-label"
	RoleEntity            scene.Role = "er.entity"
	RoleDivider           scene.Role = "er.divider"
	RoleAttributeType     scene.Role = "er.attribute-type"
	RoleAttributeName     scene.Role = "er.attribute-name"
	RoleAttributeKeys     scene.Role = "er.attribute-keys"
	RoleAttributeComment  scene.Role = "er.attribute-comment"
)

// cellRoles are the roles of the four cells of an attribute's row, in their order.
var cellRoles = [4]scene.Role{RoleAttributeType, RoleAttributeName, RoleAttributeKeys, RoleAttributeComment}

// Layout lays out a parsed ER diagram as a scene, as Pele's renderEr draws it: in its own
// direction, and, for one that runs across and does not fit o.MaxWidth, down instead when
// that is narrower.
func Layout(db *DB, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	if !common.RunsAcross(db.Direction) {
		return lay(db, config, o, false)
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return lay(db, config, o, down) })
}

// numberOption is a number of the diagram's er config, if it is one above 0, at most 2000,
// as Pele's numberOption reads it.
func numberOption(config diagram.Config, key string, fallback float64) float64 {
	section, _ := config["er"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v > 0 {
		return min(v, 2000)
	}
	return fallback
}

// A row is a row of an entity's table: its type, name, keys and comment, and its height.
type row struct {
	cells [4]text.Label
	h     float64
}

// A view is an entity or a subgraph as it is drawn: Pele's NodeView.
type view struct {
	node   *GraphNode
	author css.Author
	label  text.Label
	w, h   float64
	headH  float64
	rows   []row
	// cols are the left edge of each column, from the left edge of the box.
	cols             []float64
	rowSize, keySize float64
	// below says that self-relationships loop out of the bottom side rather than the right.
	below bool
}

// A selfLoop is a relationship from an entity to itself, with the stretch of the side it
// loops out of: Pele's Loop.
type selfLoop struct {
	label *text.Label
	band  float64
	at    float64
}

// cellText is an attribute's text as it is laid out, as Pele's cellText makes it: Mermaid
// reads `~` pairs as angle brackets, which are written as entities so that the label is not
// read as holding a tag.
func cellText(raw string) string {
	converted := common.ParseGenericTypes(raw)
	if converted == raw {
		return raw
	}
	return strings.ReplaceAll(strings.ReplaceAll(converted, "<", "&lt;"), ">", "&gt;")
}

// lay lays the diagram out once, as Pele's draw does. turned lays out a diagram that runs
// across as one that runs down.
func lay(db *DB, config diagram.Config, o *layout.Options, turned bool) *scene.Scene {
	padding := o.Pad()
	graph := BuildGraph(db)
	size := o.Size()
	smallSize := jsmath.Round(float64(size * 0.875))
	measure := o.Measure()
	// labelOf lays out a label with the diagram's measurer and icons, as each call of Pele's
	// layoutLabel in draw does.
	labelOf := func(raw string, markdown bool, size, max float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: face, Markdown: markdown, Icons: o.Icons})
	}

	index := make(map[string]int, len(graph.Nodes))
	for i, n := range graph.Nodes {
		index[n.ID] = i
	}

	var cedges []graphs.CEdge
	var edgeLabels []*text.Label
	var drawn []*GraphEdge
	loops := map[int][]*selfLoop{}
	var loopOrder []int
	var loopOf []*selfLoop
	for _, edge := range graph.Edges {
		src, ok1 := index[edge.Start]
		dst, ok2 := index[edge.End]
		if !ok1 || !ok2 {
			continue
		}
		label := labelOf(edge.Label, true, smallSize, 200, 0)
		var lp *selfLoop
		if src == dst {
			lp = &selfLoop{label: &label}
			if _, ok := loops[src]; !ok {
				loopOrder = append(loopOrder, src)
			}
			loops[src] = append(loops[src], lp)
		}
		loopOf = append(loopOf, lp)
		drawn = append(drawn, edge)
		edgeLabels = append(edgeLabels, &label)
		ce := graphs.CEdge{Src: src, Dst: dst, Minlen: 1}
		if label.Width > 0 {
			ce.LabelW = label.Width + 12
		}
		if label.Height > 0 {
			ce.LabelH = label.Height + 4
		}
		cedges = append(cedges, ce)
	}

	// horizontal is whether the level node i is laid out in runs across: its nearest
	// subgraph's direction that sets one, or the diagram's. Subgraphs can name each other as
	// members, so the walk up is bounded. It is the horizontal of Pele's draw.
	horizontal := func(i int) bool {
		at := i
		for steps := 0; steps < len(graph.Nodes); steps++ {
			n := graph.Nodes[at]
			if !n.HasParent {
				break
			}
			up, ok := index[n.Parent]
			if !ok {
				break
			}
			if dir := graph.Nodes[up].Dir; dir != "" {
				return dir == "LR" || dir == "RL"
			}
			at = up
		}
		return db.Direction == "LR" || db.Direction == "RL"
	}

	views := make([]view, 0, len(graph.Nodes))
	cnodes := make([]graphs.CNode, 0, len(graph.Nodes))
	for i, node := range graph.Nodes {
		decls := node.CSSCompiledStyles
		if len(node.CSSStyles) > 0 {
			decls = append(append([]string(nil), node.CSSCompiledStyles...), node.CSSStyles...)
		}
		author := css.Declarations(decls)
		var face text.Face
		if author.Bold {
			face |= text.Bold
		}
		if author.Italic {
			face |= text.Italic
		}
		nameSize, rowSize, keySize := size, smallSize, jsmath.Round(float64(size*0.75))
		if author.FontSize > 0 {
			nameSize, rowSize, keySize = author.FontSize, author.FontSize, author.FontSize
		}
		var label text.Label
		if node.IsGroup {
			label = labelOf(node.Label, true, nameSize, 4000, face)
		} else {
			name := node.Alias
			if name == "" {
				name = node.Label
			}
			label = labelOf(name, true, nameSize, 240, face)
		}
		own := loops[i]
		// Loops leave by a side the layout keeps free of other edges: the right when ranks
		// run down, the bottom when they run across.
		below := len(own) > 0 && !node.IsGroup && horizontal(i)
		v := view{node: node, author: author, label: label, rowSize: rowSize, keySize: keySize, below: below}
		reach, bands := 0.0, 0.0
		for _, lp := range own {
			if below {
				lp.band = max(loopBand, lp.label.Width+12)
				reach = max(reach, lp.label.Height)
			} else {
				lp.band = max(loopBand, lp.label.Height+6)
				reach = max(reach, lp.label.Width)
			}
			bands += lp.band
		}
		minW := float64(minWidth)
		if below {
			minW = max(minWidth, bands+8)
		}
		var c graphs.CNode
		if node.IsGroup {
			c = graphs.NewCNode(0, 0, -1)
			c.IsGroup = true
			if node.Dir != "" {
				c.Dir = graphs.Direction(node.Dir)
			}
			c.PadX = groupPad
			if label.Height > 0 {
				c.PadTop = label.Height + 16
			} else {
				c.PadTop = groupPad
			}
			c.PadBottom = groupPad
			c.MinW = max(label.Width+2*groupPad-12, minWidth)
		} else {
			nameH := max(label.Height, jsmath.Round(float64(nameSize*1.5)))
			if len(node.Attributes) == 0 {
				v.w = max(label.Width+2*boxPadX, minW)
				if below {
					v.h = max(nameH+2*boxPadY, 0)
				} else {
					v.h = max(nameH+2*boxPadY, bands+8)
				}
			} else {
				rowLine := jsmath.Round(float64(rowSize * 1.5))
				var widths [4]float64
				for _, a := range node.Attributes {
					cells := [4]text.Label{
						labelOf(cellText(a.Type), true, rowSize, noWrap, face),
						labelOf(cellText(a.Name), true, rowSize, noWrap, face),
						labelOf(strings.Join(a.Keys, ", "), false, keySize, noWrap, face),
						labelOf(cellText(a.Comment), true, rowSize, commentWrap, face),
					}
					h := rowLine
					for k := range cells {
						if cells[k].Width > widths[k] {
							widths[k] = cells[k].Width
						}
						if cells[k].Height > h {
							h = cells[k].Height
						}
					}
					v.rows = append(v.rows, row{cells: cells, h: h + rowPad})
					v.h += h + rowPad
				}
				columns := 0
				inner := 0.0
				for _, w := range widths {
					if w == 0 {
						continue
					}
					if columns > 0 {
						inner += w + cellGap
					} else {
						inner += w
					}
					columns++
				}
				v.w = max(inner+2*cellPadX, label.Width+2*boxPadX, minW)
				// A box wider than its table shares the spare width between the columns.
				spare := 0.0
				if columns > 0 {
					spare = (v.w - inner - 2*cellPadX) / float64(columns)
				}
				x := float64(cellPadX)
				for _, w := range widths {
					v.cols = append(v.cols, x)
					if w > 0 {
						x += w + spare + cellGap
					}
				}
				v.headH = nameH + 2*headPadY
				v.h += v.headH
			}
			// The layout keeps a node centered in its box, so room for loops is added on both
			// sides.
			room := 0.0
			if len(own) > 0 {
				room = float64(2 * (markerLength + loopLength*0.75 + 10 + reach))
			}
			if below {
				c = graphs.NewCNode(v.w, v.h+room, -1)
			} else {
				c = graphs.NewCNode(v.w+room, v.h, -1)
			}
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

	rootDir := graphs.Direction(db.Direction)
	if turned {
		rootDir = graphs.TB
	}
	// The host may ask for edges at right angles, or straight, as it may of a flowchart.
	curve := common.RoutingCurve(o.Routing)
	result := graphs.Compound(cnodes, cedges, rootDir, graphs.LayeredOptions{
		Tracks:   curve == "step",
		Straight: curve == "linear",
		NodeSep:  numberOption(config, "nodeSpacing", nodeSep),
		EdgeSep:  edgeSep,
		RankSep:  numberOption(config, "rankSpacing", rankSep),
		PortSep:  24,
	})

	spreadEnds(cedges, cnodes, views)

	// A subgraph gets its size from the layout, so its loops hang outside it and may widen
	// the diagram.
	layoutWidth := result.Width
	for _, i := range loopOrder {
		c := &cnodes[i]
		if !c.IsGroup {
			continue
		}
		reach := 0.0
		for _, lp := range loops[i] {
			reach = max(reach, lp.label.Width)
		}
		layoutWidth = max(layoutWidth, c.X+c.W/2+markerLength+loopLength*0.75+10+reach)
	}

	title := labelOf(db.Title, false, size, 4000, text.Bold)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	inner := max(layoutWidth, title.Width)
	ox := padding + (inner-layoutWidth)/2
	oy := padding + titleHeight
	graphs.Shift(cnodes, cedges, ox, oy)
	sc := &scene.Scene{
		Type:     "er",
		Width:    math.Ceil(inner + 2*padding),
		Height:   math.Ceil(result.Height + titleHeight + 2*padding),
		FontSize: size,
		Title:    db.AccTitle,
		Desc:     db.AccDescr,
	}
	// node is cnode i as the drawing helpers take it, where Pele's draw hands them the cnode
	// itself.
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
		classes := common.ClassNames(common.TrimDefault(n.CSSClasses))
		if c.IsGroup {
			v.w, v.h = c.W, c.H
			box := scene.Rect{X: c.X - c.W/2, Y: c.Y - c.H/2, W: c.W, H: c.H}
			titleX := draw.ClusterTitleX(node(i), v.label.Width, crossings, groupPad)
			cluster := scene.Item{
				Kind: scene.Group, Role: RoleCluster, ID: n.ID, Class: classes, Box: box,
				Parts:  []scene.Part{{Box: box, Rounded: true}},
				Style:  style.Style{Fill: style.SurfaceAlt, FillOpacity: style.N(0.5), Stroke: style.Border, Text: style.Muted},
				Author: v.author.Style,
			}
			if !v.label.Empty() {
				cluster.Label = &v.label
				cluster.At = scene.Point{X: titleX, Y: box.Y + 8 + v.label.Height/2}
			}
			clusters = append(clusters, cluster)
			continue
		}
		nodes = append(nodes, entity(v, c.X, c.Y, classes, o)...)
	}

	// Self-relationships share the side they loop out of, squeezed together if it is too
	// short for them.
	for _, i := range loopOrder {
		v := &views[i]
		own := loops[i]
		total := 0.0
		for _, lp := range own {
			total += lp.band
		}
		side := v.h
		if v.below {
			side = v.w
		}
		scale := min(1, max(side-8, 0)/total)
		at := (-total * scale) / 2
		for _, lp := range own {
			lp.band = float64(lp.band * scale)
			lp.at = at + lp.band/2
			at += lp.band
		}
	}

	for i := range cedges {
		ce := &cedges[i]
		edge := drawn[i]
		label := edgeLabels[i]
		start, end := scene.MarkerType("er."+edge.ArrowTypeStart), scene.MarkerType("er."+edge.ArrowTypeEnd)
		var path *scene.Path
		var ends [2]scene.Marker
		var route []scene.Point
		if lp := loopOf[i]; lp != nil {
			c := &cnodes[ce.Src]
			v := &views[ce.Src]
			spread := max(min(lp.band/2-6, 13), 1)
			far := markerLength + loopLength*0.75 + 6
			if v.below {
				x := c.X + lp.at
				y := c.Y + v.h/2
				turn := y + markerLength
				out := y + markerLength + loopLength
				path = new(scene.Path).MoveTo(x-spread, y).LineTo(x-spread, turn).
					CubeTo(x-spread, out, x+spread, out, x+spread, turn).LineTo(x+spread, y)
				ends = [2]scene.Marker{
					{Type: start, At: scene.Point{X: x - spread, Y: y}, DX: 0, DY: -1},
					{Type: end, At: scene.Point{X: x + spread, Y: y}, DX: 0, DY: -1},
				}
				ce.LabelX = x
				ce.LabelY = y + far + label.Height/2
			} else {
				x := c.X + v.w/2
				y := c.Y + lp.at
				turn := x + markerLength
				out := x + markerLength + loopLength
				path = new(scene.Path).MoveTo(x, y-spread).LineTo(turn, y-spread).
					CubeTo(out, y-spread, out, y+spread, turn, y+spread).LineTo(x, y+spread)
				ends = [2]scene.Marker{
					{Type: start, At: scene.Point{X: x, Y: y - spread}, DX: -1, DY: 0},
					{Type: end, At: scene.Point{X: x, Y: y + spread}, DX: -1, DY: 0},
				}
				ce.LabelX = x + far + label.Width/2
				ce.LabelY = y
			}
		} else {
			r := straighten(ce.Route)
			if len(r) < 6 {
				continue
			}
			inset(r, 0, &views[ce.Src], &cnodes[ce.Src])
			inset(r, len(r)-3, &views[ce.Dst], &cnodes[ce.Dst])
			// A curve stops short of each box so that the marker sits on a straight run of
			// line.
			dx := math.Abs(r[len(r)-3] - r[0])
			dy := math.Abs(r[len(r)-2] - r[1])
			direct := len(r) == 6 && (dx < 0.01 || dy < 0.01 || jsmath.Hypot(dx, dy) <= 3*markerLength)
			stub := 0.0
			if !direct {
				stub = markerLength
			}
			ep := draw.RoutePath(r, curve, stub, stub, draw.OwnReach(stub), draw.OwnReach(stub))
			path = ep.Path
			if stub > 0 {
				path = new(scene.Path).MoveTo(ep.SX, ep.SY)
				ep.Path.Each(func(op scene.Op, a []float64) {
					switch op {
					case scene.MoveTo:
						path.LineTo(a[0], a[1])
					case scene.LineTo:
						path.LineTo(a[0], a[1])
					case scene.QuadTo:
						path.QuadTo(a[0], a[1], a[2], a[3])
					case scene.CubeTo:
						path.CubeTo(a[0], a[1], a[2], a[3], a[4], a[5])
					case scene.Close:
						path.Close()
					}
				})
				path.LineTo(ep.EX, ep.EY)
			}
			ends = ep.Ends(start, end)
			for k := 0; k+1 < len(r); k += 3 {
				route = append(route, scene.Point{X: r[k], Y: r[k+1]})
			}
		}
		// Pele draws the edges in a group that ends their lines round.
		own := style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound}
		if edge.Pattern == "dashed" {
			own.Dash = style.Dashes(4, 3)
		}
		if len(route) == 0 {
			// A self-loop, which the layout gives no route: the corners its curve is drawn about.
			route = path.Corners()
		}
		edges = append(edges, scene.Item{
			Kind: scene.Edge, Role: RoleRelationship, ID: edge.ID,
			Box:   path.Bounds(),
			Parts: []scene.Part{{Path: path, Paint: scene.StrokeOnly}},
			Route: route,
			Ends:  ends,
			Style: own,
		})
		if label.Width > 0 {
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: RoleRelationshipLabel, ID: edge.ID,
				Box:   scene.Rect{X: ce.LabelX - (label.Width+8)/2, Y: ce.LabelY - label.Height/2, W: label.Width + 8, H: label.Height},
				Parts: []scene.Part{draw.LabelPatch(label, ce.LabelX, ce.LabelY)},
				Label: label,
				At:    scene.Point{X: ce.LabelX, Y: ce.LabelY},
				Style: style.Style{Fill: style.Bg, Stroke: style.None},
			})
		}
	}

	sc.Items = make([]scene.Item, 0, len(titles)+len(clusters)+len(edges)+len(labels)+len(nodes))
	for _, layer := range [][]scene.Item{titles, clusters, edges, labels, nodes} {
		sc.Items = append(sc.Items, layer...)
	}
	return sc
}

// entity is an entity drawn about x, y, as Pele's draw draws one: its box, with its name, and,
// when it has attributes, its table: the stripes of every other row and its outline drawn
// again over them as parts, and the line under its name and the text of each cell as items
// of their own after it.
func entity(v *view, x, y float64, classes []string, o *layout.Options) []scene.Item {
	n := v.node
	w, h := v.w, v.h
	x0, y0 := -w/2, -h/2
	box := scene.Rect{X: x + x0, Y: y + y0, W: w, H: h}
	it := scene.Item{
		Kind: scene.Node, Role: RoleEntity, Shape: "rounded", ID: n.Label, Class: classes,
		Box:    box,
		Parts:  []scene.Part{{Box: box, Rounded: true}},
		Style:  style.Style{Fill: style.Surface, Stroke: style.Border},
		Author: v.author.Style,
		Label:  &v.label, At: scene.Point{X: x, Y: y},
	}
	if hasClass(classes, "internal-link") && !v.label.Empty() && !o.NoLinks {
		// The name ends up in an href, so it gets the same check as a URL.
		name := n.Alias
		if name == "" {
			name = n.Label
		}
		display := text.DecodeEntities(name)
		if _, ok := url.LinkURL(url.SanitizeURL(display), true, o.LinkSchemes); ok {
			it.Link = display
		}
	}
	if len(v.rows) == 0 {
		return []scene.Item{it}
	}
	it.At.Y = y + (y0 + v.headH/2)
	rowLine := jsmath.Round(float64(v.rowSize * 1.5))
	textStyle := style.Style{Text: v.author.Style.Text, Opacity: v.author.Style.Opacity}
	var cells []scene.Item
	top := y0 + v.headH
	for r := range v.rows {
		rw := &v.rows[r]
		if r%2 == 0 {
			stripe := scene.Rect{X: x + x0, Y: y + top, W: w, H: rw.h}
			if r == len(v.rows)-1 {
				// The last row follows the box's rounded bottom corners: a rounded box,
				// squared off on top.
				half := stripe
				half.H = rw.h / 2
				it.Parts = append(it.Parts, scene.Part{Box: stripe, Rounded: true, Paint: scene.Shade}, scene.Part{Box: half, Paint: scene.Shade})
			} else {
				it.Parts = append(it.Parts, scene.Part{Box: stripe, Paint: scene.Shade})
			}
		}
		baseline := top + rowPad/2 + rowLine/2 + float64(v.rowSize*0.35)
		for k := range rw.cells {
			l := &rw.cells[k]
			if l.Empty() {
				continue
			}
			own := style.Style{Text: style.Muted}
			if k == 1 {
				own = style.Style{}
			}
			// The label is centered on At down, and its first line's baseline is where Pele
			// writes it.
			at := y + (baseline - float64(l.Size*0.35) + l.Height/2 - l.LineHeight/2)
			cells = append(cells, scene.Item{
				Kind: scene.Text, Role: cellRoles[k], ID: n.Label,
				Label: l, At: scene.Point{X: x + (x0 + v.cols[k]), Y: at}, Align: scene.Start,
				Style: own, Author: textStyle,
			})
		}
		top += rw.h
	}
	it.Parts = append(it.Parts, scene.Part{Box: box, Rounded: true, Paint: scene.StrokeOnly})
	// The line under the name is an item of its own, as Pele gives it only the stroke color
	// the author set and none of the rest of the entity's style. Pele draws it before the
	// outline drawn again over the stripes; here it is drawn after it, in the same color.
	at := y + (y0 + v.headH)
	line := new(scene.Path).MoveTo(x+x0, at).LineTo(x+(x0+w), at)
	divider := scene.Item{
		Kind: scene.Mark, Role: RoleDivider, ID: n.Label, Box: line.Bounds(),
		Parts:  []scene.Part{{Path: line, Paint: scene.StrokeOnly}},
		Style:  style.Style{Stroke: style.Border},
		Author: style.Style{Stroke: v.author.Style.Stroke},
	}
	return append([]scene.Item{it, divider}, cells...)
}

// straighten drops the points of a route that lie on a straight run between their
// neighbours, as Pele's straighten does.
func straighten(route []float64) []float64 {
	out := append([]float64(nil), route[:min(3, len(route))]...)
	for k := 3; k < len(route); k += 3 {
		prev := len(out) - 3
		// flat reports whether point k lies on a straight run along axis between the last
		// point kept and the next, as the flat of Pele's straighten does.
		flat := func(axis int) bool {
			return math.Abs(out[prev+axis]-route[k+axis]) < 0.01 && math.Abs(route[k+axis]-route[k+3+axis]) < 0.01
		}
		if k+3 < len(route) && (flat(0) || flat(1)) {
			continue
		}
		out = append(out, route[k], route[k+1], route[k+2])
	}
	return out
}

// An end is the end of a route on a side of a box, and where along that side its next point
// lies: Pele's End.
type end struct {
	route  []float64
	at     int
	toward float64
}

// spreadEnds spaces the ends of the edges on each side of a box along it, in the order they
// fan out, as Pele's spreadEnds does: the layout brings every edge that ends on one side to
// its middle, where their markers would cover each other.
func spreadEnds(cedges []graphs.CEdge, cnodes []graphs.CNode, views []view) {
	sides := map[int][]*end{}
	var order []int
	// add files the end of a route at index at under the side of node's box it lies on, with
	// where the next point lies along that side, as the add of Pele's spreadEnds does.
	add := func(node int, route []float64, at, next int) {
		c := &cnodes[node]
		dx := route[at] - c.X
		dy := route[at+1] - c.Y
		var side int
		switch {
		case math.Abs(math.Abs(dy)-c.H/2) < 0.5:
			side = 2
			if dy < 0 {
				side = 0
			}
		case math.Abs(math.Abs(dx)-c.W/2) < 0.5:
			side = 1
			if dx < 0 {
				side = 3
			}
		default:
			return
		}
		key := node*4 + side
		if _, ok := sides[key]; !ok {
			order = append(order, key)
		}
		sides[key] = append(sides[key], &end{route: route, at: at, toward: route[next+(side&1)]})
	}
	for _, e := range cedges {
		last := len(e.Route) - 3
		if e.Src == e.Dst || last < 3 {
			continue
		}
		add(e.Src, e.Route, 0, 3)
		add(e.Dst, e.Route, last, last-3)
	}
	for _, key := range order {
		ends := sides[key]
		if len(ends) < 2 {
			continue
		}
		node := key >> 2
		vertical := key&1 == 1
		c := &cnodes[node]
		w, h := views[node].w, views[node].h
		if c.IsGroup {
			w, h = c.W, c.H
		}
		length, center, axis := w, c.X, 0
		if vertical {
			length, center, axis = h, c.Y, 1
		}
		step := max(min(portGap, (length-2*portMargin)/float64(len(ends)-1)), 0)
		slices.SortStableFunc(ends, func(a, b *end) int {
			switch {
			case a.toward < b.toward:
				return -1
			case a.toward > b.toward:
				return 1
			}
			return 0
		})
		for i, e := range ends {
			e.route[e.at+axis] = center + float64((float64(i)-float64(len(ends)-1)/2)*step)
		}
	}
}

// inset moves a route's end from the side of the larger box an entity with a
// self-relationship is laid out in onto the entity's own side, as Pele's inset does.
func inset(route []float64, at int, v *view, c *graphs.CNode) {
	if c.IsGroup {
		return
	}
	dx := route[at] - c.X
	dy := route[at+1] - c.Y
	if c.W != v.w && math.Abs(math.Abs(dx)-c.W/2) < 0.5 {
		route[at] -= jsmath.Sign(dx) * (c.W - v.w) / 2
	}
	if c.H != v.h && math.Abs(math.Abs(dy)-c.H/2) < 0.5 {
		route[at+1] -= jsmath.Sign(dy) * (c.H - v.h) / 2
	}
}

// hasClass reports whether name is among the classes, as the test / internal-link(?: |$)/ of
// Pele's draw finds it in the string of them.
func hasClass(classes []string, name string) bool {
	return slices.Contains(classes, name)
}
