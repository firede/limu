package draw

import (
	"math"

	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// What lies on a fill a style sets, as Pele's svg/edges.ts finds it: whether an edge runs
// inside one fill, or crosses the border of one. A diagram looks for it among the groups whose
// author gives them a fill, and records it in the scene; the style then picks the color the
// edge is drawn in on that fill, or lays a casing under it.

// A Fill is a box a style fills, Pele's Fill: its left, top, right and bottom, and the
// declarations its author gives it, of which css.Paints is true. A diagram gathers its fills
// of the groups whose author paints them.
type Fill struct {
	Box   [4]float64
	Decls []string
}

// OnFills is what Pele's overFills makes of an edge on fills: the ground to give the edge's
// style where it lies inside one fill that ink picks a color on (css.Inks), unless its
// author's declarations, line, set it a stroke (edgeInk); and whether it needs a casing,
// where it crosses the border of a fill or lies inside one otherwise.
func OnFills(p *scene.Path, tips []float64, fills []Fill, line []string) (ground style.Paint, casing bool) {
	holder, crossing := FillUnder(p, tips, fills)
	switch {
	case crossing:
		return style.Paint{}, true
	case holder < 0:
		return style.Paint{}, false
	case css.Sets(line, "stroke") || !css.Inks(fills[holder].Decls):
		return style.Paint{}, true
	}
	return css.Declarations(fills[holder].Decls).Style.Fill, false
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
// border it crosses, where Pele's gives the empty string. A fill the edge only touches,
// within half a unit of its border, does not count.
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

// caseWidth is how much wider than its edge a casing is, Pele's CASE.
const caseWidth = 4

// Casing is an item of the role given that draws a casing under an edge that crosses the
// border of a fill: its path and the outlines of its markers stroked in the background, wider
// than the edge, with round ends and corners, as Pele's edgeCasing and casingsSvg draw it, so
// that the edge stands out where it runs over the fill. Its width is that of the edge, the
// width author gives it or 2.5 where it is thick, and four more, and it takes the author's
// opacities. It gives nil for an edge whose author strokes it in none, which needs none. A
// diagram puts its casings after its groups and before its edges, as Pele does.
func Casing(role scene.Role, path *scene.Path, ends [2]scene.Marker, author style.Style, thick bool) *scene.Item {
	// Pele draws none for a stroke of none or transparent.
	if s := author.Stroke; s.Set() && (!s.Visible() || s.Token().Name() == "" && s.Own().A == 0) {
		return nil
	}
	width := author.StrokeWidth.Or(1)
	if thick {
		width = 2.5
	}
	it := &scene.Item{
		Kind:  scene.Mark,
		Role:  role,
		Parts: []scene.Part{{Path: path, Paint: scene.StrokeOnly}},
		Style: style.Style{
			Fill: style.Bg, Stroke: style.Bg, StrokeWidth: style.N(width + caseWidth),
			Cap: style.CapRound, Join: style.JoinRound,
			StrokeOpacity: author.StrokeOpacity, Opacity: author.Opacity,
		},
	}
	box := path.Bounds()
	for _, m := range ends {
		for _, part := range m.Outline() {
			if part.Paint != scene.StrokeOnly {
				part.Paint = scene.FillAndStroke
			}
			it.Parts = append(it.Parts, part)
			if part.Path != nil {
				box = union(box, part.Path.Bounds())
			} else {
				box = union(box, part.Box)
			}
		}
	}
	it.Box = box
	return it
}

// union is the box about two boxes.
func union(a, b scene.Rect) scene.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}
