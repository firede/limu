package jsmath

import "math"

// Math.sin and Math.cos as V8 works them out: fdlibm's s_sin.c and s_cos.c, from V8's
// src/base/ieee754.cc, which Node builds without V8_USE_LIBM_TRIG_FUNCTIONS and so without
// glibc's. Node on arm64 gives what they give compiled by clang with -ffp-contract=on, its
// default, which fuses a product into the sum or difference it is written in, in the same
// expression, into one multiply-add: the product on the left of the operator if there is
// one, else the one on its right. math.FMA writes out each of those here; every other
// product is rounded on its own, as float64(x*y). Built without fused multiply-adds, as
// for x86-64, V8 gives other last bits for about one input in two hundred.

// Sin is Math.sin, as fdlibm's sin does.
func Sin(x float64) float64 {
	ix := int32(math.Float64bits(x)>>32) & 0x7fffffff
	if ix <= 0x3fe921fb {
		return kernelSin(x, 0, 0)
	}
	if ix >= 0x7ff00000 {
		return x - x // NaN, for Inf or NaN
	}
	n, y0, y1 := remPio2(x)
	switch n & 3 {
	case 0:
		return kernelSin(y0, y1, 1)
	case 1:
		return kernelCos(y0, y1)
	case 2:
		return -kernelSin(y0, y1, 1)
	default:
		return -kernelCos(y0, y1)
	}
}

// Cos is Math.cos, as fdlibm's cos does.
func Cos(x float64) float64 {
	ix := int32(math.Float64bits(x)>>32) & 0x7fffffff
	if ix <= 0x3fe921fb {
		return kernelCos(x, 0)
	}
	if ix >= 0x7ff00000 {
		return x - x // NaN, for Inf or NaN
	}
	n, y0, y1 := remPio2(x)
	switch n & 3 {
	case 0:
		return kernelCos(y0, y1)
	case 1:
		return -kernelSin(y0, y1, 1)
	case 2:
		return -kernelCos(y0, y1)
	default:
		return kernelSin(y0, y1, 1)
	}
}

// kernelSin is sin(x+y) for |x| up to about π/4, y the tail of x, which is taken to be 0
// when iy is 0, as fdlibm's __kernel_sin does.
func kernelSin(x, y float64, iy int) float64 {
	const (
		half = 5.00000000000000000000e-01
		S1   = -1.66666666666666324348e-01
		S2   = 8.33333333332248946124e-03
		S3   = -1.98412698298579493134e-04
		S4   = 2.75573137070700676789e-06
		S5   = -2.50507602534068634195e-08
		S6   = 1.58969099521155010221e-10
	)
	fma := math.FMA
	ix := int32(math.Float64bits(x)>>32) & 0x7fffffff
	if ix < 0x3e400000 { // |x| < 2**-27
		return x
	}
	z := float64(x * x)
	v := float64(z * x)
	r := fma(z, fma(z, fma(z, fma(z, S6, S5), S4), S3), S2)
	if iy == 0 {
		return fma(v, fma(z, r, S1), x)
	}
	return x - fma(-v, S1, fma(z, fma(half, y, -float64(v*r)), -y))
}

// kernelCos is cos(x+y) for |x| up to about π/4, y the tail of x, as fdlibm's __kernel_cos
// does.
func kernelCos(x, y float64) float64 {
	const (
		C1 = 4.16666666666666019037e-02
		C2 = -1.38888888888741095749e-03
		C3 = 2.48015872894767294178e-05
		C4 = -2.75573143513906633035e-07
		C5 = 2.08757232129817482790e-09
		C6 = -1.13596475577881948265e-11
	)
	fma := math.FMA
	ix := int32(math.Float64bits(x)>>32) & 0x7fffffff
	if ix < 0x3e400000 { // |x| < 2**-27
		return 1
	}
	z := float64(x * x)
	r := float64(z * fma(z, fma(z, fma(z, fma(z, fma(z, C6, C5), C4), C3), C2), C1))
	if ix < 0x3fd33333 { // |x| < 0.3
		return 1 - fma(0.5, z, -fma(z, r, -float64(x*y)))
	}
	var qx float64
	if ix > 0x3fe90000 { // |x| > 0.78125
		qx = 0.28125
	} else {
		qx = math.Float64frombits(uint64(uint32(ix-0x00200000)) << 32) // |x|/4
	}
	iz := fma(0.5, z, -qx)
	a := 1 - qx
	return a - (iz - fma(z, r, -float64(x*y)))
}

// twoOverPi is 2/π in 24-bit pieces, 396 hexadecimal digits of it.
var twoOverPi = [...]int32{
	0xA2F983, 0x6E4E44, 0x1529FC, 0x2757D1, 0xF534DD, 0xC0DB62, 0x95993C,
	0x439041, 0xFE5163, 0xABDEBB, 0xC561B7, 0x246E3A, 0x424DD2, 0xE00649,
	0x2EEA09, 0xD1921C, 0xFE1DEB, 0x1CB129, 0xA73EE8, 0x8235F5, 0x2EBB44,
	0x84E99C, 0x7026B4, 0x5F7E41, 0x3991D6, 0x398353, 0x39F49C, 0x845F8B,
	0xBDF928, 0x3B1FF8, 0x97FFDE, 0x05980F, 0xEF2F11, 0x8B5A0A, 0x6D1F6D,
	0x367ECF, 0x27CB09, 0xB74F46, 0x3F669E, 0x5FEA2D, 0x7527BA, 0xC7EBE5,
	0xF17B3D, 0x0739F7, 0x8A5292, 0xEA6BFB, 0x5FB11F, 0x8D5D08, 0x560330,
	0x46FC7B, 0x6BABF0, 0xCFBC20, 0x9AF436, 0x1DA9E3, 0x91615E, 0xE61B08,
	0x659985, 0x5F14A0, 0x68408D, 0xFFD880, 0x4D7327, 0x310606, 0x1556CA,
	0x73A8C9, 0x60E27B, 0xC08C6B,
}

// npio2HW holds the high words of n·π/2 for n from 1 to 32.
var npio2HW = [...]int32{
	0x3FF921FB, 0x400921FB, 0x4012D97C, 0x401921FB, 0x401F6A7A, 0x4022D97C,
	0x4025FDBB, 0x402921FB, 0x402C463A, 0x402F6A7A, 0x4031475C, 0x4032D97C,
	0x40346B9C, 0x4035FDBB, 0x40378FDB, 0x403921FB, 0x403AB41B, 0x403C463A,
	0x403DD85A, 0x403F6A7A, 0x40407E4C, 0x4041475C, 0x4042106C, 0x4042D97C,
	0x4043A28C, 0x40446B9C, 0x404534AC, 0x4045FDBB, 0x4046C6CB, 0x40478FDB,
	0x404858EB, 0x404921FB,
}

// remPio2 is x less the nearest multiple n of π/2, as y0+y1, and n, as fdlibm's
// __ieee754_rem_pio2 does: with π/2 in three pieces of 33 bits for |x| up to 2^19·π/2,
// and with __kernel_rem_pio2 beyond.
func remPio2(x float64) (n int32, y0, y1 float64) {
	const (
		half    = 5.00000000000000000000e-01
		two24   = 1.67772160000000000000e+07
		invpio2 = 6.36619772367581382433e-01
		pio2_1  = 1.57079632673412561417e+00
		pio2_1t = 6.07710050650619224932e-11
		pio2_2  = 6.07710050630396597660e-11
		pio2_2t = 2.02226624879595063154e-21
		pio2_3  = 2.02226624871116645580e-21
		pio2_3t = 8.47842766036889956997e-32
	)
	fma := math.FMA
	bits := math.Float64bits(x)
	hx := int32(bits >> 32)
	ix := hx & 0x7fffffff
	if ix <= 0x3fe921fb { // |x| ~<= π/4, no need for reduction
		return 0, x, 0
	}
	if ix < 0x4002d97c { // |x| < 3π/4, special case with n = ±1
		if hx > 0 {
			z := x - pio2_1
			if ix != 0x3ff921fb { // 33+53 bits of π are enough
				y0 = z - pio2_1t
				y1 = (z - y0) - pio2_1t
			} else { // near π/2, use 33+33+53 bits of π
				z -= pio2_2
				y0 = z - pio2_2t
				y1 = (z - y0) - pio2_2t
			}
			return 1, y0, y1
		}
		z := x + pio2_1
		if ix != 0x3ff921fb {
			y0 = z + pio2_1t
			y1 = (z - y0) + pio2_1t
		} else {
			z += pio2_2
			y0 = z + pio2_2t
			y1 = (z - y0) + pio2_2t
		}
		return -1, y0, y1
	}
	if ix <= 0x413921fb { // |x| ~<= 2^19·(π/2), medium size
		t := math.Abs(x)
		n = int32(fma(t, invpio2, half))
		fn := float64(n)
		r := fma(-fn, pio2_1, t)
		w := float64(fn * pio2_1t) // first round, good to 85 bits
		if n < 32 && ix != npio2HW[n-1] {
			y0 = r - w // quick check, no cancellation
		} else {
			j := ix >> 20
			y0 = r - w
			high := uint32(math.Float64bits(y0) >> 32)
			i := j - int32((high>>20)&0x7ff)
			if i > 16 { // second iteration, good to 118 bits
				t = r
				w = float64(fn * pio2_2)
				r = t - w
				w = fma(fn, pio2_2t, -((t - r) - w))
				y0 = r - w
				high = uint32(math.Float64bits(y0) >> 32)
				i = j - int32((high>>20)&0x7ff)
				if i > 49 { // third iteration, 151 bits, for every case
					t = r
					w = float64(fn * pio2_3)
					r = t - w
					w = fma(fn, pio2_3t, -((t - r) - w))
					y0 = r - w
				}
			}
		}
		y1 = (r - y0) - w
		if hx < 0 {
			return -n, -y0, -y1
		}
		return n, y0, y1
	}
	// All other, large, arguments.
	if ix >= 0x7ff00000 { // Inf or NaN
		y0 = x - x
		return 0, y0, y0
	}
	// z = scalbn(|x|, ilogb(x)-23)
	e0 := ix>>20 - 1046
	z := math.Float64frombits(uint64(uint32(ix-int32(uint32(e0)<<20)))<<32 | bits&0xffffffff)
	var tx [3]float64
	for i := range 2 {
		tx[i] = float64(int32(z))
		z = float64((z - tx[i]) * two24)
	}
	tx[2] = z
	nx := 3
	for tx[nx-1] == 0 { // skip zero terms
		nx--
	}
	n, y0, y1 = kernelRemPio2(tx[:nx], e0)
	if hx < 0 {
		return -n, -y0, -y1
	}
	return n, y0, y1
}

// kernelRemPio2 is x·2^(e0-24·i) summed over the 24-bit pieces x[i], less the nearest
// multiple of π/2, as y0+y1, and that multiple modulo 8, as fdlibm's __kernel_rem_pio2
// does with prec 2, the precision __ieee754_rem_pio2 asks of it.
func kernelRemPio2(x []float64, e0 int32) (int32, float64, float64) {
	const (
		two24  = 1.67772160000000000000e+07
		twon24 = 5.96046447753906250000e-08
		jk     = 4 // init_jk[prec] for prec 2
		jp     = jk
	)
	PIo2 := [...]float64{
		1.57079625129699707031e+00,
		7.54978941586159635335e-08,
		5.39030252995776476554e-15,
		3.28200341580791294123e-22,
		1.27065575308067607349e-29,
		1.22933308981111328932e-36,
		2.73370053816464559624e-44,
		2.16741683877804819444e-51,
	}
	fma := math.FMA
	var (
		iq        [20]int32
		f, fq, q  [20]float64
		z, fw     float64
		n, ih     int32
		carry, jz int32
	)

	// Determine jx, jv and q0; note that 3 > q0.
	jx := int32(len(x)) - 1
	jv := (e0 - 3) / 24
	if jv < 0 {
		jv = 0
	}
	q0 := e0 - 24*(jv+1)

	// Set up f[0] to f[jx+jk] where f[jx+jk] = ipio2[jv+jk].
	j := jv - jx
	m := jx + jk
	for i := int32(0); i <= m; i, j = i+1, j+1 {
		if j < 0 {
			f[i] = 0
		} else {
			f[i] = float64(twoOverPi[j])
		}
	}

	// Compute q[0], q[1], … q[jk].
	for i := int32(0); i <= jk; i++ {
		fw = 0
		for j := int32(0); j <= jx; j++ {
			fw = fma(x[j], f[jx+i-j], fw)
		}
		q[i] = fw
	}

	jz = jk
	for {
		// Distill q[] into iq[], in reverse.
		z = q[jz]
		for i, j := int32(0), jz; j > 0; i, j = i+1, j-1 {
			fw = float64(int32(float64(twon24 * z)))
			iq[i] = int32(fma(-two24, fw, z))
			z = q[j-1] + fw
		}

		// Compute n.
		z = math.Ldexp(z, int(q0))                     // the actual value of z
		z = fma(-8.0, math.Floor(float64(z*0.125)), z) // trim off integers >= 8
		n = int32(z)
		z -= float64(n)
		ih = 0
		if q0 > 0 { // iq[jz-1] is needed to determine n
			i := iq[jz-1] >> (24 - q0)
			n += i
			iq[jz-1] -= i << (24 - q0)
			ih = iq[jz-1] >> (23 - q0)
		} else if q0 == 0 {
			ih = iq[jz-1] >> 23
		} else if z >= 0.5 {
			ih = 2
		}

		if ih > 0 { // q > 0.5
			n++
			carry = 0
			for i := int32(0); i < jz; i++ { // compute 1-q
				j := iq[i]
				if carry == 0 {
					if j != 0 {
						carry = 1
						iq[i] = 0x1000000 - j
					}
				} else {
					iq[i] = 0xffffff - j
				}
			}
			if q0 > 0 { // rare case: chance is 1 in 12
				switch q0 {
				case 1:
					iq[jz-1] &= 0x7fffff
				case 2:
					iq[jz-1] &= 0x3fffff
				}
			}
			if ih == 2 {
				z = 1 - z
				if carry != 0 {
					z -= math.Ldexp(1, int(q0))
				}
			}
		}

		// Check if recomputation is needed.
		if z == 0 {
			j := int32(0)
			for i := jz - 1; i >= jk; i-- {
				j |= iq[i]
			}
			if j == 0 { // need recomputation
				k := int32(1)
				for jk >= k && iq[jk-k] == 0 { // k = number of terms needed
					k++
				}
				for i := jz + 1; i <= jz+k; i++ { // add q[jz+1] to q[jz+k]
					f[jx+i] = float64(twoOverPi[jv+i])
					fw = 0
					for j := int32(0); j <= jx; j++ {
						fw = fma(x[j], f[jx+i-j], fw)
					}
					q[i] = fw
				}
				jz += k
				continue
			}
		}
		break
	}

	// Chop off zero terms.
	if z == 0 {
		jz--
		q0 -= 24
		for iq[jz] == 0 {
			jz--
			q0 -= 24
		}
	} else { // break z into 24 bits if necessary
		z = math.Ldexp(z, -int(q0))
		if z >= two24 {
			fw = float64(int32(float64(twon24 * z)))
			iq[jz] = int32(fma(-two24, fw, z))
			jz++
			q0 += 24
			iq[jz] = int32(fw)
		} else {
			iq[jz] = int32(z)
		}
	}

	// Convert the integer "bit" chunks to floating-point values.
	fw = math.Ldexp(1, int(q0))
	for i := jz; i >= 0; i-- {
		q[i] = float64(fw * float64(iq[i]))
		fw *= twon24
	}

	// Compute PIo2[0, …, jp]·q[jz, …, 0].
	for i := jz; i >= 0; i-- {
		fw = 0
		for k := int32(0); k <= jp && k <= jz-i; k++ {
			fw = fma(PIo2[k], q[i+k], fw)
		}
		fq[jz-i] = fw
	}

	// Compress fq[] into y0 and y1.
	fw = 0
	for i := jz; i >= 0; i-- {
		fw += fq[i]
	}
	y0 := fw
	if ih != 0 {
		y0 = -fw
	}
	fw = fq[0] - fw
	for i := int32(1); i <= jz; i++ {
		fw += fq[i]
	}
	y1 := fw
	if ih != 0 {
		y1 = -fw
	}
	return n & 7, y0, y1
}
