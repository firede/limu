package sankey

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
// and the model are written as the oracle writes Pele's, field by field, with offsets in
// UTF-16 code units and numbers JSON has no words for spelled out, so that any field that
// differs shows.

type peleResult struct {
	Model any            `json:"model"`
	Error map[string]any `json:"error"`
}

type peleCase struct {
	Src      string
	Type     *string
	Tokens   []any
	Prepared string
	Direct   peleResult
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Src      string         `json:"src"`
		Type     *string        `json:"type"`
		Tokens   []any          `json:"tokens"`
		Prepared string         `json:"prepared"`
		Direct   peleResult     `json:"direct"`
		Model    any            `json:"model"`
		Error    map[string]any `json:"error"`
		// Same says the pipeline's result is the direct one.
		Same bool `json:"same"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Src, c.Type, c.Tokens, c.Prepared, c.Direct = raw.Src, raw.Type, raw.Tokens, raw.Prepared, raw.Direct
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
	o := object{}
	o.opt("title", m.Title)
	o.opt("accTitle", m.AccTitle)
	o.opt("accDescr", m.AccDescr)
	nodes := []any{}
	for _, n := range m.Nodes {
		nodes = append(nodes, []any{n.ID, n.Index})
	}
	links := []any{}
	for _, l := range m.Links {
		links = append(links, []any{m.Nodes[l.Source].Index, m.Nodes[l.Target].Index, fixture.Number(l.Value)})
	}
	o["nodes"], o["links"] = nodes, links
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
	if err := fixture.Load("testdata/pele/sankey.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 7000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, models := 0, 0
	for _, c := range p.Cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Tokens)
		if ok {
			ok = compare(t, "prepared text", c.Src, prepareTextForParsing(c.Src, nil), c.Prepared)
		}
		if ok {
			m := newModel()
			err := parseSankey(c.Src, m, c.Src)
			ok = check(t, "parseSankey", c.Src, m, err, c.Direct)
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
			case typ == diagram.Sankey:
				m, err := ParsePrepared(p)
				ok = check(t, "ParsePrepared", c.Src, m, err, c.peleResult)
				if err == nil {
					models++
				}
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	if models < 1000 {
		t.Errorf("only %d cases parse", models)
	}
}

func TestParseFloat(t *testing.T) {
	for s, want := range map[string]float64{
		"1": 1, " 2.5x": 2.5, ".5": 0.5, "5.": 5, "-.5": -0.5, "+3": 3, "1e3": 1000, "1e": 1, "1e+": 1, "1E-2": 0.01,
		"0x10": 0, "1_000": 1, "Infinityx": inf(1), "-Infinity": inf(-1), "1e400": inf(1), " \n7": 7,
	} {
		if got := parseFloat(s); got != want {
			t.Errorf("parseFloat(%q) = %v, want %v", s, got, want)
		}
	}
	for _, s := range []string{"", "x", ".", "-", "+.e1", "infinity", "e5"} {
		if got := parseFloat(s); got == got {
			t.Errorf("parseFloat(%q) = %v, want NaN", s, got)
		}
	}
}

func inf(sign int) float64 { return math.Inf(sign) }
