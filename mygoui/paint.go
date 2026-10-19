// Package mygoui draws limu's scenes with MyGo's native UI: Paint draws a scene on a
// ui.Painter, View shows one in an element that tells which item the pointer is over and
// which was clicked, and Fonts measure text for layout with the fonts it is drawn in.
//
// It draws what the svg package writes, as MyGo draws it: the parts of each item, filled
// and stroked, the markers at the ends of edges, and labels, in the style the look settles
// for each. Lines end and turn their corners as the style's Cap and Join say, as SVG's
// stroke-linecap and stroke-linejoin do: MyGo strokes paths only with round ends and
// corners, so a line with others is drawn by filling its outline.
package mygoui

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// Options say how a scene is drawn.
type Options struct {
	Look style.Look
	// Fonts are the fonts the scene was measured in; nil is DefaultFonts.
	Fonts *Fonts
	// Scale is how many DIPs a unit of the scene takes; 0 is 1.
	Scale float64
	// Icon, when set, draws an icon of a label, by its name, in r, in the label's color, as
	// with ui.Painter.Icon; labels leave room only for icons the measurer was told of.
	// Without it, icons are left out.
	Icon func(p *ui.Painter, name string, r ui.Rect, c ui.Color)
	// Draw, when set, draws each item in place of the default, which next draws. It is
	// given the item's style, settled from the look.
	Draw func(p *Painter, it *scene.Item, s style.Resolved, next func())
}

func (o *Options) scale() float64 {
	if o == nil || !(o.Scale > 0) {
		return 1
	}
	return o.Scale
}

// A Painter draws a scene on a ui.Painter, which it embeds: it places the scene's units in
// the painter's DIPs, and draws an item as Paint does, for an Options.Draw of one's own.
type Painter struct {
	*ui.Painter
	o     *Options
	fonts *Fonts
	x, y  float64 // where the scene's origin is, in DIPs
	k     float64 // DIPs a unit
}

// Paint draws a scene on p with its top left corner at x, y, in DIPs, in the order of its
// items.
func Paint(p *ui.Painter, sc *scene.Scene, x, y float32, o *Options) {
	if o == nil {
		o = &Options{}
	}
	pp := &Painter{Painter: p, o: o, fonts: o.Fonts, x: float64(x), y: float64(y), k: o.scale()}
	if pp.fonts == nil {
		pp.fonts = DefaultFonts
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		s := o.Look.Resolve(string(it.Role), it.Style, it.Author)
		if o.Draw != nil {
			o.Draw(pp, it, s, func() { pp.Item(it, s) })
		} else {
			pp.Item(it, s)
		}
	}
}

// Point is where a point of the scene is, in DIPs.
func (p *Painter) Point(pt scene.Point) (x, y float32) {
	return float32(p.x + pt.X*p.k), float32(p.y + pt.Y*p.k)
}

// Rect is where a box of the scene is, in DIPs.
func (p *Painter) Rect(r scene.Rect) ui.Rect {
	x, y := p.Point(scene.Point{X: r.X, Y: r.Y})
	return ui.Rect{X: x, Y: y, W: p.Len(r.W), H: p.Len(r.H)}
}

// Len is a length of the scene, in DIPs.
func (p *Painter) Len(v float64) float32 { return float32(v * p.k) }

// Path is a path of the scene, in DIPs.
func (p *Painter) Path(sp *scene.Path) *ui.Path {
	var out ui.Path
	sp.Each(func(op scene.Op, a []float64) {
		pt := func(i int) (float32, float32) { return p.Point(scene.Point{X: a[i], Y: a[i+1]}) }
		switch op {
		case scene.MoveTo:
			out.MoveTo(pt(0))
		case scene.LineTo:
			out.LineTo(pt(0))
		case scene.QuadTo:
			cx, cy := pt(0)
			x, y := pt(2)
			out.QuadTo(cx, cy, x, y)
		case scene.CubeTo:
			c1x, c1y := pt(0)
			c2x, c2y := pt(2)
			x, y := pt(4)
			out.CubeTo(c1x, c1y, c2x, c2y, x, y)
		case scene.Close:
			out.Close()
		}
	})
	return &out
}

// Color is a color of a style at an opacity, as MyGo takes it.
func Color(c style.Color, opacity float64) ui.Color {
	a := float64(c.A)
	if opacity < 1 {
		a = math.Round(a * max(opacity, 0))
	}
	return ui.Color{R: c.R, G: c.G, B: c.B, A: uint8(a)}
}

// Item draws an item as Paint does: its parts, its markers, its picture and its label.
func (p *Painter) Item(it *scene.Item, s style.Resolved) {
	p.Parts(it, s)
	p.Ends(it, s)
	p.Picture(it, s)
	p.Label(it, s)
}

// Parts draws the parts of an item's outline: paths and boxes, the boxes' corners rounded
// by the style's radius when they are Rounded, each filled and stroked as its Paint says.
func (p *Painter) Parts(it *scene.Item, s style.Resolved) {
	for _, part := range it.Parts {
		fill, stroke := Color(s.Fill, s.Opacity), Color(s.Stroke, s.Opacity)
		switch part.Paint {
		case scene.StrokeOnly:
			fill = ui.Color{}
		case scene.FillOnly:
			stroke = ui.Color{}
		case scene.Solid:
			fill = Color(s.Line, s.Opacity)
		case scene.Hollow:
			fill = Color(s.Bg, s.Opacity)
		case scene.Shade:
			stroke = ui.Color{}
			fill = Color(shade(s.Bg, s.Fill, it.Author.Fill.Visible()), s.Opacity)
		}
		if part.Rounded || part.Path == nil {
			radius := 0.0
			if part.Rounded {
				radius = max(s.Radius, 0)
			}
			p.box(part.Box, radius, fill, stroke, s)
			continue
		}
		if fill.A > 0 {
			p.fill(part.Path, fill)
		}
		if stroke.A > 0 {
			p.stroke(part.Path, stroke, penOf(s, s.Dash))
		}
	}
}

// shade is the color of a part painted Shade: the background, or, over a fill its author
// set, the background at 60% laid over that fill.
func shade(bg, fill style.Color, tint bool) style.Color {
	if !tint || fill.A == 0 {
		return bg
	}
	mix := func(x, y uint8) uint8 { return uint8(float64(x)*0.6 + float64(y)*0.4 + 0.5) }
	return style.Color{R: mix(bg.R, fill.R), G: mix(bg.G, fill.G), B: mix(bg.B, fill.B), A: fill.A}
}

// box draws a box of an outline. MyGo strokes a box inside its edge, where SVG strokes it
// about its edge: the box stroked is grown by half the stroke's width. A box with square
// corners is stroked mitred, or with round joins with its outer corners rounded by half the
// stroke's width; one with bevelled corners, or dashed, is stroked as a path.
func (p *Painter) box(b scene.Rect, radius float64, fill, stroke ui.Color, s style.Resolved) {
	if b.W < 0 || b.H < 0 {
		return
	}
	radius = min(radius, b.W/2, b.H/2)
	if fill.A > 0 {
		p.Fill(p.Rect(b), fill, p.Len(radius))
	}
	if stroke.A == 0 || !(s.StrokeWidth > 0) {
		return
	}
	if len(s.Dash) > 0 || radius == 0 && s.Join == style.JoinBevel {
		p.stroke(new(scene.Path).RoundRect(b, radius), stroke, penOf(s, s.Dash))
		return
	}
	w := s.StrokeWidth
	r := 0.0
	if radius > 0 {
		r = radius + w/2
	} else if s.Join == style.JoinRound {
		r = w / 2
	}
	p.Painter.Stroke(p.Rect(scene.Rect{X: b.X - w/2, Y: b.Y - w/2, W: b.W + w, H: b.H + w}), stroke, p.Len(r), p.Len(w))
}

// Ends draws the markers at the ends of an edge, in the color of its stroke, each part as
// scene.Marker.Outline says, solid, with the edge's caps and with its corners turned as
// the marker's type says or else as the edge's are.
func (p *Painter) Ends(it *scene.Item, s style.Resolved) {
	c := Color(s.Stroke, s.Opacity)
	if c.A == 0 {
		return
	}
	for _, m := range it.Ends {
		pn := penOf(s, nil)
		if j := m.Type.Join(); j.Set() {
			pn.join = j
		}
		for _, part := range m.Outline() {
			path := part.Path
			if path == nil {
				path = new(scene.Path).RoundRect(part.Box, 0)
			}
			switch part.Paint {
			case scene.FillOnly, scene.Solid:
				p.fill(path, c)
			case scene.FillAndStroke:
				p.fill(path, c)
				p.stroke(path, c, pn)
			case scene.StrokeOnly:
				p.stroke(path, c, pn)
			case scene.Hollow:
				p.fill(path, Color(s.Bg, s.Opacity))
				p.stroke(path, c, pn)
			}
		}
	}
}

// MyGo draws a path from a mask of coverage, the size of the path's bounds, which it keeps
// in an atlas of 4096 by 4096 pixels that it shares with the glyphs of text, and draws none
// wider or taller than 4096 pixels. So that a scene with many long lines leaves room in the
// atlas, an opaque line longer than piece device pixels is stroked in pieces no larger than
// that, whose masks are thin along any line; and a path too large for a mask is drawn in
// pieces, only where the painter shows (ui.Painter.Visible).
const (
	maxMask = 4000
	piece   = 256
)

// tolerance is how far from a curve the lines it is cut into may stray, in the scene's
// units: a quarter of a device pixel.
func (p *Painter) tolerance() float64 { return 0.25 / (p.k * float64(p.Scale())) }

// across is the device pixels a box of the scene takes across, with a stroke w DIPs wide
// about it.
func (p *Painter) across(b scene.Rect, w float32) float32 {
	return (max(p.Len(b.W), p.Len(b.H)) + w + 2) * p.Scale()
}

func (p *Painter) fill(sp *scene.Path, c ui.Color) {
	if p.across(sp.Bounds(), 0) < maxMask {
		p.FillPath(p.Path(sp), c)
		return
	}
	p.fillTiles(sp.Flatten(p.tolerance()), c, maxMask)
}

// stroke strokes a path of the scene with a pen. MyGo strokes it when its ends and corners
// are round, which are all MyGo draws; with others, its outline is filled.
func (p *Painter) stroke(sp *scene.Path, c ui.Color, pn pen) {
	if c.A == 0 || !(pn.width > 0) {
		return
	}
	if pn.cap != style.CapRound || pn.join != style.JoinRound {
		lines := sp.Flatten(p.tolerance())
		if len(pn.dash) > 0 {
			lines = scene.Dash(lines, pn.dash, 0)
		}
		p.strokeOutline(lines, c, pn)
		return
	}
	w := p.Len(pn.width)
	size := p.across(sp.Bounds(), w)
	large := size >= maxMask
	if len(pn.dash) == 0 && !large && (size <= piece || c.A < 255) {
		p.StrokePath(p.Path(sp), w, c)
		return
	}
	lines := sp.Flatten(p.tolerance())
	if len(pn.dash) > 0 {
		lines = scene.Dash(lines, pn.dash, 0)
	}
	// Translucent pieces would show darker where their caps overlap: they are cut only
	// when they must be.
	most := float32(maxMask)
	if c.A == 255 {
		most = piece
	}
	p.strokeLines(lines, w, c, max(most/p.Scale()-w-2, 8), large)
}

// strokeLines strokes polylines of the scene w DIPs wide, in pieces no more than limit DIPs
// across, and with cull only those that show. With round joins and caps, a closed polyline
// is drawn as an open one back to its start.
func (p *Painter) strokeLines(lines []scene.Polyline, w float32, c ui.Color, limit float32, cull bool) {
	view := p.Visible()
	var path ui.Path
	var x0, y0, x1, y1 float32 // the piece's bounds
	empty := true
	flush := func() {
		if !empty && (!cull || overlaps(ui.Rect{X: x0 - w, Y: y0 - w, W: x1 - x0 + 2*w, H: y1 - y0 + 2*w}, view)) {
			p.StrokePath(&path, w, c)
		}
		path.Reset()
		empty = true
	}
	for _, l := range lines {
		pts := l.Points
		if l.Closed {
			pts = append(pts[:len(pts):len(pts)], pts[0])
		}
		var last [2]float32
		for i, pt := range pts {
			x, y := p.Point(pt)
			if i > 0 {
				// A line longer than a piece is cut.
				n := int(math.Ceil(float64(max(abs(x-last[0]), abs(y-last[1])) / (limit / 2))))
				for j := 1; j <= n; j++ {
					k := float32(j) / float32(n)
					sx, sy := last[0]+(x-last[0])*k, last[1]+(y-last[1])*k
					if !empty && max(x1, sx)-min(x0, sx) <= limit && max(y1, sy)-min(y0, sy) <= limit {
						path.LineTo(sx, sy)
						x0, y0, x1, y1 = min(x0, sx), min(y0, sy), max(x1, sx), max(y1, sy)
					} else {
						px, py := last[0]+(x-last[0])*float32(j-1)/float32(n), last[1]+(y-last[1])*float32(j-1)/float32(n)
						flush()
						path.MoveTo(px, py).LineTo(sx, sy)
						x0, y0, x1, y1 = min(px, sx), min(py, sy), max(px, sx), max(py, sy)
						empty = false
					}
				}
			} else if !empty && max(x1, x)-min(x0, x) <= limit && max(y1, y)-min(y0, y) <= limit {
				path.MoveTo(x, y)
				x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
			} else {
				flush()
				path.MoveTo(x, y)
				x0, y0, x1, y1 = x, y, x, y
				empty = false
			}
			last = [2]float32{x, y}
		}
	}
	flush()
}

// fillTiles fills polygons of the scene, where the painter shows, a tile of tile device
// pixels at a time: each polygon is clipped to the tile, which keeps how many times it winds
// about each point inside it, and so what a fill with the non-zero rule covers. The tiles'
// sides are on the edges of device pixels, so that where two meet neither covers a part of
// a pixel the other covers too, and no seam shows.
func (p *Painter) fillTiles(polys []scene.Polyline, c ui.Color, tile int) {
	var dips [][][2]float32
	x0, y0 := float32(math.Inf(1)), float32(math.Inf(1))
	x1, y1 := float32(math.Inf(-1)), float32(math.Inf(-1))
	for _, l := range polys {
		var poly [][2]float32
		for _, pt := range l.Points {
			x, y := p.Point(pt)
			poly = append(poly, [2]float32{x, y})
			x0, y0, x1, y1 = min(x0, x), min(y0, y), max(x1, x), max(y1, y)
		}
		dips = append(dips, poly)
	}
	// Only what shows, and a pixel about it.
	v := p.Visible()
	x0, y0 = max(x0, v.X-1), max(y0, v.Y-1)
	x1, y1 = min(x1, v.X+v.W+1), min(y1, v.Y+v.H+1)
	s := p.Scale()
	// The tiles, in device pixels, from the pixel the first is in.
	px0, py0 := int(math.Floor(float64(x0*s))), int(math.Floor(float64(y0*s)))
	var path ui.Path
	for ty := py0; float32(ty) < y1*s; ty += tile {
		for tx := px0; float32(tx) < x1*s; tx += tile {
			r := ui.Rect{X: float32(tx) / s, Y: float32(ty) / s, W: float32(tile) / s, H: float32(tile) / s}
			path.Reset()
			for _, poly := range dips {
				clipped := clipPolygon(poly, r)
				for i, pt := range clipped {
					if i == 0 {
						path.MoveTo(pt[0], pt[1])
					} else {
						path.LineTo(pt[0], pt[1])
					}
				}
				if len(clipped) > 0 {
					path.Close()
				}
			}
			p.FillPath(&path, c)
		}
	}
}

// clipPolygon is a polygon clipped to a box, as Sutherland and Hodgman do it.
func clipPolygon(poly [][2]float32, r ui.Rect) [][2]float32 {
	edges := [4]struct {
		axis int
		at   float32
		keep float32 // 1 keeps what is above at, -1 what is below
	}{{0, r.X, 1}, {0, r.X + r.W, -1}, {1, r.Y, 1}, {1, r.Y + r.H, -1}}
	for _, e := range edges {
		if len(poly) == 0 {
			break
		}
		in := func(pt [2]float32) bool { return (pt[e.axis]-e.at)*e.keep >= 0 }
		var out [][2]float32
		prev := poly[len(poly)-1]
		for _, pt := range poly {
			if in(pt) != in(prev) {
				k := (e.at - prev[e.axis]) / (pt[e.axis] - prev[e.axis])
				out = append(out, [2]float32{prev[0] + (pt[0]-prev[0])*k, prev[1] + (pt[1]-prev[1])*k})
			}
			if in(pt) {
				out = append(out, pt)
			}
			prev = pt
		}
		poly = out
	}
	return poly
}

func overlaps(a, b ui.Rect) bool {
	return a.X < b.X+b.W && a.Y < b.Y+b.H && a.X+a.W > b.X && a.Y+a.H > b.Y
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// Label draws an item's label as the svg package writes it, about the item's At: each line
// centered on it, or starting or ending there as its Align says, the baseline of a line a
// little below the middle of its line's height, as Pele's labelSvg puts it, and the whole
// turned by its Rotate. Each span is set in its face at the label's size, and centered on
// the room layout measured for it.
func (p *Painter) Label(it *scene.Item, s style.Resolved) {
	l := it.Label
	if l == nil || l.Empty() {
		return
	}
	c := Color(s.Text, s.Opacity)
	if c.A == 0 {
		return
	}
	halo := Color(s.Halo, s.Opacity)
	draw := func() {
		cx, cy := it.At.X, it.At.Y
		shift := l.Size * 0.35
		y := cy - l.Height/2 + l.LineHeight/2 + shift
		size := p.Len(l.Size)
		for i, line := range l.Lines {
			var x float64
			switch it.Align {
			case scene.Start:
				x = cx
			case scene.End:
				x = cx - l.Widths[i]
			default:
				x = cx - l.Widths[i]/2
			}
			for _, sp := range line {
				switch {
				case sp.Icon != "":
					if p.o.Icon != nil {
						side := l.Size * 1.1
						p.o.Icon(p.Painter, sp.Icon, p.Rect(scene.Rect{X: x, Y: y - shift - side/2, W: side, H: side}), c)
					}
				case sp.Text != "":
					glyphs, w := p.fonts.Shape(sp.Text, size, sp.Face)
					gx, gy := p.Point(scene.Point{X: x + sp.Width/2, Y: y})
					if halo.A > 0 && s.HaloWidth > 0 {
						p.halo(glyphs, gx-w/2, gy, p.Len(s.HaloWidth/2), halo)
					}
					p.Glyphs(glyphs, gx-w/2, gy, c)
				}
				x += sp.Width
			}
			y += l.LineHeight
		}
	}
	if it.Rotate != 0 {
		cx, cy := p.Point(it.At)
		p.Transform(ui.Rotation(float32(it.Rotate)), cx, cy, draw)
		return
	}
	draw()
}

// halo draws glyphs about where they go, r away in sixteen directions, in a color: MyGo
// strokes no text, and this is near what SVG's stroke of it looks like.
func (p *Painter) halo(glyphs []ui.Glyph, x, y, r float32, c ui.Color) {
	for i := range 16 {
		a := float64(i) * math.Pi / 8
		p.Glyphs(glyphs, x+r*float32(math.Cos(a)), y+r*float32(math.Sin(a)), c)
	}
}

// Picture draws an item's picture: an icon, through Options.Icon, in the color of its text.
// Images are left to an Options.Draw of the host's, which can load them.
func (p *Painter) Picture(it *scene.Item, s style.Resolved) {
	pic := it.Picture
	if pic == nil || pic.Icon == "" || p.o.Icon == nil {
		return
	}
	p.o.Icon(p.Painter, pic.Icon, p.Rect(pic.Box), Color(s.Text, s.Opacity))
}
