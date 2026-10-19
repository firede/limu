package cynefin

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

func encStr(s string, has bool) any {
	if !has {
		return undefined
	}
	return s
}

// encSet is a string Pele leaves undefined when it is empty.
func encSet(s string) any { return encStr(s, s != "") }

func dumpAST(t *ast) map[string]any {
	domains := []any{}
	for _, d := range t.domains {
		domains = append(domains, []any{d.domain, append([]string{}, d.items...)})
	}
	transitions := []any{}
	for _, tr := range t.transitions {
		transitions = append(transitions, []any{tr.from, tr.to, encStr(tr.label, tr.hasLabel)})
	}
	return map[string]any{
		"title":       encStr(t.title, t.hasTitle),
		"accTitle":    encStr(t.accTitle, t.hasAccTitle),
		"accDescr":    encStr(t.accDescr, t.hasAccDescr),
		"domains":     domains,
		"transitions": transitions,
	}
}

func dumpModel(m *Model) map[string]any {
	domains := []any{}
	for _, d := range m.Domains {
		domains = append(domains, []any{d.Name, d.Name, append([]string{}, d.Items...)})
	}
	transitions := []any{}
	for _, t := range m.Transitions {
		transitions = append(transitions, []any{t.From, t.To, encSet(t.Label)})
	}
	return map[string]any{
		"title":       encSet(m.Title),
		"accTitle":    encSet(m.AccTitle),
		"accDescr":    encSet(m.AccDescr),
		"domains":     domains,
		"transitions": transitions,
	}
}

func dumpTokens(src string) [][2]string {
	toks, err := common.Tokenize(src, cynefinTable, "cynefin")
	if err != nil {
		return nil
	}
	out := [][2]string{}
	for i, k := range toks.Kinds[:len(toks.Kinds)-1] {
		out = append(out, [2]string{cynefinTokens[k].Name, src[toks.Starts[i]:toks.Ends[i]]})
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
	data, err := fixture.Read("testdata/pele/cynefin.json")
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
	failed, models, errors := 0, 0, 0
	for _, c := range cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Direct.Tokens)
		tree, err := parseCynefin(c.Src)
		switch {
		case err != nil && c.Direct.Error == nil:
			t.Errorf("parseCynefin(%q): %v, Pele had none", c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "parseCynefin error", c.Src, dumpError(err), c.Direct.Error) && ok
		case c.Direct.Error != nil:
			t.Errorf("parseCynefin(%q): no error, Pele %v", c.Src, c.Direct.Error)
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
		case p.Type == diagram.Cynefin:
			m, err := ParsePrepared(p)
			switch {
			case err != nil && c.Error == nil:
				t.Errorf("ParsePrepared(%q): %v, Pele had none", c.Src, err)
				ok = false
			case err != nil:
				errors++
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
	if models < 2000 || errors < 2000 {
		t.Errorf("only %d models and %d errors", models, errors)
	}
}
