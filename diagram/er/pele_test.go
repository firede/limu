package er

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The DB and
// the graph are written as the oracle writes Pele's, field by field, so that any field that
// differs shows.

type peleCase struct {
	Src    string          `json:"src"`
	Type   *string         `json:"type"`
	DB     any             `json:"db"`
	Graph  any             `json:"graph"`
	Error  map[string]any  `json:"error"`
	Direct *peleCaseResult `json:"direct"`
}

type peleCaseResult struct {
	DB    any            `json:"db"`
	Graph any            `json:"graph"`
	Error map[string]any `json:"error"`
}

// object builds a JSON object, leaving out the fields Pele leaves undefined.
type object map[string]any

func (o object) opt(key, value string) {
	if value != "" {
		o[key] = value
	}
}

func strs(s []string) []any {
	out := make([]any, len(s))
	for i, x := range s {
		out[i] = x
	}
	return out
}

func dumpAttributes(list []Attribute) []any {
	out := []any{}
	for _, a := range list {
		out = append(out, object{"type": a.Type, "name": a.Name, "keys": strs(a.Keys), "comment": a.Comment})
	}
	return out
}

func dumpDB(db *DB) object {
	o := object{"direction": db.Direction}
	o.opt("title", db.Title)
	o.opt("accTitle", db.AccTitle)
	o.opt("accDescr", db.AccDescr)
	entities := []any{}
	for _, e := range db.Entities {
		entities = append(entities, object{"name": e.Label, "id": e.ID, "label": e.Label, "alias": e.Alias,
			"attributes": dumpAttributes(e.Attributes), "cssClasses": e.CSSClasses, "cssStyles": strs(e.CSSStyles)})
	}
	o["entities"] = entities
	relationships := []any{}
	for _, r := range db.Relationships {
		relationships = append(relationships, object{"entityA": r.EntityA, "roleA": r.RoleA, "entityB": r.EntityB,
			"cardA": r.RelSpec.CardA, "relType": r.RelSpec.RelType, "cardB": r.RelSpec.CardB})
	}
	o["relationships"] = relationships
	classes := []any{}
	for _, c := range db.Classes {
		classes = append(classes, object{"id": c.ID, "styles": strs(c.Styles), "textStyles": strs(c.TextStyles)})
	}
	o["classes"] = classes
	subgraphs := []any{}
	for _, s := range db.Subgraphs {
		v := object{"id": s.ID, "nodes": strs(s.Nodes), "title": s.Title, "classes": strs(s.Classes),
			"cssStyles": strs(s.CSSStyles), "labelType": s.LabelType}
		v.opt("dir", s.Dir)
		subgraphs = append(subgraphs, v)
	}
	o["subgraphs"] = subgraphs
	return o
}

func dumpGraph(g *Graph) object {
	nodes := []any{}
	for _, n := range g.Nodes {
		v := object{"id": n.ID, "label": n.Label, "alias": n.Alias, "attributes": dumpAttributes(n.Attributes),
			"isGroup": n.IsGroup, "cssStyles": strs(n.CSSStyles), "cssCompiledStyles": strs(n.CSSCompiledStyles),
			"cssClasses": n.CSSClasses, "labelType": n.LabelType}
		if n.HasParent {
			v["parentId"] = n.Parent
		}
		v.opt("dir", n.Dir)
		nodes = append(nodes, v)
	}
	edges := []any{}
	for _, e := range g.Edges {
		edges = append(edges, object{"id": e.ID, "start": e.Start, "end": e.End, "label": e.Label,
			"arrowTypeStart": e.ArrowTypeStart, "arrowTypeEnd": e.ArrowTypeEnd, "pattern": e.Pattern})
	}
	return object{"nodes": nodes, "edges": edges}
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

// result compares what parsing gave with what Pele's did.
func result(t *testing.T, what, src string, db *DB, err error, want *peleCaseResult) bool {
	t.Helper()
	switch {
	case err != nil && want.Error == nil:
		t.Errorf("%s of %q: %v, Pele had none", what, src, err)
		return false
	case err != nil:
		return compare(t, what+" error", src, dumpError(err), want.Error)
	case want.Error != nil:
		t.Errorf("%s of %q: no error, Pele %v", what, src, want.Error)
		return false
	}
	return compare(t, what+" db", src, dumpDB(db), want.DB) && compare(t, what+" graph", src, dumpGraph(BuildGraph(db)), want.Graph)
}

func TestParseMatchesPele(t *testing.T) {
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/er.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 3000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
		ok := true
		pre, err := diagram.Prepare(c.Src, diagram.Options{})
		var typ diagram.Type
		if pre != nil {
			typ = pre.Type
		} else if e, isPele := err.(*diagram.Error); isPele && e.Code == diagram.UnsupportedDiagram {
			err = nil
		}
		switch {
		case err != nil:
			if c.Error == nil {
				t.Errorf("Prepare(%q): %v, Pele had none", c.Src, err)
				ok = false
			} else {
				ok = compare(t, "error", c.Src, dumpError(err), c.Error)
			}
		case c.Type == nil && typ != "" || c.Type != nil && string(typ) != *c.Type:
			t.Errorf("type of %q: %q, Pele %v", c.Src, typ, c.Type)
			ok = false
		case typ != diagram.ER:
			db := NewDB()
			err := ParseInto(c.Src, db)
			ok = result(t, "direct", c.Src, db, err, c.Direct)
		default:
			db, err := ParsePrepared(pre)
			ok = result(t, "parse", c.Src, db, err, &peleCaseResult{DB: c.DB, Graph: c.Graph, Error: c.Error})
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
