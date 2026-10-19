// Package jsmath is the arithmetic of JavaScript where Go's differs from it, for code ported
// from Pele that must come out as Pele's does to the last bit.
package jsmath

import "math"

// Round is Math.round: the nearest integer, halves up, and zero with the sign of x where
// that is the nearest, as Math.round(-0.25) is -0. It is not math.Floor(x+0.5), which
// rounds 0.49999999999999994 up, as the sum is 1. x is converted where it is subtracted
// from, as Go, once Round is inlined, would fuse a product passed as x into that
// subtraction: Round(22.505*100) would be 2250.
func Round(x float64) float64 {
	r := math.Floor(x)
	if float64(x)-r >= 0.5 {
		r++
	}
	if r == 0 {
		return math.Copysign(0, x)
	}
	return r
}

// Sign is Math.sign: -1, 1, or x itself when it is zero, of either sign, or NaN.
func Sign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return x
}

// Or1 is x || 1.
func Or1(x float64) float64 {
	if x == 0 || math.IsNaN(x) {
		return 1
	}
	return x
}

// SignOr1 is Math.sign(x) || 1.
func SignOr1(x float64) float64 { return Or1(Sign(x)) }

// Hypot is Math.hypot of two numbers as V8 works it out, which is not always math.Hypot's
// last bit: each is divided by the larger, their squares are summed with Kahan's
// compensation, and the root is scaled back.
func Hypot(a, b float64) float64 {
	a, b = math.Abs(a), math.Abs(b)
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return math.Inf(1)
	}
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.NaN()
	}
	m := max(a, b)
	if m == 0 {
		return 0
	}
	var sum, comp float64
	for _, v := range [2]float64{a, b} {
		n := v / m
		summand := float64(n*n) - comp
		prelim := sum + summand
		comp = (prelim - sum) - summand
		sum = prelim
	}
	return math.Sqrt(sum) * m
}
