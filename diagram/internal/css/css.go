// Package css reads what a diagram's author sets in its text with classDef and style lines,
// in the declarations of CSS they are written in, as Pele's resolveStyle does: the style
// they give a thing, and the weight, slant and size of its text, which layout takes.
package css

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/style"
)

// Author is what a diagram's author set in its text, with classDef and style lines, in the
// declarations of CSS they are written in.
type Author struct {
	Style style.Style
	// What changes the size of text, which layout takes.
	Bold, Italic bool
	FontSize     float64 // 0 when it was not set
}

// Declarations reads Mermaid's style declarations, as "fill:#f9f" and "stroke-width:4px",
// as Pele's resolveStyle does: later ones win, !important is ignored, and a value that could
// reach outside the diagram is dropped. Of the properties, fill, stroke, color, the widths,
// dashes, caps and joins, opacities and rx or ry are kept, and the font's weight, style and
// size. A fill-opacity and a stroke-opacity are the style's FillOpacity and StrokeOpacity,
// apart from the paints, as CSS cascades them.
func Declarations(decls []string) Author {
	var a Author
	var weight, fontStyle, fontSize string
	for _, decl := range decls {
		prop, value, ok := strings.Cut(decl, ":")
		prop = chars.ToLower(chars.Trim(prop))
		if !ok || prop == "" {
			continue
		}
		value = chars.Trim(replaceFold(value, "!important", ""))
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
			a.Style.FillOpacity = opacity(value)
		case "stroke-opacity":
			a.Style.StrokeOpacity = opacity(value)
		case "opacity":
			a.Style.Opacity = opacity(value)
		case "stroke-width":
			a.Style.StrokeWidth = style.Num{}
			if f, ok := length(value); ok && f >= 0 {
				a.Style.StrokeWidth = style.N(f)
			}
		case "stroke-dasharray":
			a.Style.Dash = dashes(value)
		case "stroke-linecap":
			// Pele keeps the last value of a property alone, and one the browser drops
			// leaves the property to what is under it.
			a.Style.Cap = style.Cap(keyword(value, caps))
		case "stroke-linejoin":
			a.Style.Join = style.Join(keyword(value, joins))
		case "rx", "ry":
			if f, ok := length(value); ok && f >= 0 {
				a.Style.Radius = style.N(f)
			}
		case "font-weight":
			weight = value
		case "font-style":
			fontStyle = value
		case "font-size":
			fontSize = value
		}
	}
	a.Bold = weight == "bold" || weight == "bolder" || chars.Number(weight) >= 600
	a.Italic = fontStyle == "italic"
	if px, ok := fontPixels(fontSize); ok && px > 0 && px <= 200 {
		a.FontSize = px
	}
	return a
}

// FillDeclared reports whether declarations set a fill with a value that is not dropped,
// whether or not it can be read: whether the shape Pele's resolveStyle writes matches
// /[";]fill:/, as the diagrams that draw a tint only where the author gave no fill test it.
func FillDeclared(decls []string) bool {
	css := ShapeCSS(decls)
	return strings.HasPrefix(css, "fill:") || strings.Contains(css, ";fill:")
}

// FontFamilyDeclared reports whether declarations set a font family with a value that is not
// dropped: whether the text style Pele's resolveStyle writes matches /[";]font-family:/, as
// the diagrams that set a part in the monospace face where the author gave no font of its
// own test it. That style holds each property of text that is kept, color written as fill,
// with the last value given it, in the order the properties first came, and with & and "
// written as entities.
func FontFamilyDeclared(decls []string) bool {
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
		case "color":
			prop = "fill"
		case "font-size", "font-family", "font-weight", "font-style", "text-decoration", "letter-spacing",
			"word-spacing", "opacity":
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
	escape := strings.NewReplacer("&", "&amp;", `"`, "&quot;")
	for k, prop := range props {
		b.WriteString(prop + ":")
		b.WriteString(escape.Replace(values[k]))
		b.WriteByte(';')
	}
	text := b.String()
	return strings.HasPrefix(text, "font-family:") || strings.Contains(text, ";font-family:")
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
// in the order the properties first came, "fill:red;stroke:blue;", with & and " written as
// entities; "" for none. (A value with < or >, or with a character XML does not allow, is
// dropped before, as unsafe.) A diagram that decides by what the attribute holds,
// as a treemap does whether a class fills a section, reads it here.
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
	caps  = map[string]uint8{"butt": uint8(style.CapButt), "round": uint8(style.CapRound), "square": uint8(style.CapSquare), "initial": uint8(style.CapButt), "inherit": 0, "unset": 0}
	joins = map[string]uint8{"miter": uint8(style.JoinMiter), "round": uint8(style.JoinRound), "bevel": uint8(style.JoinBevel), "initial": uint8(style.JoinMiter), "inherit": 0, "unset": 0}
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
func paint(value string) style.Paint {
	if chars.EqualFoldASCII(value, "none") {
		return style.None
	}
	if c, ok := style.ParseColor(value); ok {
		if c.A == 0 {
			return style.None
		}
		return style.Of(c)
	}
	return style.Paint{}
}

// opacity reads an opacity, as 0.5 or 50%, kept from 0 to 1, or unset for one it cannot read.
func opacity(value string) style.Num {
	pct := strings.HasSuffix(value, "%")
	f, err := strconv.ParseFloat(strings.TrimSuffix(value, "%"), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return style.Num{}
	}
	if pct {
		f /= 100
	}
	return style.N(max(0, min(1, f)))
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
	f, err := strconv.ParseFloat(strings.TrimSuffix(cssTrim(value), "px"), 64)
	return f, err == nil
}

// cssTrim is s without the white space of CSS about it: spaces, tabs, line feeds, carriage
// returns and form feeds. A browser reads a value Pele writes with these and no others;
// what JavaScript's trim takes off besides, as a no-break space, was taken off before.
func cssTrim(s string) string { return strings.Trim(s, " \t\n\r\f") }

// lowerASCII is s with its ASCII letters in lower case and its other letters as they are, as
// CSS compares keywords, and as a regular expression's i flag without u compares the ASCII
// letters of a pattern.
func lowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

func dashes(value string) style.Dash {
	if chars.EqualFoldASCII(value, "none") {
		return style.Solid
	}
	var out []float64
	for f := range strings.FieldsFuncSeq(value, func(r rune) bool { return r == ',' || strings.ContainsRune(" \t\n\r\f", r) }) {
		n, ok := length(f)
		if !ok || n < 0 {
			return style.Dash{}
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return style.Dash{}
	}
	return style.Dashes(out...)
}

// unsafe matches Pele's RE_UNSAFE, /url\s*\(|expression\s*\(|javascript:|[<>{};\\]|@import|image-set|\/\*/i:
// what in a value could load or run something. The i flag, without u, folds the case of
// ASCII letters only, and \s is JavaScript's white space. It also matches RE_INVALID_XML,
// which Pele's resolveStyle drops a value for too, since leaving out the characters XML does
// not allow could make what is left unsafe: a control character but the tab, line feed and
// carriage return, DEL, U+FFFE or U+FFFF. (A Go string holds no lone surrogate.)
func unsafe(value string) bool {
	if strings.ContainsAny(value, "<>{};\\") {
		return true
	}
	for _, r := range value {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return true
		}
	}
	v := lowerASCII(value)
	for _, bad := range []string{"javascript:", "@import", "image-set", "/*"} {
		if strings.Contains(v, bad) {
			return true
		}
	}
	for _, fn := range []string{"url", "expression"} {
		for i := strings.Index(v, fn); i >= 0; {
			if strings.HasPrefix(chars.TrimStart(v[i+len(fn):]), "(") {
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

// Paints reports whether declarations paint a shape's fill, as Pele's paints reads the style
// its resolveStyle writes: a fill that is neither none nor transparent, at an opacity above 0.
// A color or an opacity it cannot read counts as opaque.
func Paints(decls []string) bool {
	fill, ok := shapeValue(decls, "fill")
	if !ok {
		return false
	}
	fill = lowerASCII(chars.Trim(fill))
	if fill == "" || fill == "none" || fill == "transparent" {
		return false
	}
	alpha := 1.0
	if c, ok := style.ParseColor(fill); ok {
		alpha = float64(c.A) / 255
	}
	scale := opacities(decls)
	if math.IsNaN(scale) {
		scale = 1
	}
	return alpha*scale > 0
}

// Inks reports whether declarations give a shape a fill that Pele's ink picks a text color on:
// a color it reads, drawn at no less than half its opacity, with opacities it reads. The color
// it picks is the style's to settle (style.Style.OnFill and Ground).
func Inks(decls []string) bool {
	fill, ok := shapeValue(decls, "fill")
	if !ok {
		return false
	}
	c, ok := style.ParseColor(fill)
	if !ok || lowerASCII(chars.Trim(fill)) == "transparent" {
		return false
	}
	// An opacity that is not a plain number may still be low, so none is picked.
	return float64(c.A)/255*opacities(decls) >= 0.5
}

// Sets reports whether declarations give a shape's property a value resolveStyle keeps, read
// or not, as Pele tests the style it writes for one with /[";]prop:/.
func Sets(decls []string, prop string) bool {
	_, ok := shapeValue(decls, prop)
	return ok
}

// shapeValue is the value resolveStyle keeps for a property of a shape, the last one given.
func shapeValue(decls []string, prop string) (string, bool) {
	var value string
	found := false
	for _, decl := range decls {
		p, v, ok := strings.Cut(decl, ":")
		if !ok || p == "" || chars.ToLower(chars.Trim(p)) != prop {
			continue
		}
		v = chars.Trim(replaceFold(v, "!important", ""))
		if v == "" || unsafe(v) {
			continue
		}
		value, found = v, true
	}
	return value, found
}

// opacities is the product of the fill opacity and the opacity resolveStyle keeps, as Pele's
// opacity reads them, NaN where one is not a plain number.
func opacities(decls []string) float64 {
	product := 1.0
	for _, prop := range []string{"fill-opacity", "opacity"} {
		if v, ok := shapeValue(decls, prop); ok {
			product *= amount(chars.Trim(v))
		}
	}
	return product
}

// amount reads an opacity as Pele's amount does, a number or a percentage that
// /^[+-]?(?:\d+(?:\.\d+)?|\.\d+)(?:e[+-]?\d+)?(%|deg|grad|rad|turn)?$/i takes, kept from 0 to
// 1, and NaN for anything else, an angle among them.
func amount(raw string) float64 {
	s := raw
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}
	digits := func(i int) int {
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		return i
	}
	i := digits(0)
	switch {
	case i > 0:
		if i < len(s) && s[i] == '.' {
			if j := digits(i + 1); j > i+1 {
				i = j
			}
		}
	case i < len(s) && s[i] == '.':
		if i = digits(1); i == 1 {
			return math.NaN()
		}
	default:
		return math.NaN()
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		if k := digits(j); k > j {
			i = k
		}
	}
	number := raw[:len(raw)-len(s)+i]
	unit := lowerASCII(s[i:])
	if unit != "" && unit != "%" {
		return math.NaN()
	}
	// A number too large for a float64 is infinite, as parseFloat reads it.
	f, _ := strconv.ParseFloat(number, 64)
	if unit == "%" {
		f /= 100
	}
	return max(0, min(1, f))
}
