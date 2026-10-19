package cynefin

import (
	"encoding/json"
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

// The fixture is what Pele's render draws of Cynefin diagrams, read back out of its SVG as
// one line per thing drawn, in the order drawn, with its numbers written out in full;
// ./scripts/oracle.sh writes it. The test writes the same lines from the scene, its numbers
// as JavaScript writes them, so that a difference in the last bit shows. Pele's path data,
// arcs and all, is read with scene.ParsePath and written out again as the scene's paths are.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		FontSize float64  `json:"fontSize"`
		Icons    bool     `json:"icons"`
		Lines    []string `json:"lines"`
	} `json:"cases"`
	Randoms [][2]float64 `json:"randoms"`
	Hashes  [][2]any     `json:"hashes"`
}

func loadRender(t *testing.T) *peleRender {
	t.Helper()
	data, err := fixture.Read("testdata/pele/render.json")
	if err != nil {
		t.Fatal(err)
	}
	var p peleRender
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return &p
}

// hostIcons are the icons the oracle's host has, in the cases drawn with icons.
var hostIcons = map[string]bool{"star": true, "fa:fa-car": true, "fa:fa-coffee": true}

func TestLayoutMatchesPele(t *testing.T) {
	p := loadRender(t)
	if len(p.Cases) < 1800 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, items, transitions := 0, 0, 0
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
		o := &layout.Options{FontSize: c.FontSize}
		if c.Icons {
			o.Icons = func(name string) bool { return hostIcons[name] }
		}
		sc := Layout(m, pre.Config, o)
		for _, it := range sc.Items {
			switch it.Role {
			case RoleItem, RoleOverflow:
				items++
			case RoleTransition:
				transitions++
			}
		}
		got := sceneLines(sc)
		want := peleLines(t, c.Lines)
		if i := pelesvg.FirstDiff(got, want); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, size %v, icons %v), line %d\n got %s\nPele %s", c.Src, c.FontSize, c.Icons, i,
					pelesvg.At(got, i), pelesvg.At(want, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if items < 8000 || transitions < 1800 {
		t.Errorf("only %d items and %d transitions drawn", items, transitions)
	}
}

func TestSeedsMatchPele(t *testing.T) {
	p := loadRender(t)
	if len(p.Randoms) < 2000 || len(p.Hashes) < 10 {
		t.Fatalf("only %d seeds and %d strings", len(p.Randoms), len(p.Hashes))
	}
	for _, r := range p.Randoms {
		if got := seededRandom(r[0]); got != r[1] {
			t.Errorf("seededRandom(%v) = %v, Pele %v", r[0], got, r[1])
		}
	}
	for _, h := range p.Hashes {
		s, want := h[0].(string), int32(h[1].(float64))
		if got := hashString(s); got != want {
			t.Errorf("hashString(%q) = %d, Pele %d", s, got, want)
		}
	}
}

// pathData writes a path as the test compares paths: each step's letter and its numbers in
// full.
func pathData(p *scene.Path) string {
	var b strings.Builder
	p.Each(func(op scene.Op, a []float64) {
		b.WriteByte("MLQCZ"[op])
		for i := 0; i < len(a); i += 2 {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(chars.NumberString(a[i]) + "," + chars.NumberString(a[i+1]))
		}
	})
	return b.String()
}

// peleLines are the fixture's lines with the data of each path read as the scene would draw
// it.
func peleLines(t *testing.T, lines []string) []string {
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = line
		if !strings.HasPrefix(line, "path ") {
			continue
		}
		// path, class, id, the data, and the six fields of the pen.
		f := strings.Split(line, " ")
		d := strings.Join(f[3:len(f)-6], " ")
		p, err := scene.ParsePath(d)
		if err != nil {
			t.Fatalf("path %q: %v", d, err)
		}
		out[i] = strings.Join(append(append(f[:3:3], pathData(p)), f[len(f)-6:]...), " ")
	}
	return out
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Bg:     "var(--_bg)",
	style.Fg:     "var(--_fg)",
	style.Muted:  "var(--_m)",
	style.Line:   "var(--_l)",
	style.Border: "var(--_b)",
	style.None:   "none",
}

// paintName is a paint as the oracle writes it: a series by its number, and its opacity
// after an @.
func paintName(p style.Paint) string {
	if s, ok := paintNames[p]; ok {
		return s
	}
	for k := range 8 {
		for _, a := range []float64{1, 0.14, 0.18} {
			if p == style.Series(k).Alpha(a) {
				s := "series-" + strconv.Itoa(k+1)
				if a != 1 {
					s += "@" + chars.NumberString(a)
				}
				return s
			}
		}
	}
	return "?"
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
	// What a part is filled and stroked with, the stroke's width, its dash, cap and join.
	pen := func(it *scene.Item, paint scene.Painting) string {
		fill, stroke := paintName(pelesvg.OwnFill(it)), paintName(it.Style.Stroke)
		switch paint {
		case scene.StrokeOnly:
			fill = "none"
		case scene.FillOnly:
			stroke = "-"
		}
		width := "-"
		if w, ok := it.Style.StrokeWidth.Get(); ok {
			width = n(w)
		}
		dash := "-"
		if it.Style.Dash.Set() {
			var s []string
			for _, v := range it.Style.Dash.Pattern() {
				s = append(s, n(v))
			}
			dash = strings.Join(s, ",")
			if dash == "" {
				dash = "none"
			}
		}
		return fill + " " + stroke + " " + width + " " + dash + " " + pelesvg.CapJoin(it)
	}
	part := func(it *scene.Item, class, id string) {
		p := it.Parts[0]
		if p.Path == nil {
			r := "-"
			if p.Rounded {
				r = "r"
			}
			lines = append(lines, "rect "+pt(p.Box.X, p.Box.Y)+","+pt(p.Box.W, p.Box.H)+" "+r+" "+pen(it, p.Paint))
			return
		}
		lines = append(lines, "path "+class+" "+id+" "+pathData(p.Path)+" "+pen(it, p.Paint))
	}
	// A label as the oracle writes one: its class, size, fill, weight and style as Pele's
	// attributes give them, where it is centered, its measures, and its lines and spans.
	// The size is Pele's font-size, which it sets on all text but titles.
	label := func(it *scene.Item, class string, sized bool, weight, face string) {
		l := it.Label
		if l == nil || l.Empty() {
			return
		}
		size := "-"
		if sized {
			size = n(l.Size)
		}
		fill := "-"
		if it.Style.Text.Set() {
			fill = paintName(it.Style.Text)
		}
		var spans []string
		for i, line := range l.Lines {
			var s []string
			for _, span := range line {
				if span.Icon != "" {
					s = append(s, "i:"+span.Icon+" "+n(span.Width))
				} else {
					s = append(s, "t:"+faces[span.Face&3]+":"+span.Text+" "+n(span.Width))
				}
			}
			spans = append(spans, n(l.Widths[i])+": "+strings.Join(s, " + "))
		}
		lines = append(lines, "label "+class+" "+size+" "+fill+" "+weight+" "+face+" "+pt(it.At.X, it.At.Y)+" "+pt(l.Width, l.Height)+" "+
			n(l.Size)+" "+n(l.LineHeight)+" | "+strings.Join(spans, " | "))
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			label(it, "pele-title", false, "var(--_tw)", "-")
		case RoleRegion:
			part(it, "pele-cynefin-region", pelesvg.ID(it.ID))
		case RoleBoundary, RoleBackdrop:
			part(it, "-", "-")
		case RoleCliff:
			part(it, "pele-cynefin-cliff", "-")
		case RoleDomain:
			lines = append(lines, "group pele-cynefin-domain "+pelesvg.ID(it.ID))
			label(it, "pele-label", false, "var(--_hw)", "-")
		case RoleDescription:
			label(it, "pele-cynefin-subtitle", true, "-", "italic")
		case RoleItem:
			lines = append(lines, "group pele-node "+pelesvg.ID(it.ID))
			part(it, "", "")
			label(it, "pele-label", true, "-", "-")
		case RoleOverflow:
			lines = append(lines, "group pele-node pele-cynefin-overflow -")
			part(it, "", "")
			label(it, "pele-label", true, "-", "-")
		case RoleTransition:
			lines = append(lines, "group pele-edge "+pelesvg.ID(it.ID))
			part(it, "-", "-")
			m := it.Ends[1]
			lines = append(lines, "marker "+string(m.Type)+" "+pt(m.At.X, m.At.Y)+" "+pt(m.DX, m.DY)+" "+paintName(it.Style.Stroke))
		case RoleTransitionLabel:
			label(it, "pele-edge-label", true, "-", "-")
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
