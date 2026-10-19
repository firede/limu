package eventmodeling

import (
	"encoding/json"
	"fmt"
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
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of event modeling diagrams, read back out of its
// SVG as one line per thing drawn, in the order drawn, with every number in full;
// ./scripts/oracle.sh writes it. The test writes the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src                 string   `json:"src"`
		MaxWidth            float64  `json:"maxWidth"`
		FontSize            float64  `json:"fontSize"`
		AutoDirection       *bool    `json:"autoDirection"`
		DirectionBreakpoint float64  `json:"directionBreakpoint"`
		Lines               []string `json:"lines"`
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
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, down := 0, 0
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
		o := &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize, DirectionBreakpoint: c.DirectionBreakpoint}
		o.KeepDirection = c.AutoDirection != nil && !*c.AutoDirection
		sc := Layout(m, pre.Config, o)
		for _, it := range sc.Items {
			if it.Role == RoleLaneLabel {
				down++
				break
			}
		}
		got := sceneLines(sc)
		if i := pelesvg.FirstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v, auto %v, breakpoint %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize,
					c.AutoDirection, c.DirectionBreakpoint, i, pelesvg.At(got, i), pelesvg.At(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if down < 500 {
		t.Errorf("only %d laid out down", down)
	}
}

// paintName is a paint as Pele's SVG names it, "-" for none given.
func paintName(p style.Paint) string {
	if !p.Set() || p == style.Fg {
		return "-"
	}
	for k := range 8 {
		if p.Token() == style.Series(k).Token() {
			return fmt.Sprintf("var(--pele-series-%d,%s)", k+1, style.Default().Series[k])
		}
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
	case style.Border:
		return "var(--_b)"
	}
	return "?"
}

// classes are the classes Pele gives the text of an item of each role, "-" for none.
var classes = map[scene.Role]string{
	RoleTitle:     "pele-title",
	RoleLane:      "pele-cluster-label",
	RoleLaneLabel: "pele-cluster-label",
	RoleData:      "pele-em-data",
	RoleFrameID:   "pele-em-frame",
}

// clean is text as Pele's esc writes it, the control characters XML does not allow dropped.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, s)
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
	// lineText is the text of a line of a label, as Pele writes it, and whether it has any.
	lineText := func(line []text.Span) string {
		s := ""
		for _, span := range line {
			s += span.Text
		}
		return clean(s)
	}
	// look is what the oracle writes of a text after where it sits.
	look := func(it *scene.Item) string {
		l := it.Label
		class, ok := classes[it.Role]
		if !ok {
			class = "-"
			if it.Kind == scene.Node && it.Role != RoleNote && it.Role != RoleSpecification {
				class = "pele-label"
			}
		}
		anchor := map[scene.Align]string{scene.Middle: "middle", scene.Start: "start", scene.End: "end"}[it.Align]
		weight, family := "-", "-"
		if len(l.Lines) > 0 && len(l.Lines[0]) > 0 {
			if f := l.Lines[0][0].Face; f&text.Bold != 0 {
				weight = "b"
			}
			if f := l.Lines[0][0].Face; f&text.Mono != 0 {
				family = "mono"
			}
		}
		return " " + class + " %s " + anchor + " " + n(l.Size) + " " + paintName(it.Style.Text) + " " + weight + " " + family
	}
	// Where Pele's text puts a label: centered on its point as labelSvg writes it, its lines
	// starting at it as lines writes them, or a line of its own each, as leftLabelSvg does.
	label := func(it *scene.Item) {
		l := it.Label
		if l == nil || len(l.Lines) == 0 {
			return
		}
		y := it.At.Y - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
		format := look(it)
		switch {
		case it.Role == RoleLaneLabel:
			top := it.At.Y - l.Height/2
			for i, line := range l.Lines {
				if s := lineText(line); s != "" {
					at := pt(it.At.X, top+float64((float64(i)+0.5)*l.LineHeight)+float64(l.Size*0.35))
					lines = append(lines, "t"+fmt.Sprintf(format, at)+" | "+s)
				}
			}
		case it.Align == scene.Start:
			var texts []string
			first := -1
			for i, line := range l.Lines {
				if s := lineText(line); s != "" {
					if first < 0 {
						first = i
					}
					texts = append(texts, s)
				}
			}
			at := pt(it.At.X, y+float64(float64(first)*l.LineHeight))
			lines = append(lines, "t"+fmt.Sprintf(format, at)+" | "+strings.Join(texts, " | "))
		default:
			var texts []string
			for _, line := range l.Lines {
				s := lineText(line)
				if len(line) == 0 {
					s = " "
				}
				texts = append(texts, s)
			}
			lines = append(lines, "t"+fmt.Sprintf(format, pt(it.At.X, y))+" | "+strings.Join(texts, " | "))
		}
	}
	path := func(p *scene.Path) []string {
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
		return subpaths
	}
	rect := func(it *scene.Item, part *scene.Part) {
		r := part.Box
		radius := "-"
		if part.Rounded {
			radius = "r"
		}
		fill, opacity, stroke := it.Style.Fill, "-", paintName(it.Style.Stroke)
		switch part.Paint {
		case scene.Shade:
			fill, stroke = style.Bg, "-"
		case scene.FillOnly:
			stroke = "-"
		}
		if f := fill.Factor(); f < 1 {
			opacity = strconv.FormatFloat(float64(f), 'g', -1, 32)
		}
		lines = append(lines, "rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+radius+" "+paintName(fill)+" "+opacity+" "+stroke)
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle, RoleData, RoleFrameID, RoleNoteText, RoleSpecText:
			label(it)
		case RoleLaneLabel:
			lines = append(lines, "lane "+it.ID)
			label(it)
		case RoleLane:
			lines = append(lines, "lane "+it.ID)
			rect(it, &it.Parts[0])
			label(it)
		case RoleNote, RoleSpecification:
			lines = append(lines, strings.TrimPrefix(string(it.Role), "eventmodeling.")+" "+it.ID)
			rect(it, &it.Parts[0])
			label(it)
		case RoleRelation:
			lines = append(lines, "edge "+it.ID)
			for _, s := range path(it.Parts[0].Path) {
				lines = append(lines, "path "+s+" "+paintName(it.Style.Stroke)+" none "+pelesvg.CapJoin(it))
			}
			for _, m := range it.Ends {
				if outline := m.Outline(); outline != nil {
					lines = append(lines, "marker "+strings.Join(path(outline[0].Path), " ")+" "+paintName(it.Style.Stroke))
				}
			}
		case "eventmodeling.ui", "eventmodeling.command", "eventmodeling.event", "eventmodeling.readmodel", "eventmodeling.processor":
			s := "node " + it.ID + " " + strings.TrimPrefix(string(it.Role), "eventmodeling.")
			if len(it.Class) > 0 {
				s += " " + strings.Join(it.Class, " ")
			}
			lines = append(lines, s)
			for k := range it.Parts {
				rect(it, &it.Parts[k])
			}
			label(it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
