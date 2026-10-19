package jsmath

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixture is what V8 makes of Math.sin, Math.cos, Math.atan2, Math.acos and Math.asin;
// ./scripts/oracle.sh writes it, with its jsmath suite.
type mathFixture struct {
	Trig  [][3]json.RawMessage `json:"trig"`
	Atan2 [][3]json.RawMessage `json:"atan2"`
	Acos  [][2]json.RawMessage `json:"acos"`
	Asin  [][2]json.RawMessage `json:"asin"`
}

func readMath(t *testing.T) mathFixture {
	t.Helper()
	var p mathFixture
	if err := fixture.Load("testdata/pele/math.json", &p); err != nil {
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

func TestAsinMatchesV8(t *testing.T) {
	p := readMath(t)
	as := check{t: t, name: "Asin"}
	for _, c := range p.Asin {
		x := decode(c[0])
		as.see(Asin(x), decode(c[1]), x)
	}
	as.done(25000)
}

var percent = 22.505

// A product passed to Round is rounded on its own before Round takes it, as JavaScript
// rounds it, and not fused with what Round subtracts from it once it is inlined.
func TestRoundOfAProduct(t *testing.T) {
	if r := Round(percent * 100); r != 2251 {
		t.Errorf("Round(22.505*100) = %v, Math.round gives 2251", r)
	}
}
