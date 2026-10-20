package limu

import (
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
)

// TestParseMatchesPele reads each source of the limu suite as a whole, with Parse and then
// Layout, as Pele's parse and render read it: the types it finds, as Pele and as Mermaid name
// them, or its error, which holds them once the type is known.
func TestParseMatchesPele(t *testing.T) {
	type peleError struct {
		Code        string  `json:"code"`
		Message     string  `json:"message"`
		Type        *string `json:"type"`
		MermaidType *string `json:"mermaidType"`
	}
	var f struct {
		Cases []struct {
			Source      string     `json:"source"`
			Type        string     `json:"type"`
			MermaidType string     `json:"mermaidType"`
			Parsed      bool       `json:"parsed"`
			Error       *peleError `json:"error"`
		} `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/limu.json", &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Cases) < 2000 {
		t.Fatalf("only %d cases", len(f.Cases))
	}
	str := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}
	failed := 0
	for _, c := range f.Cases {
		d, err := Parse(c.Source, diagram.Options{})
		if err == nil {
			_, err = d.Layout(nil)
		}
		switch {
		case c.Error != nil:
			want := diagram.Error{Code: diagram.Code(c.Error.Code), Message: c.Error.Message,
				Type: diagram.Type(str(c.Error.Type)), MermaidType: diagram.MermaidType(str(c.Error.MermaidType))}
			e, ok := err.(*diagram.Error)
			if !ok || (d != nil) != c.Parsed || e.Code != want.Code || e.Message != want.Message ||
				e.Type != want.Type || e.MermaidType != want.MermaidType {
				t.Errorf("%q: parsed %v, error %#v; Pele parsed %v, error %#v", c.Source, d != nil, err, c.Parsed, want)
				failed++
			}
		case err != nil:
			t.Errorf("%q: %v; Pele had none", c.Source, err)
			failed++
		case string(d.Type()) != c.Type || string(d.MermaidType()) != c.MermaidType:
			t.Errorf("%q: type %q (%q); Pele %q (%q)", c.Source, d.Type(), d.MermaidType(), c.Type, c.MermaidType)
			failed++
		}
		if failed > 20 {
			t.Fatal("too many failures")
		}
	}
}
