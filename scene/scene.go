// Package scene is a diagram laid out: a list of items, each a thing of the diagram, where it
// is and what it is, in the order they are drawn — but not how they look. A backend draws a
// scene: the svg package as SVG, others on screens of their own.
//
// An item says what it is by its Kind, its Role, as "flowchart.node", and, if it has an
// outline, by its Shape, as "diam". Its colors are tokens of the theme (see the style
// package), settled only when it is drawn, so a scene is drawn in any theme without being
// laid out again.
//
// An item also carries what a plain backend needs to draw it: its outline as Parts, paths
// and boxes; for an edge, the markers at its ends; a label laid out. A backend with a look of
// its own may draw an item its own way from its Role, Shape, Box and, for an edge, its Route
// instead.
package scene

import (
	"math"
	"slices"

	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// A Scene is a diagram laid out. Every number in a scene limu.Layout or Diagram.Layout gives
// that places or sizes something is finite (see Finite): where Pele would draw NaN, they give
// an error instead. A diagram package's own Layout makes no such promise.
type Scene struct {
	Type          string // the kind of diagram: "flowchart", "sequence"…
	Width, Height float64
	FontSize      float64 // of text with no size of its own
	Title, Desc   string  // for assistive technology: accTitle and accDescr
	Items         []Item  // in the order they are drawn
}

// A Kind is the sort of thing an item is.
type Kind uint8

const (
	Node  Kind = iota + 1 // a shape, most often with a label: a flowchart's node, an actor
	Group                 // a box that holds others: a subgraph, a loop, a lane
	Edge                  // a line from one thing to another, with markers at its ends
	Text                  // a label on its own: a title, an edge's label, an axis's
	Mark                  // a shape with no label: a lifeline, a tick, a slice of a pie
)

// A Role says which part of which diagram an item is, as "flowchart.node" or
// "sequence.lifeline". Styles are given by role.
type Role string

// A Shape names an item's outline, as Mermaid does: "rect", "rounded", "diam", "cyl"…
type Shape string

// An Item is a thing of a diagram, as it is drawn: a node, a group, an edge, a label on its
// own or a mark.
type Item struct {
	Kind  Kind
	Role  Role
	Shape Shape
	ID    string // the id the diagram's text gave it, if any
	// Class is the classes the diagram's author gave the item, by name: with class statements
	// and classDef's default, with :::name, or in a list of classes where the diagram takes
	// one, as a kanban card or a tree view's entry does. A class's style is already in Author.
	Class []string
	// Tags are what the diagram says the item is beyond its role, which Pele writes among the
	// classes of what it draws: a C4 element's kind ("person", "system_db"…) and "external",
	// a C4 relation's kind ("rel", "birel"…) and a deployment node's ("node", "nodeL",
	// "nodeR"); a gantt task's state ("done", "active", "crit", "milestone", "unscheduled");
	// the branch a gantt section or a mindmap's node or edge is in ("section-0"…); a wardley
	// component's sourcing ("build", "buy", "outsource", "market"); "reset" on an event
	// modeling frame that resets; and "clickable", which Mermaid gives what has a click.
	//
	// The svg backend writes the tags and then the classes as the element's classes, for a
	// host's CSS.
	Tags []string

	// Box is where the item is: the box of its shape, for a node or a group; for an edge, the
	// box its path lies in, the control points of its curves included, and for a mark, of
	// what it draws. A Text item's Box is
	// where the diagram sets its label, as an edge label's patch, and is empty where the
	// diagram gives the label no box of its own; LabelBox is where its label is drawn.
	Box Rect

	// Parts draw the outline; a backend that draws the shape its own way ignores them.
	Parts []Part

	// For an edge: the route it takes, as the corners of a line that runs straight between
	// them, and the markers at its ends. Its curves are in Parts. Every edge has a route; a
	// self-loop's, which no layout routes, is the corners its curve is drawn about
	// (Path.Corners).
	Route []Point
	Ends  [2]Marker

	// Label is drawn about At: centered on it, or with its lines starting or ending there,
	// as Align says, and turned by Rotate degrees about it.
	Label  *text.Label
	At     Point
	Align  Align
	Rotate float64

	// Picture is an image or an icon the item shows, as a flowchart's node may.
	Picture *Picture

	Style  style.Style // the diagram's own
	Author style.Style // what the author set with classDef and style lines

	Link    string // where the item leads, when it is a link
	Tooltip string
}

// Move moves the item by dx, dy: its box, parts, route, markers, label and picture. Its
// parts and paths are copied before they are moved, as other items may share them.
func (it *Item) Move(dx, dy float64) {
	box := func(r *Rect) {
		r.X += dx
		r.Y += dy
	}
	point := func(p *Point) {
		p.X += dx
		p.Y += dy
	}
	box(&it.Box)
	parts := slices.Clone(it.Parts)
	for k := range parts {
		if parts[k].Path == nil {
			box(&parts[k].Box)
		}
		if p := parts[k].Path; p != nil {
			q := &Path{Ops: p.Ops, Args: slices.Clone(p.Args)}
			for j := 0; j+1 < len(q.Args); j += 2 {
				q.Args[j] += dx
				q.Args[j+1] += dy
			}
			parts[k].Path = q
		}
	}
	it.Parts = parts
	route := slices.Clone(it.Route)
	for k := range route {
		point(&route[k])
	}
	it.Route = route
	for k := range it.Ends {
		point(&it.Ends[k].At)
	}
	point(&it.At)
	if it.Picture != nil {
		pic := *it.Picture
		box(&pic.Box)
		it.Picture = &pic
	}
}

// A Picture is an image, or an icon by name, drawn in a box.
type Picture struct {
	Image string // the image's address, checked as the diagram's options allow
	Icon  string // or the icon's name, as "fa:fa-car", drawn in the color of text
	Box   Rect
	Fit   bool // the image keeps its proportions in the box; it fills it otherwise
}

// Align is where a label's lines are about the point it is drawn at, across; down, the
// label is always centered on it.
type Align uint8

const (
	Middle Align = iota // each line centered on the point
	Start               // each line starting at it
	End                 // each line ending at it
)

// A Part is a piece of an item's outline. It is a path when Path is set, and Box and Rounded
// are then unused; when Path is nil, it is the box Box, its corners rounded by the style's
// radius when Rounded is set.
type Part struct {
	Path    *Path
	Box     Rect
	Rounded bool
	Paint   Painting
}

// LabelBox is the box the item's label is drawn in, laid about At as Align says and turned
// by Rotate degrees about it: the box the turned label lies in. It is empty for an item with
// no label.
func (it *Item) LabelBox() Rect {
	l := it.Label
	if l == nil || l.Empty() {
		return Rect{}
	}
	x := it.At.X - l.Width/2
	switch it.Align {
	case Start:
		x = it.At.X
	case End:
		x = it.At.X - l.Width
	}
	r := Rect{X: x, Y: it.At.Y - l.Height/2, W: l.Width, H: l.Height}
	if it.Rotate == 0 {
		return r
	}
	sin, cos := math.Sincos(it.Rotate * math.Pi / 180)
	turn := func(px, py float64) (float64, float64) {
		dx, dy := px-it.At.X, py-it.At.Y
		return it.At.X + dx*cos - dy*sin, it.At.Y + dx*sin + dy*cos
	}
	x0, y0 := math.Inf(1), math.Inf(1)
	x1, y1 := math.Inf(-1), math.Inf(-1)
	for _, c := range [4][2]float64{{r.X, r.Y}, {r.X + r.W, r.Y}, {r.X, r.Y + r.H}, {r.X + r.W, r.Y + r.H}} {
		px, py := turn(c[0], c[1])
		x0, y0, x1, y1 = min(x0, px), min(y0, py), max(x1, px), max(y1, py)
	}
	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// Painting is how a part is painted in its item's style.
type Painting uint8

const (
	// FillAndStroke fills the part and strokes its outline.
	FillAndStroke Painting = iota
	// StrokeOnly strokes the part only: an edge, or a detail on a shape, as the lid of a
	// cylinder.
	StrokeOnly
	// FillOnly fills the part and does not stroke it: the body of a datastore, which lines
	// close at its top and bottom only.
	FillOnly
	// Solid fills the part in the theme's color of lines, and strokes it: the dot of a final
	// state, the bar of a fork.
	Solid
	// Hollow fills the part with the theme's background, and strokes it: the ring of a final
	// state, which hides what is beneath it.
	Hollow
	// Shade fills the part with the theme's background and does not stroke it: a stripe of
	// an entity's table. When the item's author set a fill, it is the background at 60%
	// opacity, laid over the parts drawn before it, as Pele draws an entity's stripes.
	Shade
)

// A MarkerType is the kind of marker at an end of an edge, by Mermaid's names:
// "arrow_point", "arrow_circle", "arrow_cross"… The empty one is no marker: an end Pele
// draws no marker at has the empty type, whatever name the diagram's model gives it ("none",
// or "arrow_open" for an open line). Every type but the empty one is in the scene's table
// of markers, with a trim and an outline.
type MarkerType string

// A Marker is what an edge has at an end: its kind, where its tip is, and the way it
// points, as a unit vector.
type Marker struct {
	Type   MarkerType
	At     Point
	DX, DY float64
}
