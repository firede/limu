package scene

import "math"

// Finite reports whether every number that places or sizes something in the scene is finite:
// its size, and the boxes, paths, routes, markers, labels and pictures of its items. A layout
// given numbers at the edge of what a float64 holds, as a sankey diagram's flow of 5e-324,
// can work out NaN where Pele does too; limu.Layout gives an error for such a scene rather
// than hand it to a backend.
func (sc *Scene) Finite() bool {
	ok := func(vs ...float64) bool {
		for _, v := range vs {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return false
			}
		}
		return true
	}
	box := func(r Rect) bool { return ok(r.X, r.Y, r.W, r.H) }
	if !ok(sc.Width, sc.Height, sc.FontSize) {
		return false
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		if !box(it.Box) || !ok(it.At.X, it.At.Y, it.Rotate) {
			return false
		}
		for _, p := range it.Parts {
			if !box(p.Box) || p.Path != nil && !ok(p.Path.Args...) {
				return false
			}
		}
		for _, p := range it.Route {
			if !ok(p.X, p.Y) {
				return false
			}
		}
		for _, m := range it.Ends {
			if !ok(m.At.X, m.At.Y, m.DX, m.DY) {
				return false
			}
		}
		if l := it.Label; l != nil {
			if !ok(l.Width, l.Height, l.Size, l.LineHeight) || !ok(l.Widths...) {
				return false
			}
			for _, line := range l.Lines {
				for _, s := range line {
					if !ok(s.Width) {
						return false
					}
				}
			}
		}
		if it.Picture != nil && !box(it.Picture.Box) {
			return false
		}
	}
	return true
}
