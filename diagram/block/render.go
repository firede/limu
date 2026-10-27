package block

import (
	"math"
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/diagram/internal/draw"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of a block diagram, as Pele's render.ts has them.
const (
	gapDefault = 12
	wrapWidth  = 200
	corner     = 8
	// obstacleBudget is how many checks of an edge against a block a diagram may take:
	// checking every edge against every block is quadratic, so large diagrams route without
	// them.
	obstacleBudget = 2_000_000
	// sqrt1_2 is Math.SQRT1_2.
	sqrt1_2 = 0.7071067811865476
)

// The roles of the items of a block diagram.
const (
	RoleTitle     scene.Role = "block.title"
	RoleComposite scene.Role = "block.composite"
	RoleBlock     scene.Role = "block.block"
	RoleEdge      scene.Role = "block.edge"
	RoleEdgeLabel scene.Role = "block.edge-label"
)

// shapes are the shapes Pele draws the types of block with, Pele's SHAPES; any other is a
// rect.
var shapes = map[string]string{
	"square":              "rect",
	"round":               "rounded",
	"circle":              "circle",
	"doublecircle":        "dbl-circ",
	"diamond":             "diam",
	"hexagon":             "hex",
	"stadium":             "stadium",
	"subroutine":          "fr-rect",
	"cylinder":            "cyl",
	"rect_left_inv_arrow": "odd",
	"lean_right":          "lean-r",
	"lean_left":           "lean-l",
	"trapezoid":           "trap-b",
	"inv_trapezoid":       "trap-t",
}

// fixedShape says which shapes keep their own size in a larger cell, where the others stretch
// to fill it, Pele's FIXED; wideShape which only widen, because their slants and curves are
// set by their height, Pele's WIDE.
func fixedShape(s string) bool { return s == "circle" || s == "dbl-circ" || s == "diam" }

func wideShape(s string) bool {
	switch s {
	case "stadium", "hex", "odd", "lean-r", "lean-l", "trap-b", "trap-t":
		return true
	}
	return false
}

// The kinds of view, Pele's Kind.
const (
	kindLeaf = iota
	kindArrow
	kindComposite
	kindSpace
)

// A view is a block as it is drawn, Pele's View: its cell, and what it is drawn with.
type view struct {
	cell
	block  *Block
	kind   int
	index  int
	shape  string
	label  text.Label
	author css.Author
	mask   int
	dx, dy float64
	// bw and bh are the size as drawn, which for a fixed shape is smaller than its cell.
	bw, bh float64
}

// A route is an edge as it is drawn, Pele's Route: two points for a straight line, three for
// a bend, none for a self-loop.
type route struct {
	edge   *Block
	points []float64
	loop   int
	label  text.Label
	lx, ly float64
}

func isRound(v *view) bool {
	return v.kind == kindLeaf && (v.shape == "circle" || v.shape == "dbl-circ")
}

// edgeY is where a vertical line at x meets the outline, below the center (sign 1) or above
// it (sign -1), as Pele's edgeY finds it.
func edgeY(v *view, x, sign float64) float64 {
	hw := v.bw / 2
	hh := v.bh / 2
	dx := math.Abs(x - v.x)
	reach := hh
	switch {
	case isRound(v):
		reach = math.Sqrt(max(float64(hw*hw)-float64(dx*dx), 0))
	case v.kind == kindLeaf && v.shape == "diam":
		reach = float64(hh * max(1-dx/hw, 0))
	case v.kind == kindLeaf:
		side := 0
		if sign > 0 {
			side = 2
		}
		reach = hh - draw.ShapeInset(v.shape, v.bw, v.bh, side, 0)
	case v.kind == kindArrow && arrowAxis(v.mask) == 'x':
		reach = hh - overhang
	}
	return v.y + float64(sign*reach)
}

// edgeX is where a horizontal line at y meets the outline, right of the center (sign 1) or
// left of it (sign -1), as Pele's edgeX finds it.
func edgeX(v *view, y, sign float64) float64 {
	hw := v.bw / 2
	hh := v.bh / 2
	dy := math.Abs(y - v.y)
	reach := hw
	switch {
	case isRound(v):
		reach = math.Sqrt(max(float64(hh*hh)-float64(dy*dy), 0))
	case v.kind == kindLeaf && v.shape == "diam":
		reach = float64(hw * max(1-dy/hh, 0))
	case v.kind == kindLeaf:
		side := 3
		if sign > 0 {
			side = 1
		}
		reach = hw - draw.ShapeInset(v.shape, v.bw, v.bh, side, 0)
	case v.kind == kindArrow && arrowAxis(v.mask) == 'y':
		reach = hw - overhang
	}
	return v.x + float64(sign*reach)
}

// border is where the line from the center toward tx, ty leaves the outline, as Pele's
// border finds it, added to out.
func border(v *view, tx, ty float64, out []float64) []float64 {
	dx := tx - v.x
	dy := ty - v.y
	hw := v.bw / 2
	hh := v.bh / 2
	var k float64
	switch {
	case isRound(v):
		k = hw / jsmath.Or1(jsmath.Hypot(dx, dy))
	case v.kind == kindLeaf && v.shape == "diam":
		k = 1 / jsmath.Or1(math.Abs(dx)/hw+math.Abs(dy)/hh)
	default:
		kx, ky := math.Inf(1), math.Inf(1)
		if dx != 0 {
			kx = hw / math.Abs(dx)
		}
		if dy != 0 {
			ky = hh / math.Abs(dy)
		}
		k = min(kx, ky)
	}
	if math.IsInf(k, 1) {
		k = 0
	}
	return append(out, v.x+float64(dx*k), v.y+float64(dy*k))
}

// crossesBox reports whether the segment from x1, y1 to x2, y2 crosses the box of v, a
// little inside its outline, as Pele's crossesBox does.
func crossesBox(x1, y1, x2, y2 float64, v *view) bool {
	l := v.x - v.bw/2 + 1
	r := v.x + v.bw/2 - 1
	t := v.y - v.bh/2 + 1
	b := v.y + v.bh/2 - 1
	if x1 < l && x2 < l || x1 > r && x2 > r || y1 < t && y2 < t || y1 > b && y2 > b {
		return false
	}
	if x1 == x2 || y1 == y2 {
		return true
	}
	// The segment is diagonal and its bounding box overlaps: the box is hit unless all four
	// corners are on one side.
	dx := x2 - x1
	dy := y2 - y1
	side := func(x, y float64) float64 { return jsmath.Sign(float64(dx*(y-y1)) - float64(dy*(x-x1))) }
	first := side(l, t)
	return first == 0 || side(r, t) != first || side(r, b) != first || side(l, b) != first
}

// hits counts the obstacles other than s and t that a path through points crosses, as
// Pele's hits does.
func hits(points []float64, s, t *view, obstacles []*view) int {
	count := 0
	for _, v := range obstacles {
		if v == s || v == t {
			continue
		}
		for i := 0; i+3 < len(points); i += 2 {
			if crossesBox(points[i], points[i+1], points[i+2], points[i+3], v) {
				count++
				break
			}
		}
	}
	return count
}

// routeEdge is the path of an edge from s to t, as Pele's routeEdge finds it: a straight line
// where the two blocks share a row or a column, otherwise one bend between facing sides, or a
// straight line between their outlines where that crosses fewer blocks. obstacles is nil
// where the diagram is too large to check.
func routeEdge(s, t *view, shift float64, obstacles []*view) []float64 {
	sl := s.x - s.bw/2
	sr := s.x + s.bw/2
	st := s.y - s.bh/2
	sb := s.y + s.bh/2
	tl := t.x - t.bw/2
	tr := t.x + t.bw/2
	tt := t.y - t.bh/2
	tb := t.y + t.bh/2
	x1 := max(sl, tl)
	x2 := min(sr, tr)
	y1 := max(st, tt)
	y2 := min(sb, tb)
	overX := x2-x1 > 6
	overY := y2-y1 > 6

	if overX && overY {
		// One block is inside the other: join the inner one to the nearest side of the
		// outer one.
		inner, outer := t, s
		if float64(s.bw*s.bh) <= float64(t.bw*t.bh) {
			inner, outer = s, t
		}
		up := inner.y - inner.bh/2 - (outer.y - outer.bh/2)
		down := outer.y + outer.bh/2 - (inner.y + inner.bh/2)
		left := inner.x - inner.bw/2 - (outer.x - outer.bw/2)
		right := outer.x + outer.bw/2 - (inner.x + inner.bw/2)
		least := min(up, down, left, right)
		var points []float64
		switch least {
		case down:
			points = []float64{inner.x, edgeY(inner, inner.x, 1), inner.x, outer.y + outer.bh/2}
		case up:
			points = []float64{inner.x, edgeY(inner, inner.x, -1), inner.x, outer.y - outer.bh/2}
		case right:
			points = []float64{edgeX(inner, inner.y, 1), inner.y, outer.x + outer.bw/2, inner.y}
		default:
			points = []float64{edgeX(inner, inner.y, -1), inner.y, outer.x - outer.bw/2, inner.y}
		}
		if inner == s {
			return points
		}
		return []float64{points[2], points[3], points[0], points[1]}
	}
	if overX {
		margin := min(6, (x2-x1)/2)
		x := min(max((x1+x2)/2+shift, x1+margin), x2-margin)
		down := -1.0
		if t.y > s.y {
			down = 1
		}
		return []float64{x, edgeY(s, x, down), x, edgeY(t, x, -down)}
	}
	if overY {
		margin := min(6, (y2-y1)/2)
		y := min(max((y1+y2)/2+shift, y1+margin), y2-margin)
		right := -1.0
		if t.x > s.x {
			right = 1
		}
		return []float64{edgeX(s, y, right), y, edgeX(t, y, -right), y}
	}

	right, down := -1.0, -1.0
	if t.x > s.x {
		right = 1
	}
	if t.y > s.y {
		down = 1
	}
	across := []float64{edgeX(s, s.y, right), s.y, t.x, s.y, t.x, edgeY(t, t.x, -down)}
	along := []float64{s.x, edgeY(s, s.x, down), s.x, t.y, edgeX(t, t.y, -right), t.y}
	wide := math.Abs(t.x-s.x) >= math.Abs(t.y-s.y)
	best, other := along, across
	if wide {
		best, other = across, along
	}
	if obstacles == nil {
		return best
	}
	least := hits(best, s, t, obstacles)
	if least == 0 {
		return best
	}
	if count := hits(other, s, t, obstacles); count < least {
		best = other
		least = count
	}
	if least == 0 {
		return best
	}
	direct := border(s, t.x, t.y, nil)
	direct = border(t, s.x, s.y, direct)
	if hits(direct, s, t, obstacles) < least {
		return direct
	}
	return best
}

// trace is the path through two or three points, its bend rounded and room left for
// markers, as Pele's trace builds it.
func trace(p []float64, startTrim, endTrim float64) draw.EdgePath {
	n := len(p)
	sx, sy := p[0], p[1]
	ex, ey := p[n-2], p[n-1]
	ax := p[2] - sx
	ay := p[3] - sy
	first := jsmath.Hypot(ax, ay)
	ax /= jsmath.Or1(first)
	ay /= jsmath.Or1(first)
	bx := ex - p[n-4]
	by := ey - p[n-3]
	last := jsmath.Hypot(bx, by)
	bx /= jsmath.Or1(last)
	by /= jsmath.Or1(last)
	if n == 4 && startTrim+endTrim > first {
		scale := first / (startTrim + endTrim)
		startTrim *= scale
		endTrim *= scale
	}
	startTrim = min(startTrim, first)
	endTrim = min(endTrim, last)
	path := new(scene.Path).MoveTo(sx+float64(ax*startTrim), sy+float64(ay*startTrim))
	if n == 6 {
		r := max(0, min(corner, first-startTrim, last-endTrim))
		path.LineTo(p[2]-float64(ax*r), p[3]-float64(ay*r))
		path.QuadTo(p[2], p[3], p[2]+float64(bx*r), p[3]+float64(by*r))
	}
	path.LineTo(ex-float64(bx*endTrim), ey-float64(by*endTrim))
	return draw.EdgePath{Path: path, SX: sx, SY: sy, SDX: -ax, SDY: -ay, EX: ex, EY: ey, EDX: bx, EDY: by}
}

// loopArc is the arc of a self-loop, the kth on its block, as Pele's loopArc gives it: around
// the top right corner of the block, or around the matching point on a round or diamond
// outline. It gives the arc's center and radius and the angles where it leaves and returns.
func loopArc(v *view, k int) (px, py, r, a1, a2 float64) {
	hw := v.bw / 2
	hh := v.bh / 2
	r = max(1, min(10+5*float64(k-1), hw, hh))
	if isRound(v) {
		px = v.x + float64(hw*sqrt1_2)
		py = v.y - float64(hw*sqrt1_2)
		half := 2 * jsmath.Asin(min(1, r/(2*hw)))
		angle := func(at float64) float64 {
			return jsmath.Atan2(v.y-float64(hw*jsmath.Sin(at))-py, v.x+float64(hw*jsmath.Cos(at))-px)
		}
		return px, py, r, angle(math.Pi/4 + half), angle(math.Pi/4 - half)
	}
	if v.kind == kindLeaf && v.shape == "diam" {
		slope := jsmath.Atan2(hh, hw)
		return v.x + hw/2, v.y - hh/2, r, slope + math.Pi, slope
	}
	return v.x + hw, v.y - hh, r, math.Pi, math.Pi / 2
}

// loopPath is the path of the kth self-loop on a block, as Pele's loopPath draws it: an arc,
// with each marker along the chord it covers, so that it follows the curve instead of the
// tangent at its tip.
func loopPath(v *view, k int, startTrim, endTrim float64) draw.EdgePath {
	px, py, r, a1, a2 := loopArc(v, k)
	from := a1 + startTrim/r
	to := a2 - endTrim/r
	sweep := math.Mod(to-from, 2*math.Pi)
	if sweep <= 0 {
		sweep += 2 * math.Pi
	}
	x1 := px + float64(r*jsmath.Cos(from))
	y1 := py + float64(r*jsmath.Sin(from))
	x2 := px + float64(r*jsmath.Cos(to))
	y2 := py + float64(r*jsmath.Sin(to))
	sx := px + float64(r*jsmath.Cos(a1))
	sy := py + float64(r*jsmath.Sin(a1))
	ex := px + float64(r*jsmath.Cos(a2))
	ey := py + float64(r*jsmath.Sin(a2))
	sl := jsmath.Hypot(sx-x1, sy-y1)
	el := jsmath.Hypot(ex-x2, ey-y2)
	e := draw.EdgePath{
		Path: new(scene.Path).MoveTo(x1, y1).ArcTo(r, r, 0, sweep > math.Pi, true, x2, y2),
		SX:   sx, SY: sy, EX: ex, EY: ey,
		SDX: jsmath.Sin(a1), SDY: -jsmath.Cos(a1),
		EDX: -jsmath.Sin(a2), EDY: jsmath.Cos(a2),
	}
	if sl > 0 {
		e.SDX, e.SDY = (sx-x1)/sl, (sy-y1)/sl
	}
	if el > 0 {
		e.EDX, e.EDY = (ex-x2)/el, (ey-y2)/el
	}
	return e
}

// spots are the places along a leg a label is tried at, in turn, Pele's SPOTS.
var spots = [...]float64{0.5, 0.3, 0.7, 0.15, 0.85}

// A box is the box of a container's title, which stacked labels keep clear of, Pele's Box.
type box struct {
	x, y, bw, bh float64
}

// covers reports whether a label w by h centered on x, y covers a block other than s and t,
// or a title, as Pele's covers does. obstacles is nil where the diagram is too large to
// check, and then nothing is covered.
func covers(x, y, w, h float64, s, t *view, obstacles []*view, titles []box) bool {
	if obstacles == nil {
		return false
	}
	for _, v := range obstacles {
		if v != s && v != t && math.Abs(x-v.x) < (w+v.bw)/2 && math.Abs(y-v.y) < (h+v.bh)/2 {
			return true
		}
	}
	for _, v := range titles {
		if math.Abs(x-v.x) < (w+v.bw)/2 && math.Abs(y-v.y) < (h+v.bh)/2 {
			return true
		}
	}
	return false
}

// labelSpot is the point along the leg of p from at where a label w by h covers no block
// other than the two it joins, as Pele's labelSpot picks it.
func labelSpot(p []float64, at int, w, h float64, s, t *view, obstacles []*view) float64 {
	if obstacles == nil {
		return 0.5
	}
	for _, spot := range spots {
		x := p[at] + float64((p[at+2]-p[at])*spot)
		y := p[at+1] + float64((p[at+3]-p[at+1])*spot)
		if !covers(x, y, w, h, s, t, obstacles, nil) {
			return spot
		}
	}
	return 0.5
}

// labelGap is the room between labels stacked along a line, Pele's LABEL_GAP.
const labelGap = 4

// stackLabels stacks the labels of the edges of stack, which join the same two blocks, along
// the line they share where they cover each other, if they fit between its markers and cover
// no other block or title, as Pele's stackLabels does. It gives how much longer the line
// would have to be for them to fit, or 0.
func stackLabels(stack []int, routes []route, legs [][]float64, s, t *view, obstacles []*view, titles []box) float64 {
	vertical := legs[stack[0]][0] == legs[stack[0]][2]
	lo, hi := math.Inf(-1), math.Inf(1)
	need := float64(-labelGap)
	// along is the length of a label along the line.
	along := func(r *route) float64 {
		if vertical {
			return r.label.Height
		}
		return r.label.Width + 8
	}
	crowded := false
	for a, i := range stack {
		leg := legs[i]
		if vertical {
			if leg[0] != leg[2] {
				return 0
			}
			lo = max(lo, min(leg[1], leg[3]))
			hi = min(hi, max(leg[1], leg[3]))
		} else {
			if leg[1] != leg[3] {
				return 0
			}
			lo = max(lo, min(leg[0], leg[2]))
			hi = min(hi, max(leg[0], leg[2]))
		}
		p := &routes[i]
		need += along(p) + labelGap
		for _, j := range stack[:a] {
			q := &routes[j]
			if math.Abs(p.lx-q.lx) < (p.label.Width+q.label.Width)/2+8 && math.Abs(p.ly-q.ly) < (p.label.Height+q.label.Height)/2 {
				crowded = true
			}
		}
	}
	if !crowded {
		return 0
	}
	if room := hi - lo - 2*scene.MarkerSize; need > room {
		return need - room
	}
	places := make([]float64, 0, len(stack))
	next := (lo + hi - need) / 2
	for _, i := range stack {
		r := &routes[i]
		size := along(r)
		c := next + size/2
		x, y := c, r.ly
		if vertical {
			x, y = r.lx, c
		}
		if covers(x, y, r.label.Width+8, r.label.Height, s, t, obstacles, titles) {
			return 0
		}
		places = append(places, c)
		next += size + labelGap
	}
	for k, i := range stack {
		if vertical {
			routes[i].ly = places[k]
		} else {
			routes[i].lx = places[k]
		}
	}
	return 0
}

// gapsOf names the gaps of a container, its rows' or its columns'.
type gapsOf struct {
	box      *view
	vertical bool
}

// widen widens the gap between s and t in the nearest container they share by short, between
// rows where vertical is set and between columns otherwise, as Pele's widen does, and reports
// whether there was a gap to widen. base keeps the gaps as they were when the labels were
// measured, so that widening the same gap again does not add to what it added.
func widen(parents []*view, s, t *view, vertical bool, short float64, base map[gapsOf][]float64) bool {
	chain := map[*view]bool{}
	for v := s; v != nil; v = parents[v.index] {
		chain[v] = true
	}
	var tb *view
	container := t
	for container != nil && !chain[container] {
		tb = container
		container = parents[container.index]
	}
	if container == nil || tb == nil || container == s {
		return false
	}
	sb := s
	for parents[sb.index] != container {
		sb = parents[sb.index]
	}
	var i int
	gaps := &container.colGaps
	if vertical {
		if sb.row == tb.row {
			return false
		}
		i = int(max(sb.row, tb.row)) - 1
		gaps = &container.rowGaps
	} else {
		l, r := tb, sb
		if sb.col < tb.col {
			l, r = sb, tb
		}
		if r.col < l.col+l.span {
			return false
		}
		i = int(r.col) - 1
	}
	key := gapsOf{container, vertical}
	if _, ok := base[key]; !ok {
		base[key] = slices.Clone(*gaps)
	}
	for len(*gaps) <= i {
		*gaps = append(*gaps, 0)
	}
	(*gaps)[i] = max((*gaps)[i], at(base[key], i)+short)
	return true
}

// Layout lays out a parsed block diagram as a scene, as Pele's renderBlock draws it: with
// room to spare, and tightly instead when that does not fit o.MaxWidth and is narrower. The
// labels of edges between the same two blocks are stacked along their line where they would
// cover each other, and the gap between the blocks is widened where they do not fit, if that
// leaves less of them covered and fits o.MaxWidth. An error is a *diagram.Error, for a
// composite of no columns.
func Layout(db *DB, config diagram.Config, o *layout.Options) (*scene.Scene, error) {
	if o == nil {
		o = &layout.Options{}
	}
	var err error
	sc := common.Tighten(o, func(tight bool) *scene.Scene {
		sc, e := lay(db, config, o, tight)
		if e != nil {
			err = e
			return &scene.Scene{}
		}
		return sc
	})
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// lay lays the diagram out once, as Pele's draw does.
func lay(db *DB, config diagram.Config, o *layout.Options, tight bool) (*scene.Scene, error) {
	size := o.Size()
	edgeSize := jsmath.Round(size * 0.875)
	measure := o.Measure()
	pad := o.Pad()
	gap := float64(gapDefault)
	section, _ := config["block"].(diagram.Config)
	if configured, ok := section["padding"].(float64); ok && configured >= 0 {
		gap = min(configured, 200)
	}
	if tight {
		gap = min(gap, 6)
	}
	wrap := float64(wrapWidth)
	if tight {
		wrap = 80
	}
	labelOf := func(raw string, size, max float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: face, Icons: o.Icons})
	}

	// authorOf reads the declarations of a block's classes and styles, as Pele's styleOf
	// does: a style list is split on commas only, so one entry can still hold several
	// declarations.
	authorOf := func(b *Block) css.Author {
		var decls []string
		add := func(styles []string) {
			for _, s := range styles {
				decls = append(decls, strings.Split(s, ";")...)
			}
		}
		names := b.Classes
		if len(names) == 0 {
			names = []string{"default"}
		}
		for _, name := range names {
			if def, ok := db.Classes[name]; ok {
				add(def.Styles)
			}
		}
		add(b.Styles)
		return css.Declarations(decls)
	}

	newView := func(b *Block, index int) *view {
		v := &view{block: b, index: index, kind: kindLeaf}
		if b.Type != "space" {
			v.author = authorOf(b)
		}
		var face text.Face
		if v.author.Bold {
			face |= text.Bold
		}
		if v.author.Italic {
			face |= text.Italic
		}
		fontSize := size
		if v.author.FontSize > 0 {
			fontSize = v.author.FontSize
		}
		switch b.Type {
		case "space":
			v.kind = kindSpace
		case "composite":
			v.kind = kindComposite
			room := 4000.0
			if tight {
				room = common.TitleRoom(o)
			}
			v.label = labelOf(b.Label, fontSize, room, face)
			if v.label.Height > 0 {
				v.head = v.label.Height + 4
			}
			v.nw = v.label.Width + 2*gap
			v.nh = v.head + 2*gap
		case "block_arrow":
			v.kind = kindArrow
			v.label = labelOf(b.Label, fontSize, wrap, face)
			blank := true
			for _, line := range v.label.Lines {
				for _, part := range line {
					if part.Icon != "" || chars.Trim(part.Text) != "" {
						blank = false
					}
				}
			}
			v.mask = arrowMask(b.Directions)
			var s arrowSize
			if blank {
				s = sizeArrow(v.mask, 0, 0)
			} else {
				s = sizeArrow(v.mask, v.label.Width, v.label.Height)
			}
			v.nw, v.nh, v.dx, v.dy = s.w, s.h, s.dx, s.dy
		default:
			v.shape = "rect"
			if s, ok := shapes[b.Type]; ok {
				v.shape = s
			}
			v.label = labelOf(b.Label, fontSize, wrap, face)
			s := draw.ShapeSize(v.shape, v.label.Width, v.label.Height)
			v.nw = s.W
			// A cylinder's caps deepen as it widens, so it is sized for the deepest ones.
			v.nh = s.H
			if v.shape == "cyl" {
				v.nh = max(s.H, v.label.Height+40)
			}
			v.dy = s.DY
		}
		v.columns = -1
		if v.kind == kindComposite && b.Columns != nil {
			v.columns = *b.Columns
		}
		v.span = 1
		if v.kind != kindSpace && b.WidthInColumns != nil {
			v.span = *b.WidthInColumns
		}
		v.space = v.kind == kindSpace
		return v
	}

	root := newView(db.Root, 0)
	root.nw = 0
	root.nh = 0
	order := []*view{root}
	// parents holds the container of each view, by its index.
	parents := []*view{nil}
	byID := map[string]*view{}
	for i := 0; i < len(order); i++ {
		parent := order[i]
		if parent.kind != kindComposite {
			continue
		}
		for _, child := range parent.block.Children {
			v := newView(child, len(order))
			parent.children = append(parent.children, &v.cell)
			parents = append(parents, parent)
			order = append(order, v)
			if v.kind != kindSpace {
				byID[child.ID] = v
			}
		}
	}

	obstacles := []*view{}
	var titles []box
	// The layout keeps the widest a container has been, so each run starts from the widths it
	// needs.
	widths := make([]float64, len(order))
	for _, v := range order {
		widths[v.index] = v.nw
	}
	// place lays the blocks out in their gaps, as Pele's place does.
	place := func() (width, height float64, err error) {
		for _, v := range order {
			v.nw = widths[v.index]
		}
		width, height, err = layoutCells(&root.cell, gap)
		if err != nil {
			return 0, 0, err
		}
		obstacles = obstacles[:0]
		titles = titles[:0]
		for _, v := range order {
			switch {
			case v.kind == kindLeaf && fixedShape(v.shape):
				v.bw = v.nw
				v.bh = v.nh
			case v.kind == kindArrow:
				v.bw = v.nw
				if v.span > 1 {
					v.bw = v.w
				}
				v.bh = v.nh
			default:
				v.bw = v.w
				v.bh = v.h
				if v.kind == kindLeaf && wideShape(v.shape) {
					v.bh = v.nh
				}
			}
			if v.kind == kindLeaf && v.shape == "cyl" {
				v.dy = min(12, max(6, v.bw/14)) / 2
			}
			if v.kind == kindLeaf || v.kind == kindArrow {
				obstacles = append(obstacles, v)
			} else if v.head > 0 {
				titles = append(titles, box{x: v.x, y: v.y - v.h/2 + (v.head+gap)/2 + 2, bw: v.label.Width, bh: v.label.Height})
			}
		}
		return width, height, nil
	}
	contentW, contentH, err := place()
	if err != nil {
		return nil, err
	}

	// Edges between the same two blocks are spread apart instead of drawn on top of each
	// other.
	pairs := map[int]int{}
	labelled := map[int]bool{}
	var ends [][2]*view
	var drawn []*Block
	keyOf := func(s, t *view) int { return min(s.index, t.index)*len(order) + max(s.index, t.index) }
	for _, edge := range db.Edges {
		s, ok1 := byID[edge.Start]
		t, ok2 := byID[edge.End]
		if !ok1 || !ok2 {
			continue
		}
		key := keyOf(s, t)
		pairs[key]++
		if edge.Label != "" {
			labelled[key] = true
		}
		ends = append(ends, [2]*view{s, t})
		drawn = append(drawn, edge)
	}
	// check is the blocks an edge is checked against, nil where there would be too many
	// checks; place keeps obstacles up to date.
	checking := len(drawn)*len(obstacles) <= obstacleBudget
	check := func() []*view {
		if checking {
			return obstacles
		}
		return nil
	}
	routes := make([]route, 0, len(drawn))
	// stacks are the labelled edges between each pair of blocks joined by more than one, in
	// the order of the first.
	stacks := map[int][]int{}
	var stackKeys []int
	legs := make([][]float64, len(drawn))
	routeAll := func() {
		seen := map[int]int{}
		routes = routes[:0]
		clear(stacks)
		stackKeys = stackKeys[:0]
		for i := range drawn {
			s, t := ends[i][0], ends[i][1]
			key := keyOf(s, t)
			total := 1
			if n, ok := pairs[key]; ok {
				total = n
			}
			k := seen[key]
			seen[key] = k + 1
			label := labelOf(drawn[i].Label, edgeSize, wrapWidth, 0)
			var points []float64
			var lx, ly float64
			loop := 0
			if s == t {
				px, py, r, _, _ := loopArc(s, k+1)
				lx = px + r + 6 + label.Width/2
				ly = py - r/2
				loop = k + 1
			} else {
				step := 14.0
				if labelled[key] {
					step = jsmath.Round(edgeSize*1.5) + 2
				}
				points = routeEdge(s, t, float64((float64(k)-float64(total-1)/2)*step), check())
				n := len(points)
				// The label goes on the longer leg.
				firstLeg := math.Abs(points[2]-points[0]) + math.Abs(points[3]-points[1])
				lastLeg := math.Abs(points[n-2]-points[n-4]) + math.Abs(points[n-1]-points[n-3])
				at := 0
				if n == 6 && lastLeg > firstLeg {
					at = 2
				}
				spot := 0.5
				if label.Width > 0 {
					spot = labelSpot(points, at, label.Width+8, label.Height, s, t, check())
				}
				lx = points[at] + float64((points[at+2]-points[at])*spot)
				ly = points[at+1] + float64((points[at+3]-points[at+1])*spot)
				if label.Width > 0 && total > 1 {
					legs[i] = slices.Clone(points[at : at+4])
					if _, ok := stacks[key]; !ok {
						stackKeys = append(stackKeys, key)
					}
					stacks[key] = append(stacks[key], i)
				}
			}
			routes = append(routes, route{edge: drawn[i], points: points, loop: loop, label: label, lx: lx, ly: ly})
		}
	}

	// settle routes the edges and stacks the labels of each pair of blocks that cover each
	// other, as Pele's settle does. Where they do not fit and grow is set, it widens the gap
	// between the blocks, across the rows always and across the columns unless the layout is
	// tight, which keeps columns narrow to fit the host, and reports whether it did.
	settle := func(grow bool) bool {
		routeAll()
		base := map[gapsOf][]float64{}
		grown := false
		for _, key := range stackKeys {
			stack := stacks[key]
			if len(stack) < 2 {
				continue
			}
			s, t := ends[stack[0]][0], ends[stack[0]][1]
			short := stackLabels(stack, routes, legs, s, t, check(), titles)
			vertical := legs[stack[0]][0] == legs[stack[0]][2]
			if grow && short > 0 && (vertical || !tight) && widen(parents, s, t, vertical, math.Ceil(short), base) {
				grown = true
			}
		}
		return grown
	}
	// overlap is the area of the edge labels that cover each other, as Pele's overlap sums
	// it.
	overlap := func() float64 {
		area := 0.0
		for i := range routes {
			p := &routes[i]
			for j := i + 1; j < len(routes); j++ {
				q := &routes[j]
				pw := (p.label.Width + 8) / 2
				qw := (q.label.Width + 8) / 2
				w := min(p.lx+pw, q.lx+qw) - max(p.lx-pw, q.lx-qw)
				h := min(p.ly+p.label.Height/2, q.ly+q.label.Height/2) - max(p.ly-p.label.Height/2, q.ly-q.label.Height/2)
				if p.label.Width > 0 && q.label.Width > 0 && w > 0 && h > 0 {
					area += float64(w * h)
				}
			}
		}
		return area
	}
	var minX, minY, maxX, maxY float64
	grow := func(x, y, w, h float64) {
		if x-w/2 < minX {
			minX = x - w/2
		}
		if x+w/2 > maxX {
			maxX = x + w/2
		}
		if y-h/2 < minY {
			minY = y - h/2
		}
		if y+h/2 > maxY {
			maxY = y + h/2
		}
	}
	// bound finds the extent of the drawing and gives its width, as Pele's bound does.
	bound := func() float64 {
		minX, minY = 0, 0
		maxX, maxY = contentW, contentH
		for i := range routes {
			r := &routes[i]
			if r.loop > 0 {
				px, py, rr, _, _ := loopArc(ends[i][0], r.loop)
				grow(px, py, 2*rr+2, 2*rr+2)
			}
			if r.label.Width > 0 {
				grow(r.lx, r.ly, r.label.Width+8, r.label.Height)
			}
		}
		return maxX - minX
	}
	if settle(len(drawn)*len(drawn) <= obstacleBudget) {
		before := overlap()
		narrow := bound()
		// A container stretched by its row gives some of a new gap back, so the gap may take
		// a few rounds.
		grown := true
		for round := 0; round < 3 && grown; round++ {
			if contentW, contentH, err = place(); err != nil {
				return nil, err
			}
			grown = settle(round < 2)
		}
		// The drawing only grows if that leaves less of the labels covering each other, and
		// within the host's width.
		wide := bound()
		room := math.Inf(1)
		if o.MaxWidth > 0 {
			room = o.MaxWidth
		}
		if overlap() > before-1 || wide > narrow && math.Ceil(wide+float64(2*pad)) > room {
			for _, v := range order {
				v.rowGaps, v.colGaps = nil, nil
			}
			if contentW, contentH, err = place(); err != nil {
				return nil, err
			}
			settle(false)
		}
	}
	bound()

	titleRoom := 4000.0
	if tight {
		titleRoom = common.TitleRoom(o)
	}
	title := labelOf(db.Title, size, titleRoom, text.Bold)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	inner := max(maxX-minX, title.Width)
	ox := pad - minX + (inner-(maxX-minX))/2
	oy := pad + titleHeight - minY
	for _, v := range order {
		v.x += ox
		v.y += oy
	}
	sc := &scene.Scene{
		Type:     "block",
		Width:    math.Ceil(inner + 2*pad),
		Height:   math.Ceil(maxY - minY + titleHeight + 2*pad),
		FontSize: size,
	}

	var heading, clusters, nodes, edges, labels []scene.Item
	if !title.Empty() {
		heading = append(heading, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At: scene.Point{X: sc.Width / 2, Y: pad + title.Height/2},
		})
	}
	for _, v := range order[1:] {
		if v.kind == kindSpace {
			continue
		}
		b := v.block
		classes := common.ClassNames(strings.Join(b.Classes, " "))
		if v.kind == kindComposite {
			top := v.y - v.h/2
			box := scene.Rect{X: v.x - v.w/2, Y: top, W: v.w, H: v.h}
			it := scene.Item{
				Kind: scene.Group, Role: RoleComposite, ID: b.ID, Class: classes, Box: box,
				Parts:  []scene.Part{{Box: box, Rounded: true}},
				Style:  style.Style{Fill: style.SurfaceAlt, FillOpacity: style.N(0.5), Stroke: style.Border, Text: style.Muted},
				Author: v.author.Style,
			}
			if !v.label.Empty() {
				it.Label = &v.label
				it.At = scene.Point{X: v.x, Y: top + (v.head+gap)/2 + 2}
			}
			clusters = append(clusters, it)
			continue
		}
		it := scene.Item{
			Kind: scene.Node, Role: RoleBlock, Shape: scene.Shape(v.shape), ID: b.ID, Class: classes,
			Box:    scene.Rect{X: v.x - v.bw/2, Y: v.y - v.bh/2, W: v.bw, H: v.bh},
			Style:  style.Style{Fill: style.Surface, Stroke: style.Border},
			Author: v.author.Style,
			At:     scene.Point{X: v.x + v.dx, Y: v.y + v.dy},
		}
		if v.kind == kindArrow {
			it.Shape = "block-arrow"
			it.Parts = []scene.Part{{Path: drawArrow(v.mask, v.x, v.y, v.bw, v.bh)}}
			// Pele turns the corners of an arrow round.
			it.Style.Join = style.JoinRound
		} else {
			it.Parts = draw.Outline(v.shape, v.x, v.y, v.bw, v.bh)
		}
		if !v.label.Empty() {
			it.Label = &v.label
		}
		nodes = append(nodes, it)
	}

	for i := range routes {
		r := &routes[i]
		edge := r.edge
		start, end := scene.MarkerType(edge.ArrowTypeStart), scene.MarkerType(edge.ArrowTypeEnd)
		var path draw.EdgePath
		if r.loop > 0 {
			path = loopPath(ends[i][0], r.loop, start.Trim(), end.Trim())
		} else {
			for k := 0; k < len(r.points); k += 2 {
				r.points[k] += ox
				r.points[k+1] += oy
			}
			path = trace(r.points, start.Trim(), end.Trim())
		}
		// Pele draws the edges in a group with round caps.
		own := style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound}
		if edge.Thickness == "thick" {
			own.StrokeWidth = style.N(2.5)
		}
		if edge.Pattern == "dotted" {
			own.Dash = style.Dashes(3, 4)
		}
		it := scene.Item{
			Kind: scene.Edge, Role: RoleEdge, ID: edge.ID,
			Box:   path.Path.Bounds(),
			Parts: []scene.Part{{Path: path.Path, Paint: scene.StrokeOnly}},
			Ends:  path.Ends(start, end),
			Style: own,
		}
		for k := 0; k+1 < len(r.points); k += 2 {
			it.Route = append(it.Route, scene.Point{X: r.points[k], Y: r.points[k+1]})
		}
		if len(it.Route) == 0 {
			// A self-loop, which the layout gives no route: the corners its curve is drawn about.
			it.Route = it.Parts[0].Path.Corners()
		}
		edges = append(edges, it)
		if r.label.Width > 0 {
			x := r.lx + ox
			y := r.ly + oy
			label := &r.label
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: RoleEdgeLabel, ID: edge.ID,
				Box:   scene.Rect{X: x - (label.Width+8)/2, Y: y - label.Height/2, W: label.Width + 8, H: label.Height},
				Parts: []scene.Part{draw.LabelPatch(label, x, y)},
				Label: label,
				At:    scene.Point{X: x, Y: y},
				Style: style.Style{Fill: style.Bg, Stroke: style.None},
			})
		}
	}

	sc.Items = make([]scene.Item, 0, len(heading)+len(clusters)+len(edges)+len(nodes)+len(labels))
	for _, layer := range [][]scene.Item{heading, clusters, edges, nodes, labels} {
		sc.Items = append(sc.Items, layer...)
	}
	return sc, nil
}
