package jsmath

import (
	"math"
	"strconv"
)

// Log10 is Math.log10 as V8 works it out: fdlibm's e_log10.c and e_log.c, which give exact
// powers of ten their whole logarithms, where Go's math.Log10 does not always. V8 is
// compiled with its products fused into the sums they are in, which fma writes out here.
func Log10(x float64) float64 {
	const (
		two54     = 1.80143985094819840000e+16
		ivln10    = 4.34294481903251816668e-01
		log10_2hi = 3.01029995663611771306e-01
		log10_2lo = 3.69423907715893078616e-13
	)
	bits := math.Float64bits(x)
	hx := int32(bits >> 32)
	lx := uint32(bits)
	k := int32(0)
	if hx < 0x00100000 {
		if (hx&0x7fffffff)|int32(lx) == 0 {
			return math.Inf(-1)
		}
		if hx < 0 {
			return math.NaN()
		}
		k -= 54
		x *= two54
		bits = math.Float64bits(x)
		hx = int32(bits >> 32)
		lx = uint32(bits)
	}
	if hx >= 0x7ff00000 {
		return x + x
	}
	if hx == 0x3ff00000 && lx == 0 {
		return 0
	}
	k += hx>>20 - 1023
	i := int32(uint32(k) >> 31)
	hx = hx&0x000fffff | (0x3ff-i)<<20
	y := float64(k + i)
	x = math.Float64frombits(uint64(uint32(hx))<<32 | uint64(lx))
	z := math.FMA(y, log10_2lo, float64(ivln10*fdlibmLog(x)))
	return math.FMA(y, log10_2hi, z)
}

// fdlibmLog is fdlibm's e_log.c, as V8 has it.
func fdlibmLog(x float64) float64 {
	const (
		ln2Hi = 6.93147180369123816490e-01
		ln2Lo = 1.90821492927058770002e-10
		two54 = 1.80143985094819840000e+16
		lg1   = 6.666666666666735130e-01
		lg2   = 3.999999999940941908e-01
		lg3   = 2.857142874366239149e-01
		lg4   = 2.222219843214978396e-01
		lg5   = 1.818357216161805012e-01
		lg6   = 1.531383769920937332e-01
		lg7   = 1.479819860511658591e-01
	)
	fma := math.FMA
	bits := math.Float64bits(x)
	hx := int32(bits >> 32)
	lx := uint32(bits)
	k := int32(0)
	if hx < 0x00100000 {
		if (hx&0x7fffffff)|int32(lx) == 0 {
			return math.Inf(-1)
		}
		if hx < 0 {
			return math.NaN()
		}
		k -= 54
		x *= two54
		hx = int32(math.Float64bits(x) >> 32)
	}
	if hx >= 0x7ff00000 {
		return x + x
	}
	k += hx>>20 - 1023
	hx &= 0x000fffff
	i := (hx + 0x95f64) & 0x100000
	x = math.Float64frombits(uint64(uint32(hx|(i^0x3ff00000)))<<32 | math.Float64bits(x)&0xffffffff)
	k += i >> 20
	f := x - 1.0
	if (0x000fffff & (2 + hx)) < 3 {
		if f == 0 {
			if k == 0 {
				return 0
			}
			dk := float64(k)
			return fma(dk, ln2Hi, float64(dk*ln2Lo))
		}
		r := float64(float64(f*f) * fma(-0.33333333333333333, f, 0.5))
		if k == 0 {
			return f - r
		}
		dk := float64(k)
		return fma(dk, ln2Hi, -(fma(-dk, ln2Lo, r) - f))
	}
	s := f / (2.0 + f)
	dk := float64(k)
	z := float64(s * s)
	i = hx - 0x6147a
	w := float64(z * z)
	j := 0x6b851 - hx
	t1 := float64(w * fma(w, fma(w, lg6, lg4), lg2))
	t2 := float64(z * fma(w, fma(w, fma(w, lg7, lg5), lg3), lg1))
	i |= j
	r := t2 + t1
	if i > 0 {
		hfsq := float64(float64(0.5*f) * f)
		if k == 0 {
			return f - fma(-s, hfsq+r, hfsq)
		}
		return fma(dk, ln2Hi, -((hfsq - fma(s, hfsq+r, float64(dk*ln2Lo))) - f))
	}
	if k == 0 {
		return fma(-s, f-r, f)
	}
	return fma(dk, ln2Hi, -(fma(s, f-r, -float64(dk*ln2Lo)) - f))
}

// Pow10 is Math.pow(10, n) for a whole n, which V8 rounds correctly, as a literal 1eN is.
func Pow10(n float64) float64 {
	if n != math.Trunc(n) || math.IsInf(n, 0) {
		return math.Pow(10, n)
	}
	if n > 400 {
		return math.Inf(1)
	}
	if n < -400 {
		return 0
	}
	v, _ := strconv.ParseFloat("1e"+strconv.Itoa(int(n)), 64)
	return v
}
