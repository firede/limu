package radar

import (
	"math"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The scene's roles: "radar.title" for the title; "radar.graticule" for the rings of the
// graticule, circles or polygons, as the parts of one item; "radar.axes" for the spokes, one
// line from the center for each axis; "radar.curve" for each curve drawn, its id its name;
// "radar.axis-label" for the label of each axis, its id the axis's name; and "radar.legend"
// for each entry of the legend, its id the curve's name, a swatch with the curve's label.
const (
	RoleTitle     scene.Role = "radar.title"
	RoleGraticule scene.Role = "radar.graticule"
	RoleAxes      scene.Role = "radar.axes"
	RoleCurve     scene.Role = "radar.curve"
	RoleAxisLabel scene.Role = "radar.axis-label"
	RoleLegend    scene.Role = "radar.legend"
)

// The measures of Pele's radar render: the side of a legend's swatch, the height of a row of
// the legend, and the gap between the chart and its legend.
const (
	swatch = 12
	row    = 26
	gap    = 24
)

// point is a place about the chart's center, Pele's Point.
type point struct{ x, y float64 }

// option is a number of the chart's own section of the config, at most hi, or fallback for
// anything but a number of at least lo, as Pele's option reads it.
func option(config diagram.Config, key string, fallback, lo, hi float64) float64 {
	section, _ := config["radar"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v >= lo {
		return min(v, hi)
	}
	return fallback
}

// relativeRadius is how far from the center a value is drawn, clipped to the range from lo
// to hi, as Pele's relativeRadius gives it. It is scaled before it is divided, unless the
// product overflows, where it is divided first.
func relativeRadius(value, lo, hi, radius float64) float64 {
	clipped := min(max(value, lo), hi)
	scaled := radius * (clipped - lo)
	if math.IsInf(scaled, 0) || math.IsNaN(scaled) {
		return radius * ((clipped - lo) / (hi - lo))
	}
	return scaled / (hi - lo)
}

// closedRoundCurve is a closed Catmull-Rom spline through the points, as cubic Bézier
// segments, as Pele's closedRoundCurve draws it.
func closedRoundCurve(points []point, tension float64) *scene.Path {
	n := len(points)
	p := new(scene.Path).MoveTo(points[0].x, points[0].y)
	for i := range n {
		p0 := points[(i-1+n)%n]
		p1 := points[i]
		p2 := points[(i+1)%n]
		p3 := points[(i+2)%n]
		p.CubeTo(
			p1.x+float64((p2.x-p0.x)*tension), p1.y+float64((p2.y-p0.y)*tension),
			p2.x-float64((p3.x-p1.x)*tension), p2.y-float64((p3.y-p1.y)*tension),
			p2.x, p2.y,
		)
	}
	return p.Close()
}

// polygon is a closed figure through the points, as Pele's polygon writes one.
func polygon(points []point) *scene.Path {
	xy := make([]float64, 0, 2*len(points))
	for _, p := range points {
		xy = append(xy, p.x, p.y)
	}
	return new(scene.Path).Polygon(xy...)
}

// Layout lays a radar chart out as a scene, as Pele's renderRadar draws it: the chart at the
// width and height the chart's config gives, narrower when the host is, with its legend at
// its right, or below it when the chart does not fit the host's width with the legend beside
// it.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	natural := option(config, "width", 320, 1, 4000)
	return common.FitWidth(o, natural, 160, func(width float64, tight bool) *scene.Scene {
		return draw(m, config, o, width, (option(config, "height", 320, 1, 4000)*width)/natural, tight)
	})
}

// draw lays the chart out at a width and a height, as Pele's draw does: about its center,
// which is then moved to where the margins, the labels, the legend and the title leave it.
// tight is set when the chart does not fit the width it has with its legend beside it.
func draw(m *Model, config diagram.Config, o *layout.Options, width, height float64, tight bool) *scene.Scene {
	pad := o.Pad()
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	measure := o.Measure()
	label := func(raw string) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: small, MaxWidth: 4000, Icons: o.Icons})
	}

	axisScale := option(config, "axisScaleFactor", 1, 0, 10)
	labelFactor := option(config, "axisLabelFactor", 1.05, 0, 10)
	tension := option(config, "curveTension", 0.17, 0, 1)
	radius := min(width, height) / 2

	axes, curves := m.Axes, m.Curves
	ticks, graticule, lo := m.Options.Ticks, m.Options.Graticule, m.Options.Min
	count := len(axes)
	unit := make([]point, count)
	for i := range axes {
		angle := (2*math.Pi*float64(i))/float64(count) - math.Pi/2
		unit[i] = point{jsmath.Cos(angle), jsmath.Sin(angle)}
	}

	hi := m.Options.Max
	if math.IsNaN(hi) {
		hi = math.Inf(-1)
		for _, curve := range curves {
			for _, value := range curve.Entries {
				if value > hi {
					hi = value
				}
			}
		}
	}

	// The chart is laid out about its center. Margins reserve room; labels and the legend
	// widen it.
	x0 := -width/2 - option(config, "marginLeft", 16, 0, 2000)
	x1 := width/2 + option(config, "marginRight", 16, 0, 2000)
	y0 := -height/2 - option(config, "marginTop", 16, 0, 2000)
	y1 := height/2 + option(config, "marginBottom", 16, 0, 2000)

	var items []scene.Item
	if ticks > 0 {
		grid := new(scene.Path)
		for i := 0.0; i < ticks; i++ {
			r := min(radius, (radius*(i+1))/ticks)
			switch {
			case graticule == "circle":
				grid.Ellipse(0, 0, r, r)
			case count > 0:
				// A polygon of no axes has no points, and draws nothing.
				ring := make([]float64, 0, 2*count)
				for _, u := range unit {
					ring = append(ring, u.x*r, u.y*r)
				}
				grid.Polygon(ring...)
			}
		}
		if len(grid.Ops) > 0 {
			items = append(items, scene.Item{
				Kind: scene.Mark, Role: RoleGraticule, Shape: scene.Shape(graticule), Box: grid.Bounds(),
				Parts: []scene.Part{{Path: grid, Paint: scene.StrokeOnly}},
				Style: style.Style{Stroke: style.SurfaceAlt},
			})
		}
	}

	var spokes *scene.Path
	var labels []scene.Item
	for i, u := range unit {
		reach := radius * axisScale
		if spokes == nil {
			spokes = new(scene.Path)
		}
		spokes.MoveTo(0, 0).LineTo(u.x*reach, u.y*reach)
		l := label(axes[i].Label)
		at := float64(radius*labelFactor) + 6
		// Each label hangs off its axis on the side away from the center.
		var dx, dy float64
		switch {
		case u.x > 0.01:
			dx = l.Width / 2
		case u.x < -0.01:
			dx = -l.Width / 2
		}
		switch {
		case u.y > 0.01:
			dy = l.Height / 2
		case u.y < -0.01:
			dy = -l.Height / 2
		}
		cx := float64(u.x*at) + dx
		cy := float64(u.y*at) + dy
		if len(l.Lines) > 0 {
			labels = append(labels, scene.Item{
				Kind: scene.Text, Role: RoleAxisLabel, ID: axes[i].Name, Label: &l,
				At: scene.Point{X: cx, Y: cy}, Style: style.Style{Text: style.Fg},
			})
		}
		x0 = min(x0, cx-l.Width/2)
		x1 = max(x1, cx+l.Width/2)
		y0 = min(y0, cy-l.Height/2)
		y1 = max(y1, cy+l.Height/2)
	}
	if spokes != nil {
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: RoleAxes, Box: spokes.Bounds(),
			Parts: []scene.Part{{Path: spokes, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.Border},
		})
	}

	for index, curve := range curves {
		// Mermaid leaves out a curve that does not have a value for every axis.
		if len(curve.Entries) != count {
			continue
		}
		points := make([]point, count)
		for i, value := range curve.Entries {
			r := relativeRadius(value, lo, hi, radius)
			if math.IsNaN(r) || math.IsInf(r, 0) {
				r = 0
			}
			points[i] = point{unit[i].x * r, unit[i].y * r}
		}
		var path *scene.Path
		if graticule == "circle" {
			path = closedRoundCurve(points, tension)
		} else {
			path = polygon(points)
		}
		color := style.Series(index)
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: RoleCurve, ID: curve.Name, Box: path.Bounds(),
			Parts: []scene.Part{{Path: path}},
			Style: style.Style{Fill: color, FillOpacity: style.N(0.2), Stroke: color, StrokeWidth: style.N(1.5), Join: style.JoinRound},
		})
	}
	items = append(items, labels...)

	var legend []text.Label
	if m.Options.ShowLegend {
		for _, curve := range curves {
			legend = append(legend, label(curve.Label))
		}
	}
	legendWidth := 0.0
	for _, l := range legend {
		legendWidth = max(legendWidth, l.Width)
	}
	legendHeight := float64(len(legend) * row)
	legendX := x1 + gap
	legendTop := -legendHeight / 2
	if tight {
		legendX = -(swatch + 8 + legendWidth) / 2
		legendTop = y1 + gap/2
	}
	if len(legend) > 0 && tight {
		x0 = min(x0, legendX)
		x1 = max(x1, -legendX)
		y1 = legendTop + legendHeight
	} else if len(legend) > 0 {
		x1 = legendX + swatch + 8 + legendWidth
		y0 = min(y0, -legendHeight/2)
		y1 = max(y1, legendHeight/2)
	}
	for index := range legend {
		l := &legend[index]
		y := legendTop + float64(index*row) + row/2
		it := scene.Item{
			Kind: scene.Node, Role: RoleLegend, Shape: "rect", ID: curves[index].Name,
			Box:   scene.Rect{X: legendX, Y: y - row/2, W: swatch + 8 + l.Width, H: row},
			Parts: []scene.Part{{Box: scene.Rect{X: legendX, Y: y - swatch/2, W: swatch, H: swatch}, Rounded: true}},
			At:    scene.Point{X: legendX + swatch + 8 + l.Width/2, Y: y},
			Style: style.Style{Fill: style.Series(index), FillOpacity: style.N(0.4), Stroke: style.Series(index), Text: style.Fg, Radius: style.N(2)},
		}
		if len(l.Lines) > 0 {
			it.Label = l
		}
		items = append(items, it)
	}

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: common.TitleRoom(o), Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 8
	}
	bodyWidth := max(x1-x0, title.Width)
	totalWidth := math.Ceil(bodyWidth + 2*pad)
	totalHeight := math.Ceil(y1 - y0 + titleHeight + 2*pad)
	ox := pad + (bodyWidth-(x1-x0))/2 - x0
	oy := pad + titleHeight - y0
	for i := range items {
		items[i].Move(ox, oy)
	}

	if len(title.Lines) > 0 {
		items = append([]scene.Item{{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: totalWidth / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		}}, items...)
	}
	return &scene.Scene{
		Type:     "radar",
		Width:    totalWidth,
		Height:   totalHeight,
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
		Items:    items,
	}
}
