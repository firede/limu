package class

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of class diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. The test writes the
// same lines from the scene, numbers as Pele's SVG writes them.

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
	failed := 0
	for _, c := range p.Cases {
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
	box := func(r scene.Rect) string { return pt(r.X, r.Y) + "," + num(r.W) + "," + num(r.H) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Where Pele's labelSvg puts the lines of a label centered on (cx, cy), and the runs of a
	// line with an icon in it, and then the icons.
	var icons []string
	textAt := func(l *text.Label, cx, cy float64) string {
		shift := float64(l.Size * 0.35)
		y := cy - l.Height/2 + l.LineHeight/2 + shift
		var at []string
		for i, line := range l.Lines {
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
			y += l.LineHeight
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
		if held != nil && it.Role != "class.annotation" {
			lines = append(lines, held...)
			held = nil
		}
		// Pele's escText writes a data-id with its entities decoded.
		id := peleText(text.DecodeEntities(it.ID))
		switch it.Role {
		case "class.title":
			lines = append(lines, "title "+textAt(it.Label, it.At.X, it.At.Y))
		case "class.namespace":
			lines = append(lines, "ns "+id+" "+box(it.Box))
			label(it.Label, it.At.X, it.At.Y)
		case "class.relation", "class.note-edge":
			kind := "relation"
			if it.Role == "class.note-edge" {
				kind = "note"
			}
			if d := it.Style.Dash.Pattern(); len(d) == 2 && d[0] == 6 {
				kind += " dashed"
			} else if len(d) == 2 && d[0] == 2 {
				kind += " dotted"
			}
			lines = append(lines, "edge "+id+" "+kind+" "+pelesvg.D(it.Parts[0].Path))
			for _, m := range it.Ends {
				if m.Type != "" {
					lines = append(lines, peleMarker(m))
				}
			}
		case "class.namespace-title":
			lines = append(lines, "struck "+id)
			label(it.Label, it.At.X, it.At.Y)
		case "class.label":
			lines = append(lines, "label "+id+" "+box(it.Box))
			label(it.Label, it.At.X, it.At.Y)
		case "class.cardinality":
			lines = append(lines, "card "+peleText(spans(it.Label))+" "+textAt(it.Label, it.At.X, it.At.Y))
		case "class.class":
			lines = append(lines, "class "+id+" ["+strings.Join(it.Class, " ")+"]")
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			if it.Tooltip != "" {
				lines = append(lines, "tip "+peleText(it.Tooltip))
			}
			lines = append(lines, "rect "+box(it.Parts[0].Box))
			if len(it.Parts) > 1 {
				lines = append(lines, "divider "+pelesvg.D(it.Parts[1].Path))
			}
			held = []string{}
			if len(it.Label.Lines) > 0 {
				held = append(held, "t "+textAt(it.Label, it.At.X, it.At.Y))
				held = append(held, icons...)
				icons = icons[:0]
			}
		case "class.annotation":
			lines = append(lines, "annotation "+peleText(spans(it.Label))+" "+textAt(it.Label, it.At.X, it.At.Y))
		case "class.attribute", "class.method":
			kind := strings.TrimPrefix(string(it.Role), "class.")
			if len(it.Parts) > 0 {
				kind += " static"
			} else if it.Label.Lines[0][0].Face&text.Italic != 0 {
				kind += " abstract"
			}
			lines = append(lines, "member "+kind+" "+textAt(it.Label, it.At.X, it.At.Y)+" "+peleText(spans(it.Label)))
		case "class.note":
			lines = append(lines, "note "+id, "path "+pelesvg.D(it.Parts[0].Path), "fold "+pelesvg.D(it.Parts[1].Path))
			label(it.Label, it.At.X, it.At.Y)
		case "class.interface":
			lines = append(lines, "interface "+id)
			label(it.Label, it.At.X, it.At.Y)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return append(lines, held...)
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
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, s)
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
