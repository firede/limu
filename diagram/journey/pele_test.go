package journey

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The tokens
// and the model are written as the oracle writes Pele's, with offsets in UTF-16 code units,
// so that any field that differs shows.

type peleResult struct {
	Tokens any            `json:"tokens"`
	Model  any            `json:"model"`
	Error  map[string]any `json:"error"`
}

type peleCase struct {
	Src    string  `json:"src"`
	Type   *string `json:"type"`
	Direct peleResult
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Src    string         `json:"src"`
		Type   *string        `json:"type"`
		Direct peleResult     `json:"direct"`
		Tokens any            `json:"tokens"`
		Model  any            `json:"model"`
		Error  map[string]any `json:"error"`
		// Same says the pipeline's result is the direct one.
		Same bool `json:"same"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Src, c.Type, c.Direct = raw.Src, raw.Type, raw.Direct
	c.peleResult = peleResult{raw.Tokens, raw.Model, raw.Error}
	if raw.Same {
		c.peleResult = raw.Direct
	}
	return nil
}

// enc is a number as the oracle's enc writes it.
func enc(f float64) any {
	switch {
	case math.IsNaN(f):
		return map[string]any{"$n": "NaN"}
	case math.IsInf(f, 1):
		return map[string]any{"$n": "Infinity"}
	case math.IsInf(f, -1):
		return map[string]any{"$n": "-Infinity"}
	case f == 0 && math.Signbit(f):
		return map[string]any{"$n": "-0"}
	}
	return f
}

func strs(s []string) []any {
	out := make([]any, len(s))
	for i, x := range s {
		out[i] = x
	}
	return out
}

func dumpTokens(src string) []any {
	t := tokenize(src)
	out := []any{}
	for i, typ := range t.types {
		out = append(out, []any{tokenNames[typ], t.texts[i], chars.Len16(src[:t.starts[i]])})
	}
	return out
}

func dumpModel(m *Model) map[string]any {
	tasks := []any{}
	for _, t := range m.Tasks {
		tasks = append(tasks, map[string]any{"task": t.Task, "score": enc(t.Score), "people": strs(t.People), "section": t.Section})
	}
	o := map[string]any{"sections": strs(m.Sections), "tasks": tasks}
	opt := func(key, value string) {
		if value != "" {
			o[key] = value
		}
	}
	opt("title", m.Title)
	opt("accTitle", m.AccTitle)
	opt("accDescr", m.AccDescr)
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

// check compares the tokens of text given to the parser, and its parse, with Pele's.
func check(t *testing.T, what, text string, m *Model, err error, want peleResult) bool {
	t.Helper()
	if !compare(t, what+" tokens", text, dumpTokens(text), want.Tokens) {
		return false
	}
	switch {
	case err != nil && want.Error == nil:
		t.Errorf("%s(%q): %v, Pele had none", what, text, err)
		return false
	case err != nil:
		return compare(t, what+" error", text, dumpError(err), want.Error)
	case want.Error != nil:
		t.Errorf("%s(%q): no error, Pele %v", what, text, want.Error)
		return false
	}
	return compare(t, what+" model", text, dumpModel(m), want.Model)
}

func TestParseMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/journey.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 5000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, piped := 0, 0
	for _, c := range p.Cases {
		m, err := Parse(c.Src, "")
		ok := check(t, "Parse", c.Src, m, err, c.Direct)
		if ok {
			pre, err := diagram.Prepare(c.Src, diagram.Options{})
			var typ diagram.Type
			if pre != nil {
				typ = pre.Type
			}
			switch {
			case err != nil:
				if e, isPele := err.(*diagram.Error); isPele && e.Code == diagram.UnsupportedDiagram && c.Error == nil {
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
			case typ == diagram.Journey:
				m, err := ParsePrepared(pre)
				ok = check(t, "ParsePrepared", pre.Source(false), m, err, c.peleResult)
				piped++
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	if piped < 3000 {
		t.Errorf("only %d cases were journey diagrams", piped)
	}
}
