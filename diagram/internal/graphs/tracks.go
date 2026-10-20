package graphs

import (
	"math"
	"slices"

	"github.com/firede/limu/internal/jsmath"
)

// How many short lines a curve is taken as when measuring how near two pass.
const samples = 16

// assignTracks gives every piece that does not run straight down a track across its gap, and
// counts the tracks of each gap, as Pele's assignTracks does. Two pieces in one gap share a
// track only where they cannot be mistaken for each other: where they do not overlap, or where
// they leave from or arrive at the same point, as the branches of a tree do. Which of two goes
// above the other is whichever makes them cross less. With curved, the tracks are levels for
// curves to turn onto, and only the pieces that would run along or too near another get one:
// each group of those counts its tracks from its own first. room is the height of the gap under
// each rank, and tall receives the height that a gap whose curves mostly cross each other should
// have instead of levels.
func assignTracks(p *pieces, trackCount []int, curved bool, room, tall []float64) {
	from, to, rank, wide := p.from, p.to, p.rank, p.wide
	total := len(from)
	p.track = make([]int, total)
	p.levels = make([]int, total)
	track, levels := p.track, p.levels
	// The pieces of each gap, the gaps in the order Pele's Map meets them.
	var ranks []int
	byRank := make(map[int][]int)
	for k := 0; k < total; k++ {
		track[k] = -1
		if math.Abs(from[k]-to[k]) < inLine {
			continue
		}
		if _, ok := byRank[rank[k]]; !ok {
			ranks = append(ranks, rank[k])
		}
		byRank[rank[k]] = append(byRank[rank[k]], k)
	}
	if len(ranks) == 0 {
		return
	}
	// The pieces are compared pair by pair, so where each begins and ends across the flow is
	// worked out once.
	LO := make([]float64, total)
	HI := make([]float64, total)
	for k := 0; k < total; k++ {
		LO[k] = math.Min(from[k], to[k])
		HI[k] = math.Max(from[k], to[k])
	}
	within := func(x float64, k int) int {
		if x >= LO[k]-inLine && x <= HI[k]+inLine {
			return 1
		}
		return 0
	}
	mates := func(a, b int) bool { return math.Abs(from[a]-from[b]) < inLine || math.Abs(to[a]-to[b]) < inLine }
	overlap := func(a, b int) bool { return LO[a] < HI[b]-inLine && LO[b] < HI[a]-inLine }
	// Whether a and b need tracks of their own.
	clash := func(a, b int) bool { return overlap(a, b) && !mates(a, b) }
	// Whether one of a and b could cross the other, depending on which is above.
	meet := func(a, b int) bool {
		return overlap(a, b) || within(to[a], b)+within(from[b], a)+within(to[b], a)+within(from[a], b) > 0
	}
	// How often a and b cross with a on the higher track: a's way down through b, and b's way
	// down to itself through a.
	crossings := func(a, b int) int { return within(to[a], b) + within(from[b], a) }
	// nearPairs gives the pairs of pieces in list that come near enough across the flow to meet,
	// as places in the list, the smaller first. Pieces further apart than this cannot meet or
	// clash.
	nearPairs := func(list []int) []int {
		byLo := make([]int, len(list))
		for i := range byLo {
			byLo[i] = i
		}
		slices.SortStableFunc(byLo, func(a, b int) int { return sign(LO[list[a]] - LO[list[b]]) })
		var pairs []int
		for x := 0; x < len(byLo); x++ {
			i := byLo[x]
			reach := HI[list[i]] + 4*inLine
			for y := x + 1; y < len(byLo) && LO[list[byLo[y]]] <= reach; y++ {
				j := byLo[y]
				pairs = append(pairs, min(i, j), max(i, j))
			}
		}
		return pairs
	}

	// lay gives the pieces of one gap in list their tracks, and each the number of tracks that
	// the pieces it has to do with take between them, counting its own from the first of those.
	// It returns the number of tracks. The pieces come without tracks.
	lay := func(list []int) int {
		// A plain order to start from, and to keep where nothing says otherwise: the pieces that
		// reach furthest go first, so that those of one fan nest.
		slices.SortStableFunc(list, func(a, b int) int {
			return sign(or(HI[b]-LO[b]-(HI[a]-LO[a]), or(LO[a]-LO[b], float64(a-b))))
		})
		count := len(list)
		var above [][]int
		group := make([]int, count)
		for i := range group {
			group[i] = i
		}
		find := func(i int) int {
			for group[i] != i {
				g := group[group[i]]
				group[i] = g
				i = g
			}
			return i
		}
		weighed := count <= maxWeighed
		// Past maxWeighed, pairs are not kept: in a dense gap nearly every pair is near.
		var pairs []int
		var clashes [][]int // the pieces that each may not share a track with
		if weighed {
			pairs = nearPairs(list)
			clashes = make([][]int, count)
		}
		for p := 0; p < len(pairs); p += 2 {
			i, j := pairs[p], pairs[p+1]
			if !clash(list[i], list[j]) {
				continue
			}
			clashes[i] = append(clashes[i], j)
			clashes[j] = append(clashes[j], i)
		}
		sequence := make([]int, count)
		for i := range sequence {
			sequence[i] = i
		}
		if weighed {
			// above[j] lists the pieces that should be above list[j]. The order follows as many
			// of these as it can: a piece is placed once all that should be above it are, and
			// when none is free, the one with the fewest still waiting.
			waiting := make([]int, count)
			above = make([][]int, count)
			after := make([][]int, count)
			for p := 0; p < len(pairs); p += 2 {
				i, j := pairs[p], pairs[p+1]
				a := list[i]
				b := list[j]
				if !meet(a, b) {
					continue
				}
				ri := find(i)
				group[ri] = find(j)
				ab := crossings(a, b)
				ba := crossings(b, a)
				if ab == ba {
					continue
				}
				first, second := j, i
				if ab < ba {
					first, second = i, j
				}
				above[second] = append(above[second], first)
				after[first] = append(after[first], second)
				waiting[second]++
			}
			placed := make([]bool, count)
			sequence = sequence[:0]
			for done := 0; done < count; done++ {
				pick := -1
				for i := 0; i < count; i++ {
					if placed[i] {
						continue
					}
					if pick < 0 || waiting[i] < waiting[pick] {
						pick = i
					}
					if waiting[i] == 0 {
						break
					}
				}
				placed[pick] = true
				sequence = append(sequence, pick)
				for _, j := range after[pick] {
					waiting[j]--
				}
			}
		}
		// Each piece takes the first track below those it should be under that no piece it
		// clashes with is on. Unweighed, the pieces already on each track are looked through
		// instead.
		taken := make(map[int]bool)
		var onTrack [][]int
		rows := 0
		for _, at := range sequence {
			k := list[at]
			t := 0
			if above != nil {
				for _, i := range above[at] {
					if track[list[i]] >= t {
						t = track[list[i]] + 1
					}
				}
			}
			if weighed {
				clear(taken)
				for _, i := range clashes[at] {
					taken[track[list[i]]] = true
				}
				for taken[t] {
					t++
				}
			} else {
				for t < len(onTrack) && slices.ContainsFunc(onTrack[t], func(other int) bool { return clash(k, other) }) {
					t++
				}
				for len(onTrack) <= t {
					onTrack = append(onTrack, nil)
				}
				onTrack[t] = append(onTrack[t], k)
			}
			track[k] = t
			if t >= rows {
				rows = t + 1
			}
		}
		first := make([]int, count)
		last := make([]int, count)
		for i := range first {
			first[i], last[i] = rows, -1
		}
		for i, k := range list {
			root := find(i)
			first[root] = min(first[root], track[k])
			last[root] = max(last[root], track[k])
		}
		for i, k := range list {
			root := find(i)
			levels[k] = last[root] - first[root] + 1
			if curved {
				track[k] -= first[root]
			}
		}
		return rows
	}

	// near says whether a and b come nearer than limit as plain curves across a gap that leaves
	// them height to turn in: between points along each and the other drawn as a run of short
	// lines.
	sample := func(k int, h float64, out []float64) {
		c := Ease(to[k]-from[k], h)
		for i := 0; i <= samples; i++ {
			t := float64(i) / samples
			u := 1 - t
			out[2*i] = from[k] + float64(float64(float64((to[k]-from[k])*t)*t)*(3-float64(2*t)))
			a := float64(3 * c * t * u * u)
			b := float64(3 * (1 - c) * t * t * u)
			out[2*i+1] = h * (a + b + float64(t*t*t))
		}
	}
	// Pele shares these between layouts; here each has its own, as layouts may run at once.
	lineA := make([]float64, 2*samples+2)
	lineB := make([]float64, 2*samples+2)
	reaches := func(points, line []float64, limit float64) bool {
		// A square of a distance that is this far past the square of the limit is past it,
		// rounding and all.
		bound := limit * limit * (1 + 1e-9)
		for i := 0; i <= samples; i++ {
			px := points[2*i]
			py := points[2*i+1]
			for j := 0; j < samples; j++ {
				dx := line[2*j+2] - line[2*j]
				dy := line[2*j+3] - line[2*j+1]
				t := math.Max(0, math.Min(1, (float64((px-line[2*j])*dx)+float64((py-line[2*j+1])*dy))/or(float64(dx*dx)+float64(dy*dy), 1)))
				ex := px - line[2*j] - float64(t*dx)
				ey := py - line[2*j+1] - float64(t*dy)
				if float64(ex*ex)+float64(ey*ey) < bound && jsmath.Hypot(ex, ey) < limit {
					return true
				}
			}
		}
		return false
	}
	near := func(a, b int, height, limit float64) bool {
		h := math.Max(height/2, height-2*levelLead)
		sample(a, h, lineA)
		sample(b, h, lineB)
		return reaches(lineA, lineB, limit) || reaches(lineB, lineA, limit)
	}

	for _, r := range ranks {
		all := byRank[r]
		if !curved {
			trackCount[r] = lay(all)
			continue
		}
		if len(all) > maxWeighed {
			continue
		}
		// The pieces that turn onto levels: first those that run far across and have another
		// such piece to be told apart from, then any two that would pass too near each other as
		// plain curves.
		var far []int
		for _, k := range all {
			if wide[k] {
				far = append(far, k)
			}
		}
		lay(far)
		var level orderedSet
		for _, k := range far {
			if levels[k] > 1 {
				level.add(k)
			}
		}
		pairs := nearPairs(all)
		for p := 0; p < len(pairs); p += 2 {
			a := all[pairs[p]]
			b := all[pairs[p+1]]
			if (level.has(a) && level.has(b)) || !clash(a, b) || (from[a] < to[a]) != (from[b] < to[b]) {
				continue
			}
			limit := float64(levelTouch)
			if wide[a] || wide[b] {
				limit = levelClear
			}
			if near(a, b, room[r], limit) {
				level.add(a)
				level.add(b)
			}
		}
		for _, k := range far {
			track[k] = -1
			levels[k] = 0
		}
		if len(level.list) == 0 {
			continue
		}
		crossing := 0
		alongside := 0
		// The pieces that could cross one of those that turn.
		beside := make(map[int]bool)
		for p := 0; p < len(pairs); p += 2 {
			a := all[pairs[p]]
			b := all[pairs[p+1]]
			if level.has(a) && level.has(b) {
				if !overlap(a, b) {
					continue
				}
				if (from[a] < to[a]) != (from[b] < to[b]) {
					crossing++
				} else if !mates(a, b) {
					alongside++
				}
			} else if (level.has(a) || level.has(b)) && meet(a, b) {
				beside[a] = true
				beside[b] = true
			}
		}
		if crossing > alongside {
			for _, k := range all {
				if (HI[k]-LO[k])*crossedSlope > tall[r] {
					tall[r] = (HI[k] - LO[k]) * crossedSlope
				}
			}
			continue
		}
		// With them go the short pieces that could cross one of them, so that each has its place
		// among the levels and crosses none.
		var members []int
		for _, k := range all {
			if level.has(k) || (!wide[k] && beside[k]) {
				members = append(members, k)
			}
		}
		lay(members)
		most := 0
		for _, k := range members {
			if levels[k] > most {
				most = levels[k]
			}
		}
		trackCount[r] = most
	}
}

// orderedSet is a set of numbers that lists them in the order they were added, as a
// JavaScript Set does.
type orderedSet struct {
	list []int
	in   map[int]bool
}

func (s *orderedSet) add(k int) {
	if s.in == nil {
		s.in = make(map[int]bool)
	}
	if !s.in[k] {
		s.in[k] = true
		s.list = append(s.list, k)
	}
}

func (s *orderedSet) has(k int) bool { return s.in[k] }

// straighten moves the bends in ranks that hold only bends onto the straight line between the
// nearest points of their edges that are held in place, and marks the ones that can be left out
// of the route, as Pele's straighten does. firstDummy and span give each edge's run of bends,
// one in every rank. sep is the room kept between the bends of twin edges, or 0 to leave those
// out as well.
func straighten(g *graph, xs []float64, empty []bool, firstDummy, span []int, sep float64, onLine []bool) {
	maxRank, EF, ET, EFD, ETD := g.maxRank, g.EF, g.ET, g.EFD, g.ETD
	succStart, succE, predStart, predE, layerStart, order := g.succStart, g.succE, g.predStart, g.predE, g.layerStart, g.order

	want := make([]float64, len(xs))
	// The points that each bend's edge runs between, across the ranks that hold only bends.
	from := make([]float64, len(xs))
	to := make([]float64, len(xs))
	for ei, d0 := range firstDummy {
		if d0 < 0 {
			continue
		}
		end := d0 + span[ei] - 1
		// A bend's rank is its place in the run, counted from the rank after the edge's first node.
		base := g.RANK[d0]
		for d := d0; d < end; d++ {
			if !empty[base+d-d0] {
				continue
			}
			last := d
			for last+1 < end && empty[base+last+1-d0] {
				last++
			}
			above := predE[predStart[d]]
			below := succE[succStart[last]]
			x0 := xs[EF[above]] + EFD[above]
			x1 := xs[ET[below]] + ETD[below]
			steps := float64(last - d + 2)
			for k := d; k <= last; k++ {
				want[k] = x0 + ((x1-x0)*float64(k-d+1))/steps
				from[k] = x0
				to[k] = x1
			}
			d = last
		}
	}

	// Nothing in such a rank is in a bend's way, so each one goes onto its line and out of the
	// route. Only twins stay: bends of edges that run between the same two points, which would
	// otherwise be drawn as one. They take places a little apart, around the line.
	var twins, row []int
	for r := 1; r < maxRank; r++ {
		if !empty[r] {
			continue
		}
		row = append(row[:0], order[layerStart[r]:layerStart[r+1]]...)
		slices.SortStableFunc(row, func(a, b int) int {
			return sign(or(from[a]-from[b], or(to[a]-to[b], float64(a-b))))
		})
		for i := 0; i < len(row); {
			twins = twins[:0]
			j := i
			for ; j < len(row) && math.Abs(from[row[j]]-from[row[i]]) < 1 && math.Abs(to[row[j]]-to[row[i]]) < 1; j++ {
				twins = append(twins, row[j])
			}
			// The twins keep the order the rank gave them, so that two edges do not swap sides on
			// the way.
			slices.SortStableFunc(twins, func(a, b int) int { return g.pos[a] - g.pos[b] })
			for k, d := range twins {
				xs[d] = want[d] + float64((float64(k)-float64(len(twins)-1)/2)*sep)
				if len(twins) == 1 || sep == 0 {
					onLine[d] = true
				}
			}
			i = j
		}
	}
}

// pastLabels leaves out the bends that need not be there, as Pele's pastLabels does. In a rank
// that holds labels and bends but no nodes, a bend is there to take its edge straight through
// between the labels. Where the edge would pass clear of every label anyway, the bend is left
// out, and the edge turns in the whole height between the ranks on either side.
func pastLabels(g *graph, xs []float64, firstDummy, span []int, clear float64, onLine []bool) {
	maxRank, W, KIND, RANK, EF, ET, EFD, ETD := g.maxRank, g.W, g.KIND, g.RANK, g.EF, g.ET, g.EFD, g.ETD
	succStart, succE, predStart, predE, layerStart, order := g.succStart, g.succE, g.predStart, g.predE, g.layerStart, g.order
	labelled := make([]bool, maxRank+1)
	any := false
	for r := 1; r < maxRank; r++ {
		labels := false
		others := false
		for i := layerStart[r]; i < layerStart[r+1]; i++ {
			kind := KIND[order[i]]
			if kind == kindLabel {
				labels = true
			} else if kind != kindDummy {
				others = true
			}
		}
		if labels && !others {
			labelled[r] = true
			any = true
		}
	}
	if !any {
		return
	}
	for ei, d0 := range firstDummy {
		if d0 < 0 {
			continue
		}
		for d, end := d0, d0+span[ei]-1; d < end; d++ {
			if KIND[d] != kindDummy || !labelled[RANK[d]] {
				continue
			}
			// The points on either side that the edge is held at, past any bends already left out.
			first := d
			last := d
			for first > d0 && onLine[first-1] {
				first--
			}
			for last+1 < end && onLine[last+1] {
				last++
			}
			above := predE[predStart[first]]
			below := succE[succStart[last]]
			from := xs[EF[above]] + EFD[above]
			to := xs[ET[below]] + ETD[below]
			lo := math.Min(from, to) - clear
			hi := math.Max(from, to) + clear
			free := true
			r := RANK[d]
			for i := layerStart[r]; free && i < layerStart[r+1]; i++ {
				label := order[i]
				if KIND[label] == kindLabel && xs[label]+W[label]/2 > lo && xs[label]-W[label]/2 < hi {
					free = false
				}
			}
			if !free {
				continue
			}
			onLine[d] = true
			// The bends left out between the two points lie on the line between them.
			steps := float64(last - first + 2)
			for k := first; k <= last; k++ {
				xs[k] = from + ((to-from)*float64(k-first+1))/steps
			}
		}
	}
}
