// Package scene is a diagram laid out: each of its parts, where it is and what it is, in
// the order it is drawn — but not how it looks. A backend draws a scene: the svg package as
// SVG, others on screens of their own.
//
// A part says what it is by its Role, as "flowchart.node", and, if it has an outline, by its
// Shape, as "diam". Its colors are tokens of the theme (see the style package), settled
// only when it is drawn, so a scene is drawn in any theme without being laid out again.
//
// A part also carries what a plain backend needs to draw it: the outline as Parts, the
// route of a line as a Path, the markers at its ends, a label laid out. A backend with a look
// of its own may draw a part its own way from its Role, Shape, Box and Route instead.
package scene

import (
	"slices"

	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// A Scene is a diagram laid out.
type Scene struct {
	Type          string // the kind of diagram: "flowchart", "sequence"…
	Width, Height float64
	FontSize      float64 // of text with no size of its own
	Title, Desc   string  // for assistive technology: accTitle and accDescr
	Items         []Item  // in the order they are drawn
}

// A Kind is the sort of thing a part is.
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

// A Shape names a part's outline, as Mermaid does: "rect", "rounded", "diam", "cyl"…
type Shape string

// An Item is a part of a diagram.
type Item struct {
	Kind  Kind
	Role  Role
	Shape Shape
	ID    string   // the id the diagram's text gave it, if any
	Class []string // classes its author gave it with classDef

	// Box is where the part is. For an edge, it holds its route.
	Box Rect

	// Parts draw the outline; a backend that draws the shape its own way ignores them.
	Parts []Part

	// For an edge: the route it takes, as the corners of a line that runs straight between
	// them, and the markers at its ends. Its curves are in Parts.
	Route []Point
	Ends  [2]Marker

	// Label is drawn about At: centered on it, or with its lines starting or ending there,
	// as Align says, and turned by Rotate degrees about it.
	Label  *text.Label
	At     Point
	Align  Align
	Rotate float64

	// Picture is an image or an icon the part shows, as a flowchart's node may.
	Picture *Picture

	Style  style.Style // the diagram's own
	Author style.Style // what the author set with classDef and style lines

	Link    string // where the part leads, when it is a link
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
		box(&parts[k].Box)
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

// A Part is a piece of a part's outline: a path, or, when Path is nil, the box Box, its
// corners rounded by the style's radius when Rounded is set.
type Part struct {
	Path    *Path
	Box     Rect
	Rounded bool
	Paint   Painting
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
	// an entity's table. Over a fill the item's author set, it is the background at 60% laid
	// over that fill, a tint of it.
	Shade
)

// A MarkerType is the kind of marker at an end of an edge, by Mermaid's names:
// "arrow_point", "arrow_circle", "arrow_cross"… The empty one is no marker.
type MarkerType string

// A Marker is what an edge has at an end: its kind, where its tip is, and the way it
// points, as a unit vector.
type Marker struct {
	Type   MarkerType
	At     Point
	DX, DY float64
}
