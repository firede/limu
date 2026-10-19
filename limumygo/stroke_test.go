package limumygo

import (
	"testing"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// winding is how many times polygons wind about a point.
func winding(polys []scene.Polyline, pt scene.Point) int {
	n := 0
	for _, l := range polys {
		ps := l.Points
		for i := range ps {
			a, b := ps[i], ps[(i+1)%len(ps)]
			side := (b.X-a.X)*(pt.Y-a.Y) - (pt.X-a.X)*(b.Y-a.Y)
			switch {
			case a.Y <= pt.Y && b.Y > pt.Y && side > 0:
				n++
			case a.Y > pt.Y && b.Y <= pt.Y && side < 0:
				n--
			}
		}
	}
	return n
}

// The outline of a stroke covers what SVG's stroke covers with each cap and join: a line
// from 0,0 to 10,0 and down to 10,10, two units wide, and a closed square.
func TestOutline(t *testing.T) {
	corner := []scene.Polyline{{Points: []scene.Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}}}}
	square := []scene.Polyline{{Points: []scene.Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}, Closed: true}}
	dot := []scene.Polyline{{Points: []scene.Point{{X: 5, Y: 5}, {X: 5, Y: 5}}}}
	for _, c := range []struct {
		name  string
		lines []scene.Polyline
		cap   style.Cap
		join  style.Join
		in    []scene.Point
		out   []scene.Point
	}{
		{"butt", corner, style.CapButt, style.JoinMiter,
			[]scene.Point{{X: 0.1, Y: 0.9}, {X: 5, Y: -0.9}, {X: 10.9, Y: -0.9}, {X: 9.1, Y: 9.9}},
			[]scene.Point{{X: -0.1, Y: 0}, {X: 10, Y: 10.1}, {X: 5, Y: 1.1}, {X: 8.9, Y: 5}}},
		{"square", corner, style.CapSquare, style.JoinBevel,
			[]scene.Point{{X: -0.9, Y: -0.9}, {X: 10.9, Y: 10.9}, {X: 10.4, Y: -0.4}},
			[]scene.Point{{X: -1.1, Y: 0}, {X: 10.9, Y: -0.9}}},
		{"round", corner, style.CapRound, style.JoinRound,
			[]scene.Point{{X: -0.6, Y: -0.6}, {X: 10.6, Y: -0.6}},
			[]scene.Point{{X: -0.9, Y: -0.9}, {X: 10.9, Y: -0.9}}},
		{"ring", square, style.CapButt, style.JoinMiter,
			[]scene.Point{{X: -0.9, Y: -0.9}, {X: 5, Y: 0.9}, {X: 10.9, Y: 10.9}, {X: 0.9, Y: 0.9}},
			[]scene.Point{{X: 5, Y: 5}, {X: 1.1, Y: 1.1}, {X: -1.1, Y: 5}}},
		{"dot butt", dot, style.CapButt, style.JoinMiter, nil, []scene.Point{{X: 5, Y: 5}}},
		{"dot square", dot, style.CapSquare, style.JoinMiter, []scene.Point{{X: 5.9, Y: 4.1}}, []scene.Point{{X: 6.1, Y: 5}}},
		{"dot round", dot, style.CapRound, style.JoinMiter, []scene.Point{{X: 5.6, Y: 4.4}}, []scene.Point{{X: 5.9, Y: 4.1}}},
	} {
		polys := outline(c.lines, 1, c.cap, c.join, 0.01)
		for _, pt := range c.in {
			if winding(polys, pt) == 0 {
				t.Errorf("%s: %v not covered", c.name, pt)
			}
		}
		for _, pt := range c.out {
			if winding(polys, pt) != 0 {
				t.Errorf("%s: %v covered", c.name, pt)
			}
		}
	}
}

// A corner sharper than SVG's miter limit allows is bevelled.
func TestOutlineMiterLimit(t *testing.T) {
	// A turn back to 1 in 10 from where the line came: the point would be 20 half widths out.
	sharp := []scene.Polyline{{Points: []scene.Point{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 0, Y: 1}}}}
	polys := outline(sharp, 1, style.CapButt, style.JoinMiter, 0.01)
	if winding(polys, scene.Point{X: 11.5, Y: 0}) != 0 {
		t.Error("a corner past the miter limit was mitred")
	}
	if winding(polys, scene.Point{X: 10.03, Y: 0}) == 0 {
		t.Error("the corner's bevel is missing")
	}
}
