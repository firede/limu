package xychart

import (
	"math"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of Pele's XY chart render.
const (
	tick      = 4
	swatch    = 12
	row       = 26
	legendGap = 24
	limit     = 1e300
	pad       = common.Padding
)

// mark is a labelled place along an axis, as a fraction of its length: Pele's Mark.
type mark struct {
	at    float64
	label text.Label
}

// record is a section of the config that is an object, and nothing otherwise, as Pele's
// record reads it.
func record(value any) diagram.Config {
	c, _ := value.(diagram.Config)
	return c
}

// on is a setting that is true or false, or fallback when it is anything else, as Pele's on
// reads it.
func on(config diagram.Config, key string, fallback bool) bool {
	if b, ok := config[key].(bool); ok {
		return b
	}
	return fallback
}

// dimension is a size the config gives, kept from 100 to 4000 and rounded, or fallback for
// anything but a number above zero, as Pele's dimension reads it.
func dimension(value any, fallback float64) float64 {
	if v, ok := value.(float64); ok && v > 0 {
		return jsmath.Round(min(max(v, 100), 4000))
	}
	return fallback
}

// finite is a value kept within ±1e300, and false for a value that is not a finite number,
// as Pele's finite gives undefined for it.
func finite(value float64) (float64, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return min(max(value, -limit), limit), true
}

// widest is the width of the widest label of marks, as Pele's widest gives it.
func widest(marks []mark) float64 {
	width := 0.0
	for _, m := range marks {
		width = max(width, m.label.Width)
	}
	return width
}

// plain reports whether a label is one line of one span of plain text, which Pele's aligned
// writes as one text element anchored where it is placed.
func plain(l *text.Label) bool {
	return len(l.Lines) == 1 && len(l.Lines[0]) == 1 && l.Lines[0][0].Icon == "" && l.Lines[0][0].Face == 0
}

// aligned is a label with its left edge, center or right edge at x and its middle at y, as
// Pele's aligned writes it: a plain one-line label is anchored at x, so that it keeps its
// place when the text is wider or narrower than measured, and any other is centered on where
// its measured edge puts its middle. An empty label is nothing, as Pele draws nothing for it.
func aligned(role scene.Role, label text.Label, x, y float64, align scene.Align, paint style.Paint) []scene.Item {
	if label.Empty() {
		return nil
	}
	it := scene.Item{Kind: scene.Text, Role: role, Label: &label, Style: style.Style{Text: paint}}
	if plain(&label) {
		it.At, it.Align = scene.Point{X: x, Y: y}, align
		return []scene.Item{it}
	}
	switch align {
	case scene.Start:
		x += label.Width / 2
	case scene.End:
		x += -label.Width / 2
	}
	it.At = scene.Point{X: x, Y: y}
	return []scene.Item{it}
}

// union is the smallest box that holds a and b.
func union(a, b scene.Rect) scene.Rect {
	x0, y0 := min(a.X, b.X), min(a.Y, b.Y)
	x1, y1 := max(a.X+a.W, b.X+b.W), max(a.Y+a.H, b.Y+b.H)
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// Layout lays an XY chart out as a scene, as Pele's renderXyChart draws it: the plot at the
// width and height the chart's config gives, narrower when the host is, with its axes, the
// bars and lines of each series over a grid, and a legend at its right. A chart with no
// series is an error, as in Mermaid.
func Layout(m *Model, config diagram.Config, o *layout.Options) (*scene.Scene, error) {
	if o == nil {
		o = &layout.Options{}
	}
	if len(m.Plots) == 0 {
		return nil, diagram.NewError("No Plot to render, please provide a plot with some data", diagram.Semantic, "xychart")
	}
	chart := record(config["xyChart"])
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	tiny := jsmath.Round(size * 0.75)
	measure := o.Measure()
	natural := dimension(chart["width"], 700)
	width := common.CapWidth(o, natural, 240, 0)
	// A narrower chart is not as much shorter, so that the plot keeps room for its marks.
	height := jsmath.Round(dimension(chart["height"], 500) * max(0.6, width/natural))
	orientation := chart["chartOrientation"]
	if m.Orientation != "" {
		orientation = m.Orientation
	}
	horizontal := orientation == "horizontal"
	// The value of a bar is always written past its end, which is what
	// showDataLabelOutsideBar asks for in Mermaid. Inside, whether it fits changes from bar to
	// bar and with the width of the chart.
	showValues := on(chart, "showDataLabel", false)
	x := m.XAxis
	band := x.Type == "band"
	plots := m.Plots
	layText := func(raw string, fontSize, max float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: fontSize, MaxWidth: max, Face: face, Bounded: true, Icons: o.Icons})
	}
	label := func(raw string, fontSize, max float64) text.Label { return layText(raw, fontSize, max, 0) }
	lineHeight := label("0", small, 4000).Height
	tinyHeight := label("0", tiny, 4000).Height

	titleText := ""
	if on(chart, "showTitle", true) {
		titleText = m.Title
	}
	title := layText(titleText, size, width-2*pad, text.Bold)
	titleSpace := 0.0
	if title.Height > 0 {
		titleSpace = title.Height + 12
	}

	type entry struct {
		index int
		label text.Label
	}
	var legend []entry
	legendWidth := 0.0
	if on(chart, "showLegend", true) {
		for index, plot := range plots {
			if plot.Title == "" {
				continue
			}
			l := label(plot.Title, small, width*0.3)
			legend = append(legend, entry{index, l})
			legendWidth = max(legendWidth, l.Width+swatch+8)
		}
	}
	legendSpace := 0.0
	if len(legend) > 0 {
		legendSpace = legendWidth + legendGap
	}

	// The value range: as written, or taken from the data, where bars stand on zero and the
	// range grows to whole steps.
	bars := 0
	for _, plot := range plots {
		if plot.Type == "bar" {
			bars++
		}
	}
	lo, hi := m.YAxis.Min, m.YAxis.Max
	exact := m.HasSetYAxis && !math.IsInf(lo, 0) && !math.IsNaN(lo) && !math.IsInf(hi, 0) && !math.IsNaN(hi) && lo != hi
	if exact {
		lo, _ = finite(lo)
		hi, _ = finite(hi)
	} else {
		lo, hi = math.Inf(1), math.Inf(-1)
		for _, plot := range plots {
			for _, d := range plot.Data {
				value, ok := finite(d.Value)
				if !ok {
					continue
				}
				if value < lo {
					lo = value
				}
				if value > hi {
					hi = value
				}
			}
		}
		if lo > hi {
			lo, hi = 0, 0
		}
		if bars > 0 {
			lo = min(lo, 0)
			hi = max(hi, 0)
		}
		if lo == hi {
			if lo == 0 {
				hi = 1
			} else {
				lo -= math.Abs(lo) / 2
				hi += math.Abs(hi) / 2
			}
		}
	}

	valueMarks := func(length, spacing float64) []mark {
		step := tickStep(math.Abs(hi-lo), max(2, min(10, math.Floor(length/spacing))))
		if !exact {
			lo = tickValue(step, math.Floor(tickIndex(step, lo)+1e-9))
			hi = tickValue(step, math.Ceil(tickIndex(step, hi)-1e-9))
		}
		var marks []mark
		for _, value := range ticksBetween(min(lo, hi), max(lo, hi), step) {
			marks = append(marks, mark{(value - lo) / (hi - lo), label(chars.NumberString(value), small, 4000)})
		}
		return marks
	}

	// Places along the category axis are fractions of its length. Categories and bars sit in
	// the middle of equal slots; the points of a line on a numeric axis run from end to end.
	count := 1
	if band {
		count = max(len(x.Categories), 1)
	} else {
		for _, plot := range plots {
			count = max(count, len(plot.Data))
		}
	}
	slot := 1 / float64(count)
	inset := 0.0
	if band || bars > 0 {
		inset = slot / 2
	}
	x0, has0 := 0.0, false
	x1, has1 := 0.0, false
	if !band {
		x0, has0 = finite(x.Min)
		x1, has1 = finite(x.Max)
	}
	ranged := has0 && has1 && x0 != x1
	place := func(i, n int) float64 {
		if band {
			return (float64(i) + 0.5) * slot
		}
		at := 0.5
		switch {
		case n > 1:
			at = float64(i) / float64(n-1)
		case ranged:
			at = 0
		}
		return inset + float64(at*(1-float64(2*inset)))
	}

	categoryConf := record(chart["xAxis"])
	valueConf := record(chart["yAxis"])
	leftConf, bottomConf := valueConf, categoryConf
	if horizontal {
		leftConf, bottomConf = categoryConf, valueConf
	}
	leftText, bottomText := "", ""
	if on(leftConf, "showTitle", true) {
		leftText = m.YAxis.Title
		if horizontal {
			leftText = x.Title
		}
	}
	if on(bottomConf, "showTitle", true) {
		bottomText = x.Title
		if horizontal {
			bottomText = m.YAxis.Title
		}
	}
	leftTitle := label(leftText, small, height*0.8)
	bottomTitle := label(bottomText, small, width*0.8)
	leftTick, bottomTick := 0.0, 0.0
	if on(leftConf, "showTick", true) {
		leftTick = tick
	}
	if on(bottomConf, "showTick", true) {
		bottomTick = tick
	}
	leftLabels := on(leftConf, "showLabel", true)
	bottomLabels := on(bottomConf, "showLabel", true)

	wrapAt := width * 0.3
	if !horizontal {
		wrapAt = max(48, float64((width-2*pad-64-legendSpace)*slot)-8)
	}
	var categories []text.Label
	if band && on(categoryConf, "showLabel", true) {
		for _, category := range x.Categories {
			categories = append(categories, label(category, small, wrapAt))
		}
	}
	categoryWidth, categoryHeight := 0.0, 0.0
	for _, l := range categories {
		categoryWidth = max(categoryWidth, l.Width)
		categoryHeight = max(categoryHeight, l.Height)
	}

	// The labelled places of the category axis. Labels that would run into each other are
	// thinned out.
	categoryMarks := func(length float64) []mark {
		var marks []mark
		switch {
		case band:
			need := 8.0
			if len(categories) > 0 {
				need = categoryWidth + 12
				if horizontal {
					need = categoryHeight
				}
			}
			every := max(1, math.Ceil(need/(length*slot)))
			for i := 0.0; i < float64(len(x.Categories)); i += every {
				var l text.Label
				if int(i) < len(categories) {
					l = categories[int(i)]
				}
				marks = append(marks, mark{(i + 0.5) * slot, l})
			}
		case ranged:
			spacing := 80.0
			if horizontal {
				spacing = 32
			}
			step := tickStep(math.Abs(x1-x0), max(2, min(10, math.Floor(length/spacing))))
			for _, value := range ticksBetween(min(x0, x1), max(x0, x1), step) {
				marks = append(marks, mark{inset + float64(((value-x0)/(x1-x0))*(1-float64(2*inset))), label(chars.NumberString(value), small, 4000)})
			}
		case has0:
			marks = append(marks, mark{0.5, label(chars.NumberString(x0), small, 4000)})
		}
		return marks
	}

	// Room past the end of the value axis for what is written beside a point or a bar.
	pointLabelWidth, valueWidth := 0.0, 0.0
	for _, plot := range plots {
		for _, l := range plot.PointLabels {
			pointLabelWidth = max(pointLabelWidth, measure.Width(l, tiny, 0))
		}
		if showValues && horizontal && plot.Type == "bar" {
			for _, d := range plot.Data {
				s := ""
				if v, ok := finite(d.Value); ok {
					s = chars.NumberString(v)
				}
				valueWidth = max(valueWidth, measure.Width(s, tiny, 0))
			}
		}
	}

	bottomLabelHeight := 0.0
	if bottomLabels {
		bottomLabelHeight = categoryHeight
		if horizontal || !band {
			bottomLabelHeight = lineHeight
		}
	}
	bottomSpace := bottomTick
	if bottomLabelHeight > 0 {
		bottomSpace += bottomLabelHeight + 4
	}
	if bottomTitle.Height > 0 {
		bottomSpace += bottomTitle.Height + 4
	}
	headroom := 0.0
	switch {
	case horizontal:
	case pointLabelWidth > 0 || showValues:
		headroom = tinyHeight + 4
	case leftLabels:
		headroom = lineHeight / 2
	}
	top := pad + titleSpace + headroom
	plotHeight := max(1, height-pad-bottomSpace-top)
	bottom := top + plotHeight

	var leftMarks []mark
	if horizontal {
		leftMarks = categoryMarks(plotHeight)
	} else {
		leftMarks = valueMarks(plotHeight, 56)
	}
	leftLabelWidth := 0.0
	if leftLabels {
		leftLabelWidth = widest(leftMarks)
	}
	left := float64(pad)
	if leftTitle.Height > 0 {
		left += leftTitle.Height + 6
	}
	if leftLabelWidth > 0 {
		left += leftLabelWidth + 6
	}
	left += leftTick
	besidePoints, besideValues := 0.0, 0.0
	if pointLabelWidth > 0 {
		besidePoints = pointLabelWidth + 12
	}
	if valueWidth > 0 {
		besideValues = valueWidth + 8
	}
	beside := max(besidePoints, besideValues)
	reach := 0.0
	if horizontal {
		reach = min(width*0.25, beside)
	}
	plotWidth := max(1, width-pad-left-legendSpace-reach)
	var bottomMarks []mark
	if horizontal {
		bottomMarks = valueMarks(plotWidth, 80)
	} else {
		bottomMarks = categoryMarks(plotWidth)
	}
	if bottomLabels && legendSpace+reach == 0 {
		over := 0.0
		for _, mk := range bottomMarks {
			over = max(over, float64(mk.at*plotWidth)+mk.label.Width/2-plotWidth)
		}
		plotWidth = max(1, plotWidth-over)
	}
	right := left + plotWidth

	px := func(c, v float64) float64 {
		if horizontal {
			return left + float64(v*plotWidth)
		}
		return left + float64(c*plotWidth)
	}
	py := func(c, v float64) float64 {
		if horizontal {
			return top + float64(c*plotHeight)
		}
		return bottom - float64(v*plotHeight)
	}
	scale := func(value float64) float64 { return (value - lo) / (hi - lo) }
	clamp := func(v float64) float64 { return min(max(v, 0), 1) }

	// A line across the plot at a place on the value axis.
	rule := func(p *scene.Path, v float64) {
		if horizontal {
			p.MoveTo(px(0, v), top).LineTo(px(0, v), bottom)
		} else {
			p.MoveTo(left, py(0, v)).LineTo(right, py(0, v))
		}
	}

	grid := new(scene.Path)
	gridMarks := leftMarks
	if horizontal {
		gridMarks = bottomMarks
	}
	for _, mk := range gridMarks {
		rule(grid, mk.at)
	}

	axis := new(scene.Path)
	var labels []scene.Item
	if on(leftConf, "showAxisLine", true) {
		axis.MoveTo(left, top).LineTo(left, bottom)
	}
	if on(bottomConf, "showAxisLine", true) {
		axis.MoveTo(left, bottom).LineTo(right, bottom)
	}
	for _, mk := range leftMarks {
		y := bottom - float64(mk.at*plotHeight)
		if horizontal {
			y = top + float64(mk.at*plotHeight)
		}
		if leftTick != 0 {
			axis.MoveTo(left, y).LineTo(left+-leftTick, y)
		}
		if leftLabels {
			labels = append(labels, aligned("xychart.axis-label", mk.label, left-leftTick-6, y, scene.End, style.Muted)...)
		}
	}
	for _, mk := range bottomMarks {
		at := left + float64(mk.at*plotWidth)
		if bottomTick != 0 {
			axis.MoveTo(at, bottom).LineTo(at, bottom+bottomTick)
		}
		if bottomLabels {
			labels = append(labels, aligned("xychart.axis-label", mk.label, at, bottom+bottomTick+4+mk.label.Height/2, scene.Middle, style.Muted)...)
		}
	}

	span := plotWidth * slot
	if horizontal {
		span = plotHeight * slot
	}
	group := span * 0.7
	thickness := group / float64(max(bars, 1))
	gap := 0.0
	if bars > 1 {
		gap = min(2, thickness*0.2)
	}
	base := clamp(scale(0))
	// Bars that go both ways get a line to stand on.
	if bars > 0 && base > 0 && base < 1 {
		rule(axis, base)
	}
	// Lines are drawn over the bars wherever they were declared, as a bar would hide part of
	// one under it. A series keeps the color of its place in the source.
	var series, lines, outside, pointLabels []scene.Item
	bar := 0
	for index, plot := range plots {
		color := style.Series(index)
		// A series declared before the categories can hold more values than there are
		// categories.
		n := len(plot.Data)
		if band {
			n = min(n, count)
		}
		if plot.Type == "bar" {
			offset := float64((float64(bar)-float64(bars-1)/2)*thickness) - (thickness-gap)/2
			bar++
			var parts []scene.Part
			var bounds scene.Rect
			for i := 0; i < n; i++ {
				value, ok := finite(plot.Data[i].Value)
				if !ok {
					continue
				}
				c := place(i, n)
				v := clamp(scale(value))
				from := min(base, v)
				to := max(base, v)
				var bx, by, bw, bh float64
				if horizontal {
					bx, by = px(c, from), py(c, 0)+offset
					bw, bh = float64((to-from)*plotWidth), thickness-gap
				} else {
					bx, by = px(c, 0)+offset, py(c, to)
					bw, bh = thickness-gap, float64((to-from)*plotHeight)
				}
				r := scene.Rect{X: bx, Y: by, W: bw, H: bh}
				if len(parts) == 0 {
					bounds = r
				} else {
					bounds = union(bounds, r)
				}
				parts = append(parts, scene.Part{Box: r, Paint: scene.FillOnly})
				if !showValues {
					continue
				}
				s := chars.NumberString(value)
				w := measure.Width(s, tiny, 0)
				// Towards the end of the bar: 1 when it runs up or right from the baseline,
				// -1 the other way.
				out := -1.0
				if v >= base {
					out = 1
				}
				tx := bx + bw/2
				ty := by + bh/2
				align := scene.Middle
				if horizontal {
					if bh < tiny {
						continue
					}
					tx = bx
					align = scene.End
					if out > 0 {
						tx = bx + bw
						align = scene.Start
					}
					tx += float64(out * 6)
				} else {
					if w+2 > span/float64(bars) {
						continue
					}
					ty = by + bh
					if out > 0 {
						ty = by
					}
					ty -= float64(out * (tinyHeight/2 + 2))
				}
				l := common.Line(s, w, tiny, 0)
				outside = append(outside, scene.Item{
					Kind: scene.Text, Role: "xychart.data-label", Label: l, Align: align,
					At: scene.Point{X: tx, Y: ty}, Style: style.Style{Text: style.Fg},
				})
			}
			series = append(series, scene.Item{
				Kind: scene.Mark, Role: "xychart.bars", ID: plot.Title, Box: bounds, Parts: parts,
				Style: style.Style{Fill: color},
			})
			continue
		}

		// A line is cut where it leaves the value range, so it stays inside the plot.
		d := new(scene.Path)
		var dots []scene.Part
		pen := false
		pc := 0.0
		pv := math.NaN()
		for i := 0; i < n; i++ {
			value, ok := finite(plot.Data[i].Value)
			if !ok {
				pv = math.NaN()
				continue
			}
			c := place(i, n)
			v := scale(value)
			visible := v >= 0 && v <= 1
			if pv != pv {
				if visible {
					d.MoveTo(px(c, v), py(c, v))
				}
				pen = visible
			} else {
				dv := v - pv
				t0, t1 := 0.0, 1.0
				if dv != 0 {
					a := -pv / dv
					b := (1 - pv) / dv
					t0 = max(t0, min(a, b))
					t1 = min(t1, max(a, b))
				} else if !visible {
					t0 = 2
				}
				if t0 <= t1 {
					c0 := pc + float64((c-pc)*t0)
					v0 := pv + float64(dv*t0)
					c1 := pc + float64((c-pc)*t1)
					v1 := pv + float64(dv*t1)
					if !pen || t0 > 0 {
						d.MoveTo(px(c0, v0), py(c0, v0))
					}
					d.LineTo(px(c1, v1), py(c1, v1))
				}
				pen = visible
			}
			pc = c
			pv = v
			if !visible {
				continue
			}
			cx := px(c, v)
			cy := py(c, v)
			if span >= 12 {
				dots = append(dots, scene.Part{Path: new(scene.Path).Ellipse(cx, cy, 3, 3), Paint: scene.FillOnly})
			}
			raw := ""
			if i < len(plot.PointLabels) {
				raw = plot.PointLabels[i]
			}
			l := label(raw, tiny, 4000)
			if l.Height > 0 {
				half := l.Width / 2
				if horizontal {
					pointLabels = append(pointLabels, aligned("xychart.point-label", l, cx+10, cy, scene.Start, style.Fg)...)
				} else {
					pointLabels = append(pointLabels, aligned("xychart.point-label", l, max(min(cx, right-half), left+half), cy-6-l.Height/2, scene.Middle, style.Fg)...)
				}
			}
		}
		var parts []scene.Part
		if len(d.Ops) > 0 {
			parts = append(parts, scene.Part{Path: d, Paint: scene.StrokeOnly})
		}
		parts = append(parts, dots...)
		it := scene.Item{
			Kind: scene.Mark, Role: "xychart.line", ID: plot.Title, Parts: parts,
			Style: style.Style{Fill: color, Stroke: color, StrokeWidth: style.N(2), Cap: style.CapRound, Join: style.JoinRound},
		}
		for k, part := range parts {
			if k == 0 {
				it.Box = part.Path.Bounds()
			} else {
				it.Box = union(it.Box, part.Path.Bounds())
			}
		}
		lines = append(lines, it)
	}

	var titles []scene.Item
	if leftTitle.Height > 0 {
		l := leftTitle
		titles = append(titles, scene.Item{
			Kind: scene.Text, Role: "xychart.axis-title", Label: &l, Rotate: -90,
			At: scene.Point{X: pad + leftTitle.Height/2, Y: top + plotHeight/2}, Style: style.Style{Text: style.Fg},
		})
	}
	if !bottomTitle.Empty() {
		l := bottomTitle
		titles = append(titles, scene.Item{
			Kind: scene.Text, Role: "xychart.axis-title", Label: &l,
			At: scene.Point{X: left + plotWidth/2, Y: height - pad - bottomTitle.Height/2}, Style: style.Style{Text: style.Fg},
		})
	}

	var legendItems []scene.Item
	legendHeight := 0.0
	for _, e := range legend {
		legendHeight += max(row, e.label.Height+6)
	}
	ly := top + max(0, (plotHeight-legendHeight)/2)
	lx := right + reach + legendGap
	for _, e := range legend {
		h := max(row, e.label.Height+6)
		if ly+h > height {
			break
		}
		y := ly + h/2
		color := style.Series(e.index)
		it := scene.Item{
			Kind: scene.Node, Role: "xychart.legend", ID: plots[e.index].Title,
			Box: scene.Rect{X: lx, Y: ly, W: swatch + 8 + e.label.Width, H: h},
			At:  scene.Point{X: lx + swatch + 8 + e.label.Width/2, Y: y},
		}
		if plots[e.index].Type == "bar" {
			it.Shape = "rect"
			it.Parts = []scene.Part{{Box: scene.Rect{X: lx, Y: y - swatch/2, W: swatch, H: swatch}, Rounded: true, Paint: scene.FillOnly}}
			it.Style = style.Style{Fill: color, Radius: style.N(2)}
		} else {
			it.Shape = "line"
			it.Parts = []scene.Part{{Path: new(scene.Path).MoveTo(lx, y).LineTo(lx+swatch, y), Paint: scene.StrokeOnly}}
			it.Style = style.Style{Stroke: color, StrokeWidth: style.N(2), Cap: style.CapRound}
		}
		if !e.label.Empty() {
			l := e.label
			it.Label = &l
		}
		legendItems = append(legendItems, it)
		ly += h
	}

	sc := &scene.Scene{
		Type:     "xychart",
		Width:    width,
		Height:   height,
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: "xychart.title", Label: &title,
			At: scene.Point{X: width / 2, Y: pad + title.Height/2}, Style: style.Style{Text: style.Fg},
		})
	}
	if len(grid.Ops) > 0 {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Mark, Role: "xychart.grid", Box: grid.Bounds(),
			Parts: []scene.Part{{Path: grid, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.SurfaceAlt},
		})
	}
	sc.Items = append(sc.Items, series...)
	sc.Items = append(sc.Items, lines...)
	if len(axis.Ops) > 0 {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Mark, Role: "xychart.axis", Box: axis.Bounds(),
			Parts: []scene.Part{{Path: axis, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.Border},
		})
	}
	sc.Items = append(sc.Items, labels...)
	sc.Items = append(sc.Items, titles...)
	sc.Items = append(sc.Items, outside...)
	sc.Items = append(sc.Items, pointLabels...)
	sc.Items = append(sc.Items, legendItems...)
	return sc, nil
}
