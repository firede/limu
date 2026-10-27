package style

import "testing"

func TestParseColor(t *testing.T) {
	for _, c := range []struct {
		in   string
		want Color
		ok   bool
	}{
		{"#f9f", Color{255, 153, 255, 255}, true},
		{"#F9F8", Color{255, 153, 255, 136}, true},
		{"#123456", Color{0x12, 0x34, 0x56, 255}, true},
		{"#12345678", Color{0x12, 0x34, 0x56, 0x78}, true},
		{"#12345", Color{}, false},
		{"#ggg", Color{}, false},
		{"red", Color{255, 0, 0, 255}, true},
		{" RebeccaPurple ", Color{102, 51, 153, 255}, true},
		{"transparent", Color{}, true},
		{"rgb(255, 0, 128)", Color{255, 0, 128, 255}, true},
		{"rgba(255,0,128,0.5)", Color{255, 0, 128, 128}, true},
		{"rgb(100% 0% 50% / 50%)", Color{255, 0, 128, 128}, true},
		{"hsl(120, 100%, 50%)", Color{0, 255, 0, 255}, true},
		{"hsl(0deg 100% 50%)", Color{255, 0, 0, 255}, true},
		{"hsla(240, 100%, 50%, 0.25)", Color{0, 0, 255, 64}, true},
		{"var(--x)", Color{}, false},
		{"currentColor", Color{}, false},
		{"rgb(1,2)", Color{}, false},
	} {
		got, ok := ParseColor(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("ParseColor(%q) = %v, %v; want %v, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestColorString(t *testing.T) {
	if s := Hex("#0a0b0c").String(); s != "#0a0b0c" {
		t.Errorf("got %s", s)
	}
	if s := (Color{1, 2, 3, 4}).String(); s != "#01020304" {
		t.Errorf("got %s", s)
	}
}

// What leaves the cap and join unset resolves them to SVG's initial values; what sets them
// is cascaded as the rest of a style is.
func TestCapJoin(t *testing.T) {
	var look *Look
	if r := look.Resolve("x", Style{}, Style{}); r.Cap != CapButt || r.Join != JoinMiter {
		t.Errorf("unset %v %v", r.Cap, r.Join)
	}
	over := &Look{Overrides: Overrides{"x.edge": {Join: JoinBevel}}}
	own := Style{Cap: CapRound, Join: JoinRound}
	if r := over.Resolve("x.edge", own, Style{}); r.Cap != CapRound || r.Join != JoinBevel {
		t.Errorf("override %v %v", r.Cap, r.Join)
	}
	if r := over.Resolve("x.edge", own, Style{Cap: CapSquare}); r.Cap != CapSquare || r.Join != JoinBevel {
		t.Errorf("author %v %v", r.Cap, r.Join)
	}
	if s := look.Cascade("x", Style{}, Style{}); s.Cap.Set() || s.Join.Set() {
		t.Errorf("cascade settled %v %v", s.Cap, s.Join)
	}
	if CapRound.String() != "round" || JoinMiter.String() != "miter" || Cap(0).String() != "" {
		t.Error("names")
	}
}

// A fill and its opacity cascade each on their own, as CSS's do: an author's fill over a
// part the diagram draws at an opacity is drawn at that opacity, and an author's opacity
// over the diagram's fill draws that fill at it.
func TestOpacitiesCascade(t *testing.T) {
	var look *Look
	red := Of(Color{255, 0, 0, 255})
	own := Style{Fill: SurfaceAlt, FillOpacity: N(0.5), Stroke: Border}
	if r := look.Resolve("x", own, Style{Fill: red}); r.Fill != (Color{255, 0, 0, 128}) {
		t.Errorf("author's fill at the diagram's opacity: %v", r.Fill)
	}
	author := Style{FillOpacity: N(0.25), StrokeOpacity: N(0.5)}
	r := look.Resolve("x", Style{Fill: Surface, Stroke: red}, author)
	surface := Default().Surface
	if r.Fill != (Color{surface.R, surface.G, surface.B, 64}) || r.Stroke != (Color{255, 0, 0, 128}) {
		t.Errorf("author's opacities over the diagram's paints: %v %v", r.Fill, r.Stroke)
	}
	if r.Marker != (Color{255, 0, 0, 255}) {
		t.Errorf("a marker is filled in %v", r.Marker)
	}
	if s := look.Cascade("x", own, author); s.FillOpacity != N(0.25) || s.StrokeOpacity != N(0.5) || s.Fill != SurfaceAlt {
		t.Errorf("cascade %+v", s)
	}
}

func TestCascade(t *testing.T) {
	look := &Look{Overrides: Overrides{"flowchart.node": {Fill: Accent, StrokeWidth: N(2)}}}
	own := Style{Fill: Surface, Stroke: Border, Text: Fg}
	author := Style{Stroke: Of(Hex("#123"))}

	s := look.Cascade("flowchart.node", own, author)
	if s.Fill != Accent || s.Stroke != Of(Hex("#123")) || s.Text != Fg {
		t.Errorf("paints %+v", s)
	}
	if w, _ := s.StrokeWidth.Get(); w != 2 {
		t.Errorf("stroke width %v", w)
	}
	if r, _ := s.Radius.Get(); r != 4 {
		t.Errorf("radius from the theme %v", r)
	}

	other := look.Cascade("flowchart.edge", own, Style{})
	if other.Fill != Surface {
		t.Errorf("an override reached another role: %+v", other)
	}

	var nothing *Look
	if r := nothing.Resolve("x", own, Style{}); r.Fill != Default().Surface || r.Opacity != 1 {
		t.Errorf("nil look %+v", r)
	}
}

func TestPaint(t *testing.T) {
	th := Default()
	if (Paint{}).Set() || (Paint{}).Visible() || !None.Set() || None.Visible() {
		t.Error("unset and none")
	}
	if c := Series(9).Color(th); c != th.Series[1] {
		t.Errorf("series wraps: %v", c)
	}
	if c := Surface.Alpha(0.5).Alpha(0.5).Color(th); c.A != 64 {
		t.Errorf("alpha %v", c.A)
	}
	if Surface.Alpha(0) != None {
		t.Error("alpha 0 is none")
	}
	if n := Series(2).Token().Name(); n != "series-3" {
		t.Errorf("name %q", n)
	}
}

func TestTextDefaultsToFg(t *testing.T) {
	var look Look
	if r := look.Resolve("x", Style{}, Style{}); r.Text != Default().Fg || r.Fill.A != 0 {
		t.Errorf("got %+v", r)
	}
}

func TestOver(t *testing.T) {
	th := Default()
	th.SurfaceAlt, th.Bg = Hex("#000"), Hex("#fff")
	if c := SurfaceAlt.Alpha(0.75).Over(Bg).Color(th); c != (Color{64, 64, 64, 255}) {
		t.Errorf("got %v", c)
	}
	if p := None.Over(Bg); p != None {
		t.Errorf("none over anything is none: %v", p)
	}
}

func TestInkOnFill(t *testing.T) {
	look := Look{Overrides: Overrides{"x.lit": {Fill: Of(Hex("#ffd700"))}}}
	th := Default()
	own := Style{Text: Muted, OnFill: true}
	for _, c := range []struct {
		name        string
		role        string
		own, author Style
		text        Color
	}{
		{"no fill set", "x", own, Style{}, th.Muted},
		{"a dark fill", "x", own, Style{Fill: Of(Hex("#000080"))}, th.Bg},
		{"a light fill", "x", own, Style{Fill: Of(Hex("#ffe4e1"))}, th.Fg},
		// #7d7d7d is just darker than where #1f1f1f and #fff contrast with it equally, #7f7f7f
		// just lighter.
		{"just dark", "x", own, Style{Fill: Of(Hex("#7d7d7d"))}, th.Bg},
		{"just light", "x", own, Style{Fill: Of(Hex("#7f7f7f"))}, th.Fg},
		{"a text color set", "x", own, Style{Fill: Of(Hex("#000")), Text: Of(Hex("#f00"))}, Hex("#f00")},
		{"a faint fill", "x", own, Style{Fill: Of(Hex("#000")), FillOpacity: N(0.7), Opacity: N(0.7)}, th.Muted},
		{"a translucent color", "x", own, Style{Fill: Of(Color{0, 0, 0, 100})}, th.Muted},
		{"not on its fill", "x", Style{Text: Muted}, Style{Fill: Of(Hex("#000"))}, th.Muted},
		{"an override's fill", "x.lit", own, Style{}, th.Fg},
	} {
		if got := look.Resolve(c.role, c.own, c.author).Text; got != c.text {
			t.Errorf("%s: text %v, want %v", c.name, got, c.text)
		}
	}
	// In a dark theme, the light Fg is the one that stands out on a dark fill.
	dark := Look{Theme: KohuDark()}
	if got := dark.Resolve("x", own, Style{Fill: Of(Hex("#000080"))}).Text; got != KohuDark().Fg {
		t.Errorf("dark theme on navy: %v", got)
	}
}
