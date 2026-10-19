package timeline

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

// The measures of a timeline, as Pele's render.ts has them.
const (
	padX        = 12
	padY        = 8
	minText     = 152
	maxText     = 216
	eventTextTD = 264
	gap         = 20
	axis        = 20
	spine       = 14
	eventGap    = 8
	sectionGap  = 8
	tail        = 12
	arrow       = scene.MarkerSize
	pad         = common.Padding
)

// group is a section and the periods in it, Pele's Group: section is -1 for the periods
// that come before the first section.
type group struct {
	section int
	periods []int
}

// Layout lays out a timeline diagram as a scene, as Pele's renderTimeline draws it: across,
// unless the text asks for it to run down or it runs down to fit the host's width.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if m.Direction == "TD" {
		return lay(m, config, o, true)
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return lay(m, config, o, down) })
}

// roundedBox is a box whose corners take the theme's radius, as Pele's box draws it.
func roundedBox(x, y, w, h float64, role scene.Role, kind scene.Kind, s style.Style) scene.Item {
	r := scene.Rect{X: x, Y: y, W: w, H: h}
	return scene.Item{
		Kind: kind, Role: role, Shape: "rounded", Box: r,
		Parts: []scene.Part{{Box: r, Rounded: true, Paint: scene.FillOnly}},
		Style: s,
	}
}

// lay lays out a timeline, across or down, as Pele's draw does.
func lay(m *Model, config diagram.Config, o *layout.Options, vertical bool) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	measure := o.Measure()
	multicolor := true
	if section, ok := config["timeline"].(diagram.Config); ok {
		multicolor = section["disableMulticolor"] != true
	}
	periods := m.Periods
	hasSections := len(m.Sections) > 0

	// Mermaid draws nothing for periods that come before the first section; here they lead,
	// in a neutral color.
	groups := make([]*group, len(m.Sections))
	for k := range groups {
		groups[k] = &group{section: k}
	}
	leading := &group{section: -1}
	for i, p := range periods {
		if p.SectionIndex >= 0 && p.SectionIndex < len(groups) {
			groups[p.SectionIndex].periods = append(groups[p.SectionIndex].periods, i)
		} else {
			leading.periods = append(leading.periods, i)
		}
	}
	if len(leading.periods) > 0 {
		groups = append([]*group{leading}, groups...)
	}

	// Columns widen, up to a point, for a word that would not fit; past that the word is
	// cut. Running down, the two columns share the width the host has.
	room := math.Inf(1)
	if vertical && o.MaxWidth > 0 {
		room = o.MaxWidth - 2*pad
	}
	const taken = axis + spine*2 + 4*padX
	periodText := min(minText, max(80, jsmath.Round((room-taken)*0.38)))
	eventText := min(eventTextTD, max(110, room-taken-periodText))
	textW := periodText
	eventTextW := func() float64 {
		if vertical {
			return eventText
		}
		return textW
	}
	layLabels := func(lay func(raw string, measure text.Measurer, size, max float64, base text.Face) text.Label) ([]text.Label, [][]text.Label) {
		periodLabels := make([]text.Label, len(periods))
		eventLabels := make([][]text.Label, len(periods))
		for i, p := range periods {
			periodLabels[i] = lay(p.Text, measure, size, textW, 0)
		}
		for i, p := range periods {
			eventLabels[i] = make([]text.Label, len(p.Events))
			for k, e := range p.Events {
				eventLabels[i][k] = lay(e, measure, small, eventTextW(), 0)
			}
		}
		return periodLabels, eventLabels
	}
	periodLabels, eventLabels := layLabels(common.PlainLabel)
	widest := 0.0
	for _, l := range periodLabels {
		widest = max(widest, l.Width)
	}
	if !vertical {
		for _, labels := range eventLabels {
			for _, l := range labels {
				widest = max(widest, l.Width)
			}
		}
	}
	overflow := widest > textW
	if vertical {
		for _, labels := range eventLabels {
			for _, l := range labels {
				overflow = overflow || l.Width > eventText
			}
		}
	}
	if overflow {
		limit := periodText
		if math.IsInf(room, 1) {
			limit = maxText
		}
		textW = min(max(math.Ceil(widest), periodText), limit)
		periodLabels, eventLabels = layLabels(common.FitLabel)
	}

	lineHeight := jsmath.Round(size * 1.5)
	eventLineHeight := jsmath.Round(small * 1.5)
	colW := textW + 2*padX
	eventW := eventTextW() + 2*padX
	chipHeight := func(l *text.Label) float64 { return max(l.Height, lineHeight) + 2*padY }
	eventHeight := func(l *text.Label) float64 { return max(l.Height, eventLineHeight) + 2*padY }
	colorOf := func(g *group, period int) style.Paint {
		switch {
		case g.section >= 0:
			return style.Series(g.section)
		case hasSections:
			return style.Muted
		case multicolor:
			return style.Series(period)
		}
		return style.Series(0)
	}

	hasEvents := false
	for _, labels := range eventLabels {
		if len(labels) > 0 {
			hasEvents = true
		}
	}
	slots := 0
	for _, g := range groups {
		slots += max(len(g.periods), 1)
	}

	pitch := colW + gap
	drawn := len(periods) > 0
	var bodyW float64
	switch {
	case vertical:
		var beside float64
		if hasEvents {
			beside = axis + spine*2 + eventW
		} else if drawn {
			beside = axis + tail
		}
		bodyW = colW + beside
	case slots > 0:
		var end float64
		if drawn {
			end = tail + arrow
		}
		bodyW = float64(float64(slots)*pitch) - gap + end
	}
	title := common.FitLabel(m.Title, measure, size, max(bodyW, 320), text.Bold)
	titleH := 0.0
	if title.Height > 0 {
		titleH = title.Height + 16
	}
	width := max(bodyW, title.Width)
	left := pad + (width-bodyW)/2
	top := pad + titleH

	sectionLabel := func(g *group, w float64) text.Label {
		return common.FitLabel(m.Sections[g.section], measure, size, w-2*padX, text.Bold)
	}
	var items []scene.Item
	header := func(g *group, label text.Label, x, y, w, h float64) {
		it := roundedBox(x, y, w, h, "timeline.section", scene.Group, style.Style{Fill: style.Series(g.section), Text: style.Bg})
		it.ID = chars.Trim(m.Sections[g.section])
		it.Label = &label
		it.At = scene.Point{X: x + w/2, Y: y + h/2}
		items = append(items, it)
	}
	period := func(i int, color style.Paint, x, y, h float64) {
		it := roundedBox(x, y, colW, h, "timeline.period", scene.Node, style.Style{Fill: color, Text: style.Bg})
		it.ID = chars.Trim(periods[i].Text)
		label := periodLabels[i]
		it.Label = &label
		it.At = scene.Point{X: x + colW/2, Y: y + h/2}
		items = append(items, it)
	}
	event := func(label text.Label, color style.Paint, x, y, h float64) scene.Item {
		it := roundedBox(x, y, eventW, h, "timeline.event", scene.Node, style.Style{Fill: color, FillOpacity: style.N(0.16)})
		it.Label = &label
		it.At = scene.Point{X: x + eventW/2, Y: y + h/2}
		return it
	}
	connector := func(p *scene.Path) {
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: "timeline.connector", Box: p.Bounds(),
			Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.Border},
		})
	}
	dot := func(x, y float64, color style.Paint) {
		p := new(scene.Path).Ellipse(x, y, 4, 4)
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: "timeline.dot", Shape: "circle", Box: scene.Rect{X: x - 4, Y: y - 4, W: 8, H: 8},
			Parts: []scene.Part{{Path: p, Paint: scene.FillAndStroke}},
			Style: style.Style{Fill: color, Stroke: style.Bg, StrokeWidth: style.N(2)},
		})
	}
	// The axis, drawn before the rest: a line from (x0, y0) to a tip at (x1, y1), which it
	// stops short of for the arrowhead there, pointing along (dx, dy).
	axisLine := func(x0, y0, x1, y1, dx, dy float64) {
		p := new(scene.Path).MoveTo(x0, y0)
		if dx != 0 {
			p.LineTo(x1-arrow+1, y0)
		} else {
			p.LineTo(x0, y1-arrow+1)
		}
		axisItem := scene.Item{
			Kind: scene.Edge, Role: "timeline.axis",
			Box:   scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0},
			Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
			Route: []scene.Point{{X: x0, Y: y0}, {X: x1, Y: y1}},
			Ends:  [2]scene.Marker{{}, {Type: "arrow_point", At: scene.Point{X: x1, Y: y1}, DX: dx, DY: dy}},
			Style: style.Style{Stroke: style.Line},
		}
		items = append([]scene.Item{axisItem}, items...)
	}

	var height float64
	if vertical {
		axisX := left + colW + axis
		spineX := axisX + spine
		eventsX := spineX + spine
		y := top
		for _, g := range groups {
			if g.section >= 0 {
				label := sectionLabel(g, bodyW)
				h := chipHeight(&label)
				header(g, label, left, y, bodyW, h)
				y += h + sectionGap + 4
			}
			for _, i := range g.periods {
				color := colorOf(g, i)
				h := chipHeight(&periodLabels[i])
				cy := y + h/2
				d := new(scene.Path).MoveTo(left+colW, cy).LineTo(axisX, cy)
				var cards []scene.Item
				ey := y
				last := cy
				for k := range eventLabels[i] {
					label := &eventLabels[i][k]
					eh := eventHeight(label)
					last = ey + eh/2
					d.MoveTo(spineX, last).LineTo(eventsX, last)
					cards = append(cards, event(*label, color, eventsX, ey, eh))
					ey += eh + eventGap
				}
				if len(cards) > 0 {
					d.MoveTo(axisX, cy).LineTo(spineX, cy).LineTo(spineX, last)
				}
				period(i, color, left, y, h)
				connector(d)
				dot(axisX, cy, color)
				items = append(items, cards...)
				y = max(y+h, ey-eventGap) + gap
			}
			if len(g.periods) == 0 {
				y += gap - sectionGap - 4
			}
		}
		if len(groups) > 0 {
			y -= gap
		}
		if drawn {
			tip := y + tail + arrow
			axisLine(axisX, top, axisX, tip, 0, 1)
			y = tip
		}
		height = y + pad
	} else {
		headerH := 0.0
		sectionLabels := make([]*text.Label, len(groups))
		for k, g := range groups {
			if g.section < 0 {
				continue
			}
			label := sectionLabel(g, float64(float64(max(len(g.periods), 1))*pitch)-gap)
			headerH = max(headerH, chipHeight(&label))
			sectionLabels[k] = &label
		}
		chipH := 0.0
		for i := range periodLabels {
			chipH = max(chipH, chipHeight(&periodLabels[i]))
		}
		periodsY := top
		if hasSections {
			periodsY = top + (headerH + sectionGap)
		}
		axisY := periodsY + chipH
		if chipH > 0 {
			axisY += axis
		}
		eventsY := axisY + axis
		bottom := top + headerH
		if drawn {
			bottom = axisY + tail
		}
		slot := 0
		for k, g := range groups {
			span := max(len(g.periods), 1)
			gx := left + float64(float64(slot)*pitch)
			if label := sectionLabels[k]; label != nil {
				header(g, *label, gx, top, float64(float64(span)*pitch)-gap, headerH)
			}
			for n, i := range g.periods {
				color := colorOf(g, i)
				x := gx + float64(float64(n)*pitch)
				cx := x + colW/2
				end := axisY
				if len(eventLabels[i]) > 0 {
					end = eventsY
				}
				d := new(scene.Path).MoveTo(cx, periodsY+chipH).LineTo(cx, end)
				var cards []scene.Item
				ey := eventsY
				for k := range eventLabels[i] {
					label := &eventLabels[i][k]
					eh := eventHeight(label)
					if ey > eventsY {
						d.MoveTo(cx, ey-eventGap).LineTo(cx, ey)
					}
					cards = append(cards, event(*label, color, x, ey, eh))
					ey += eh + eventGap
				}
				bottom = max(bottom, ey-eventGap)
				period(i, color, x, periodsY, chipH)
				connector(d)
				dot(cx, axisY, color)
				items = append(items, cards...)
			}
			slot += span
		}
		if drawn {
			axisLine(left, axisY, left+bodyW, axisY, 1, 0)
		}
		height = bottom + pad
	}

	totalWidth := math.Ceil(width + 2*pad)
	totalHeight := math.Ceil(max(height, pad*2))
	if !title.Empty() {
		items = append([]scene.Item{{
			Kind: scene.Text, Role: "timeline.title", Label: &title,
			At:    scene.Point{X: totalWidth / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		}}, items...)
	}
	return &scene.Scene{
		Type:     "timeline",
		Width:    totalWidth,
		Height:   totalHeight,
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
		Items:    items,
	}
}
