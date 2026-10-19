package mindmap

import (
	"math"
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/draw"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The roles of a mindmap's items: "mindmap.title" for the title from front matter,
// "mindmap.root" for the root, "mindmap.node" for every other node, and "mindmap.edge" for
// the link from a node to each of its children.
const (
	RoleTitle scene.Role = "mindmap.title"
	RoleRoot  scene.Role = "mindmap.root"
	RoleNode  scene.Role = "mindmap.node"
	RoleEdge  scene.Role = "mindmap.edge"
)

// shapeNames are the shapes of Mermaid's node types, by index, as Pele's SHAPES has them. A
// node without delimiters has none: it is text on a line, except the root.
var shapeNames = [...]string{"", "rounded", "rect", "circle", "cloud", "bang", "hex"}

// rootShape is the root's shape when its text gives it none, Pele's ROOT_SHAPE.
const rootShape = "stadium"

// The spacing of a map, as Pele's render.ts has it.
const (
	gapX = 36
	gapY = 8
	// groupGap is between two subtrees of which at least one has children of its own.
	groupGap   = 18
	branchGapX = 52
	branchGapY = 28
	textPad    = 6
	pillPad    = 12
	tint       = 0.22
	iconGap    = 6
	// Running down, each level is set in by indent, and links leave a node stem from its
	// left edge.
	indent = 28
	stem   = 12
	bend   = 8
	rowGap = 10
)

// classNames is a class attribute's names, as Pele's classNames cleans them.
func classNames(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return !(r == '_' || r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	})
}

// outline is a shape w by h drawn about the origin and moved to cx, cy, as Pele draws a node's
// shape in the node's own frame.
func outline(shape string, cx, cy, w, h float64) []scene.Part {
	frame := scene.Item{Parts: draw.Outline(shape, 0, 0, w, h)}
	frame.Move(cx, cy)
	return frame.Parts
}

// Layout lays a mindmap out as a scene, as Pele's renderMindmap draws it: the first-level
// branches spread to the right of the root and to its left, split where the two sides are
// closest in height, each subtree in a band of its own. A map wider than the width the host
// has is laid out as an outline instead when that is narrower: the root at the top, and each
// level set in under its parent.
//
// Each first-level branch takes a series color, style.Series of its section, which its links
// are stroked in and its nodes outlined in and filled with at 22%. A node with no shape of its
// own is plain text where its branch ends, and text in a pill, a stadium, where the branch
// goes on from it. The root is in the theme's surface and border. Every item of a branch has
// the class "section-" and its section, as Pele's group about the branch has.
//
// A node's icon is a Picture of it, beside its text, when the host can draw it
// (layout.Options.Icons); one it cannot draw takes no room.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return lay(m, config, o, down) })
}

// lay is Pele's draw. down lays the map out as an outline, far narrower than branches
// spreading to both sides, and as tall as it needs to be.
func lay(m *Model, config diagram.Config, o *layout.Options, down bool) *scene.Scene {
	size := o.Size()
	measure := o.Measure()
	pad := o.Pad()
	conf, _ := config["mindmap"].(diagram.Config)
	wrap := 200.0
	if v, ok := conf["maxNodeWidth"].(float64); ok && v > 0 {
		wrap = min(v, 2000)
	}
	glyph := float64(size * 1.25)
	has := func(name string) bool { return name != "" && o.Icons != nil && o.Icons(text.DecodeEntities(name)) }

	nodes := m.Nodes
	n := len(nodes)
	parent := make([]int, n)
	depth := make([]int, n)
	for i, node := range nodes {
		for _, child := range node.Children {
			parent[child.ID] = i
			depth[child.ID] = depth[i] + 1
		}
	}
	// An outline has the width the host gives it, less what its level is set in.
	room := math.Inf(1)
	if down && o.MaxWidth > 0 {
		room = o.MaxWidth - 2*pad
	}
	labels := make([]text.Label, n)
	shapes := make([]string, n)
	w := make([]float64, n)
	h := make([]float64, n)
	dy := make([]float64, n)
	// Where links meet the node, measured down from its top: its middle.
	anchor := make([]float64, n)
	// Whether the host has the node's icon. One it does not have takes no room.
	drawn := make([]bool, n)

	for i, node := range nodes {
		drawn[i] = has(node.Icon)
		reserved := 0.0
		if drawn[i] {
			reserved = glyph + iconGap
		}
		fit := min(wrap, max(72, room-float64(float64(depth[i])*indent)-2*textPad-reserved))
		label := text.Layout(node.Descr, text.Options{Measurer: measure, Size: size, MaxWidth: fit, Bounded: true, Markdown: true, Icons: o.Icons})
		labels[i] = label
		tw := label.Width
		if drawn[i] {
			gap := 0.0
			if label.Width > 0 {
				gap = iconGap
			}
			tw += glyph + gap
		}
		th := max(label.Height, jsmath.Round(float64(size*1.5)))
		shape := ""
		if node.Type >= 0 && node.Type < len(shapeNames) {
			shape = shapeNames[node.Type]
		}
		if shape == "" && i == 0 {
			shape = rootShape
		}
		// In an outline the root heads a column, and the trunk drops from under its left end.
		// A circle or a cloud would stand wide of that, so the root is a pill there unless it
		// is a box already.
		if down && i == 0 && shape != "rect" && shape != "rounded" {
			shape = rootShape
		}
		shapes[i] = shape
		if shape == "" {
			// A node with no shape of its own is plain text when it ends a branch, and text in
			// a tinted pill when the branch goes on from it, which sets the levels apart.
			pill := len(node.Children) > 0
			p, extra := float64(textPad), 0.0
			if pill {
				p, extra = pillPad, 6
			}
			w[i] = tw + float64(2*p)
			h[i] = th + extra
			anchor[i] = h[i] / 2
		} else {
			s := draw.ShapeSize(shape, tw, th)
			w[i], h[i], dy[i] = s.W, s.H, s.DY
			anchor[i] = s.H / 2
		}
	}

	// Each subtree takes a band of height extent. rel is a subtree's offset among its
	// siblings, own the node's offset inside its band, and kids the offset of its children's
	// bands.
	extent := make([]float64, n)
	rel := make([]float64, n)
	own := make([]float64, n)
	kids := make([]float64, n)
	stacked := 0.0

	// stack stacks subtrees top to bottom and returns the height at which their parent's
	// anchor belongs.
	stack := func(list []*Node, from, to int, gap float64) float64 {
		y, last := 0.0, 0.0
		for k := from; k < to; k++ {
			c := list[k].ID
			rel[c] = y
			grouped := len(list[k].Children) > 0 || k+1 < to && len(list[k+1].Children) > 0
			last = gap
			if grouped {
				last = max(gap, groupGap)
			}
			y += extent[c] + last
		}
		stacked = y - last
		a, b := list[from].ID, list[to-1].ID
		return (rel[a] + own[a] + anchor[a] + rel[b] + own[b] + anchor[b]) / 2
	}

	// Nodes are in depth-first order, so going backwards visits children before their
	// parent.
	for i := n - 1; i > 0; i-- {
		children := nodes[i].Children
		if len(children) == 0 {
			extent[i] = h[i]
			continue
		}
		top := stack(children, 0, len(children), gapY) - anchor[i]
		low := min(0, top)
		extent[i] = max(stacked, top+h[i]) - low
		own[i] = top - low
		kids[i] = -low
	}

	x := make([]float64, n)
	y := make([]float64, n)
	band := make([]float64, n)
	side := make([]float64, n)
	// Running down: how far the trunk under each node has been drawn.
	trunk := make([]float64, n)

	if down {
		// Nodes are in depth-first order, which is the order of an outline's rows.
		row := 0.0
		for i := range nodes {
			if i > 0 && parent[i] == 0 {
				row += rowGap
			}
			x[i] = float64(float64(depth[i])*indent) + w[i]/2
			y[i] = row
			row += h[i] + rowGap
		}
	} else if n > 0 {
		// The first branches go right and the rest left, split where the two sides are
		// closest in height.
		branches := nodes[0].Children
		count := len(branches)
		total := 0.0
		for _, branch := range branches {
			total += extent[branch.ID]
		}
		split := count
		best := math.Inf(1)
		right := 0.0
		for k := 1; k < count; k++ {
			right += extent[branches[k-1].ID]
			diff := math.Abs(float64(2*right) - total + float64(float64(2*k-count)*branchGapY))
			if diff < best {
				best = diff
				split = k
			}
		}
		for from, to, dir := 0, split, 1.0; from < count; from, to, dir = to, count, -1 {
			middle := stack(branches, from, to, branchGapY)
			for k := from; k < to; k++ {
				c := branches[k].ID
				band[c] = rel[c] - middle
				side[c] = dir
			}
		}
		y[0] = -h[0] / 2
		for i, node := range nodes {
			if i > 0 {
				y[i] = band[i] + own[i]
			}
			gap := float64(branchGapX)
			if i > 0 {
				gap = gapX
			}
			for _, child := range node.Children {
				c := child.ID
				if i > 0 {
					side[c] = side[i]
					band[c] = band[i] + kids[i] + rel[c]
				}
				x[c] = x[i] + float64(side[c]*(w[i]/2+gap+w[c]/2))
			}
		}
	}

	var items, edges, body []scene.Item
	minX, maxX, minY, maxY := 0.0, 0.0, 0.0, 0.0
	var color style.Paint
	var branchClass string
	flush := func() {
		items = append(items, edges...)
		items = append(items, body...)
		edges, body = nil, nil
	}
	// inset is how far node i's shape lies in from its box on a side, none for a node with
	// no shape of its own.
	inset := func(shape string, i, on int) float64 {
		if shape == "" {
			return 0
		}
		return draw.ShapeInset(shape, w[i], h[i], on, 0)
	}

	// The root is drawn last, on top of the links that leave it.
	for k := 1; k <= n; k++ {
		i := k % n
		node := nodes[i]
		shape := shapes[i]
		label := labels[i]
		cx := x[i]
		top := y[i]
		minX = min(minX, cx-w[i]/2)
		maxX = max(maxX, cx+w[i]/2)
		minY = min(minY, top)
		maxY = max(maxY, top+h[i])

		if i == 0 || parent[i] == 0 {
			flush()
		}
		if i > 0 && parent[i] == 0 {
			color = style.Series(node.Section)
			branchClass = "section-" + strconv.Itoa(node.Section)
		}
		if i > 0 {
			p := parent[i]
			path := new(scene.Path)
			var route []scene.Point
			if down {
				// The link drops from under its parent and turns in to the node's left. It
				// starts where the link to the sibling above turned in, so each stretch of a
				// shared trunk has one color. Under the root's rounded end, the trunk starts
				// where that end has come down to the bottom.
				s := float64(stem)
				if p == 0 {
					s = min(max(stem, h[0]/2), indent-bend)
				}
				x1 := x[p] - w[p]/2 + min(s, w[p]/2)
				y1 := trunk[p]
				if y1 == 0 || math.IsNaN(y1) {
					y1 = y[p] + h[p]
				}
				left := cx - w[i]/2
				x2 := left + inset(shape, i, 3)
				y2 := top + anchor[i]
				b := max(0, min(bend, x2-x1, y2-y1))
				// The next link carries on from where this one leaves the trunk, so the trunk
				// has no gap.
				trunk[p] = y2 - b
				path.MoveTo(x1, y1).LineTo(x1, y2-b).QuadTo(x1, y2, x1+b, y2).LineTo(x2, y2)
				route = []scene.Point{{X: x1, Y: y1}, {X: x1, Y: y2}, {X: x2, Y: y2}}
			} else {
				dir := side[i]
				from := 1
				if dir < 0 {
					from = 3
				}
				x1 := x[p] + float64(dir*(w[p]/2-inset(shapes[p], p, from)))
				y1 := y[p] + anchor[p]
				near := cx - (dir*w[i])/2
				in := 0.0
				if shape != "" {
					in = float64(dir * inset(shape, i, 4-from))
				}
				x2 := near + in
				y2 := top + anchor[i]
				mid := (x1 + x2) / 2
				path.MoveTo(x1, y1).CubeTo(mid, y1, mid, y2, x2, y2)
				route = []scene.Point{{X: x1, Y: y1}, {X: x2, Y: y2}}
			}
			edges = append(edges, scene.Item{
				Kind: scene.Edge, Role: RoleEdge, Class: []string{branchClass},
				Box:   path.Bounds(),
				Parts: []scene.Part{{Path: path, Paint: scene.StrokeOnly}},
				Route: route,
				Style: style.Style{Stroke: color, StrokeWidth: style.N(1.5)},
			})
		}

		pill := shape == "" && len(node.Children) > 0
		cy := top + h[i]/2
		it := scene.Item{
			Kind: scene.Node, Role: RoleNode, Shape: scene.Shape(shape), ID: node.NodeID,
			Box: scene.Rect{X: cx - w[i]/2, Y: top, W: w[i], H: h[i]},
			// Every node off the root is filled with a tint of its branch's color and
			// outlined in it, whether its shape was given in the source or is the pill of a
			// node that branches. The root is neutral.
			Style: style.Style{Fill: color.Alpha(tint), Stroke: color, Text: style.Fg},
		}
		if i == 0 {
			it.Role = RoleRoot
			it.Style = style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg}
			it.Class = classNames(node.Class)
		} else {
			it.Class = append([]string{branchClass}, classNames(node.Class)...)
		}
		switch {
		case shape != "":
			it.Parts = outline(shape, cx, cy, w[i], h[i])
		case pill:
			it.Shape = rootShape
			it.Parts = outline(rootShape, cx, cy, w[i], h[i])
		default:
			it.Shape = "text"
			it.Style = style.Style{Text: style.Fg}
		}
		shift := 0.0
		if drawn[i] {
			gap := 0.0
			if label.Width > 0 {
				gap = iconGap
			}
			shift = (glyph + gap) / 2
			it.Picture = &scene.Picture{
				Icon: text.DecodeEntities(node.Icon),
				Box:  scene.Rect{X: cx + (-shift - label.Width/2), Y: cy + (dy[i] - glyph/2), W: glyph, H: glyph},
			}
		}
		if !label.Empty() {
			it.Label = &labels[i]
			if down && shape == "" {
				// An outline's text starts at the node's left, where centered lines would
				// look ragged.
				it.At = scene.Point{X: cx + (shift - label.Width/2), Y: cy + dy[i]}
				it.Align = scene.Start
			} else {
				it.At = scene.Point{X: cx + shift, Y: cy + dy[i]}
			}
		}
		if i > 0 {
			body = append(body, it)
		} else {
			items = append(items, it)
		}
	}

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: common.TitleRoom(o), Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 16
	}
	width := max(maxX-minX, title.Width)
	sc := &scene.Scene{
		Type:     "mindmap",
		Width:    math.Ceil(width + 2*pad),
		Height:   math.Ceil(maxY - minY + titleHeight + 2*pad),
		FontSize: size,
	}
	ox := pad - minX + (width-maxX+minX)/2
	oy := pad + titleHeight - minY
	for k := range items {
		items[k].Move(ox, oy)
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: sc.Width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	sc.Items = append(sc.Items, items...)
	return sc
}
