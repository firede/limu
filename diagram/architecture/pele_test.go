package architecture

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it: the tokens
// and the tree of the text given to the parser as it is, and the model of the text as it
// comes out of Pele's parseSource. A string Pele leaves undefined or empty is null in it.

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

// opt is a string as the fixture writes it: null for none.
func opt(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func dumpEdge(lhs string, lg bool, ld Side, li bool, title string, ri bool, rd Side, rhs string, rg bool) []any {
	return []any{lhs, lg, string(ld), li, opt(title), ri, string(rd), rhs, rg}
}

func dumpAST(t *ast) map[string]any {
	groups, services, junctions, edges, alignments := []any{}, []any{}, []any{}, []any{}, []any{}
	for _, g := range t.groups {
		groups = append(groups, []any{g.id, opt(g.icon), opt(g.title), opt(g.in)})
	}
	for _, s := range t.services {
		services = append(services, []any{s.id, opt(s.icon), opt(s.iconText), opt(s.title), opt(s.in)})
	}
	for _, j := range t.junctions {
		junctions = append(junctions, []any{j.id, opt(j.in)})
	}
	for _, e := range t.edges {
		edges = append(edges, dumpEdge(e.lhsID, e.lhsGroup, e.lhsDir, e.lhsInto, e.title, e.rhsInto, e.rhsDir, e.rhsID, e.rhsGroup))
	}
	for _, a := range t.alignments {
		alignments = append(alignments, []any{a.direction, a.members})
	}
	return map[string]any{
		"title": opt(t.title), "accTitle": opt(t.accTitle), "accDescr": opt(t.accDescr),
		"groups": groups, "services": services, "junctions": junctions, "edges": edges, "alignments": alignments,
	}
}

func dumpModel(m *Model) map[string]any {
	nodes, groups, edges, hints := []any{}, []any{}, []any{}, []any{}
	for _, n := range m.Nodes {
		if n.Junction {
			nodes = append(nodes, []any{n.ID, "junction", opt(n.In)})
		} else {
			nodes = append(nodes, []any{n.ID, "service", opt(n.Icon), opt(n.IconText), opt(n.Title), opt(n.In)})
		}
	}
	for _, g := range m.Groups {
		groups = append(groups, []any{g.ID, opt(g.Icon), opt(g.Title), opt(g.In)})
	}
	for _, e := range m.Edges {
		edges = append(edges, dumpEdge(e.LhsID, e.LhsGroup, e.LhsDir, e.LhsInto, e.Title, e.RhsInto, e.RhsDir, e.RhsID, e.RhsGroup))
	}
	for _, h := range m.LayoutHints {
		hints = append(hints, []any{h.Direction, h.Members})
	}
	return map[string]any{
		"title": opt(m.Title), "accTitle": opt(m.AccTitle), "accDescr": opt(m.AccDescr),
		"nodes": nodes, "groups": groups, "edges": edges, "hints": hints,
	}
}

func dumpTokens(src string) [][2]string {
	toks, err := common.Tokenize(src, archTable, "architecture")
	if err != nil {
		return nil
	}
	out := [][2]string{}
	for i, k := range toks.Kinds[:len(toks.Kinds)-1] {
		out = append(out, [2]string{archTokens[k].Name, src[toks.Starts[i]:toks.Ends[i]]})
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
	data, err := fixture.Read("testdata/pele/architecture.json")
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
	if len(cases) < 6000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed, models := 0, 0
	for _, c := range cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Direct.Tokens)
		tree, err := parseArchitecture(c.Src)
		switch {
		case err != nil && c.Direct.Error == nil:
			t.Errorf("parseArchitecture(%q): %v, Pele had none", c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "parseArchitecture error", c.Src, dumpError(err), c.Direct.Error) && ok
		case c.Direct.Error != nil:
			t.Errorf("parseArchitecture(%q): no error, Pele %v", c.Src, c.Direct.Error)
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
		case p.Type == diagram.Architecture:
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
				models++
				ok = compare(t, "model", c.Src, dumpModel(m), c.Model) && ok
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	if models < 1000 {
		t.Errorf("only %d models", models)
	}
}
