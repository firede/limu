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
				Style: style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound},
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

// A part painted Shade is filled with the background, or a tint of a fill its author set,
// and not stroked.
func TestRenderShade(t *testing.T) {
	stripe := scene.Part{Box: scene.Rect{X: 0, Y: 10, W: 50, H: 10}, Paint: scene.Shade}
	item := func(author style.Style) scene.Item {
		return scene.Item{
			Kind: scene.Node, Role: "er.entity", Box: scene.Rect{W: 50, H: 30},
			Parts: []scene.Part{{Box: scene.Rect{W: 50, H: 30}}, stripe},
			Style: style.Style{Fill: style.Surface, Stroke: style.Border}, Author: author,
		}
	}
	sc := &scene.Scene{Width: 50, Height: 30, FontSize: 16, Items: []scene.Item{item(style.Style{})}}
	if got := Render(sc, nil); !strings.Contains(got, `height="10" fill="var(--limu-bg,#ffffff)" stroke="none"/>`) {
		t.Errorf("no stripe of the background:\n%s", got)
	}
	sc.Items[0] = item(style.Style{Fill: style.Of(style.Hex("#ff9966"))})
	if got := Render(sc, nil); !strings.Contains(got, `fill="color-mix(in srgb,var(--limu-bg,#ffffff) 60%,#ff9966)" stroke="none"/>`) {
		t.Errorf("no tint of the author's fill:\n%s", got)
	}
	if got := Render(sc, &Options{Literal: true}); !strings.Contains(got, `fill="#ffd6c2" stroke="none"/>`) {
		t.Errorf("no tint of the author's fill, written as a color:\n%s", got)
	}
}

// A cap and a join are written where they are set: an edge's on its line and on the markers
// it strokes, a marker's own join over the edge's, and an author's over the diagram's.
func TestRenderCapJoin(t *testing.T) {
	line := new(scene.Path).MoveTo(0, 0).LineTo(40, 0).LineTo(40, 30)
	edge := scene.Item{
		Kind: scene.Edge, Role: "class.edge",
		Parts: []scene.Part{{Path: line, Paint: scene.StrokeOnly}},
		Ends: [2]scene.Marker{
			{Type: "arrow_cross", At: scene.Point{}, DX: -1},
			{Type: "class.extension", At: scene.Point{X: 40, Y: 30}, DY: 1},
		},
		Style: style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound},
	}
	sc := &scene.Scene{Width: 50, Height: 40, FontSize: 16, Items: []scene.Item{edge}}
	got := Render(sc, nil)
	for _, want := range []string{
		`<path d="M0,0L40,0L40,30" fill="none" stroke="var(--limu-line,#1f1f1f)" stroke-linecap="round"/>`,
		`<path class="limu-marker" d="M7.5,3.5L0.5,-3.5M7.5,-3.5L0.5,3.5" fill="none" stroke="var(--limu-line,#1f1f1f)" stroke-linecap="round"/>`,
		`stroke-linecap="round" stroke-linejoin="round"/>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %s in\n%s", want, got)
		}
	}
	sc.Items[0].Author = style.Declarations([]string{"stroke-linecap:square", "stroke-linejoin:bevel"}).Style
	got = Render(sc, nil)
	if !strings.Contains(got, `L40,30" fill="none" stroke="var(--limu-line,#1f1f1f)" stroke-linecap="square" stroke-linejoin="bevel"/>`) ||
		!strings.Contains(got, `stroke-linecap="square" stroke-linejoin="round"/>`) {
		t.Errorf("the author's cap and join, under the marker's join:\n%s", got)
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
