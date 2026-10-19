package jsdate

import (
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/firede/limu/internal/fixture"
)

// The fixture is what V8, running Pele, makes of the same dates in each zone;
// ./scripts/oracle.sh writes it. A null is NaN.

type zoneCases struct {
	Zone   string               `json:"zone"`
	Parse  [][2]json.RawMessage `json:"parse"`
	Fields [][2]json.RawMessage `json:"fields"`
	Built  [][3]json.RawMessage `json:"built"`
	Set    [][4]json.RawMessage `json:"set"`
}

func num(raw json.RawMessage) float64 {
	if string(raw) == "null" {
		return math.NaN()
	}
	var f float64
	if err := json.Unmarshal(raw, &f); err != nil {
		panic(err)
	}
	return f
}

func nums(raw json.RawMessage) []float64 {
	var vs []json.RawMessage
	if err := json.Unmarshal(raw, &vs); err != nil {
		panic(err)
	}
	out := make([]float64, len(vs))
	for i, v := range vs {
		out[i] = num(v)
	}
	return out
}

func same(a, b float64) bool { return a == b || math.IsNaN(a) && math.IsNaN(b) }

func TestDatesMatchV8(t *testing.T) {
	var p struct {
		Zones []zoneCases `json:"zones"`
	}
	if err := fixture.Load("testdata/pele/jsdate.json", &p); err != nil {
		t.Fatal(err)
	}
	for _, zc := range p.Zones {
		loc, err := time.LoadLocation(zc.Zone)
		if err != nil {
			t.Fatal(err)
		}
		z := In(loc)
		failed := 0
		fail := func(format string, args ...any) {
			if failed++; failed <= 20 {
				t.Errorf(zc.Zone+": "+format, args...)
			}
		}
		for _, c := range zc.Parse {
			var s string
			json.Unmarshal(c[0], &s)
			if got, want := z.Parse(s), num(c[1]); !same(got, want) {
				fail("Parse(%q) = %v, V8 %v", s, got, want)
			}
		}
		for _, c := range zc.Fields {
			tv := num(c[0])
			want := nums(c[1])
			got := []float64{z.FullYear(tv), z.Month(tv), z.Date(tv), z.Day(tv), z.Hours(tv), z.Minutes(tv), z.Seconds(tv),
				z.Milliseconds(tv), z.TimezoneOffset(tv), UTCSeconds(tv)}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				fail("fields of %v = %v, V8 %v", tv, got, want)
			}
		}
		for _, c := range zc.Built {
			f := nums(c[0])
			if got, want := z.New(f[0], f[1], f[2], f[3], f[4], f[5], f[6]), num(c[1]); !same(got, want) {
				fail("New(%v) = %v, V8 %v", f, got, want)
			}
			if got, want := UTC(f[0], f[1], f[2], f[3], f[4], f[5], f[6]), num(c[2]); !same(got, want) {
				fail("UTC(%v) = %v, V8 %v", f, got, want)
			}
		}
		for _, c := range zc.Set {
			tv := num(c[0])
			var op string
			json.Unmarshal(c[1], &op)
			var got float64
			switch op {
			case "date":
				got = z.SetDate(tv, num(c[2]))
			case "month":
				got = z.SetMonth(tv, num(c[2]))
			case "monthDate":
				a := nums(c[2])
				got = z.SetMonthDate(tv, a[0], a[1])
			case "fullYear":
				got = z.SetFullYear(tv, num(c[2]))
			case "hours":
				a := nums(c[2])
				got = z.SetHours(tv, a[0], a[1], a[2], a[3])
			}
			if want := num(c[3]); !same(got, want) {
				fail("set %s(%v, %s) = %v, V8 %v", op, tv, c[2], got, want)
			}
		}
		if failed > 0 {
			t.Errorf("%s: %d differ", zc.Zone, failed)
		}
		if len(zc.Parse) < 5000 || len(zc.Fields) < 3000 || len(zc.Built) < 3000 {
			t.Errorf("%s: too few cases", zc.Zone)
		}
	}
}
