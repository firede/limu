package ishikawa

import (
	"encoding/json"
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

// The fixture is what Pele's render draws of Ishikawa diagrams, read back out of its SVG as
// one line per thing drawn, in the order drawn, with every number in full; ./scripts/oracle.sh
// writes it. Pele draws the body of a diagram drawn across about a point of its own; the
// oracle moves it to where it is on the page, as the scene holds it. Each label is written as
// the point it is centered on and what is in it. The test writes the same lines from the
// scene. The outline of the head Pele writes with an arc; the test draws it again with
// scene.Path's ArcTo from Pele's corners, radii and flags, and compares that path with the
// item's.

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
var hostIcons = []string{"fa:fa-car", "fa:fa-user", "fab:fa-github"}

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
		got := sceneLines(Layout(m, pre.Config, o))
		want := make([]string, len(c.Lines))
		for i, line := range c.Lines {
			want[i] = peleLine(line)
		}
		if i := pelesvg.FirstDiff(got, want); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v, icons %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize, c.Icons, i,
					pelesvg.At(got, i), pelesvg.At(want, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// peleLine is a line of the fixture, with the outline of the head drawn again as limu draws
// arcs.
func peleLine(line string) string {
	rest, ok := strings.CutPrefix(line, "outline ")
	if !ok {
		return line
	}
	f := strings.Fields(rest)
	p := new(scene.Path)
	point := func(s string) (float64, float64) {
		x, y, _ := strings.Cut(s, ",")
		return number(x), number(y)
	}
	i := 0
	for ; i < len(f) && f[i] != "var(--_s)"; i++ {
		switch f[i] {
		case "M":
			p.MoveTo(point(f[i+1]))
			i++
		case "L":
			p.LineTo(point(f[i+1]))
			i++
		case "A":
			a := strings.Split(f[i+1], ",")
			x, y := point(f[i+2])
			p.ArcTo(number(a[0]), number(a[1]), number(a[2]), a[3] == "1", a[4] == "1", x, y)
			i += 2
		case "Z":
			p.Close()
		default:
			return line
		}
	}
	return "outline " + pathLine(p) + " " + strings.Join(f[i:], " ")
}

func number(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return v
}

// pathLine is a path's commands, every number in full.
func pathLine(p *scene.Path) string {
	n := chars.NumberString
	var out []string
	p.Each(func(op scene.Op, a []float64) {
		s := string("MLQCZ"[op])
		for k := 0; k < len(a); k += 2 {
			s += " " + n(a[k]) + "," + n(a[k+1])
		}
		out = append(out, s)
	})
	return strings.Join(out, " ")
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Fg:      "var(--_fg)",
	style.Muted:   "var(--_m)",
	style.Line:    "var(--_l)",
	style.Surface: "var(--_s)",
	style.Border:  "var(--_b)",
	style.None:    "none",
}

// paintName is a paint as Pele's SVG names it, "-" for none given.
func paintName(p style.Paint) string {
	if !p.Set() {
		return "-"
	}
	if s, ok := paintNames[p]; ok {
		return s
	}
	return "?"
}

// sceneLines writes a scene as the oracle reads Pele's SVG, every number as JavaScript's
// String writes it.
func sceneLines(sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Drawn across, a diagram has a spine, and the weight of headings is the head's only.
	across := slices.ContainsFunc(sc.Items, func(it scene.Item) bool { return it.Role == RoleSpine })
	label := func(it *scene.Item) {
		l := it.Label
		if l == nil || l.Empty() {
			return
		}
		fill := "-"
		if it.Style.Text != style.Fg {
			fill = paintName(it.Style.Text)
		}
		weight := "-"
		switch {
		case it.Role == RoleTitle:
			weight = "var(--_tw)"
		case it.Role == RoleHead, it.Role == RoleCategory && !across:
			weight = "var(--_hw)"
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
					s = append(s, strconv.Itoa(int(span.Face))+":"+span.Text+":"+n(span.Width))
				}
			}
			spans = append(spans, strings.Join(s, " "))
		}
		lines = append(lines, "label "+fill+" "+weight+" "+n(l.Size)+" "+pt(it.At.X, it.At.Y)+" "+
			n(l.Size)+","+n(l.LineHeight)+","+n(l.Width)+","+n(l.Height)+" | "+strings.Join(widths, ",")+" | "+strings.Join(spans, " | "))
	}
	paint := func(it *scene.Item, part *scene.Part) string {
		fill, stroke := it.Style.Fill, it.Style.Stroke
		switch part.Paint {
		case scene.StrokeOnly:
			fill = style.None
		case scene.FillOnly:
			stroke = style.None
		}
		return paintName(fill) + " " + paintName(stroke) + " " + pelesvg.CapJoin(it)
	}
	path := func(kind string, it *scene.Item) {
		lines = append(lines, kind+" "+pathLine(it.Parts[0].Path)+" "+paint(it, &it.Parts[0]))
		for _, m := range it.Ends {
			if m.Type != "" {
				lines = append(lines, "marker "+string(m.Type)+" "+pt(m.At.X, m.At.Y)+" "+pt(m.DX, m.DY)+" "+paintName(it.Style.Stroke))
			}
		}
	}
	rect := func(it *scene.Item) {
		part := &it.Parts[0]
		r := part.Box
		if part.Path != nil {
			// A box with corners of its own, a path in the item's box.
			r = it.Box
		}
		radius := "?"
		switch {
		case part.Path == nil && part.Rounded:
			radius = "r"
		case part.Path != nil:
			for _, rx := range []float64{4, 12} {
				if want := new(scene.Path).RoundRect(r, rx); slices.Equal(want.Ops, part.Path.Ops) && slices.Equal(want.Args, part.Path.Args) {
					radius = "r" + n(rx)
				}
			}
		}
		lines = append(lines, "rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+radius+" "+paint(it, part))
	}
	kinds := map[scene.Role]string{RoleHead: "head", RoleCategory: "category", RoleCause: "cause"}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			label(it)
		case RoleSpine:
			path("spine", it)
		case RoleBones:
			lines = append(lines, "category "+pelesvg.ID(it.ID))
			path("bones", it)
		case RoleLink:
			path("link", it)
		case RoleCategory, RoleHead, RoleCause:
			switch {
			case !across:
				lines = append(lines, "node "+kinds[it.Role]+" "+pelesvg.ID(it.ID))
				if len(it.Parts) > 0 {
					rect(it)
				}
			case it.Role == RoleHead:
				lines = append(lines, "head "+pelesvg.ID(it.ID))
				lines = append(lines, "outline "+pathLine(it.Parts[0].Path)+" "+paint(it, &it.Parts[0]))
			case it.Role == RoleCategory:
				rect(it)
			}
			label(it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
