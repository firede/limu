package css

import (
	"reflect"
	"testing"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// series writes the paints a palette sets by their series' numbers, as Pele's palette test
// reads its --pele-series-N.
func series(p Palette) map[int]style.Paint {
	out := map[int]style.Paint{}
	for i, s := range p.Series {
		if s.Set() {
			out[i+1] = s
		}
	}
	return out
}

func same(a, b map[int]style.Paint) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

var (
	red  = style.Of(style.Hex("#f00"))
	blue = style.Of(style.Hex("#00f"))
)

// The cases of Pele's palette test, as limu reads the colors it passes on.
func TestReadPalette(t *testing.T) {
	vars := func(v map[string]any) map[string]any { return map[string]any{"themeVariables": v} }
	for _, c := range []struct {
		typ  string
		vars map[string]any
		want map[int]style.Paint
	}{
		{"pie", map[string]any{"pie1": "red", "pie2": "blue"}, map[int]style.Paint{1: red, 2: blue}},
		{"gitGraph", map[string]any{"git0": "red", "git1": "blue"}, map[int]style.Paint{1: red, 2: blue}},
		{"timeline", map[string]any{"cScale0": "red", "cScale1": "blue"}, map[int]style.Paint{1: red, 2: blue}},
		{"radar", map[string]any{"cScale0": "red", "cScale1": "blue"}, map[int]style.Paint{1: red, 2: blue}},
		{"treemap", map[string]any{"cScale0": "red", "cScale1": "blue"}, map[int]style.Paint{1: red, 2: blue}},
		{"mindmap", map[string]any{"cScale0": "green", "cScale1": "red", "cScale2": "blue"}, map[int]style.Paint{1: red, 2: blue}},
		// A venn palette repeats the colors it sets.
		{"venn", map[string]any{"venn2": "red", "venn4": "blue"}, map[int]style.Paint{1: red, 2: blue, 3: red, 4: blue, 5: red, 6: blue, 7: red, 8: blue}},
		// xychart's list repeats, and keeps the commas inside its functions.
		{"xychart", map[string]any{"xyChart": map[string]any{"plotColorPalette": "rgb(255, 0, 0), #00F"}},
			map[int]style.Paint{1: red, 2: blue, 3: red, 4: blue, 5: red, 6: blue, 7: red, 8: blue}},
		{"xychart", map[string]any{"xyChart": map[string]any{"plotColorPalette": "url(a), blue"}}, map[int]style.Paint{2: blue, 4: blue, 6: blue, 8: blue}},
		// Palettes of other diagrams and other theme colors are passed over.
		{"pie", map[string]any{"cScale0": "red", "primaryColor": "red"}, map[int]style.Paint{}},
		{"journey", map[string]any{"fillType0": "red", "cScale0": "red"}, map[int]style.Paint{}},
		{"flowchart", map[string]any{"pie1": "red"}, map[int]style.Paint{}},
		// What limu reads besides hex codes and names, and what draws nothing.
		{"pie", map[string]any{"pie1": "hsl(0.5turn 100% 50%)", "pie2": " Grey ", "pie3": "transparent", "pie4": "currentColor", "pie5": 5},
			map[int]style.Paint{1: style.Of(style.Hex("#0ff")), 2: style.Of(style.Hex("#808080")), 3: style.None, 4: style.Fg}},
	} {
		if got := series(ReadPalette(c.typ, vars(c.vars))); !same(got, c.want) {
			t.Errorf("%s %v: %v, not %v", c.typ, c.vars, got, c.want)
		}
	}
	if got := series(ReadPalette("pie", map[string]any{"themeVariables": "red"})); len(got) != 0 {
		t.Errorf("theme variables that are not a record: %v", got)
	}
}

// Pele's palette passes on colors that depend on the page, which limu can't work out and
// leaves in the theme's colors, and refuses what is not a color.
func TestSafeColor(t *testing.T) {
	for _, value := range []string{
		"var(--pele-accent)", "light-dark(#000, #fff)", "color-mix(in srgb, red 40%, blue)", "oklch(70% 0.1 200)",
		"hsl(0 50% 50% / 0.5)", "rgb(0, 0, 0)", "#FFCC00", "MidnightBlue", "VAR( --pele-x)", "rgb (1 2 3)",
	} {
		if _, ok := safeColor(value); !ok {
			t.Errorf("%q refused", value)
		}
	}
	for _, value := range []string{
		"url(https://example.com/a.svg)", "url(//example.com/a.svg)", "image-set(a 1x)", "red;stroke:blue",
		`"><script>alert(1)</script>`, "expression(alert(1))", "rgb(0, 0", "red)", "red}", "javascript:alert(1)",
		"/*", "reddish", "#ff00000", "var(--page-color)", "color-mix(in srgb, var(--page-color) 50%, red)",
		"((red))", "", "   ",
	} {
		if _, ok := safeColor(value); ok {
			t.Errorf("%q passed", value)
		}
	}
	for _, value := range []string{"var(--pele-accent)", "color-mix(in srgb, red, blue)", "oklch(0.5 0.1 200)", "light-dark(red, blue)"} {
		if p := PaletteColor(value); p.Set() {
			t.Errorf("PaletteColor(%q) = %v, which limu can't work out", value, p)
		}
	}
}

// A label on a series takes a color picked on it where Pele's ink reads the palette's color
// at no less than half its opacity.
func TestPaletteGround(t *testing.T) {
	p := ReadPalette("pie", map[string]any{"themeVariables": map[string]any{
		"pie1": "#ffeb3b", "pie2": "navy", "pie3": "var(--pele-accent)", "pie4": "rgb(0 0 0 / 0.3)",
		"pie5": "transparent", "pie6": "currentColor", "pie7": "hsl(0.5turn 60% 20%)",
	}})
	for i, want := range []bool{true, true, false, false, false, false, true, false} {
		if got := p.Ground(i).Set(); got != want {
			t.Errorf("series %d: ground %v, not %v", i+1, got, want)
		}
	}
	if p.Ground(8) != style.Of(style.Hex("#ffeb3b")) || p.Ground(-7) != style.Of(style.Hex("#000080")) {
		t.Error("the series do not repeat")
	}
}

// The palette's colors go into the author's style of what is drawn in their series, in the
// fields drawn in it, at the opacity they are drawn at, under what the author's own style sets.
func TestPalettePaint(t *testing.T) {
	p := ReadPalette("pie", map[string]any{"themeVariables": map[string]any{"pie1": "red", "pie3": "blue"}})
	items := []scene.Item{
		{Style: style.Style{Fill: style.Series(0), Stroke: style.Bg, Text: style.Series(8)}},
		{Style: style.Style{Fill: style.Series(2).Alpha(0.5), FillOpacity: style.N(0.2)}},
		{Style: style.Style{Fill: style.Series(1), Stroke: style.Series(0)}, Author: style.Style{Stroke: style.Of(style.Hex("#0f0"))}},
	}
	p.Paint(items)
	for k, want := range []style.Style{
		{Fill: red, Text: red},
		{Fill: blue.Alpha(0.5)},
		{Stroke: style.Of(style.Hex("#0f0"))},
	} {
		if got := items[k].Author; !reflect.DeepEqual(got, want) {
			t.Errorf("item %d: author %+v, not %+v", k, got, want)
		}
	}
}
