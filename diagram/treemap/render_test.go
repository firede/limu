package treemap

import (
	"fmt"
	"net/url"
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

// The fixture is what Pele's render draws of treemaps, read back out of its SVG as one line
// per thing drawn, in the order drawn, with its numbers written out in full; ./scripts/oracle.sh
// writes it. The test writes the same lines from the scene, its numbers as JavaScript's String
// writes them, and so compares them to the last bit.
//
// Where Pele writes the style an author's class gives a shape or a text, the test reads it as
// limu reads declarations, and compares what that gives with the item's author style: of a
// shape, all but the color of its text; of a text, its color and opacity.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		MaxWidth float64  `json:"maxWidth"`
		Padding  *float64 `json:"padding"`
		FontSize float64  `json:"fontSize"`
		fixture.Drawing
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 5000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
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
		got := sceneLines(Layout(m, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize, Padding: c.Padding}))
		if msg := f.Diff(k, &c.Drawing, got, authorLine); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v, padding %v), %s", c.Src, c.MaxWidth, c.FontSize, (&layout.Options{Padding: c.Padding}).Pad(), msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// shapeView is what of an author's style Pele writes on a shape: all but the color of text.
func shapeView(s style.Style) string {
	s.Text = style.Paint{}
	return fmt.Sprintf("%+v", s)
}

// textView is what of an author's style Pele writes on a text that changes how limu draws
// it: its color and its opacity.
func textView(s style.Style) string {
	return fmt.Sprintf("%+v", style.Style{Text: s.Text, Opacity: s.Opacity})
}

// author writes what view takes of an author's style: as "author=" and the view, or as
// "css=", as Pele writes no style, where the view is that of no style. A line of Pele's with
// no style is so read as it is, and the oracle keeps only the lines with one in full.
func author(view func(style.Style) string, s style.Style) string {
	if v := view(s); v != view(style.Style{}) {
		return "author=" + v
	}
	return "css="
}

// authorLine is a line of the fixture with the style Pele writes, "css=", read as limu reads
// declarations, "author=": on a shape as it is, and on a text with its fill, which is the
// color the author gave, read as that color.
func authorLine(line string) string {
	at := strings.Index(line, " css=")
	if at < 0 {
		return line
	}
	end := strings.IndexByte(line[at+1:], ' ')
	if end < 0 {
		end = len(line)
	} else {
		end += at + 1
	}
	written, err := url.PathUnescape(line[at+len(" css=") : end])
	if err != nil {
		return line
	}
	decls := strings.Split(written, ";")
	var field string
	if strings.HasPrefix(line, "section ") || strings.HasPrefix(line, "leaf ") {
		field = author(shapeView, css.Declarations(decls).Style)
	} else {
		for i, d := range decls {
			if strings.HasPrefix(d, "fill:") {
				decls[i] = "color:" + d[len("fill:"):]
			}
		}
		field = author(textView, css.Declarations(decls).Style)
	}
	return line[:at] + " " + field + line[end:]
}

// series are the colors Pele's seriesColor gives, as its SVG writes them.
var series = []string{"#4c78a8", "#f58518", "#54a24b", "#e45756", "#72b7b2", "#eeca3b", "#b279a2", "#9d755d"}

// fillName is how a box is filled, as the oracle reads Pele's fill and fill-opacity.
func fillName(p style.Paint) string {
	for k, hex := range series {
		name := fmt.Sprintf("var(--pele-series-%d,%s)", k+1, hex)
		switch p {
		case style.Series(k):
			return name + " -"
		case style.Series(k).Alpha(0.12):
			return name + " 0.12"
		}
	}
	return "? ?"
}

// strokeName is how a box is stroked, "-" for none.
func strokeName(p style.Paint) string {
	if p == style.None {
		return "-"
	}
	name := fillName(p)
	return name[:strings.IndexByte(name, ' ')]
}

var textPaints = map[style.Paint]string{style.Fg: "var(--_fg)", style.Muted: "var(--_m)", style.Bg: "var(--_bg)"}

var anchors = [...]string{scene.Middle: "middle", scene.Start: "start", scene.End: "end"}

// written is text as Pele's esc writes it, without the control characters XML does not
// allow.
func written(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := chars.NumberString
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// The lines of a label drawn about At, as Pele's text and labelSvg place them, after
	// prefix: a line of one line of text at its baseline, and a title's lines below each
	// other from where the first goes.
	label := func(prefix, size, weight string, it *scene.Item) {
		l := it.Label
		if l == nil || l.Empty() {
			return
		}
		y := it.At.Y + float64(l.Size*0.35)
		if it.Role == RoleTitle {
			y = it.At.Y - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
		}
		if size == "" {
			size = num(l.Size)
		}
		for _, line := range l.Lines {
			at := prefix + "t " + pt(it.At.X, y) + " " + size + " " + anchors[it.Align] + " " + textPaints[it.Style.Text] + " " +
				weight + " " + author(textView, it.Author) + " "
			if len(line) == 0 {
				lines = append(lines, at+" ")
			}
			for k, span := range line {
				if k == 0 {
					lines = append(lines, at+written(span.Text))
				} else {
					lines = append(lines, prefix+"+ "+written(span.Text))
				}
			}
			y += l.LineHeight
		}
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			label("title ", "-", "var(--_tw)", it)
		case RoleSection, RoleLeaf:
			kind := "section"
			if it.Role == RoleLeaf {
				kind = "leaf"
			}
			r := it.Parts[0].Box
			rounded := "-"
			if it.Parts[0].Rounded {
				rounded = "r"
			}
			lines = append(lines, fmt.Sprintf("%s %s [%s] %s,%s %s %s %s %s %s", kind, pelesvg.ID(it.ID), strings.Join(it.Class, " "),
				pt(r.X, r.Y), pt(r.W, r.H), rounded, fillName(pelesvg.OwnFill(it)), strokeName(it.Style.Stroke), pelesvg.CapJoin(it),
				author(shapeView, it.Author)))
			weight := "-"
			if it.Label != nil && len(it.Label.Lines) > 0 && it.Label.Lines[0][0].Face&text.Bold != 0 {
				weight = "var(--_hw)"
			}
			label("", "", weight, it)
		case RoleValue:
			label("", "", "-", it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
