package architecture

import (
	"math"
	"slices"
)

// Grid placement, as Pele's layout.ts has it. Mermaid hands the edge sides to a
// force-directed solver as relative placement constraints. Pele applies them directly: every
// service and junction takes a cell, an edge a:R -- L:b puts b in the cell to the right of a,
// and a group is a solid rectangle of cells, so a group never encloses a node that is not in
// it. Each container (the diagram, then every group) is laid out on its own, innermost first,
// and a finished group is placed in its parent as one block.

// grid is Pele's Grid: the cell of each node, in the order of the nodes passed in, and the
// cell rectangle and parent of each group. A group with nothing in it takes one cell.
type grid struct {
	col, row                                 []int
	groupCol, groupRow, groupCols, groupRows []int
	groupParent                              []int
	groupEmpty                               []bool
	nodeParent                               []int
}

var (
	gridDX = [4]int{-1, 1, 0, 0}
	gridDY = [4]int{0, 0, -1, 1}
)

// sideIndex is the index of a side in gridDX and gridDY, as Pele's sideIndex gives it.
func sideIndex(s Side) int {
	switch s {
	case 'L':
		return 0
	case 'R':
		return 1
	case 'T':
		return 2
	}
	return 3
}

// small is the most cells a block found through the cell map holds; larger ones are compared
// one by one.
const small = 64

// cellKey is a cell as the key of the cell map.
type cellKey struct{ x, y int }

// slideKey is a slide asked for, as the key of what it came to.
type slideKey struct{ rx, ry, rw, rh, sx, sy int }

// placeOnGrid places the nodes and groups on a grid as the edges and the align directives
// ask, as Pele's placeOnGrid does.
func placeOnGrid(nodes []*Node, groups []*Group, edges []Edge, hints []LayoutHint) grid {
	N := len(nodes)
	G := len(groups)
	root := G
	nodeIndex := map[string]int{}
	groupIndex := map[string]int{}
	// Items are nodes (0..N-1) then groups (N..N+G-1). Containers are groups (0..G-1) then the
	// root.
	parent := make([]int, N+G)
	for i := range parent {
		parent[i] = root
	}
	depth := make([]int, G+1)
	kids := make([][]int, G+1)
	for g, group := range groups {
		groupIndex[group.ID] = g
		p, ok := -1, false
		if group.In != "" {
			p, ok = groupIndex[group.In]
		}
		if ok && p != g {
			parent[N+g] = p
			depth[g] = depth[p] + 1
		} else {
			depth[g] = 1
		}
	}
	for i, node := range nodes {
		nodeIndex[node.ID] = i
		if node.In != "" {
			if p, ok := groupIndex[node.In]; ok {
				parent[i] = p
			}
		}
		kids[parent[i]] = append(kids[parent[i]], i)
	}
	for g := range G {
		kids[parent[N+g]] = append(kids[parent[N+g]], N+g)
	}

	// A link asks for one node to sit beside another. It belongs to the innermost container
	// that holds both ends, and there it relates the two children of that container the ends
	// are in.
	var linkA, linkB, sideA, sideB, itemA, itemB []int
	hintLinks := make([][]int, N+G)
	edgeLinks := make([][]int, N+G)
	addLink := func(a, b, sa, sb int, lists [][]int) {
		if a == b {
			return
		}
		ia, ib := a, b
		ca, cb := parent[a], parent[b]
		for ca != cb {
			if depth[ca] >= depth[cb] {
				ia = N + ca
				ca = parent[ia]
			} else {
				ib = N + cb
				cb = parent[ib]
			}
		}
		k := len(linkA)
		linkA = append(linkA, a)
		linkB = append(linkB, b)
		sideA = append(sideA, sa)
		sideB = append(sideB, sb)
		itemA = append(itemA, ia)
		itemB = append(itemB, ib)
		lists[ia] = append(lists[ia], k)
		lists[ib] = append(lists[ib], k)
	}
	for _, hint := range hints {
		row := hint.Direction == "row"
		for i := 1; i < len(hint.Members); i++ {
			a, okA := nodeIndex[hint.Members[i-1]]
			b, okB := nodeIndex[hint.Members[i]]
			if okA && okB {
				if row {
					addLink(a, b, 1, 0, hintLinks)
				} else {
					addLink(a, b, 3, 2, hintLinks)
				}
			}
		}
	}
	// An edge between two like sides says least about where its ends go, so those come last.
	for _, likeSides := range [2]bool{false, true} {
		for _, edge := range edges {
			if (edge.LhsDir == edge.RhsDir) != likeSides {
				continue
			}
			a, okA := nodeIndex[edge.LhsID]
			b, okB := nodeIndex[edge.RhsID]
			if okA && okB {
				addLink(a, b, sideIndex(edge.LhsDir), sideIndex(edge.RhsDir), edgeLinks)
			}
		}
	}

	// Position of each item in its container, and its size in cells.
	x := make([]int, N+G)
	y := make([]int, N+G)
	w := make([]int, N+G)
	h := make([]int, N+G)
	for i := range w {
		w[i], h[i] = 1, 1
	}
	placed := make([]bool, N+G)

	var order []int
	cells := map[cellKey]int{}
	var large []int
	reach := map[slideKey]int{}
	first := 0
	ax, ay := 0, 0

	anchor := func(node, item int) {
		ax, ay = 0, 0
		for cur := node; cur != item; cur = N + parent[cur] {
			ax += x[cur]
			ay += y[cur]
		}
	}

	put := func(item, px, py int) {
		x[item] = px
		y[item] = py
		placed[item] = true
		order = append(order, item)
		if w[item]*h[item] > small {
			large = append(large, item)
			return
		}
		for cy := py; cy < py+h[item]; cy++ {
			for cx := px; cx < px+w[item]; cx++ {
				cells[cellKey{cx, cy}] = item
			}
		}
	}

	overlaps := func(item, rx, ry, rw, rh int) bool {
		return x[item] < rx+rw && rx < x[item]+w[item] && y[item] < ry+rh && ry < y[item]+h[item]
	}

	collide := func(rx, ry, rw, rh int) int {
		if rw*rh > small {
			for i := first; i < len(order); i++ {
				if overlaps(order[i], rx, ry, rw, rh) {
					return order[i]
				}
			}
			return -1
		}
		for cy := ry; cy < ry+rh; cy++ {
			for cx := rx; cx < rx+rw; cx++ {
				if hit, ok := cells[cellKey{cx, cy}]; ok {
					return hit
				}
			}
		}
		for _, item := range large {
			if overlaps(item, rx, ry, rw, rh) {
				return item
			}
		}
		return -1
	}

	// How far a block must slide along (sx, sy) to be clear. Each step jumps past what it
	// hit, and the result is kept, because a later block starting from the same place cannot
	// stop sooner.
	slide := func(rx, ry, rw, rh, sx, sy int) int {
		key := slideKey{rx, ry, rw, rh, sx, sy}
		k := reach[key]
		for {
			hit := collide(rx+k*sx, ry+k*sy, rw, rh)
			if hit < 0 {
				break
			}
			switch {
			case sx > 0:
				k = x[hit] + w[hit] - rx
			case sx < 0:
				k = rx + rw - x[hit]
			case sy > 0:
				k = y[hit] + h[hit] - ry
			default:
				k = ry + rh - y[hit]
			}
		}
		reach[key] = k
		return k
	}

	follow := func(from, k int) {
		forward := itemA[k] == from
		to, sp, sq := itemA[k], sideB[k], sideA[k]
		fromLink, toLink := linkB[k], linkA[k]
		if forward {
			to, sp, sq = itemB[k], sideA[k], sideB[k]
			fromLink, toLink = linkA[k], linkB[k]
		}
		if placed[to] {
			return
		}
		anchor(fromLink, from)
		pax, pay := ax, ay
		anchor(toLink, to)
		dx, dy := gridDX[sp], gridDY[sp]
		shiftX, shiftY := 0, 0
		facing := gridDX[sp]+gridDX[sq] == 0 && gridDY[sp]+gridDY[sq] == 0
		if sp == sq {
			// Like sides: the edge loops around, so the nodes stack across the direction it
			// leaves in.
			dx = 0
			if gridDX[sp] == 0 {
				dx = 1
			}
			dy = 1 - dx
		} else if !facing {
			// A bend: one step along the side the edge leaves, one step back from the side it
			// enters.
			shiftX, shiftY = -gridDX[sq], -gridDY[sq]
		}
		var bx, by int
		if dx != 0 {
			if dx > 0 {
				bx = x[from] + w[from]
			} else {
				bx = x[from] - w[to]
			}
			by = y[from] + pay + shiftY - ay
		} else {
			if dy > 0 {
				by = y[from] + h[from]
			} else {
				by = y[from] - h[to]
			}
			bx = x[from] + pax + shiftX - ax
		}
		if facing {
			// When several placed nodes reach this one through the same pair of sides, line it
			// up with the middle one, not with whichever happened to be visited first.
			if list := edgeLinks[to]; len(list) > 1 {
				var wanted []int
				for _, j := range list {
					toIsB := itemB[j] == to
					other, so, st, lo, lt := itemB[j], sideB[j], sideA[j], linkB[j], linkA[j]
					if toIsB {
						other, so, st, lo, lt = itemA[j], sideA[j], sideB[j], linkA[j], linkB[j]
					}
					if !placed[other] || so != sp || st != sq {
						continue
					}
					anchor(lo, other)
					at := x[other] + ax
					if dx != 0 {
						at = y[other] + ay
					}
					anchor(lt, to)
					if dx != 0 {
						wanted = append(wanted, at-ay)
					} else {
						wanted = append(wanted, at-ax)
					}
				}
				if len(wanted) > 1 {
					slices.Sort(wanted)
					if dx != 0 {
						by = wanted[(len(wanted)-1)>>1]
					} else {
						bx = wanted[(len(wanted)-1)>>1]
					}
				}
			}
			// Taken: move sideways, to whichever side is nearer, so the edge can still leave
			// straight.
			px := 0
			if dx == 0 {
				px = 1
			}
			py := 1 - px
			forth := slide(bx, by, w[to], h[to], px, py)
			back := 0
			if forth != 0 {
				back = slide(bx, by, w[to], h[to], -px, -py)
			}
			near := -back
			if forth <= back {
				near = forth
			}
			bx += px * near
			by += py * near
		} else {
			far := slide(bx, by, w[to], h[to], dx, dy)
			bx += dx * far
			by += dy * far
		}
		put(to, bx, by)
	}

	layout := func(c int) {
		items := kids[c]
		if len(items) == 0 {
			return
		}
		order = order[:0]
		var starts []int
		for _, start := range items {
			if placed[start] {
				continue
			}
			first = len(order)
			starts = append(starts, first)
			clear(cells)
			large = large[:0]
			clear(reach)
			put(start, 0, 0)
			// Declared alignments are followed as soon as a node is placed, ahead of any edge.
			hintAt, edgeAt, linkAt := first, first, 0
			for {
				if hintAt < len(order) {
					from := order[hintAt]
					hintAt++
					for _, k := range hintLinks[from] {
						follow(from, k)
					}
					continue
				}
				if edgeAt >= len(order) {
					break
				}
				list := edgeLinks[order[edgeAt]]
				if linkAt >= len(list) {
					edgeAt++
					linkAt = 0
					continue
				}
				follow(order[edgeAt], list[linkAt])
				linkAt++
			}
		}
		starts = append(starts, len(order))

		// Unconnected parts go side by side, wrapping into rows that keep the whole roughly
		// square.
		count := len(starts) - 1
		minX := make([]int, count)
		minY := make([]int, count)
		partW := make([]int, count)
		partH := make([]int, count)
		area, widest := 0, 0
		for p := range count {
			x0, y0 := math.MaxInt, math.MaxInt
			x1, y1 := math.MinInt, math.MinInt
			for i := starts[p]; i < starts[p+1]; i++ {
				item := order[i]
				x0 = min(x0, x[item])
				y0 = min(y0, y[item])
				x1 = max(x1, x[item]+w[item])
				y1 = max(y1, y[item]+h[item])
			}
			minX[p], minY[p] = x0, y0
			partW[p], partH[p] = x1-x0, y1-y0
			area += partW[p] * partH[p]
			widest = max(widest, partW[p])
		}
		limit := max(widest, int(math.Ceil(math.Sqrt(float64(area*2)))))
		cx, cy, shelf, width := 0, 0, 0, 0
		for p := range count {
			if cx > 0 && cx+partW[p] > limit {
				cy += shelf
				cx = 0
				shelf = 0
			}
			for i := starts[p]; i < starts[p+1]; i++ {
				x[order[i]] += cx - minX[p]
				y[order[i]] += cy - minY[p]
			}
			cx += partW[p]
			shelf = max(shelf, partH[p])
			width = max(width, cx)
		}
		if c < G {
			w[N+c] = width
			h[N+c] = cy + shelf
		}
	}
	for c := G - 1; c >= 0; c-- {
		layout(c)
	}
	layout(root)

	out := grid{
		col: make([]int, N), row: make([]int, N), nodeParent: make([]int, N),
		groupCol: make([]int, G), groupRow: make([]int, G), groupParent: make([]int, G), groupEmpty: make([]bool, G),
		groupCols: slices.Clone(w[N:]), groupRows: slices.Clone(h[N:]),
	}
	for g := range G {
		p := parent[N+g]
		out.groupParent[g] = -1
		out.groupCol[g] = x[N+g]
		out.groupRow[g] = y[N+g]
		if p != root {
			out.groupParent[g] = p
			out.groupCol[g] += out.groupCol[p]
			out.groupRow[g] += out.groupRow[p]
		}
		out.groupEmpty[g] = len(kids[g]) == 0
	}
	for i := range N {
		p := parent[i]
		out.nodeParent[i] = -1
		out.col[i] = x[i]
		out.row[i] = y[i]
		if p != root {
			out.nodeParent[i] = p
			out.col[i] += out.groupCol[p]
			out.row[i] += out.groupRow[p]
		}
	}
	return out
}
