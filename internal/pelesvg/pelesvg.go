// Package pelesvg writes numbers and paths as Pele's SVG writes them, for tests that compare
// with what it draws.
package pelesvg

import (
	"math"
	"strconv"
	"strings"

	"github.com/firede/limu/scene"
)

// Num is Pele's num: a number written to two decimals, as its SVG writes coordinates.
func Num(v float64) string {
	n := round(float64(v * 100))
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign, n = "-", -n
	}
	if n >= 1e11 {
		return strconv.FormatFloat(round(float64(v*100))/100, 'f', -1, 64)
	}
	q := math.Floor(n / 100)
	s := sign + strconv.FormatFloat(q, 'f', -1, 64)
	switch i := int(n - float64(q*100)); {
	case i == 0:
	case i%10 == 0:
		s += "." + strconv.Itoa(i/10)
	case i < 10:
		s += ".0" + strconv.Itoa(i)
	default:
		s += "." + strconv.Itoa(i)
	}
	return s
}

// D writes a path as Pele writes the d of one.
func D(p *scene.Path) string {
	var b strings.Builder
	p.Each(func(op scene.Op, a []float64) {
		b.WriteByte("MLQCZ"[op])
		for i := 0; i < len(a); i += 2 {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(Num(a[i]) + "," + Num(a[i+1]))
		}
	})
	return b.String()
}

// round is JavaScript's Math.round.
func round(x float64) float64 {
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	return r
}
