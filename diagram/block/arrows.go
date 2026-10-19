package block

import (
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/scene"
)

// The directions an arrow block points in, as bits of a mask, Pele's RIGHT, LEFT, UP and
// DOWN.
const (
	dirRight = 1
	dirLeft  = 2
	dirUp    = 4
	dirDown  = 8
)

const (
	// overhang is how far a head reaches past each side of its shaft, Pele's OVERHANG.
	overhang = 7
	maxDepth = 20
	nub      = 12
)

// An arrowSize is the size of an arrow block about its label, and the shift of the label
// from its center, away from the heads, Pele's ArrowSize.
type arrowSize struct{ w, h, dx, dy float64 }

// arrowMask is the directions an arrow block points in, as Pele's arrowMask reads them.
// Mermaid compares directions exactly, so `( up )` draws no arrow there; here spaces are
// ignored.
func arrowMask(directions []string) int {
	mask := 0
	for _, d := range directions {
		switch chars.Trim(d) {
		case "right":
			mask |= dirRight
		case "left":
			mask |= dirLeft
		case "up":
			mask |= dirUp
		case "down":
			mask |= dirDown
		case "x":
			mask |= dirRight | dirLeft
		case "y":
			mask |= dirUp | dirDown
		}
	}
	return mask
}

// bit is 1 where the mask has the flag, and 0 where it does not.
func bit(mask, flag int) float64 {
	if mask&flag != 0 {
		return 1
	}
	return 0
}

// arrowAxis is the axis an arrow points along, 'x' or 'y', or 0 for one that points along
// both, as Pele's arrowAxis gives it. An arrow along one axis has a shaft and heads wider than
// it; one that points along both is a box with a small point on each side it names.
func arrowAxis(mask int) byte {
	if mask != 0 && mask&(dirUp|dirDown) == 0 {
		return 'x'
	}
	if mask != 0 && mask&(dirRight|dirLeft) == 0 {
		return 'y'
	}
	return 0
}

// sizeArrow is the size of an arrow block about a label tw by th, as Pele's arrowSize gives
// it.
func sizeArrow(mask int, tw, th float64) arrowSize {
	axis := arrowAxis(mask)
	across := bit(mask, dirLeft) - bit(mask, dirRight)
	along := bit(mask, dirUp) - bit(mask, dirDown)
	switch axis {
	case 'x':
		h := max(th+8, 14) + 2*overhang
		depth := min(h/2, maxDepth)
		heads := 1.0
		if mask == dirRight|dirLeft {
			heads = 2
		}
		return arrowSize{w: max(tw+12, 16) + float64(depth*heads), h: h, dx: (across * depth) / 2}
	case 'y':
		w := max(tw+16, 14) + 2*overhang
		depth := min(w/2, maxDepth)
		heads := 1.0
		if mask == dirUp|dirDown {
			heads = 2
		}
		return arrowSize{w: w, h: max(th+8, 16) + float64(depth*heads), dy: (along * depth) / 2}
	}
	return arrowSize{
		w:  max(tw+16, 24) + float64(nub*(bit(mask, dirLeft)+bit(mask, dirRight))),
		h:  max(th+8, 24) + float64(nub*(bit(mask, dirUp)+bit(mask, dirDown))),
		dx: (across * nub) / 2,
		dy: (along * nub) / 2,
	}
}

// drawArrow is the outline of an arrow block w by h centered on cx, cy, as Pele's drawArrow
// draws it: a polygon of the points Pele gives about the center, each moved to the center
// last, as Pele's SVG moves it.
func drawArrow(mask int, cx, cy, w, h float64) *scene.Path {
	axis := arrowAxis(mask)
	x := w / 2
	y := h / 2
	var p []float64
	switch axis {
	case 'x':
		depth := min(y, maxDepth)
		s := y - overhang
		x0, x1 := -x, x
		if mask&dirLeft != 0 {
			x0 = -x + depth
		}
		if mask&dirRight != 0 {
			x1 = x - depth
		}
		p = []float64{x0, -s, x1, -s}
		if mask&dirRight != 0 {
			p = append(p, x1, -y, x, 0, x1, y)
		}
		p = append(p, x1, s, x0, s)
		if mask&dirLeft != 0 {
			p = append(p, x0, y, -x, 0, x0, -y)
		}
	case 'y':
		depth := min(x, maxDepth)
		s := x - overhang
		y0, y1 := -y, y
		if mask&dirUp != 0 {
			y0 = -y + depth
		}
		if mask&dirDown != 0 {
			y1 = y - depth
		}
		p = []float64{s, y0, s, y1}
		if mask&dirDown != 0 {
			p = append(p, x, y1, 0, y, -x, y1)
		}
		p = append(p, -s, y1, -s, y0)
		if mask&dirUp != 0 {
			p = append(p, -x, y0, 0, -y, x, y0)
		}
	default:
		x0, x1, y0, y1 := -x, x, -y, y
		if mask&dirLeft != 0 {
			x0 = -x + nub
		}
		if mask&dirRight != 0 {
			x1 = x - nub
		}
		if mask&dirUp != 0 {
			y0 = -y + nub
		}
		if mask&dirDown != 0 {
			y1 = y - nub
		}
		mx := (x0 + x1) / 2
		my := (y0 + y1) / 2
		kx := min(10, (x1-x0)/2)
		ky := min(10, (y1-y0)/2)
		p = []float64{x0, y0}
		if mask&dirUp != 0 {
			p = append(p, mx-kx, y0, mx, -y, mx+kx, y0)
		}
		p = append(p, x1, y0)
		if mask&dirRight != 0 {
			p = append(p, x1, my-ky, x, my, x1, my+ky)
		}
		p = append(p, x1, y1)
		if mask&dirDown != 0 {
			p = append(p, mx+kx, y1, mx, y, mx-kx, y1)
		}
		p = append(p, x0, y1)
		if mask&dirLeft != 0 {
			p = append(p, x0, my+ky, -x, my, x0, my-ky)
		}
	}
	for i := 0; i < len(p); i += 2 {
		p[i] += cx
		p[i+1] += cy
	}
	return new(scene.Path).Polygon(p...)
}
