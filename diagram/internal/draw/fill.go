package draw

import (
	"math"

	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/scene"
)

// What lies on a fill a style sets, as Pele's svg/edges.ts finds it: whether an edge runs
// inside one fill, or crosses the border of one. A diagram looks for it among the groups whose
// author gives them a fill, and records it in the scene; the style then picks the color the
// edge is drawn in on that fill, or lays a casing under it.

// A Fill is a box a style fills, Pele's Fill: its left, top, right and bottom.
type Fill struct {
	Box [4]float64
}

// written is a number as Pele's num writes it into a path and reads it back: rounded to two
// places, as Pele's written does.
func written(v float64) float64 {
	n := jsmath.Round(float64(v * 100))
	if n == 0 {
		return 0
	}
	return n / 100
}

// samplePath is the points a path passes through, as Pele's sample takes them from the
// numbers of its path: each point a line runs to, and eight points along each curve, with
// NaN between the figures. The path's numbers are taken as Pele writes them.
func samplePath(p *scene.Path) []float64 {
	var out []float64
	var x, y float64
	p.Each(func(op scene.Op, args []float64) {
		a := make([]float64, len(args))
		for i, v := range args {
			a[i] = written(v)
		}
		switch op {
		case scene.MoveTo, scene.LineTo:
			if op == scene.MoveTo && len(out) > 0 {
				out = append(out, math.NaN(), math.NaN())
			}
			x, y = a[0], a[1]
			out = append(out, x, y)
		case scene.QuadTo, scene.CubeTo:
			cubic := op == scene.CubeTo
			x1, y1, x2, y2 := a[0], a[1], a[2], a[3]
			x3, y3 := x2, y2
			if cubic {
				x3, y3 = a[4], a[5]
			}
			for t := 0.125; t <= 1; t += 0.125 {
				// De Casteljau's steps, in the order Pele takes them, which its rounding
				// follows.
				ax := x + float64((x1-x)*t)
				ay := y + float64((y1-y)*t)
				bx := x1 + float64((x2-x1)*t)
				by := y1 + float64((y2-y1)*t)
				if cubic {
					cx := x2 + float64((x3-x2)*t)
					cy := y2 + float64((y3-y2)*t)
					ax += float64((bx - ax) * t)
					ay += float64((by - ay) * t)
					bx += float64((cx - bx) * t)
					by += float64((cy - by) * t)
				}
				out = append(out, ax+float64((bx-ax)*t), ay+float64((by-ay)*t))
			}
			x, y = x3, y3
		}
		// A close adds no point, as Pele's Z, which has no numbers, adds none.
	})
	return out
}

// clips reports whether the segment from (x, y) along (dx, dy) passes into the box, by
// Liang and Barsky's clipping, as Pele's clips does. NaN takes the comparisons as JavaScript's
// Math.min and Math.max take it.
func clips(x, y, dx, dy, l, t, r, b float64) bool {
	lo, hi := 0.0, 1.0
	for side := range 4 {
		var p, q float64
		switch side {
		case 0:
			p, q = -dx, x-l
		case 1:
			p, q = dx, r-x
		case 2:
			p, q = -dy, y-t
		default:
			p, q = dy, b-y
		}
		switch {
		case p == 0:
			if q < 0 {
				hi = -1
			}
		case p < 0:
			lo = math.Max(lo, q/p)
		default:
			hi = math.Min(hi, q/p)
		}
	}
	return lo < hi
}

// enters reports whether a line through points passes into the box, as Pele's enters does.
func enters(points []float64, l, t, r, b float64) bool {
	for k := 0; k+2 < len(points); k += 2 {
		x, y, x2, y2 := points[k], points[k+1], points[k+2], points[k+3]
		if x < l && x2 < l || x > r && x2 > r || y < t && y2 < t || y > b && y2 > b {
			continue
		}
		if clips(x, y, x2-x, y2-y, l, t, r, b) {
			return true
		}
	}
	return false
}

// FillUnder is Pele's fillUnder: of fills, the innermost that holds the whole of an edge's
// path and the tips of its markers, given as x and y in turn. It gives the index of that fill,
// or -1 where the edge enters none, and reports crossing where the edge enters a fill whose
// border it crosses, as Pele's gives ''. A fill the edge only touches, within half a unit of
// its border, does not count.
func FillUnder(p *scene.Path, tips []float64, fills []Fill) (holder int, crossing bool) {
	if len(fills) == 0 {
		return -1, false
	}
	points := samplePath(p)
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for k := 0; k < len(points)+len(tips); k += 2 {
		var x, y float64
		if k < len(points) {
			x, y = points[k], points[k+1]
		} else {
			x, y = tips[k-len(points)], tips[k+1-len(points)]
		}
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	holder = -1
	least := math.Inf(1)
	for i, fill := range fills {
		l, t, r, b := fill.Box[0], fill.Box[1], fill.Box[2], fill.Box[3]
		if maxX < l+0.5 || minX > r-0.5 || maxY < t+0.5 || minY > b-0.5 {
			continue
		}
		touched := enters(points, l+0.5, t+0.5, r-0.5, b-0.5)
		for k := 0; k < len(tips) && !touched; k += 2 {
			touched = tips[k] > l+0.5 && tips[k] < r-0.5 && tips[k+1] > t+0.5 && tips[k+1] < b-0.5
		}
		if !touched {
			continue
		}
		if minX < l || maxX > r || minY < t || maxY > b {
			return -1, true
		}
		if area := (r - l) * (b - t); area < least {
			least = area
			holder = i
		}
	}
	return holder, false
}
