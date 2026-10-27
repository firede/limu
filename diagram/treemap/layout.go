package treemap

import (
	"math"
	"sort"

	"github.com/firede/limu/internal/jsmath"
)

// box is where a node of the treemap goes, Pele's Box.
type box struct {
	node   *Node
	parent *box
	// series is the place among the top-level nodes of the branch the box belongs to, which
	// picks its color.
	series int
	// value is a leaf's own value, or the sum of the leaves under a section.
	value    float64
	children []*box
	// headed says whether a section keeps room for its header, and header is that room. A
	// section too small for one gives its whole box to its children.
	headed     bool
	header     float64
	x, y, w, h float64
}

// insets are Pele's Insets: the gap between boxes, and the room a section keeps for its
// header, or for a short one when the full one leaves its children too little, and about its
// children on the other three sides.
type insets struct {
	gap, header, short, side float64
}

// clamp is v within low and high, as Pele's clamp gives it: low where v is not at least low.
func clamp(v, low, high float64) float64 {
	if v >= low {
		return min(v, high)
	}
	return low
}

// newBox is Pele's box: a leaf's value counts when it is a positive number.
func newBox(node *Node, parent *box, series int) *box {
	value := 0.0
	if !node.Section && !math.IsInf(node.Value, 0) && !math.IsNaN(node.Value) && node.Value > 0 {
		value = node.Value
	}
	return &box{node: node, parent: parent, series: series, value: value}
}

// squarify lays boxes out in the rectangle as one squarified treemap, as Pele's squarify
// does: rows are added along the shorter side, and a row stops growing when another box would
// make its boxes less square.
func squarify(boxes []*box, total, x, y, w, h, gap float64) {
	rest := total
	from := 0
	for from < len(boxes) {
		scale := (w * h) / rest
		side := min(w, h)
		sum := float64(boxes[from].value * scale)
		largest := sum
		worst := max((side*side*largest)/(sum*sum), (sum*sum)/(side*side*largest))
		to := from + 1
		for to < len(boxes) {
			smallest := float64(boxes[to].value * scale)
			next := sum + smallest
			ratio := max((side*side*largest)/(next*next), (next*next)/(side*side*smallest))
			if ratio > worst {
				break
			}
			worst = ratio
			sum = next
			to++
		}

		thickness := sum / side
		offset := 0.0
		for i := from; i < to; i++ {
			b := boxes[i]
			length := side - offset
			if i != to-1 {
				length = (b.value * scale) / thickness
			}
			bx, by := x, y+offset
			across, down := thickness, length
			if w < h {
				bx, by = x+offset, y
				across, down = length, thickness
			}
			// The gap is left out where it would leave the box no size.
			gx, gy := 0.0, 0.0
			if jsmath.Round(bx+across-gap/2) > jsmath.Round(bx+gap/2) {
				gx = gap / 2
			}
			if jsmath.Round(by+down-gap/2) > jsmath.Round(by+gap/2) {
				gy = gap / 2
			}
			b.x = jsmath.Round(bx + gx)
			b.y = jsmath.Round(by + gy)
			b.w = jsmath.Round(bx+across-gx) - b.x
			b.h = jsmath.Round(by+down-gy) - b.y
			offset += length
			rest -= b.value
		}
		if w >= h {
			x += thickness
			w -= thickness
		} else {
			y += thickness
			h -= thickness
		}
		from = to
	}
}

// layoutTreemap gives the boxes of positive value, those with no size among them, parents
// before their children, as Pele's layoutTreemap does. A box is kept within the room its
// parent has for it.
func layoutTreemap(roots []*Node, width, height float64, in insets) []*box {
	var all []*box
	top := make([]*box, len(roots))
	for i, node := range roots {
		top[i] = newBox(node, nil, i)
	}
	all = append(all, top...)
	for i := 0; i < len(all); i++ {
		parent := all[i]
		for _, node := range parent.node.Children {
			child := newBox(node, parent, parent.series)
			parent.children = append(parent.children, child)
			all = append(all, child)
		}
	}
	for i := len(all) - 1; i >= 0; i-- {
		if b := all[i]; b.parent != nil {
			b.parent.value += b.value
		}
	}

	gap, header, short, side := in.gap, in.header, in.short, in.side
	var placed []*box
	place := func(children []*box, x, y, w, h float64) {
		var shown []*box
		for _, b := range children {
			if b.value > 0 {
				shown = append(shown, b)
			}
		}
		// The values are positive, and only finite ones are laid out, so b.value - a.value
		// orders them as a comparison does.
		sort.SliceStable(shown, func(i, j int) bool { return shown[i].value > shown[j].value })
		total := 0.0
		for _, b := range shown {
			total += b.value
		}
		if len(shown) == 0 || math.IsInf(total, 0) || math.IsNaN(total) {
			return
		}
		w = max(0, w)
		h = max(0, h)
		if w > 0 && h > 0 {
			squarify(shown, total, x-gap/2, y-gap/2, w+gap, h+gap, gap)
		}
		for _, b := range shown {
			x0 := clamp(b.x, x, x+w)
			y0 := clamp(b.y, y, y+h)
			b.w = clamp(b.x+b.w, x0, x+w) - x0
			b.h = clamp(b.y+b.h, y0, y+h) - y0
			b.x = x0
			b.y = y0
			placed = append(placed, b)
		}
	}

	place(top, 0, 0, width, height)
	for i := 0; i < len(placed); i++ {
		b := placed[i]
		if len(b.children) == 0 {
			continue
		}
		top := short
		if b.h-header-side >= 12 {
			top = header
		}
		b.headed = b.w > max(24, float64(2*side)) && b.h-top-side >= 8
		if b.headed {
			b.header = top
			place(b.children, b.x+side, b.y+top, b.w-float64(2*side), b.h-top-side)
		} else {
			b.header = 0
			place(b.children, b.x, b.y, b.w, b.h)
		}
	}
	return placed
}
