package treemap

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it: the tokens
// and the tree of the text given to the parser as it is, and the model of the text as it
// comes out of Pele's parseSource.

type peleDirect struct {
	Tokens [][2]string    `json:"tokens"`
	AST    map[string]any `json:"ast"`
	Error  map[string]any `json:"error"`
}

type peleCase struct {
	Src    string         `json:"src"`
	Type   *string        `json:"type"`
	Direct peleDirect     `json:"direct"`
	Model  map[string]any `json:"model"`
	Error  map[string]any `json:"error"`
}

var undefined = map[string]any{"$": "undefined"}

// enc is a number as the oracle writes it.
func enc(f float64) any { return yaml.Encode(f, false) }

func encStr(s string, has bool) any {
	if !has {
		return undefined
	}
	return s
}

func dumpAST(t *ast) map[string]any {
	rows := []any{}
	for _, r := range t.rows {
		indent := any(undefined)
		if r.hasIndent {
			indent = enc(float64(r.indent))
		}
		if r.classDef {
			rows = append(rows, []any{"classDef", indent, r.className, encStr(r.styleText, r.hasStyle)})
			continue
		}
		typ, value := "Section", any(undefined)
		if r.item.leaf {
			typ, value = "Leaf", enc(r.item.value)
		}
		rows = append(rows, []any{"row", indent, typ, r.item.name, value, encStr(r.item.selector, r.item.hasClass)})
	}
	return map[string]any{
		"title":    encStr(t.title, t.hasTitle),
		"accTitle": encStr(t.accTitle, t.hasAccTitle),
		"accDescr": encStr(t.accDescr, t.hasAccDescr),
		"rows":     rows,
	}
}

func dumpNode(n *Node) map[string]any {
	out := map[string]any{"name": n.Name, "children": undefined, "value": undefined, "classSelector": undefined, "css": undefined}
	if n.Section {
		children := []any{}
		for _, c := range n.Children {
			children = append(children, dumpNode(c))
		}
		out["children"] = children
	} else {
		out["value"] = enc(n.Value)
	}
	if n.Class != "" {
		out["classSelector"] = n.Class
	}
	if n.CSS != nil {
		out["css"] = n.CSS
	}
	return out
}

func dumpModel(m *Model) map[string]any {
	roots := []any{}
	for _, n := range m.Roots {
		roots = append(roots, dumpNode(n))
	}
	classes := []any{}
	for _, c := range m.Classes {
		classes = append(classes, []any{c.Name, c.Styles})
	}
	out := map[string]any{"roots": roots, "classes": classes}
	for k, v := range map[string]string{"title": m.Title, "accTitle": m.AccTitle, "accDescr": m.AccDescr} {
		if v != "" {
			out[k] = v
		}
	}
	return out
}

func dumpTokens(src string) [][2]string {
	toks, err := common.Tokenize(src, treemapTable, "treemap")
	if err != nil {
		return nil
	}
	out := [][2]string{}
	for i, k := range toks.Kinds[:len(toks.Kinds)-1] {
		out = append(out, [2]string{treemapTokens[k].Name, src[toks.Starts[i]:toks.Ends[i]]})
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
	data, err := fixture.Read("testdata/pele/treemap.json")
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
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Direct.Tokens)
		tree, err := parseTreemap(c.Src)
		switch {
		case err != nil && c.Direct.Error == nil:
			t.Errorf("parseTreemap(%q): %v, Pele had none", c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "parseTreemap error", c.Src, dumpError(err), c.Direct.Error) && ok
		case c.Direct.Error != nil:
			t.Errorf("parseTreemap(%q): no error, Pele %v", c.Src, c.Direct.Error)
			ok = false
		default:
			ok = compare(t, "tree", c.Src, dumpAST(tree), c.Direct.AST) && ok
		}

		p, err := diagram.Prepare(c.Src, diagram.Options{})
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
		case c.Type == nil || string(p.Type) != *c.Type:
			t.Errorf("type of %q: %q, Pele %v", c.Src, p.Type, c.Type)
			ok = false
		case p.Type == diagram.Treemap:
			m, err := ParsePrepared(p)
			switch {
			case err != nil && c.Error == nil:
				t.Errorf("ParsePrepared(%q): %v, Pele had none", c.Src, err)
				ok = false
			case err != nil:
				ok = compare(t, "ParsePrepared error", c.Src, dumpError(err), c.Error) && ok
			case c.Error != nil:
				t.Errorf("ParsePrepared(%q): no error, Pele %v", c.Src, c.Error)
				ok = false
			default:
				ok = compare(t, "model", c.Src, dumpModel(m), c.Model) && ok
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
