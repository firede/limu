package agentflow

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/draw"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of agent flows, read back out of its SVG as one
// line per thing drawn, in the order drawn, with every number of its geometry; the
// agentflow-render suite of ./scripts/oracle.sh writes it. The test writes the same lines
// from the scene, numbers as Pele's SVG writes them.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		MaxWidth float64  `json:"maxWidth"`
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
	if len(p.Cases) < 2000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, narrow := 0, 0
	for _, c := range p.Cases {
		if c.MaxWidth > 0 {
			narrow++
		}
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
		got := sceneLines(Layout(db, pre.Config, &layout.Options{MaxWidth: c.MaxWidth}))
		if i := firstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, i, at(got, i), at(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
	t.Logf("%d cases, %d of them narrow", len(p.Cases), narrow)
}

func firstDiff(a, b []string) int {
	for i := range max(len(a), len(b)) {
		if i >= len(a) || i >= len(b) || a[i] != b[i] {
			return i
		}
	}
	return -1
}

func at(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "(none)"
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	box := func(r scene.Rect, cx, cy float64) string {
		return pt(r.X-cx, r.Y-cy) + "," + num(r.W) + "," + num(r.H)
	}
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Where Pele's text puts the first line of a label centered on (cx, cy).
	textAt := func(l *text.Label, cx, cy float64) string {
		shift := float64(l.Size * 0.35)
		return "t " + pt(cx, cy-l.Height/2+l.LineHeight/2+shift)
	}
	label := func(l *text.Label, cx, cy float64) {
		if l != nil && len(l.Lines) > 0 {
			lines = append(lines, textAt(l, cx, cy))
		}
	}
	for i := 0; i < len(sc.Items); i++ {
		it := &sc.Items[i]
		id := peleID(it.ID)
		classes := strings.Join(it.Class, " ")
		kind, _ := strings.CutPrefix(string(it.Role), "agentflow.")
		switch kind {
		case "title":
			if it.Label != nil && len(it.Label.Lines) > 0 {
				lines = append(lines, "title "+textAt(it.Label, it.At.X, it.At.Y))
			}
		case "flow":
			lines = append(lines, fmt.Sprintf("flow %s [%s] %s %s", id, classes, box(it.Box, 0, 0), slotOf(it)))
			label(it.Label, it.At.X, it.At.Y)
		case "sequence", "reference", "failure":
			flags := ""
			if it.Style.Dash.Set() {
				flags = " dotted"
			}
			lines = append(lines, fmt.Sprintf("edge %s [%s]%s %s", id, strings.TrimSpace("pele-edge-"+kind+" "+classes), flags, pelesvg.D(it.Parts[0].Path)))
			for _, m := range it.Ends {
				if s := peleMarker(m); s != "" {
					lines = append(lines, "m "+s)
				}
			}
		case "edge-label":
			lines = append(lines, "label "+id+" "+box(it.Box, 0, 0))
			label(it.Label, it.At.X, it.At.Y)
		case "collapsed":
			cx, cy := it.Box.X+it.Box.W/2, it.Box.Y+it.Box.H/2
			lines = append(lines, fmt.Sprintf("node %s collapsed [%s] translate(%s) %s", id, classes, pt(cx, cy), slotOf(it)))
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			lines = append(lines, "rect "+box(it.Box, cx, cy)+" ")
			if len(it.Parts) != 1 || it.Parts[0].Box != it.Box || !it.Parts[0].Rounded {
				lines = append(lines, "parts differ from the box")
			}
			if i+2 >= len(sc.Items) || sc.Items[i+1].Role != "agentflow.collapsed-rule" || sc.Items[i+2].Role != "agentflow.collapsed-dots" {
				lines = append(lines, "no rule and dots")
			} else {
				d := ""
				sc.Items[i+1].Parts[0].Path.Each(func(op scene.Op, a []float64) {
					if op == scene.MoveTo {
						d += "M" + pt(a[0]-cx, a[1]-cy)
					} else {
						d += "H" + num(a[0]-cx)
					}
				})
				lines = append(lines, "path "+d+" dashed")
				d = ""
				sc.Items[i+2].Parts[0].Path.Each(func(op scene.Op, a []float64) {
					if op == scene.MoveTo {
						d += "M" + pt(a[0]-cx, a[1]-cy) + "a1.75,1.75 0 1 0 3.5,0a1.75,1.75 0 1 0 -3.5,0Z"
					}
				})
				lines = append(lines, "path "+d)
				i += 2
			}
			label(it.Label, it.At.X-cx, it.At.Y-cy)
		case "tool", "task", "decision", "input", "refdoc", "connector", "action":
			cx, cy := it.Box.X+it.Box.W/2, it.Box.Y+it.Box.H/2
			lines = append(lines, fmt.Sprintf("node %s %s [%s] translate(%s) %s", id, it.Shape, strings.TrimSpace("pele-kind-"+kind+" "+classes), pt(cx, cy), slotOf(it)))
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			lines = append(lines, shapeLines(string(it.Shape), it.Box.W, it.Box.H)...)
			if partsOf(it.Parts) != partsOf(draw.Outline(string(it.Shape), cx, cy, it.Box.W, it.Box.H)) {
				lines = append(lines, "parts differ from the outline")
			}
			label(it.Label, it.At.X-cx, it.At.Y-cy)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}

// slotOf is the palette slot an item is drawn in, as the oracle reads it: the series color
// of its outline.
func slotOf(it *scene.Item) string {
	for n := range 8 {
		if it.Style.Stroke == style.Series(n) {
			return fmt.Sprintf("s%d", n)
		}
	}
	return "no slot"
}

// partsOf writes the parts of an outline, numbers as Pele writes them: the center of a node,
// taken from its box, may be off the one it was drawn about in the last bit.
func partsOf(parts []scene.Part) string {
	var b strings.Builder
	for _, p := range parts {
		r := p.Box
		fmt.Fprintf(&b, "%s,%s,%s,%s %v %d", pelesvg.Num(r.X), pelesvg.Num(r.Y), pelesvg.Num(r.W), pelesvg.Num(r.H), p.Rounded, p.Paint)
		if p.Path != nil {
			b.WriteString(" " + pelesvg.D(p.Path))
		}
		b.WriteString("; ")
	}
	return b.String()
}

// shapeLines are the elements Pele's drawShape writes for the shapes agent flows draw, as
// the oracle reads them.
func shapeLines(shape string, w, h float64) []string {
	num := pelesvg.Num
	rect := func(rx string) string {
		return "rect " + num(-w/2) + "," + num(-h/2) + "," + num(w) + "," + num(h) + " " + rx
	}
	poly := func(xy ...float64) string {
		s := "polygon "
		for i := 0; i < len(xy); i += 2 {
			if i > 0 {
				s += " "
			}
			s += num(xy[i]) + "," + num(xy[i+1])
		}
		return s
	}
	switch shape {
	case "rect", "connector":
		return []string{rect("")}
	case "rounded":
		return []string{rect(num(min(12, h/2)))}
	case "fr-rect":
		x := w/2 - 8
		return []string{rect(""), "path M" + num(-x) + "," + num(-h/2) + "V" + num(h/2) + "M" + num(x) + "," + num(-h/2) + "V" + num(h/2)}
	case "diam":
		return []string{poly(0, -h/2, w/2, 0, 0, h/2, -w/2, 0)}
	case "hex":
		m := h / 4
		return []string{poly(-w/2+m, -h/2, w/2-m, -h/2, w/2, 0, w/2-m, h/2, -w/2+m, h/2, -w/2, 0)}
	case "lean-r":
		s := h / 4
		return []string{poly(-w/2+s, -h/2, w/2, -h/2, w/2-s, h/2, -w/2, h/2)}
	case "lin-doc":
		x, top, base, q := w/2, -h/2, h/2-10, w/4
		return []string{
			"path M" + num(-x) + "," + num(top) + "H" + num(x) + "V" + num(base) + "q" + num(-q) + ",-12 " + num(-2*q) + ",0t" + num(-2*q) + ",0Z",
			"path M" + num(-w/2+8) + "," + num(-h/2) + "V" + num(h/2-6),
		}
	}
	return []string{"shape " + shape + " is not one agent flows draw"}
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
	x, y, dx, dy := m.At.X, m.At.Y, m.DX, m.DY
	nx, ny := -dy, dx
	switch m.Type {
	case "arrow_point":
		bx, by := x-float64(dx*arrow), y-float64(dy*arrow)
		half := float64(arrow * 0.42)
		return "M" + num(x) + "," + num(y) + "L" + num(bx+float64(nx*half)) + "," + num(by+float64(ny*half)) +
			"L" + num(bx-float64(nx*half)) + "," + num(by-float64(ny*half)) + "Z"
	case "arrow_circle":
		return "o " + num(x-float64(dx*4)) + "," + num(y-float64(dy*4))
	case "arrow_cross":
		cx, cy := x-float64(dx*4), y-float64(dy*4)
		const a = 3.5
		ux, uy := float64((dx+nx)*a), float64((dy+ny)*a)
		vx, vy := float64((dx-nx)*a), float64((dy-ny)*a)
		return "M" + num(cx-ux) + "," + num(cy-uy) + "L" + num(cx+ux) + "," + num(cy+uy) +
			"M" + num(cx-vx) + "," + num(cy-vy) + "L" + num(cx+vx) + "," + num(cy+vy)
	}
	return ""
}
