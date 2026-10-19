package main

import (
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/internal/gallery"
)

// tester shows the playground with a text, as -png does, without a window.
func tester(t *testing.T, src, look string) (*app, *ui.Tester) {
	t.Helper()
	a := newApp(src, look)
	return a, testerFor(a)
}

func testerFor(a *app) *ui.Tester {
	tt := ui.NewTester(a.view, width, height)
	tt.Frame()
	return tt
}

// The playground lays out and draws every featured example, in either appearance, and says
// what it is in the status bar.
func TestFeaturedExamples(t *testing.T) {
	for i, typ := range gallery.All() {
		for _, ex := range typ.Examples {
			if !ex.Featured {
				continue
			}
			look := appearanceNames[1+i%2]
			a, tt := tester(t, ex.Source, look)
			if a.scene == nil || a.problem != nil {
				t.Errorf("%s %q in %s: no diagram (%+v)", typ.Type, ex.Title, look, a.problem)
				continue
			}
			if want := typ.Title + " · laid out in "; !tt.HasText(want) {
				t.Errorf("%s %q: no %q in %q", typ.Type, ex.Title, want, tt.Texts())
			}
		}
	}
}

// The playground starts from the first featured flowchart.
func TestFirstSample(t *testing.T) {
	fc, _ := gallery.Find("flowchart")
	if firstSample() != fc.Examples[0].Source || !fc.Examples[0].Featured {
		t.Error("the first sample is not the first featured flowchart")
	}
	a, _ := tester(t, firstSample(), appearanceLight)
	if a.scene == nil || a.typ != "flowchart" {
		t.Errorf("first sample laid out as %v, %q", a.scene != nil, a.typ)
	}
}

// An edit is laid out editDelay after it, not before; an error keeps the last diagram,
// dimmed, says where it is, underlines it, and selects it when the status is clicked.
func TestEditAndError(t *testing.T) {
	a, tt := tester(t, "flowchart LR\n  A --> B\n", appearanceLight)
	good := a.scene
	if good == nil {
		t.Fatal("no diagram")
	}
	src := "flowchart LR\n  A --> B\n  C{{ --> }\n"
	a.buf.Set(src)
	tt.Frame()
	if a.problem != nil || a.plan.version == a.buf.Version() {
		t.Fatal("laid out before the delay")
	}
	tt.Advance(editDelay)
	if a.problem == nil {
		t.Fatal("no problem after the delay")
	}
	if a.scene != good {
		t.Error("the last good diagram is not kept")
	}
	want := "Line 3, column 11: Parse error: Expecting 'DIAMOND_STOP', got 'TEXT'"
	if a.problem.Status != want || !tt.HasText(want) {
		t.Errorf("status %q, texts %q", a.problem.Status, tt.Texts())
	}
	ds := a.code.Diagnostics()
	if len(ds) != 1 || src[ds[0].Start:ds[0].End] != "}" {
		t.Fatalf("diagnostics %+v", ds)
	}
	if err := tt.Click(want); err != nil {
		t.Fatal(err)
	}
	if s := a.code.Selection(); s.Anchor != ds[0].Start || s.Head != ds[0].End {
		t.Errorf("selection %+v, want the diagnostic's %d to %d", s, ds[0].Start, ds[0].End)
	}
	// Mending the text shows its diagram and clears the diagnostic.
	a.buf.Set("flowchart LR\n  A --> B --> C\n")
	tt.Frame()
	tt.Advance(editDelay)
	if a.problem != nil || a.scene == good || len(a.code.Diagnostics()) != 0 {
		t.Errorf("after the fix: problem %+v, diagnostics %+v", a.problem, a.code.Diagnostics())
	}
}

// A text loaded, as an example, is laid out at once.
func TestLoad(t *testing.T) {
	a, tt := tester(t, "flowchart LR\n  A --> B\n", appearanceLight)
	pie, _ := gallery.Find("pie")
	a.load(pie.Examples[0].Source)
	tt.Frame()
	if a.typ != "pie" {
		t.Errorf("loaded a %q", a.typ)
	}
}

// Blank text shows a placeholder.
func TestBlank(t *testing.T) {
	a, tt := tester(t, "  \n", appearanceLight)
	if a.scene != nil || !a.blank || !tt.HasText("Nothing to draw yet") {
		t.Errorf("texts %q", tt.Texts())
	}
}

// A narrower pane lays the diagram out again to fit it; another appearance only draws it
// again, in the scene it has.
func TestWidthAndAppearance(t *testing.T) {
	a, tt := tester(t, "flowchart LR\n  A --> B --> C --> D --> E --> F --> G\n", appearanceLight)
	sc, stamp, wide := a.scene, a.plan.stamp, a.scene.Width
	a.setAppearance(appearanceDark)
	tt.Frame()
	if a.plan.stamp != stamp || a.scene != sc {
		t.Error("a change of appearance laid the diagram out again")
	}
	if got := tt.Image().RGBAAt(4, 4); got != (color.RGBA{0x0c, 0x16, 0x18, 0xff}) {
		t.Errorf("the toolbar is %v in Dark", got)
	}
	tt.SetSize(700, height)
	tt.Frame()
	if a.plan.stamp == stamp || !(a.scene.Width < wide) {
		t.Errorf("a narrower pane laid out %v wide, as before %v", a.scene.Width, wide)
	}
}

// System follows the system's appearance.
func TestSystemAppearance(t *testing.T) {
	_, tt := tester(t, "flowchart LR\n  A --> B\n", appearanceSystem)
	if got := tt.Image().RGBAAt(4, 4); got != (color.RGBA{0xf1, 0xf6, 0xf5, 0xff}) {
		t.Errorf("the toolbar is %v in a light system", got)
	}
	tt.SetDark(true)
	tt.Frame()
	if got := tt.Image().RGBAAt(4, 4); got != (color.RGBA{0x0c, 0x16, 0x18, 0xff}) {
		t.Errorf("the toolbar is %v in a dark system", got)
	}
}

// The SVG view shows the scene's SVG, indented, in the appearance, and Copy copies it as
// the svg package wrote it.
func TestSVGView(t *testing.T) {
	a, tt := tester(t, "flowchart LR\n  A --> B\n", appearanceDark)
	a.shows = viewSVG
	tt.Frame()
	text := a.svgBuf.String()
	if !strings.HasPrefix(text, "<svg ") || !strings.Contains(text, "\n  <g ") || !strings.Contains(text, "#d0dddc") {
		t.Errorf("SVG view %.200q", text)
	}
	if err := tt.Click("Copy"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != a.svgDoc || strings.Contains(got, "\n") {
		t.Errorf("copied %.200q", got)
	}
}

// A PNG is the scene at twice its size, on the appearance's background.
func TestScenePNG(t *testing.T) {
	a, _ := tester(t, "flowchart LR\n  A --> B\n", appearanceDark)
	img := scenePNG(a.scene, appearanceNamed(appearanceDark, false), 2)
	b := img.Bounds()
	if b.Dx() != 2*int(math.Ceil(a.scene.Width)) || b.Dy() != 2*int(math.Ceil(a.scene.Height)) {
		t.Errorf("image %v for a scene %v by %v", b, a.scene.Width, a.scene.Height)
	}
	if got := img.RGBAAt(1, 1); got != (color.RGBA{0x0c, 0x16, 0x18, 0xff}) {
		t.Errorf("background %v", got)
	}
}

// The item under the pointer is named in the status bar.
func TestHovered(t *testing.T) {
	a, tt := tester(t, "flowchart LR\n  A[Start] --> B\n", appearanceLight)
	r, ok := tt.Find("Flowchart diagram")
	if !ok {
		t.Fatal("no diagram")
	}
	for _, it := range a.scene.Items {
		if it.ID == "A" && it.Role == "flowchart.node" {
			tt.Move(r.X+float32(it.Box.X+it.Box.W/2), r.Y+float32(it.Box.Y+it.Box.H/2))
			break
		}
	}
	if !tt.HasText("flowchart.node A") {
		t.Errorf("texts %q", tt.Texts())
	}
}

// The Examples menu has a submenu for each type, of its examples, and choosing one shows
// it; the Appearance menu changes the appearance.
func TestMenus(t *testing.T) {
	a, tt := tester(t, "flowchart LR\n  A --> B\n", appearanceLight)
	if err := tt.Click("Examples"); err != nil {
		t.Fatal(err)
	}
	items := tt.Menu()
	all := gallery.All()
	if len(items) != len(all) {
		t.Fatalf("Examples has %d items, want one for each of %d types", len(items), len(all))
	}
	for _, typ := range all {
		if typ.Beta && !strings.Contains(strings.Join(items, "|"), typ.Title+" (beta)") {
			t.Errorf("%s is not marked beta in %q", typ.Title, items)
		}
	}
	pie, _ := gallery.Find("pie")
	last := pie.Examples[len(pie.Examples)-1]
	if err := tt.ChooseMenuItem(pie.Title, last.Title); err != nil {
		t.Fatal(err)
	}
	if a.typ != "pie" || a.buf.String() != last.Source {
		t.Errorf("chose a pie, showing a %q", a.typ)
	}
	if err := tt.Click("Appearance"); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(tt.Menu(), "|"); got != "System|-|Light|Dark" {
		t.Errorf("Appearance menu %q", got)
	}
	if err := tt.ChooseMenuItem(appearanceDark); err != nil {
		t.Fatal(err)
	}
	if a.appearance != appearanceDark {
		t.Errorf("appearance %q", a.appearance)
	}
}

// Copy link copies a link to the website's playground with the text, which the playground
// reads back; a text too long for a link is not copied.
func TestCopyLink(t *testing.T) {
	src := "flowchart LR\n  A[Start] --> B\n"
	_, tt := tester(t, src, appearanceLight)
	if err := tt.Click("Copy link"); err != nil {
		t.Fatal(err)
	}
	link := tt.Clipboard()
	hash, ok := strings.CutPrefix(link, siteURL)
	if !ok {
		t.Fatalf("copied %q", link)
	}
	if got, ok := linkedSource(hash); !ok || got != src {
		t.Errorf("the link holds %q, %v", got, ok)
	}
	_, tt = tester(t, strings.Repeat("%%\n", maxLinkHash), appearanceLight)
	tt.SetClipboard("")
	if err := tt.Click("Copy link"); err != nil {
		t.Fatal(err)
	}
	if tt.Clipboard() != "" || !tt.HasText("This diagram's text is too long for a link") {
		t.Errorf("copied %.40q of a text too long", tt.Clipboard())
	}
}

// A narrow window has the text above the diagram, rather than beside it.
func TestNarrow(t *testing.T) {
	_, tt := tester(t, "flowchart LR\n  A --> B\n", appearanceLight)
	tt.SetSize(400, height)
	tt.Frame()
	tt.Frame()
	code, ok1 := tt.Find("Mermaid text")
	drawing, ok2 := tt.Find("Flowchart diagram")
	if !ok1 || !ok2 || !(code.Y+code.H <= drawing.Y) {
		t.Errorf("text at %+v, diagram at %+v", code, drawing)
	}
}

// The page that loads the playground in a browser is told once that it shows what the
// text made, a diagram or an error.
func TestShown(t *testing.T) {
	for _, src := range []string{"flowchart LR\n  A --> B\n", "flowchart LR\n  A -->\n", ""} {
		a := newApp(src, appearanceLight)
		n := 0
		a.shown = func() { n++ }
		tt := testerFor(a)
		tt.Frame()
		if n != 1 {
			t.Errorf("%q: shown %d times", src, n)
		}
	}
}

// A download is named after the diagram's type.
func TestFileName(t *testing.T) {
	a, _ := tester(t, "pie\n  \"a\": 1\n", appearanceLight)
	if got := a.fileName(".svg"); got != "pie.svg" {
		t.Errorf("fileName = %q", got)
	}
	if got := newApp("", appearanceLight).fileName(".png"); got != "diagram.png" {
		t.Errorf("fileName = %q", got)
	}
}
