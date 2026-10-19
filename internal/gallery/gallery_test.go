package gallery

import (
	"regexp"
	"testing"

	"github.com/firede/limu"
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Every example is of the type it is listed under, and limu lays it out at each width the
// website draws it at.
func TestExamplesLayOut(t *testing.T) {
	for _, typ := range All() {
		for _, ex := range typ.Examples {
			got, err := diagram.DetectType(ex.Source)
			if err != nil || got != typ.Type {
				t.Errorf("%s %q: detected as %q, %v", typ.Type, ex.Title, got, err)
				continue
			}
			d, err := limu.Parse(ex.Source, diagram.Options{})
			if err != nil {
				t.Errorf("%s %q: %v", typ.Type, ex.Title, err)
				continue
			}
			for _, w := range []float64{0, 640, 480, 320} {
				if _, err := d.Layout(&layout.Options{MaxWidth: w}); err != nil {
					t.Errorf("%s %q at %v: %v", typ.Type, ex.Title, w, err)
				}
			}
		}
	}
}

// Every type limu reads but info has one entry, with a title, a description, three to six
// examples, and one or two featured examples, which come first.
func TestTypes(t *testing.T) {
	seen := map[diagram.Type]bool{}
	for _, typ := range All() {
		if seen[typ.Type] {
			t.Errorf("%s: listed twice", typ.Type)
		}
		seen[typ.Type] = true
		if typ.Title == "" || typ.Description == "" {
			t.Errorf("%s: title %q, description %q", typ.Type, typ.Title, typ.Description)
		}
		if n := len(typ.Examples); n < 3 || n > 6 {
			t.Errorf("%s: %d examples", typ.Type, n)
		}
		featured, last := 0, true
		titles := map[string]bool{}
		for _, ex := range typ.Examples {
			if ex.Featured {
				featured++
				if !last {
					t.Errorf("%s %q: featured after one that is not", typ.Type, ex.Title)
				}
			}
			last = ex.Featured
			if ex.Title == "" || titles[ex.Title] {
				t.Errorf("%s: title %q empty or not unique", typ.Type, ex.Title)
			}
			titles[ex.Title] = true
		}
		if featured < 1 || featured > 2 {
			t.Errorf("%s: %d featured examples", typ.Type, featured)
		}
		if got, ok := Find(string(typ.Type)); !ok || got.Title != typ.Title {
			t.Errorf("Find(%q) = %q, %v", typ.Type, got.Title, ok)
		}
	}
	for _, typ := range limu.Types() {
		if typ != diagram.Info && !seen[typ] {
			t.Errorf("%s: no examples", typ)
		}
	}
	if seen[diagram.Info] {
		t.Errorf("info is listed")
	}
	if _, ok := Find("nothing"); ok {
		t.Errorf("Find found a type that is not one")
	}
}

// The types not in beta come first, each part by title.
func TestOrder(t *testing.T) {
	types := All()
	for i := 1; i < len(types); i++ {
		a, b := types[i-1], types[i]
		if a.Beta && !b.Beta || a.Beta == b.Beta && a.Title > b.Title {
			t.Errorf("%s before %s", a.Type, b.Type)
		}
	}
}

// placeholder matches the words that stand in for content in syntax's own examples; not
// "bar", which is a keyword of XY charts.
var placeholder = regexp.MustCompile(`\b(Label|foo|baz|Lorem|ipsum|Class0\d|id\d|Alice|Bob|John)\b`)

// An example draws something a reader could meet, not a stand-in for it.
func TestNoPlaceholders(t *testing.T) {
	for _, typ := range All() {
		for _, ex := range typ.Examples {
			if m := placeholder.FindString(ex.Source); m != "" {
				t.Errorf("%s %q: %q stands in for content", typ.Type, ex.Title, m)
			}
		}
	}
}
