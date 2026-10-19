package c4

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it: the tokens
// of the text given to the lexer as it is, the model parseC4 builds of it, and the model of
// the text as it comes out of Pele's parseSource.

type peleDirect struct {
	Tokens [][2]string    `json:"tokens"`
	Model  map[string]any `json:"model"`
	Error  map[string]any `json:"error"`
}

type peleCase struct {
	Src    string         `json:"src"`
	Type   *string        `json:"type"`
	Direct peleDirect     `json:"direct"`
	Model  map[string]any `json:"model"`
	Error  map[string]any `json:"error"`
}

// enc is a number as the oracle writes it.
func enc(f float64) any { return yaml.Encode(f, false) }

var (
	elementFields = []string{"type", "descr", "techn", "sprite", "tags", "link", "bgColor", "fontColor", "borderColor",
		"shadowing", "shape", "legendText", "legendSprite"}
	relFields = []string{"techn", "descr", "sprite", "tags", "link", "textColor", "lineColor"}
)

func dumpNamed(n *Named) any { return []string{n.Key, n.Value} }

func dumpExtra(list []Named) []any {
	out := []any{}
	for _, n := range list {
		out = append(out, []string{n.Key, n.Value})
	}
	return out
}

// dumpModel writes a model as the oracle writes Pele's: every field that is defined, a parent
// by its place among the boundaries.
func dumpModel(db *DB) map[string]any {
	index := map[*Boundary]int{}
	for i, b := range db.Boundaries {
		index[b] = i
	}
	element := func(e *Element) map[string]any {
		o := map[string]any{"alias": e.Alias, "label": e.Label, "parent": nil}
		if e.Parent != nil {
			o["parent"] = index[e.Parent]
		}
		if e.AliasAttr != nil {
			o["aliasAttr"] = dumpNamed(e.AliasAttr)
		}
		if e.LabelAttr != nil {
			o["labelAttr"] = dumpNamed(e.LabelAttr)
		}
		for _, f := range elementFields {
			if e.Defined(f) {
				p, _ := e.field(f)
				o[f] = *p
			}
		}
		if e.Extra != nil {
			o["extra"] = dumpExtra(e.Extra)
		}
		return o
	}
	shapes := []any{}
	for _, s := range db.Shapes {
		o := element(&s.Element)
		o["kind"] = s.Kind
		shapes = append(shapes, o)
	}
	boundaries := []any{}
	for _, b := range db.Boundaries {
		o := element(&b.Element)
		if b.NodeType != "" {
			o["nodeType"] = b.NodeType
		}
		boundaries = append(boundaries, o)
	}
	rels := []any{}
	for _, r := range db.Rels {
		o := map[string]any{"type": r.Type, "from": r.From, "to": r.To, "label": r.Label}
		if r.FromAttr != nil {
			o["fromAttr"] = dumpNamed(r.FromAttr)
		}
		if r.ToAttr != nil {
			o["toAttr"] = dumpNamed(r.ToAttr)
		}
		if r.LabelAttr != nil {
			o["labelAttr"] = dumpNamed(r.LabelAttr)
		}
		for _, f := range relFields {
			if r.Defined(f) {
				p, _ := r.field(f)
				o[f] = *p
			}
		}
		if r.Defined("offsetX") {
			o["offsetX"] = enc(r.OffsetX)
		}
		if r.Defined("offsetY") {
			o["offsetY"] = enc(r.OffsetY)
		}
		if r.Extra != nil {
			o["extra"] = dumpExtra(r.Extra)
		}
		rels = append(rels, o)
	}
	out := map[string]any{
		"c4Type": db.C4Type, "shapes": shapes, "boundaries": boundaries, "rels": rels,
		"shapeInRow": enc(db.ShapeInRow), "boundaryInRow": enc(db.BoundaryInRow),
	}
	if db.HasTitle {
		out["title"] = db.Title
	}
	if db.HasAccDescr {
		out["accDescr"] = db.AccDescr
	}
	return out
}

func dumpTokens(src string) [][2]string {
	t := tokenize(src)
	out := [][2]string{}
	for i, k := range t.types {
		out = append(out, [2]string{tokenNames[k], src[t.starts[i]:t.ends[i]]})
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
	if err := fixture.Load("testdata/pele/c4.json", &p); err != nil {
		t.Fatal(err)
	}
	return p.Cases
}

func TestParseMatchesPele(t *testing.T) {
	cases := loadCases(t)
	if len(cases) < 5000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed := 0
	for _, c := range cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Direct.Tokens)
		db := newDB()
		err := parseC4(c.Src, db)
		switch {
		case err != nil && c.Direct.Error == nil:
			t.Errorf("parseC4(%q): %v, Pele had none", c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "parseC4 error", c.Src, dumpError(err), c.Direct.Error) && ok
		case c.Direct.Error != nil:
			t.Errorf("parseC4(%q): no error, Pele %v", c.Src, c.Direct.Error)
			ok = false
		default:
			ok = compare(t, "direct model", c.Src, dumpModel(db), c.Direct.Model) && ok
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
		case p.Type == diagram.C4:
			db, err := ParsePrepared(p)
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
				ok = compare(t, "model", c.Src, dumpModel(db), c.Model) && ok
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
