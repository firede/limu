package xychart

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
)

// The fixture is what Pele's render draws of XY charts, read back out of its SVG as one line
// per thing drawn, in the order drawn, with every number in full, and the errors of the
// charts it refuses; ./scripts/oracle.sh writes it. The test writes the same lines from the
// scene.

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
	if len(p.Cases) < 3500 {
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
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), %s", c.Src, c.MaxWidth, c.FontSize, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if refused < 20 {
		t.Errorf("only %d charts refused", refused)
	}
}

// paintName is a paint as Pele's SVG names it, "-" for none given.
func paintName(p style.Paint) string {
	if !p.Set() || p == style.Fg {
		return "-"
	}
	for k := range 8 {
		if p == style.Series(k) {
			return fmt.Sprintf("var(--pele-series-%d,%s)", k+1, style.Default().Series[k])
		}
	}
	switch p {
	case style.Muted:
		return "var(--_m)"
	case style.SurfaceAlt:
		return "var(--_a)"
	case style.Border:
		return "var(--_b)"
	}
	return "?"
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
	contexts := map[scene.Role]string{
		RoleTitle: "title", RoleAxisLabel: "axis-labels", RoleAxisTitle: "axis-titles",
		RoleDataLabel: "data-labels", RolePointLabel: "point-labels", RoleLegend: "legend",
	}
	// Where Pele's text puts the first line of a label, with its fill, anchor and size and the
	// text of its lines: a plain one-line label that Pele's aligned writes anchored at its
	// point, and any other as Pele's labelSvg centers it.
	textAt := func(it *scene.Item) {
		l := it.Label
		x, y := it.At.X, it.At.Y
		if it.Rotate != 0 {
			lines = append(lines, "rotate "+pt(x, y)+" "+n(it.Rotate))
			x, y = 0, 0
		}
		anchor := "middle"
		aligned := it.Role == RoleAxisLabel || it.Role == RolePointLabel || it.Role == RoleDataLabel
		if aligned && plain(l) {
			anchor = map[scene.Align]string{scene.Middle: "middle", scene.Start: "start", scene.End: "end"}[it.Align]
			y = y + float64(l.Size*0.35)
		} else {
			y = y - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
		}
		var content []string
		for _, line := range l.Lines {
			s := ""
			for _, span := range line {
				s += span.Text
			}
			if len(line) == 0 {
				s = " "
			}
			content = append(content, pelesvg.ID(s))
		}
		lines = append(lines, contexts[it.Role]+" t "+pt(x, y)+" "+paintName(it.Style.Text)+" "+anchor+" "+n(l.Size)+" | "+strings.Join(content, " | "))
	}
	pen := func(it *scene.Item) string {
		s := it.Author.Over(it.Style)
		name := func(v string) string {
			if v == "" {
				return "-"
			}
			return v
		}
		width := "-"
		if w, ok := s.StrokeWidth.Get(); ok {
			width = n(w)
		}
		return paintName(s.Stroke) + " " + width + " " + name(s.Cap.String()) + " " + name(s.Join.String())
	}
	path := func(it *scene.Item, context string, p *scene.Path) {
		var subpaths []string
		p.Each(func(op scene.Op, a []float64) {
			s := string("MLQCZ"[op])
			for i := 0; i < len(a); i += 2 {
				s += " " + pt(a[i], a[i+1])
			}
			if op == scene.MoveTo || len(subpaths) == 0 {
				subpaths = append(subpaths, s)
			} else {
				subpaths[len(subpaths)-1] += " " + s
			}
		})
		for _, s := range subpaths {
			lines = append(lines, context+" path "+s+" "+pen(it))
		}
	}
	rect := func(it *scene.Item, part *scene.Part) {
		r := part.Box
		radius := "-"
		if part.Rounded {
			radius = "r" + n(it.Style.Radius.Or(0))
		}
		lines = append(lines, "rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+radius+" "+paintName(it.Style.Fill))
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle, RoleAxisLabel, RoleAxisTitle, RoleDataLabel, RolePointLabel:
			textAt(it)
		case RoleGrid:
			path(it, "grid", it.Parts[0].Path)
		case RoleAxis:
			path(it, "axis", it.Parts[0].Path)
		case RoleBars:
			lines = append(lines, "bars "+id(it.ID)+" "+paintName(it.Style.Fill))
			for k := range it.Parts {
				rect(it, &it.Parts[k])
			}
		case RoleLine:
			lines = append(lines, "line "+id(it.ID)+" "+paintName(it.Style.Fill))
			for k := range it.Parts {
				part := &it.Parts[k]
				if part.Paint == scene.StrokeOnly {
					path(it, "line", part.Path)
					continue
				}
				// The circle, as Path.Ellipse draws it: from its right end, four quarters.
				a := part.Path.Args
				cx, cy := a[6], a[1]
				lines = append(lines, "circle "+pt(cx, cy)+" "+pelesvg.Num(a[0]-cx)+" "+paintName(it.Style.Fill))
			}
		case RoleLegend:
			lines = append(lines, "legend "+pelesvg.ID(it.ID))
			part := &it.Parts[0]
			if part.Path != nil {
				path(it, "legend", part.Path)
			} else {
				rect(it, part)
			}
			if it.Label != nil {
				textAt(it)
			}
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}

// id is a series' title as Pele writes its data-id, "-" for none.
func id(title string) string {
	if title == "" {
		return "-"
	}
	return pelesvg.ID(title)
}
