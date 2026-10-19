package jsmath

import (
	"math"
	"math/big"
	"math/bits"
	"sync"
)

// Pow is Math.pow(x, y) and x ** y, as V8's math::pow works them out with its default
// --use-std-math-pow: NaN for a NaN exponent and for ±1 to an infinite one, as ECMAScript
// asks; x*x for an exponent of 2 and the square root of x+0 for 0.5, as V8's optimizing
// compilers do; and the C library's pow for the rest.
//
// The C library's pow is the platform's, so Math.pow is not the same on every platform.
// Here it is pow correctly rounded, x^y to the nearest float64. Apple's pow, which Node
// calls on macOS, gives that but for about one result in a thousand, which it puts one
// ulp away, as it does for 0.99^317; glibc's, on Linux, misses about as often. Apple's
// gives it for the powers Pele takes: of 10, 2 and 1/2 to whole exponents, and of 0.99
// to the first few.
func Pow(x, y float64) float64 {
	switch {
	case math.IsNaN(y):
		return math.NaN()
	case math.IsInf(y, 0) && (x == 1 || x == -1):
		return math.NaN()
	case y == 2:
		return float64(x * x)
	case y == 0.5:
		if math.IsInf(x, 0) {
			return math.Inf(1)
		}
		return math.Sqrt(x + 0)
	}
	return cPow(x, y)
}

// cPow is C's pow, with the special cases C99 gives it, and its value otherwise rounded
// correctly.
func cPow(x, y float64) float64 {
	switch {
	case y == 0 || x == 1:
		return 1
	case math.IsNaN(x) || math.IsNaN(y):
		return math.NaN()
	}
	yInt := y == math.Trunc(y)
	yOdd := yInt && math.Abs(y) < 1<<53 && int64(y)&1 == 1
	switch {
	case x == 0:
		if y < 0 {
			if yOdd {
				return math.Copysign(math.Inf(1), x)
			}
			return math.Inf(1)
		}
		if yOdd {
			return x
		}
		return 0
	case math.IsInf(y, 0):
		if x == -1 {
			return 1
		}
		if (math.Abs(x) < 1) == (y > 0) {
			return 0
		}
		return math.Inf(1)
	case math.IsInf(x, 0):
		if y < 0 {
			if x < 0 && yOdd {
				return math.Copysign(0, -1)
			}
			return 0
		}
		if x < 0 && yOdd {
			return math.Inf(-1)
		}
		return math.Inf(1)
	case x < 0 && !yInt:
		return math.NaN()
	}
	r := powPositive(math.Abs(x), y)
	if x < 0 && yOdd {
		return -r
	}
	return r
}

// powPositive is a^y rounded to the nearest float64, for a finite a > 0 and a finite y ≠ 0.
// A power that is a float64 or halfway between two is worked out exactly; any other is
// worked out to more and more bits until the bits it might yet be wrong in no longer
// change how it rounds, as Ziv's method does.
func powPositive(a, y float64) float64 {
	// Out of range by far, whichever way it is rounded.
	t := y * math.Log(a) / math.Ln2
	if t > 1100 {
		return math.Inf(1)
	}
	if t < -1200 {
		return 0
	}
	if r, ok := exactPow(a, y); ok {
		return r
	}
	ya := new(big.Float).SetFloat64(y)
	for p := uint(96); ; p *= 2 {
		w := p + 64
		z := new(big.Float).SetPrec(w).Mul(bigLog(a, w), ya)
		r := bigExp(z, w)
		// r is a^y to within a few parts in 2^w; a^y is surely between r less and r more
		// 2^(16-p) of it, and has the float64 they both round to, if they round alike.
		eps := new(big.Float).SetMantExp(big.NewFloat(1), 16-int(p))
		lo := new(big.Float).SetPrec(w).Sub(r, new(big.Float).SetPrec(w).Mul(r, eps))
		hi := new(big.Float).SetPrec(w).Add(r, new(big.Float).SetPrec(w).Mul(r, eps))
		flo, _ := lo.Float64()
		fhi, _ := hi.Float64()
		if flo == fhi {
			return flo
		}
		if p >= 1<<13 { // never reached: a power that is not dyadic rounds long before
			return flo
		}
	}
}

// exactPow is a^y, and true, when it is a dyadic rational, which it must be to be a float64
// or halfway between two, and so to be beyond what powPositive's approximations can round.
// With a = m·2^e for an odd m, that is when m is 1 and e·y whole, when y is whole and m^y
// short enough, or when y is n/2^k and m a 2^k-th power; m < 2^53 bounds k and n.
func exactPow(a, y float64) (float64, bool) {
	m, e := oddPart(a)
	if m == 1 {
		ey := new(big.Float).SetPrec(128).Mul(big.NewFloat(float64(e)), big.NewFloat(y))
		if !ey.IsInt() {
			return 0, false
		}
		n, _ := ey.Int64()
		return math.Ldexp(1, int(n)), true
	}
	if y == math.Trunc(y) {
		if math.Abs(y) > 64 { // 3^35 > 2^54: m^y is too long to be a float64 or a midpoint
			return 0, false
		}
		return scaledPower(m, int64(y), int64(y)*e), true
	}
	if y < 0 {
		return 0, false
	}
	// y = n/2^k with n odd.
	k := 0
	n := y
	for n != math.Trunc(n) {
		n *= 2
		k++
		if k > 5 { // m ≥ 3^(2^k) would not fit in 53 bits
			return 0, false
		}
	}
	if n > 54 || (int64(n)*e)%(1<<k) != 0 {
		return 0, false
	}
	u := m
	for range k {
		s := isqrt(u)
		if s*s != u {
			return 0, false
		}
		u = s
	}
	return scaledPower(u, int64(n), int64(n)*e>>k), true
}

// oddPart is a as m·2^e for an odd m.
func oddPart(a float64) (uint64, int64) {
	b := math.Float64bits(a)
	exp := int64(b>>52) & 0x7ff
	mant := b & (1<<52 - 1)
	if exp == 0 {
		exp = 1
	} else {
		mant |= 1 << 52
	}
	tz := bits.TrailingZeros64(mant)
	return mant >> tz, exp - 1075 + int64(tz)
}

// isqrt is the integer square root of u, rounded down.
func isqrt(u uint64) uint64 {
	s := uint64(math.Sqrt(float64(u)))
	for s*s > u {
		s--
	}
	for (s+1)*(s+1) <= u {
		s++
	}
	return s
}

// scaledPower is m^n·2^s rounded to the nearest float64.
func scaledPower(m uint64, n, s int64) float64 {
	p := new(big.Int).Exp(new(big.Int).SetUint64(m), big.NewInt(max(n, -n)), nil)
	num, den := big.NewInt(1), big.NewInt(1)
	if n >= 0 {
		num = p
	} else {
		den = p
	}
	if s >= 0 {
		num.Lsh(num, uint(s))
	} else {
		den.Lsh(den, uint(-s))
	}
	f, _ := new(big.Rat).SetFrac(num, den).Float64()
	return f
}

// bigLog is ln a, to about w bits, for a finite a > 0: a = m·2^e with m between √2/2 and
// √2, and ln m = 2·atanh((m-1)/(m+1)), whose series converges by more than 5 bits a term.
func bigLog(a float64, w uint) *big.Float {
	wp := w + 16
	m, e := math.Frexp(a)
	if m < math.Sqrt2/2 {
		m *= 2
		e--
	}
	t := new(big.Float).SetPrec(wp).SetFloat64(m - 1) // exact, as m is near 1
	den := new(big.Float).SetPrec(wp).SetFloat64(m)
	t.Quo(t, den.Add(den, big.NewFloat(1)))
	t2 := new(big.Float).SetPrec(wp).Mul(t, t)
	sum := new(big.Float).SetPrec(wp).Set(t)
	term := new(big.Float).SetPrec(wp).Set(t)
	q := new(big.Float).SetPrec(wp)
	for k := int64(1); term.Sign() != 0; k++ {
		term.Mul(term, t2)
		q.Quo(term, new(big.Float).SetInt64(2*k+1))
		sum.Add(sum, q)
		if q.Sign() == 0 || q.MantExp(nil) < sum.MantExp(nil)-int(wp)-4 {
			break
		}
	}
	sum.SetMantExp(sum, 1)
	if e != 0 {
		sum.Add(sum, new(big.Float).SetPrec(wp).Mul(ln2(wp), new(big.Float).SetInt64(int64(e))))
	}
	return sum
}

// bigExp is e^z, to about w bits, for |z| below about 900: z = k·ln 2 + r, and e^r is
// (e^(r/2^10))^(2^10), from its Taylor series.
func bigExp(z *big.Float, w uint) *big.Float {
	const s = 10
	wp := w + 32
	zf, _ := z.Float64()
	k := int64(math.Round(zf / math.Ln2))
	r := new(big.Float).SetPrec(wp).Mul(ln2(wp+16), new(big.Float).SetInt64(k))
	r.Sub(z, r)
	r.SetMantExp(r, -s)
	sum := new(big.Float).SetPrec(wp).SetInt64(1)
	term := new(big.Float).SetPrec(wp).SetInt64(1)
	for n := int64(1); ; n++ {
		term.Mul(term, r)
		term.Quo(term, new(big.Float).SetInt64(n))
		sum.Add(sum, term)
		if term.Sign() == 0 || term.MantExp(nil) < -int(wp)-4 {
			break
		}
	}
	for range s {
		sum.Mul(sum, sum)
	}
	return sum.SetMantExp(sum, int(k))
}

var ln2Cache struct {
	sync.Mutex
	v *big.Float
}

// ln2 is ln 2 to w bits or more: 2·atanh(1/3), worked out once to the most bits asked yet.
func ln2(w uint) *big.Float {
	ln2Cache.Lock()
	defer ln2Cache.Unlock()
	if ln2Cache.v != nil && ln2Cache.v.Prec() >= w {
		return ln2Cache.v
	}
	wp := max(w, 512) + 16
	third := new(big.Float).SetPrec(wp).Quo(big.NewFloat(1), big.NewFloat(3))
	ninth := new(big.Float).SetPrec(wp).Mul(third, third)
	sum := new(big.Float).SetPrec(wp).Set(third)
	term := new(big.Float).SetPrec(wp).Set(third)
	q := new(big.Float).SetPrec(wp)
	for k := int64(1); ; k++ {
		term.Mul(term, ninth)
		q.Quo(term, new(big.Float).SetInt64(2*k+1))
		sum.Add(sum, q)
		if q.MantExp(nil) < sum.MantExp(nil)-int(wp)-4 {
			break
		}
	}
	sum.SetMantExp(sum, 1)
	ln2Cache.v = sum
	return sum
}
