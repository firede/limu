package pelesvg

import (
	"strconv"
	"strings"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// Series is the lines the render oracles write of the series colors a palette sets, one
// "series N color" for each series the scene's palette sets, in order: a color as
// style.Color's String writes it, "none" for one that draws nothing, or the name of the
// theme's color it takes, as "fg".
func Series(sc *scene.Scene) []string {
	var lines []string
	for i, p := range sc.Palette {
		if p.Set() {
			lines = append(lines, seriesLine(i+1, p))
		}
	}
	return lines
}

func seriesLine(n int, p style.Paint) string {
	name := "none"
	switch {
	case p.Token().Name() != "":
		name = p.Token().Name()
	case p.Visible():
		name = p.Own().String()
	}
	return "series " + strconv.Itoa(n) + " " + name
}

// ReadSeries reads a line of Pele's "series N value" as limu reads value, with read, into the
// line Series writes of it, or "" where limu leaves the series unset, as for a value it can't
// work out. Any other line is as it is. The suites keep the series lines in full, for a test
// to give this to fixture.Drawings' Diff.
func ReadSeries(line string, read func(string) style.Paint) string {
	rest, ok := strings.CutPrefix(line, "series ")
	if !ok {
		return line
	}
	number, value, ok := strings.Cut(rest, " ")
	n, err := strconv.Atoi(number)
	if !ok || err != nil {
		return line
	}
	p := read(value)
	if !p.Set() {
		return ""
	}
	return seriesLine(n, p)
}

// InkNames are the colors Pele's ink picks for text on a fill, by the theme's paints the
// default look settles them as.
var InkNames = map[style.Paint]string{style.Fg: "var(--pele-fg-on-light,#1f1f1f)", style.Bg: "var(--pele-fg-on-dark,#fff)"}

// SeriesInk is the color the default look picks for the text of an item that lies on a series
// its scene's palette sets, Fg or Bg, as Pele's seriesInkFill gives it the color its ink
// picks, and unset where the item's text lies on no such series. The look settles it with the
// scene's palette in place of its series colors. Pele writes that color as an attribute, which
// a color the author sets in the style covers, so the author's color is passed over here.
func SeriesInk(sc *scene.Scene, it *scene.Item) style.Paint {
	if !it.Style.TextGround.Visible() {
		return style.Paint{}
	}
	var look *style.Look
	author := it.Author
	author.Text = style.Paint{}
	return look.WithSeries(sc.Palette).Cascade(string(it.Role), it.Style, author).Text
}
