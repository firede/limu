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
	data, err := fixture.Read("testdata/pele/log10.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Cases [][2]json.RawMessage `json:"cases"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
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
