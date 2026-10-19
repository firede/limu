package timeline

import (
	"fmt"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of timeline diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. Pele writes every
// number where it is on the page, as the scene holds it. The test writes the same lines from
// the scene.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		MaxWidth float64  `json:"maxWidth"`
		FontSize float64  `json:"fontSize"`
		Lines    []string `json:"lines"`
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	if err := fixture.Load("testdata/pele/render.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
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
		if i := pelesvg.FirstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize, i, pelesvg.At(got, i), pelesvg.At(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// paintName is a paint as Pele's SVG names it, "-" for none given. Text in the theme's color
// of text takes the fill Pele's SVG gives the whole drawing, and is given none of its own.
func paintName(p style.Paint) string {
	if !p.Set() || p == style.Fg {
		return "-"
	}
	name := "?"
	switch p.Token() {
	case style.Bg.Token():
		name = "var(--_bg)"
	case style.Muted.Token():
		name = "var(--_m)"
	case style.Line.Token():
		name = "var(--_l)"
	case style.Border.Token():
		name = "var(--_b)"
	case style.None.Token():
		name = "none"
	}
	for k := range 8 {
		if p.Token() == style.Series(k).Token() {
			name = fmt.Sprintf("var(--pele-series-%d,%s)", k+1, style.Default().Series[k])
		}
	}
	if f := p.Factor(); f != 1 {
		name += "/" + pelesvg.Num(float64(f))
	}
	return name
}

// partPaint is how a part of an item is painted, as the oracle reads Pele's fill, stroke and
// stroke width.
func partPaint(it *scene.Item, part *scene.Part) string {
	var s string
	switch part.Paint {
	case scene.StrokeOnly:
		s = "none " + paintName(it.Style.Stroke)
	case scene.FillOnly:
		s = paintName(pelesvg.OwnFill(it)) + " -"
	default:
		s = paintName(pelesvg.OwnFill(it)) + " " + paintName(it.Style.Stroke)
	}
	if w, ok := it.Style.StrokeWidth.Get(); ok {
		s += " w" + pelesvg.Num(w)
	}
	return s
}

// xmlText is text as Pele's SVG holds it: without the control characters XML does not allow.
func xmlText(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Where Pele's text puts the first line of a label centered on its point, its paint, its
	// size, and the text of its lines.
	textAt := func(it *scene.Item) string {
		l := it.Label
		y := it.At.Y - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
		s := "t " + pt(it.At.X, y) + " " + paintName(it.Style.Text) + " " + num(l.Size)
		if it.Align != scene.Middle {
			s += " not centered"
		}
		for _, line := range l.Lines {
			var b strings.Builder
			for _, span := range line {
				b.WriteString(span.Text)
			}
			if len(line) == 0 {
				b.WriteString(" ")
			}
			s += " | " + xmlText(b.String())
		}
		return s
	}
	subpaths := func(p *scene.Path) string {
		var out []string
		p.Each(func(op scene.Op, a []float64) {
			w := string("MLQCZ"[op])
			for i := 0; i < len(a); i += 2 {
				w += " " + pt(a[i], a[i+1])
			}
			out = append(out, w)
		})
		return strings.Join(out, " ")
	}
	box := func(it *scene.Item) {
		part := &it.Parts[0]
		r := part.Box
		s := "rect " + pt(r.X, r.Y) + "," + pt(r.W, r.H)
		if part.Rounded {
			s += " r"
		}
		lines = append(lines, s+" "+partPaint(it, part))
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			lines = append(lines, "title "+textAt(it))
		case RoleAxis:
			lines = append(lines, "pele-axis "+subpaths(it.Parts[0].Path)+" "+partPaint(it, &it.Parts[0]))
			for _, m := range it.Ends {
				for _, part := range m.Outline() {
					lines = append(lines, "pele-marker "+subpaths(part.Path)+" "+paintName(it.Style.Stroke)+" none")
				}
			}
		case RoleSection, RolePeriod:
			lines = append(lines, strings.TrimPrefix(string(it.Role), "timeline.")+" "+xmlText(text.DecodeEntities(it.ID)))
			box(it)
			lines = append(lines, textAt(it))
		case RoleEvent:
			lines = append(lines, "event")
			box(it)
			lines = append(lines, textAt(it))
		case RoleConnector:
			lines = append(lines, "pele-edge "+subpaths(it.Parts[0].Path)+" "+partPaint(it, &it.Parts[0]))
		case RoleDot:
			// The circle, as Path.Ellipse draws it: from its right end, four quarters.
			var args [][]float64
			it.Parts[0].Path.Each(func(_ scene.Op, a []float64) { args = append(args, a) })
			cx, cy := args[1][4], args[0][1]
			lines = append(lines, "circle "+pt(cx, cy)+" "+num(args[0][0]-cx)+" "+partPaint(it, &it.Parts[0]))
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
