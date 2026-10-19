package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/firede/limu/style"
)

func TestAppearances(t *testing.T) {
	cases := []struct {
		name         string
		dark         bool
		bg, fg, data string
	}{
		{appearanceLight, false, "#ffffff", "#1f2328", "#006edb"},
		{appearanceDark, true, "#0d1117", "#f0f6fc", "#0576ff"},
		{appearanceDarkDimmed, true, "#212830", "#d1d7e0", "#0576ff"},
	}
	for _, tc := range cases {
		ap := appearanceNamed(tc.name, !tc.dark)
		th := ap.Diagram.Theme
		if ap.Name != tc.name || ap.Dark != tc.dark || ap.UI.Dark != tc.dark {
			t.Errorf("%s: named %q, dark %v and %v", tc.name, ap.Name, ap.Dark, ap.UI.Dark)
		}
		if th.Bg != style.Hex(tc.bg) || th.Fg != style.Hex(tc.fg) || th.Series[0] != style.Hex(tc.data) {
			t.Errorf("%s: Bg %v, Fg %v, Series[0] %v", tc.name, th.Bg, th.Fg, th.Series[0])
		}
		// The window and the diagram are on one background, in one text color.
		if ui.Hex(tc.bg) != ap.UI.Background || ui.Hex(tc.fg) != ap.UI.Text || ap.Code.Background != ap.UI.Background {
			t.Errorf("%s: window %v on %v, code on %v", tc.name, ap.UI.Text, ap.UI.Background, ap.Code.Background)
		}
		// Every color of the theme is set, opaque, and the series are told apart.
		colors := map[string]style.Color{
			"Bg": th.Bg, "Fg": th.Fg, "Muted": th.Muted, "Line": th.Line, "Surface": th.Surface,
			"SurfaceAlt": th.SurfaceAlt, "Border": th.Border, "Accent": th.Accent, "Critical": th.Critical,
		}
		seen := map[style.Color]bool{}
		for i, c := range th.Series {
			colors["Series"+string(rune('0'+i))] = c
			if seen[c] {
				t.Errorf("%s: series %d is %v, as one before it", tc.name, i, c)
			}
			seen[c] = true
		}
		for name, c := range colors {
			if c.A != 255 {
				t.Errorf("%s: %s is %v, not opaque", tc.name, name, c)
			}
		}
		if len(colors) != 17 || th.Radius != 6 || th.StrokeWidth != 1 {
			t.Errorf("%s: %d colors, radius %v, stroke %v", tc.name, len(colors), th.Radius, th.StrokeWidth)
		}
		// Groups stand apart from the canvas and from shapes.
		if th.SurfaceAlt == th.Bg || th.SurfaceAlt == th.Surface {
			t.Errorf("%s: SurfaceAlt %v", tc.name, th.SurfaceAlt)
		}
		if ap.Code.Tokens[editor.TokenKeyword].Color == (ui.Color{}) || ap.Code.Tokens[editor.TokenString].Color == (ui.Color{}) {
			t.Errorf("%s: code without colors of its own", tc.name)
		}
	}
	// System follows the system, and a name unknown is System.
	if appearanceNamed(appearanceSystem, false) != appearanceNamed(appearanceLight, true) ||
		appearanceNamed(appearanceSystem, true) != appearanceNamed(appearanceDark, false) ||
		appearanceNamed("Solarized", true) != appearanceNamed(appearanceDark, false) {
		t.Error("System is not GitHub Light or Dark as the system is")
	}
	if !knownAppearance(appearanceDarkDimmed) || knownAppearance("Solarized") {
		t.Error("knownAppearance")
	}
}

// A translucent color laid over another is the mix CSS draws.
func TestOver(t *testing.T) {
	if got := over(style.Hex("#818b981f"), style.Hex("#ffffff")); got != style.Hex("#f0f1f2") {
		t.Errorf("over = %v", got)
	}
}
