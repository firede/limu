package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
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

// The palettes are the website's colors, as website/src/styles/base.scss has them: the
// light theme in the first :root that sets color-scheme, the dark one in its mixin.
func TestPalettes(t *testing.T) {
	f, err := os.Open("../../../website/src/styles/base.scss")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	vars := map[string]map[string]string{}
	block := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "color-scheme: light;":
			block = "light"
		case line == "@mixin dark {":
			block = "dark"
		case line == "}":
			block = ""
		case block != "" && strings.HasPrefix(line, "--"):
			name, value, _ := strings.Cut(strings.TrimSuffix(line, ";"), ": ")
			if vars[block] == nil {
				vars[block] = map[string]string{}
			}
			vars[block][name[2:]] = value
		}
	}
	for block, p := range map[string]palette{"light": kohu, "dark": kohuDark} {
		want := vars[block]
		got := map[string]string{
			"ink": p.ink, "panel": p.panel, "line": p.line, "line-hover": p.lineHover,
			"faint": p.faint, "muted": p.muted, "body": p.body, "paper": p.paper,
			"accent": p.accent, "accent-ink": p.accentInk, "red": p.red, "green": p.green,
			"selection-background": p.selection,
			"syntax-punctuation":   p.syntaxPunctuation, "syntax-keyword": p.syntaxKeyword,
			"syntax-type": p.syntaxType, "syntax-constant": p.syntaxConstant,
			"syntax-number": p.syntaxNumber, "syntax-variable": p.syntaxVariable,
			"syntax-function": p.syntaxFunction, "syntax-string": p.syntaxString,
			"syntax-comment": p.syntaxComment,
		}
		for name, hex := range got {
			css, ok := want[name]
			if !ok {
				t.Errorf("%s: base.scss has no --%s", block, name)
				continue
			}
			if c, ok := cssRGBA(css); ok {
				css = c
			}
			if css != hex {
				t.Errorf("%s: --%s is %s, the palette's %s", block, name, css, hex)
			}
		}
		if p.dark != (block == "dark") {
			t.Errorf("%s: dark %v", block, p.dark)
		}
	}
}

// cssRGBA is a color written as rgb(r g b / a%) in hex, with its alpha, as the palettes
// write translucent colors.
func cssRGBA(s string) (string, bool) {
	var r, g, b, a int
	if _, err := fmt.Sscanf(s, "rgb(%d %d %d / %d%%)", &r, &g, &b, &a); err != nil {
		return "", false
	}
	return fmt.Sprintf("#%02x%02x%02x%02x", r, g, b, int(math.Round(float64(a)*255/100))), true
}
