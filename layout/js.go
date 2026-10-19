package layout

import "math"

// The ports keep JavaScript's meaning where it reaches what is placed. These are the pieces
// of it that Go does not have as JavaScript has them.

// Go may fuse a product and a sum into one operation, rounded once, where JavaScript rounds
// each. A product that is added to or taken from something is written float64(x*y), which
// rounds it on its own and keeps Go from fusing it.

// sign is how a JavaScript sort reads what its comparator returns: below zero, above zero,
// or else, for zero and for NaN alike, a tie.
func sign(x float64) int {
	if x < 0 {
		return -1
	}
	if x > 0 {
		return 1
	}
	return 0
}

// or is JavaScript's x || y on numbers: y when x is zero or NaN.
func or(x, y float64) float64 {
	if x != 0 && x == x {
		return x
	}
	return y
}

// toInt32 is what storing a number in an Int32Array keeps of it: its whole part, wrapped to
// 32 bits, and 0 for NaN and the infinities.
func toInt32(x float64) int {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return 0
	}
	t := math.Mod(math.Trunc(x), 1<<32)
	return int(int32(uint32(int64(t))))
}

// hypot is V8's Math.hypot for two numbers, which sums the squares of the two scaled by the
// larger, with Kahan's compensation, and is not always Go's math.Hypot to the last bit.
func hypot(a, b float64) float64 {
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return math.Inf(1)
	}
	if math.IsNaN(a) || math.IsNaN(b) {
		return math.NaN()
	}
	a = math.Abs(a)
	b = math.Abs(b)
	most := max(a, b)
	if most == 0 {
		return 0
	}
	sum := 0.0
	compensation := 0.0
	for _, v := range [2]float64{a, b} {
		n := v / most
		summand := float64(n*n) - compensation
		preliminary := sum + summand
		compensation = (preliminary - sum) - summand
		sum = preliminary
	}
	return math.Sqrt(sum) * most
}
