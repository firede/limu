package pie

import (
	"math"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/yaml"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The scene's roles: "pie.title" for the title; "pie.slice" for a slice, or for the empty
// circle of a pie with no slice to draw; "pie.slice-label" for a slice's percentage, on the
// slice in the background's color or outside the rim in the text's; and "pie.legend" for an
// entry of the legend, its swatch with its label.
const (
	RoleTitle      scene.Role = "pie.title"
	RoleSlice      scene.Role = "pie.slice"
	RoleSliceLabel scene.Role = "pie.slice-label"
	RoleLegend     scene.Role = "pie.legend"
)

// The measures of Pele's pie render.
const (
	radius0 = 120
	swatch  = 12
	row     = 26
	gap     = 32
	outside = 28
	pad     = common.Padding
)

// option is a setting of the pie's config, as Pele's option reads it: nil where the config
// has no pie section that is an object.
func option(config diagram.Config, key string) any {
	section, _ := config["pie"].(diagram.Config)
	return section[key]
}

// point is where an angle, clockwise from the top, meets a circle of a radius about the
// pie's center, as Pele's point gives it, about that center. Each product is rounded on its
// own, as JavaScript rounds it, before a sum takes it.
func point(angle, r float64) (float64, float64) {
	return float64(jsmath.Sin(angle) * r), float64(-jsmath.Cos(angle) * r)
}

// slice is the outline of a slice from angle start to end, with an outer radius and an inner
// one for a donut, about the center cx, cy, as Pele's slice draws it: a whole ring as two
// half arcs, or a wedge from the center, or a piece of a ring.
func slice(cx, cy, start, end, outer, inner float64) *scene.Path {
	p := new(scene.Path)
	if end-start >= math.Pi*2-1e-9 {
		ring := func(r float64, sweep bool) {
			p.MoveTo(cx, cy-r)
			p.ArcTo(r, r, 0, true, sweep, cx, cy+r)
			p.ArcTo(r, r, 0, true, sweep, cx, cy-r)
			p.Close()
		}
		ring(outer, true)
		if inner > 0 {
			ring(inner, false)
		}
		return p
	}
	large := end-start > math.Pi
	at := func(angle, r float64) (float64, float64) {
		x, y := point(angle, r)
		return cx + x, cy + y
	}
	if inner <= 0 {
		p.MoveTo(cx, cy)
		p.LineTo(at(start, outer))
		x, y := at(end, outer)
		return p.ArcTo(outer, outer, 0, large, true, x, y).Close()
	}
	p.MoveTo(at(start, outer))
	x, y := at(end, outer)
	p.ArcTo(outer, outer, 0, large, true, x, y)
	p.LineTo(at(end, inner))
	x, y = at(start, inner)
	return p.ArcTo(inner, inner, 0, large, false, x, y).Close()
}

// Layout lays a pie chart out as a scene, as Pele's renderPie draws it: the pie, its
// percentages on the slices, or outside the rim for a slice too thin for its own, and a
// legend beside, above, below or in the middle of it, as the config says. A pie that does not
// fit the host's width is drawn smaller, with a legend at its side put under it.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.FitWidth(o, radius0*2, 140, func(diameter float64, tight bool) *scene.Scene {
		return draw(m, config, o, diameter/2, tight)
	})
}

// clearing is a label placed outside the rim, about the pie's center, as Pele's placedOutside
// holds it.
type clearing struct{ x, y, width float64 }

// draw lays the pie out with a radius, as Pele's draw does; tight is set when the chart does
// not fit the width it has with its legend beside it.
func draw(m *Model, config diagram.Config, o *layout.Options, radius float64, tight bool) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	measure := o.Measure()

	textPosition := 0.75
	if v, ok := option(config, "textPosition").(float64); ok && v >= 0 && v <= 1 {
		textPosition = v
	}
	inner := 0.0
	if v, ok := option(config, "donutHole").(float64); ok && v > 0 && v <= 0.9 {
		inner = v * radius
	}
	placed := "right"
	if v := option(config, "legendPosition"); v != nil {
		placed = yaml.String(v)
	}
	legendPosition := placed
	if tight && (placed == "left" || placed == "right") {
		legendPosition = "bottom"
	}

	entries := m.Sections
	sum := 0.0
	for _, e := range entries {
		sum += e.Value
	}

	legend := make([]text.Label, len(entries))
	for i, e := range entries {
		raw := e.Label
		if m.ShowData {
			raw = e.Label + " [" + chars.NumberString(e.Value) + "]"
		}
		legend[i] = text.Layout(raw, text.Options{Measurer: measure, Size: small, MaxWidth: 4000, Icons: o.Icons})
	}
	legendWidth := 0.0
	for _, l := range legend {
		legendWidth = max(legendWidth, l.Width)
	}
	if len(legend) > 0 {
		legendWidth += swatch + 8
	}
	legendHeight := float64(len(legend) * row)

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: common.TitleRoom(o), Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 16
	}

	// A slice too thin for its percentage gets the label outside the rim, which needs a
	// margin.
	shares := make([]float64, len(entries))
	for i, e := range entries {
		share := 0.0
		if sum > 0 {
			share = e.Value / sum
		}
		if share*100 >= 1 && jsmath.ToFixed(share*100, 0) != "0" {
			shares[i] = share
		}
	}
	narrow := make([]bool, len(entries))
	margin := 0.0
	for i, share := range shares {
		if share == 0 || share >= 0.5 {
			continue
		}
		chord := 2 * radius * textPosition * jsmath.Sin(share*math.Pi)
		if chord < measure.Width(jsmath.ToFixed(share*100, 0)+"%", small, 0)+6 {
			narrow[i] = true
			margin = outside
		}
	}

	side := ""
	if legendPosition == "left" || legendPosition == "right" || legendPosition == "center" {
		side = legendPosition
	}
	stacked := legendPosition == "top" || legendPosition == "bottom"
	diameter := (radius + margin) * 2
	width, height := diameter, diameter
	if side == "left" || side == "right" {
		if legendWidth > 0 {
			width += gap + legendWidth
		}
		height = max(height, legendHeight)
	} else if stacked {
		width = max(width, legendWidth)
		if legendHeight > 0 {
			height += 20 + legendHeight
		}
	}
	width = max(width, title.Width)

	var cx float64
	switch side {
	case "left":
		cx = pad + (legendWidth + gap + diameter/2)
	case "right":
		cx = pad + diameter/2
	default:
		cx = pad + width/2
	}
	bodyTop := pad + titleHeight
	var cy float64
	switch {
	case legendPosition == "top":
		cy = bodyTop + (legendHeight + 20 + diameter/2)
	case stacked:
		cy = bodyTop + diameter/2
	default:
		cy = bodyTop + height/2
	}

	var slices, labels, outsides []scene.Item
	angle := 0.0
	rim := radius + outside/2
	var placedOutside []clearing
	clear := func(x, y, width float64) bool {
		for _, other := range placedOutside {
			if !(math.Abs(x-other.x) >= (width+other.width)/2+2 || math.Abs(y-other.y) >= small+2) {
				return false
			}
		}
		return true
	}
	sliceStyle := style.Style{Stroke: style.Bg, StrokeWidth: style.N(1), Join: style.JoinRound}
	for index, e := range entries {
		share := shares[index]
		if share == 0 {
			continue
		}
		end := angle + float64(share*math.Pi*2)
		path := slice(cx, cy, angle, end, radius, inner)
		st := sliceStyle
		st.Fill = style.Series(index)
		slices = append(slices, scene.Item{
			Kind: scene.Mark, Role: RoleSlice, ID: e.Label,
			Box: path.Bounds(), At: scene.Point{X: cx, Y: cy},
			Parts: []scene.Part{{Path: path}},
			Style: st,
		})
		mid := (angle + end) / 2
		s := jsmath.ToFixed(share*100, 0) + "%"
		if narrow[index] {
			width := measure.Width(s, small, 0)
			at := mid
			for step := 0; step < 360; step++ {
				x, y := point(at, rim)
				if clear(x, y, width) {
					break
				}
				at += 2 / rim
			}
			x, y := point(at, rim)
			placedOutside = append(placedOutside, clearing{x, y, width})
			l := common.Line(s, width, small, 0)
			outsides = append(outsides, scene.Item{
				Kind: scene.Text, Role: RoleSliceLabel, Label: l,
				At: scene.Point{X: cx + x, Y: cy + y}, Style: style.Style{Text: style.Fg},
			})
		} else {
			x, y := point(mid, radius*textPosition)
			l := common.Line(s, measure.Width(s, small, 0), small, 0)
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: RoleSliceLabel, Label: l,
				At: scene.Point{X: cx + x, Y: cy + y}, Style: style.Style{Text: style.Bg},
			})
		}
		angle = end
	}
	if len(slices) == 0 {
		path := new(scene.Path).Ellipse(cx, cy, radius, radius)
		st := sliceStyle
		st.Stroke = style.Border
		slices = append(slices, scene.Item{
			Kind: scene.Mark, Role: RoleSlice,
			Box: path.Bounds(), At: scene.Point{X: cx, Y: cy},
			Parts: []scene.Part{{Path: path, Paint: scene.StrokeOnly}},
			Style: st,
		})
	}

	var lx, ly float64
	switch {
	case side == "right":
		lx = cx + diameter/2 + gap
		ly = bodyTop + (height-legendHeight)/2
	case side == "left":
		lx = pad
		ly = bodyTop + (height-legendHeight)/2
	case side == "center":
		lx = cx - legendWidth/2
		ly = cy - legendHeight/2
	default:
		lx = pad + (width-legendWidth)/2
		if legendPosition == "top" {
			ly = bodyTop
		} else {
			ly = bodyTop + diameter + 20
		}
	}
	var legendItems []scene.Item
	for index, l := range legend {
		y := ly + float64(float64(index)*row) + row/2
		it := scene.Item{
			Kind: scene.Node, Role: RoleLegend, Shape: "rect", ID: entries[index].Label,
			Box:   scene.Rect{X: lx, Y: y - row/2, W: swatch + 8 + l.Width, H: row},
			Parts: []scene.Part{{Box: scene.Rect{X: lx, Y: y - swatch/2, W: swatch, H: swatch}, Rounded: true, Paint: scene.FillOnly}},
			At:    scene.Point{X: lx + swatch + 8 + l.Width/2, Y: y},
			Style: style.Style{Fill: style.Series(index), Text: style.Fg, Radius: style.N(2)},
		}
		if !l.Empty() {
			it.Label = &legend[index]
		}
		legendItems = append(legendItems, it)
	}

	totalWidth := math.Ceil(width + 2*pad)
	totalHeight := math.Ceil(height + titleHeight + 2*pad)
	sc := &scene.Scene{
		Type:     "pie",
		Width:    totalWidth,
		Height:   totalHeight,
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At: scene.Point{X: totalWidth / 2, Y: pad + title.Height/2}, Style: style.Style{Text: style.Fg},
		})
	}
	sc.Items = append(sc.Items, slices...)
	sc.Items = append(sc.Items, labels...)
	sc.Items = append(sc.Items, outsides...)
	sc.Items = append(sc.Items, legendItems...)
	return sc
}
