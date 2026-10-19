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

func TestDeclarations(t *testing.T) {
	a := Declarations([]string{
		"fill:#f9f", "stroke:#333", "stroke-width:4px", "color: red !IMPORTANT",
		"stroke-dasharray: 5 5", "font-weight:bold", "font-style:italic", "font-size:20px",
		"fill-opacity:0.5", "rx:10",
	})
	th := Default()
	if c := a.Style.Fill.Color(th); c != (Color{255, 153, 255, 128}) {
		t.Errorf("fill %v", c)
	}
	if c := a.Style.Stroke.Color(th); c != Hex("#333") {
		t.Errorf("stroke %v", c)
	}
	if c := a.Style.Text.Color(th); c != Hex("#f00") {
		t.Errorf("text %v", c)
	}
	if w, _ := a.Style.StrokeWidth.Get(); w != 4 {
		t.Errorf("stroke width %v", w)
	}
	if r, _ := a.Style.Radius.Get(); r != 10 {
		t.Errorf("radius %v", r)
	}
	if d := a.Style.Dash.Pattern(); len(d) != 2 || d[0] != 5 || d[1] != 5 {
		t.Errorf("dash %v", d)
	}
	if !a.Bold || !a.Italic || a.FontSize != 20 {
		t.Errorf("text %v %v %v", a.Bold, a.Italic, a.FontSize)
	}
}

// Caps and joins are read as the browser takes what Pele passes through: a keyword in
// either case. Pele keeps only the last value of each, so one the browser drops, or inherit,
// leaves them to what is under them.
func TestDeclarationsCapJoin(t *testing.T) {
	for _, c := range []struct {
		decls []string
		cap   Cap
		join  Join
	}{
		{[]string{"stroke-linecap:round", "stroke-linejoin:bevel"}, CapRound, JoinBevel},
		{[]string{"stroke-linecap: SQUARE !important", "stroke-linejoin:Miter"}, CapSquare, JoinMiter},
		{[]string{"stroke-linecap:round", "stroke-linecap:initial", "stroke-linejoin:initial"}, CapButt, JoinMiter},
		{[]string{"stroke-linecap:round", "stroke-linecap:inherit", "stroke-linejoin:round", "stroke-linejoin:unset"}, 0, 0},
		{[]string{"stroke-linecap:round", "stroke-linecap:flat", "stroke-linejoin:round", "stroke-linejoin:miter-clip"}, 0, 0},
		{[]string{"stroke-linecap:ſquare", "stroke-linejoin:arcs"}, 0, 0},
	} {
		s := Declarations(c.decls).Style
		if s.Cap != c.cap || s.Join != c.join {
			t.Errorf("%q: cap %v join %v, want %v %v", c.decls, s.Cap, s.Join, c.cap, c.join)
		}
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

// Font sizes and weights are read as Pele's resolveStyle reads them, with JavaScript's
// /^[\d.]+(px)?$/, parseFloat and Number.
func TestDeclarationsFont(t *testing.T) {
	for size, want := range map[string]float64{
		"12px": 12, "12": 12, ".5": 0.5, "1.2.3": 1.2, "3.": 3, "1e2": 0, "+5": 0, "5pxpx": 0, "50%": 0,
		".": 0, "px": 0, "0": 0, "201": 0, "200px": 200, "12 px": 0,
	} {
		if got := Declarations([]string{"font-size:" + size}).FontSize; got != want {
			t.Errorf("font-size:%s = %v, want %v", size, got, want)
		}
	}
	for weight, want := range map[string]bool{"bold": true, "600": true, " 700 ": true, "0x258": true, "599": false, "6e2": true, "x": false} {
		if got := Declarations([]string{"font-weight:" + weight}).Bold; got != want {
			t.Errorf("font-weight:%s bold %v, want %v", weight, got, want)
		}
	}
}

func TestDeclarationsDropUnsafe(t *testing.T) {
	a := Declarations([]string{"fill:url(#x)", "stroke:  url (x)", "color:red;x", "fill-opacity", ":red", "font-weight:500"})
	if a.Style.Fill.Set() || a.Style.Stroke.Set() || a.Style.Text.Set() || a.Bold {
		t.Errorf("kept %+v", a)
	}
	if b := Declarations([]string{"font-weight:700", "fill:none"}); !b.Bold || b.Style.Fill != None {
		t.Errorf("got %+v", b)
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
