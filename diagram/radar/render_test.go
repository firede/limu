package radar

import (
	"encoding/json"
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
)

// The fixture is what Pele's render draws of radar charts, read back out of its SVG as one
// line per thing drawn, in the order drawn, with every number in full; ./scripts/oracle.sh
// writes it. The test writes the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		MaxWidth float64  `json:"maxWidth"`
		FontSize float64  `json:"fontSize"`
		Lines    []string `json:"lines"`
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/render.json")
	if err != nil {
		t.Fatal(err)
	}
	var p peleRender
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 3500 {
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
		sc := Layout(m, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize})
		got := sceneLines(sc)
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

// paintName is a paint as Pele's SVG names it, with the opacity it is drawn at, "-" for full.
func paintName(p style.Paint) (string, string) {
	for k := range 8 {
		for _, a := range []float64{1, 0.2, 0.4} {
			if p == style.Series(k).Alpha(a) {
				opacity := "-"
				if a != 1 {
					opacity = chars.NumberString(a)
				}
				return fmt.Sprintf("var(--pele-series-%d,%s)", k+1, style.Default().Series[k]), opacity
			}
		}
	}
	switch p {
	case style.Fg:
		return "var(--_fg)", "-"
	case style.SurfaceAlt:
		return "var(--_a)", "-"
	case style.Border:
		return "var(--_b)", "-"
	}
	return "?", "?"
}

// sceneLines writes a scene as the oracle reads Pele's SVG, every number as JavaScript's
// String writes it.
func sceneLines(sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Title != "" {
		lines = append(lines, "acc-title "+pelesvg.ID(sc.Title))
	}
	if sc.Desc != "" {
		lines = append(lines, "acc-desc "+pelesvg.ID(sc.Desc))
	}
	pen := func(it *scene.Item, part *scene.Part) string {
		fill, opacity := "none", "-"
		if part.Paint != scene.StrokeOnly {
			fill, opacity = paintName(pelesvg.OwnFill(it))
		}
		stroke, _ := paintName(it.Style.Stroke)
		width := "-"
		if w, ok := it.Style.StrokeWidth.Get(); ok {
			width = n(w)
		}
		return strings.Join([]string{fill, opacity, stroke, width, pelesvg.CapJoin(it)}, " ")
	}
	what := func(it *scene.Item) string {
		switch it.Role {
		case RoleTitle:
			return "title -"
		case RoleGraticule:
			return "graticule"
		case RoleAxes:
			return "axes"
		case RoleCurve:
			return "curve " + it.ID
		case RoleAxisLabel:
			return "axis-label " + it.ID
		case RoleLegend:
			return "legend " + it.ID
		}
		return "unknown role " + string(it.Role)
	}
	label := func(it *scene.Item) {
		l := it.Label
		var text []string
		for _, line := range l.Lines {
			s := ""
			for _, span := range line {
				s += span.Text
			}
			text = append(text, s)
		}
		fill, _ := paintName(it.Style.Text)
		lines = append(lines, what(it)+" label "+pt(it.At.X, it.At.Y)+" "+pt(l.Width, l.Height)+" "+fill+" "+n(l.Size)+" | "+strings.Join(text, " | "))
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		for k := range it.Parts {
			part := &it.Parts[k]
			if part.Path == nil {
				r := part.Box
				radius := "-"
				if part.Rounded {
					radius = "r" + n(it.Style.Radius.Or(0))
				}
				lines = append(lines, what(it)+" rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+radius+" "+pen(it, part))
				continue
			}
			var subpaths [][]string
			var args [][]float64
			part.Path.Each(func(op scene.Op, a []float64) {
				if op == scene.MoveTo {
					subpaths = append(subpaths, nil)
					args = append(args, nil)
				}
				s := string("MLQCZ"[op])
				for j := 0; j < len(a); j += 2 {
					s += " " + pt(a[j], a[j+1])
				}
				last := len(subpaths) - 1
				subpaths[last] = append(subpaths[last], s)
				args[last] = append(args[last], a...)
			})
			for j, s := range subpaths {
				if it.Role == RoleGraticule && it.Shape == "circle" {
					// The circle, as Path.Ellipse draws it: from its right end, a quarter down to
					// its bottom first.
					a := args[j]
					lines = append(lines, what(it)+" circle "+pt(a[6], a[1])+" "+pt(a[0], a[7])+" "+pen(it, part))
					continue
				}
				lines = append(lines, what(it)+" path "+strings.Join(s, " ")+" "+pen(it, part))
			}
		}
		if it.Label != nil {
			label(it)
		}
	}
	return lines
}
