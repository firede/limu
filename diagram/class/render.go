package class

import (
	"math"
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/draw"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The spacing of a class diagram, as Pele's render.ts has it.
const (
	nodeSep      = 40
	edgeSep      = 16
	rankSep      = 56
	groupPad     = 20
	loop         = 26
	padX         = 12 // between a class's side and its text
	titlePad     = 8  // above and below a class's name, and about a note's text
	sectionPad   = 6  // above and below the members of a compartment
	emptySection = 10 // the height of a compartment with no members
	minWidth     = 64 // of a class
	fold         = 8  // of a note's corner
	padding      = common.Padding
)

// Layout lays out a parsed class diagram as a scene, as Pele's renderClass draws it: across
// and loosely, and, when that does not fit o.MaxWidth, tightly, or down instead of across,
// whichever is narrower.
func Layout(db *DB, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	fitted := func(down bool) *scene.Scene {
		return common.Tighten(o, func(tight bool) *scene.Scene { return lay(db, config, o, down, tight) })
	}
	if common.RunsAcross(db.Direction) {
		return common.TurnToFit(o, fitted)
	}
	return fitted(false)
}

// numberOption is a number of the diagram's config, if it is one above 0, at most 2000.
func numberOption(section diagram.Config, key string, fallback float64) float64 {
	if v, ok := section[key].(float64); ok && v > 0 {
		return min(v, 2000)
	}
	return fallback
}

// A line is a member of a class as it is drawn: Pele's Line.
type line struct {
	text       string
	width      float64
	classifier string
}

// A view is a node as it is drawn: Pele's View.
type view struct {
	node        *GraphNode
	author      style.Author
	label       text.Label
	w, h        float64
	loops       int
	annotations []line
	attributes  []line
	methods     []line
	titleH      float64
	attributesH float64
	fontSize    float64
}

// An end is where an edge meets the side of a class: Pele's End.
type end struct {
	route  []float64
	at     int
	toward float64
	order  int
	room   float64
	side   int
	// after says whether the text of this end goes on the right of, or below, its line.
	after bool
}

// A card is the text at an end of a relation: its multiplicity.
type card struct {
	text  string
	width float64
	x, y  float64
}

// lay lays the diagram out once, as Pele's draw does. turned lays out a diagram that runs
// across as one that runs down; tight gives up the room that keeps a namespace's name clear
// of an edge coming in beside it.
func lay(db *DB, config diagram.Config, o *layout.Options, turned, tight bool) *scene.Scene {
	conf, _ := config["class"].(diagram.Config)
	graph := BuildGraph(db, Hierarchical(config))
	hideEmpty, _ := conf["hideEmptyMembersBox"].(bool)
	size := o.Size()
	small := jsmath.Round(float64(size * 0.875))
	tiny := jsmath.Round(float64(size * 0.8))
	tinyHeight := jsmath.Round(float64(tiny * 1.4))
	measure := o.Measure()
	rootDir := layout.Direction(db.Direction)
	if turned {
		rootDir = layout.TB
	}
	labelOf := func(raw string, markdown bool, size, max float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: face, Markdown: markdown, Icons: o.Icons})
	}

	nodes := graph.Nodes
	children := make([]int, len(nodes))
	for _, n := range nodes {
		if n.Parent >= 0 {
			children[n.Parent]++
		}
	}

	// Namespaces that an edge may cross into are made wide enough for the title to sit
	// beside it.
	crossed := make([]bool, len(nodes))
	for _, e := range graph.Edges {
		if e.Source < 0 || e.Target < 0 {
			continue
		}
		a, b := nodes[e.Source].Parent, nodes[e.Target].Parent
		if a == b {
			continue
		}
		for g := a; g >= 0 && !crossed[g]; g = nodes[g].Parent {
			crossed[g] = true
		}
		for g := b; g >= 0 && !crossed[g]; g = nodes[g].Parent {
			crossed[g] = true
		}
	}

	memberLine := func(m *Member, size float64) line {
		t := chars.Trim(text.DecodeEntities(m.DisplayText()))
		var face text.Face
		if m.Classifier == "*" {
			face = text.Italic
		}
		return line{text: t, width: measure.Width(t, size, face), classifier: m.Classifier}
	}
	classView := func(node *GraphNode, cls *Class) view {
		author := classStyle(db, cls)
		fontSize := size
		if author.FontSize > 0 {
			fontSize = author.FontSize
		}
		memberSize := jsmath.Round(float64(fontSize * 0.875))
		lineHeight := jsmath.Round(float64(memberSize * 1.5))
		// The generic type is escaped so that `<T>` is not read as a tag.
		generic := ""
		if cls.Type != "" {
			t := common.ParseGenericTypes(cls.Type)
			generic = "&lt;" + strings.ReplaceAll(strings.ReplaceAll(t, "<", "&lt;"), ">", "&gt;") + "&gt;"
		}
		face := text.Bold
		if author.Italic {
			face |= text.Italic
		}
		label := labelOf(cls.Label+generic, false, fontSize, 4000, face)
		var annotations []line
		for _, a := range cls.Annotations {
			t := "«" + chars.Trim(text.DecodeEntities(a)) + "»"
			annotations = append(annotations, line{text: t, width: measure.Width(t, tiny, 0)})
		}
		var attributes, methods []line
		for _, m := range cls.Members {
			attributes = append(attributes, memberLine(m, memberSize))
		}
		for _, m := range cls.Methods {
			methods = append(methods, memberLine(m, memberSize))
		}
		inner := label.Width
		for _, a := range annotations {
			inner = max(inner, a.width)
		}
		for _, l := range attributes {
			inner = max(inner, l.width)
		}
		for _, l := range methods {
			inner = max(inner, l.width)
		}
		sections := !(hideEmpty && len(attributes) == 0 && len(methods) == 0)
		section := func(count int) float64 {
			switch {
			case !sections:
				return 0
			case count > 0:
				return 2*sectionPad + float64(count)*lineHeight
			}
			return emptySection
		}
		titleH := 2*titlePad + float64(len(annotations))*tinyHeight + max(label.Height, jsmath.Round(float64(fontSize*1.5)))
		attributesH := section(len(attributes))
		return view{
			node: node, author: author, label: label,
			w: max(inner+2*padX, minWidth), h: titleH + attributesH + section(len(methods)),
			annotations: annotations, attributes: attributes, methods: methods,
			titleH: titleH, attributesH: attributesH, fontSize: fontSize,
		}
	}

	views := make([]view, 0, len(nodes))
	cnodes := make([]layout.CNode, 0, len(nodes))
	for i, node := range nodes {
		var v view
		var c layout.CNode
		switch {
		case node.IsGroup:
			label := labelOf(node.Label, false, size, 4000, 0)
			v = view{node: node, label: label, fontSize: size}
			if children[i] > 0 {
				c = layout.NewCNode(0, 0, -1)
				c.IsGroup = true
				c.PadX = groupPad
				c.PadTop = groupPad
				if label.Height > 0 {
					c.PadTop = label.Height + 16
				}
				c.PadBottom = groupPad
				extra := 0.0
				if crossed[i] && !tight {
					extra = min(label.Width+24, 240)
				}
				c.MinW = label.Width + 2*groupPad - 12 + extra
			} else {
				v.w = max(label.Width+2*groupPad, 80)
				v.h = label.Height + 2*groupPad
				c = layout.NewCNode(v.w, v.h, -1)
			}
		case node.Class != nil:
			v = classView(node, node.Class)
			c = layout.NewCNode(v.w, v.h, -1)
		case node.Shape == "note":
			label := labelOf(node.Label, true, small, 260, 0)
			v = view{node: node, label: label, w: max(label.Width+2*padX, 40), h: label.Height + 2*titlePad, fontSize: small}
			c = layout.NewCNode(v.w, v.h, -1)
		default:
			label := labelOf(node.Label, false, small, 200, 0)
			v = view{node: node, label: label, w: max(label.Width, 12) + 4, h: max(label.Height, 12), fontSize: small}
			c = layout.NewCNode(v.w, v.h, -1)
		}
		c.Parent = node.Parent
		c.Seq = float64(i)
		views = append(views, v)
		cnodes = append(cnodes, c)
	}

	var cedges []layout.CEdge
	var drawn []*GraphEdge
	var edgeLabels []text.Label
	var startRoom, endRoom []float64
	// Edges leave one side of a class and arrive at the other, so a class with many of them
	// is widened until they, and the text at their ends, fit side by side.
	crowd := make([]float64, len(views)*4)
	meet := func(item, slot int, room float64) {
		crowd[item*4+slot]++
		if room > crowd[item*4+slot+1] {
			crowd[item*4+slot+1] = room
		}
	}
	for _, edge := range graph.Edges {
		if edge.Source < 0 || edge.Target < 0 {
			continue
		}
		label := labelOf(edge.Label, true, small, 200, 0)
		startW, endW := 0.0, 0.0
		if edge.StartLabelRight != "" {
			startW = measure.Width(text.DecodeEntities(edge.StartLabelRight), tiny, 0)
		}
		if edge.EndLabelLeft != "" {
			endW = measure.Width(text.DecodeEntities(edge.EndLabelLeft), tiny, 0)
		}
		if edge.Source == edge.Target {
			views[edge.Source].loops++
			room := 0.0
			if label.Width > 0 {
				room = label.Width + 12
			}
			cnodes[edge.Source].W += 2 * (loop + room)
		} else {
			meet(edge.Source, 0, startW)
			meet(edge.Target, 2, endW)
		}
		startRoom = append(startRoom, startW)
		endRoom = append(endRoom, endW)
		drawn = append(drawn, edge)
		edgeLabels = append(edgeLabels, label)
		ce := layout.CEdge{Src: edge.Source, Dst: edge.Target, Minlen: 1}
		if label.Width > 0 {
			ce.LabelW = label.Width + 12
		}
		if label.Height > 0 {
			ce.LabelH = label.Height + 4
		}
		cedges = append(cedges, ce)
	}

	if rootDir == layout.TB || rootDir == layout.BT {
		for i := range views {
			if views[i].node.Class == nil {
				continue
			}
			need := 0.0
			for slot := 0; slot < 4; slot += 2 {
				count, room := crowd[i*4+slot], crowd[i*4+slot+1]
				if count > 1 {
					per := 14.0
					if room > 0 {
						per = room + 24
					}
					need = max(need, float64((count-1)*per)+24)
				}
			}
			need = min(need, 480)
			if need > views[i].w {
				cnodes[i].W += need - views[i].w
				views[i].w = need
			}
		}
	}

	lettered := false
	for i := 0; i < len(drawn) && !lettered; i++ {
		lettered = startRoom[i] > 0 || endRoom[i] > 0
	}
	rs := float64(rankSep)
	if lettered {
		rs += 16
	}
	result := layout.Compound(cnodes, cedges, rootDir, layout.LayeredOptions{
		NodeSep: numberOption(conf, "nodeSpacing", nodeSep),
		EdgeSep: edgeSep,
		RankSep: numberOption(conf, "rankSpacing", rs),
		PortSep: 20,
		Apart:   true,
	})
	node := func(i int) draw.Node {
		c := &cnodes[i]
		return draw.Node{X: c.X, Y: c.Y, W: c.W, H: c.H, Group: c.IsGroup}
	}

	// Every edge reaches the middle of a side. Where several share a side they are spread
	// along it, in the order of where they come from. An end with text runs straight for the
	// length of the text, which sits beside it on the side the line then bends toward, away
	// from the neighbors that bend the other way. Neighbors are set apart far enough for the
	// text between them.
	sides := map[int][]*end{}
	var sideKeys []int
	attach := func(ce *layout.CEdge, at, item int, room float64, order int) *end {
		route := ce.Route
		side := draw.EndSide(route, at, node(item))
		if side < 0 {
			return nil
		}
		slack := (cnodes[item].W - views[item].w) / 2
		if side == 1 {
			route[at] -= slack
		} else if side == 3 {
			route[at] += slack
		}
		across := side & 1
		from := at - 3
		if at == 0 {
			from = 3
		}
		toward := route[from+across]
		bend := toward - route[at+across]
		after := bend >= 0
		if across == 1 {
			after = bend > 0
		}
		e := &end{route: route, at: at, toward: toward, order: order, room: room, side: side, after: after}
		key := item*4 + side
		if _, ok := sides[key]; !ok {
			sideKeys = append(sideKeys, key)
		}
		sides[key] = append(sides[key], e)
		return e
	}
	starts := make([]*end, len(cedges))
	ends := make([]*end, len(cedges))
	for i := range cedges {
		ce := &cedges[i]
		if ce.Src != ce.Dst && len(ce.Route) >= 6 {
			starts[i] = attach(ce, 0, ce.Src, startRoom[i], i)
			ends[i] = attach(ce, len(ce.Route)-3, ce.Dst, endRoom[i], i)
		}
	}
	// onBorder reports whether a point lies on the border of one of the namespaces around a
	// class.
	onBorder := func(item int, x, y float64) bool {
		for g, hops := cnodes[item].Parent, 0; g >= 0 && hops < 10000; g, hops = cnodes[g].Parent, hops+1 {
			c := &cnodes[g]
			dx, dy := math.Abs(x-c.X), math.Abs(y-c.Y)
			if math.Abs(dy-c.H/2) < 0.5 && dx <= c.W/2+0.5 || math.Abs(dx-c.W/2) < 0.5 && dy <= c.H/2+0.5 {
				return true
			}
		}
		return false
	}
	var gaps []float64
	for _, key := range sideKeys {
		list := sides[key]
		if len(list) < 2 {
			continue
		}
		item, across := key>>2, key&1
		length := views[item].w
		if across == 1 {
			length = views[item].h
		}
		slices.SortStableFunc(list, func(a, b *end) int {
			if d := a.toward - b.toward; d != 0 {
				if d < 0 {
					return -1
				}
				return 1
			}
			return a.order - b.order
		})
		gaps = gaps[:0]
		total := 0.0
		for j := 0; j+1 < len(list); j++ {
			a, b := list[j], list[j+1]
			gap := 24.0
			if across == 1 {
				gap = 20
			}
			if a.room > 0 && a.after {
				if across == 1 {
					gap += tinyHeight + 6
				} else {
					gap += a.room
				}
			}
			if b.room > 0 && !b.after {
				if across == 1 {
					gap += tinyHeight + 6
				} else {
					gap += b.room
				}
			}
			gaps = append(gaps, gap)
			total += gap
		}
		scale := min(1, (length-24)/total)
		if scale <= 0 {
			continue
		}
		at := cnodes[item].X
		if across == 1 {
			at = cnodes[item].Y
		}
		at -= float64(total*scale) / 2
		for j, e := range list {
			route, endAt := e.route, e.at
			// An edge that crosses the border of the class's namespace close by runs straight
			// to it: the point where it crosses moves into line with its end.
			step := -3
			if endAt == 0 {
				step = 3
			}
			first := endAt + step
			border := -1
			for k := first; k >= 0 && k < len(route) && math.Abs(route[k+across]-route[first+across]) < 0.5; k += step {
				if onBorder(item, route[k], route[k+1]) {
					border = k
					break
				}
			}
			if border >= 0 && math.Abs(route[first+across]-at) <= 24 {
				for k := first; k != border+step; k += step {
					route[k+across] = at
				}
			}
			route[endAt+across] = at
			if j < len(gaps) {
				at += float64(gaps[j] * scale)
			}
		}
	}

	straighten := func(route []float64, e *end, room float64, first bool) []float64 {
		if e == nil || room <= 0 {
			return route
		}
		across := e.side & 1
		at := len(route) - 3
		from := at - 3
		if first {
			at, from = 0, 3
		}
		along := at + 1 - across
		reach := math.Abs(route[from+1-across] - route[along])
		length := tinyHeight + 6
		if across == 1 {
			length = room + 10
		}
		length = min(length, float64(reach*0.35))
		if length < 4 {
			return route
		}
		point := []float64{route[at], route[at+1], route[at+2]}
		if e.side == 0 || e.side == 3 {
			point[1-across] -= length
		} else {
			point[1-across] += length
		}
		if first {
			return slices.Insert(route, 3, point...)
		}
		return slices.Insert(route, at, point...)
	}
	for i := range cedges {
		cedges[i].Route = straighten(cedges[i].Route, starts[i], startRoom[i], true)
		cedges[i].Route = straighten(cedges[i].Route, ends[i], endRoom[i], false)
	}

	minX, minY := 0.0, 0.0
	maxX, maxY := result.Width, result.Height
	var cards []card
	addCard := func(raw string, width, x, y float64, side int, after bool) {
		if raw == "" {
			return
		}
		cx, cy := x, y
		if side&1 == 1 {
			if side == 3 {
				cx -= 5 + width/2
			} else {
				cx += 5 + width/2
			}
			if after {
				cy += 7 + tinyHeight/2
			} else {
				cy -= 7 + tinyHeight/2
			}
		} else {
			if after {
				cx += 9 + width/2
			} else {
				cx -= 9 + width/2
			}
			if side == 0 {
				cy -= 3 + tinyHeight/2
			} else {
				cy += 3 + tinyHeight/2
			}
		}
		cards = append(cards, card{text: text.DecodeEntities(raw), width: width, x: cx, y: cy})
		minX = min(minX, cx-width/2)
		maxX = max(maxX, cx+width/2)
		minY = min(minY, cy-tinyHeight/2)
		maxY = max(maxY, cy+tinyHeight/2)
	}
	loopSeen := map[int]int{}
	loopAt := make([]int, len(cedges))
	for i := range cedges {
		ce := &cedges[i]
		edge := drawn[i]
		route := ce.Route
		if ce.Src == ce.Dst {
			k := loopSeen[ce.Src]
			loopSeen[ce.Src] = k + 1
			loopAt[i] = k
			c := &cnodes[ce.Src]
			v := &views[ce.Src]
			x := c.X + v.w/2
			spread := min(v.h/2-4, float64(8+k*6))
			addCard(edge.StartLabelRight, startRoom[i], x, c.Y-spread, 1, false)
			addCard(edge.EndLabelLeft, endRoom[i], x, c.Y+spread, 1, true)
		} else if len(route) >= 6 {
			last := len(route) - 3
			fromSide, fromAfter := 2, true
			if s := starts[i]; s != nil {
				fromSide, fromAfter = s.side, s.after
			}
			toSide, toAfter := 0, true
			if e := ends[i]; e != nil {
				toSide, toAfter = e.side, e.after
			}
			addCard(edge.StartLabelRight, startRoom[i], route[0], route[1], fromSide, fromAfter)
			addCard(edge.EndLabelLeft, endRoom[i], route[last], route[last+1], toSide, toAfter)
		}
	}

	ox := padding - minX
	oy := padding - minY
	width := math.Ceil(maxX - minX + 2*padding)
	height := math.Ceil(maxY - minY + 2*padding)

	crossings := draw.Crossings{}
	if len(nodes) > 0 && nodes[0].IsGroup {
		for _, e := range cedges {
			draw.MarkCrossings(crossings, e.Route)
		}
	}

	var clusters, struck, nodeItems []scene.Item
	for i := range views {
		v := &views[i]
		c := &cnodes[i]
		n := v.node
		x, y := c.X+ox, c.Y+oy
		w, h := v.w, v.h
		if c.IsGroup {
			w, h = c.W, c.H
		}
		if n.IsGroup {
			titleX := c.X
			if c.IsGroup {
				titleX = draw.ClusterTitleX(node(i), v.label.Width, crossings, groupPad)
			}
			titleY := y - h/2 + 8 + v.label.Height/2
			box := scene.Rect{X: x - w/2, Y: y - h/2, W: w, H: h}
			cluster := scene.Item{
				Kind: scene.Group, Role: "class.namespace", ID: n.ID, Box: box,
				Parts: []scene.Part{{Box: box, Rounded: true}},
				Style: style.Style{Fill: style.SurfaceAlt.Alpha(0.5), Stroke: style.Border, Text: style.Muted},
			}
			if !v.label.Empty() {
				at := scene.Point{X: titleX + ox, Y: titleY}
				if c.IsGroup && draw.TitleCrossed(node(i), v.label.Width, titleX, crossings) {
					depth := draw.StruckDepth(func(k int) int { return nodes[k].Parent }, i)
					struck = append(struck, scene.Item{
						Kind: scene.Text, Role: "class.namespace-title", ID: n.ID, Label: &v.label, At: at,
						Style: style.Style{
							Text:      style.Muted,
							Halo:      style.SurfaceAlt.Alpha(draw.StruckTint(depth) / 100).Over(style.Bg),
							HaloWidth: style.N(draw.StruckHalo),
						},
					})
				} else {
					cluster.Label, cluster.At = &v.label, at
				}
			}
			clusters = append(clusters, cluster)
			continue
		}
		switch {
		case n.Class != nil:
			nodeItems = append(nodeItems, classItems(v, n.Class, x, y, tiny, tinyHeight, o)...)
		case n.Shape == "note":
			r, t := w/2, -h/2
			outline := new(scene.Path).MoveTo(x+(-r), y+t).LineTo(x+(r-fold), y+t).LineTo(x+r, y+(t+fold)).
				LineTo(x+r, y+(-t)).LineTo(x+(-r), y+(-t)).Close()
			corner := new(scene.Path).MoveTo(x+(r-fold), y+t).LineTo(x+(r-fold), y+(t+fold)).LineTo(x+r, y+(t+fold))
			nodeItems = append(nodeItems, scene.Item{
				Kind: scene.Node, Role: "class.note", Shape: "note", ID: n.ID,
				Box:   scene.Rect{X: x - w/2, Y: y - h/2, W: w, H: h},
				Parts: []scene.Part{{Path: outline}, {Path: corner, Paint: scene.StrokeOnly}},
				Label: &v.label, At: scene.Point{X: x, Y: y},
				// Pele turns the corners of both of a note's paths round.
				Style: style.Style{Fill: style.SurfaceAlt, Stroke: style.Border, Join: style.JoinRound},
			})
		default:
			nodeItems = append(nodeItems, scene.Item{
				Kind: scene.Node, Role: "class.interface", Shape: "interface", ID: n.ID,
				Box:   scene.Rect{X: x - w/2, Y: y - h/2, W: w, H: h},
				Label: &v.label, At: scene.Point{X: x, Y: y},
			})
		}
	}

	var edges, labels []scene.Item
	for i := range cedges {
		ce := &cedges[i]
		edge := drawn[i]
		label := &edgeLabels[i]
		start, finish := markerType(edge.ArrowTypeStart), markerType(edge.ArrowTypeEnd)
		startTrim, endTrim := start.Trim(), finish.Trim()
		var path draw.EdgePath
		var route []float64
		if ce.Src == ce.Dst {
			c := &cnodes[ce.Src]
			v := &views[ce.Src]
			k := loopAt[i]
			x := c.X + ox + v.w/2
			y := c.Y + oy
			spread := min(v.h/2-4, float64(8+k*6))
			reach := float64(loop + k*8)
			p := new(scene.Path).MoveTo(x+startTrim, y-spread).
				CubeTo(x+reach+startTrim, y-spread, x+reach+endTrim, y+spread, x+endTrim, y+spread)
			path = draw.EdgePath{Path: p, SX: x, SY: y - spread, SDX: -1, SDY: 0, EX: x, EY: y + spread, EDX: -1, EDY: 0}
			ce.LabelX = c.X + v.w/2 + reach + 4 + ce.LabelW/2
			ce.LabelY = c.Y
		} else {
			route = ce.Route
			for k := 0; k+1 < len(route); k += 3 {
				route[k] += ox
				route[k+1] += oy
			}
			path = draw.RoutePath(route, "", startTrim, endTrim, draw.OwnReach(startTrim), draw.OwnReach(endTrim))
		}
		// Pele draws the edges in a group that ends their lines round.
		own := style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound}
		role := scene.Role("class.relation")
		switch edge.Pattern {
		case "dashed":
			own.Dash = style.Dashes(6, 4)
		case "dotted":
			own.Dash = style.Dashes(2, 4)
			role = "class.note-edge"
		}
		it := scene.Item{
			Kind: scene.Edge, Role: role, ID: edge.ID,
			Box:   path.Path.Bounds(),
			Parts: []scene.Part{{Path: path.Path, Paint: scene.StrokeOnly}},
			Ends:  path.Ends(start, finish),
			Style: own,
		}
		for k := 0; k+1 < len(route); k += 3 {
			it.Route = append(it.Route, scene.Point{X: route[k], Y: route[k+1]})
		}
		edges = append(edges, it)
		if label.Width > 0 {
			x, y := ce.LabelX+ox, ce.LabelY+oy
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: "class.label", ID: edge.ID,
				Box:   scene.Rect{X: x - (label.Width+8)/2, Y: y - label.Height/2, W: label.Width + 8, H: label.Height},
				Parts: []scene.Part{draw.LabelPatch(label, x, y)},
				Label: label, At: scene.Point{X: x, Y: y},
				Style: style.Style{Fill: style.Bg, Stroke: style.None},
			})
		}
	}

	var cardItems []scene.Item
	for _, c := range cards {
		l := oneLine(c.text, c.width, tiny, tinyHeight, 0)
		cardItems = append(cardItems, scene.Item{
			Kind: scene.Text, Role: "class.cardinality", Label: &l, At: scene.Point{X: c.x + ox, Y: c.y + oy},
			Style: style.Style{Text: style.Muted},
		})
	}

	sc := &scene.Scene{Type: "class", Width: width, Height: height, FontSize: size, Title: db.AccTitle, Desc: db.AccDescr}
	sc.Items = make([]scene.Item, 0, 1+len(clusters)+len(edges)+len(struck)+len(labels)+len(cardItems)+len(nodeItems))
	for _, layer := range [][]scene.Item{clusters, edges, struck, labels, cardItems, nodeItems} {
		sc.Items = append(sc.Items, layer...)
	}
	common.WithTitle(sc, db.Title, "class.title", measure, size)
	return sc
}

// classStyle is what the author set for a class: the declarations of the classDefs of its
// classes, and then its own. Pele's classStyle keeps the last declaration of each property
// that has an effect and resolves those, which comes to the same.
func classStyle(db *DB, cls *Class) style.Author {
	var decls []string
	for _, name := range strings.Split(cls.CSSClasses, " ") {
		if def := db.styleClasses[name]; def != nil {
			for _, s := range def.Styles {
				decls = append(decls, strings.Split(s, ",")...)
			}
		}
	}
	return style.Declarations(append(decls, cls.Styles...))
}

// oneLine is a label of one line of plain text, as Pele writes the text of a member, an
// annotation or a multiplicity: as it is, with no markup read.
func oneLine(s string, width, size, lineHeight float64, face text.Face) text.Label {
	return text.Label{
		Lines:  [][]text.Span{{{Text: s, Face: face, Width: width}}},
		Widths: []float64{width}, Width: width, Height: lineHeight, Size: size, LineHeight: lineHeight,
	}
}

// classItems draws a class centered on x, y, as Pele's classSvg does: the box, with its
// dividers, and the class's name as its label, then each annotation and member as a text of
// its own. A static member is underlined, as Pele's text-decoration does, by a line under it.
func classItems(v *view, cls *Class, x, y, tiny, tinyHeight float64, o *layout.Options) []scene.Item {
	w, h := v.w, v.h
	left, top := -w/2, -h/2
	memberSize := jsmath.Round(float64(v.fontSize * 0.875))
	lineHeight := jsmath.Round(float64(memberSize * 1.5))
	box := scene.Rect{X: x + left, Y: y + top, W: w, H: h}
	parts := []scene.Part{{Box: box, Rounded: true}}
	first := top + v.titleH
	second := first + v.attributesH
	// The lines under the name and the attributes are an item of their own, as Pele gives
	// them only the stroke color the author set and none of the rest of the class's style.
	var dividers []scene.Item
	if h > v.titleH {
		divider := new(scene.Path).MoveTo(x+left, y+first).LineTo(x+(-left), y+first).
			MoveTo(x+left, y+second).LineTo(x+(-left), y+second)
		dividers = append(dividers, scene.Item{
			Kind: scene.Mark, Role: "class.divider", ID: cls.ID, Box: divider.Bounds(),
			Parts:  []scene.Part{{Path: divider, Paint: scene.StrokeOnly}},
			Style:  style.Style{Stroke: style.Border},
			Author: style.Style{Stroke: v.author.Style.Stroke},
		})
	}
	at := top + titlePad
	var annotations []scene.Item
	for _, a := range v.annotations {
		l := oneLine(a.text, a.width, tiny, tinyHeight, 0)
		annotations = append(annotations, scene.Item{
			Kind: scene.Text, Role: "class.annotation", ID: cls.ID, Label: &l,
			At:    scene.Point{X: x, Y: y + (at + tinyHeight/2)},
			Style: style.Style{Text: style.Muted},
		})
		at += tinyHeight
	}
	it := scene.Item{
		Kind: scene.Node, Role: "class.class", Shape: "classBox", ID: cls.ID, Class: common.ClassNames(common.TrimDefault(cls.CSSClasses)),
		Box: box, Parts: parts,
		Label: &v.label, At: scene.Point{X: x, Y: y + (at+first-titlePad)/2},
		Style:   style.Style{Fill: style.Surface, Stroke: style.Border},
		Author:  v.author.Style,
		Tooltip: text.DecodeEntities(cls.Tooltip),
	}
	if cls.Link != "" {
		if href, ok := url.LinkURL(cls.Link, !o.NoLinks, o.LinkSchemes); ok {
			it.Link = href
		}
	}
	items := append(append([]scene.Item{it}, dividers...), annotations...)
	lines := func(list []line, role scene.Role, from float64) {
		middle := from + sectionPad + lineHeight/2
		for _, ln := range list {
			var face text.Face
			if ln.classifier == "*" {
				face = text.Italic
			}
			l := oneLine(ln.text, ln.width, memberSize, lineHeight, face)
			m := scene.Item{
				Kind: scene.Text, Role: role, ID: cls.ID, Label: &l,
				At: scene.Point{X: x + (left + padX), Y: y + middle}, Align: scene.Start,
				Author: style.Style{Text: v.author.Style.Text},
			}
			if ln.classifier == "$" {
				// The underline of Arial, 0.106 of the size below the baseline.
				under := y + middle + float64(memberSize*0.35) + float64(memberSize*0.106)
				m.Parts = []scene.Part{{Path: new(scene.Path).MoveTo(m.At.X, under).LineTo(m.At.X+ln.width, under), Paint: scene.StrokeOnly}}
				// The underline is in the color of the text, the author's if they gave one.
				m.Style.Stroke = style.Fg
				m.Author.Stroke = v.author.Style.Text
			}
			items = append(items, m)
			middle += lineHeight
		}
	}
	lines(v.attributes, "class.attribute", first)
	lines(v.methods, "class.method", second)
	return items
}

// markerType is the scene's marker of an end of a relation.
func markerType(name string) scene.MarkerType {
	if name == "none" || name == "" {
		return ""
	}
	return scene.MarkerType("class." + name)
}
