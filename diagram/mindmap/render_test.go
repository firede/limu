package mindmap

import (
	"fmt"
	"math"
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
)

// The fixture is what Pele's render draws of mindmaps, read back out of its SVG as one line
// per thing drawn, in the order drawn, with every number in full; ./scripts/oracle.sh writes
// it. Pele draws the map about a point of its own and the inside of a node about the node; the
// oracle moves them to where they are on the page, as the scene holds them. Each label is
// written as the point it is drawn about and what is in it, and each node as its box. The
// test writes the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src        string  `json:"src"`
		MaxWidth   float64 `json:"maxWidth"`
		FontSize   float64 `json:"fontSize"`
		Breakpoint float64 `json:"breakpoint"`
		Keep       bool    `json:"keep"`
		Icons      bool    `json:"icons"`
		fixture.Drawing
	} `json:"cases"`
}

// hostIcons are the icons the host has in the cases drawn with icons, as the oracle's ICONS.
var hostIcons = []string{"fa fa-book", "star", "phone", "fa:fa-car", "mdi:home", "fa fa-x"}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	icons := func(name string) bool { return slices.Contains(hostIcons, name) }
	failed := 0
	for k, c := range p.Cases {
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
		o := &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize, DirectionBreakpoint: c.Breakpoint, KeepDirection: c.Keep}
		if c.Icons {
			o.Icons = icons
		}
		got := sceneLines(t, Layout(m, pre.Config, o))
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, %+v), %s", c.Src, *o, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Surface: "var(--_s)",
	style.Border:  "var(--_b)",
}

// series are Pele's series colors, as its seriesColor writes them.
var series = []string{"#4c78a8", "#f58518", "#54a24b", "#e45756", "#72b7b2", "#eeca3b", "#b279a2", "#9d755d"}

// paintName is a paint as Pele's SVG names it, "-" for none given, with its opacity after a
// slash.
func paintName(p style.Paint) string {
	if !p.Set() || p == style.None {
		return "-"
	}
	if s, ok := paintNames[p]; ok {
		return s
	}
	for k := range series {
		if p == style.Series(k) {
			return fmt.Sprintf("var(--pele-series-%d,%s)", k+1, series[k])
		}
		if p == style.Series(k).Alpha(tint) {
			return fmt.Sprintf("var(--pele-series-%d,%s)/%s", k+1, series[k], chars.NumberString(tint))
		}
	}
	return "?"
}

// sceneLines writes a scene as the oracle reads Pele's SVG, every number as JavaScript's
// String writes it.
func sceneLines(t *testing.T, sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	label := func(it *scene.Item, class string) {
		l := it.Label
		if l == nil {
			return
		}
		fill := "-"
		if it.Style.Text != style.Fg {
			fill = paintName(it.Style.Text)
		}
		align := "middle"
		if it.Align == scene.Start {
			align = "start"
		}
		var widths, spans []string
		for _, w := range l.Widths {
			widths = append(widths, n(w))
		}
		for _, line := range l.Lines {
			var s []string
			for _, span := range line {
				if span.Icon != "" {
					s = append(s, "i:"+span.Icon+":"+n(span.Width))
				} else {
					s = append(s, "t:"+strconv.Itoa(int(span.Face))+":"+span.Text+":"+n(span.Width))
				}
			}
			spans = append(spans, strings.Join(s, "|"))
		}
		lines = append(lines, "label "+class+" "+fill+" "+align+" "+pt(it.At.X, it.At.Y)+" "+
			n(l.Size)+","+n(l.LineHeight)+","+n(l.Width)+","+n(l.Height)+" | "+strings.Join(widths, ",")+" | "+strings.Join(spans, " / "))
	}
	width := func(it *scene.Item) string {
		if v, ok := it.Style.StrokeWidth.Get(); ok {
			return n(v)
		}
		return "-"
	}
	section := -1
	for i := range sc.Items {
		it := &sc.Items[i]
		// Pele draws each first-level branch in a group of its own, its links first.
		if it.Role == RoleEdge || it.Role == RoleNode {
			s, err := strconv.Atoi(strings.TrimPrefix(it.Tags[0], "section-"))
			if err != nil {
				t.Errorf("item %d has no section: %v", i, it.Tags)
			}
			if s != section || it.Role == RoleEdge && i > 0 && sc.Items[i-1].Role == RoleNode {
				lines = append(lines, "branch "+strconv.Itoa(s))
				section = s
			}
		}
		switch it.Role {
		case RoleTitle:
			label(it, "pele-title")
		case RoleEdge:
			var steps []string
			it.Parts[0].Path.Each(func(op scene.Op, a []float64) {
				s := string("MLQCZ"[op])
				for k := 0; k < len(a); k += 2 {
					s += " " + pt(a[k], a[k+1])
				}
				steps = append(steps, s)
			})
			fill := "?"
			if it.Parts[0].Paint == scene.StrokeOnly {
				fill = "none"
			}
			lines = append(lines, "edge "+strings.Join(steps, " ")+" "+fill+" "+paintName(it.Style.Stroke)+" "+width(it)+" "+pelesvg.CapJoin(it))
		case RoleNode, RoleRoot:
			shape := string(it.Shape)
			if shape == rootShape && it.Role == RoleNode {
				shape = "branch"
			}
			root := ""
			classes := it.Class
			if it.Role == RoleRoot {
				root = " root"
			}
			b := it.Box
			lines = append(lines, fmt.Sprintf("node %s %s%s [%s] %s,%s", pelesvg.ID(it.ID), shape, root, strings.Join(classes, " "), pt(b.X, b.Y), pt(b.W, b.H)))
			for _, part := range it.Parts {
				if part.Paint != scene.FillAndStroke {
					lines = append(lines, "part painted "+strconv.Itoa(int(part.Paint)))
					continue
				}
				r := "-"
				if part.Rounded {
					r = "r"
				}
				lines = append(lines, "part "+r+" "+paintName(pelesvg.OwnFill(it))+" "+paintName(it.Style.Stroke)+" "+width(it)+" "+pelesvg.CapJoin(it))
			}
			frame := scene.Item{Parts: draw.Outline(string(it.Shape), 0, 0, b.W, b.H)}
			frame.Move(b.X+b.W/2, b.Y+b.H/2)
			if !sameParts(it.Parts, frame.Parts) {
				lines = append(lines, "parts differ from the outline")
			}
			if pic := it.Picture; pic != nil {
				lines = append(lines, "icon "+pelesvg.ID(pic.Icon)+" "+pt(pic.Box.X, pic.Box.Y)+","+pt(pic.Box.W, pic.Box.H))
			}
			label(it, "pele-label")
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}

// sameParts reports whether a node's parts are the outline drawn about the center of its box.
// The center taken from the box may be off the one the node was drawn about in the last bit,
// and the outline was moved onto the page after it was drawn, so the numbers are allowed
// that much. The box of a part that is a path means nothing, and Item.Move moves it all the
// same, so it is left out. Pele's outlines of the shapes are compared in package draw.
func sameParts(got, want []scene.Part) bool {
	near := func(a, b float64) bool { return math.Abs(a-b) <= 1e-9*max(1, math.Abs(a)) }
	if len(got) != len(want) {
		return false
	}
	for k := range got {
		g, w := got[k], want[k]
		if g.Rounded != w.Rounded || g.Paint != w.Paint || (g.Path == nil) != (w.Path == nil) {
			return false
		}
		if g.Path == nil {
			if !near(g.Box.X, w.Box.X) || !near(g.Box.Y, w.Box.Y) || !near(g.Box.W, w.Box.W) || !near(g.Box.H, w.Box.H) {
				return false
			}
			continue
		}
		if !slices.Equal(g.Path.Ops, w.Path.Ops) || len(g.Path.Args) != len(w.Path.Args) {
			return false
		}
		for j := range g.Path.Args {
			if !near(g.Path.Args[j], w.Path.Args[j]) {
				return false
			}
		}
	}
	return true
}
