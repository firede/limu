package c4

import (
	"math"
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/diagram/internal/draw"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of the drawing, as Pele's render.ts has them.
const (
	pairShift = 7
	narrowGap = 36
	widen     = 24
	maxOffset = 2000
	// A label may cover this much of a shape's edge, where there is no text.
	edge = 8
	// Above this many box comparisons, labels stay at the middle and lines are not steered
	// around shapes.
	careBudget = 2000000
	// arrow is the length of an arrowhead, Pele's ARROW.
	arrow = scene.MarkerSize
	// patchRadius is how much the corners of a relation label's patch are rounded.
	patchRadius = 3
)

// slide is the places along a relation to try for its label, nearest the middle first.
var slide = [...]float64{0.5, 0.4, 0.6, 0.3, 0.7, 0.2, 0.8}

// The roles of the items of a C4 diagram.
const (
	RoleTitle         scene.Role = "c4.title"
	RoleBoundary      scene.Role = "c4.boundary"
	RoleDeployment    scene.Role = "c4.deployment"
	RoleBoundaryLabel scene.Role = "c4.boundary-label"
	RoleBoundaryType  scene.Role = "c4.boundary-type"
	RoleBoundaryDescr scene.Role = "c4.boundary-descr"
	RoleRel           scene.Role = "c4.rel"
	RoleCasing        scene.Role = "c4.casing"
	RoleElement       scene.Role = "c4.element"
	RoleElementType   scene.Role = "c4.element-type"
	RoleElementTechn  scene.Role = "c4.element-techn"
	RoleElementDescr  scene.Role = "c4.element-descr"
	RoleRelLabel      scene.Role = "c4.rel-label"
	RoleRelTechn      scene.Role = "c4.rel-techn"
)

// userDecls are the declarations of the colors given by UpdateElementStyle and
// UpdateRelStyle, as Pele's userStyle writes them.
func userDecls(fill, stroke, color string) []string {
	var decls []string
	if fill != "" {
		decls = append(decls, "fill:"+fill)
	}
	if stroke != "" {
		decls = append(decls, "stroke:"+stroke)
	}
	if color != "" {
		decls = append(decls, "color:"+color)
	}
	return decls
}

// userStyle is the author's style of colors given by UpdateElementStyle and UpdateRelStyle,
// as Pele's userStyle declares them.
func userStyle(fill, stroke, color string) style.Style {
	decls := userDecls(fill, stroke, color)
	if decls == nil {
		return style.Style{}
	}
	return css.Declarations(decls).Style
}

// textGround is the fill under the texts of a shape or boundary whose author's declarations
// are decls, where Pele's inkFill picks a color for them on it, and none elsewhere.
func textGround(decls []string) style.Paint {
	if !css.InksText(decls, decls) {
		return style.Paint{}
	}
	return css.Declarations(decls).Style.Fill
}

// unhide puts back the `#name;` that entity encoding hid in an address as ﬂ°°name¶ß, as
// Pele's link does, so that the check sees what a browser would: /ﬂ°°?/g becomes # and /¶ß/g
// becomes ;.
func unhide(u string) string {
	if !strings.Contains(u, "¶ß") {
		return u
	}
	var b strings.Builder
	for i := 0; i < len(u); {
		switch {
		case strings.HasPrefix(u[i:], "ﬂ°"):
			i += len("ﬂ°")
			if strings.HasPrefix(u[i:], "°") {
				i += len("°")
			}
			b.WriteByte('#')
		case strings.HasPrefix(u[i:], "¶ß"):
			i += len("¶ß")
			b.WriteByte(';')
		default:
			b.WriteByte(u[i])
			i++
		}
	}
	return b.String()
}

// link is where a link the diagram gives leads, as Pele's link and linked check it, or ""
// for none.
func link(u string, o *layout.Options) string {
	if u == "" {
		return ""
	}
	href, ok := url.LinkURL(url.SanitizeURL(unhide(u)), !o.NoLinks, o.LinkSchemes)
	if !ok {
		return ""
	}
	return href
}

// Layout lays out a parsed C4 diagram as a scene, as Pele's renderC4 draws it: on Mermaid's
// grid with room to spare, and, when that does not fit o.MaxWidth, tightly, with the gap
// between columns that leaves the fewest relations without a lane of their own. A relation
// between things the diagram does not have is an error, a *diagram.Error.
func Layout(db *DB, _ diagram.Config, o *layout.Options) (*scene.Scene, error) {
	if o == nil {
		o = &layout.Options{}
	}
	loose, _, err := lay(db, o, false, narrowGap)
	if err != nil {
		return nil, err
	}
	return common.Tighten(o, func(tight bool) *scene.Scene {
		if !tight {
			return loose
		}
		best, squeezed, _ := lay(db, o, true, narrowGap)
		for gap := float64(narrowGap + widen); squeezed > 0 && gap <= narrowGap+3*widen; gap += widen {
			if wider, left, _ := lay(db, o, true, gap); left < squeezed {
				best, squeezed = wider, left
			}
		}
		return best
	}), nil
}

// lay lays the diagram out once, as Pele's draw does, and gives how many routes found no
// lane of their own. The scene holds every place on the page, where Pele draws all but its
// title in a group moved by ox, oy, and the inside of each shape about its middle: each
// number is moved as Pele's SVG moves it, the moves added up from the outermost in, so that
// the scene holds what Pele draws to the bit.
func lay(db *DB, o *layout.Options, tight bool, gap float64) (*scene.Scene, int, error) {
	size := o.Size()
	small := jsmath.Round(float64(size * 0.8125))
	l := labeler{measure: o.Measure(), icons: o.Icons}
	pad := o.Pad()
	room := 0.0
	if tight {
		width := o.MaxWidth
		if !(width > 0) {
			width = 320
		}
		room = max(80, width-2*pad)
	}
	g := layoutC4(db, l, size, small, tight, room, gap)
	order := g.order
	root := order[0]

	minX, minY, maxX, maxY := 0.0, 0.0, root.w, root.h
	// grow widens the bounds of the drawing to hold a box, as the grow of Pele's draw does.
	grow := func(x0, y0, x1, y1 float64) {
		minX = min(minX, x0)
		minY = min(minY, y0)
		maxX = max(maxX, x1)
		maxY = max(maxY, y1)
	}

	ends := make([]*box, 0, 2*len(db.Rels))
	pairs := map[*box]map[*box]bool{}
	for _, rel := range db.Rels {
		var a, b *box
		if rel.FromAttr == nil {
			if e := db.getC4Shape(rel.From); e != nil {
				a = g.boxes[e]
			}
		}
		if rel.ToAttr == nil {
			if e := db.getC4Shape(rel.To); e != nil {
				b = g.boxes[e]
			}
		}
		if a == nil || b == nil {
			return nil, 0, common.NewError(`C4 rel "`+rel.From+`" -> "`+rel.To+`" references an unknown shape or boundary`, diagram.Semantic, "c4")
		}
		ends = append(ends, a, b)
		if pairs[a] == nil {
			pairs[a] = map[*box]bool{}
		}
		pairs[a][b] = true
	}

	// What a relation's label should not cover: shapes, arrowheads, boundary headings, and
	// the labels placed before it.
	var nodes []*box
	var taken []float64
	for _, gr := range order {
		for _, n := range gr.nodes {
			nodes = append(nodes, &n.box)
			taken = append(taken, n.x+edge, n.y+edge, n.x+n.w-edge, n.y+n.h-edge)
		}
	}
	rels := float64(len(db.Rels))
	careful := float64(rels*(float64(len(nodes)+len(order))+float64(3*rels))) <= careBudget
	// covered is how much of a box the taken boxes cover, as the covered of Pele's draw gives
	// it.
	covered := func(x0, y0, x1, y1 float64) float64 {
		area := 0.0
		for i := 0; i < len(taken); i += 4 {
			w := min(x1, taken[i+2]) - max(x0, taken[i])
			if w <= 0 {
				continue
			}
			if h := min(y1, taken[i+3]) - max(y0, taken[i+1]); h > 0 {
				area += float64(w * h)
			}
		}
		return area
	}
	// offset is a relation's offset kept within maxOffset, 0 for none, as the offset of Pele's
	// draw gives it.
	offset := func(v float64) float64 {
		if !truthy(v) {
			return 0
		}
		return max(-maxOffset, min(maxOffset, v))
	}

	// Every line is drawn before any label is placed, so that labels can keep off the
	// arrowheads. Lines, labels and boundaries are made where Pele works them out, before
	// the group they are drawn in is moved, and moved with it at the end.
	trim := scene.MarkerType("arrow_point").Trim()
	// The boundaries whose author paints them a fill, as Pele gathers its fills: a relation
	// inside one is drawn in the color that stands out on it, and one that crosses the border
	// of one is laid on a casing.
	var fills []draw.Fill
	for _, gr := range order {
		if gr == root {
			continue
		}
		if decls := userDecls(gr.boundary.BgColor, "", ""); css.Paints(decls) {
			fills = append(fills, draw.Fill{Box: [4]float64{gr.x, gr.y, gr.x + gr.w, gr.y + gr.h}, Decls: decls})
		}
	}
	ls := newLanes()
	for _, gr := range order {
		if gr != root {
			ls.frame(&gr.box)
		}
	}
	var others []*box
	if careful {
		others = nodes
	}
	routes := make([]*route, len(db.Rels))
	var edges, casings []scene.Item
	for index, rel := range db.Rels {
		a, b := ends[index*2], ends[index*2+1]
		startArrow := rel.Type == "birel" || rel.Type == "rel_b"
		endArrow := rel.Type != "rel_b"
		shift := 0.0
		if a != b && pairs[b][a] {
			shift = pairShift
		}
		st, et := 0.0, 0.0
		if startArrow {
			st = trim
		}
		if endArrow {
			et = trim
		}
		r := routeOf(a, b, shift, others, careful, ls, st, et)
		routes[index] = r
		grow(min(r.x0, r.x1), min(r.y0, r.y1), max(r.x0, r.x1), max(r.y0, r.y1))
		if startArrow && careful {
			taken = append(taken, r.sx-arrow, r.sy-arrow, r.sx+arrow, r.sy+arrow)
		}
		if endArrow && careful {
			taken = append(taken, r.ex-arrow, r.ey-arrow, r.ex+arrow, r.ey+arrow)
		}
		it := scene.Item{
			Kind: scene.Edge, Role: RoleRel, ID: rel.From + "-" + rel.To, Tags: []string{rel.Type},
			Box:   r.path.Bounds(),
			Parts: []scene.Part{{Path: r.path, Paint: scene.StrokeOnly}},
			// A line ends round, as Pele draws it in the group of edges, which sets round
			// caps.
			Style:  style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound},
			Author: userStyle("", rel.LineColor, ""),
		}
		for k := 0; k+1 < len(r.points); k += 2 {
			it.Route = append(it.Route, scene.Point{X: r.points[k], Y: r.points[k+1]})
		}
		if len(it.Route) == 0 {
			// A self-loop, which the layout gives no route: the corners its curve is drawn about.
			it.Route = it.Parts[0].Path.Corners()
		}
		if startArrow {
			it.Ends[0] = scene.Marker{Type: "arrow_point", At: scene.Point{X: r.sx, Y: r.sy}, DX: r.sdx, DY: r.sdy}
		}
		if endArrow {
			it.Ends[1] = scene.Marker{Type: "arrow_point", At: scene.Point{X: r.ex, Y: r.ey}, DX: r.edx, DY: r.edy}
		}
		// A relation inside one fill is drawn in the color of its labels, and one that crosses
		// the border of a fill is laid on a casing, as Pele's overFills finds.
		line := userDecls("", rel.LineColor, "")
		ground, cased := draw.OnFills(r.path, []float64{r.sx, r.sy, r.ex, r.ey}, fills, line)
		it.Style.Ground = ground
		if cased {
			if c := draw.Casing(RoleCasing, r.path, it.Ends, line, false); c != nil {
				c.ID = it.ID
				casings = append(casings, *c)
			}
		}
		edges = append(edges, it)
	}

	// A boundary's heading goes at the left, the middle or the right of its top edge: the
	// first of them no line crosses.
	for _, gr := range order {
		if !careful || gr.headW == 0 {
			continue
		}
		top := gr.y + headTop
		bottom := gr.y + gr.head - lane
		fewest := math.Inf(1)
		for _, x := range [...]float64{headLeft, (gr.w - gr.headW) / 2, gr.w - headLeft - gr.headW} {
			count := 0
			for _, r := range routes {
				if crosses(r.points, gr.x+x-4, top, gr.x+x+gr.headW+4, bottom) {
					count++
				}
			}
			if float64(count) < fewest {
				fewest = float64(count)
				gr.headX = x
			}
			if count == 0 {
				break
			}
		}
		taken = append(taken, gr.x+gr.headX-4, top, gr.x+gr.headX+gr.headW+4, bottom)
	}

	var labels []scene.Item
	for index, rel := range db.Rels {
		r := routes[index]
		id := rel.From + "-" + rel.To
		author := userStyle("", "", rel.TextColor)
		raw := rel.Label
		if db.C4Type == "C4Dynamic" {
			raw = strconv.Itoa(index+1) + ": " + raw
		}
		label := l.label(raw, small, r.room, 0)
		technRaw := ""
		if rel.Techn != "" {
			technRaw = "[" + rel.Techn + "]"
		}
		techn := l.label(technRaw, small, r.room, text.Italic)
		lw := max(label.Width, techn.Width) + labelPad
		lh := label.Height + techn.Height
		if lh == 0 {
			continue
		}

		// The label sits on the line, or just beside it when another relation runs the
		// opposite way.
		beside := float64(math.Abs(r.nx)*(lw/2)) + float64(math.Abs(r.ny)*(lh/2)) + 2
		aside := beside
		if truthy(r.lean) {
			aside = max(0, min(r.lean, beside-14))
		}
		// Manual offsets are tuned to the original spacing. Compact labels are placed beside
		// their routed arrows so an offset cannot leave them floating outside a boundary.
		offX, offY := 0.0, 0.0
		if !tight {
			offX, offY = offset(rel.OffsetX), offset(rel.OffsetY)
		}
		dx := float64(r.nx*aside) + offX
		dy := float64(r.ny*aside) + offY
		at := 0.5
		if careful {
			// Slide along the line to the spot that covers least. Offsets from the diagram
			// are kept unless a spot on the line itself covers less than any they lead to:
			// they are often tuned for Mermaid's spacing.
			least := math.Inf(1)
			moved := dx != float64(r.nx*aside) || dy != float64(r.ny*aside)
			passes := 1
			if moved {
				passes = 2
			}
			for pass := 0; pass < passes && least > 0; pass++ {
				px, py := dx, dy
				if pass > 0 {
					px, py = float64(r.nx*aside), float64(r.ny*aside)
				}
				for _, t := range slide {
					cx := r.x0 + float64((r.x1-r.x0)*t) + px
					cy := r.y0 + float64((r.y1-r.y0)*t) + py
					area := covered(cx-lw/2, cy-lh/2, cx+lw/2, cy+lh/2)
					if area < least {
						least = area
						at = t
						dx = px
						dy = py
					}
					if area == 0 {
						break
					}
				}
			}
		}
		cx := r.x0 + float64((r.x1-r.x0)*at) + dx
		top := r.y0 + float64((r.y1-r.y0)*at) + dy - lh/2
		if careful {
			taken = append(taken, cx-lw/2, top, cx+lw/2, top+lh)
		}
		grow(cx-lw/2, top, cx+lw/2, top+lh)
		href := link(rel.Link, o)
		patch := scene.Rect{X: cx - lw/2, Y: top, W: lw, H: lh}
		it := scene.Item{
			Kind: scene.Text, Role: RoleRelLabel, ID: id,
			Box:     patch,
			Parts:   []scene.Part{{Path: new(scene.Path).RoundRect(patch, patchRadius), Paint: scene.FillOnly}},
			At:      scene.Point{X: cx, Y: top + label.Height/2},
			Style:   style.Style{Fill: style.Bg, Stroke: style.None, Text: style.Fg},
			Author:  style.Style{Text: author.Text},
			Link:    href,
			Tooltip: text.DecodeEntities(rel.Descr),
		}
		if !label.Empty() {
			it.Label = &label
		}
		labels = append(labels, it)
		if !techn.Empty() {
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: RoleRelTechn, ID: id, Label: &techn,
				At:     scene.Point{X: cx, Y: top + label.Height + techn.Height/2},
				Style:  style.Style{Text: style.Muted},
				Author: style.Style{Text: author.Text},
				Link:   href,
			})
		}
	}

	var clusters []scene.Item
	for _, gr := range order {
		if gr == root {
			continue
		}
		b := gr.boundary
		author := userStyle(b.BgColor, b.BorderColor, b.FontColor)
		// The heading is set on the boundary's fill, in the color that stands out on a fill
		// the author gives it, as Pele's inkFill picks it.
		ground := textGround(userDecls(b.BgColor, b.BorderColor, b.FontColor))
		frame := scene.Rect{X: gr.x, Y: gr.y, W: gr.w, H: gr.h}
		it := scene.Item{
			Kind: scene.Group, Role: RoleBoundary, Shape: "rect", ID: b.Alias,
			Box:    frame,
			Parts:  []scene.Part{{Box: frame, Rounded: true}},
			Style:  style.Style{Fill: style.None, Stroke: style.Border, Dash: style.Dashes(6, 4)},
			Author: style.Style{Fill: author.Fill, Stroke: author.Stroke},
		}
		if b.NodeType != "" {
			it.Role = RoleDeployment
			it.Tags = []string{b.NodeType}
			it.Style.Dash = style.Dash{}
		}
		clusters = append(clusters, it)
		href := link(b.Link, o)
		x := gr.x + gr.headX
		y := gr.y + headTop
		for k, role := range [...]scene.Role{RoleBoundaryLabel, RoleBoundaryType, RoleBoundaryDescr} {
			label := &gr.labels[k]
			if !label.Empty() {
				own := style.Style{Text: style.Muted, TextGround: ground}
				if k == 0 {
					own.Text = style.Fg
				}
				clusters = append(clusters, scene.Item{
					Kind: scene.Text, Role: role, ID: b.Alias, Label: label, Align: scene.Start,
					At:     scene.Point{X: x, Y: y + label.Height/2},
					Style:  own,
					Author: style.Style{Text: author.Text},
					Link:   href,
				})
			}
			y += label.Height
		}
	}

	title := l.label(db.Title, size, 4000, text.Bold)
	if tight {
		title = l.label(db.Title, size, common.TitleRoom(o), text.Bold)
	}
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	inner := max(maxX-minX, title.Width)
	width := math.Ceil(inner + 2*pad)
	height := math.Ceil(maxY - minY + titleHeight + 2*pad)
	ox := pad - minX + (inner-(maxX-minX))/2
	oy := pad + titleHeight - minY

	var shapes []scene.Item
	for _, gr := range order {
		for _, n := range gr.nodes {
			shape := n.shape
			external := strings.HasPrefix(shape.Kind, "external")
			decls := userDecls(shape.BgColor, shape.BorderColor, shape.FontColor)
			author := userStyle(shape.BgColor, shape.BorderColor, shape.FontColor)
			// The texts are set on the shape's fill, in the color that stands out on a fill the
			// author gives it, as Pele's inkFill picks it: the name on the shape's own, and the
			// rest on the fill under them.
			inks := css.InksText(decls, decls)
			ground := textGround(decls)
			w, h, form := n.w, n.h, n.form
			// The middle of the shape, which Pele draws its inside about.
			nx := ox + (n.x + w/2)
			ny := oy + (n.y + h/2)
			dx := 0.0
			if form == "h-cyl" {
				dx = -queueRx(h) / 2
			}
			dy := 0.0
			switch form {
			case "person":
				dy = head / 2
			case "cyl":
				dy = cylRy(w) / 2
			}
			own := style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg, OnFill: inks}
			if external {
				own.Fill = style.Bg
				own.Dash = style.Dashes(4, 3)
			}
			href := link(shape.Link, o)
			tags := []string{shape.Kind}
			if external {
				tags = append(tags, "external")
			}
			it := scene.Item{
				Kind: scene.Node, Role: RoleElement, Shape: scene.Shape(form), ID: shape.Alias, Tags: tags,
				Box:    scene.Rect{X: nx - w/2, Y: ny - h/2, W: w, H: h},
				Parts:  draw.Outline(form, nx, ny, w, h),
				Style:  own,
				Author: author,
				Link:   href,
			}
			labels := &n.labels
			y := dy - n.textH/2
			ys := [4]float64{}
			for k := range labels {
				ys[k] = y + labels[k].Height/2
				if k == 2 {
					y += labels[k].Height + 4
				} else {
					y += labels[k].Height
				}
			}
			if !labels[1].Empty() {
				it.Label = &labels[1]
				it.At = scene.Point{X: nx + dx, Y: ny + ys[1]}
			}
			shapes = append(shapes, it)
			for _, k := range [...]int{0, 2, 3} {
				if labels[k].Empty() {
					continue
				}
				role, own := RoleElementType, style.Style{Text: style.Muted, TextGround: ground}
				switch k {
				case 2:
					role = RoleElementTechn
				case 3:
					role, own = RoleElementDescr, style.Style{Text: style.Fg, TextGround: ground}
				}
				shapes = append(shapes, scene.Item{
					Kind: scene.Text, Role: role, ID: shape.Alias, Label: &labels[k],
					At:     scene.Point{X: nx + dx, Y: ny + ys[k]},
					Style:  own,
					Author: style.Style{Text: author.Text},
					Link:   href,
				})
			}
		}
	}

	sc := &scene.Scene{
		Type:     "c4",
		Width:    width,
		Height:   height,
		FontSize: size,
		Title:    db.AccTitle,
		Desc:     db.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	for _, layer := range [][]scene.Item{clusters, casings, edges} {
		for i := range layer {
			layer[i].Move(ox, oy)
		}
		sc.Items = append(sc.Items, layer...)
	}
	sc.Items = append(sc.Items, shapes...)
	for i := range labels {
		labels[i].Move(ox, oy)
	}
	sc.Items = append(sc.Items, labels...)
	return sc, ls.squeezed, nil
}
