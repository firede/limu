package ishikawa

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

// The scene's roles. Across, in the order drawn: "ishikawa.title" for the title;
// "ishikawa.spine" for the spine, with the arrowhead at its end; and for each category, its
// "ishikawa.bones", the bone that meets the spine and those of the causes off it, one path,
// with the arrowhead where it meets the spine, its "ishikawa.category", the box with its name
// at the far end of its bone, and an "ishikawa.cause" for the text of each of its causes;
// then "ishikawa.head", the effect at the head of the spine. Down, the title, an
// "ishikawa.link" for the line from each node to its parent, and then the effect, the
// categories and the causes, in the order of the outline.
const (
	RoleTitle    scene.Role = "ishikawa.title"
	RoleSpine    scene.Role = "ishikawa.spine"
	RoleBones    scene.Role = "ishikawa.bones"
	RoleCategory scene.Role = "ishikawa.category"
	RoleCause    scene.Role = "ishikawa.cause"
	RoleHead     scene.Role = "ishikawa.head"
	RoleLink     scene.Role = "ishikawa.link"
)

// The measures of the layout, as Pele's render.ts has them.
const (
	// slant is how far a bone moves toward the tail for each unit it rises from the spine.
	slant    = 0.5
	firstRow = 14
	rowGap   = 6
	boneEnd  = 8
	twigStep = 22
	stub     = 18
	textGap  = 5
	pairGap  = 26
	headGap  = 34
	tail     = 28
	boxPadX  = 10
	boxPadY  = 5
	// arrow is the length of an arrowhead, Pele's ARROW.
	arrow = scene.MarkerSize
)

// length is the length of a bone for each unit it rises, Pele's LENGTH.
var length = jsmath.Hypot(slant, 1)

// A bone is a cause below a category, Pele's Bone. Bones at even depths run level, from the
// slanted bone above them toward the tail; bones at odd depths slant away from the spine,
// from the level bone above them.
type bone struct {
	label  text.Label
	depth  int
	parent int
	// slot is its place among its siblings, counted from the spine outward.
	slot int
	kids int
	// rise is how far its row is from the spine.
	rise float64
	// at is where it meets its parent, measured level from the category's bone: 0 on it,
	// negative toward the tail.
	at float64
	// textEnd is where its text ends.
	textEnd float64
}

// A category is a child of the effect, Pele's Category.
type category struct {
	node       *Node
	label      text.Label
	boxW, boxH float64
	bones      []bone
	// rise is the length of its bone, as distance from the spine.
	rise float64
	// reach is how far its causes reach toward the tail, measured level from its bone.
	reach float64
}

// A frame is a step of arrange's walk, Pele's Frame.
type frame struct {
	node   *Node
	depth  int
	parent int
	slot   int
	// bone is set when the bone was made and only its row is still to come.
	bone int
}

// arrange gives every cause of a category a row of its own, so that no two texts can meet,
// and measures how far the whole of it reaches, as Pele's arrange does. A level bone takes the
// row nearest the spine and its twigs the rows beyond; a slanted twig takes the row beyond
// everything that hangs off it. The outline may be nested as deep as its indentation goes,
// so the walk keeps a stack of its own.
func arrange(node *Node, up bool, o *layout.Options, size, small float64) *category {
	measure := o.Measure()
	label := text.Layout(node.Text, text.Options{Measurer: measure, Size: size, MaxWidth: 180, Icons: o.Icons})
	lineHeight := jsmath.Round(small * 1.5)
	var bones []bone
	var rows []int
	var stack []frame
	// Above the spine the first cause goes farthest out, so that the causes read from the top
	// down.
	push := func(children []*Node, depth, parent int) {
		n := len(children)
		for slot := n - 1; slot >= 0; slot-- {
			child := children[slot]
			if up {
				child = children[n-1-slot]
			}
			stack = append(stack, frame{node: child, depth: depth, parent: parent, slot: slot, bone: -1})
		}
	}
	push(node.Children, 2, -1)
	for len(stack) > 0 {
		f := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if f.bone != -1 {
			rows = append(rows, f.bone)
			continue
		}
		index := len(bones)
		bones = append(bones, bone{
			label:  text.Layout(f.node.Text, text.Options{Measurer: measure, Size: small, MaxWidth: 200, Icons: o.Icons}),
			depth:  f.depth,
			parent: f.parent,
			slot:   f.slot,
			kids:   len(f.node.Children),
		})
		if f.depth%2 == 0 {
			rows = append(rows, index)
		} else {
			f.bone = index
			stack = append(stack, f)
		}
		push(f.node.Children, f.depth+1, index)
	}

	rise := float64(firstRow)
	for _, index := range rows {
		height := max(bones[index].label.Height, lineHeight) + rowGap
		bones[index].rise = rise + height/2
		rise += height
	}
	boxW := label.Width + 2*boxPadX
	reach := boxW / 2
	for i := range bones {
		b := &bones[i]
		if b.depth%2 == 0 {
			if b.parent != -1 {
				b.at = bones[b.parent].at
			}
			run := float64(stub)
			if b.kids > 0 {
				run = float64((b.kids + 1) * twigStep)
			}
			b.textEnd = b.at - run - textGap
		} else {
			// Twigs stand along their level bone, the one nearest the spine nearest the tail.
			parent := &bones[b.parent]
			b.at = parent.at - float64((parent.kids-b.slot)*twigStep)
			b.textEnd = b.at - textGap
		}
		reach = max(reach, b.label.Width-b.textEnd)
	}
	return &category{node: node, label: label, boxW: boxW, boxH: label.Height + 2*boxPadY, bones: bones, rise: rise + boneEnd, reach: reach}
}

// Layout lays an Ishikawa diagram out as a scene, as Pele's renderIshikawa does: across, the
// effect at the head of a spine and the categories off it by turns above and below, or, when
// the host is narrow, down, as an outline.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return draw(m, o, down) })
}

// stroked is the style of a line Pele strokes in the color of lines.
var stroked = style.Style{Fill: style.None, Stroke: style.Line}

// draw lays the diagram out across, as Pele's draw does, or down.
func draw(m *Model, o *layout.Options, down bool) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	measure := o.Measure()
	const pad = common.Padding
	if down {
		return verticalCauses(m, o)
	}
	root := m.Root
	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: 4000, Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}

	var body []scene.Item
	var minX, maxX, minY, maxY float64
	// The head is drawn last, once it is known where the body goes; see below.
	var head text.Label
	var headW, headH float64
	if root != nil {
		head = text.Layout(root.Text, text.Options{Measurer: measure, Size: size, MaxWidth: 200, Face: text.Bold, Icons: o.Icons})
		headH = max(head.Height+24, 48)
		headW = head.Width + 24 + headH/2
		categories := make([]*category, len(root.Children))
		for i, node := range root.Children {
			categories[i] = arrange(node, i%2 == 0, o, size, small)
		}
		// Even a category without causes has a bone long enough to carry its name clear of
		// the head.
		for _, c := range categories {
			c.rise = max(c.rise, headH/2+14)
		}

		// Categories meet the spine in pairs, one above and one below, each pair as far
		// toward the tail from the one before as the wider of the two needs.
		var xs []float64
		x := -float64(headGap)
		for i := 0; i < len(categories); i += 2 {
			if i > 0 {
				room := 0.0
				for side := 0; side < 2; side++ {
					before := categories[i-2+side]
					next := 0.0
					if i+side < len(categories) {
						next = categories[i+side].boxW / 2
					}
					room = max(room, before.reach+next)
				}
				x -= room + pairGap
			}
			xs = append(xs, x, x)
		}
		end := -float64(headGap)
		if len(categories) > 0 {
			end = x - tail
		}

		maxX = headW
		minX = end
		minY = -headH / 2
		maxY = headH / 2
		body = append(body, scene.Item{
			Kind: scene.Edge, Role: RoleSpine,
			Box:   scene.Rect{X: end, Y: 0, W: -(arrow - 1) - end, H: 0},
			Parts: []scene.Part{{Path: new(scene.Path).MoveTo(end, 0).LineTo(-(arrow - 1), 0), Paint: scene.StrokeOnly}},
			Route: []scene.Point{{X: end, Y: 0}, {X: -(arrow - 1), Y: 0}},
			Ends:  [2]scene.Marker{{}, {Type: "arrow_point", At: scene.Point{}, DX: 1, DY: 0}},
			Style: stroked,
		})
		for i, c := range categories {
			sign := -1.0
			if i%2 != 0 {
				sign = 1
			}
			ax := xs[i]
			px := func(at, rise float64) float64 { return ax + at - float64(rise*slant) }
			endX := px(0, c.rise)
			endY := sign * c.rise
			// The bone stops short of the spine to leave its arrowhead a clean point.
			const trim = arrow - 1
			fromX, fromY := ax-float64(slant/length*trim), sign*trim/length
			d := new(scene.Path).MoveTo(fromX, fromY).LineTo(endX, endY)
			var causes []scene.Item
			for _, b := range c.bones {
				y := sign * b.rise
				if b.depth%2 == 0 {
					d.MoveTo(px(b.at, b.rise), y).LineTo(px(b.textEnd+textGap, b.rise), y)
				} else {
					from := c.bones[b.parent].rise
					d.MoveTo(px(b.at, from), sign*from).LineTo(px(b.at, b.rise), y)
				}
				right := px(b.textEnd, b.rise)
				l := b.label
				ink := style.Fg
				if b.depth > 2 {
					ink = style.Muted
				}
				causes = append(causes, scene.Item{
					Kind: scene.Text, Role: RoleCause,
					Box:   scene.Rect{X: right - l.Width, Y: y - l.Height/2, W: l.Width, H: l.Height},
					Label: &l, At: scene.Point{X: right - l.Width/2, Y: y},
					Style: style.Style{Text: ink},
				})
				minX = min(minX, right-l.Width)
			}
			boxY := endY + float64(sign*(c.boxH/2))
			minX = min(minX, endX-c.boxW/2)
			maxX = max(maxX, endX+c.boxW/2)
			minY = min(minY, boxY-c.boxH/2)
			maxY = max(maxY, boxY+c.boxH/2)
			body = append(body, scene.Item{
				Kind: scene.Edge, Role: RoleBones, ID: c.node.Text,
				Box:   d.Bounds(),
				Parts: []scene.Part{{Path: d, Paint: scene.StrokeOnly}},
				Route: []scene.Point{{X: fromX, Y: fromY}, {X: endX, Y: endY}},
				Ends:  [2]scene.Marker{{Type: "arrow_point", At: scene.Point{X: ax, Y: 0}, DX: slant / length, DY: -sign / length}},
				Style: stroked,
			})
			box := scene.Rect{X: endX - c.boxW/2, Y: boxY - c.boxH/2, W: c.boxW, H: c.boxH}
			label := c.label
			body = append(body, scene.Item{
				Kind: scene.Node, Role: RoleCategory, Shape: "rect", ID: c.node.Text,
				Box:   box,
				Parts: []scene.Part{{Box: box, Rounded: true}},
				Label: &label, At: scene.Point{X: endX, Y: boxY},
				Style: style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg},
			})
			body = append(body, causes...)
		}
	}

	inner := max(maxX-minX, title.Width)
	width := math.Ceil(inner + 2*pad)
	height := math.Ceil(maxY - minY + titleHeight + 2*pad)
	ox := pad + (inner-(maxX-minX))/2 - minX
	oy := pad + titleHeight - minY
	sc := &scene.Scene{Type: "ishikawa", Width: width, Height: height, FontSize: size}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	for i := range body {
		body[i].Move(ox, oy)
	}
	sc.Items = append(sc.Items, body...)
	if root != nil {
		// The head's outline is drawn where it is on the page, its arc between the corners
		// as Pele writes them moved there, so that the curves it is drawn with are those of
		// that arc.
		r := headH / 2
		top, bottom, left, right := -r+oy, r+oy, 0+ox, headW-r+ox
		outline := new(scene.Path).MoveTo(left, top).LineTo(right, top).ArcTo(r, r, 0, false, true, right, bottom).LineTo(left, bottom).Close()
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Node, Role: RoleHead, Shape: "head", ID: root.Text,
			Box:   scene.Rect{X: left, Y: top, W: headW, H: headH},
			Parts: []scene.Part{{Path: outline}},
			Label: &head, At: scene.Point{X: 12 + head.Width/2 + ox, Y: 0 + oy},
			Style: style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg, Join: style.JoinRound},
		})
	}
	return sc
}

// An entry is a node of the outline drawn down, Pele's entry of verticalCauses.
type entry struct {
	node       *Node
	depth      int
	parent     int
	x, y, w, h float64
	label      text.Label
}

// verticalCauses lays the diagram out down, as an outline, as Pele's verticalCauses does: the
// effect and the categories framed, the causes under them indented by their depth, each with
// a line from its parent.
func verticalCauses(m *Model, o *layout.Options) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	measure := o.Measure()
	const pad = common.Padding
	room := max(140, o.MaxWidth-2*pad)
	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: common.TitleRoom(o), Face: text.Bold, Icons: o.Icons})
	var order []entry
	type pend struct {
		node          *Node
		depth, parent int
	}
	var pending []pend
	if m.Root != nil {
		pending = append(pending, pend{m.Root, 0, -1})
	}
	depth := 0
	for len(pending) > 0 {
		e := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		index := len(order)
		order = append(order, entry{node: e.node, depth: e.depth, parent: e.parent})
		depth = max(depth, e.depth)
		for i := len(e.node.Children) - 1; i >= 0; i-- {
			pending = append(pending, pend{e.node.Children[i], e.depth + 1, index})
		}
	}
	indent := min(20, max(2, (room-120)/float64(max(1, depth))))
	y := float64(pad)
	if title.Height != 0 {
		y += title.Height + 12
	}
	width := title.Width
	var nodes, edges []scene.Item
	for i := range order {
		e := &order[i]
		e.x = pad + float64(float64(e.depth)*indent)
		labelRoom := max(60, room-float64(float64(e.depth)*indent)-20)
		labelSize, face := small, text.Face(0)
		if e.depth < 2 {
			labelSize, face = size, text.Bold
		}
		e.label = common.BreakLabel(text.Layout(e.node.Text, text.Options{Measurer: measure, Size: labelSize, MaxWidth: labelRoom, Face: face, Icons: o.Icons}), measure, labelRoom)
		e.w = e.label.Width + 20
		e.h = e.label.Height + 12
		e.y = y
		y += e.h + 12
		width = max(width, e.x-pad+e.w)
		box := scene.Rect{X: e.x, Y: e.y, W: e.w, H: e.h}
		label := e.label
		it := scene.Item{
			Kind: scene.Text, Role: RoleCause, ID: e.node.Text,
			Box:   box,
			Label: &label, At: scene.Point{X: e.x + e.w/2, Y: e.y + e.h/2},
			Style: style.Style{Text: style.Fg},
		}
		if e.depth < 2 {
			// The effect and the categories are framed, in boxes with corners of their own.
			rx := 4.0
			it.Kind, it.Role, it.Shape = scene.Node, RoleCategory, "rect"
			if e.depth == 0 {
				rx = 12
				it.Role = RoleHead
			}
			it.Parts = []scene.Part{{Path: new(scene.Path).RoundRect(box, rx)}}
			it.Style = style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg}
		}
		nodes = append(nodes, it)
		if e.parent >= 0 {
			parent := &order[e.parent]
			gx := parent.x + indent/2
			from, mid, to := scene.Point{X: gx, Y: parent.y + parent.h}, scene.Point{X: gx, Y: e.y + e.h/2}, scene.Point{X: e.x, Y: e.y + e.h/2}
			path := new(scene.Path).MoveTo(from.X, from.Y).LineTo(mid.X, mid.Y).LineTo(to.X, to.Y)
			edges = append(edges, scene.Item{
				Kind: scene.Edge, Role: RoleLink, ID: e.node.Text,
				Box:   path.Bounds(),
				Parts: []scene.Part{{Path: path, Paint: scene.StrokeOnly}},
				Route: []scene.Point{from, mid, to},
				Style: style.Style{Fill: style.None, Stroke: style.Border},
			})
		}
	}
	totalWidth := math.Ceil(width + 2*pad)
	height := math.Ceil(max(pad+title.Height, y-12) + pad)
	sc := &scene.Scene{Type: "ishikawa", Width: totalWidth, Height: height, FontSize: size}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: totalWidth / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	sc.Items = append(sc.Items, edges...)
	sc.Items = append(sc.Items, nodes...)
	return sc
}
