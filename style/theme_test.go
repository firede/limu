package style

import "testing"

// Kohu and KohuDark keep the contrast the website holds them to, in docs/website.md of
// firede/limu-website: text at 7:1 against the background and surfaces, muted text, lines
// and the accent at 4.5:1 against the background, the series at 3:1 and outlines at 2.5:1
// against the surface.
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
