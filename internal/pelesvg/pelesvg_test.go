package pelesvg

import (
	"math"
	"testing"
)

// What Pele's num writes of these numbers, as Node 24 runs it.
func TestNum(t *testing.T) {
	for _, c := range []struct {
		v    float64
		want string
	}{
		{0, "0"},
		{math.Copysign(0, -1), "0"},
		{-0.004, "0"},
		{-0.005, "0"},
		{0.005, "0.01"},
		{22.505, "22.51"},
		{-22.505, "-22.5"},
		{1.5, "1.5"},
		{-1.25, "-1.25"},
		{1e9 - 0.01, "999999999.99"},
		{1e9, "1000000000"},
		{-1e9, "-1000000000"},
		{123456789.125, "123456789.13"},
		{1e21, "999999999999999900000"},
		{-1e21, "-999999999999999900000"},
		{1.5e22, "1.5e+22"},
		{1e300, "1e+300"},
		{math.Inf(1), "Infinity"},
		{math.Inf(-1), "-Infinity"},
		{math.NaN(), "NaN"},
	} {
		if got := Num(c.v); got != c.want {
			t.Errorf("Num(%v) = %s, Pele %s", c.v, got, c.want)
		}
	}
}
