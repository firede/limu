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
