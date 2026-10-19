package scene

import "github.com/firede/limu/style"

// The heads of sequence diagrams' messages that are not flowcharts', as Pele's sequence
// render draws them: an open head of two strokes, and half heads, a solid triangle or a
// single stroke, on one side of the line. A half head's side is named as seen along the way
// it points: "left" is the side of the normal a quarter turn to the left of its direction,
// as frame gives it. The line runs to the tip of all of them, so none trims it.
//
// The heads are drawn for messages, which run across; their points are offset from the
// line by the normal, which is down or up for a line across.

// The length of a head and how far its points are from the line.
const (
	seqHead   = MarkerSize
	seqSpread = 4.5
	seqStick  = 5
)

func init() {
	// Pele turns the corner of an open head round.
	markerJoins["sequence.open"] = style.JoinRound
	markers["sequence.open"] = markerKind{0, func(m Marker) []Part {
		x, y, dx, dy, nx, ny := m.frame()
		bx, by := x-float64(dx*seqHead), y-float64(dy*seqHead)
		// The first point is above the line for a head that points either way across.
		sx, sy := float64(nx*dx*seqSpread), float64(ny*dx*seqSpread)
		p := new(Path).MoveTo(bx-sx, by-sy).LineTo(x, y).LineTo(bx+sx, by+sy)
		return []Part{{Path: p, Paint: StrokeOnly}}
	}}
	half := func(side float64, stick bool) markerKind {
		return markerKind{0, func(m Marker) []Part {
			x, y, dx, dy, nx, ny := m.frame()
			bx, by := x-float64(dx*seqHead), y-float64(dy*seqHead)
			if stick {
				p := new(Path).MoveTo(x, y).LineTo(bx+float64(nx*side*seqStick), by+float64(ny*side*seqStick))
				return []Part{{Path: p, Paint: StrokeOnly}}
			}
			p := new(Path).Polygon(x, y, bx+float64(nx*side*seqSpread), by+float64(ny*side*seqSpread), bx, by)
			return []Part{{Path: p, Paint: FillOnly}}
		}}
	}
	markers["sequence.half_left"] = half(1, false)
	markers["sequence.half_right"] = half(-1, false)
	markers["sequence.stick_left"] = half(1, true)
	markers["sequence.stick_right"] = half(-1, true)
}
