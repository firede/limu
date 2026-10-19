package journey

import (
	"strconv"
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

// The fixture is what Pele's render draws of journey diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. Pele draws a face
// about its middle; the oracle moves it to where it is on the page, as the scene holds it,
// before it writes numbers as Pele's SVG writes them. The test writes the same lines from the
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
	if len(p.Cases) < 3000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, down := 0, 0
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
		got := sceneLines(Layout(m, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize}))
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), %s", c.Src, c.MaxWidth, c.FontSize, msg)
			}
		}
		for _, l := range got {
			if strings.HasPrefix(l, "step ") {
				down++
				break
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if down < 500 {
		t.Errorf("only %d cases laid out down", down)
	}
}

// paintName is a paint as Pele's SVG names it, "-" for none given.
func paintName(p style.Paint) string {
	if !p.Set() {
		return "-"
	}
	if name := p.Token().Name(); strings.HasPrefix(name, "series-") {
		return name
	}
	switch p {
	case style.Bg:
		return "var(--_bg)"
	case style.Muted:
		return "var(--_m)"
	case style.Line:
		return "var(--_l)"
	case style.Surface:
		return "var(--_s)"
	case style.SurfaceAlt:
		return "var(--_a)"
	case style.Border:
		return "var(--_b)"
	}
	return "?"
}

// partPaint is how a part of an item is painted, as the oracle reads Pele's fill and stroke.
func partPaint(it *scene.Item, part *scene.Part) string {
	switch part.Paint {
	case scene.StrokeOnly:
		return "none " + paintName(it.Style.Stroke)
	case scene.FillOnly:
		return paintName(it.Style.Fill) + " -"
	case scene.Hollow:
		return "var(--_bg) " + paintName(it.Style.Stroke)
	}
	return paintName(it.Style.Fill) + " " + paintName(it.Style.Stroke)
}

// pen is how an item's lines end and turn their corners, as the oracle reads Pele's
// stroke-linecap and stroke-linejoin: the author's or the diagram's, or "-" for neither.
func pen(it *scene.Item) string {
	s := it.Author.Over(it.Style)
	name := func(v string) string {
		if v == "" {
			return "-"
		}
		return v
	}
	return name(s.Cap.String()) + " " + name(s.Join.String())
}

// peleText is text as Pele writes it in its SVG: characters XML does not allow dropped.
func peleText(s string) string {
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
	down := true
	for _, it := range sc.Items {
		if it.Role == RoleGrid {
			down = false
		}
	}
	// Where Pele's text puts the first line of a label drawn about it, with its fill, its
	// anchor and the text of its lines.
	textAt := func(it *scene.Item) {
		l := it.Label
		if l == nil || len(l.Lines) == 0 {
			return
		}
		anchor := map[scene.Align]string{scene.Middle: "middle", scene.Start: "start", scene.End: "end"}[it.Align]
		var content []string
		for _, line := range l.Lines {
			s := ""
			for _, span := range line {
				s += span.Text
			}
			if len(line) == 0 {
				s = " "
			}
			content = append(content, peleText(s))
		}
		y := it.At.Y - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
		lines = append(lines, "t "+pt(it.At.X, y)+" "+paintName(it.Style.Text)+" "+anchor+" | "+strings.Join(content, " / "))
	}
	subpaths := func(p *scene.Path) []string {
		var out []string
		p.Each(func(op scene.Op, a []float64) {
			s := string("MLQCZ"[op])
			for i := 0; i < len(a); i += 2 {
				s += " " + pt(a[i], a[i+1])
			}
			if op == scene.MoveTo || len(out) == 0 {
				out = append(out, s)
			} else {
				out[len(out)-1] += " " + s
			}
		})
		return out
	}
	// ellipse is the ellipse Path.Ellipse draws, by its middle and radii.
	ellipse := func(p *scene.Path) string {
		cx, cy := p.Args[6], p.Args[1]
		return "ellipse " + pt(cx, cy) + " " + pt(p.Args[0]-cx, p.Args[7]-cy)
	}
	path := func(it *scene.Item, name string) {
		dash := "-"
		if it.Style.Dash.Set() {
			var parts []string
			for _, v := range it.Style.Dash.Pattern() {
				parts = append(parts, num(v))
			}
			dash = strings.Join(parts, " ")
		}
		lines = append(lines, name+" "+dash)
		for _, s := range subpaths(it.Parts[0].Path) {
			lines = append(lines, "path "+s+" "+partPaint(it, &it.Parts[0])+" "+pen(it))
		}
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
	step := 0
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle, RoleScale, RoleScore:
			textAt(it)
		case RoleGrid:
			path(it, "grid")
		case RoleStem:
			path(it, "stems")
		case RoleLine:
			path(it, "edge")
		case RoleFace:
			lines = append(lines, "face "+string(it.Shape))
			for k := range 3 {
				lines = append(lines, ellipse(it.Parts[k].Path)+" "+partPaint(it, &it.Parts[k]))
			}
			mouth := it.Parts[3].Path
			s := "path " + subpaths(mouth)[0]
			if it.Shape != "neutral" {
				sweep := "0"
				if it.Shape == "sad" {
					sweep = "1"
				}
				end := mouth.Last()
				s = "path M " + pt(mouth.Args[0], mouth.Args[1]) + " A 6,6,0,0," + sweep + " " + pt(end.X, end.Y)
			}
			lines = append(lines, s+" "+partPaint(it, &it.Parts[3])+" "+pen(it))
		case RoleSection:
			if it.Kind == scene.Group {
				lines = append(lines, "section "+peleText(text.DecodeEntities(it.ID)))
				box(it)
			}
			textAt(it)
		case RoleTask:
			if down {
				lines = append(lines, "step "+strconv.Itoa(step))
				step++
			} else {
				lines = append(lines, "task "+peleText(text.DecodeEntities(it.ID)))
			}
			box(it)
			textAt(it)
		case RoleActor, RoleLegend:
			if it.Role == RoleLegend {
				lines = append(lines, "legend "+peleText(text.DecodeEntities(it.ID)))
			}
			lines = append(lines, ellipse(it.Parts[0].Path)+" "+partPaint(it, &it.Parts[0]))
			if it.Tooltip != "" {
				lines = append(lines, "tip "+peleText(text.DecodeEntities(it.Tooltip)))
			}
			textAt(it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
