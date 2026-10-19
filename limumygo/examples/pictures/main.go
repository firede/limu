// Command pictures draws the pictures of limu's website that show one scene drawn four
// ways: as the data it is, in Kohu, in KohuDark, and as Liquid Glass, which replaces how
// each item is drawn. The diagram is laid out once, and every picture is drawn from that
// scene. It writes them to a directory, where the website serves them by default:
//
//	go run ./examples/pictures
//	go run ./examples/pictures -out pictures
//
// The scene is written as SVG in currentColor, so that a page can color it. The others
// are drawn with MyGo and written as PNG, at twice their size in DIPs by default. They are rendered on a Mac and committed, as MyGo
// sets text with Pango on Linux and would draw them otherwise.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
	"github.com/firede/limu"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/limumygo"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// source is the diagram every picture draws.
const source = `flowchart LR
  draft[Draft] --> review[Review]
  review -->|approve| ship([Ship])
  review -.->|revise| draft`

// pad is the space around the diagram in each picture, in DIPs.
const pad = 16

func main() {
	out := flag.String("out", "../website/public/pictures", "the directory to write the pictures to")
	scale := flag.Float64("scale", 2, "device pixels per DIP of the PNG pictures")
	flag.Parse()

	sc, err := limu.Layout(source, &layout.Options{Measurer: limumygo.DefaultFonts})
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(*out, 0o755); err != nil {
		log.Fatal(err)
	}
	write := func(name string, b []byte) {
		if err := os.WriteFile(filepath.Join(*out, name), b, 0o644); err != nil {
			log.Fatal(err)
		}
	}
	write("scene.svg", []byte(wireframe(sc)))
	k := float32(*scale)
	for _, p := range []struct {
		name string
		img  *image.RGBA
	}{
		{"kohu.png", render(sc, themed(style.Kohu()), k)},
		{"kohu-dark.png", render(sc, themed(style.KohuDark()), k)},
		{"glass.png", render(sc, glassy(), k)},
	} {
		var b strings.Builder
		if err := png.Encode(&b, p.img); err != nil {
			log.Fatal(err)
		}
		write(p.name, []byte(b.String()))
	}
}

// A look is how a picture draws the scene: what it paints under it, and the options it
// draws it with.
type look struct {
	ground func(p *ui.Painter, r ui.Rect)
	opts   *limumygo.Options
}

// themed draws the scene the default way, in a theme, on the theme's background.
func themed(th *style.Theme) look {
	return look{
		ground: func(p *ui.Painter, r ui.Rect) { p.Fill(r, limumygo.Color(th.Bg, 1), 0) },
		opts:   &limumygo.Options{Look: style.Look{Theme: th}, Fonts: limumygo.DefaultFonts},
	}
}

// render draws the scene in a look, with pad around it, at scale device pixels per DIP.
func render(sc *scene.Scene, l look, scale float32) *image.RGBA {
	w, h := int(sc.Width+0.5)+2*pad, int(sc.Height+0.5)+2*pad
	return ui.Render(func(c *ui.Context) {
		ui.Box(c).Size(float32(w), float32(h)).Draw(func(p *ui.Painter, r ui.Rect) {
			l.ground(p, r)
			limumygo.Paint(p, sc, r.X+pad, r.Y+pad, l.opts)
		})
	}, w, h, scale)
}

// glassy is Liquid Glass: soft colors and a grid under the diagram, its lines and text in
// white, and each node and edge label a pane of clear glass, which what is under it shows
// through, bent at its rim.
func glassy() look {
	th := *style.Kohu()
	th.Line = style.Hex("#ffffff")
	th.Fg = style.Hex("#ffffff")
	const away = 4000
	ground := func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, ui.LinearGradient{From: ui.Hex("#23817d"), To: ui.Hex("#3d6aa3"), Angle: 135, Oklab: true}, 0)
		// Glows of the series colors, which the glass bends where it passes over them: the
		// shadow of a disc far to the left, cast back onto the picture, as a shadow is not
		// painted under what casts it.
		for _, g := range []struct {
			x, y, d float32
			c       string
		}{{0.12, 0.15, 0.7, "#e37d90"}, {0.55, 0.95, 0.8, "#d9a748"}, {0.88, 0.2, 0.6, "#8cbf6a"}, {0.4, 0.1, 0.4, "#ad92d1"}} {
			d := g.d * r.H
			b := ui.Rect{X: r.X + g.x*r.W - d/2 - away, Y: r.Y + g.y*r.H - d/2, W: d, H: d}
			p.Shadow(b, d/2, away, 0, d/2, d/8, ui.Hex(g.c))
		}
		// A grid of dark lines, as on a canvas, which the glass bends at its rim, and which the
		// white of the diagram stands out over.
		line := ui.RGBA(6, 21, 24, 0.4)
		for x := r.X + 8; x < r.X+r.W; x += 16 {
			p.Line(x, r.Y, x, r.Y+r.H, 1.25, line)
		}
		for y := r.Y + 8; y < r.Y+r.H; y += 16 {
			p.Line(r.X, y, r.X+r.W, y, 1.25, line)
		}
	}
	draw := func(p *limumygo.Painter, it *scene.Item, s style.Resolved, next func()) {
		var r ui.Rect
		switch {
		case it.Kind == scene.Node:
			r = p.Rect(it.Box)
		case it.Kind == scene.Text && it.Label != nil && !it.Label.Empty():
			b := it.LabelBox()
			r = p.Rect(scene.Rect{X: b.X - 8, Y: b.Y - 2, W: b.W + 16, H: b.H + 4})
		default:
			next()
			return
		}
		// A box keeps its corners rounded as the glass's, and any other outline, as a
		// stadium's, rounds them fully.
		radius := r.H / 2
		if len(it.Parts) > 0 && it.Parts[0].Path == nil {
			radius = min(radius, p.Len(12))
		}
		glass.Paint(p.Painter, r, radius, glass.Glass{Style: glass.Clear})
		p.Label(it, s)
	}
	return look{ground: ground, opts: &limumygo.Options{Look: style.Look{Theme: &th}, Fonts: limumygo.DefaultFonts, Draw: draw}}
}

// wireframe writes the scene as SVG with no style: each item's box dashed, its parts and
// markers outlined, an edge's route as dots at its corners, and each label's box with its
// text, all in currentColor.
func wireframe(sc *scene.Scene) string {
	var b strings.Builder
	w, h := sc.Width+2*pad, sc.Height+2*pad
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %s %s" width="%s" height="%s" fill="none" stroke="currentColor" stroke-width="1" font-family="system-ui, sans-serif">`, num(w), num(h), num(w), num(h))
	fmt.Fprintf(&b, `<g transform="translate(%d %d)">`, pad, pad)
	rect := func(r scene.Rect, attrs string) {
		fmt.Fprintf(&b, `<rect x="%s" y="%s" width="%s" height="%s" %s/>`, num(r.X), num(r.Y), num(r.W), num(r.H), attrs)
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		if it.Kind != scene.Edge && it.Box.W > 0 && it.Box.H > 0 {
			rect(it.Box, `stroke-dasharray="3 3" opacity=".45"`)
		}
		parts := it.Parts
		for _, m := range it.Ends {
			parts = append(parts[:len(parts):len(parts)], m.Outline()...)
		}
		for _, part := range parts {
			if part.Path == nil {
				rect(part.Box, "")
				continue
			}
			fmt.Fprintf(&b, `<path d="%s"/>`, pathData(part.Path))
		}
		if it.Kind == scene.Edge {
			for _, pt := range it.Route {
				fmt.Fprintf(&b, `<circle cx="%s" cy="%s" r="2" fill="currentColor" stroke="none" opacity=".6"/>`, num(pt.X), num(pt.Y))
			}
		}
		if l := it.Label; l != nil && !l.Empty() {
			lb := it.LabelBox()
			rect(lb, `stroke-dasharray="1 2" fill="currentColor" fill-opacity=".06"`)
			anchor, x := "middle", it.At.X
			switch it.Align {
			case scene.Start:
				anchor = "start"
			case scene.End:
				anchor = "end"
			}
			for j, line := range l.Lines {
				var t strings.Builder
				for _, sp := range line {
					t.WriteString(sp.Text)
				}
				y := lb.Y + (float64(j)+0.5)*l.LineHeight
				fmt.Fprintf(&b, `<text x="%s" y="%s" font-size="%s" text-anchor="%s" dominant-baseline="central" fill="currentColor" stroke="none" opacity=".8">%s</text>`,
					num(x), num(y), num(l.Size), anchor, escape(t.String()))
			}
		}
	}
	b.WriteString("</g></svg>\n")
	return b.String()
}

// pathData writes a path as the d attribute of an SVG path.
func pathData(p *scene.Path) string {
	var b strings.Builder
	p.Each(func(op scene.Op, args []float64) {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(map[scene.Op]string{scene.MoveTo: "M", scene.LineTo: "L", scene.QuadTo: "Q", scene.CubeTo: "C", scene.Close: "Z"}[op])
		for _, v := range args {
			b.WriteByte(' ')
			b.WriteString(num(v))
		}
	})
	return b.String()
}

// num writes a number with at most two decimals.
func num(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "-0" {
		return "0"
	}
	return s
}

func escape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
