package architecture

import (
	"encoding/json"
	"math"
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

// The fixture is what Pele's render draws of architecture diagrams, read back out of its SVG
// as one line per thing drawn, in the order drawn, with every number in full; ./scripts/oracle.sh
// writes it. Pele draws the body of a diagram about a point of its own and the inside of a
// node about the node; the oracle moves them to where they are on the page, as the scene holds
// them. Each label is written as the point it is centered on and what is in it. The test
// writes the same lines from the scene.

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
var hostIcons = []string{"logos:aws-lambda", "mdi:home", "star", "database", "fa:fa-car"}

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
		got := sceneLines(t, Layout(m, pre.Config, o))
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
	style.Fg:                    "var(--_fg)",
	style.Bg:                    "var(--_bg)",
	style.Muted:                 "var(--_m)",
	style.Line:                  "var(--_l)",
	style.Surface:               "var(--_s)",
	style.SurfaceAlt.Alpha(0.5): "var(--_a)/0.5",
	style.Border:                "var(--_b)",
}

// paintName is a paint as Pele's SVG names it, "-" for none given.
func paintName(p style.Paint) string {
	if !p.Set() || p == style.None {
		return "-"
	}
	if s, ok := paintNames[p]; ok {
		return s
	}
	return "?"
}

// textClasses are what the oracle calls the labels of each role, by the class Pele gives them.
var textClasses = map[scene.Role]string{
	"architecture.title":      "title",
	"architecture.group":      "group",
	"architecture.service":    "service",
	"architecture.icon-text":  "icon-text",
	"architecture.edge-label": "edge",
}

// sceneLines writes a scene as the oracle reads Pele's SVG, every number as JavaScript's
// String writes it.
func sceneLines(t *testing.T, sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Title != "" {
		lines = append(lines, "acc-title "+pelesvg.ID(sc.Title))
	}
	if sc.Desc != "" {
		lines = append(lines, "acc-desc "+pelesvg.ID(sc.Desc))
	}
	label := func(it *scene.Item) {
		l := it.Label
		if l == nil {
			return
		}
		fill := "-"
		if it.Style.Text != style.Fg {
			fill = paintName(it.Style.Text)
		}
		var widths, spans []string
		for _, w := range l.Widths {
			widths = append(widths, n(w))
		}
		for _, line := range l.Lines {
			var s []string
			for _, span := range line {
				if span.Icon != "" {
					s = append(s, "i:"+span.Icon+":"+n(span.Width))
				} else {
					s = append(s, "t:"+strconv.Itoa(int(span.Face))+":"+span.Text+":"+n(span.Width))
				}
			}
			spans = append(spans, strings.Join(s, "|"))
		}
		lines = append(lines, "label "+textClasses[it.Role]+" "+fill+" "+pt(it.At.X, it.At.Y)+" "+
			n(l.Size)+","+n(l.LineHeight)+","+n(l.Width)+","+n(l.Height)+" | "+strings.Join(widths, ",")+" | "+strings.Join(spans, " / "))
	}
	rect := func(it *scene.Item, part *scene.Part) {
		r := part.Box
		if part.Path != nil {
			// A label's patch, a path with the item's box.
			r = it.Box
		}
		radius := "-"
		switch {
		case part.Rounded:
			radius = "r"
		case part.Path != nil:
			radius = "r3"
		}
		stroke := paintName(it.Style.Stroke)
		lines = append(lines, "rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+radius+" "+paintName(pelesvg.OwnFill(it))+" "+stroke+" "+pelesvg.CapJoin(it))
	}
	// The icon of an item: a Picture the host draws, or Pele's glyph, the item after it.
	icon := func(owner *scene.Item, next *scene.Item) bool {
		if pic := owner.Picture; pic != nil {
			b := pic.Box
			lines = append(lines, "icon "+pelesvg.ID(pic.Icon)+" "+pt(b.X, b.Y)+","+pt(b.W, b.H), "host")
		}
		if next == nil || next.Role != "architecture.icon" {
			return false
		}
		b := next.Box
		name := string(next.Shape)
		lines = append(lines, "icon "+name+" "+pt(b.X, b.Y)+","+pt(b.W, b.H))
		width := "?"
		if w, ok := next.Style.StrokeWidth.Get(); ok && w == float64(glyphStroke*b.W)/24 {
			width = n(glyphStroke)
		}
		want, _ := scene.ParsePath(glyphs[name])
		want = want.Scaled(b.W/24, b.X, b.Y)
		got := next.Parts[0].Path
		if !slices.Equal(got.Ops, want.Ops) || len(got.Args) != len(want.Args) {
			t.Errorf("glyph %s: steps differ", name)
		} else {
			for k := range got.Args {
				if math.Abs(got.Args[k]-want.Args[k]) > 1e-9 {
					t.Errorf("glyph %s: %v, not %v", name, got.Args, want.Args)
					break
				}
			}
		}
		fill := "?"
		if next.Style.Fill == style.None {
			fill = "none"
		}
		lines = append(lines, "glyph "+glyphs[name]+" "+paintName(next.Style.Stroke)+" "+width+" "+pelesvg.CapJoin(next)+" "+fill)
		return true
	}
	for i := 0; i < len(sc.Items); i++ {
		it := &sc.Items[i]
		var next *scene.Item
		if i+1 < len(sc.Items) {
			next = &sc.Items[i+1]
		}
		switch it.Role {
		case "architecture.title":
			label(it)
		case "architecture.group":
			lines = append(lines, "group "+pelesvg.ID(it.ID))
			rect(it, &it.Parts[0])
			if icon(it, next) {
				i++
			}
			label(it)
		case "architecture.edge":
			lines = append(lines, "edge "+pelesvg.ID(it.ID))
			var steps []string
			it.Parts[0].Path.Each(func(op scene.Op, a []float64) {
				s := string("MLQCZ"[op])
				for k := 0; k < len(a); k += 2 {
					s += " " + pt(a[k], a[k+1])
				}
				steps = append(steps, s)
			})
			lines = append(lines, "path "+strings.Join(steps, " ")+" "+pelesvg.CapJoin(it))
			for _, m := range it.Ends {
				if m.Type != "" {
					lines = append(lines, "marker "+string(m.Type)+" "+pt(m.At.X, m.At.Y)+" "+pt(m.DX, m.DY)+" "+paintName(it.Style.Stroke))
				}
			}
		case "architecture.edge-label":
			lines = append(lines, "edge-label "+pelesvg.ID(it.ID))
			rect(it, &it.Parts[0])
			label(it)
		case "architecture.service":
			lines = append(lines, "service "+pelesvg.ID(it.ID))
			rect(it, &it.Parts[0])
			if icon(it, next) {
				i++
			} else if next != nil && next.Role == "architecture.icon-text" {
				label(next)
				i++
			}
			label(it)
		case "architecture.junction":
			lines = append(lines, "junction "+pelesvg.ID(it.ID))
			// The circle, as Path.Ellipse draws it: from its right end, four quarters.
			a := it.Parts[0].Path.Args
			cx, cy := a[6], a[1]
			lines = append(lines, "circle "+pt(cx, cy)+" "+pelesvg.Num(a[0]-cx)+" "+paintName(pelesvg.OwnFill(it)))
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
