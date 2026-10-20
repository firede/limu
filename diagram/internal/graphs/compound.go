package graphs

import (
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
// where they fan out. A curve is taken as this many straight pieces.
const (
	fan         = 24
	curvePieces = 4
)

// routeCrossings is how often the routes of edges cross, taking each as straight lines between
// its points, as Pele's routeCrossings does.
func routeCrossings(nodes []CNode, edges []CEdge) int {
	var ax, ay, bx, by []float64
	var owner []int
	// 1 for a piece of the first part of a route, 2 for one of the last.
	var ends []int
	for ei := range edges {
		e := &edges[ei]
		if e.Src == e.Dst {
			continue
		}
		r := e.Route
		part := 0
		add := func(x0, y0, x1, y1 float64) {
			ends = append(ends, part)
			ax = append(ax, math.Min(x0, x1))
			bx = append(bx, math.Max(x0, x1))
			if x0 <= x1 {
				ay = append(ay, y0)
				by = append(by, y1)
			} else {
				ay = append(ay, y1)
				by = append(by, y0)
			}
			owner = append(owner, ei)
		}
		for k := 3; k < len(r); k += 3 {
			part = 0
			if k == 3 {
				part |= 1
			}
			if k == len(r)-3 {
				part |= 2
			}
			x0, y0, x1, y1 := r[k-3], r[k-2], r[k], r[k+1]
			if math.Abs(x1-x0) < 0.01 || math.Abs(y1-y0) < 0.01 {
				add(x0, y0, x1, y1)
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
				add(px, py, x, y)
				px, py = x, y
			}
		}
	}
	near := func(v int, x, y float64) bool {
		return math.Abs(x-nodes[v].X) < nodes[v].W/2+fan && math.Abs(y-nodes[v].Y) < nodes[v].H/2+fan
	}
	count := len(ax)
	ids := make([]int, count)
	for i := range ids {
		ids[i] = i
	}
	slices.SortStableFunc(ids, func(p, q int) int { return sign(or(ax[p]-ax[q], float64(p-q))) })
	left := make([]float64, count)
	right := make([]float64, count)
	low := make([]float64, count)
	high := make([]float64, count)
	for i, p := range ids {
		left[i] = ax[p]
		right[i] = bx[p]
		low[i] = math.Min(ay[p], by[p])
		high[i] = math.Max(ay[p], by[p])
	}
	total := 0
	for i := 0; i < count; i++ {
		p := ids[i]
		dx := bx[p] - ax[p]
		dy := by[p] - ay[p]
		for j := i + 1; j < count && left[j] <= right[i]; j++ {
			if high[j] < low[i] || low[j] > high[i] {
				continue
			}
			q := ids[j]
			if owner[p] == owner[q] {
				continue
			}
			d1 := float64(dx*(ay[q]-ay[p])) - float64(dy*(ax[q]-ax[p]))
			d2 := float64(dx*(by[q]-ay[p])) - float64(dy*(bx[q]-ax[p]))
			// A point that lies on the other line counts as on one side of it, so that a crossing
			// at the end of one piece is counted once, in one of the two pieces it joins.
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
			t := d3 / (d3 - d4)
			x := ax[p] + float64(t*dx)
			y := ay[p] + float64(t*dy)
			// Where two edges meet at a node, the parts of them that lead to it fan in.
			fans := func(v, pe, qe int) bool { return near(v, x, y) || (ends[p]&pe != 0 && ends[q]&qe != 0) }
			if ep.Src == eq.Src && fans(ep.Src, 1, 1) ||
				ep.Src == eq.Dst && fans(ep.Src, 1, 2) ||
				ep.Dst == eq.Src && fans(ep.Dst, 2, 1) ||
				ep.Dst == eq.Dst && fans(ep.Dst, 2, 2) {
				continue
			}
			total++
		}
	}
	return total
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

	// The flat layout: the groups are laid out one at a time, each on its own, so this is what
	// gives them an order for their nodes, and for the ports on the far sides of their borders,
	// that agrees with the rest of the drawing. A group's place is the average of its nodes'.
	// Only where every group runs the way the whole drawing does.
	flatCrossings := 0
	flat := len(order) > 1 && ports > 1 && len(edges) <= maxOrderedEdges && ports <= maxOrderedPorts
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
		flatCrossings = layered(lnodes, ledges, flatOpt).crossings
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
			size := layered(lnodes, ledges, opt)
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
	keyed = flat
	place()
	crossings := -1
	if flat {
		crossings = routeCrossings(nodes, edges)
		if float64(crossings) > float64(float64(flatCrossings)*flatSlack)+flatSpare {
			first := takeSnapshot(nodes, edges, order, total)
			keyed = false
			place()
			own := routeCrossings(nodes, edges)
			if own >= crossings {
				keyed = true
				total = first.restore(nodes, edges, order)
			} else {
				crossings = own
			}
		}
	}

	// The groups were laid out from the inside out, so the ports on a group's border are in an
	// order that only guesses where their edges go once they leave it. Where more than one edge
	// passes the far side of a group, the drawing is laid out again with those ports in the order
	// of where their edges went the first time, and kept if that crosses less.
	if ports > 1 && ports <= maxOrderedPorts && len(edges) <= maxOrderedEdges {
		again := false
		for _, l := range order {
			if l.id < 0 {
				continue
			}
			box := &nodes[l.id]
			vertical := l.dir == TB || l.dir == BT
			end := opposite(startSide(l.dir))
			count := 0
			for _, port := range l.ports {
				if port.side != end {
					continue
				}
				// The port's place in the route, and the point next to it outside the group.
				route := edges[port.e].Route
				bx := l.x + port.px
				by := l.y + port.py
				at := -1
				for k := 0; k < len(route) && (at < 0 || port.out); k += 3 {
					if math.Abs(route[k]-bx) < 0.01 && math.Abs(route[k+1]-by) < 0.01 {
						at = k
					}
				}
				next := at - 3
				if port.out {
					next = at + 3
				}
				if at < 0 || next < 0 || next >= len(route) {
					continue
				}
				if vertical {
					port.hint = route[next] - (box.X - box.W/2)
				} else {
					port.hint = route[next+1] - (box.Y - box.H/2)
				}
				count++
			}
			if count > 1 {
				again = true
			}
		}
		if again {
			before := crossings
			if crossings < 0 {
				before = routeCrossings(nodes, edges)
			}
			if before > 0 {
				first := takeSnapshot(nodes, edges, order, total)
				place()
				if routeCrossings(nodes, edges) >= before {
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
