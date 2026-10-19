package svg

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

var update = flag.Bool("update", false, "write the golden files anew")

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s differs; look at it, then go test -update\n got %s\nwant %s", name, got, want)
	}
}

// sample is a small flowchart laid out by hand: a group holding a box, a diamond, an edge
// between them with its label, and a title.
func sample() *scene.Scene {
	label := func(s string, size float64, f text.Face) *text.Label {
		l := text.Layout(s, text.Options{Measurer: text.Arial, Size: size, MaxWidth: 200, Face: f})
		return &l
	}
	start := label("Start", 16, 0)
	is := label("Is it <b>ready</b>?", 16, 0)
	yes := label("Yes", 14, 0)
	group := label("Group & co", 16, 0)
	edge := new(scene.Path).MoveTo(100, 80).CubeTo(100, 100, 100, 100, 100, 113)
	return &scene.Scene{
		Type: "flowchart", Width: 240, Height: 220, FontSize: 16, Title: "A <sample>",
		Items: []scene.Item{
			{
				Kind: scene.Group, Role: "flowchart.cluster", ID: "g1",
				Box:   scene.Rect{X: 8, Y: 8, W: 224, H: 204},
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 8, Y: 8, W: 224, H: 204}}},
				Label: group, At: scene.Point{X: 120, Y: 24},
				Style: style.Style{Fill: style.SurfaceAlt.Alpha(0.5), Stroke: style.Border, Text: style.Muted},
			},
			{
				Kind: scene.Edge, Role: "flowchart.edge", ID: "L_A_B_0",
				Parts: []scene.Part{{Path: edge, Paint: scene.StrokeOnly}},
				Ends:  [2]scene.Marker{{}, {Type: "arrow_point", At: scene.Point{X: 100, Y: 120}, DY: 1}},
				Style: style.Style{Stroke: style.Line, Fill: style.None},
			},
			{
				Kind: scene.Text, Role: "flowchart.edge-label",
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 100 - yes.Width/2 - 4, Y: 100 - yes.Height/2, W: yes.Width + 8, H: yes.Height}}},
				Label: yes, At: scene.Point{X: 100, Y: 100},
				Style: style.Style{Fill: style.Bg, Stroke: style.None, Radius: style.N(3)},
			},
			{
				Kind: scene.Node, Role: "flowchart.node", Shape: "rect", ID: "A", Class: []string{"hot"},
				Box:   scene.Rect{X: 60, Y: 40, W: 80, H: 40},
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 60, Y: 40, W: 80, H: 40}}},
				Label: start, At: scene.Point{X: 100, Y: 60},
				Style:  style.Style{Fill: style.Surface, Stroke: style.Border},
				Author: style.Declarations([]string{"fill:#f9f", "stroke-width:2px"}).Style,
				Link:   "https://example.com/?a=1&b=2", Tooltip: "Where it starts",
			},
			{
				Kind: scene.Node, Role: "flowchart.node", Shape: "diam", ID: "B",
				Box:   scene.Rect{X: 30, Y: 120, W: 140, H: 70},
				Parts: []scene.Part{{Path: new(scene.Path).Polygon(100, 120, 170, 155, 100, 190, 30, 155)}},
				Label: is, At: scene.Point{X: 100, Y: 155},
				Style: style.Style{Fill: style.Surface, Stroke: style.Border},
			},
		},
	}
}

func TestRender(t *testing.T) {
	golden(t, "sample.svg", Render(sample(), nil))
}

func TestRenderLiteral(t *testing.T) {
	th := style.Default()
	th.Surface = style.Hex("#ddd")
	got := Render(sample(), &Options{Literal: true, Look: style.Look{Theme: th}})
	if strings.Contains(got, "var(--limu-") && !strings.Contains(got, "var(--limu-font") {
		t.Error("a variable was written")
	}
	if !strings.Contains(got, `fill="#dddddd"`) {
		t.Error("the theme's surface was not used")
	}
	golden(t, "sample-literal.svg", got)
}

func TestRenderOverrides(t *testing.T) {
	look := style.Look{Overrides: style.Overrides{
		"flowchart.node": {Fill: style.Accent, Dash: style.Dashes(3, 2)},
	}}
	got := Render(sample(), &Options{Look: look})
	// The author's fill on A wins over the override; B takes the override.
	if !strings.Contains(got, `fill="#ff99ff"`) || !strings.Contains(got, `fill="var(--limu-accent,#5b7bd5)"`) {
		t.Errorf("overrides not applied:\n%s", got)
	}
	if strings.Count(got, `stroke-dasharray="3 2"`) != 2 {
		t.Error("dashes on both nodes")
	}
}

func TestRenderDraw(t *testing.T) {
	got := Render(sample(), &Options{Draw: func(w *Writer, it *scene.Item, s style.Style, next func()) {
		if it.Shape != "diam" {
			next()
			return
		}
		w.Raw(`<circle class="mine" fill="`)
		w.Paint(s.Fill)
		w.Raw(`"/>`)
	}})
	if !strings.Contains(got, `<circle class="mine" fill="var(--limu-surface,#f3f3f3)"/>`) || strings.Contains(got, "limu-shape-diam") {
		t.Errorf("draw not used:\n%s", got)
	}
	if !strings.Contains(got, "limu-shape-rect") {
		t.Error("next not drawn")
	}
}

func TestNum(t *testing.T) {
	for in, want := range map[float64]string{
		0: "0", 1: "1", -1: "-1", 0.004: "0", -0.004: "0", 12.5: "12.5", 12.05: "12.05",
		12.345: "12.35", 1.999: "2", -3.25: "-3.25", 0.1: "0.1", 123456789012.5: "123456789012.5",
	} {
		var b strings.Builder
		num(&b, in)
		if b.String() != want {
			t.Errorf("num(%v) = %s, want %s", in, b.String(), want)
		}
	}
}

func TestEsc(t *testing.T) {
	var b strings.Builder
	esc(&b, "a<b>&\"c\"\x01\x7f\td")
	if b.String() != "a&lt;b&gt;&amp;&quot;c&quot;\td" {
		t.Errorf("got %q", b.String())
	}
}

func TestSafeLink(t *testing.T) {
	w := &Writer{o: &Options{}}
	for in, want := range map[string]string{
		"https://a.b/c":         "https://a.b/c",
		" HTTP://a.b ":          "HTTP://a.b",
		"mailto:x@y.z":          "mailto:x@y.z",
		"javascript:alert(1)":   "",
		"JaVaScRiPt:alert(1)":   "",
		"data:text/html,x":      "",
		"java\tscript:alert(1)": "",
		"page.html":             "page.html",
		"/notes/a:b":            "/notes/a:b",
		"#section":              "#section",
		"?q=a:b":                "?q=a:b",
		"//evil.example/x":      "",
		"vbscript:x":            "",
	} {
		if got := w.safeLink(in); got != want {
			t.Errorf("safeLink(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAlign(t *testing.T) {
	l := text.Layout("ab<br>abcd", text.Options{Measurer: text.Arial, Size: 10})
	w0, w1 := l.Widths[0], l.Widths[1]
	// Each span is written centered on itself.
	for align, xs := range map[scene.Align][2]float64{
		scene.Middle: {50, 50},
		scene.Start:  {50 + w0/2, 50 + w1/2},
		scene.End:    {50 - w0/2, 50 - w1/2},
	} {
		sc := &scene.Scene{Type: "t", Width: 100, Height: 100, FontSize: 10, Items: []scene.Item{
			{Kind: scene.Text, Label: &l, At: scene.Point{X: 50, Y: 50}, Align: align},
		}}
		got := Render(sc, nil)
		for _, x := range xs {
			var b strings.Builder
			num(&b, x)
			if !strings.Contains(got, `<tspan x="`+b.String()+`"`) {
				t.Errorf("align %d: no line at x %s:\n%s", align, b.String(), got)
			}
		}
	}
}
