package main

import (
	"testing"

	"github.com/firede/limu/mygoui/examples/playground/internal/gallery"
)

// The toolbar goes to the page of examples, which shows the featured examples of every
// type; the sidebar shows all of one type's; and a card's Open loads its text in the
// playground.
func TestExamplesPage(t *testing.T) {
	a, tt := tester(t, "pie\n  \"a\": 1\n", appearanceLight)
	if err := tt.Click("Examples"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if p := a.router.Path(); p != "/examples" {
		t.Fatalf("the toolbar's Examples went to %q", p)
	}
	n := 0
	for _, g := range gallery.All() {
		for _, ex := range g.Examples {
			if ex.Featured {
				n++
			}
		}
	}
	if got := len(entries(featured)); got != n {
		t.Errorf("the page shows %d featured examples, want %d", got, n)
	}
	first := gallery.All()[0].Examples[0]
	if _, ok := tt.Find(first.Title); !ok {
		t.Errorf("the first card, %q, does not show", first.Title)
	}
	seq, _ := gallery.Find("sequence")
	if err := tt.Click(seq.Title); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if p := a.router.Path(); p != "/examples/sequence" {
		t.Fatalf("the sidebar's %s went to %q", seq.Title, p)
	}
	if got := len(entries("sequence")); got != len(seq.Examples) {
		t.Errorf("the sequence page shows %d examples, want %d", got, len(seq.Examples))
	}
	if _, ok := tt.Find(seq.Description); !ok {
		t.Errorf("the sequence page does not say what a sequence diagram is")
	}
	if err := tt.Click("Open"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if p := a.router.Path(); p != "/playground" || a.buf.String() != seq.Examples[0].Source || a.typ != "sequence" {
		t.Errorf("Open went to %q with a %q diagram", p, a.typ)
	}
	if err := tt.Click("Examples"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if p := a.router.Path(); p != "/examples/sequence" {
		t.Errorf("going back to the examples went to %q, not the type chosen last", p)
	}
}

// Every example's card lays out, as its drawing or as the error limu gives.
func TestCards(t *testing.T) {
	a := newApp("", appearanceLight, nil)
	a.ex.drawn = map[string]*card{}
	for _, g := range gallery.All() {
		for _, ex := range g.Examples {
			cd := a.drawing(ex.Source)
			if cd.pending || cd.sc == nil || cd.err != nil {
				t.Errorf("%s %q: %v", g.Type, ex.Title, cd.err)
			}
		}
	}
}
