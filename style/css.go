package style

import (
	"slices"
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
)

// Author is what a diagram's author set in its text, with classDef and style lines, in the
// declarations of CSS they are written in.
type Author struct {
	Style Style
	// What changes the size of text, which layout takes.
	Bold, Italic bool
	FontSize     float64 // 0 when it was not set
	// FillOpacity and StrokeOpacity are the fill-opacity and stroke-opacity the author set,
	// from 0 to 1, unset when none was or the last could not be read. Style's fill and stroke
	// already have them; a diagram whose shapes have a fill, a stroke or an opacity of their
	// own, which the author's replace each on its own, reads them here.
	FillOpacity, StrokeOpacity Num
}

// Declarations reads Mermaid's style declarations, as "fill:#f9f" and "stroke-width:4px",
// as Pele's resolveStyle does: later ones win, !important is ignored, and a value that could
// reach outside the diagram is dropped. Of the properties, fill, stroke, color, the widths,
// dashes, caps and joins, opacities and rx or ry are kept, and the font's weight, style and
// size.
func Declarations(decls []string) Author {
	var a Author
	fill, stroke := 1.0, 1.0
	var weight, fontStyle, fontSize string
	for _, decl := range decls {
		prop, value, ok := strings.Cut(decl, ":")
		prop = strings.ToLower(strings.TrimSpace(prop))
		if !ok || prop == "" {
			continue
		}
		value = strings.TrimSpace(replaceFold(value, "!important", ""))
		if value == "" || unsafe(value) {
			continue
		}
		switch prop {
		case "fill":
			a.Style.Fill = paint(value)
		case "stroke":
			a.Style.Stroke = paint(value)
		case "color":
			a.Style.Text = paint(value)
		// Pele keeps the last value of a property alone, so one that cannot be read leaves
		// the property unset, as a browser drops what it cannot take.
		case "fill-opacity":
			a.FillOpacity = opacity(value)
			fill = a.FillOpacity.Or(1)
		case "stroke-opacity":
			a.StrokeOpacity = opacity(value)
			stroke = a.StrokeOpacity.Or(1)
		case "opacity":
			a.Style.Opacity = Num{}
			if f, ok := length(value); ok {
				a.Style.Opacity = N(max(0, min(1, f)))
			}
		case "stroke-width":
			a.Style.StrokeWidth = Num{}
			if f, ok := length(value); ok && f >= 0 {
				a.Style.StrokeWidth = N(f)
			}
		case "stroke-dasharray":
			a.Style.Dash = dashes(value)
		case "stroke-linecap":
			// Pele keeps the last value of a property alone, and one the browser drops
			// leaves the property to what is under it.
			a.Style.Cap = Cap(keyword(value, caps))
		case "stroke-linejoin":
			a.Style.Join = Join(keyword(value, joins))
		case "rx", "ry":
			if f, ok := length(value); ok && f >= 0 {
				a.Style.Radius = N(f)
			}
		case "font-weight":
			weight = value
		case "font-style":
			fontStyle = value
		case "font-size":
			fontSize = value
		}
	}
	if fill < 1 {
		a.Style.Fill = a.Style.Fill.Alpha(fill)
	}
	if stroke < 1 {
		a.Style.Stroke = a.Style.Stroke.Alpha(stroke)
	}
	a.Bold = weight == "bold" || weight == "bolder" || chars.Number(weight) >= 600
	a.Italic = fontStyle == "italic"
	if px, ok := fontPixels(fontSize); ok && px > 0 && px <= 200 {
		a.FontSize = px
	}
	return a
}

// LineDeclared reports whether declarations set anything of a line: a property of a shape
// other than its fill and fill opacity, with a value that is not dropped, whether or not it
// can be read. It is whether the line of Pele's resolveStyle is not empty.
func LineDeclared(decls []string) bool {
	for _, decl := range decls {
		prop, value, ok := strings.Cut(decl, ":")
		if !ok {
			continue
		}
		switch chars.ToLower(chars.Trim(prop)) {
		case "stroke", "stroke-width", "stroke-dasharray", "stroke-dashoffset", "stroke-linecap",
			"stroke-linejoin", "stroke-opacity", "opacity", "rx", "ry":
			value = chars.Trim(replaceFold(value, "!important", ""))
			if value != "" && !unsafe(value) {
				return true
			}
		}
	}
	return false
}

// ShapeCSS is the style Pele's resolveStyle writes on a shape of declarations, as it writes
// it into an attribute: each property of a shape that is kept, with the last value given it,
// in the order the properties first came, "fill:red;stroke:blue;", with &, <, > and " written
// as entities and the control characters XML does not allow dropped; "" for none. A diagram
// that decides by what the attribute holds, as a treemap does whether a class fills a
// section, reads it here.
func ShapeCSS(decls []string) string {
	var props, values []string
	for _, decl := range decls {
		prop, value, ok := strings.Cut(decl, ":")
		if !ok || prop == "" {
			continue
		}
		prop = chars.ToLower(chars.Trim(prop))
		value = chars.Trim(replaceFold(value, "!important", ""))
		if value == "" || unsafe(value) {
			continue
		}
		switch prop {
		case "fill", "fill-opacity", "stroke", "stroke-width", "stroke-dasharray", "stroke-dashoffset",
			"stroke-linecap", "stroke-linejoin", "stroke-opacity", "opacity", "rx", "ry":
		default:
			continue
		}
		if k := slices.Index(props, prop); k >= 0 {
			values[k] = value
		} else {
			props, values = append(props, prop), append(values, value)
		}
	}
	var b strings.Builder
	for k, prop := range props {
		b.WriteString(prop + ":")
		for _, r := range values[k] {
			switch {
			case r == '&':
				b.WriteString("&amp;")
			case r == '"':
				b.WriteString("&quot;")
			case r <= 0x08 || r == 0x0b || r == 0x0c || r >= 0x0e && r <= 0x1f || r == 0x7f:
			default:
				b.WriteRune(r)
			}
		}
		b.WriteByte(';')
	}
	return b.String()
}

// The keywords of stroke-linecap and stroke-linejoin, by the Cap or Join each is. Pele
// passes the properties through to the browser, which takes these, initial for the
// property's initial value, and inherit or unset for what the element would inherit, which
// is what an unset style takes from under it here. It takes no others: SVG 2's miter-clip
// and arcs are not drawn by browsers, and a value they do not take is dropped.
var (
	caps  = map[string]uint8{"butt": uint8(CapButt), "round": uint8(CapRound), "square": uint8(CapSquare), "initial": uint8(CapButt), "inherit": 0, "unset": 0}
	joins = map[string]uint8{"miter": uint8(JoinMiter), "round": uint8(JoinRound), "bevel": uint8(JoinBevel), "initial": uint8(JoinMiter), "inherit": 0, "unset": 0}
)

// keyword reads a keyword of CSS, in either case of its ASCII letters, from those given, or
// 0 for one it is not.
func keyword(value string, of map[string]uint8) uint8 {
	for k, v := range of {
		if len(k) == len(value) && asciiFold(k, value) {
			return v
		}
	}
	return 0
}

// paint is a paint of a color, None for none, or unset for one it cannot read.
func paint(value string) Paint {
	if strings.EqualFold(value, "none") {
		return None
	}
	if c, ok := ParseColor(value); ok {
		if c.A == 0 {
			return None
		}
		return Of(c)
	}
	return Paint{}
}

// opacity reads an opacity, as 0.5 or 50%, kept from 0 to 1, or unset for one it cannot read.
func opacity(value string) Num {
	if f, _, ok := number(value, 1); ok {
		return N(max(0, min(1, f)))
	}
	return Num{}
}

// fontPixels reads a font size as Pele's resolveStyle does: one that /^[\d.]+(px)?$/ takes,
// read with parseFloat, which stops at a second point.
func fontPixels(value string) (float64, bool) {
	body := strings.TrimSuffix(value, "px")
	if body == "" || strings.TrimLeft(body, "0123456789.") != "" {
		return 0, false
	}
	end := strings.IndexByte(body, '.')
	if end >= 0 {
		if next := strings.IndexByte(body[end+1:], '.'); next >= 0 {
			body = body[:end+1+next]
		}
	}
	f, err := strconv.ParseFloat(body, 64)
	return f, err == nil
}

// length reads a number of pixels, with or without px.
func length(value string) (float64, bool) {
	f, err := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(value), "px"), 64)
	return f, err == nil
}

func dashes(value string) Dash {
	if strings.EqualFold(value, "none") {
		return Solid
	}
	var out []float64
	for f := range strings.FieldsFuncSeq(value, func(r rune) bool { return r == ',' || r == ' ' }) {
		n, ok := length(f)
		if !ok || n < 0 {
			return Dash{}
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return Dash{}
	}
	return Dashes(out...)
}

// unsafe matches Pele's RE_UNSAFE: what in a value could load or run something.
func unsafe(value string) bool {
	if strings.ContainsAny(value, "<>{};\\") {
		return true
	}
	v := strings.ToLower(value)
	for _, bad := range []string{"javascript:", "@import", "image-set", "/*"} {
		if strings.Contains(v, bad) {
			return true
		}
	}
	for _, fn := range []string{"url", "expression"} {
		for i := strings.Index(v, fn); i >= 0; {
			if strings.HasPrefix(strings.TrimLeft(v[i+len(fn):], " \t\n\r\f"), "(") {
				return true
			}
			next := strings.Index(v[i+1:], fn)
			if next < 0 {
				break
			}
			i += 1 + next
		}
	}
	return false
}

// replaceFold replaces old, in ASCII letters, wherever s has it in either case.
func replaceFold(s, old, repl string) string {
	var b strings.Builder
	last := 0
	for i := 0; i+len(old) <= len(s); i++ {
		if asciiFold(s[i:i+len(old)], old) {
			b.WriteString(s[last:i])
			b.WriteString(repl)
			i += len(old) - 1
			last = i + 1
		}
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

func asciiFold(a, b string) bool {
	for i := range len(a) {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}
