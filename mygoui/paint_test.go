package mygoui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// An item's picture is drawn through DrawImage or DrawIcon, in its box at the options'
// scale, and an icon of a label through DrawIcon, in the color of the item's text at its
// opacity.
func TestPictureHooks(t *testing.T) {
	l := text.Layout("fa:fa-car Car", text.Options{Measurer: goFonts, Size: 10, Icons: func(string) bool { return true }})
	sc := &scene.Scene{Width: 100, Height: 50, FontSize: 10, Items: []scene.Item{
		{Kind: scene.Node, Picture: &scene.Picture{Image: "https://example.com/a.png", Box: scene.Rect{X: 10, Y: 5, W: 20, H: 10}, Fit: true},
			Style: style.Style{Opacity: style.N(0.5)}},
		{Kind: scene.Node, Picture: &scene.Picture{Icon: "fa:fa-tree", Box: scene.Rect{X: 40, Y: 5, W: 10, H: 10}},
			Label: &l, At: scene.Point{X: 70, Y: 30}, Style: style.Style{Text: style.Accent}},
	}}
	var calls []string
	o := &Options{Fonts: goFonts, Scale: 2,
		DrawImage: func(p *ui.Painter, src string, r ui.Rect, fit bool, opacity float64) {
			calls = append(calls, fmt.Sprintf("image %s %v %v %v", src, r, fit, opacity))
		},
		DrawIcon: func(p *ui.Painter, name string, r ui.Rect, c ui.Color) {
			calls = append(calls, fmt.Sprintf("icon %s %v,%v %v", name, r.X, r.Y, c))
		},
	}
	ui.Render(func(c *ui.Context) { View(c, sc, o) }, 200, 100, 1)
	accent := Color(style.Default().Accent, 1)
	side := 10 * 1.1
	want := []string{
		"image https://example.com/a.png {20 10 40 20} true 0.5",
		fmt.Sprintf("icon fa:fa-tree 80,10 %v", accent),
		fmt.Sprintf("icon fa:fa-car %v,%v %v", float32(2*(70-l.Widths[0]/2)), float32(2*(30-side/2)), accent),
	}
	// A frame may be painted more than once; the last is compared.
	if len(calls) < len(want) || fmt.Sprint(calls[len(calls)-len(want):]) != fmt.Sprint(want) {
		t.Errorf("drawn:\n%q\nwant:\n%q", calls, want)
	}
}

// A part painted Shade is the background, or over a fill its author set, the background at
// 60% opacity, as the svg package writes it.
func TestShade(t *testing.T) {
	bg := style.Hex("#ffffff")
	if got := shade(bg, false); got != bg {
		t.Errorf("no author's fill: %v", got)
	}
	if got := shade(bg, true); got != (style.Color{R: 255, G: 255, B: 255, A: 153}) {
		t.Errorf("over an author's fill: %v", got)
	}
}
