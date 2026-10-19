package class

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
	Flat   any             `json:"flat"`
	Error  map[string]any  `json:"error"`
	Direct *peleCaseResult `json:"direct"`
}

type peleCaseResult struct {
	DB    any            `json:"db"`
	Graph any            `json:"graph"`
	Flat  any            `json:"flat"`
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

func dumpMember(m *Member) object {
	return object{"id": m.ID, "memberType": m.MemberType, "visibility": m.Visibility, "classifier": m.Classifier,
		"parameters": m.Parameters, "returnType": m.ReturnType, "display": m.DisplayText()}
}

func members(list []*Member) []any {
	out := []any{}
	for _, m := range list {
		out = append(out, dumpMember(m))
	}
	return out
}

func dumpEnd(e End) any {
	if e == NoEnd {
		return "none"
	}
	return int(e)
}

func dumpDB(db *DB) object {
	o := object{"direction": db.Direction}
	o.opt("title", db.Title)
	o.opt("accTitle", db.AccTitle)
	o.opt("accDescr", db.AccDescr)
	classes := []any{}
	for _, c := range db.Classes {
		v := object{"id": c.ID, "type": c.Type, "label": c.Label, "cssClasses": c.CSSClasses,
			"methods": members(c.Methods), "members": members(c.Members), "annotations": strs(c.Annotations),
			"styles": strs(c.Styles)}
		v.opt("parent", c.Parent)
		v.opt("link", c.Link)
		v.opt("linkTarget", c.LinkTarget)
		if c.HaveCallback {
			v["haveCallback"] = true
		}
		v.opt("tooltip", c.Tooltip)
		classes = append(classes, v)
	}
	o["classes"] = classes
	relations := []any{}
	for _, r := range db.Relations {
		v := object{"id1": r.ID1, "id2": r.ID2, "relationTitle1": r.RelationTitle1, "relationTitle2": r.RelationTitle2,
			"type1": dumpEnd(r.Type1), "type2": dumpEnd(r.Type2), "lineType": r.LineType}
		if r.HasTitle {
			v["title"] = r.Title
		}
		relations = append(relations, v)
	}
	o["relations"] = relations
	notes := []any{}
	for _, n := range db.Notes {
		v := object{"id": n.ID, "text": n.Text, "index": n.Index}
		v.opt("class", n.Class)
		v.opt("parent", n.Parent)
		notes = append(notes, v)
	}
	o["notes"] = notes
	namespaces := []any{}
	for _, ns := range db.Namespaces {
		classes, notes, children := []any{}, []any{}, []any{}
		for _, c := range ns.Classes {
			classes = append(classes, c.ID)
		}
		for _, n := range ns.Notes {
			notes = append(notes, n.ID)
		}
		for _, c := range ns.Children {
			children = append(children, c.ID)
		}
		v := object{"id": ns.ID, "label": ns.Label, "classes": classes, "notes": notes, "children": children,
			"explicit": ns.Explicit}
		v.opt("parent", ns.Parent)
		namespaces = append(namespaces, v)
	}
	o["namespaces"] = namespaces
	interfaces := []any{}
	for _, x := range db.Interfaces {
		interfaces = append(interfaces, object{"id": x.ID, "label": x.Label, "classId": x.ClassID})
	}
	o["interfaces"] = interfaces
	styleClasses := []any{}
	for _, s := range db.StyleClasses {
		styleClasses = append(styleClasses, object{"id": s.ID, "styles": strs(s.Styles), "textStyles": strs(s.TextStyles)})
	}
	o["styleClasses"] = styleClasses
	return o
}

func dumpGraph(g *Graph) object {
	nodes := []any{}
	for _, n := range g.Nodes {
		v := object{"id": n.ID, "label": n.Label, "isGroup": n.IsGroup, "shape": n.Shape, "parent": n.Parent}
		v.opt("parentId", n.ParentID)
		if n.Class != nil {
			v["classId"] = n.Class.ID
		}
		if n.ColorIndex >= 0 {
			v["colorIndex"] = n.ColorIndex
		}
		nodes = append(nodes, v)
	}
	edges := []any{}
	for _, e := range g.Edges {
		v := object{"id": e.ID, "start": e.Start, "end": e.End, "source": e.Source, "target": e.Target,
			"arrowTypeStart": e.ArrowTypeStart, "arrowTypeEnd": e.ArrowTypeEnd, "startLabelRight": e.StartLabelRight,
			"endLabelLeft": e.EndLabelLeft, "pattern": e.Pattern, "classes": e.Classes}
		if e.HasLabel {
			v["label"] = e.Label
		}
		edges = append(edges, v)
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

type peleFixture struct {
	Diagrams []peleCase `json:"diagrams"`
	Members  []struct {
		Input  string `json:"input"`
		Kind   string `json:"kind"`
		Member any    `json:"member"`
	} `json:"members"`
}

func loadFixture(t *testing.T) peleFixture {
	t.Helper()
	var p peleFixture
	if err := fixture.Load("testdata/pele/class.json", &p); err != nil {
		t.Fatal(err)
	}
	return p
}

// dumpParsed is what the oracle writes of a DB: it, its graph, and its graph with namespaces
// not nested where it has any.
func checkParsed(t *testing.T, what, src string, db *DB, hierarchical bool, wantDB, wantGraph, wantFlat any) bool {
	t.Helper()
	if !compare(t, what+" db", src, dumpDB(db), wantDB) ||
		!compare(t, what+" graph", src, dumpGraph(BuildGraph(db, hierarchical)), wantGraph) {
		return false
	}
	var flat any
	if hierarchical && len(db.Namespaces) > 0 {
		flat = dumpGraph(BuildGraph(db, false))
	}
	return compare(t, what+" flat graph", src, flat, wantFlat)
}

func TestParseMatchesPele(t *testing.T) {
	p := loadFixture(t)
	if len(p.Diagrams) < 5000 {
		t.Fatalf("only %d cases", len(p.Diagrams))
	}
	failed := 0
	for _, c := range p.Diagrams {
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
		case typ != diagram.Class:
			db := NewDB()
			if err := ParseInto(c.Src, db); err != nil {
				if c.Direct.Error == nil {
					t.Errorf("ParseInto(%q): %v, Pele had none", c.Src, err)
					ok = false
				} else {
					ok = compare(t, "direct error", c.Src, dumpError(err), c.Direct.Error)
				}
			} else if c.Direct.Error != nil {
				t.Errorf("ParseInto(%q): no error, Pele %v", c.Src, c.Direct.Error)
				ok = false
			} else {
				ok = checkParsed(t, "direct", c.Src, db, true, c.Direct.DB, c.Direct.Graph, c.Direct.Flat)
			}
		default:
			db, err := ParsePrepared(pre)
			if err != nil {
				if c.Error == nil {
					t.Errorf("Parse(%q): %v, Pele had none", c.Src, err)
					ok = false
				} else {
					ok = compare(t, "error", c.Src, dumpError(err), c.Error)
				}
			} else if c.Error != nil {
				t.Errorf("Parse(%q): no error, Pele %v", c.Src, c.Error)
				ok = false
			} else {
				ok = checkParsed(t, "parsed", c.Src, db, Hierarchical(pre.Config), c.DB, c.Graph, c.Flat)
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}

func TestMembersMatchPele(t *testing.T) {
	p := loadFixture(t)
	if len(p.Members) < 1000 {
		t.Fatalf("only %d members", len(p.Members))
	}
	failed := 0
	for _, m := range p.Members {
		if !compare(t, m.Kind, m.Input, dumpMember(newMember(m.Input, m.Kind)), m.Member) {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
