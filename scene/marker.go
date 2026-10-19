package scene

// MarkerSize is the length of an arrowhead.
const MarkerSize = 8

// Trim is how far an edge stops short of its end to leave room for a marker of the type.
func (t MarkerType) Trim() float64 {
	switch t {
	case "arrow_point":
		return MarkerSize - 1
	case "arrow_circle":
		return MarkerSize
	case "arrow_cross":
		return 4
	}
	return 0
}

// Outline is the marker's shape, as Pele draws it, and whether it is filled; a cross is
// stroked. It is nil for a marker of no type, or of one it does not know.
func (m Marker) Outline() (p *Path, filled bool) {
	x, y, dx, dy := m.At.X, m.At.Y, m.DX, m.DY
	nx, ny := -dy, dx
	switch m.Type {
	case "arrow_point":
		bx, by := x-dx*MarkerSize, y-dy*MarkerSize
		half := MarkerSize * 0.42
		return new(Path).Polygon(x, y, bx+nx*half, by+ny*half, bx-nx*half, by-ny*half), true
	case "arrow_circle":
		return new(Path).Ellipse(x-dx*4, y-dy*4, 3.5, 3.5), true
	case "arrow_cross":
		cx, cy := x-dx*4, y-dy*4
		const a = 3.5
		ux, uy := (dx+nx)*a, (dy+ny)*a
		vx, vy := (dx-nx)*a, (dy-ny)*a
		p = new(Path).MoveTo(cx-ux, cy-uy).LineTo(cx+ux, cy+uy).MoveTo(cx-vx, cy-vy).LineTo(cx+vx, cy+vy)
		return p, false
	}
	return nil, false
}
