package class

import (
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

// The fixture is what Pele's render draws of class diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. The test writes the
// same lines from the scene, numbers as Pele's SVG writes them.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		// Mono says the members are set in the monospace face, as Pele's monospace option
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
			o.Monospace = layout.Members
		}
		got := sceneLines(Layout(db, pre.Config, o))
		if msg := f.Diff(k, &c.Drawing, got, pelesvg.TagsLast("clickable")); msg != "" {
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
	box := func(r scene.Rect) string { return pt(r.X, r.Y) + "," + num(r.W) + "," + num(r.H) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Where Pele's labelSvg puts the lines of a label centered on (cx, cy), and the runs of a
	// line with an icon in it, and then the icons.
	var icons []string
	textAt := func(l *text.Label, cx, cy float64) string {
		shift := float64(l.Size * 0.35)
		top := cy - l.Height/2
		var at []string
		for i, line := range l.Lines {
			y := top + l.LineHeight/2 + shift
			iconic := false
			for _, s := range line {
				iconic = iconic || s.Icon != ""
			}
			if !iconic {
				at = append(at, pt(cx, y))
			} else {
				x := cx - l.Widths[i]/2
				for _, s := range line {
					if s.Icon != "" {
						side := float64(l.Size * 1.1)
						icons = append(icons, "icon "+s.Icon+" "+pt(x, y-shift-side/2)+","+num(side)+","+num(side))
					} else if s.Text != "" {
						at = append(at, pt(x, y))
					}
					x += s.Width
				}
			}
			top += l.LineHeight
		}
		return strings.Join(at, " ")
	}
	label := func(l *text.Label, cx, cy float64) {
		if l != nil && len(l.Lines) > 0 {
			lines = append(lines, "t "+textAt(l, cx, cy))
			lines = append(lines, icons...)
			icons = icons[:0]
		}
	}
	// A class's name is written after its annotations, which are items of their own after it.
	var held []string
	for i := range sc.Items {
		it := &sc.Items[i]
		if held != nil && it.Role != RoleAnnotation && it.Role != RoleDivider {
			lines = append(lines, held...)
			held = nil
		}
		// Pele writes a member starting at its x, with no anchor, and every other label with
		// labelSvg, centered on its point. The oracle reads only where a text is, so the
		// alignment is checked here.
		align := scene.Middle
		if it.Role == RoleAttribute || it.Role == RoleMethod {
			align = scene.Start
		}
		if it.Label != nil && !it.Label.Empty() && it.Align != align {
			lines = append(lines, "misaligned "+string(it.Role))
		}
		// Pele's escText writes a data-id with its entities decoded.
		id := peleText(text.DecodeEntities(it.ID))
		switch it.Role {
		case RoleTitle:
			lines = append(lines, "title "+textAt(it.Label, it.At.X, it.At.Y))
		case RoleNamespace:
			lines = append(lines, "ns "+id+" "+box(it.Box))
			label(it.Label, it.At.X, it.At.Y)
		case RoleRelation, RoleNoteEdge:
			kind := "relation"
			if it.Role == RoleNoteEdge {
				kind = "note"
			}
			if d := it.Style.Dash.Pattern(); len(d) == 2 && d[0] == 6 {
				kind += " dashed"
			} else if len(d) == 2 && d[0] == 2 {
				kind += " dotted"
			}
			lines = append(lines, "edge "+id+" "+kind+" "+pelesvg.D(it.Parts[0].Path)+" "+pen(it, ""))
			for _, m := range it.Ends {
				if m.Type != "" {
					lines = append(lines, peleMarker(m)+" "+pen(it, m.Type))
				}
			}
		case RoleNamespaceTitle:
			lines = append(lines, "struck "+id)
			label(it.Label, it.At.X, it.At.Y)
		case RoleEdgeLabel:
			lines = append(lines, "label "+id+" "+box(it.Box))
			label(it.Label, it.At.X, it.At.Y)
		case RoleCardinality:
			lines = append(lines, "card "+peleText(spans(it.Label))+" "+textAt(it.Label, it.At.X, it.At.Y))
		case RoleClass:
			lines = append(lines, "class "+id+" ["+pelesvg.Classes(it)+"]")
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			if it.Tooltip != "" {
				lines = append(lines, "tip "+peleText(it.Tooltip))
			}
			lines = append(lines, "rect "+box(it.Parts[0].Box)+" "+pen(it, ""))
			// Pele draws the lines under the name and the attributes, an item of their own
			// after the class, before its label.
			if i+1 < len(sc.Items) && sc.Items[i+1].Role == RoleDivider {
				lines = append(lines, "divider "+pelesvg.D(sc.Items[i+1].Parts[0].Path))
			}
			held = []string{}
			if len(it.Label.Lines) > 0 {
				held = append(held, "t "+textAt(it.Label, it.At.X, it.At.Y)+ink(it))
				held = append(held, icons...)
				icons = icons[:0]
			}
		case RoleDivider:
			// Written with its class.
		case RoleAnnotation:
			lines = append(lines, "annotation "+peleText(spans(it.Label))+" "+textAt(it.Label, it.At.X, it.At.Y)+ink(it))
		case RoleAttribute, RoleMethod:
			kind := strings.TrimPrefix(string(it.Role), "class.")
			if len(it.Parts) > 0 {
				kind += " static"
			} else if it.Label.Lines[0][0].Face&text.Italic != 0 {
				kind += " abstract"
			}
			if it.Label.Lines[0][0].Face&text.Mono != 0 {
				kind += " mono"
			}
			lines = append(lines, "member "+kind+" "+textAt(it.Label, it.At.X, it.At.Y)+" "+peleText(spans(it.Label))+ink(it))
		case RoleNote:
			lines = append(lines, "note "+id, "path "+pelesvg.D(it.Parts[0].Path)+" "+pen(it, ""), "fold "+pelesvg.D(it.Parts[1].Path)+" "+pen(it, ""))
			label(it.Label, it.At.X, it.At.Y)
		case RoleInterface:
			lines = append(lines, "interface "+id)
			label(it.Label, it.At.X, it.At.Y)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return append(lines, held...)
}

// inkNames are the colors Pele's ink picks, by the theme's paints the default look settles
// them as.
var inkNames = map[style.Paint]string{style.Fg: "var(--pele-fg-on-light,#1f1f1f)", style.Bg: "var(--pele-fg-on-dark,#fff)"}

// ink is what the oracle writes of the color ink picks for an item's text on a fill: " ink="
// and the color, where the default look picks one, and nothing elsewhere. The diagram's own
// text is taken as a paint ink never picks, so that a color picked shows even where it is the
// same as the diagram's.
func ink(it *scene.Item) string {
	var look style.Look
	own := it.Style
	own.Text = style.SurfaceAlt
	bare := own
	bare.OnFill, bare.Ground, bare.TextGround = false, style.Paint{}, style.Paint{}
	if got, plain := look.Cascade(string(it.Role), own, it.Author), look.Cascade(string(it.Role), bare, it.Author); got.Text != plain.Text {
		return " ink=" + inkNames[got.Text]
	}
	return ""
}

// spans is the text of a label of one line.
func spans(l *text.Label) string {
	var b strings.Builder
	for _, s := range l.Lines[0] {
		b.WriteString(s.Text)
	}
	return b.String()
}

// peleText is text as Pele writes it into its SVG, characters XML does not allow dropped.
func peleText(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, s)
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

// peleMarker is the marker Pele draws at an end, as the oracle reads it: how it is painted
// and its outline, or the middle of a lollipop's ring.
func peleMarker(m scene.Marker) string {
	parts := m.Outline()
	if len(parts) != 1 {
		return "m unknown " + string(m.Type)
	}
	paint := map[scene.Painting]string{scene.FillOnly: "fill", scene.FillAndStroke: "fill+stroke", scene.StrokeOnly: "stroke"}[parts[0].Paint]
	if m.Type == "class.lollipop" {
		return "m " + paint + " o " + pelesvg.Num(m.At.X-float64(m.DX*5)) + "," + pelesvg.Num(m.At.Y-float64(m.DY*5))
	}
	return "m " + paint + " " + pelesvg.D(parts[0].Path)
}
