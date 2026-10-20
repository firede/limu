package scene

import (
	"math"
	"testing"
)

func TestFlatten(t *testing.T) {
	lines := new(Path).MoveTo(0, 0).LineTo(10, 0).MoveTo(5, 5).MoveTo(0, 10).LineTo(10, 10).LineTo(10, 20).Close().Flatten(0.1)
	if len(lines) != 2 {
		t.Fatalf("%d polylines, want 2 (a lone point is left out)", len(lines))
	}
	if l := lines[0]; l.Closed || len(l.Points) != 2 || l.Points[1] != (Point{10, 0}) {
		t.Errorf("first %v", l)
	}
	if l := lines[1]; !l.Closed || len(l.Points) != 3 {
		t.Errorf("second %v", l)
	}

	// A circle stays within the tolerance of itself, and closes without repeating its start.
	const r, tol = 50.0, 0.1
	c := new(Path).Ellipse(0, 0, r, r).Flatten(tol)
	if len(c) != 1 || !c[0].Closed {
		t.Fatalf("circle %v", c)
	}
	pts := c[0].Points
	if pts[0] == pts[len(pts)-1] {
		t.Error("the start is repeated")
	}
	for i, p := range pts {
		q := pts[(i+1)%len(pts)]
		mid := Point{(p.X + q.X) / 2, (p.Y + q.Y) / 2}
		if d := r - math.Hypot(mid.X, mid.Y); d > tol*1.01 {
			t.Fatalf("strays %v from the circle", d)
		}
	}
	if len(pts) > 200 {
		t.Errorf("%d points for a circle", len(pts))
	}

	// A step after a close starts where the figure began.
	l := new(Path).MoveTo(1, 1).LineTo(5, 1).LineTo(5, 5).Close().LineTo(0, 0).Flatten(0)
	if len(l) != 2 || l[1].Points[0] != (Point{1, 1}) {
		t.Errorf("after close %v", l)
	}
	// Curves end exactly where they say.
	q := new(Path).MoveTo(0, 0).QuadTo(10, 10, 20, 0.3).Flatten(0.01)
	if end := q[0].Points[len(q[0].Points)-1]; end != (Point{20, 0.3}) {
		t.Errorf("quad ends at %v", end)
	}
	// The points are the same on every platform: where Go fuses the products into the sum,
	// on arm64, this one's x would be 169.94974746830584.
	o := new(Path).Ellipse(165, 83, 7, 7).Flatten(0.25)
	if p := o[0].Points[2]; p != (Point{169.94974746830582, 87.94974746830583}) {
		t.Errorf("circle's third point %v", p)
	}
}

func TestDash(t *testing.T) {
	line := []Polyline{{Points: []Point{{0, 0}, {10, 0}}}}
	xs := func(ls []Polyline) [][2]float64 {
		var out [][2]float64
		for _, l := range ls {
			out = append(out, [2]float64{l.Points[0].X, l.Points[len(l.Points)-1].X})
		}
		return out
	}
	for _, c := range []struct {
		pattern []float64
		offset  float64
		want    [][2]float64
	}{
		{[]float64{3, 2}, 0, [][2]float64{{0, 3}, {5, 8}}},
		{[]float64{3}, 0, [][2]float64{{0, 3}, {6, 9}}}, // 3 3
		{[]float64{3, 2}, 1, [][2]float64{{0, 2}, {4, 7}, {9, 10}}},
		{[]float64{3, 2}, -1, [][2]float64{{1, 4}, {6, 9}}},
		{[]float64{0, 5}, 0, [][2]float64{{0, 0}, {5, 5}, {10, 10}}}, // dots
	} {
		got := xs(Dash(line, c.pattern, c.offset))
		if len(got) != len(c.want) {
			t.Errorf("%v offset %v: %v, want %v", c.pattern, c.offset, got, c.want)
			continue
		}
		for i := range got {
			if !near(got[i][0], c.want[i][0]) || !near(got[i][1], c.want[i][1]) {
				t.Errorf("%v offset %v: %v, want %v", c.pattern, c.offset, got, c.want)
				break
			}
		}
	}

	// A dash goes round a corner, and round the close of a closed line.
	square := []Polyline{{Points: []Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}, Closed: true}}
	d := Dash(square, []float64{12, 3}, 0)
	if first := d[0].Points; len(first) != 3 || first[1] != (Point{10, 0}) || first[2] != (Point{10, 2}) {
		t.Errorf("round the corner %v", first)
	}
	if last := d[len(d)-1].Points; last[len(last)-1] != (Point{0, 0}) {
		t.Errorf("the last dash ends at %v", last)
	}

	// Patterns SVG draws solid.
	for _, p := range [][]float64{nil, {0, 0}, {3, -1}} {
		if got := Dash(line, p, 0); len(got) != 1 || len(got[0].Points) != 2 {
			t.Errorf("%v: %v", p, got)
		}
	}
}
