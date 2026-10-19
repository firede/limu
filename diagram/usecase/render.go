package usecase

import (
	"math"
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/draw"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The spacing and measures of a use case diagram, as Pele's render.ts has them.
const (
	nodeSep   = 32
	edgeSep   = 16
	rankSep   = 72
	groupPad  = 20
	loop      = 26
	figureW   = 40
	figureH   = 60
	figureGap = 6
	tabPad    = 6
	cellPad   = 10
	rowPad    = 8
	fold      = 9
	extension = 12
	portGap   = 14
)

// The kinds of node, Pele's Kind.
const (
	kindActor = iota
	kindEllipse
	kindBox
	kindJSON
	kindBoundary
)

// A view is a node as it is drawn: Pele's View. w and h are the size of what is drawn; the
// layout's box is larger when the node carries a self-relationship.
type view struct {
	node       *GraphNode
	kind       int
	author     style.Author
	label      text.Label
	stereotype text.Label
	w, h       float64
	rows       [][2]text.Label
	keyW       float64
	headH      float64
	loops      int
}

// numberOption is a number of the diagram's config, if it is one above 0, at most 2000, as
// Pele's numberOption reads it.
func numberOption(config diagram.Config, key string, fallback float64) float64 {
	section, _ := config["usecase"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v > 0 {
		return min(v, 2000)
	}
	return fallback
}

// literal is a plain label as the label parser is to show it: no tags, no markdown, no `\n`
// breaks. Its markup characters are handed over as character references, which the parser
// turns back into characters, as Pele's literal does.
func literal(s string) string {
	if !strings.ContainsAny(s, "&<>\\") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '&':
			b.WriteString("&#38;")
		case '<':
			b.WriteString("&#60;")
		case '>':
			b.WriteString("&#62;")
		case '\\':
			b.WriteString("&#92;")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// anchor is the point of a node's outline on one of its four sides (0 top, 1 right, 2
// bottom, 3 left), along away from the middle of that side, as Pele's anchor finds it. Lines
// meet an actor at its figure, not at its label.
func anchor(v *view, c *layout.CNode, side int, along float64) (float64, float64) {
	sign := -1.0
	if side == 1 || side == 2 {
		sign = 1
	}
	upright := side == 1 || side == 3
	reach := v.h / 2
	if upright {
		reach = v.w / 2
	}
	middle := 0.0
	switch v.kind {
	case kindActor:
		if upright {
			reach = figureW/2 + 3
			middle = (figureH - v.h) / 2
		} else {
			reach += 2
		}
	case kindEllipse:
		span := v.w / 2
		if upright {
			span = v.h / 2
		}
		q := along / span
		reach *= math.Sqrt(max(1-float64(q*q), 0))
	}
	if upright {
		return c.X + float64(sign*reach), c.Y + middle + along
	}
	return c.X + along, c.Y + float64(sign*reach)
}

// sideLength is how much of a side edges may spread over, as Pele's sideLength gives it.
func sideLength(v *view, side int) float64 {
	upright := side == 1 || side == 3
	if v.kind == kindActor {
		if upright {
			return float64(figureH) * 0.5
		}
		return float64(figureW) * 0.6
	}
	length := v.w
	if upright {
		length = v.h
	}
	if v.kind == kindEllipse {
		return length * 0.6
	}
	return length - 16
}

// attachEnds spaces the edges that end on one side of a node along the side, in the order
// they fan out, and puts each on the node's outline, as Pele's attachEnds does: the layout
// brings them all to the middle of the side.
func attachEnds(cedges []layout.CEdge, cnodes []layout.CNode, views []view) {
	type end struct {
		route  []float64
		at     int
		toward float64
	}
	// The ends on each side, and the sides in the order they were first met, the order
	// Pele's Map gives them in.
	sides := map[int][]end{}
	var keys []int
	add := func(index int, route []float64, at, next int) {
		c := &cnodes[index]
		if c.IsGroup {
			return
		}
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
		key := index*4 + side
		e := end{route, at, route[next+side&1]}
		if _, ok := sides[key]; !ok {
			keys = append(keys, key)
		}
		sides[key] = append(sides[key], e)
	}
	for i := range cedges {
		e := &cedges[i]
		last := len(e.Route) - 3
		if e.Src == e.Dst || last < 3 {
			continue
		}
		add(e.Src, e.Route, 0, 3)
		add(e.Dst, e.Route, last, last-3)
	}
	for _, key := range keys {
		ends := sides[key]
		index, side := key>>2, key&3
		v := &views[index]
		step := 0.0
		if len(ends) > 1 {
			step = min(portGap, sideLength(v, side)/float64(len(ends)-1))
		}
		slices.SortStableFunc(ends, func(a, b end) int {
			switch d := a.toward - b.toward; {
			case d < 0:
				return -1
			case d > 0:
				return 1
			}
			return 0
		})
		for i, e := range ends {
			along := float64((float64(i) - float64(len(ends)-1)/2) * step)
			x, y := anchor(v, &cnodes[index], side, along)
			e.route[e.at] = x
			e.route[e.at+1] = y
		}
	}
}

// markerTrim is how far an edge stops short of its end for a marker of the type, as Pele's
// trim says.
func markerTrim(t string) float64 {
	if t == "extension" {
		return extension
	}
	return scene.MarkerType(t).Trim()
}

// markerType is the scene's type of a marker of Pele's: "" for none, and the hollow triangle
// of a generalization under the diagram's own name.
func markerType(t string) scene.MarkerType {
	switch t {
	case "none", "":
		return ""
	case "extension":
		return "usecase.extension"
	}
	return scene.MarkerType(t)
}

// Layout lays out a parsed use case diagram as a scene, as Pele's renderUsecase draws it: as
// its direction says, and, when that runs across and does not fit o.MaxWidth, down instead,
// if that is narrower.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	if !common.RunsAcross(m.Direction) {
		return lay(m, config, o, false)
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return lay(m, config, o, down) })
}

// classNames is a class attribute's names, as Pele's classNames cleans them.
func classNames(names []string) []string {
	return strings.FieldsFunc(strings.Join(names, " "), func(r rune) bool {
		return !(r == '_' || r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	})
}

// figure is the parts of an actor's figure, drawn about fx, fy, as Pele's figure draws them:
// a head and limbs, a hollow body, a solid silhouette, or a box for an icon, with the slash
// of a business actor across its head.
func figure(v *view, fx, fy float64) (parts []scene.Part, picture *scene.Picture) {
	n := v.node
	head, radius := -21.0, 9.0
	pt := func(x, y float64) (float64, float64) { return fx + x, fy + y }
	switch n.ActorType {
	case "hollow":
		radius = 8
		body := new(scene.Path)
		xy := []float64{-20, -9, 20, -9, 20, 0, 5.5, 0, 20, 15.5, 12, 25.5, 0, 12, -12, 25.5, -20, 15.5, -5.5, 0, -20, 0}
		for i := 0; i < len(xy); i += 2 {
			xy[i], xy[i+1] = pt(xy[i], xy[i+1])
		}
		body.Polygon(xy...)
		parts = append(parts,
			scene.Part{Path: new(scene.Path).Ellipse(fx, fy-21, 8, 8), Paint: scene.Hollow},
			scene.Part{Path: body, Paint: scene.Hollow})
	case "awesome":
		body := new(scene.Path)
		body.MoveTo(pt(-20, 25))
		cubic := func(a, b, c, d, e, f float64) {
			x1, y1 := pt(a, b)
			x2, y2 := pt(c, d)
			x3, y3 := pt(e, f)
			body.CubeTo(x1, y1, x2, y2, x3, y3)
		}
		cubic(-20, 9, -12, -3, 0, -3)
		cubic(12, -3, 20, 9, 20, 25)
		cubic(20, 28, 18, 30, 15, 30)
		body.LineTo(pt(-15, 30))
		cubic(-18, 30, -20, 28, -20, 25)
		body.Close()
		// Pele draws the head as two arcs of one path; it is the same circle.
		parts = append(parts,
			scene.Part{Path: new(scene.Path).Ellipse(fx, fy-19, 11, 11), Paint: scene.FillOnly},
			scene.Part{Path: body, Paint: scene.FillOnly})
	case "icon":
		head, radius = -14, 12
		parts = append(parts, scene.Part{Box: scene.Rect{X: fx - 22, Y: fy - 26, W: 44, H: 44}, Rounded: true})
		picture = &scene.Picture{Icon: n.Icon, Box: scene.Rect{X: fx - 14, Y: fy - 18, W: 28, H: 28}}
	default:
		limbs := new(scene.Path)
		for _, l := range [][4]float64{{0, -12, 0, 10}, {-14, -3, 14, -3}, {0, 10, -12, 30}, {0, 10, 12, 30}} {
			limbs.MoveTo(pt(l[0], l[1]))
			limbs.LineTo(pt(l[2], l[3]))
		}
		parts = append(parts,
			scene.Part{Path: new(scene.Path).Ellipse(fx, fy-21, 9, 9)},
			scene.Part{Path: limbs, Paint: scene.StrokeOnly})
	}
	if n.Business {
		// The slash of a business actor: a chord across the lower right of the head.
		slash := new(scene.Path)
		slash.MoveTo(pt(float64(0.12*radius), head+float64(0.99*radius)))
		slash.LineTo(pt(float64(0.92*radius), head-float64(0.39*radius)))
		parts = append(parts, scene.Part{Path: slash, Paint: scene.StrokeOnly})
	}
	return parts, picture
}

// titleX is where a boundary's title sits along its top, as Pele's titleX places it: in the
// middle, unless edges enter through the top there, and then in the stretch between them
// that is nearest the middle, or the widest if none has room.
func titleX(c *layout.CNode, width float64, crossings draw.Crossings) float64 {
	xs, ok := crossings[jsmath.Round(c.Y-c.H/2)]
	if width == 0 || !ok {
		return c.X
	}
	left := c.X - c.W/2 + 6
	right := c.X + c.W/2 - 6
	var cuts []float64
	for _, x := range xs {
		if x > left && x < right {
			cuts = append(cuts, x)
		}
	}
	slices.SortStableFunc(cuts, func(a, b float64) int {
		switch {
		case a < b:
			return -1
		case a > b:
			return 1
		}
		return 0
	})
	cuts = append(cuts, right)
	need := width/2 + 6
	best, bestOff, widest, from := c.X, math.Inf(1), 0.0, left
	for _, cut := range cuts {
		if cut-from >= 2*need {
			x := min(max(c.X, from+need), cut-need)
			if math.Abs(x-c.X) < bestOff {
				bestOff = math.Abs(x - c.X)
				best = x
			}
		} else if math.IsInf(bestOff, 1) && cut-from > widest {
			widest = cut - from
			best = (from + cut) / 2
		}
		from = cut
	}
	return best
}

// leftLabel is the text of a label whose lines start at left, about cy, inside a node whose
// middle is (x, y), as Pele's leftLabel draws it: the label centered on the middle of its line
// when it has one, and otherwise one label for each line, centered on the middle of that line.
// left and cy are about the node's middle, as Pele works them out; each point is moved to the
// node last, as Pele's SVG moves it, so that the scene holds what Pele draws to the bit. it is
// the item each label is drawn as.
func leftLabel(l *text.Label, x, y, left, cy float64, it scene.Item) []scene.Item {
	if len(l.Lines) < 2 {
		it.Label = l
		it.At = scene.Point{X: x + (left + l.Width/2), Y: y + cy}
		return []scene.Item{it}
	}
	items := make([]scene.Item, len(l.Lines))
	for i := range l.Lines {
		// A line on its own, as wide as it is.
		line := *l
		line.Lines, line.Widths, line.Width, line.Height = l.Lines[i:i+1], l.Widths[i:i+1], l.Widths[i], l.LineHeight
		it.Label = &line
		it.At = scene.Point{
			X: x + (left + l.Widths[i]/2),
			Y: y + (cy - l.Height/2 + float64((float64(i)+0.5)*l.LineHeight)),
		}
		items[i] = it
	}
	return items
}

// lay lays the diagram out once, as Pele's draw does. turned lays out a diagram that runs
// across as one that runs down.
func lay(m *Model, config diagram.Config, o *layout.Options, turned bool) *scene.Scene {
	graph := BuildGraph(m)
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	tiny := jsmath.Round(size * 0.8125)
	measure := o.Measure()
	pad := o.Pad()
	rootDir := layout.Direction(m.Direction)
	if turned {
		rootDir = layout.TB
	}
	sideways := rootDir == layout.LR || rootDir == layout.RL
	labelOf := func(raw string, markdown bool, size, wrap float64, face text.Face) text.Label {
		if raw != "" && !markdown {
			raw = literal(raw)
		}
		return text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: wrap, Face: face, Markdown: markdown, Icons: o.Icons})
	}

	// The classes each node's author gave it.
	classes := map[string][]string{}
	for _, a := range m.Actors {
		classes[a.ID] = a.Classes
	}
	for _, u := range m.UseCases {
		classes[u.ID] = u.Classes
	}
	for _, j := range m.JSONNodes {
		classes[j.ID] = j.Classes
	}
	boundaryClasses := map[string][]string{}
	for _, b := range m.SystemBoundaries {
		boundaryClasses[b.ID] = b.Classes
	}
	relationClasses := map[*GraphEdge][]string{}
	for i, r := range m.Relationships {
		relationClasses[graph.Edges[i]] = r.Classes
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
		var base text.Face
		if author.Bold {
			base |= text.Bold
		}
		if author.Italic {
			base |= text.Italic
		}
		fontSize := size
		if author.FontSize > 0 {
			fontSize = author.FontSize
		}
		markdown := node.LabelType == LabelMarkdown
		var stereotype text.Label
		if node.Stereotype != "" {
			stereotype = labelOf("«"+node.Stereotype+"»", false, tiny, 200, base)
		}
		v := view{node: node, kind: kindBox, author: author, label: stereotype, stereotype: stereotype}
		var c layout.CNode
		switch {
		case node.IsGroup:
			v.kind = kindBoundary
			v.label = labelOf(node.Label, markdown, fontSize, 4000, base)
			titleH := v.label.Height
			c = layout.NewCNode(0, 0, -1)
			c.IsGroup = true
			c.PadX = groupPad
			c.PadBottom = groupPad
			switch {
			case node.BoundaryType == "package":
				c.PadTop = titleH + 2*tabPad + 16
			case titleH > 0:
				c.PadTop = titleH + 16
			default:
				c.PadTop = groupPad
			}
			c.MinW = v.label.Width + 2*groupPad
		case node.Shape == "usecaseJsonTable":
			v.kind = kindJSON
			v.label = labelOf(node.Label, false, fontSize, 4000, base|text.Bold)
			valueW := 0.0
			for _, row := range node.JSONRows {
				key := labelOf(row.Key, false, small, 4000, base)
				value := labelOf(row.Value, false, small, 260, base)
				v.rows = append(v.rows, [2]text.Label{key, value})
				v.keyW = max(v.keyW, key.Width)
				valueW = max(valueW, value.Width)
				v.h += max(value.Height, jsmath.Round(small*1.5)) + rowPad
			}
			// The root of an empty object has a value and no key, and then there is no key
			// column.
			if v.keyW > 0 {
				v.keyW += 2 * cellPad
			}
			v.headH = max(v.label.Height, jsmath.Round(fontSize*1.5)) + rowPad
			rowsW := 0.0
			if len(v.rows) > 0 {
				rowsW = v.keyW + valueW + 2*cellPad
			}
			v.w = max(v.label.Width+2*cellPad, rowsW, 72)
			v.h += v.headH
			c = layout.NewCNode(v.w, v.h, -1)
		case node.ActorType != "":
			v.kind = kindActor
			v.label = labelOf(node.Label, markdown, fontSize, 140, base)
			v.w = max(figureW, v.label.Width, stereotype.Width)
			v.h = figureH + figureGap + stereotype.Height + v.label.Height
			c = layout.NewCNode(v.w, v.h, -1)
		default:
			labelSize := fontSize
			if node.Shape == "note" {
				labelSize = small
			}
			v.label = labelOf(node.Label, markdown, labelSize, 180, base)
			tw := max(v.label.Width, stereotype.Width)
			th := v.label.Height + stereotype.Height
			switch node.Shape {
			case "usecaseEllipse", "usecaseBusiness":
				v.kind = kindEllipse
				// An oval needs more room around its text than a box, and a business oval
				// room for its slash.
				extra := 4.0
				if node.Business {
					extra = 16
				}
				v.w = max(tw+2*node.Padding+extra, 88)
				v.h = max(th+28, 48)
			case "note":
				v.w = tw + 2*node.Padding + 6
				v.h = th + 18
			default:
				v.w = max(tw+2*node.Padding+12, 80)
				v.h = max(th+24, 44)
			}
			c = layout.NewCNode(v.w, v.h, -1)
		}
		c.Seq = float64(len(views))
		views = append(views, v)
		cnodes = append(cnodes, c)
	}
	for i, node := range graph.Nodes {
		cnodes[i].Parent = -1
		if node.ParentID != "" {
			if p, ok := index[node.ParentID]; ok {
				cnodes[i].Parent = p
			}
		}
	}

	var cedges []layout.CEdge
	var edgeLabels []text.Label
	var drawn []*GraphEdge
	for _, edge := range graph.Edges {
		src, ok1 := index[edge.Start]
		dst, ok2 := index[edge.End]
		if !ok1 || !ok2 {
			continue
		}
		semantic := edge.RelationshipType == "include" || edge.RelationshipType == "extend"
		raw := edge.Label
		if semantic {
			raw = "«" + edge.Label + "»"
		}
		label := labelOf(raw, edge.LabelType == LabelMarkdown, small, 200, 0)
		if src == dst {
			// A self-relationship loops out across the flow, and the node's box grows to hold
			// it.
			views[src].loops++
			if sideways {
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
		ce := layout.CEdge{Src: src, Dst: dst, Minlen: float64(edge.MinLen)}
		if label.Width > 0 {
			ce.LabelW = label.Width + 12
		}
		if label.Height > 0 {
			ce.LabelH = label.Height + 4
		}
		cedges = append(cedges, ce)
	}

	result := layout.Compound(cnodes, cedges, rootDir, layout.LayeredOptions{
		NodeSep: numberOption(config, "nodeSpacing", nodeSep),
		EdgeSep: edgeSep,
		RankSep: numberOption(config, "rankSpacing", rankSep),
		PortSep: 20,
	})

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: 4000, Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	inner := max(result.Width, title.Width)
	ox := pad + (inner-result.Width)/2
	oy := pad + titleHeight
	layout.Shift(cnodes, cedges, ox, oy)
	sc := &scene.Scene{
		Type:     "usecase",
		Width:    math.Ceil(inner + 2*pad),
		Height:   math.Ceil(result.Height + titleHeight + 2*pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}

	// Where edges cross each boundary's top border, so that a title can keep out of their
	// way.
	crossings := draw.Crossings{}
	for _, e := range cedges {
		draw.MarkCrossings(crossings, e.Route)
	}
	attachEnds(cedges, cnodes, views)

	var titles, clusters, edges, labels, nodes []scene.Item
	if !title.Empty() {
		titles = append(titles, scene.Item{
			Kind: scene.Text, Role: "usecase.title", Label: &title,
			At: scene.Point{X: sc.Width / 2, Y: pad + title.Height/2},
		})
	}
	for i := range views {
		v := &views[i]
		c := &cnodes[i]
		n := v.node
		x, y := c.X, c.Y
		if v.kind == kindBoundary {
			left, top := x-c.W/2, y-c.H/2
			it := scene.Item{
				Kind: scene.Group, Role: "usecase.system", Shape: "rect", ID: n.ID, Class: classNames(boundaryClasses[n.ID]),
				Box:    scene.Rect{X: left, Y: top, W: c.W, H: c.H},
				Style:  style.Style{Fill: style.SurfaceAlt.Alpha(0.5), Stroke: style.Border, Text: style.Muted},
				Author: v.author.Style,
			}
			if n.BoundaryType == "package" {
				it.Shape = "package"
				tabW := min(v.label.Width+2*tabPad+8, c.W)
				tabH := v.label.Height + 2*tabPad
				it.Parts = []scene.Part{
					{Box: scene.Rect{X: left, Y: top, W: tabW, H: tabH}},
					{Box: scene.Rect{X: left, Y: top + tabH, W: c.W, H: c.H - tabH}},
				}
				it.At = scene.Point{X: left + tabW/2, Y: top + tabH/2}
			} else {
				it.Parts = []scene.Part{{Box: it.Box, Rounded: true}}
				it.At = scene.Point{X: titleX(c, v.label.Width, crossings), Y: top + 8 + v.label.Height/2}
			}
			if !v.label.Empty() {
				it.Label = &v.label
			}
			clusters = append(clusters, it)
			continue
		}

		w, h := v.w, v.h
		it := scene.Item{
			Kind: scene.Node, ID: n.ID, Class: classNames(classes[n.ID]),
			Box:    scene.Rect{X: x - w/2, Y: y - h/2, W: w, H: h},
			Style:  style.Style{Fill: style.Surface, Stroke: style.Border},
			Author: v.author.Style,
		}
		var stereotype *scene.Item
		var cells []scene.Item
		switch v.kind {
		case kindActor:
			it.Role = "usecase.actor"
			it.Shape = "actor"
			if n.ActorType != "normal" {
				it.Shape += scene.Shape("-" + n.ActorType)
			}
			if n.Business {
				it.Shape += "-business"
			}
			if n.ActorType == "awesome" {
				it.Style.Fill = style.Border
			}
			top := -h / 2
			it.Parts, it.Picture = figure(v, x, y+(top+figureH/2))
			if n.ActorType == "hollow" {
				// Pele turns the corners of a hollow body round; its head and a business
				// actor's slash, a circle and a single stroke, have no corners to turn.
				it.Style.Join = style.JoinRound
			}
			if !v.stereotype.Empty() {
				stereotype = &scene.Item{
					Kind: scene.Text, Role: "usecase.stereotype", ID: n.ID, Label: &v.stereotype,
					At:    scene.Point{X: x, Y: y + (top + figureH + figureGap + v.stereotype.Height/2)},
					Style: style.Style{Text: style.Muted},
				}
			}
			it.At = scene.Point{X: x, Y: y + (h/2 - v.label.Height/2)}
		case kindJSON:
			it.Role = "usecase.json"
			it.Shape = "table"
			x0 := -w / 2
			ry := -h/2 + v.headH
			lines := new(scene.Path)
			lines.MoveTo(x+x0, y+ry).LineTo(x+(x0+w), y+ry)
			if v.keyW > 0 {
				lines.MoveTo(x+(x0+v.keyW), y+ry).LineTo(x+(x0+v.keyW), y+h/2)
			}
			it.Parts = []scene.Part{{Box: it.Box, Rounded: true}, {Path: lines, Paint: scene.StrokeOnly}}
			it.At = scene.Point{X: x, Y: y + (-h/2 + v.headH/2)}
			for k := range v.rows {
				key, value := &v.rows[k][0], &v.rows[k][1]
				rowH := max(value.Height, jsmath.Round(small*1.5)) + rowPad
				cells = append(cells, scene.Item{
					Kind: scene.Text, Role: "usecase.json-key", ID: n.ID, Label: key,
					At:    scene.Point{X: x + (x0 + cellPad + key.Width/2), Y: y + (ry + rowH/2)},
					Style: style.Style{Text: style.Muted},
				})
				cells = append(cells, leftLabel(value, x, y, x0+v.keyW+cellPad, ry+rowH/2, scene.Item{
					Kind: scene.Text, Role: "usecase.json-value", ID: n.ID,
					Author: style.Style{Text: v.author.Style.Text},
				})...)
				ry += rowH
			}
		default:
			it.Role = "usecase.usecase"
			switch {
			case v.kind == kindEllipse:
				it.Shape = "ellipse"
				it.Parts = []scene.Part{{Path: new(scene.Path).Ellipse(x, y, w/2, h/2)}}
				if n.Business {
					it.Shape = "ellipse-business"
					// The slash of a business use case, cutting off the right end of the oval
					// beside the text.
					from, to := w/2-15, w/2-5
					arc := func(at float64) float64 {
						q := at / (w / 2)
						return (h / 2) * math.Sqrt(1-float64(q*q))
					}
					slash := new(scene.Path).MoveTo(x+from, y+arc(from)).LineTo(x+to, y+(-arc(to)))
					it.Parts = append(it.Parts, scene.Part{Path: slash, Paint: scene.StrokeOnly})
				}
			case n.Shape == "note":
				it.Role = "usecase.note"
				it.Shape = "note"
				it.Style.Fill = style.SurfaceAlt
				// Pele turns the corners of the note and of its fold round.
				it.Style.Join = style.JoinRound
				r, b := w/2, h/2
				outline := new(scene.Path).Polygon(x-r, y-b, x+(r-fold), y-b, x+r, y+(fold-b), x+r, y+b, x-r, y+b)
				corner := new(scene.Path).MoveTo(x+(r-fold), y-b).LineTo(x+(r-fold), y+(fold-b)).LineTo(x+r, y+(fold-b))
				it.Parts = []scene.Part{{Path: outline}, {Path: corner, Paint: scene.StrokeOnly}}
			default:
				it.Shape = "rect"
				it.Parts = []scene.Part{{Box: it.Box, Rounded: true}}
			}
			if !v.stereotype.Empty() {
				stereotype = &scene.Item{
					Kind: scene.Text, Role: "usecase.stereotype", ID: n.ID, Label: &v.stereotype,
					At:    scene.Point{X: x, Y: y + (-v.label.Height / 2)},
					Style: style.Style{Text: style.Muted},
				}
			}
			it.At = scene.Point{X: x, Y: y + v.stereotype.Height/2}
		}
		if !v.label.Empty() {
			it.Label = &v.label
		}
		nodes = append(nodes, it)
		if stereotype != nil {
			nodes = append(nodes, *stereotype)
		}
		nodes = append(nodes, cells...)
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
		start, end := edge.ArrowTypeStart, edge.ArrowTypeEnd
		var path draw.EdgePath
		var route []float64
		if ce.Src == ce.Dst {
			c := &cnodes[ce.Src]
			v := &views[ce.Src]
			k := loopSeen[ce.Src]
			loopSeen[ce.Src]++
			side := 1
			if sideways {
				side = 2
			}
			spread := min(sideLength(v, side)/2, 8+float64(k*6))
			reach := float64(loop + k*8)
			ax, ay := anchor(v, c, side, -spread)
			bx, by := anchor(v, c, side, spread)
			l := jsmath.Hypot(reach, spread)
			if l == 0 {
				l = 1
			}
			ux, uy := reach/l, spread/l
			if sideways {
				ux, uy = spread/l, reach/l
			}
			st, et := markerTrim(start), markerTrim(end)
			var sx, sy float64
			if sideways {
				sx, sy = ax-float64(ux*st), ay+float64(uy*st)
			} else {
				sx, sy = ax+float64(ux*st), ay-float64(uy*st)
			}
			ex, ey := bx+float64(ux*et), by+float64(uy*et)
			// Both ends leave the node leaning away from each other, and the curve joins them
			// outside it.
			out := func(x, y, sign float64) (float64, float64) {
				if sideways {
					return x + float64(sign*spread), y + reach
				}
				return x + reach, y + float64(sign*spread)
			}
			c1x, c1y := out(ax, ay, -1)
			c2x, c2y := out(bx, by, 1)
			p := new(scene.Path).MoveTo(sx, sy).CubeTo(c1x, c1y, c2x, c2y, ex, ey)
			path = draw.EdgePath{Path: p, SX: ax, SY: ay, SDX: -ux, SDY: uy, EX: bx, EY: by, EDX: -ux, EDY: -uy}
			if sideways {
				path.SDX, path.SDY = ux, -uy
				ce.LabelX = c.X
				ce.LabelY = max(ay, by) + float64(reach*0.75) + 6 + label.Height/2
			} else {
				ce.LabelX = max(ax, bx) + float64(reach*0.75) + 8 + label.Width/2
				ce.LabelY = c.Y
			}
		} else {
			if len(ce.Route) < 6 {
				continue
			}
			route = ce.Route
			st, et := markerTrim(start), markerTrim(end)
			path = draw.RoutePath(route, "", st, et, draw.OwnReach(st), draw.OwnReach(et))
		}
		// A line ends round, as Pele draws it in the group of edges, which sets round caps.
		own := style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound}
		switch {
		case edge.Internal:
			own.Dash = style.Dashes(2, 3)
		case edge.Pattern == "dotted":
			own.Dash = style.Dashes(5, 4)
		case edge.Animate:
			own.Dash = style.Dashes(9, 5)
		}
		role := scene.Role("usecase.relation")
		if edge.Internal {
			role = "usecase.note-link"
		}
		it := scene.Item{
			Kind: scene.Edge, Role: role, Shape: scene.Shape(edge.RelationshipType), ID: edge.ID,
			Class:  classNames(relationClasses[edge]),
			Box:    path.Path.Bounds(),
			Parts:  []scene.Part{{Path: path.Path, Paint: scene.StrokeOnly}},
			Ends:   path.Ends(markerType(start), markerType(end)),
			Style:  own,
			Author: author.Style,
		}
		for k := 0; k+1 < len(route); k += 3 {
			it.Route = append(it.Route, scene.Point{X: route[k], Y: route[k+1]})
		}
		edges = append(edges, it)
		if label.Width > 0 {
			lx, ly := ce.LabelX, ce.LabelY
			own := style.Style{Fill: style.Bg, Stroke: style.None}
			if edge.RelationshipType == "include" || edge.RelationshipType == "extend" {
				own.Text = style.Muted
			}
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: "usecase.label", ID: edge.ID,
				Box:    scene.Rect{X: lx - (label.Width+8)/2, Y: ly - label.Height/2, W: label.Width + 8, H: label.Height},
				Parts:  []scene.Part{draw.LabelPatch(label, lx, ly)},
				Label:  label,
				At:     scene.Point{X: lx, Y: ly},
				Style:  own,
				Author: style.Style{Text: author.Style.Text, Opacity: author.Style.Opacity},
			})
		}
	}

	sc.Items = make([]scene.Item, 0, len(titles)+len(clusters)+len(edges)+len(labels)+len(nodes))
	for _, layer := range [][]scene.Item{titles, clusters, edges, labels, nodes} {
		sc.Items = append(sc.Items, layer...)
	}
	return sc
}
