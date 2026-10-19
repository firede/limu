package graphs

import (
	"slices"
)

// reduceCrossings orders the nodes of each rank so that edges cross as little as they can, as
// Pele's reduceCrossings does: sweeps up and down that sort each rank by the mean place of its
// nodes' neighbours, keeping the order with the fewest crossings.
func reduceCrossings(g *graph, nodes []lNode) {
	n, real, maxRank, W := g.n, g.real, g.maxRank, g.W
	EF, ET, EFD, ETD := g.EF, g.ET, g.EFD, g.ETD
	succStart, succE, predStart, predE := g.succStart, g.succE, g.predStart, g.predE
	layerStart, order, pos, offsets := g.layerStart, g.order, g.pos, g.offsets
	widest := 0
	for r := 0; r <= maxRank; r++ {
		widest = max(widest, layerStart[r+1]-layerStart[r])
	}
	// With offsets, edges that reach one node at different offsets take separate places along it.
	var place []int
	if offsets {
		place = make([]int, len(EF))
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
	var ids []int

	count := func() int {
		total := 0
		for r := 0; r < maxRank; r++ {
			southLen := layerStart[r+2] - layerStart[r+1]
			if place != nil {
				places := 0
				for i := layerStart[r+1]; i < layerStart[r+2]; i++ {
					from := predStart[order[i]]
					to := predStart[order[i]+1]
					mixed := false
					for k := from + 1; k < to; k++ {
						if ETD[predE[k]] != ETD[predE[from]] {
							mixed = true
						}
					}
					if mixed {
						ids = append(ids[:0], predE[from:to]...)
						slices.SortStableFunc(ids, func(a, b int) int { return sign(ETD[a] - ETD[b]) })
						for k := range ids {
							if k > 0 && ETD[ids[k]] != ETD[ids[k-1]] {
								places++
							}
							place[ids[k]] = places
						}
					} else {
						for k := from; k < to; k++ {
							place[predE[k]] = places
						}
					}
					places++
				}
				southLen = places
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
						targets[t] = place[succE[k]]
					} else {
						targets[t] = pos[ET[succE[k]]]
					}
					t++
				}
				d := t - t0
				if d < 2 {
					continue
				}
				mixed := false
				if place != nil {
					for k := from + 1; k < to; k++ {
						if EFD[succE[k]] != EFD[succE[from]] {
							mixed = true
						}
					}
				}
				if place != nil && mixed {
					ids = append(ids[:0], succE[from:to]...)
					slices.SortStableFunc(ids, func(a, b int) int {
						return sign(or(EFD[a]-EFD[b], float64(place[a]-place[b])))
					})
					for k := 0; k < d; k++ {
						targets[t0+k] = place[ids[k]]
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

	best := count()
	if best == 0 {
		return
	}
	bestOrder := slices.Clone(order)
	bary := make([]float64, n)
	moving := make([]int, widest)
	leftBias := func(a, b int) int { return sign(or(bary[a]-bary[b], float64(pos[a]-pos[b]))) }
	rightBias := func(a, b int) int { return sign(or(bary[a]-bary[b], float64(pos[b]-pos[a]))) }

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
			if movable < 2 {
				continue
			}
			if biasRight {
				slices.SortStableFunc(moving[:movable], rightBias)
			} else {
				slices.SortStableFunc(moving[:movable], leftBias)
			}
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

	// The sweeps are run twice from the same start, going down first and going up first. They
	// often end with as few crossings either way but a different order, and then the one that
	// keeps more of the nodes in the order they were given is the less surprising.
	// Only for graphs small enough to be read node by node: a second run costs as much as the
	// first.
	initial := slices.Clone(order)
	start := best
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

	// A drawing and its mirror image cross as often too. Of those two, likewise.
	if !small {
		return
	}
	for i := 0; i < real; i++ {
		if nodes[i].Pin != 0 {
			return
		}
	}
	if chosenOut == 0 {
		return
	}
	for r := 0; r <= maxRank; r++ {
		slices.Reverse(order[layerStart[r]:layerStart[r+1]])
	}
	settlePositions()
	if count() > best || outOfOrder() >= chosenOut {
		copy(order, bestOrder)
	}
}
