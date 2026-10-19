package scene

// MarkerSize is the length of an arrowhead.
const MarkerSize = 8

// A markerKind is what a type of marker is: how far an edge stops short of its end for it,
// and its outline.
type markerKind struct {
	trim    float64
	outline func(m Marker) []Part
}

// markers are the types of marker the scene knows, by name. Pele's flowchart markers are
// here; a diagram with markers of its own adds them, under names with the diagram's prefix
// ("class.extension"), in a file of its own, from an init function.
var markers = map[MarkerType]markerKind{
	"arrow_point": {MarkerSize - 1, func(m Marker) []Part {
		x, y, dx, dy, nx, ny := m.frame()
		bx, by := x-float64(dx*MarkerSize), y-float64(dy*MarkerSize)
		half := MarkerSize * 0.42
		lx, ly := float64(nx*half), float64(ny*half)
		return []Part{{Path: new(Path).Polygon(x, y, bx+lx, by+ly, bx-lx, by-ly), Paint: FillOnly}}
	}},
	"arrow_circle": {MarkerSize, func(m Marker) []Part {
		x, y, dx, dy, _, _ := m.frame()
		return []Part{{Path: new(Path).Ellipse(x-float64(dx*4), y-float64(dy*4), 3.5, 3.5), Paint: FillOnly}}
	}},
	"arrow_cross": {4, func(m Marker) []Part {
		x, y, dx, dy, nx, ny := m.frame()
		cx, cy := x-float64(dx*4), y-float64(dy*4)
		const a = 3.5
		ux, uy := float64((dx+nx)*a), float64((dy+ny)*a)
		vx, vy := float64((dx-nx)*a), float64((dy-ny)*a)
		p := new(Path).MoveTo(cx-ux, cy-uy).LineTo(cx+ux, cy+uy).MoveTo(cx-vx, cy-vy).LineTo(cx+vx, cy+vy)
		return []Part{{Path: p, Paint: StrokeOnly}}
	}},
}

// frame is where a marker's tip is, the way it points, and the normal to that, a quarter
// turn to the left of it.
func (m Marker) frame() (x, y, dx, dy, nx, ny float64) {
	return m.At.X, m.At.Y, m.DX, m.DY, -m.DY, m.DX
}

// Trim is how far an edge stops short of its end to leave room for a marker of the type.
func (t MarkerType) Trim() float64 { return markers[t].trim }

// Outline is the marker's shape, as Pele draws it: parts painted in the color of the edge's
// line. FillOnly fills a part in it and FillAndStroke strokes it too; StrokeOnly strokes it
// only, and Hollow fills it with the background and strokes it, so that it hides the line
// beneath. It is nil for a marker of no type, or of one the scene does not know.
func (m Marker) Outline() []Part {
	if k, ok := markers[m.Type]; ok {
		return k.outline(m)
	}
	return nil
}
