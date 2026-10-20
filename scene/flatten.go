package scene

import "math"

// A Polyline is points joined by straight lines; a closed one is joined from its last point
// back to its first as well.
type Polyline struct {
	Points []Point
	Closed bool
}

// Flatten is the path as polylines, one for each of its figures, with its curves cut into
// straight lines that stray from them by no more than tol (0.1 when it is not above zero).
// It is for backends that draw no curves, or no dashes, and for finding what is near a
// point. A figure of a single point is left out.
//
// A product added to a sum is written float64(x*y), which Go rounds on its own, so that Go
// does not fuse the two where it would, on arm64, and the points are the same on every
// platform. A backend that puts them on pixels draws the same pixels everywhere.
func (p *Path) Flatten(tol float64) []Polyline {
	if tol <= 0 {
		tol = 0.1
	}
	var out []Polyline
	var cur []Point
	var at Point // where the path is
	flush := func(closed bool) {
		if closed && len(cur) > 2 && cur[len(cur)-1] == cur[0] {
			cur = cur[:len(cur)-1]
		}
		if len(cur) > 1 {
			out = append(out, Polyline{Points: cur, Closed: closed})
		}
		cur = nil
	}
	// begin starts a figure where the path is, for a step that draws with none begun, as
	// one after a close does.
	begin := func() {
		if len(cur) == 0 {
			cur = append(cur, at)
		}
	}
	p.Each(func(op Op, a []float64) {
		switch op {
		case MoveTo:
			flush(false)
			at = Point{a[0], a[1]}
			cur = append(cur, at)
		case LineTo:
			begin()
			at = Point{a[0], a[1]}
			cur = append(cur, at)
		case QuadTo:
			begin()
			c, end := Point{a[0], a[1]}, Point{a[2], a[3]}
			// A quadratic curve cut into n even steps strays from them by at most
			// |p0 - 2p1 + p2| / (4 n²).
			n := steps(dist(at.X-2*c.X+end.X, at.Y-2*c.Y+end.Y)/4, tol)
			for i := 1; i <= n; i++ {
				t := float64(i) / float64(n)
				u := 1 - t
				cur = append(cur, Point{
					float64(u*u*at.X) + float64(2*u*t*c.X) + float64(t*t*end.X),
					float64(u*u*at.Y) + float64(2*u*t*c.Y) + float64(t*t*end.Y),
				})
			}
			cur[len(cur)-1] = end
			at = end
		case CubeTo:
			begin()
			c1, c2, end := Point{a[0], a[1]}, Point{a[2], a[3]}, Point{a[4], a[5]}
			// A cubic curve cut into n even steps strays from them by at most 3/4 of the
			// larger of |p0 - 2p1 + p2| and |p1 - 2p2 + p3|, over n².
			dd := max(dist(at.X-2*c1.X+c2.X, at.Y-2*c1.Y+c2.Y), dist(c1.X-2*c2.X+end.X, c1.Y-2*c2.Y+end.Y))
			n := steps(dd*3/4, tol)
			for i := 1; i <= n; i++ {
				t := float64(i) / float64(n)
				u := 1 - t
				k0, k1, k2, k3 := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
				cur = append(cur, Point{
					float64(k0*at.X) + float64(k1*c1.X) + float64(k2*c2.X) + float64(k3*end.X),
					float64(k0*at.Y) + float64(k1*c1.Y) + float64(k2*c2.Y) + float64(k3*end.Y),
				})
			}
			cur[len(cur)-1] = end
			at = end
		case Close:
			if len(cur) > 0 {
				at = cur[0]
				flush(true)
			}
		}
	})
	flush(false)
	return out
}

// steps is how many even steps keep a curve within tol of them, given its error over one
// step, err: err / n² ≤ tol.
func steps(err, tol float64) int {
	n := math.Ceil(math.Sqrt(err / tol))
	if !(n >= 1) {
		return 1
	}
	return int(min(n, 256))
}

func dist(dx, dy float64) float64 { return math.Hypot(dx, dy) }

// Dash cuts polylines into dashes, as SVG's stroke-dasharray and stroke-dashoffset do: the
// pattern is the lengths of dashes and gaps in turn, taken twice over when it has an odd
// number of them, and starts anew with each polyline, offset by offset along it. A dash goes
// round the corners it meets; a dash of no length is a polyline of two points that are the
// same, which a stroke with round caps draws as a dot. A pattern with a length below zero,
// or none above it, leaves the polylines as they are, as SVG draws them solid.
func Dash(lines []Polyline, pattern []float64, offset float64) []Polyline {
	if len(pattern)%2 == 1 {
		pattern = append(append([]float64(nil), pattern...), pattern...)
	}
	total := 0.0
	for _, l := range pattern {
		if l < 0 || math.IsNaN(l) || math.IsInf(l, 0) {
			return lines
		}
		total += l
	}
	if !(total > 0) || math.IsNaN(offset) || math.IsInf(offset, 0) {
		return lines
	}
	n := len(pattern)
	var out []Polyline
	for _, line := range lines {
		pts := line.Points
		if len(pts) < 2 {
			continue
		}
		// Where the pattern is at the start of the line: in its entry idx, with remain
		// of it to go.
		o := math.Mod(offset, total)
		if o < 0 {
			o += total
		}
		idx := 0
		for o > 0 && o >= pattern[idx] {
			o -= pattern[idx]
			idx = (idx + 1) % n
		}
		remain := pattern[idx] - o
		on := idx%2 == 0
		var dash []Point
		if on {
			dash = []Point{pts[0]}
		}
		segs := len(pts) - 1
		if line.Closed {
			segs++
		}
		for i := range segs {
			a, b := pts[i], pts[(i+1)%len(pts)]
			length := dist(b.X-a.X, b.Y-a.Y)
			if length == 0 {
				continue
			}
			along := func(pos float64) Point {
				if pos >= length {
					return b
				}
				k := pos / length
				return Point{a.X + float64((b.X-a.X)*k), a.Y + float64((b.Y-a.Y)*k)}
			}
			pos := 0.0
			for {
				if remain <= 0 {
					// The entry is done: a dash ends, or one begins.
					if on {
						if len(dash) == 1 {
							dash = append(dash, dash[0])
						}
						out = append(out, Polyline{Points: dash})
						dash = nil
					}
					idx = (idx + 1) % n
					remain, on = pattern[idx], idx%2 == 0
					if on {
						dash = []Point{along(pos)}
					}
					continue
				}
				if pos >= length {
					break
				}
				step := min(remain, length-pos)
				pos += step
				remain -= step
				if on {
					dash = append(dash, along(pos))
				}
			}
		}
		if on && len(dash) > 1 {
			out = append(out, Polyline{Points: dash})
		}
	}
	return out
}
