package quadrant

import (
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The scene's roles: "quadrant.title" for the title; "quadrant.quadrant" for each of the
// four quadrants, its id its number, with its label; "quadrant.axis" for each label of the
// axes, those of the y axis turned; "quadrant.point" for each point, its id its label's
// text, drawn the last first, as Mermaid puts each new point under the ones before it; and
// "quadrant.point-label" for each point's label, in the order of the points.
const (
	RoleTitle      scene.Role = "quadrant.title"
	RoleQuadrant   scene.Role = "quadrant.quadrant"
	RoleAxis       scene.Role = "quadrant.axis"
	RolePoint      scene.Role = "quadrant.point"
	RolePointLabel scene.Role = "quadrant.point-label"
)

// The measures of Pele's quadrant render: the gap between the quadrants, and the most points
// whose labels look for a free place.
const (
	gap        = 4
	mostPoints = 300
)

// dot is a point drawn, with how far it reaches from its middle and its label, Pele's Dot.
type dot struct {
	cx, cy, reach float64
	text          text.Label
}

// setting is a setting of the chart's own section of the config, as Pele's setting reads it.
func setting(config diagram.Config, key string) any {
	section, _ := config["quadrantChart"].(diagram.Config)
	return section[key]
}

// length is a setting that is a measure, as Pele's length reads it: a number from 0 up, and
// at most hi, or fallback.
func length(config diagram.Config, key string, fallback, hi float64) float64 {
	if v, ok := setting(config, key).(float64); ok && v >= 0 {
		return min(v, hi)
	}
	return fallback
}

// hex is a hex code with its #, as Pele's hex writes it.
func hex(color string) string {
	if strings.HasPrefix(color, "#") {
		return color
	}
	return "#" + color
}

// pick is the first of the values that is set, as JavaScript's ?? picks it.
func pick[T any](values ...*T) *T {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

// Layout lays a quadrant chart out as a scene, as Pele's renderQuadrant draws it: four
// quadrants in a square plot as wide as the host allows, the labels of the axes about it,
// and each point with its label below it, or above, right or left of it where that is free.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	size := o.Size()
	axisSize := jsmath.Round(size * 0.875)
	pointSize := jsmath.Round(size * 0.75)
	measure := o.Measure()
	natural := jsmath.Round(max(100, length(config, "chartWidth", 500, 4000)))
	width := jsmath.Round(common.CapWidth(o, natural, 280, 0))
	height := jsmath.Round(max(100, length(config, "chartHeight", 500, 4000)) * width / natural)
	outer := length(config, "quadrantPadding", common.Padding, 200)
	xPad := length(config, "xAxisLabelPadding", 5, 200)
	yPad := length(config, "yAxisLabelPadding", 5, 200)
	hasPoints := len(m.Points) > 0

	label := func(t *Text, fontSize, most float64) text.Label {
		if t == nil {
			return text.Label{}
		}
		return text.Layout(t.Text, text.Options{Measurer: measure, Size: fontSize, MaxWidth: max(most, 40), Markdown: t.Markdown, Icons: o.Icons})
	}

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: max(width-float64(2*outer), 40), Face: text.Bold, Icons: o.Icons})
	titlePad := length(config, "titlePadding", 10, 200)
	titleSpace := 0.0
	if title.Height > 0 {
		titleSpace = title.Height + titlePad
	}

	filled := func(t *Text) bool { return t != nil && t.Text != "" }
	hasX := filled(m.XAxisLeft) || filled(m.XAxisRight)
	hasY := filled(m.YAxisBottom) || filled(m.YAxisTop)
	// Mermaid moves the x axis labels below the chart as soon as there are points.
	xBelow := hasPoints || setting(config, "xAxisPosition") == "bottom"
	yRight := setting(config, "yAxisPosition") == "right"

	side := width - float64(2*outer)
	room := side
	if m.XAxisRight != nil {
		room = side/2 - 16
	}
	xLeft := label(m.XAxisLeft, axisSize, room)
	xRight := label(m.XAxisRight, axisSize, side/2-16)
	xHeight := max(xLeft.Height, xRight.Height)
	xSpace := 0.0
	if hasX {
		xSpace = xHeight + xPad
	}
	tall := height - float64(2*outer) - titleSpace - xSpace
	room = tall
	if m.YAxisTop != nil {
		room = tall/2 - 16
	}
	yBottom := label(m.YAxisBottom, axisSize, room)
	yTop := label(m.YAxisTop, axisSize, tall/2-16)
	yHeight := max(yBottom.Height, yTop.Height)
	ySpace := 0.0
	if hasY {
		ySpace = yHeight + yPad
	}

	left := outer
	if !yRight {
		left += ySpace
	}
	top := outer + titleSpace
	if !xBelow {
		top += xSpace
	}
	w := max(0, side-ySpace)
	h := max(0, tall)
	tileW := max(0, (w-gap)/2)
	tileH := max(0, (h-gap)/2)

	// Boxes a point label keeps clear of: the gaps between quadrants, quadrant labels,
	// points, and the labels placed before it.
	taken := []float64{left + tileW, top, left + w - tileW, top + h, left, top + tileH, left + w, top + h - tileH}
	isClear := func(from int, x0, y0, x1, y1 float64) bool {
		for k := from; k < len(taken); k += 4 {
			if x0 < taken[k+2] && x1 > taken[k] && y0 < taken[k+3] && y1 > taken[k+1] {
				return false
			}
		}
		return true
	}

	var items []scene.Item
	if !title.Empty() {
		items = append(items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: width / 2, Y: titlePad/2 + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}

	textTop := length(config, "quadrantTextTopPadding", 5, 200)
	for k := range 4 {
		x := left
		if k == 0 || k == 3 {
			x = left + w - tileW
		}
		y := top
		if k >= 2 {
			y = top + h - tileH
		}
		l := label(m.Quadrants[k], size, tileW-16)
		cx := x + tileW/2
		cy := y + tileH/2
		if hasPoints {
			cy = y + textTop + l.Height/2
		}
		taken = append(taken, cx-l.Width/2, cy-l.Height/2, cx+l.Width/2, cy+l.Height/2)
		box := scene.Rect{X: x, Y: y, W: tileW, H: tileH}
		it := scene.Item{
			Kind: scene.Node, Role: RoleQuadrant, Shape: "rect", ID: string(rune('1' + k)),
			Box:   box,
			Parts: []scene.Part{{Box: box, Rounded: true}},
			At:    scene.Point{X: cx, Y: cy},
			Style: style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg},
		}
		if !l.Empty() {
			it.Label = &l
		}
		items = append(items, it)
	}

	axis := func(l text.Label, x, y, turn float64) {
		if l.Empty() {
			return
		}
		items = append(items, scene.Item{
			Kind: scene.Text, Role: RoleAxis, Label: &l, At: scene.Point{X: x, Y: y}, Rotate: turn,
			Style: style.Style{Text: style.Muted},
		})
	}
	if hasX {
		y := top - xPad - xHeight/2
		if xBelow {
			y = top + h + xPad + xHeight/2
		}
		x := left + xLeft.Width/2
		if xRight.Height > 0 {
			x = left + w/4
		}
		axis(xLeft, x, y, 0)
		axis(xRight, left+(w*3)/4, y, 0)
	}
	if hasY {
		x := left - yPad - yHeight/2
		if yRight {
			x = left + w + yPad + yHeight/2
		}
		y := top + h - yBottom.Width/2
		if yTop.Height > 0 {
			y = top + (h*3)/4
		}
		axis(yBottom, x, y, -90)
		axis(yTop, x, top+h/4, -90)
	}

	radius := length(config, "pointRadius", 5, 200)
	textPad := length(config, "pointTextPadding", 5, 200)
	var dots []dot
	var circles []scene.Item
	for i := range m.Points {
		point := &m.Points[i]
		if !(point.X >= 0 && point.X <= 1 && point.Y >= 0 && point.Y <= 1) {
			continue
		}
		shared := m.Classes[point.Class]
		r := min(*pick(point.Radius, shared.Radius, &radius), 2000)
		color := pick(point.Color, shared.Color)
		strokeColor := pick(point.StrokeColor, shared.StrokeColor)
		strokeWidth := pick(point.StrokeWidth, shared.StrokeWidth)
		var styles []string
		if color != nil {
			styles = append(styles, "fill:"+hex(*color))
		}
		if strokeWidth != nil {
			if strokeColor != nil {
				styles = append(styles, "stroke:"+hex(*strokeColor))
			}
			styles = append(styles, "stroke-width:"+*strokeWidth)
		}
		cx := left + float64(point.X*w)
		cy := top + float64((1-point.Y)*h)
		reach := r
		if strokeWidth != nil {
			reach = r + min(chars.ParseInt(*strokeWidth, 10), 400)/2
		}
		dots = append(dots, dot{cx: cx, cy: cy, reach: reach, text: label(&point.Text, pointSize, 200)})
		it := scene.Item{
			Kind: scene.Mark, Role: RolePoint, Shape: "circle", ID: point.Text.Text,
			Box:    scene.Rect{X: cx - r, Y: cy - r, W: 2 * r, H: 2 * r},
			Parts:  []scene.Part{{Path: new(scene.Path).Ellipse(cx, cy, r, r)}},
			Style:  style.Style{Fill: style.Series(0), Stroke: style.None},
			Author: style.Declarations(styles).Style,
		}
		if point.Class != "" {
			it.Class = []string{point.Class}
		}
		// A stroke given a width and no color is drawn in the points' color.
		if strokeWidth != nil && strokeColor == nil {
			it.Style.Stroke = style.Series(0)
		}
		circles = append(circles, it)
	}
	// Mermaid puts each new point under the ones before it.
	slices.Reverse(circles)
	items = append(items, circles...)

	// A label goes below its point, or above, right or left of it when that is the first
	// place that is inside the chart and free. A second round lets it cross the gaps between
	// quadrants. Large charts skip the search for a free place.
	careful := len(dots) <= mostPoints
	if careful {
		for _, d := range dots {
			taken = append(taken, d.cx-d.reach, d.cy-d.reach, d.cx+d.reach, d.cy+d.reach)
		}
	}
	for _, d := range dots {
		halfW := d.text.Width / 2
		halfH := float64(d.text.Size*0.6) + (d.text.Height-d.text.LineHeight)/2
		away := d.reach + textPad
		spots := [8]float64{d.cx, d.cy + away + halfH, d.cx, d.cy - away - halfH, d.cx + away + halfW, d.cy, d.cx - away - halfW, d.cy}
		tx, ty := d.cx, spots[1]
		found := false
		for k := 0; k < 16; k += 2 {
			spot := k % 8
			x := spots[spot]
			if spot < 4 && d.text.Width < w {
				x = min(max(spots[spot], left+halfW), left+w-halfW)
			}
			y := spots[spot+1]
			if x-halfW < left || x+halfW > left+w || y-halfH < top || y+halfH > top+h {
				continue
			}
			from := 8
			if k < 8 {
				from = 0
			}
			free := !careful || isClear(from, x-halfW, y-halfH, x+halfW, y+halfH)
			if !found || free {
				tx, ty = x, y
				found = true
			}
			if free {
				break
			}
		}
		if careful {
			taken = append(taken, tx-halfW, ty-halfH, tx+halfW, ty+halfH)
		}
		if !d.text.Empty() {
			l := d.text
			items = append(items, scene.Item{
				Kind: scene.Text, Role: RolePointLabel, Label: &l, At: scene.Point{X: tx, Y: ty},
				Style: style.Style{Text: style.Fg},
			})
		}
	}

	return &scene.Scene{
		Type:     "quadrantChart",
		Width:    width,
		Height:   height,
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
		Items:    items,
	}
}
