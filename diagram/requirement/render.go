package requirement

import (
	"cmp"
	"math"
	"slices"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/draw"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of a requirement diagram, as Pele's render.ts has them.
const (
	nodeSep   = 40
	edgeSep   = 16
	rankSep   = 56
	padX      = 14 // the room inside a box, either side of its text
	padY      = 10 // and above and below its head and its fields
	rowGap    = 2
	columnGap = 12 // between a field's name and its value
	minWidth  = 120
	nameWrap  = 280
	valueWrap = 240
	loop      = 26
	endStep   = 28 // the most the ends of edges on one side of a box are set apart by
	endMargin = 14 // and the room they leave at its corners
)

// The markers of a relationship: a containment mark at the source of contains, and an open
// arrowhead at the target of the others.
const (
	containsMarker scene.MarkerType = "requirement.contains"
	arrowMarker    scene.MarkerType = "requirement.arrow"
)

// A row is a field of a box: its name, its value and the height of the taller.
type row struct {
	key, value text.Label
	height     float64
}

// A view is a requirement or an element as it is drawn: Pele's NodeView.
type view struct {
	node       *GraphNode
	author     style.Author
	kind, name text.Label
	rows       []row
	keyWidth   float64
	headHeight float64
	w, h       float64
}

// restyle adds bold or italic to a whole label after its markdown is read, since emphasis
// markers only toggle styles the label does not start with, as Pele's restyle does.
func restyle(l text.Label, bits text.Face, m text.Measurer) text.Label {
	if bits == 0 || len(l.Lines) == 0 {
		return l
	}
	width := 0.0
	widths := make([]float64, 0, len(l.Lines))
	lines := make([][]text.Span, len(l.Lines))
	for i, line := range l.Lines {
		total := 0.0
		spans := make([]text.Span, len(line))
		for j, s := range line {
			s.Face |= bits
			if s.Icon == "" {
				s.Width = m.Width(s.Text, l.Size, s.Face)
			}
			total += s.Width
			spans[j] = s
		}
		widths = append(widths, total)
		if total > width {
			width = total
		}
		lines[i] = spans
	}
	l.Lines, l.Widths, l.Width = lines, widths, width
	return l
}

// spreadEnds sets apart the ends of edges that share a side of a box, as Pele's spreadEnds
// does. The layout attaches every edge to the middle of a side; ends that share one are
// spread along it, in the order of where they head, so their markers do not sit on each
// other.
func spreadEnds(cnodes []layout.CNode, cedges []layout.CEdge, views []view) {
	type end struct {
		route  []float64
		at     int
		toward float64
	}
	sides := map[int][]end{}
	var keys []int // in the order they were added, as a Map iterates
	for i := range cedges {
		edge := &cedges[i]
		route := edge.Route
		if edge.Src == edge.Dst || len(route) < 6 {
			continue
		}
		for _, at := range [2]int{0, len(route) - 3} {
			node, next := edge.Src, 3
			if at != 0 {
				node, next = edge.Dst, at-3
			}
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
				continue
			}
			key := node*4 + side
			toward := route[next]
			if side%2 != 0 {
				toward = route[next+1]
			}
			if _, ok := sides[key]; !ok {
				keys = append(keys, key)
			}
			sides[key] = append(sides[key], end{route, at, toward})
		}
	}
	for _, key := range keys {
		list := sides[key]
		if len(list) < 2 {
			continue
		}
		node := key >> 2
		vertical := key%2 == 1
		span := views[node].w
		if vertical {
			span = views[node].h
		}
		span -= 2 * endMargin
		step := max(0, min(endStep, span/float64(len(list)-1)))
		slices.SortStableFunc(list, func(a, b end) int { return cmp.Compare(a.toward, b.toward) })
		for j, e := range list {
			offset := float64((float64(j) - float64(len(list)-1)/2) * step)
			if vertical {
				e.route[e.at+1] = cnodes[node].Y + offset
			} else {
				e.route[e.at] = cnodes[node].X + offset
			}
		}
	}
}

// Layout lays out a parsed requirement diagram as a scene, as Pele's renderRequirement draws
// it. A diagram that runs across and does not fit o.MaxWidth is laid out running down
// instead, where that is narrower.
func Layout(db *DB, _ diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	if !common.RunsAcross(db.Direction) {
		return lay(db, o, false)
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return lay(db, o, down) })
}

// lay lays the diagram out once, as Pele's draw does. down lays out a diagram that runs
// across as one that runs down.
func lay(db *DB, o *layout.Options, down bool) *scene.Scene {
	graph := BuildGraph(db)
	size := o.Size()
	small := jsmath.Round(float64(size * 0.875))
	measure := o.Measure()
	const padding = common.Padding
	labelOf := func(raw string, markdown bool, size, max float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: face, Markdown: markdown, Icons: o.Icons})
	}

	index := make(map[string]int, len(graph.Nodes))
	views := make([]view, 0, len(graph.Nodes))
	cnodes := make([]layout.CNode, 0, len(graph.Nodes))
	for i := range graph.Nodes {
		node := &graph.Nodes[i]
		author := style.Declarations(node.CSSStyles)
		var base text.Face
		if author.Bold {
			base |= text.Bold
		}
		if author.Italic {
			base |= text.Italic
		}
		nameSize, bodySize := size, small
		if author.FontSize > 0 {
			nameSize, bodySize = author.FontSize, author.FontSize
		}
		var fields [][2]string
		kindName := "Element"
		if r := node.Requirement; r != nil {
			kindName = r.Type
			fields = [][2]string{{"ID", r.RequirementID}, {"Text", r.Text}, {"Risk", r.Risk}, {"Verification", r.VerifyMethod}}
		} else {
			fields = [][2]string{{"Type", node.Element.Type}, {"Doc Ref", node.Element.DocRef}}
		}
		kind := restyle(labelOf("«"+kindName+"»", false, bodySize, 4000, 0), base, measure)
		name := restyle(labelOf(node.ID, true, nameSize, nameWrap, 0), base|text.Bold, measure)
		var rows []row
		keyWidth, valueWidth, bodyHeight := 0.0, 0.0, 0.0
		for _, f := range fields {
			if f[1] == "" {
				continue
			}
			key := restyle(labelOf(f[0], false, bodySize, 4000, 0), base, measure)
			value := restyle(labelOf(f[1], true, bodySize, valueWrap, 0), base, measure)
			height := max(key.Height, value.Height)
			rows = append(rows, row{key, value, height})
			keyWidth = max(keyWidth, key.Width)
			valueWidth = max(valueWidth, value.Width)
			bodyHeight += height + rowGap
		}
		if len(rows) > 0 {
			bodyHeight += 2*padY - rowGap
		}
		headHeight := 2*padY + kind.Height + name.Height
		fieldsWidth := 0.0
		if len(rows) > 0 {
			fieldsWidth = keyWidth + columnGap + valueWidth + 2*padX
		}
		w := max(minWidth, max(kind.Width, name.Width)+2*padX, fieldsWidth)
		// A requirement and an element may share a name; relationships then go to the
		// requirement.
		if _, ok := index[node.ID]; !ok {
			index[node.ID] = len(views)
		}
		c := layout.NewCNode(w, headHeight+bodyHeight, -1)
		c.Seq = float64(len(views))
		views = append(views, view{node: node, author: author, kind: kind, name: name, rows: rows,
			keyWidth: keyWidth, headHeight: headHeight, w: w, h: headHeight + bodyHeight})
		cnodes = append(cnodes, c)
	}

	rootDir := layout.Direction(graph.Direction)
	if down {
		rootDir = layout.TB
	}
	sideways := rootDir == layout.LR || rootDir == layout.RL
	var cedges []layout.CEdge
	var edgeLabels []text.Label
	var drawn []*GraphEdge
	for i := range graph.Edges {
		edge := &graph.Edges[i]
		if !edge.HasStart || !edge.HasEnd {
			continue
		}
		src, dst := index[edge.Start], index[edge.End]
		label := labelOf("«"+edge.Type+"»", false, small, 4000, 0)
		if src == dst {
			// A self-loop sits beside the node, across the flow, and the node's box grows to
			// hold it.
			if sideways {
				cnodes[src].H += 2 * (loop + label.Height + 8)
			} else {
				cnodes[src].W += 2 * (loop + label.Width + 12)
			}
		}
		drawn = append(drawn, edge)
		edgeLabels = append(edgeLabels, label)
		cedges = append(cedges, layout.CEdge{Src: src, Dst: dst, Minlen: 1, LabelW: label.Width + 12, LabelH: label.Height + 4})
	}

	result := layout.Compound(cnodes, cedges, rootDir, layout.LayeredOptions{
		NodeSep: nodeSep,
		EdgeSep: edgeSep,
		RankSep: rankSep,
		PortSep: 20,
	})

	spreadEnds(cnodes, cedges, views)

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
		Type:     "requirement",
		Width:    math.Ceil(inner + 2*padding),
		Height:   math.Ceil(result.Height + titleHeight + 2*padding),
		FontSize: size,
		Title:    db.AccTitle,
		Desc:     db.AccDescr,
	}

	var titles, edges, labels, nodes []scene.Item
	if !title.Empty() {
		titles = append(titles, scene.Item{
			Kind: scene.Text, Role: "requirement.title", Label: &title,
			At: scene.Point{X: sc.Width / 2, Y: padding + title.Height/2},
		})
	}

	for i := range views {
		v := &views[i]
		x, y := cnodes[i].X, cnodes[i].Y
		left, top := -v.w/2, -v.h/2
		box := scene.Rect{X: x + left, Y: y + top, W: v.w, H: v.h}
		role := scene.Role("requirement.element")
		if v.node.Requirement != nil {
			role = "requirement.requirement"
		}
		it := scene.Item{
			Kind: scene.Node, Role: role, Shape: "rect", ID: v.node.ID,
			Class:  common.ClassNames(common.TrimDefault(v.node.CSSClasses)),
			Box:    box,
			Parts:  []scene.Part{{Box: box, Rounded: true}},
			Label:  &v.name,
			At:     scene.Point{X: x, Y: y + (top + padY + v.kind.Height + v.name.Height/2)},
			Style:  style.Style{Fill: style.Surface, Stroke: style.Border},
			Author: v.author.Style,
		}
		texts := []scene.Item{textItem("requirement.kind", &v.kind, scene.Middle, x, y+(top+padY+v.kind.Height/2), style.Muted, v.author)}
		if len(v.rows) > 0 {
			divider := top + v.headHeight
			it.Parts = append(it.Parts, scene.Part{
				Path:  new(scene.Path).MoveTo(x+left, y+divider).LineTo(x-left, y+divider),
				Paint: scene.StrokeOnly,
			})
			at := divider + padY
			for k := range v.rows {
				r := &v.rows[k]
				texts = append(texts,
					textItem("requirement.field", &r.key, scene.Start, x+(left+padX), y+(at+r.key.Height/2), style.Muted, v.author),
					textItem("requirement.value", &r.value, scene.Start, x+(left+padX+v.keyWidth+columnGap), y+(at+r.value.Height/2), style.Fg, v.author))
				at += r.height + rowGap
			}
		}
		nodes = append(nodes, it)
		nodes = append(nodes, texts...)
	}

	loopSeen := make([]int, len(views))
	for i := range cedges {
		ce := &cedges[i]
		edge := drawn[i]
		label := &edgeLabels[i]
		var start, end scene.MarkerType
		if edge.Contains {
			start = containsMarker
		} else {
			end = arrowMarker
		}
		startTrim := start.Trim()
		labelX, labelY := ce.LabelX, ce.LabelY
		var path draw.EdgePath
		var route []float64
		if ce.Src == ce.Dst {
			v := &views[ce.Src]
			cx, cy := cnodes[ce.Src].X, cnodes[ce.Src].Y
			k := loopSeen[ce.Src]
			loopSeen[ce.Src]++
			half := v.h / 2
			if sideways {
				half = v.w / 2
			}
			spread := min(half-4, float64(12+k*6))
			reach := float64(loop + k*8)
			// Local frame: out points away from the node, along runs along its side.
			at := func(out, along float64) (float64, float64) {
				if sideways {
					return cx + along, cy + v.h/2 + out
				}
				return cx + v.w/2 + out, cy + along
			}
			// A cubic whose control points are this far out peaks at reach.
			pull := reach / 0.75
			p := new(scene.Path).MoveTo(at(startTrim, -spread))
			ax, ay := at(pull, -spread)
			bx, by := at(pull, spread)
			ex, ey := at(0, spread)
			p.CubeTo(ax, ay, bx, by, ex, ey)
			path = draw.EdgePath{Path: p}
			if sideways {
				path.SX, path.SY, path.SDX, path.SDY = cx-spread, cy+v.h/2, 0, -1
				path.EX, path.EY, path.EDX, path.EDY = cx+spread, cy+v.h/2, 0, -1
				labelX, labelY = cx, cy+v.h/2+reach+4+ce.LabelH/2
			} else {
				path.SX, path.SY, path.SDX, path.SDY = cx+v.w/2, cy-spread, -1, 0
				path.EX, path.EY, path.EDX, path.EDY = cx+v.w/2, cy+spread, -1, 0
				labelX, labelY = cx+v.w/2+reach+4+ce.LabelW/2, cy
			}
		} else {
			route = append([]float64(nil), ce.Route...)
			path = draw.RoutePath(route, "", startTrim, 0, draw.OwnReach(startTrim), 0)
		}
		// Pele draws the edges in a group that ends their lines round.
		own := style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound}
		if !edge.Contains {
			own.Dash = style.Dashes(5, 4)
		}
		it := scene.Item{
			Kind: scene.Edge, Role: "requirement.relationship", ID: edge.ID,
			Box:   path.Path.Bounds(),
			Parts: []scene.Part{{Path: path.Path, Paint: scene.StrokeOnly}},
			Ends:  path.Ends(start, end),
			Style: own,
		}
		for k := 0; k+1 < len(route); k += 3 {
			it.Route = append(it.Route, scene.Point{X: route[k], Y: route[k+1]})
		}
		edges = append(edges, it)
		labels = append(labels, scene.Item{
			Kind: scene.Text, Role: "requirement.relationship-label", ID: edge.ID,
			Box:   scene.Rect{X: labelX - (label.Width+8)/2, Y: labelY - label.Height/2, W: label.Width + 8, H: label.Height},
			Parts: []scene.Part{draw.LabelPatch(label, labelX, labelY)},
			Label: label,
			At:    scene.Point{X: labelX, Y: labelY},
			Style: style.Style{Fill: style.Bg, Stroke: style.None, Text: style.Muted},
		})
	}

	sc.Items = make([]scene.Item, 0, len(titles)+len(edges)+len(labels)+len(nodes))
	for _, layer := range [][]scene.Item{titles, edges, labels, nodes} {
		sc.Items = append(sc.Items, layer...)
	}
	return sc
}

// textItem is a text of a box beside its name: its kind, or a field's name or value, drawn
// about x, y in the color own and the color of text its author gave the box.
func textItem(role scene.Role, l *text.Label, align scene.Align, x, y float64, own style.Paint, author style.Author) scene.Item {
	box := scene.Rect{X: x - l.Width/2, Y: y - l.Height/2, W: l.Width, H: l.Height}
	if align == scene.Start {
		box.X = x
	}
	return scene.Item{
		Kind: scene.Text, Role: role, Box: box, Label: l, At: scene.Point{X: x, Y: y}, Align: align,
		Style:  style.Style{Text: own},
		Author: style.Style{Text: author.Style.Text},
	}
}
