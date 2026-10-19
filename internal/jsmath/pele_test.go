package jsmath

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixture is what V8 makes of Math.log10; ./scripts/oracle.sh writes it, with the
// numbers JSON has no words for written as the oracle's enc writes them.

func decode(raw json.RawMessage) float64 {
	var special struct {
		N string `json:"$n"`
	}
	if json.Unmarshal(raw, &special) == nil && special.N != "" {
		switch special.N {
		case "NaN":
			return math.NaN()
		case "Infinity":
			return math.Inf(1)
		case "-Infinity":
			return math.Inf(-1)
		case "-0":
			return math.Copysign(0, -1)
		}
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		panic(err)
	}
	return f
}

func TestLog10MatchesV8(t *testing.T) {
	var p struct {
		Cases [][2]json.RawMessage `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/log10.json", &p); err != nil {
		t.Fatal(err)
	}
	failed := 0
	for _, c := range p.Cases {
		x, want := decode(c[0]), decode(c[1])
		got := Log10(x)
		if math.Float64bits(got) != math.Float64bits(want) && !(math.IsNaN(got) && math.IsNaN(want)) {
			if failed++; failed <= 20 {
				t.Errorf("Log10(%v) = %v, V8 %v", x, got, want)
			}
		}
	}
	if failed > 0 || len(p.Cases) < 20000 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// The fixture of format is what V8's toFixed, toExponential and toPrecision write of
// numbers: each case is a number, the digits given to toFixed and toExponential and what
// they write, and the precision given to toPrecision and what it writes.
func TestFormatMatchesV8(t *testing.T) {
	var p struct {
		Cases [][6]json.RawMessage `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/format.json", &p); err != nil {
		t.Fatal(err)
	}
	str := func(raw json.RawMessage) string {
		var s string
		json.Unmarshal(raw, &s)
		return s
	}
	failed := 0
	for _, c := range p.Cases {
		x := decode(c[0])
		f, prec := int(decode(c[1])), int(decode(c[4]))
		for _, check := range []struct {
			name      string
			n         int
			got, want string
		}{
			{"ToFixed", f, ToFixed(x, f), str(c[2])},
			{"ToExponential", f, ToExponential(x, f), str(c[3])},
			{"ToPrecision", prec, ToPrecision(x, prec), str(c[5])},
		} {
			if check.got != check.want {
				if failed++; failed <= 20 {
					t.Errorf("%s(%v, %d) = %s, V8 %s", check.name, x, check.n, check.got, check.want)
				}
			}
		}
	}
	if failed > 0 || len(p.Cases) < 20000 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}
