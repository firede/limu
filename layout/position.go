package layout

import "math"

// halfRoom is half the room a node, bend or port takes across the flow, with its share of the
// gap beside it, as Pele's halfRoom is. A port on a group's border is a point that an edge
// passes through, and needs no more than a bend.
func halfRoom(g *graph, opt LayeredOptions, v int) float64 {
	sep := opt.EdgeSep
	if g.KIND[v] == KindNode {
		sep = opt.NodeSep
	}
	return g.W[v]/2 + sep/2
}

// position places every node across the flow, as Pele's position does: Brandes and Köpf's
// assignment of coordinates, four aligned layouts, balanced.
func position(g *graph, opt LayeredOptions) []float64 {
	n, maxRank, W, KIND := g.n, g.maxRank, g.W, g.KIND
	EF, ET, EFD, ETD := g.EF, g.ET, g.EFD, g.ETD
	succStart, succE, predStart, predE := g.succStart, g.succE, g.predStart, g.predE
	layerStart, order, pos := g.layerStart, g.order, g.pos
	segments := len(EF)
	gap := make([]float64, n)
	for i := 0; i < n; i++ {
		gap[i] = halfRoom(g, opt, i)
	}

	// Neighbours in left-to-right order, so the medians can be read off directly.
	key := make([]int, segments)
	for e := 0; e < segments; e++ {
		key[e] = pos[ET[e]]
	}
	for v := 0; v < n; v++ {
		if succStart[v+1]-succStart[v] > 1 {
			sortRange(succE, succStart[v], succStart[v+1], key)
		}
	}
	for e := 0; e < segments; e++ {
		key[e] = pos[EF[e]]
	}
	for v := 0; v < n; v++ {
		if predStart[v+1]-predStart[v] > 1 {
			sortRange(predE, predStart[v], predStart[v+1], key)
		}
	}

	// An edge between two real nodes may not cross a dummy-to-dummy segment; mark those that would.
	conflict := make([]bool, segments)
	for r := 1; r <= maxRank; r++ {
		prevLen := layerStart[r] - layerStart[r-1]
		s := layerStart[r]
		end := layerStart[r+1]
		k0 := 0
		scan := s
		for i := s; i < end; i++ {
			v := order[i]
			inner := -1
			if isDummy(KIND[v]) {
				u := EF[predE[predStart[v]]]
				if isDummy(KIND[u]) {
					inner = u
				}
			}
			k1 := prevLen
			if inner >= 0 {
				k1 = pos[inner]
			}
			if inner >= 0 || i == end-1 {
				for j := scan; j <= i; j++ {
					w := order[j]
					wDummy := isDummy(KIND[w])
					for k := predStart[w]; k < predStart[w+1]; k++ {
						e := predE[k]
						up := pos[EF[e]]
						if (up < k0 || k1 < up) && !(wDummy && isDummy(KIND[EF[e]])) {
							conflict[e] = true
						}
					}
				}
				scan = i + 1
				k0 = k1
			}
		}
	}

	var results [4][]float64
	root := make([]int, n)
	align := make([]int, n)
	// Offset of each node from its block's root, so that aligned edges meet at their attachment
	// points.
	shift := make([]float64, n)
	eFrom := make([]int, n)
	eTo := make([]int, n)
	eW := make([]float64, n)
	indeg := make([]int, n)
	outStart := make([]int, n+1)
	outE := make([]int, n)
	fill := make([]int, n)
	topo := make([]int, n)

	for variant := 0; variant < 4; variant++ {
		up := variant < 2
		right := variant&1 == 1
		start, list, other, ownDx, otherDx := succStart, succE, ET, EFD, ETD
		if up {
			start, list, other, ownDx, otherDx = predStart, predE, EF, ETD, EFD
		}
		for i := 0; i < n; i++ {
			root[i] = i
			align[i] = i
			shift[i] = 0
		}

		for step := 0; step <= maxRank; step++ {
			r := maxRank - step
			if up {
				r = step
			}
			s := layerStart[r]
			end := layerStart[r+1]
			adjacent := r + 1
			if up {
				adjacent = r - 1
			}
			adjacentLast := 0
			if adjacent >= 0 && adjacent <= maxRank {
				adjacentLast = layerStart[adjacent+1] - layerStart[adjacent] - 1
			}
			prevIdx := -1
			i, last, delta := s, end, 1
			if right {
				i, last, delta = end-1, s-1, -1
			}
			for ; i != last; i += delta {
				v := order[i]
				from := start[v]
				d := start[v+1] - from
				if d == 0 {
					continue
				}
				lo := from + ((d - 1) >> 1)
				hi := from + (d >> 1)
				for pass := 0; pass < 2; pass++ {
					if pass == 1 && lo == hi {
						break
					}
					at := hi
					if (pass == 0) != right {
						at = lo
					}
					e := list[at]
					w := other[e]
					idx := pos[w]
					if right {
						idx = adjacentLast - pos[w]
					}
					if align[v] == v && prevIdx < idx && !conflict[e] {
						align[w] = v
						root[v] = root[w]
						align[v] = root[v]
						shift[v] = shift[w] + otherDx[e] - ownDx[e]
						prevIdx = idx
					}
				}
			}
		}

		blockEdges := 0
		clear(indeg)
		clear(outStart)
		for r := 0; r <= maxRank; r++ {
			s := layerStart[r]
			end := layerStart[r+1]
			for i := s + 1; i < end; i++ {
				a, b := order[i-1], order[i]
				if right {
					a, b = b, a
				}
				sep := gap[a] + gap[b]
				eFrom[blockEdges] = root[a]
				eTo[blockEdges] = root[b]
				if right {
					eW[blockEdges] = sep - shift[a] + shift[b]
				} else {
					eW[blockEdges] = sep + shift[a] - shift[b]
				}
				indeg[root[b]]++
				outStart[root[a]+1]++
				blockEdges++
			}
		}
		for i := 0; i < n; i++ {
			outStart[i+1] += outStart[i]
		}
		copy(fill, outStart[:n])
		for e := 0; e < blockEdges; e++ {
			outE[fill[eFrom[e]]] = e
			fill[eFrom[e]]++
		}

		x := make([]float64, n)
		count := 0
		for i := 0; i < n; i++ {
			if root[i] == i && indeg[i] == 0 {
				topo[count] = i
				count++
			}
		}
		for q := 0; q < count; q++ {
			v := topo[q]
			for k := outStart[v]; k < outStart[v+1]; k++ {
				e := outE[k]
				w := eTo[e]
				if x[v]+eW[e] > x[w] {
					x[w] = x[v] + eW[e]
				}
				indeg[w]--
				if indeg[w] == 0 {
					topo[count] = w
					count++
				}
			}
		}
		for q := count - 1; q >= 0; q-- {
			v := topo[q]
			least := math.Inf(1)
			for k := outStart[v]; k < outStart[v+1]; k++ {
				e := outE[k]
				if cand := x[eTo[e]] - eW[e]; cand < least {
					least = cand
				}
			}
			if !math.IsInf(least, 1) && least > x[v] {
				x[v] = least
			}
		}
		xs := make([]float64, n)
		for i := 0; i < n; i++ {
			if right {
				xs[i] = -x[root[i]] + shift[i]
			} else {
				xs[i] = x[root[i]] + shift[i]
			}
		}
		results[variant] = xs
	}

	narrowest := 0
	bestWidth := math.Inf(1)
	var mins, maxs [4]float64
	for k := 0; k < 4; k++ {
		lo := math.Inf(1)
		hi := math.Inf(-1)
		xs := results[k]
		for i := 0; i < n; i++ {
			half := W[i] / 2
			if xs[i]-half < lo {
				lo = xs[i] - half
			}
			if xs[i]+half > hi {
				hi = xs[i] + half
			}
		}
		mins[k] = lo
		maxs[k] = hi
		if hi-lo < bestWidth {
			bestWidth = hi - lo
			narrowest = k
		}
	}
	for k := 0; k < 4; k++ {
		delta := maxs[narrowest] - maxs[k]
		if k&1 == 0 {
			delta = mins[narrowest] - mins[k]
		}
		if delta != 0 {
			xs := results[k]
			for i := 0; i < n; i++ {
				xs[i] += delta
			}
		}
	}

	out := make([]float64, n)
	for i := 0; i < n; i++ {
		a := results[0][i]
		b := results[1][i]
		c := results[2][i]
		d := results[3][i]
		if a > b {
			a, b = b, a
		}
		if c > d {
			c, d = d, c
		}
		if a > c {
			a, c = c, a
		}
		if b > d {
			b, d = d, b
		}
		if b > c {
			b, c = c, b
		}
		out[i] = (b + c) / 2
	}
	return out
}
