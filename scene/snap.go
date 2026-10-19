package scene

import "math"

// Snap moves everything in the scene onto a grid of unit: the corners of boxes, the points
// of paths and routes, the tips of markers and the points labels are drawn about. It is for
// a backend that draws on whole pixels, after layout; a box's sides are snapped each on its
// own, so boxes that touch go on touching. The scene's size is rounded up.
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
			part.Box = box(part.Box)
			if p := part.Path; p != nil && !paths[p] {
				paths[p] = true
				for k := range p.Args {
					p.Args[k] = at(p.Args[k])
				}
			}
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
