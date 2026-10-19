package architecture

import (
	"math"
	"slices"
	"unicode/utf8"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/draw"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of Pele's render.ts: how far an edge leaves a node straight, the most its
// corners are rounded by, the room between a service's icon and its title, and the radius of
// a junction's dot.
const (
	stub     = 12
	corner   = 6
	labelGap = 4
	dot      = 3
)

// The scene's roles: "architecture.title" for the title above the diagram;
// "architecture.group" for each group's box, with its title, its id the group's;
// "architecture.edge" for each edge, its id its ends' ids joined by -, and
// "architecture.edge-label" for its label on a patch of the background;
// "architecture.service" for each service's square, with its title beneath it, and
// "architecture.icon-text" for the text it shows in place of an icon; "architecture.junction"
// for each junction's dot; and "architecture.icon" for an icon Pele draws itself, its id its
// service's or group's, its shape its name, right after the item it belongs to.
const (
	RoleTitle     scene.Role = "architecture.title"
	RoleGroup     scene.Role = "architecture.group"
	RoleEdge      scene.Role = "architecture.edge"
	RoleEdgeLabel scene.Role = "architecture.edge-label"
	RoleService   scene.Role = "architecture.service"
	RoleIconText  scene.Role = "architecture.icon-text"
	RoleJunction  scene.Role = "architecture.junction"
	RoleIcon      scene.Role = "architecture.icon"
)

// setting is a number of the diagram's config, held between lo and hi, or fallback, as Pele's
// setting reads it.
func setting(config diagram.Config, key string, fallback, lo, hi float64) float64 {
	section, _ := config["architecture"].(diagram.Config)
	if v, ok := section[key].(float64); ok && !math.IsNaN(v) && !math.IsInf(v, 0) {
		return min(hi, max(lo, v))
	}
	return fallback
}

// rank maps the grid lines in use to consecutive indexes, so empty rows and columns take no
// room, as Pele's rank does.
func rank(values []int) map[int]int {
	slices.Sort(values)
	out := map[int]int{}
	for _, v := range values {
		if _, ok := out[v]; !ok {
			out[v] = len(out)
		}
	}
	return out
}

// prefix16 is the end of the first n UTF-16 code units of s, in bytes, and whether the n
// units end inside a character, between the two of a surrogate pair. The end then takes the
// whole character.
func prefix16(s string, n int) (end int, split bool) {
	units := 0
	for i, r := range s {
		if units >= n {
			return i, false
		}
		units++
		if r > 0xffff {
			units++
			if units > n {
				return i + utf8.RuneLen(r), true
			}
		}
	}
	return len(s), false
}

// fit cuts a label down to a box, ending what is left with an ellipsis, as Pele's fit does.
// Where its search measures text that ends between the two halves of a surrogate pair, it
// measures the whole character, which Pele's metricsMeasurer gives the same width.
func fit(label text.Label, maxWidth float64, maxLines int, measure text.Measurer) text.Label {
	cut := len(label.Lines) > maxLines
	if !cut && label.Width <= maxWidth {
		return label
	}
	dots := measure.Width("…", label.Size, 0)
	var lines [][]text.Span
	var widths []float64
	widest := 0.0
	for i := range min(maxLines, len(label.Lines)) {
		line := slices.Clone(label.Lines[i])
		width := label.Widths[i]
		if width > maxWidth || cut && i == maxLines-1 {
			for {
				if len(line) == 0 {
					line = append(line, text.Span{Text: "…", Width: dots})
					width = dots
					break
				}
				span := &line[len(line)-1]
				rest := width - span.Width
				if span.Icon == "" && rest+dots < maxWidth {
					lo, hi := 0, chars.Len16(span.Text)
					for lo < hi {
						mid := (lo + hi + 1) >> 1
						end, _ := prefix16(span.Text, mid)
						if rest+measure.Width(span.Text[:end], label.Size, span.Face)+dots <= maxWidth {
							lo = mid
						} else {
							hi = mid - 1
						}
					}
					end, split := prefix16(span.Text, lo)
					if split {
						end, _ = prefix16(span.Text, lo-1)
					}
					span.Text = chars.TrimEnd(span.Text[:end]) + "…"
					span.Width = measure.Width(span.Text, label.Size, span.Face)
					width = rest + span.Width
					break
				}
				line = line[:len(line)-1]
				width = rest
			}
		}
		lines = append(lines, line)
		widths = append(widths, width)
		widest = max(widest, width)
	}
	label.Lines = lines
	label.Widths = widths
	label.Width = widest
	label.Height = float64(len(lines)) * label.LineHeight
	return label
}

// A port is where an edge leaves or reaches a node, Pele's Port: the point, the way out of
// the side, the box the route has to stay out of when it cannot leave straight, how far short
// of the point an arrowhead stops (the radius of a junction's dot), and whether a junction is
// reached through the border of its group, so that the line carries on to the junction.
type port struct {
	x, y, dx, dy   float64
	x0, y0, x1, y1 float64
	gap            float64
	wire           bool
}

// route is an orthogonal route from one port to another, each left along its side's normal,
// as Pele's route gives it.
func route(s, e *port) []float64 {
	sx, sy, sdx, sdy := s.x, s.y, s.dx, s.dy
	ex, ey, edx, edy := e.x, e.y, e.dx, e.dy
	s1x := sx + float64(sdx*stub)
	s1y := sy + float64(sdy*stub)
	e1x := ex + float64(edx*stub)
	e1y := ey + float64(edy*stub)
	if float64(sdx*edx)+float64(sdy*edy) == 0 {
		cx, cy := sx, ey
		if sdx != 0 {
			cx, cy = ex, sy
		}
		if float64((cx-sx)*sdx)+float64((cy-sy)*sdy) > 0 && float64((cx-ex)*edx)+float64((cy-ey)*edy) > 0 {
			return []float64{sx, sy, cx, cy, ex, ey}
		}
		if sdx != 0 {
			return []float64{sx, sy, s1x, s1y, s1x, e1y, e1x, e1y, ex, ey}
		}
		return []float64{sx, sy, s1x, s1y, e1x, s1y, e1x, e1y, ex, ey}
	}
	sideways := sdx != 0
	offset := ex - sx
	if sideways {
		offset = ey - sy
	}
	if sdx == -edx && sdy == -edy {
		ahead := float64((ex-sx)*sdx) + float64((ey-sy)*sdy)
		if ahead > 0 {
			if math.Abs(offset) < 0.5 {
				return []float64{sx, sy, ex, ey}
			}
			// Turn midway between the two boxes, which is clear of both titles, when there is
			// such a place.
			if sideways {
				lo, hi := e.x1, s.x0
				if sdx > 0 {
					lo, hi = s.x1, e.x0
				}
				mx := (sx + ex) / 2
				if lo < hi {
					mx = (lo + hi) / 2
				}
				return []float64{sx, sy, mx, sy, mx, ey, ex, ey}
			}
			lo, hi := e.y1, s.y0
			if sdy > 0 {
				lo, hi = s.y1, e.y0
			}
			my := (sy + ey) / 2
			if lo < hi {
				my = (lo + hi) / 2
			}
			return []float64{sx, sy, sx, my, ex, my, ex, ey}
		}
	} else if math.Abs(offset) >= 1 {
		if sideways {
			x := min(sx, ex) - stub
			if sdx > 0 {
				x = max(sx, ex) + stub
			}
			return []float64{sx, sy, x, sy, x, ey, ex, ey}
		}
		y := min(sy, ey) - stub
		if sdy > 0 {
			y = max(sy, ey) + stub
		}
		return []float64{sx, sy, sx, y, ex, y, ex, ey}
	}
	// The straight way is blocked by one of the two nodes: go between them, or around both.
	if sideways {
		var lane float64
		switch {
		case s.y1+stub <= e.y0:
			lane = (s.y1 + e.y0) / 2
		case e.y1+stub <= s.y0:
			lane = (e.y1 + s.y0) / 2
		default:
			lane = max(s.y1, e.y1) + stub
		}
		return []float64{sx, sy, s1x, sy, s1x, lane, e1x, lane, e1x, ey, ex, ey}
	}
	var lane float64
	switch {
	case s.x1+stub <= e.x0:
		lane = (s.x1 + e.x0) / 2
	case e.x1+stub <= s.x0:
		lane = (e.x1 + s.x0) / 2
	default:
		lane = max(s.x1, e.x1) + stub
	}
	return []float64{sx, sy, sx, s1y, lane, s1y, lane, e1y, ex, e1y, ex, ey}
}

// simplify drops repeated points and joins runs that continue in a straight line, as Pele's
// simplify does.
func simplify(p []float64) []float64 {
	out := []float64{p[0], p[1]}
	for i := 2; i < len(p); i += 2 {
		x, y := p[i], p[i+1]
		n := len(out)
		sameX := math.Abs(x-out[n-2]) < 0.01
		sameY := math.Abs(y-out[n-1]) < 0.01
		if sameX && sameY {
			continue
		}
		if n >= 4 && (sameX && math.Abs(out[n-4]-out[n-2]) < 0.01 || sameY && math.Abs(out[n-3]-out[n-1]) < 0.01) {
			out[n-2] = x
			out[n-1] = y
		} else {
			out = append(out, x, y)
		}
	}
	return out
}

// pathData is the path through the points, trimmed at each end and with its corners rounded,
// as Pele's pathData draws it.
func pathData(p []float64, startTrim, endTrim float64) *scene.Path {
	n := len(p) / 2
	xs := make([]float64, n)
	ys := make([]float64, n)
	for i := range n {
		xs[i] = p[i*2]
		ys[i] = p[i*2+1]
	}
	trim := func(at, toward int, amount float64) {
		l := jsmath.Hypot(xs[toward]-xs[at], ys[toward]-ys[at])
		t := min(amount, max(0, l-1))
		if l > 0 && t > 0 {
			xs[at] += float64(((xs[toward] - xs[at]) / l) * t)
			ys[at] += float64(((ys[toward] - ys[at]) / l) * t)
		}
	}
	trim(0, 1, startTrim)
	trim(n-1, n-2, endTrim)
	path := new(scene.Path).MoveTo(xs[0], ys[0])
	for i := 1; i < n-1; i++ {
		before := jsmath.Hypot(xs[i]-xs[i-1], ys[i]-ys[i-1])
		after := jsmath.Hypot(xs[i+1]-xs[i], ys[i+1]-ys[i])
		r := min(corner, before/2, after/2)
		if r < 0.5 {
			path.LineTo(xs[i], ys[i])
			continue
		}
		inX := xs[i] - float64(((xs[i]-xs[i-1])/before)*r)
		inY := ys[i] - float64(((ys[i]-ys[i-1])/before)*r)
		outX := xs[i] + float64(((xs[i+1]-xs[i])/after)*r)
		outY := ys[i] + float64(((ys[i+1]-ys[i])/after)*r)
		path.LineTo(inX, inY).QuadTo(xs[i], ys[i], outX, outY)
	}
	return path.LineTo(xs[n-1], ys[n-1])
}

// Layout lays an architecture diagram out as a scene, as Pele's renderArchitecture draws it:
// the services, junctions and groups on the grid placeOnGrid gives them, the rows and columns
// as wide as what is in them needs, and the edges in orthogonal routes between the sides they
// name. A diagram wider than the width the host has is laid out again more tightly, and kept
// so when that is narrower.
//
// An icon the host can draw (layout.Options.Icons) is a Picture of the service or the group;
// one of the five Pele knows itself, database, server, disk, internet and cloud, is an item of
// its own, architecture.icon, its outline stroked in the color of text. A name neither knows
// draws nothing, as Pele draws an empty icon.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.Tighten(o, func(tight bool) *scene.Scene { return lay(m, config, o, tight) })
}

func lay(m *Model, config diagram.Config, o *layout.Options, tight bool) *scene.Scene {
	iconSize := setting(config, "iconSize", 48, 8, 512)
	size := setting(config, "fontSize", o.Size(), 4, 128)
	groupPad := setting(config, "padding", 20, 0, 256)
	if tight {
		groupPad = min(groupPad, 12)
	}
	small := jsmath.Round(float64(size * 0.875))
	measure := o.Measure()
	pad := o.Pad()
	half := iconSize / 2
	layoutLabel := func(raw string, markdown bool, size, maxWidth float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: maxWidth, Bounded: true, Face: face,
			Markdown: markdown, Icons: o.Icons})
	}

	nodes := m.Nodes
	groups := m.Groups
	N := len(nodes)
	G := len(groups)
	g := placeOnGrid(nodes, groups, m.Edges, m.LayoutHints)
	index := map[string]int{}
	for i, node := range nodes {
		index[node.ID] = i
	}

	// A service is its icon square with the title beneath; a junction takes the same room.
	labels := make([]text.Label, N)
	boxW := make([]float64, N)
	boxH := make([]float64, N)
	wrap := max(float64(iconSize*2.25), 96)
	if tight {
		wrap = max(iconSize, 64)
	}
	for i, node := range nodes {
		title := ""
		if !node.Junction {
			title = node.Title
		}
		label := layoutLabel(title, true, size, wrap, 0)
		labels[i] = label
		boxW[i] = max(iconSize, float64(2*math.Ceil(label.Width/2)))
		boxH[i] = iconSize
		if label.Height > 0 {
			boxH[i] = iconSize + (labelGap + label.Height)
		}
	}

	var colValues, rowValues []int
	for i := range N {
		colValues = append(colValues, g.col[i])
		rowValues = append(rowValues, g.row[i])
	}
	for k := range G {
		if g.groupEmpty[k] {
			colValues = append(colValues, g.groupCol[k])
			rowValues = append(rowValues, g.groupRow[k])
		}
	}
	colOf := rank(colValues)
	rowOf := rank(rowValues)
	C := len(colOf)
	R := len(rowOf)
	colW := make([]float64, C)
	rowH := make([]float64, R)
	gap := max(40, jsmath.Round(float64(iconSize*0.75)))
	if tight {
		gap = max(24, 2*stub)
	}
	gapX := make([]float64, C)
	gapY := make([]float64, R)
	for c := range gapX {
		gapX[c] = gap
	}
	for r := range gapY {
		gapY[r] = gap
	}
	nodeCol := make([]int, N)
	nodeRow := make([]int, N)
	for i := range N {
		c, r := colOf[g.col[i]], rowOf[g.row[i]]
		nodeCol[i], nodeRow[i] = c, r
		colW[c] = max(colW[c], boxW[i])
		rowH[r] = max(rowH[r], boxH[i])
	}
	for k := range G {
		if !g.groupEmpty[k] {
			continue
		}
		c, r := colOf[g.groupCol[k]], rowOf[g.groupRow[k]]
		colW[c] = max(colW[c], iconSize)
		rowH[r] = max(rowH[r], half)
	}

	// An edge label needs the gap its edge crosses to be wide enough to hold it.
	E := len(m.Edges)
	edgeLabels := make([]text.Label, E)
	ends := make([]int, E*2)
	for k, edge := range m.Edges {
		a, b := -1, -1
		if i, ok := index[edge.LhsID]; ok {
			a = i
		}
		if i, ok := index[edge.RhsID]; ok {
			b = i
		}
		ends[k*2], ends[k*2+1] = a, b
		label := layoutLabel(edge.Title, true, small, 160, 0)
		edgeLabels[k] = label
		if label.Width == 0 || a < 0 || b < 0 {
			continue
		}
		if nodeCol[a] != nodeCol[b] {
			c := min(nodeCol[a], nodeCol[b])
			gapX[c] = max(gapX[c], math.Ceil(label.Width)+28)
		} else {
			colW[nodeCol[a]] = max(colW[nodeCol[a]], float64(2*math.Ceil(label.Width/2))+8)
			if nodeRow[a] != nodeRow[b] {
				r := min(nodeRow[a], nodeRow[b])
				gapY[r] = max(gapY[r], math.Ceil(label.Height)+24)
			}
		}
	}

	// Room around each group, kept on the rows and columns at its edges. A group that shares
	// an edge with a group inside it stands that much further out. Children come after their
	// parents, so walking backwards settles every group before the one that holds it.
	groupLabels := make([]text.Label, G)
	c0 := make([]int, G)
	c1 := make([]int, G)
	r0 := make([]int, G)
	r1 := make([]int, G)
	insL := make([]float64, G)
	insR := make([]float64, G)
	insT := make([]float64, G)
	insB := make([]float64, G)
	headW := make([]float64, G)
	headH := make([]float64, G)
	colL := make([]float64, C)
	colR := make([]float64, C)
	rowT := make([]float64, R)
	rowB := make([]float64, R)
	badge := jsmath.Round(float64(size * 1.25))
	groupWrap := 320.0
	if tight {
		groupWrap = max(80, common.TitleRoom(o)/2)
	}
	for k, group := range groups {
		c0[k] = colOf[g.groupCol[k]]
		c1[k] = colOf[g.groupCol[k]+g.groupCols[k]-1]
		r0[k] = rowOf[g.groupRow[k]]
		r1[k] = rowOf[g.groupRow[k]+g.groupRows[k]-1]
		label := layoutLabel(group.Title, true, size, groupWrap, 0)
		groupLabels[k] = label
		icon := 0.0
		if group.Icon != "" {
			icon = badge
		}
		headW[k] = label.Width + icon
		if label.Width > 0 && icon > 0 {
			headW[k] += 6
		}
		headH[k] = max(label.Height, icon)
	}
	for k := G - 1; k >= 0; k-- {
		left := groupPad + insL[k]
		right := groupPad + insR[k]
		top := groupPad + insT[k]
		if headH[k] > 0 {
			top = max(groupPad, headH[k]+12) + insT[k]
		}
		bottom := groupPad + insB[k]
		if headW[k] > 0 {
			inner := left + right
			for c := c0[k]; c <= c1[k]; c++ {
				if c < c1[k] {
					inner += colW[c] + (colR[c] + gapX[c] + colL[c+1])
				} else {
					inner += colW[c]
				}
			}
			if headW[k]+20 > inner {
				right += math.Ceil(headW[k] + 20 - inner)
			}
		}
		insL[k], insR[k], insT[k], insB[k] = left, right, top, bottom
		colL[c0[k]] = max(colL[c0[k]], left)
		colR[c1[k]] = max(colR[c1[k]], right)
		rowT[r0[k]] = max(rowT[r0[k]], top)
		rowB[r1[k]] = max(rowB[r1[k]], bottom)
		p := g.groupParent[k]
		if p < 0 {
			continue
		}
		if c0[p] == c0[k] {
			insL[p] = max(insL[p], left)
		}
		if c1[p] == c1[k] {
			insR[p] = max(insR[p], right)
		}
		if r0[p] == r0[k] {
			insT[p] = max(insT[p], top)
		}
		if r1[p] == r1[k] {
			insB[p] = max(insB[p], bottom)
		}
	}

	X := make([]float64, C)
	Y := make([]float64, R)
	innerW := 0.0
	for c := range C {
		X[c] = innerW + colL[c]
		innerW = X[c] + colW[c] + colR[c]
		if c < C-1 {
			innerW += gapX[c]
		}
	}
	innerH := 0.0
	for r := range R {
		Y[r] = innerH + rowT[r]
		innerH = Y[r] + rowH[r] + rowB[r]
		if r < R-1 {
			innerH += gapY[r]
		}
	}
	// What is drawn, which an edge that has to go around a node can push past the grid.
	minX, minY, maxX, maxY := 0.0, 0.0, innerW, innerH

	groupLeft := func(k int) float64 { return X[c0[k]] - insL[k] }
	groupRight := func(k int) float64 { return X[c1[k]] + colW[c1[k]] + insR[k] }
	groupTop := func(k int) float64 { return Y[r0[k]] - insT[k] }
	groupBottom := func(k int) float64 { return Y[r1[k]] + rowH[r1[k]] + insB[k] }
	centerX := func(i int) float64 { return X[nodeCol[i]] + colW[nodeCol[i]]/2 }

	portOf := func(i, side int, viaGroup bool) *port {
		dx, dy := float64(gridDX[side]), float64(gridDY[side])
		cx := centerX(i)
		y := Y[nodeRow[i]]
		p := &port{x: cx, y: y + half, dx: dx, dy: dy, x0: cx - boxW[i]/2, y0: y, x1: cx + boxW[i]/2, y1: y + boxH[i]}
		switch {
		case nodes[i].Junction:
			p.x0, p.x1 = cx, cx
			p.y0, p.y1 = p.y, p.y
			p.gap = dot
		case dx != 0:
			p.x = cx + float64(dx*half)
		case dy < 0:
			p.y = y
		default:
			p.y = y + boxH[i]
			if labels[i].Height > 0 {
				p.y += 2
			}
		}
		k := -1
		if viaGroup {
			k = g.nodeParent[i]
		}
		if k >= 0 {
			p.x0, p.x1 = groupLeft(k), groupRight(k)
			p.y0, p.y1 = groupTop(k), groupBottom(k)
			p.gap = 0
			p.wire = nodes[i].Junction
			if dx != 0 {
				p.x = p.x1
				if dx < 0 {
					p.x = p.x0
				}
			} else {
				p.y = p.y1
				if dy < 0 {
					p.y = p.y0
				}
			}
		}
		return p
	}

	var edgeItems, labelItems []scene.Item
	// Vertical runs of the edges, to keep group titles out from under them.
	var runs []float64
	trimLength := scene.MarkerType("arrow_point").Trim()
	edgeStyle := style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound, Join: style.JoinRound}
	for k, edge := range m.Edges {
		a, b := ends[k*2], ends[k*2+1]
		if a < 0 || b < 0 {
			continue
		}
		s := portOf(a, sideIndex(edge.LhsDir), edge.LhsGroup)
		e := portOf(b, sideIndex(edge.RhsDir), edge.RhsGroup)
		if s.x == e.x && s.y == e.y && s.dx == e.dx {
			// An edge from a side back to the same side: part the two ends so it shows as a
			// loop.
			s.x -= float64(8 * math.Abs(s.dy))
			s.y -= float64(8 * math.Abs(s.dx))
			e.x += float64(8 * math.Abs(s.dy))
			e.y += float64(8 * math.Abs(s.dx))
		}
		routed := route(s, e)
		if s.wire {
			routed = append([]float64{centerX(a), Y[nodeRow[a]] + half}, routed...)
		}
		if e.wire {
			routed = append(routed, centerX(b), Y[nodeRow[b]]+half)
		}
		points := simplify(routed)
		if len(points) < 4 {
			continue
		}
		id := edge.LhsID + "-" + edge.RhsID
		startTrim, endTrim := 0.0, 0.0
		if edge.LhsInto && !s.wire {
			startTrim = trimLength + s.gap
		}
		if edge.RhsInto && !e.wire {
			endTrim = trimLength + e.gap
		}
		path := pathData(points, startTrim, endTrim)
		it := scene.Item{
			Kind: scene.Edge, Role: RoleEdge, ID: id,
			Box:   path.Bounds(),
			Parts: []scene.Part{{Path: path, Paint: scene.StrokeOnly}},
			Style: edgeStyle,
		}
		for i := 0; i < len(points); i += 2 {
			it.Route = append(it.Route, scene.Point{X: points[i], Y: points[i+1]})
		}
		if edge.LhsInto {
			it.Ends[0] = scene.Marker{Type: "arrow_point", At: scene.Point{X: s.x + float64(s.dx*s.gap), Y: s.y + float64(s.dy*s.gap)}, DX: -s.dx, DY: -s.dy}
		}
		if edge.RhsInto {
			it.Ends[1] = scene.Marker{Type: "arrow_point", At: scene.Point{X: e.x + float64(e.dx*e.gap), Y: e.y + float64(e.dy*e.gap)}, DX: -e.dx, DY: -e.dy}
		}
		edgeItems = append(edgeItems, it)

		best, bestLength, flat, flatLength := -1, -1.0, -1, -1.0
		label := &edgeLabels[k]
		for i := 0; i < len(points); i += 2 {
			minX = min(minX, points[i]-4)
			maxX = max(maxX, points[i]+4)
			minY = min(minY, points[i+1]-4)
			maxY = max(maxY, points[i+1]+4)
		}
		for i := 0; i+3 < len(points); i += 2 {
			horizontal := math.Abs(points[i+1]-points[i+3]) < 0.01
			length := math.Abs(points[i]-points[i+2]) + math.Abs(points[i+1]-points[i+3])
			if !horizontal {
				runs = append(runs, points[i], min(points[i+1], points[i+3]), max(points[i+1], points[i+3]))
			}
			if length > bestLength {
				best, bestLength = i, length
			}
			if horizontal && length > flatLength {
				flat, flatLength = i, length
			}
		}
		if label.Width > 0 {
			at := best
			if flatLength >= label.Width+8 {
				at = flat
			}
			x := (points[at] + points[at+2]) / 2
			y := (points[at+1] + points[at+3]) / 2
			w := label.Width + 8
			upright := math.Abs(points[at]-points[at+2]) < 0.01
			// On a short run next to a node the label would land on the node, so it moves off
			// to the side.
			for _, p := range [2]*port{s, e} {
				if x+w/2 <= p.x0 || x-w/2 >= p.x1 || y+label.Height/2 <= p.y0 || y-label.Height/2 >= p.y1 {
					continue
				}
				if upright {
					if x >= (p.x0+p.x1)/2 {
						x = p.x1 + w/2 + 4
					} else {
						x = p.x0 - w/2 - 4
					}
				} else if y >= (p.y0+p.y1)/2 {
					y = p.y1 + label.Height/2 + 4
				} else {
					y = p.y0 - label.Height/2 - 4
				}
			}
			minX = min(minX, x-w/2)
			maxX = max(maxX, x+w/2)
			minY = min(minY, y-label.Height/2)
			maxY = max(maxY, y+label.Height/2)
			patch := draw.LabelPatch(label, x, y)
			labelItems = append(labelItems, scene.Item{
				Kind: scene.Text, Role: RoleEdgeLabel, ID: id,
				Box:   draw.LabelPatchBox(label, x, y),
				Parts: []scene.Part{patch},
				Label: label,
				At:    scene.Point{X: x, Y: y},
				Style: style.Style{Fill: style.Bg, Stroke: style.None, Text: style.Fg},
			})
		}
	}

	// glyph is the icon of a service or a group in the box at x, y, side wide: a Picture of the
	// owner where the host draws it, or an item of its own for one of Pele's glyphs.
	glyph := func(owner *scene.Item, name string, x, y, side float64) *scene.Item {
		box := scene.Rect{X: x, Y: y, W: side, H: side}
		if common.HostIcon(o, name) {
			owner.Picture = &scene.Picture{Icon: text.DecodeEntities(name), Box: box}
			return nil
		}
		outline, ok := glyphPaths[name]
		if !ok {
			return nil
		}
		return common.Glyph(RoleIcon, name, owner.ID, outline, box, style.Fg)
	}

	var groupItems []scene.Item
	for k, group := range groups {
		x := groupLeft(k)
		y := groupTop(k)
		w := groupRight(k) - x
		box := scene.Rect{X: x, Y: y, W: w, H: groupBottom(k) - y}
		it := scene.Item{
			Kind: scene.Group, Role: RoleGroup, Shape: "rect", ID: group.ID,
			Box:   box,
			Parts: []scene.Part{{Box: box, Rounded: true}},
			Style: style.Style{Fill: style.SurfaceAlt, FillOpacity: style.N(0.5), Stroke: style.Border, Text: style.Muted},
		}
		var icon *scene.Item
		if headW[k] > 0 {
			// The title starts at the left, or after the edges that come down through that
			// corner.
			band := y + headH[k] + 12
			var crossings []float64
			for i := 0; i < len(runs); i += 3 {
				if runs[i+1] < band && runs[i+2] > y && runs[i] > x && runs[i] < x+w {
					crossings = append(crossings, runs[i])
				}
			}
			slices.Sort(crossings)
			crossings = append(crossings, x+w-4)
			start := x + 10
			for _, crossing := range crossings {
				if crossing-6-start >= headW[k] {
					break
				}
				start = max(start, crossing+6)
			}
			if start+headW[k] > x+w-4 {
				start = x + 10
			}
			cy := y + 6 + headH[k]/2
			textX := start
			if group.Icon != "" {
				icon = glyph(&it, group.Icon, start, cy-badge/2, badge)
				textX += badge + 6
			}
			if label := &groupLabels[k]; !label.Empty() {
				it.Label = label
				it.At = scene.Point{X: textX + label.Width/2, Y: cy}
			}
		}
		groupItems = append(groupItems, it)
		if icon != nil {
			groupItems = append(groupItems, *icon)
		}
	}

	var nodeItems []scene.Item
	glyphSide := float64(iconSize * 0.5)
	textSize := max(6, jsmath.Round(float64(size*0.75)))
	for i, node := range nodes {
		cx, cy := centerX(i), Y[nodeRow[i]]
		if node.Junction {
			nodeItems = append(nodeItems, scene.Item{
				Kind: scene.Node, Role: RoleJunction, Shape: "circle", ID: node.ID,
				Box:   scene.Rect{X: cx - dot, Y: cy + half - dot, W: 2 * dot, H: 2 * dot},
				Parts: []scene.Part{{Path: new(scene.Path).Ellipse(cx, cy+half, dot, dot), Paint: scene.FillOnly}},
				Style: style.Style{Fill: style.Line},
			})
			continue
		}
		box := scene.Rect{X: cx - half, Y: cy, W: iconSize, H: iconSize}
		it := scene.Item{
			Kind: scene.Node, Role: RoleService, Shape: "rect", ID: node.ID,
			Box:   box,
			Parts: []scene.Part{{Box: box, Rounded: true}},
			Style: style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg},
		}
		if label := &labels[i]; !label.Empty() {
			it.Label = label
			it.At = scene.Point{X: cx, Y: cy + (iconSize + labelGap + label.Height/2)}
		}
		var inner *scene.Item
		if node.Icon != "" {
			inner = glyph(&it, node.Icon, cx-glyphSide/2, cy+(iconSize-glyphSide)/2, glyphSide)
		} else if node.IconText != "" {
			label := layoutLabel(node.IconText, false, textSize, iconSize-8, 0)
			lineHeight := label.LineHeight
			if lineHeight == 0 {
				lineHeight = 1
			}
			lines := max(1, int(math.Floor((iconSize-4)/lineHeight)))
			label = fit(label, iconSize-8, lines, measure)
			if !label.Empty() {
				inner = &scene.Item{
					Kind: scene.Text, Role: RoleIconText, ID: node.ID,
					Label: &label,
					At:    scene.Point{X: cx, Y: cy + half},
					Style: style.Style{Text: style.Fg},
				}
			}
		}
		nodeItems = append(nodeItems, it)
		if inner != nil {
			nodeItems = append(nodeItems, *inner)
		}
	}

	titleRoom := 4000.0
	if tight {
		titleRoom = common.TitleRoom(o)
	}
	title := layoutLabel(m.Title, false, size, titleRoom, text.Bold)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	width := math.Ceil(max(maxX-minX, title.Width) + 2*pad)
	height := math.Ceil(maxY - minY + titleHeight + 2*pad)
	ox := jsmath.Round((width-(maxX-minX))/2 - minX)
	oy := jsmath.Round(pad + titleHeight - minY)

	sc := &scene.Scene{Type: "architecture", Width: width, Height: height, FontSize: size, Title: m.AccTitle, Desc: m.AccDescr}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	body := len(sc.Items)
	sc.Items = append(sc.Items, groupItems...)
	sc.Items = append(sc.Items, edgeItems...)
	sc.Items = append(sc.Items, labelItems...)
	sc.Items = append(sc.Items, nodeItems...)
	for i := body; i < len(sc.Items); i++ {
		sc.Items[i].Move(ox, oy)
	}
	return sc
}
