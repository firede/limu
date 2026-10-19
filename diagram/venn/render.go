package venn

import (
	"math"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The scene's roles: "venn.title" for the title above the diagram; "venn.set" for each set's
// circle, its id the set's; "venn.overlap" for the outline of an overlap a style gives a look
// of its own, its id its sets joined by |; "venn.label" for the label of a set or an overlap,
// its id its sets joined by |; and "venn.text" for each text written in a region, its id the
// text's, under the region's label.
const (
	RoleTitle   scene.Role = "venn.title"
	RoleSet     scene.Role = "venn.set"
	RoleOverlap scene.Role = "venn.overlap"
	RoleLabel   scene.Role = "venn.label"
	RoleText    scene.Role = "venn.text"
)

// The measures of Pele's venn render: the radius the largest circle is drawn at, the most the
// scale grows for what is written in the regions, and the opacity sets are filled at.
const (
	radius      = 100
	maxGrowth   = 1.6
	fillOpacity = 0.2
	pad         = common.Padding
)

// line is a line of what is written in a region, its label first, then its texts; id is that
// of a text, and isText whether it is one.
type line struct {
	id     string
	isText bool
	raw    string
	size   float64
	base   text.Face
	label  text.Label
}

// region is a set or a stated overlap, Pele's Region.
type region struct {
	sets   []string
	inside []int
	spot   spot
	lines  []line
	decls  []string
}

// keyOf is one key for a list of set ids, as Pele's keyOf gives one: a single id is the key,
// and a list of more is written so that it cannot be mistaken for one, as an id holds no
// quotation mark.
func keyOf(ids []string) string {
	if len(ids) == 1 {
		return ids[0]
	}
	return `["` + strings.Join(ids, `","`) + `"]`
}

// Layout lays a venn diagram out as a scene, as Pele's renderVenn draws it: a circle for each
// set, filled in its series' color, scaled so that the largest has a radius of 100 or, within
// limits, more for what is written in the regions; each region's label and texts where the
// region has most room; and the overlaps a style gives a look of their own outlined. Pele
// reads nothing of the config for a venn diagram, and draws it at its own size whatever the
// width of the host.
func Layout(m *Model, _ diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	tiny := jsmath.Round(size * 0.8125)
	measure := o.Measure()
	layText := func(raw string, fontSize, most float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: fontSize, MaxWidth: most, Face: face, Icons: o.Icons})
	}

	circles := layoutCircles(m.Subsets)
	index := map[string]int{}
	for i, c := range circles {
		index[c.id] = i
	}

	// Styles for the same targets add up, later declarations winning.
	styles := map[string][]Declaration{}
	for _, s := range m.Styles {
		key := keyOf(s.Targets)
		merged := styles[key]
		for _, d := range s.Styles {
			merged = setDeclaration(merged, d.Property, d.Value)
		}
		styles[key] = merged
	}
	declsOf := func(key string) []string {
		var list []string
		for _, d := range styles[key] {
			list = append(list, d.Property+":"+d.Value)
		}
		return list
	}

	// Circles that touch each circle; only those can cut into a region of it.
	neighbors := make([][]int, len(circles))
	for a := range circles {
		for b := a + 1; b < len(circles); b++ {
			if jsmath.Hypot(circles[a].x-circles[b].x, circles[a].y-circles[b].y) < circles[a].r+circles[b].r {
				neighbors[a] = append(neighbors[a], b)
				neighbors[b] = append(neighbors[b], a)
			}
		}
	}

	// One region for each set and each stated overlap; a later statement about the same sets
	// replaces an earlier one.
	var order []string
	stated := map[string]*Subset{}
	for i := range m.Subsets {
		key := keyOf(m.Subsets[i].Sets)
		if _, ok := stated[key]; !ok {
			order = append(order, key)
		}
		stated[key] = &m.Subsets[i]
	}
	regions := map[string]*region{}
	var all []*region
	for _, key := range order {
		subset := stated[key]
		var sets []string
		seen := map[string]bool{}
		for _, id := range subset.Sets {
			if !seen[id] {
				seen[id] = true
				sets = append(sets, id)
			}
		}
		inside := make([]int, len(sets))
		within := map[int]bool{}
		for k, id := range sets {
			inside[k] = index[id]
			within[inside[k]] = true
		}
		smallest := inside[0]
		for _, i := range inside {
			if circles[i].r < circles[smallest].r {
				smallest = i
			}
		}
		decls := declsOf(key)
		author := style.Declarations(decls)
		var raw string
		has := false
		if subset.Label != nil {
			raw, has = *subset.Label, true
		} else if len(subset.Sets) == 1 {
			raw, has = subset.Sets[0], true
		}
		var base text.Face
		if author.Bold {
			base |= text.Bold
		}
		if author.Italic {
			base |= text.Italic
		}
		var outside []int
		for _, i := range neighbors[smallest] {
			if !within[i] {
				outside = append(outside, i)
			}
		}
		at := findSpot(circles, inside, outside)
		// A region that other circles cover, or all but cover, is written across them
		// instead.
		if at.margin < circles[smallest].r*0.2 {
			at = findSpot(circles, inside, nil)
		}
		r := &region{sets: sets, inside: inside, spot: at, decls: decls}
		if has && raw != "" {
			fontSize := small
			if len(sets) == 1 {
				fontSize = size
			}
			r.lines = append(r.lines, line{raw: raw, size: fontSize, base: base, label: layText(raw, fontSize, 4000, base)})
		}
		regions[key] = r
		all = append(all, r)
	}
	for _, node := range m.TextNodes {
		raw := node.ID
		if node.Label != nil {
			raw = *node.Label
		}
		if r := regions[keyOf(node.Sets)]; r != nil {
			r.lines = append(r.lines, line{id: node.ID, isText: true, raw: raw, size: tiny, label: layText(raw, tiny, 4000, 0)})
		}
	}

	// Lengths so far are in units of the layout. The scale gives the largest circle a set
	// radius, and grows, within limits, until every region holds what is written in it.
	largest := 0.0
	for _, c := range circles {
		if c.r > largest {
			largest = c.r
		}
	}
	scale := 1.0
	if largest > 0 {
		scale = radius / largest
	}
	growth := 1.0
	for _, r := range all {
		if r.spot.margin <= 0 {
			continue
		}
		width, height := 0.0, 0.0
		for _, l := range r.lines {
			width = max(width, l.label.Width)
			height += l.label.Height
		}
		need := jsmath.Hypot(width, height)/2 + 4
		growth = max(growth, min(need/(r.spot.margin*scale), maxGrowth))
	}
	scale = float64(scale * growth)

	// What is still too wide for its region is wrapped to it.
	for _, r := range all {
		room := max(r.spot.margin*scale*1.8, 72)
		for k := range r.lines {
			l := &r.lines[k]
			if l.label.Width > room {
				l.label = layText(l.raw, l.size, room, l.base)
			}
		}
	}

	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, c := range circles {
		minX = min(minX, (c.x-c.r)*scale)
		minY = min(minY, (c.y-c.r)*scale)
		maxX = max(maxX, (c.x+c.r)*scale)
		maxY = max(maxY, (c.y+c.r)*scale)
	}
	for _, r := range all {
		wide, tall := 0.0, 0.0
		for _, l := range r.lines {
			wide = max(wide, l.label.Width)
			tall += l.label.Height
		}
		minX = min(minX, float64(r.spot.x*scale)-wide/2)
		maxX = max(maxX, float64(r.spot.x*scale)+wide/2)
		minY = min(minY, float64(r.spot.y*scale)-tall/2)
		maxY = max(maxY, float64(r.spot.y*scale)+tall/2)
	}
	if len(circles) == 0 {
		minX, minY, maxX, maxY = 0, 0, 0, 0
	}

	titleText := ""
	if m.Title != "" {
		titleText = normalizeText(m.Title)
	}
	title := layText(titleText, size, 4000, text.Bold)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	inner := max(maxX-minX, title.Width)
	ox := pad + (inner-(maxX-minX))/2 - minX
	oy := pad + titleHeight - minY
	width := math.Ceil(inner + 2*pad)
	height := math.Ceil(maxY - minY + titleHeight + 2*pad)
	placed := make([]circle, len(circles))
	for i, c := range circles {
		placed[i] = circle{id: c.id, x: float64(c.x*scale) + ox, y: float64(c.y*scale) + oy, r: float64(c.r * scale)}
	}

	sc := &scene.Scene{Type: "venn", Width: width, Height: height, FontSize: size}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}

	for i, c := range placed {
		var decls []string
		if r := regions[c.id]; r != nil {
			decls = r.decls
		}
		own, author := setPaint(i, decls)
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Mark, Role: RoleSet, Shape: "circle", ID: c.id,
			Box:    scene.Rect{X: c.x - c.r, Y: c.y - c.r, W: 2 * c.r, H: 2 * c.r},
			Parts:  []scene.Part{{Path: new(scene.Path).Ellipse(c.x, c.y, c.r, c.r)}},
			Style:  own,
			Author: author,
		})
	}

	for _, r := range all {
		// An overlap is only drawn when a style gives it a look of its own; otherwise the
		// circles show it.
		if len(r.sets) < 2 || style.ShapeCSS(r.decls) == "" {
			continue
		}
		shared := make([]circle, len(r.inside))
		for k, i := range r.inside {
			shared[k] = placed[i]
		}
		path := regionPath(shared)
		if path == nil {
			continue
		}
		author := plainNone(style.Declarations(r.decls).Style)
		author.Text = style.Paint{}
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Mark, Role: RoleOverlap, ID: strings.Join(r.sets, "|"),
			Box:    path.Bounds(),
			Parts:  []scene.Part{{Path: path}},
			Style:  style.Style{Fill: style.None, Stroke: style.None},
			Author: author,
		})
	}

	for _, r := range all {
		total := 0.0
		for _, l := range r.lines {
			total += l.label.Height
		}
		if total == 0 {
			continue
		}
		id := strings.Join(r.sets, "|")
		x := float64(r.spot.x*scale) + ox
		y := float64(r.spot.y*scale) + oy - total/2
		for k := range r.lines {
			l := &r.lines[k]
			if !l.label.Empty() {
				it := scene.Item{
					Kind: scene.Text, Role: RoleLabel, ID: id, Label: &l.label,
					At:     scene.Point{X: x, Y: y + l.label.Height/2},
					Style:  style.Style{Text: style.Fg},
					Author: textStyle(r.decls),
				}
				if l.isText {
					it.Role, it.ID = RoleText, l.id
					it.Style.Text = style.Muted
					it.Author = textStyle(declsOf(l.id))
				}
				sc.Items = append(sc.Items, it)
			}
			y += l.label.Height
		}
	}
	return sc
}

// setPaint is the style of the ith set's circle, Pele's fill and stroke in the set's series
// color with a fill-opacity of 0.2, and what the author's declarations set over it. A browser
// takes a fill, a fill-opacity, a stroke and a stroke-opacity the author set over those Pele
// writes on the circle each on its own, so the author's fill is drawn at Pele's opacity, and
// the series color at the author's.
func setPaint(i int, decls []string) (own, author style.Style) {
	a := style.Declarations(decls)
	author = a.Style
	author.Text = style.Paint{}
	opacity := fillOpacity
	if f, ok := a.FillOpacity.Get(); ok {
		opacity = f
	} else if author.Fill.Visible() {
		author.Fill = author.Fill.Alpha(fillOpacity)
	}
	own = style.Style{Fill: style.Series(i).Alpha(opacity), Stroke: style.Series(i)}
	if f, ok := a.StrokeOpacity.Get(); ok {
		own.Stroke = own.Stroke.Alpha(f)
	}
	return own, plainNone(author)
}

// plainNone is a style with a fill or stroke of none at an opacity, which draws nothing,
// as none.
func plainNone(s style.Style) style.Style {
	if s.Fill.Set() && !s.Fill.Visible() {
		s.Fill = style.None
	}
	if s.Stroke.Set() && !s.Stroke.Visible() {
		s.Stroke = style.None
	}
	return s
}

// textStyle is what of an author's declarations a text takes: its color and its opacity.
func textStyle(decls []string) style.Style {
	s := style.Declarations(decls).Style
	return style.Style{Text: s.Text, Opacity: s.Opacity}
}
