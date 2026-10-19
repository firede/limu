package layout

import (
	"math"
	"slices"
)

// The layered layout is Pele's layout/layered.ts: a Sugiyama-style layout of one flat graph.
// Ranks run down the y axis; callers rotate the result for other directions.
//
// What the layout of edges aims for, most important first. Where two aims pull apart, the one
// higher up decides, and each names where it is carried out.
//
//  1. Lines can be told apart. No two touch or run along each other, and none runs through a
//     node or a label. Ends on one side of a node have places of their own (spread), and
//     curves that would pass within a few pixels turn at levels of their own (assignTracks).
//  2. Lines cross as little as the order of the nodes allows (reduceCrossings, and the order
//     of the levels in assignTracks).
//  3. An edge takes the shortest and simplest way: straight if it can, else one curve, else a
//     curve with a level run. One that passes several ranks runs in line with one of its ends
//     and turns once, at the other (alignLabels, straighten). Ends slide along their sides,
//     and nodes move a little, where that makes an edge straight (settle, snap).
//  4. An edge meets the middle of a side. An end with a side to itself stays there, and ends
//     that share a side sit around the middle in the order of where they lead. An end moves off
//     the middle only for the aims above (settle).
//  5. The drawing is balanced, and what is alike is drawn alike. A label sits midway along the
//     line that shows, the curves either side of it have much the same room to turn in, and a
//     gap whose curves mostly cross has them all as the same plain curve rather than each on a
//     level of its own (HeadRoom in Layered, the crossed gaps in assignTracks).
//  6. An edge leaves and reaches a node square to it, and an arrowhead points straight at the
//     node (Pele's routePath in svg/edges.ts, which draws the routes).
//  7. Nodes keep the order they were written in, where nothing above says otherwise.
//  8. The drawing stays compact. A gap between ranks grows only to hold the levels in it, or
//     to let curves that cross do so at a clear angle.
//
// The port keeps JavaScript's arithmetic: the same operations on float64 in the same order,
// stable sorts with the comparators Pele gives, and insertion order wherever Pele iterates a
// Map or a Set.

// Kind is what a node of the layered layout stands for, as Pele's Kind is.
type Kind uint8

const (
	KindNode      Kind = iota // a node of the diagram
	KindDummy                 // a bend of an edge, in a rank that the edge passes
	KindLabel                 // the label of an edge, in a rank that the edge passes
	KindStartPort             // where an edge passes the border of a group on the side its flow starts from
	KindEndPort               // where an edge passes the border of a group on the side its flow ends at
)

// LNode is a node of the layered layout, as Pele's LNode is. Layered sets X, Y (its middle)
// and Rank.
type LNode struct {
	W, H float64
	Kind Kind
	// Pin keeps a node at one end of its rank: below zero at the start, above zero at the end.
	Pin float64
	// Span is how much of the side that edges meet may be shared out between them; 0 keeps them
	// all at its middle.
	Span float64
	X, Y float64
	Rank int
}

// LEdge is an edge of the layered layout, as Pele's LEdge is. Layered sets Reversed, Points
// (x, y pairs from the tail to the head), and LabelX and LabelY for an edge with a label; where
// nodes spread the ends of their edges, it sets TailDx and HeadDx as well.
type LEdge struct {
	Tail, Head     int
	Minlen         float64
	LabelW, LabelH float64
	// Where the ends meet their nodes, from the middle of the side.
	TailDx, HeadDx float64
	// An end that is held where its offset puts it, and takes no part in sharing out a node's side.
	TailFixed, HeadFixed bool
	Reversed             bool
	Points               []float64
	LabelX, LabelY       float64
}

// LayeredOptions are Pele's LayeredOptions.
type LayeredOptions struct {
	NodeSep float64
	EdgeSep float64
	RankSep float64
	PortSep float64
	// Tracks routes edges at right angles, each on a track of its own across the gap between
	// two ranks.
	Tracks bool
	// Straight draws every edge straight from point to point: none is given a level to turn onto.
	Straight bool
	// Apart says that the caller sets the ends of the edges on one side of a node apart itself,
	// so two edges between the same two nodes need no bend to keep them from being drawn as one.
	Apart bool
	// HeadRoom is the room that the marker at the head of an edge takes. A rank of labels whose
	// edges all head the same way sits half of this nearer their tails, so that each label is
	// midway along the line that shows, and the curves either side of it have the same room to
	// turn in.
	HeadRoom float64
}

// LayeredResult is the size of a layered layout, as Pele's LayeredResult is.
type LayeredResult struct {
	Width, Height float64
}

// NewLNode is a node of the given size and kind, as Pele's lnode makes.
func NewLNode(w, h float64, kind Kind) LNode {
	return LNode{W: w, H: h, Kind: kind}
}

// NewLEdge is an edge from tail to head, as Pele's ledge makes; Pele's defaults are a minlen
// of 1 and no label.
func NewLEdge(tail, head int, minlen, labelW, labelH float64) LEdge {
	return LEdge{Tail: tail, Head: head, Minlen: minlen, LabelW: labelW, LabelH: labelH, Points: []float64{}}
}

const maxSweeps = 24

// An edge gets a bend point in every rank it skips. Past this many, the longest edges are
// drawn as one direct curve instead, which keeps huge graphs from exhausting time and memory.
const maxDummies = 20000

// The room between two edges that meet one side of a node, and the least that keeps their
// arrowheads apart.
const (
	portPitch = 16
	minPitch  = 8
)

// Where edges are not given levels, a gap between ranks grows to keep the edges that cross it
// at least this steep, up to this many times its usual height. Shallower than that, the edges
// of a wide rank are hard to tell apart.
const (
	minSlope = 0.125
	maxGap   = 3
)

// Tracks: the room kept clear between a rank and the nearest track, the room between two
// tracks when the gap has to grow to hold them, and the most they spread to when it has room
// to spare.
const (
	trackMargin = 18
	trackPitch  = 10
	trackSpread = 14
)

// A curve is one plain S unless it runs more than this many times as far across as the gap is
// high, and so does another that it would run along. Then each turns onto a level of its own
// to run across on. The levels of a gap keep this far from its ranks, to leave room to turn
// in, and from each other.
const (
	levelWide   = 2
	levelMargin = 28
	levelPitch  = 12
)

// Two plain curves that would pass too near each other turn at levels of their own as well:
// nearer than levelClear where one of them runs far across, since it stays near the other for
// a long way, and otherwise nearer than levelTouch. levelLead is the room a plain curve is
// taken to keep straight at each end of its gap.
const (
	levelLead  = 12
	levelClear = 8
	levelTouch = 4
)

// Levels tell apart curves that run alongside each other. Where the curves of a gap cross each
// other more often than that, runs along levels only make the crossings long and shallow, so
// each curve is one plain S instead, and the gap is made this many times as tall as its widest
// curve runs across (and no taller than maxGap allows), for them to cross at a clear angle.
const crossedSlope = 0.5

// How far out of line two points may be and still count as in line.
const inLine = 0.5

// A node, with whatever it is already in line with, moves this far at most to bring one more
// of its edges into line.
const snapReach = 12

// An end that cannot be brought into line leans towards where its edge goes, but keeps this
// far from the ends of the part of its side that edges spread along. On a narrow node that
// leaves it in the middle.
const leanClear = 40

// Past this many edges across one gap, tracks are given out in a plain order: weighing every
// pair of edges against each other would take too long.
const maxWeighed = 400

// End is what the end of a segment is to the node it meets, as Pele's End is.
type End uint8

const (
	endHeld End = iota
	endLeaves
	endArrives
)

func isDummy(kind Kind) bool {
	return kind == KindDummy || kind == KindLabel
}

// sortRange sorts list[from:to] by key, and by value where the keys are equal, as Pele's
// sortRange does: by insertion, since the runs are short.
func sortRange(list []int, from, to int, key []int) {
	for i := from + 1; i < to; i++ {
		item := list[i]
		k := key[item]
		j := i - 1
		for j >= from && (key[list[j]] > k || (key[list[j]] == k && list[j] > item)) {
			list[j+1] = list[j]
			j--
		}
		list[j+1] = item
	}
}

// graph is Pele's Graph: the graph that gets ordered and positioned, with every edge cut into
// rank-to-rank segments.
type graph struct {
	n, real, maxRank int
	W                []float64
	KIND             []Kind
	RANK             []int
	// Segment endpoints, and where each segment attaches on them.
	EF, ET            []int
	EFD, ETD          []float64
	succStart, succE  []int
	predStart, predE  []int
	layerStart, order []int
	pos               []int
	offsets           bool
}

// pieces is Pele's Pieces: the pieces of edges that each cross one gap: where they leave the
// rank above and reach the one below, the rank above, the bend they end at (or -1 at the edge's
// last node), and the track they run across on (or -1 when they have none). Where only some
// pieces are given a level to turn onto, wide marks those that run far across, levels is how
// many tracks the group a piece belongs to takes, or 0 for a piece that needs none, and each
// piece's track counts from its group's first. A piece that is not wide curves from the level
// above its own.
type pieces struct {
	from, to []float64
	rank     []int
	lower    []int
	track    []int
	levels   []int
	wide     []bool
}

// Layered lays out a flat graph, as Pele's layered does: it sets where every node is, and the
// points every edge runs through, and returns the size of the whole. Ranks run down.
// It takes no edge from a node to itself: Pele's typed arrays make nonsense of one, and here
// it would run past the end of a slice. Compound keeps those out and places their labels itself.
func Layered(nodes []LNode, edges []LEdge, opt LayeredOptions) LayeredResult {
	real := len(nodes)
	if real == 0 {
		return LayeredResult{}
	}
	m := len(edges)

	// Every edge gets a middle rank of its own when labels or parallel edges need the room.
	step := 1
	if m > 0 {
		seen := make(map[int]struct{})
		for i := range edges {
			e := &edges[i]
			a := min(e.Tail, e.Head)
			b := max(e.Tail, e.Head)
			key := a*real + b
			if _, ok := seen[key]; e.LabelW > 0 || ok {
				step = 2
				break
			}
			seen[key] = struct{}{}
		}
	}

	Orient(nodes, edges)
	Rank(nodes, edges, step)

	span := make([]int, m)
	dummies := 0
	for ei := range edges {
		e := &edges[ei]
		span[ei] = absInt(nodes[e.Head].Rank - nodes[e.Tail].Rank)
		if span[ei] > 1 {
			dummies += span[ei] - 1
		}
	}
	direct := make([]bool, m)
	if dummies > maxDummies {
		var longest []int
		for ei := 0; ei < m; ei++ {
			if span[ei] > 1 {
				longest = append(longest, ei)
			}
		}
		slices.SortStableFunc(longest, func(a, b int) int {
			if c := span[b] - span[a]; c != 0 {
				return c
			}
			return a - b
		})
		for _, ei := range longest {
			if dummies <= maxDummies {
				break
			}
			direct[ei] = true
			dummies -= span[ei] - 1
		}
	}

	n := real + dummies
	W := make([]float64, n)
	H := make([]float64, n)
	KIND := make([]Kind, n)
	RANK := make([]int, n)
	maxRank := 0
	for i := 0; i < real; i++ {
		node := &nodes[i]
		W[i] = node.W
		H[i] = node.H
		KIND[i] = node.Kind
		RANK[i] = node.Rank
		if node.Rank > maxRank {
			maxRank = node.Rank
		}
	}

	// The graph that gets ordered and positioned: every edge cut into rank-to-rank segments.
	segments := 0
	for ei := 0; ei < m; ei++ {
		if !direct[ei] {
			segments += span[ei]
		}
	}
	EF := make([]int, segments)
	ET := make([]int, segments)
	EFD := make([]float64, segments)
	ETD := make([]float64, segments)
	firstDummy := filled(m, -1)
	// Which way the edges with a label in each rank head: 1 down, 2 up, 3 both.
	heads := make([]uint8, maxRank+1)
	labelNode := filled(m, -1)
	spans := false
	for i := 0; i < real; i++ {
		if nodes[i].Span > 0 && nodes[i].Kind == KindNode {
			spans = true
		}
	}
	// What each segment's end is to the node it meets, for the ends that may move along the
	// node's side.
	var EFK, ETK []End
	if spans {
		EFK = make([]End, segments)
		ETK = make([]End, segments)
	}
	firstSeg := filled(m, -1)
	lastSeg := filled(m, -1)
	offsets := false
	{
		next := real
		ce := 0
		for ei := 0; ei < m; ei++ {
			if direct[ei] {
				continue
			}
			e := &edges[ei]
			a, b, aDx, bDx := e.Tail, e.Head, e.TailDx, e.HeadDx
			if e.Reversed {
				a, b, aDx, bDx = e.Head, e.Tail, e.HeadDx, e.TailDx
			}
			if aDx != 0 || bDx != 0 {
				offsets = true
			}
			ra := RANK[a]
			rb := RANK[b]
			labelRank := -1
			if e.LabelW > 0 {
				// The label goes halfway along. An edge that passes ranks of nodes on its way has
				// its label in one of those, beside the nodes, where it takes no height of its
				// own: the one halfway, or of the two either side of halfway, the one nearer the
				// node the edge starts from, which is the lower when the edge runs against the
				// flow. An edge between two neighbouring ranks has its label in the rank between
				// them.
				labelRank = ra + ((rb - ra) >> 1)
				if step == 2 && rb-ra >= 4 {
					if labelRank&1 == 1 {
						labelRank += reversedStep(e.Reversed)
					}
				} else if step == 2 && labelRank&1 == 0 {
					labelRank += reversedStep(e.Reversed)
				}
			}
			prev := a
			if rb-ra > 1 {
				firstDummy[ei] = next
			}
			firstSeg[ei] = ce
			if EFK != nil && !pick(e.Reversed, e.HeadFixed, e.TailFixed) {
				EFK[ce] = endLeaves
				if e.Reversed {
					EFK[ce] = endArrives
				}
			}
			for r := ra + 1; r < rb; r++ {
				d := next
				next++
				RANK[d] = r
				if r == labelRank {
					W[d] = e.LabelW
					H[d] = e.LabelH
					KIND[d] = KindLabel
					labelNode[ei] = d
					if e.Reversed {
						heads[r] |= 2
					} else {
						heads[r] |= 1
					}
				} else {
					KIND[d] = KindDummy
				}
				EF[ce] = prev
				ET[ce] = d
				if prev == a {
					EFD[ce] = aDx
				}
				ce++
				prev = d
			}
			EF[ce] = prev
			ET[ce] = b
			if prev == a {
				EFD[ce] = aDx
			}
			ETD[ce] = bDx
			if ETK != nil && !pick(e.Reversed, e.TailFixed, e.HeadFixed) {
				ETK[ce] = endArrives
				if e.Reversed {
					ETK[ce] = endLeaves
				}
			}
			lastSeg[ei] = ce
			ce++
		}
	}

	succStart := make([]int, n+1)
	predStart := make([]int, n+1)
	for e := 0; e < segments; e++ {
		succStart[EF[e]+1]++
		predStart[ET[e]+1]++
	}
	for i := 0; i < n; i++ {
		succStart[i+1] += succStart[i]
		predStart[i+1] += predStart[i]
	}
	succE := make([]int, segments)
	predE := make([]int, segments)
	{
		sf := slices.Clone(succStart[:n])
		pf := slices.Clone(predStart[:n])
		for e := 0; e < segments; e++ {
			succE[sf[EF[e]]] = e
			sf[EF[e]]++
			predE[pf[ET[e]]] = e
			pf[ET[e]]++
		}
	}

	layerStart := make([]int, maxRank+2)
	for i := 0; i < n; i++ {
		layerStart[RANK[i]+1]++
	}
	for r := 0; r <= maxRank; r++ {
		layerStart[r+1] += layerStart[r]
	}
	order := make([]int, n)
	pos := make([]int, n)

	// Initial order: depth-first from the top so that subtrees start out contiguous.
	{
		byRank := make([]int, n)
		fill := slices.Clone(layerStart[:maxRank+1])
		for i := 0; i < n; i++ {
			byRank[fill[RANK[i]]] = i
			fill[RANK[i]]++
		}
		copy(fill, layerStart[:maxRank+1])
		visited := make([]bool, n)
		var stack []int
		for k := 0; k < n; k++ {
			if visited[byRank[k]] {
				continue
			}
			stack = append(stack, byRank[k])
			for len(stack) > 0 {
				v := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				if visited[v] {
					continue
				}
				visited[v] = true
				order[fill[RANK[v]]] = v
				fill[RANK[v]]++
				for j := succStart[v+1] - 1; j >= succStart[v]; j-- {
					w := ET[succE[j]]
					if !visited[w] {
						stack = append(stack, w)
					}
				}
			}
		}
	}

	pinned := false
	for i := 0; i < real; i++ {
		if nodes[i].Pin != 0 {
			pinned = true
		}
	}
	if pinned {
		index := make([]int, n)
		pinOf := func(v int) float64 {
			if v < real {
				return nodes[v].Pin
			}
			return 0
		}
		for r := 0; r <= maxRank; r++ {
			layer := order[layerStart[r]:layerStart[r+1]]
			for i, v := range layer {
				index[v] = i
			}
			slices.SortStableFunc(layer, func(a, b int) int {
				return sign(or(pinOf(a)-pinOf(b), float64(index[a]-index[b])))
			})
		}
	}
	setPositions := func() {
		for r := 0; r <= maxRank; r++ {
			s := layerStart[r]
			for i, e := s, layerStart[r+1]; i < e; i++ {
				pos[order[i]] = i - s
			}
		}
	}
	setPositions()

	g := &graph{n: n, real: real, maxRank: maxRank, W: W, KIND: KIND, RANK: RANK, EF: EF, ET: ET, EFD: EFD, ETD: ETD,
		succStart: succStart, succE: succE, predStart: predStart, predE: predE, layerStart: layerStart, order: order, pos: pos, offsets: offsets}

	if segments > 0 && maxRank > 0 {
		reduceCrossings(g, nodes)
		setPositions()
	}

	if EFK != nil && ETK != nil {
		spread(g, nodes, EFK, ETK)
	}

	xs := position(g, opt)

	empty := emptyRanks(g)
	if EFK != nil && ETK != nil {
		free := freeEnds(g, EFK, ETK)
		alignLabels(g, nodes, xs, free, opt, empty)
		settle(g, nodes, xs, EFK, ETK, free, empty)
		snap(g, xs, empty, opt)
		for ei := 0; ei < m; ei++ {
			if direct[ei] {
				continue
			}
			e := &edges[ei]
			aDx := EFD[firstSeg[ei]]
			bDx := ETD[lastSeg[ei]]
			e.TailDx, e.HeadDx = aDx, bDx
			if e.Reversed {
				e.TailDx, e.HeadDx = bDx, aDx
			}
		}
	}

	// A rank that holds nothing but the bends of edges is only there because some other edge
	// has a label or a twin. Its bends move onto the straight line between the ends of their
	// edges, and one that gets there is left out of the route, so that the edge is drawn as a
	// single curve.
	onLine := make([]bool, n)
	if empty != nil && !opt.Tracks {
		sep := opt.EdgeSep
		if opt.Apart {
			sep = 0
		}
		straighten(g, xs, empty, firstDummy, span, sep, onLine)
	}
	if !opt.Tracks {
		pastLabels(g, xs, firstDummy, span, opt.EdgeSep, onLine)
	}
	// An edge is cut where it passes a rank that holds something, and each piece crosses one
	// gap. With Tracks, every piece that does not run straight gets a track to run across on at
	// right angles. Otherwise, where edges spread along their nodes, a piece gets one only if it
	// needs a level to turn onto. pieceStart gives every edge's run of pieces.
	pieceStart := make([]int, m+1)
	var p *pieces
	if opt.Tracks || (EFK != nil && !opt.Straight) {
		p = &pieces{}
	}
	trackCount := make([]int, maxRank+1)
	if p != nil {
		for ei := 0; ei < m; ei++ {
			pieceStart[ei] = len(p.from)
			if direct[ei] {
				continue
			}
			d0 := firstDummy[ei]
			end := d0
			if d0 >= 0 {
				end = d0 + span[ei] - 1
			}
			// An edge that keeps a bend to stay clear of its twin is drawn through the bend, as
			// it is.
			twin := false
			for d := d0; d < end; d++ {
				// So is one that has a bend left out in a rank of labels: its curve crosses that
				// rank.
				bare := empty != nil && empty[RANK[d]]
				if !opt.Tracks && bare != onLine[d] {
					twin = true
				}
			}
			if twin {
				continue
			}
			x := xs[EF[firstSeg[ei]]] + EFD[firstSeg[ei]]
			rank := RANK[EF[firstSeg[ei]]]
			for d := d0; d < end; d++ {
				if empty != nil && empty[RANK[d]] {
					continue
				}
				p.from = append(p.from, x)
				p.to = append(p.to, xs[d])
				p.rank = append(p.rank, rank)
				p.lower = append(p.lower, d)
				x = xs[d]
				rank = RANK[d]
			}
			p.from = append(p.from, x)
			p.to = append(p.to, xs[ET[lastSeg[ei]]]+ETD[lastSeg[ei]])
			p.rank = append(p.rank, rank)
			p.lower = append(p.lower, -1)
		}
		pieceStart[m] = len(p.from)
	}

	// Ranks that hold only border ports or their pass-through dummies need little room of their own.
	portOnly := func(r int, kind Kind) bool {
		found := false
		for i := layerStart[r]; i < layerStart[r+1]; i++ {
			k := KIND[order[i]]
			if k == kind {
				found = true
			} else if k != KindDummy {
				return false
			}
		}
		return found
	}
	startRanks := 0
	if portOnly(0, KindStartPort) {
		startRanks = step
	}
	endRanks := 0
	if portOnly(maxRank, KindEndPort) {
		endRanks = step
	}
	// The longest way across that any edge travels in the gap below each rank.
	run := make([]float64, maxRank+1)
	for e := 0; e < segments; e++ {
		across := math.Abs(xs[EF[e]] + EFD[e] - xs[ET[e]] - ETD[e])
		if across > run[RANK[EF[e]]] {
			run[RANK[EF[e]]] = across
		}
	}
	bandTop := make([]float64, maxRank+1)
	bandBottom := make([]float64, maxRank+1)
	usual := func(r int) float64 {
		if r < startRanks || r >= maxRank-endRanks {
			return opt.PortSep / float64(step)
		}
		return opt.RankSep / float64(step)
	}
	// The room under each rank. Edges that are not cut into pieces need more where they run far
	// across.
	gapUnder := make([]float64, maxRank+1)
	for r := 0; r < maxRank; r++ {
		if p != nil {
			gapUnder[r] = usual(r)
		} else {
			gapUnder[r] = math.Max(usual(r), math.Min((maxGap*opt.RankSep)/float64(step), run[r]*minSlope))
		}
	}
	// A rank between two ranks of nodes, which holds labels and bends only.
	between := make([]bool, maxRank+1)
	for r := range between {
		between[r] = true
	}
	for i := 0; i < real; i++ {
		between[RANK[i]] = false
	}
	for r := 1; r < maxRank; r++ {
		if opt.Tracks || !between[r] || (heads[r] != 1 && heads[r] != 2) {
			continue
		}
		under := r
		if heads[r] == 1 {
			under = r - 1
		}
		shift := math.Min(opt.HeadRoom/2, gapUnder[under]/2)
		if heads[r] == 1 {
			gapUnder[r-1] += -shift
			gapUnder[r] += shift
		} else {
			gapUnder[r-1] += shift
			gapUnder[r] += -shift
		}
	}
	// The part of its rank that an edge runs straight through at a bend or label. A bare bend
	// beside labels keeps to the middle half, which leaves the curves either side of it more
	// room and still keeps them clear of the labels' corners.
	inset := func(d int) float64 {
		if KIND[d] == KindDummy && between[RANK[d]] && !opt.Tracks {
			return (bandBottom[RANK[d]] - bandTop[RANK[d]]) / 4
		}
		return 0
	}
	// The rank that the pieces leaving each rank arrive at, past any that hold only bends, and
	// the room between the two, which grows to hold the gap's tracks or levels: these keep
	// margin from the ranks, and between leastPitch and mostPitch from each other.
	margin, leastPitch, mostPitch := float64(levelMargin), float64(levelPitch), float64(levelPitch)
	if opt.Tracks {
		margin, leastPitch, mostPitch = trackMargin, trackPitch, trackSpread
	}
	below := make([]int, maxRank+1)
	if p != nil {
		room := make([]float64, maxRank+1)
		for r := 0; r < maxRank; r++ {
			if empty != nil && empty[r] {
				continue
			}
			next := r + 1
			room[r] = gapUnder[r]
			for next < maxRank && empty != nil && empty[next] {
				room[r] += gapUnder[next]
				next++
			}
			below[r] = next
		}
		for k := range p.from {
			p.wide = append(p.wide, math.Abs(p.from[k]-p.to[k]) > levelWide*room[p.rank[k]])
		}
		tall := make([]float64, maxRank+1)
		assignTracks(p, trackCount, !opt.Tracks, room, tall)
		for r := 0; r < maxRank; r++ {
			if trackCount[r] > 0 {
				gapUnder[r] += math.Max(0, float64(2*margin)+float64(float64(trackCount[r]-1)*leastPitch)-room[r])
			} else if tall[r] > 0 {
				gapUnder[r] += math.Max(0, math.Min(maxGap*opt.RankSep, tall[r])-room[r])
			}
		}
	}
	y := 0.0
	for r := 0; r <= maxRank; r++ {
		h := 0.0
		for i := layerStart[r]; i < layerStart[r+1]; i++ {
			if H[order[i]] > h {
				h = H[order[i]]
			}
		}
		bandTop[r] = y
		bandBottom[r] = y + h
		y += h + gapUnder[r]
	}
	height := y
	// Where a piece runs across its gap, or would on another track of its group: its tracks sit
	// in the middle of the gap, the first at the top.
	trackY := func(piece, onTrack int) float64 {
		r := p.rank[piece]
		top := bandBottom[r]
		bottom := bandTop[below[r]]
		count := p.levels[piece]
		if opt.Tracks {
			count = trackCount[r]
		}
		pitch := 0.0
		if count > 1 {
			pitch = math.Min(mostPitch, (bottom-top-float64(2*margin))/float64(count-1))
		}
		return (top+bottom)/2 + float64((float64(onTrack)-float64(count-1)/2)*pitch)
	}

	minX := math.Inf(1)
	maxX := math.Inf(-1)
	for i := 0; i < n; i++ {
		half := W[i] / 2
		if xs[i]-half < minX {
			minX = xs[i] - half
		}
		if xs[i]+half > maxX {
			maxX = xs[i] + half
		}
	}
	// Where edges spread along their nodes, a rank whose nodes only send edges on stands on one
	// line, and a rank whose nodes only receive them hangs from one, so that the edges of nodes
	// of different heights set out from, or arrive at, the same level. In any other rank the
	// nodes are centered.
	arrives := make([]bool, maxRank+1)
	leaves := make([]bool, maxRank+1)
	for i := 0; i < real; i++ {
		if KIND[i] != KindNode || EFK == nil {
			continue
		}
		if predStart[i+1] > predStart[i] {
			arrives[RANK[i]] = true
		}
		if succStart[i+1] > succStart[i] {
			leaves[RANK[i]] = true
		}
	}
	for i := 0; i < real; i++ {
		node := &nodes[i]
		r := node.Rank
		node.X = xs[i] - minX
		if KIND[i] == KindNode && leaves[r] && !arrives[r] {
			node.Y = bandBottom[r] - node.H/2
		} else if KIND[i] == KindNode && arrives[r] && !leaves[r] {
			node.Y = bandTop[r] + node.H/2
		} else {
			node.Y = (bandTop[r] + bandBottom[r]) / 2
		}
	}

	for ei := 0; ei < m; ei++ {
		e := &edges[ei]
		first, last := &nodes[e.Tail], &nodes[e.Head]
		firstDx, lastDx := e.TailDx, e.HeadDx
		if e.Reversed {
			first, last = last, first
			firstDx, lastDx = lastDx, firstDx
		}
		pts := []float64{first.X + firstDx, first.Y + first.H/2}
		d0 := firstDummy[ei]
		if p != nil && pieceStart[ei+1] > pieceStart[ei] {
			for k := pieceStart[ei]; k < pieceStart[ei+1]; k++ {
				if p.track[k] >= 0 && (opt.Tracks || p.wide[k]) {
					across := trackY(k, p.track[k])
					pts = append(pts, p.from[k]-minX, across, p.to[k]-minX, across)
				} else if p.track[k] > 0 {
					// A short piece among levels runs straight until it is past the level before
					// its own, and turns in one S from there, so that it crosses none of them.
					pts = append(pts, p.from[k]-minX, trackY(k, p.track[k]-1))
				}
				d := p.lower[k]
				if d < 0 {
					continue
				}
				pts = append(pts, xs[d]-minX, bandTop[RANK[d]]+inset(d))
				if bandBottom[RANK[d]] > bandTop[RANK[d]] {
					pts = append(pts, xs[d]-minX, bandBottom[RANK[d]]-inset(d))
				}
			}
		} else if d0 >= 0 {
			for d, end := d0, d0+span[ei]-1; d < end; d++ {
				if onLine[d] {
					continue
				}
				x := xs[d] - minX
				top := bandTop[RANK[d]]
				bottom := bandBottom[RANK[d]]
				pts = append(pts, x, top+inset(d))
				if bottom > top {
					pts = append(pts, x, bottom-inset(d))
				}
			}
		}
		pts = append(pts, last.X+lastDx, last.Y-last.H/2)
		if e.Reversed {
			for i, j := 0, len(pts)-2; i < j; i, j = i+2, j-2 {
				pts[i], pts[j] = pts[j], pts[i]
				pts[i+1], pts[j+1] = pts[j+1], pts[i+1]
			}
		}
		e.Points = pts
		if ln := labelNode[ei]; ln >= 0 {
			e.LabelX = xs[ln] - minX
			e.LabelY = (bandTop[RANK[ln]] + bandBottom[RANK[ln]]) / 2
		} else if e.LabelW > 0 {
			e.LabelX = (pts[0] + pts[len(pts)-2]) / 2
			e.LabelY = (pts[1] + pts[len(pts)-1]) / 2
		}
	}

	return LayeredResult{Width: maxX - minX, Height: height}
}

// reversedStep is the way a label's rank moves off the middle of its edge: down, towards the
// edge's first node when it runs against the flow, else up.
func reversedStep(reversed bool) int {
	if reversed {
		return 1
	}
	return -1
}

// pick is JavaScript's c ? a : b.
func pick[T any](c bool, a, b T) T {
	if c {
		return a
	}
	return b
}

func filled(n, v int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = v
	}
	return s
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Orient reverses the edges that close a cycle, found by depth-first search in declaration
// order, as Pele's orient does. An edge into a start port or out of an end port is reversed
// too.
func Orient(nodes []LNode, edges []LEdge) {
	n := len(nodes)
	m := len(edges)
	start := make([]int, n+1)
	for i := 0; i < m; i++ {
		e := &edges[i]
		e.Reversed = nodes[e.Head].Kind == KindStartPort || nodes[e.Tail].Kind == KindEndPort
		if !e.Reversed {
			start[e.Tail+1]++
		}
	}
	for i := 0; i < n; i++ {
		start[i+1] += start[i]
	}
	out := make([]int, start[n])
	{
		fill := slices.Clone(start[:n])
		for i := 0; i < m; i++ {
			if !edges[i].Reversed {
				out[fill[edges[i].Tail]] = i
				fill[edges[i].Tail]++
			}
		}
	}
	state := make([]uint8, n)
	cursor := slices.Clone(start[:n])
	var stack []int
	for root := 0; root < n; root++ {
		if state[root] != 0 {
			continue
		}
		stack = append(stack, root)
		state[root] = 1
		for len(stack) > 0 {
			v := stack[len(stack)-1]
			if cursor[v] < start[v+1] {
				e := &edges[out[cursor[v]]]
				cursor[v]++
				w := e.Head
				if state[w] == 1 {
					e.Reversed = true
				} else if state[w] == 0 {
					state[w] = 1
					stack = append(stack, w)
				}
			} else {
				state[v] = 2
				stack = stack[:len(stack)-1]
			}
		}
	}
}

// Rank gives every node its rank, as Pele's rank does: the longest path from the top, with
// each edge at least minlen times step ranks long, and then each node moved towards the side
// that has more of its edges. Start ports go a step above the rest, and end ports a step below.
func Rank(nodes []LNode, edges []LEdge, step int) {
	n := len(nodes)
	m := len(edges)
	from := make([]int, m)
	to := make([]int, m)
	length := make([]int, m)
	outStart := make([]int, n+1)
	inStart := make([]int, n+1)
	for i := 0; i < m; i++ {
		e := &edges[i]
		a, b := e.Tail, e.Head
		if e.Reversed {
			a, b = b, a
		}
		from[i] = a
		to[i] = b
		length[i] = toInt32(math.Max(1, e.Minlen) * float64(step))
		outStart[a+1]++
		inStart[b+1]++
	}
	for i := 0; i < n; i++ {
		outStart[i+1] += outStart[i]
		inStart[i+1] += inStart[i]
	}
	outE := make([]int, m)
	inE := make([]int, m)
	{
		of := slices.Clone(outStart[:n])
		inf := slices.Clone(inStart[:n])
		for i := 0; i < m; i++ {
			outE[of[from[i]]] = i
			of[from[i]]++
			inE[inf[to[i]]] = i
			inf[to[i]]++
		}
	}

	r := make([]int, n)
	indeg := make([]int, n)
	for i := 0; i < n; i++ {
		indeg[i] = inStart[i+1] - inStart[i]
	}
	topo := make([]int, n)
	count := 0
	for i := 0; i < n; i++ {
		if indeg[i] == 0 {
			topo[count] = i
			count++
		}
	}
	for q := 0; q < count; q++ {
		v := topo[q]
		for k := outStart[v]; k < outStart[v+1]; k++ {
			ei := outE[k]
			w := to[ei]
			if r[v]+length[ei] > r[w] {
				r[w] = r[v] + length[ei]
			}
			indeg[w]--
			if indeg[w] == 0 {
				topo[count] = w
				count++
			}
		}
	}

	// Longest-path ranking leaves loosely attached nodes far from their neighbours.
	// Move each node toward the side that has more edges.
	for pass := 0; pass < 8; pass++ {
		changed := false
		for q := count - 1; q >= 0; q-- {
			v := topo[q]
			ins := inStart[v+1] - inStart[v]
			outs := outStart[v+1] - outStart[v]
			if ins == 0 && outs == 0 {
				continue
			}
			lo, loSet := 0, false
			hi, hiSet := 0, false
			for k := inStart[v]; k < inStart[v+1]; k++ {
				ei := inE[k]
				if c := r[from[ei]] + length[ei]; !loSet || c > lo {
					lo, loSet = c, true
				}
			}
			for k := outStart[v]; k < outStart[v+1]; k++ {
				ei := outE[k]
				if c := r[to[ei]] - length[ei]; !hiSet || c < hi {
					hi, hiSet = c, true
				}
			}
			next := r[v]
			if outs > ins && hiSet {
				next = hi
			} else if ins > outs && loSet {
				next = lo
			}
			if next != r[v] {
				r[v] = next
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	lo, hi, found := 0, 0, false
	startPorts := false
	for i := 0; i < n; i++ {
		kind := nodes[i].Kind
		if kind == KindStartPort {
			startPorts = true
		}
		if kind == KindStartPort || kind == KindEndPort {
			continue
		}
		if !found || r[i] < lo {
			lo = r[i]
		}
		if !found || r[i] > hi {
			hi = r[i]
		}
		found = true
	}
	for i := 0; i < n; i++ {
		if nodes[i].Kind == KindStartPort {
			r[i] = lo - step
		} else if nodes[i].Kind == KindEndPort {
			r[i] = hi + step
		}
	}
	if startPorts {
		lo -= step
	}
	for i := 0; i < n; i++ {
		nodes[i].Rank = r[i] - lo
	}
}
