package ishikawa

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it: the tokens
// of the text given to the lexer as it is, the model or the error of the text given to the
// parser as it is, and the model or the error of the text as Pele's parse reads it, after the
// preprocessing every diagram goes through.

type peleCase struct {
	Src    string         `json:"src"`
	Tokens [][2]string    `json:"tokens"`
	Direct map[string]any `json:"direct"`
	Type   *string        `json:"type"`
	Model  any            `json:"model"`
	Error  map[string]any `json:"error"`
}

func dumpNode(n *Node) []any {
	children := []any{}
	for _, c := range n.Children {
		children = append(children, dumpNode(c))
	}
	return []any{n.Text, children}
}

func dumpModel(m *Model) map[string]any {
	out := map[string]any{"title": map[string]any{"$": "undefined"}, "root": nil}
	if m.HasTitle {
		out["title"] = m.Title
	}
	if m.Root != nil {
		out["root"] = dumpNode(m.Root)
	}
	return out
}

func dumpTokens(src string) [][2]string {
	t := tokenize(src)
	out := [][2]string{}
	for i, typ := range t.types[:len(t.types)-2] {
		out = append(out, [2]string{tokenNames[typ], src[t.starts[i]:t.ends[i]]})
	}
	return out
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

// result is a model or an error, as the fixture holds them.
func result(m *Model, err error) map[string]any {
	if err != nil {
		return map[string]any{"error": dumpError(err)}
	}
	return map[string]any{"model": dumpModel(m)}
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
	g, w := roundTrip(got), roundTrip(want)
	if reflect.DeepEqual(g, w) {
		return true
	}
	gj, _ := json.Marshal(g)
	wj, _ := json.Marshal(w)
	if len(gj) > 2000 {
		gj = append(gj[:2000], "…"...)
	}
	if len(wj) > 2000 {
		wj = append(wj[:2000], "…"...)
	}
	t.Errorf("%s of %q\n got %s\nPele %s", what, src, gj, wj)
	return false
}

func TestParseMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/ishikawa.json")
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
	failed := 0
	for _, c := range p.Cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Tokens)
		d := newDB()
		var direct *Model
		err := parseIshikawa(c.Src, d)
		if err == nil {
			direct = d.model
			direct.HasTitle = false
		}
		ok = compare(t, "parseIshikawa", c.Src, result(direct, err), c.Direct) && ok

		pre, err := diagram.Prepare(c.Src, diagram.Options{})
		switch {
		case err != nil:
			if e, isPele := err.(*diagram.Error); isPele && e.Code == diagram.UnsupportedDiagram && c.Type == nil && c.Error == nil {
				break
			}
			if c.Error == nil {
				t.Errorf("Prepare(%q): %v, Pele had none", c.Src, err)
				ok = false
			} else {
				ok = compare(t, "error", c.Src, dumpError(err), c.Error) && ok
			}
		case c.Type == nil || string(pre.Type) != *c.Type:
			t.Errorf("type of %q: %q, Pele %v", c.Src, pre.Type, c.Type)
			ok = false
		case pre.Type == diagram.Ishikawa:
			want := map[string]any{"model": c.Model}
			if c.Error != nil {
				want = map[string]any{"error": c.Error}
			}
			m, err := ParsePrepared(pre)
			ok = compare(t, "ParsePrepared", c.Src, result(m, err), want) && ok
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
