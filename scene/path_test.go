package scene

import (
	"math"
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
}
