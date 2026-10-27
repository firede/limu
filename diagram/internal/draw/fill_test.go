package draw

import (
	"testing"

	"github.com/firede/limu/scene"
)

func TestFillUnder(t *testing.T) {
	outer := Fill{Box: [4]float64{0, 0, 200, 200}}
	inner := Fill{Box: [4]float64{50, 50, 150, 150}}
	fills := []Fill{outer, inner}
	line := func(x0, y0, x1, y1 float64) *scene.Path { return new(scene.Path).MoveTo(x0, y0).LineTo(x1, y1) }
	for _, c := range []struct {
		name     string
		path     *scene.Path
		tips     []float64
		holder   int
		crossing bool
	}{
		{"inside both", line(60, 60, 140, 140), []float64{60, 60, 140, 140}, 1, false},
		{"inside the outer", line(10, 10, 40, 40), nil, 0, false},
		{"across the inner's border", line(10, 10, 100, 100), nil, -1, true},
		{"off both", line(300, 300, 400, 300), nil, -1, false},
		// Along the border, within half a unit of it, is not inside.
		{"along the inner's border", line(50.2, 60, 50.2, 140), nil, 0, false},
		{"a curve out and back", new(scene.Path).MoveTo(60, 100).CubeTo(0, 0, 0, 200, 140, 100), nil, -1, true},
	} {
		h, x := FillUnder(c.path, c.tips, fills)
		if h != c.holder || x != c.crossing {
			t.Errorf("%s: %d, %v; want %d, %v", c.name, h, x, c.holder, c.crossing)
		}
	}
}

// Pele leaves out the casing of an edge stroked in none or transparent as written, and only
// then.
func TestCasingOfNone(t *testing.T) {
	path := new(scene.Path).MoveTo(0, 0).LineTo(10, 0)
	for _, c := range []struct {
		line  []string
		cased bool
	}{
		{nil, true},
		{[]string{"stroke:none"}, false},
		{[]string{"stroke: transparent !important"}, false},
		{[]string{"stroke:NONE"}, true},
		{[]string{"stroke:rgba(0,0,0,0)"}, true},
		{[]string{"stroke:none", "stroke:red"}, true},
	} {
		if got := Casing("x", path, [2]scene.Marker{}, c.line, false) != nil; got != c.cased {
			t.Errorf("Casing of %q: %v", c.line, got)
		}
	}
	if it := Casing("x", path, [2]scene.Marker{}, []string{"stroke-width:3"}, false); it.Style.StrokeWidth.Or(0) != 7 {
		t.Errorf("width %v", it.Style.StrokeWidth)
	}
}

// A casing doesn't share its edge's path, which snapping would then move as the casing's.
func TestCasingOwnsItsPath(t *testing.T) {
	path := new(scene.Path).MoveTo(0, 0).LineTo(10, 0)
	if it := Casing("x", path, [2]scene.Marker{}, nil, false); it.Parts[0].Path == path {
		t.Error("the casing shares the edge's path")
	}
}
