package gallery

import (
	"testing"

	"github.com/firede/limu"
	"github.com/firede/limu/diagram"
)

// Every example is of the type it is listed under, and limu lays it out.
func TestExamplesLayOut(t *testing.T) {
	for _, typ := range All() {
		for _, ex := range typ.Examples {
			got, err := diagram.DetectType(ex.Source)
			if err != nil || got != typ.Type {
				t.Errorf("%s %q: detected as %q, %v", typ.Type, ex.Title, got, err)
				continue
			}
			if _, err := limu.Layout(ex.Source, nil); err != nil {
				t.Errorf("%s %q: %v", typ.Type, ex.Title, err)
			}
		}
	}
}

// Every type limu reads but info has one entry, with a title, a description and a featured
// example, and its featured examples come first.
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

// The types not in beta come first, as on Pele's examples page. gen.mjs sorts each part by
// title with JavaScript's localeCompare, which this does not repeat.
func TestOrder(t *testing.T) {
	types := All()
	for i := 1; i < len(types); i++ {
		a, b := types[i-1], types[i]
		if a.Beta && !b.Beta {
			t.Errorf("%s, in beta, before %s", a.Type, b.Type)
		}
	}
}
