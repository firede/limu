package graphs

import (
	"slices"
)

// sortByBary sorts the first count nodes of list by barycenter, and those with the same one by
// position, the leftmost first or, with right, the rightmost, as Pele's sortByBary does: a
// merge sort over runs put in order by insertion. spare is room to merge into.
func sortByBary(list []int, count int, bary []float64, pos []int, right bool, spare []int) {
	before := func(a, b int) bool {
		return bary[a] < bary[b] || (bary[a] == bary[b] && (right && pos[a] > pos[b] || !right && pos[a] < pos[b]))
	}
	const run = 8
	for s := 0; s < count; s += run {
		for i, end := s+1, min(count, s+run); i < end; i++ {
			item := list[i]
			j := i - 1
			for j >= s && before(item, list[j]) {
				list[j+1] = list[j]
				j--
			}
			list[j+1] = item
		}
	}
	from, to := list, spare
	for width := run; width < count; width *= 2 {
		for s := 0; s < count; s += 2 * width {
			mid := min(count, s+width)
			end := min(count, s+2*width)
			i, j, k := s, mid, s
			for i < mid && j < end {
				if before(from[j], from[i]) {
					to[k] = from[j]
					j++
				} else {
					to[k] = from[i]
					i++
				}
				k++
			}
			k += copy(to[k:end], from[i:mid])
			copy(to[k:end], from[j:end])
		}
		from, to = to, from
	}
	if &from[0] != &list[0] {
		copy(list[:count], from[:count])
	}
}

// reduceCrossings orders the nodes of each rank so that edges cross as little as they can, as
// Pele's reduceCrossings does: sweeps up and down that sort each rank by the mean place of its
// nodes' neighbours, keeping the order with the fewest crossings. It returns how often edges
// cross in that order, and how often they did in the order the nodes came in.
func reduceCrossings(g *graph, nodes []lNode) (int, int) {
	n, real, maxRank, W := g.n, g.real, g.maxRank, g.W
	EF, ET, EFD, ETD := g.EF, g.ET, g.EFD, g.ETD
	succStart, succE, predStart, predE := g.succStart, g.succE, g.predStart, g.predE
	layerStart, order, pos, offsets := g.layerStart, g.order, g.pos, g.offsets
	fixed, GRP, groupParent := g.fixed, g.GRP, g.groupParent
	widest := 0
	for r := 0; r <= maxRank; r++ {
		widest = max(widest, layerStart[r+1]-layerStart[r])
	}
	if widest < 2 {
		return 0, 0
	}
	// With offsets, edges that reach one node at different offsets take separate places along it.
	var place []int
	if offsets {
		place = make([]int, len(EF))
	}
	// Neither changes with the order: which of its node's places each edge reaches, counted from
	// the first, and how many places each node has.
	var placeIn, places []int
	// Each node's edges down, in the order of their offsets where those differ, which mixed marks.
	leaving := succE
	var mixed []bool
	if place != nil {
		placeIn = make([]int, len(EF))
		places = make([]int, n)
		leaving = slices.Clone(succE)
		mixed = make([]bool, n)
		arriving := slices.Clone(predE)
		byOffset := func(list []int, from, to int, dx []float64) bool {
			for k := from + 1; k < to; k++ {
				if dx[list[k]] == dx[list[from]] {
					continue
				}
				slices.SortStableFunc(list[from:to], func(a, b int) int { return sign(dx[a] - dx[b]) })
				return true
			}
			return false
		}
		for v := 0; v < n; v++ {
			places[v] = 1
			if byOffset(arriving, predStart[v], predStart[v+1], ETD) {
				for k := predStart[v] + 1; k < predStart[v+1]; k++ {
					if ETD[arriving[k]] != ETD[arriving[k-1]] {
						places[v]++
					}
					placeIn[arriving[k]] = places[v] - 1
				}
			}
			if byOffset(leaving, succStart[v], succStart[v+1], EFD) {
				mixed[v] = true
			}
		}
	}
	size := 1
	limit := widest
	if place != nil {
		limit = widest + len(EF)
	}
	for size < limit {
		size <<= 1
	}
	tree := make([]int, 2*size)
	targets := make([]int, len(EF))

	count := func() int {
		total := 0
		for r := 0; r < maxRank; r++ {
			southLen := layerStart[r+2] - layerStart[r+1]
			if place != nil {
				next := 0
				for i := layerStart[r+1]; i < layerStart[r+2]; i++ {
					v := order[i]
					for k := predStart[v]; k < predStart[v+1]; k++ {
						place[predE[k]] = next + placeIn[predE[k]]
					}
					next += places[v]
				}
				southLen = next
			}
			if southLen < 2 {
				continue
			}
			t := 0
			for i := layerStart[r]; i < layerStart[r+1]; i++ {
				v := order[i]
				from := succStart[v]
				to := succStart[v+1]
				t0 := t
				for k := from; k < to; k++ {
					if place != nil {
						targets[t] = place[leaving[k]]
					} else {
						targets[t] = pos[ET[succE[k]]]
					}
					t++
				}
				d := t - t0
				if d < 2 {
					continue
				}
				if place != nil && mixed[v] {
					// Edges that leave at one offset go in the order of their places.
					for a := t0 + 1; a < t; a++ {
						item := targets[a]
						dx := EFD[leaving[from+a-t0]]
						b := a - 1
						for b >= t0 && EFD[leaving[from+b-t0]] == dx && targets[b] > item {
							targets[b+1] = targets[b]
							b--
						}
						targets[b+1] = item
					}
				} else if d <= 12 {
					for a := t0 + 1; a < t; a++ {
						item := targets[a]
						b := a - 1
						for b >= t0 && targets[b] > item {
							targets[b+1] = targets[b]
							b--
						}
						targets[b+1] = item
					}
				} else {
					slices.Sort(targets[t0:t])
				}
			}
			if t < 2 {
				continue
			}
			leaves := 1
			for leaves < southLen {
				leaves <<= 1
			}
			clear(tree[:2*leaves])
			for k := 0; k < t; k++ {
				i := targets[k] + leaves
				tree[i]++
				for i > 1 {
					if i&1 == 0 {
						total += tree[i+1]
					}
					i >>= 1
					tree[i]++
				}
			}
		}
		return total
	}

	// How often edges cross in the order the nodes came in, before any groups are put together.
	declared := count()
	bary := make([]float64, n)
	moving := make([]int, widest)
	spare := make([]int, widest)

	// arrange puts the nodes of rank r in the order of key, keeping the nodes of each group
	// together: a group goes where the average of its nodes' keys puts it among the nodes and
	// groups beside it.
	groups := len(groupParent)
	depth := make([]int, groups)
	for gr := 0; gr < groups; gr++ {
		for p := gr; p >= 0; p = groupParent[p] {
			depth[gr]++
		}
	}
	groupSum := make([]float64, groups)
	groupCount := make([]int, groups)
	groupFirst := make([]int, groups)
	arrange := func(r int, key []float64) {
		s := layerStart[r]
		e := layerStart[r+1]
		for i := s; i < e; i++ {
			for gr := GRP[order[i]]; gr >= 0; gr = groupParent[gr] {
				groupSum[gr] = 0
				groupCount[gr] = 0
			}
		}
		for i := s; i < e; i++ {
			v := order[i]
			for gr := GRP[v]; gr >= 0; gr = groupParent[gr] {
				if groupCount[gr] == 0 {
					groupFirst[gr] = i
				}
				groupSum[gr] += key[v]
				groupCount[gr]++
			}
		}
		// Nodes are items numbered as they are, and groups after them. Two nodes compare as the
		// two items they are in, side by side in the group that holds both.
		value := func(item int) float64 {
			if item >= n {
				return groupSum[item-n] / float64(groupCount[item-n])
			}
			return key[item]
		}
		// Items that tie keep the order they had.
		first := func(item int) int {
			if item >= n {
				return groupFirst[item-n]
			}
			return s + pos[item]
		}
		compare := func(a, b int) int {
			ia, ib := a, b
			ca, cb := GRP[a], GRP[b]
			da, db := 0, 0
			if ca >= 0 {
				da = depth[ca]
			}
			if cb >= 0 {
				db = depth[cb]
			}
			for ; da > db; da-- {
				ia = n + ca
				ca = groupParent[ca]
			}
			for ; db > da; db-- {
				ib = n + cb
				cb = groupParent[cb]
			}
			for ca != cb {
				ia = n + ca
				ca = groupParent[ca]
				ib = n + cb
				cb = groupParent[cb]
			}
			return sign(or(value(ia)-value(ib), float64(first(ia)-first(ib))))
		}
		slices.SortStableFunc(order[s:e], compare)
		for i := s; i < e; i++ {
			pos[order[i]] = i - s
		}
	}
	if GRP != nil {
		at := make([]float64, n)
		for r := 0; r <= maxRank; r++ {
			for i := layerStart[r]; i < layerStart[r+1]; i++ {
				at[order[i]] = float64(i - layerStart[r])
			}
			arrange(r, at)
		}
	}
	best := declared
	if GRP != nil {
		best = count()
	}
	if best == 0 {
		return 0, declared
	}
	bestOrder := slices.Clone(order)
	// The order the nodes came in, with groups put together, is kept unless the sweeps find one
	// that crosses less than it and less than the nodes did before that.
	start := min(best, declared)

	sweep := func(down, biasRight bool) {
		start, list, other, otherDx := succStart, succE, ET, ETD
		if down {
			start, list, other, otherDx = predStart, predE, EF, EFD
		}
		for step := 1; step <= maxRank; step++ {
			r := maxRank - step
			if down {
				r = step
			}
			if fixed[r] {
				continue
			}
			s := layerStart[r]
			e := layerStart[r+1]
			movable := 0
			for i := s; i < e; i++ {
				v := order[i]
				pin := 0.0
				if v < real {
					pin = nodes[v].Pin
				}
				if pin != 0 {
					bary[v] = pin * 1e9
					moving[movable] = v
					movable++
					continue
				}
				from := start[v]
				to := start[v+1]
				if from == to {
					if GRP != nil {
						// Where groups keep their nodes together, one with no neighbours stays about
						// where it is.
						across := layerStart[r+2] - layerStart[r+1]
						if down {
							across = layerStart[r] - layerStart[r-1]
						}
						bary[v] = float64((i-s)*across) / float64(e-s)
						continue
					}
					bary[v] = -1
					continue
				}
				sum := 0.0
				for k := from; k < to; k++ {
					u := other[list[k]]
					sum += float64(pos[u])
					if offsets && W[u] > 0 {
						sum += otherDx[list[k]] / W[u]
					}
				}
				bary[v] = sum / float64(to-from)
				moving[movable] = v
				movable++
			}
			if GRP != nil {
				arrange(r, bary)
				continue
			}
			if movable < 2 {
				continue
			}
			sortByBary(moving, movable, bary, pos, biasRight, spare)
			k := 0
			for i := s; i < e; i++ {
				if bary[order[i]] != -1 {
					order[i] = moving[k]
					k++
				}
			}
			for i := s; i < e; i++ {
				pos[order[i]] = i - s
			}
		}
	}

	// How many pairs of nodes in one rank are not in the order they were given.
	outOfOrder := func() int {
		total := 0
		for r := 0; r <= maxRank; r++ {
			for i := layerStart[r]; i < layerStart[r+1]; i++ {
				if order[i] >= real {
					continue
				}
				for j := i + 1; j < layerStart[r+1]; j++ {
					if order[j] < order[i] {
						total++
					}
				}
			}
		}
		return total
	}
	settlePositions := func() {
		for r := 0; r <= maxRank; r++ {
			for i := layerStart[r]; i < layerStart[r+1]; i++ {
				pos[order[i]] = i - layerStart[r]
			}
		}
	}

	// transpose swaps neighbours in a rank wherever that leaves fewer crossings with the ranks on
	// either side, until no swap does. Sweeps move nodes by the average place of their
	// neighbours, which can leave two side by side that would cross less the other way round.
	pinOf := func(v int) float64 {
		if v < real {
			return nodes[v].Pin
		}
		return 0
	}
	// Where an edge meets the node at its other end, along that node's rank.
	at := func(e int, up bool) float64 {
		u, dx := ET[e], ETD[e]
		if up {
			u, dx = EF[e], EFD[e]
		}
		if offsets && W[u] > 0 {
			return float64(pos[u]) + dx/W[u]
		}
		return float64(pos[u])
	}
	// How often the edges of a cross those of b, with a left of b.
	between := func(a, b int) int {
		c := 0
		for k := predStart[a]; k < predStart[a+1]; k++ {
			x := at(predE[k], true)
			for j := predStart[b]; j < predStart[b+1]; j++ {
				if x > at(predE[j], true) {
					c++
				}
			}
		}
		for k := succStart[a]; k < succStart[a+1]; k++ {
			x := at(succE[k], false)
			for j := succStart[b]; j < succStart[b+1]; j++ {
				if x > at(succE[j], false) {
					c++
				}
			}
		}
		return c
	}
	transpose := func() {
		for pass := 0; pass < 8; pass++ {
			improved := false
			for r := 0; r <= maxRank; r++ {
				if fixed[r] {
					continue
				}
				s := layerStart[r]
				for i := s; i+1 < layerStart[r+1]; i++ {
					a := order[i]
					b := order[i+1]
					if pinOf(a) != pinOf(b) || (GRP != nil && GRP[a] != GRP[b]) || between(b, a) >= between(a, b) {
						continue
					}
					order[i] = b
					order[i+1] = a
					pos[b] = i - s
					pos[a] = i + 1 - s
					improved = true
				}
			}
			if !improved {
				break
			}
		}
	}

	// The sweeps are run twice from the same start, going down first and going up first. They
	// often end with as few crossings either way but a different order, and then the one that
	// keeps more of the nodes in the order they were given is the less surprising.
	// Only for graphs small enough to be read node by node: a second run costs as much as the
	// first.
	initial := slices.Clone(order)
	small := n <= 600
	chosen := bestOrder
	chosenOut := -1 // Pele's Infinity: no attempt is chosen yet
	attempts := 1
	if small {
		attempts = 2
	}
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			copy(order, initial)
			settlePositions()
		}
		found := start
		foundOrder := slices.Clone(initial)
		for i, stale := 0, 0; stale < 4 && i < maxSweeps; i, stale = i+1, stale+1 {
			sweep((i+attempt)%2 == 0, i%4 >= 2)
			c := count()
			if c < found {
				found = c
				copy(foundOrder, order)
				stale = 0
				if c == 0 {
					break
				}
			}
		}
		copy(order, foundOrder)
		if small && found > 0 {
			settlePositions()
			transpose()
			if c := count(); c < found {
				found = c
				copy(foundOrder, order)
			}
		}
		out := 0
		if small {
			out = outOfOrder()
		}
		if attempt == 0 || found < best || (found == best && out < chosenOut) {
			best = found
			chosen = foundOrder
			chosenOut = out
		}
	}
	copy(order, chosen)
	settlePositions()
	copy(bestOrder, chosen)
	// With groups, the order kept can cross more than the count it was kept by.
	if GRP != nil {
		best = count()
	}

	// A drawing and its mirror image cross as often too. Of those two, likewise.
	if !small {
		return best, declared
	}
	for i := 0; i < real; i++ {
		if nodes[i].Pin != 0 {
			return best, declared
		}
	}
	if chosenOut == 0 || fixed[0] || fixed[maxRank] {
		return best, declared
	}
	for r := 0; r <= maxRank; r++ {
		slices.Reverse(order[layerStart[r]:layerStart[r+1]])
	}
	settlePositions()
	if count() > best || outOfOrder() >= chosenOut {
		copy(order, bestOrder)
	}
	return best, declared
}
