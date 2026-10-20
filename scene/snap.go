package scene

import (
	"math"
	"slices"
)

// Snap moves everything in the scene onto a grid of unit: the corners of boxes, the points
// of paths and routes, the tips of markers and the points labels are drawn about. It is for
// a backend that draws on whole pixels, after layout; a box's sides are snapped each on its
// own, so boxes that touch go on touching. The scene's size is rounded up.
//
// A path of an edge, a node, a group or a text item with straight lines in it has each point
// it passes through snapped, so that its lines meet the boxes and markers they lead to, and
// each control point of a curve moved with the point it leans on, so that a rounded corner
// still turns from one line into the next. A path of curves alone, such as a circle, or one
// of a mark, such as an icon, is moved as a whole, by what puts its first point on the grid,
// so that it keeps its shape, as rounding each of its points would not.
func (sc *Scene) Snap(unit float64) {
	if !(unit > 0) {
		return
	}
	at := func(v float64) float64 { return math.Round(v/unit) * unit }
	box := func(r Rect) Rect {
		x0, y0 := at(r.X), at(r.Y)
		return Rect{x0, y0, at(r.X+r.W) - x0, at(r.Y+r.H) - y0}
	}
	point := func(p Point) Point { return Point{at(p.X), at(p.Y)} }
	sc.Width = math.Ceil(sc.Width/unit) * unit
	sc.Height = math.Ceil(sc.Height/unit) * unit
	paths := map[*Path]bool{}
	for i := range sc.Items {
		it := &sc.Items[i]
		it.Box = box(it.Box)
		for j := range it.Parts {
			part := &it.Parts[j]
			p := part.Path
			if p == nil {
				part.Box = box(part.Box)
				continue
			}
			if paths[p] {
				continue
			}
			paths[p] = true
			snapPath(p, it.Kind != Mark && slices.Contains(p.Ops, LineTo), at)
		}
		for j := range it.Route {
			it.Route[j] = point(it.Route[j])
		}
		for j := range it.Ends {
			it.Ends[j].At = point(it.Ends[j].At)
		}
		it.At = point(it.At)
		if it.Picture != nil {
			pic := *it.Picture
			pic.Box = box(pic.Box)
			it.Picture = &pic
		}
	}
}

// snapPath puts a path on the grid at gives, as Snap says: through its points, when through
// is set, and otherwise as a whole.
func snapPath(p *Path, through bool, at func(float64) float64) {
	if len(p.Args) < 2 {
		return
	}
	if !through {
		x, y := at(p.Args[0]), at(p.Args[1])
		dx, dy := x-p.Args[0], y-p.Args[1]
		for k := 2; k+1 < len(p.Args); k += 2 {
			p.Args[k] += dx
			p.Args[k+1] += dy
		}
		p.Args[0], p.Args[1] = x, y
		return
	}
	// The point a step starts from, before and after it is snapped.
	var x0, y0, sx0, sy0 float64
	a := p.Args
	k := 0
	for _, op := range p.Ops {
		n := op.Args()
		if n == 0 {
			continue
		}
		x, y := a[k+n-2], a[k+n-1]
		sx, sy := at(x), at(y)
		switch op {
		case QuadTo:
			// The control point moves along each axis with the end nearer to it on that axis,
			// as a corner's control point sits level with one line and in line with the other.
			a[k] += nearer(a[k], x0, sx0-x0, x, sx-x)
			a[k+1] += nearer(a[k+1], y0, sy0-y0, y, sy-y)
		case CubeTo:
			a[k] += sx0 - x0
			a[k+1] += sy0 - y0
			a[k+2] += sx - x
			a[k+3] += sy - y
		}
		a[k+n-2], a[k+n-1] = sx, sy
		x0, y0, sx0, sy0 = x, y, sx, sy
		k += n
	}
}

// nearer is d0, how far the start moved, when v is no farther from v0 than from v1, and
// otherwise d1, how far the end moved.
func nearer(v, v0, d0, v1, d1 float64) float64 {
	if math.Abs(v-v0) <= math.Abs(v-v1) {
		return d0
	}
	return d1
}
