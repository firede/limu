package jsmath

import (
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The methods of JavaScript's numbers that write them with a set number of digits, as the
// language defines them and V8 writes them: each rounds the exact value of the double, and
// a value halfway between two results takes the larger one, where Go's strconv takes the
// even one.

// ToFixed is Number.prototype.toFixed: x with digits digits after the point. NaN is "NaN",
// and a value of 1e21 or more across, or an infinity, is written as String() writes it.
func ToFixed(x float64, digits int) string {
	if math.IsNaN(x) {
		return "NaN"
	}
	if math.Abs(x) >= 1e21 {
		return chars.NumberString(x)
	}
	sign := ""
	if x < 0 {
		sign, x = "-", -x
	}
	r := new(big.Rat).SetFloat64(x)
	r.Mul(r, ratPow10(digits))
	s := roundHalfUp(r).String()
	if digits > 0 {
		if len(s) <= digits {
			s = strings.Repeat("0", digits+1-len(s)) + s
		}
		s = s[:len(s)-digits] + "." + s[len(s)-digits:]
	}
	return sign + s
}

// ToExponential is Number.prototype.toExponential with its digits given: x as one digit,
// digits more after the point, and an exponent, "1.50e+3".
func ToExponential(x float64, digits int) string {
	sign, m, e, ok := significand(x, digits+1)
	if !ok {
		return m
	}
	if digits > 0 {
		m = m[:1] + "." + m[1:]
	}
	return sign + m + exponent(e)
}

// ToPrecision is Number.prototype.toPrecision with its precision given: x to that many
// significant digits, written out in full when its exponent is from -6 to one less than the
// precision, and with an exponent otherwise.
func ToPrecision(x float64, precision int) string {
	sign, m, e, ok := significand(x, precision)
	if !ok {
		return m
	}
	switch {
	case e < -6 || e >= precision:
		if precision > 1 {
			m = m[:1] + "." + m[1:]
		}
		return sign + m + exponent(e)
	case e == precision-1:
		return sign + m
	case e >= 0:
		return sign + m[:e+1] + "." + m[e+1:]
	}
	return sign + "0." + strings.Repeat("0", -(e+1)) + m
}

// significand gives the sign of x, its first n significant digits rounded, and the exponent
// of the first, as toExponential and toPrecision find them: the n digits and exponent whose
// value is nearest x, the larger of two as near. Of NaN and the infinities it gives what
// they write as and false.
func significand(x float64, n int) (sign, digits string, e int, ok bool) {
	if math.IsNaN(x) {
		return "", "NaN", 0, false
	}
	if x < 0 {
		sign, x = "-", -x
	}
	if math.IsInf(x, 0) {
		return "", sign + "Infinity", 0, false
	}
	if x == 0 {
		return sign, strings.Repeat("0", n), 0, true
	}
	r := new(big.Rat).SetFloat64(x)
	// e is the exponent of x's first digit: 10^e <= x < 10^(e+1).
	e = int(math.Floor(math.Log10(x)))
	for r.Cmp(ratPow10(e)) < 0 {
		e--
	}
	for r.Cmp(ratPow10(e+1)) >= 0 {
		e++
	}
	q := new(big.Rat).Quo(r, ratPow10(e-n+1))
	digits = roundHalfUp(q).String()
	if len(digits) > n {
		// Rounded up to the next power of ten.
		e++
		digits = digits[:n]
	}
	return sign, digits, e, true
}

func exponent(e int) string {
	if e < 0 {
		return "e-" + strconv.Itoa(-e)
	}
	return "e+" + strconv.Itoa(e)
}

// ratPow10 is 10^n exactly.
func ratPow10(n int) *big.Rat {
	p := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(abs(n))), nil)
	if n < 0 {
		return new(big.Rat).SetFrac(big.NewInt(1), p)
	}
	return new(big.Rat).SetInt(p)
}

// roundHalfUp is the integer nearest r, which is not negative, the larger of two as near.
func roundHalfUp(r *big.Rat) *big.Int {
	num := new(big.Int).Lsh(r.Num(), 1)
	num.Add(num, r.Denom())
	den := new(big.Int).Lsh(r.Denom(), 1)
	return num.Quo(num, den)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
