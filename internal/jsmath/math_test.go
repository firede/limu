package jsmath

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixture is what V8 makes of Math.sin, Math.cos, Math.atan2, Math.acos and Math.pow;
// ./scripts/oracle.sh writes it, with its jsmath suite.
type mathFixture struct {
	Trig  [][3]json.RawMessage `json:"trig"`
	Atan2 [][3]json.RawMessage `json:"atan2"`
	Acos  [][2]json.RawMessage `json:"acos"`
	Pow   [][3]json.RawMessage `json:"pow"`
}

func readMath(t *testing.T) mathFixture {
	t.Helper()
	data, err := fixture.Read("testdata/pele/math.json")
	if err != nil {
		t.Fatal(err)
	}
	var p mathFixture
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

// angle reads an angle of the fixture: [k, n] is k·π/n, as k * Math.PI / n works it out.
func angle(raw json.RawMessage) float64 {
	var kn [2]float64
	if json.Unmarshal(raw, &kn) == nil {
		return kn[0] * math.Pi / kn[1]
	}
	return decode(raw)
}

func same(a, b float64) bool {
	return math.Float64bits(a) == math.Float64bits(b) || math.IsNaN(a) && math.IsNaN(b)
}

// check counts the cases whose results differ from V8's, reporting the first few.
type check struct {
	t      *testing.T
	name   string
	n, bad int
}

func (c *check) see(got, want float64, args ...float64) {
	c.n++
	if !same(got, want) {
		if c.bad++; c.bad <= 20 {
			c.t.Errorf("%s%v = %v (%#x), V8 %v (%#x)", c.name, args, got, math.Float64bits(got), want, math.Float64bits(want))
		}
	}
}

func (c *check) done(least int) {
	if c.bad > 0 || c.n < least {
		c.t.Errorf("%s: %d of %d differ", c.name, c.bad, c.n)
	}
}

func TestTrigMatchesV8(t *testing.T) {
	p := readMath(t)
	sin, cos := check{t: t, name: "Sin"}, check{t: t, name: "Cos"}
	for _, c := range p.Trig {
		x := angle(c[0])
		sin.see(Sin(x), decode(c[1]), x)
		cos.see(Cos(x), decode(c[2]), x)
	}
	sin.done(30000)
	cos.done(30000)
}

func TestAtan2MatchesV8(t *testing.T) {
	p := readMath(t)
	at := check{t: t, name: "Atan2"}
	for _, c := range p.Atan2 {
		y, x := decode(c[0]), decode(c[1])
		at.see(Atan2(y, x), decode(c[2]), y, x)
	}
	at.done(30000)
}

func TestAcosMatchesV8(t *testing.T) {
	p := readMath(t)
	ac := check{t: t, name: "Acos"}
	for _, c := range p.Acos {
		x := decode(c[0])
		ac.see(Acos(x), decode(c[1]), x)
	}
	ac.done(20000)
}

func TestPowMatchesV8(t *testing.T) {
	p := readMath(t)
	exact := check{t: t, name: "Pow"}
	near, n := 0, 0
	for _, c := range p.Pow {
		x, y := decode(c[0]), decode(c[1])
		got, want := Pow(x, y), decode(c[2])
		if powIsExact(x, y, want) {
			exact.see(got, want, x, y)
			continue
		}
		// Elsewhere V8 has the platform's pow, which on macOS is Apple's: a result it puts
		// one ulp from the nearest is allowed, if they are rare.
		n++
		if same(got, want) {
			continue
		}
		if d := int64(math.Float64bits(got)) - int64(math.Float64bits(want)); (d == 1 || d == -1) && math.Signbit(got) == math.Signbit(want) {
			near++
			continue
		}
		t.Errorf("Pow(%v, %v) = %v (%#x), V8 %v (%#x)", x, y, got, math.Float64bits(got), want, math.Float64bits(want))
	}
	exact.done(8000)
	t.Logf("Pow: %d of %d exact, %d of %d one ulp from V8's", exact.n, exact.n, near, n)
	if near*500 > n || n < 20000 {
		t.Errorf("Pow: %d of %d one ulp from V8's", near, n)
	}
}

// powIsExact reports whether Pow must give what V8 gives to the last bit: for the special
// values, the exponents V8 works out itself, and the powers Pele takes, of 10, 2 and 1/2 to
// whole exponents and of 0.99 to the first few, which Apple's pow rounds correctly.
func powIsExact(x, y, want float64) bool {
	finite := func(v float64) bool { return !math.IsInf(v, 0) && !math.IsNaN(v) }
	whole := y == math.Trunc(y)
	switch {
	case !finite(x) || !finite(y) || x == 0 || y == 0 || !finite(want) || want == 0:
		return true
	case y == 2 || y == 0.5:
		return true
	case whole && (x == 10 || x == 2 || x == 0.5):
		return true
	case whole && x == 0.99 && y >= 0 && y < 6:
		return true
	}
	return false
}
