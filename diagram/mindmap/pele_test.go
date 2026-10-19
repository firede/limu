package mindmap

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it: the tokens
// of each case's text, with offsets in UTF-16 code units, and the model of the map or the
// error, from the text as it is and as Pele's parse is given it.

type peleResult struct {
	Model any            `json:"model"`
	Error map[string]any `json:"error"`
}

type peleTokens struct {
	Types  []int    `json:"types"`
	Texts  []string `json:"texts"`
	Starts []int    `json:"starts"`
}

type peleCase struct {
	Src    string
	Type   *string
	Tokens peleTokens
	Direct peleResult
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Src    string         `json:"src"`
		Type   *string        `json:"type"`
		Tokens peleTokens     `json:"tokens"`
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

// dumpModel writes a model as the oracle writes Pele's, without the fields Pele leaves
// undefined: a node's children by their ids, the root's section, and a class or an icon
// not given.
func dumpModel(m *Model) map[string]any {
	nodes := []any{}
	for _, n := range m.Nodes {
		children := []int{}
		for _, c := range n.Children {
			children = append(children, c.ID)
		}
		o := map[string]any{"id": n.ID, "nodeId": n.NodeID, "level": n.Level, "descr": n.Descr, "type": n.Type,
			"children": children, "isRoot": n.IsRoot}
		if !n.IsRoot {
			o["section"] = n.Section
		}
		if n.Class != "" {
			o["class"] = n.Class
		}
		if n.Icon != "" {
			o["icon"] = n.Icon
		}
		nodes = append(nodes, o)
	}
	out := map[string]any{"nodes": nodes}
	if m.Title != "" {
		out["title"] = m.Title
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

// tokens writes a source's tokens as the oracle writes Pele's, with offsets in UTF-16 code
// units.
func tokens(src string) peleTokens {
	tok := Tokenize(src)
	out := peleTokens{Types: tok.Types, Texts: tok.Texts, Starts: make([]int, len(tok.Starts))}
	for i, s := range tok.Starts {
		out.Starts[i] = chars.Len16(src[:s])
	}
	return out
}

func TestParseMatchesPele(t *testing.T) {
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/mindmap.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 5500 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
		ok := compare(t, "tokens", c.Src, tokens(c.Src), c.Tokens)
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
			case typ == diagram.Mindmap:
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
