package c4

import (
	"math"
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/text"
)

// The measures of the grid, as Pele's layout.ts has them.
const (
	shapeW    = 216
	shapeMinH = 64
	padX      = 14
	padY      = 12
	gapX      = 100
	gapY      = 88
	// Room for a relation to run along the inside of a boundary.
	groupPad = lane + 8
	groupGap = 48
	// Where a boundary's heading starts, from its top-left corner.
	headLeft = 16
	headTop  = 10
)

// forms are the shape keywords Mermaid accepts in $shape, $sprite or $tags, as names of the
// shared shapes.
var forms = map[string]string{
	"person":    "person",
	"box":       "rect",
	"rounded":   "rect",
	"component": "rect",
	"cylinder":  "cyl",
	"database":  "cyl",
	"db":        "cyl",
	"queue":     "h-cyl",
	"pipe":      "h-cyl",
}

// node is a shape laid out, Pele's Node.
type node struct {
	box
	shape *Shape
	// Type, name, technology, description.
	labels [4]text.Label
	textH  float64
}

// group is a boundary laid out, Pele's Group.
type group struct {
	box
	boundary *Boundary
	kids     []*group
	nodes    []*node
	// Name, type, description; none for the top level.
	labels []text.Label
	head   float64
	headW  float64
	headX  float64
}

// grid is the diagram laid out, Pele's Layout: the boundaries that are drawn, parents before
// children, the first the unnamed top level, and the box of each shape and boundary.
type grid struct {
	order []*group
	boxes map[*Element]*box
}

// formOf is the shared shape a shape is drawn as, as Pele's formOf picks it.
func formOf(shape *Shape) string {
	if f, ok := forms[chars.ToLower(shape.Shape)]; ok {
		return f
	}
	if f, ok := forms[chars.ToLower(shape.Sprite)]; ok {
		return f
	}
	if shape.Tags != "" {
		for _, tag := range strings.Split(shape.Tags, ",") {
			if f, ok := forms[chars.ToLower(chars.Trim(tag))]; ok {
				return f
			}
		}
	}
	switch kind := shape.Kind; {
	case strings.HasSuffix(kind, "person"):
		return "person"
	case strings.HasSuffix(kind, "_db"):
		return "cyl"
	case strings.HasSuffix(kind, "_queue"):
		return "h-cyl"
	}
	return "rect"
}

// labeler lays out labels as Pele's layoutLabel does, with the measurer and the icons of the
// options.
type labeler struct {
	measure text.Measurer
	icons   func(string) bool
}

// label lays out one label, as Pele's layoutLabel does when given the measurer, here with
// the icons of the options too.
func (l labeler) label(raw string, size, maxWidth float64, face text.Face) text.Label {
	return text.Layout(raw, text.Options{Measurer: l.measure, Size: size, MaxWidth: maxWidth, Bounded: true, Face: face, Icons: l.icons})
}

// measure lays out a shape's labels and sizes it about them, as Pele's measure does.
func measure(shape *Shape, l labeler, size, small, width float64) *node {
	form := formOf(shape)
	capW, capH := 0.0, 0.0
	switch form {
	case "h-cyl":
		capW = 36
	case "person":
		capH = head
	case "cyl":
		capH = 24
	}
	inner := width - 2*padX - capW
	techn := ""
	if shape.Techn != "" {
		techn = "[" + shape.Techn + "]"
	}
	n := &node{shape: shape}
	n.labels = [4]text.Label{
		l.label("«"+strings.ReplaceAll(shape.Kind, "_", " ")+"»", small-1, inner, 0),
		l.label(shape.Label, size, inner, text.Bold),
		l.label(techn, small, inner, 0),
		l.label(shape.Descr, small, inner, 0),
	}
	tw := 0.0
	if n.labels[3].Height > 0 {
		n.textH = 4
	}
	for _, label := range n.labels {
		tw = max(tw, label.Width)
		n.textH += label.Height
	}
	n.form = form
	n.w = max(width, tw+2*padX+capW)
	n.h = max(shapeMinH, n.textH+2*padY+capH)
	return n
}

// rows places boxes left to right in rows of perRow, from (0, 0), and returns the width and
// height the rows take, as Pele's rows does. Shapes are stretched to the height of their row;
// boundaries keep their own. perRow is a count as parseInt reads it, which may be past any
// int.
func rows(boxes []*box, perRow, gapX, gapY float64, stretch bool) (float64, float64) {
	width, y := 0.0, 0.0
	count := float64(len(boxes))
	for from := 0.0; from < count; from += perRow {
		to := int(min(count, from+perRow))
		x, rowH := 0.0, 0.0
		for i := int(from); i < to; i++ {
			rowH = max(rowH, boxes[i].h)
		}
		for i := int(from); i < to; i++ {
			b := boxes[i]
			b.x = x
			b.y = y
			if stretch {
				b.h = rowH
			}
			x += b.w + gapX
		}
		width = max(width, x-gapX)
		y += rowH + gapY
	}
	if len(boxes) > 0 {
		return width, y - gapY
	}
	return width, 0
}

// layoutC4 lays the diagram out on Mermaid's grid, as Pele's layoutC4 does: in each boundary,
// its shapes in rows of c4ShapeInRow, then the boundaries inside it in rows of
// c4BoundaryInRow, each sized around its own content. tight lays it out to fit room, with gap
// between its columns.
func layoutC4(db *DB, l labeler, size, small float64, tight bool, room, gap float64) grid {
	pad := float64(groupPad)
	if tight {
		pad = 12
	}
	groups := make(map[*Boundary]*group, len(db.Boundaries))
	all := make([]*group, 0, len(db.Boundaries))
	for _, b := range db.Boundaries {
		labelRoom := 4000.0
		if tight {
			labelRoom = max(80, room-2*headLeft-2*groupPad*2)
		}
		var labels []text.Label
		if b.Parent != nil {
			typ := ""
			if b.Type != "" {
				typ = "[" + b.Type + "]"
			}
			labels = []text.Label{
				l.label(b.Label, size, labelRoom, text.Bold),
				l.label(typ, small, labelRoom, 0),
				l.label(b.Descr, small, labelRoom, 0),
			}
		}
		h, headW := 0.0, 0.0
		for _, label := range labels {
			h += label.Height
			headW = max(headW, label.Width)
		}
		g := &group{boundary: b, labels: labels, headW: headW, headX: headLeft, box: box{form: "rect"}}
		if h > 0 {
			g.head = h + headTop + 6 + lane
		} else {
			g.head = pad
		}
		groups[b] = g
		all = append(all, g)
	}
	root := groups[db.Boundaries[0]]
	for _, g := range all {
		if parent := g.boundary.Parent; parent != nil {
			if p := groups[parent]; p != nil {
				p.kids = append(p.kids, g)
			}
		}
	}
	// A boundary that ends up inside itself is not reachable from the top, as in Mermaid.
	order := []*group{root}
	depths := map[*Boundary]int{root.boundary: 0}
	for i := 0; i < len(order); i++ {
		for _, kid := range order[i].kids {
			depths[kid.boundary] = depths[order[i].boundary] + 1
			order = append(order, kid)
		}
	}
	for _, shape := range db.Shapes {
		if shape.Parent == nil {
			continue
		}
		depth := float64(depths[shape.Parent])
		nodeWidth := float64(shapeW)
		if tight {
			nodeWidth = max(100, min(shapeW, room-float64(2*depth*pad)-32))
		}
		if g := groups[shape.Parent]; g != nil {
			g.nodes = append(g.nodes, measure(shape, l, size, small, nodeWidth))
		}
	}

	perRow := max(1, db.ShapeInRow)
	groupsPerRow := max(1, db.BoundaryInRow)
	kidsTop := map[*group]float64{}
	for i := len(order) - 1; i >= 0; i-- {
		g := order[i]
		columnGap := float64(gapX)
		if tight {
			columnGap = gap
		}
		// count is how many boxes go in a row: limit, or when tight as many of the widest box
		// as fit room, as the count of Pele's layoutC4 gives it.
		count := func(boxes []*box, limit, gap float64) float64 {
			if !tight {
				return limit
			}
			widest := 1.0
			for _, b := range boxes {
				widest = max(widest, b.w)
			}
			return max(1, min(limit, math.Floor((room+gap-2*pad)/(widest+gap))))
		}
		nodeBoxes := make([]*box, len(g.nodes))
		for k, n := range g.nodes {
			nodeBoxes[k] = &n.box
		}
		kidBoxes := make([]*box, len(g.kids))
		for k, kid := range g.kids {
			kidBoxes[k] = &kid.box
		}
		gridW, gridH := rows(nodeBoxes, count(nodeBoxes, perRow, columnGap), columnGap, gapY, true)
		kidsW, kidsH := rows(kidBoxes, count(kidBoxes, groupsPerRow, groupGap), groupGap, groupGap, false)
		between := 0.0
		if gridH > 0 && kidsH > 0 {
			between = groupGap
		}
		frame := 0.0
		if g != root {
			frame = pad
		}
		kidsTop[g] = gridH + between
		g.w = max(gridW, kidsW, g.headW+2*headLeft-2*frame) + 2*frame
		top := 0.0
		if frame != 0 {
			top = g.head
		}
		g.h = top + gridH + between + kidsH + frame
	}

	boxes := map[*Element]*box{}
	for _, g := range order {
		left, top := g.x, g.y
		if g != root {
			left, top = g.x+pad, g.y+g.head
		}
		for _, n := range g.nodes {
			n.x += left
			n.y += top
			boxes[&n.shape.Element] = &n.box
		}
		for _, kid := range g.kids {
			kid.x += left
			kid.y += top + kidsTop[g]
		}
		if g != root {
			boxes[&g.boundary.Element] = &g.box
		}
	}
	return grid{order: order, boxes: boxes}
}
