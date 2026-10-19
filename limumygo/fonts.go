package limumygo

import (
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/text"
)

// Fonts are the fonts a scene is measured and drawn in: a family for text, and one for
// monospaced text, shaped by the system's text engine through ui.Shape, as MyGo shapes the
// text of its elements. As a text.Measurer they measure labels for layout, and the painter
// draws with the same fonts, so that text fills the room layout gave it.
//
// Bold text is set in the family's bold weight, italic text in its italics, and monospaced
// text in the mono family. Fonts keep the text they shaped last, so that measuring and
// drawing a label again costs nothing; they are safe to use from several goroutines.
type Fonts struct {
	family, mono string

	mu       sync.Mutex
	cur, old map[shapeKey]shaped
}

// cacheSize is how many runs of text Fonts keep, at the most twice over: when the runs
// shaped since the cache last turned over reach it, the older ones are let go.
const cacheSize = 4096

type shapeKey struct {
	s    string
	size float32
	face text.Face
}

type shaped struct {
	glyphs []ui.Glyph
	width  float32
}

// NewFonts returns the fonts of a family, and of a family for monospaced text. A family is
// named as ui.Font takes it, a list of families tried in turn; "" is the system's font, and
// "monospace" its monospaced one.
func NewFonts(family, mono string) *Fonts {
	return &Fonts{family: family, mono: mono}
}

// DefaultFonts are the system's font and its monospaced one, which a nil Options.Fonts
// draws in.
var DefaultFonts = NewFonts("", "monospace")

var _ text.Measurer = (*Fonts)(nil)

// Font is the font text in a face is set in at a size, in DIPs.
func (f *Fonts) Font(size float64, face text.Face) ui.Font {
	font := ui.Font{Family: f.family, Size: float32(size)}
	if face&text.Mono != 0 {
		font.Family = f.mono
	}
	if face&text.Bold != 0 {
		font.Weight = 700
	}
	font.Italic = face&text.Italic != 0
	return font
}

// Width is how far text in a face at a size advances, in DIPs: the sum of its glyphs'
// advances, as ui.Shape places them.
func (f *Fonts) Width(s string, size float64, face text.Face) float64 {
	_, w := f.Shape(s, float32(size), face)
	return float64(w)
}

// Shape is text in a face at a size shaped on one line, for ui.Painter.Glyphs, and how far
// it advances. The glyphs are shared: move them only in a copy.
func (f *Fonts) Shape(s string, size float32, face text.Face) ([]ui.Glyph, float32) {
	if s == "" || !(size > 0) {
		return nil, 0
	}
	key := shapeKey{s, size, face}
	f.mu.Lock()
	r, ok := f.cur[key]
	if !ok {
		if r, ok = f.old[key]; ok {
			f.keep(key, r)
		}
	}
	f.mu.Unlock()
	if ok {
		return r.glyphs, r.width
	}
	r.glyphs = ui.Shape(s, f.Font(float64(size), face))
	for _, g := range r.glyphs {
		r.width += g.Advance
	}
	f.mu.Lock()
	f.keep(key, r)
	f.mu.Unlock()
	return r.glyphs, r.width
}

// keep puts a run in the cache, turning it over when it is full; f.mu is held.
func (f *Fonts) keep(key shapeKey, r shaped) {
	if len(f.cur) >= cacheSize || f.cur == nil {
		f.old, f.cur = f.cur, make(map[shapeKey]shaped, 64)
	}
	f.cur[key] = r
}
