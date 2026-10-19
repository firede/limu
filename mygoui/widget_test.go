package mygoui

import (
	"image/color"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/mygoui/internal/samples"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

func id(it *scene.Item) string {
	if it == nil {
		return "<nil>"
	}
	return it.ID + "/" + string(it.Role)
}

func TestItemAt(t *testing.T) {
	sc := samples.Flowchart(goFonts)
	for _, c := range []struct {
		x, y float64
		want string
	}{
		{100, 60, "A/flowchart.node"},
		{100, 155, "B/flowchart.node"},
		{100, 100, "/flowchart.edge-label"},   // the label of the edge, a text with no box
		{101.5, 85, "L_A_B_0/flowchart.edge"}, // near the line
		{100, 117, "L_A_B_0/flowchart.edge"},  // on the arrow
		{110, 85, "g1/flowchart.cluster"},     // too far from the line
		{200, 200, "g1/flowchart.cluster"},
		{120, 24, "g1/flowchart.cluster"}, // its label
		{2, 2, "<nil>"},
	} {
		if got := id(ItemAt(sc, scene.Point{X: c.x, Y: c.y}, Slop)); got != c.want {
			t.Errorf("at %v,%v: %s, want %s", c.x, c.y, got, c.want)
		}
	}

	// A turned label is found where it is drawn.
	f := samples.Features(goFonts)
	if got := id(ItemAt(f, scene.Point{X: 60, Y: 140}, 0)); got != "up/test.text" {
		t.Errorf("turned label: %s", got)
	}
	if got := id(ItemAt(f, scene.Point{X: 90, Y: 170}, 0)); got != "frame/test.frame" {
		t.Errorf("beside the turned label: %s", got)
	}
	// Lines only are found near them, as edges are.
	if got := id(ItemAt(f, scene.Point{X: 41, Y: 100}, 2)); got != "life/sequence.lifeline" {
		t.Errorf("lifeline: %s", got)
	}
}

func TestView(t *testing.T) {
	sc := samples.Flowchart(goFonts)
	var hovered, clicked *scene.Item
	tt := ui.NewTester(func(c *ui.Context) {
		w := View(c, sc, &Options{Fonts: goFonts, Scale: 2})
		hovered = w.Hovered()
		if it := w.Clicked(); it != nil {
			clicked = it
		}
	}, 480, 440)
	if _, ok := tt.Find("A <sample>"); !ok {
		t.Error("the diagram is not named by its title")
	}
	tt.Move(200, 120) // over A, at twice the size
	if id(hovered) != "A/flowchart.node" {
		t.Errorf("hovered %s", id(hovered))
	}
	if tt.Cursor() != ui.CursorPointer {
		t.Error("no hand over a link")
	}
	// A tip shows once the pointer rests on what has it, which it had not as it came.
	tt.Move(201, 121)
	tt.Advance(2 * time.Second)
	if !tt.HasText("Where it starts") {
		t.Errorf("no tooltip: %q", tt.Texts())
	}
	tt.Move(203, 170) // near the edge
	if id(hovered) != "L_A_B_0/flowchart.edge" {
		t.Errorf("hovered %s", id(hovered))
	}
	if tt.Cursor() == ui.CursorPointer {
		t.Error("a hand over no link")
	}
	tt.ClickAt(200, 310)
	if id(clicked) != "B/flowchart.node" {
		t.Errorf("clicked %s", id(clicked))
	}
}

// Paths too large for one of MyGo's masks are drawn in pieces.
func TestLargePaths(t *testing.T) {
	sc := &scene.Scene{Width: 100, Height: 100, Items: []scene.Item{
		{
			Kind:  scene.Mark,
			Parts: []scene.Part{{Path: new(scene.Path).Polygon(-2000, -2000, 3000, -2000, 3000, 3000, -2000, 3000)}},
			Style: style.Style{Fill: style.Of(style.Hex("#00f")), Stroke: style.None},
		},
		{
			Kind:  scene.Edge,
			Parts: []scene.Part{{Path: new(scene.Path).MoveTo(-3000, 50).CubeTo(-1000, 50, 1000, 50, 3000, 50), Line: true}},
			Style: style.Style{Stroke: style.Of(style.Hex("#f00")), StrokeWidth: style.N(4)},
		},
		{
			Kind:  scene.Edge,
			Parts: []scene.Part{{Path: new(scene.Path).MoveTo(25, -3000).LineTo(25, 3000), Line: true}},
			Style: style.Style{Stroke: style.Of(style.Hex("#0f0")), StrokeWidth: style.N(4), Dash: style.Dashes(1000, 1)},
		},
	}}
	img := ui.Render(func(c *ui.Context) {
		ui.Box(c).Size(100, 100).Draw(func(p *ui.Painter, r ui.Rect) { Paint(p, sc, r.X, r.Y, nil) })
	}, 100, 100, 1)
	for _, c := range []struct {
		x, y int
		want color.RGBA
	}{
		{80, 80, color.RGBA{0, 0, 255, 255}},
		{80, 50, color.RGBA{255, 0, 0, 255}},
		{25, 80, color.RGBA{0, 255, 0, 255}},
	} {
		if got := img.RGBAAt(c.x, c.y); got != c.want {
			t.Errorf("at %d,%d: %v, want %v", c.x, c.y, got, c.want)
		}
	}
}
