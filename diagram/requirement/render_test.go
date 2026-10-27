package requirement

import (
	"encoding/json"
	"fmt"
	"strconv"
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

// The fixture is what Pele's render draws of requirement diagrams, read back out of its SVG
// as one line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. The test
// writes the same lines from the scene, numbers as Pele's SVG writes them.

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
		got := sceneLines(Layout(db, pre.Config, &layout.Options{MaxWidth: c.MaxWidth}))
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v), %s", short(c.Src), c.MaxWidth, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
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

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	box := func(r scene.Rect, cx, cy float64) string {
		return pt(r.X-cx, r.Y-cy) + "," + num(r.W) + "," + num(r.H)
	}
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Title != "" {
		lines = append(lines, "acc-title "+peleText(sc.Title))
	}
	if sc.Desc != "" {
		lines = append(lines, "acc-desc "+peleText(sc.Desc))
	}
	// text writes an item's label, drawn about x, y, as the oracle reads Pele's text element.
	text := func(it *scene.Item, class, size, weight string, x, y float64) {
		if s := textLine(it.Label, it.Align, class, size, weight, x, y); s != "" {
			lines = append(lines, s+ink(it))
		}
	}
	labels := false
	for i := 0; i < len(sc.Items); i++ {
		it := &sc.Items[i]
		switch it.Role {
		case RoleTitle:
			text(it, "pele-title", "", "var(--_tw)", it.At.X, it.At.Y)
		case RoleRelationship:
			kind := "arrow"
			if it.Ends[0].Type == containsMarker {
				kind = "contains"
			}
			lines = append(lines, fmt.Sprintf("edge %s %s", peleText(it.ID), kind))
			dashed := ""
			if it.Style.Dash.Set() {
				dashed = " dashed"
			}
			lines = append(lines, "path "+pelesvg.D(it.Parts[0].Path)+dashed+" "+pen(it, ""))
			for _, m := range it.Ends {
				for _, part := range m.Outline() {
					lines = append(lines, "m "+markerPart(part.Path)+" "+pen(it, m.Type))
				}
			}
		case RoleRelationshipLabel:
			if !labels {
				lines = append(lines, "labels "+num(it.Label.Size))
				labels = true
			}
			lines = append(lines, "label "+peleText(it.ID), "rect "+box(it.Box, 0, 0)+" "+pen(it, ""))
			text(it, "-", "", "", it.At.X, it.At.Y)
		case RoleRequirement, RoleElement:
			// Pele draws a box about its middle, which it moves to with a translate. The scene
			// places its parts where they end up, so the numbers Pele writes are worked out
			// here as its render.ts works them out, from the sizes of the box and its texts,
			// and the scene is held to placing each part exactly there. Taking the middle off
			// where the scene places a part would be off by a bit in the last place, which
			// shows when a number falls on a half of a hundredth.
			cx, cy := it.At.X, it.Box.Y+it.Box.H/2
			kind := strings.TrimPrefix(string(it.Role), "requirement.")
			w, h := it.Box.W, it.Box.H
			left, top := -w/2, -h/2
			place := func(what string, x, y, relX, relY float64) {
				if x != cx+relX || y != cy+relY {
					lines = append(lines, fmt.Sprintf("the scene places the %s at %v,%v, not %v,%v", what, x, y, cx+relX, cy+relY))
				}
			}
			lines = append(lines,
				fmt.Sprintf("node %s %s [%s] translate(%s)", peleText(it.ID), kind, strings.Join(it.Class, " "), pt(cx, cy)),
				"rect "+pt(left, top)+","+num(w)+","+num(h)+" "+pen(it, ""))
			place("box", it.Box.X, it.Box.Y, left, top)
			k := &sc.Items[i+1]
			i++
			kindY := top + 10 + k.Label.Height/2
			place("kind", k.At.X, k.At.Y, 0, kindY)
			text(k, "pele-stereotype", num(k.Label.Size), "", 0, kindY)
			nameY := top + 10 + k.Label.Height + it.Label.Height/2
			place("name", it.At.X, it.At.Y, 0, nameY)
			text(it, "pele-label", num(it.Label.Size), "var(--_hw)", 0, nameY)
			var fields []*scene.Item
			for i+1 < len(sc.Items) && (sc.Items[i+1].Role == RoleField || sc.Items[i+1].Role == RoleValue) {
				i++
				fields = append(fields, &sc.Items[i])
			}
			if len(it.Parts) > 1 {
				divider := top + (20 + k.Label.Height + it.Label.Height)
				var xs, ys []float64
				it.Parts[1].Path.Each(func(_ scene.Op, a []float64) { xs, ys = append(xs, a[0]), append(ys, a[1]) })
				place("divider", xs[0], ys[0], left, divider)
				place("divider's end", xs[1], ys[1], -left, divider)
				lines = append(lines, "divider M"+pt(left, divider)+"H"+num(-left)+" "+pen(it, ""), "fields "+num(fields[0].Label.Size))
				keyWidth := 0.0
				for f := 0; f < len(fields); f += 2 {
					keyWidth = max(keyWidth, fields[f].Label.Width)
				}
				at := divider + 10
				for f := 0; f+1 < len(fields); f += 2 {
					key, value := fields[f], fields[f+1]
					keyX, valueX := left+14, left+14+keyWidth+12
					place("field", key.At.X, key.At.Y, keyX, at+key.Label.Height/2)
					place("value", value.At.X, value.At.Y, valueX, at+value.Label.Height/2)
					text(key, "pele-field", "", "", keyX, at+key.Label.Height/2)
					text(value, "pele-value", "", "", valueX, at+value.Label.Height/2)
					at += max(key.Label.Height, value.Label.Height) + 2
				}
			}
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
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

// markerPart writes a part of a marker as the oracle reads Pele's: a circle by its middle and
// radius, a path by its d.
func markerPart(p *scene.Path) string {
	var ops []scene.Op
	var args [][]float64
	p.Each(func(op scene.Op, a []float64) {
		ops = append(ops, op)
		args = append(args, append([]float64(nil), a...))
	})
	if len(ops) > 1 && ops[1] == scene.CubeTo {
		// An ellipse: it starts at the right of its middle, and its first curve ends below it.
		cx, cy := args[1][4], args[0][1]
		return "o " + pelesvg.Num(cx) + "," + pelesvg.Num(cy) + "," + pelesvg.Num(args[0][0]-cx)
	}
	return pelesvg.D(p)
}

// textLine writes a label as the oracle reads the text element Pele writes for it: centered
// on x, y as Pele's labelSvg writes it, or with its lines starting at x, as its startText
// does.
func textLine(l *text.Label, align scene.Align, class, size, weight string, x, y float64) string {
	num := pelesvg.Num
	shared := sharedFace(l)
	italic := 0
	if weight == "" && shared&text.Bold != 0 {
		weight = "var(--_w)"
	}
	if weight == "" {
		weight = "-"
	}
	if size == "" {
		size = "-"
	}
	if shared&text.Italic != 0 {
		italic = 1
	}
	anchor := "middle"
	if align == scene.Start {
		anchor = "start"
	}
	var spans []string
	span := func(pos string, f text.Face, s string) {
		if f&text.Bold != 0 {
			pos += "+b"
		}
		if f&text.Italic != 0 {
			pos += "+i"
		}
		var q strings.Builder
		enc := json.NewEncoder(&q)
		enc.SetEscapeHTML(false)
		enc.Encode(dropControls(s))
		// JSON.stringify leaves the line and paragraph separators as they are; Go escapes them.
		quoted := strings.NewReplacer(`\u2028`, "\u2028", `\u2029`, "\u2029").Replace(strings.TrimSuffix(q.String(), "\n"))
		spans = append(spans, pos+":"+quoted)
	}
	shift := float64(l.Size * 0.35)
	top := y - l.Height/2
	ly := top + l.LineHeight/2 + shift
	if align == scene.Start {
		for i, line := range l.Lines {
			// startText places each line by its middle, as Pele's lineMiddle gives it.
			ly = top + float64((float64(i)+0.5)*l.LineHeight) + shift
			first := true
			for _, s := range line {
				if s.Icon != "" || s.Text == "" {
					continue
				}
				pos := ""
				if first {
					pos = "@" + num(x) + "," + num(ly)
				}
				span(pos, s.Face&^shared, s.Text)
				first = false
			}
		}
	} else if len(l.Lines) == 1 && len(l.Lines[0]) == 1 && l.Lines[0][0].Icon == "" && l.Lines[0][0].Face == 0 {
		// Pele writes a label of one plain run without a tspan, and no emphasis.
		span("@"+num(x)+","+num(ly), 0, l.Lines[0][0].Text)
		italic = 0
	} else {
		for _, line := range l.Lines {
			for j, s := range line {
				pos := ""
				if j == 0 {
					pos = "@" + num(x) + "," + num(ly)
				}
				span(pos, s.Face&^shared, s.Text)
			}
			if len(line) == 0 {
				span("@"+num(x)+","+num(ly), 0, " ")
			}
			top += l.LineHeight
			ly = top + l.LineHeight/2 + shift
		}
	}
	if len(spans) == 0 {
		return ""
	}
	return "text " + class + " fs=" + size + " w=" + weight + " i=" + strconv.Itoa(italic) + " a=" + anchor + " " + strings.Join(spans, " ")
}

// sharedFace is the emphasis every run of a label has, as Pele's sharedStyle finds it.
func sharedFace(l *text.Label) text.Face {
	shared := text.Bold | text.Italic
	any := false
	for _, line := range l.Lines {
		for _, s := range line {
			if s.Icon != "" || s.Text == "" {
				continue
			}
			shared &= s.Face
			any = true
		}
	}
	if !any {
		return 0
	}
	return shared
}

// dropControls drops the characters XML does not allow, as Pele's esc does.
func dropControls(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, s)
}

// peleText is text as Pele writes it with escText: entities decoded, characters XML does not
// allow dropped.
func peleText(s string) string { return dropControls(text.DecodeEntities(s)) }
