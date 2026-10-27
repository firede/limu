package css

import (
	"slices"
	"testing"

	"github.com/firede/limu/style"
)

func TestFontFamilyDeclared(t *testing.T) {
	for _, c := range []struct {
		decls []string
		want  bool
	}{
		{[]string{"font-family:Courier"}, true},
		{[]string{"color:red", "Font-Family: Arial !important"}, true},
		{[]string{"font-family: "}, false},
		{[]string{"font-family:url(x)"}, false},
		{[]string{"font-family:a\x01"}, false},
		{[]string{"fill:#eee", "stroke:#333", "font-size:12px"}, false},
		// An entity Pele writes for a quote or an ampersand ends in the semicolon it tests for.
		{[]string{`color:"x"font-family:y`}, true},
		{[]string{"color:a&font-family:b"}, true},
		{[]string{"color:font-family:b"}, false},
	} {
		if got := FontFamilyDeclared(c.decls); got != c.want {
			t.Errorf("%q: %v, want %v", c.decls, got, c.want)
		}
	}
}

func TestDeclarations(t *testing.T) {
	a := Declarations([]string{
		"fill:#f9f", "stroke:#333", "stroke-width:4px", "color: red !IMPORTANT",
		"stroke-dasharray: 5 5", "font-weight:bold", "font-style:italic", "font-size:20px",
		"fill-opacity:0.5", "rx:10",
	})
	th := style.Default()
	if c := a.Style.Fill.Color(th); c != (style.Color{R: 255, G: 153, B: 255, A: 255}) || a.Style.FillOpacity != style.N(0.5) {
		t.Errorf("fill %v at %v", c, a.Style.FillOpacity)
	}
	if c := a.Style.Stroke.Color(th); c != style.Hex("#333") {
		t.Errorf("stroke %v", c)
	}
	if c := a.Style.Text.Color(th); c != style.Hex("#f00") {
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
		cap   style.Cap
		join  style.Join
	}{
		{[]string{"stroke-linecap:round", "stroke-linejoin:bevel"}, style.CapRound, style.JoinBevel},
		{[]string{"stroke-linecap: SQUARE !important", "stroke-linejoin:Miter"}, style.CapSquare, style.JoinMiter},
		{[]string{"stroke-linecap:round", "stroke-linecap:initial", "stroke-linejoin:initial"}, style.CapButt, style.JoinMiter},
		{[]string{"stroke-linecap:round", "stroke-linecap:inherit", "stroke-linejoin:round", "stroke-linejoin:unset"}, 0, 0},
		{[]string{"stroke-linecap:round", "stroke-linecap:flat", "stroke-linejoin:round", "stroke-linejoin:miter-clip"}, 0, 0},
		{[]string{"stroke-linecap:\u017fquare", "stroke-linejoin:arcs"}, 0, 0},
	} {
		s := Declarations(c.decls).Style
		if s.Cap != c.cap || s.Join != c.join {
			t.Errorf("%q: cap %v join %v, want %v %v", c.decls, s.Cap, s.Join, c.cap, c.join)
		}
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
	if b := Declarations([]string{"font-weight:700", "fill:none"}); !b.Bold || b.Style.Fill != style.None {
		t.Errorf("got %+v", b)
	}
}

// Declarations are trimmed and lowered as Pele's resolveStyle does, with JavaScript's trim and
// toLowerCase, and their values then read as a browser reads CSS, with its white space only
// and its keywords in either case of their ASCII letters.
func TestDeclarationsReadAsJavaScript(t *testing.T) {
	red := style.Of(style.Color{R: 255, G: 0, B: 0, A: 255})
	for decl, want := range map[string]style.Paint{
		"fill:\ufeffred":       red, // trim takes off a byte order mark
		"\u00a0FILL\u3000:red": red,
		"fill:red\u0085":       {}, // which trim leaves, and a browser does not read
		"fill:RED":             red,
		"fill:blac\u212a":      {}, // a Kelvin sign is no k to CSS
		"fill:rgb (255,0,0)":   {},
		"fill:rgb(255\n0\r0)":  red,
		"fill:rgb(255\n0\f0)":  {}, // a form feed is no character XML allows
		"fill:NONE":            style.None,
	} {
		if got := Declarations([]string{decl}).Style.Fill; got != want {
			t.Errorf("%q: fill %+v, want %+v", decl, got, want)
		}
	}
	if s := Declarations([]string{"stroke-dasharray:4\t2"}).Style.Dash; !slices.Equal(s.Pattern(), []float64{4, 2}) {
		t.Errorf("dashes %v", s.Pattern())
	}
	// \s in Pele's RE_UNSAFE is JavaScript's; its i flag folds ASCII letters only.
	if Declarations([]string{"fill:url\u3000(x)"}).Style.Fill.Set() {
		t.Errorf("kept a url( after an ideographic space")
	}
	if ShapeCSS([]string{"rx:javascr\u0130pt:x"}) == "" || ShapeCSS([]string{"rx:JavaScript:x"}) != "" {
		t.Errorf("folded other than ASCII letters")
	}
	for value, want := range map[string]float64{"0.5": 0.5, "50%": 0.5, "150%": 1, "-1": 0} {
		if got, ok := Declarations([]string{"opacity:" + value}).Style.Opacity.Get(); !ok || got != want {
			t.Errorf("opacity:%s = %v %v, want %v", value, got, ok, want)
		}
	}
	if _, ok := Declarations([]string{"opacity:0.5px"}).Style.Opacity.Get(); ok {
		t.Errorf("read an opacity in pixels")
	}
}

// A later value of a property that cannot be read leaves it unset, as Pele writes only the
// last and a browser drops it.
func TestDeclarationsLastWins(t *testing.T) {
	a := Declarations([]string{"stroke-width:3px", "stroke-width:0x", "opacity:0.5", "opacity:half", "fill-opacity:0.5", "fill:red", "fill-opacity:x"})
	_, width := a.Style.StrokeWidth.Get()
	_, opacity := a.Style.Opacity.Get()
	if width || opacity || a.Style.Fill != style.Of(style.Color{R: 255, G: 0, B: 0, A: 255}) {
		t.Errorf("kept %+v", a.Style)
	}
}

func TestDeclarationsOpacities(t *testing.T) {
	a := Declarations([]string{"fill-opacity:0.5", "stroke-opacity:150%"})
	if f, ok := a.Style.FillOpacity.Get(); !ok || f != 0.5 {
		t.Errorf("fill-opacity %v %v", f, ok)
	}
	if f, ok := a.Style.StrokeOpacity.Get(); !ok || f != 1 {
		t.Errorf("stroke-opacity %v %v", f, ok)
	}
	a = Declarations([]string{"fill-opacity:0.5", "fill-opacity:x", "stroke-opacity:20%"})
	if _, ok := a.Style.FillOpacity.Get(); ok {
		t.Errorf("kept a fill-opacity the last declaration could not give")
	}
	if f, _ := a.Style.StrokeOpacity.Get(); f != 0.2 {
		t.Errorf("stroke-opacity %v", f)
	}
	// An opacity alone is kept, with no paint for it to fold into.
	if a := Declarations([]string{"fill-opacity:0.5"}); a.Style.Fill.Set() || a.Style.FillOpacity != style.N(0.5) {
		t.Errorf("fill-opacity alone %+v", a.Style)
	}
}

func TestShapeCSS(t *testing.T) {
	for _, c := range []struct {
		decls []string
		want  string
	}{
		{nil, ""},
		{[]string{"color:red", "font-size:20px"}, ""},
		{[]string{" FILL :red !important", "stroke:blue", "fill:green", "x:y", ":z", "rx"}, "fill:green;stroke:blue;"},
		{[]string{`stroke:a"b&c`, "fill:url(x)", `stroke-width:\1`}, "stroke:a&quot;b&amp;c;"},
		// A value with a character XML does not allow is dropped.
		{[]string{"opacity:0.5", "stroke-linecap:round", "stroke-dasharray:1\x7f2", "rx:\x01", "ry:\ufffe"}, "opacity:0.5;stroke-linecap:round;"},
	} {
		if got := ShapeCSS(c.decls); got != c.want {
			t.Errorf("ShapeCSS(%q) = %q, want %q", c.decls, got, c.want)
		}
	}
}

func TestInksAndPaints(t *testing.T) {
	for _, c := range []struct {
		decls        []string
		inks, paints bool
	}{
		{nil, false, false},
		{[]string{"fill:#000080"}, true, true},
		{[]string{"fill:none"}, false, false},
		{[]string{"fill:transparent"}, false, false},
		{[]string{"fill:var(--x)"}, false, true},
		{[]string{"fill:#000", "fill-opacity:0.4"}, false, true},
		{[]string{"fill:#000", "fill-opacity:abc"}, false, true},
		{[]string{"fill:#000", "fill-opacity:50%"}, true, true},
		{[]string{"fill:#000", "opacity:0.7", "fill-opacity:0.7"}, false, true},
		{[]string{"fill:#000", "opacity:0"}, false, false},
		{[]string{"fill:#000", "opacity:1e999"}, true, true},
		{[]string{"fill:#000", "opacity:1deg"}, false, true},
		{[]string{"fill:rgba(0,0,0,0.4)"}, false, true},
		// Colors as Pele's color reads them: an empty argument, a mix of numbers and
		// percentages, or none in the comma syntax is no color it reads, and an alpha is
		// taken as it is written, not as a byte.
		{[]string{"fill:rgb(10,20,,30)"}, false, true},
		{[]string{"fill:rgb(10%,20,30)"}, false, true},
		{[]string{"fill:rgb(none,0,0)"}, false, true},
		{[]string{"fill:rgb(none 0 0)"}, true, true},
		{[]string{"fill:rgb(0 0 0 / 0.499)"}, false, true},
		{[]string{"fill:rgb(0 0 0 / 0.5 / 1)"}, false, true},
		{[]string{"fill:hsl(200,80,30)"}, false, true},
		{[]string{"fill:hsl(200deg 80% 30%)"}, true, true},
		{[]string{"fill:hsl(1turn,80%,30%)"}, true, true},
		{[]string{"fill:hsl(1e999,80%,30%)"}, false, true},
		{[]string{"fill:#0000"}, false, false},
		{[]string{"fill:#00000080"}, true, true},
		{[]string{"fill:DarkSlateGrey"}, true, true},
		{[]string{"fill:red#"}, false, true},
	} {
		if got := Inks(c.decls); got != c.inks {
			t.Errorf("Inks(%q) = %v", c.decls, got)
		}
		if got := Paints(c.decls); got != c.paints {
			t.Errorf("Paints(%q) = %v", c.decls, got)
		}
	}
}

func TestInksText(t *testing.T) {
	navy := []string{"fill:#000080"}
	for _, c := range []struct {
		shape, text []string
		want        bool
	}{
		{navy, nil, true},
		{navy, []string{"color:#fff"}, false},
		// A color Pele can't read still keeps ink from picking one.
		{navy, []string{"color:#f"}, false},
		// One it drops does not.
		{navy, []string{"color:url(x)"}, true},
		{[]string{"fill:none"}, nil, false},
	} {
		if got := InksText(c.shape, c.text); got != c.want {
			t.Errorf("InksText(%q, %q) = %v", c.shape, c.text, got)
		}
	}
}

// Widths and opacities are numbers as CSS writes them, which Go's ParseFloat reads more of.
func TestCSSNumbers(t *testing.T) {
	for value, want := range map[string]float64{
		"1e1": 10, "+3": 3, "3PX": 3, " 2px ": 2,
		"inf": -1, "NaN": -1, "0x10": -1, "1.": -1, "1e400": -1, "2em": -1,
	} {
		got := Declarations([]string{"stroke-width:" + value}).Style.StrokeWidth.Or(-1)
		if got != want {
			t.Errorf("stroke-width:%s gives %v, not %v", value, got, want)
		}
	}
	for value, want := range map[string]float64{"0.5": 0.5, "50%": 0.5, "2": 1, "0.": -1, "inf": -1, "5deg": -1} {
		if got := Declarations([]string{"opacity:" + value}).Style.Opacity.Or(-1); got != want {
			t.Errorf("opacity:%s gives %v, not %v", value, got, want)
		}
	}
}
