package treemap

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixture of format is what Pele's formatter writes of values with specifiers of
// d3-format; ./scripts/oracle.sh writes it, with the numbers JSON has no words for written as
// the oracle's enc writes them.

func decodeNumber(raw json.RawMessage) float64 {
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

func TestFormatterMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/format.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Cases [][3]json.RawMessage `json:"cases"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	failed := 0
	for _, c := range p.Cases {
		var spec, want string
		json.Unmarshal(c[0], &spec)
		json.Unmarshal(c[2], &want)
		v := decodeNumber(c[1])
		if got := formatter(spec)(v); got != want {
			if failed++; failed <= 20 {
				t.Errorf("formatter(%q)(%v) = %q, Pele %q", spec, v, got, want)
			}
		}
	}
	if failed > 0 || len(p.Cases) < 10000 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}
