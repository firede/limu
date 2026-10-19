package scene

// The markers of class diagrams, as Pele's diagrams/class/markers.ts draws them: the hollow
// triangle of an extension, the diamonds of a composition, filled, and of an aggregation,
// hollow, the arrowhead of a dependency, and the ring of a lollipop. Pele draws them inside
// the group of the edge, which fills nothing and strokes in the color of lines, so those
// it gives no fill of their own are strokes only: their insides show what is behind them, as
// the line stops at their backs.
const (
	classTriangle = 12 // the length of an extension's triangle
	classDiamond  = 16 // the length of a diamond
	classHalf     = 6  // half the width of a triangle, and one more than a diamond's
	classRing     = 5  // the radius of a lollipop's ring
)

func init() {
	markers["class.extension"] = markerKind{classTriangle, func(m Marker) []Part {
		x, y, dx, dy, nx, ny := m.frame()
		bx, by := x-float64(dx*classTriangle), y-float64(dy*classTriangle)
		p := new(Path).Polygon(x, y, bx+float64(nx*classHalf), by+float64(ny*classHalf),
			bx-float64(nx*classHalf), by-float64(ny*classHalf))
		return []Part{{Path: p, Paint: StrokeOnly}}
	}}
	markers["class.composition"] = markerKind{classDiamond, func(m Marker) []Part {
		return []Part{{Path: classDiamondPath(m), Paint: FillAndStroke}}
	}}
	markers["class.aggregation"] = markerKind{classDiamond, func(m Marker) []Part {
		return []Part{{Path: classDiamondPath(m), Paint: StrokeOnly}}
	}}
	markers["class.lollipop"] = markerKind{2 * classRing, func(m Marker) []Part {
		x, y, dx, dy, _, _ := m.frame()
		p := new(Path).Ellipse(x-float64(dx*classRing), y-float64(dy*classRing), classRing, classRing)
		return []Part{{Path: p, Paint: StrokeOnly}}
	}}
	// A dependency's arrowhead is Pele's arrow_point, its products kept apart from the sums
	// they feed.
	markers["class.dependency"] = markerKind{MarkerSize - 1, func(m Marker) []Part {
		x, y, dx, dy, nx, ny := m.frame()
		bx, by := x-float64(dx*MarkerSize), y-float64(dy*MarkerSize)
		half := MarkerSize * 0.42
		p := new(Path).Polygon(x, y, bx+float64(nx*half), by+float64(ny*half), bx-float64(nx*half), by-float64(ny*half))
		return []Part{{Path: p, Paint: FillOnly}}
	}}
}

// classDiamondPath is the outline of a composition's or an aggregation's diamond, its tip at
// the marker's.
func classDiamondPath(m Marker) *Path {
	x, y, dx, dy, nx, ny := m.frame()
	mx, my := x-float64(dx*classDiamond)/2, y-float64(dy*classDiamond)/2
	const half = classHalf - 1
	return new(Path).Polygon(x, y, mx+float64(nx*half), my+float64(ny*half),
		x-float64(dx*classDiamond), y-float64(dy*classDiamond), mx-float64(nx*half), my-float64(ny*half))
}
