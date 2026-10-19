package scene

// The markers of requirement diagrams, as Pele's src/diagrams/requirement/render.ts draws
// them: both stroked in the color of the edge's line, filled with nothing.
const (
	// requirementContainsRadius is the radius of the circle of a containment mark.
	requirementContainsRadius = 6
	// requirementArrowLength and requirementArrowHalf are the length of an open arrowhead
	// and half its width.
	requirementArrowLength = 9
	requirementArrowHalf   = 4.5
)

func init() {
	// The containment mark, as Pele's containsMarker draws it: a circled plus whose rim
	// touches the tip, the circle beyond it. An edge stops short of its end by the circle's
	// width, so that it ends at the far side of the rim.
	markers["requirement.contains"] = markerKind{2 * requirementContainsRadius, func(m Marker) []Part {
		x, y, dx, dy, _, _ := m.frame()
		const r = requirementContainsRadius
		cx, cy := x-float64(dx*r), y-float64(dy*r)
		plus := new(Path).
			MoveTo(cx-float64(dx*r), cy-float64(dy*r)).LineTo(cx+float64(dx*r), cy+float64(dy*r)).
			MoveTo(cx+float64(dy*r), cy-float64(dx*r)).LineTo(cx-float64(dy*r), cy+float64(dx*r))
		return []Part{
			{Path: new(Path).Ellipse(cx, cy, r, r), Paint: StrokeOnly},
			{Path: plus, Paint: StrokeOnly},
		}
	}}
	// An open arrowhead with its tip at the end of the edge, as Pele's arrowMarker draws it.
	// The edge runs on to the tip.
	markers["requirement.arrow"] = markerKind{0, func(m Marker) []Part {
		x, y, dx, dy, _, _ := m.frame()
		bx, by := x-float64(dx*requirementArrowLength), y-float64(dy*requirementArrowLength)
		p := new(Path).
			MoveTo(bx-float64(dy*requirementArrowHalf), by+float64(dx*requirementArrowHalf)).
			LineTo(x, y).
			LineTo(bx+float64(dy*requirementArrowHalf), by-float64(dx*requirementArrowHalf))
		return []Part{{Path: p, Paint: StrokeOnly}}
	}}
}
