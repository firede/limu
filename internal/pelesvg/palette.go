package pelesvg

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// A Palette is what a test finds of a palette in a scene, by comparing it with its bare scene,
// the same diagram laid out without its palette's theme variables.
type Palette struct {
	// Lines are the lines the render oracles write of the series colors a palette sets, one
	// "series N color" for each series the scene draws something in with the palette's color,
	// in order: a color as style.Color's String writes it, "none" for one that draws
	// nothing, or the name of the theme's color it takes, as "fg". A difference from what a
	// palette should give, as one item of a series without the color the others have, is a
	// line that says so.
	Lines []string
	// Drawn says which series the scene draws something in that the author's own style does
	// not cover, where a palette's color would show.
	Drawn [8]bool
	// Bare is the scene with each item's author's style as the bare scene has it, and the rest
	// as the scene has it, for writing the author's own style as Pele writes it apart from
	// its palette.
	Bare *scene.Scene
}

// ReadPalette finds what a scene carries of a palette, against its bare scene. Each item drawn
// in a series, in the fill, the stroke or the text of its own style, has the palette's color
// in that field of its author's style, at the opacity its own style draws the series at,
// unless the author's own style sets that field. Nothing else differs from the bare scene but
// the ground of text on a series, which the lines of text check against Pele's.
func ReadPalette(sc, bare *scene.Scene) Palette {
	var p Palette
	view := *sc
	view.Items = append([]scene.Item(nil), sc.Items...)
	p.Bare = &view
	if len(sc.Items) != len(bare.Items) {
		p.Lines = append(p.Lines, fmt.Sprintf("palette: %d items, %d without it", len(sc.Items), len(bare.Items)))
		return p
	}
	var colors [8]string
	var unset [8]bool
	for k := range sc.Items {
		it, b := sc.Items[k], bare.Items[k]
		same := it
		same.Author.Fill, same.Author.Stroke, same.Author.Text = b.Author.Fill, b.Author.Stroke, b.Author.Text
		same.Style.TextGround = b.Style.TextGround
		if !reflect.DeepEqual(same, b) {
			p.Lines = append(p.Lines, fmt.Sprintf("palette: item %d %s differs from its bare scene's", k, it.Role))
			continue
		}
		view.Items[k].Author = b.Author
		for _, f := range [...]struct {
			name                string
			own, author, before style.Paint
		}{
			{"fill", it.Style.Fill, it.Author.Fill, b.Author.Fill},
			{"stroke", it.Style.Stroke, it.Author.Stroke, b.Author.Stroke},
			{"text", it.Style.Text, it.Author.Text, b.Author.Text},
		} {
			n, ok := strings.CutPrefix(f.own.Token().Name(), "series-")
			if !ok || f.before.Set() {
				if f.author != f.before {
					p.Lines = append(p.Lines, fmt.Sprintf("palette: item %d %s's %s, %v, is not the author's own", k, it.Role, f.name, f.author))
				}
				continue
			}
			i := int(n[0] - '1')
			p.Drawn[i] = true
			if !f.author.Set() {
				unset[i] = true
				continue
			}
			if f.author.Factor() != f.own.Factor() {
				p.Lines = append(p.Lines, fmt.Sprintf("palette: item %d %s's %s is at %v, not %v", k, it.Role, f.name, f.author.Factor(), f.own.Factor()))
			}
			name := paintName(f.author)
			if colors[i] != "" && colors[i] != name {
				p.Lines = append(p.Lines, fmt.Sprintf("palette: series %d is %s and %s", i+1, colors[i], name))
			}
			colors[i] = name
		}
	}
	var lines []string
	for i, name := range colors {
		switch {
		case name == "":
		case unset[i]:
			lines = append(lines, fmt.Sprintf("palette: series %d is %s and unset", i+1, name))
		default:
			lines = append(lines, "series "+strconv.Itoa(i+1)+" "+name)
		}
	}
	p.Lines = append(lines, p.Lines...)
	return p
}

// paintName is a palette's color as the series lines write it, whatever its opacity.
func paintName(p style.Paint) string {
	switch {
	case p.Token().Name() != "":
		return p.Token().Name()
	case p.Visible():
		return p.Own().String()
	}
	return "none"
}

// ReadSeries reads a line of Pele's "series N value" as limu reads value, with read, into the
// line ReadPalette writes of it. It reads it as "" where limu leaves the series unset, as for a
// value it can't work out, or where the scene draws nothing in the series that would show it,
// as Pele sets every series its palette names. Any other line is as it is. The suites keep the
// series lines in full, for a test to give this to fixture.Drawings' Diff.
func (p *Palette) ReadSeries(line string, read func(string) style.Paint) string {
	rest, ok := strings.CutPrefix(line, "series ")
	if !ok {
		return line
	}
	number, value, ok := strings.Cut(rest, " ")
	n, err := strconv.Atoi(number)
	if !ok || err != nil || n < 1 || n > 8 {
		return line
	}
	c := read(value)
	if !c.Set() || !p.Drawn[n-1] {
		return ""
	}
	return "series " + number + " " + paintName(c)
}

// InkNames are the colors Pele's ink picks for text on a fill, by the theme's paints the
// default look settles them as.
var InkNames = map[style.Paint]string{style.Fg: "var(--pele-fg-on-light,#1f1f1f)", style.Bg: "var(--pele-fg-on-dark,#fff)"}

// SeriesInk is the color the default look picks for the text of an item that lies on a series
// a palette sets, Fg or Bg, as Pele's seriesInkFill gives it the color its ink picks, and unset
// where the item's text lies on no such series. Pele writes that color as an attribute, which
// a color the author sets in the style covers, so the author's color is passed over here.
func SeriesInk(it *scene.Item) style.Paint {
	if !it.Style.TextGround.Visible() {
		return style.Paint{}
	}
	var look *style.Look
	author := it.Author
	author.Text = style.Paint{}
	return look.Cascade(string(it.Role), it.Style, author).Text
}

// WithoutPalette is a diagram's configuration without its theme variables, which limu reads
// only for a palette, for laying out the bare scene ReadPalette compares with.
func WithoutPalette(config map[string]any) map[string]any {
	out := make(map[string]any, len(config))
	for k, v := range config {
		if k != "themeVariables" {
			out[k] = v
		}
	}
	return out
}

// InsertSeries is the lines of a drawing with the series lines put where the render oracles write
// them: after its size and the lines of its title and description for assistive technology.
func InsertSeries(lines, series []string) []string {
	at := 0
	for at < len(lines) && (at == 0 || strings.HasPrefix(lines[at], "acc-")) {
		at++
	}
	out := append(append(append([]string(nil), lines[:at]...), series...), lines[at:]...)
	return out
}
