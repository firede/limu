package gantt

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of gantt charts, each in one of several time zones
// against a fixed time, read back out of its SVG as one line per thing drawn, in the order
// drawn; ./scripts/oracle.sh writes it. The test writes the same lines from the scene.

type peleRender struct {
	Zones []string `json:"zones"`
	Cases []struct {
		Src      string         `json:"src"`
		Zone     string         `json:"zone"`
		Now      int64          `json:"now"`
		MaxWidth float64        `json:"maxWidth"`
		FontSize float64        `json:"fontSize"`
		Lines    []string       `json:"lines"`
		Error    map[string]any `json:"error"`
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
	zones := map[string]*time.Location{}
	for i, loc := range loadZones(t, p.Zones) {
		zones[p.Zones[i]] = loc
	}
	failed := 0
	fail := func(format string, args ...any) {
		if failed++; failed <= 20 {
			t.Errorf(format, args...)
		}
	}
	for _, c := range p.Cases {
		pre, err := diagram.Prepare(c.Src, diagram.Options{})
		if err != nil {
			fail("Prepare(%q): %v", c.Src, err)
			continue
		}
		db, err := ParsePrepared(pre)
		if err != nil {
			fail("ParsePrepared(%q): %v", c.Src, err)
			continue
		}
		o := &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize, Now: time.UnixMilli(c.Now), Location: zones[c.Zone]}
		sc, err := Layout(db, pre.Config, o)
		switch {
		case c.Error != nil && err == nil:
			fail("Layout(%q) in %s: no error, Pele %v", c.Src, c.Zone, c.Error)
			continue
		case c.Error != nil:
			got := dumpError(err)
			if got["code"] != c.Error["code"] || got["message"] != c.Error["message"] {
				fail("Layout(%q) in %s: %v, Pele %v", c.Src, c.Zone, err, c.Error)
			}
			continue
		case err != nil:
			fail("Layout(%q) in %s: %v", c.Src, c.Zone, err)
			continue
		}
		got := sceneLines(sc)
		if i := pelesvg.FirstDiff(got, c.Lines); i >= 0 {
			fail("Layout(%q) in %s at %d, max %v, size %v, line %d\n got %s\nPele %s",
				c.Src, c.Zone, c.Now, c.MaxWidth, c.FontSize, i, pelesvg.At(got, i), pelesvg.At(c.Lines, i))
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Bg:                   "var(--_bg)",
	style.Fg:                   "var(--_fg)",
	style.Muted:                "var(--_m)",
	style.Line:                 "var(--_l)",
	style.Surface:              "var(--_s)",
	style.SurfaceAlt:           "var(--_a)",
	style.Border:               "var(--_b)",
	style.Accent:               "var(--_c)",
	style.Accent.Alpha(0.25):   "var(--_c)/0.25",
	style.Critical:             "var(--pele-critical,#e45756)",
	style.Critical.Alpha(0.25): "var(--pele-critical,#e45756)/0.25",
	style.None:                 "none",
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

// pen is how an item's lines end and turn their corners, as the oracle reads Pele's
// stroke-linecap and stroke-linejoin: the author's or the diagram's, or "-" for neither.
func pen(it *scene.Item) string {
	s := it.Author.Over(it.Style)
	name := func(v string) string {
		if v == "" {
			return "-"
		}
		return v
	}
	return name(s.Cap.String()) + " " + name(s.Join.String())
}

// quote is a string as JavaScript's JSON.stringify writes it.
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// dropControls drops the control characters Pele's esc drops.
func dropControls(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// peleID is an id as Pele writes a data-id: entities decoded, the control characters XML does
// not allow dropped.
func peleID(id string) string { return dropControls(text.DecodeEntities(id)) }

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// Where Pele's text puts the first line of a label centered on (cx, cy).
	firstLine := func(l *text.Label, cx, cy float64) string {
		return pt(cx, cy-l.Height/2+l.LineHeight/2+float64(l.Size*0.35))
	}
	textAt := func(it *scene.Item) string {
		s := "t " + firstLine(it.Label, it.At.X, it.At.Y) + " " + paintName(it.Style.Text)
		if it.Align != scene.Middle {
			s += " not centered"
		}
		return s
	}
	sized := func(it *scene.Item) string { return textAt(it) + " s" + num(it.Label.Size) }
	paths := func(p *scene.Path, paint string) {
		var cur []string
		flush := func() {
			if len(cur) > 0 {
				lines = append(lines, "path "+strings.Join(cur, " ")+" "+paint)
			}
			cur = nil
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
		})
		flush()
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case "gantt.title":
			lines = append(lines, "title "+textAt(it))
		case "gantt.section":
			r := it.Box
			if len(it.Parts) != 1 || it.Parts[0].Box != r || it.Style.Stroke != style.None {
				lines = append(lines, "band not one box")
			}
			index := strings.TrimPrefix(strings.Join(it.Tags, " "), "section-")
			lines = append(lines, fmt.Sprintf("band %s %s,%s %s -", index, pt(r.X, r.Y), pt(r.W, r.H), paintName(pelesvg.OwnFill(it))))
		case "gantt.excluded":
			r := it.Box
			if len(it.Parts) != 1 || it.Parts[0].Box != r || it.Style.Fill != style.SurfaceAlt {
				lines = append(lines, "excluded not one box")
			}
			lines = append(lines, "excluded "+pt(r.X, r.Y)+","+pt(r.W, r.H))
		case "gantt.grid", "gantt.axis":
			lines = append(lines, strings.TrimPrefix(string(it.Role), "gantt."))
			paths(it.Parts[0].Path, "- "+paintName(it.Style.Stroke)+" "+pen(it))
		case "gantt.vert":
			lines = append(lines, "vert "+peleID(it.ID))
			paths(it.Parts[0].Path, "- - "+pen(it))
		case "gantt.today":
			// The author's style of the line, which Pele writes as it is, takes the place of the
			// triangle.
			if len(it.Parts) == 1 {
				lines = append(lines, "today styled")
			} else {
				lines = append(lines, "today -")
			}
			paths(it.Parts[0].Path, "none "+paintName(it.Style.Stroke)+" "+pen(it))
			if len(it.Parts) > 1 {
				paths(it.Parts[1].Path, paintName(pelesvg.OwnFill(it))+" - "+pen(it))
			}
		case "gantt.task":
			lines = append(lines, fmt.Sprintf("task %s [%s]", peleID(it.ID), strings.Join(it.Tags, " ")))
			if it.Link != "" {
				lines = append(lines, "link "+it.Link)
			}
			paint := paintName(pelesvg.OwnFill(it)) + " " + paintName(it.Style.Stroke)
			for _, part := range it.Parts {
				if part.Path != nil {
					paths(part.Path, paint+" "+pen(it))
					continue
				}
				r := part.Box
				s := "rect " + pt(r.X, r.Y) + "," + pt(r.W, r.H)
				if part.Rounded {
					s += " r"
				}
				lines = append(lines, s+" "+paint)
			}
			if it.Label != nil {
				lines = append(lines, sized(it))
			}
		case "gantt.section-label":
			lines = append(lines, "section "+peleID(it.ID), sized(it))
		case "gantt.vert-label":
			lines = append(lines, "vert-label "+peleID(it.ID), sized(it))
		case "gantt.tick":
			l := it.Label
			if len(l.Lines) != 1 || len(l.Lines[0]) != 1 || it.Style.Text != style.Muted {
				lines = append(lines, "tick not one span")
			}
			lines = append(lines, "tick "+firstLine(l, it.At.X, it.At.Y)+" s"+num(l.Size)+" "+quote(dropControls(l.Lines[0][0].Text)))
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
