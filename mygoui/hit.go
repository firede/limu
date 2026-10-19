package mygoui

import (
	"math"

	"github.com/firede/limu/scene"
)

// ItemAt is the item of a scene at a point, in the scene's units, drawn last of those there,
// and so on top; nil when there is none. An item is there when its box holds the point, or
// the box of its label; an edge, and an item whose parts are all lines, when one of its
// lines passes within slop of the point, or one of its markers holds it.
func ItemAt(sc *scene.Scene, pt scene.Point, slop float64) *scene.Item {
	for i := len(sc.Items) - 1; i >= 0; i-- {
		if it := &sc.Items[i]; hits(it, pt, slop) {
			return it
		}
	}
	return nil
}

func hits(it *scene.Item, pt scene.Point, slop float64) bool {
	if labelHolds(it, pt) {
		return true
	}
	if it.Kind == scene.Edge || linesOnly(it) {
		for _, part := range it.Parts {
			if part.Path == nil {
				if near(part.Box, pt, slop) {
					return true
				}
				continue
			}
			for _, l := range part.Path.Flatten(0.5) {
				if lineNear(l, pt, slop) {
					return true
				}
			}
		}
		for _, m := range it.Ends {
			for _, part := range m.Outline() {
				b := part.Box
				if part.Path != nil {
					b = part.Path.Bounds()
				}
				if near(b, pt, slop) {
					return true
				}
			}
		}
		if len(it.Parts) == 0 {
			for i := 1; i < len(it.Route); i++ {
				if segmentNear(it.Route[i-1], it.Route[i], pt, slop) {
					return true
				}
			}
		}
		return false
	}
	if it.Box.W > 0 || it.Box.H > 0 {
		return it.Box.Contains(pt)
	}
	for _, part := range it.Parts {
		b := part.Box
		if part.Path != nil {
			b = part.Path.Bounds()
		}
		if b.Contains(pt) {
			return true
		}
	}
	return false
}

func linesOnly(it *scene.Item) bool {
	for _, part := range it.Parts {
		if part.Paint != scene.StrokeOnly {
			return false
		}
	}
	return len(it.Parts) > 0
}

// labelHolds reports whether the box of an item's label holds a point, turned with it.
func labelHolds(it *scene.Item, pt scene.Point) bool {
	l := it.Label
	if l == nil || l.Empty() {
		return false
	}
	if it.Rotate != 0 {
		// The point turned back about At, into the label's own axes.
		sin, cos := math.Sincos(-it.Rotate * math.Pi / 180)
		dx, dy := pt.X-it.At.X, pt.Y-it.At.Y
		pt = scene.Point{X: it.At.X + dx*cos - dy*sin, Y: it.At.Y + dx*sin + dy*cos}
	}
	x := it.At.X - l.Width/2
	switch it.Align {
	case scene.Start:
		x = it.At.X
	case scene.End:
		x = it.At.X - l.Width
	}
	return scene.Rect{X: x, Y: it.At.Y - l.Height/2, W: l.Width, H: l.Height}.Contains(pt)
}

// near reports whether a point is within slop of a box.
func near(b scene.Rect, pt scene.Point, slop float64) bool {
	return scene.Rect{X: b.X - slop, Y: b.Y - slop, W: b.W + 2*slop, H: b.H + 2*slop}.Contains(pt)
}

func lineNear(l scene.Polyline, pt scene.Point, slop float64) bool {
	n := len(l.Points)
	for i := 1; i < n; i++ {
		if segmentNear(l.Points[i-1], l.Points[i], pt, slop) {
			return true
		}
	}
	return l.Closed && segmentNear(l.Points[n-1], l.Points[0], pt, slop)
}

// segmentNear reports whether a point is within slop of the line from a to b.
func segmentNear(a, b, pt scene.Point, slop float64) bool {
	dx, dy := b.X-a.X, b.Y-a.Y
	t := 0.0
	if l := dx*dx + dy*dy; l > 0 {
		t = max(0, min(1, ((pt.X-a.X)*dx+(pt.Y-a.Y)*dy)/l))
	}
	return math.Hypot(a.X+dx*t-pt.X, a.Y+dy*t-pt.Y) <= slop
}
