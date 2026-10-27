package limumygo

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

// miterLimit is SVG's initial stroke-miterlimit, which the svg package leaves as it is: a
// mitred corner whose point would be more than this many times half the line's width from
// the corner is bevelled.
const miterLimit = 4

// dashes is the pattern of a pen's dashes in DIPs, taken twice over when it has an odd
// number of lengths, or nil for a line that SVG draws solid: one with no dashes, with a
// length below zero or not finite, or with none above zero.
func (p *Painter) dashes(pn pen) []float64 {
	total := 0.0
	for _, l := range pn.dash {
		if l *= p.k; !(l >= 0) || l > math.MaxFloat32 {
			return nil
		}
		total += l
	}
	if !(total > 0) || total > math.MaxFloat32 {
		return nil
	}
	out := make([]float64, 0, 2*len(pn.dash))
	for _, l := range pn.dash {
		out = append(out, l*p.k)
	}
	if len(out)%2 == 1 {
		out = append(out, out...)
	}
	return out
}

// with sets how MyGo strokes a path with a pen, as SVG strokes it: its ends and corners, the
// miter limit, and the dashes given, in DIPs, offset DIPs into their pattern.
func (p *Painter) with(path *ui.Path, pn pen, dash []float64, offset float64) *ui.Path {
	cap, join := ui.ButtCap, ui.MiterJoin
	switch pn.cap {
	case style.CapRound:
		cap = ui.RoundCap
	case style.CapSquare:
		cap = ui.SquareCap
	}
	switch pn.join {
	case style.JoinRound:
		join = ui.RoundJoin
	case style.JoinBevel:
		join = ui.BevelJoin
	}
	path.Cap(cap).Join(join).MiterLimit(miterLimit)
	if len(dash) > 0 {
		lengths := make([]float32, len(dash))
		for i, l := range dash {
			lengths[i] = float32(l)
		}
		path.Dash(float32(offset), lengths...)
	}
	return path
}

// A run is a piece of a line that strokeLines strokes on its own: its points in DIPs, and
// how far into the dashes it starts, or that it is stroked solid.
type run struct {
	pts    []scene.Point
	closed bool
	offset float64
	solid  bool
}

// strokeLines strokes opaque polylines of the scene with a pen whose dashes, in DIPs, are
// dash, in pieces no more than about piece device pixels across, and only the pieces that
// show. The pieces draw what the line whole would. A line is cut only inside a side long
// enough that the ends of the pieces on either side of the cut, whatever their caps, lie on
// the side's straight stroke, and the piece after a cut starts a device pixel before it, so
// that no seam shows where they meet. Each piece starts as far into the dashes as it is
// along its line. A closed line is cut open in the middle of its longest side, or, dashed,
// where its dashes end and start, with the dash that goes through its start, which MyGo
// turns there as at a corner, stroked as a line of its own. A line that has no side long
// enough to cut is stroked whole.
func (p *Painter) strokeLines(lines []scene.Polyline, c ui.Color, pn pen, dash []float64) {
	w := float64(p.Len(pn.width))
	limit := max(piece/float64(p.Scale())-w-2, 8)
	over := 1 / float64(p.Scale())
	margin := w/2 + over
	var runs []run
	cut := func(pts []scene.Point, r run) {
		runs = cutRun(runs, pts, r, limit, margin, over)
	}
	for _, l := range lines {
		// The polyline in DIPs, without repeated points.
		var pts []scene.Point
		for _, pt := range l.Points {
			q := scene.Point{X: p.x + pt.X*p.k, Y: p.y + pt.Y*p.k}
			if n := len(pts); n == 0 || pts[n-1] != q {
				pts = append(pts, q)
			}
		}
		closed := l.Closed
		if closed && len(pts) > 2 && pts[0] == pts[len(pts)-1] {
			pts = pts[:len(pts)-1]
		}
		if len(pts) < 2 || extent(pts) <= limit {
			runs = append(runs, run{pts: pts, closed: closed})
			continue
		}
		if !closed {
			cut(pts, run{})
			continue
		}
		t := newTrack(pts)
		total := t.length()
		if len(dash) == 0 || total <= dash[0] {
			// Solid all round: from the middle of its longest side round to it, and a
			// device pixel past.
			i := 0
			for k := range len(pts) {
				if t.at[k+1]-t.at[k] > t.at[i+1]-t.at[i] {
					i = k
				}
			}
			if t.at[i+1]-t.at[i] < 2*margin {
				runs = append(runs, run{pts: pts, closed: true})
				continue
			}
			from := (t.at[i] + t.at[i+1]) / 2
			cut(t.part(from, from+total+over), run{solid: true})
			continue
		}
		into, on := dashAt(dash, total)
		if !on {
			cut(t.part(0, total), run{})
			continue
		}
		// The dash through the start, from where the last starts to where the first ends,
		// solid, and the rest from the first gap.
		cut(t.part(total-into, total+dash[0]), run{solid: true})
		if total-into > dash[0] {
			cut(t.part(dash[0], total-into), run{offset: dash[0]})
		}
	}

	view := p.Visible()
	reach := float32(w*miterLimit/2 + 1) // to the points of miters, and past square caps
	var path ui.Path
	var b [4]float64 // the bounds of what is in path
	var group run    // what the runs in path have in common
	empty := true
	flush := func() {
		r := ui.Rect{X: float32(b[0]) - reach, Y: float32(b[1]) - reach, W: float32(b[2]-b[0]) + 2*reach, H: float32(b[3]-b[1]) + 2*reach}
		if !empty && overlaps(r, view) {
			d := dash
			if group.solid {
				d = nil
			}
			p.StrokePath(p.with(&path, pn, d, group.offset), float32(w), c)
		}
		path.Reset()
		empty = true
	}
	for _, r := range runs {
		rb := bounds(r.pts)
		u := [4]float64{min(b[0], rb[0]), min(b[1], rb[1]), max(b[2], rb[2]), max(b[3], rb[3])}
		if !empty && (max(u[2]-u[0], u[3]-u[1]) > limit || len(dash) > 0 && (r.offset != group.offset || r.solid != group.solid)) {
			flush()
		}
		if empty {
			b, group = rb, r
		} else {
			b = u
		}
		for i, q := range r.pts {
			if i == 0 {
				path.MoveTo(float32(q.X), float32(q.Y))
			} else {
				path.LineTo(float32(q.X), float32(q.Y))
			}
		}
		if len(r.pts) == 1 {
			// A line of no length, which shows its caps.
			path.LineTo(float32(r.pts[0].X), float32(r.pts[0].Y))
		}
		if r.closed {
			path.Close()
		}
		empty = false
	}
	flush()
}

// cutRun adds to runs an open polyline in DIPs, stroked solid or starting as far into its
// dashes as the run r says, cut into runs no more than about limit DIPs across, for
// strokeLines: each cut inside a side, at least margin from its ends, with the run after it
// starting over DIPs before it.
func cutRun(runs []run, pts []scene.Point, r run, limit, margin, over float64) []run {
	cur := run{pts: pts[:1:1], offset: r.offset, solid: r.solid}
	b := bounds(cur.pts)
	fits := func(q scene.Point) bool {
		return max(b[2], q.X)-min(b[0], q.X) <= limit && max(b[3], q.Y)-min(b[1], q.Y) <= limit
	}
	grow := func(q scene.Point) {
		b = [4]float64{min(b[0], q.X), min(b[1], q.Y), max(b[2], q.X), max(b[3], q.Y)}
	}
	along := 0.0 // how far along the line the side starts
	for i := 1; i < len(pts); i++ {
		a, z := pts[i-1], pts[i]
		l := math.Hypot(z.X-a.X, z.Y-a.Y)
		// The side is looked at in steps of no more than half the limit, and cut where a
		// step would take the run past it.
		n := max(int(math.Ceil(l/(limit/2))), 1)
		least := margin // the least a cut may be along the side
		for j := 1; j <= n; j++ {
			s := lerp(a, z, float64(j)/float64(n))
			if fits(s) {
				grow(s)
				continue
			}
			at := max(l*float64(j-1)/float64(n), least)
			if at > l-margin {
				grow(s)
				continue
			}
			cur.pts = append(cur.pts, lerp(a, z, at/l))
			runs = append(runs, cur)
			start := lerp(a, z, (at-over)/l)
			cur = run{pts: []scene.Point{start}, offset: r.offset + along + at - over, solid: r.solid}
			b = bounds(cur.pts)
			grow(s)
			least = at + margin
		}
		cur.pts = append(cur.pts, z)
		along += l
	}
	return append(runs, cur)
}

// dashAt is how far into an entry of a pattern of dashes and gaps a line is at d along it, d
// above zero, and whether the entry is a dash: of the entry that ends at d, rather than the
// one that starts there.
func dashAt(pattern []float64, d float64) (into float64, on bool) {
	period := 0.0
	for _, l := range pattern {
		period += l
	}
	o := math.Mod(d, period)
	if o == 0 {
		o = period
	}
	k := 0
	for k < len(pattern)-1 && o > pattern[k] {
		o -= pattern[k]
		k++
	}
	return o, k%2 == 0
}

// A track is a closed polyline in DIPs, with how far along it each point is, back to its
// start; it goes round again past its end.
type track struct {
	pts []scene.Point // ending with the first
	at  []float64
}

func newTrack(pts []scene.Point) track {
	pts = append(pts[:len(pts):len(pts)], pts[0])
	at := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		at[i] = at[i-1] + math.Hypot(pts[i].X-pts[i-1].X, pts[i].Y-pts[i-1].Y)
	}
	return track{pts, at}
}

func (t track) length() float64 { return t.at[len(t.at)-1] }

// point is the track's ith point, counting on round it, and how far along it is.
func (t track) point(i int) (scene.Point, float64) {
	n := len(t.pts) - 1
	return t.pts[i%n], t.at[i%n] + float64(i/n)*t.length()
}

// part is the polyline of the track from a along it to b, a below b.
func (t track) part(a, b float64) []scene.Point {
	var out []scene.Point
	add := func(q scene.Point) {
		if n := len(out); n == 0 || out[n-1] != q {
			out = append(out, q)
		}
	}
	for i := 1; ; i++ {
		p0, d0 := t.point(i - 1)
		p1, d1 := t.point(i)
		if d1 < a {
			continue
		}
		if len(out) == 0 {
			add(lerp(p0, p1, (a-d0)/(d1-d0)))
		}
		if b <= d1 {
			add(lerp(p0, p1, (b-d0)/(d1-d0)))
			return out
		}
		add(p1)
	}
}

func lerp(a, b scene.Point, k float64) scene.Point {
	return scene.Point{X: a.X + (b.X-a.X)*k, Y: a.Y + (b.Y-a.Y)*k}
}

// bounds is the box about points, as its least x and y and its greatest.
func bounds(pts []scene.Point) [4]float64 {
	b := [4]float64{math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)}
	for _, q := range pts {
		b = [4]float64{min(b[0], q.X), min(b[1], q.Y), max(b[2], q.X), max(b[3], q.Y)}
	}
	return b
}

// extent is how far points reach across or down, whichever is farther.
func extent(pts []scene.Point) float64 {
	b := bounds(pts)
	return max(b[2]-b[0], b[3]-b[1])
}
