package venn

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
	Model any            `json:"model"`
	Error map[string]any `json:"error"`
}

type peleCase struct {
	Src      string  `json:"src"`
	Type     *string `json:"type"`
	Tokens   any     `json:"tokens"`
	Indented any     `json:"indented"`
	Direct   peleResult
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Src      string         `json:"src"`
		Type     *string        `json:"type"`
		Tokens   any            `json:"tokens"`
		Indented any            `json:"indented"`
		Direct   peleResult     `json:"direct"`
		Model    any            `json:"model"`
		Error    map[string]any `json:"error"`
		// Same says the pipeline's result is the direct one.
		Same bool `json:"same"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Src, c.Type, c.Tokens, c.Indented, c.Direct = raw.Src, raw.Type, raw.Tokens, raw.Indented, raw.Direct
	c.peleResult = peleResult{raw.Model, raw.Error}
	if raw.Same {
		c.peleResult = raw.Direct
	}
	return nil
}

// fixedIndent answers the lexer's question about indentation always the same.
type fixedIndent bool

func (f fixedIndent) getIndentMode() bool { return bool(f) }
func (fixedIndent) setIndentMode(bool)    {}
func (f fixedIndent) lex(src string) []any {
	l := newLexer(src, f)
	out := []any{}
	for {
		typ := l.next()
		out = append(out, []any{tokenNames[typ], l.text, chars.Len16(src[:l.start])})
		if typ == tEnd || typ == tError {
			return out
		}
	}
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

func optional(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

func dumpModel(m *Model) map[string]any {
	subsets := []any{}
	for _, s := range m.Subsets {
		subsets = append(subsets, map[string]any{"sets": s.Sets, "size": enc(s.Size), "label": optional(s.Label)})
	}
	texts := []any{}
	for _, t := range m.TextNodes {
		texts = append(texts, map[string]any{"sets": t.Sets, "id": t.ID, "label": optional(t.Label)})
	}
	styles := []any{}
	for _, s := range m.Styles {
		decls := []any{}
		for _, d := range s.Styles {
			decls = append(decls, []any{d.Property, d.Value})
		}
		styles = append(styles, map[string]any{"targets": s.Targets, "styles": decls})
	}
	var title any
	if m.Title != "" {
		title = m.Title
	}
	return map[string]any{"title": title, "subsets": subsets, "textNodes": texts, "styles": styles}
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

// check compares a parse with Pele's.
func check(t *testing.T, what, text string, m *Model, err error, want peleResult) bool {
	t.Helper()
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
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/venn.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 8000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, piped, parsed := 0, 0, 0
	for _, c := range p.Cases {
		ok := compare(t, "tokens", c.Src, fixedIndent(false).lex(c.Src), c.Tokens) &&
			compare(t, "indented tokens", c.Src, fixedIndent(true).lex(c.Src), c.Indented)
		if ok {
			m, err := Parse(c.Src, "")
			ok = check(t, "Parse", c.Src, m, err, c.Direct)
		}
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
			case typ == diagram.Venn:
				m, err := ParsePrepared(pre)
				ok = check(t, "ParsePrepared", pre.Source(false), m, err, c.peleResult)
				piped++
				if err == nil {
					parsed++
				}
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	if piped < 7000 || parsed < 1800 {
		t.Errorf("only %d cases were venn diagrams, %d of them parsed", piped, parsed)
	}
}
