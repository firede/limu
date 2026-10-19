package cynefin

import (
	"math"
	"strconv"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The scene's roles, in the order Pele draws them: "cynefin.title" for the title;
// "cynefin.region" for the tint of each of the four quadrants, its id its domain's;
// "cynefin.boundary" for each of the three dashed wavy lines between them and
// "cynefin.cliff" for the S between Clear and Chaotic; "cynefin.backdrop" for the ellipse of
// the background that hides what is under Confusion, and "cynefin.region" again for its tint
// and outline; for each domain, a "cynefin.domain" group, its id its name and its label its
// name as a heading, "cynefin.description" for its decision model and its kind of practice,
// and "cynefin.item" for each item it lists, its id the item's text, with
// "cynefin.overflow" for the count of those Confusion leaves out; and "cynefin.transition"
// for each transition, its id its ends joined by a hyphen, with "cynefin.transition-label"
// for its label.
const (
	RoleTitle           scene.Role = "cynefin.title"
	RoleRegion          scene.Role = "cynefin.region"
	RoleBoundary        scene.Role = "cynefin.boundary"
	RoleCliff           scene.Role = "cynefin.cliff"
	RoleBackdrop        scene.Role = "cynefin.backdrop"
	RoleDomain          scene.Role = "cynefin.domain"
	RoleDescription     scene.Role = "cynefin.description"
	RoleItem            scene.Role = "cynefin.item"
	RoleOverflow        scene.Role = "cynefin.overflow"
	RoleTransition      scene.Role = "cynefin.transition"
	RoleTransitionLabel scene.Role = "cynefin.transition-label"
)

// quadrant is a domain drawn in a quadrant: its column and row, and the series of its tint.
type quadrant struct {
	name     string
	col, row float64
	series   int
}

// quadrants are Pele's QUADRANTS.
var quadrants = []quadrant{
	{Complex, 0, 0, 0},
	{Complicated, 1, 0, 1},
	{Chaotic, 0, 1, 3},
	{Clear, 1, 1, 2},
}

const confusionColor = 4

// meta is Pele's META: each domain's name, its decision model and its kind of practice.
var meta = map[string][3]string{
	Complex:     {"Complex", "Probe → Sense → Respond", "Emergent Practices"},
	Complicated: {"Complicated", "Sense → Analyse → Respond", "Good Practices"},
	Clear:       {"Clear", "Sense → Categorise → Respond", "Best Practices"},
	Chaotic:     {"Chaotic", "Act → Sense → Respond", "Novel Practices"},
	Confusion:   {"Confusion", "", "Disorder"},
}

// The measures of Pele's cynefin render.
const (
	maxConfusionItems = 3
	itemMaxWidth      = 200
	itemPadX          = 10
	itemPadY          = 4
	itemGap           = 6
	headGap           = 10
	margin            = 20
	clearance         = 10
	steps             = 48
)

// defaultID is the id Pele hashes for the seed of the boundaries when the config gives none:
// the id prefix of its render, 'pele' unless the host names another. limu has no ids, so the
// boundaries are drawn as Pele draws them by default.
const defaultID = "pele"

// arrowPoint is the marker of transitions, and arrowTrim how far a line stops short of it, as
// Pele's markerTrim gives it.
const arrowPoint scene.MarkerType = "arrow_point"

var arrowTrim = arrowPoint.Trim()

// badge is an item of a domain, Pele's Badge.
type badge struct {
	text     string
	label    text.Label
	w, h     float64
	overflow bool
}

// block is what a domain draws, Pele's Block: its name, its descriptions and its items, the
// size they take, and the point they are centered on.
type block struct {
	name      string
	title     text.Label
	subtitles []text.Label
	badges    []badge
	w, h      float64
	cx, cy    float64
}

// setting is a number of the diagram's section of the config, between lo and hi, or
// fallback, as Pele's setting reads it.
func setting(config diagram.Config, key string, fallback, lo, hi float64) float64 {
	section, _ := config["cynefin"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v >= lo && v <= hi {
		return v
	}
	return fallback
}

// Layout lays a Cynefin diagram out as a scene, as Pele's renderCynefin draws it: the four
// quadrants, as large as the config asks or their domains need, with wavy boundaries between
// them and the cliff between Clear and Chaotic; Confusion in an ellipse in the middle; each
// domain's name, descriptions and items; and the transitions, curves that bow away from the
// middle.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	size := o.Size()
	small := jsmath.Round(size * 0.8125)
	itemSize := jsmath.Round(size * 0.875)
	measure := o.Measure()
	pad := o.Pad()
	section, _ := config["cynefin"].(diagram.Config)
	showDescriptions := section["showDomainDescriptions"] != false
	amplitude := setting(config, "boundaryAmplitude", 8, 0, 50)
	seed := resolveSeed(setting(config, "seed", 0, -1e15, 1e15), defaultID)
	itemLine := jsmath.Round(itemSize * 1.5)

	layoutText := func(raw string, fontSize, maxWidth float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: fontSize, MaxWidth: maxWidth, Face: face, Icons: o.Icons})
	}
	makeBadge := func(s string, overflow bool) badge {
		l := layoutText(s, itemSize, itemMaxWidth, 0)
		return badge{text: s, label: l, w: l.Width + 2*itemPadX, h: max(l.Height, itemLine) + 2*itemPadY, overflow: overflow}
	}
	makeBlock := func(name string) *block {
		about := meta[name]
		b := &block{name: name, title: layoutText(about[0], size, 4000, text.Bold)}
		if showDescriptions {
			for _, s := range about[1:] {
				if s != "" {
					b.subtitles = append(b.subtitles, layoutText(s, small, 4000, text.Italic))
				}
			}
		}
		var items []string
		if d := m.Domain(name); d != nil {
			items = d.Items
		}
		shown := len(items)
		if name == Confusion && len(items) > maxConfusionItems {
			shown = maxConfusionItems
		}
		for _, item := range items[:shown] {
			b.badges = append(b.badges, makeBadge(item, false))
		}
		if shown < len(items) {
			b.badges = append(b.badges, makeBadge("+"+strconv.Itoa(len(items)-shown)+" more", true))
		}
		b.w, b.h = b.title.Width, b.title.Height
		for _, s := range b.subtitles {
			b.w = max(b.w, s.Width)
			b.h += s.Height
		}
		if len(b.badges) > 0 {
			b.h += headGap - itemGap
		}
		for _, badge := range b.badges {
			b.w = max(b.w, badge.w)
			b.h += itemGap + badge.h
		}
		return b
	}

	blocks := make([]*block, len(quadrants))
	for i, q := range quadrants {
		blocks[i] = makeBlock(q.name)
	}
	confusion := makeBlock(Confusion)
	rx := max(88, float64(confusion.w*0.71)+8)
	ry := max(64, float64(confusion.h*0.71)+6)

	quadW := setting(config, "width", 640, 100, 10000) / 2
	quadH := setting(config, "height", 440, 100, 10000) / 2
	for _, b := range blocks {
		quadW = max(quadW, b.w+2*margin)
		quadH = max(quadH, b.h+2*margin)
	}
	// A block sits in the middle of its quadrant, so its inner corner must stay clear of the
	// ellipse.
	for _, b := range blocks {
		dx := (quadW - b.w) / 2
		dy := (quadH - b.h) / 2
		ex, ey := dx/(rx+clearance), dy/(ry+clearance)
		if float64(ex*ex)+float64(ey*ey) >= 1 {
			continue
		}
		wider := b.w + 2*(rx+clearance)
		taller := b.h + 2*(ry+clearance)
		if float64(wider*quadH) <= float64(quadW*taller) {
			quadW = max(quadW, wider)
		} else {
			quadH = max(quadH, taller)
		}
	}

	title := layoutText(m.Title, size, 4000, text.Bold)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	width := 2 * quadW
	height := 2 * quadH
	x0 := float64(pad)
	y0 := pad + titleHeight
	cx := x0 + quadW
	cy := y0 + quadH
	right := x0 + width
	bottom := y0 + height

	foldOptions := fold
	foldOptions.segments, foldOptions.pinEnd = 4, true
	westOptions := horizon
	westOptions.segments, westOptions.pinEnd = 4, true
	eastOptions := horizon
	eastOptions.segments, eastOptions.pinStart = 4, true
	foldLine := wave(y0, cy, cx, seed, amplitude, foldOptions)
	west := wave(x0, cx, cy, seed+100, amplitude, westOptions)
	east := wave(cx, right, cy, seed+200, amplitude, eastOptions)
	drop := cliff(cx, cy, bottom, float64(width*0.03))

	var items []scene.Item
	if !title.Empty() {
		items = append(items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At: scene.Point{X: x0 + width/2, Y: pad + title.Height/2},
		})
	}

	// Each quadrant's tint is bounded by the page and by the lines between it and its
	// neighbours, followed one way or the other.
	complexRegion := new(scene.Path).MoveTo(x0, y0).LineTo(foldLine[0], foldLine[1])
	forward(complexRegion, foldLine)
	backward(complexRegion, west)
	complicatedRegion := new(scene.Path).MoveTo(foldLine[0], foldLine[1]).LineTo(right, y0).LineTo(last(east))
	backward(complicatedRegion, east)
	backward(complicatedRegion, foldLine)
	chaoticRegion := new(scene.Path).MoveTo(west[0], west[1])
	forward(chaoticRegion, west)
	forward(chaoticRegion, drop)
	chaoticRegion.LineTo(x0, bottom)
	clearRegion := new(scene.Path).MoveTo(east[0], east[1])
	forward(clearRegion, east)
	clearRegion.LineTo(right, bottom).LineTo(last(drop))
	backward(clearRegion, drop)
	regions := map[string]*scene.Path{
		Complex:     complexRegion.Close(),
		Complicated: complicatedRegion.Close(),
		Chaotic:     chaoticRegion.Close(),
		Clear:       clearRegion.Close(),
	}
	for _, q := range quadrants {
		p := regions[q.name]
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: RoleRegion, ID: q.name, Box: p.Bounds(),
			Parts: []scene.Part{{Path: p, Paint: scene.FillOnly}},
			Style: style.Style{Fill: style.Series(q.series).Alpha(0.14)},
		})
	}
	stroke := func(role scene.Role, curve []float64, st style.Style) {
		p := new(scene.Path).MoveTo(curve[0], curve[1])
		forward(p, curve)
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: role, Box: p.Bounds(),
			Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
			Style: st,
		})
	}
	boundary := style.Style{Stroke: style.Border, Dash: style.Dashes(6, 3)}
	stroke(RoleBoundary, foldLine, boundary)
	stroke(RoleBoundary, west, boundary)
	stroke(RoleBoundary, east, boundary)
	stroke(RoleCliff, drop, style.Style{Stroke: style.Line, StrokeWidth: style.N(2), Dash: style.Solid, Cap: style.CapRound})

	oval := ellipse(cx, cy, rx, ry)
	ovalBox := scene.Rect{X: cx - rx, Y: cy - ry, W: 2 * rx, H: 2 * ry}
	items = append(items,
		scene.Item{
			Kind: scene.Mark, Role: RoleBackdrop, Shape: "ellipse", Box: ovalBox,
			Parts: []scene.Part{{Path: oval, Paint: scene.FillOnly}},
			Style: style.Style{Fill: style.Bg},
		},
		scene.Item{
			Kind: scene.Mark, Role: RoleRegion, Shape: "ellipse", ID: Confusion, Box: ovalBox,
			Parts: []scene.Part{{Path: oval}},
			Style: style.Style{Fill: style.Series(confusionColor).Alpha(0.18), Stroke: style.Border, Dash: style.Dashes(4, 2)},
		})

	drawBlock := func(b *block) {
		y := b.cy - b.h/2
		l := b.title
		items = append(items, scene.Item{
			Kind: scene.Group, Role: RoleDomain, ID: b.name,
			Box:   scene.Rect{X: b.cx - b.w/2, Y: y, W: b.w, H: b.h},
			Label: &l, At: scene.Point{X: b.cx, Y: y + l.Height/2},
		})
		y += l.Height
		for _, s := range b.subtitles {
			s := s
			items = append(items, scene.Item{
				Kind: scene.Text, Role: RoleDescription, ID: b.name, Label: &s,
				At: scene.Point{X: b.cx, Y: y + s.Height/2}, Style: style.Style{Text: style.Muted},
			})
			y += s.Height
		}
		y += headGap - itemGap
		for _, item := range b.badges {
			y += itemGap
			box := scene.Rect{X: b.cx - item.w/2, Y: y, W: item.w, H: item.h}
			it := scene.Item{
				Kind: scene.Node, Role: RoleItem, Shape: "rounded", ID: item.text, Box: box,
				Parts: []scene.Part{{Box: box, Rounded: true}},
				At:    scene.Point{X: b.cx, Y: y + item.h/2},
				Style: style.Style{Fill: style.Bg, Stroke: style.Border},
			}
			if item.overflow {
				it.Role, it.ID = RoleOverflow, ""
				it.Style.Dash = style.Dashes(3, 2)
				it.Style.Text = style.Muted
			}
			if !item.label.Empty() {
				l := item.label
				it.Label = &l
			}
			items = append(items, it)
			y += item.h
		}
	}

	byName := map[string]*block{}
	for i, q := range quadrants {
		b := blocks[i]
		b.cx = x0 + float64(quadW*(q.col+0.5))
		b.cy = y0 + float64(quadH*(q.row+0.5))
		byName[q.name] = b
		drawBlock(b)
	}
	confusion.cx, confusion.cy = cx, cy
	byName[Confusion] = confusion
	drawBlock(confusion)

	inside := func(b *block, p scene.Point) bool {
		x, y := p.X, p.Y
		if b == confusion {
			ex, ey := (x-cx)/(rx+4), (y-cy)/(ry+4)
			return float64(ex*ex)+float64(ey*ey) <= 1
		}
		return math.Abs(x-b.cx) <= b.w/2+8 && math.Abs(y-b.cy) <= b.h/2+8
	}

	seen := map[string]float64{}
	for _, transition := range m.Transitions {
		from, to := byName[transition.From], byName[transition.To]
		if from == nil || to == nil || from == to {
			continue
		}
		key := transition.To + " " + transition.From
		if transition.From < transition.To {
			key = transition.From + " " + transition.To
		}
		repeat := seen[key]
		seen[key] = repeat + 1

		dx, dy := to.cx-from.cx, to.cy-from.cy
		length := jsmath.Or1(jsmath.Hypot(dx, dy))
		mx, my := (from.cx+to.cx)/2, (from.cy+to.cy)/2
		// Curves bow away from the middle of the diagram, which keeps them off the ellipse.
		outward := -1.0
		if float64(-dy*(mx-cx))+float64(dx*(my-cy)) >= 0 {
			outward = 1
		}
		nx, ny := (-dy/length)*outward, (dx/length)*outward
		// A diagonal runs through the middle, so it bows far enough to pass the ellipse.
		diagonal := from != confusion && to != confusion && math.Abs(dx) > 1 && math.Abs(dy) > 1
		bow := float64(length * 0.15)
		if diagonal {
			bow = 2 * (jsmath.Hypot(float64(rx*nx), float64(ry*ny)) + 16)
		}
		bow += float64(repeat * 56)
		px, py := mx+float64(nx*bow), my+float64(ny*bow)
		point := func(t float64) (float64, float64) {
			u := 1 - t
			return float64(u*u*from.cx) + float64(2*u*t*px) + float64(t*t*to.cx),
				float64(u*u*from.cy) + float64(2*u*t*py) + float64(t*t*to.cy)
		}
		crossing := func(b *block, lo, hi float64, leaving bool) float64 {
			for range 16 {
				mid := (lo + hi) / 2
				if inside(b, at(point(mid))) == leaving {
					lo = mid
				} else {
					hi = mid
				}
			}
			return (lo + hi) / 2
		}
		i := 1
		for i <= steps && inside(from, at(point(float64(i)/steps))) {
			i++
		}
		if i > steps {
			continue
		}
		t0 := crossing(from, float64(i-1)/steps, float64(i)/steps, true)
		for i <= steps && !inside(to, at(point(float64(i)/steps))) {
			i++
		}
		t1 := 1.0
		if i <= steps {
			t1 = crossing(to, float64(i-1)/steps, float64(i)/steps, false)
		}

		sx, sy := point(t0)
		ex, ey := point(t1)
		blossom := func(a, b, c float64) float64 {
			return float64((1-t0)*(1-t1)*a) + float64((float64((1-t0)*t1)+float64(t0*(1-t1)))*b) + float64(t0*t1*c)
		}
		qx, qy := blossom(from.cx, px, to.cx), blossom(from.cy, py, to.cy)
		el := jsmath.Or1(jsmath.Hypot(ex-qx, ey-qy))
		ux, uy := (ex-qx)/el, (ey-qy)/el
		head := scene.Marker{Type: arrowPoint, At: scene.Point{X: ex, Y: ey}, DX: ux, DY: uy}
		ex -= float64(ux * arrowTrim)
		ey -= float64(uy * arrowTrim)

		id := transition.From + "-" + transition.To
		p := new(scene.Path).MoveTo(sx, sy).QuadTo(qx, qy, ex, ey)
		it := scene.Item{
			Kind: scene.Edge, Role: RoleTransition, ID: id, Box: p.Bounds(),
			Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
			Route: []scene.Point{{X: sx, Y: sy}, {X: ex, Y: ey}},
			Style: style.Style{Stroke: style.Line},
		}
		it.Ends[1] = head
		items = append(items, it)

		if transition.Label != "" {
			l := layoutText(transition.Label, small, itemMaxWidth, 0)
			lx, ly := point((t0 + t1) / 2)
			away := float64(math.Abs(nx)*l.Width)/2 + float64(math.Abs(ny)*l.Height)/2 + 6
			half := l.Width/2 + 4
			tx := min(max(lx+float64(nx*away), x0+half), max(right-half, x0+half))
			if !l.Empty() {
				items = append(items, scene.Item{
					Kind: scene.Text, Role: RoleTransitionLabel, ID: id, Label: &l,
					At: scene.Point{X: tx, Y: ly + float64(ny*away)},
				})
			}
		}
	}

	return &scene.Scene{
		Type:     "cynefin",
		Width:    math.Ceil(max(width, title.Width) + 2*pad),
		Height:   math.Ceil(height + titleHeight + 2*pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
		Items:    items,
	}
}

// at is the point of two coordinates.
func at(x, y float64) scene.Point { return scene.Point{X: x, Y: y} }

// last is the end point of a curve.
func last(curve []float64) (float64, float64) { return curve[len(curve)-2], curve[len(curve)-1] }

// forward adds to p the curves from a curve's start point to its end, as the commands of
// Pele's forward draw them.
func forward(p *scene.Path, curve []float64) {
	for i := 2; i < len(curve); i += 6 {
		p.CubeTo(curve[i], curve[i+1], curve[i+2], curve[i+3], curve[i+4], curve[i+5])
	}
}

// backward adds to p the curves from a curve's end point back to its start, as the commands
// of Pele's backward draw them.
func backward(p *scene.Path, curve []float64) {
	for i := len(curve) - 6; i >= 2; i -= 6 {
		p.CubeTo(curve[i+2], curve[i+3], curve[i], curve[i+1], curve[i-2], curve[i-1])
	}
}

// ellipse is the outline Pele's ellipsePath draws: two arcs from the left end of the
// ellipse, over its top and back under it.
func ellipse(cx, cy, rx, ry float64) *scene.Path {
	return new(scene.Path).MoveTo(cx-rx, cy).ArcTo(rx, ry, 0, true, true, cx+rx, cy).ArcTo(rx, ry, 0, true, true, cx-rx, cy).Close()
}
