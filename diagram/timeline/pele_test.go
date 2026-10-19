package timeline

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The tokens
// and the model are written as the oracle writes Pele's, field by field, with offsets in
// UTF-16 code units, so that any field that differs shows.

type peleResult struct {
	Model any            `json:"model"`
	Error map[string]any `json:"error"`
}

type peleCase struct {
	Src    string
	Type   *string
	Tokens []any
	Direct peleResult
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Src    string         `json:"src"`
		Type   *string        `json:"type"`
		Tokens []any          `json:"tokens"`
		Direct peleResult     `json:"direct"`
		Model  any            `json:"model"`
		Error  map[string]any `json:"error"`
		// Same says the pipeline's result is the direct one.
		Same bool `json:"same"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Src, c.Type, c.Tokens, c.Direct = raw.Src, raw.Type, raw.Tokens, raw.Direct
	c.peleResult = peleResult{raw.Model, raw.Error}
	if raw.Same {
		c.peleResult = raw.Direct
	}
	return nil
}

// object builds a JSON object, leaving out the fields Pele leaves undefined.
type object map[string]any

func (o object) opt(key, value string) {
	if value != "" {
		o[key] = value
	}
}

func dumpTokens(src string) []any {
	t := tokenize(src)
	out := []any{}
	for i, typ := range t.Types {
		out = append(out, []any{typ, t.Texts[i], chars.Len16(src[:t.Starts[i]])})
	}
	return out
}

func dumpModel(m *Model) object {
	o := object{"direction": m.Direction, "sections": append([]string{}, m.Sections...)}
	o.opt("title", m.Title)
	o.opt("accTitle", m.AccTitle)
	o.opt("accDescr", m.AccDescr)
	periods := []any{}
	for _, p := range m.Periods {
		periods = append(periods, object{"text": p.Text, "events": p.Events, "section": p.Section, "sectionIndex": p.SectionIndex})
	}
	o["periods"] = periods
	return o
}

func dumpError(err error) map[string]any {
	e, ok := err.(*diagram.Error)
	if !ok {
		return map[string]any{"crash": err.Error()}
	}
	out := map[string]any{"code": string(e.Code), "message": e.Message, "type": nil, "line": e.Line,
		"column": e.Column, "snippet": e.Snippet}
	if e.Type != "" {
		out["type"] = e.Type
	}
	return out
}

// roundTrip is v as JSON reads it back, for comparing with a fixture.
func roundTrip(v any) any {
	data, _ := json.Marshal(v)
	var out any
	json.Unmarshal(data, &out)
	return out
}

func compare(t *testing.T, what, src string, got, want any) bool {
	t.Helper()
	g := roundTrip(got)
	w := roundTrip(want)
	if reflect.DeepEqual(g, w) {
		return true
	}
	gj, _ := json.Marshal(g)
	wj, _ := json.Marshal(w)
	t.Errorf("%s of %q\n got %s\nPele %s", what, src, gj, wj)
	return false
}

// check compares a parse with Pele's result.
func check(t *testing.T, what, src string, m *Model, err error, want peleResult) bool {
	t.Helper()
	switch {
	case err != nil && want.Error == nil:
		t.Errorf("%s(%q): %v, Pele had none", what, src, err)
		return false
	case err != nil:
		return compare(t, what+" error", src, dumpError(err), want.Error)
	case want.Error != nil:
		t.Errorf("%s(%q): no error, Pele %v", what, src, want.Error)
		return false
	}
	return compare(t, what+" model", src, dumpModel(m), want.Model)
}

func TestParseMatchesPele(t *testing.T) {
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/timeline.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 5000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Tokens)
		if ok {
			m, err := Parse(c.Src, "")
			ok = check(t, "Parse", c.Src, m, err, c.Direct)
		}
		if ok {
			p, err := diagram.Prepare(c.Src, diagram.Options{})
			var typ diagram.Type
			if p != nil {
				typ = p.Type
			}
			switch {
			case err != nil:
				// Pele's suite stops where no type is found, with no error.
				if e, isPele := err.(*diagram.Error); isPele && e.Code == diagram.UnsupportedDiagram && c.Error == nil && c.Type == nil {
					break
				}
				if c.Error == nil {
					t.Errorf("Prepare(%q): %v, Pele had none", c.Src, err)
					ok = false
				} else {
					ok = compare(t, "error", c.Src, dumpError(err), c.Error)
				}
			case c.Type == nil && typ != "" || c.Type != nil && string(typ) != *c.Type:
				t.Errorf("type of %q: %q, Pele %v", c.Src, typ, c.Type)
				ok = false
			case typ == diagram.Timeline:
				m, err := ParsePrepared(p)
				ok = check(t, "ParsePrepared", c.Src, m, err, c.peleResult)
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
