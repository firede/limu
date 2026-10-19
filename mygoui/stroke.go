package mygoui

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// A pen is how a line is stroked: its width and dashes, in the scene's units, and its ends
// and corners.
type pen struct {
	width float64
	dash  []float64
	cap   style.Cap
	join  style.Join
}

// penOf is the pen of a style, with the dashes given.
func penOf(s style.Resolved, dash []float64) pen {
	return pen{width: s.StrokeWidth, dash: dash, cap: s.Cap, join: s.Join}
}

// miterLimit is SVG's initial stroke-miterlimit: a mitred corner whose point would be more
// than this many times half the line's width from the corner is bevelled.
const miterLimit = 4

// outline is the outline of polylines stroked hw to each side, as SVG strokes them with a cap
// and a join, as polygons to fill with the non-zero rule. It is MyGo's strokeInto with the
// ends and corners SVG has: each side of a polyline is offset from it, turning around the
// outside of a corner as the join says and through the corner on the inside, where the
// stroke covers itself; an open polyline is outlined as the loop that goes there and back,
// whose turns at its ends are its caps, and a closed one as a loop on each side, the other
// side going the other way, so that its inside is not filled. Each edge of the stroke is
// drawn once, so that its antialiasing is exact. Round ends and corners are cut into lines
// that stray from their arcs by no more than tol.
// It works around MyGo, as offsetLoop, arc and strokeOutline do, until ui.Painter.StrokePath
// takes caps, joins, a miter limit and dashes, as egoist/mygo#530 asks.
func outline(lines []scene.Polyline, hw float64, cap style.Cap, join style.Join, tol float64) []scene.Polyline {
	var out []scene.Polyline
	for _, l := range lines {
		// The polyline without repeated points.
		var pts []scene.Point
		for _, pt := range l.Points {
			if n := len(pts); n == 0 || pts[n-1] != pt {
				pts = append(pts, pt)
			}
		}
		if l.Closed && len(pts) > 2 && pts[0] == pts[len(pts)-1] {
			pts = pts[:len(pts)-1]
		}
		switch {
		case len(pts) == 0:
		case len(pts) == 1:
			// A line of no length shows only its caps: a dot, a square, or nothing.
			c := pts[0]
			switch cap {
			case style.CapRound:
				out = append(out, scene.Polyline{Points: arc(nil, c, hw, 0, 2*math.Pi, tol), Closed: true})
			case style.CapSquare:
				out = append(out, scene.Polyline{Points: []scene.Point{
					{X: c.X - hw, Y: c.Y - hw}, {X: c.X + hw, Y: c.Y - hw}, {X: c.X + hw, Y: c.Y + hw}, {X: c.X - hw, Y: c.Y + hw},
				}, Closed: true})
			}
		case l.Closed && len(pts) > 2:
			n := len(pts)
			none := func(int) bool { return false }
			out = append(out, offsetLoop(pts, hw, cap, join, none, tol))
			back := make([]scene.Point, n)
			for k, pt := range pts {
				back[n-1-k] = pt
			}
			out = append(out, offsetLoop(back, hw, cap, join, none, tol))
		default:
			// There and back. A closed line of two points turns at each end as at a corner.
			n := len(pts)
			loop := pts[:n:n]
			for k := n - 2; k > 0; k-- {
				loop = append(loop, pts[k])
			}
			ends := func(k int) bool { return !l.Closed && (k == 0 || k == n-1) }
			out = append(out, offsetLoop(loop, hw, cap, join, ends, tol))
		}
	}
	return out
}

// offsetLoop is the side of a closed polyline hw to its left (a segment's normal), turning
// around each corner as the join says, and around each point that end reports to be an end
// of the line, where the loop turns straight back, as the cap says.
// It goes with outline, after egoist/mygo#530.
func offsetLoop(pts []scene.Point, hw float64, cap style.Cap, join style.Join, end func(k int) bool, tol float64) scene.Polyline {
	n := len(pts)
	normal := func(k int) (dx, dy, nx, ny float64) {
		a, b := pts[k], pts[(k+1)%n]
		dx, dy = b.X-a.X, b.Y-a.Y
		l := math.Hypot(dx, dy)
		dx, dy = dx/l, dy/l
		return dx, dy, -dy * hw, dx * hw
	}
	dx, dy, nx, ny := normal(0)
	out := []scene.Point{{X: pts[0].X + nx, Y: pts[0].Y + ny}}
	for k := range n {
		b := pts[(k+1)%n]
		out = append(out, scene.Point{X: b.X + nx, Y: b.Y + ny})
		ex, ey, mx, my := normal((k + 1) % n)
		after := scene.Point{X: b.X + mx, Y: b.Y + my}
		// Rounded apart, as fused multiply-adds would leave a polyline turning straight
		// back a hair off it.
		cross, dot := float64(dx*ey)-float64(dy*ex), float64(dx*ex)+float64(dy*ey)
		from := math.Atan2(ny, nx)
		switch {
		case end((k + 1) % n):
			switch cap {
			case style.CapRound:
				out = arc(out, b, hw, from, -math.Pi, tol)
			case style.CapSquare:
				ox, oy := dx*hw, dy*hw
				out = append(out, scene.Point{X: b.X + nx + ox, Y: b.Y + ny + oy}, scene.Point{X: b.X + mx + ox, Y: b.Y + my + oy})
			}
		case cross > 0:
			// The inside of the corner: through it.
			out = append(out, b)
		case cross < 0 || dot < 0:
			// The outside, or the line turning straight back.
			switch {
			case join == style.JoinRound:
				turn := math.Atan2(cross, dot)
				if cross == 0 {
					turn = -math.Pi
				}
				out = arc(out, b, hw, from, turn, tol)
			case join == style.JoinBevel, 1+dot < 2.0/(miterLimit*miterLimit):
			default:
				// The point where the sides meet, (n+m)/(1+n·m) from the corner, for
				// normals n and m hw long.
				k := 1 / (1 + dot)
				out = append(out, scene.Point{X: b.X + (nx+mx)*k, Y: b.Y + (ny+my)*k})
			}
		}
		out = append(out, after)
		dx, dy, nx, ny = ex, ey, mx, my
	}
	return scene.Polyline{Points: out, Closed: true}
}

// arc adds to pts the points of an arc of radius r about c, from the angle from turning by
// turn, past its start, cut into lines that stray from it by no more than tol.
// It goes with outline, after egoist/mygo#530.
func arc(pts []scene.Point, c scene.Point, r, from, turn, tol float64) []scene.Point {
	step := 2 * math.Acos(max(1-tol/r, -1))
	steps := max(int(math.Ceil(math.Abs(turn)/step)), 1)
	for s := 1; s <= steps; s++ {
		sin, cos := math.Sincos(from + turn*float64(s)/float64(steps))
		pts = append(pts, scene.Point{X: c.X + r*cos, Y: c.Y + r*sin})
	}
	return pts
}

// strokeOutline strokes polylines of the scene by filling their outline, for the ends and
// corners MyGo does not draw: it strokes only with round ones.
// It goes with outline, after egoist/mygo#530.
func (p *Painter) strokeOutline(lines []scene.Polyline, c ui.Color, pn pen) {
	polys := outline(lines, pn.width/2, pn.cap, pn.join, p.tolerance()/4)
	if len(polys) == 0 {
		return
	}
	b := bounds(polys)
	size := p.across(b, 0)
	if size < maxMask && (size <= piece || c.A < 255) {
		var path ui.Path
		for _, l := range polys {
			for i, pt := range l.Points {
				x, y := p.Point(pt)
				if i == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			path.Close()
		}
		p.FillPath(&path, c)
		return
	}
	// An opaque outline is filled in tiles no larger than a piece, as the round strokes
	// are cut into pieces, and any too large for a mask in tiles as large as one.
	tile := maxMask
	if c.A == 255 {
		tile = piece
	}
	p.fillTiles(polys, c, tile)
}

// bounds is the box about polylines.
func bounds(lines []scene.Polyline) scene.Rect {
	x0, y0 := math.Inf(1), math.Inf(1)
	x1, y1 := math.Inf(-1), math.Inf(-1)
	for _, l := range lines {
		for _, pt := range l.Points {
			x0, y0, x1, y1 = min(x0, pt.X), min(y0, pt.Y), max(x1, pt.X), max(y1, pt.Y)
		}
	}
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}
