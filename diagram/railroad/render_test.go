package railroad

import (
	"encoding/json"
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
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of railroad diagrams, read back out of its SVG as
// one line per thing drawn, in the order drawn, with its numbers written out in full and moved
// to where the scene has them; ./scripts/oracle.sh writes it. The test writes the same lines
// from the scene, its numbers as JavaScript's String writes them, and so compares them to the
// last bit. Pele draws tracks with arcs and the dots at their ends as circles; the test draws
// them again with scene.Path's ArcTo and Ellipse from Pele's numbers, and compares those paths
// with the items'.

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
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
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
		want := make([]string, len(c.Lines))
		for i, line := range c.Lines {
			want[i] = peleLine(line)
		}
		if i := pelesvg.FirstDiff(got, want); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, c.FontSize, i,
					pelesvg.At(got, i), pelesvg.At(want, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// peleLine is a line of the fixture with the paths and circles Pele draws drawn again as limu
// draws them.
func peleLine(line string) string {
	fields := strings.Fields(line)
	switch fields[0] {
	case "track", "arrows":
		bar := strings.Index(line, " | ")
		return line[:bar] + " | " + pathLine(pelePath(line[bar+3:]))
	case "start", "end":
		// start cx,cy r fill
		xy := strings.Split(fields[1], ",")
		x, y, r := number(xy[0]), number(xy[1]), number(fields[2])
		return fields[0] + " " + fields[3] + " | " + pathLine(new(scene.Path).Ellipse(x, y, r, r))
	}
	return line
}

func number(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return v
}

// pelePath is a path the oracle writes, M, L, A and Z with every number in full, drawn as limu
// draws it.
func pelePath(d string) *scene.Path {
	p := new(scene.Path)
	f := strings.Fields(d)
	num := func(i int) float64 { return number(f[i]) }
	for i := 0; i < len(f); {
		switch f[i] {
		case "M":
			p.MoveTo(num(i+1), num(i+2))
			i += 3
		case "L":
			p.LineTo(num(i+1), num(i+2))
			i += 3
		case "A":
			p.ArcTo(num(i+1), num(i+2), num(i+3), f[i+4] == "1", f[i+5] == "1", num(i+6), num(i+7))
			i += 8
		case "Z":
			p.Close()
			i++
		default:
			panic("unknown command " + f[i])
		}
	}
	return p
}

// pathLine is a path's steps, every number in full.
func pathLine(p *scene.Path) string {
	var out []string
	p.Each(func(op scene.Op, a []float64) {
		s := string("MLQCZ"[op])
		for _, v := range a {
			s += " " + chars.NumberString(v)
		}
		out = append(out, s)
	})
	return strings.Join(out, " ")
}

// written is text as Pele's esc writes it, without the control characters XML does not allow.
func written(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Fg:      "var(--_fg)",
	style.Muted:   "var(--_m)",
	style.Line:    "var(--_l)",
	style.Bg:      "var(--_bg)",
	style.Surface: "var(--_s)",
	style.Border:  "var(--_b)",
}

func paintName(p style.Paint) string {
	if s, ok := paintNames[p]; ok {
		return s
	}
	return "?"
}

// faceName is a face as the oracle names what Pele writes of it.
func faceName(f text.Face) string {
	switch f {
	case text.Mono:
		return "mono"
	case text.Italic:
		return "italic"
	case text.Bold:
		return "var(--_hw)"
	}
	return "-"
}

// labelText is the text of a one-line label, and the face of its first span.
func labelText(l *text.Label) (string, text.Face) {
	if l == nil || len(l.Lines) == 0 {
		return "", 0
	}
	s := ""
	for _, span := range l.Lines[0] {
		s += span.Text
	}
	return written(s), l.Lines[0][0].Face
}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Title != "" {
		lines = append(lines, "acc-title "+pelesvg.ID(sc.Title))
	}
	if sc.Desc != "" {
		lines = append(lines, "acc-desc "+pelesvg.ID(sc.Desc))
	}
	// What a stroke is drawn with: its paint, width, dashes, and cap and join.
	pen := func(it *scene.Item) string {
		s := it.Style
		width, dash := "-", "-"
		if w, ok := s.StrokeWidth.Get(); ok {
			width = n(w)
		}
		if s.Dash.Set() {
			var parts []string
			for _, d := range s.Dash.Pattern() {
				parts = append(parts, n(d))
			}
			dash = strings.Join(parts, " ")
		}
		return paintName(s.Stroke) + " " + width + " " + dash + " " + pelesvg.CapJoin(it)
	}
	// The middle of a box's text and what it writes, as the oracle reads Pele's.
	leafText := func(it *scene.Item) string {
		at := "t " + pt(it.At.X, it.At.Y)
		s, face := labelText(it.Label)
		if s == "" {
			return at
		}
		return at + " " + n(it.Label.Size) + " " + faceName(face) + " | " + s
	}
	for i := 0; i < len(sc.Items); i++ {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			l := it.Label
			y := it.At.Y - l.Height/2 + l.LineHeight/2 + float64(l.Size*0.35)
			var content []string
			for _, line := range l.Lines {
				s := ""
				for _, span := range line {
					s += span.Text
				}
				if len(line) == 0 {
					s = " "
				}
				content = append(content, written(s))
			}
			lines = append(lines, "title t "+pt(it.At.X, y)+" | "+strings.Join(content, " | "))
		case RoleName:
			s, face := labelText(it.Label)
			align := map[scene.Align]string{scene.Middle: "middle", scene.Start: "start", scene.End: "end"}[it.Align]
			lines = append(lines, "rule "+pelesvg.ID(it.ID),
				"name t "+pt(it.At.X, it.At.Y)+" "+n(it.Label.Size)+" "+faceName(face)+" "+align+" | "+s)
		case RoleTrack:
			fill := "?"
			if it.Parts[0].Paint == scene.StrokeOnly {
				fill = "none"
			}
			lines = append(lines, "track "+fill+" "+pen(it)+" | "+pathLine(it.Parts[0].Path))
		case RoleArrows:
			stroke := "?"
			if it.Parts[0].Paint == scene.FillOnly {
				stroke = "-"
			}
			lines = append(lines, "arrows "+paintName(it.Style.Fill)+" "+stroke+" | "+pathLine(it.Parts[0].Path))
		case RoleStart, RoleEnd:
			name := "start"
			if it.Role == RoleEnd {
				name = "end"
			}
			if it.Parts[0].Paint != scene.FillOnly {
				name += " stroked"
			}
			lines = append(lines, name+" "+paintName(it.Style.Fill)+" | "+pathLine(it.Parts[0].Path))
		case RoleCount:
			s, _ := labelText(it.Label)
			lines = append(lines, "count t "+pt(it.At.X, it.At.Y)+" "+n(it.Label.Size)+" "+paintName(it.Style.Text)+" | "+s)
		case RoleTerminal, RoleNonterminal, RoleSpecial:
			kind := strings.TrimPrefix(string(it.Role), "railroad.")
			if it.Role == RoleNonterminal {
				kind += " " + pelesvg.ID(it.ID)
			}
			part := it.Parts[0]
			r := part.Box
			radius := "-"
			if part.Rounded {
				radius = "theme"
				if v, ok := it.Style.Radius.Get(); ok {
					radius = n(v)
				}
			}
			head := kind + " " + pt(r.X, r.Y) + "," + pt(r.W, r.H) + " " + radius + " " + paintName(it.Style.Fill) + " " + pen(it)
			if it.Role != RoleSpecial {
				lines = append(lines, head+" | "+leafText(it))
				continue
			}
			// The marks are the two items after the box: the first starts where the whole
			// text starts, and the last ends where it ends.
			mid, face := labelText(it.Label)
			marks := "marks missing"
			if i+2 < len(sc.Items) && sc.Items[i+1].Role == RoleSpecialMark && sc.Items[i+2].Role == RoleSpecialMark {
				a, b := &sc.Items[i+1], &sc.Items[i+2]
				sa, fa := labelText(a.Label)
				sb, _ := labelText(b.Label)
				half := text.Arial.Width(sa+mid+sb, a.Label.Size, text.Italic) / 2
				marks = "marks misplaced"
				if a.At == (scene.Point{X: it.At.X - half, Y: it.At.Y}) && a.Align == scene.Start &&
					b.At == (scene.Point{X: it.At.X + half, Y: it.At.Y}) && b.Align == scene.End &&
					a.Style.Text == style.Muted && b.Style.Text == style.Muted && (it.Label == nil || face == fa) {
					marks = n(a.Label.Size) + " " + faceName(fa) + " | " + sa + "|" + mid + "|" + sb
				}
				i += 2
			}
			lines = append(lines, head+" | t "+pt(it.At.X, it.At.Y)+" "+marks)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
