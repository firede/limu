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
	// Icons gives the inside of an icon's <svg>, on a box of 24 by 24, by the icon's name;
	// labels leave room only for icons the measurer was told of.
	Icons func(name string) string
	// Draw, when set, draws each item in place of the default, which next draws. It is
	// given the item's style, settled from the look.
	Draw func(w *Writer, it *scene.Item, s style.Style, next func())
}

// Render writes a scene as an SVG document.
func Render(sc *scene.Scene, o *Options) string {
	if o == nil {
		o = &Options{}
	}
	w := &Writer{o: o, th: o.Look.ThemeOrDefault()}
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
			o.Draw(w, it, st, func() { w.item(sc, it, st) })
		} else {
			w.item(sc, it, st)
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
	icons []icon
}

// Raw writes markup as it is.
func (w *Writer) Raw(s string) { w.b.WriteString(s) }

// Text writes text, escaped for an attribute or an element.
func (w *Writer) Text(s string) { esc(&w.b, s) }

// Num writes a number to two decimals, with no more digits than it needs.
func (w *Writer) Num(f float64) { num(&w.b, f) }

// Paint writes a paint as a color: a variable of the theme's, or a color, or none. Its
// opacity is not written; see Opacity.
func (w *Writer) Paint(p style.Paint) {
	if !p.Visible() {
		w.b.WriteString("none")
		return
	}
	name := p.Token().Name()
	switch {
	case name == "" && p.Base() == 0:
		w.b.WriteString(p.Own().String())
		return
	case name == "" || w.o.Literal:
		// A color of its own mixed with the theme's is settled here, as are all of them
		// when they are written as colors.
		c := p.Color(w.th)
		if p.Base() == 0 {
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

// paintAttr writes name="paint", and its opacity when it has one.
func (w *Writer) paintAttr(name string, p style.Paint) {
	w.b.WriteByte(' ')
	w.b.WriteString(name)
	w.b.WriteString(`="`)
	w.Paint(p)
	w.b.WriteByte('"')
	a := float64(p.Factor())
	if p.Visible() && p.Token().Name() == "" {
		a *= float64(p.Own().A) / 255
	} else if p.Visible() {
		a *= float64(w.th.Color(p.Token()).A) / 255
	}
	if p.Visible() && a < 1 && p.Base() == 0 {
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

// picture writes an image, or an icon in the color of the part's text.
func (w *Writer) picture(pic *scene.Picture, st style.Style) {
	b := &w.b
	box := func() {
		b.WriteString(` x="`)
		w.Num(pic.Box.X)
		b.WriteString(`" y="`)
		w.Num(pic.Box.Y)
		b.WriteString(`" width="`)
		w.Num(pic.Box.W)
		b.WriteString(`" height="`)
		w.Num(pic.Box.H)
		b.WriteByte('"')
	}
	switch {
	case pic.Image != "":
		href := safeURL(pic.Image, w.o.ImageSchemes, imageSchemes)
		if href == "" {
			return
		}
		b.WriteString(`<image href="`)
		w.Text(href)
		b.WriteByte('"')
		box()
		if pic.Fit {
			b.WriteString(` preserveAspectRatio="xMidYMid meet"/>`)
		} else {
			b.WriteString(` preserveAspectRatio="none"/>`)
		}
	case pic.Icon != "" && w.o.Icons != nil:
		b.WriteString(`<svg class="limu-icon" data-icon="`)
		w.Text(pic.Icon)
		b.WriteByte('"')
		box()
		b.WriteString(` color="`)
		w.Paint(st.Text)
		b.WriteString(`" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`)
		b.WriteString(w.o.Icons(pic.Icon))
		b.WriteString("</svg>")
	}
}

// item writes an item as Pele does: a group of its outline, its markers and its label.
func (w *Writer) item(sc *scene.Scene, it *scene.Item, st style.Style) {
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
	sw := st.StrokeWidth.Or(1)
	for _, part := range it.Parts {
		if part.Rounded || part.Path == nil {
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
		}
		w.paintAttr("fill", fill)
		w.paintAttr("stroke", stroke)
		if stroke.Visible() {
			w.strokeAttrs(st, sw)
		}
		b.WriteString("/>")
	}
	// Markers are drawn solid, on a dashed edge too, as Pele draws them.
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
			w.paintAttr("fill", fill)
			w.paintAttr("stroke", stroke)
			if stroke.Visible() {
				w.strokeAttrs(solid, sw)
			}
			b.WriteString("/>")
		}
	}
	if pic := it.Picture; pic != nil {
		w.picture(pic, st)
	}
	if it.Label != nil && !it.Label.Empty() {
		w.label(sc, it, st)
	}
	if link != "" {
		b.WriteString("</a>")
	}
	b.WriteString("</g>")
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
}

var kindClass = [...]string{"item", "node", "group", "edge", "text", "mark"}

// label writes a label about the item's At, as Pele's labelSvg does: a line's
// baseline is a little below the middle of its line's height. A halo is the text again
// under it, stroked.
func (w *Writer) label(sc *scene.Scene, it *scene.Item, st style.Style) {
	if st.Halo.Visible() {
		w.text(sc, it, func() {
			w.b.WriteString(` fill="none"`)
			w.paintAttr("stroke", st.Halo)
			w.b.WriteString(` stroke-width="`)
			w.Num(st.HaloWidth.Or(0))
			w.b.WriteString(`" stroke-linejoin="round"`)
		}, false)
	}
	w.text(sc, it, func() {
		if st.Text != style.Fg {
			w.paintAttr("fill", st.Text)
		}
	}, true)
}

// text writes a label's <text>, with attrs writing its attributes of paint.
func (w *Writer) text(sc *scene.Scene, it *scene.Item, attrs func(), icons bool) {
	b := &w.b
	l := it.Label
	cx, cy := it.At.X, it.At.Y
	shift := l.Size * 0.35
	y := cy - l.Height/2 + l.LineHeight/2 + shift
	b.WriteString(`<text text-anchor="middle" xml:space="preserve"`)
	attrs()
	if l.Size != sc.FontSize {
		b.WriteString(` font-size="`)
		w.Num(l.Size)
		b.WriteByte('"')
	}
	if it.Rotate != 0 {
		b.WriteString(` transform="rotate(`)
		w.Num(it.Rotate)
		b.WriteByte(' ')
		w.Num(cx)
		b.WriteByte(' ')
		w.Num(cy)
		b.WriteString(`)"`)
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
			if icons && s.Icon != "" && w.o.Icons != nil {
				w.icons = append(w.icons, icon{s.Icon, x, y - shift - l.Size*1.1/2, l.Size * 1.1})
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
		b.WriteString(`<svg class="limu-icon" data-icon="`)
		w.Text(ic.name)
		b.WriteString(`" x="`)
		w.Num(ic.x)
		b.WriteString(`" y="`)
		w.Num(ic.y)
		b.WriteString(`" width="`)
		w.Num(ic.side)
		b.WriteString(`" height="`)
		w.Num(ic.side)
		b.WriteString(`" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`)
		b.WriteString(w.o.Icons(ic.name))
		b.WriteString("</svg>")
	}
	w.icons = w.icons[:0]
}

type icon struct {
	name       string
	x, y, side float64
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
