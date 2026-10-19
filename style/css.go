package style

import (
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
}

// Declarations reads Mermaid's style declarations, as "fill:#f9f" and "stroke-width:4px",
// as Pele's resolveStyle does: later ones win, !important is ignored, and a value that could
// reach outside the diagram is dropped. Of the properties, fill, stroke, color, the widths,
// dashes, opacities and rx or ry are kept, and the font's weight, style and size.
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
		case "fill-opacity":
			fill = fraction(value, fill)
		case "stroke-opacity":
			stroke = fraction(value, stroke)
		case "opacity":
			if f, ok := length(value); ok {
				a.Style.Opacity = N(max(0, min(1, f)))
			}
		case "stroke-width":
			if f, ok := length(value); ok && f >= 0 {
				a.Style.StrokeWidth = N(f)
			}
		case "stroke-dasharray":
			a.Style.Dash = dashes(value)
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

// fraction reads an opacity, as 0.5 or 50%, or gives back what was there.
func fraction(value string, was float64) float64 {
	f, _, ok := number(value, 1)
	if !ok {
		return was
	}
	return max(0, min(1, f))
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
