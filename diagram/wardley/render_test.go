package wardley

import (
	"encoding/json"
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

// The fixture is what Pele's render draws of Wardley maps, read back out of its SVG as one
// line per thing drawn, in the order drawn, with its numbers written out in full;
// ./scripts/oracle.sh writes it. The test writes the same lines from the scene, its numbers
// as JavaScript writes them, so that a difference in the last bit shows.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		MaxWidth float64  `json:"maxWidth"`
		FontSize float64  `json:"fontSize"`
		Icons    bool     `json:"icons"`
		Lines    []string `json:"lines"`
	} `json:"cases"`
}

// hostIcons are the icons the oracle's host has, in the cases drawn with icons.
var hostIcons = map[string]bool{"star": true, "fa:fa-car": true, "fa:fa-coffee": true}

func TestLayoutMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/render.json")
	if err != nil {
		t.Fatal(err)
	}
	var p peleRender
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, nodes := 0, 0
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
		nodes += len(m.Nodes)
		o := &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize}
		if c.Icons {
			o.Icons = func(name string) bool { return hostIcons[name] }
		}
		got := sceneLines(Layout(m, pre.Config, o))
		if i := pelesvg.FirstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v, icons %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize, c.Icons, i,
					pelesvg.At(got, i), pelesvg.At(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if nodes < 10000 {
		t.Errorf("only %d nodes drawn", nodes)
	}
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Bg:         "var(--_bg)",
	style.Fg:         "var(--_fg)",
	style.Muted:      "var(--_m)",
	style.Line:       "var(--_l)",
	style.Surface:    "var(--_s)",
	style.SurfaceAlt: "var(--_a)",
	style.Border:     "var(--_b)",
	style.Accent:     "var(--_c)",
	style.None:       "none",
}

func paintName(p style.Paint) string {
	if s, ok := paintNames[p]; ok {
		return s
	}
	return "?"
}

// The classes Pele gives the text of each role, and the weight of headings it draws some in.
var textClasses = map[scene.Role]string{
	RoleTitle:      "pele-title",
	RoleStageLabel: "pele-wardley-stage-label",
	RoleAxisLabel:  "pele-wardley-axis-label",
	RoleLinkLabel:  "pele-edge-label",
	RoleLabel:      "pele-label",
	RoleNote:       "pele-wardley-note",
}

var headings = map[scene.Role]bool{RoleAxisLabel: true, RoleNote: true, RoleAnnotation: true, RoleAccelerator: true, RoleDeaccelerator: true}

// The classes of Pele's groups of each role of node.
var nodeClasses = map[scene.Role]string{
	RoleAnchor:            "pele-wardley-anchor",
	RoleComponent:         "pele-wardley-component",
	RolePipelineParent:    "pele-wardley-pipeline-parent",
	RolePipelineComponent: "pele-wardley-pipeline-component",
}

var faces = [...]string{"-", "b", "i", "bi"}

// sceneLines writes a scene as the oracle reads Pele's SVG.
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
	dash := func(it *scene.Item) string {
		if !it.Style.Dash.Set() {
			return "-"
		}
		var s []string
		for _, v := range it.Style.Dash.Pattern() {
			s = append(s, n(v))
		}
		return strings.Join(s, " ")
	}
	// What a part is filled and stroked with, its dash, cap and join.
	pen := func(it *scene.Item, paint scene.Painting) string {
		fill, stroke := paintName(it.Style.Fill), paintName(it.Style.Stroke)
		switch paint {
		case scene.StrokeOnly:
			fill = "none"
		case scene.FillOnly:
			stroke = "-"
		case scene.Hollow:
			fill = paintName(style.Bg)
		}
		return fill + " " + stroke + " " + dash(it) + " " + pelesvg.CapJoin(it)
	}
	part := func(it *scene.Item, p scene.Part) {
		if p.Path == nil {
			r := "-"
			if p.Rounded {
				r = "r"
			}
			lines = append(lines, "rect "+pt(p.Box.X, p.Box.Y)+","+pt(p.Box.W, p.Box.H)+" "+r+" "+pen(it, p.Paint))
			return
		}
		var ops []scene.Op
		var args [][]float64
		p.Path.Each(func(op scene.Op, a []float64) {
			ops = append(ops, op)
			args = append(args, a)
		})
		if len(ops) == 6 && ops[1] == scene.CubeTo {
			// A circle, as Path.Ellipse draws it: from its right end, four quarters about
			// its middle, the first to its bottom.
			cx, cy := args[1][4], args[0][1]
			lines = append(lines, "circle "+pt(cx, cy)+" "+pt(args[0][0], args[1][5])+" "+pen(it, p.Paint))
			return
		}
		var s []string
		for k, op := range ops {
			switch op {
			case scene.MoveTo:
				s = append(s, "M "+pt(args[k][0], args[k][1]))
			case scene.LineTo:
				s = append(s, "L "+pt(args[k][0], args[k][1]))
			case scene.Close:
				s = append(s, "Z")
			default:
				s = append(s, "?")
			}
		}
		lines = append(lines, "path "+strings.Join(s, " ")+" "+pen(it, p.Paint))
	}
	// A label as the oracle writes one: where it is drawn about, its class, size, fill and
	// weight as Pele's attributes give them, its measures, and its spans.
	label := func(it *scene.Item, class, weight string) {
		l := it.Label
		if l == nil || l.Empty() {
			return
		}
		kind := "label"
		if it.Align == scene.Start {
			kind = "start"
			// Pele writes nothing of a label whose lines start at a point and hold only
			// icons, which it leaves out of such a label.
			text := false
			for _, line := range l.Lines {
				for _, s := range line {
					text = text || s.Icon == ""
				}
			}
			if !text {
				return
			}
		}
		size := n(l.Size)
		if it.Role == RoleTitle {
			size = "-"
		}
		fill := "-"
		if it.Style.Text.Set() && it.Style.Text != style.Fg {
			fill = paintName(it.Style.Text)
		}
		turn := ""
		if it.Rotate != 0 {
			turn = " turn " + n(it.Rotate)
		}
		var spans []string
		for _, line := range l.Lines {
			var s []string
			for _, span := range line {
				if span.Icon != "" {
					s = append(s, "i:"+span.Icon)
				} else {
					s = append(s, "t:"+faces[span.Face&3]+":"+span.Text)
				}
			}
			spans = append(spans, strings.Join(s, " + "))
		}
		lines = append(lines, kind+" "+class+" "+size+" "+fill+" "+weight+turn+" "+pt(it.At.X, it.At.Y)+" "+pt(l.Width, l.Height)+" "+
			n(l.Size)+" "+n(l.LineHeight)+" | "+strings.Join(spans, " | "))
	}
	var node scene.Role
	for i := range sc.Items {
		it := &sc.Items[i]
		weight := "-"
		if headings[it.Role] || it.Role == RoleLabel && node == RoleAnchor {
			weight = "var(--_hw)"
		}
		class := textClasses[it.Role]
		if class == "" {
			class = "-"
		}
		switch it.Role {
		case RoleTitle:
			label(it, class, "var(--_tw)")
		case RoleStage, RoleGrid, RoleAxis, RoleTrend:
			part(it, it.Parts[0])
		case RolePipelineLine:
			// Pele leaves the fill of the line unset, which a straight line does not show.
			s := len(lines)
			part(it, it.Parts[0])
			lines[s] = strings.Replace(lines[s], " none ", " - ", 1)
		case RoleStageLabel, RoleAxisLabel, RoleLinkLabel, RoleLabel, RoleNote, RoleAnnotationText:
			label(it, class, weight)
		case RolePipeline:
			lines = append(lines, "group pele-wardley-pipeline "+pelesvg.ID(it.ID))
			part(it, it.Parts[0])
		case RoleLink:
			cls := "pele-edge"
			if it.Style.Dash.Set() {
				cls += " pele-wardley-dashed"
			}
			lines = append(lines, "group "+cls+" "+pelesvg.ID(it.ID))
			part(it, it.Parts[0])
		case RoleTrendTarget, RoleInertia:
			part(it, it.Parts[0])
		case RoleAnchor, RoleComponent, RolePipelineParent, RolePipelineComponent:
			node = it.Role
			lines = append(lines, "group pele-node "+nodeClasses[it.Role]+" "+pelesvg.ID(it.ID))
			for _, p := range it.Parts {
				part(it, p)
			}
		case RoleAnnotation:
			lines = append(lines, "group pele-wardley-annotation "+pelesvg.ID(it.ID))
			part(it, it.Parts[0])
			label(it, class, weight)
		case RoleAnnotations:
			lines = append(lines, "group pele-wardley-annotations -")
			part(it, it.Parts[0])
		case RoleAccelerator, RoleDeaccelerator:
			lines = append(lines, "group pele-wardley-"+strings.TrimPrefix(string(it.Role), "wardley.")+" "+pelesvg.ID(it.ID))
			part(it, it.Parts[0])
			label(it, class, weight)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
		// Pele draws the marker at the end of a line first.
		if it.Kind == scene.Edge {
			for _, k := range [2]int{1, 0} {
				if m := it.Ends[k]; m.Type != "" {
					lines = append(lines, "marker "+string(m.Type)+" "+pt(m.At.X, m.At.Y)+" "+pt(m.DX, m.DY)+" "+paintName(it.Style.Stroke))
				}
			}
		}
	}
	return lines
}
