package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/limumygo"
	"github.com/firede/limu/limumygo/internal/samples"
	"github.com/firede/limu/scene"
)

// The view reports what the pointer is over, and what it clicked, with its link.
func TestView(t *testing.T) {
	fonts := limumygo.DefaultFonts
	a := &app{scenes: []*scene.Scene{samples.Flowchart(fonts)}, opts: &limumygo.Options{Fonts: fonts}}
	tt := ui.NewTester(a.view, width, height)
	r, ok := tt.Find("A <sample>")
	if !ok {
		t.Fatal("no diagram")
	}
	tt.ClickAt(r.X+100, r.Y+60)
	if want := `Clicked: flowchart.node A "Start" → https://example.com/?a=1&b=2`; !tt.HasText(want) {
		t.Errorf("texts %q", tt.Texts())
	}
	if !tt.HasText(`Over: flowchart.node A "Start"`) {
		t.Errorf("texts %q", tt.Texts())
	}
}
