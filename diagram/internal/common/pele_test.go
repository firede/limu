package common

import (
	"encoding/json"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it.
func TestGenericsMatchPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/common.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Generics []struct{ Input, Output string } `json:"generics"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Generics) < 1000 {
		t.Fatalf("only %d cases", len(p.Generics))
	}
	for _, c := range p.Generics {
		if got := ParseGenericTypes(c.Input); got != c.Output {
			t.Errorf("ParseGenericTypes(%q) = %q, Pele %q", c.Input, got, c.Output)
		}
	}
}
