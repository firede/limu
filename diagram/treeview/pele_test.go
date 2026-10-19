package treeview

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it: what the
// box-drawing preprocessor makes of the text, the tokens and the tree of the text given to
// the parser as it is, and the model of the text as it comes out of Pele's parseSource, with
// each entry's icon and the layout settings.

type peleDirect struct {
	Tokens [][2]string    `json:"tokens"`
	AST    map[string]any `json:"ast"`
	Error  map[string]any `json:"error"`
}

type peleBox struct {
	Text    *string        `json:"text"`
	LineMap [][2]int       `json:"lineMap"`
	Error   map[string]any `json:"error"`
}

type peleCase struct {
	Src    string         `json:"src"`
	Type   *string        `json:"type"`
	Direct peleDirect     `json:"direct"`
	Box    peleBox        `json:"box"`
	Model  map[string]any `json:"model"`
	Error  map[string]any `json:"error"`
}

var undefined = map[string]any{"$": "undefined"}

func encStr(s string, has bool) any {
	if !has {
		return undefined
	}
	return s
}

// encOpt is a string Pele leaves undefined when it is empty.
func encOpt(s string) any { return encStr(s, s != "") }

func dumpAST(t *ast) map[string]any {
	nodes := []any{}
	for _, n := range t.nodes {
		var indent any = undefined
		if n.hasIndent {
			indent = n.indent
		}
		nodes = append(nodes, []any{indent, n.name, encStr(n.classAnnotation, n.hasClass), encStr(n.iconAnnotation, n.hasIcon),
			encStr(n.descAnn, n.hasDesc)})
	}
	return map[string]any{
		"title":    encStr(t.title, t.hasTitle),
		"accTitle": encStr(t.accTitle, t.hasAccTitle),
		"accDescr": encStr(t.accDescr, t.hasAccDescr),
		"nodes":    nodes,
	}
}

func dumpNode(n *Node, icons IconConfig) []any {
	children := []any{}
	for _, c := range n.Children {
		children = append(children, dumpNode(c, icons))
	}
	return []any{n.ID, n.Level, n.Name, string(n.Type), encOpt(n.Icon), encOpt(n.CSSClass), encOpt(n.Description),
		encOpt(getNodeIcon(n, icons)), children}
}

func dumpModel(m *Model, config diagram.Config) map[string]any {
	icons := iconConfig(config)
	s := settings(config)
	out := map[string]any{
		"count":    m.Count,
		"root":     dumpNode(m.Root, icons),
		"icons":    []any{icons.ShowIcons, icons.DefaultIconPack},
		"settings": []any{yaml.Encode(s.paddingX, false), yaml.Encode(s.paddingY, false), yaml.Encode(s.rowIndent, false), yaml.Encode(s.lineThickness, false)},
	}
	for k, v := range map[string]string{"title": m.Title, "accTitle": m.AccTitle, "accDescr": m.AccDescr} {
		if v != "" {
			out[k] = v
		}
	}
	return out
}

func dumpTokens(src string) [][2]string {
	toks, err := tokenize(src)
	if err != nil {
		return nil
	}
	out := [][2]string{}
	for i, k := range toks.Kinds[:len(toks.Kinds)-1] {
		out = append(out, [2]string{treeViewTokens[k].Name, src[toks.Starts[i]:toks.Ends[i]]})
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

func loadCases(t *testing.T) []peleCase {
	t.Helper()
	data, err := fixture.Read("testdata/pele/treeview.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return p.Cases
}

func TestParseMatchesPele(t *testing.T) {
	cases := loadCases(t)
	if len(cases) < 5000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed, models, refused, boxed := 0, 0, 0, 0
	for _, c := range cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Direct.Tokens)
		tree, err := parseTreeView(c.Src)
		switch {
		case err != nil && c.Direct.Error == nil:
			t.Errorf("parseTreeView(%q): %v, Pele had none", c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "parseTreeView error", c.Src, dumpError(err), c.Direct.Error) && ok
		case c.Direct.Error != nil:
			t.Errorf("parseTreeView(%q): no error, Pele %v", c.Src, c.Direct.Error)
			ok = false
		default:
			ok = compare(t, "tree", c.Src, dumpAST(tree), c.Direct.AST) && ok
		}

		text, lineMap, err := preprocessBoxDrawing(c.Src)
		switch {
		case err != nil && c.Box.Error == nil:
			t.Errorf("preprocessBoxDrawing(%q): %v, Pele had none", c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "preprocessBoxDrawing error", c.Src, dumpError(err), c.Box.Error) && ok
		case c.Box.Error != nil:
			t.Errorf("preprocessBoxDrawing(%q): no error, Pele %v", c.Src, c.Box.Error)
			ok = false
		default:
			pairs := [][2]int{}
			for k, line := range lineMap {
				pairs = append(pairs, [2]int{k + 1, line})
			}
			if len(pairs) > 0 {
				boxed++
			}
			ok = compare(t, "preprocessBoxDrawing", c.Src, map[string]any{"text": text, "lineMap": pairs},
				map[string]any{"text": c.Box.Text, "lineMap": c.Box.LineMap}) && ok
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
		case p.Type == diagram.TreeView:
			m, err := ParsePrepared(p)
			switch {
			case err != nil && c.Error == nil:
				t.Errorf("ParsePrepared(%q): %v, Pele had none", c.Src, err)
				ok = false
			case err != nil:
				refused++
				ok = compare(t, "ParsePrepared error", c.Src, dumpError(err), c.Error) && ok
			case c.Error != nil:
				t.Errorf("ParsePrepared(%q): no error, Pele %v", c.Src, c.Error)
				ok = false
			default:
				models++
				ok = compare(t, "model", c.Src, dumpModel(m, p.Config), c.Model) && ok
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	if models < 3000 || refused < 1000 || boxed < 1000 {
		t.Errorf("only %d models, %d errors and %d trees drawn with boxes", models, refused, boxed)
	}
}
