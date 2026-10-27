package sankey

import (
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// The fixture is what Pele's render draws of sankey diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn, with every number in full, moved as the scene
// moves it, and the errors of the diagrams it refuses; ./scripts/oracle.sh writes it. The
// test writes the same lines from the scene, and so compares them to the last bit.
//
// Where Pele writes the style the config gives a bar or a band, the test reads it as limu
// reads declarations, at the opacity the band's group gives it, and compares what that gives
// with the item's author style.

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
	if len(p.Cases) < 5000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, refused := 0, 0
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
		var got []string
		sc, err := Layout(m, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize})
		if err != nil {
			e := err.(*diagram.Error)
			got = []string{"error " + string(e.Code) + " " + e.Message}
			refused++
		} else {
			got = sceneLines(sc)
		}
		if msg := f.Diff(k, &c.Drawing, got, authorLine); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), %s", c.Src, c.MaxWidth, c.FontSize, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if refused < 20 {
		t.Errorf("only %d diagrams refused", refused)
	}
}

// author writes an author's style: as "author=" and the style, or as "css=", as Pele writes
// no style, where the author set none. A line of Pele's with no style is so read as it is, and
// the oracle keeps only the lines with one in full.
func author(s style.Style) string {
	if v := fmt.Sprintf("%+v", s); v != fmt.Sprintf("%+v", style.Style{}) {
		return "author=" + v
	}
	return "css="
}

// authorLine is a line of the fixture with the style Pele writes, "css=", read as limu reads
// declarations, "author=". A band's own style holds the fill opacity of the group Pele draws it in.
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
	return line[:at] + " " + author(css.Declarations(strings.Split(written, ";")).Style) + line[end:]
}

// series are the colors Pele's seriesColor gives, as its SVG writes them.
var series = []string{"#4c78a8", "#f58518", "#54a24b", "#e45756", "#72b7b2", "#eeca3b", "#b279a2", "#9d755d"}

// paintName is a paint as Pele's SVG names it, "-" for the text's own color, which Pele
// leaves to the root, and a paint at an opacity other than factor "?".
func paintName(p style.Paint, factor float32) string {
	if p.Factor() != factor {
		return "?"
	}
	if p == style.Fg {
		return "-"
	}
	for k, hex := range series {
		if p.Token() == style.Series(k).Token() {
			return fmt.Sprintf("var(--pele-series-%d,%s)", k+1, hex)
		}
	}
	switch p {
	case style.Muted:
		return "var(--_m)"
	case style.Bg:
		return "var(--_bg)"
	}
	return "?"
}

// sceneLines writes a scene as the oracle reads Pele's SVG, every number as JavaScript's
// String writes it.
func sceneLines(sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Title != "" || sc.Desc != "" {
		lines = append(lines, "acc")
	}
	items := sc.Items
	if len(items) > 0 && items[0].Role == RoleTitle {
		it := &items[0]
		l := it.Label
		y := it.At.Y - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
		var content []string
		for _, line := range l.Lines {
			s := ""
			for _, span := range line {
				s += span.Text
			}
			if len(line) == 0 {
				s = " "
			}
			content = append(content, xmlText(s))
		}
		size := "-"
		if l.Size != sc.FontSize {
			size = n(l.Size)
		}
		lines = append(lines, "title "+pt(it.At.X, y)+" "+size+" | "+strings.Join(content, " | "))
		items = items[1:]
	}
	lines = append(lines, "links 0.4")
	// The labels' size, which Pele writes on their group: that of every label, which shrinks
	// with a narrow or crowded chart, or with none, the size it starts from.
	labelSize := jsmath.Round(float64(sc.FontSize * 0.875))
	for i := range items {
		if items[i].Role == RoleLabel {
			labelSize = items[i].Label.Size
			break
		}
	}
	labelsLine := "labels " + n(labelSize)
	labelled := false
	// Where each node's bar starts, by its id.
	bars := map[string]float64{}
	for i := 0; i < len(items); i++ {
		it := &items[i]
		switch it.Role {
		case RoleLink:
			var d []string
			it.Parts[0].Path.Each(func(op scene.Op, a []float64) {
				s := string("MLQCZ"[op])
				for k := 0; k < len(a); k += 2 {
					s += " " + pt(a[k], a[k+1])
				}
				d = append(d, s)
			})
			lines = append(lines, "link "+pelesvg.ID(it.ID)+" "+strings.Join(d, " ")+" "+paintName(pelesvg.OwnFill(it), 0.4)+" "+author(it.Author)+
				" "+pelesvg.CapJoin(it)+" | "+pelesvg.ID(it.Tooltip))
		case RoleNode:
			r := it.Parts[0].Box
			radius := "-"
			if it.Parts[0].Rounded {
				radius = "r" + n(it.Style.Radius.Or(0))
			}
			stroke := "-"
			if it.Style.Stroke.Set() || it.Parts[0].Paint != scene.FillOnly {
				stroke = "?"
			}
			bars[it.ID] = r.X
			lines = append(lines, "node "+pelesvg.ID(it.ID)+" "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+radius+" "+
				paintName(it.Style.Fill, 1)+" "+stroke+" "+author(it.Author)+" "+pelesvg.CapJoin(it))
		case RoleLabel:
			if !labelled {
				labelled = true
				lines = append(lines, labelsLine)
			}
			// The id and the value of a node, as Pele writes them in one text: anchored where
			// the id starts when the label is right of the bar, and where the value ends when
			// it is left of it.
			id := it
			var value *scene.Item
			if i+1 < len(items) && items[i+1].Role == RoleValue && items[i+1].ID == it.ID {
				value = &items[i+1]
				i++
			}
			anchor, at := "end", id
			switch {
			case value == nil:
				if id.Align == scene.Start {
					anchor = "start"
				}
			case id.At.X > bars[id.ID]:
				anchor = "start"
			default:
				at = value
			}
			line := "label " + pelesvg.ID(it.ID) + " " + pt(at.At.X, at.At.Y) + " " + anchor + " " + paintName(id.Style.Text, 1) +
				" | " + xmlText(id.Label.Lines[0][0].Text)
			if value != nil {
				line += " | " + paintName(value.Style.Text, 1) + " " + xmlText(value.Label.Lines[0][0].Text)
				if value.Style.Halo != id.Style.Halo || value.Style.HaloWidth != id.Style.HaloWidth || n(value.At.Y) != n(id.At.Y) {
					line += " | value apart"
				}
			}
			if h := id.Style; h.Halo.Set() {
				line += " | halo none " + paintName(h.Halo, 1) + " " + n(h.HaloWidth.Or(0)) + " round"
			}
			lines = append(lines, line)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	if !labelled {
		lines = append(lines, labelsLine)
	}
	return lines
}

// xmlText is text as Pele's esc writes it, without the control characters XML does not allow.
func xmlText(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, s)
}
