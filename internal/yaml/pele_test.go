package yaml

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixtures are what Pele makes of the same cases; ./scripts/oracle.sh writes them.

func TestParseMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/yaml.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		YAML []struct {
			Source string  `json:"source"`
			Value  any     `json:"value"`
			Error  *string `json:"error"`
		} `json:"yaml"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.YAML) < 1000 {
		t.Fatalf("only %d cases", len(p.YAML))
	}
	failed := 0
	for _, c := range p.YAML {
		v, err := Parse(c.Source)
		if c.Error != nil {
			if err == nil || err.Error() != *c.Error {
				got, _ := json.Marshal(Encode(v, false))
				t.Errorf("Parse(%q)\n got %v, %s\nPele error %q", c.Source, err, got, *c.Error)
				failed++
			}
		} else if err != nil {
			t.Errorf("Parse(%q): %v\nPele %v", c.Source, err, c.Value)
			failed++
		} else {
			raw, _ := json.Marshal(Encode(v, false))
			var got any
			json.Unmarshal(raw, &got)
			if !reflect.DeepEqual(got, c.Value) {
				want, _ := json.Marshal(c.Value)
				t.Errorf("Parse(%q)\n got %s\nPele %s", c.Source, raw, want)
				failed++
			}
		}
		if failed > 20 {
			t.Fatal("too many failures")
		}
	}
}
