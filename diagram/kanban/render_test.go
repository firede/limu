package kanban

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

// The fixture is what Pele's render draws of kanban boards, read back out of its SVG as one
// line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. Pele draws a
// column's insides about the column and a card's about the card; the oracle moves them to
// where they are on the page, as the scene holds them, before it writes numbers as Pele's SVG
// writes them. Each line of a column's or a card's text is written as where it starts or ends,
// and what is in it. The test writes the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		MaxWidth float64  `json:"maxWidth"`
		FontSize float64  `json:"fontSize"`
		Icons    bool     `json:"icons"`
		Lines    []string `json:"lines"`
	} `json:"cases"`
}

// hostIcons are the icons the host has in the cases drawn with icons, as the oracle's ICONS.
var hostIcons = []string{"star", "inbox", "phone", "fa:fa-car", "fa fa-x"}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	if err := fixture.Load("testdata/pele/render.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 3500 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	icons := func(name string) bool { return slices.Contains(hostIcons, name) }
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
		o := &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize}
		if c.Icons {
			o.Icons = icons
		}
		got := sceneLines(Layout(m, pre.Config, o))
		if i := pelesvg.FirstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v, icons %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize, c.Icons, i, pelesvg.At(got, i), pelesvg.At(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Bg:       "var(--_bg)",
	style.Muted:    "var(--_m)",
	style.Surface:  "var(--_s)",
	style.Border:   "var(--_b)",
	style.Accent:   "var(--_c)",
	style.Critical: "var(--pele-critical,#e45756)",
}

// paintName is a paint as Pele's SVG names it, "-" for none given; Pele gives text in the
// theme's color of text no fill of its own.
func paintName(p style.Paint) string {
	if !p.Set() || p == style.Fg {
		return "-"
	}
	if s, ok := paintNames[p]; ok {
		return s
	}
	return "?"
}

// textClasses are the classes Pele gives the text of an item of each role.
var textClasses = map[scene.Role]string{
	RoleColumn:   "pele-cluster-label",
	RoleCard:     "pele-label",
	RoleTicket:   "pele-ticket",
	RoleAssigned: "pele-assigned",
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	box := func(r scene.Rect) string { return pt(r.X, r.Y) + "," + pt(r.W, r.H) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// The lines of a label as textBlock writes them: each starting or ending at the item's
	// At, the first one's box starting where the label's does.
	block := func(it *scene.Item) {
		l := it.Label
		if l == nil {
			return
		}
		anchor := "start"
		if it.Align == scene.End {
			anchor = "end"
		}
		top := it.At.Y - l.Height/2
		for i, line := range l.Lines {
			var spans []string
			for _, s := range line {
				if s.Icon != "" {
					spans = append(spans, "i:"+s.Icon+":"+chars.NumberString(s.Width))
				} else {
					spans = append(spans, "t:"+strconv.Itoa(int(s.Face))+":"+s.Text+":"+chars.NumberString(s.Width))
				}
			}
			cy := top + float64((float64(i)+0.5)*l.LineHeight)
			lines = append(lines, fmt.Sprintf("line %s %s %s %s %s %s", textClasses[it.Role], paintName(it.Style.Text),
				pt(it.At.X, cy), anchor, chars.NumberString(l.Widths[i]), strings.Join(spans, "|")))
		}
	}
	rect := func(it *scene.Item) {
		part := it.Parts[0]
		s := "rect " + box(part.Box)
		if part.Rounded {
			s += " r"
		}
		stroke := "-"
		if part.Paint != scene.FillOnly {
			stroke = paintName(it.Style.Stroke)
		}
		lines = append(lines, s+" "+paintName(it.Style.Fill)+" "+stroke)
	}
	icon := func(it *scene.Item) {
		if pic := it.Picture; pic != nil {
			lines = append(lines, "icon "+pic.Icon+" "+box(pic.Box))
		}
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			l := it.Label
			lines = append(lines, "title "+pt(it.At.X, it.At.Y-l.Height/2+l.LineHeight/2+float64(l.Size*0.35)))
		case RoleColumn:
			lines = append(lines, fmt.Sprintf("column %s [%s]", pelesvg.ID(it.ID), strings.Join(it.Class, " ")))
			rect(it)
			block(it)
		case RoleCount:
			l := it.Label
			lines = append(lines, fmt.Sprintf("count %s %s %s", pt(it.At.X, it.At.Y-l.Height/2+l.LineHeight/2+float64(l.Size*0.35)),
				paintName(it.Style.Text), num(l.Size)))
			// Pele draws a column's icon after its count.
			icon(&sc.Items[i-1])
		case RoleCard:
			classes := it.Class
			// Pele marks a card of a priority with a class, which the stripe after it says.
			if i+1 < len(sc.Items) && sc.Items[i+1].Role == RolePriority {
				classes = append([]string{"pele-priority-" + string(sc.Items[i+1].Shape)}, classes...)
			}
			lines = append(lines, fmt.Sprintf("card %s [%s]", pelesvg.ID(it.ID), strings.Join(classes, " ")))
			rect(it)
			// Pele draws the stripe of a priority before the card's text.
			if i+1 < len(sc.Items) && sc.Items[i+1].Role == RolePriority {
				stripe(&lines, &sc.Items[i+1])
			}
			block(it)
			icon(it)
		case RolePriority:
		case RoleTicket:
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			block(it)
		case RoleAssigned:
			block(it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}

// stripe writes the stripe of a card's priority as the oracle reads Pele's.
func stripe(lines *[]string, it *scene.Item) {
	num := pelesvg.Num
	var pts []string
	it.Parts[0].Path.Each(func(op scene.Op, a []float64) {
		pts = append(pts, string("MLQCZ"[op])+" "+num(a[0])+","+num(a[1]))
	})
	opacity := "-"
	if v, ok := it.Style.Opacity.Get(); ok {
		opacity = chars.NumberString(v)
	}
	s := it.Author.Over(it.Style)
	pen := func(v string) string {
		if v == "" {
			return "-"
		}
		return v
	}
	*lines = append(*lines, fmt.Sprintf("priority %s %s %s %s %s %s", strings.Join(pts, " "), paintName(it.Style.Stroke), opacity,
		chars.NumberString(it.Style.StrokeWidth.Or(1)), pen(s.Cap.String()), pen(s.Join.String())))
}
