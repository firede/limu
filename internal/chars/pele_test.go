package chars

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/firede/limu/internal/fixture"
)

// The fixtures are what Pele and JavaScript make of the same cases; ./scripts/oracle.sh
// writes them.

type peleChars struct {
	Letters  [][2]int `json:"letters"`
	ToString []struct {
		N json.RawMessage `json:"n"`
		S string          `json:"s"`
	} `json:"toString"`
	Number []struct {
		S   string          `json:"s"`
		N   json.RawMessage `json:"n"`
		Hex json.RawMessage `json:"hex"`
		Dec json.RawMessage `json:"dec"`
	} `json:"number"`
	ToLowerCase []struct {
		S     string `json:"s"`
		Lower string `json:"lower"`
	} `json:"toLowerCase"`
}

// number reads a number as the oracle writes it, with {"$n": …} for those JSON has no word for.
func number(t *testing.T, raw json.RawMessage) float64 {
	f, err := fixture.Float(raw)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func same(a, b float64) bool {
	return a == b && math.Signbit(a) == math.Signbit(b) || math.IsNaN(a) && math.IsNaN(b)
}

func load(t *testing.T) peleChars {
	var p peleChars
	if err := fixture.Load("testdata/pele/chars.json", &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUnicodeLettersMatchPele(t *testing.T) {
	p := load(t)
	want := make([]bool, 0x10000)
	for _, r := range p.Letters {
		for c := r[0]; c <= r[1]; c++ {
			want[c] = true
		}
	}
	for c := range want {
		if IsUnicodeLetter(c) != want[c] {
			t.Errorf("IsUnicodeLetter(%#x) = %v", c, !want[c])
		}
	}
}

func TestToLowerMatchesJavaScript(t *testing.T) {
	p := load(t)
	if len(p.ToLowerCase) < 1000 {
		t.Fatalf("only %d cases", len(p.ToLowerCase))
	}
	for _, c := range p.ToLowerCase {
		if got := ToLower(c.S); got != c.Lower {
			t.Errorf("ToLower(%q) = %q, JavaScript %q", c.S, got, c.Lower)
		}
	}
}

func TestNumbersMatchJavaScript(t *testing.T) {
	p := load(t)
	for _, c := range p.ToString {
		n := number(t, c.N)
		if got := NumberString(n); got != c.S {
			t.Errorf("NumberString(%v) = %q, JavaScript %q", n, got, c.S)
		}
	}
	for _, c := range p.Number {
		if got, want := Number(c.S), number(t, c.N); !same(got, want) {
			t.Errorf("Number(%q) = %v, JavaScript %v", c.S, got, want)
		}
		if got, want := ParseInt(c.S, 16), number(t, c.Hex); !same(got, want) {
			t.Errorf("ParseInt(%q, 16) = %v, JavaScript %v", c.S, got, want)
		}
		if got, want := ParseInt(c.S, 10), number(t, c.Dec); !same(got, want) {
			t.Errorf("ParseInt(%q, 10) = %v, JavaScript %v", c.S, got, want)
		}
	}
}

func TestUnits(t *testing.T) {
	s := "a😀é"
	if Len16(s) != 4 {
		t.Error(Len16(s))
	}
	if c, size := UnitAt(s, 1); c != 0xd83d || size != 4 {
		t.Error(c, size)
	}
	if c := UnitBefore(s, 5); c != 0xde00 {
		t.Error(c)
	}
	if c, _ := UnitAt(s, 9); c != -1 {
		t.Error(c)
	}
	if ToLower("İA") != "i̇a" {
		t.Error(ToLower("İA"))
	}
}
