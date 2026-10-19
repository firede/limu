package usecase

import (
	"encoding/json"
	"fmt"
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

// The fixture is what Pele's render draws of use case diagrams, read back out of its SVG as
// one line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. Pele draws the
// inside of a node about the node's middle; the oracle moves it to where it is on the page, as
// the scene holds it, before it writes numbers as Pele's SVG writes them. The test writes the
// same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		MaxWidth float64  `json:"maxWidth"`
		FontSize float64  `json:"fontSize"`
		Lines    []string `json:"lines"`
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
	if len(p.Cases) < 3000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, skipped := 0, 0
	for _, c := range p.Cases {
		// A Go string cannot hold a surrogate with no partner, which JSON can write and
		// JavaScript measures as wide; the parser holds U+FFFD in its place.
		if loneSurrogate(c.Src) {
			skipped++
			continue
		}
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
		if i := firstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize, i, lineAt(got, i), lineAt(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	if skipped > 4 {
		t.Errorf("%d cases skipped", skipped)
	}
}

// loneSurrogate reports whether s has a JSON escape of a surrogate that is not one of a
// pair.
func loneSurrogate(s string) bool {
	unit := func(i int) (int, bool) {
		if i+6 > len(s) || s[i] != '\\' || s[i+1] != 'u' {
			return 0, false
		}
		v, err := strconv.ParseUint(s[i+2:i+6], 16, 16)
		return int(v), err == nil
	}
	for i := 0; i < len(s); i++ {
		u, ok := unit(i)
		if !ok || u < 0xd800 || u > 0xdfff {
			continue
		}
		if u <= 0xdbff {
			if low, ok := unit(i + 6); ok && low >= 0xdc00 && low <= 0xdfff {
				i += 11
				continue
			}
		}
		return true
	}
	return false
}

func firstDiff(a, b []string) int {
	for i := range max(len(a), len(b)) {
		if i >= len(a) || i >= len(b) || a[i] != b[i] {
			return i
		}
	}
	return -1
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "(none)"
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Bg:                    "var(--_bg)",
	style.Fg:                    "var(--_fg)",
	style.Muted:                 "var(--_m)",
	style.Line:                  "var(--_l)",
	style.Surface:               "var(--_s)",
	style.SurfaceAlt:            "var(--_a)",
	style.SurfaceAlt.Alpha(0.5): "var(--_a)/0.5",
	style.Border:                "var(--_b)",
	style.None:                  "none",
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

// partPaint is how a part of an item is painted, as the oracle reads Pele's fill and stroke,
// and when it has a stroke, its ends, and its joins where it has corners.
func partPaint(it *scene.Item, part *scene.Part, corners bool) string {
	name := paintName
	stroke := name(it.Style.Stroke)
	var s string
	switch part.Paint {
	case scene.StrokeOnly:
		s = "none " + stroke
	case scene.FillOnly:
		return name(it.Style.Fill) + " -"
	case scene.Hollow:
		s = "var(--_bg) " + stroke
	default:
		s = name(it.Style.Fill) + " " + stroke
	}
	if stroke != "-" && stroke != "none" {
		s += ends(it, 0, corners)
	}
	return s
}

// ends is how an item's strokes end and turn their corners, after its author's style, as the
// oracle reads them: " cap=" and " join=" where they are not SVG's initial butt and miter. A
// marker's own join, when it has one, is over the item's; without corners, there is no join.
func ends(it *scene.Item, join style.Join, corners bool) string {
	s := it.Author.Over(it.Style)
	if join.Set() {
		s.Join = join
	}
	if !corners {
		s.Join = 0
	}
	out := ""
	if s.Cap.Set() && s.Cap != style.CapButt {
		out += " cap=" + s.Cap.String()
	}
	if s.Join.Set() && s.Join != style.JoinMiter {
		out += " join=" + s.Join.String()
	}
	return out
}

// classes is the class attribute Pele writes on an item, after its pele- classes: the classes
// it gives a node or a boundary of its own, which the item's role and shape say, then those
// of the diagram's author. The class of an edge's animation is left out on both sides.
func classes(it *scene.Item) string {
	var names []string
	shape := string(it.Shape)
	business := strings.HasSuffix(shape, "-business")
	shape = strings.TrimSuffix(shape, "-business")
	switch it.Role {
	case "usecase.system":
		names = []string{"system-boundary", "system-boundary-" + shape}
	case "usecase.actor":
		kind := strings.TrimPrefix(strings.TrimPrefix(shape, "actor"), "-")
		if kind == "" {
			kind = "normal"
		}
		names = []string{"usecase-actor", "usecase-actor-" + kind}
	case "usecase.usecase":
		names = []string{"usecase-element", "usecase-" + shape}
	case "usecase.note":
		names = []string{"usecase-note"}
	case "usecase.json":
		names = []string{"usecase-json-table"}
	}
	if business {
		names = append(names, "usecase-business")
	}
	for _, c := range it.Class {
		if !strings.HasPrefix(c, "edge-animation-") {
			names = append(names, c)
		}
	}
	return strings.Join(names, " ")
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Where Pele's text puts the first line of a label centered on (cx, cy), and the paint
	// the item that holds it gives its text. Pele centers every text of a use case diagram.
	textAt := func(owner *scene.Item, l *text.Label, cx, cy float64) string {
		shift := float64(l.Size * 0.35)
		s := "t " + pt(cx, cy-l.Height/2+l.LineHeight/2+shift) + " " + paintName(owner.Style.Text)
		if owner.Align != scene.Middle {
			s += " not centered"
		}
		return s
	}
	label := func(owner *scene.Item, l *text.Label, cx, cy float64) {
		if l != nil && len(l.Lines) > 0 {
			lines = append(lines, textAt(owner, l, cx, cy))
		}
	}
	// The subpaths of a path, as the oracle writes them.
	subpaths := func(p *scene.Path) []string {
		var out []string
		var cur []string
		var ops []scene.Op
		var args [][]float64
		flush := func() {
			if len(cur) == 0 {
				return
			}
			// An ellipse, as Path.Ellipse draws it: from its right end, four quarters, closed.
			if len(ops) == 6 && ops[0] == scene.MoveTo && ops[5] == scene.Close &&
				ops[1] == scene.CubeTo && ops[2] == scene.CubeTo && ops[3] == scene.CubeTo && ops[4] == scene.CubeTo {
				cx, cy := args[1][4], args[0][1]
				out = append(out, "ellipse "+pt(cx, cy)+" "+pt(args[0][0]-cx, args[1][5]-cy))
			} else {
				out = append(out, strings.Join(cur, " "))
			}
			cur, ops, args = nil, nil, nil
		}
		p.Each(func(op scene.Op, a []float64) {
			if op == scene.MoveTo {
				flush()
			}
			s := string("MLQCZ"[op])
			for i := 0; i < len(a); i += 2 {
				s += " " + pt(a[i], a[i+1])
			}
			cur = append(cur, s)
			ops = append(ops, op)
			args = append(args, append([]float64(nil), a...))
		})
		flush()
		return out
	}
	shape := func(it *scene.Item, part *scene.Part) {
		if part.Path == nil {
			r := part.Box
			s := "rect " + pt(r.X, r.Y) + "," + pt(r.W, r.H)
			if part.Rounded {
				s += " r"
			}
			lines = append(lines, s+" "+partPaint(it, part, true))
			return
		}
		for _, s := range subpaths(part.Path) {
			// A circle and a single straight line have no corners.
			corners := !strings.HasPrefix(s, "ellipse") && !(strings.HasPrefix(s, "M ") && strings.Count(s, " ") == 3 && strings.Contains(s, " L "))
			if !strings.HasPrefix(s, "ellipse") {
				s = "path " + s
			}
			lines = append(lines, s+" "+partPaint(it, part, corners))
		}
	}
	for i := 0; i < len(sc.Items); i++ {
		it := &sc.Items[i]
		id := peleID(it.ID)
		switch it.Role {
		case "usecase.title":
			lines = append(lines, "title "+textAt(it, it.Label, it.At.X, it.At.Y))
		case "usecase.system":
			lines = append(lines, fmt.Sprintf("cluster %s [%s]", id, classes(it)))
			for k := range it.Parts {
				shape(it, &it.Parts[k])
			}
			label(it, it.Label, it.At.X, it.At.Y)
		case "usecase.relation", "usecase.note-link":
			dash := "-"
			if it.Style.Dash.Set() {
				var parts []string
				for _, v := range it.Style.Dash.Pattern() {
					parts = append(parts, num(v))
				}
				dash = strings.Join(parts, " ")
			}
			if (it.Role == "usecase.note-link") != (it.Shape == "note") {
				lines = append(lines, "role "+string(it.Role)+" of a line of shape "+string(it.Shape))
			}
			lines = append(lines, fmt.Sprintf("edge %s %s [%s] %s", id, it.Shape, classes(it), dash))
			for _, s := range subpaths(it.Parts[0].Path) {
				lines = append(lines, "path "+s+ends(it, 0, true))
			}
			for _, m := range it.Ends {
				if s := peleMarker(m); s != "" {
					if m.Type == "arrow_cross" || m.Type == "usecase.extension" {
						// The markers Pele strokes.
						s += ends(it, m.Type.Join(), true)
					}
					lines = append(lines, "m "+s)
				}
			}
		case "usecase.label":
			r := it.Box
			if it.Parts[0].Box != r {
				lines = append(lines, "patch not the label's box")
			}
			lines = append(lines, "label "+id+" "+pt(r.X, r.Y)+","+pt(r.W, r.H))
			label(it, it.Label, it.At.X, it.At.Y)
		case "usecase.actor", "usecase.usecase", "usecase.note", "usecase.json":
			// Where the node's middle is, which Pele's SVG moves what is in the node by, is
			// checked through where that is drawn.
			lines = append(lines, fmt.Sprintf("node %s [%s]", id, classes(it)))
			// What follows a node in the scene and is drawn inside it in Pele's SVG.
			var stereotype *scene.Item
			var cells []*scene.Item
			for i+1 < len(sc.Items) {
				next := &sc.Items[i+1]
				if next.Role == "usecase.stereotype" {
					stereotype = next
				} else if next.Role == "usecase.json-key" || next.Role == "usecase.json-value" {
					cells = append(cells, next)
				} else {
					break
				}
				i++
			}
			for k := range it.Parts {
				shape(it, &it.Parts[k])
			}
			if pic := it.Picture; pic != nil {
				r := pic.Box
				lines = append(lines, "icon "+peleID(pic.Icon)+" "+pt(r.X, r.Y)+","+pt(r.W, r.H))
			}
			if stereotype != nil {
				label(stereotype, stereotype.Label, stereotype.At.X, stereotype.At.Y)
			}
			label(it, it.Label, it.At.X, it.At.Y)
			for _, c := range cells {
				label(c, c.Label, c.At.X, c.At.Y)
			}
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

// peleMarker is the outline Pele's marker draws, as the oracle reads it.
func peleMarker(m scene.Marker) string {
	const arrow = scene.MarkerSize
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	x, y, dx, dy := m.At.X, m.At.Y, m.DX, m.DY
	nx, ny := -dy, dx
	switch m.Type {
	case "arrow_point":
		bx, by := x-float64(dx*arrow), y-float64(dy*arrow)
		half := float64(arrow * 0.42)
		return "M " + pt(x, y) + " L " + pt(bx+float64(nx*half), by+float64(ny*half)) +
			" L " + pt(bx-float64(nx*half), by-float64(ny*half)) + " Z"
	case "arrow_circle":
		return "o " + pt(x-float64(dx*4), y-float64(dy*4))
	case "arrow_cross":
		cx, cy := x-float64(dx*4), y-float64(dy*4)
		const a = 3.5
		ux, uy := float64((dx+nx)*a), float64((dy+ny)*a)
		vx, vy := float64((dx-nx)*a), float64((dy-ny)*a)
		return "M " + pt(cx-ux, cy-uy) + " L " + pt(cx+ux, cy+uy) + " M " + pt(cx-vx, cy-vy) + " L " + pt(cx+vx, cy+vy)
	case "usecase.extension":
		// The scene's own outline, which ports Pele's extension.
		var s []string
		for _, part := range m.Outline() {
			part.Path.Each(func(op scene.Op, a []float64) {
				w := string("MLQCZ"[op])
				for i := 0; i < len(a); i += 2 {
					w += " " + pt(a[i], a[i+1])
				}
				s = append(s, w)
			})
		}
		return strings.Join(s, " ")
	}
	return ""
}
