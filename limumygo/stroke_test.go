package limumygo

import (
	"image"
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// A line is stroked with the ends, corners and dashes SVG strokes it with: one from 0,0 to
// 10,0 and down to 10,10, two units wide, and a closed square, drawn ten DIPs a unit.
func TestStrokePens(t *testing.T) {
	corner := new(scene.Path).MoveTo(0, 0).LineTo(10, 0).LineTo(10, 10)
	square := new(scene.Path).Polygon(0, 0, 10, 0, 10, 10, 0, 10)
	dot := new(scene.Path).MoveTo(5, 5).LineTo(5, 5)
	type pt = scene.Point
	for _, c := range []struct {
		name    string
		path    *scene.Path
		cap     style.Cap
		join    style.Join
		dash    []float64
		in, out []pt
	}{
		{"butt", corner, style.CapButt, style.JoinMiter, nil,
			[]pt{{X: 0.1, Y: 0.9}, {X: 5, Y: -0.9}, {X: 10.9, Y: -0.9}, {X: 9.1, Y: 9.9}},
			[]pt{{X: -0.1, Y: 0}, {X: 10, Y: 10.1}, {X: 5, Y: 1.1}, {X: 8.9, Y: 5}}},
		{"square", corner, style.CapSquare, style.JoinBevel, nil,
			[]pt{{X: -0.9, Y: -0.9}, {X: 10.9, Y: 10.9}, {X: 10.4, Y: -0.4}},
			[]pt{{X: -1.1, Y: 0}, {X: 10.9, Y: -0.9}}},
		{"round", corner, style.CapRound, style.JoinRound, nil,
			[]pt{{X: -0.6, Y: -0.6}, {X: 10.6, Y: -0.6}},
			[]pt{{X: -0.9, Y: -0.9}, {X: 10.9, Y: -0.9}}},
		{"ring", square, style.CapButt, style.JoinMiter, nil,
			[]pt{{X: -0.9, Y: -0.9}, {X: 5, Y: 0.9}, {X: 10.9, Y: 10.9}, {X: 0.9, Y: 0.9}},
			[]pt{{X: 5, Y: 5}, {X: 1.1, Y: 1.1}, {X: -1.1, Y: 5}}},
		{"dot butt", dot, style.CapButt, style.JoinMiter, nil, nil, []pt{{X: 5, Y: 5}}},
		{"dot square", dot, style.CapSquare, style.JoinMiter, nil, []pt{{X: 5.9, Y: 4.1}}, []pt{{X: 6.1, Y: 5}}},
		{"dot round", dot, style.CapRound, style.JoinMiter, nil, []pt{{X: 5.6, Y: 4.4}}, []pt{{X: 5.9, Y: 4.1}}},
		// A turn back to 1 in 10 from where the line came, whose point would be 20 half
		// widths out, past the miter limit: bevelled.
		{"miter limit", new(scene.Path).MoveTo(0, 0).LineTo(10, 0).LineTo(0, 1), style.CapButt, style.JoinMiter, nil,
			[]pt{{X: 9.5, Y: 0}}, []pt{{X: 11.5, Y: 0}}},
		// Dashes of 2 and gaps of 1, flat at their ends.
		{"dashes", new(scene.Path).MoveTo(0, 5).LineTo(11, 5), style.CapButt, style.JoinMiter, []float64{2, 1},
			[]pt{{X: 1, Y: 5}, {X: 4, Y: 5}, {X: 7, Y: 5}, {X: 9.5, Y: 5}},
			[]pt{{X: -0.1, Y: 5}, {X: 2.5, Y: 5}, {X: 5.5, Y: 5}, {X: 8.5, Y: 5}}},
		// Round a square 32 long, the last dash ends 2 into a dash: it goes on through the
		// start, and turns the corner there mitred.
		{"dashes round", new(scene.Path).Polygon(0, 0, 8, 0, 8, 8, 0, 8), style.CapButt, style.JoinMiter, []float64{3, 2},
			[]pt{{X: -0.9, Y: -0.9}, {X: 0, Y: 1}, {X: 2, Y: 0}},
			[]pt{{X: 4, Y: 0}, {X: 0, Y: 3}}},
	} {
		dash := style.Solid
		if c.dash != nil {
			dash = style.Dashes(c.dash...)
		}
		sc := &scene.Scene{Width: 20, Height: 20, Items: []scene.Item{{
			Kind:  scene.Edge,
			Parts: []scene.Part{{Path: c.path, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.Of(style.Hex("#f00")), StrokeWidth: style.N(2), Cap: c.cap, Join: c.join, Dash: dash},
		}}}
		img := ui.Render(func(ctx *ui.Context) {
			ui.Box(ctx).Size(200, 200).Draw(func(p *ui.Painter, r ui.Rect) {
				Paint(p, sc, r.X+20, r.Y+20, &Options{Scale: 10})
			})
		}, 200, 200, 1)
		at := func(q pt) (x, y int) { return int(math.Floor(20 + 10*q.X)), int(math.Floor(20 + 10*q.Y)) }
		for _, q := range c.in {
			if px := img.RGBAAt(at(q)); !(px.R > 200 && px.G < 55) {
				t.Errorf("%s: %v not covered: %v", c.name, q, px)
			}
		}
		for _, q := range c.out {
			if px := img.RGBAAt(at(q)); !(px.R < 55 || px.G > 200) {
				t.Errorf("%s: %v covered: %v", c.name, q, px)
			}
		}
	}
}

// An opaque line stroked in pieces draws what it does stroked whole, but for the
// antialiasing of its edges where the pieces overlap, which is darker, and where a cut, put
// on a quarter of a device pixel, turns a piece's edge a little. A dash or a corner missing
// would differ by far more.
func TestStrokePieces(t *testing.T) {
	zigzag := new(scene.Path).MoveTo(20, 20).LineTo(680, 60).LineTo(40, 400).LineTo(650, 650)
	diamond := new(scene.Path).Polygon(350, 20, 680, 350, 350, 680, 20, 350)
	hexagon := new(scene.Path).Polygon(200, 30, 500, 30, 670, 350, 500, 670, 200, 670, 30, 350)
	for _, c := range []struct {
		name string
		path *scene.Path
		pen  pen
	}{
		{"butt", zigzag, pen{width: 6, cap: style.CapButt, join: style.JoinMiter}},
		{"butt dashed", zigzag, pen{width: 6, dash: []float64{30, 12}, cap: style.CapButt, join: style.JoinMiter}},
		{"square dashed", zigzag, pen{width: 6, dash: []float64{30, 12}, cap: style.CapSquare, join: style.JoinBevel}},
		{"round dashed", zigzag, pen{width: 6, dash: []float64{30, 12, 2}, cap: style.CapRound, join: style.JoinRound}},
		{"closed", hexagon, pen{width: 6, cap: style.CapSquare, join: style.JoinMiter}},
		// About 1866.8 round, the last dash goes on through the start.
		{"closed dashed", diamond, pen{width: 6, dash: []float64{40, 20}, cap: style.CapButt, join: style.JoinMiter}},
		{"closed dashed apart", diamond, pen{width: 6, dash: []float64{40, 30}, cap: style.CapSquare, join: style.JoinBevel}},
	} {
		draw := func(whole bool) *image.RGBA {
			return ui.Render(func(ctx *ui.Context) {
				ui.Box(ctx).Size(700, 700).Draw(func(up *ui.Painter, r ui.Rect) {
					p := &Painter{Painter: up, o: &Options{}, fonts: goFonts, x: float64(r.X), y: float64(r.Y), k: 1}
					red := ui.Color{R: 255, A: 255}
					if whole {
						p.StrokePath(p.with(p.Path(c.path), c.pen, p.dashes(c.pen), 0), p.Len(c.pen.width), red)
					} else {
						p.strokeLines(c.path.Flatten(p.tolerance()), red, c.pen, p.dashes(c.pen))
					}
				})
			}, 700, 700, 1)
		}
		whole, pieces := draw(true), draw(false)
		most := 0
		for i := range whole.Pix {
			d := int(whole.Pix[i]) - int(pieces.Pix[i])
			most = max(most, d, -d)
		}
		if most > 128 {
			t.Errorf("%s: drawn in pieces, a pixel differs by %d", c.name, most)
		}
	}
}
