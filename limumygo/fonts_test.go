package limumygo

import (
	"math"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/text"
)

func TestFonts(t *testing.T) {
	f := NewFonts("Go", "Go Mono")
	regular := f.Width("Hello, world", 16, 0)
	if !(regular > 50 && regular < 150) {
		t.Fatalf("width %v: is the text engine there?", regular)
	}
	if bold := f.Width("Hello, world", 16, text.Bold); !(bold > regular) {
		t.Errorf("bold %v, regular %v", bold, regular)
	}
	if italic := f.Width("Hello, world", 16, text.Italic); italic == regular {
		t.Error("italics in the regular face")
	}
	// Each character of a monospaced font advances as far.
	if a, b := f.Width("iiii", 16, text.Mono), f.Width("MMMM", 16, text.Mono); a != b {
		t.Errorf("mono %v and %v", a, b)
	}
	// Pango, on Linux, rounds each glyph's advance to a whole pixel: twice the size is near
	// twice the width only to a DIP a glyph.
	if double := f.Width("Hello, world", 32, 0); math.Abs(double-2*regular) > max(0.02*regular, float64(len("Hello, world"))) {
		t.Errorf("twice the size: %v, %v", double, regular)
	}
	if f.Width("", 16, 0) != 0 || f.Width("x", 0, 0) != 0 {
		t.Error("nothing to measure")
	}

	// The width is that of the glyphs drawn.
	glyphs, w := f.Shape("Hello, world", 16, 0)
	again, _ := f.Shape("Hello, world", 16, 0)
	if &glyphs[0] != &again[0] {
		t.Error("shaped again")
	}
	sum := float32(0)
	for _, g := range ui.Shape("Hello, world", ui.Font{Family: "Go", Size: 16}) {
		sum += g.Advance
	}
	if w != sum || float64(w) != regular {
		t.Errorf("width %v, glyphs %v, measured %v", w, sum, regular)
	}

	// The cache is bounded.
	for i := range 3 * cacheSize {
		f.Width(string(rune('a'+i%26))+string(rune(0x4e00+i)), 12, 0)
	}
	if n := len(f.cur) + len(f.old); n > 2*cacheSize {
		t.Errorf("%d runs kept", n)
	}

	// Layout measures with them.
	l := text.Layout("Is it <b>ready</b>?", text.Options{Measurer: f, Size: 16})
	if l.Width != f.Width("Is it ", 16, 0)+f.Width("ready", 16, text.Bold)+f.Width("?", 16, 0) {
		t.Errorf("label %v", l.Width)
	}
}
