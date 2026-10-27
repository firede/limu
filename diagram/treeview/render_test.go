package treeview

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// The fixture is what Pele's render draws of tree views, read back out of its SVG as one line
// per thing drawn, in the order drawn, with every number in full; ./scripts/oracle.sh writes
// it. The test writes the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		FontSize float64 `json:"fontSize"`
		Icons    bool    `json:"icons"`
		fixture.Drawing
	} `json:"cases"`
}

// hostIcons are the icons the host has in the cases drawn with icons, as the oracle's ICONS.
var hostIcons = []string{"logos:react", "mdi:home", "material:typescript", "mermaid-treeview:folder", "fa:fa-car", "p:x", "material:docker"}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	icons := func(name string) bool { return slices.Contains(hostIcons, name) }
	failed, tight := 0, 0
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
		o := &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize}
		if c.Icons {
			o.Icons = icons
		}
		sc := Layout(m, pre.Config, o)
		if c.MaxWidth > 0 && sc.Width < Layout(m, pre.Config, &layout.Options{FontSize: c.FontSize, Icons: o.Icons}).Width {
			tight++
		}
		got := sceneLines(t, sc)
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v, icons %v), %s", c.Src, c.MaxWidth, c.FontSize, c.Icons,
					msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if tight < 500 {
		t.Errorf("only %d laid out tightly", tight)
	}
}

// paintName is a paint as Pele's SVG names it, "-" for none given.
func paintName(p style.Paint) string {
	switch p.Token() {
	case style.Fg.Token():
		return "-"
	case style.Muted.Token():
		return "var(--_m)"
	case style.Border.Token():
		return "var(--_b)"
	case style.Accent.Token():
		return "var(--_c)"
	}
	if p == style.None {
		return "none"
	}
	return "?"
}

// clean is text as Pele's esc writes it, the control characters XML does not allow dropped.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, s)
}

// sceneLines writes a scene as the oracle reads Pele's SVG, every number as JavaScript's
// String writes it.
func sceneLines(t *testing.T, sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	box := func(r scene.Rect) string { return pt(r.X, r.Y) + "," + pt(r.W, r.H) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Title != "" {
		lines = append(lines, "acc-title "+pelesvg.ID(sc.Title))
	}
	if sc.Desc != "" {
		lines = append(lines, "acc-desc "+pelesvg.ID(sc.Desc))
	}
	// The class Pele writes a label of each role with, and its weight and style.
	looks := map[scene.Role][2]string{
		RoleTitle:       {"pele-title", "b -"},
		RoleDirectory:   {"pele-label", "b -"},
		RoleFile:        {"pele-label", "- -"},
		RoleDescription: {"pele-tree-description", "- i"},
	}
	label := func(it *scene.Item) {
		l := it.Label
		if l == nil {
			return
		}
		// Pele's labelSvg writes a label of any lines, its leftLabelSvg only the lines with
		// text.
		some := it.Role == RoleTitle && len(l.Lines) > 0
		for _, line := range l.Lines {
			for _, span := range line {
				some = some || span.Text != "" || span.Icon != ""
			}
		}
		if !some {
			return
		}
		anchor := map[scene.Align]string{scene.Middle: "middle", scene.Start: "start", scene.End: "end"}[it.Align]
		look := looks[it.Role]
		at := anchor + " " + look[0] + " " + paintName(it.Style.Text) + " " + look[1] + " " + pt(it.At.X, it.At.Y)
		if len(l.Lines) == 1 && len(l.Lines[0]) == 1 && l.Lines[0][0].Icon == "" {
			span := l.Lines[0][0]
			lines = append(lines, "label "+at+" | "+strconv.Itoa(int(span.Face))+":"+clean(span.Text))
			return
		}
		var spans, widths []string
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
		for _, w := range l.Widths {
			widths = append(widths, n(w))
		}
		lines = append(lines, "label "+at+" "+n(l.Size)+","+n(l.LineHeight)+","+n(l.Width)+","+n(l.Height)+
			" | "+strings.Join(widths, ",")+" | "+strings.Join(spans, " / "))
	}
	steps := func(p *scene.Path) string {
		var s []string
		p.Each(func(op scene.Op, a []float64) {
			step := string("MLQCZ"[op])
			for i := 0; i < len(a); i += 2 {
				step += " " + pt(a[i], a[i+1])
			}
			s = append(s, step)
		})
		return strings.Join(s, " ")
	}
	// glyph writes the icon Pele draws itself.
	glyph := func(it *scene.Item) {
		b := it.Box
		name := string(it.Shape)
		lines = append(lines, "icon "+name+" "+box(b))
		width := "?"
		if w, ok := it.Style.StrokeWidth.Get(); ok && w == float64(common.GlyphStroke*b.W)/24 {
			width = n(common.GlyphStroke)
		}
		want := glyphPaths[name].Scaled(b.W/24, b.X, b.Y)
		got := it.Parts[0].Path
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
		lines = append(lines, "glyph "+name+" "+paintName(it.Style.Stroke)+" "+width+" "+pelesvg.CapJoin(it)+" "+paintName(it.Style.Fill))
	}
	for i := 0; i < len(sc.Items); i++ {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle, RoleDescription:
			label(it)
		case RoleGuide:
			width, _ := it.Style.StrokeWidth.Get()
			lines = append(lines, "guide "+steps(it.Parts[0].Path)+" "+paintName(it.Style.Stroke)+" "+n(width)+" "+
				paintName(it.Style.Fill)+" "+pelesvg.CapJoin(it))
		case RoleDirectory, RoleFile:
			kind := "file"
			if it.Role == RoleDirectory {
				kind = "dir"
			}
			lines = append(lines, "node "+kind+" "+pelesvg.ID(it.ID)+" ["+strings.Join(it.Class, " ")+"]")
			for _, part := range it.Parts {
				r := "-"
				if part.Rounded {
					r = "r"
				}
				lines = append(lines, "rect "+box(part.Box)+" "+r+" "+paintName(it.Style.Fill)+"/"+
					strconv.FormatFloat(it.Style.FillOpacity.Or(1), 'g', -1, 32))
			}
			if pic := it.Picture; pic != nil {
				lines = append(lines, "icon "+pelesvg.ID(pic.Icon)+" "+box(pic.Box), "host")
			}
			if i+1 < len(sc.Items) && sc.Items[i+1].Role == RoleIcon {
				i++
				glyph(&sc.Items[i])
			}
			label(it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}

// TestEmpty lays out a tree view with no entries, as Pele draws it: the padding about
// nothing.
func TestEmpty(t *testing.T) {
	m, err := Parse("treeView-beta\n", "")
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, nil, nil)
	if sc.Width != 16 || sc.Height != 16 || len(sc.Items) != 0 {
		t.Errorf("got %v by %v with %d items", sc.Width, sc.Height, len(sc.Items))
	}
}
