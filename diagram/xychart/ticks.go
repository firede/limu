package xychart

import (
	"math"

	"github.com/firede/limu/internal/jsmath"
)

// tickStep is a step of 1, 2 or 5 times a power of ten that divides span into about count
// parts, as Pele's tickStep finds it. A step below one comes back as the negative of its
// inverse, which is a whole number, so that tick values are found by division and print
// without rounding noise (0.3, not 0.30000000000000004).
func tickStep(span, count float64) float64 {
	raw := span / max(count, 1)
	power := math.Floor(jsmath.Log10(raw))
	unit := raw / jsmath.Pow10(power)
	factor := 1.0
	switch {
	case unit >= 7.07:
		factor = 10
	case unit >= 3.16:
		factor = 5
	case unit >= 1.41:
		factor = 2
	}
	if power < 0 {
		return -jsmath.Pow10(-power) / factor
	}
	return factor * jsmath.Pow10(power)
}

// tickIndex is the number of steps value is from zero, as Pele's tickIndex gives it.
func tickIndex(step, value float64) float64 {
	if step > 0 {
		return value / step
	}
	return value * -step
}

// tickValue is the value index steps from zero, as Pele's tickValue gives it.
func tickValue(step, index float64) float64 {
	if step > 0 {
		return index * step
	}
	return index / -step
}

// ticksBetween is the multiples of a step from lo to hi, both included, at most 100 of
// them, as Pele's ticksBetween finds them.
func ticksBetween(lo, hi, step float64) []float64 {
	first := jsmath.Round(tickIndex(step, lo))
	if tickValue(step, first) < lo {
		first++
	}
	last := jsmath.Round(tickIndex(step, hi))
	if tickValue(step, last) > hi {
		last--
	}
	var out []float64
	for i := first; i <= last && len(out) < 100; i++ {
		out = append(out, tickValue(step, i))
	}
	return out
}
