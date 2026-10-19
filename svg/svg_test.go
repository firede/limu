package svg

import (
	"encoding/xml"
	"flag"
	"io"
	"math"
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
				Author: style.Style{Fill: style.Of(style.Hex("#f9f")), StrokeWidth: style.N(2)},
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

// A part painted Shade is filled with the background, or over a fill its author set, with
// the background at 60%, and not stroked; a fill with an alpha of its own does not change
// that, as variables or as colors.
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
	for _, fill := range []string{"#ff9966", "#ff996680"} {
		sc.Items[0] = item(style.Style{Fill: style.Of(style.Hex(fill))})
		if got := Render(sc, nil); !strings.Contains(got, `height="10" fill="var(--limu-bg,#ffffff)" fill-opacity="0.6" stroke="none"/>`) {
			t.Errorf("%s: no stripe of the background at 60%%:\n%s", fill, got)
		}
		if got := Render(sc, &Options{Literal: true}); !strings.Contains(got, `height="10" fill="#ffffff" fill-opacity="0.6" stroke="none"/>`) {
			t.Errorf("%s: no stripe of the background at 60%%, written as a color:\n%s", fill, got)
		}
	}
}

// A color of the author's own with an alpha is written opaque, its alpha its opacity, and
// not twice: fill:#ff000080 is drawn at 0.5, not at 0.25.
func TestRenderOwnAlpha(t *testing.T) {
	sc := sample()
	sc.Items[3].Author = style.Style{Fill: style.Of(style.Hex("#ff000080")), Stroke: style.Of(style.Hex("#00ff0040"))}
	for _, literal := range []bool{false, true} {
		got := Render(sc, &Options{Literal: literal})
		if !strings.Contains(got, `fill="#ff0000" fill-opacity="0.5" stroke="#00ff00" stroke-opacity="0.25"`) {
			t.Errorf("literal %v: the author's alpha not written once:\n%s", literal, got)
		}
	}
}

// A style's fill and stroke opacities are written with its paints, times the paints' own;
// a marker is filled in the stroke's color alone and stroked at the stroke opacity.
func TestRenderFillOpacity(t *testing.T) {
	sc := sample()
	sc.Items[3].Author = style.Style{Fill: style.Of(style.Hex("#ff000080")), FillOpacity: style.N(0.5), StrokeOpacity: style.N(0.25)}
	got := Render(sc, &Options{Literal: true})
	if !strings.Contains(got, `fill="#ff0000" fill-opacity="0.25"`) || !strings.Contains(got, `stroke-opacity="0.25"`) {
		t.Errorf("opacities not written:\n%s", got)
	}
	edge := &scene.Scene{Width: 20, Height: 20, Items: []scene.Item{{
		Kind: scene.Edge, Role: "flowchart.edge",
		Parts: []scene.Part{{Path: new(scene.Path).MoveTo(0, 10).LineTo(20, 10), Paint: scene.StrokeOnly}},
		Ends:  [2]scene.Marker{{}, {Type: "arrow_point", At: scene.Point{X: 20, Y: 10}, DX: 1}},
		Style: style.Style{Stroke: style.Of(style.Hex("#f00")), StrokeOpacity: style.N(0.25)},
	}}}
	got = Render(edge, &Options{Literal: true})
	if !strings.Contains(got, `class="limu-marker" d="M20,10L12,13.36L12,6.64Z" fill="#ff0000" stroke="none"/>`) {
		t.Errorf("marker not filled opaque:\n%s", got)
	}
}

// An item's tags and then its classes are written as the element's classes.
func TestRenderTagsAndClasses(t *testing.T) {
	sc := &scene.Scene{Width: 10, Height: 10, Items: []scene.Item{{
		Kind: scene.Node, Role: "c4.element", Tags: []string{"person", "external"}, Class: []string{"mine"},
	}}}
	if got := Render(sc, nil); !strings.Contains(got, `class="limu-node limu-c4-element person external mine"`) {
		t.Errorf("classes not written:\n%s", got)
	}
}

// icons is an IconMarkup that has every icon, a circle.
func icons(string) string { return `<circle cx="12" cy="12" r="10"/>` }

// An icon in a label is drawn in the color of the label's text, as an item's picture is,
// at its opacity.
func TestRenderLabelIconColor(t *testing.T) {
	l := text.Layout("fa:fa-car Car", text.Options{Measurer: text.Arial, Size: 16, Icons: func(string) bool { return true }})
	sc := &scene.Scene{Type: "t", Width: 100, Height: 40, FontSize: 16, Items: []scene.Item{
		{Kind: scene.Node, Label: &l, At: scene.Point{X: 50, Y: 20}, Style: style.Style{Text: style.Muted.Alpha(0.5)},
			Picture: &scene.Picture{Icon: "fa:fa-tree", Box: scene.Rect{X: 0, Y: 0, W: 20, H: 20}}},
	}}
	got := Render(sc, &Options{IconMarkup: icons})
	for _, icon := range []string{"fa:fa-car", "fa:fa-tree"} {
		at := strings.Index(got, `data-icon="`+icon+`"`)
		if at < 0 {
			t.Fatalf("no icon %s:\n%s", icon, got)
		}
		tag := got[at : at+strings.Index(got[at:], ">")]
		if !strings.Contains(tag, ` color="var(--limu-muted,#6e6e6e)" opacity="0.5" `) {
			t.Errorf("icon %s not in the text's color: %s", icon, tag)
		}
	}
}

// A label turned by its Rotate is written in a group turned about its At, its icons in it:
// the title of a lane of a swimlane diagram drawn across, with an icon, keeps its icon by
// its text, where the icon fell outside the diagram, at x -18.9.
func TestRenderRotatedIcon(t *testing.T) {
	l := text.Layout("fa:fa-user Customer", text.Options{Measurer: text.Arial, Size: 16, Icons: func(string) bool { return true }})
	sc := &scene.Scene{Type: "swimlane", Width: 60, Height: 200, FontSize: 16, Items: []scene.Item{
		{Kind: scene.Text, Role: "swimlane.title", Label: &l, At: scene.Point{X: 20, Y: 100}, Rotate: -90,
			Style: style.Style{Halo: style.Bg, HaloWidth: style.N(4)}},
	}}
	got := Render(sc, &Options{IconMarkup: icons})
	open := strings.Index(got, `<g transform="rotate(-90 20 100)">`)
	icon := strings.Index(got, `<svg class="limu-icon"`)
	end := strings.LastIndex(got, "</g></g>")
	if open < 0 || icon < open || end < icon || strings.Count(got, "<text ") != 2 || strings.Contains(got, "<text text-anchor=\"middle\" xml:space=\"preserve\" transform") {
		t.Errorf("the label and its icon not turned together:\n%s", got)
	}
}

// A Draw of one's own writes a paint with its opacity, and keeps the default's label.
func TestRenderDrawPieces(t *testing.T) {
	got := Render(sample(), &Options{Draw: func(w *Writer, it *scene.Item, s style.Style, next func()) {
		if it.ID != "g1" {
			next()
			return
		}
		w.Raw(`<g class="mine"><circle`)
		w.PaintAttr("fill", s.Fill)
		w.Raw(` opacity="`)
		w.Num(w.Opacity(s.Fill))
		w.Raw(`"/>`)
		w.Parts(it, s)
		w.Label(it, s)
		w.Raw(`</g>`)
	}})
	want := `<g class="mine"><circle fill="var(--limu-surface-alt,#e6e6e6)" fill-opacity="0.5" opacity="0.5"/><rect x="8" y="8" width="224" height="204" rx="4" fill="var(--limu-surface-alt,#e6e6e6)" fill-opacity="0.5" stroke="var(--limu-border,#8a8a8a)"/><text text-anchor="middle" xml:space="preserve" fill="var(--limu-muted,#6e6e6e)"><tspan x="120" y="29.6">Group &amp; co</tspan></text></g>`
	if !strings.Contains(got, want) {
		t.Errorf("no %s in\n%s", want, got)
	}
}

// What XML does not allow is not written, in text or in attributes, and what is written
// reads as XML.
func TestRenderValidXML(t *testing.T) {
	bad := "a\xffb\uFFFEc\uFFFFd\xed\xa0\xbde\x01f\x7f<&>"
	l := text.Layout("x", text.Options{Measurer: text.Arial, Size: 16})
	sc := &scene.Scene{Type: bad, Title: bad, Desc: bad, Width: 10, Height: 10, FontSize: 16, Items: []scene.Item{
		{Kind: scene.Node, ID: bad, Class: []string{bad}, Tooltip: bad, Label: &l},
	}}
	got := Render(sc, nil)
	d := xml.NewDecoder(strings.NewReader(got))
	for {
		_, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%v in\n%s", err, got)
		}
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
	sc.Items[0].Author = style.Style{Cap: style.CapSquare, Join: style.JoinBevel}
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
		// The product is rounded before Math.round takes it: fused into one multiply-add
		// with the half, as Go does on arm64, -0.005 would be -0.01.
		-0.005: "0", 22.505: "22.51",
		math.NaN(): "0", math.Inf(1): "0", math.Inf(-1): "0",
		1e300: "1" + strings.Repeat("0", 300), -1e300: "-1" + strings.Repeat("0", 300),
	} {
		var b strings.Builder
		num(&b, in)
		if b.String() != want {
			t.Errorf("num(%v) = %s, want %s", in, b.String(), want)
		}
	}
}

func TestEsc(t *testing.T) {
	for in, want := range map[string]string{
		"a<b>&\"c\"\x01\x7f\td": "a&lt;b&gt;&amp;&quot;c&quot;\td",
		// Bytes that are not UTF-8 are U+FFFD, a surrogate's encoding three of them.
		"a\xffb": "a�b", "\xed\xa0\xbd!": "���!", "\xe2\x82": "��",
		// U+FFFE and U+FFFF are left out; other characters are kept.
		"x￾y￿z": "xyz", "é�😀\U0010FFFD": "é�😀\U0010FFFD",
	} {
		var b strings.Builder
		esc(&b, in)
		if b.String() != want {
			t.Errorf("esc(%q) = %q, want %q", in, b.String(), want)
		}
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
