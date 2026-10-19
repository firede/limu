package venn

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of venn diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn, with its numbers written out in full;
// ./scripts/oracle.sh writes it. The test writes the same lines from the scene, its numbers as
// JavaScript's String writes them, and so compares them to the last bit.
//
// Where Pele writes the style an author gives a circle, an overlap or a text, the test works
// out what a browser draws from it and from the attributes Pele writes beside it, reading the
// declarations as limu reads them, and compares that with the item's style under its author's.
// The outline of an overlap Pele writes with arcs; the test draws it again with scene.Path's
// ArcTo from Pele's corners, radii and flags, and compares that path with the item's.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		FontSize float64 `json:"fontSize"`
		fixture.Drawing
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 6000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, overlaps := 0, 0
	for k, c := range p.Cases {
		pre, err := diagram.Prepare(c.Src, diagram.Options{})
		if err != nil {
			t.Errorf("Prepare(%q): %v", c.Src, err)
			continue
		}
		m, err := ParsePrepared(pre)
		if err != nil {
			t.Errorf("ParsePrepared(%q): %v", c.Src, err)
			continue
		}
		got := sceneLines(Layout(m, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize}))
		for _, line := range got {
			if strings.HasPrefix(line, "overlap ") {
				overlaps++
			}
		}
		if msg := f.Diff(k, &c.Drawing, got, peleLine); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), %s", c.Src, c.MaxWidth, c.FontSize, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if overlaps < 500 {
		t.Errorf("only %d overlaps drawn", overlaps)
	}
}

// series are the colors Pele's seriesColor gives, as its SVG writes them.
var series = []string{"#4c78a8", "#f58518", "#54a24b", "#e45756", "#72b7b2", "#eeca3b", "#b279a2", "#9d755d"}

// seriesOf is the series a color Pele writes is, by its number.
func seriesOf(color string) style.Paint {
	for k, hex := range series {
		if color == fmt.Sprintf("var(--pele-series-%d,%s)", k+1, hex) {
			return style.Series(k)
		}
	}
	return style.Paint{}
}

// declarations are the declarations of the style Pele writes on an element, as the oracle
// writes it, encoded.
func declarations(field string) []string {
	css, err := url.PathUnescape(strings.TrimPrefix(field, "css="))
	if err != nil {
		panic(err)
	}
	return strings.Split(css, ";")
}

// lastOpacity is the value of the last declaration of an opacity among decls that a browser
// takes, a number or a percentage, kept from 0 to 1, and whether there is one.
func lastOpacity(decls []string, prop string) (float64, bool) {
	value, found := "", false
	for _, d := range decls {
		if p, v, ok := strings.Cut(d, ":"); ok && strings.TrimSpace(p) == prop {
			value, found = strings.TrimSpace(v), true
		}
	}
	if !found {
		return 0, false
	}
	scale := 1.0
	if strings.HasSuffix(value, "%") {
		value, scale = strings.TrimSuffix(value, "%"), 100
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return max(0, min(1, f/scale)), true
}

// without is decls without the declarations of prop.
func without(decls []string, prop string) []string {
	var out []string
	for _, d := range decls {
		if p, _, _ := strings.Cut(d, ":"); strings.TrimSpace(p) != prop {
			out = append(out, d)
		}
	}
	return out
}

// view is a style as the test compares it.
func view(s style.Style) string { return fmt.Sprintf("%+v", s) }

// author writes the style an author gave a text: as "author=" and its view, or as "css=", as
// Pele writes no style, where the author gave none. A text of Pele's with no style is so read
// as it is, and the oracle keeps only the texts with one in full.
func author(s style.Style) string {
	if v := view(s); v != view(style.Style{}) {
		return "author=" + v
	}
	return "css="
}

// peleLine is a line of the fixture with the style Pele writes on an element read as what a
// browser draws: of a circle, its fill and stroke from the author's style or Pele's
// attributes, each at the opacity the author's style or Pele's attributes give, under the
// rest of the author's style; of an overlap, the author's style over no fill or stroke; and of
// a text, the color and the opacity the author gave. An overlap's outline is drawn again as
// limu draws arcs.
func peleLine(line string) string {
	fields := strings.Split(line, " ")
	switch fields[0] {
	case "set":
		// set id cx,cy r fill fill-opacity stroke cap join css=
		n := len(fields)
		decls := declarations(fields[n-1])
		fill, stroke := seriesOf(fields[n-6]), seriesOf(fields[n-4])
		opacity, err := strconv.ParseFloat(fields[n-5], 64)
		if err != nil {
			panic(err)
		}
		if f, ok := lastOpacity(decls, "fill-opacity"); ok {
			opacity = f
		}
		strokeOpacity, hasStrokeOpacity := lastOpacity(decls, "stroke-opacity")
		s := css.Declarations(without(without(decls, "fill-opacity"), "stroke-opacity")).Style
		s.Text = style.Paint{}
		if _, ok := lastOpacity(decls, "fill-opacity"); !ok && !hasStrokeOpacity && view(s) == view(style.Style{}) {
			// A style that sets nothing a browser draws is read as no style, and the line as
			// Pele writes a set the author gave none, as sceneLines writes it too.
			return strings.Join(fields[:n-1], " ") + " css="
		}
		if !s.Fill.Set() {
			s.Fill = fill
		}
		s.FillOpacity = style.N(opacity)
		if !s.Stroke.Set() {
			s.Stroke = stroke
		}
		if hasStrokeOpacity {
			s.StrokeOpacity = style.N(strokeOpacity)
		}
		return strings.Join(fields[:n-6], " ") + " " + strings.Join(fields[n-3:n-1], " ") + " style=" + view(s)
	case "overlap":
		// overlap id fill cap join css= | d
		bar := afterCSS(line)
		head := strings.Split(line[:bar], " ")
		n := len(head)
		s := css.Declarations(declarations(head[n-1])).Style
		s.Text = style.Paint{}
		s = s.Over(style.Style{Fill: style.None, Stroke: style.None})
		return strings.Join(head[:n-4], " ") + " " + strings.Join(head[n-3:n-1], " ") + " style=" + view(s) + " | " +
			pathLine(peleOutline(line[bar+3:]))
	case "title", "label", "text":
		// kind id t x,y fill size weight style css= | lines
		bar := afterCSS(line)
		head := strings.Split(line[:bar], " ")
		n := len(head)
		decls := declarations(head[n-1])
		for i, d := range decls {
			if strings.HasPrefix(d, "fill:") {
				decls[i] = "color:" + d[len("fill:"):]
			}
		}
		s := css.Declarations(decls).Style
		return strings.Join(head[:n-1], " ") + " " + author(style.Style{Text: s.Text, Opacity: s.Opacity}) + line[bar:]
	}
	return line
}

// setAttributes is a set's fill, fill opacity and stroke as Pele writes them on its circle,
// where the author gave the set no style and its own style is no more than those, and whether
// it is. A set of Pele's with no style is so read as it is, and the oracle keeps only the
// sets with one in full.
func setAttributes(it *scene.Item) (string, bool) {
	s := it.Style
	opacity, ok := s.FillOpacity.Get()
	if !ok || view(it.Author) != view(style.Style{}) ||
		view(s) != view(style.Style{Fill: s.Fill, FillOpacity: s.FillOpacity, Stroke: s.Stroke}) {
		return "", false
	}
	name := func(p style.Paint) string {
		for k, hex := range series {
			if p == style.Series(k) {
				return fmt.Sprintf("var(--pele-series-%d,%s)", k+1, hex)
			}
		}
		return ""
	}
	fill, stroke := name(s.Fill), name(s.Stroke)
	if fill == "" || stroke == "" {
		return "", false
	}
	return fill + " " + chars.NumberString(opacity) + " " + stroke, true
}

// afterCSS is where the part of a line after its style begins, as an id before it may hold
// anything.
func afterCSS(line string) int {
	at := strings.LastIndex(line[:strings.Index(line, " css=")+1], " ") + 1
	return at + strings.Index(line[at:], " | ")
}

// peleOutline is a path Pele writes, M, L, A and Z with every number in full, drawn as limu
// draws it.
func peleOutline(d string) *scene.Path {
	p := new(scene.Path)
	f := strings.Fields(d)
	num := func(i int) float64 {
		v, err := strconv.ParseFloat(f[i], 64)
		if err != nil {
			panic(err)
		}
		return v
	}
	for i := 0; i < len(f); {
		switch f[i] {
		case "M":
			p.MoveTo(num(i+1), num(i+2))
			i += 3
		case "L":
			p.LineTo(num(i+1), num(i+2))
			i += 3
		case "A":
			p.ArcTo(num(i+1), num(i+2), num(i+3), f[i+4] == "1", f[i+5] == "1", num(i+6), num(i+7))
			i += 8
		case "Z":
			p.Close()
			i++
		default:
			panic("unknown command " + f[i])
		}
	}
	return p
}

// pathLine is a path's commands, every number in full.
func pathLine(p *scene.Path) string {
	var out []string
	p.Each(func(op scene.Op, a []float64) {
		s := string("MLQCZ"[op])
		for _, v := range a {
			s += " " + chars.NumberString(v)
		}
		out = append(out, s)
	})
	return strings.Join(out, " ")
}

// written is text as Pele's esc writes it, without the control characters XML does not allow.
func written(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

var textPaints = map[style.Paint]string{style.Fg: "var(--_fg)", style.Muted: "var(--_m)"}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := chars.NumberString
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// A label as Pele's labelSvg writes it about At: one text with no spans when it is one
	// line of one span of plain text, and otherwise spans, with the emphasis every span has
	// written on the text, unless the attributes give a weight of their own, and the rest on
	// each span.
	label := func(kind, id string, it *scene.Item) {
		l := it.Label
		y := it.At.Y - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
		plain := len(l.Lines) == 1 && len(l.Lines[0]) == 1 && l.Lines[0][0].Icon == "" && l.Lines[0][0].Face == 0
		shared, any := text.Bold|text.Italic, false
		for _, line := range l.Lines {
			for _, span := range line {
				if span.Icon != "" || span.Text == "" {
					continue
				}
				shared &= span.Face
				any = true
			}
		}
		if !any || plain {
			shared = 0
		}
		weight, italic := "-", "-"
		if it.Role == RoleTitle {
			weight = "var(--_tw)"
		} else if shared&text.Bold != 0 {
			weight = "var(--_w)"
		}
		if shared&text.Italic != 0 {
			italic = "italic"
		}
		var content []string
		for _, line := range l.Lines {
			s := ""
			for _, span := range line {
				marks := ""
				if !plain {
					if span.Face&^shared&text.Bold != 0 {
						marks += "b"
					}
					if span.Face&^shared&text.Italic != 0 {
						marks += "i"
					}
				}
				if marks != "" {
					s += "[" + marks + "]"
				}
				s += written(span.Text)
			}
			if len(line) == 0 {
				s = " "
			}
			content = append(content, s)
		}
		lines = append(lines, kind+" "+id+" t "+pt(it.At.X, y)+" "+textPaints[it.Style.Text]+" "+num(l.Size)+" "+weight+" "+italic+
			" "+author(it.Author)+" | "+strings.Join(content, " | "))
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			label("title", "-", it)
		case RoleSet:
			// The circle, as Path.Ellipse draws it: from its right end, four quarters; its
			// radius half its box's width.
			a := it.Parts[0].Path.Args
			cx, cy, r := a[6], a[1], it.Box.W/2
			if pathLine(it.Parts[0].Path) != pathLine(new(scene.Path).Ellipse(cx, cy, r, r)) {
				lines = append(lines, "set drawn other than as its circle")
			}
			if attrs, ok := setAttributes(it); ok {
				lines = append(lines, "set "+pelesvg.ID(it.ID)+" "+pt(cx, cy)+" "+num(r)+" "+attrs+" "+pelesvg.CapJoin(it)+" css=")
				continue
			}
			lines = append(lines, "set "+pelesvg.ID(it.ID)+" "+pt(cx, cy)+" "+num(r)+" "+pelesvg.CapJoin(it)+
				" style="+view(it.Author.Over(it.Style)))
		case RoleOverlap:
			lines = append(lines, "overlap "+pelesvg.ID(it.ID)+" "+pelesvg.CapJoin(it)+" style="+view(it.Author.Over(it.Style))+
				" | "+pathLine(it.Parts[0].Path))
		case RoleLabel:
			label("label", pelesvg.ID(it.ID), it)
		case RoleText:
			label("text", pelesvg.ID(it.ID), it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
