package er

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
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of ER diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. The test writes
// the same lines from the scene, numbers as Pele's SVG writes them. Pele draws an entity's
// parts and text in a frame about its center; the oracle moves them into the diagram's.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		// Mono says the attributes are set in the monospace face, as Pele's monospace option
		// lists them.
		Mono bool `json:"mono"`
		fixture.Drawing
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 2000 {
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
		o := &layout.Options{MaxWidth: c.MaxWidth}
		if c.Mono {
			o.Monospace = layout.Attributes
		}
		got := sceneLines(Layout(db, pre.Config, o))
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, mono %v), %s", c.Src, c.MaxWidth, c.Mono, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	box := func(r scene.Rect) string {
		return pt(r.X, r.Y) + "," + num(r.W) + "," + num(r.H)
	}
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Where Pele's text puts the first line of a label centered on (cx, cy) down.
	textAt := func(l *text.Label, cx, cy float64) string {
		shift := float64(l.Size * 0.35)
		return "t " + pt(cx, cy-l.Height/2+l.LineHeight/2+shift)
	}
	label := func(l *text.Label, cx, cy float64) {
		if l != nil && len(l.Lines) > 0 {
			lines = append(lines, textAt(l, cx, cy))
		}
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		id := peleID(it.ID)
		classes := strings.Join(it.Class, " ")
		switch it.Role {
		case RoleTitle:
			lines = append(lines, "title "+textAt(it.Label, it.At.X, it.At.Y))
		case RoleCluster:
			lines = append(lines, fmt.Sprintf("cluster %s [%s] %s %s", id, classes, box(it.Box), pen(it, "")))
			label(it.Label, it.At.X, it.At.Y)
		case RoleRelationship:
			kind := "non-identifying dashed"
			if !it.Style.Dash.Set() {
				kind = "identifying"
			}
			lines = append(lines, fmt.Sprintf("edge %s %s %s %s", id, kind, pelesvg.D(it.Parts[0].Path), pen(it, "")))
			for _, m := range it.Ends {
				lines = append(lines, markerLines(m, pen(it, m.Type))...)
			}
		case RoleRelationshipLabel:
			lines = append(lines, "label "+id+" "+box(it.Box))
			label(it.Label, it.At.X, it.At.Y)
		case RoleEntity:
			lines = append(lines, fmt.Sprintf("node %s [%s]", id, classes), "box "+box(it.Parts[0].Box)+" "+pen(it, ""))
			for _, part := range it.Parts[1:] {
				switch {
				case part.Paint == scene.Shade && part.Rounded:
					lines = append(lines, "stripe "+box(part.Box)+" rounded")
				case part.Paint == scene.Shade:
					lines = append(lines, "stripe "+box(part.Box))
				default:
					// Pele draws the line under the name, an item of its own after the
					// entity, before the outline drawn again.
					if i+1 < len(sc.Items) && sc.Items[i+1].Role == RoleDivider {
						lines = append(lines, "divider "+pelesvg.D(sc.Items[i+1].Parts[0].Path))
					}
					lines = append(lines, "outline "+box(part.Box)+" "+pen(it, ""))
				}
			}
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			label(it.Label, it.At.X, it.At.Y)
		case RoleDivider:
			// Written with its entity.
		case RoleAttributeType, RoleAttributeName, RoleAttributeKeys, RoleAttributeComment:
			kind := strings.TrimPrefix(string(it.Role), "er.attribute-")
			if it.Align != scene.Start {
				lines = append(lines, "misaligned "+kind)
			}
			if allMono(it.Label) {
				kind += " mono"
			}
			lines = append(lines, kind+" "+num(it.Label.Size)+" "+cellAt(it.Label, it.At.X, it.At.Y))
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}

// allMono reports whether every span of a label with text is in the monospace face, and there
// is one: whether Pele's textLeft names the monospace family on the label's text, as it does
// where Pele's sharedStyle holds Mono.
func allMono(l *text.Label) bool {
	any := false
	for _, line := range l.Lines {
		for _, span := range line {
			if span.Icon != "" || span.Text == "" {
				continue
			}
			if span.Face&text.Mono == 0 {
				return false
			}
			any = true
		}
	}
	return any
}

// cellAt is where Pele's textLeft puts a cell's text, which starts at x and is centered on y
// down: on its first span with text, if it has one.
func cellAt(l *text.Label, x, y float64) string {
	for i, line := range l.Lines {
		sx := x
		for _, span := range line {
			if span.Text != "" || span.Icon != "" {
				shift := float64(l.Size * 0.35)
				first := y - l.Height/2 + l.LineHeight/2 + shift
				return "t " + pelesvg.Num(sx) + "," + pelesvg.Num(first+float64(float64(i)*l.LineHeight))
			}
			sx += span.Width
		}
	}
	return "t none"
}

// The ring a marker of a type has, and how far back along its line it is.
var ringBack = map[scene.MarkerType]float64{"er.zero_or_one": 13.5, "er.zero_or_more": 14}

// markerLines are the lines of a marker's outline, as the oracle reads Pele's: a path's d,
// " bg" after one filled with the background, and a ring as its center and radius. Pele
// writes a ring as a circle, which limu draws as curves, so the center is worked out as Pele
// works it out, and the ring's curves must lie about it. Each ends with pen, how the marker
// ends and turns its lines.
func markerLines(m scene.Marker, pen string) []string {
	var out []string
	for _, part := range m.Outline() {
		if back, ok := ringBack[m.Type]; ok && part.Path.Ops[1] == scene.CubeTo {
			x, y := m.At.X-float64(m.DX*back), m.At.Y-float64(m.DY*back)
			b := part.Path.Bounds()
			if math.Abs(b.X+b.W/2-x) > 1e-9 || math.Abs(b.Y+b.H/2-y) > 1e-9 || part.Paint != scene.Hollow {
				out = append(out, "ring astray")
			}
			out = append(out, "m o "+pelesvg.Num(x)+","+pelesvg.Num(y)+" "+pelesvg.Num(b.W/2)+" "+pen)
			continue
		}
		s := "m " + pelesvg.D(part.Path)
		if part.Paint == scene.Hollow {
			s += " bg"
		}
		out = append(out, s+" "+pen)
	}
	return out
}

// pen is how an item's lines end and turn their corners, as the oracle reads them, "cap/join":
// the author's style over the diagram's own, and a marker's own join over both.
func pen(it *scene.Item, marker scene.MarkerType) string {
	s := it.Author.Over(it.Style)
	if j := marker.Join(); j.Set() {
		s.Join = j
	}
	if !s.Cap.Set() {
		s.Cap = style.CapButt
	}
	if !s.Join.Set() {
		s.Join = style.JoinMiter
	}
	return s.Cap.String() + "/" + s.Join.String()
}

// peleID is an id as Pele writes a data-id: entities decoded, characters XML does not allow
// dropped.
func peleID(id string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, text.DecodeEntities(id))
}
