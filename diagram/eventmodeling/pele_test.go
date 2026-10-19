package eventmodeling

import (
	"encoding/json"
	"reflect"
	"slices"
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

// opt is a string Pele may leave undefined, as the oracle writes it.
func opt(s string, has bool) any {
	if !has {
		return undefined
	}
	return s
}

// list is a list of strings, never null.
func list(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func dumpAST(t *ast) map[string]any {
	frames := []any{}
	for _, f := range t.frames {
		typ := "EmTimeFrame"
		if f.reset {
			typ = "EmResetFrame"
		}
		frames = append(frames, []any{typ, f.name, f.modelEntityType, f.entityIdentifier, list(f.sourceFrames),
			opt(f.dataReference, f.hasDataReference), opt(f.dataType, f.dataType != ""), opt(f.dataInlineValue, f.hasInline)})
	}
	data := []any{}
	for _, d := range t.dataEntities {
		data = append(data, []any{d.name, opt(d.dataType, d.dataType != ""), d.dataBlockValue})
	}
	notes := []any{}
	for _, n := range t.noteEntities {
		notes = append(notes, []any{n.sourceFrame, opt(n.dataType, n.dataType != ""), n.dataBlockValue})
	}
	gwts := []any{}
	for _, g := range t.gwtEntities {
		gwts = append(gwts, []any{g.sourceFrame, list(g.given), list(g.when), list(g.then)})
	}
	return map[string]any{
		"title":    opt(t.title, t.hasTitle),
		"accTitle": opt(t.accTitle, t.hasAccTitle),
		"accDescr": opt(t.accDescr, t.hasAccDescr),
		"entities": list(t.modelEntities),
		"frames":   frames,
		"data":     data,
		"notes":    notes,
		"gwts":     gwts,
	}
}

func dumpModel(m *Model) map[string]any {
	lanes := []any{}
	for _, l := range m.Lanes {
		lanes = append(lanes, []any{l.Index, l.Label, string(l.Group), opt(l.Namespace, l.Namespace != "")})
	}
	boxes := []any{}
	for _, b := range m.Boxes {
		boxes = append(boxes, []any{b.ID, string(b.Kind), b.Entity, b.Name, opt(b.Namespace, b.Namespace != ""), b.Reset,
			slices.Index(m.Lanes, b.Lane), opt(b.Data, b.HasData), opt(b.DataType, b.DataType != "")})
	}
	relations := []any{}
	for _, r := range m.Relations {
		relations = append(relations, []string{r.Source.ID, r.Target.ID})
	}
	notes := []any{}
	for _, n := range m.Notes {
		notes = append(notes, []any{n.Frame, n.Text, opt(n.DataType, n.DataType != "")})
	}
	specs := []any{}
	for _, s := range m.Specifications {
		specs = append(specs, []any{s.Frame, list(s.Given), list(s.When), list(s.Then)})
	}
	out := map[string]any{"lanes": lanes, "boxes": boxes, "relations": relations, "notes": notes, "specifications": specs,
		"entities": list(m.Entities)}
	for k, v := range map[string]string{"title": m.Title, "accTitle": m.AccTitle, "accDescr": m.AccDescr} {
		if v != "" {
			out[k] = v
		}
	}
	return out
}

func dumpTokens(src string) [][2]string {
	toks, err := common.Tokenize(src, eventModelingTable, "eventmodeling")
	if err != nil {
		return nil
	}
	out := [][2]string{}
	for i, k := range toks.Kinds[:len(toks.Kinds)-1] {
		out = append(out, [2]string{eventModelingTokens[k].Name, src[toks.Starts[i]:toks.Ends[i]]})
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
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/eventmodeling.json", &p); err != nil {
		t.Fatal(err)
	}
	return p.Cases
}

func TestParseMatchesPele(t *testing.T) {
	cases := loadCases(t)
	if len(cases) < 7000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed, models, refused := 0, 0, 0
	for _, c := range cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Direct.Tokens)
		tree, err := parseEventModel(c.Src)
		switch {
		case err != nil && c.Direct.Error == nil:
			t.Errorf("parseEventModel(%q): %v, Pele had none", c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "parseEventModel error", c.Src, dumpError(err), c.Direct.Error) && ok
		case c.Direct.Error != nil:
			t.Errorf("parseEventModel(%q): no error, Pele %v", c.Src, c.Direct.Error)
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
		case p.Type == diagram.EventModeling:
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
				ok = compare(t, "model", c.Src, dumpModel(m), c.Model) && ok
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	if models < 2000 || refused < 500 {
		t.Errorf("only %d models and %d errors", models, refused)
	}
}
