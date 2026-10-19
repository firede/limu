package requirement

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The
// tokens, the DB and the graph are written as the oracle writes Pele's, field by field, so
// that any field that differs shows.

type peleCase struct {
	Src    string          `json:"src"`
	Type   *string         `json:"type"`
	Tokens any             `json:"tokens"`
	DB     any             `json:"db"`
	Graph  any             `json:"graph"`
	Error  map[string]any  `json:"error"`
	Direct *peleCaseResult `json:"direct"`
}

type peleCaseResult struct {
	Tokens any            `json:"tokens"`
	DB     any            `json:"db"`
	Graph  any            `json:"graph"`
	Error  map[string]any `json:"error"`
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

func dumpTokens(src string) []any {
	out := []any{}
	for _, t := range tokenize(src) {
		out = append(out, []any{tokenNames[t.typ], t.text, chars.Len16(src[:t.start])})
	}
	return out
}

func dumpDB(db *DB) object {
	o := object{"direction": db.Direction}
	o.opt("title", db.Title)
	o.opt("accTitle", db.AccTitle)
	o.opt("accDescr", db.AccDescr)
	reqs := []any{}
	for _, r := range db.Requirements {
		reqs = append(reqs, []any{r.Name, object{"name": r.Name, "type": r.Type, "requirementId": r.RequirementID,
			"text": r.Text, "risk": r.Risk, "verifyMethod": r.VerifyMethod, "cssStyles": strs(r.CSSStyles),
			"classes": strs(r.Classes)}})
	}
	o["requirements"] = reqs
	elems := []any{}
	for _, e := range db.Elements {
		elems = append(elems, []any{e.Name, object{"name": e.Name, "type": e.Type, "docRef": e.DocRef,
			"cssStyles": strs(e.CSSStyles), "classes": strs(e.Classes)}})
	}
	o["elements"] = elems
	rels := []any{}
	for _, r := range db.Relations {
		rels = append(rels, object{"type": r.Type, "src": r.Src, "dst": r.Dst})
	}
	o["relations"] = rels
	classes := []any{}
	for _, c := range db.Classes {
		classes = append(classes, []any{c.ID, object{"id": c.ID, "styles": strs(c.Styles), "textStyles": strs(c.TextStyles)}})
	}
	o["classes"] = classes
	return o
}

func dumpGraph(g *Graph) object {
	nodes := []any{}
	for _, n := range g.Nodes {
		v := object{"id": n.ID, "cssStyles": strs(n.CSSStyles), "cssClasses": n.CSSClasses}
		if n.Requirement != nil {
			v["requirement"] = n.Requirement.Name
		}
		if n.Element != nil {
			v["element"] = n.Element.Name
		}
		nodes = append(nodes, v)
	}
	edges := []any{}
	for _, e := range g.Edges {
		v := object{"id": e.ID, "type": e.Type, "contains": e.Contains}
		if e.HasStart {
			v["start"] = e.Start
		}
		if e.HasEnd {
			v["end"] = e.End
		}
		edges = append(edges, v)
	}
	return object{"direction": g.Direction, "nodes": nodes, "edges": edges}
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
	t.Errorf("%s of %q\n got %s\nPele %s", what, short(src), gj, wj)
	return false
}

// result compares parsing src with what Pele made of it.
func result(t *testing.T, src string, tokens, wantDB, wantGraph any, wantErr map[string]any, parse func() (*DB, error)) bool {
	t.Helper()
	ok := compare(t, "tokens", src, dumpTokens(src), tokens)
	db, err := parse()
	switch {
	case err != nil && wantErr == nil:
		t.Errorf("parse of %q: %v, Pele had none", src, err)
		return false
	case err != nil:
		return compare(t, "error", src, dumpError(err), wantErr) && ok
	case wantErr != nil:
		t.Errorf("parse of %q: no error, Pele %v", src, wantErr)
		return false
	}
	return compare(t, "db", src, dumpDB(db), wantDB) && compare(t, "graph", src, dumpGraph(BuildGraph(db)), wantGraph) && ok
}

func loadCases(t *testing.T) []peleCase {
	t.Helper()
	data, err := fixture.Read("testdata/pele/requirement.json")
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
	if len(cases) < 3000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed := 0
	for _, c := range cases {
		ok := true
		p, err := diagram.Prepare(c.Src, diagram.Options{})
		var typ diagram.Type
		if p != nil {
			typ = p.Type
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
		case typ != diagram.Requirement:
			d := c.Direct
			ok = result(t, c.Src, d.Tokens, d.DB, d.Graph, d.Error, func() (*DB, error) {
				db := NewDB()
				return db, ParseInto(c.Src, db)
			})
		default:
			ok = result(t, p.Source(false), c.Tokens, c.DB, c.Graph, c.Error, func() (*DB, error) { return ParsePrepared(p) })
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}

// short is a source cut short for a message.
func short(s string) string {
	if len(s) > 300 {
		return s[:300] + "…"
	}
	return s
}
