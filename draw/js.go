package draw

import "math"

// The arithmetic of JavaScript where Go's differs from it.

// jsRound is Math.round: the nearest integer, halves up.
func jsRound(x float64) float64 {
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	return r
}

// jsSign is Math.sign: -1, 1, or x itself when it is zero, of either sign, or NaN.
func jsSign(x float64) float64 {
	switch {
	case x > 0:
		return 1
	case x < 0:
		return -1
	}
	return x
}

// jsOr1 is x || 1.
func jsOr1(x float64) float64 {
	if x == 0 || math.IsNaN(x) {
		return 1
	}
	return x
}

// jsSignOr1 is Math.sign(x) || 1.
func jsSignOr1(x float64) float64 { return jsOr1(jsSign(x)) }

// jsHypot is Math.hypot of two numbers as V8 works it out, which is not always math.Hypot's
// last bit: each is divided by the larger, their squares are summed with Kahan's
// compensation, and the root is scaled back.
func jsHypot(a, b float64) float64 {
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
