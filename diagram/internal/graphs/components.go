package graphs

import "math"

// WrapComponents moves the parts of a laid out graph that nothing joins into rows no wider
// than width where it can, a gap apart, without changing the layout inside a part, as Pele's
// wrapComponents does. A node is in the part of the group it is in, and an edge joins the
// parts of its ends. The parts are taken in the order their first nodes come, and each moves
// to the start of the row it fits in, or of a new row. It gives the size of the graph so
// moved, and false, moving nothing, where there is one part or the rows are no narrower.
func WrapComponents(nodes []CNode, edges []CEdge, width, gap float64) (CompoundResult, bool) {
	parent := make([]int, len(nodes))
	for i := range parent {
		parent[i] = i
	}
	root := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	join := func(a, b int) { parent[root(a)] = root(b) }
	for i := range nodes {
		if nodes[i].Parent >= 0 {
			join(i, nodes[i].Parent)
		}
	}
	for _, e := range edges {
		join(e.Src, e.Dst)
	}

	type part struct {
		nodes, edges             []int
		left, top, right, bottom float64
	}
	var parts []*part
	of := map[int]*part{}
	include := func(p *part, x, y, w, h float64) {
		p.left = math.Min(p.left, x-w/2)
		p.top = math.Min(p.top, y-h/2)
		p.right = math.Max(p.right, x+w/2)
		p.bottom = math.Max(p.bottom, y+h/2)
	}
	for i := range nodes {
		id := root(i)
		p := of[id]
		if p == nil {
			p = &part{left: math.Inf(1), top: math.Inf(1), right: math.Inf(-1), bottom: math.Inf(-1)}
			of[id] = p
			parts = append(parts, p)
		}
		c := &nodes[i]
		p.nodes = append(p.nodes, i)
		include(p, c.X, c.Y, c.W, c.H)
	}
	if len(parts) < 2 {
		return CompoundResult{}, false
	}
	for k := range edges {
		e := &edges[k]
		p := of[root(e.Src)]
		p.edges = append(p.edges, k)
		for i := 0; i+1 < len(e.Route); i += 3 {
			include(p, e.Route[i], e.Route[i+1], 0, 0)
		}
		if e.LabelW > 0 {
			include(p, e.LabelX, e.LabelY, e.LabelW, e.LabelH)
		}
	}

	x, y, rowHeight, right := 0.0, 0.0, 0.0, 0.0
	from, to := math.Inf(1), math.Inf(-1)
	shifts := make([]float64, 0, 2*len(parts))
	for _, p := range parts {
		w, h := p.right-p.left, p.bottom-p.top
		if x > 0 && x+w > width {
			y += rowHeight + gap
			x = 0
			rowHeight = 0
		}
		shifts = append(shifts, x-p.left, y-p.top)
		from = math.Min(from, p.left)
		to = math.Max(to, p.right)
		right = math.Max(right, x+w)
		rowHeight = math.Max(rowHeight, h)
		x += w + gap
	}
	if right >= to-from {
		return CompoundResult{}, false
	}
	// Each part moves as Pele's shiftLayout moves a graph.
	for k, p := range parts {
		ox, oy := shifts[2*k], shifts[2*k+1]
		for _, i := range p.nodes {
			nodes[i].X += ox
			nodes[i].Y += oy
		}
		for _, i := range p.edges {
			e := &edges[i]
			for j := 0; j+1 < len(e.Route); j += 3 {
				e.Route[j] += ox
				e.Route[j+1] += oy
			}
			e.LabelX += ox
			e.LabelY += oy
		}
	}
	return CompoundResult{Width: right, Height: y + rowHeight}, true
}
