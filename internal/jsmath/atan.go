package jsmath

import "math"

// Math.atan2 and Math.acos as V8 works them out: fdlibm's e_atan2.c, s_atan.c and e_acos.c,
// from V8's src/base/ieee754.cc, with products fused as clang fuses them (see trig.go).

// Atan2 is Math.atan2(y, x), as fdlibm's __ieee754_atan2 does.
func Atan2(y, x float64) float64 {
	const (
		piO4 float64 = 7.8539816339744827900e-01
		piO2 float64 = 1.5707963267948965580e+00
		pi   float64 = 3.1415926535897931160e+00
	)
	xb, yb := math.Float64bits(x), math.Float64bits(y)
	hx, lx := int32(xb>>32), uint32(xb)
	hy, ly := int32(yb>>32), uint32(yb)
	ix := hx & 0x7fffffff
	iy := hy & 0x7fffffff
	if uint32(ix)|(lx|-lx)>>31 > 0x7ff00000 || uint32(iy)|(ly|-ly)>>31 > 0x7ff00000 {
		return x + y // x or y is NaN
	}
	if (hx-0x3ff00000)|int32(lx) == 0 { // x is 1
		return fdlibmAtan(y)
	}
	m := (hy>>31)&1 | (hx>>30)&2 // 2·sign(x) + sign(y)

	if uint32(iy)|ly == 0 { // y is 0
		switch m {
		case 0, 1:
			return y // atan(±0, +anything) = ±0
		case 2:
			return pi + atan2Tiny // atan(+0, -anything) = π
		default:
			return -pi - atan2Tiny // atan(-0, -anything) = -π
		}
	}
	if uint32(ix)|lx == 0 { // x is 0
		if hy < 0 {
			return -piO2 - atan2Tiny
		}
		return piO2 + atan2Tiny
	}
	if ix == 0x7ff00000 { // x is Inf
		if iy == 0x7ff00000 {
			switch m {
			case 0:
				return piO4 + atan2Tiny // atan(+Inf, +Inf)
			case 1:
				return -piO4 - atan2Tiny // atan(-Inf, +Inf)
			case 2:
				return 3.0*piO4 + atan2Tiny // atan(+Inf, -Inf)
			default:
				return -3.0*piO4 - atan2Tiny // atan(-Inf, -Inf)
			}
		}
		switch m {
		case 0:
			return 0 // atan(+…, +Inf)
		case 1:
			return math.Copysign(0, -1) // atan(-…, +Inf)
		case 2:
			return pi + atan2Tiny // atan(+…, -Inf)
		default:
			return -pi - atan2Tiny // atan(-…, -Inf)
		}
	}
	if iy == 0x7ff00000 { // y is Inf
		if hy < 0 {
			return -piO2 - atan2Tiny
		}
		return piO2 + atan2Tiny
	}

	// Compute y/x.
	var z float64
	k := (iy - ix) >> 20
	if k > 60 { // |y/x| > 2^60
		z = math.FMA(0.5, atan2PiLo, piO2)
		m &= 1
	} else if hx < 0 && k < -60 { // 0 > |y|/x > -2^-60
		z = 0
	} else {
		z = fdlibmAtan(math.Abs(y / x))
	}
	switch m {
	case 0:
		return z // atan(+, +)
	case 1:
		return -z // atan(-, +)
	case 2:
		return pi - (z - atan2PiLo) // atan(+, -)
	default:
		return (z - atan2PiLo) - pi // atan(-, -)
	}
}

// atan2Tiny and atan2PiLo are volatile in fdlibm's e_atan2.c, so that what is added with
// them is added when the function runs; they are variables here so that Go does not add
// them up as constants, exactly, at compile time.
var (
	atan2Tiny = 1.0e-300
	atan2PiLo = 1.2246467991473531772e-16
)

// fdlibmAtan is Math.atan, as fdlibm's atan does.
func fdlibmAtan(x float64) float64 {
	atanhi := [...]float64{
		4.63647609000806093515e-01, // atan(0.5) hi
		7.85398163397448278999e-01, // atan(1.0) hi
		9.82793723247329054082e-01, // atan(1.5) hi
		1.57079632679489655800e+00, // atan(inf) hi
	}
	atanlo := [...]float64{
		2.26987774529616870924e-17, // atan(0.5) lo
		3.06161699786838301793e-17, // atan(1.0) lo
		1.39033110312309984516e-17, // atan(1.5) lo
		6.12323399573676603587e-17, // atan(inf) lo
	}
	aT := [...]float64{
		3.33333333333329318027e-01,
		-1.99999999998764832476e-01,
		1.42857142725034663711e-01,
		-1.11111104054623557880e-01,
		9.09088713343650656196e-02,
		-7.69187620504482999495e-02,
		6.66107313738753120669e-02,
		-5.83357013379057348645e-02,
		4.97687799461593236017e-02,
		-3.65315727442169155270e-02,
		1.62858201153657823623e-02,
	}
	fma := math.FMA
	bits := math.Float64bits(x)
	hx := int32(bits >> 32)
	ix := hx & 0x7fffffff
	if ix >= 0x44100000 { // |x| >= 2^66
		if ix > 0x7ff00000 || (ix == 0x7ff00000 && uint32(bits) != 0) {
			return x + x // NaN
		}
		if hx > 0 {
			return atanhi[3] + atanlo[3]
		}
		return -atanhi[3] - atanlo[3]
	}
	id := -1
	if ix < 0x3fdc0000 { // |x| < 0.4375
		if ix < 0x3e400000 { // |x| < 2^-27
			return x
		}
	} else {
		x = math.Abs(x)
		if ix < 0x3ff30000 { // |x| < 1.1875
			if ix < 0x3fe60000 { // 7/16 <= |x| < 11/16
				id = 0
				x = fma(2.0, x, -1) / (2.0 + x)
			} else { // 11/16 <= |x| < 19/16
				id = 1
				x = (x - 1) / (x + 1)
			}
		} else if ix < 0x40038000 { // |x| < 2.4375
			id = 2
			x = (x - 1.5) / fma(1.5, x, 1)
		} else { // 2.4375 <= |x| < 2^66
			id = 3
			x = -1.0 / x
		}
	}
	// End of argument reduction.
	z := float64(x * x)
	w := float64(z * z)
	// Break the sum of aT[i]·z^(i+1) from i = 0 to 10 into odd and even polynomials.
	s1 := float64(z * fma(w, fma(w, fma(w, fma(w, fma(w, aT[10], aT[8]), aT[6]), aT[4]), aT[2]), aT[0]))
	s2 := float64(w * fma(w, fma(w, fma(w, fma(w, aT[9], aT[7]), aT[5]), aT[3]), aT[1]))
	if id < 0 {
		return fma(-x, s1+s2, x)
	}
	z = atanhi[id] - (fma(x, s1+s2, -atanlo[id]) - x)
	if hx < 0 {
		return -z
	}
	return z
}

// Acos is Math.acos, as fdlibm's __ieee754_acos does.
func Acos(x float64) float64 {
	const (
		pi     float64 = 3.14159265358979311600e+00
		pio2Hi float64 = 1.57079632679489655800e+00
		pio2Lo float64 = 6.12323399573676603587e-17
		pS0            = 1.66666666666666657415e-01
		pS1            = -3.25565818622400915405e-01
		pS2            = 2.01212532134862925881e-01
		pS3            = -4.00555345006794114027e-02
		pS4            = 7.91534994289814532176e-04
		pS5            = 3.47933107596021167570e-05
		qS1            = -2.40339491173441421878e+00
		qS2            = 2.02094576023350569471e+00
		qS3            = -6.88283971605453293030e-01
		qS4            = 7.70381505559019352791e-02
	)
	fma := math.FMA
	// p and q are the numerator and denominator of the rational approximation of
	// (asin(√z) - √z)/√z, as fdlibm writes them out in each branch.
	p := func(z float64) float64 {
		return float64(z * fma(z, fma(z, fma(z, fma(z, fma(z, pS5, pS4), pS3), pS2), pS1), pS0))
	}
	q := func(z float64) float64 {
		return fma(z, fma(z, fma(z, fma(z, qS4, qS3), qS2), qS1), 1)
	}
	bits := math.Float64bits(x)
	hx := int32(bits >> 32)
	ix := hx & 0x7fffffff
	if ix >= 0x3ff00000 { // |x| >= 1
		if (ix-0x3ff00000)|int32(uint32(bits)) == 0 { // |x| is 1
			if hx > 0 {
				return 0 // acos(1) = 0
			}
			return pi + 2.0*pio2Lo // acos(-1) = π
		}
		return math.NaN() // acos(|x| > 1) is NaN
	}
	if ix < 0x3fe00000 { // |x| < 0.5
		if ix <= 0x3c600000 { // |x| < 2^-57
			return pio2Hi + pio2Lo
		}
		z := float64(x * x)
		r := p(z) / q(z)
		return pio2Hi - (x - fma(-x, r, pio2Lo))
	}
	if hx < 0 { // x < -0.5
		z := (1 + x) * 0.5
		pz, qz := p(z), q(z)
		s := math.Sqrt(z)
		r := pz / qz
		w := fma(r, s, -pio2Lo)
		return fma(-2.0, s+w, pi)
	}
	// x > 0.5
	z := (1 - x) * 0.5
	s := math.Sqrt(z)
	df := math.Float64frombits(math.Float64bits(s) &^ 0xffffffff)
	c := fma(-df, df, z) / (s + df)
	r := p(z) / q(z)
	w := fma(r, s, c)
	return 2.0 * (df + w)
}
