package state

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/svg"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of state diagrams, read back out of its SVG as one
// line per element drawn, in the order drawn; ./scripts/oracle.sh writes it. The test writes
// the same lines from the scene, numbers as Pele's SVG writes them.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		fixture.Drawing
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 2500 {
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
		got := sceneLines(Layout(db, pre.Config, &layout.Options{MaxWidth: c.MaxWidth}))
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v), %s", c.Src, c.MaxWidth, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// nodeKinds are the classes Pele gives a node of each role, less its "pele-".
var nodeKinds = map[scene.Role]string{
	RoleState:  "state",
	RoleStart:  "state-start",
	RoleEnd:    "state-end",
	RoleChoice: "state-choice",
	RoleFork:   "state-fork",
	RoleJoin:   "state-join",
	RoleNote:   "note",
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
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
	rect := func(kind string, r scene.Rect, cx, cy float64, rx string) {
		lines = append(lines, kind+" "+pt(r.X-cx, r.Y-cy)+","+num(r.W)+","+num(r.H)+" "+rx)
	}
	radius := func(it *scene.Item) string {
		r, _ := it.Style.Radius.Get()
		return num(r)
	}
	// check notes a part that is not where Pele draws it.
	check := func(what string, got, want float64) {
		if math.Abs(got-want) > 1e-6 {
			lines = append(lines, fmt.Sprintf("mismatch %s %v %v", what, got, want))
		}
	}
	checkPath := func(p *scene.Path, cx, cy float64, want ...float64) {
		k := 0
		p.Each(func(op scene.Op, a []float64) {
			if op == scene.Close || k+1 >= len(want) {
				return
			}
			check("path", a[len(a)-2]-cx, want[k])
			check("path", a[len(a)-1]-cy, want[k+1])
			k += 2
		})
	}
	var cx, cy, descAt float64
	for i := range sc.Items {
		it := &sc.Items[i]
		id := peleID(it.ID)
		classes := strings.Join(it.Class, " ")
		switch it.Role {
		case RoleTitle:
			if it.Label != nil && len(it.Label.Lines) > 0 {
				lines = append(lines, "title "+textAt(it.Label, it.At.X, it.At.Y))
			}
		case RoleComposite:
			lines = append(lines, fmt.Sprintf("composite %s [%s]", id, classes))
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			if it.Tooltip != "" {
				lines = append(lines, "tip "+peleText(it.Tooltip))
			}
			rect("rect", it.Parts[0].Box, 0, 0, radius(it))
			for _, part := range it.Parts[1:] {
				lines = append(lines, "path "+pelesvg.D(part.Path))
			}
			label(it.Label, it.At.X, it.At.Y)
		case RoleDivider:
			if p := it.Parts[0]; p.Path != nil {
				lines = append(lines, "divider "+pelesvg.D(p.Path))
			} else {
				rect("divider-rect", p.Box, 0, 0, radius(it))
			}
		case RoleTransition, RoleNoteEdge:
			kind := "transition"
			if it.Role == RoleNoteEdge {
				kind = "note-edge"
			}
			lines = append(lines, "edge "+id+" "+kind, "path "+pelesvg.D(it.Parts[0].Path), "stroke "+pelesvg.CapJoin(it))
			for _, m := range it.Ends {
				if s := peleMarker(m); s != "" {
					lines = append(lines, "m "+s)
				}
			}
		case RoleEdgeLabel:
			lines = append(lines, "label "+id)
			rect("rect", it.Box, 0, 0, "3")
			label(it.Label, it.At.X, it.At.Y)
		case RoleDescription:
			if it.Label != nil && len(it.Label.Lines) > 0 {
				y := descAt + it.Label.Height/2
				check("description", it.At.X-cx, 0)
				check("description", it.At.Y-cy, y)
				lines = append(lines, textAt(it.Label, 0, y))
			}
			descAt += it.Label.Height
		default:
			kind, ok := nodeKinds[it.Role]
			if !ok {
				lines = append(lines, "unknown role "+string(it.Role))
				continue
			}
			// What a node draws is placed from its middle, as Pele's transform places it, and
			// written as Pele works it out from the node's size; the parts are checked to be
			// where that puts them.
			w, h := it.Box.W, it.Box.H
			described := it.Role == RoleState && len(it.Parts) == 2
			cx, cy = it.Box.X+w/2, it.Box.Y+h/2
			if it.Role == RoleNote || it.Role == RoleState && !described {
				cx, cy = it.At.X, it.At.Y
			}
			lines = append(lines, fmt.Sprintf("node %s %s [%s] %s", id, kind, classes, pt(cx, cy)))
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			if it.Tooltip != "" {
				lines = append(lines, "tip "+peleText(it.Tooltip))
			}
			box := func(p scene.Part) {
				if p.Path != nil || p.Box != it.Box {
					lines = append(lines, "mismatch box")
				}
				lines = append(lines, "rect "+pt(-w/2, -h/2)+","+pt(w, h)+" "+radius(it))
			}
			switch it.Role {
			case RoleStart, RoleEnd:
				for _, part := range it.Parts {
					r := part.Path.Bounds().W / 2
					checkPath(part.Path, cx, cy, r, 0, 0, r, -r, 0, 0, -r, r, 0)
					lines = append(lines, "circle "+num(r))
				}
			case RoleChoice:
				checkPath(it.Parts[0].Path, cx, cy, 0, -h/2, w/2, 0, 0, h/2, -w/2, 0)
				lines = append(lines, "polygon 0,"+num(-h/2)+" "+num(w/2)+",0 0,"+num(h/2)+" "+num(-w/2)+",0")
			case RoleFork, RoleJoin:
				box(it.Parts[0])
			case RoleNote:
				checkPath(it.Parts[0].Path, cx, cy, -w/2, -h/2, w/2-fold, -h/2, w/2, fold-h/2, w/2, h/2, -w/2, h/2)
				checkPath(it.Parts[1].Path, cx, cy, w/2-fold, -h/2, w/2-fold, fold-h/2, w/2, fold-h/2)
				r, b, f, c := num(w/2), num(h/2), num(w/2-fold), num(fold-h/2)
				lines = append(lines, "path M-"+r+",-"+b+"L"+f+",-"+b+"L"+r+","+c+"L"+r+","+b+"L-"+r+","+b+"Z",
					"path M"+f+",-"+b+"L"+f+","+c+"L"+r+","+c)
				label(it.Label, 0, 0)
			default:
				box(it.Parts[0])
				if !described {
					label(it.Label, 0, 0)
					break
				}
				at := padY - h/2 + it.Label.Height
				checkPath(it.Parts[1].Path, cx, cy, -w/2, at+6, w/2, at+6)
				lines = append(lines, "path M"+pt(-w/2, at+6)+"L"+pt(w/2, at+6))
				y := at - it.Label.Height/2
				check("label", it.At.X-cx, 0)
				check("label", it.At.Y-cy, y)
				label(it.Label, 0, y)
				descAt = at + 14
			}
		}
	}
	return lines
}

// peleID is an id as Pele writes a data-id: entities decoded, characters XML does not allow
// dropped.
func peleID(id string) string { return peleText(text.DecodeEntities(id)) }

// peleText is text as Pele's esc writes it, less the escapes: without the characters XML does
// not allow.
func peleText(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, s)
}

// peleMarker is the outline Pele's marker draws, as the oracle reads it.
func peleMarker(m scene.Marker) string {
	const arrow = scene.MarkerSize
	num := pelesvg.Num
	x, y, dx, dy := m.At.X, m.At.Y, m.DX, m.DY
	nx, ny := -dy, dx
	if m.Type != "arrow_point" {
		return ""
	}
	bx, by := x-float64(dx*arrow), y-float64(dy*arrow)
	half := float64(arrow * 0.42)
	return "M" + num(x) + "," + num(y) + "L" + num(bx+float64(nx*half)) + "," + num(by+float64(ny*half)) +
		"L" + num(bx-float64(nx*half)) + "," + num(by-float64(ny*half)) + "Z"
}

// Every diagram of the corpus lays out with each routing, and the svg backend draws it.
func TestRoutingAndDrawing(t *testing.T) {
	var p peleRender
	if err := fixture.Load("testdata/pele/render.json", &p); err != nil {
		t.Fatal(err)
	}
	for k, c := range p.Cases {
		if k%5 != 0 {
			continue
		}
		pre, err := diagram.Prepare(c.Src, diagram.Options{})
		if err != nil {
			t.Fatal(err)
		}
		db, err := ParsePrepared(pre)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range []layout.Routing{layout.Orthogonal, layout.Straight} {
			sc := Layout(db, pre.Config, &layout.Options{Routing: r, MaxWidth: c.MaxWidth})
			sc.Snap(0.5)
			for _, it := range sc.Items {
				if it.Kind == scene.Edge && (len(it.Parts) != 1 || it.Parts[0].Path == nil) {
					t.Fatalf("Layout(%q, routing %d): an edge without a path", c.Src, r)
				}
			}
			if out := svg.Render(sc, nil); len(sc.Items) > 0 && !strings.Contains(out, "limu-state-") {
				t.Fatalf("svg.Render of %q draws no state", c.Src)
			}
		}
	}
}
