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
	// inkedLabel is a label with the color ink picks for its item's text.
	inkedLabel := func(it *scene.Item) {
		if l := it.Label; l != nil && len(l.Lines) > 0 {
			lines = append(lines, textAt(l, it.At.X, it.At.Y)+ink(it))
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
			opaque := ""
			if _, ok := it.Style.FillOpacity.Get(); !ok {
				opaque = " opaque"
			}
			lines = append(lines, fmt.Sprintf("cluster %s [%s] %s %s%s", id, classes, box(it.Box), pen(it, ""), opaque))
			inkedLabel(it)
		case RoleCasing:
			// The casing's path, and the outlines of its relationship's markers, which follows
			// it.
			lines = append(lines, "casing "+pelesvg.D(it.Parts[0].Path))
			var edge *scene.Item
			for k := i + 1; k < len(sc.Items) && edge == nil; k++ {
				if sc.Items[k].Role == RoleRelationship && sc.Items[k].ID == it.ID {
					edge = &sc.Items[k]
				}
			}
			if edge == nil {
				lines = append(lines, "casing of no relationship")
				break
			}
			n := 1
			for _, m := range edge.Ends {
				lines = append(lines, casingLines(m)...)
				n += len(m.Outline())
			}
			if n != len(it.Parts) {
				lines = append(lines, "casing's markers differ")
			}
		case RoleRelationship:
			kind := "non-identifying dashed"
			if !it.Style.Dash.Set() {
				kind = "identifying"
			}
			lines = append(lines, fmt.Sprintf("edge %s %s %s %s%s", id, kind, pelesvg.D(it.Parts[0].Path), pen(it, ""), strokeInk(it)))
			for _, m := range it.Ends {
				lines = append(lines, markerLines(m, pen(it, m.Type), it.Style.Ground.Visible())...)
			}
		case RoleRelationshipLabel:
			lines = append(lines, "label "+id+" "+box(it.Box))
			label(it.Label, it.At.X, it.At.Y)
		case RoleEntity:
			lines = append(lines, fmt.Sprintf("node %s [%s]", id, classes), "box "+box(it.Parts[0].Box)+" "+pen(it, ""))
			for _, part := range it.Parts[1:] {
				switch {
				case part.Paint == scene.Shade && part.Rounded:
					lines = append(lines, "stripe "+box(part.Box)+" rounded"+shadeName(it))
				case part.Paint == scene.Shade:
					lines = append(lines, "stripe "+box(part.Box)+shadeName(it))
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
			inkedLabel(it)
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
			lines = append(lines, kind+" "+num(it.Label.Size)+" "+cellAt(it.Label, it.At.X, it.At.Y)+ink(it))
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
// " bg" after one filled with the background, or " on" after one filled with the fill the
// edge lies in, where on says it lies in one, and a ring as its center and radius, " on"
// after it where it is filled so. Pele writes a ring as a circle, which limu draws as
// curves, so the center is worked out as Pele works it out, and the ring's curves must lie
// about it. Each ends with pen, how the marker ends and turns its lines.
func markerLines(m scene.Marker, pen string, on bool) []string {
	var out []string
	for _, part := range outline(m) {
		hollow := ""
		if part.hollow && on {
			hollow = " on"
		} else if part.hollow && !part.ring {
			hollow = " bg"
		}
		out = append(out, part.astray...)
		out = append(out, "m "+part.shape+hollow+" "+pen)
	}
	return out
}

// casingLines are the lines of the outline of a marker in a casing, as the oracle reads
// Pele's: a path's d or a ring's center and radius, and " bg" after one filled with the
// background, as the casing fills what the marker fills.
func casingLines(m scene.Marker) []string {
	var out []string
	for _, part := range outline(m) {
		s := "casing " + part.shape
		if part.hollow {
			s += " bg"
		}
		out = append(append(out, part.astray...), s)
	}
	return out
}

// A markerPart is a part of a marker's outline as the oracle writes it: a path's d, or a
// ring's center and radius after "o ", whether it is hollow, and the lines that say a ring
// is not where Pele draws it.
type markerPart struct {
	shape        string
	ring, hollow bool
	astray       []string
}

// outline is a marker's outline as the oracle reads Pele's.
func outline(m scene.Marker) []markerPart {
	var out []markerPart
	for _, part := range m.Outline() {
		if back, ok := ringBack[m.Type]; ok && part.Path.Ops[1] == scene.CubeTo {
			x, y := m.At.X-float64(m.DX*back), m.At.Y-float64(m.DY*back)
			b := part.Path.Bounds()
			p := markerPart{shape: "o " + pelesvg.Num(x) + "," + pelesvg.Num(y) + " " + pelesvg.Num(b.W/2), ring: true, hollow: part.Paint == scene.Hollow}
			if math.Abs(b.X+b.W/2-x) > 1e-9 || math.Abs(b.Y+b.H/2-y) > 1e-9 || part.Paint != scene.Hollow {
				p.astray = []string{"ring astray"}
			}
			out = append(out, p)
			continue
		}
		out = append(out, markerPart{shape: pelesvg.D(part.Path), hollow: part.Paint == scene.Hollow})
	}
	return out
}

// inkNames are the colors Pele's ink picks, by the theme's paints the default look settles
// them as.
var inkNames = map[style.Paint]string{style.Fg: "var(--pele-fg-on-light,#1f1f1f)", style.Bg: "var(--pele-fg-on-dark,#fff)"}

// inked is the item's style as the default look settles it, and that style without the
// colors picked on a fill: the text on its own fill or on the fill under it, and the stroke
// on a ground. The diagram's own text and stroke are taken as a paint neither picks, so that
// a color picked differs from what the style would be without it even where it is the same
// as the default.
func inked(it *scene.Item) (got, plain style.Style) {
	var look style.Look
	own := it.Style
	own.Text, own.Stroke = style.SurfaceAlt, style.SurfaceAlt
	bare := own
	bare.OnFill, bare.Ground, bare.TextGround = false, style.Paint{}, style.Paint{}
	return look.Cascade(string(it.Role), own, it.Author), look.Cascade(string(it.Role), bare, it.Author)
}

// ink is what the oracle writes of the color ink picks for an item's text: " ink=" and the
// color, or nothing where it picks none.
func ink(it *scene.Item) string {
	if got, plain := inked(it); got.Text != plain.Text {
		return " ink=" + inkNames[got.Text]
	}
	return ""
}

// strokeInk is what the oracle writes of the color ink picks for an edge's stroke.
func strokeInk(it *scene.Item) string {
	if got, plain := inked(it); got.Stroke != plain.Stroke {
		return " ink=" + inkNames[got.Stroke]
	}
	return ""
}

// shadeName is what the oracle writes of the shade of an entity's stripes on a fill ink picks
// a color on, the other of the two it picks from: " shade=" and that color, or nothing where
// the stripes are in the background.
func shadeName(it *scene.Item) string {
	var look style.Look
	if p, ok := look.Shade(string(it.Role), it.Author); ok {
		return " shade=" + inkNames[p]
	}
	return ""
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
