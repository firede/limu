package block

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
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
	Src    string
	Type   *string
	Tokens []any
	Direct peleResult
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Src    string         `json:"src"`
		Type   *string        `json:"type"`
		Tokens []any          `json:"tokens"`
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

var undefined = map[string]any{"$": "undefined"}

// num is an optional number as the oracle writes it.
func num(f *float64) any {
	if f == nil {
		return undefined
	}
	return yaml.Encode(*f, false)
}

// list is an optional list of strings as the oracle writes it.
func list(s []string) any {
	if s == nil {
		return undefined
	}
	return append([]string{}, s...)
}

func dumpTokens(src string) []any {
	t := tokenize(src)
	out := []any{}
	for i, typ := range t.Types {
		out = append(out, []any{typ, t.Texts[i], chars.Len16(src[:t.Starts[i]])})
	}
	return out
}

func flat(b *Block) []any {
	var id, label, color any = undefined, undefined, undefined
	if b.ID != "" {
		id = b.ID
	}
	if b.HasLabel {
		label = b.Label
	}
	if b.ColorIndex != nil {
		color = *b.ColorIndex
	}
	return []any{id, b.Type, label, num(b.Columns), num(b.WidthInColumns), num(b.Width), list(b.Directions),
		list(b.Classes), list(b.Styles), color}
}

func tree(b *Block) []any {
	var children any = undefined
	if b.Children != nil {
		kids := []any{}
		for _, c := range b.Children {
			kids = append(kids, tree(c))
		}
		children = kids
	}
	return append(flat(b), children)
}

func dumpModel(db *DB) map[string]any {
	o := map[string]any{"root": tree(db.Root)}
	if db.Title != "" {
		o["title"] = db.Title
	}
	blocks := []any{}
	for _, b := range db.Blocks {
		blocks = append(blocks, b.ID)
	}
	o["blocks"] = blocks
	edges := []any{}
	for _, e := range db.Edges {
		var label any = undefined
		if e.HasLabel {
			label = e.Label
		}
		edges = append(edges, []any{e.ID, e.Start, e.End, label, e.Type, e.Thickness, e.Pattern, list(e.Directions), e.ArrowTypeEnd, e.ArrowTypeStart})
	}
	o["edges"] = edges
	classes := []any{}
	for _, k := range db.ClassOrder {
		c := db.Classes[k]
		classes = append(classes, []any{k, c.ID, c.Styles, c.TextStyles})
	}
	o["classes"] = classes
	byID := []any{}
	for _, k := range db.IDOrder {
		b := db.ByID[k]
		var children any = undefined
		if b.Children != nil {
			ids := []any{}
			for _, c := range b.Children {
				ids = append(ids, c.ID)
			}
			children = ids
		}
		byID = append(byID, append(append([]any{k}, flat(b)...), children))
	}
	o["byId"] = byID
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
	if len(gj) > 3000 {
		gj = gj[:3000]
	}
	if len(wj) > 3000 {
		wj = wj[:3000]
	}
	t.Errorf("%s of %q\n got %s\nPele %s", what, src, gj, wj)
	return false
}

// check compares a parse with Pele's result.
func check(t *testing.T, what, src string, db *DB, err error, want peleResult) bool {
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
	return compare(t, what+" model", src, dumpModel(db), want.Model)
}

func TestParseMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/block.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 7000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Tokens)
		if ok {
			db := NewDB()
			err := parseBlock(c.Src, db)
			if err != nil {
				db = nil
			}
			ok = check(t, "parseBlock", c.Src, db, err, c.Direct)
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
			case typ == diagram.Block:
				db, err := ParsePrepared(p)
				ok = check(t, "ParsePrepared", c.Src, db, err, c.peleResult)
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
