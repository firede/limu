package sequence

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of sequence diagrams, read back out of its SVG as
// one line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. Pele draws
// everything in a group moved by the padding past the diagram's extents; the oracle moves
// each place by it, as the scene holds it, before it writes numbers as Pele's SVG writes
// them. The test writes the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		FontSize float64 `json:"fontSize"`
		fixture.Drawing
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
		got := sceneLines(Layout(db, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize}))
		if msg := f.Diff(k, &c.Drawing, got, authorFill); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), %s", c.Src, c.MaxWidth, c.FontSize, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
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

func paintName(p style.Paint) string {
	if s, ok := paintNames[p]; ok {
		return s
	}
	return "?"
}

// partPaint is how a part of an item is painted, as the oracle reads Pele's fill and stroke,
// with the width of the stroke where it is not the theme's, its ends and corners, and whether
// the author set a fill.
func partPaint(it *scene.Item, paint scene.Painting) string {
	fill, stroke := paintName(pelesvg.OwnFill(it)), paintName(it.Style.Stroke)
	switch paint {
	case scene.StrokeOnly:
		fill = "none"
	case scene.FillOnly:
		stroke = "none"
	}
	s := fill + " " + stroke
	if w, ok := it.Style.StrokeWidth.Get(); ok {
		s += " w=" + pelesvg.Num(w)
	}
	s += ends(it.Author.Over(it.Style), stroke, 0)
	if it.Author.Fill.Set() {
		s += " author=" + fmt.Sprint(it.Author.Fill)
	}
	return s
}

// authorFill writes the fill an author gave in a line of Pele's as the paint the style package
// reads it as. Pele writes whatever fill the author gave into the style; a browser drops one it
// cannot read, and draws the theme's paint, as limu does with a fill it cannot read.
func authorFill(line string) string {
	at := strings.Index(line, " author=\"")
	if at < 0 {
		return line
	}
	dec := json.NewDecoder(strings.NewReader(line[at+len(" author="):]))
	var value string
	if err := dec.Decode(&value); err != nil {
		return line
	}
	rest := line[at+len(" author=")+int(dec.InputOffset()):]
	if fill := css.Declarations([]string{"fill:" + value}).Style.Fill; fill.Set() {
		return line[:at] + " author=" + fmt.Sprint(fill) + rest
	}
	return line[:at] + rest
}

// markerPaint is how a part of a marker is painted, in the color of its edge's line.
func markerPaint(it *scene.Item, paint scene.Painting, join style.Join) string {
	line := paintName(it.Style.Stroke)
	if paint == scene.FillOnly {
		return line + " none"
	}
	return "none " + line + ends(it.Author.Over(it.Style), line, join)
}

// ends is how a stroke ends and turns its corners, as the oracle reads them: " cap=" and
// " join=" where they are not SVG's initial butt and miter, and nothing where nothing is
// stroked. A marker's own join, when it has one, is over the style's.
func ends(s style.Style, stroke string, join style.Join) string {
	if stroke == "none" {
		return ""
	}
	if join.Set() {
		s.Join = join
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

// clean drops what XML does not allow, as Pele's esc does.
func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// stringified writes a string as JSON.stringify does, which leaves <, >, & and the separators of
// lines and paragraphs as they are. What clean leaves of the controls is a tab or an end of
// line.
func stringified(s string) string {
	return `"` + strings.NewReplacer(`"`, `\"`, `\`, `\\`, "\t", `\t`, "\n", `\n`, "\r", `\r`).Replace(s) + `"`
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	id := func(s string) string { return " #" + pelesvg.ID(s) }
	content := func(l *text.Label) string {
		var out []string
		for _, line := range l.Lines {
			var b strings.Builder
			for _, s := range line {
				b.WriteString(s.Text)
			}
			if len(line) == 0 {
				b.WriteString(" ")
			}
			out = append(out, clean(b.String()))
		}
		return stringified(strings.Join(out, "|"))
	}
	// The text of a label centered on At, its first line where Pele's labelSvg puts it; a
	// tab's kind and a message's number Pele writes as one line by its own sum.
	label := func(it *scene.Item) {
		l := it.Label
		if l == nil || len(l.Lines) == 0 {
			return
		}
		shift := float64(l.Size * 0.35)
		y := it.At.Y - l.Height/2 + l.LineHeight/2 + shift
		if it.Role == RoleFrameKind || it.Role == RoleNumber {
			y = it.At.Y + shift
		}
		lines = append(lines, "text "+pt(it.At.X, y)+" "+paintName(it.Style.Text)+" "+content(l))
	}
	// The subpaths of a path, with the curves an arc was drawn with written as the arc, by
	// where it ends.
	subpaths := func(p *scene.Path) []string {
		var out []string
		var cur []string
		curve := false
		p.Each(func(op scene.Op, a []float64) {
			if op == scene.MoveTo && len(cur) > 0 {
				out = append(out, strings.Join(cur, " "))
				cur = nil
			}
			switch op {
			case scene.CubeTo:
				s := "A " + pt(a[4], a[5])
				if curve {
					cur[len(cur)-1] = s
				} else {
					cur = append(cur, s)
				}
			case scene.Close:
				cur = append(cur, "Z")
			default:
				cur = append(cur, string("MLQ"[op])+" "+pt(a[len(a)-2], a[len(a)-1]))
			}
			curve = op == scene.CubeTo
		})
		if len(cur) > 0 {
			out = append(out, strings.Join(cur, " "))
		}
		return out
	}
	// An ellipse, as Path.Ellipse draws it: from its right end, four quarters, closed.
	ellipse := func(p *scene.Path) (string, bool) {
		if len(p.Ops) != 6 || p.Ops[0] != scene.MoveTo || p.Ops[5] != scene.Close {
			return "", false
		}
		for _, op := range p.Ops[1:5] {
			if op != scene.CubeTo {
				return "", false
			}
		}
		cx, cy := p.Args[6], p.Args[1]
		return "ellipse " + pt(cx, cy) + " " + num(p.Args[0]-cx), true
	}
	part := func(it *scene.Item, pa *scene.Part, suffix string) {
		if pa.Path == nil {
			r := pa.Box
			radius := "-"
			if pa.Rounded {
				radius = "r"
				if v, ok := it.Style.Radius.Get(); ok {
					radius = "rx=" + strconv.FormatFloat(v, 'f', -1, 64)
				}
			}
			lines = append(lines, "rect "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+radius+" "+partPaint(it, pa.Paint)+suffix)
			return
		}
		if s, ok := ellipse(pa.Path); ok {
			lines = append(lines, s+" "+partPaint(it, pa.Paint)+suffix)
			return
		}
		dash := ""
		if it.Style.Dash.Set() {
			var d []string
			for _, v := range it.Style.Dash.Pattern() {
				d = append(d, num(v))
			}
			dash = " dash=" + strings.Join(d, " ")
		}
		for _, s := range subpaths(pa.Path) {
			lines = append(lines, "path "+s+" "+partPaint(it, pa.Paint)+dash+suffix)
		}
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			label(it)
		case RoleBox:
			box := ""
			if it.ID != "" {
				box = id(it.ID)
			}
			lines = append(lines, "g pele-cluster pele-box"+box)
			part(it, &it.Parts[0], "")
			label(it)
		case RoleRect, RoleDivider:
			part(it, &it.Parts[0], "")
		case RoleLifeline, RoleActivation, RoleDestroy:
			part(it, &it.Parts[0], id(it.ID))
		case RoleFrame:
			lines = append(lines, "g pele-frame pele-frame-"+string(it.Shape))
			part(it, &it.Parts[0], "")
		case RoleCondition, RoleFrameKind:
			for k := range it.Parts {
				part(it, &it.Parts[k], "")
			}
			label(it)
		case RoleMessage:
			lines = append(lines, "g pele-edge pele-message"+id(it.ID))
			part(it, &it.Parts[0], "")
			for _, m := range it.Ends {
				for _, pa := range m.Outline() {
					for _, s := range subpaths(pa.Path) {
						lines = append(lines, "path "+s+" "+markerPaint(it, pa.Paint, m.Type.Join()))
					}
				}
			}
			for k := 1; k < len(it.Parts); k++ {
				part(it, &it.Parts[k], "")
			}
		case RoleNumber:
			lines = append(lines, "g pele-sequence-number")
			part(it, &it.Parts[0], "")
			label(it)
		case RoleMessageLabel:
			lines = append(lines, "g pele-edge-label"+id(it.ID))
			for k := range it.Parts {
				part(it, &it.Parts[k], "")
			}
			label(it)
		case RoleNote:
			lines = append(lines, "g pele-note"+id(it.ID))
			part(it, &it.Parts[0], "")
			label(it)
		case RoleActor:
			kind := string(it.Shape)
			if kind == "rect" {
				kind = "participant"
			}
			cls := "pele-node pele-actor pele-actor-" + kind
			// A participant drawn at the foot of its lifeline comes before the one at its
			// head.
			for k := i + 1; k < len(sc.Items); k++ {
				if sc.Items[k].Role == RoleActor && sc.Items[k].ID == it.ID {
					cls += " pele-actor-footer"
					break
				}
			}
			for _, c := range it.Class {
				cls += " " + c
			}
			lines = append(lines, "g "+cls+id(it.ID))
			for k := range it.Parts {
				part(it, &it.Parts[k], "")
			}
			label(it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
