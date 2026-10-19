package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// The mark is the plume of limu kohu, drawn as docs/website.md says: on a box of 64, a stem
// and seven branchlets alternating up it, each a stroke that tapers toward its tip and ends
// round. It is in currentColor, so that a page sets its color.

type point struct{ x, y float64 }

// num writes a coordinate to two places, without the zeros after them.
func num(v float64) string {
	s := strconv.FormatFloat(v, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	return strings.TrimSuffix(s, ".")
}

func cubic(p [4]point, t float64) point {
	u := 1 - t
	a, b, c, d := u*u*u, 3*u*u*t, 3*u*t*t, t*t*t
	return point{a*p[0].x + b*p[1].x + c*p[2].x + d*p[3].x, a*p[0].y + b*p[1].y + c*p[2].y + d*p[3].y}
}

func tangent(p [4]point, t float64) point {
	u := 1 - t
	a, b, c := 3*u*u, 6*u*t, 3*t*t
	return point{a*(p[1].x-p[0].x) + b*(p[2].x-p[1].x) + c*(p[3].x-p[2].x), a*(p[1].y-p[0].y) + b*(p[2].y-p[1].y) + c*(p[3].y-p[2].y)}
}

// A cap is a round end of a stroke, drawn as a circle over the stroke's square end.
type cap struct {
	at point
	r  float64
}

// taper is the outline of a stroke along a cubic, w0 wide at its start and w1 at its end,
// and the round caps of its ends.
func taper(p [4]point, w0, w1 float64) (string, [2]cap) {
	const n = 24
	var left, right [n + 1]point
	for i := range n + 1 {
		t := float64(i) / n
		at, d := cubic(p, t), tangent(p, t)
		m := math.Hypot(d.x, d.y)
		if m == 0 {
			m = 1
		}
		nx, ny := -d.y/m, d.x/m
		w := (w0 + (w1-w0)*t) / 2
		left[i] = point{at.x + nx*w, at.y + ny*w}
		right[i] = point{at.x - nx*w, at.y - ny*w}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "M%s %s", num(left[0].x), num(left[0].y))
	for _, q := range left[1:] {
		fmt.Fprintf(&b, "L%s %s", num(q.x), num(q.y))
	}
	for i := n; i >= 0; i-- {
		fmt.Fprintf(&b, "L%s %s", num(right[i].x), num(right[i].y))
	}
	b.WriteString("Z")
	return b.String(), [2]cap{{cubic(p, 0), w0 / 2}, {cubic(p, 1), w1 / 2}}
}

var stem = [4]point{{31, 60}, {28.5, 45}, {36.5, 29}, {33, 5}}

// A branchlet leaves the stem at t, to the side (-1 left, 1 right), and is length long.
type branchlet struct{ t, side, length float64 }

var branchlets = []branchlet{{0.18, -1, 17.5}, {0.31, 1, 16.5}, {0.44, -1, 14.5}, {0.57, 1, 12.5}, {0.69, -1, 10}, {0.80, 1, 7.5}, {0.89, -1, 5}}

// branch is a branchlet's curve from where it leaves a stem: out from the stem and rising,
// then turning up toward its tip.
func branch(at point, side, l float64) [4]point {
	return [4]point{at, {at.x + side*l*0.36, at.y - l*0.14}, {at.x + side*l*0.64, at.y - l*0.34}, {at.x + side*l*0.78, at.y - l*0.68}}
}

// plume is the mark's strokes, where the box of 64 has them, before they are centered.
func plume() string {
	var strokes, caps strings.Builder
	add := func(d string, cs [2]cap) {
		fmt.Fprintf(&strokes, `<path d="%s"/>`, d)
		for _, c := range cs {
			fmt.Fprintf(&caps, `<circle cx="%s" cy="%s" r="%s"/>`, num(c.at.x), num(c.at.y), num(c.r))
		}
	}
	add(taper(stem, 4.6, 2.6))
	for _, b := range branchlets {
		add(taper(branch(cubic(stem, b.t), b.side, b.length), 3.4-1.2*b.t, 2))
	}
	return `<g fill="currentColor">` + strokes.String() + caps.String() + `</g>`
}

// centering is how far the plume moves to be centered on its box by its bounds, then 0.6
// to the right.
func centering() point {
	minX, minY, maxX, maxY := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	add := func(q point) {
		minX, minY, maxX, maxY = min(minX, q.x), min(minY, q.y), max(maxX, q.x), max(maxY, q.y)
	}
	for i := range 51 {
		add(cubic(stem, float64(i)/50))
	}
	for _, b := range branchlets {
		c := branch(cubic(stem, b.t), b.side, b.length)
		for i := range 21 {
			add(cubic(c, float64(i)/20))
		}
	}
	return point{32 - (minX+maxX)/2 + 0.6, 32 - (minY+maxY)/2}
}

// markSVG is the mark on its box of 64.
func markSVG() string {
	c := centering()
	return fmt.Sprintf(`<g transform="translate(%s %s)">%s</g>`, num(c.x), num(c.y), plume())
}

// line is a stroke of even width along a cubic, with round ends.
func line(p [4]point, w float64) string {
	return fmt.Sprintf(`<path d="M%s %sC%s %s %s %s %s %s" fill="none" stroke="currentColor" stroke-width="%s" stroke-linecap="round"/>`,
		num(p[0].x), num(p[0].y), num(p[1].x), num(p[1].y), num(p[2].x), num(p[2].y), num(p[3].x), num(p[3].y), num(w))
}

// mark16SVG is the mark's cut for 16 pixels, on a box of 16: a stem and three branchlets.
func mark16SVG() string {
	s := [4]point{{8, 15}, {7.4, 11}, {9.1, 7}, {8.4, 1.2}}
	var out strings.Builder
	out.WriteString(line(s, 1.9))
	for _, b := range []branchlet{{0.3, -1, 6.6}, {0.52, 1, 5.6}, {0.74, -1, 4}} {
		out.WriteString(line(branch(cubic(s, b.t), b.side, b.length), 1.7))
	}
	return out.String()
}

// squircle is a superellipse of exponent 5 filling a box of size, as MyGo and Apple round
// an icon's corners: there is no corner where an arc meets a straight edge.
func squircle(size float64, k int) string {
	c := size / 2
	var b strings.Builder
	for i := range 4 * k {
		a := 2 * math.Pi * float64(i) / float64(4*k)
		ca, sa := math.Cos(a), math.Sin(a)
		x := c + c*math.Copysign(math.Pow(math.Abs(ca), 2.0/5), ca)
		y := c + c*math.Copysign(math.Pow(math.Abs(sa), 2.0/5), sa)
		if i == 0 {
			b.WriteString("M")
		} else {
			b.WriteString("L")
		}
		fmt.Fprintf(&b, "%s %s", num(x), num(y))
	}
	b.WriteString("Z")
	return b.String()
}

// The icon is the plume in the dark theme's accent, in the dark page's water, with a rim a
// shade lighter so that it keeps its edge on dark grounds.
const (
	iconWater  = "#0c1618"
	iconRim    = "#2a4247"
	iconAccent = "#ee8494"
)

// iconSVG is the icon on its box of 64: the plume scaled to 0.74 about the center and set
// 0.5 lower.
func iconSVG() string {
	const k = 0.74
	c := centering()
	sq := squircle(64, 64)
	return fmt.Sprintf(`<path d="%s" fill="%s"/><path d="%s" fill="none" stroke="%s" stroke-width="2" transform="translate(1 1) scale(0.96875)"/>`, sq, iconWater, sq, iconRim) +
		fmt.Sprintf(`<g transform="translate(%s %s) scale(%s)" color="%s">%s</g>`, num(32-32*k+c.x*k), num(32-32*k+c.y*k+0.5), num(k), iconAccent, plume())
}

// iconFullSVG is the icon's plume on the water to the box's edges, with no rim, for
// Apple's devices, which round an icon themselves.
func iconFullSVG() string {
	const k = 0.74
	c := centering()
	return fmt.Sprintf(`<rect width="64" height="64" fill="%s"/>`, iconWater) +
		fmt.Sprintf(`<g transform="translate(%s %s) scale(%s)" color="%s">%s</g>`, num(32-32*k+c.x*k), num(32-32*k+c.y*k+0.5), num(k), iconAccent, plume())
}

// icon16SVG is the icon's cut for 16 pixels: the mark's cut, scaled to 0.8 about the center.
func icon16SVG() string {
	sq := squircle(16, 24)
	return fmt.Sprintf(`<path d="%s" fill="%s"/><path d="%s" fill="none" stroke="%s" stroke-width="1" transform="translate(0.5 0.5) scale(0.9375)"/>`, sq, iconWater, sq, iconRim) +
		fmt.Sprintf(`<g transform="translate(1.6 1.6) scale(0.8)" color="%s">%s</g>`, iconAccent, mark16SVG())
}

// svgFile is a drawing on a box of size as an SVG file of its own.
func svgFile(size int, body string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d">%s</svg>`+"\n", size, size, size, size, body)
}
