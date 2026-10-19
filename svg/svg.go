// Package svg draws a scene as SVG.
//
// Colors of the theme are written as CSS variables, --limu-surface and the rest, with the
// theme's colors as their fallbacks, so a page restyles a diagram without drawing it again;
// with Options.Literal they are written as colors.
package svg

import (
	"strings"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// Options say how a scene is written.
type Options struct {
	Look style.Look
	// Literal writes the theme's colors as colors, not as variables.
	Literal bool
	// The fonts the scene was measured in; the page's --limu-font and --limu-font-mono come
	// first when it sets them.
	FontFamily, MonoFamily string
	// LinkSchemes are the schemes a link may use besides a relative address; nil is http,
	// https, mailto and tel. A link of another is drawn as no link.
	LinkSchemes []string
	// ImageSchemes are the schemes an image may be loaded with besides a relative address;
	// nil is http and https.
	ImageSchemes []string
	// IconMarkup gives the markup inside an icon's <svg>, on a box of 24 by 24, by the
	// icon's name: of an item's picture, or of an icon in a label. The icon's <svg> sets
	// its color to that of the item's text, for the markup to draw in as currentColor.
	// Labels leave room only for icons the measurer was told of; without IconMarkup, icons
	// are left out.
	IconMarkup func(name string) string
	// Draw, when set, writes each item in place of the default, which next writes; the
	// Writer's Item, Parts, Ends, Picture and Label write the default's pieces. It is given
	// the item's style cascaded from the look, its paints not settled into colors, so that
	// Paint writes the theme's as CSS variables.
	Draw func(w *Writer, it *scene.Item, s style.Style, next func())
}

// Render writes a scene as an SVG document.
func Render(sc *scene.Scene, o *Options) string {
	if o == nil {
		o = &Options{}
	}
	w := &Writer{o: o, th: o.Look.ThemeOrDefault(), sc: sc}
	b := &w.b
	b.Grow(256 + 160*len(sc.Items))
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" class="limu limu-`)
	w.Text(sc.Type)
	b.WriteString(`" viewBox="0 0 `)
	w.Num(sc.Width)
	b.WriteByte(' ')
	w.Num(sc.Height)
	b.WriteString(`" width="`)
	w.Num(sc.Width)
	b.WriteString(`" height="`)
	w.Num(sc.Height)
	b.WriteString(`" font-size="`)
	w.Num(sc.FontSize)
	b.WriteString(`" style="font-family:`)
	w.Text(w.family(false))
	b.WriteString(`" fill="`)
	w.Paint(style.Fg)
	b.WriteString(`" role="graphics-document document" aria-roledescription="`)
	w.Text(sc.Type)
	b.WriteString(`">`)
	if sc.Title != "" {
		b.WriteString("<title>")
		w.Text(sc.Title)
		b.WriteString("</title>")
	}
	if sc.Desc != "" {
		b.WriteString("<desc>")
		w.Text(sc.Desc)
		b.WriteString("</desc>")
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		st := o.Look.Cascade(string(it.Role), it.Style, it.Author)
		if o.Draw != nil {
			o.Draw(w, it, st, func() { w.Item(it, st) })
		} else {
			w.Item(it, st)
		}
	}
	b.WriteString("</svg>")
	return b.String()
}

// A Writer is the document being written, for an Options.Draw of one's own.
type Writer struct {
	b     strings.Builder
	o     *Options
	th    *style.Theme
	sc    *scene.Scene
	icons []icon
}

// Raw writes markup as it is.
func (w *Writer) Raw(s string) { w.b.WriteString(s) }

// Text writes text, escaped for an attribute or an element.
func (w *Writer) Text(s string) { esc(&w.b, s) }

// Num writes a number to two decimals, with no more digits than it needs.
func (w *Writer) Num(f float64) { num(&w.b, f) }

// Paint writes a paint as an opaque color: a variable of the theme's, or a color, or none.
// The rest of its opacity is Opacity, which PaintAttr writes with it.
func (w *Writer) Paint(p style.Paint) {
	if !p.Visible() {
		w.b.WriteString("none")
		return
	}
	name := p.Token().Name()
	if name == "" || w.o.Literal {
		// A color of its own, or one mixed with the theme's, is settled here, as are all
		// of them when they are written as colors.
		var c style.Color
		switch {
		case p.Base() != 0:
			c = p.Color(w.th)
		case name == "":
			c = p.Own()
		default:
			c = w.th.Color(p.Token())
		}
		c.A = 255
		w.b.WriteString(c.String())
		return
	}
	if p.Base() != 0 {
		// The paint's opacity is its share of the mix.
		w.b.WriteString("color-mix(in srgb,")
		w.variable(p.Token())
		w.b.WriteByte(' ')
		w.Num(100 * float64(p.Factor()) * float64(w.th.Color(p.Token()).A) / 255)
		w.b.WriteString("%,")
		w.variable(p.Base())
		w.b.WriteByte(')')
		return
	}
	w.variable(p.Token())
}

// Opacity is the part of a paint's opacity that Paint does not write: its factor
// (style.Paint.Alpha) times the alpha of its color, its own or the theme's. A paint laid over
// a color of the theme (style.Paint.Over) is opaque, as Paint writes the mix; one that draws
// nothing is 0.
func (w *Writer) Opacity(p style.Paint) float64 {
	switch {
	case !p.Visible():
		return 0
	case p.Base() != 0:
		return 1
	}
	c := p.Own()
	if p.Token().Name() != "" {
		c = w.th.Color(p.Token())
	}
	return float64(p.Factor()) * float64(c.A) / 255
}

// variable writes a token of the theme as a CSS variable, its color in the theme the
// fallback.
func (w *Writer) variable(t style.Token) {
	w.b.WriteString("var(--limu-")
	w.b.WriteString(t.Name())
	w.b.WriteByte(',')
	c := w.th.Color(t)
	c.A = 255
	w.b.WriteString(c.String())
	w.b.WriteByte(')')
}

// PaintAttr writes an attribute of paint, name="…" as Paint writes it, and name-opacity
// with its Opacity when that is under 1: for "fill", fill="…" fill-opacity="…".
func (w *Writer) PaintAttr(name string, p style.Paint) {
	w.b.WriteByte(' ')
	w.b.WriteString(name)
	w.b.WriteString(`="`)
	w.Paint(p)
	w.b.WriteByte('"')
	if a := w.Opacity(p); p.Visible() && a < 1 {
		w.b.WriteByte(' ')
		w.b.WriteString(name)
		w.b.WriteString(`-opacity="`)
		w.Num(a)
		w.b.WriteByte('"')
	}
}

// Path writes the path's steps, as the d of a <path>.
func (w *Writer) Path(p *scene.Path) {
	p.Each(func(op scene.Op, a []float64) {
		w.b.WriteByte("MLQCZ"[op])
		for i, f := range a {
			if i > 0 {
				w.b.WriteByte(',')
			}
			w.Num(f)
		}
	})
}

func (w *Writer) family(mono bool) string {
	if mono {
		return "var(--limu-font-mono," + or(w.o.MonoFamily, "monospace") + ")"
	}
	return "var(--limu-font," + or(w.o.FontFamily, "sans-serif") + ")"
}

func or(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// Item writes an item as Pele does: a group of its outline, its markers, its picture and
// its label, with its tooltip and its link, at the style's opacity.
func (w *Writer) Item(it *scene.Item, st style.Style) {
	b := &w.b
	b.WriteString(`<g class="limu-`)
	b.WriteString(kindClass[it.Kind])
	if it.Role != "" {
		b.WriteString(" limu-")
		w.Text(strings.ReplaceAll(string(it.Role), ".", "-"))
	}
	if it.Shape != "" {
		b.WriteString(" limu-shape-")
		w.Text(string(it.Shape))
	}
	for _, c := range it.Class {
		b.WriteByte(' ')
		w.Text(c)
	}
	b.WriteByte('"')
	if it.ID != "" {
		b.WriteString(` data-id="`)
		w.Text(it.ID)
		b.WriteByte('"')
	}
	if o, ok := st.Opacity.Get(); ok && o < 1 {
		b.WriteString(` opacity="`)
		w.Num(o)
		b.WriteByte('"')
	}
	b.WriteByte('>')
	if it.Tooltip != "" {
		b.WriteString("<title>")
		w.Text(it.Tooltip)
		b.WriteString("</title>")
	}
	link := w.safeLink(it.Link)
	if link != "" {
		b.WriteString(`<a href="`)
		w.Text(link)
		b.WriteString(`">`)
	}
	w.Parts(it, st)
	w.Ends(it, st)
	w.Picture(it, st)
	w.Label(it, st)
	if link != "" {
		b.WriteString("</a>")
	}
	b.WriteString("</g>")
}

// Parts writes the parts of an item's outline: paths, and boxes as <rect>s, the boxes'
// corners rounded by the style's radius when they are Rounded, each filled and stroked as
// its Paint says. SVG takes a radius too large for a box as half its width across and half
// its height down, so that a narrow box's corners are elliptical; the mygoui package takes
// the smaller of the two both ways.
func (w *Writer) Parts(it *scene.Item, st style.Style) {
	b := &w.b
	sw := st.StrokeWidth.Or(1)
	for _, part := range it.Parts {
		if part.Path == nil {
			b.WriteString(`<rect x="`)
			w.Num(part.Box.X)
			b.WriteString(`" y="`)
			w.Num(part.Box.Y)
			b.WriteString(`" width="`)
			w.Num(part.Box.W)
			b.WriteString(`" height="`)
			w.Num(part.Box.H)
			b.WriteByte('"')
			if r := st.Radius.Or(0); part.Rounded && r > 0 {
				b.WriteString(` rx="`)
				w.Num(r)
				b.WriteByte('"')
			}
		} else {
			b.WriteString(`<path d="`)
			w.Path(part.Path)
			b.WriteByte('"')
		}
		fill, stroke := st.Fill, st.Stroke
		switch part.Paint {
		case scene.StrokeOnly:
			fill = style.None
		case scene.FillOnly:
			stroke = style.None
		case scene.Solid:
			fill = style.Line
		case scene.Hollow:
			fill = style.Bg
		case scene.Shade:
			fill, stroke = shade(it), style.None
		}
		w.PaintAttr("fill", fill)
		w.PaintAttr("stroke", stroke)
		if stroke.Visible() {
			w.strokeAttrs(st, sw)
		}
		b.WriteString("/>")
	}
}

// shade is the fill of an item's parts painted Shade: the theme's background, or, when the
// item's author set a fill, the background at 60% over it, as Pele draws an entity's
// stripes at an opacity of 0.6 when its style sets a fill. The parts below show through,
// in SVG and in the mygoui package alike.
func shade(it *scene.Item) style.Paint {
	if it.Author.Fill.Set() {
		return style.Bg.Alpha(0.6)
	}
	return style.Bg
}

// Ends writes the markers at the ends of an edge, in the color of its stroke, each part as
// scene.Marker.Outline says, solid, with the edge's caps and with its corners turned as the
// marker's type says or else as the edge's are, as Pele draws them.
func (w *Writer) Ends(it *scene.Item, st style.Style) {
	b := &w.b
	sw := st.StrokeWidth.Or(1)
	solid := st
	solid.Dash = style.Dash{}
	for _, m := range it.Ends {
		for _, part := range m.Outline() {
			b.WriteString(`<path class="limu-marker" d="`)
			if part.Path != nil {
				w.Path(part.Path)
			} else {
				w.Path(new(scene.Path).RoundRect(part.Box, 0))
			}
			b.WriteByte('"')
			fill, stroke := st.Stroke, st.Stroke
			switch part.Paint {
			case scene.FillOnly, scene.Solid:
				stroke = style.None
			case scene.StrokeOnly:
				fill = style.None
			case scene.Hollow:
				fill = style.Bg
			}
			w.PaintAttr("fill", fill)
			w.PaintAttr("stroke", stroke)
			if stroke.Visible() {
				ms := solid
				if j := m.Type.Join(); j.Set() {
					ms.Join = j
				}
				w.strokeAttrs(ms, sw)
			}
			b.WriteString("/>")
		}
	}
}

func (w *Writer) strokeAttrs(st style.Style, sw float64) {
	b := &w.b
	if sw != 1 {
		b.WriteString(` stroke-width="`)
		w.Num(sw)
		b.WriteByte('"')
	}
	if d := st.Dash.Pattern(); len(d) > 0 {
		b.WriteString(` stroke-dasharray="`)
		for i, f := range d {
			if i > 0 {
				b.WriteByte(' ')
			}
			w.Num(f)
		}
		b.WriteByte('"')
	}
	if st.Cap.Set() {
		b.WriteString(` stroke-linecap="`)
		b.WriteString(st.Cap.String())
		b.WriteByte('"')
	}
	if st.Join.Set() {
		b.WriteString(` stroke-linejoin="`)
		b.WriteString(st.Join.String())
		b.WriteByte('"')
	}
}

var kindClass = [...]string{"item", "node", "group", "edge", "text", "mark"}

// Picture writes an item's picture: an image, if the options allow its address, or an icon
// through Options.IconMarkup, in the color of the item's text.
func (w *Writer) Picture(it *scene.Item, st style.Style) {
	pic := it.Picture
	if pic == nil {
		return
	}
	b := &w.b
	switch {
	case pic.Image != "":
		href := safeURL(pic.Image, w.o.ImageSchemes, imageSchemes)
		if href == "" {
			return
		}
		b.WriteString(`<image href="`)
		w.Text(href)
		b.WriteByte('"')
		w.box(pic.Box)
		if pic.Fit {
			b.WriteString(` preserveAspectRatio="xMidYMid meet"/>`)
		} else {
			b.WriteString(` preserveAspectRatio="none"/>`)
		}
	case pic.Icon != "" && w.o.IconMarkup != nil:
		w.icon(pic.Icon, pic.Box, st.Text)
	}
}

// box writes the attributes x, y, width and height of a box.
func (w *Writer) box(r scene.Rect) {
	b := &w.b
	b.WriteString(` x="`)
	w.Num(r.X)
	b.WriteString(`" y="`)
	w.Num(r.Y)
	b.WriteString(`" width="`)
	w.Num(r.W)
	b.WriteString(`" height="`)
	w.Num(r.H)
	b.WriteByte('"')
}

// icon writes an icon by its name in a box, its color c for its markup's currentColor, at
// c's opacity.
func (w *Writer) icon(name string, r scene.Rect, c style.Paint) {
	b := &w.b
	b.WriteString(`<svg class="limu-icon" data-icon="`)
	w.Text(name)
	b.WriteByte('"')
	w.box(r)
	b.WriteString(` color="`)
	w.Paint(c)
	b.WriteByte('"')
	if a := w.Opacity(c); a < 1 {
		b.WriteString(` opacity="`)
		w.Num(a)
		b.WriteByte('"')
	}
	b.WriteString(` viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`)
	b.WriteString(w.o.IconMarkup(name))
	b.WriteString("</svg>")
}

// Label writes an item's label about its At, as Pele's labelSvg does: a line's baseline is
// a little below the middle of its line's height. A halo is the text again under it,
// stroked. A label turned by its Rotate is written in a group turned about its At, with its
// icons.
func (w *Writer) Label(it *scene.Item, st style.Style) {
	if it.Label == nil || it.Label.Empty() {
		return
	}
	b := &w.b
	if it.Rotate != 0 {
		b.WriteString(`<g transform="rotate(`)
		w.Num(it.Rotate)
		b.WriteByte(' ')
		w.Num(it.At.X)
		b.WriteByte(' ')
		w.Num(it.At.Y)
		b.WriteString(`)">`)
	}
	if st.Halo.Visible() {
		w.text(it, st, true)
	}
	w.text(it, st, false)
	if it.Rotate != 0 {
		b.WriteString("</g>")
	}
}

// text writes a label's <text>: its halo, or the text itself, followed by its icons.
func (w *Writer) text(it *scene.Item, st style.Style, halo bool) {
	b := &w.b
	l := it.Label
	cx, cy := it.At.X, it.At.Y
	shift := l.Size * 0.35
	y := cy - l.Height/2 + l.LineHeight/2 + shift
	b.WriteString(`<text text-anchor="middle" xml:space="preserve"`)
	switch {
	case halo:
		b.WriteString(` fill="none"`)
		w.PaintAttr("stroke", st.Halo)
		b.WriteString(` stroke-width="`)
		w.Num(st.HaloWidth.Or(0))
		b.WriteString(`" stroke-linejoin="round"`)
	case st.Text != style.Fg:
		w.PaintAttr("fill", st.Text)
	}
	if l.Size != w.sc.FontSize {
		b.WriteString(` font-size="`)
		w.Num(l.Size)
		b.WriteByte('"')
	}
	b.WriteByte('>')
	for i, line := range l.Lines {
		var x float64
		switch it.Align {
		case scene.Start:
			x = cx
		case scene.End:
			x = cx - l.Widths[i]
		default:
			x = cx - l.Widths[i]/2
		}
		if len(line) == 0 {
			b.WriteString(`<tspan x="`)
			w.Num(x)
			b.WriteString(`" y="`)
			w.Num(y)
			b.WriteString(`"> </tspan>`)
		}
		for _, s := range line {
			if !halo && s.Icon != "" && w.o.IconMarkup != nil {
				side := l.Size * 1.1
				w.icons = append(w.icons, icon{s.Icon, scene.Rect{X: x, Y: y - shift - side/2, W: side, H: side}})
			}
			if s.Icon == "" && s.Text != "" {
				b.WriteString(`<tspan x="`)
				w.Num(x + s.Width/2)
				b.WriteString(`" y="`)
				w.Num(y)
				b.WriteByte('"')
				if s.Face&text.Bold != 0 {
					b.WriteString(` font-weight="bold"`)
				}
				if s.Face&text.Italic != 0 {
					b.WriteString(` font-style="italic"`)
				}
				if s.Face&text.Mono != 0 {
					b.WriteString(` font-family="`)
					w.Text(w.family(true))
					b.WriteByte('"')
				}
				b.WriteByte('>')
				w.Text(s.Text)
				b.WriteString("</tspan>")
			}
			x += s.Width
		}
		y += l.LineHeight
	}
	b.WriteString("</text>")
	for _, ic := range w.icons {
		w.icon(ic.name, ic.box, st.Text)
	}
	w.icons = w.icons[:0]
}

type icon struct {
	name string
	box  scene.Rect
}

var (
	linkSchemes  = []string{"http", "https", "mailto", "tel"}
	imageSchemes = []string{"http", "https"}
)

// safeLink is the link, if it is relative or of a scheme the options allow, or nothing.
// The diagrams sanitize what their authors write as Mermaid does; this is the last check.
func (w *Writer) safeLink(link string) string {
	return safeURL(link, w.o.LinkSchemes, linkSchemes)
}

func safeURL(link string, schemes, defaults []string) string {
	link = strings.TrimSpace(link)
	for _, r := range link {
		if r < 0x20 || r >= 0x7f && r <= 0x9f {
			return ""
		}
	}
	if schemes == nil {
		schemes = defaults
	}
	end := strings.IndexAny(link, ":/?#")
	if end <= 0 || link[end] != ':' {
		if strings.HasPrefix(link, "//") || strings.HasPrefix(link, "\\\\") {
			return ""
		}
		return link
	}
	scheme := strings.ToLower(link[:end])
	for _, s := range schemes {
		if s == scheme {
			return link
		}
	}
	return ""
}
