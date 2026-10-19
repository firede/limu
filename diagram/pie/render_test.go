package pie

import (
	"fmt"
	"slices"
	"strconv"
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

// The fixture is what Pele's render draws of pie charts, read back out of its SVG as one line
// per thing drawn, in the order drawn, with every number in full and moved to where the scene
// holds it; ./scripts/oracle.sh writes it. The test writes the same lines from the scene.

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
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, outside, empty := 0, 0, 0
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
		sc := Layout(m, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize, Padding: c.Padding})
		got := sceneLines(sc, c.Line)
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v, padding %v), %s", c.Src, c.MaxWidth, c.FontSize, (&layout.Options{Padding: c.Padding}).Pad(), msg)
			}
		}
		for _, it := range sc.Items {
			if it.Role == RoleSliceLabel && it.Style.Text == style.Fg {
				outside++
				break
			}
		}
		if len(sc.Items) > 0 && slices.ContainsFunc(sc.Items, func(it scene.Item) bool { return it.Role == RoleSlice && it.ID == "" }) {
			empty++
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if outside < 200 || empty < 50 {
		t.Errorf("only %d pies with percentages outside, %d with no slice", outside, empty)
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
	case style.Bg:
		return "var(--_bg)"
	case style.Border:
		return "var(--_b)"
	case style.None:
		return "none"
	}
	return "?"
}

// arc is an arc of Pele's path, as the oracle writes it: its radii, turn and flags, and
// where it ends.
type arc struct {
	rx, ry, rot  float64
	large, sweep bool
	to           scene.Point
	text         string
}

// arcs are the arcs of a line of Pele's, in order.
func arcs(line string) []arc {
	var out []arc
	fields := strings.Fields(line)
	for i, f := range fields {
		if f != "A" || i+2 >= len(fields) {
			continue
		}
		a := append(strings.Split(fields[i+1], ","), strings.Split(fields[i+2], ",")...)
		if len(a) != 7 {
			continue
		}
		var v [7]float64
		for k := range a {
			v[k], _ = strconv.ParseFloat(a[k], 64)
		}
		out = append(out, arc{v[0], v[1], v[2], v[3] == 1, v[4] == 1, scene.Point{X: v[5], Y: v[6]}, fields[i+1]})
	}
	return out
}

// sceneLines writes a scene as the oracle reads Pele's SVG, every number as JavaScript's
// String writes it. The curves an arc was drawn with are written as the arc of Pele's line
// at the same place, want(i), where ArcTo draws that arc with exactly those curves; elsewhere
// they are written as curves, which no line of Pele's has. want is the line Pele drew at an
// index, or "" where the fixture does not hold it; the oracle keeps the lines of paths and
// circles in full.
func sceneLines(sc *scene.Scene, want func(int) string) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Title != "" {
		lines = append(lines, "acc-title "+pelesvg.ID(sc.Title))
	}
	if sc.Desc != "" {
		lines = append(lines, "acc-desc "+pelesvg.ID(sc.Desc))
	}
	hint := func() string { return want(len(lines)) }
	contexts := map[scene.Role]string{RoleTitle: "title", RoleSliceLabel: "slice-labels", RoleLegend: "legend"}
	// Where Pele's text puts the first line of a label, with its fill, anchor and size and the
	// text of its lines: a percentage at its point, and any other as Pele's labelSvg centers
	// it.
	textAt := func(it *scene.Item) {
		l := it.Label
		x, y := it.At.X, it.At.Y
		if it.Role == RoleSliceLabel {
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
		lines = append(lines, contexts[it.Role]+" t "+pt(x, y)+" "+paintName(it.Style.Text)+" middle "+n(l.Size)+" | "+strings.Join(content, " | "))
	}
	pen := func(it *scene.Item) string {
		width := "-"
		if w, ok := it.Style.StrokeWidth.Get(); ok {
			width = n(w)
		}
		return paintName(it.Style.Stroke) + " " + width + " " + pelesvg.CapJoin(it)
	}
	path := func(p *scene.Path, hints []arc) string {
		var out []string
		var last scene.Point
		ops, args := p.Ops, p.Args
		at := 0
		for i := 0; i < len(ops); {
			op := ops[i]
			a := args[at : at+op.Args()]
			if op != scene.CubeTo {
				switch op {
				case scene.MoveTo, scene.LineTo:
					out = append(out, string("ML"[op])+" "+pt(a[0], a[1]))
					last = scene.Point{X: a[0], Y: a[1]}
				case scene.Close:
					out = append(out, "Z")
				default:
					out = append(out, "?")
				}
				i++
				at += op.Args()
				continue
			}
			// A run of curves: the arc of the hint at this place, if ArcTo draws it so.
			j, end := i, at
			for j < len(ops) && ops[j] == scene.CubeTo {
				end += ops[j].Args()
				j++
			}
			// The run may hold several arcs: each hint takes the curves up to the first that
			// ends where its arc does, if ArcTo draws the arc with exactly those.
			curves := args[at:end]
			for len(curves) > 0 {
				taken := 0
				if len(hints) > 0 {
					h := hints[0]
					hints = hints[1:]
					for k := 6; k <= len(curves); k += 6 {
						if curves[k-2] != h.to.X || curves[k-1] != h.to.Y {
							continue
						}
						q := new(scene.Path).MoveTo(last.X, last.Y).ArcTo(h.rx, h.ry, h.rot, h.large, h.sweep, h.to.X, h.to.Y)
						if slices.Equal(q.Args[2:], curves[:k]) && !slices.Contains(q.Ops[1:], scene.LineTo) {
							out = append(out, "A "+h.text+" "+pt(h.to.X, h.to.Y))
							taken = k
						}
						break
					}
				}
				if taken == 0 {
					s := "C"
					for k := 0; k < len(curves); k += 2 {
						s += " " + pt(curves[k], curves[k+1])
					}
					out = append(out, s)
					taken = len(curves)
				}
				last = scene.Point{X: curves[taken-2], Y: curves[taken-1]}
				curves = curves[taken:]
			}
			i, at = j, end
		}
		return strings.Join(out, " ")
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle, RoleSliceLabel:
			textAt(it)
		case RoleSlice:
			part := &it.Parts[0]
			if part.Paint == scene.StrokeOnly {
				// The circle of a pie with no slice, as Path.Ellipse draws it about the pie's
				// center, with the radius of Pele's circle where that draws exactly it.
				r := "?"
				if f := strings.Fields(hint()); len(f) > 2 && f[0] == "circle" {
					if v, err := strconv.ParseFloat(f[2], 64); err == nil &&
						slices.Equal(new(scene.Path).Ellipse(it.At.X, it.At.Y, v, v).Args, part.Path.Args) {
						r = n(v)
					}
				}
				lines = append(lines, "circle "+pt(it.At.X, it.At.Y)+" "+r+" none "+pen(it))
				continue
			}
			lines = append(lines, "slice "+pelesvg.ID(it.ID)+" "+paintName(it.Style.Fill)+" "+pen(it))
			lines = append(lines, "path "+path(part.Path, arcs(hint())))
		case RoleLegend:
			lines = append(lines, "legend "+pelesvg.ID(it.ID))
			r := it.Parts[0].Box
			lines = append(lines, "rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" r"+n(it.Style.Radius.Or(0))+" "+paintName(it.Style.Fill))
			if it.Label != nil {
				textAt(it)
			}
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
