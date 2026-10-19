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
	value      float64
	children   []*box
	x, y, w, h float64
}

// insets are Pele's Insets: the gap between boxes, and the room a section keeps for its
// header and about its children on the other three sides.
type insets struct {
	gap, header, side float64
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
			x0 := jsmath.Round(bx + gap/2)
			y0 := jsmath.Round(by + gap/2)
			b.x = x0
			b.y = y0
			b.w = jsmath.Round(bx+across-gap/2) - x0
			b.h = jsmath.Round(by+down-gap/2) - y0
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

// layoutTreemap gives every box with a size, parents before their children, as Pele's
// layoutTreemap does.
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

	gap, header, side := in.gap, in.header, in.side
	var placed []*box
	place := func(children []*box, x, y, w, h float64) {
		if w <= 0 || h <= 0 {
			return
		}
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
		squarify(shown, total, x-gap/2, y-gap/2, w+gap, h+gap, gap)
		for _, b := range shown {
			if b.w > 0 && b.h > 0 {
				placed = append(placed, b)
			}
		}
	}

	place(top, 0, 0, width, height)
	for i := 0; i < len(placed); i++ {
		b := placed[i]
		if len(b.children) > 0 {
			place(b.children, b.x+side, b.y+header, b.w-float64(2*side), b.h-header-side)
		}
	}
	return placed
}
