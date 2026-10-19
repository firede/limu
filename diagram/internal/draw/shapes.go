// Package draw is the geometry of what diagrams draw: the outlines of shapes, the paths of
// edges, where titles go. It is ported from Pele's svg/shapes.ts, svg/edges.ts and
// svg/cluster.ts, function for function; where Pele writes SVG, it gives scene values.
//
// What layout or placement depends on — the sizes, insets and spans of shapes, the ends of
// routes, the tips of edges, the places of titles — is what Pele makes of it to the last bit.
// The outlines are Pele's, in absolute coordinates: where Pele draws with lines and curves,
// the numbers it writes, and where it writes arcs, circles and ellipses, which become cubic
// curves here, the same figures but not the same numbers. A box with the theme's corners is
// a box, as Pele's rect is, with its corners left to the backend.
//
// Go may fuse a multiplication and an addition into one step, rounded once, where
// JavaScript rounds twice. The products that feed sums are converted with float64(…),
// which keeps them apart, so the sums are JavaScript's.
package draw

import (
	"math"

	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/scene"
)

// The padding about a label in the shapes that hold one.
const (
	PadX = 18
	PadY = 12
)

// A Size is the size of a shape drawn about a label, and how far below the shape's center
// the label is drawn (above it, if DY is negative).
type Size struct{ W, H, DY float64 }

// Kept clear at each end of a straight side, for the corner.
const cornerClear = 12

type shapeDef struct {
	size func(tw, th float64) Size
	draw func(d *drawing, w, h float64)
	// Distance from the bounding box to the outline on a side: 0 top, 1 right, 2 bottom,
	// 3 left. t is how far along the side from its middle, to the right or down.
	inset func(w, h float64, side int, t float64) float64
	// How much of a side edges may spread along, centered on its middle: the top and
	// bottom, or the left and right when sideways. A shape without one keeps its edges at
	// the middle.
	span    func(w, h float64, sideways bool) float64
	noLabel bool
}

func flat(w, h float64, sideways bool) float64 {
	if sideways {
		return h - 2*cornerClear
	}
	return w - 2*cornerClear
}

// flatAcross is straight along the top and bottom except for cut(h) at each end, and not
// straight at the sides.
func flatAcross(cut func(h float64) float64) func(w, h float64, sideways bool) float64 {
	return func(w, h float64, sideways bool) float64 {
		if sideways {
			return 0
		}
		return w - float64(2*cut(h)) - 2*cornerClear
	}
}

// bulge is how far an ellipse's outline has fallen away from its bounding box, t from the
// middle of a side half long.
func bulge(t, half, depth float64) float64 {
	k := min(1, math.Abs(t)/half)
	return depth * (1 - math.Sqrt(1-float64(k*k)))
}

func roundInset(w, h float64, side int, t float64) float64 {
	if side&1 != 0 {
		return bulge(t, h/2, w/2)
	}
	return bulge(t, w/2, h/2)
}

func roundSpan(w, h float64, sideways bool) float64 {
	if sideways {
		return h * 0.6
	}
	return w * 0.6
}

// capRY is the height of the caps of a cylinder w wide.
func capRY(w float64) float64 { return min(12, max(6, w/14)) }

func capInset(w, _ float64, side int, t float64) float64 {
	if side&1 != 0 {
		return 0
	}
	return bulge(t, w/2, capRY(w))
}

func fixed(w, h float64) func(tw, th float64) Size {
	return func(float64, float64) Size { return Size{w, h, 0} }
}

func triSize(tw, th float64, flip bool) Size {
	w := max(float64(1.6*tw)+40, 80)
	h := (th + 10) / (1 - (tw+8)/w)
	dy := h/2 - 10 - th/2
	if flip {
		dy = -dy
	}
	return Size{w, h, dy}
}

func leanInset(_, h float64, side int, _ float64) float64 {
	if side&1 != 0 {
		return h / 8
	}
	return 0
}

func box(tw, th float64) Size {
	return Size{max(tw+2*PadX, 2*PadX+12), th + 2*PadY, 0}
}

// sides gives an inset of a on the left and right, and b on the top and bottom.
func sides(a, b float64) func(w, h float64, side int, t float64) float64 {
	return func(_, _ float64, side int, _ float64) float64 {
		if side&1 != 0 {
			return a
		}
		return b
	}
}

// onSide gives an inset of v on one side, and none on the others.
func onSide(which int, v float64) func(w, h float64, side int, t float64) float64 {
	return func(_, _ float64, side int, _ float64) float64 {
		if side == which {
			return v
		}
		return 0
	}
}

// The sizes of the many shapes whose height is the label's and its padding.
func padded(extraW, extraH, dy float64) func(tw, th float64) Size {
	return func(tw, th float64) Size { return Size{tw + 2*PadX + extraW, th + 2*PadY + extraH, dy} }
}

// shapes are Pele's SHAPES, by name.
var shapes = map[string]*shapeDef{
	"rect": {size: box, span: flat, draw: func(d *drawing, w, h float64) { d.rounded(-w/2, -h/2, w, h) }},
	"rounded": {size: box, span: flat, draw: func(d *drawing, w, h float64) {
		d.rect(-w/2, -h/2, w, h, min(12, h/2), scene.FillAndStroke)
	}},
	"stadium": {
		inset: func(_, h float64, side int, t float64) float64 {
			if side&1 != 0 {
				return bulge(t, h/2, h/2)
			}
			return 0
		},
		span: func(w, h float64, sideways bool) float64 {
			if sideways {
				return h * 0.6
			}
			return w - h
		},
		size: func(tw, th float64) Size {
			h := th + 2*PadY
			return Size{tw + float64(h*0.5) + 2*PadX - 8, h, 0}
		},
		draw: func(d *drawing, w, h float64) { d.rect(-w/2, -h/2, w, h, h/2, scene.FillAndStroke) },
	},
	"fr-rect": {span: flat, size: padded(16, 0, 0), draw: func(d *drawing, w, h float64) {
		x := w/2 - 8
		d.rounded(-w/2, -h/2, w, h)
		d.pen().M(-x, -h/2).V(h/2).M(x, -h/2).V(h / 2).done(scene.StrokeOnly)
	}},
	"cyl": {
		inset: capInset,
		span: func(w, h float64, sideways bool) float64 {
			if sideways {
				return h - 2*cornerClear - 12
			}
			return w * 0.6
		},
		size: func(tw, th float64) Size {
			w := max(tw+2*PadX, 56)
			ry := capRY(w)
			return Size{w, th + 2*PadY + float64(2*ry), ry / 2}
		},
		draw: func(d *drawing, w, h float64) {
			rx, ry := w/2, capRY(w)
			top, bottom := -h/2+ry, h/2-ry
			d.pen().M(-rx, top).A(rx, ry, false, true, rx, top).V(bottom).A(rx, ry, false, true, -rx, bottom).Z().done(scene.FillAndStroke)
			d.pen().M(-rx, top).A(rx, ry, false, false, rx, top).done(scene.StrokeOnly)
		},
	},
	"circle": {
		inset: roundInset,
		span:  roundSpan,
		size: func(tw, th float64) Size {
			r := max(math.Sqrt(float64(tw*tw)+float64(th*th))+16, 44)
			return Size{r, r, 0}
		},
		draw: func(d *drawing, w, _ float64) { d.circle(0, 0, w/2, scene.FillAndStroke) },
	},
	"dbl-circ": {
		inset: roundInset,
		span:  roundSpan,
		size: func(tw, th float64) Size {
			r := max(math.Sqrt(float64(tw*tw)+float64(th*th))+26, 54)
			return Size{r, r, 0}
		},
		draw: func(d *drawing, w, _ float64) {
			d.circle(0, 0, w/2, scene.FillAndStroke)
			d.circle(0, 0, w/2-5, scene.StrokeOnly)
		},
	},
	"ellipse": {
		inset: roundInset,
		span:  roundSpan,
		size: func(tw, th float64) Size {
			return Size{float64(tw*1.3) + 2*PadX, float64(th*1.2) + 2*PadY, 0}
		},
		draw: func(d *drawing, w, h float64) { d.ellipse(0, 0, w/2, h/2, scene.FillAndStroke) },
	},
	"diam": {
		inset: func(w, h float64, side int, t float64) float64 {
			if side&1 != 0 {
				return (math.Abs(t) * w) / h
			}
			return (math.Abs(t) * h) / w
		},
		span: roundSpan,
		size: func(tw, th float64) Size {
			a, b := tw+16, th+8
			return Size{a + float64(1.6*b), a/1.6 + b, 0}
		},
		draw: func(d *drawing, w, h float64) { d.poly(scene.FillAndStroke, 0, -h/2, w/2, 0, 0, h/2, -w/2, 0) },
	},
	"hex": {
		inset: func(_, _ float64, side int, t float64) float64 {
			if side&1 != 0 {
				return math.Abs(t) / 2
			}
			return 0
		},
		span: func(w, h float64, sideways bool) float64 {
			if sideways {
				return h * 0.5
			}
			return w - h/2 - cornerClear
		},
		size: func(tw, th float64) Size {
			h := th + 2*PadY
			return Size{tw + 2*PadX + h/2, h, 0}
		},
		draw: func(d *drawing, w, h float64) {
			m := h / 4
			d.poly(scene.FillAndStroke, -w/2+m, -h/2, w/2-m, -h/2, w/2, 0, w/2-m, h/2, -w/2+m, h/2, -w/2, 0)
		},
	},
	"odd": {
		inset: func(_, h float64, side int, _ float64) float64 {
			if side == 3 {
				return h / 4
			}
			return 0
		},
		span: flatAcross(func(float64) float64 { return 0 }),
		size: func(tw, th float64) Size {
			h := th + 2*PadY
			return Size{tw + 2*PadX + h/4, h, 0}
		},
		draw: func(d *drawing, w, h float64) {
			notch := h / 4
			d.poly(scene.FillAndStroke, -w/2, -h/2, w/2, -h/2, w/2, h/2, -w/2, h/2, -w/2+notch, 0)
		},
	},
	"lean-r": leaning(func(d *drawing, w, h, s float64) {
		d.poly(scene.FillAndStroke, -w/2+s, -h/2, w/2, -h/2, w/2-s, h/2, -w/2, h/2)
	}),
	"lean-l": leaning(func(d *drawing, w, h, s float64) {
		d.poly(scene.FillAndStroke, -w/2, -h/2, w/2-s, -h/2, w/2, h/2, -w/2+s, h/2)
	}),
	"trap-b": leaning(func(d *drawing, w, h, s float64) {
		d.poly(scene.FillAndStroke, -w/2+s, -h/2, w/2-s, -h/2, w/2, h/2, -w/2, h/2)
	}),
	"trap-t": leaning(func(d *drawing, w, h, s float64) {
		d.poly(scene.FillAndStroke, -w/2, -h/2, w/2, -h/2, w/2-s, h/2, -w/2+s, h/2)
	}),
	// Pele draws a text shape as a box with neither fill nor stroke, which shows nothing;
	// here it has no pieces.
	"text": {
		span: flat,
		size: func(tw, th float64) Size { return Size{tw + 8, th + 8, 0} },
		draw: func(*drawing, float64, float64) {},
	},
	"sm-circ": {noLabel: true, inset: roundInset, span: roundSpan, size: fixed(14, 14),
		draw: func(d *drawing, w, _ float64) { d.circle(0, 0, w/2, scene.FillAndStroke) }},
	"f-circ": {noLabel: true, inset: roundInset, span: roundSpan, size: fixed(14, 14),
		draw: func(d *drawing, w, _ float64) { d.circle(0, 0, w/2, scene.Solid) }},
	"fr-circ": {noLabel: true, inset: roundInset, span: roundSpan, size: fixed(18, 18),
		draw: func(d *drawing, w, _ float64) {
			d.circle(0, 0, w/2, scene.FillAndStroke)
			d.circle(0, 0, w/2-4, scene.Solid)
		}},
	"fork": {
		noLabel: true,
		span: func(w, h float64, sideways bool) float64 {
			if sideways {
				return h - 8
			}
			return w - 8
		},
		size: fixed(72, 8),
		draw: func(d *drawing, w, h float64) { d.rect(-w/2, -h/2, w, h, 2, scene.Solid) },
	},
	"notch-rect": {size: box, span: flat, draw: func(d *drawing, w, h float64) {
		const c = 10
		d.poly(scene.FillAndStroke, -w/2+c, -h/2, w/2, -h/2, w/2, h/2, -w/2, h/2, -w/2, -h/2+c)
	}},
	"lin-rect": {span: flat, size: padded(8, 0, 0), draw: func(d *drawing, w, h float64) {
		d.rounded(-w/2, -h/2, w, h)
		d.pen().M(-w/2+8, -h/2).V(h / 2).done(scene.StrokeOnly)
	}},
	"div-rect": {span: flat, size: padded(0, 8, 4), draw: func(d *drawing, w, h float64) {
		d.rounded(-w/2, -h/2, w, h)
		d.pen().M(-w/2, -h/2+8).H(w / 2).done(scene.StrokeOnly)
	}},
	"win-pane": {span: flat, size: padded(8, 8, 4), draw: func(d *drawing, w, h float64) {
		d.rounded(-w/2, -h/2, w, h)
		d.pen().M(-w/2+8, -h/2).V(h/2).M(-w/2, -h/2+8).H(w / 2).done(scene.StrokeOnly)
	}},
	"tri": {
		inset: func(w, _ float64, side int, _ float64) float64 {
			if side&1 != 0 {
				return w / 4
			}
			return 0
		},
		size: func(tw, th float64) Size { return triSize(tw, th, false) },
		draw: func(d *drawing, w, h float64) { d.poly(scene.FillAndStroke, 0, -h/2, w/2, h/2, -w/2, h/2) },
	},
	"flip-tri": {
		inset: func(w, _ float64, side int, _ float64) float64 {
			if side&1 != 0 {
				return w / 4
			}
			return 0
		},
		size: func(tw, th float64) Size { return triSize(tw, th, true) },
		draw: func(d *drawing, w, h float64) { d.poly(scene.FillAndStroke, -w/2, -h/2, w/2, -h/2, 0, h/2) },
	},
	"sl-rect": {inset: onSide(0, 5), size: padded(0, 10, 5), draw: func(d *drawing, w, h float64) {
		d.poly(scene.FillAndStroke, -w/2, -h/2+10, w/2, -h/2, w/2, h/2, -w/2, h/2)
	}},
	"notch-pent": {span: flat, size: padded(0, 0, 0), draw: func(d *drawing, w, h float64) {
		const c = 10
		d.poly(scene.FillAndStroke, -w/2+c, -h/2, w/2-c, -h/2, w/2, -h/2+c, w/2, h/2, -w/2, h/2, -w/2, -h/2+c)
	}},
	"hourglass": {
		noLabel: true,
		inset: func(w, _ float64, side int, _ float64) float64 {
			if side&1 != 0 {
				return w / 2
			}
			return 0
		},
		size: fixed(32, 40),
		draw: func(d *drawing, w, h float64) {
			d.poly(scene.FillAndStroke, -w/2, -h/2, w/2, -h/2, -w/2, h/2, w/2, h/2)
		},
	},
	"delay": {
		size: func(tw, th float64) Size {
			h := th + 2*PadY
			return Size{tw + 2*PadX + h/4, h, 0}
		},
		draw: func(d *drawing, w, h float64) {
			r := h / 2
			d.pen().M(-w/2, -r).H(w/2-r).A(r, r, false, true, w/2-r, r).H(-w / 2).Z().done(scene.FillAndStroke)
		},
	},
	"h-cyl": {
		size: func(tw, th float64) Size {
			h := th + 2*PadY
			rx := min(12, h/4)
			return Size{tw + 2*PadX + float64(2*rx), h, 0}
		},
		draw: func(d *drawing, w, h float64) {
			ry, rx := h/2, min(12, h/4)
			l, r := -w/2+rx, w/2-rx
			d.pen().M(l, -ry).H(r).A(rx, ry, false, true, r, ry).H(l).A(rx, ry, false, true, l, -ry).Z().done(scene.FillAndStroke)
			d.pen().M(r, -ry).A(rx, ry, false, false, r, ry).done(scene.StrokeOnly)
		},
	},
	"cross-circ": {noLabel: true, inset: roundInset, span: roundSpan, size: fixed(44, 44),
		draw: func(d *drawing, w, _ float64) {
			r := w / 2
			k := r * (math.Sqrt2 / 2)
			d.circle(0, 0, r, scene.FillAndStroke)
			d.pen().M(-k, -k).L(k, k).M(k, -k).L(-k, k).done(scene.StrokeOnly)
		}},
	"st-rect": {
		span: func(w, h float64, sideways bool) float64 {
			if sideways {
				return h - 2*cornerClear - 8
			}
			return w - 2*cornerClear - 8
		},
		size: padded(8, 8, 4),
		draw: func(d *drawing, w, h float64) {
			bw, bh := w-8, h-8
			for i := range 3 {
				d.rounded(-w/2+float64((2-i)*4), -h/2+float64(i*4), bw, bh)
			}
		},
	},
	"doc": {inset: onSide(2, 10), size: padded(0, 10, -5), draw: func(d *drawing, w, h float64) { d.wave(w, h) }},
	"flag": {inset: sides(0, 8), size: padded(0, 16, 0), draw: func(d *drawing, w, h float64) {
		x, y, q := w/2, h/2-8, w/4
		d.pen().M(-x, -y).q(q, -10, 2*q, 0).t(2*q, 0).V(y).q(-q, 10, -2*q, 0).t(-2*q, 0).Z().done(scene.FillAndStroke)
	}},
	"datastore": {size: box, span: flat, draw: func(d *drawing, w, h float64) {
		d.add(scene.Part{Box: scene.Rect{X: d.cx - w/2, Y: d.cy - h/2, W: w, H: h}}, scene.FillOnly)
		d.pen().M(-w/2, -h/2).H(w/2).M(-w/2, h/2).H(w / 2).done(scene.StrokeOnly)
	}},
	"folder": {
		size: func(tw, th float64) Size { return Size{max(tw+2*PadX, 64), th + 2*PadY + 8, 4} },
		draw: func(d *drawing, w, h float64) {
			x, y := w/2, h/2
			tab := min(w*0.4, 48)
			d.pen().M(-x, -y).H(-x+tab).L(-x+tab+8, -y+8).H(x).V(y).H(-x).Z().done(scene.FillAndStroke)
		},
	},
	"bucket": {
		size: padded(20, 10, 3),
		inset: func(_, h float64, side int, _ float64) float64 {
			if side&1 != 0 {
				return h / 12
			}
			return 0
		},
		draw: func(d *drawing, w, h float64) {
			x, y := w/2, h/2
			const ry = 5
			taper := h / 6
			d.pen().M(-x, -y+ry).A(x, ry, false, true, x, -y+ry).L(x-taper, y-ry).A(x-taper, ry, false, true, -x+taper, y-ry).Z().done(scene.FillAndStroke)
			d.pen().M(-x, -y+ry).A(x, ry, false, false, x, -y+ry).done(scene.StrokeOnly)
		},
	},
	"console": {
		span: flat,
		size: func(tw, th float64) Size { return Size{max(tw+2*PadX, 72), th + 2*PadY + 12, 6} },
		draw: func(d *drawing, w, h float64) {
			d.rounded(-w/2, -h/2, w, h)
			d.pen().M(-w/2, -h/2+12).H(w/2).M(-w/2+7, -h/2+4).l(3, 2.5).l(-3, 2.5).M(-w/2+13, -h/2+9).h(4).done(scene.StrokeOnly)
		},
	},
	"browser": {
		span: flat,
		size: func(tw, th float64) Size { return Size{max(tw+2*PadX, 72), th + 2*PadY + 12, 6} },
		draw: func(d *drawing, w, h float64) {
			d.rounded(-w/2, -h/2, w, h)
			d.pen().M(-w/2, -h/2+12).H(w / 2).done(scene.StrokeOnly)
			for i := range 3 {
				d.circle(-w/2+8+float64(i*7), -h/2+6, 1.5, scene.StrokeOnly)
			}
		},
	},
	"person": {
		size: func(tw, th float64) Size { return Size{max(tw+2*PadX, 56), th + 2*PadY + 26, 13} },
		draw: func(d *drawing, w, h float64) {
			top := -h / 2
			d.rect(-w/2, top+26, w, h-26, 12, scene.FillAndStroke)
			d.circle(0, top+14, 13, scene.FillAndStroke)
		},
	},
	"bang": {
		size:  padded(28, 24, 0),
		inset: func(float64, float64, int, float64) float64 { return 3 },
		draw: func(d *drawing, w, h float64) {
			const spikes = 14
			pts := make([]float64, 0, 4*spikes)
			for i := range 2 * spikes {
				angle := (math.Pi*float64(i))/spikes - math.Pi/2
				k := 0.82
				if i%2 == 0 {
					k = 1
				}
				pts = append(pts, jsmath.Cos(angle)*(w/2)*k, jsmath.Sin(angle)*(h/2)*k)
			}
			d.poly(scene.FillAndStroke, pts...)
		},
	},
	"cloud": {
		size:  padded(32, 24, 0),
		inset: sides(2, 1),
		draw: func(d *drawing, w, h float64) {
			x, y := w/2, h/2
			r := float64(y * 0.55)
			d.pen().M(-x+r, y).
				A(r, r, false, true, -x+float64(r*0.6), -y*0.1).
				A(x*0.42, y*0.7, false, true, -x*0.1, -y*0.72).
				A(x*0.4, y*0.62, false, true, x*0.62, -y*0.42).
				A(r*1.05, r*1.05, false, true, x-r, y).Z().done(scene.FillAndStroke)
		},
	},
	// Pele draws the box of a brace with neither fill nor stroke; here it has no piece.
	"brace":   {size: padded(12, 0, 0), draw: func(d *drawing, w, h float64) { d.brace(-w/2+12, h, -1) }},
	"brace-r": {size: padded(12, 0, 0), draw: func(d *drawing, w, h float64) { d.brace(w/2-12, h, 1) }},
	"braces": {size: padded(24, 0, 0), draw: func(d *drawing, w, h float64) {
		d.brace(-w/2+12, h, -1)
		d.brace(w/2-12, h, 1)
	}},
	"bolt": {
		noLabel: true,
		size:    fixed(36, 56),
		inset: func(w, _ float64, side int, _ float64) float64 {
			if side&1 != 0 {
				return w / 4
			}
			return 0
		},
		draw: func(d *drawing, w, h float64) {
			d.poly(scene.FillAndStroke, w*0.15, -h/2, -w/2, h*0.08, -w*0.05, h*0.08, -w*0.15, h/2, w/2, -h*0.08, w*0.05, -h*0.08)
		},
	},
	"lin-cyl": {
		inset: capInset,
		span: func(w, h float64, sideways bool) float64 {
			if sideways {
				return h - 2*cornerClear - 20
			}
			return w * 0.6
		},
		size: func(tw, th float64) Size {
			w := max(tw+2*PadX, 56)
			ry := capRY(w)
			return Size{w, th + 2*PadY + float64(2*ry) + 8, ry/2 + 4}
		},
		draw: func(d *drawing, w, h float64) {
			rx, ry := w/2, capRY(w)
			top, bottom := -h/2+ry, h/2-ry
			d.pen().M(-rx, top).A(rx, ry, false, true, rx, top).V(bottom).A(rx, ry, false, true, -rx, bottom).Z().done(scene.FillAndStroke)
			d.pen().M(-rx, top).A(rx, ry, false, false, rx, top).M(-rx, top+8).A(rx, ry, false, false, rx, top+8).done(scene.StrokeOnly)
		},
	},
	"curv-trap": {
		size: func(tw, th float64) Size {
			h := th + 2*PadY
			return Size{tw + 2*PadX + float64(h*0.6), h, 0}
		},
		draw: func(d *drawing, w, h float64) {
			x, y, k := w/2, h/2, h*0.3
			d.pen().M(-x+k, -y).H(x-k).A(k, y, false, true, x-k, y).H(-x+k).L(-x, 0).Z().done(scene.FillAndStroke)
		},
	},
	"docs": {
		size:  padded(8, 18, -1),
		inset: onSide(2, 10),
		draw: func(d *drawing, w, h float64) {
			for i := range 3 {
				moved := &drawing{cx: d.cx + float64((1-i)*4), cy: d.cy + float64((i-1)*4)}
				moved.wave(w-8, h-8)
				d.pieces = append(d.pieces, moved.pieces...)
			}
		},
	},
	"bow-rect": {
		size: func(tw, th float64) Size {
			h := th + 2*PadY
			return Size{tw + 2*PadX + float64(h*0.4), h, 0}
		},
		inset: func(_, h float64, side int, _ float64) float64 {
			if side == 1 {
				return h * 0.2
			}
			return 0
		},
		draw: func(d *drawing, w, h float64) {
			x, y, k := w/2, h/2, h*0.2
			d.pen().M(-x+k, -y).H(x).A(k, y, false, false, x, y).H(-x+k).A(k, y, false, true, -x+k, -y).Z().done(scene.FillAndStroke)
		},
	},
	"tag-rect": {size: box, span: flat, draw: func(d *drawing, w, h float64) {
		d.rounded(-w/2, -h/2, w, h)
		d.pen().M(w/2-12, h/2).L(w/2, h/2-12).done(scene.StrokeOnly)
	}},
	"tag-doc": {size: padded(0, 10, -5), inset: onSide(2, 10), draw: func(d *drawing, w, h float64) {
		d.wave(w, h)
		d.pen().M(w/2-12, h/2-14).L(w/2, h/2-26).done(scene.StrokeOnly)
	}},
	"lin-doc": {size: padded(8, 10, -5), inset: onSide(2, 10), draw: func(d *drawing, w, h float64) {
		d.wave(w, h)
		d.pen().M(-w/2+8, -h/2).V(h/2 - 6).done(scene.StrokeOnly)
	}},
	"collapsedGroup": {size: padded(8, 8, -2), draw: func(d *drawing, w, h float64) {
		d.rounded(-w/2+6, -h/2+8, w-12, h-8)
		d.rounded(-w/2, -h/2, w, h-4)
	}},
}

// leaning is a shape with sides that lean in by a quarter of its height, at the top or the
// bottom.
func leaning(draw func(d *drawing, w, h, s float64)) *shapeDef {
	return &shapeDef{
		inset: leanInset,
		span:  flatAcross(func(h float64) float64 { return h / 4 }),
		size: func(tw, th float64) Size {
			h := th + 2*PadY
			return Size{tw + 2*PadX + h/2, h, 0}
		},
		draw: func(d *drawing, w, h float64) { draw(d, w, h, h/4) },
	}
}

var fallback = shapes["rect"]

// ShapeSize is the size of a shape drawn about a label tw by th, as Pele's shapeSize gives
// it. A shape of a name it does not know is a rect.
func ShapeSize(shape string, tw, th float64) Size {
	if s, ok := shapes[shape]; ok {
		return s.size(tw, th)
	}
	return fallback.size(tw, th)
}

// Outline is a shape w by h centered on cx, cy, as Pele's drawShape draws it: one piece for
// each element Pele writes, in its order, but none for those it writes with neither fill
// nor stroke, which show nothing. A text shape has no pieces. A shape of a name it does not
// know is a rect.
func Outline(shape string, cx, cy, w, h float64) []scene.Part {
	s, ok := shapes[shape]
	if !ok {
		s = fallback
	}
	d := &drawing{cx: cx, cy: cy}
	s.draw(d, w, h)
	return d.pieces
}

// ShapeInset is how far the outline of a shape w by h lies in from its bounding box, on a
// side (0 top, 1 right, 2 bottom, 3 left), t along the side from its middle, as Pele's
// shapeInset gives it. A shape of a name it does not know has none.
func ShapeInset(shape string, w, h float64, side int, t float64) float64 {
	if s, ok := shapes[shape]; ok && s.inset != nil {
		return s.inset(w, h, side, t)
	}
	return 0
}

// ShapeSpan is how much of a side of a shape w by h edges may spread along, when the flow
// runs down or, if sideways, across, as Pele's shapeSpan gives it.
func ShapeSpan(shape string, w, h float64, sideways bool) float64 {
	if s, ok := shapes[shape]; ok && s.span != nil {
		return max(0, s.span(w, h, sideways))
	}
	return 0
}

// ShapeHasLabel reports whether a shape is drawn with a label, as Pele's shapeHasLabel does.
func ShapeHasLabel(shape string) bool {
	s, ok := shapes[shape]
	return !ok || !s.noLabel
}

// A Node is a box laid out, as much of Pele's CNode as the drawing needs: its center, its
// size, and whether it is a group of others.
type Node struct {
	X, Y, W, H float64
	Group      bool
}

// EndSide is the side of a node's box that a route's end at route[at:at+2] lies on: 0 top,
// 1 right, 2 bottom, 3 left, or -1 for none, as Pele's endSide gives it.
func EndSide(route []float64, at int, c Node) int {
	dx := route[at] - c.X
	dy := route[at+1] - c.Y
	if math.Abs(math.Abs(dy)-c.H/2) < 0.5 {
		if dy < 0 {
			return 0
		}
		return 2
	}
	if math.Abs(math.Abs(dx)-c.W/2) < 0.5 {
		if dx < 0 {
			return 3
		}
		return 1
	}
	return -1
}

// InsetRoute moves a route's end at route[at:at+2] from a node's box onto the outline of
// the shape drawn in it, w by h, as Pele's insetRoute does. A group's ends stay.
func InsetRoute(route []float64, at int, shape string, w, h float64, c Node) {
	if c.Group {
		return
	}
	side := EndSide(route, at, c)
	if side == 0 || side == 2 {
		amount := ShapeInset(shape, w, h, side, route[at]-c.X) + (c.H-h)/2
		if side == 0 {
			route[at+1] += amount
		} else {
			route[at+1] -= amount
		}
	} else if side >= 0 {
		amount := ShapeInset(shape, w, h, side, route[at+1]-c.Y) + (c.W-w)/2
		if side == 3 {
			route[at] += amount
		} else {
			route[at] -= amount
		}
	}
}

// A drawing gathers the pieces of a shape centered on cx, cy. The shapes are written in
// their own frame, centered on the origin, as Pele writes them.
type drawing struct {
	cx, cy float64
	pieces []scene.Part
}

func (d *drawing) add(p scene.Part, paint scene.Painting) {
	p.Paint = paint
	d.pieces = append(d.pieces, p)
}

// rounded is a box with the corners of the theme's radius.
func (d *drawing) rounded(x, y, w, h float64) {
	d.add(scene.Part{Box: scene.Rect{X: d.cx + x, Y: d.cy + y, W: w, H: h}, Rounded: true}, scene.FillAndStroke)
}

// rect is a box with corners rounded by rx, as SVG draws a rect with rx and no ry: the
// corners are rx across and as much down, each made no larger than half the side.
func (d *drawing) rect(x, y, w, h, rx float64, paint scene.Painting) {
	rx, ry := min(rx, w/2), min(rx, h/2)
	if rx <= 0 {
		d.poly(paint, x, y, x+w, y, x+w, y+h, x, y+h)
		return
	}
	d.pen().M(x+rx, y).H(x+w-rx).A(rx, ry, false, true, x+w, y+ry).V(y+h-ry).A(rx, ry, false, true, x+w-rx, y+h).
		H(x+rx).A(rx, ry, false, true, x, y+h-ry).V(y+ry).A(rx, ry, false, true, x+rx, y).Z().done(paint)
}

func (d *drawing) circle(x, y, r float64, paint scene.Painting) { d.ellipse(x, y, r, r, paint) }

func (d *drawing) ellipse(x, y, rx, ry float64, paint scene.Painting) {
	d.add(scene.Part{Path: new(scene.Path).Ellipse(d.cx+x, d.cy+y, rx, ry)}, paint)
}

func (d *drawing) poly(paint scene.Painting, xy ...float64) {
	at := make([]float64, len(xy))
	for i, v := range xy {
		if i%2 == 0 {
			at[i] = d.cx + v
		} else {
			at[i] = d.cy + v
		}
	}
	d.add(scene.Part{Path: new(scene.Path).Polygon(at...)}, paint)
}

// wave is a document: a box whose bottom runs in a wave, as Pele's wave draws it.
func (d *drawing) wave(w, h float64) {
	x, top, base, q := w/2, -h/2, h/2-10, w/4
	d.pen().M(-x, top).H(x).V(base).q(-q, -12, -2*q, 0).t(-2*q, 0).Z().done(scene.FillAndStroke)
}

// brace is a curly brace at x, h high, opening to the right when dir is -1, as Pele's
// brace draws it.
func (d *drawing) brace(x, h, dir float64) {
	y, k := h/2, 6*dir
	d.pen().M(x, -y).q(k, 0, k, 6).V(-6).q(0, 6, k, 6).q(-k, 0, -k, 6).V(y-6).q(0, 6, -k, 6).done(scene.StrokeOnly)
}

func (d *drawing) pen() *pen { return &pen{d: d, p: new(scene.Path)} }

// A pen follows SVG's path commands in the shape's frame: where it is, where the figure
// began, and the control point of the last quadratic curve, which T reflects. It takes
// each number it is given to hundredths, as Pele writes it, so that an arc whose radii
// barely reach across it is the arc Pele draws: the turn of such an arc follows the last
// decimal. Where a command is relative, or reflects a control point, the point is worked
// out from those as SVG reads them, and is not taken to hundredths again.
type pen struct {
	d            *drawing
	p            *scene.Path
	x, y, x0, y0 float64
	qx, qy       float64
	quad         bool
}

func (p *pen) at(x, y float64) *pen {
	p.x, p.y, p.quad = x, y, false
	return p
}

func (p *pen) M(x, y float64) *pen {
	x, y = r2(x), r2(y)
	p.p.MoveTo(p.d.cx+x, p.d.cy+y)
	p.x0, p.y0 = x, y
	return p.at(x, y)
}

func (p *pen) L(x, y float64) *pen { return p.lineTo(r2(x), r2(y)) }

func (p *pen) H(x float64) *pen      { return p.lineTo(r2(x), p.y) }
func (p *pen) V(y float64) *pen      { return p.lineTo(p.x, r2(y)) }
func (p *pen) l(dx, dy float64) *pen { return p.lineTo(p.x+r2(dx), p.y+r2(dy)) }
func (p *pen) h(dx float64) *pen     { return p.lineTo(p.x+r2(dx), p.y) }

// lineTo is a line to x, y, which are as SVG reads them.
func (p *pen) lineTo(x, y float64) *pen {
	p.p.LineTo(p.d.cx+x, p.d.cy+y)
	return p.at(x, y)
}

func (p *pen) A(rx, ry float64, large, sweep bool, x, y float64) *pen {
	x, y = r2(x), r2(y)
	p.p.ArcTo(r2(rx), r2(ry), 0, large, sweep, p.d.cx+x, p.d.cy+y)
	return p.at(x, y)
}

func (p *pen) Q(cx, cy, x, y float64) *pen { return p.quadTo(r2(cx), r2(cy), r2(x), r2(y)) }

func (p *pen) q(dcx, dcy, dx, dy float64) *pen {
	return p.quadTo(p.x+r2(dcx), p.y+r2(dcy), p.x+r2(dx), p.y+r2(dy))
}

// t is SVG's t: a quadratic curve by dx, dy whose control point is the last one's
// reflected about where the pen is.
func (p *pen) t(dx, dy float64) *pen {
	cx, cy := p.x, p.y
	if p.quad {
		cx, cy = 2*p.x-p.qx, 2*p.y-p.qy
	}
	return p.quadTo(cx, cy, p.x+r2(dx), p.y+r2(dy))
}

// quadTo is a quadratic curve to x, y about cx, cy, which are as SVG reads them.
func (p *pen) quadTo(cx, cy, x, y float64) *pen {
	p.p.QuadTo(p.d.cx+cx, p.d.cy+cy, p.d.cx+x, p.d.cy+y)
	p.at(x, y)
	p.qx, p.qy, p.quad = cx, cy, true
	return p
}

func (p *pen) Z() *pen {
	p.p.Close()
	return p.at(p.x0, p.y0)
}

func (p *pen) done(paint scene.Painting) { p.d.add(scene.Part{Path: p.p}, paint) }

// r2 is a number to hundredths, as Pele's num writes it.
func r2(v float64) float64 { return jsmath.Round(float64(v*100)) / 100 }
