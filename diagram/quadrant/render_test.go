package quadrant

import (
	"fmt"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of quadrant charts, read back out of its SVG as one
// line per thing drawn, in the order drawn, with its numbers written out in full;
// ./scripts/oracle.sh writes it. The test writes the same lines from the scene, its numbers
// as JavaScript writes them, so that a difference in the last bit shows.

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
	failed, points := 0, 0
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
		points += len(m.Points)
		got := sceneLines(Layout(m, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize}))
		if i := pelesvg.FirstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize, i,
					pelesvg.At(got, i), pelesvg.At(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if points < 10000 {
		t.Errorf("only %d points drawn", points)
	}
}

// paintName is a paint as Pele's SVG names it, and an author's color as the oracle writes
// one.
func paintName(p style.Paint) string {
	switch p {
	case style.Fg:
		return "var(--_fg)"
	case style.Muted:
		return "var(--_m)"
	case style.Surface:
		return "var(--_s)"
	case style.Border:
		return "var(--_b)"
	case style.Series(0):
		return "var(--pele-series-1,#4c78a8)"
	case style.None:
		return "none"
	}
	if p.Token().Name() == "" && p.Visible() && p.Factor() == 1 {
		c := p.Own()
		if c.A == 255 {
			return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
		}
	}
	return "?"
}

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

// faceName is emphasis as the oracle writes it.
func faceName(f text.Face) string {
	s := ""
	if f&text.Bold != 0 {
		s += "b"
	}
	if f&text.Italic != 0 {
		s += "i"
	}
	if s == "" {
		return "-"
	}
	return s
}

// shared is the emphasis every span of a label with text has, as Pele's sharedStyle finds it.
func shared(l *text.Label) text.Face {
	all := text.Bold | text.Italic
	any := false
	for _, line := range l.Lines {
		for _, s := range line {
			if s.Icon != "" || s.Text == "" {
				continue
			}
			all &= s.Face
			any = true
		}
	}
	if !any {
		return 0
	}
	return all
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// The lines of a label drawn about cx, cy, as Pele's labelSvg writes them: a label of a
	// single span with no emphasis as one <text>, and any other as a <tspan> for each line
	// and each span. A title's weight is set on its <text>.
	label := func(it *scene.Item, cx, cy float64, title bool) {
		l := it.Label
		if l == nil || l.Empty() {
			return
		}
		y := cy - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
		head := func(face text.Face) string {
			return "t " + pt(cx, y) + " " + n(l.Size) + " " + paintName(it.Style.Text) + " " + faceName(face) + " "
		}
		if len(l.Lines) == 1 && len(l.Lines[0]) == 1 && l.Lines[0][0].Icon == "" && l.Lines[0][0].Face == 0 && !title {
			lines = append(lines, head(0)+written(l.Lines[0][0].Text))
			return
		}
		share := shared(l)
		top := share
		if title {
			lines = append(lines, "title")
			top = text.Bold | share&text.Italic
		}
		for _, line := range l.Lines {
			for k, s := range line {
				face := top | s.Face&^share
				if k == 0 {
					lines = append(lines, head(face)+written(s.Text))
				} else {
					lines = append(lines, "+ "+faceName(face)+" "+written(s.Text))
				}
			}
			if len(line) == 0 {
				lines = append(lines, head(top)+" ")
			}
			y += l.LineHeight
		}
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			label(it, it.At.X, it.At.Y, true)
		case RoleQuadrant:
			r := it.Parts[0].Box
			rounded := "-"
			if it.Parts[0].Rounded {
				rounded = "r"
			}
			lines = append(lines, "quadrant "+it.ID, "rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+rounded+" "+
				paintName(it.Style.Fill)+" "+paintName(it.Style.Stroke)+" "+pelesvg.CapJoin(it))
			label(it, it.At.X, it.At.Y, false)
		case RoleAxis:
			if it.Rotate != 0 {
				lines = append(lines, "turn "+pt(it.At.X, it.At.Y)+" "+n(it.Rotate))
				label(it, 0, 0, false)
			} else {
				label(it, it.At.X, it.At.Y, false)
			}
		case RolePoint:
			var args [][]float64
			it.Parts[0].Path.Each(func(op scene.Op, a []float64) { args = append(args, a) })
			// A circle, as Path.Ellipse draws it: from its right end, four quarters about
			// its middle.
			cx, cy, r := args[1][4], args[0][1], it.Box.W/2
			s := "point " + pelesvg.ID(it.ID) + " [" + strings.Join(it.Class, " ") + "] " + pt(cx, cy) + " " + n(r)
			if args[0][0] != cx+r || it.Box.X != cx-r {
				s += " not a circle about its box"
			}
			pen := it.Author.Over(it.Style)
			width := "-"
			if v, ok := it.Author.StrokeWidth.Get(); ok {
				width = n(v)
			}
			lines = append(lines, s+" "+paintName(pen.Fill)+" "+paintName(pen.Stroke)+" "+width+" "+pelesvg.CapJoin(it))
		case RolePointLabel:
			label(it, it.At.X, it.At.Y, false)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
