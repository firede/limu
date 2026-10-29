// Command pictures draws the pictures of limu's website that show one scene drawn four
// ways: as the data it is, in Kohu, in KohuDark, and as Liquid Glass, which replaces how
// each item is drawn. The diagram is laid out once, and every picture is drawn from that
// scene. It writes them to a directory, where the website serves them by default:
//
//	go run ./examples/pictures
//	go run ./examples/pictures -out pictures
//
// The scene is written as SVG in currentColor, so that a page can color it. The others
// are drawn with MyGo and written as PNG, at twice their size in DIPs by default.
//
// Given the website's font, Hanken Grotesk, as a TrueType file, it also draws the pictures
// a link to the website or to the repository shows: og.png, 1200 by 630, and github.png,
// 1280 by 640, for the repository's social preview.
//
//	go run ./examples/pictures -font HankenGrotesk.ttf
//
// The pictures are rendered on a Mac and committed, as MyGo sets text with Pango on Linux
// and would draw them otherwise.
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
	"github.com/egoist/mygo/ui/shader"
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
	// The website, firede/limu-website, is checked out beside limu.
	out := flag.String("out", "../../limu-website/public/pictures", "the directory to write the pictures to")
	scale := flag.Float64("scale", 2, "device pixels per DIP of the PNG pictures")
	font := flag.String("font", "", "Hanken Grotesk as a TrueType file, to draw the pictures for links with")
	mark := flag.String("mark", "../../limu-website/public/mark.svg", "limu's mark as SVG")
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
	pngs := []struct {
		name string
		img  *image.RGBA
	}{
		{"kohu.png", render(sc, themed(style.Kohu()), k)},
		{"kohu-dark.png", render(sc, themed(style.KohuDark()), k)},
		{"glass.png", render(sc, glassy(limumygo.DefaultFonts), k)},
	}
	if *font != "" {
		data, err := os.ReadFile(*font)
		if err != nil {
			log.Fatal(err)
		}
		if err := ui.RegisterFont(data, cardFont); err != nil {
			log.Fatal(err)
		}
		svg, err := os.ReadFile(*mark)
		if err != nil {
			log.Fatal(err)
		}
		m, err := ui.ParseSVG(svg)
		if err != nil {
			log.Fatal(err)
		}
		pngs = append(pngs, struct {
			name string
			img  *image.RGBA
		}{"og.png", card(1200, 630, m)}, struct {
			name string
			img  *image.RGBA
		}{"github.png", card(1280, 640, m)})
	}
	for _, p := range pngs {
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

// glassy is Liquid Glass, with text in fonts: the canvas under the diagram, its lines and
// text in white, and each node and edge label a pane of clear glass, which what is under it
// shows through, bent at its rim.
func glassy(fonts *limumygo.Fonts) look {
	th := *style.Kohu()
	th.Line = style.Hex("#ffffff")
	th.Fg = style.Hex("#ffffff")
	draw := func(p *limumygo.Painter, it *scene.Item, s style.Resolved, next func()) {
		var r ui.Rect
		switch {
		case it.Kind == scene.Node:
			r = p.Rect(it.Box)
		case it.Kind == scene.Text && it.Label != nil && !it.Label.Empty():
			b, em := it.LabelBox(), it.Label.Size
			r = p.Rect(scene.Rect{X: b.X - em/2, Y: b.Y - em/8, W: b.W + em, H: b.H + em/4})
		default:
			next()
			return
		}
		// A box's corners are rounded by a quarter of its height, and any other outline, as
		// a stadium's, is rounded fully.
		radius := r.H / 2
		if len(it.Parts) > 0 && it.Parts[0].Path == nil {
			radius = r.H / 4
		}
		glass.Paint(p.Painter, r, radius, glass.Glass{Style: glass.Clear})
		p.Label(it, s)
	}
	return look{ground: canvas(16), opts: &limumygo.Options{Look: style.Look{Theme: &th}, Fonts: fonts, Draw: draw}}
}

// canvas paints what is under the glass: a gradient of limu's teal and blue, glows of the
// series colors, which the glass bends where it passes over them, and a grid of dark lines
// cell DIPs apart, as on a canvas, which it bends at its rim, and which the white of a
// diagram stands out over.
func canvas(cell float32) func(p *ui.Painter, r ui.Rect) {
	// A glow is the shadow of a disc far to the left, cast back onto the picture, as a
	// shadow is not painted under what casts it.
	const away = 8000
	return func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, ui.LinearGradient{From: ui.Hex("#23817d"), To: ui.Hex("#3d6aa3"), Angle: 135, Oklab: true}, 0)
		for _, g := range []struct {
			x, y, d float32
			c       string
		}{{0.12, 0.15, 0.7, "#e37d90"}, {0.55, 0.95, 0.8, "#d9a748"}, {0.88, 0.2, 0.6, "#8cbf6a"}, {0.4, 0.1, 0.4, "#ad92d1"}} {
			d := g.d * r.H
			b := ui.Rect{X: r.X + g.x*r.W - d/2 - away, Y: r.Y + g.y*r.H - d/2, W: d, H: d}
			p.Shadow(b, d/2, away, 0, d/2, d/8, ui.Hex(g.c))
		}
		line := ui.RGBA(6, 21, 24, 0.4)
		for x := r.X + cell/2; x < r.X+r.W; x += cell {
			p.Line(x, r.Y, x, r.Y+r.H, 1.25, line)
		}
		for y := r.Y + cell/2; y < r.Y+r.H; y += cell {
			p.Line(r.X, y, r.X+r.W, y, 1.25, line)
		}
	}
}

// cardFont is the family the website's font is registered as.
const cardFont = "Hanken Grotesk"

// cardSource is the diagram a picture for links draws: how limu works, read from the
// bottom up, as the bubbles it is drawn in rise.
const cardSource = `flowchart BT
  mermaid((Mermaid)) -->|layout| scene((Scene))
  scene --> native((Native))
  scene --> svg((SVG))`

// water is the shader of the sea a picture for links is set in: light near the surface
// and deep below, with shafts of light slanting down, and caustics, the net of light the
// surface throws, which fade with depth.
var water = shader.MustCompile(`package main

var Time float

// caustic is the brightness of the caustics at uv, at time t.
func caustic(uv vec2, t float) float {
	p := mod(uv*6.28318, 6.28318) - 250.0
	i := p
	c := 1.0
	inten := 0.005
	for n := 0; n < 5; n++ {
		tt := t * (1.0 - 3.5/float(n+1))
		i = p + vec2(cos(tt-i.x)+sin(tt+i.y), sin(tt-i.y)+cos(tt+i.x))
		c += 1.0 / length(vec2(p.x/(sin(i.x+tt)/inten), p.y/(cos(i.y+tt)/inten)))
	}
	c /= 5.0
	c = 1.17 - pow(c, 1.4)
	return clamp(pow(abs(c), 8.0), 0.0, 1.0)
}

func Fragment(dst vec4, pos vec2) vec4 {
	size := rectSize()
	depth := pos.y / size.y
	col := mix(vec3(0.20, 0.56, 0.62), vec3(0.08, 0.40, 0.49), smoothstep(0.0, 0.55, depth))
	col = mix(col, vec3(0.03, 0.15, 0.24), smoothstep(0.45, 1.0, depth))
	x := pos.x/size.y + depth*0.45
	rays := 0.5 + 0.5*sin(x*7.0+4.0)*sin(x*3.1+0.4)
	col += vec3(0.60, 0.90, 0.88) * pow(rays, 4.0) * 0.24 * (1.0 - smoothstep(0.0, 0.85, depth))
	c := caustic(pos/size.y*2.4, Time)
	col += vec3(0.75, 1.0, 0.95) * c * 0.30 * (1.0 - smoothstep(0.0, 0.42, depth))
	return vec4(col, 1.0)
}
`)

// bubble paints a bubble of clear glass over r, rounded by radius, with a glint of the
// light above, up and to the left.
func bubble(p *ui.Painter, r ui.Rect, radius float32) {
	glass.Paint(p, r, radius, glass.Glass{Style: glass.Clear})
	// The glint is the shadow of a dot far to the left, cast back, as for canvas's glows.
	const away = 8000
	g := min(r.W, r.H) * 0.16
	p.Shadow(ui.Rect{X: r.X + r.W*0.24 - away, Y: r.Y + r.H*0.2, W: g, H: g}, g/2, away, 0, g*0.6, 0, ui.RGBA(255, 255, 255, 0.85))
}

// card draws the picture a link shows, w by h DIPs, under the sea: limu's name, what it is
// and where it is on the left, and on the right how it works, in bubbles rising from limu
// kohu, drawn with limu's mark, that grows from the floor. Its text is in the website's
// font.
func card(w, h int, mark *ui.SVG) *image.RGBA {
	W, H := float32(w), float32(h)
	k := H / 630 // what the sizes below, set for a picture 630 high, are multiplied by
	fonts := limumygo.NewFonts(cardFont, "monospace")
	sc, err := limu.Layout(cardSource, &layout.Options{Measurer: fonts, FontSize: float64(22 * k), KeepDirection: true})
	if err != nil {
		log.Fatal(err)
	}
	th := *style.Kohu()
	th.Line = style.Hex("#ffffff")
	th.Fg = style.Hex("#ffffff")
	opts := &limumygo.Options{
		Look: style.Look{Theme: &th, Overrides: style.Overrides{
			"flowchart.edge": {StrokeWidth: style.N(float64(2 * k)), StrokeOpacity: style.N(0.75)},
		}},
		Fonts: fonts,
		// Each node and edge label is a bubble.
		Draw: func(p *limumygo.Painter, it *scene.Item, s style.Resolved, next func()) {
			var r ui.Rect
			switch {
			case it.Kind == scene.Node:
				r = p.Rect(it.Box)
			case it.Kind == scene.Text && it.Label != nil && !it.Label.Empty():
				b, em := it.LabelBox(), it.Label.Size
				r = p.Rect(scene.Rect{X: b.X - em/2, Y: b.Y - em/8, W: b.W + em, H: b.H + em/4})
			default:
				next()
				return
			}
			bubble(p.Painter, r, r.H/2)
			p.Label(it, s)
		},
	}
	white := ui.RGB(255, 255, 255)
	return ui.Render(func(c *ui.Context) {
		ui.Box(c).Size(W, H).Draw(func(p *ui.Painter, r ui.Rect) {
			water.Draw(p, r, shader.Options{Uniforms: map[string]any{"Time": 4.2}})
			// Limu kohu, the far ones in the water's color, and the near ones in limu's.
			deep := ui.Hex("#0e3a4a")
			for _, a := range []struct {
				x, s, far float32
				c         string
			}{{0.62, 170, 0.55, "#b5435c"}, {0.98, 200, 0.5, "#c0668a"}, {0.88, 210, 0.3, "#b5435c"}, {0.76, 170, 0, "#a92e4a"}, {0.94, 150, 0, "#c0668a"}, {0.67, 120, 0.15, "#c0668a"}} {
				s := a.s * k
				p.Icon(mark, ui.Rect{X: r.X + a.x*W - s/2, Y: r.Y + H - s*0.92, W: s, H: s}, ui.Hex(a.c).Mix(deep, a.far))
			}
			// Small bubbles rising from the limu at 0.76 to the diagram's first, which stands
			// over it.
			for _, b := range []struct{ x, y, d float32 }{{0.764, 0.735, 7}, {0.758, 0.708, 10}} {
				d := b.d * k
				bubble(p, ui.Rect{X: r.X + b.x*W - d/2, Y: r.Y + b.y*H - d/2, W: d, H: d}, d/2)
			}
			limumygo.Paint(p, sc, r.X+0.76*W-float32(sc.Width)/2, r.Y+22*k, opts)
			x := r.X + 80*k
			p.RichText(x-6*k, r.Y+150*k, 0, ui.Span{Text: "limu", Font: cardFont, Size: 128 * k, Weight: 700, Color: white})
			p.RichText(x, r.Y+330*k, 0,
				ui.Span{Text: "A diagram layout engine in Rust and Go.\n", Font: cardFont, Size: 38 * k, Weight: 600, Color: white},
				ui.Span{Text: "Drawn natively, in a terminal, or as SVG.", Font: cardFont, Size: 38 * k, Weight: 400, Color: white.Alpha(0.85)})
			p.RichText(x, r.Y+H-90*k, 0, ui.Span{Text: "limu.firede.com", Font: cardFont, Size: 26 * k, Weight: 500, Color: white.Alpha(0.8)})
		})
	}, w, h, 1)
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
