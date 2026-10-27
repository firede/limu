package graphs

import (
	"github.com/firede/limu/internal/jsmath"
	"math"
	"slices"
)

// The compound layout is Pele's layout/compound.ts: graphs whose nodes may be nested in groups.

// Dir is the direction a diagram, or a group in it, flows in, as Pele's Dir is.
type Dir string

const (
	TB Dir = "TB" // top to bottom
	BT Dir = "BT" // bottom to top
	LR Dir = "LR" // left to right
	RL Dir = "RL" // right to left
)

// Direction is the direction a diagram names, or top to bottom for anything else, as Pele's
// direction is.
func Direction(dir string) Dir {
	switch d := Dir(dir); d {
	case BT, LR, RL:
		return d
	}
	return TB
}

// CNode is a node of the compound layout, or a group of them, as Pele's CNode is. Compound
// sets X and Y (its middle), and a group's W and H.
type CNode struct {
	W, H    float64
	Parent  int // the group it is in, or -1
	IsGroup bool
	// Dir is the direction a group flows in; "" is its parent's.
	Dir                     Dir
	PadX, PadTop, PadBottom float64
	MinW                    float64
	// Seq is the order of declaration; a group takes the smallest value among its members.
	Seq float64
	// Span is how much of a side its edges may spread along, measured across the flow; 0 keeps
	// them at its middle.
	Span float64
	X, Y float64
}

// CEdge is an edge of the compound layout, as Pele's CEdge is. Compound sets Route, and LabelX
// and LabelY for an edge with a label.
type CEdge struct {
	Src, Dst       int
	Minlen         float64
	LabelW, LabelH float64
	// Route is x, y, axis triples; axis is 0 where the edge runs vertically into the point, 1
	// horizontally.
	Route          []float64
	LabelX, LabelY float64
}

// CompoundResult is the size of a compound layout, as Pele's CompoundResult is.
type CompoundResult struct {
	Width, Height float64
}

// side is a side of a group's box, as Pele's Side is.
type side uint8

const (
	sideTop side = iota
	sideRight
	sideBottom
	sideLeft
)

// portEdge is Pele's PortEdge: an edge where it passes the border of a group, inner being what
// it comes from inside the group, and route its way from there to the border, or back. hint is
// where, across the flow, the edge goes on outside the group, from an earlier layout, or NaN.
type portEdge struct {
	e      int
	inner  int
	out    bool
	side   side
	hint   float64
	px, py float64
	route  []float64
}

// levelEdge is Pele's LevelEdge: an edge between two items of one level.
type levelEdge struct {
	e, a, b int
}

// level is Pele's Level: the root or a group, laid out on its own.
type level struct {
	id     int
	dir    Dir
	items  []int
	edges  []levelEdge
	ports  []*portEdge
	portOf map[int]*portEdge
	routes map[int][]float64
	x, y   float64
}

const maxPorts = 40000

// The largest drawings whose groups are given an order from a layout of the whole drawing, and
// are laid out again to bring the ports on their borders into a better order.
const (
	maxOrderedPorts = 400
	maxOrderedEdges = 500
)

// How much more the groups may cross, laid out in the order of the flat layout, than the flat
// layout did, before their own order is tried as well.
const (
	flatSlack = 1.5
	flatSpare = 4
)

// snapshot is what a layout leaves behind, to go back to when another try turns out worse, as
// Pele's Snapshot is.
type snapshot struct {
	nodes  []float64
	routes [][]float64
	labels []float64
	levels []float64
	total  CompoundResult
}

func takeSnapshot(nodes []CNode, edges []CEdge, levels []*level, total CompoundResult) snapshot {
	s := snapshot{total: total}
	for i := range nodes {
		c := &nodes[i]
		s.nodes = append(s.nodes, c.X, c.Y, c.W, c.H)
	}
	for i := range edges {
		e := &edges[i]
		s.routes = append(s.routes, e.Route)
		s.labels = append(s.labels, e.LabelX, e.LabelY)
	}
	for _, l := range levels {
		s.levels = append(s.levels, l.x, l.y)
		for _, port := range l.ports {
			s.levels = append(s.levels, port.px, port.py)
		}
	}
	return s
}

// restore puts back what s holds, as Pele's restore does, and returns the size it had.
func (s *snapshot) restore(nodes []CNode, edges []CEdge, levels []*level) CompoundResult {
	for i := range nodes {
		c := &nodes[i]
		c.X, c.Y, c.W, c.H = s.nodes[4*i], s.nodes[4*i+1], s.nodes[4*i+2], s.nodes[4*i+3]
	}
	for i := range edges {
		e := &edges[i]
		e.Route = s.routes[i]
		e.LabelX, e.LabelY = s.labels[2*i], s.labels[2*i+1]
	}
	k := 0
	for _, l := range levels {
		l.x, l.y = s.levels[k], s.levels[k+1]
		k += 2
		for _, port := range l.ports {
			port.px, port.py = s.levels[k], s.levels[k+1]
			k += 2
		}
	}
	return s.total
}

// Where two edges meet at a node, they are not counted as crossing within this much of it,
// where they fan out.
const fan = 24

// The straight pieces a curve is taken as: an S, and a rounded corner, which turns half as far.
// stepPieces is the most pieces one step of a route can make: a step at right angles, an S, or
// a straight run into a rounded corner.
const (
	curvePieces  = 3
	cornerPieces = 2
	stepPieces   = max(3, curvePieces, cornerPieces+1)
)

// handle is how far the handles of a rounded corner reach along its two legs, as the drawing has
// them. It is a variable, so that 1 - handle is worked out in float64, as JavaScript does.
var handle = 0.62

// routeStyle is how the drawing joins the points of a route, as Pele's Style is: with curves
// that round its corners, with steps at right angles, or with straight lines.
type routeStyle uint8

const (
	styleCurved routeStyle = iota
	styleStepped
	styleLinear
)

// styleOf is how a drawing laid out with opt joins the points of its routes, as Pele's styleOf
// says.
func styleOf(opt LayeredOptions) routeStyle {
	switch {
	case opt.Tracks:
		return styleStepped
	case opt.Straight:
		return styleLinear
	}
	return styleCurved
}

// The most pieces in a band that are put in order one at a time.
const maxInsert = 128

// routeCrossings is how often the routes of edges cross, joining their points as the drawing
// will, as Pele's routeCrossings does. Where two edges meet at one node, they are not counted as
// crossing near it, where they fan out, or where they join one straight run to it.
func routeCrossings(nodes []CNode, edges []CEdge, style routeStyle) int {
	size := 0
	for i := range edges {
		if e := &edges[i]; e.Src != e.Dst && len(e.Route) > 3 {
			size += (len(e.Route)/3 - 1) * stepPieces
		}
	}
	ax := make([]float64, size)
	ay := make([]float64, size)
	bx := make([]float64, size)
	by := make([]float64, size)
	low := make([]float64, size)
	high := make([]float64, size)
	owner := make([]int, size)
	// 1 for a piece of the first part of a route, 2 for one of the last.
	ends := make([]int, size)
	count := 0
	top := math.Inf(1)
	bottom := math.Inf(-1)
	// The ends of the last piece, to run it on where its route goes on straight the same way.
	fromX, fromY, toX, toY := math.NaN(), math.NaN(), math.NaN(), math.NaN()
	add := func(ei, part int, x0, y0, x1, y1 float64) {
		p := count - 1
		on := x0 == toX && y0 == toY && ((y1 == y0 && fromY == y0) || (x1 == x0 && fromX == x0))
		if on && owner[p] == ei && ends[p] == part && float64((x0-fromX)*(x1-x0))+float64((y0-fromY)*(y1-y0)) > 0 {
			count--
			x0, y0 = fromX, fromY
		}
		fromX, fromY, toX, toY = x0, y0, x1, y1
		ends[count] = part
		ax[count] = math.Min(x0, x1)
		bx[count] = math.Max(x0, x1)
		if x0 <= x1 {
			ay[count], by[count] = y0, y1
		} else {
			ay[count], by[count] = y1, y0
		}
		low[count] = math.Min(y0, y1)
		high[count] = math.Max(y0, y1)
		if low[count] < top {
			top = low[count]
		}
		if high[count] > bottom {
			bottom = high[count]
		}
		owner[count] = ei
		count++
	}
	for ei := range edges {
		e := &edges[ei]
		if e.Src == e.Dst {
			continue
		}
		r := e.Route
		aligned := func(k int) bool { return math.Abs(r[k]-r[k-3]) < 0.01 || math.Abs(r[k+1]-r[k-2]) < 0.01 }
		length := func(k int) float64 { return math.Abs(r[k]-r[k-3]) + math.Abs(r[k+1]-r[k-2]) }
		// Whether the route turns a corner at point k, which a smooth drawing rounds.
		turns := func(k int) bool {
			return style == styleCurved && k > 0 && k < len(r)-3 && aligned(k) && aligned(k+3) &&
				length(k) >= 0.01 && length(k+3) >= 0.01 && (math.Abs(r[k]-r[k-3]) < 0.01) != (math.Abs(r[k+3]-r[k]) < 0.01)
		}
		if len(r) < 6 {
			continue
		}
		atX, atY := r[0], r[1]
		for k := 3; k < len(r); k += 3 {
			part := 0
			if k == 3 {
				part |= 1
			}
			if k == len(r)-3 {
				part |= 2
			}
			x0, y0, x1, y1 := atX, atY, r[k], r[k+1]
			atX, atY = x1, y1
			if aligned(k) || style == styleLinear {
				if !turns(k) {
					add(ei, part, x0, y0, x1, y1)
					continue
				}
				// The turn takes all of a leg that ends at a node and half of one it shares with
				// the next turn.
				before := length(k)
				if turns(k - 3) {
					before = length(k) / 2
				}
				after := length(k + 3)
				if turns(k + 3) {
					after = length(k+3) / 2
				}
				ux := (x1 - r[k-3]) / length(k)
				uy := (y1 - r[k-2]) / length(k)
				vx := (r[k+3] - x1) / length(k+3)
				vy := (r[k+4] - y1) / length(k+3)
				sx := x1 - float64(ux*before)
				sy := y1 - float64(uy*before)
				add(ei, part, x0, y0, sx, sy)
				c1x := x1 - float64(ux*before*(1-handle))
				c1y := y1 - float64(uy*before*(1-handle))
				c2x := x1 + float64(vx*after*(1-handle))
				c2y := y1 + float64(vy*after*(1-handle))
				atX = x1 + float64(vx*after)
				atY = y1 + float64(vy*after)
				turn := part
				if k+3 == len(r)-3 {
					turn |= 2
				}
				px, py := sx, sy
				for i := 1; i <= cornerPieces; i++ {
					t := float64(i) / cornerPieces
					u := 1 - t
					x := float64(u*u*u*sx) + float64(3*u*u*t*c1x) + float64(3*u*t*t*c2x) + float64(t*t*t*atX)
					y := float64(u*u*u*sy) + float64(3*u*u*t*c1y) + float64(3*u*t*t*c2y) + float64(t*t*t*atY)
					add(ei, turn, px, py, x, y)
					px, py = x, y
				}
				continue
			}
			if style == styleStepped {
				// A step at right angles, halfway along the flow.
				down := r[k+2] == 0
				mx, my := (x0+x1)/2, y0
				nx, ny := mx, y1
				if down {
					mx, my = x0, (y0+y1)/2
					nx, ny = x1, my
				}
				add(ei, part, x0, y0, mx, my)
				add(ei, part, mx, my, nx, ny)
				add(ei, part, nx, ny, x1, y1)
				continue
			}
			// An S along the flow, as the drawing has it, in a few straight pieces.
			down := r[k+2] == 0
			c := Ease(y1-y0, x1-x0)
			if down {
				c = Ease(x1-x0, y1-y0)
			}
			px, py := x0, y0
			for i := 1; i <= curvePieces; i++ {
				t := float64(i) / curvePieces
				u := 1 - t
				across := float64(t*t) * (3 - float64(2*t))
				along := float64(3*c*t*u*u) + float64(3*(1-c)*t*t*u) + float64(t*t*t)
				fx, fy := along, across
				if down {
					fx, fy = across, along
				}
				x := x0 + float64((x1-x0)*fx)
				y := y0 + float64((y1-y0)*fy)
				add(ei, part, px, py, x, y)
				px, py = x, y
			}
		}
	}
	// Bands across the drawing, each swept from left to right on its own. A pair of pieces is
	// looked at in the band that the lower of their tops falls in.
	bands := 1
	scale := 0.0
	if bottom > top {
		bands = int(jsmath.Round(math.Sqrt(float64(count))))
		scale = float64(bands) / (bottom - top)
	}
	bandOf := func(y float64) int { return min(bands-1, int(math.Floor((y-top)*scale))) }
	first := make([]int, count)
	bandStart := make([]int, bands+1)
	for p := 0; p < count; p++ {
		first[p] = bandOf(low[p])
		for b, last := first[p], bandOf(high[p]); b <= last; b++ {
			bandStart[b+1]++
		}
	}
	for b := 0; b < bands; b++ {
		bandStart[b+1] += bandStart[b]
	}
	fill := slices.Clone(bandStart[:bands])
	inBand := make([]int, bandStart[bands])
	for p := 0; p < count; p++ {
		for b, last := first[p], bandOf(high[p]); b <= last; b++ {
			inBand[fill[b]] = p
			fill[b]++
		}
	}
	// The straight run each route leaves its first node by and the one it reaches its last node
	// by. Edges that meet at a node and join into one run to it merge there rather than cross.
	runs := make([]float64, len(edges)*8)
	lined := func(r []float64, a, b, c int) bool {
		return (math.Abs(r[a]-r[b]) < inLine && math.Abs(r[a]-r[c]) < inLine) ||
			(math.Abs(r[a+1]-r[b+1]) < inLine && math.Abs(r[a+1]-r[c+1]) < inLine)
	}
	for ei := range edges {
		r := edges[ei].Route
		if len(r) < 6 {
			continue
		}
		k := 3
		for k+3 < len(r) && lined(r, 0, k, k+3) {
			k += 3
		}
		last := len(r) - 3
		j := last - 3
		for j >= 3 && lined(r, last, j, j-3) {
			j -= 3
		}
		copy(runs[ei*8:], []float64{r[0], r[1], r[k], r[k+1], r[j], r[j+1], r[last], r[last+1]})
	}
	// Whether a point lies on the first (1) or last (2) run of an edge's route.
	onRun := func(ei, end int, x, y float64) bool {
		o := ei * 8
		if end != 1 {
			o += 4
		}
		x0 := runs[o]
		y0 := runs[o+1]
		dx := runs[o+2] - x0
		dy := runs[o+3] - y0
		length := float64(dx*dx) + float64(dy*dy)
		t := 0.0
		if length > 0 {
			t = math.Max(0, math.Min(1, (float64((x-x0)*dx)+float64((y-y0)*dy))/length))
		}
		return jsmath.Hypot(x-x0-float64(t*dx), y-y0-float64(t*dy)) < inLine
	}
	total := 0
	for band := 0; band < bands; band++ {
		bandEnd := bandStart[band+1]
		// In order of the pieces' left ends, and of where they were added among those level.
		if bandEnd-bandStart[band] > maxInsert {
			slices.SortStableFunc(inBand[bandStart[band]:bandEnd], func(p, q int) int { return sign(or(ax[p]-ax[q], float64(p-q))) })
		} else {
			for i := bandStart[band] + 1; i < bandEnd; i++ {
				p := inBand[i]
				j := i
				for ; j > bandStart[band] && ax[inBand[j-1]] > ax[p]; j-- {
					inBand[j] = inBand[j-1]
				}
				inBand[j] = p
			}
		}
		for s := bandStart[band]; s < bandEnd; s++ {
			p := inBand[s]
			dx := bx[p] - ax[p]
			dy := by[p] - ay[p]
			here := first[p] == band
			for t := s + 1; t < bandEnd; t++ {
				q := inBand[t]
				if ax[q] > bx[p] {
					break
				}
				if high[q] < low[p] || low[q] > high[p] || owner[p] == owner[q] || (!here && first[q] != band) {
					continue
				}
				d1 := float64(dx*(ay[q]-ay[p])) - float64(dy*(ax[q]-ax[p]))
				d2 := float64(dx*(by[q]-ay[p])) - float64(dy*(bx[q]-ax[p]))
				// A point that lies on the other line counts as on one side of it, so that a
				// crossing at the end of one piece is counted once, in one of the two pieces it
				// joins.
				if (d1 > 0) == (d2 > 0) {
					continue
				}
				ex := bx[q] - ax[q]
				ey := by[q] - ay[q]
				d3 := float64(ex*(ay[p]-ay[q])) - float64(ey*(ax[p]-ax[q]))
				d4 := float64(ex*(by[p]-ay[q])) - float64(ey*(bx[p]-ax[q]))
				if (d3 > 0) == (d4 > 0) {
					continue
				}
				ep := &edges[owner[p]]
				eq := &edges[owner[q]]
				at := d3 / (d3 - d4)
				x := ax[p] + float64(at*dx)
				y := ay[p] + float64(at*dy)
				// Where two edges meet at a node, the parts of them that lead to it fan in.
				fans := func(v, pe, qe int) bool {
					return (ends[p]&pe != 0 && ends[q]&qe != 0) ||
						(math.Abs(x-nodes[v].X) < nodes[v].W/2+fan && math.Abs(y-nodes[v].Y) < nodes[v].H/2+fan) ||
						(onRun(owner[p], pe, x, y) && onRun(owner[q], qe, x, y))
				}
				if ep.Src == eq.Src && fans(ep.Src, 1, 1) ||
					ep.Src == eq.Dst && fans(ep.Src, 1, 2) ||
					ep.Dst == eq.Src && fans(ep.Dst, 2, 1) ||
					ep.Dst == eq.Dst && fans(ep.Dst, 2, 2) {
					continue
				}
				total++
			}
		}
	}
	return total
}

// InheritCrossedDirections gives each group that an edge crosses its parent's direction, as
// Pele's inheritCrossedDirections does, to match Mermaid. An edge crosses the groups that hold
// one of its ends and not the other. ends holds the nodes each edge runs between, -1 for an
// end that is no node.
func InheritCrossedDirections(nodes []CNode, ends [][2]int) {
	ancestors := func(i int) map[int]bool {
		set := map[int]bool{}
		for p := nodes[i].Parent; p >= 0 && !set[p]; p = nodes[p].Parent {
			set[p] = true
		}
		return set
	}
	for _, e := range ends {
		src, dst := e[0], e[1]
		if src < 0 || dst < 0 || src == dst {
			continue
		}
		a := ancestors(src)
		b := ancestors(dst)
		for g := range a {
			if !b[g] {
				nodes[g].Dir = ""
			}
		}
		for g := range b {
			if !a[g] {
				nodes[g].Dir = ""
			}
		}
	}
}

// Shift moves a laid out graph by ox, oy, as Pele's shiftLayout does.
func Shift(nodes []CNode, edges []CEdge, ox, oy float64) {
	for i := range nodes {
		nodes[i].X += ox
		nodes[i].Y += oy
	}
	for i := range edges {
		e := &edges[i]
		for k := 0; k < len(e.Route); k += 3 {
			e.Route[k] += ox
			e.Route[k+1] += oy
		}
		e.LabelX += ox
		e.LabelY += oy
	}
}

// NewCNode is a node of the given size in the given group, or -1 for none, as Pele's cnode
// makes.
func NewCNode(w, h float64, parent int) CNode {
	return CNode{W: w, H: h, Parent: parent}
}

func startSide(dir Dir) side {
	switch dir {
	case TB:
		return sideTop
	case BT:
		return sideBottom
	case LR:
		return sideLeft
	}
	return sideRight
}

func opposite(s side) side {
	return (s + 2) & 3
}

// Compound lays out a graph whose nodes may be nested in groups, as Pele's compoundLayout does.
// Each group is laid out on its own, in its own direction, and then placed in its parent as a
// single box. An edge that crosses a group's border passes through a port on the side facing
// its other end.
func Compound(nodes []CNode, edges []CEdge, rootDir Dir, opt LayeredOptions) CompoundResult {
	n := len(nodes)
	levelOf := make(map[int]*level)
	var order []*level
	makeLevel := func(id int, dir Dir) *level {
		l := &level{id: id, dir: dir, portOf: make(map[int]*portEdge), routes: make(map[int][]float64)}
		levelOf[id] = l
		return l
	}
	root := makeLevel(-1, rootDir)

	children := make([][]int, n)
	var rootItems []int
	for i := 0; i < n; i++ {
		p := nodes[i].Parent
		if p >= 0 && p < n && p != i && nodes[p].IsGroup {
			children[p] = append(children[p], i)
		} else {
			nodes[i].Parent = -1
			rootItems = append(rootItems, i)
		}
	}
	root.items = rootItems
	order = append(order, root)
	depth := make([]int, n)
	placed := make([]bool, n)
	descend := func(from int) {
		for q := from; q < len(order); q++ {
			l := order[q]
			for _, item := range l.items {
				placed[item] = true
				if l.id < 0 {
					depth[item] = 0
				} else {
					depth[item] = depth[l.id] + 1
				}
				if nodes[item].IsGroup {
					dir := nodes[item].Dir
					if dir == "" {
						dir = l.dir
					}
					child := makeLevel(item, dir)
					child.items = children[item]
					order = append(order, child)
				}
			}
		}
	}
	descend(0)
	// Groups that contain each other are unreachable from the root; lift them out.
	for i := 0; i < n; i++ {
		if placed[i] {
			continue
		}
		former := nodes[i].Parent
		children[former] = slices.DeleteFunc(slices.Clone(children[former]), func(c int) bool { return c == i })
		nodes[i].Parent = -1
		root.items = append(root.items, i)
		placed[i] = true
		depth[i] = 0
		if nodes[i].IsGroup {
			from := len(order)
			dir := nodes[i].Dir
			if dir == "" {
				dir = rootDir
			}
			child := makeLevel(i, dir)
			child.items = children[i]
			order = append(order, child)
			descend(from)
		}
	}

	for q := len(order) - 1; q > 0; q-- {
		l := order[q]
		seq := math.Inf(1)
		for _, item := range l.items {
			if nodes[item].Seq < seq {
				seq = nodes[item].Seq
			}
		}
		if !math.IsInf(seq, 1) {
			nodes[l.id].Seq = seq
		}
	}
	for _, l := range order {
		slices.SortStableFunc(l.items, func(a, b int) int {
			return sign(or(nodes[a].Seq-nodes[b].Seq, float64(a-b)))
		})
	}

	// Every group border an edge crosses costs a port. Past the budget, an edge is drawn between
	// the groups that hold its ends, at the level they share.
	ports := 0
	for i := range edges {
		e := &edges[i]
		a := e.Src
		b := e.Dst
		crossed := 0
		for ; depth[a] > depth[b]; crossed++ {
			a = nodes[a].Parent
		}
		for ; depth[b] > depth[a]; crossed++ {
			b = nodes[b].Parent
		}
		for ; a != b && nodes[a].Parent != nodes[b].Parent; crossed += 2 {
			a = nodes[a].Parent
			b = nodes[b].Parent
		}
		if a == b {
			continue
		}
		if ports+crossed > maxPorts {
			e.Src = a
			e.Dst = b
		} else {
			ports += crossed
		}
	}

	var loops, stray []int
	for ei := range edges {
		e := &edges[ei]
		e.Route = []float64{}
		if e.Src == e.Dst {
			loops = append(loops, ei)
			continue
		}
		a := e.Src
		b := e.Dst
		var aPath, bPath []int
		for depth[a] > depth[b] {
			aPath = append(aPath, a)
			a = nodes[a].Parent
		}
		for depth[b] > depth[a] {
			bPath = append(bPath, b)
			b = nodes[b].Parent
		}
		if a == b {
			stray = append(stray, ei)
			continue
		}
		for nodes[a].Parent != nodes[b].Parent {
			aPath = append(aPath, a)
			bPath = append(bPath, b)
			a = nodes[a].Parent
			b = nodes[b].Parent
		}
		levelOf[nodes[a].Parent].edges = append(levelOf[nodes[a].Parent].edges, levelEdge{e: ei, a: a, b: b})
		for _, way := range [2]struct {
			path []int
			out  bool
		}{{aPath, true}, {bPath, false}} {
			for _, inner := range way.path {
				l := levelOf[nodes[inner].Parent]
				port := &portEdge{e: ei, inner: inner, out: way.out, side: sideTop, hint: math.NaN(), route: []float64{}}
				l.ports = append(l.ports, port)
				l.portOf[portKey(ei, way.out)] = port
			}
		}
	}

	local := make([]int, n)
	endpointBelow := func(item int, e *CEdge, out bool) bool {
		if out {
			return e.Src != item
		}
		return e.Dst != item
	}

	// Where each node sits across the flow in a layout of the whole drawing as one flat graph, in
	// which the nodes of each group stay side by side, and with keyed, the layouts of the levels
	// start from it. farHint gives a port on the far side of a group the order of its edge there.
	key := make([]float64, n)
	for i := range key {
		key[i] = math.NaN()
	}
	keyed := false
	var farHint func(l *level, port *portEdge) float64

	build := func(l *level, sized bool) ([]lNode, []lEdge, []int) {
		vertical := l.dir == TB || l.dir == BT
		start := startSide(l.dir)
		lnodes := make([]lNode, 0, len(l.items)+len(l.ports))
		ledges := make([]lEdge, 0, len(l.edges)+len(l.ports))
		for i, item := range l.items {
			local[item] = i
			node := &nodes[item]
			var ln lNode
			if sized {
				if vertical {
					ln = newLNode(node.W, node.H, kindNode)
				} else {
					ln = newLNode(node.H, node.W, kindNode)
				}
			} else {
				ln = newLNode(0, 0, kindNode)
			}
			if node.IsGroup {
				ln.Span = math.Min(node.Span, ln.W-float64(2*node.PadX))
			} else {
				ln.Span = node.Span
			}
			if keyed {
				ln.Key = key[item]
			}
			lnodes = append(lnodes, ln)
		}
		held := func(item, ei int, out bool) bool {
			return nodes[item].IsGroup && endpointBelow(item, &edges[ei], out)
		}
		offset := func(item, ei int, out bool) float64 {
			if !sized || !nodes[item].IsGroup || !endpointBelow(item, &edges[ei], out) {
				return 0
			}
			port := levelOf[item].portOf[portKey(ei, out)]
			if vertical {
				return port.px - nodes[item].W/2
			}
			return port.py - nodes[item].H/2
		}
		for _, le := range l.edges {
			e := &edges[le.e]
			edge := newLEdge(local[le.a], local[le.b], e.Minlen, 0, 0)
			if sized {
				if vertical {
					edge.LabelW, edge.LabelH = e.LabelW, e.LabelH
				} else {
					edge.LabelW, edge.LabelH = e.LabelH, e.LabelW
				}
				edge.TailDx = offset(le.a, le.e, true)
				edge.HeadDx = offset(le.b, le.e, false)
				edge.TailFixed = held(le.a, le.e, true)
				edge.HeadFixed = held(le.b, le.e, false)
			}
			ledges = append(ledges, edge)
		}
		dummies := filled(len(l.ports), -1)
		for i, port := range l.ports {
			if port.side == start || port.side == opposite(start) {
				d := len(lnodes)
				kind := kindEndPort
				if port.side == start {
					kind = kindStartPort
				}
				ln := newLNode(0, 0, kind)
				if keyed {
					if port.out {
						ln.Key = key[edges[port.e].Dst]
					} else {
						ln.Key = key[edges[port.e].Src]
					}
				}
				ln.Hint = port.hint
				if math.IsNaN(port.hint) && keyed && farHint != nil {
					ln.Hint = farHint(l, port)
				}
				lnodes = append(lnodes, ln)
				dummies[i] = d
				var edge lEdge
				if port.out {
					edge = newLEdge(local[port.inner], d, 1, 0, 0)
					edge.TailDx = offset(port.inner, port.e, true)
					edge.TailFixed = held(port.inner, port.e, true)
				} else {
					edge = newLEdge(d, local[port.inner], 1, 0, 0)
					edge.HeadDx = offset(port.inner, port.e, false)
					edge.HeadFixed = held(port.inner, port.e, false)
				}
				ledges = append(ledges, edge)
			} else if port.side == sideTop || port.side == sideLeft {
				lnodes[local[port.inner]].Pin = -1
			} else {
				lnodes[local[port.inner]].Pin = 1
			}
		}
		return lnodes, ledges, dummies
	}

	// Whether the drawing is small enough to be laid out more than once.
	ordered := ports > 1 && ports <= maxOrderedPorts && len(edges) <= maxOrderedEdges

	// The flat layout: the groups are laid out one at a time, each on its own, so this is what
	// gives them an order for their nodes, and for the ports on the far sides of their borders,
	// that agrees with the rest of the drawing. A group's place is the average of its nodes'.
	// Only where every group runs the way the whole drawing does.
	flatCrossings := 0
	flat := len(order) > 1 && ordered
	for _, l := range order {
		if l.dir != rootDir {
			flat = false
		}
	}
	if flat {
		vertical := rootDir == TB || rootDir == BT
		leaf := filled(n, -1)
		group := filled(n, -1)
		var lnodes []lNode
		var of, parent []int
		for i := 0; i < n; i++ {
			if nodes[i].IsGroup {
				group[i] = len(parent)
				parent = append(parent, -1)
			}
		}
		for i := 0; i < n; i++ {
			p := nodes[i].Parent
			up := -1
			if p >= 0 {
				up = group[p]
			}
			if nodes[i].IsGroup {
				parent[group[i]] = up
				continue
			}
			leaf[i] = len(lnodes)
			if vertical {
				lnodes = append(lnodes, newLNode(nodes[i].W, nodes[i].H, kindNode))
			} else {
				lnodes = append(lnodes, newLNode(nodes[i].H, nodes[i].W, kindNode))
			}
			of = append(of, up)
		}
		var ledges []lEdge
		ledgeOf := filled(len(edges), -1)
		for ei := range edges {
			e := &edges[ei]
			if e.Src == e.Dst || leaf[e.Src] < 0 || leaf[e.Dst] < 0 {
				continue
			}
			ledgeOf[ei] = len(ledges)
			if vertical {
				ledges = append(ledges, newLEdge(leaf[e.Src], leaf[e.Dst], e.Minlen, e.LabelW, e.LabelH))
			} else {
				ledges = append(ledges, newLEdge(leaf[e.Src], leaf[e.Dst], e.Minlen, e.LabelH, e.LabelW))
			}
		}
		flatOpt := opt
		flatOpt.groups = &nodeGroups{of: of, parent: parent}
		flatOpt.orderOnly = true
		flatLayout := layered(lnodes, ledges, flatOpt)
		// The fewest crossings the drawing has been seen to need as one flat graph: in the order
		// of the flat layout, or in the order it was written, where groups are not kept together.
		flatCrossings = min(flatLayout.crossings, flatLayout.declared)
		sum := make([]float64, n)
		count := make([]int, n)
		// The last rank each group reaches.
		last := filled(n, -1)
		for i := 0; i < n; i++ {
			if leaf[i] < 0 {
				continue
			}
			key[i] = lnodes[leaf[i]].X
			for p := nodes[i].Parent; p >= 0; p = nodes[p].Parent {
				sum[p] += key[i]
				count[p]++
				last[p] = max(last[p], lnodes[leaf[i]].Rank)
			}
		}
		for i := 0; i < n; i++ {
			if count[i] > 0 {
				key[i] = sum[i] / float64(count[i])
			}
		}
		// The ports that edges leave a group by, on its far side, go in the order of where their
		// edges pass the group's last rank in the flat layout, and those that pass it at one
		// place, in the order of where they pass the rank after it. The ports on the near side,
		// and those of edges that come back in, are left to the group to order.
		farHint = func(l *level, port *portEdge) float64 {
			le := ledgeOf[port.e]
			if le < 0 || l.id < 0 || !port.out || port.side != opposite(startSide(l.dir)) {
				return math.NaN()
			}
			points := ledges[le].Points
			inside := key[edges[port.e].Src]
			outside := 0.0
			for k := 0; k < len(points); k += 2 {
				if points[k] == float64(last[l.id]) {
					inside = points[k+1]
				}
				if points[k] == float64(last[l.id]+1) {
					outside = points[k+1]
				}
			}
			return inside + float64(outside*1e-6)
		}
	}

	// What each level's layout was given and gave back, every time it was laid out, where the
	// drawing may be laid out again. A level given the same again, as it is where neither its
	// ports nor its groups changed, comes out the same.
	type laidOut struct {
		given  []float64
		lnodes []lNode
		ledges []lEdge
		size   layeredResult
	}
	laid := make(map[int][]*laidOut)
	layOut := func(l *level, lnodes []lNode, ledges []lEdge) ([]lNode, []lEdge, layeredResult) {
		if !ordered {
			return lnodes, ledges, layered(lnodes, ledges, opt)
		}
		var given []float64
		for i := range lnodes {
			ln := &lnodes[i]
			given = append(given, ln.W, ln.H, float64(ln.Kind), ln.Pin, ln.Hint, ln.Key, ln.Span)
		}
		for i := range ledges {
			le := &ledges[i]
			given = append(given, float64(le.Tail), float64(le.Head), le.Minlen, le.LabelW, le.LabelH, le.TailDx, le.HeadDx,
				boolNumber(le.TailFixed), boolNumber(le.HeadFixed))
		}
		for _, p := range laid[l.id] {
			if slices.EqualFunc(p.given, given, sameValue) {
				return p.lnodes, p.ledges, p.size
			}
		}
		done := &laidOut{given, lnodes, ledges, layered(lnodes, ledges, opt)}
		laid[l.id] = append(laid[l.id], done)
		return done.lnodes, done.ledges, done.size
	}

	var total CompoundResult
	place := func() {
		// Top-down: rank each level to learn which side of a group every crossing edge leaves from.
		for _, l := range order {
			for _, port := range l.ports {
				if nodes[port.inner].IsGroup && endpointBelow(port.inner, &edges[port.e], port.out) {
					levelOf[port.inner].portOf[portKey(port.e, port.out)].side = port.side
				}
			}
			if len(l.edges) == 0 {
				continue
			}
			needed := false
			for _, le := range l.edges {
				e := &edges[le.e]
				if (nodes[le.a].IsGroup && e.Src != le.a) || (nodes[le.b].IsGroup && e.Dst != le.b) {
					needed = true
				}
			}
			if !needed {
				continue
			}
			lnodes, ledges, _ := build(l, false)
			orient(lnodes, ledges)
			rank(lnodes, ledges, 1)
			start := startSide(l.dir)
			for i, le := range l.edges {
				e := &edges[le.e]
				aAbove := !ledges[i].Reversed
				if nodes[le.a].IsGroup && e.Src != le.a {
					s := start
					if aAbove {
						s = opposite(start)
					}
					levelOf[le.a].portOf[portKey(le.e, true)].side = s
				}
				if nodes[le.b].IsGroup && e.Dst != le.b {
					s := opposite(start)
					if aAbove {
						s = start
					}
					levelOf[le.b].portOf[portKey(le.e, false)].side = s
				}
			}
		}

		// Bottom-up: lay out each level now that the sizes of its groups are known.
		for q := len(order) - 1; q >= 0; q-- {
			l := order[q]
			var group *CNode
			if l.id >= 0 {
				group = &nodes[l.id]
			}
			vertical := l.dir == TB || l.dir == BT
			flip := l.dir == BT || l.dir == RL
			lnodes, ledges, dummies := build(l, true)
			lnodes, ledges, size := layOut(l, lnodes, ledges)
			contentW, contentH := size.Height, size.Width
			if vertical {
				contentW, contentH = size.Width, size.Height
			}
			var padX, padTop, padBottom, minW float64
			if group != nil {
				padX, padTop, padBottom, minW = group.PadX, group.PadTop, group.PadBottom, group.MinW
			}
			boxW := math.Max(contentW+float64(2*padX), minW)
			boxH := contentH + padTop + padBottom
			offX := (boxW - contentW) / 2
			offY := padTop
			tx := func(lx, ly float64) float64 {
				if vertical {
					return offX + lx
				}
				if flip {
					return offX + (size.Height - ly)
				}
				return offX + ly
			}
			ty := func(lx, ly float64) float64 {
				if !vertical {
					return offY + lx
				}
				if flip {
					return offY + (size.Height - ly)
				}
				return offY + ly
			}
			axis := 1.0
			if vertical {
				axis = 0
			}
			convert := func(points []float64) []float64 {
				out := make([]float64, 0, len(points)/2*3+3)
				for i := 0; i < len(points); i += 2 {
					out = append(out, tx(points[i], points[i+1]), ty(points[i], points[i+1]), axis)
				}
				return out
			}

			for i, item := range l.items {
				nodes[item].X = tx(lnodes[i].X, lnodes[i].Y)
				nodes[item].Y = ty(lnodes[i].X, lnodes[i].Y)
			}
			for i, le := range l.edges {
				edge := &ledges[i]
				l.routes[le.e] = convert(edge.Points)
				e := &edges[le.e]
				if e.LabelW > 0 {
					e.LabelX = tx(edge.LabelX, edge.LabelY)
					e.LabelY = ty(edge.LabelX, edge.LabelY)
				}
			}
			k := len(l.edges)
			for i, port := range l.ports {
				item := &nodes[port.inner]
				if dummies[i] >= 0 {
					route := convert(ledges[k].Points)
					k++
					at := 0
					if port.out {
						at = len(route) - 3
					}
					bx := route[at]
					by := route[at+1]
					switch port.side {
					case sideTop:
						by = 0
					case sideBottom:
						by = boxH
					case sideLeft:
						bx = 0
					default:
						bx = boxW
					}
					if port.out {
						route = append(route, bx, by, axis)
					} else {
						route = append([]float64{bx, by, axis}, route...)
					}
					port.px = bx
					port.py = by
					port.route = route
					continue
				}
				ix := item.X
				iy := item.Y
				if item.IsGroup && endpointBelow(port.inner, &edges[port.e], port.out) {
					inner := levelOf[port.inner].portOf[portKey(port.e, port.out)]
					ix += inner.px - item.W/2
					iy += inner.py - item.H/2
				} else if port.side == sideTop {
					iy -= item.H / 2
				} else if port.side == sideBottom {
					iy += item.H / 2
				} else if port.side == sideLeft {
					ix -= item.W / 2
				} else {
					ix += item.W / 2
				}
				bx := ix
				by := iy
				switch port.side {
				case sideTop:
					by = 0
				case sideBottom:
					by = boxH
				case sideLeft:
					bx = 0
				default:
					bx = boxW
				}
				a := 1.0
				if port.side == sideTop || port.side == sideBottom {
					a = 0
				}
				port.px = bx
				port.py = by
				if port.out {
					port.route = []float64{ix, iy, a, bx, by, a}
				} else {
					port.route = []float64{bx, by, a, ix, iy, a}
				}
			}

			if group != nil {
				group.W = boxW
				group.H = boxH
			} else {
				total = CompoundResult{Width: boxW, Height: boxH}
			}
		}

		// Top-down again: turn positions inside each box into diagram coordinates.
		for _, l := range order {
			ox := l.x
			oy := l.y
			for _, item := range l.items {
				node := &nodes[item]
				node.X += ox
				node.Y += oy
				if node.IsGroup {
					child := levelOf[item]
					child.x = node.X - node.W/2
					child.y = node.Y - node.H/2
				}
			}
		}

		appendPart := func(route, part []float64, ox, oy float64) []float64 {
			for i := 0; i < len(part); i += 3 {
				x := part[i] + ox
				y := part[i+1] + oy
				length := len(route)
				if length >= 3 && math.Abs(route[length-3]-x) < 0.01 && math.Abs(route[length-2]-y) < 0.01 {
					continue
				}
				route = append(route, x, y, part[i+2])
			}
			return route
		}

		var outward, inward []*level
		for _, l := range order {
			for _, le := range l.edges {
				e := &edges[le.e]
				route := []float64{}
				outward = outward[:0]
				for item := e.Src; nodes[item].Parent != l.id; item = nodes[item].Parent {
					outward = append(outward, levelOf[nodes[item].Parent])
				}
				for _, lv := range outward {
					route = appendPart(route, lv.portOf[portKey(le.e, true)].route, lv.x, lv.y)
				}
				route = appendPart(route, l.routes[le.e], l.x, l.y)
				inward = inward[:0]
				for item := e.Dst; nodes[item].Parent != l.id; item = nodes[item].Parent {
					inward = append(inward, levelOf[nodes[item].Parent])
				}
				for i := len(inward) - 1; i >= 0; i-- {
					route = appendPart(route, inward[i].portOf[portKey(le.e, false)].route, inward[i].x, inward[i].y)
				}
				e.Route = route
				if e.LabelW > 0 {
					e.LabelX += l.x
					e.LabelY += l.y
				}
			}
		}
	}
	// The groups are laid out in the order of the flat layout. Where that crosses much more than
	// the flat layout did, they are laid out again in their own order too, and whichever crosses
	// less is kept: the flat layout does not see the groups as the boxes they are drawn as, and
	// now and then its order does not suit them.
	style := styleOf(opt)
	keyed = flat
	place()
	crossings := -1
	if flat {
		crossings = routeCrossings(nodes, edges, style)
		if float64(crossings) > float64(float64(flatCrossings)*flatSlack)+flatSpare {
			first := takeSnapshot(nodes, edges, order, total)
			keyed = false
			place()
			own := routeCrossings(nodes, edges, style)
			if own >= crossings {
				keyed = true
				total = first.restore(nodes, edges, order)
			} else {
				crossings = own
			}
		}
	}

	// The groups were laid out from the inside out, so the ports on a group's border are in an
	// order that only guesses where their edges go once they leave it. Where the ports on the far
	// side of a group are not in the order of where their edges went, the drawing is laid out
	// again with them in that order, and kept if that crosses less. Not where the groups kept
	// their own order instead, where it gains little for the time it takes.
	if keyed == flat && ordered {
		again := false
		for _, l := range order {
			if l.id < 0 {
				continue
			}
			box := &nodes[l.id]
			vertical := l.dir == TB || l.dir == BT
			end := opposite(startSide(l.dir))
			// Where each port is across the flow, and where its edge goes.
			var seen [][2]float64
			for _, port := range l.ports {
				if port.side != end {
					continue
				}
				// The port's place in the route, and the first point after it, outside the group,
				// that is not in line with it: where the edge turns to, past a straight run out of
				// the port.
				route := edges[port.e].Route
				bx := l.x + port.px
				by := l.y + port.py
				at := -1
				for k := 0; k < len(route) && (at < 0 || port.out); k += 3 {
					if math.Abs(route[k]-bx) < 0.01 && math.Abs(route[k+1]-by) < 0.01 {
						at = k
					}
				}
				if at < 0 {
					continue
				}
				a, step := 1, -3
				if vertical {
					a = 0
				}
				if port.out {
					step = 3
				}
				next := at + step
				for next >= 0 && next < len(route) && math.Abs(route[next+a]-route[at+a]) < inLine {
					next += step
				}
				if next < 0 || next >= len(route) {
					continue
				}
				if vertical {
					port.hint = route[next+a] - (box.X - box.W/2)
				} else {
					port.hint = route[next+a] - (box.Y - box.H/2)
				}
				if vertical {
					seen = append(seen, [2]float64{port.px, port.hint})
				} else {
					seen = append(seen, [2]float64{port.py, port.hint})
				}
			}
			slices.SortStableFunc(seen, func(p, q [2]float64) int { return sign(p[0] - q[0]) })
			for i := 1; i < len(seen); i++ {
				if seen[i-1][1] > seen[i][1] {
					again = true
				}
			}
		}
		if again {
			before := crossings
			if crossings < 0 {
				before = routeCrossings(nodes, edges, style)
			}
			if before > 0 {
				first := takeSnapshot(nodes, edges, order, total)
				place()
				if routeCrossings(nodes, edges, style) >= before {
					total = first.restore(nodes, edges, order)
				}
			}
		}
	}

	for _, ei := range stray {
		e := &edges[ei]
		a := &nodes[e.Src]
		b := &nodes[e.Dst]
		e.Route = []float64{a.X, a.Y, 0, b.X, b.Y, 0}
		e.LabelX = (a.X + b.X) / 2
		e.LabelY = (a.Y + b.Y) / 2
	}
	for _, ei := range loops {
		e := &edges[ei]
		node := &nodes[e.Src]
		e.LabelX = node.X + node.W/2 + 28 + e.LabelW/2
		e.LabelY = node.Y
	}

	return total
}

// portKey is how Pele keys the ports of a level: by edge, the way out before the way in.
func portKey(e int, out bool) int {
	if out {
		return e * 2
	}
	return e*2 + 1
}

// boolNumber is what JavaScript's unary + makes of a boolean.
func boolNumber(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// sameValue is JavaScript's Object.is for numbers: NaN is itself, and 0 is not -0.
func sameValue(a, b float64) bool {
	if a != a {
		return b != b
	}
	return a == b && math.Signbit(a) == math.Signbit(b)
}
