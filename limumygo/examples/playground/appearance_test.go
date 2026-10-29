package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/firede/limu/style"
)

func TestAppearances(t *testing.T) {
	cases := []struct {
		name string
		dark bool
		look *style.Theme
	}{
		{appearanceLight, false, style.Kohu()},
		{appearanceDark, true, style.KohuDark()},
	}
	for _, tc := range cases {
		ap := appearanceNamed(tc.name, !tc.dark)
		if ap.Name != tc.name || ap.Dark != tc.dark || ap.UI.Dark != tc.dark {
			t.Errorf("%s: named %q, dark %v and %v", tc.name, ap.Name, ap.Dark, ap.UI.Dark)
		}
		// Diagrams are drawn in limu's own themes, as the website draws them.
		if *ap.Diagram.Theme != *tc.look || ap.Diagram.Overrides != nil {
			t.Errorf("%s: diagrams in %+v", tc.name, ap.Diagram)
		}
		// The window, the code and the diagram are on one background.
		bg := tc.look.Bg
		if ui.RGB(bg.R, bg.G, bg.B) != ap.UI.Background || ap.Code.Background != ap.UI.Background {
			t.Errorf("%s: window on %v, code on %v, diagram on %v", tc.name, ap.UI.Background, ap.Code.Background, bg)
		}
		if ap.Code.Tokens[editor.TokenKeyword].Color == (ui.Color{}) || ap.Code.Tokens[editor.TokenString].Color == (ui.Color{}) {
			t.Errorf("%s: code without colors of its own", tc.name)
		}
	}
	// System follows the system, and a name unknown is System.
	if appearanceNamed(appearanceSystem, false) != appearanceNamed(appearanceLight, true) ||
		appearanceNamed(appearanceSystem, true) != appearanceNamed(appearanceDark, false) ||
		appearanceNamed("GitHub Dark", true) != appearanceNamed(appearanceDark, false) {
		t.Error("System is not Light or Dark as the system is")
	}
	if !knownAppearance(appearanceDark) || knownAppearance("GitHub Dark") {
		t.Error("knownAppearance")
	}
}
