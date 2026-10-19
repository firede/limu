package graphs

import (
	"math"
	"slices"
)

// sideEnds lists in ids the ends of edges that may move along one side of node v (0 the side
// its edges arrive at, 1 the other), in the order of the nodes at their other ends, so that no
// two cross, as Pele's sideEnds does. room receives the least distance between each end and the
// one before it, and the result is their sum. When there is too little room for a full pitch
// between each pair, the edges that leave side by side move closer together, since they have no
// arrowheads to keep apart. With less room still, neighbours that run the same way share a
// place, which keeps an arrowhead apart from an edge that leaves beside it.
func sideEnds(g *graph, v, side int, span float64, kinds []end, ids *[]int, room *[]float64) float64 {
	start, list, other := g.predStart, g.predE, g.EF
	if side != 0 {
		start, list, other = g.succStart, g.succE, g.ET
	}
	pos := g.pos
	*ids = (*ids)[:0]
	for k := start[v]; k < start[v+1]; k++ {
		if kinds[list[k]] != endHeld {
			*ids = append(*ids, list[k])
		}
	}
	id := *ids
	if len(id) < 2 {
		return 0
	}
	slices.SortStableFunc(id, func(a, b int) int {
		if c := pos[other[a]] - pos[other[b]]; c != 0 {
			return c
		}
		return a - b
	})
	fill := func(share func(leaves, same bool) float64) float64 {
		total := 0.0
		r := *room
		if cap(r) < len(id) {
			r = make([]float64, len(id))
		}
		r = r[:len(id)]
		*room = r
		r[0] = 0
		for k := 1; k < len(id); k++ {
			same := kinds[id[k]] == kinds[id[k-1]]
			r[k] = share(same && kinds[id[k]] == endLeaves, same)
			total += r[k]
		}
		return total
	}
	total := fill(func(bool, bool) float64 { return 1 })
	if span/total < minPitch {
		total = fill(func(leaves, _ bool) float64 {
			if leaves {
				return 0.5
			}
			return 1
		})
	}
	if span/total < minPitch {
		total = fill(func(_, same bool) float64 {
			if same {
				return 0
			}
			return 1
		})
	}
	if total == 0 {
		return 0
	}
	pitch := math.Min(portPitch, span/total)
	r := *room
	for k := 1; k < len(id); k++ {
		r[k] *= pitch
	}
	return total * pitch
}

// spread gives the edges that meet one side of a node places of their own along it, around
// its middle, as Pele's spread does.
func spread(g *graph, nodes []lNode, EFK, ETK []end) {
	var ids []int
	var room []float64
	for v := 0; v < g.real; v++ {
		span := math.Min(nodes[v].Span, g.W[v])
		if span <= 0 || nodes[v].Kind != kindNode {
			continue
		}
		for side := 0; side < 2; side++ {
			kinds, dx := ETK, g.ETD
			if side != 0 {
				kinds, dx = EFK, g.EFD
			}
			total := sideEnds(g, v, side, span, kinds, &ids, &room)
			if total == 0 {
				continue
			}
			at := -total / 2
			for k := range ids {
				at += room[k]
				dx[ids[k]] = at
			}
		}
	}
}

// follow is the segment of an edge that reaches the next point held in place, on the way up
// from segment e (side 0) or down: past the bends in ranks that hold nothing else. It is
// Pele's follow.
func follow(g *graph, empty []bool, e, side int) int {
	if empty == nil {
		return e
	}
	for hops := 0; hops < 100000; hops++ {
		u := g.ET[e]
		if side == 0 {
			u = g.EF[e]
		}
		if g.KIND[u] != kindDummy || !empty[g.RANK[u]] {
			break
		}
		if side == 0 {
			e = g.predE[g.predStart[u]]
		} else {
			e = g.succE[g.succStart[u]]
		}
	}
	return e
}

// freeEnds counts, as Pele's freeEnds does, how many ends that may move along it each node has
// on the side its edges arrive at, and on the other. An end that has a side to itself keeps to
// the middle; ends that share one cannot all be there, so those may move.
func freeEnds(g *graph, EFK, ETK []end) [2][]int {
	free := [2][]int{make([]int, g.n), make([]int, g.n)}
	for e := range g.EF {
		if ETK[e] != endHeld {
			free[0][g.ET[e]]++
		}
		if EFK[e] != endHeld {
			free[1][g.EF[e]]++
		}
	}
	return free
}

// Ease is how far along the flow the handles of a plain S reach, as a share of the way it
// travels along it, as Pele's ease is. Half gives an even S. One that runs more than eight
// times as far across as along turns sooner and crosses in a straighter line, down to a
// quarter at sixteen times. The layout uses this to tell how near two curves will pass, and
// the drawing to draw them.
func Ease(across, along float64) float64 {
	return math.Max(0.25, math.Min(0.5, (4*math.Abs(along))/or(math.Abs(across), 1)))
}

// alignLabels brings the labels and bends of each edge into line with an end of the edge, as
// Pele's alignLabels does, so that it runs straight from that end and turns once, at the other:
// with the end that has fewer edges beside it on its node, which is the one nearer the middle
// of its side, or else with the other, and as near to that as what is beside them in their
// ranks leaves room. The other end may then move along its node to meet the line. Bends in
// ranks that hold nothing else are left to straighten. An edge with no one line that all of
// its labels and bends have room on has each brought into line with the nearest point above or
// below that is held in place.
func alignLabels(g *graph, nodes []lNode, xs []float64, free [2][]int, opt LayeredOptions, empty []bool) {
	sep := opt.EdgeSep
	real, maxRank, KIND, RANK, W := g.real, g.maxRank, g.KIND, g.RANK, g.W
	EF, ET, EFD, ETD := g.EF, g.ET, g.EFD, g.ETD
	succStart, succE, predStart, predE := g.succStart, g.succE, g.predStart, g.predE
	layerStart, order, pos := g.layerStart, g.order, g.pos
	placed := func(v int) bool {
		return KIND[v] == kindLabel || (KIND[v] == kindDummy && !(empty != nil && empty[RANK[v]]))
	}
	// Where segment e ends on the way up (side 0) or down, and how many ends share that side of
	// the node there: none for a point that is not on a node.
	end := func(e, side int) (float64, int) {
		u, x := ET[e], 0.0
		if side == 0 {
			u = EF[e]
			x = xs[u] + EFD[e]
		} else {
			x = xs[u] + ETD[e]
		}
		if u < real && nodes[u].Kind == kindNode {
			return x, free[1-side][u]
		}
		return x, 0
	}
	// The room the neighbours in its rank leave a label or bend: a full gap from a node or
	// between two labels, half of one between a label and a line that passes it.
	room := func(v int) (float64, float64) {
		s := layerStart[RANK[v]]
		i := s + pos[v]
		clear := func(other int) float64 {
			var gap float64
			if KIND[other] == kindNode {
				gap = opt.NodeSep/2 + sep/2
			} else if KIND[other] == KIND[v] {
				gap = sep
			} else {
				gap = sep / 2
			}
			return W[other]/2 + W[v]/2 + gap
		}
		lo, hi := math.Inf(-1), math.Inf(1)
		if i > s {
			lo = xs[order[i-1]] + clear(order[i-1])
		}
		if i+1 < layerStart[RANK[v]+1] {
			hi = xs[order[i+1]] - clear(order[i+1])
		}
		return lo, hi
	}
	// The place between lo and hi that is in line with one of two ends, or else nearest the
	// first choice.
	choose := func(x, upX float64, upN int, downX float64, downN int, lo, hi float64) float64 {
		var upFirst bool
		if upN != downN {
			upFirst = upN < downN
		} else {
			upFirst = math.Abs(upX-x) <= math.Abs(downX-x)
		}
		first, second := downX, upX
		if upFirst {
			first, second = upX, downX
		}
		if first >= lo && first <= hi {
			return first
		}
		if second >= lo && second <= hi {
			return second
		}
		return math.Max(lo, math.Min(hi, first))
	}
	done := make([]bool, g.n)
	var chain []int
	for r := 1; r < maxRank; r++ {
		for i := layerStart[r]; i < layerStart[r+1]; i++ {
			if !placed(order[i]) || done[order[i]] {
				continue
			}
			// The edge's labels and bends from its first node to its last, and the room they share.
			top := order[i]
			for isDummy(KIND[EF[predE[predStart[top]]]]) {
				top = EF[predE[predStart[top]]]
			}
			bottom := top
			lo := math.Inf(-1)
			hi := math.Inf(1)
			chain = chain[:0]
			for d := top; isDummy(KIND[d]); d = ET[succE[succStart[d]]] {
				bottom = d
				if !placed(d) {
					continue
				}
				done[d] = true
				chain = append(chain, d)
				least, most := room(d)
				if least > lo {
					lo = least
				}
				if most < hi {
					hi = most
				}
			}
			if lo <= hi {
				upX, upN := end(predE[predStart[top]], 0)
				downX, downN := end(succE[succStart[bottom]], 1)
				x := choose(xs[chain[0]], upX, upN, downX, downN, lo, hi)
				for _, d := range chain {
					xs[d] = x
				}
				continue
			}
			for _, d := range chain {
				least, most := room(d)
				if least > most {
					continue
				}
				upX, upN := end(follow(g, empty, predE[predStart[d]], 0), 0)
				downX, downN := end(follow(g, empty, succE[succStart[d]], 1), 1)
				xs[d] = choose(xs[d], upX, upN, downX, downN, least, most)
			}
		}
	}
}

// settle moves an end along its side, once the nodes have their places, when that lets its edge
// run straight, as Pele's settle does: when the point the edge runs to, or some part of the
// side it runs to, is in line with the end's own side. Any other end leans towards the point
// its edge runs to, where its side is long enough, and an end with a side to itself stays in
// the middle. On a group, the ends keep clear of the points where other edges pass through its
// border.
func settle(g *graph, nodes []lNode, xs []float64, EFK, ETK []end, free [2][]int, empty []bool) {
	real, W := g.real, g.W
	EF, ET, EFD, ETD := g.EF, g.ET, g.EFD, g.ETD
	succStart, succE, predStart, predE, pos := g.succStart, g.succE, g.predStart, g.predE, g.pos
	var ids []int
	var room []float64
	var all []int
	var sum []float64
	var count []int
	// Where the end of segment e on node v should be, from the middle of its side. The edge is
	// followed up from the segment (side 0) or down, past the bends in ranks that hold nothing
	// else, to where it next comes to a point that is held in place or to another node's side.
	wanted := func(e, side, v int, span float64, again bool) float64 {
		if free[side][v] < 2 {
			return 0
		}
		e = follow(g, empty, e, side)
		var u int
		var point float64
		if side == 0 {
			u = EF[e]
			point = xs[u] + EFD[e]
		} else {
			u = ET[e]
			point = xs[u] + ETD[e]
		}
		// Once the other end has its place, this one comes into line with it if it can.
		if again && math.Abs(point-xs[v]) <= span/2+inLine {
			return math.Max(-span/2, math.Min(span/2, point-xs[v]))
		}
		// The part of the other side that its end may move along, if it may move.
		middle := point
		width := 0.0
		otherKinds := ETK
		if side == 0 {
			otherKinds = EFK
		}
		if u < real && nodes[u].Kind == kindNode && otherKinds[e] != endHeld && free[1-side][u] >= 2 {
			width = math.Max(0, math.Min(nodes[u].Span, W[u]))
			if width > 0 {
				middle = xs[u]
			}
		}
		lo := math.Max(xs[v]-span/2, middle-width/2)
		hi := math.Min(xs[v]+span/2, middle+width/2)
		// Two sides with no part in line: the end leans towards the other, if its side is long
		// enough.
		if lo > hi+inLine {
			lean := math.Max(0, span/2-leanClear)
			return math.Max(-lean, math.Min(lean, point-xs[v]))
		}
		// Both ends settle on one place in the part the two sides share: nearer the middle of
		// the narrower.
		place := (float64(xs[v]*width) + float64(middle*span)) / (width + span)
		return math.Max(lo, math.Min(hi, place)) - xs[v]
	}
	// The ends of an edge follow each other, so the nodes are gone over until none moves: a few
	// times at most, in practice.
	for pass, moved := 0, true; moved && pass < 6; pass++ {
		moved = false
		for v := 0; v < real; v++ {
			span := math.Min(nodes[v].Span, W[v])
			if span <= 0 || nodes[v].Kind != kindNode {
				continue
			}
			for side := 0; side < 2; side++ {
				kinds, dx, other, start, list := ETK, ETD, EF, predStart, predE
				if side != 0 {
					kinds, dx, other, start, list = EFK, EFD, ET, succStart, succE
				}
				sideEnds(g, v, side, span, kinds, &ids, &room)
				if len(ids) == 0 {
					continue
				}
				if len(ids) == 1 {
					if len(room) == 0 {
						room = append(room, 0)
					}
					room[0] = 0
				}
				// Every end on the side in order, the held ones among them, which divide the rest
				// into runs.
				all = all[:0]
				for k := start[v]; k < start[v+1]; k++ {
					all = append(all, list[k])
				}
				slices.SortStableFunc(all, func(a, b int) int {
					if c := pos[other[a]] - pos[other[b]]; c != 0 {
						return c
					}
					return a - b
				})
				k := 0
				lo := -span / 2
				for i := 0; i <= len(all); i++ {
					held := i < len(all) && kinds[all[i]] == endHeld
					if i < len(all) && !held {
						continue
					}
					// The run of free ends before this point is ids[k:], as many as were passed
					// since the last held end.
					size := 0
					for j := i - 1; j >= 0 && kinds[all[j]] != endHeld; j-- {
						size++
					}
					hi := span / 2
					if held {
						hi = dx[all[i]] - portPitch
					}
					if size > 0 {
						// The places nearest the wanted ones that keep the order and the room
						// between neighbours: an isotonic fit, by pooling neighbours that would
						// otherwise be too close.
						sum = sum[:0]
						count = count[:0]
						before := 0.0
						for j := 0; j < size; j++ {
							if j > 0 {
								before += room[k+j]
							}
							sum = append(sum, wanted(ids[k+j], side, v, span, pass > 0)-before)
							count = append(count, 1)
							// Ends that share a place for want of room stay together.
							together := j > 0 && room[k+j] == 0
							for len(sum) > 1 && (together || sum[len(sum)-2]/float64(count[len(count)-2]) > sum[len(sum)-1]/float64(count[len(count)-1])) {
								together = false
								top := sum[len(sum)-1]
								sum = sum[:len(sum)-1]
								pooled := count[len(count)-1]
								count = count[:len(count)-1]
								sum[len(sum)-1] += top
								count[len(count)-1] += pooled
							}
						}
						total := before
						j := 0
						before = 0
						for b := range sum {
							// With too little room between two held ends, the run sits in the
							// middle of what there is.
							var level float64
							if hi-total < lo {
								level = (lo + hi - total) / 2
							} else {
								level = math.Max(lo, math.Min(hi-total, sum[b]/float64(count[b])))
							}
							for c := 0; c < count[b]; c, j = c+1, j+1 {
								if j > 0 {
									before += room[k+j]
								}
								if math.Abs(dx[ids[k+j]]-level-before) > 0.01 {
									moved = true
								}
								dx[ids[k+j]] = level + before
							}
						}
						k += size
					}
					if held {
						lo = dx[all[i]] + portPitch
					}
				}
			}
		}
	}
}

// snap brings edges that are nearly straight into line, as Pele's snap does, by moving a node a
// little across the flow along with everything it is in line with already, where the nodes
// beside them leave room. What is in line stays in line, so each move leaves one more edge
// straight.
func snap(g *graph, xs []float64, empty []bool, opt LayeredOptions) {
	n, KIND, RANK, EF, ET, EFD, ETD := g.n, g.KIND, g.RANK, g.EF, g.ET, g.EFD, g.ETD
	layerStart, order, pos := g.layerStart, g.order, g.pos
	passed := func(u int) bool { return empty != nil && KIND[u] == kindDummy && empty[RANK[u]] }
	// Each edge from one point that is held in place to the next, past the bends in ranks that
	// hold nothing else.
	var a, b []int
	var da, db []float64
	for e := range EF {
		if passed(EF[e]) {
			continue
		}
		f := follow(g, empty, e, 1)
		a = append(a, EF[e])
		b = append(b, ET[f])
		da = append(da, EFD[e])
		db = append(db, ETD[f])
	}
	off := func(k int) float64 { return xs[b[k]] + db[k] - xs[a[k]] - da[k] }
	// The nodes in line with each other move as one.
	block := make([]int, n)
	members := make([][]int, n)
	for i := 0; i < n; i++ {
		block[i] = i
		members[i] = []int{i}
	}
	join := func(x, y int) {
		keep := block[x]
		drop := block[y]
		if keep == drop {
			return
		}
		if len(members[keep]) < len(members[drop]) {
			keep, drop = drop, keep
		}
		for _, v := range members[drop] {
			block[v] = keep
			members[keep] = append(members[keep], v)
		}
		members[drop] = nil
	}
	var near []int
	for k := range a {
		d := math.Abs(off(k))
		if d < 0.01 {
			join(a[k], b[k])
		} else if d <= snapReach {
			near = append(near, k)
		}
	}
	if len(near) == 0 {
		return
	}
	slices.SortStableFunc(near, func(x, y int) int {
		return sign(or(math.Abs(off(x))-math.Abs(off(y)), float64(x-y)))
	})
	gap := func(v int) float64 { return halfRoom(g, opt, v) }
	// Whether every node of a block can move by shift and keep its room from those beside it.
	free := func(id int, shift float64) bool {
		for _, v := range members[id] {
			s := layerStart[RANK[v]]
			end := layerStart[RANK[v]+1]
			i := s + pos[v]
			if i > s && block[order[i-1]] != id && xs[v]+shift-xs[order[i-1]] < gap(v)+gap(order[i-1])-0.01 {
				return false
			}
			if i+1 < end && block[order[i+1]] != id && xs[order[i+1]]-xs[v]-shift < gap(v)+gap(order[i+1])-0.01 {
				return false
			}
		}
		return true
	}
	for _, k := range near {
		d := off(k)
		if math.Abs(d) < 0.01 {
			join(a[k], b[k])
			continue
		}
		if math.Abs(d) > snapReach || block[a[k]] == block[b[k]] {
			continue
		}
		// The smaller of the two blocks moves, if it can; else the other.
		upper := block[a[k]]
		lower := block[b[k]]
		first := lower
		if len(members[upper]) <= len(members[lower]) {
			first = upper
		}
		second := upper
		if first == upper {
			second = lower
		}
		for _, id := range [2]int{first, second} {
			shift := -d
			if id == upper {
				shift = d
			}
			if !free(id, shift) {
				continue
			}
			for _, v := range members[id] {
				xs[v] += shift
			}
			join(a[k], b[k])
			break
		}
	}
}

// emptyRanks marks the ranks that hold nothing but the bends of edges, if there are any, as
// Pele's emptyRanks does; it is nil when there are none.
func emptyRanks(g *graph) []bool {
	empty := make([]bool, g.maxRank+1)
	any := false
	for r := 1; r < g.maxRank; r++ {
		only := g.layerStart[r+1] > g.layerStart[r]
		for i := g.layerStart[r]; only && i < g.layerStart[r+1]; i++ {
			if g.KIND[g.order[i]] != kindDummy {
				only = false
			}
		}
		if only {
			empty[r] = true
			any = true
		}
	}
	if !any {
		return nil
	}
	return empty
}
