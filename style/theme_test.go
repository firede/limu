package style

import (
	"math"
	"testing"
)

// luminance is a color's relative luminance, as WCAG 2 defines it.
func luminance(c Color) float64 {
	lin := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// contrast is the contrast ratio of two colors, as WCAG 2 defines it.
func contrast(a, b Color) float64 {
	la, lb := luminance(a), luminance(b)
	return (max(la, lb) + 0.05) / (min(la, lb) + 0.05)
}

// Kohu and KohuDark keep the contrast docs/website.md holds them to: text at 7:1 against
// the background and surfaces, muted text, lines and the accent at 4.5:1 against the
// background, the series at 3:1 and outlines at 2.5:1 against the surface.
func TestKohuContrast(t *testing.T) {
	for name, th := range map[string]*Theme{"Kohu": Kohu(), "KohuDark": KohuDark()} {
		check := func(what string, a, b Color, least float64) {
			if r := contrast(a, b); r < least {
				t.Errorf("%s: %s is %.2f:1, less than %g:1", name, what, r, least)
			}
		}
		check("Fg on Bg", th.Fg, th.Bg, 7)
		check("Fg on Surface", th.Fg, th.Surface, 7)
		check("Fg on SurfaceAlt", th.Fg, th.SurfaceAlt, 7)
		check("Muted on Bg", th.Muted, th.Bg, 4.5)
		check("Line on Bg", th.Line, th.Bg, 4.5)
		check("Accent on Bg", th.Accent, th.Bg, 4.5)
		check("Border on Surface", th.Border, th.Surface, 2.5)
		for i, c := range th.Series {
			check("Series "+string(rune('1'+i))+" on Surface", c, th.Surface, 3)
		}
	}
}

// Each call gives a theme of its own, which a host may change without changing the next.
func TestThemesAreFresh(t *testing.T) {
	for name, f := range map[string]func() *Theme{"Default": Default, "Kohu": Kohu, "KohuDark": KohuDark} {
		a := f()
		a.Accent = Color{}
		if b := f(); b.Accent == (Color{}) {
			t.Errorf("%s: changing one theme changed the next", name)
		}
	}
}
