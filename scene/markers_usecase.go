package scene

// The markers of use case diagrams.
func init() {
	// The hollow triangle of a generalization, with its tip at the end of the line, as
	// Pele's extension in its use case render draws it.
	markers["usecase.extension"] = markerKind{12, func(m Marker) []Part {
		x, y, dx, dy, _, _ := m.frame()
		const length, half = 12, 6
		bx, by := x-float64(dx*length), y-float64(dy*length)
		p := new(Path).Polygon(x, y,
			bx-float64(dy*half), by+float64(dx*half),
			bx+float64(dy*half), by-float64(dx*half))
		return []Part{{Path: p, Paint: Hollow}}
	}}
}
