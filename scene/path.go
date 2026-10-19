package scene

import "math"

// A Point is a place in a diagram, in its units, y down.
type Point struct{ X, Y float64 }

// A Rect is a box: its top left corner, its width and height.
type Rect struct{ X, Y, W, H float64 }

// Center is the middle of the box.
func (r Rect) Center() Point { return Point{r.X + r.W/2, r.Y + r.H/2} }

// Contains reports whether p is in the box.
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X <= r.X+r.W && p.Y >= r.Y && p.Y <= r.Y+r.H
}

// An Op is a step of a path.
type Op uint8

const (
	MoveTo Op = iota // to a point
	LineTo           // to a point
	QuadTo           // through a control point to a point
	CubeTo           // through two control points to a point
	Close            // back to where the figure began
)

// Args is how many numbers the step takes.
func (op Op) Args() int { return [...]int{2, 2, 4, 6, 0}[op] }

// A Path is lines and curves: the steps, and the numbers they take, in turn. Arcs are made
// of cubic curves when they are added, so a backend draws only the four kinds of step, as
// MyGo's paths do.
type Path struct {
	Ops  []Op
	Args []float64
}

func (p *Path) MoveTo(x, y float64) *Path { return p.add(MoveTo, x, y) }
func (p *Path) LineTo(x, y float64) *Path { return p.add(LineTo, x, y) }
func (p *Path) QuadTo(cx, cy, x, y float64) *Path {
	return p.add(QuadTo, cx, cy, x, y)
}
func (p *Path) CubeTo(c1x, c1y, c2x, c2y, x, y float64) *Path {
	return p.add(CubeTo, c1x, c1y, c2x, c2y, x, y)
}
func (p *Path) Close() *Path { return p.add(Close) }

func (p *Path) add(op Op, args ...float64) *Path {
	p.Ops = append(p.Ops, op)
	p.Args = append(p.Args, args...)
	return p
}

// Each calls fn with each step and its numbers.
func (p *Path) Each(fn func(op Op, args []float64)) {
	at := 0
	for _, op := range p.Ops {
		n := op.Args()
		fn(op, p.Args[at:at+n])
		at += n
	}
}

// Last is where the path is: the end of its last step.
func (p *Path) Last() Point {
	start, at := Point{}, Point{}
	p.Each(func(op Op, a []float64) {
		switch op {
		case Close:
			at = start
		default:
			at = Point{a[len(a)-2], a[len(a)-1]}
			if op == MoveTo {
				start = at
			}
		}
	})
	return at
}

// Polygon adds a closed figure through the points, given as x, y in turn.
func (p *Path) Polygon(xy ...float64) *Path {
	for i := 0; i+1 < len(xy); i += 2 {
		if i == 0 {
			p.MoveTo(xy[0], xy[1])
		} else {
			p.LineTo(xy[i], xy[i+1])
		}
	}
	return p.Close()
}

// Ellipse adds a closed ellipse with its center at cx, cy, in four curves.
func (p *Path) Ellipse(cx, cy, rx, ry float64) *Path {
	const k = 0.5522847498307936 // 4/3 (√2 − 1)
	kx, ky := rx*k, ry*k
	p.MoveTo(cx+rx, cy)
	p.CubeTo(cx+rx, cy+ky, cx+kx, cy+ry, cx, cy+ry)
	p.CubeTo(cx-kx, cy+ry, cx-rx, cy+ky, cx-rx, cy)
	p.CubeTo(cx-rx, cy-ky, cx-kx, cy-ry, cx, cy-ry)
	p.CubeTo(cx+kx, cy-ry, cx+rx, cy-ky, cx+rx, cy)
	return p.Close()
}

// RoundRect adds a closed box with corners rounded by r.
func (p *Path) RoundRect(r Rect, radius float64) *Path {
	radius = max(0, min(radius, r.W/2, r.H/2))
	if radius == 0 {
		return p.Polygon(r.X, r.Y, r.X+r.W, r.Y, r.X+r.W, r.Y+r.H, r.X, r.Y+r.H)
	}
	const k = 0.5522847498307936
	c := radius * (1 - k)
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	p.MoveTo(x0+radius, y0)
	p.LineTo(x1-radius, y0)
	p.CubeTo(x1-c, y0, x1, y0+c, x1, y0+radius)
	p.LineTo(x1, y1-radius)
	p.CubeTo(x1, y1-c, x1-c, y1, x1-radius, y1)
	p.LineTo(x0+radius, y1)
	p.CubeTo(x0+c, y1, x0, y1-c, x0, y1-radius)
	p.LineTo(x0, y0+radius)
	p.CubeTo(x0, y0+c, x0+c, y0, x0+radius, y0)
	return p.Close()
}

// ArcTo adds an elliptical arc from where the path is to x, y, as SVG's A command draws
// one: radii rx and ry, the ellipse turned by rot degrees, the large or the small way
// round, and clockwise (on the screen) when sweep is true.
func (p *Path) ArcTo(rx, ry, rot float64, large, sweep bool, x, y float64) *Path {
	from := p.Last()
	x1, y1 := from.X, from.Y
	if x1 == x && y1 == y {
		return p
	}
	rx, ry = math.Abs(rx), math.Abs(ry)
	if rx == 0 || ry == 0 {
		return p.LineTo(x, y)
	}
	// The endpoint to center conversion of SVG's implementation notes, F.6.5.
	phi := rot * math.Pi / 180
	sin, cos := math.Sincos(phi)
	dx, dy := (x1-x)/2, (y1-y)/2
	x1p := cos*dx + sin*dy
	y1p := -sin*dx + cos*dy
	if l := x1p*x1p/(rx*rx) + y1p*y1p/(ry*ry); l > 1 {
		s := math.Sqrt(l)
		rx, ry = rx*s, ry*s
	}
	num := rx*rx*ry*ry - rx*rx*y1p*y1p - ry*ry*x1p*x1p
	den := rx*rx*y1p*y1p + ry*ry*x1p*x1p
	co := math.Sqrt(max(0, num/den))
	if large == sweep {
		co = -co
	}
	cxp, cyp := co*rx*y1p/ry, -co*ry*x1p/rx
	cx := cos*cxp - sin*cyp + (x1+x)/2
	cy := sin*cxp + cos*cyp + (y1+y)/2
	angle := func(ux, uy, vx, vy float64) float64 { return math.Atan2(ux*vy-uy*vx, ux*vx+uy*vy) }
	t1 := angle(1, 0, (x1p-cxp)/rx, (y1p-cyp)/ry)
	dt := angle((x1p-cxp)/rx, (y1p-cyp)/ry, (-x1p-cxp)/rx, (-y1p-cyp)/ry)
	if !sweep && dt > 0 {
		dt -= 2 * math.Pi
	} else if sweep && dt < 0 {
		dt += 2 * math.Pi
	}
	if math.IsNaN(cx) || math.IsInf(cx, 0) || math.IsNaN(cy) || math.IsInf(cy, 0) ||
		math.IsNaN(t1) || !(math.Abs(dt) <= 2*math.Pi) || dt == 0 {
		// Radii or ends that are not finite numbers place no ellipse, and an arc of no turn
		// is a line; the path still reaches its end.
		return p.LineTo(x, y)
	}
	// At most a quarter turn a curve. A turn of a quarter, or of a half, comes out of the
	// conversion a little over it as often as not: a quarter, as of a rounded corner, to a part
	// in 10^12 or less, and a half, whose center the conversion finds from the square root of a
	// difference near 0, to a part in 10^6 where its radius is small beside its coordinates. A
	// turn within a part in 10^4 of a whole number of quarters is that many curves, each a
	// little over a quarter at most, which strays from the ellipse no more than a quarter does.
	n := max(1, int(math.Ceil(math.Abs(dt)/(math.Pi/2)-1e-4)))
	step := dt / float64(n)
	k := 4.0 / 3 * math.Tan(step/4)
	point := func(t float64) (float64, float64, float64, float64) {
		st, ct := math.Sincos(t)
		ex, ey := rx*ct, ry*st  // on the ellipse
		tx, ty := -rx*st, ry*ct // along it
		return cos*ex - sin*ey + cx, sin*ex + cos*ey + cy, cos*tx - sin*ty, sin*tx + cos*ty
	}
	t := t1
	ax, ay, adx, ady := point(t)
	for i := range n {
		t += step
		bx, by, bdx, bdy := point(t)
		if i == n-1 {
			bx, by = x, y
		}
		p.CubeTo(ax+k*adx, ay+k*ady, bx-k*bdx, by-k*bdy, bx, by)
		ax, ay, adx, ady = bx, by, bdx, bdy
	}
	return p
}

// Bounds is the box the path's points and control points lie in.
func (p *Path) Bounds() Rect {
	if len(p.Args) < 2 {
		return Rect{}
	}
	x0, y0, x1, y1 := p.Args[0], p.Args[1], p.Args[0], p.Args[1]
	for i := 0; i+1 < len(p.Args); i += 2 {
		x, y := p.Args[i], p.Args[i+1]
		x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
	}
	return Rect{x0, y0, x1 - x0, y1 - y0}
}
