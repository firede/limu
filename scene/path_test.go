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
	p, filled := Marker{Type: "arrow_point", At: Point{10, 0}, DX: 1}.Outline()
	if !filled || p.Bounds() != (Rect{2, -3.36, 8, 6.72}) {
		t.Errorf("arrow %v %v", p.Bounds(), filled)
	}
	if p, _ := (Marker{}).Outline(); p != nil {
		t.Error("no type, no marker")
	}
	if MarkerType("arrow_point").Trim() != 7 || MarkerType("").Trim() != 0 {
		t.Error("trim")
	}
}
