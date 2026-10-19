package gantt

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of a gantt chart, as Pele's render.ts has them.
const (
	chartWidth = 800
	minPlot    = 240
	labelGap   = 8
)

// setting is a number of the chart's config, if it is a finite one, held between least and
// most, as Pele's setting reads it.
func setting(config diagram.Config, key string, least, most float64) (float64, bool) {
	g, _ := config["gantt"].(diagram.Config)
	v, ok := g[key].(float64)
	if !ok || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return min(most, max(least, v)), true
}

// settingOr is setting, or fallback when the config has none, as Pele writes
// setting(…) ?? fallback.
func settingOr(config diagram.Config, key string, least, most, fallback float64) float64 {
	if v, ok := setting(config, key, least, most); ok {
		return v
	}
	return fallback
}

// textSetting is a string of the chart's config, as Pele's text reads it.
func textSetting(config diagram.Config, key string) (string, bool) {
	g, _ := config["gantt"].(diagram.Config)
	s, ok := g[key].(string)
	return s, ok
}

// flag is whether a setting of the chart's config is true, as Pele's flag reads it.
func flag(config diagram.Config, key string) bool {
	g, _ := config["gantt"].(diagram.Config)
	v, ok := g[key].(bool)
	return ok && v
}

// breakAt is the length of a match of /<br\s*\/?>/i at i, or 0: Pele's RE_BREAK, written out
// by hand.
func breakAt(s string, i int) int {
	if i+3 > len(s) || s[i] != '<' || !chars.EqualFoldASCII(s[i+1:i+3], "br") {
		return 0
	}
	j := chars.SpaceRun(s, i+3)
	if j < len(s) && s[j] == '/' {
		j++
	}
	if j < len(s) && s[j] == '>' {
		return j + 1 - i
	}
	return 0
}

// oneLine lays out a task's label on one row, as Pele's oneLine does: <br> is a space, and
// only the first line is kept.
func oneLine(raw string, o *layout.Options, size float64) text.Label {
	var b strings.Builder
	last := 0
	for i := 0; i < len(raw); i++ {
		if n := breakAt(raw, i); n > 0 {
			b.WriteString(raw[last:i])
			b.WriteByte(' ')
			i += n - 1
			last = i + 1
		}
	}
	if last > 0 {
		b.WriteString(raw[last:])
		raw = b.String()
	}
	label := text.Layout(raw, text.Options{Measurer: o.Measure(), Size: size, MaxWidth: 4000, Icons: o.Icons})
	if len(label.Lines) <= 1 {
		return label
	}
	label.Lines, label.Widths = label.Lines[:1], label.Widths[:1]
	label.Width, label.Height = label.Widths[0], label.LineHeight
	return label
}

// scheduled is whether scheduling gave the task a start and an end, as Pele's scheduled
// says. One that waits on a task that never resolves has neither.
func scheduled(t *Task) bool {
	finite := func(m moment) bool { v := m.value(); return !math.IsNaN(v) && !math.IsInf(v, 0) }
	return finite(t.start) && finite(t.end)
}

// packRows puts each task of a section on the first row that is free when it starts, as
// Pele's packRows does for Mermaid's compact mode. It gives the number of rows taken.
func packRows(tasks []*Task, offset int, rowOf map[*Task]int) int {
	var sorted []*Task
	for _, t := range tasks {
		if scheduled(t) {
			sorted = append(sorted, t)
		}
	}
	slices.SortStableFunc(sorted, func(a, b *Task) int {
		if d := a.start.ms - b.start.ms; d != 0 && !math.IsNaN(d) {
			if d < 0 {
				return -1
			}
			return 1
		}
		return a.Order - b.Order
	})
	var free []float64
	for _, t := range sorted {
		row := 0
		for row < len(free) && t.start.ms < free[row] {
			row++
		}
		if row == len(free) {
			free = append(free, 0)
		}
		free[row] = t.end.ms
		rowOf[t] = offset + row
	}
	rows := len(free)
	for _, t := range tasks {
		if _, ok := rowOf[t]; !ok {
			rowOf[t] = offset + rows
			rows++
		}
	}
	return max(rows, 1)
}

// A bar is a task as it is drawn, Pele's Bar.
type bar struct {
	task        *Task
	label       text.Label
	row         int
	x0, x1      float64
	labelX      float64
	inside      bool
	unscheduled bool
}

// tickText is a tick's label, which Pele writes as plain text centered on its point: one line
// of one span, as wide as the measurer says.
func tickText(s string, width, size float64) *text.Label {
	lh := jsmath.Round(size * text.LineHeight)
	return &text.Label{
		Lines:  [][]text.Span{{{Text: s, Width: width}}},
		Widths: []float64{width}, Width: width, Height: lh, Size: size, LineHeight: lh,
	}
}

// Layout schedules a parsed gantt chart's tasks and lays the chart out as a scene, as Pele's
// renderGantt draws it: a row for each task, on a time axis as wide as the chart may be. It
// schedules in o.Location against o.Now; an error is a *diagram.Error, from scheduling.
func Layout(db *DB, config diagram.Config, o *layout.Options) (*scene.Scene, error) {
	if o == nil {
		o = &layout.Options{}
	}
	size := o.Size()
	measure := o.Measure()
	const pad = common.Padding
	now := float64(o.Clock().UnixMilli())

	labelSize := settingOr(config, "fontSize", 4, 72, jsmath.Round(size*0.8125))
	sectionSize := settingOr(config, "sectionFontSize", 4, 72, labelSize)
	tickSize := jsmath.Round(size * 0.75)
	barHeight := settingOr(config, "barHeight", 4, 400, jsmath.Round(labelSize*1.85))
	barGap := settingOr(config, "barGap", 0, 400, jsmath.Round(barHeight/4))
	rowHeight := barHeight + barGap
	sectionStyles := jsmath.Round(settingOr(config, "numberSectionStyles", 1, 1000, 4))
	compact := db.DisplayMode == "compact"
	topAxis := db.TopAxis || flag(config, "topAxis")

	if err := db.Schedule(o.Clock(), o.Zone()); err != nil {
		return nil, err
	}
	z := db.zone
	tasks := db.Tasks
	var rowTasks, verts []*Task
	for _, t := range tasks {
		if !t.Vert {
			rowTasks = append(rowTasks, t)
		} else if scheduled(t) {
			verts = append(verts, t)
		}
	}

	// The sections, in the order their first tasks come.
	categories := map[string]int{}
	var sections []string
	for _, t := range rowTasks {
		if _, ok := categories[t.Section]; !ok {
			categories[t.Section] = len(sections)
			sections = append(sections, t.Section)
		}
	}

	rowOf := map[*Task]int{}
	var rowSection []string
	if compact {
		groups := map[string][]*Task{}
		for _, t := range rowTasks {
			groups[t.Section] = append(groups[t.Section], t)
		}
		for _, section := range sections {
			rows := packRows(groups[section], len(rowSection), rowOf)
			for range rows {
				rowSection = append(rowSection, section)
			}
		}
	} else {
		for _, t := range rowTasks {
			rowOf[t] = len(rowSection)
			rowSection = append(rowSection, t.Section)
		}
	}
	rowCount := len(rowSection)

	start, stop := math.Inf(1), math.Inf(-1)
	for _, t := range tasks {
		if !scheduled(t) {
			continue
		}
		start = min(start, t.start.ms)
		stop = max(stop, t.end.ms)
	}
	hasPlot := start <= stop

	title := text.Layout(db.Title, text.Options{Measurer: measure, Size: size, MaxWidth: 4000, Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 16
	}
	sc := &scene.Scene{Type: "gantt", FontSize: size, Title: db.AccTitle, Desc: db.AccDescr}
	// titleItem is the chart's title, which Pele's renderGantt writes with labelSvg first.
	titleItem := func() []scene.Item {
		if title.Empty() {
			return nil
		}
		return []scene.Item{{
			Kind: scene.Text, Role: "gantt.title", Label: &title,
			At: scene.Point{X: sc.Width / 2, Y: pad + title.Height/2},
		}}
	}

	if !hasPlot {
		sc.Width = math.Ceil(max(title.Width, 1) + 2*pad)
		sc.Height = math.Ceil(max(title.Height, 1) + 2*pad)
		sc.Items = titleItem()
		return sc, nil
	}

	sectionLabels := map[string]*text.Label{}
	sectionWidth := 0.0
	for _, section := range sections {
		label := text.Layout(section, text.Options{Measurer: measure, Size: sectionSize, MaxWidth: 180, Icons: o.Icons})
		sectionLabels[section] = &label
		sectionWidth = max(sectionWidth, label.Width)
	}

	axisFormat := db.AxisFormat
	if axisFormat == "" {
		if db.DateFormat == "D" {
			axisFormat = "%d"
		} else if s, ok := textSetting(config, "axisFormat"); ok {
			axisFormat = s
		} else {
			axisFormat = "%Y-%m-%d"
		}
	}
	// tickLabel is Pele's tickLabel: a tick's time in the axis format.
	tickLabel := func(t float64) string { return z.timeFormat(axisFormat, t) }
	sampleWidth := max(measure.Width(tickLabel(start), tickSize, 0), measure.Width(tickLabel(stop), tickSize, 0))

	leftPadding, ok := setting(config, "leftPadding", 0, 2000)
	if !ok {
		leftPadding = 0
		if sectionWidth > 0 {
			leftPadding = sectionWidth + 24
		}
		leftPadding = max(leftPadding, sampleWidth/2)
	}
	left := pad + leftPadding
	rightPadding := settingOr(config, "rightPadding", 0, 2000, max(16, sampleWidth/2))
	target := common.CapWidth(o, settingOr(config, "useWidth", 100, 20000, chartWidth), 280, 0)
	plotWidth := max(minPlot, target-left-rightPadding-pad)
	right := left + plotWidth
	scale := 0.0
	if stop > start {
		scale = plotWidth / (stop - start)
	}
	// x is Pele's x: where a time falls on the plot, its middle when the chart spans no time.
	x := func(t float64) float64 {
		if scale == 0 {
			return left + plotWidth/2
		}
		return left + float64((t-start)*scale)
	}

	tickHeight := jsmath.Round(tickSize * 1.75)
	topPadding, hasTop := setting(config, "topPadding", 0, 2000)
	gridStart, hasGrid := setting(config, "gridLineStartPadding", 0, 2000)
	headHeight := pad + titleHeight
	if topAxis {
		headHeight += tickHeight
	}
	plotTop := max(headHeight, topPadding)
	// Mermaid starts its grid lines above the first row by the difference between these two
	// settings. Here that only uses room a larger topPadding leaves free.
	gridLead := 0.0
	if hasTop && hasGrid {
		gridLead = max(0, min(topPadding-gridStart, plotTop-headHeight))
	}
	plotHeight := float64(max(rowCount, 1)) * rowHeight
	plotBottom := plotTop + plotHeight

	// Bars and where their labels go: inside when they fit, otherwise beside the bar on the
	// side with room.
	var bars []bar
	extent := right + rightPadding
	for _, t := range rowTasks {
		label := oneLine(t.Task, o, labelSize)
		row := rowOf[t]
		if !scheduled(t) {
			labelX := left + labelGap + label.Width/2
			extent = max(extent, labelX+label.Width/2)
			bars = append(bars, bar{task: t, label: label, row: row, x0: math.NaN(), x1: math.NaN(), labelX: labelX, unscheduled: true})
			continue
		}
		x0 := x(t.start.ms)
		end := t.end
		if t.renderEnd.defined {
			end = t.renderEnd
		}
		grow := 2.0
		if t.Milestone {
			grow = 0
		}
		x1 := max(x(end.ms), x0+grow)
		if t.Milestone {
			mid := x0 + (x(t.end.ms)-x0)/2
			x0 = mid - barHeight/2
			x1 = mid + barHeight/2
		}
		inside := !t.Milestone && label.Width+2*labelGap <= x1-x0
		labelX := (x0 + x1) / 2
		if !inside {
			fitsRight := x1+labelGap+label.Width <= right+rightPadding
			fitsLeft := x0-labelGap-label.Width >= left
			if fitsRight || !fitsLeft {
				labelX = x1 + labelGap + label.Width/2
			} else {
				labelX = x0 - labelGap - label.Width/2
			}
			extent = max(extent, labelX+label.Width/2)
		}
		bars = append(bars, bar{task: t, label: label, row: row, x0: x0, x1: x1, labelX: labelX, inside: inside})
	}

	interval := db.TickInterval
	if interval == "" {
		interval, _ = textSetting(config, "tickInterval")
	}
	weekday := db.Weekday
	if weekday == "" {
		var ok bool
		if weekday, ok = textSetting(config, "weekday"); !ok {
			weekday = "sunday"
		}
	}
	count := max(2, min(10, math.Floor(plotWidth/(sampleWidth+24))))
	tickTimes, ok := z.intervalTicks(interval, weekday, start, stop, int(math.Ceil(plotWidth/3)))
	if !ok {
		tickTimes = z.autoTicks(start, stop, count)
	}

	grid := new(scene.Path)
	var axisBottom, axisTop []scene.Item
	lastRight := math.Inf(-1)
	lastLabel := ""
	bottomY := plotBottom + tickHeight/2 + 2
	topY := plotTop - gridLead - tickHeight/2 - 2
	for _, t := range tickTimes {
		tx := x(t)
		grid.MoveTo(tx, plotTop-gridLead).LineTo(tx, plotBottom)
		label := tickLabel(t)
		width := measure.Width(label, tickSize, 0)
		// A label is skipped when it would run into the one before it or only repeat it; its
		// grid line stays.
		if tx-width/2 < lastRight+8 || label == lastLabel {
			continue
		}
		lastRight = tx + width/2
		lastLabel = label
		extent = max(extent, lastRight)
		tick := scene.Item{
			Kind: scene.Text, Role: "gantt.tick", Label: tickText(label, width, tickSize),
			At: scene.Point{X: tx, Y: bottomY}, Style: style.Style{Text: style.Muted},
		}
		axisBottom = append(axisBottom, tick)
		if topAxis {
			tick.At.Y = topY
			axisTop = append(axisTop, tick)
		}
	}

	// Vertical markers take no row. Their labels go under the axis, on as many lines as it
	// takes to keep them apart.
	var vertLines, vertLabels []scene.Item
	var vertEnds []float64
	vertTop := plotBottom + tickHeight + 4
	vertLine := jsmath.Round(labelSize * 1.5)
	slices.SortStableFunc(verts, func(a, b *Task) int {
		switch d := a.start.ms - b.start.ms; {
		case d < 0:
			return -1
		case d > 0:
			return 1
		}
		return 0
	})
	for _, t := range verts {
		vx := x(t.start.ms)
		label := oneLine(t.Task, o, labelSize)
		cx := max(pad+label.Width/2, vx)
		line := 0
		for line < len(vertEnds) && cx-label.Width/2 < vertEnds[line]+8 {
			line++
		}
		if line == len(vertEnds) {
			vertEnds = append(vertEnds, 0)
		}
		vertEnds[line] = cx + label.Width/2
		extent = max(extent, vertEnds[line])
		path := new(scene.Path).MoveTo(vx, plotTop-gridLead).LineTo(vx, plotBottom)
		vertLines = append(vertLines, scene.Item{
			Kind: scene.Mark, Role: "gantt.vert", ID: t.ID, Box: path.Bounds(),
			Parts: []scene.Part{{Path: path, Paint: scene.StrokeOnly}},
			Style: style.Style{Fill: style.None, Stroke: style.Line, StrokeWidth: style.N(1.5), Dash: style.Dashes(4, 3)},
		})
		if !label.Empty() {
			vertLabels = append(vertLabels, scene.Item{
				Kind: scene.Text, Role: "gantt.vert-label", ID: t.ID, Label: &label,
				At: scene.Point{X: cx, Y: vertTop + float64(float64(line)*vertLine) + vertLine/2},
			})
		}
	}

	vertRoom := 0.0
	if len(vertEnds) > 0 {
		vertRoom = 4 + float64(float64(len(vertEnds))*vertLine)
	}
	sc.Width = math.Ceil(max(extent, pad+title.Width) + pad)
	sc.Height = math.Ceil(plotBottom + tickHeight + vertRoom + pad)

	// Section bands, one per run of rows that share a section, shaded on alternate sections.
	var bands, sectionOut []scene.Item
	for row := 0; row < rowCount; {
		section := rowSection[row]
		end := row + 1
		for end < rowCount && rowSection[end] == section {
			end++
		}
		index := math.Mod(float64(categories[section]), sectionStyles)
		y := plotTop + float64(float64(row)*rowHeight)
		h := float64(end-row) * rowHeight
		if len(categories) > 1 || section != "" {
			fill := style.None
			if math.Mod(index, 2) == 0 {
				fill = style.Surface
			}
			box := scene.Rect{X: 0, Y: y, W: sc.Width, H: h}
			bands = append(bands, scene.Item{
				Kind: scene.Mark, Role: "gantt.section", Shape: "rect", ID: section,
				Tags: []string{"section-" + strconv.FormatFloat(index, 'f', -1, 64)},
				Box:  box, Parts: []scene.Part{{Box: box}},
				Style: style.Style{Fill: fill, Stroke: style.None},
			})
		}
		if label := sectionLabels[section]; !label.Empty() {
			sectionOut = append(sectionOut, scene.Item{
				Kind: scene.Text, Role: "gantt.section-label", ID: section, Label: label,
				At: scene.Point{X: pad + 8 + label.Width/2, Y: y + h/2},
			})
		}
		row = end
	}

	// Excluded days, as Mermaid marks them: a day at a time from the first task's start, and
	// not at all when the chart spans more than five years.
	var excluded []scene.Item
	if (len(db.Excludes) > 0 || len(db.Includes) > 0) && z.add(start, 6, "y") > stop {
		from, to := math.NaN(), math.NaN()
		have := false
		// flush is Pele's flush: it marks the run of excluded days from from to to, if there is one.
		flush := func() {
			if !have {
				return
			}
			dayStart := z.SetHours(from, 0, 0, 0, 0)
			dayEnd := z.SetHours(to, 23, 59, 59, 999)
			ex0 := max(left, x(dayStart))
			ex1 := min(right, x(dayEnd))
			if ex1 > ex0 {
				box := scene.Rect{X: ex0, Y: plotTop, W: ex1 - ex0, H: plotHeight}
				excluded = append(excluded, scene.Item{
					Kind: scene.Mark, Role: "gantt.excluded", Shape: "rect", Box: box, Parts: []scene.Part{{Box: box}},
					Style: style.Style{Fill: style.SurfaceAlt, Stroke: style.None},
				})
			}
			have = false
		}
		for day := start; day <= stop; day = z.addDays(day, 1) {
			invalid, err := db.isInvalidDate(day)
			if err != nil {
				return nil, err
			}
			if invalid {
				if !have {
					from, have = day, true
				}
				to = day
			} else {
				flush()
			}
		}
		flush()
	}

	var taskItems []scene.Item
	for i := range bars {
		b := &bars[i]
		t := b.task
		cy := plotTop + float64(float64(b.row)*rowHeight) + rowHeight/2
		state := ""
		switch {
		case t.Active:
			state = "active"
		case t.Done:
			state = "done"
		}
		// The classes Pele gives the task besides pele-task, the task's tags.
		var class []string
		if state != "" {
			class = append(class, state)
		}
		if t.Crit {
			class = append(class, "crit")
		}
		if t.Milestone {
			class = append(class, "milestone")
		}
		if slices.Contains(t.Classes, "clickable") {
			class = append(class, "clickable")
		}
		it := scene.Item{Kind: scene.Node, Role: "gantt.task", ID: t.ID}
		if b.unscheduled {
			class = append(class, "unscheduled")
			it.Style.Text = style.Muted
		} else {
			// A critical task takes the critical color in place of the accent. Done, it keeps
			// the fill of finished work and only its border is in that color.
			color := style.Accent
			if t.Crit {
				color = style.Critical
			}
			switch state {
			case "done":
				it.Style.Fill = style.SurfaceAlt
			case "active":
				it.Style.Fill, it.Style.FillOpacity = color, style.N(0.25)
			default:
				it.Style.Fill = color
			}
			it.Style.Stroke = color
			if state == "done" && !t.Crit {
				it.Style.Stroke = style.Border
			}
			if t.Milestone {
				cx := (b.x0 + b.x1) / 2
				r := barHeight / 2
				it.Shape = "milestone"
				path := new(scene.Path).Polygon(cx, cy-r, cx+r, cy, cx, cy+r, cx-r, cy)
				it.Box = path.Bounds()
				it.Parts = []scene.Part{{Path: path}}
				// Pele turns a milestone's corners round.
				it.Style.Join = style.JoinRound
			} else {
				it.Shape = "rect"
				it.Box = scene.Rect{X: b.x0, Y: cy - barHeight/2, W: b.x1 - b.x0, H: barHeight}
				it.Parts = []scene.Part{{Box: it.Box, Rounded: true}}
			}
			if b.inside {
				if state == "" {
					it.Style.Text = style.Bg
				}
			} else if state == "done" {
				it.Style.Text = style.Muted
			}
		}
		it.Tags = class
		if !b.label.Empty() {
			it.Label = &b.label
		}
		it.At = scene.Point{X: b.labelX, Y: cy}
		if link, ok := db.Links[t.ID]; ok {
			if href, ok := url.LinkURL(link, !o.NoLinks, o.LinkSchemes); ok {
				it.Link = href
			}
		}
		taskItems = append(taskItems, it)
	}

	var today []scene.Item
	if db.TodayMarker != "off" && now >= start && now <= stop && stop > start {
		tx := x(now)
		top := plotTop - gridLead
		line := new(scene.Path).MoveTo(tx, top).LineTo(tx, plotBottom)
		it := scene.Item{
			Kind: scene.Mark, Role: "gantt.today", Box: line.Bounds(),
			Parts: []scene.Part{{Path: line, Paint: scene.StrokeOnly}},
			Style: style.Style{Fill: style.Accent, Stroke: style.Accent, StrokeWidth: style.N(2)},
		}
		decls := strings.Split(db.TodayMarker, ",")
		if db.TodayMarker != "" && style.LineDeclared(decls) {
			// The author's style is that of a line: it fills nothing and colors no text.
			author := style.Declarations(decls).Style
			author.Fill, author.Text = style.Paint{}, style.Paint{}
			it.Author = author
		} else {
			mark := new(scene.Path).MoveTo(tx-4, top-5).LineTo(tx+4, top-5).LineTo(tx, top+1).Close()
			it.Parts = append(it.Parts, scene.Part{Path: mark, Paint: scene.FillOnly})
			it.Box = scene.Rect{X: tx - 4, Y: top - 5, W: 8, H: plotBottom - (top - 5)}
		}
		today = append(today, it)
	}

	axisLine := new(scene.Path).MoveTo(left, plotBottom).LineTo(right, plotBottom)
	if topAxis {
		axisLine.MoveTo(left, plotTop-gridLead).LineTo(right, plotTop-gridLead)
	}
	var lines []scene.Item
	if len(grid.Ops) > 0 {
		lines = append(lines, scene.Item{
			Kind: scene.Mark, Role: "gantt.grid", Box: grid.Bounds(),
			Parts: []scene.Part{{Path: grid, Paint: scene.StrokeOnly}},
			Style: style.Style{Fill: style.None, Stroke: style.Border, Dash: style.Dashes(1, 3)},
		})
	}
	lines = append(lines, scene.Item{
		Kind: scene.Mark, Role: "gantt.axis", Box: axisLine.Bounds(),
		Parts: []scene.Part{{Path: axisLine, Paint: scene.StrokeOnly}},
		Style: style.Style{Fill: style.None, Stroke: style.Border},
	})

	for _, layer := range [][]scene.Item{titleItem(), bands, excluded, lines, vertLines, today, taskItems, sectionOut, axisBottom, axisTop, vertLabels} {
		sc.Items = append(sc.Items, layer...)
	}
	return sc, nil
}
