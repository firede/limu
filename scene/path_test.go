package scene

import (
	"math"
	"slices"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestArcTo(t *testing.T) {
	for _, c := range []struct {
		large, sweep bool
		curves       int
		mid          Point
	}{
		// From (10,0) to (-10,0) on a circle of radius 10 about the origin.
		{false, true, 2, Point{0, 10}},
		{false, false, 2, Point{0, -10}},
	} {
		p := new(Path).MoveTo(10, 0).ArcTo(10, 10, 0, c.large, c.sweep, -10, 0)
		if got := len(p.Ops) - 1; got != c.curves {
			t.Fatalf("%d curves", got)
		}
		if end := p.Last(); !near(end.X, -10) || !near(end.Y, 0) {
			t.Errorf("ends at %v", end)
		}
		if m := (Point{p.Args[6], p.Args[7]}); !near(m.X, c.mid.X) || !near(m.Y, c.mid.Y) {
			t.Errorf("sweep %v passes %v, want %v", c.sweep, m, c.mid)
		}
	}
	// Radii too small to reach are scaled up, as SVG does.
	p := new(Path).MoveTo(0, 0).ArcTo(1, 1, 0, false, true, 10, 0)
	if b := p.Bounds(); b.H < 4.9 {
		t.Errorf("bounds %v", b)
	}
	if p := new(Path).MoveTo(0, 0).ArcTo(0, 5, 0, false, true, 4, 0); p.Ops[1] != LineTo {
		t.Error("a zero radius is a line")
	}
	// A quarter turn is one curve, though the angle the conversion finds is a little over.
	for _, r := range []float64{1, 3, 4.5, 7.25, 10, 1e3} {
		for _, at := range []Point{{0, 0}, {0.1, 0.2}, {123.45, 678.9}, {-3.3, 1e4}} {
			for _, sweep := range []bool{false, true} {
				p := new(Path).MoveTo(at.X+r, at.Y).ArcTo(r, r, 0, false, sweep, at.X, at.Y+r)
				if got := len(p.Ops) - 1; got != 1 {
					t.Errorf("a quarter turn of radius %v from %v in %d curves", r, at, got)
				}
				if p := new(Path).MoveTo(at.X+r, at.Y).ArcTo(r, r, 0, false, sweep, at.X-r, at.Y); len(p.Ops)-1 != 2 {
					t.Errorf("a half turn of radius %v from %v in %d curves", r, at, len(p.Ops)-1)
				}
			}
		}
	}
	// What is not a finite number places no ellipse: the path goes on to its end in a line.
	for _, c := range [][3]float64{{math.NaN(), 5, 4}, {math.Inf(1), 5, 4}, {5, 5, math.NaN()}} {
		p := new(Path).MoveTo(0, 0).ArcTo(c[0], c[1], 0, false, true, c[2], 3)
		if len(p.Ops) != 2 || p.Ops[1] != LineTo || p.Args[3] != 3 {
			t.Errorf("ArcTo(%v) %v %v", c, p.Ops, p.Args)
		}
	}
}

func TestShapes(t *testing.T) {
	if b := new(Path).Ellipse(5, 5, 4, 2).Bounds(); !near(b.X, 1) || !near(b.W, 8) || !near(b.H, 4) {
		t.Errorf("ellipse %v", b)
	}
	r := new(Path).RoundRect(Rect{0, 0, 20, 10}, 100)
	if b := r.Bounds(); b != (Rect{0, 0, 20, 10}) {
		t.Errorf("round rect %v", b)
	}
	if last := r.Last(); last != (Point{5, 0}) {
		t.Errorf("closes at %v", last)
	}
}

func TestMarkers(t *testing.T) {
	parts := Marker{Type: "arrow_point", At: Point{10, 0}, DX: 1}.Outline()
	if len(parts) != 1 || parts[0].Paint != FillOnly || parts[0].Path.Bounds() != (Rect{2, -3.36, 8, 6.72}) {
		t.Errorf("arrow %+v", parts)
	}
	if cross := (Marker{Type: "arrow_cross", DX: 1}).Outline(); len(cross) != 1 || cross[0].Paint != StrokeOnly {
		t.Errorf("cross %+v", cross)
	}
	if (Marker{}).Outline() != nil || (Marker{Type: "nonexistent"}).Outline() != nil {
		t.Error("no type, no marker")
	}
	if MarkerType("arrow_point").Trim() != 7 || MarkerType("").Trim() != 0 {
		t.Error("trim")
	}
}

func TestSnap(t *testing.T) {
	shared := new(Path).MoveTo(0.4, 0.6).LineTo(10.5, 3.2)
	sc := &Scene{Width: 10.2, Height: 7, Items: []Item{
		{Box: Rect{0.4, 0.4, 9.2, 3.3}, Parts: []Part{{Path: shared}, {Path: shared}}, At: Point{4.6, 2.5}},
		{Route: []Point{{1.2, 3.7}}, Ends: [2]Marker{{At: Point{2.49, 0}}}},
	}}
	sc.Snap(1)
	if sc.Width != 11 || sc.Height != 7 {
		t.Errorf("size %v %v", sc.Width, sc.Height)
	}
	if b := sc.Items[0].Box; b != (Rect{0, 0, 10, 4}) {
		t.Errorf("box %v", b)
	}
	if a := shared.Args; a[0] != 0 || a[1] != 1 || a[2] != 11 || a[3] != 3 {
		t.Errorf("a shared path is snapped once: %v", a)
	}
	if r, m := sc.Items[1].Route[0], sc.Items[1].Ends[0].At; r != (Point{1, 4}) || m != (Point{2, 0}) {
		t.Errorf("route %v marker %v", r, m)
	}

	// A path with curves moves as a whole, by what puts its first point on the grid; a path
	// part's box is not snapped, as it is not used.
	circle := new(Path).Ellipse(10.3, 20.6, 3.3, 3.3)
	want := slices.Clone(circle.Args)
	sc = &Scene{Items: []Item{{Parts: []Part{{Path: circle, Box: Rect{0.4, 0.4, 1, 1}}}}}}
	sc.Snap(1)
	if a := circle.Args; a[0] != 14 || a[1] != 21 {
		t.Errorf("a circle's first point is at %v,%v", a[0], a[1])
	}
	for k := 2; k+1 < len(circle.Args); k += 2 {
		dx, dy := circle.Args[k]-want[k], circle.Args[k+1]-want[k+1]
		if math.Abs(dx-0.4) > 1e-9 || math.Abs(dy-0.4) > 1e-9 {
			t.Errorf("point %d moved by %v,%v", k/2, dx, dy)
		}
	}
	if b := sc.Items[0].Parts[0].Box; b != (Rect{0.4, 0.4, 1, 1}) {
		t.Errorf("a path part's box snapped to %v", b)
	}

	// An edge's lines land on the grid, and a rounded corner's control points stay on the
	// lines it joins. A mark's path of the same steps moves as a whole.
	corner := func() *Path {
		return new(Path).MoveTo(0.4, 0.6).LineTo(10.3, 0.6).CubeTo(12.3, 0.6, 13.3, 1.6, 13.3, 3.6).
			LineTo(13.3, 20.2).QuadTo(13.3, 22.6, 15.7, 22.6)
	}
	edge, mark := corner(), corner()
	sc = &Scene{Items: []Item{{Kind: Edge, Parts: []Part{{Path: edge}}}, {Kind: Mark, Parts: []Part{{Path: mark}}}}}
	sc.Snap(1)
	if want := []float64{0, 1, 10, 1, 12, 1, 13, 2, 13, 4, 13, 20, 13, 23, 16, 23}; !slices.Equal(edge.Args, want) {
		t.Errorf("an edge's corner snapped to %v, not %v", edge.Args, want)
	}
	if a := mark.Args; a[0] != 0 || a[1] != 1 || math.Abs(a[10]-12.9) > 1e-9 || math.Abs(a[11]-20.6) > 1e-9 {
		t.Errorf("a mark's path moved to %v", a)
	}
}
