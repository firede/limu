package scene

// The crow's feet of entity relationship diagrams, as Pele's erMarker draws them: bars and
// feet stroked in the color of the line, and rings and a diamond that hide the line beneath
// them. A marker's tip is where the line meets the box, and it reaches back along the line
// from there, at most 18. The line runs under the marker to the box, so none trims it.
//
// The names are Mermaid's cardinalities in lower case, under the prefix "er.".

// The measures of Pele's erMarker: a bar's half length, a foot's half width at the box and
// its length, and a ring's radius.
const (
	erBar        = 5
	erFoot       = 6
	erFootLength = 10
	erRing       = 3.5
)

func init() {
	markers["er.only_one"] = markerKind{0, func(m Marker) []Part {
		p := new(Path)
		erBarAt(p, m, 6)
		erBarAt(p, m, 10)
		return []Part{{Path: p, Paint: StrokeOnly}}
	}}
	markers["er.zero_or_one"] = markerKind{0, func(m Marker) []Part {
		p := new(Path)
		erBarAt(p, m, 6)
		return []Part{{Path: p, Paint: StrokeOnly}, erRingAt(m, 13.5)}
	}}
	markers["er.one_or_more"] = markerKind{0, func(m Marker) []Part {
		p := new(Path)
		erFootAt(p, m)
		erBarAt(p, m, erFootLength+3.5)
		return []Part{{Path: p, Paint: StrokeOnly}}
	}}
	markers["er.zero_or_more"] = markerKind{0, func(m Marker) []Part {
		p := new(Path)
		erFootAt(p, m)
		return []Part{{Path: p, Paint: StrokeOnly}, erRingAt(m, erFootLength+erRing+0.5)}
	}}
	markers["er.md_parent"] = markerKind{0, func(m Marker) []Part {
		ax, ay := erAt(m, 0, 0)
		bx, by := erAt(m, 6, 4)
		cx, cy := erAt(m, 12, 0)
		dx, dy := erAt(m, 6, -4)
		return []Part{{Path: new(Path).Polygon(ax, ay, bx, by, cx, cy, dx, dy), Paint: Hollow}}
	}}
}

// erAt is the point back along the line from the marker's tip and side across it, to the
// right of the way it points, as Pele's erMarker places its points.
func erAt(m Marker, back, side float64) (float64, float64) {
	x, y, dx, dy := m.At.X, m.At.Y, m.DX, m.DY
	return x - float64(dx*back) - float64(dy*side), y - float64(dy*back) + float64(dx*side)
}

// erBarAt adds a bar across the line, back from the tip.
func erBarAt(p *Path, m Marker, back float64) {
	p.MoveTo(erAt(m, back, -erBar))
	p.LineTo(erAt(m, back, erBar))
}

// erFootAt adds a crow's foot: two toes from the box that meet on the line.
func erFootAt(p *Path, m Marker) {
	p.MoveTo(erAt(m, 0, -erFoot))
	p.LineTo(erAt(m, erFootLength, 0))
	p.LineTo(erAt(m, 0, erFoot))
}

// erRingAt is a ring on the line, back from the tip.
func erRingAt(m Marker, back float64) Part {
	cx, cy := m.At.X-float64(m.DX*back), m.At.Y-float64(m.DY*back)
	return Part{
		Path:  new(Path).Ellipse(cx, cy, erRing, erRing),
		Box:   Rect{X: cx - erRing, Y: cy - erRing, W: 2 * erRing, H: 2 * erRing},
		Paint: Hollow,
	}
}
