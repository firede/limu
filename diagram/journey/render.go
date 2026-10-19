package journey

import (
	"math"
	"strconv"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The measures of Pele's journey render.
const (
	padX      = 10
	padY      = 8
	minText   = 124
	maxText   = 200
	gap       = 12
	bandPad   = 6
	gutter    = 24
	level     = 26
	faceR     = 11
	dot       = 5
	dotStep   = 13
	legendGap = 20
)

// Layout lays a user journey diagram out as a scene, as Pele's renderJourney draws it: across,
// tasks as cards in a row with each one's score as a face on a chart under them, or down,
// when the host's width calls for it and allows it.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene {
		if down {
			return verticalJourney(m, o)
		}
		return across(m, o)
	})
}

// mood is the expression of a face for a score, as Pele chooses its mouth: a score that is
// not a number is neutral.
func mood(score float64) scene.Shape {
	switch {
	case score > 3:
		return "happy"
	case score < 3:
		return "sad"
	}
	return "neutral"
}

// face is a face about x, y, as Pele's face draws it: a ring, two eyes and a mouth, in the
// color of lines. Pele draws the mouth with round caps; the face is given them whole, as
// they show nowhere else on it: its ring is closed and its eyes are not stroked.
func face(x, y float64, mood scene.Shape) scene.Item {
	mouth := new(scene.Path)
	switch mood {
	case "happy":
		mouth.MoveTo(x+-5, y+2).ArcTo(6, 6, 0, false, false, x+5, y+2)
	case "sad":
		mouth.MoveTo(x+-5, y+6.5).ArcTo(6, 6, 0, false, true, x+5, y+6.5)
	default:
		mouth.MoveTo(x+-4.5, y+4.5).LineTo(x+4.5, y+4.5)
	}
	return scene.Item{
		Kind: scene.Mark, Role: RoleFace, Shape: mood,
		Box: scene.Rect{X: x - faceR, Y: y - faceR, W: 2 * faceR, H: 2 * faceR},
		Parts: []scene.Part{
			{Path: new(scene.Path).Ellipse(x, y, faceR, faceR), Paint: scene.Hollow},
			{Path: new(scene.Path).Ellipse(x+-3.6, y+-2.8, 1.3, 1.3), Paint: scene.FillOnly},
			{Path: new(scene.Path).Ellipse(x+3.6, y+-2.8, 1.3, 1.3), Paint: scene.FillOnly},
			{Path: mouth, Paint: scene.StrokeOnly},
		},
		Style: style.Style{Fill: style.Line, Stroke: style.Line, Cap: style.CapRound},
	}
}

// actorNames is everyone named on a task but the actor with no name, and the index of each,
// which is the series it is drawn in.
func actorNames(tasks []Task) ([]string, map[string]int) {
	var names []string
	for _, name := range Actors(tasks) {
		if name != "" {
			names = append(names, name)
		}
	}
	index := map[string]int{}
	for i, name := range names {
		index[name] = i
	}
	return names, index
}

// named is a task's people but the actor with no name.
func named(people []string) []string {
	var out []string
	for _, name := range people {
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// textItem is a label on its own, its first line's baseline at y.
func textItem(role scene.Role, label text.Label, x, y float64, align scene.Align, paint style.Paint) scene.Item {
	return scene.Item{
		Kind: scene.Text, Role: role, Label: &label, Align: align,
		At:    scene.Point{X: x, Y: y - float64(label.Size*0.35)},
		Style: style.Style{Text: paint},
	}
}

// across lays the journey out across, as Pele's draw does with down false.
func across(m *Model, o *layout.Options) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	tiny := jsmath.Round(size * 0.75)
	measure := o.Measure()
	const pad = float64(common.Padding)
	tasks := m.Tasks
	n := len(tasks)

	// `Task: 5:` names an actor with no name; Mermaid gives it a dot and an empty legend row.
	names, actor := actorNames(tasks)

	// Cards widen, up to a point, for a word that would not fit; past that the word is cut.
	textW := float64(minText)
	labels := make([]text.Label, n)
	for i, task := range tasks {
		labels[i] = common.PlainLabel(task.Task, measure, small, textW, 0)
	}
	widest := 0.0
	for _, label := range labels {
		widest = max(widest, label.Width)
	}
	if widest > textW {
		textW = min(math.Ceil(widest), maxText)
		for i, task := range tasks {
			labels[i] = common.FitLabel(task.Task, measure, small, textW, 0)
		}
	}

	cardW := textW + 2*padX
	pitch := cardW + gap
	plotW := 0.0
	if n > 0 {
		plotW = float64(float64(n)*pitch) - gap
	}

	lineH := jsmath.Round(small * 1.5)
	limit := max(plotW, 240)
	type entry struct {
		name    string
		label   text.Label
		x, y, h float64
	}
	var legend []entry
	legendW, legendH, lx := 0.0, 0.0, 0.0
	for _, name := range names {
		label := common.FitLabel(name, measure, small, limit-2*dot-6, 0)
		w := 2*dot + 6 + label.Width
		if lx > 0 && lx+w > limit {
			lx = 0
			legendH += 4
		}
		y := legendH
		if lx > 0 {
			y = legend[len(legend)-1].y
		}
		h := max(label.Height, lineH)
		legend = append(legend, entry{name, label, lx, y, h})
		lx += w
		legendW = max(legendW, lx)
		legendH = max(legendH, y+h)
		lx += legendGap
	}

	bodyW := 0.0
	if n > 0 {
		bodyW = gutter + plotW
	}
	title := common.FitLabel(m.Title, measure, size, max(bodyW, 320), text.Bold)
	titleH := 0.0
	if title.Height > 0 {
		titleH = title.Height + 16
	}
	legendRow := 0.0
	if n > 0 {
		legendRow = gutter + legendW
	}
	width := max(bodyW, title.Width, legendRow)
	left := pad + (width-bodyW)/2 + gutter
	top := pad + titleH

	// A band covers a run of tasks that share a section, as in Mermaid; a section with no
	// tasks draws nothing.
	type run struct {
		start, count int
		label        *text.Label
	}
	var runs []run
	bandH := 0.0
	for i := 0; i < n; {
		end := i + 1
		for end < n && tasks[end].Section == tasks[i].Section {
			end++
		}
		var label *text.Label
		if tasks[i].Section != "" {
			l := common.FitLabel(tasks[i].Section, measure, small, float64(float64(end-i)*pitch)-gap-2*padX, text.Bold)
			label = &l
			bandH = max(bandH, max(l.Height, lineH)+2*bandPad)
		}
		runs = append(runs, run{i, end - i, label})
		i = end
	}

	textH := lineH
	for _, label := range labels {
		textH = max(textH, label.Height)
	}
	dotsH := 0.0
	if len(names) > 0 {
		dotsH = 2*dot + 6
	}
	cardH := textH + dotsH + 2*padY
	cardsY := top
	if bandH > 0 {
		cardsY = top + (bandH + bandPad)
	}
	chartTop := cardsY + cardH + faceR + 16
	chartBottom := chartTop + 4*level
	right := left + plotW

	var items []scene.Item
	bottom := top
	if n > 0 {
		grid := new(scene.Path)
		var scale []scene.Item
		for score := 5; score >= 1; score-- {
			y := chartTop + float64(float64(5-score)*level)
			grid.MoveTo(left-6, y).LineTo(right, y)
			label := text.Layout(strconv.Itoa(score), text.Options{Measurer: measure, Size: tiny})
			scale = append(scale, scene.Item{
				Kind: scene.Text, Role: RoleScale, Label: &label, Align: scene.End,
				At: scene.Point{X: left - 12, Y: y}, Style: style.Style{Text: style.Muted},
			})
		}
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: RoleGrid, Box: grid.Bounds(),
			Parts: []scene.Part{{Path: grid, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.SurfaceAlt},
		})
		items = append(items, scale...)

		line := new(scene.Path)
		var route []scene.Point
		stems := new(scene.Path)
		var faces []scene.Item
		cards := make([][]scene.Item, n)
		for i, task := range tasks {
			x := left + float64(float64(i)*pitch)
			cx := x + cardW/2
			score := task.Score
			lv := 3.0
			if !math.IsNaN(score) {
				lv = min(5, max(1, score))
			}
			y := chartTop + float64((5-lv)*level)
			if i == 0 {
				line.MoveTo(cx, y)
			} else {
				line.LineTo(cx, y)
			}
			route = append(route, scene.Point{X: cx, Y: y})
			stems.MoveTo(cx, cardsY+cardH).LineTo(cx, y-faceR)
			faces = append(faces, face(cx, y, mood(score)))

			label := labels[i]
			card := scene.Item{
				Kind: scene.Node, Role: RoleTask, Shape: "rect", ID: chars.Trim(task.Task),
				Box:   scene.Rect{X: x, Y: cardsY, W: cardW, H: cardH},
				Style: style.Style{Fill: style.Surface, Stroke: style.Border},
				At:    scene.Point{X: cx, Y: cardsY + padY + textH/2},
			}
			card.Parts = []scene.Part{{Box: card.Box, Rounded: true}}
			if !label.Empty() {
				card.Label = &label
			}
			cards[i] = append(cards[i], card)
			people := named(task.People)
			step := min(dotStep, (cardW-2*padX-2*dot)/float64(max(len(people)-1, 1)))
			for k, name := range people {
				dx := cx + float64((float64(k)-float64(len(people)-1)/2)*step)
				dy := cardsY + cardH - padY - dot
				cards[i] = append(cards[i], scene.Item{
					Kind: scene.Mark, Role: RoleActor, Shape: "circle", ID: name, Tooltip: name,
					Box:   scene.Rect{X: dx - dot, Y: dy - dot, W: 2 * dot, H: 2 * dot},
					Parts: []scene.Part{{Path: new(scene.Path).Ellipse(dx, dy, dot, dot)}},
					Style: style.Style{Fill: style.Series(actor[name]), Stroke: style.Surface},
				})
			}
		}
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: RoleStem, Box: stems.Bounds(),
			Parts: []scene.Part{{Path: stems, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.Border, Dash: style.Dashes(2, 3)},
		})
		if n > 1 {
			// Pele turns the line's corners round.
			items = append(items, scene.Item{
				Kind: scene.Edge, Role: RoleLine, Box: line.Bounds(), Route: route,
				Parts: []scene.Part{{Path: line, Paint: scene.StrokeOnly}},
				Style: style.Style{Stroke: style.Line, Join: style.JoinRound},
			})
		}
		items = append(items, faces...)

		for _, r := range runs {
			if r.label != nil {
				x := left + float64(float64(r.start)*pitch)
				w := float64(float64(r.count)*pitch) - gap
				band := scene.Item{
					Kind: scene.Group, Role: RoleSection, Shape: "rect", ID: chars.Trim(tasks[r.start].Section),
					Box:   scene.Rect{X: x, Y: top, W: w, H: bandH},
					Label: r.label, At: scene.Point{X: x + w/2, Y: top + bandH/2},
					Style: style.Style{Fill: style.SurfaceAlt},
				}
				band.Parts = []scene.Part{{Box: band.Box, Rounded: true, Paint: scene.FillOnly}}
				if r.label.Empty() {
					band.Label = nil
				}
				items = append(items, band)
			}
			for _, card := range cards[r.start : r.start+r.count] {
				items = append(items, card...)
			}
		}
		bottom = chartBottom + faceR

		if len(legend) > 0 {
			x0 := left + (plotW-legendW)/2
			y0 := bottom + 16
			for _, e := range legend {
				cy := y0 + e.y + e.h/2
				cx := x0 + e.x + dot
				it := scene.Item{
					Kind: scene.Node, Role: RoleLegend, Shape: "circle", ID: e.name,
					Box:   scene.Rect{X: x0 + e.x, Y: cy - e.h/2, W: 2*dot + 6 + e.label.Width, H: e.h},
					Parts: []scene.Part{{Path: new(scene.Path).Ellipse(cx, cy, dot, dot), Paint: scene.FillOnly}},
					At:    scene.Point{X: x0 + e.x + 2*dot + 6 + e.label.Width/2, Y: cy},
					Style: style.Style{Fill: style.Series(actor[e.name])},
				}
				if !e.label.Empty() {
					label := e.label
					it.Label = &label
				}
				items = append(items, it)
			}
			bottom = y0 + legendH
		}
	}

	sc := &scene.Scene{
		Type:     "journey",
		Width:    math.Ceil(width + 2*pad),
		Height:   math.Ceil(max(bottom, pad) + pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At: scene.Point{X: pad + width/2, Y: pad + title.Height/2},
		})
	}
	sc.Items = append(sc.Items, items...)
	return sc
}

// verticalJourney lays the journey out down, as Pele's verticalJourney does: a card for each
// task, under the head of its section, with its face and score at its right and its people
// under its name.
func verticalJourney(m *Model, o *layout.Options) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	measure := o.Measure()
	const pad = float64(common.Padding)
	maxWidth := o.MaxWidth
	if !(maxWidth > 0) {
		maxWidth = 320
	}
	room := max(140, maxWidth-2*pad)
	title := common.FitLabel(m.Title, measure, size, room, text.Bold)
	_, actors := actorNames(m.Tasks)
	y := pad
	if title.Height != 0 {
		y = pad + (title.Height + 16)
	}
	section, started := "", false
	var items []scene.Item
	width := title.Width
	for index, task := range m.Tasks {
		if !started || task.Section != section {
			section, started = task.Section, true
			head := common.FitLabel(section, measure, size, room-16, text.Bold)
			if head.Height != 0 {
				items = append(items, scene.Item{
					Kind: scene.Text, Role: RoleSection, Label: &head,
					At: scene.Point{X: pad + 8 + head.Width/2, Y: y + head.Height/2},
				})
				width = max(width, head.Width+16)
				y += head.Height + 8
			}
		}
		// Pele measures a task's label at the small size and writes its text with no
		// font-size, so that a browser draws it at the diagram's own size, wider than it was
		// measured. Here it is drawn at the size it was measured at, as the scene's labels
		// are; where it is placed is Pele's.
		label := common.FitLabel(task.Task, measure, small, room-72, 0)
		type person struct {
			name  string
			label text.Label
		}
		var people []person
		for _, name := range named(task.People) {
			people = append(people, person{name, common.FitLabel(name, measure, small, room-44, 0)})
		}
		textH := max(label.Height, 42)
		h := 2*padY + textH
		for _, p := range people {
			h += max(16, p.label.Height) + 4
		}
		w := max(room, label.Width+72)
		card := scene.Item{
			Kind: scene.Node, Role: RoleTask, Shape: "rect", ID: chars.Trim(task.Task),
			Box:   scene.Rect{X: pad, Y: y, W: w, H: h},
			At:    scene.Point{X: pad + padX + label.Width/2, Y: y + padY + label.Height/2},
			Style: style.Style{Fill: style.Surface, Stroke: style.Border},
		}
		card.Parts = []scene.Part{{Box: card.Box, Rounded: true}}
		if !label.Empty() {
			card.Label = &label
		}
		items = append(items, card)
		cx := pad + w - 28
		items = append(items, face(cx, y+padY+faceR, mood(task.Score)))
		if !math.IsNaN(task.Score) {
			score := text.Layout(scoreText(min(5, max(1, task.Score))), text.Options{Measurer: measure, Size: small})
			items = append(items, textItem(RoleScore, score, cx, y+padY+2*faceR+small, scene.Middle, style.Paint{}))
		}
		py := y + padY + textH + 4
		for _, p := range people {
			ph := max(16, p.label.Height)
			cy := py + ph/2
			it := scene.Item{
				Kind: scene.Node, Role: RoleActor, Shape: "circle", ID: p.name,
				Box:   scene.Rect{X: pad + 16 - dot, Y: py, W: 12 + dot + p.label.Width, H: ph},
				Parts: []scene.Part{{Path: new(scene.Path).Ellipse(pad+16, cy, dot, dot), Paint: scene.FillOnly}},
				At:    scene.Point{X: pad + 28 + p.label.Width/2, Y: cy},
				Style: style.Style{Fill: style.Series(actors[p.name])},
			}
			if !p.label.Empty() {
				label := p.label
				it.Label = &label
			}
			items = append(items, it)
			py += ph + 4
		}
		y += h
		if index+1 < len(m.Tasks) {
			x := pad + w/2
			link := new(scene.Path).MoveTo(x, y).LineTo(x, y+16)
			items = append(items, scene.Item{
				Kind: scene.Edge, Role: RoleLine, Box: link.Bounds(),
				Route: []scene.Point{{X: x, Y: y}, {X: x, Y: y + 16}},
				Parts: []scene.Part{{Path: link, Paint: scene.StrokeOnly}},
				Style: style.Style{Stroke: style.Line},
			})
		}
		y += 16
		width = max(width, w)
	}
	totalWidth := math.Ceil(width + 2*pad)
	last := y
	if len(m.Tasks) > 0 {
		last = y - 16
	}
	sc := &scene.Scene{
		Type:     "journey",
		Width:    totalWidth,
		Height:   math.Ceil(max(pad+title.Height, last) + pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At: scene.Point{X: totalWidth / 2, Y: pad + title.Height/2},
		})
	}
	sc.Items = append(sc.Items, items...)
	return sc
}

// scoreText is a score from 1 to 5 as Pele's num writes it: to two decimals, with no zeros
// at the end.
func scoreText(v float64) string {
	n := jsmath.Round(float64(v * 100))
	q := math.Floor(n / 100)
	s := strconv.FormatFloat(q, 'f', -1, 64)
	switch i := int(n - float64(q*100)); {
	case i == 0:
	case i%10 == 0:
		s += "." + strconv.Itoa(i/10)
	case i < 10:
		s += ".0" + strconv.Itoa(i)
	default:
		s += "." + strconv.Itoa(i)
	}
	return s
}
