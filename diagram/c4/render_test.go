package c4

import (
	"fmt"
	"regexp"
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

// The fixture is what Pele's render draws of C4 diagrams, read back out of its SVG as one line
// per thing drawn, in the order drawn, with every number in full; ./scripts/oracle.sh writes
// it. The test writes the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		FontSize float64 `json:"fontSize"`
		fixture.Drawing
		Error map[string]any `json:"error"`
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
	failed := 0
	for k, c := range p.Cases {
		pre, err := diagram.Prepare(c.Src, diagram.Options{})
		if err != nil {
			t.Errorf("Prepare(%q): %v", c.Src, err)
			continue
		}
		db, err := ParsePrepared(pre)
		if err != nil {
			t.Errorf("ParsePrepared(%q): %v", c.Src, err)
			continue
		}
		sc, err := Layout(db, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize})
		switch {
		case err != nil && c.Error == nil:
			t.Errorf("Layout(%q): %v, Pele had none", c.Src, err)
			failed++
		case err != nil:
			if !compare(t, "Layout error", c.Src, dumpError(err), c.Error) {
				failed++
			}
		case c.Error != nil:
			t.Errorf("Layout(%q): no error, Pele %v", c.Src, c.Error)
			failed++
		default:
			got := sceneLines(sc)
			if msg := f.Diff(k, &c.Drawing, got, readCSS); msg != "" {
				if failed++; failed <= 20 {
					t.Errorf("Layout(%q, max %v, size %v), %s", c.Src, c.MaxWidth, c.FontSize, msg)
				}
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

var cssAt = regexp.MustCompile(`(?s) css=(.*)$`)

// readCSS writes the style attribute at the end of one of the oracle's lines as the author's
// colors limu reads of it.
func readCSS(line string) string {
	m := cssAt.FindStringSubmatchIndex(line)
	if m == nil {
		return line
	}
	decls := strings.Split(line[m[2]:m[3]], ";")
	return line[:m[0]] + " css=" + colors(style.Declarations(decls).Style.Fill, style.Declarations(decls).Style.Stroke)
}

// colors is an author's fill and stroke as the test writes them.
func colors(fill, stroke style.Paint) string {
	paint := func(p style.Paint) string {
		switch {
		case !p.Set():
			return "-"
		case p == style.None:
			return "none"
		}
		c := p.Own()
		return fmt.Sprintf("#%02x%02x%02x%02x/%v", c.R, c.G, c.B, c.A, p.Factor())
	}
	return "fill=" + paint(fill) + " stroke=" + paint(stroke)
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Bg:      "var(--_bg)",
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

func dash(st style.Style) string {
	if !st.Dash.Set() || len(st.Dash.Pattern()) == 0 {
		return "-"
	}
	var parts []string
	for _, v := range st.Dash.Pattern() {
		parts = append(parts, chars.NumberString(v))
	}
	return strings.Join(parts, " ")
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := chars.NumberString
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	box := func(r scene.Rect) string { return pt(r.X, r.Y) + "," + pt(r.W, r.H) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Desc != "" {
		lines = append(lines, "acc-desc "+sc.Desc)
	}
	// A label as the oracle writes one: where it is drawn about, its size, the size of its
	// type and its lines, and the paint of its text. A label whose lines start at the point
	// and hold no text Pele writes nothing of.
	label := func(it *scene.Item, extra string) {
		l := it.Label
		if l == nil {
			return
		}
		kind := "t"
		if it.Align == scene.Start {
			kind = "l"
			empty := true
			for _, line := range l.Lines {
				for _, s := range line {
					if s.Text != "" {
						empty = false
					}
				}
			}
			if empty {
				return
			}
		}
		var text []string
		for _, line := range l.Lines {
			var spans []string
			for _, s := range line {
				spans = append(spans, strconv.Itoa(int(s.Face))+":"+s.Text)
			}
			text = append(text, strings.Join(spans, "+"))
		}
		fill := paintName(it.Style.Text)
		if it.Style.Text == style.Fg {
			fill = "-"
		}
		lines = append(lines, fmt.Sprintf("%s %s %s %s | %s %s%s css=%s", kind, pt(it.At.X, it.At.Y), pt(l.Width, l.Height),
			num(l.Size), strings.Join(text, " | "), extra, fill, colors(it.Author.Text, style.Paint{})))
	}
	capJoin := func(it *scene.Item) string { return pelesvg.CapJoin(it) }
	for i := 0; i < len(sc.Items); i++ {
		it := &sc.Items[i]
		id := pelesvg.ID(it.ID)
		switch it.Role {
		case RoleTitle:
			label(it, "title ")
		case RoleBoundary, RoleDeployment:
			kind := "boundary"
			if it.Role == RoleDeployment {
				kind = "node"
			}
			lines = append(lines, "cluster "+kind+" "+id)
			r := it.Parts[0]
			rounded := ""
			if r.Rounded {
				rounded = " r"
			}
			lines = append(lines, fmt.Sprintf("rect %s%s %s %s %s css=%s", box(r.Box), rounded, paintName(it.Style.Fill),
				paintName(it.Style.Stroke), dash(it.Style), colors(it.Author.Fill, it.Author.Stroke)))
			linked := false
			for i+1 < len(sc.Items) && sc.Items[i+1].ID == it.ID {
				next := &sc.Items[i+1]
				if next.Role != RoleBoundaryLabel && next.Role != RoleBoundaryType && next.Role != RoleBoundaryDescr {
					break
				}
				if next.Link != "" && !linked {
					lines = append(lines, "link "+next.Link)
					linked = true
				}
				label(next, "")
				i++
			}
		case RoleRel:
			lines = append(lines, fmt.Sprintf("edge %s %s %s %s css=%s", id, paintName(it.Style.Fill), paintName(it.Style.Stroke),
				capJoin(it), colors(style.Paint{}, it.Author.Stroke)))
			var path []string
			it.Parts[0].Path.Each(func(op scene.Op, a []float64) {
				s := string("MLQCZ"[op])
				for k := 0; k < len(a); k += 2 {
					s += " " + pt(a[k], a[k+1])
				}
				path = append(path, s)
			})
			lines = append(lines, "path "+strings.Join(path, " "))
			for _, m := range it.Ends {
				if m.Type != "" {
					lines = append(lines, fmt.Sprintf("m %s %s %s", m.Type, pt(m.At.X, m.At.Y), pt(m.DX, m.DY)))
				}
			}
		case RoleElement:
			classes := "pele-shape-" + string(it.Shape) + " pele-c4-" + it.Tags[0]
			if len(it.Tags) > 1 {
				classes += " pele-c4-external"
			}
			// The middle, which Pele moves the shape's inside by, is checked through the box
			// about it and through where the inside is drawn.
			lines = append(lines, fmt.Sprintf("node %s %s box %s", id, classes, box(it.Box)))
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			for _, part := range it.Parts {
				fill, author := paintName(it.Style.Fill), colors(it.Author.Fill, it.Author.Stroke)
				if part.Paint == scene.StrokeOnly {
					fill, author = "none", colors(style.Paint{}, it.Author.Stroke)
				}
				what := "shape"
				if part.Path == nil && part.Rounded {
					what = "rect " + box(part.Box)
				}
				lines = append(lines, fmt.Sprintf("part %s %s %s %s %s css=%s", what, fill, paintName(it.Style.Stroke), dash(it.Style), capJoin(it), author))
			}
			// The texts that follow the shape in the scene, in the order Pele writes them:
			// the type, the name, the technology, the description.
			var typ, techn, descr *scene.Item
			for i+1 < len(sc.Items) && sc.Items[i+1].ID == it.ID {
				next := &sc.Items[i+1]
				switch next.Role {
				case RoleElementType:
					typ = next
				case RoleElementTechn:
					techn = next
				case RoleElementDescr:
					descr = next
				default:
					next = nil
				}
				if next == nil {
					break
				}
				i++
			}
			for _, x := range []*scene.Item{typ, it, techn, descr} {
				if x != nil {
					label(x, "")
				}
			}
		case RoleRelLabel:
			lines = append(lines, "label "+id)
			if it.Tooltip != "" {
				lines = append(lines, "tip "+it.Tooltip)
			}
			lines = append(lines, fmt.Sprintf("patch %s %d %s - -", box(it.Box), patchRadius, paintName(it.Style.Fill)))
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			label(it, "")
			if i+1 < len(sc.Items) && sc.Items[i+1].Role == RoleRelTechn && sc.Items[i+1].ID == it.ID {
				i++
				label(&sc.Items[i], "")
			}
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
