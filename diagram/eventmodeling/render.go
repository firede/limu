package eventmodeling

import (
	"math"
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

// The scene's roles. A frame is a node of the role of its kind, "eventmodeling.command" and
// the rest, its name its label, with the tag "reset" for a reset frame; its example data
// is a text item after it. Across, a lane is a group, with its label; down, each frame's lane
// is named by a text item before it, and its number by one after it. A note and a
// specification are nodes, their heading the label, their lines a text item after each.
const (
	RoleLane          scene.Role = "eventmodeling.lane"
	RoleLaneLabel     scene.Role = "eventmodeling.lane-label"
	RoleRelation      scene.Role = "eventmodeling.relation"
	RoleData          scene.Role = "eventmodeling.data"
	RoleFrameID       scene.Role = "eventmodeling.frame-id"
	RoleNote          scene.Role = "eventmodeling.note"
	RoleNoteText      scene.Role = "eventmodeling.note-text"
	RoleSpecification scene.Role = "eventmodeling.specification"
	RoleSpecText      scene.Role = "eventmodeling.specification-text"
	RoleTitle         scene.Role = "eventmodeling.title"
)

// series is Pele's SERIES: the series color of each kind of entity. A UI has none and is
// drawn plain.
var series = map[Kind]int{
	"command":   0,
	"event":     1,
	"readmodel": 2,
	"processor": 6,
}

const (
	boxPadX     = 12
	boxPadY     = 10
	boxMinWidth = 96
	boxGap      = 16
	stagger     = 80
	lanePad     = 12
	laneGap     = 8
	dataColumns = 48
	noteColumns = 80
	cardGap     = 8
	railGap     = 8
	cardMin     = 120
	arrowRoom   = 20
	// wash is the opacity of a frame's series color over the background.
	wash = 0.22
)

// arrowTrim is how far a relation stops short of the tip of its arrow, Pele's
// markerTrim('arrow_point').
var arrowTrim = scene.MarkerType("arrow_point").Trim()

// Layout lays an event modeling diagram out as a scene, as Pele's renderEventModel does:
// across, in a lane for each group of kinds and namespace, or, when the host is narrow, down,
// a card to a frame. The diagram has no configuration of its own.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene {
		if down {
			return verticalEventModel(m, o)
		}
		return draw(m, o)
	})
}

// hardWrap splits text into lines of at most columns code units, keeping its own line
// breaks, as Pele's hardWrap does. Where Pele's cut falls between the two halves of a
// character past the Basic Multilingual Plane, the character stays whole on the line before.
func hardWrap(s string, columns int) []string {
	var out []string
	for _, line := range strings.Split(text.DecodeEntities(s), "\n") {
		if chars.Len16(line) <= columns {
			out = append(out, line)
			continue
		}
		from, units, cut := 0, 0, columns
		for i, r := range line {
			if units >= cut {
				out = append(out, line[from:i])
				from = i
				for cut <= units {
					cut += columns
				}
			}
			units++
			if r > 0xffff {
				units++
			}
		}
		out = append(out, line[from:])
	}
	return out
}

// rowsLabel is rows of text as Pele's lines writes them: a line each, step apart, in a size
// and face. An empty row is a line with no text, which nothing is drawn for.
func rowsLabel(rows []string, measure text.Measurer, size, step float64, face text.Face) text.Label {
	l := text.Label{Size: size, LineHeight: step, Height: float64(len(rows)) * step}
	for _, row := range rows {
		w := measure.Width(row, size, face)
		l.Lines = append(l.Lines, []text.Span{{Text: row, Face: face, Width: w}})
		l.Widths = append(l.Widths, w)
		l.Width = max(l.Width, w)
	}
	return l
}

// hasText reports whether a label has a line with text, as Pele writes a <text> only for one.
func hasText(l *text.Label) bool {
	for _, line := range l.Lines {
		for _, span := range line {
			if span.Text != "" {
				return true
			}
		}
	}
	return false
}

// rows is a text item of lines starting at x, the first of them below top, or nothing where
// no line has text.
func rows(role scene.Role, l text.Label, x, top float64) []scene.Item {
	if !hasText(&l) {
		return nil
	}
	return []scene.Item{{
		Kind: scene.Text, Role: role, Label: &l, Align: scene.Start,
		At:    scene.Point{X: x, Y: top + l.Height/2},
		Style: style.Style{Text: style.Fg},
	}}
}

// frameItem is the node of a frame in its box: plain for a UI, and for any other kind its
// series color laid over the background.
func frameItem(box *Box, r scene.Rect) scene.Item {
	it := scene.Item{
		Kind: scene.Node, Role: scene.Role("eventmodeling." + string(box.Kind)), Shape: "rect", ID: box.ID, Box: r,
	}
	if box.Reset {
		it.Tags = []string{"reset"}
	}
	if k, ok := series[box.Kind]; ok {
		// Pele draws the box in the background first, so that the wash hides what is under it.
		it.Parts = []scene.Part{{Box: r, Rounded: true, Paint: scene.Shade}, {Box: r, Rounded: true}}
		it.Style = style.Style{Fill: style.Series(k), FillOpacity: style.N(wash), Stroke: style.Series(k), Text: style.Fg}
	} else {
		it.Parts = []scene.Part{{Box: r, Rounded: true}}
		it.Style = style.Style{Fill: style.Bg, Stroke: style.Border, Text: style.Fg}
	}
	return it
}

// relation is the edge of a relation along a path, with an arrow at its tip pointing along
// (dx, dy).
func relation(r Relation, p *scene.Path, route []scene.Point, tip scene.Point, dx, dy float64) scene.Item {
	return scene.Item{
		Kind: scene.Edge, Role: RoleRelation, ID: r.Source.ID + "-" + r.Target.ID, Box: p.Bounds(),
		Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
		Route: append(route, tip),
		Ends:  [2]scene.Marker{{}, {Type: "arrow_point", At: tip, DX: dx, DY: dy}},
		Style: style.Style{Stroke: style.Line},
	}
}

// card is the node of a note or a specification in its box, its heading the label.
func card(kind, frame string, r scene.Rect, head *text.Label, at scene.Point) scene.Item {
	return scene.Item{
		Kind: scene.Node, Role: scene.Role("eventmodeling." + kind), Shape: "rect", ID: frame, Box: r,
		Parts: []scene.Part{{Box: r, Rounded: true}},
		Label: head, At: at,
		Style: style.Style{Fill: style.Bg, Stroke: style.Border, Text: style.Fg},
	}
}

// heading is the heading of a card on a frame, as Pele writes it: the kind of card, the
// frame's number, and the name of the frame of that number, if there is one.
func heading(kind, frame string, names map[string]string) string {
	s := kind + " · " + frame
	if name, ok := names[frame]; ok {
		s += " " + name
	}
	return s
}

// specRows are the rows of a specification, as Pele writes them.
func specRows(spec Specification) []string {
	rows := []string{"Given  " + strings.Join(spec.Given, ", ")}
	if len(spec.When) > 0 {
		rows = append(rows, "When  "+strings.Join(spec.When, ", "))
	}
	return append(rows, "Then  "+strings.Join(spec.Then, ", "))
}

// titleItem is the title, centered at the top of a scene width wide, pad under its top, or
// nothing for none.
func titleItem(title text.Label, width, pad float64) []scene.Item {
	if title.Empty() {
		return nil
	}
	return []scene.Item{{
		Kind: scene.Text, Role: RoleTitle, Label: &title,
		At:    scene.Point{X: width / 2, Y: pad + title.Height/2},
		Style: style.Style{Text: style.Fg},
	}}
}

// placed is a frame laid out across, Pele's Placed.
type placed struct {
	box        *Box
	name       text.Label
	data       []string
	x, y, w, h float64
}

// band is a lane laid out across, Pele's Band.
type band struct {
	lane        *Lane
	label       text.Label
	right, y, h float64
}

// draw lays the diagram out across, as Pele's draw does: time runs left to right, a frame in
// a new lane starts under the end of the one before it, so that the arrow between them can
// drop straight down, and a frame in the same lane follows on.
func draw(m *Model, o *layout.Options) *scene.Scene {
	size := o.Size()
	nameSize := jsmath.Round(size * 0.875)
	small := jsmath.Round(size * 0.8125)
	mono := jsmath.Round(size * 0.75)
	monoStep := jsmath.Round(mono * 1.4)
	smallStep := jsmath.Round(small * 1.5)
	measure := o.Measure()
	pad := o.Pad()
	widest := func(rows []string, size float64, face text.Face) float64 {
		width := 0.0
		for _, row := range rows {
			width = max(width, measure.Width(row, size, face))
		}
		return width
	}
	label := func(s string, size float64) text.Label {
		return text.Layout(s, text.Options{Measurer: measure, Size: size, MaxWidth: 4000, Face: text.Bold, Icons: o.Icons})
	}

	bands := make(map[*Lane]*band, len(m.Lanes))
	order := make([]*band, 0, len(m.Lanes))
	labelColumn := 0.0
	for _, lane := range m.Lanes {
		b := &band{lane: lane, label: label(lane.Label, small), right: math.Inf(-1)}
		labelColumn = max(labelColumn, b.label.Width)
		bands[lane] = b
		order = append(order, b)
	}
	labelColumn += 2 * lanePad

	items := make([]*placed, 0, len(m.Boxes))
	byBox := make(map[*Box]*placed, len(m.Boxes))
	var previous *placed
	for _, box := range m.Boxes {
		b := bands[box.Lane]
		name := label(box.Name, nameSize)
		var data []string
		if box.Data != "" {
			data = hardWrap(box.Data, dataColumns)
		}
		w := max(boxMinWidth, max(name.Width, widest(data, mono, text.Mono))+2*boxPadX)
		h := 2*boxPadY + name.Height
		if len(data) > 0 {
			h += 6 + float64(float64(len(data))*monoStep)
		}
		x := 0.0
		if previous != nil {
			staggered := previous.x + max(previous.w/2, previous.w-stagger)
			if previous.box.Lane == box.Lane {
				x = b.right + boxGap
			} else {
				x = max(staggered, b.right+boxGap)
			}
		}
		item := &placed{box: box, name: name, data: data, x: x, w: w, h: h}
		b.right = x + w
		b.h = max(b.h, h)
		items = append(items, item)
		byBox[box] = item
		previous = item
	}

	contentRight := 0.0
	for _, item := range items {
		contentRight = max(contentRight, item.x+item.w)
	}
	laneWidth := labelColumn + contentRight + lanePad

	title := label(m.Title, size)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	left := pad
	top := pad + titleHeight

	y := top
	var lanes []scene.Item
	for _, b := range order {
		b.y = y
		b.h += 2 * lanePad
		r := scene.Rect{X: left, Y: y, W: laneWidth, H: b.h}
		l := b.label
		lanes = append(lanes, scene.Item{
			Kind: scene.Group, Role: RoleLane, Shape: "rect", ID: b.lane.Label, Box: r,
			Parts: []scene.Part{{Box: r, Rounded: true, Paint: scene.FillOnly}},
			Label: &l, At: scene.Point{X: left + lanePad + l.Width/2, Y: y + b.h/2},
			Style: style.Style{Fill: style.Surface, Text: style.Muted},
		})
		y += b.h + laneGap
	}
	bottom := top
	if len(order) > 0 {
		bottom = y - laneGap
	}

	var nodes []scene.Item
	for _, item := range items {
		b := bands[item.box.Lane]
		item.x += left + labelColumn
		item.y = b.y + (b.h-item.h)/2
		node := frameItem(item.box, scene.Rect{X: item.x, Y: item.y, W: item.w, H: item.h})
		node.Label = &item.name
		node.At = scene.Point{X: item.x + item.w/2, Y: item.y + boxPadY + item.name.Height/2}
		nodes = append(nodes, node)
		nodes = append(nodes, rows(RoleData, rowsLabel(item.data, measure, mono, monoStep, text.Mono),
			item.x+boxPadX, item.y+boxPadY+item.name.Height+6)...)
	}

	var edges []scene.Item
	for _, r := range m.Relations {
		s, t := byBox[r.Source], byBox[r.Target]
		if s == nil || t == nil || s == t {
			continue
		}
		p := new(scene.Path)
		if s.box.Lane == t.box.Lane {
			// Along the lane, from the side of one box to the side of the other.
			forward := t.x >= s.x
			sx, tx := s.x, t.x+t.w
			if forward {
				sx, tx = s.x+s.w, t.x
			}
			sy := s.y + s.h/2
			ty := t.y + t.h/2
			dir := -1.0
			if tx >= sx {
				dir = 1
			}
			p.MoveTo(sx, sy).LineTo(tx-float64(dir*arrowTrim), ty)
			edges = append(edges, relation(r, p, []scene.Point{{X: sx, Y: sy}}, scene.Point{X: tx, Y: ty}, dir, 0))
			continue
		}
		// Between lanes, straight down or up where the boxes overlap, else in an S.
		down := t.y > s.y
		from := max(s.x, t.x)
		to := min(s.x+s.w, t.x+t.w)
		inset := func(w float64) float64 { return min(24, w/4) }
		var sx, tx float64
		switch {
		case to-from >= 16:
			sx = (from + to) / 2
			tx = sx
		case t.x > s.x:
			sx = s.x + s.w - inset(s.w)
			tx = t.x + inset(t.w)
		default:
			sx = s.x + inset(s.w)
			tx = t.x + t.w - inset(t.w)
		}
		sy, ty, dir := s.y, t.y+t.h, -1.0
		if down {
			sy, ty, dir = s.y+s.h, t.y, 1
		}
		end := ty - float64(dir*arrowTrim)
		mid := (sy + end) / 2
		p.MoveTo(sx, sy)
		route := []scene.Point{{X: sx, Y: sy}}
		if sx == tx {
			p.LineTo(sx, end)
		} else {
			p.CubeTo(sx, mid, tx, mid, tx, end)
			route = append(route, scene.Point{X: sx, Y: mid}, scene.Point{X: tx, Y: mid})
		}
		edges = append(edges, relation(r, p, route, scene.Point{X: tx, Y: ty}, 0, dir))
	}

	// Notes and given/when/then specifications, listed under the lanes with the frame they
	// belong to.
	names := make(map[string]string, len(m.Boxes))
	for _, box := range m.Boxes {
		names[box.ID] = box.Name
	}
	var cards []scene.Item
	cardsRight := 0.0
	addCard := func(kind, title, frame string, lines []string, textRole scene.Role) {
		head := label(heading(title, frame, names), small)
		w := max(head.Width, widest(lines, small, 0)) + 2*boxPadX
		h := 2*boxPadY + head.Height + float64(float64(len(lines))*smallStep)
		cy := bottom
		if len(cards) > 0 || len(order) != 0 {
			cy += cardGap
		}
		r := scene.Rect{X: left, Y: cy, W: w, H: h}
		cards = append(cards, card(kind, frame, r, &head, scene.Point{X: left + boxPadX + head.Width/2, Y: cy + boxPadY + head.Height/2}))
		cards = append(cards, rows(textRole, rowsLabel(lines, measure, small, smallStep, 0), left+boxPadX, cy+boxPadY+head.Height)...)
		bottom = cy + h
		cardsRight = max(cardsRight, w)
	}
	for _, note := range m.Notes {
		addCard("note", "Note", note.Frame, hardWrap(note.Text, noteColumns), RoleNoteText)
	}
	for _, spec := range m.Specifications {
		var lines []string
		for _, row := range specRows(spec) {
			lines = append(lines, hardWrap(row, noteColumns)...)
		}
		addCard("specification", "Given / When / Then", spec.Frame, lines, RoleSpecText)
	}

	width := max(0, cardsRight, title.Width)
	if len(order) > 0 {
		width = max(laneWidth, cardsRight, title.Width)
	}
	sc := &scene.Scene{
		Type:     "eventmodeling",
		Width:    math.Ceil(width + 2*pad),
		Height:   math.Ceil(bottom + pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	sc.Items = titleItem(title, sc.Width, pad)
	sc.Items = append(sc.Items, lanes...)
	sc.Items = append(sc.Items, edges...)
	sc.Items = append(sc.Items, nodes...)
	sc.Items = append(sc.Items, cards...)
	return sc
}

// cardPlace is a frame laid out down, as Pele's verticalEventModel places it.
type cardPlace struct {
	y, h, w float64
	index   int
	lane    float64 // the width of its lane's heading
}

// verticalEventModel lays the diagram out down, a card to a frame under its lane's heading,
// as Pele's verticalEventModel does. A relation between neighbouring cards drops straight
// down beside the lower card's heading; one that skips a card runs down a rail beside the
// cards. Relations into one card share a rail; others whose runs overlap take the next rail
// out, so that no two follow the same line.
func verticalEventModel(m *Model, o *layout.Options) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	mono := jsmath.Round(size * 0.75)
	pad := o.Pad()
	measure := o.Measure()
	room := max(160, o.MaxWidth-2*pad)

	order := make(map[*Box]int, len(m.Boxes))
	for i, box := range m.Boxes {
		order[box] = i
	}
	runs := map[*Box]*[2]int{}
	var runOrder []*Box
	// A relation between neighbouring cards passes the lower card's lane heading.
	passed := map[*Box]bool{}
	for _, r := range m.Relations {
		s, sok := order[r.Source]
		t, tok := order[r.Target]
		if !sok || !tok || s == t {
			continue
		}
		if s-t < 2 && t-s < 2 {
			passed[m.Boxes[max(s, t)]] = true
			continue
		}
		if run := runs[r.Target]; run != nil {
			run[0] = min(run[0], s)
			run[1] = max(run[1], s)
		} else {
			runs[r.Target] = &[2]int{min(s, t), max(s, t)}
			runOrder = append(runOrder, r.Target)
		}
	}
	slices.SortStableFunc(runOrder, func(a, b *Box) int { return runs[a][0] - runs[b][0] })
	railOf := map[*Box]int{}
	var railEnds []int
	for _, target := range runOrder {
		from, to := runs[target][0], runs[target][1]
		rail := slices.IndexFunc(railEnds, func(end int) bool { return end < from })
		if rail < 0 {
			rail = len(railEnds)
			railEnds = append(railEnds, 0)
		}
		railEnds[rail] = to
		railOf[target] = rail
	}
	gutter := 24 + float64(railGap*max(0, len(railEnds)-1))
	// Rails are never shared to save room: past what the width allows, the drawing is wider.
	cardWidth := max(cardMin, room-gutter)
	textRoom := cardWidth - 2*boxPadX
	label := func(s string, size, max float64) text.Label {
		return text.Layout(s, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: text.Bold, Icons: o.Icons})
	}
	title := label(m.Title, size, room)
	y := float64(pad)
	if title.Height != 0 {
		y += title.Height + 12
	}
	bottom := pad + title.Height
	width := max(title.Width, cardWidth+gutter)
	literal := func(raw string, size float64, face text.Face) text.Label {
		return common.LiteralLabel(text.DecodeEntities(raw), measure, size, textRoom, face)
	}

	var nodes, edges, cards []scene.Item
	places := make(map[*Box]*cardPlace, len(m.Boxes))
	for index, box := range m.Boxes {
		laneRoom := textRoom
		if passed[box] {
			laneRoom = textRoom - arrowRoom
		}
		lane := common.BreakLabel(label(box.Lane.Label, small, laneRoom), measure, laneRoom)
		name := label(box.Name, small, textRoom-36)
		var data *text.Label
		if box.Data != "" {
			l := literal(box.Data, mono, text.Mono)
			data = &l
		}
		dataWidth := 0.0
		if data != nil {
			dataWidth = data.Width
		}
		w := max(cardWidth, lane.Width+2*boxPadX, name.Width+2*boxPadX+36, dataWidth+2*boxPadX)
		headH := lane.Height + 12
		h := 2*boxPadY + name.Height
		if data != nil {
			h += data.Height + 6
		}
		cy := y + headH
		nodes = append(nodes, scene.Item{
			Kind: scene.Text, Role: RoleLaneLabel, ID: box.Lane.Label, Label: &lane, Align: scene.Start,
			At:    scene.Point{X: pad + boxPadX, Y: y + lane.Height/2},
			Style: style.Style{Text: style.Muted},
		})
		node := frameItem(box, scene.Rect{X: pad, Y: cy, W: w, H: h})
		node.Label = &name
		node.At = scene.Point{X: pad + boxPadX + name.Width/2, Y: cy + boxPadY + name.Height/2}
		nodes = append(nodes, node)
		// The frame's number, its baseline as far below the top of the box as the size of
		// the frame's name.
		id := common.Line(box.ID, measure.Width(box.ID, mono, 0), mono, 0)
		nodes = append(nodes, scene.Item{
			Kind: scene.Text, Role: RoleFrameID, ID: box.ID, Label: id, Align: scene.End,
			At:    scene.Point{X: pad + w - boxPadX, Y: cy + boxPadY + small - float64(mono*0.35)},
			Style: style.Style{Text: style.Muted},
		})
		if data != nil {
			nodes = append(nodes, rows(RoleData, *data, pad+boxPadX, cy+boxPadY+name.Height+6)...)
		}
		places[box] = &cardPlace{y: cy, h: h, w: w, index: index, lane: lane.Width}
		width = max(width, w+gutter)
		bottom = cy + h
		y = bottom + 28
	}
	for _, r := range m.Relations {
		s, t := places[r.Source], places[r.Target]
		if s == nil || t == nil || s == t {
			continue
		}
		down := t.y > s.y
		dir, sy, ty := -1.0, s.y, t.y+t.h
		if down {
			dir, sy, ty = 1, s.y+s.h, t.y
		}
		p := new(scene.Path)
		if s.index-t.index == 1 || t.index-s.index == 1 {
			// The lower card's lane heading sits in the gap, and wraps to leave the arrow room
			// beside it.
			lower := s
			if down {
				lower = t
			}
			cx := pad + max(min(s.w, t.w)/2, boxPadX+lower.lane+16)
			p.MoveTo(cx, sy).LineTo(cx, ty-float64(dir*arrowTrim))
			edges = append(edges, relation(r, p, []scene.Point{{X: cx, Y: sy}}, scene.Point{X: cx, Y: ty}, 0, dir))
			continue
		}
		rail := pad + width - gutter + 12 + float64(railGap*float64(railOf[r.Target]))
		// A card can send one relation and receive another: they leave above and arrive below
		// its middle.
		from, to := s.y+s.h/2-5, t.y+t.h/2+5
		p.MoveTo(pad+s.w, from).LineTo(rail, from).LineTo(rail, to).LineTo(pad+t.w+arrowTrim, to)
		route := []scene.Point{{X: pad + s.w, Y: from}, {X: rail, Y: from}, {X: rail, Y: to}}
		edges = append(edges, relation(r, p, route, scene.Point{X: pad + t.w, Y: to}, -1, 0))
	}
	names := make(map[string]string, len(m.Boxes))
	for _, box := range m.Boxes {
		names[box.ID] = box.Name
	}
	addCard := func(kind, title, frame, raw string, textRole scene.Role) {
		head := label(heading(title, frame, names), small, textRoom)
		body := literal(raw, small, 0)
		w := max(cardWidth, head.Width+2*boxPadX, body.Width+2*boxPadX)
		h := 2*boxPadY + head.Height + body.Height
		r := scene.Rect{X: pad, Y: y, W: w, H: h}
		cards = append(cards, card(kind, frame, r, &head, scene.Point{X: pad + boxPadX + head.Width/2, Y: y + boxPadY + head.Height/2}))
		cards = append(cards, rows(textRole, body, pad+boxPadX, y+boxPadY+head.Height)...)
		bottom = y + h
		y = bottom + cardGap
		width = max(width, w)
	}
	for _, note := range m.Notes {
		addCard("note", "Note", note.Frame, note.Text, RoleNoteText)
	}
	for _, spec := range m.Specifications {
		addCard("specification", "Given / When / Then", spec.Frame, strings.Join(specRows(spec), "\n"), RoleSpecText)
	}
	sc := &scene.Scene{
		Type:     "eventmodeling",
		Width:    math.Ceil(width + 2*pad),
		Height:   math.Ceil(bottom + pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	sc.Items = titleItem(title, sc.Width, pad)
	sc.Items = append(sc.Items, edges...)
	sc.Items = append(sc.Items, nodes...)
	sc.Items = append(sc.Items, cards...)
	return sc
}
