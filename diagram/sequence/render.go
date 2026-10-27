package sequence

import (
	"math"
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/diagram/internal/css"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The heads of messages, Pele's Head.
const (
	headNone = iota
	headArrow
	headCross
	headOpen
	headHalfSolid
	headHalfStick
)

// lineStyle is how a message is drawn, Pele's LineStyle: whether it is dotted, its head, and
// where the heads are: at the receiver, at the sender only (the reverse arrows), or at both.
type lineStyle struct {
	dotted         bool
	head           int
	atStart, atEnd bool
	bottom         bool
}

// lineStyles are the messages that are drawn as lines, by type, Pele's STYLES.
var lineStyles = func() map[int]lineStyle {
	out := map[int]lineStyle{}
	// set is Pele's style: it records how messages of a type are drawn.
	set := func(typ int, dotted bool, head, ends int, bottom bool) {
		out[typ] = lineStyle{dotted, head, ends != 1, ends != 0, bottom}
	}
	set(Solid, false, headArrow, 1, false)
	set(Dotted, true, headArrow, 1, false)
	set(SolidCross, false, headCross, 1, false)
	set(DottedCross, true, headCross, 1, false)
	set(SolidOpen, false, headNone, 1, false)
	set(DottedOpen, true, headNone, 1, false)
	set(SolidPoint, false, headOpen, 1, false)
	set(DottedPoint, true, headOpen, 1, false)
	set(BidirectionalSolid, false, headArrow, 2, false)
	set(BidirectionalDotted, true, headArrow, 2, false)
	for _, dotted := range []bool{false, true} {
		d := 0
		if dotted {
			d = 10
		}
		set(SolidTop+d, dotted, headHalfSolid, 1, false)
		set(SolidBottom+d, dotted, headHalfSolid, 1, true)
		set(StickTop+d, dotted, headHalfStick, 1, false)
		set(StickBottom+d, dotted, headHalfStick, 1, true)
		set(SolidArrowTopReverse+d, dotted, headHalfSolid, 0, false)
		set(SolidArrowBottomReverse+d, dotted, headHalfSolid, 0, true)
		set(StickArrowTopReverse+d, dotted, headHalfStick, 0, false)
		set(StickArrowBottomReverse+d, dotted, headHalfStick, 0, true)
	}
	return out
}()

// frameKinds are the kinds of block, by the type of the message that starts one.
var frameKinds = map[int]string{
	LoopStart:     "loop",
	AltStart:      "alt",
	OptStart:      "opt",
	ParStart:      "par",
	ParOverStart:  "par",
	CriticalStart: "critical",
	BreakStart:    "break",
	RectStart:     "rect",
}

// The measures of a sequence diagram, as Pele's render.ts has them.
const (
	msgPadding   = 12
	selfW        = 28
	selfH        = 22
	actW         = 10
	actStep      = 4
	actMin       = 14
	noteGap      = 10
	noteOverhang = 16
	framePadding = 20
	frameNesting = 10
	tabH         = 20
	frameText    = 240
	badgeH       = 16
	centralR     = 4
	arrowLength  = scene.MarkerSize
)

// isLineMessage reports whether a message of a type is drawn as a line. It is Pele's
// isLineMessage.
func isLineMessage(typ int) bool {
	_, ok := lineStyles[typ]
	return ok
}

// numbering is the numbers of the messages, Pele's Numbering: the number each line carries,
// whether or not it is shown, NaN for everything else.
type numbering struct {
	index   []float64
	shown   []bool
	visible bool
}

// numberMessages counts message lines as Mermaid's renderer does while it draws, as Pele's
// numberMessages does.
func numberMessages(messages []*Message, always bool) numbering {
	n := numbering{index: make([]float64, len(messages)), shown: make([]bool, len(messages))}
	current, step := 1.0, 1.0
	for i, m := range messages {
		n.index[i] = math.NaN()
		switch {
		case m.Type == Autonumber:
			if m.Numbering.Given && m.Numbering.Start != 0 {
				current = m.Numbering.Start
			}
			if m.Numbering.Given && m.Numbering.Step != 0 {
				step = m.Numbering.Step
			}
			n.visible = m.Numbering.Visible
		case isLineMessage(m.Type):
			n.index[i] = current
			n.shown[i] = n.visible || always
			current = jsmath.Round(float64((current+step)*100)) / 100
		}
	}
	return n
}

// column is a participant as it is laid out, Pele's Column.
type column struct {
	id          string
	actor       *Actor
	kind        string
	label       text.Label
	w, h        float64
	x           float64
	createAt    int
	destroyAt   int
	top         float64
	lifeStart   float64
	lifeEnd     float64
	activations []float64
}

// The kinds of view, Pele's V.
const (
	vLine = iota
	vSelf
	vNote
)

// view is a message or a note as it is laid out, Pele's View.
type view struct {
	kind  int
	a, b  int
	label text.Label
	x, w  float64
}

// seqFrame is a block as it is laid out, Pele's Frame.
type seqFrame struct {
	kind          string
	x0, x1        float64
	top           float64
	title         text.Label
	tabW          float64
	fill          string
	dividers      []int
	dividerY      []float64
	dividerLabels []text.Label
}

// boxRun is a run of participants side by side in one box, Pele's BoxRun.
type boxRun struct {
	box         *Box
	first, last int
	title       text.Label
	grow        float64
}

// option is a number of the sequence section of the config from 0 to 2000, or fallback, as
// Pele's option reads it.
func option(conf diagram.Config, key string, fallback float64) float64 {
	if v, ok := conf[key].(float64); ok && v >= 0 && v <= 2000 {
		return v
	}
	return fallback
}

// headMarker is the marker of a head whose tip is at x, y and points along dir across, as
// Pele's head draws it; up is the side a half head sits on, down the page for 1.
func headMarker(head int, x, y, dir, up float64) scene.Marker {
	m := scene.Marker{At: scene.Point{X: x, Y: y}, DX: dir}
	// The side of a half head as seen along its direction: left of it is down the page for
	// a head pointing right.
	left := up*dir > 0
	switch head {
	case headArrow:
		m.Type = "arrow_point"
	case headCross:
		m.Type = "arrow_cross"
	case headOpen:
		m.Type = "sequence.open"
	case headHalfSolid:
		m.Type = "sequence.half_right"
		if left {
			m.Type = "sequence.half_left"
		}
	case headHalfStick:
		m.Type = "sequence.stick_right"
		if left {
			m.Type = "sequence.stick_left"
		}
	}
	return m
}

// trim is how far a line stops short of its head's tip, Pele's trim.
func trim(head int) float64 {
	switch head {
	case headArrow:
		return arrowLength - 1
	case headCross:
		return 4
	}
	return 0
}

// writesFill reports whether Pele's resolveStyle keeps a fill of the value: what is left of
// it without !important and white space about it is not empty, and nothing in it could load
// or run something.
func writesFill(value string) bool {
	var b strings.Builder
	for i := 0; i < len(value); i++ {
		if i+10 <= len(value) && strings.EqualFold(value[i:i+10], "!important") {
			i += 9
			continue
		}
		b.WriteByte(value[i])
	}
	v := chars.Trim(b.String())
	if v == "" || strings.ContainsAny(v, "<>{};\\") {
		return false
	}
	lower := strings.Map(func(r rune) rune {
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return r
	}, v)
	for _, bad := range []string{"javascript:", "@import", "image-set", "/*"} {
		if strings.Contains(lower, bad) {
			return false
		}
	}
	for _, fn := range []string{"url", "expression"} {
		for i := strings.Index(lower, fn); i >= 0; {
			if j := chars.SpaceRun(lower, i+len(fn)); j < len(lower) && lower[j] == '(' {
				return false
			}
			next := strings.Index(lower[i+1:], fn)
			if next < 0 {
				break
			}
			i += 1 + next
		}
	}
	return true
}

// fillStyle is how a box or a shaded block whose author gave it a fill is painted: in the
// theme's alternate surface, at half its opacity where no fill is written, and the author's
// fill over it. Pele writes these as the fill-opacity and the resolveStyle of a fill in its
// draw, for the boxes and the rect blocks alike.
func fillStyle(fill string, written bool) (style.Style, style.Style) {
	if !written {
		return style.Style{Fill: style.SurfaceAlt, FillOpacity: style.N(0.5)}, style.Style{}
	}
	return style.Style{Fill: style.SurfaceAlt}, css.Declarations([]string{"fill:" + fill}).Style
}

// Layout lays a sequence diagram out as a scene, as Pele's renderSequence draws it: loosely,
// and tightly when that does not fit the width the host has and is narrower.
func Layout(db *DB, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.Tighten(o, func(tight bool) *scene.Scene { return lay(db, config, o, tight, 0, true) })
}

// monoMeasurer measures text in the monospace face whatever face it is asked of, as the
// messageMeasurer of Pele's draw does where monospace lists messages.
type monoMeasurer struct{ text.Measurer }

// Width is how wide the measurer it wraps measures s in the monospace face with f.
func (m monoMeasurer) Width(s string, size float64, f text.Face) float64 {
	return m.Measurer.Width(s, size, f|text.Mono)
}

// monoLabel is a label measured by a monoMeasurer with each of its spans in the monospace
// face, so that it is drawn in the face it was measured in. Pele draws the labels of messages
// in a group that sets the monospace font instead.
func monoLabel(label text.Label) text.Label {
	lines := make([][]text.Span, len(label.Lines))
	for i, line := range label.Lines {
		lines[i] = slices.Clone(line)
		for k := range lines[i] {
			lines[i][k].Face |= text.Mono
		}
	}
	label.Lines = lines
	return label
}

// lay lays the diagram out loosely, or tightly when tight is true, as Pele's draw does: the
// participants in a row with their lifelines, the messages, notes and blocks down the page in
// their order, the boxes about the participants they hold, and the participants again at the
// foot when mirrorActors is not false. Laid out tightly in a host of a set width, it widens the
// gaps between lifelines into the width to spare where that wraps messages into fewer lines.
// hold is width it keeps from that spare width, and byNeed lets it share the spare width out
// by what each gap needs rather than evenly. When what it widened comes out too wide, it lays
// the diagram out again with other values of them.
func lay(db *DB, config diagram.Config, o *layout.Options, tight bool, hold float64, byNeed bool) *scene.Scene {
	conf, _ := config["sequence"].(diagram.Config)
	size := o.Size()
	small := jsmath.Round(float64(size * 0.875))
	tiny := jsmath.Round(float64(size * 0.75))
	measure := o.Measure()
	monospaced := o.Monospace&layout.Messages != 0
	var messageMeasure text.Measurer = measure
	if monospaced {
		messageMeasure = monoMeasurer{measure}
	}
	pad := o.Pad()
	inf := math.Inf(1)
	mirror := conf["mirrorActors"] != false
	rightAngles := conf["rightAngles"] == true
	actorMargin := min(option(conf, "actorMargin", 40), choose(tight, 8, inf))
	minW := min(option(conf, "width", 64), choose(tight, 48, inf))
	minH := option(conf, "height", jsmath.Round(float64(size*1.5))+20)
	wrapW := option(conf, "width", 160)
	if tight {
		maxWidth := 320.0
		if o.MaxWidth > 0 {
			maxWidth = o.MaxWidth
		}
		wrapW = min(wrapW, max(32, (maxWidth-2*pad)/max(1, float64(len(db.Actors)))-actorMargin-24))
	}
	// Mermaid's default of 35 corresponds to Pele's default gap; other values move it by
	// the difference.
	msgGap := max(4, 14+option(conf, "messageMargin", 35)-35)
	notePad := min(option(conf, "noteMargin", 10), choose(tight, 6, inf))
	boxPad := min(option(conf, "boxMargin", 10), choose(tight, 6, inf))
	align := 0
	switch conf["messageAlign"] {
	case "left":
		align = -1
	case "right":
		align = 1
	}
	msgPad := choose(tight, 6, msgPadding)
	noteOver := choose(tight, 8, noteOverhang)
	framePad := choose(tight, 12, framePadding)
	frameNest := choose(tight, 6, frameNesting)
	messages := db.Messages
	count := len(messages)
	numbers := numberMessages(messages, conf["showSequenceNumbers"] == true)
	// textOf lays a text out at a size, at most width wide where it wraps or the layout is tight,
	// measured by by, as the text of Pele's draw does.
	textOf := func(raw string, fontSize float64, wrap bool, width float64, face text.Face, by text.Measurer) text.Label {
		limit := inf
		if tight || wrap {
			limit = width
		}
		label := text.Layout(raw, text.Options{Measurer: by, Size: fontSize, MaxWidth: limit, Face: face, Icons: o.Icons})
		if tight {
			return common.BreakLabel(label, by, max(120, width))
		}
		return label
	}
	// labelOf is textOf at the wrapping width in the regular face, the defaults of the text of
	// Pele's draw.
	labelOf := func(raw string, fontSize float64, wrap bool) text.Label {
		return textOf(raw, fontSize, wrap, wrapW, 0, measure)
	}
	// messageText lays the text of a message out at most width wide where it wraps, measured in
	// the monospace face where monospace lists messages, as the messageText of Pele's draw does.
	messageText := func(raw string, wrap bool, width float64) text.Label {
		label := textOf(raw, small, wrap, width, 0, messageMeasure)
		if monospaced {
			return monoLabel(label)
		}
		return label
	}

	var used map[string]bool
	if conf["hideUnusedParticipants"] == true {
		used = map[string]bool{}
		for _, m := range messages {
			used[m.From], used[m.To] = true, true
		}
	}

	nextLine := make([]int, count+1)
	nextLine[count] = -1
	for i := count - 1; i >= 0; i-- {
		nextLine[i] = nextLine[i+1]
		if isLineMessage(messages[i].Type) {
			nextLine[i] = i
		}
	}

	// positionOf is the index of the message at which a participant is created or destroyed, as
	// Map.get on Pele's createdActors and destroyedActors gives it in its draw.
	positionOf := func(list []Position, id string) (int, bool) {
		for _, p := range list {
			if p.ID == id {
				return p.Index, true
			}
		}
		return 0, false
	}
	var cols []*column
	colOf := map[string]int{}
	for _, actor := range db.Actors {
		id := actor.ID
		if used != nil && !used[id] {
			continue
		}
		kind := actorKind(actor.Type)
		label := labelOf(actor.Description, size, actor.Wrap)
		createAt, destroyAt := -1, -1
		if created, ok := positionOf(db.Created, id); ok {
			createAt = nextLine[min(created, count)]
		}
		if createAt >= 0 && messages[createAt].To != id {
			createAt = -1
		}
		if destroyed, ok := positionOf(db.Destroyed, id); ok {
			destroyAt = nextLine[min(destroyed, count)]
		}
		if destroyAt >= 0 && messages[destroyAt].To != id && messages[destroyAt].From != id {
			destroyAt = -1
		}
		colOf[id] = len(cols)
		c := &column{id: id, actor: actor, kind: kind, label: label, createAt: createAt, destroyAt: destroyAt, lifeEnd: -1}
		c.w, c.h = actorSize(kind, &c.label, minW, minH, choose(tight, 8, padX))
		cols = append(cols, c)
	}
	last := len(cols) - 1

	// Widths between lifelines. Every message and note says how far apart the lifelines it
	// touches must be.
	needs := map[int]float64{}
	var needKeys []int
	// need records that the lifelines lo and hi must be at least width apart, as the need of
	// Pele's draw does.
	need := func(lo, hi int, width float64) {
		if lo < 0 || hi > last {
			return
		}
		key := lo*len(cols) + hi
		if was, ok := needs[key]; was < width {
			if !ok {
				needKeys = append(needKeys, key)
			}
			needs[key] = width
		}
	}
	// badgeWidth is the width of the number badge of message i, 0 where it has none, as the
	// badgeWidth of Pele's draw gives it.
	badgeWidth := func(i int) float64 {
		if !numbers.shown[i] {
			return 0
		}
		return max(badgeH, measure.Width(chars.NumberString(numbers.index[i]), tiny, 0)+8)
	}

	views := make([]*view, count)
	// sized holds the text of each message as it has been wrapped to each width, so that the
	// layouts that try out spare width measure it once.
	sized := make([]map[float64]text.Label, count)
	// sizedText is the text of message i wrapped to room, as the sizedText of Pele's draw gives
	// it.
	sizedText := func(i int, room float64) text.Label {
		known := sized[i]
		if known == nil {
			if room == wrapW {
				return views[i].label
			}
			known = map[float64]text.Label{wrapW: views[i].label}
			sized[i] = known
		}
		label, ok := known[room]
		if !ok {
			label = messageText(messages[i].Text, true, room)
			known[room] = label
		}
		return label
	}
	for i, m := range messages {
		a, okA := colOf[m.From]
		b, okB := colOf[m.To]
		if m.From == "" || m.To == "" || !okA || !okB || m.Numbering != nil {
			continue
		}
		if m.Type == NoteType {
			label := labelOf(m.Text, small, m.Wrap)
			w := label.Width + 2*notePad
			switch {
			case m.Placement == LeftOf:
				need(a-1, a, w+2*noteGap+actW)
			case m.Placement == RightOf:
				need(a, a+1, w+2*noteGap+actW)
			case a == b:
				need(a-1, a, w/2+noteGap)
				need(a, a+1, w/2+noteGap)
			default:
				need(min(a, b), max(a, b), w-2*noteOver)
			}
			views[i] = &view{kind: vNote, a: a, b: b, label: label, w: w}
		} else if isLineMessage(m.Type) {
			label := messageText(m.Text, m.Wrap, wrapW)
			badge := badgeWidth(i)
			if a == b {
				need(a, a+1, selfW+8+label.Width+16+actW+badge)
				views[i] = &view{kind: vSelf, a: a, b: b, label: label}
			} else {
				created := 0.0
				if cols[b].createAt == i {
					created = cols[b].w / 2
				}
				central := 0.0
				if m.CentralConnection != 0 {
					central = 4 * centralR
				}
				lw := label.Width
				if tight {
					lw = min(label.Width, max(16, wrapW-16))
				}
				need(min(a, b), max(a, b), lw+2*msgPad+actW+badge+created+central)
				views[i] = &view{kind: vLine, a: a, b: b, label: label}
			}
		}
	}

	var runs []*boxRun
	for i := 0; i <= last; i++ {
		box := cols[i].actor.Box
		var run *boxRun
		if len(runs) > 0 {
			run = runs[len(runs)-1]
		}
		if box != nil && run != nil && run.box == box && run.last == i-1 {
			run.last = i
		} else if box != nil {
			runs = append(runs, &boxRun{box: box, first: i, last: i, title: labelOf(box.Name, small, box.Wrap)})
		}
	}

	gaps := make([]float64, max(last, 0))
	for i := 0; i < last; i++ {
		left, right := cols[i], cols[i+1]
		boxed := 0.0
		if left.actor.Box != right.actor.Box {
			if left.actor.Box != nil {
				boxed = boxPad
			}
			if right.actor.Box != nil {
				boxed += boxPad
			}
		}
		gaps[i] = max(left.w/2+right.w/2+actorMargin+boxed, needs[i*len(cols)+i+1])
	}
	// A message across several lifelines widens each gap it crosses by an equal share of
	// what is missing.
	type span struct {
		lo, hi int
		width  float64
	}
	var spans []span
	for _, key := range needKeys {
		lo, hi := key/len(cols), key%len(cols)
		if hi-lo > 1 {
			spans = append(spans, span{lo, hi, needs[key]})
		}
	}
	slices.SortStableFunc(spans, func(p, q span) int {
		if d := (p.hi - p.lo) - (q.hi - q.lo); d != 0 {
			return d
		}
		return p.lo - q.lo
	})
	for _, s := range spans {
		total := 0.0
		for i := s.lo; i < s.hi; i++ {
			total += gaps[i]
		}
		if total >= s.width {
			continue
		}
		share := (s.width - total) / float64(s.hi-s.lo)
		for i := s.lo; i < s.hi; i++ {
			gaps[i] += share
		}
	}
	// place sets the participants across from the gaps between their lifelines, as the place of
	// Pele's draw does.
	place := func() {
		for i := 1; i <= last; i++ {
			cols[i].x = cols[i-1].x + gaps[i-1]
		}
	}
	place()
	for _, run := range runs {
		first, end := cols[run.first], cols[run.last]
		inner := end.x + end.w/2 - (first.x - first.w/2)
		run.grow = max(0, (run.title.Width-inner)/2)
		if run.grow == 0 {
			continue
		}
		if run.first > 0 {
			gaps[run.first-1] += run.grow
		}
		if run.last < last {
			gaps[run.last] += run.grow
		}
	}

	minX, maxX := inf, math.Inf(-1)
	// extend widens the extent across of what is drawn to hold x0 to x1, as the extend of Pele's
	// draw does.
	extend := func(x0, x1 float64) {
		if x0 < minX {
			minX = x0
		}
		if x1 > maxX {
			maxX = x1
		}
	}
	// boxEdges is the left and right of a box, as the boxEdges of Pele's draw gives them.
	boxEdges := func(run *boxRun) (float64, float64) {
		first, end := cols[run.first], cols[run.last]
		return first.x - first.w/2 - boxPad - run.grow, end.x + end.w/2 + boxPad + run.grow
	}
	frames := make([]*seqFrame, count)
	// spanAll is the extent across of a block that holds no participant of its own, as the
	// spanAll of Pele's draw gives it.
	spanAll := func() (float64, float64) {
		if last < 0 {
			return 0, 120
		}
		return cols[0].x - framePad, cols[last].x + framePad
	}
	// layoutAcross places the participants from the gaps between their lifelines and works out
	// the extents across of what is drawn, of every block and of every note, and the text of the
	// messages that wrap to the room between their lifelines, as the layout of Pele's draw does.
	// It is laid out again for each share of spare width tried.
	layoutAcross := func() {
		place()
		minX, maxX = inf, math.Inf(-1)
		for _, c := range cols {
			extend(c.x-c.w/2, c.x+c.w/2)
		}
		for _, run := range runs {
			extend(boxEdges(run))
		}

		// The extents across of every block, from what it holds.
		var open []*seqFrame
		for i, m := range messages {
			typ := m.Type
			frameKind, isFrame := frameKinds[typ]
			v := views[i]
			switch {
			case isFrame:
				f := &seqFrame{kind: frameKind, x0: inf, x1: math.Inf(-1), dividers: []int{i}}
				if frameKind == "rect" && m.Numbering == nil {
					f.fill = m.Text
				}
				frames[i] = f
				open = append(open, f)
			case typ == AltElse || typ == ParAnd || typ == CriticalOption:
				if len(open) > 0 {
					f := open[len(open)-1]
					f.dividers = append(f.dividers, i)
				}
			case typ == LoopEnd || typ == AltEnd || typ == OptEnd || typ == ParEnd || typ == CriticalEnd || typ == BreakEnd || typ == RectEnd:
				if len(open) == 0 {
					continue
				}
				f := open[len(open)-1]
				open = open[:len(open)-1]
				frames[i] = f
				if f.x0 > f.x1 {
					f.x0, f.x1 = spanAll()
				}
				if f.kind != "rect" {
					f.tabW = measure.Width(f.kind, tiny, text.Bold) + 14
					// A block grows to the right to give its texts up to frameText of width
					// before they wrap.
					for k, at := range f.dividers {
						raw := messages[at].Text
						tab := 0.0
						if k == 0 {
							tab = f.tabW + 8
						}
						room := f.x1 - f.x0 - 16 - tab
						var label text.Label
						if raw != "" {
							floor := float64(frameText)
							if tight {
								floor = min(frameText, wrapW)
							}
							label = textOf("["+raw+"]", small, true, max(room, floor), 0, measure)
						}
						if label.Width > room {
							f.x1 += label.Width - room
						}
						if k == 0 {
							f.title = label
						} else {
							f.dividerLabels = append(f.dividerLabels, label)
						}
					}
				}
				extend(f.x0, f.x1)
				if len(open) > 0 {
					parent := open[len(open)-1]
					parent.x0 = min(parent.x0, f.x0-frameNest)
					parent.x1 = max(parent.x1, f.x1+frameNest)
				}
			case v != nil:
				a, b := cols[v.a], cols[v.b]
				var x0, x1 float64
				switch v.kind {
				case vNote:
					switch {
					case m.Placement == LeftOf:
						v.x = a.x - noteGap - v.w
					case m.Placement == RightOf:
						v.x = a.x + noteGap
					case v.a == v.b:
						v.x = a.x - v.w/2
					default:
						lo := min(a.x, b.x) - noteOver
						hi := max(a.x, b.x) + noteOver
						if tight || m.Wrap {
							v.label = textOf(m.Text, small, true, max(wrapW, hi-lo-2*notePad), 0, measure)
						}
						v.w = max(hi-lo, v.label.Width+2*notePad)
						v.x = (lo + hi - v.w) / 2
					}
					x0, x1 = v.x, v.x+v.w
				case vSelf:
					x0 = a.x
					x1 = a.x + actW + selfW + 8 + v.label.Width + badgeWidth(i)
				default:
					x0, x1 = min(a.x, b.x), max(a.x, b.x)
					if tight || m.Wrap {
						v.label = sizedText(i, max(wrapW, x1-x0-2*msgPad-actW))
					}
					if b.createAt == i {
						if b.x > a.x {
							x1 += b.w / 2
						} else {
							x0 -= b.w / 2
						}
					}
				}
				extend(x0, x1)
				if len(open) > 0 {
					f := open[len(open)-1]
					f.x0 = min(f.x0, x0-framePad)
					f.x1 = max(f.x1, x1+framePad)
				}
			}
		}
	}
	layoutAcross()

	// Widen gaps only if it reduces message wrapping.
	grown, uneven := false, false
	if tight && o.MaxWidth > 0 {
		spare := math.Floor(o.MaxWidth) - math.Ceil(maxX-minX+2*pad) - hold
		stretch := make([]int32, len(cols))
		want := make([]float64, len(gaps))
		var wrapped []int
		lines := 0
		for i := 0; spare > 0 && i < len(views); i++ {
			v := views[i]
			if v == nil || v.kind != vLine {
				continue
			}
			natural := messageText(messages[i].Text, false, inf)
			if len(v.label.Lines) <= len(natural.Lines) {
				continue
			}
			wrapped = append(wrapped, i)
			lines += len(v.label.Lines)
			lo, hi := min(v.a, v.b), max(v.a, v.b)
			stretch[lo]++
			stretch[hi]--
			short := (natural.Width - (cols[hi].x - cols[lo].x - 2*msgPad - actW)) / float64(hi-lo)
			for g := lo; g < hi; g++ {
				want[g] = max(want[g], short)
			}
		}
		var active int32
		crossed := 0.0
		for i := range gaps {
			active += stretch[i]
			stretch[i] = 0
			if active > 0 {
				stretch[i] = 1
			}
			crossed += float64(stretch[i])
		}
		if crossed > 0 {
			before := slices.Clone(gaps)
			// spread widens each gap by its share and lays the diagram out again, and gives the
			// lines of each message that wrapped, as the spread of Pele's draw does.
			spread := func(share func(g int) float64) []int {
				for g := range gaps {
					gaps[g] = before[g] + share(g)
				}
				layoutAcross()
				counts := make([]int, len(wrapped))
				for k, i := range wrapped {
					counts[k] = len(views[i].label.Lines)
				}
				return counts
			}
			// sum is the sum of counts, as the sum of Pele's draw gives it.
			sum := func(counts []int) int {
				total := 0
				for _, n := range counts {
					total += n
				}
				return total
			}
			// even is an equal share of the spare width for each gap a wrapped message
			// crosses, as the even of Pele's draw gives it.
			even := func(g int) float64 { return float64(stretch[g]) * spare / crossed }
			counts := spread(even)
			missing := 0.0
			for _, w := range want {
				missing += w
			}
			// needs is a share of the spare width by what gap g lacks, less a little to allow
			// for rounding error, as the needs of Pele's draw gives it.
			needs := func(g int) float64 { return want[g] * (spare - 0.01) / missing }
			// wider reports whether some gap's share by need is wider than its even share.
			wider := func() bool {
				for g := range gaps {
					if needs(g) > even(g) {
						return true
					}
				}
				return false
			}
			// Reducing wrapping requires at least one gap wider than the even split.
			if byNeed && crossed > 1 && missing > 0 && wider() {
				first := counts
				counts = spread(needs)
				uneven = sum(counts) < sum(first)
				for k, n := range counts {
					if n > first[k] {
						uneven = false
					}
				}
				if !uneven {
					counts = spread(even)
				}
			}
			grown = sum(counts) < lines
			if !grown {
				copy(gaps, before)
				layoutAcross()
			}
		}
	}

	// Placement down the page, top to bottom, drawing as it goes.
	boxTitleH := 0.0
	for _, run := range runs {
		boxTitleH = max(boxTitleH, run.title.Height)
	}
	boxTop := 0.0
	if len(runs) > 0 {
		boxTop = boxPad
		if boxTitleH > 0 {
			boxTop = boxPad + (boxTitleH + 4)
		}
	}
	rowH := 0.0
	for _, c := range cols {
		if c.createAt < 0 {
			rowH = max(rowH, c.h)
		}
	}
	headBottom := boxTop + rowH
	for _, c := range cols {
		c.top = headBottom - c.h
		c.lifeStart = headBottom
	}

	xs := make([]float64, len(cols))
	for i, c := range cols {
		xs[i] = c.x
	}
	// crosses reports whether a lifeline runs between two places across, where text would
	// be struck through. It is the crosses of Pele's draw.
	crosses := func(x0, x1 float64) bool {
		lo, hi := 0, len(xs)
		for lo < hi {
			mid := (lo + hi) >> 1
			if xs[mid] <= x0 {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		return lo < len(xs) && xs[lo] < x1
	}
	// backed is a label on a patch of the background where a lifeline runs under it, as
	// Pele's backed draws it; ok is false where there is nothing to draw.
	backed := func(it scene.Item, label text.Label, cx, cy float64) (scene.Item, bool) {
		if label.Width == 0 {
			return it, false
		}
		l := cx - label.Width/2
		if crosses(l, l+label.Width) {
			it.Parts = []scene.Part{{
				Box:     scene.Rect{X: l - 3, Y: cy - label.Height/2 + 2, W: label.Width + 6, H: label.Height - 4},
				Rounded: true, Paint: scene.FillOnly,
			}}
			it.Style.Fill, it.Style.Radius = style.Bg, style.N(3)
		}
		it.Label, it.At = &label, scene.Point{X: cx, Y: cy}
		it.Box = scene.Rect{X: l, Y: cy - label.Height/2, W: label.Width, H: label.Height}
		return it, true
	}
	// leftEdge is the left of a participant's lifeline with extra more activations, at the side
	// of its outer activation when it has one, as the leftEdge of Pele's draw gives it.
	leftEdge := func(c *column, extra int) float64 {
		if len(c.activations)+extra > 0 {
			return c.x - actW/2
		}
		return c.x
	}
	// rightEdge is the right of a participant's lifeline with extra more activations, at the side
	// of its inner activation when it has one, as the rightEdge of Pele's draw gives it.
	rightEdge := func(c *column, extra int) float64 {
		if depth := len(c.activations) + extra; depth > 0 {
			return c.x + actW/2 + float64((depth-1)*actStep)
		}
		return c.x
	}

	// Shaded blocks and activations close from the inside out. They are listed in reverse,
	// so that the inner ones are drawn over the outer.
	var rects, activations, framesOut, edges, destroys, labels, notes, actors, lifelines, boxes []scene.Item
	// endActivation closes a participant's inner activation at end, at least actMin below where
	// it opened, and gives its bottom, as the endActivation of Pele's draw does.
	endActivation := func(c *column, end float64) float64 {
		start := c.activations[len(c.activations)-1]
		c.activations = c.activations[:len(c.activations)-1]
		bottom := max(end, start+actMin)
		x := c.x - actW/2 + float64(len(c.activations)*actStep)
		extend(x, x+actW)
		box := scene.Rect{X: x, Y: start, W: actW, H: bottom - start}
		activations = append(activations, scene.Item{
			Kind: scene.Mark, Role: RoleActivation, ID: c.id, Box: box,
			Parts: []scene.Part{{Box: box, Rounded: true}},
			Style: style.Style{Fill: style.Surface, Stroke: style.Border, Radius: style.N(2)},
		})
		return bottom
	}
	// The lifeline of a destroyed participant ends in a cross a little below the message
	// that ends it, with round ends, as Pele draws its crosses in a group of them. It is the
	// destroy of Pele's draw.
	destroy := func(c *column, lineY float64) float64 {
		cx, cy := c.x, lineY+16
		c.lifeEnd = cy
		p := new(scene.Path).MoveTo(cx-6, cy-6).LineTo(cx+6, cy+6).MoveTo(cx+6, cy-6).LineTo(cx-6, cy+6)
		destroys = append(destroys, scene.Item{
			Kind: scene.Mark, Role: RoleDestroy, ID: c.id, Box: p.Bounds(),
			Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
			Style: style.Style{Stroke: style.Line, StrokeWidth: style.N(1.5), Cap: style.CapRound},
		})
		return cy + 6
	}
	// actorItem is a participant drawn with its box's top at top, with the classes its properties
	// name, as the actorSvg of Pele's draw draws it.
	actorItem := func(c *column, top float64) scene.Item {
		parts, at := drawActor(c.kind, c.x, top, c.w, c.h, &c.label)
		shape := scene.Shape(c.kind)
		if c.kind == "participant" {
			shape = "rect"
		}
		it := scene.Item{
			Kind: scene.Node, Role: RoleActor, Shape: shape, ID: c.id,
			Box:   scene.Rect{X: c.x - c.w/2, Y: top, W: c.w, H: c.h},
			Parts: parts, At: at,
			Style: style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg},
		}
		if cls, ok := c.actor.Properties.Get("class"); ok {
			if s, ok := cls.(string); ok {
				it.Class = common.ClassNames(s)
			}
		}
		if !c.label.Empty() {
			label := c.label
			it.Label = &label
		}
		return it
	}
	// badge is the number of message i in a badge of width w about cx, cy, as the badge of Pele's
	// draw draws it.
	badge := func(i int, cx, cy, w float64) scene.Item {
		label := text.Layout(chars.NumberString(numbers.index[i]), text.Options{Measurer: measure, Size: tiny})
		box := scene.Rect{X: cx - w/2, Y: cy - badgeH/2, W: w, H: badgeH}
		return scene.Item{
			Kind: scene.Node, Role: RoleNumber, Shape: "rounded", Box: box,
			Parts: []scene.Part{{Box: box, Rounded: true}},
			Label: &label, At: scene.Point{X: cx, Y: cy},
			Style: style.Style{Fill: style.Line, Stroke: style.None, Text: style.Bg, Radius: style.N(badgeH / 2)},
		}
	}

	y := headBottom + 6
	var stackOf []*seqFrame
	for i, m := range messages {
		typ := m.Type
		v := views[i]
		f := frames[i]
		switch {
		case typ == ActiveStart:
			if c, ok := colOf[m.From]; ok && m.From != "" {
				cols[c].activations = append(cols[c].activations, y)
			}
		case typ == ActiveEnd:
			if c, ok := colOf[m.From]; ok && m.From != "" && len(cols[c].activations) > 0 {
				y = max(y, endActivation(cols[c], y))
			}
		case f != nil && f.dividers[0] == i:
			stackOf = append(stackOf, f)
			if f.kind == "rect" {
				f.top = y + 8
				y = f.top
			} else {
				f.top = y + 12
				y = f.top + max(tabH, f.title.Height+2)
			}
		case f != nil:
			stackOf = stackOf[:len(stackOf)-1]
			bottom := y + 12
			box := scene.Rect{X: f.x0, Y: f.top, W: f.x1 - f.x0, H: bottom - f.top}
			if f.kind == "rect" {
				own, author := fillStyle(f.fill, f.fill != "" && writesFill(f.fill))
				rects = append(rects, scene.Item{
					Kind: scene.Group, Role: RoleRect, Shape: "rect", Box: box,
					Parts: []scene.Part{{Box: box, Rounded: true, Paint: scene.FillOnly}},
					Style: own, Author: author,
				})
			} else {
				framesOut = append(framesOut, scene.Item{
					Kind: scene.Group, Role: RoleFrame, Shape: scene.Shape(f.kind), Box: box,
					Parts: []scene.Part{{Box: box, Rounded: true, Paint: scene.StrokeOnly}},
					Style: style.Style{Stroke: style.Border},
				})
				condition := scene.Item{Kind: scene.Text, Role: RoleCondition, Style: style.Style{Text: style.Muted}}
				for k, dy := range f.dividerY {
					label := f.dividerLabels[k]
					p := new(scene.Path).MoveTo(f.x0, dy).LineTo(f.x1, dy)
					framesOut = append(framesOut, scene.Item{
						Kind: scene.Mark, Role: RoleDivider, Box: p.Bounds(),
						Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
						Style: style.Style{Stroke: style.Border, Dash: style.Dashes(4, 4)},
					})
					if it, ok := backed(condition, label, f.x0+8+label.Width/2, dy+2+label.Height/2); ok {
						framesOut = append(framesOut, it)
					}
				}
				tab := scene.Rect{X: f.x0, Y: f.top, W: f.tabW, H: tabH}
				kind := text.Layout(f.kind, text.Options{Measurer: measure, Size: tiny, Face: text.Bold})
				framesOut = append(framesOut, scene.Item{
					Kind: scene.Text, Role: RoleFrameKind, Shape: scene.Shape(f.kind), Box: tab,
					Parts: []scene.Part{{Box: tab, Rounded: true}},
					Label: &kind, At: scene.Point{X: f.x0 + f.tabW/2, Y: f.top + tabH/2},
					Style: style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Muted},
				})
				if it, ok := backed(condition, f.title, f.x0+f.tabW+8+f.title.Width/2, f.top+max(tabH, f.title.Height)/2); ok {
					framesOut = append(framesOut, it)
				}
			}
			y = bottom
		case typ == AltElse || typ == ParAnd || typ == CriticalOption:
			if len(stackOf) > 0 {
				current := stackOf[len(stackOf)-1]
				label := current.dividerLabels[len(current.dividerY)]
				current.dividerY = append(current.dividerY, y+12)
				if label.Height > 0 {
					y += 12 + label.Height
				} else {
					y += 12
				}
			}
		case v != nil && v.kind == vNote:
			a := cols[v.a]
			h := v.label.Height + 2*max(4, notePad-4)
			top := y + 10
			x := v.x
			if m.Placement == RightOf {
				x += rightEdge(a, 0) - a.x
			} else if m.Placement == LeftOf {
				x += leftEdge(a, 0) - a.x
			}
			extend(x, x+v.w)
			box := scene.Rect{X: x, Y: top, W: v.w, H: h}
			it := scene.Item{
				Kind: scene.Node, Role: RoleNote, Shape: "rect", ID: "i" + m.ID, Box: box,
				Parts: []scene.Part{{Box: box, Rounded: true}},
				At:    scene.Point{X: x + v.w/2, Y: top + h/2},
				Style: style.Style{Fill: style.SurfaceAlt, Stroke: style.Border, Text: style.Fg},
			}
			if !v.label.Empty() {
				label := v.label
				it.Label = &label
			}
			notes = append(notes, it)
			y = top + h
		case v != nil:
			st := lineStyles[typ]
			a, b := cols[v.a], cols[v.b]
			label := v.label
			bw := badgeWidth(i)
			// The activation a `+` starts belongs to this message, so the arrow stops at its
			// edge.
			starting := 0
			for k := i + 1; k < count; k++ {
				t := messages[k].Type
				if t == ActiveStart {
					if messages[k].From == m.To {
						starting++
					}
				} else if t != ActiveEnd && t != CentralConnection && t != CentralConnectionReverse {
					break
				}
			}
			up := -1.0
			if st.bottom {
				up = 1
			}
			cc := m.CentralConnection
			circleTo := cc == CentralConnection || cc == CentralConnectionDual
			circleFrom := cc == CentralConnectionReverse || cc == CentralConnectionDual
			tailAtEnd := st.atStart && !st.atEnd
			headTrim := trim(st.head)
			// A message's line and heads end round, as Pele draws them in the group of
			// edges, which sets round caps.
			msg := scene.Item{
				Kind: scene.Edge, Role: RoleMessage, ID: "i" + m.ID,
				Style: style.Style{Stroke: style.Line, Fill: style.Line, Cap: style.CapRound},
			}
			if st.dotted {
				msg.Style.Dash = style.Dashes(4, 4)
			}
			var circles []scene.Part
			// addCircle adds a dot of a central connection, as the circle of Pele's draw draws one.
			addCircle := func(cx, cy float64) { circles = append(circles, circle(cx, cy, centralR, scene.FillOnly)) }
			var number *scene.Item
			var lineY float64

			if v.kind == vSelf {
				top := y + msgGap + max(0, (label.Height-selfH)/2)
				bottom := top + selfH
				x0 := rightEdge(a, 0)
				if circleFrom {
					x0 = a.x + centralR
				}
				x1 := rightEdge(a, starting)
				if circleTo {
					x1 = a.x + centralR
				}
				out := max(x0, x1) + selfW + bw/2
				if circleFrom {
					addCircle(a.x, top)
				}
				if circleTo {
					addCircle(a.x, bottom)
				}
				if bw > 0 {
					if tailAtEnd {
						n := badge(i, x1, bottom, bw)
						number = &n
						x1 += bw / 2
					} else {
						n := badge(i, x0, top, bw)
						number = &n
						x0 += bw / 2
					}
				}
				// The loop leaves to the right and comes back heading left, which turns a
				// half head over.
				if st.atStart {
					msg.Ends[0] = headMarker(st.head, x0, top, -1, up)
				}
				if st.atEnd {
					msg.Ends[1] = headMarker(st.head, x1, bottom, -1, -up)
				}
				from, to := x0, x1
				if st.atStart {
					from = x0 + headTrim
				}
				if st.atEnd {
					to = x1 + headTrim
				}
				p := new(scene.Path).MoveTo(from, top)
				if rightAngles {
					p.LineTo(out, top).LineTo(out, bottom).LineTo(to, bottom)
				} else {
					p.LineTo(out-6, top).ArcTo(6, 6, 0, false, true, out, top+6).LineTo(out, bottom-6).
						ArcTo(6, 6, 0, false, true, out-6, bottom).LineTo(to, bottom)
				}
				msg.Route = []scene.Point{{X: x0, Y: top}, {X: out, Y: top}, {X: out, Y: bottom}, {X: x1, Y: bottom}}
				msg.Parts = append([]scene.Part{{Path: p, Paint: scene.StrokeOnly}}, circles...)
				extend(a.x, out+8+label.Width)
				if label.Width > 0 {
					l := label
					labels = append(labels, scene.Item{
						Kind: scene.Text, Role: RoleMessageLabel, ID: "i" + m.ID,
						Box:   scene.Rect{X: out + 8, Y: (top+bottom)/2 - l.Height/2, W: l.Width, H: l.Height},
						Label: &l, At: scene.Point{X: out + 8 + l.Width/2, Y: (top + bottom) / 2},
						Style: style.Style{Text: style.Fg},
					})
				}
				// A label taller than the loop reaches below it, so the next row starts under
				// the label.
				lineY = max(bottom, (top+bottom)/2+label.Height/2)
			} else {
				dir := -1.0
				if b.x > a.x {
					dir = 1
				}
				creating := b.createAt == i
				low := 6.0
				if label.Height > 0 {
					low = label.Height + 2
				}
				if creating {
					low = max(low, b.h/2-msgGap+6)
				} else {
					low = max(low, 0)
				}
				lineY = y + msgGap + low
				var x0, x1 float64
				if dir > 0 {
					x0, x1 = rightEdge(a, 0), leftEdge(b, starting)
				} else {
					x0, x1 = leftEdge(a, 0), rightEdge(b, starting)
				}
				if creating {
					b.top = lineY - b.h/2
					b.lifeStart = b.top + b.h
					if isFigure(b.kind) {
						x1 = b.x - dir*18
					} else {
						x1 = b.x - dir*(b.w/2+1)
					}
				}
				if circleFrom {
					addCircle(a.x, lineY)
					x0 = a.x + dir*centralR
				}
				if circleTo {
					addCircle(b.x, lineY)
					x1 = b.x - dir*centralR
				}
				if bw > 0 {
					// The number sits on the lifeline at the tail of the arrow, past a
					// connection circle if there is one.
					if tailAtEnd {
						cx := x1
						if circleTo {
							cx = x1 - dir*(bw/2+2)
						}
						n := badge(i, cx, lineY, bw)
						number = &n
						x1 = cx - dir*bw/2
					} else {
						cx := x0
						if circleFrom {
							cx = x0 + dir*(bw/2+2)
						}
						n := badge(i, cx, lineY, bw)
						number = &n
						x0 = cx + dir*bw/2
					}
				}
				if st.atStart {
					msg.Ends[0] = headMarker(st.head, x0, lineY, -dir, up*dir)
				}
				if st.atEnd {
					msg.Ends[1] = headMarker(st.head, x1, lineY, dir, up*dir)
				}
				from, to := x0, x1
				if st.atStart {
					from = x0 + dir*headTrim
				}
				if st.atEnd {
					to = x1 - dir*headTrim
				}
				p := new(scene.Path).MoveTo(from, lineY).LineTo(to, lineY)
				msg.Route = []scene.Point{{X: x0, Y: lineY}, {X: x1, Y: lineY}}
				msg.Parts = append([]scene.Part{{Path: p, Paint: scene.StrokeOnly}}, circles...)
				if label.Width > 0 {
					l, r := min(x0, x1), max(x0, x1)
					cx := (l + r) / 2
					if align < 0 {
						cx = l + msgPad + label.Width/2
					} else if align > 0 {
						cx = r - msgPad - label.Width/2
					}
					extend(cx-label.Width/2, cx+label.Width/2)
					it := scene.Item{Kind: scene.Text, Role: RoleMessageLabel, ID: "i" + m.ID, Style: style.Style{Text: style.Fg}}
					if it, ok := backed(it, label, cx, lineY-4-label.Height/2); ok {
						labels = append(labels, it)
					}
				}
				if creating {
					y = lineY + b.h/2 - 6
				}
			}
			msg.Box = routeBounds(msg.Route)
			edges = append(edges, msg)
			if number != nil {
				edges = append(edges, *number)
			}
			y = max(y, lineY)
			if a.destroyAt == i {
				y = max(y, destroy(a, lineY))
			}
			if b.destroyAt == i && b != a {
				y = max(y, destroy(b, lineY))
			}
		}
	}

	y += 20
	for _, c := range cols {
		for len(c.activations) > 0 {
			endActivation(c, y)
		}
	}
	bottom := y
	if mirror {
		footH := 0.0
		for _, c := range cols {
			if c.lifeEnd >= 0 {
				continue
			}
			footH = max(footH, c.h)
			actors = append(actors, actorItem(c, y))
		}
		bottom = y + footH
	}
	for _, c := range cols {
		actors = append(actors, actorItem(c, c.top))
		end := y
		if c.lifeEnd >= 0 {
			end = c.lifeEnd
		}
		if end > c.lifeStart {
			p := new(scene.Path).MoveTo(c.x, c.lifeStart).LineTo(c.x, end)
			lifelines = append(lifelines, scene.Item{
				Kind: scene.Mark, Role: RoleLifeline, ID: c.id, Box: p.Bounds(),
				Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
				Style: style.Style{Stroke: style.Border},
			})
		}
	}

	for _, run := range runs {
		x0, x1 := boxEdges(run)
		own, author := fillStyle(run.box.Fill, run.box.Fill != "transparent")
		own.Stroke, own.Text = style.Border, style.Muted
		box := scene.Rect{X: x0, Y: 0, W: x1 - x0, H: bottom + boxPad}
		it := scene.Item{
			Kind: scene.Group, Role: RoleBox, Shape: "rect", ID: run.box.Name, Box: box,
			Parts: []scene.Part{{Box: box, Rounded: true}},
			At:    scene.Point{X: (x0 + x1) / 2, Y: boxPad/2 + 2 + run.title.Height/2},
			Style: own, Author: author,
		}
		if !run.title.Empty() {
			title := run.title
			it.Label = &title
		}
		boxes = append(boxes, it)
	}
	if len(runs) > 0 {
		bottom += boxPad
	}

	minY := 0.0
	var titles []scene.Item
	titleWidth := wrapW
	if tight {
		titleWidth = common.TitleRoom(o)
	}
	title := textOf(db.Title, size, false, titleWidth, text.Bold, measure)
	if minX > maxX {
		extend(0, max(title.Width, 40))
	}
	if title.Width > 0 {
		cx := (minX + maxX) / 2
		extend(cx-title.Width/2, cx+title.Width/2)
		minY = -title.Height - 12
		titles = append(titles, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			Box:   scene.Rect{X: cx - title.Width/2, Y: minY, W: title.Width, H: title.Height},
			At:    scene.Point{X: cx, Y: minY + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}

	width := math.Ceil(maxX - minX + 2*pad)
	// Retry with equal shares or room for activation notes and rounding.
	if grown && width > o.MaxWidth {
		switch {
		case uneven:
		case hold > 0:
			hold = inf
		default:
			hold = width - math.Floor(o.MaxWidth)
		}
		return lay(db, config, o, tight, hold, false)
	}
	sc := &scene.Scene{
		Type:     "sequence",
		Width:    width,
		Height:   math.Ceil(bottom - minY + 2*pad),
		FontSize: size,
		Title:    db.AccTitle,
		Desc:     db.AccDescr,
	}
	slices.Reverse(rects)
	slices.Reverse(activations)
	for _, layer := range [][]scene.Item{titles, boxes, rects, lifelines, activations, framesOut, edges, destroys, labels, notes, actors} {
		sc.Items = append(sc.Items, layer...)
	}
	// Pele draws all of it in a group moved by the padding past the extents; the scene holds
	// where it is on the page.
	dx, dy := pad-minX, pad-minY
	for i := range sc.Items {
		sc.Items[i].Move(dx, dy)
	}
	return sc
}

// routeBounds is the box a route's corners lie in. It is limu's own, for the box of a
// message's item.
func routeBounds(route []scene.Point) scene.Rect {
	if len(route) == 0 {
		return scene.Rect{}
	}
	x0, y0, x1, y1 := route[0].X, route[0].Y, route[0].X, route[0].Y
	for _, p := range route[1:] {
		x0, y0, x1, y1 = min(x0, p.X), min(y0, p.Y), max(x1, p.X), max(y1, p.Y)
	}
	return scene.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// choose is a ? b : c. It is limu's own, for JavaScript's conditional expression.
func choose(cond bool, a, b float64) float64 {
	if cond {
		return a
	}
	return b
}
