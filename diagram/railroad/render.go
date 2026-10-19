package railroad

import (
	"math"
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

// The scene's roles, for each rule in turn: "railroad.name" for the rule's name above its
// diagram, aligned at its start; "railroad.track" for the track, one path of lines and quarter
// turns; "railroad.arrows" for the arrows on the track where a long sequence folds and on the
// loops of repetitions; "railroad.start" and "railroad.end" for the dots at the track's ends;
// and, in the order Pele draws them, "railroad.terminal" for a terminal, a pill;
// "railroad.nonterminal" for a non-terminal, a box with the theme's corners, with the name it
// refers to as its ID; "railroad.special" for special text, a dashed box, with the question
// marks about its text as "railroad.special-mark" items of their own; and "railroad.count"
// for how often a loop repeats. "railroad.title" is the diagram's title, above the rules.
const (
	RoleTitle       scene.Role = "railroad.title"
	RoleName        scene.Role = "railroad.name"
	RoleTrack       scene.Role = "railroad.track"
	RoleArrows      scene.Role = "railroad.arrows"
	RoleStart       scene.Role = "railroad.start"
	RoleEnd         scene.Role = "railroad.end"
	RoleTerminal    scene.Role = "railroad.terminal"
	RoleNonterminal scene.Role = "railroad.nonterminal"
	RoleSpecial     scene.Role = "railroad.special"
	RoleSpecialMark scene.Role = "railroad.special-mark"
	RoleCount       scene.Role = "railroad.count"
)

// kind is the kind of a laid-out element, as Pele's Kind.
type kind uint8

const (
	kTerminal kind = iota
	kNonTerminal
	kSpecial
	kSequence
	kChoice
	kBypass
	kLoop
	kFolded
)

// box is a laid-out element, as Pele's Box. The track enters at the left edge and leaves at
// the right edge on one baseline, with up above it and down below it.
type box struct {
	kind            kind
	width, up, down float64
	kids            []*box
	// text is a leaf's text, or a loop's count; id is the name a non-terminal refers to.
	text, id string
	// reach is how far a bypass rises above the baseline, or a loop drops below it.
	reach float64
}

// metrics are Pele's Metrics: the measures of a diagram's elements. gap is the space between
// elements in a sequence, and sep between stacked branches.
type metrics struct {
	measure                         text.Measurer
	size, padding, height, gap, sep float64
	arc                             float64
}

const (
	lead      = 12
	headGap   = 6
	ruleGap   = 18
	countSize = 12
	arrow     = 7
)

// option is a number of the railroad config if it is at least 0, at most max, or fallback,
// as Pele's option reads it.
func option(config diagram.Config, key string, fallback, most float64) float64 {
	section, _ := config["railroad"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v >= 0 {
		return min(v, most)
	}
	return fallback
}

// shown is text as a diagram shows it, as Pele's shown makes it: literal, so that a terminal
// such as "<br>" or "*" is itself, not markup. Mermaid's #35; codes still stand for
// characters, and escapes that would be invisible are written out.
func shown(s string) string {
	if strings.Contains(s, "¶ß") {
		s = text.DecodeEntities(strings.ReplaceAll(s, "&", "&amp;"))
	}
	if !strings.ContainsAny(s, "\n\r\t") {
		return s
	}
	return strings.NewReplacer("\n", `\n`, "\r", `\r`, "\t", `\t`).Replace(s)
}

// reverse reverses list from an index on, as Pele's reverse does.
func reverse[T any](list []T, from int) {
	for i, j := from, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
}

// even is a value rounded up to an even number, as Pele's even.
func even(value float64) float64 { return math.Ceil(value/2) * 2 }

// spaces reports whether a text is spaces alone, as /^ +$/ tests it.
func spaces(s string) bool {
	return s != "" && strings.Trim(s, " ") == ""
}

func leaf(k kind, s, id string, face text.Face, m *metrics) *box {
	half := m.height / 2
	width := max(even(m.measure.Width(s, m.size, face)+2*m.padding), m.height)
	return &box{kind: k, width: width, up: half, down: half, text: s, id: id}
}

// count is what a loop says of how often it repeats, as Pele's count writes it, or nothing
// when that is once or more, or not at all.
func count(least, most float64) string {
	if math.IsInf(most, 1) && least <= 1 {
		return ""
	}
	written := func(value float64) string {
		if math.IsInf(value, 0) || math.IsNaN(value) {
			return "∞"
		}
		return chars.NumberString(value)
	}
	switch {
	case least == most:
		return written(least) + "×"
	case math.IsInf(most, 1):
		return "≥" + written(least) + "×"
	case least == 0:
		return "≤" + written(most) + "×"
	}
	return written(least) + "–" + written(most) + "×"
}

// below is where the next branch of a choice sits below the baseline, given the one above
// it, as Pele's below gives it.
func below(offset float64, above, branch *box, m *metrics) float64 {
	return max(offset+above.down+m.sep+branch.up, 2*m.arc)
}

func bypass(kid *box, m *metrics) *box {
	reach := max(kid.up+m.sep, 2*m.arc)
	return &box{kind: kBypass, width: kid.width + 4*m.arc, up: reach, down: kid.down, kids: []*box{kid}, reach: reach}
}

// row is a sequence of boxes, measured.
func row(kids []*box, width float64) *box {
	up, down := 0.0, 0.0
	for _, kid := range kids {
		up = max(up, kid.up)
		down = max(down, kid.down)
	}
	return &box{kind: kSequence, width: width, up: up, down: down, kids: kids}
}

// measure lays out a definition as boxes, as Pele's measure does. With room, a sequence wider
// than it has folds into rows; its parts are sized for the rows inside its rails.
func measure(root *Node, m *metrics, room float64) *box {
	budgets := map[*Node]float64{}
	type budget struct {
		node  *Node
		width float64
	}
	pending := []budget{{root, room}}
	for i := 0; !math.IsInf(room, 1) && i < len(pending); i++ {
		node, width := pending[i].node, pending[i].width
		budgets[node] = width
		around := 0.0
		switch node.Type {
		case Choice, Optional:
			around = 4 * m.arc
		case Repetition:
			if node.Min == 0 {
				around = float64(6 * m.arc)
			} else {
				around = 2 * m.arc
			}
		case Sequence:
			around = float64(6 * m.arc)
		}
		for _, child := range node.parts() {
			pending = append(pending, budget{child, max(40, width-around)})
		}
	}
	return fold(root, (*Node).parts, func(node *Node, kids []*box) *box {
		room, ok := budgets[node]
		if !ok {
			room = math.Inf(1)
		}
		switch node.Type {
		case Terminal:
			s := shown(node.Text)
			if spaces(s) {
				s = strings.Repeat("␣", len(s))
			}
			return leaf(kTerminal, s, "", text.Mono, m)
		case Nonterminal:
			return leaf(kNonTerminal, shown(node.Text), node.Text, 0, m)
		case Special:
			return leaf(kSpecial, "? "+shown(node.Text)+" ?", "", text.Italic, m)
		case Sequence:
			width := float64(float64(max(0, len(kids)-1)) * m.gap)
			for _, kid := range kids {
				width += kid.width
			}
			whole := row(kids, width)
			if width <= room || len(kids) < 2 {
				return whole
			}
			var rows []*box
			var current []*box
			rowW := 0.0
			flush := func() {
				rows = append(rows, row(current, rowW))
				current = nil
				rowW = 0
			}
			for _, kid := range kids {
				if len(current) > 0 && rowW+m.gap+kid.width > max(40, room-float64(6*m.arc)) {
					flush()
				}
				gap := 0.0
				if len(current) > 0 {
					gap = m.gap
				}
				rowW += gap + kid.width
				current = append(current, kid)
			}
			flush()
			if len(rows) < 2 {
				return whole
			}
			foldedWidth, offset := 0.0, 0.0
			for i := range rows {
				foldedWidth = max(foldedWidth, rows[i].width)
				if i > 0 {
					offset += rows[i-1].down + max(m.sep, 2*m.arc) + rows[i].up
				}
			}
			return &box{kind: kFolded, width: foldedWidth + float64(6*m.arc), up: rows[0].up,
				down: offset + rows[len(rows)-1].down, kids: rows}
		case Choice:
			inner, offset := 0.0, 0.0
			for i := range kids {
				inner = max(inner, kids[i].width)
				if i > 0 {
					offset = below(offset, kids[i-1], kids[i], m)
				}
			}
			b := &box{kind: kChoice, width: inner + 4*m.arc, kids: kids}
			if len(kids) > 0 {
				b.up = kids[0].up
				b.down = offset + kids[len(kids)-1].down
			}
			return b
		case Optional:
			return bypass(kids[0], m)
		}
		least, most := node.Min, node.Max
		kid := kids[0]
		if least == 1 && most == 1 {
			return kid
		}
		s := count(least, most)
		reach := max(kid.down+m.sep, 2*m.arc)
		label := 0.0
		if s != "" {
			label = even(m.measure.Width(s, countSize, 0))
		}
		extra := float64(countSize + 6)
		if s == "" {
			extra = arrow/2.0 + 1
		}
		loop := &box{kind: kLoop, width: max(kid.width, label) + 2*m.arc, up: kid.up, down: reach + extra,
			kids: []*box{kid}, text: s, reach: reach}
		if least == 0 {
			return bypass(loop, m)
		}
		return loop
	})
}

// Layout lays a railroad diagram out as a scene, as Pele's renderRailroad does: a diagram for
// each rule, one under another, and tighter, with sequences folded into rows, when that
// does not fit the width the host has.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.Tighten(o, func(tight bool) *scene.Scene { return draw(m, config, o, tight) })
}

// off is a number of Pele's inside the group of its rules, where the scene has it: Pele moves
// the group by half a unit each way, so that lines one unit wide fall on whole pixels.
func off(v float64) float64 { return v + 0.5 }

// pen draws the track as Pele writes its path, on a path in the scene's places: its places
// moved as off moves them, and its relative arcs and lines taken from where the path is.
// Each figure is drawn on a path of its own, which ArcTo reads where it is from, and the
// figures are joined when the next starts, so that a long track is drawn in time in
// proportion to it.
type pen struct {
	whole, p *scene.Path
	x, y     float64
}

func newPen() *pen { return &pen{whole: new(scene.Path), p: new(scene.Path)} }

// path is the whole path drawn.
func (t *pen) path() *scene.Path {
	t.whole.Ops = append(t.whole.Ops, t.p.Ops...)
	t.whole.Args = append(t.whole.Args, t.p.Args...)
	t.p = new(scene.Path)
	return t.whole
}

func (t *pen) move(x, y float64) {
	t.path()
	t.x, t.y = off(x), off(y)
	t.p.MoveTo(t.x, t.y)
}

func (t *pen) across(x float64) {
	t.x = off(x)
	t.p.LineTo(t.x, t.y)
}

func (t *pen) down(y float64) {
	t.y = off(y)
	t.p.LineTo(t.x, t.y)
}

// line is a line by dx, dy from where the path is, as l is.
func (t *pen) line(dx, dy float64) {
	t.x, t.y = t.x+dx, t.y+dy
	t.p.LineTo(t.x, t.y)
}

// turn is a quarter of a circle of radius r by dx, dy from where the path is, as Pele's turn
// writes one with a: clockwise when sweep is set.
func (t *pen) turn(r float64, sweep bool, dx, dy float64) {
	t.x, t.y = t.x+dx, t.y+dy
	t.p.ArcTo(r, r, 0, false, sweep, t.x, t.y)
}

// line is a label of one line of text, measured.
func line(s string, face text.Face, size float64, m text.Measurer) *text.Label {
	if s == "" {
		return nil
	}
	return common.Line(s, m.Width(s, size, face), size, face)
}

// draw lays the diagram out, with room to spare or tightly, as Pele's draw does.
func draw(model *Model, config diagram.Config, o *layout.Options, tight bool) *scene.Scene {
	base := o.Size()
	fontSize := option(config, "fontSize", 0, 200)
	size := fontSize
	if size == 0 {
		size = jsmath.Round(base * 0.875)
	}
	headSize := fontSize
	if headSize == 0 {
		headSize = base
	}
	padding := option(config, "padding", -1, 200)
	lineHeight := jsmath.Round(size * 1.5)
	m := &metrics{
		measure: o.Measure(),
		size:    size,
		padding: padding,
		sep:     option(config, "verticalSeparation", 10, 500),
		gap:     option(config, "horizontalSeparation", 16, 500),
		arc:     option(config, "arcRadius", 10, 200),
	}
	if padding < 0 {
		m.padding = 12
		m.height = even(lineHeight + 9)
	} else {
		m.height = even(lineHeight + 2*padding)
	}
	if tight {
		m.gap = min(m.gap, 12)
		m.arc = min(m.arc, 6)
	}
	section, _ := config["railroad"].(diagram.Config)
	radius := 0.0
	if section["showMarkers"] != false {
		radius = option(config, "markerRadius", 4, 50)
	}
	stroke := style.Num{}
	if w := option(config, "strokeWidth", 0, 20); w > 0 {
		stroke = style.N(w)
	}
	pad := o.Pad()
	arc := m.arc

	room := 4000.0
	if tight {
		room = common.TitleRoom(o)
	}
	title := text.Layout(model.Title, text.Options{Measurer: m.measure, Size: base, MaxWidth: room, Face: text.Bold})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 16
	}
	headHeight := jsmath.Round(headSize * 1.5)
	// The size of a count, in a variable, so that its product is taken in float64 as
	// JavaScript takes it.
	small := float64(countSize)
	countShift := float64(small * 0.35)
	boxTop := m.height / 2

	var items []scene.Item
	var track, arrows *pen
	var nodes []scene.Item

	// line runs the track straight across, where there is room for it.
	straight := func(x1, y, x2 float64) {
		if x2 > x1 {
			track.move(x1, y)
			track.across(x2)
		}
	}
	eastToSouth := func() { track.turn(arc, true, arc, arc) }
	southToEast := func() { track.turn(arc, false, arc, arc) }
	eastToNorth := func() { track.turn(arc, false, arc, -arc) }
	northToEast := func() { track.turn(arc, true, arc, -arc) }
	loopDown := func() { track.turn(arc, false, -arc, arc) }
	loopUp := func() { track.turn(arc, false, -arc, -arc) }

	place := func(root *box, left, baseline float64) {
		boxes := []*box{root}
		at := []float64{left, baseline}
		for len(boxes) > 0 {
			b := boxes[len(boxes)-1]
			boxes = boxes[:len(boxes)-1]
			y := at[len(at)-1]
			x := at[len(at)-2]
			at = at[:len(at)-2]
			kids := b.kids
			right := x + b.width
			switch b.kind {
			case kSequence:
				kx := right
				for i := len(kids) - 1; i >= 0; i-- {
					kx -= kids[i].width
					boxes = append(boxes, kids[i])
					at = append(at, kx, y)
					if i > 0 {
						straight(kx-m.gap, y, kx)
					}
					kx -= m.gap
				}
			case kFolded:
				start := x + 2*arc
				from := len(boxes)
				offset := 0.0
				straight(x, y, start)
				for i, kid := range kids {
					ky := y + offset
					boxes = append(boxes, kid)
					at = append(at, ky, start)
					if i+1 < len(kids) {
						next := kids[i+1]
						nextOffset := offset + kid.down + max(m.sep, 2*arc) + next.up
						mid := y + offset + kid.down + max(m.sep, 2*arc)/2
						track.move(start+kid.width, ky)
						track.across(right - 2*arc)
						track.down(mid)
						track.across(x + arc)
						track.down(y + nextOffset)
						track.across(start)
						arrows.move(start-arrow, y+nextOffset-arrow/2.0)
						arrows.line(arrow, arrow/2.0)
						arrows.line(-arrow, arrow/2.0)
						arrows.p.Close()
						offset = nextOffset
					} else {
						track.move(start+kid.width, ky)
						track.across(right - arc)
						track.down(y)
						track.across(right)
					}
				}
				reverse(boxes, from)
				reverse(at, 2*from)
			case kChoice:
				inner := b.width - 4*arc
				railIn := x + arc
				railOut := right - arc
				from := len(boxes)
				offset := 0.0
				for i, kid := range kids {
					if i > 0 {
						offset = below(offset, kids[i-1], kid, m)
					}
					ky := y + offset
					kx := x + 2*arc + (inner-kid.width)/2
					last := i == len(kids)-1
					if i == 0 {
						straight(x, y, kx)
						straight(kx+kid.width, y, right)
					} else {
						if last {
							track.move(x, y)
							eastToSouth()
							track.down(ky - arc)
						} else {
							track.move(railIn, ky-arc)
						}
						southToEast()
						if kx > railIn+arc {
							track.across(kx)
						}
						track.move(kx+kid.width, ky)
						if kx > railIn+arc {
							track.across(railOut - arc)
						}
						eastToNorth()
						if last {
							track.down(y + arc)
							northToEast()
						}
					}
					boxes = append(boxes, kid)
					at = append(at, ky, kx)
				}
				// Reversed, so that the branches come off the stack top to bottom.
				reverse(boxes, from)
				reverse(at, 2*from)
			case kBypass:
				kid := kids[0]
				kx := x + 2*arc
				straight(x, y, kx)
				straight(kx+kid.width, y, right)
				track.move(x, y)
				eastToNorth()
				track.down(y - b.reach + arc)
				northToEast()
				track.across(right - 2*arc)
				eastToSouth()
				track.down(y - arc)
				southToEast()
				boxes = append(boxes, kid)
				at = append(at, kx, y)
			case kLoop:
				kid := kids[0]
				kx := x + (b.width-kid.width)/2
				loopY := y + b.reach
				middle := x + b.width/2
				straight(x, y, kx)
				straight(kx+kid.width, y, right)
				track.move(x+arc, y)
				loopDown()
				track.down(loopY - arc)
				southToEast()
				track.across(right - arc)
				eastToNorth()
				track.down(y + arc)
				loopUp()
				arrows.move(middle-arrow/2.0, loopY)
				arrows.line(arrow, -arrow/2.0)
				arrows.line(0, arrow)
				arrows.p.Close()
				if b.text != "" {
					nodes = append(nodes, scene.Item{
						Kind: scene.Text, Role: RoleCount, Label: line(b.text, 0, countSize, m.measure),
						// The count's baseline is 3 below its size under the loop.
						At:    scene.Point{X: off(middle), Y: off(loopY + countSize + 3 - countShift)},
						Style: style.Style{Text: style.Muted},
					})
				}
				boxes = append(boxes, kid)
				at = append(at, kx, y)
			default:
				r := scene.Rect{X: off(x), Y: off(y - boxTop), W: b.width, H: m.height}
				it := scene.Item{
					Kind: scene.Node, Box: r,
					Parts: []scene.Part{{Box: r, Rounded: true}},
					At:    scene.Point{X: off(x + b.width/2), Y: off(y)},
					Style: style.Style{Fill: style.Bg, Stroke: style.Border, Text: style.Fg, StrokeWidth: stroke},
				}
				switch b.kind {
				case kTerminal:
					it.Role, it.Shape = RoleTerminal, "stadium"
					it.Style.Fill = style.Surface
					it.Style.Radius = style.N(boxTop)
					it.Label = line(b.text, text.Mono, m.size, m.measure)
					nodes = append(nodes, it)
				case kNonTerminal:
					it.Role, it.Shape, it.ID = RoleNonterminal, "rounded", b.id
					it.Label = line(b.text, 0, m.size, m.measure)
					nodes = append(nodes, it)
				default:
					// Pele writes the question marks about special text in spans of the
					// muted color, in one text centered on the box. A label is of one color,
					// so the marks are items of their own: the first starts where the whole
					// starts, and the last ends where it ends.
					it.Role, it.Shape = RoleSpecial, "rounded"
					it.Style.Dash = style.Dashes(4, 3)
					it.Label = line(b.text[2:len(b.text)-2], text.Italic, m.size, m.measure)
					half := m.measure.Width(b.text, m.size, text.Italic) / 2
					mark := func(s string, x float64, align scene.Align) scene.Item {
						return scene.Item{
							Kind: scene.Text, Role: RoleSpecialMark, Label: line(s, text.Italic, m.size, m.measure),
							At: scene.Point{X: x, Y: it.At.Y}, Align: align, Style: style.Style{Text: style.Muted},
						}
					}
					nodes = append(nodes, it, mark("? ", it.At.X-half, scene.Start), mark(" ?", it.At.X+half, scene.End))
				}
			}
		}
	}

	width := title.Width
	top := pad + titleHeight
	bottom := pad + title.Height
	for _, rule := range model.Rules {
		room := math.Inf(1)
		if tight {
			room = max(80, o.MaxWidth-2*pad-4*radius-2*lead-1)
		}
		b := measure(rule.Definition, m, room)
		name := shown(rule.Name)
		baseline := top + headHeight + headGap + b.up
		start := pad + 2*radius
		left := start + lead
		end := left + b.width + lead
		track = newPen()
		arrows = newPen()
		nodes = nil
		straight(start, baseline, left)
		straight(left+b.width, baseline, end)
		place(b, left, baseline)

		items = append(items, scene.Item{
			Kind: scene.Text, Role: RoleName, ID: rule.Name, Label: line(name, text.Bold, headSize, m.measure),
			At: scene.Point{X: off(pad), Y: off(top + headHeight/2)}, Align: scene.Start,
			Style: style.Style{Text: style.Fg},
		})
		trackPath, arrowsPath := track.path(), arrows.path()
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: RoleTrack, ID: rule.Name, Box: trackPath.Bounds(),
			Parts: []scene.Part{{Path: trackPath, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.Line, StrokeWidth: stroke},
		})
		if len(arrowsPath.Ops) > 0 {
			items = append(items, scene.Item{
				Kind: scene.Mark, Role: RoleArrows, ID: rule.Name, Box: arrowsPath.Bounds(),
				Parts: []scene.Part{{Path: arrowsPath, Paint: scene.FillOnly}},
				Style: style.Style{Fill: style.Line},
			})
		}
		if radius > 0 {
			dot := func(role scene.Role, cx float64) scene.Item {
				x, y := off(cx), off(baseline)
				return scene.Item{
					Kind: scene.Mark, Role: role, Shape: "circle", ID: rule.Name,
					Box:   scene.Rect{X: x - radius, Y: y - radius, W: 2 * radius, H: 2 * radius},
					Parts: []scene.Part{{Path: new(scene.Path).Ellipse(x, y, radius, radius), Paint: scene.FillOnly}},
					Style: style.Style{Fill: style.Line},
				}
			}
			items = append(items, dot(RoleStart, pad+radius), dot(RoleEnd, end+radius))
		}
		items = append(items, nodes...)
		width = max(width, end+2*radius-pad, m.measure.Width(name, headSize, text.Bold))
		bottom = baseline + b.down
		top = bottom + m.sep + ruleGap
	}

	sc := &scene.Scene{
		Type:     "railroad",
		Width:    math.Ceil(width + 2*pad),
		Height:   math.Ceil(bottom + pad),
		FontSize: base,
		Title:    model.AccTitle,
		Desc:     model.AccDescr,
	}
	if !title.Empty() {
		items = append([]scene.Item{{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: sc.Width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		}}, items...)
	}
	sc.Items = items
	return sc
}
