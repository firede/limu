package gitgraph

import (
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

// The fixture is what Pele's render draws of git graphs, read back out of its SVG as one line
// per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. The oracle moves what
// Pele draws inside its moved group to where it is on the page, as the scene holds it, and
// turns a turned label's middle and the box behind it as limu does, before it writes numbers
// as Pele's SVG writes them. The test writes the same lines from the scene.

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
	if len(p.Cases) < 3000 {
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
		if i := fixture.FirstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize, i, lineAt(got, i), lineAt(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "(none)"
}

// paintName is a paint as the oracle names Pele's: a token of the theme by its name.
func paintName(p style.Paint) string {
	switch {
	case !p.Set():
		return "-"
	case p == style.None:
		return "none"
	case p.Token().Name() != "":
		return p.Token().Name()
	}
	return "?"
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	paint := func(it *scene.Item, part *scene.Part) string {
		fill, stroke := it.Style.Fill, it.Style.Stroke
		switch part.Paint {
		case scene.StrokeOnly:
			fill = style.None
		case scene.FillOnly:
			stroke = style.None
		case scene.Hollow:
			fill = style.Bg
		}
		if !stroke.Set() {
			stroke = style.None
		}
		s := "f=" + paintName(fill) + " s=" + paintName(stroke)
		if stroke.Visible() {
			s += " w=" + num(it.Style.StrokeWidth.Or(1))
			pen := it.Author.Over(it.Style)
			name := func(v string) string {
				if v == "" {
					return "-"
				}
				return v
			}
			s += " c=" + name(pen.Cap.String()) + " j=" + name(pen.Join.String())
		}
		dash := "-"
		if d := it.Style.Dash.Pattern(); len(d) > 0 {
			var parts []string
			for _, v := range d {
				parts = append(parts, num(v))
			}
			dash = strings.Join(parts, " ")
		}
		return s + " d=" + dash
	}
	// The subpaths of a path, as the oracle writes them; an ellipse, as Path.Ellipse draws
	// it, is a circle.
	shape := func(it *scene.Item, part *scene.Part) {
		if part.Path == nil {
			r := part.Box
			radius := "-"
			if part.Rounded {
				radius = "theme"
				if v, ok := it.Style.Radius.Get(); ok {
					radius = num(v)
				}
			}
			lines = append(lines, "rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" r="+radius+" "+paint(it, part))
			return
		}
		var ops []scene.Op
		var args [][]float64
		part.Path.Each(func(op scene.Op, a []float64) {
			ops = append(ops, op)
			args = append(args, append([]float64(nil), a...))
		})
		if len(ops) == 6 && ops[0] == scene.MoveTo && ops[1] == scene.CubeTo && ops[5] == scene.Close {
			cx, cy := args[1][4], args[0][1]
			lines = append(lines, "circle "+pt(cx, cy)+" "+num(args[0][0]-cx)+" "+paint(it, part))
			return
		}
		var s []string
		for k, op := range ops {
			w := string("MLQCZ"[op])
			for i := 0; i+1 < len(args[k]); i += 2 {
				w += " " + pt(args[k][i], args[k][i+1])
			}
			s = append(s, w)
		}
		lines = append(lines, "path "+strings.Join(s, " ")+" "+paint(it, part))
	}
	label := func(it *scene.Item) {
		l := it.Label
		if l == nil || len(l.Lines) == 0 {
			return
		}
		shift := float64(l.Size * 0.35)
		lines = append(lines, "t "+pt(it.At.X, it.At.Y-l.Height/2+l.LineHeight/2+shift)+" f="+paintName(it.Style.Text))
	}
	for i := 0; i < len(sc.Items); i++ {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			lines = append(lines, "title")
			label(it)
		case RoleLane:
			lines = append(lines, "branch "+peleID(it.ID))
			shape(it, &it.Parts[0])
		case RoleBranch:
			shape(it, &it.Parts[0])
			label(it)
		case RoleEdge:
			cls := "pele-edge"
			if it.Shape != "" {
				cls += " pele-edge-" + string(it.Shape)
			}
			lines = append(lines, "edge "+cls)
			shape(it, &it.Parts[0])
		case RoleCommit, RoleMerge:
			cls := "pele-commit pele-commit-" + string(it.Shape)
			if it.Role == RoleMerge && it.Shape != "merge" {
				cls += " pele-commit-merge"
			}
			lines = append(lines, "commit "+peleID(it.ID)+" "+cls)
			if it.Tooltip != "" {
				lines = append(lines, "tip "+peleID(it.Tooltip))
			}
			for k := range it.Parts {
				shape(it, &it.Parts[k])
			}
		case RoleCommitMark, RoleTagHole:
			for k := range it.Parts {
				shape(it, &it.Parts[k])
			}
		case RoleCommitLabel:
			lines = append(lines, "label")
			for k := range it.Parts {
				shape(it, &it.Parts[k])
			}
			if it.Rotate != 0 {
				lines = append(lines, "turned "+num(it.Rotate)+" "+pt(it.At.X, it.At.Y)+" f="+paintName(it.Style.Text))
			} else {
				label(it)
			}
		case RoleTag:
			lines = append(lines, "tag")
			shape(it, &it.Parts[0])
			// Pele draws the hole, an item of its own after the tag, before the tag's text.
			if i+1 < len(sc.Items) && sc.Items[i+1].Role == RoleTagHole {
				hole := &sc.Items[i+1]
				shape(hole, &hole.Parts[0])
				i++
			}
			label(it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}

// peleID is an id as Pele writes a data-id: entities decoded, characters XML does not allow
// dropped.
func peleID(id string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, text.DecodeEntities(id))
}
