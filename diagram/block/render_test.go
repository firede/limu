package block

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/draw"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's render draws of block diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn, with every number in full, and the errors of
// those it does not draw; ./scripts/oracle.sh writes it. The test writes the same lines from
// the scene.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		FontSize float64 `json:"fontSize"`
		fixture.Drawing
		Error string `json:"error"`
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 4000 {
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
		sc, err := Layout(db, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize})
		switch {
		case c.Error != "":
			if err == nil || err.Error() != c.Error {
				if failed++; failed <= 20 {
					t.Errorf("Layout(%q): error %v, Pele %q", c.Src, err, c.Error)
				}
			}
			continue
		case err != nil:
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q): %v", c.Src, err)
			}
			continue
		}
		got := sceneLines(sc, c.Line)
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), %s", c.Src, c.MaxWidth, c.FontSize, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// n writes a number in full, as JavaScript's String does.
func n(v float64) string { return chars.NumberString(v) }

func pt(x, y float64) string { return n(x) + "," + n(y) }

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Bg:         "var(--_bg)",
	style.Muted:      "var(--_m)",
	style.Line:       "var(--_l)",
	style.Surface:    "var(--_s)",
	style.SurfaceAlt: "var(--_a)",
	style.Border:     "var(--_b)",
	style.None:       "none",
}

// paintName is a paint as the oracle writes it: its name in Pele's SVG, and its opacity
// after a slash where it has one; "-" for none given.
func paintName(p style.Paint) string {
	if !p.Set() {
		return "-"
	}
	f := p.Factor()
	s, ok := paintNames[p.Alpha(1/float64(f))]
	if f != 1 {
		// An opaque paint and the same at an opacity are the same but for the factor.
		opaque := p
		for k := range paintNames {
			if k.Token() == p.Token() {
				opaque = k
			}
		}
		s, ok = paintNames[opaque]
		s += "/" + strconv.FormatFloat(float64(f), 'g', -1, 32)
	}
	if !ok {
		return "?"
	}
	return s
}

// partPaint is how a part of an item is painted, as the oracle reads Pele's fill and stroke.
// The diagram's fill and stroke are at the opacities the author's style laid over the
// diagram's gives, as the oracle reads them from Pele's attributes and the author's style.
func partPaint(it *scene.Item, part *scene.Part) string {
	s := it.Author.Over(it.Style)
	stroke := paintName(it.Style.Stroke.Alpha(s.StrokeOpacity.Or(1)))
	fill := paintName(it.Style.Fill.Alpha(s.FillOpacity.Or(1)))
	switch part.Paint {
	case scene.StrokeOnly:
		return "none " + stroke
	case scene.FillOnly:
		return fill + " -"
	}
	return fill + " " + stroke
}

// inkNames are the colors Pele's ink picks, by the theme's paints the default look settles them
// as.
var inkNames = map[style.Paint]string{style.Fg: "var(--pele-fg-on-light,#1f1f1f)", style.Bg: "var(--pele-fg-on-dark,#fff)"}

// inked is the item's style as the default look settles it, and that style without the
// colors picked on a fill: the text on its own fill and the stroke and text on a ground. The
// diagram's own text and stroke are taken as a paint neither picks, so that a color picked
// differs from what the style would be without it even where it is the same as the default.
func inked(it *scene.Item) (got, plain style.Style) {
	var look style.Look
	own := it.Style
	own.Text, own.Stroke = style.SurfaceAlt, style.SurfaceAlt
	bare := own
	bare.OnFill, bare.Ground = false, style.Paint{}
	return look.Cascade(string(it.Role), own, it.Author), look.Cascade(string(it.Role), bare, it.Author)
}

// textPaint is the fill the oracle reads of an item's text: the color ink picks where the
// look picks one, and the diagram's own otherwise.
func textPaint(it *scene.Item) string {
	if got, plain := inked(it); got.Text != plain.Text {
		return inkNames[got.Text]
	}
	return paintName(it.Style.Text)
}

// strokeInk is the stroke the oracle reads of an edge's group, the color ink picks on the
// fill it lies in, or "" where it lies in none.
func strokeInk(it *scene.Item) string {
	if got, plain := inked(it); got.Stroke != plain.Stroke {
		return inkNames[got.Stroke]
	}
	return ""
}

// labelLine is the line the oracle writes of a label drawn about at: the point, the label's
// size, the fill of its text, and its spans.
func labelLine(it *scene.Item, l *text.Label, at scene.Point) string {
	var lines []string
	for _, line := range l.Lines {
		var spans []string
		for _, s := range line {
			if s.Icon != "" {
				spans = append(spans, "i:"+s.Icon+":"+n(s.Width))
			} else {
				spans = append(spans, strconv.Itoa(int(s.Face))+":"+s.Text+":"+n(s.Width))
			}
		}
		lines = append(lines, strings.Join(spans, " "))
	}
	return "t " + pt(at.X, at.Y) + " " + pt(l.Width, l.Height) + " " + textPaint(it) + " | " + strings.Join(lines, " | ")
}

// marker is the outline Pele's marker draws, every number in full, as the oracle reads it,
// filled and stroked as given where it is filled.
func marker(m scene.Marker, fill, stroke string) string {
	const arrow = scene.MarkerSize
	x, y, dx, dy := m.At.X, m.At.Y, m.DX, m.DY
	nx, ny := -dy, dx
	switch m.Type {
	case "arrow_point":
		bx, by := x-float64(dx*arrow), y-float64(dy*arrow)
		half := float64(arrow * 0.42)
		return "M " + pt(x, y) + " L " + pt(bx+float64(nx*half), by+float64(ny*half)) +
			" L " + pt(bx-float64(nx*half), by-float64(ny*half)) + " Z " + fill + " " + stroke
	case "arrow_circle":
		return "o " + pt(x-float64(dx*4), y-float64(dy*4)) + " 3.5 " + fill + " " + stroke
	case "arrow_cross":
		cx, cy := x-float64(dx*4), y-float64(dy*4)
		const a = 3.5
		ux, uy := float64((dx+nx)*a), float64((dy+ny)*a)
		vx, vy := float64((dx-nx)*a), float64((dy-ny)*a)
		return "M " + pt(cx-ux, cy-uy) + " L " + pt(cx+ux, cy+uy) + " M " + pt(cx-vx, cy-vy) + " L " + pt(cx+vx, cy+vy) + " - -"
	}
	return ""
}

// arc is an arc of Pele's path, as the oracle writes it.
type arc struct {
	rx, ry, rot  float64
	large, sweep bool
	to           scene.Point
	text         string
}

// arcs are the arcs of a line of Pele's, in order.
func arcs(line string) []arc {
	var out []arc
	fields := strings.Fields(line)
	for i, f := range fields {
		if f != "A" || i+2 >= len(fields) {
			continue
		}
		a := append(strings.Split(fields[i+1], ","), strings.Split(fields[i+2], ",")...)
		if len(a) != 7 {
			continue
		}
		var v [7]float64
		for k := range a {
			v[k], _ = strconv.ParseFloat(a[k], 64)
		}
		out = append(out, arc{v[0], v[1], v[2], v[3] == 1, v[4] == 1, scene.Point{X: v[5], Y: v[6]}, fields[i+1]})
	}
	return out
}

// pathLine writes a path as the oracle writes Pele's d. A run of curves is written as the
// arc of Pele's line at the same place, hints, where ArcTo draws that arc with exactly those
// curves; elsewhere as curves, which no line of Pele's has.
func pathLine(p *scene.Path, hints []arc) string {
	var out []string
	var last scene.Point
	ops, args := p.Ops, p.Args
	at := 0
	for i := 0; i < len(ops); {
		op := ops[i]
		a := args[at : at+op.Args()]
		if op != scene.CubeTo {
			s := string("MLQCZ"[op])
			for k := 0; k < len(a); k += 2 {
				s += " " + pt(a[k], a[k+1])
			}
			if len(a) > 0 {
				last = scene.Point{X: a[len(a)-2], Y: a[len(a)-1]}
			}
			out = append(out, s)
			i++
			at += op.Args()
			continue
		}
		j, end := i, at
		for j < len(ops) && ops[j] == scene.CubeTo {
			end += ops[j].Args()
			j++
		}
		curves := args[at:end]
		s := ""
		if len(hints) > 0 {
			h := hints[0]
			hints = hints[1:]
			q := new(scene.Path).MoveTo(last.X, last.Y).ArcTo(h.rx, h.ry, h.rot, h.large, h.sweep, h.to.X, h.to.Y)
			if slices.Equal(q.Args[2:], curves) && !slices.Contains(q.Ops[1:], scene.LineTo) {
				s = "A " + h.text + " " + pt(h.to.X, h.to.Y)
			}
		}
		if s == "" {
			s = "C"
			for k := 0; k < len(curves); k += 2 {
				s += " " + pt(curves[k], curves[k+1])
			}
		}
		out = append(out, s)
		last = scene.Point{X: curves[len(curves)-2], Y: curves[len(curves)-1]}
		i, at = j, end
	}
	return strings.Join(out, " ")
}

// partsOf writes the parts of an outline, numbers as Pele writes them: the center of a block,
// taken from its box, may be off the one it was drawn about in the last bit.
func partsOf(parts []scene.Part) string {
	var b strings.Builder
	num := pelesvg.Num
	for _, p := range parts {
		r := p.Box
		fmt.Fprintf(&b, "%s,%s,%s,%s %v %d", num(r.X), num(r.Y), num(r.W), num(r.H), p.Rounded, p.Paint)
		if p.Path != nil {
			b.WriteString(" " + pelesvg.D(p.Path))
		}
		b.WriteString("; ")
	}
	return b.String()
}

// sceneLines writes a scene as the oracle reads Pele's SVG; want is the line Pele drew at an
// index, or "" where the fixture does not hold it, and the arcs of its lines are hints for
// writing the scene's curves. The oracle keeps the lines of paths with arcs in full.
func sceneLines(sc *scene.Scene, want func(int) string) []string {
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	hint := func() string { return want(len(lines)) }
	for i := range sc.Items {
		it := &sc.Items[i]
		id := pelesvg.ID(it.ID)
		classes := strings.Join(it.Class, " ")
		switch it.Role {
		case RoleTitle:
			lines = append(lines, "title "+labelLine(it, it.Label, it.At))
		case RoleComposite:
			r := it.Parts[0].Box
			s := fmt.Sprintf("cluster %s [%s] %s,%s", id, classes, pt(r.X, r.Y), pt(r.W, r.H))
			if it.Parts[0].Rounded {
				s += " r"
			}
			if r != it.Box {
				s += " box differs"
			}
			lines = append(lines, s+" "+partPaint(it, &it.Parts[0]), "pen "+pelesvg.CapJoin(it))
			if it.Label != nil {
				lines = append(lines, labelLine(it, it.Label, it.At))
			}
		case RoleBlock:
			r := it.Box
			lines = append(lines, fmt.Sprintf("node %s %s [%s] %s,%s", id, it.Shape, classes, pt(r.X, r.Y), pt(r.W, r.H)))
			for k := range it.Parts {
				part := &it.Parts[k]
				lines = append(lines, "part "+partPaint(it, part))
				if it.Shape == "block-arrow" {
					var p []string
					part.Path.Each(func(op scene.Op, a []float64) {
						if op != scene.Close {
							p = append(p, pt(a[0], a[1]))
						}
					})
					lines = append(lines, "polygon "+strings.Join(p, " "))
				}
			}
			if it.Shape != "block-arrow" {
				cx, cy := r.X+r.W/2, r.Y+r.H/2
				if partsOf(it.Parts) != partsOf(draw.Outline(string(it.Shape), cx, cy, r.W, r.H)) {
					lines = append(lines, "parts differ from the outline")
				}
			}
			if len(it.Parts) > 0 {
				lines = append(lines, "pen "+pelesvg.CapJoin(it))
			}
			if it.Label != nil {
				lines = append(lines, labelLine(it, it.Label, it.At))
			}
		case RoleEdge:
			dash, width := "-", "-"
			if it.Style.Dash.Set() {
				var parts []string
				for _, v := range it.Style.Dash.Pattern() {
					parts = append(parts, n(v))
				}
				dash = strings.Join(parts, " ")
			}
			if w, ok := it.Style.StrokeWidth.Get(); ok {
				width = n(w)
			}
			over := strokeInk(it)
			color := over
			if over == "" {
				over, color = "-", "var(--_l)"
			}
			lines = append(lines, fmt.Sprintf("edge %s %s %s %s", id, dash, width, over))
			lines = append(lines, "path "+pathLine(it.Parts[0].Path, arcs(hint())))
			lines = append(lines, "pen "+pelesvg.CapJoin(it))
			for _, m := range it.Ends {
				if s := marker(m, color, "none"); s != "" {
					lines = append(lines, "m "+s)
				}
			}
		case RoleCasing:
			// A casing is written as the oracle reads Pele's: its width where it is not the
			// casings' 5, its path, and the markers of its edge, drawn in the background.
			width := "-"
			if w := it.Style.StrokeWidth.Or(1); w != 5 {
				width = n(w)
			}
			if _, ok := it.Style.Opacity.Get(); ok || it.Style.StrokeOpacity != (style.Num{}) {
				width += " opacity"
			}
			lines = append(lines, "casing "+width+" -")
			lines = append(lines, "path "+pathLine(it.Parts[0].Path, arcs(hint())))
			var edge *scene.Item
			for k := i + 1; k < len(sc.Items) && edge == nil; k++ {
				if sc.Items[k].Role == RoleEdge && sc.Items[k].ID == it.ID {
					edge = &sc.Items[k]
				}
			}
			if edge == nil {
				lines = append(lines, "casing of no edge")
				break
			}
			var outline []scene.Part
			for _, m := range edge.Ends {
				outline = append(outline, m.Outline()...)
				if s := marker(m, "var(--_bg)", "var(--_bg)"); s != "" {
					lines = append(lines, "cm "+s)
				}
			}
			if len(outline) != len(it.Parts)-1 {
				lines = append(lines, "casing's markers differ")
			}
			for k := range outline {
				if k+1 < len(it.Parts) && pelesvg.D(outline[k].Path) != pelesvg.D(it.Parts[k+1].Path) {
					lines = append(lines, "casing's markers differ")
				}
			}
		case RoleEdgeLabel:
			r := it.Box
			if pelesvg.D(it.Parts[0].Path) != pelesvg.D(new(scene.Path).RoundRect(r, draw.LabelPatchRadius)) {
				lines = append(lines, "patch not the label's box")
			}
			lines = append(lines, "elabel "+id+" "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+paintName(it.Style.Fill))
			l := it.Label
			shift := float64(l.Size * 0.35)
			lines = append(lines, "t "+pt(it.At.X, it.At.Y-l.Height/2+l.LineHeight/2+shift))
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
