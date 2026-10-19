package state

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The tokens,
// the DB and the graph are written as the oracle writes Pele's, field by field, so that any
// field that differs shows.

type peleCase struct {
	Src      string         `json:"src"`
	Type     *string        `json:"type"`
	Tokens   []any          `json:"tokens"`
	LexError *string        `json:"lexError"`
	DB       any            `json:"db"`
	Error    map[string]any `json:"error"`
	Direct   *peleCase      `json:"direct"`
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

func dumpTokens(src string) ([]any, *string) {
	lex := tokenize(src)
	out := make([]any, len(lex.toks))
	for i, t := range lex.toks {
		out[i] = []any{tokenNames[t.typ], t.text, chars.Len16(src[:t.start])}
	}
	if lex.hasError {
		return out, &lex.error
	}
	return out, nil
}

func dumpNote(n *Note) object {
	return object{"position": n.Position, "text": n.Text}
}

func dumpStmt(s Stmt) any {
	switch s := s.(type) {
	case TextStmt:
		return string(s)
	case *StateStmt:
		o := object{"stmt": "state", "id": s.ID}
		o.opt("type", s.Type)
		switch {
		case s.Description == nil:
		case s.DescriptionList:
			o["description"] = strs(s.Description)
		default:
			o["description"] = s.Description[0]
		}
		if s.Doc != nil {
			doc := make([]any, len(s.Doc))
			for i, item := range s.Doc {
				doc[i] = dumpStmt(item)
			}
			o["doc"] = doc
		}
		if s.Note != nil {
			o["note"] = dumpNote(s.Note)
		}
		if s.Start != nil {
			o["start"] = *s.Start
		}
		if s.Classes != nil {
			o["classes"] = strs(s.Classes)
		}
		return o
	case *RelationStmt:
		o := object{"stmt": "relation", "state1": dumpStmt(s.State1), "state2": dumpStmt(s.State2)}
		if s.HasDescription {
			o["description"] = s.Description
		}
		return o
	case *ClassDefStmt:
		return object{"stmt": "classDef", "id": s.ID, "classes": s.Classes}
	case *StyleStmt:
		return object{"stmt": "style", "id": s.ID, "styleClass": s.StyleClass}
	case *ApplyClassStmt:
		return object{"stmt": "applyClass", "id": s.ID, "styleClass": s.StyleClass}
	case *DirectionStmt:
		return object{"stmt": "dir", "value": s.Value}
	case *ClickStmt:
		return object{"stmt": "click", "id": dumpStmt(s.ID), "url": s.URL, "tooltip": s.Tooltip}
	}
	return "unknown statement"
}

func dumpDB(db *DB) object {
	o := object{"direction": db.Direction()}
	o.opt("title", db.Title)
	o.opt("accTitle", db.AccTitle)
	o.opt("accDescr", db.AccDescr)
	root := []any{}
	for _, s := range db.RootDoc {
		root = append(root, dumpStmt(s))
	}
	o["rootDoc"] = root
	states := []any{}
	for _, s := range db.States {
		v := object{"id": s.ID, "type": s.Type, "descriptions": strs(s.Descriptions), "classes": strs(s.Classes),
			"styles": strs(s.Styles), "textStyles": strs(s.TextStyles)}
		if s.Doc != nil {
			v["doc"] = len(s.Doc)
		}
		if s.Note != nil {
			v["note"] = dumpNote(s.Note)
		}
		states = append(states, v)
	}
	o["states"] = states
	relations := []any{}
	for _, r := range db.Relations {
		relations = append(relations, object{"id1": r.ID1, "id2": r.ID2, "relationTitle": r.RelationTitle})
	}
	o["relations"] = relations
	classes := []any{}
	for _, c := range db.Classes {
		classes = append(classes, object{"id": c.ID, "styles": strs(c.Styles), "textStyles": strs(c.TextStyles)})
	}
	o["classes"] = classes
	links := []any{}
	for _, l := range db.Links {
		links = append(links, []any{l.ID, l.URL, l.Tooltip})
	}
	o["links"] = links
	nodes := []any{}
	for _, n := range db.Graph.Nodes {
		v := object{"id": n.ID, "label": n.Label, "description": strs(n.Description), "cssClasses": n.CSSClasses,
			"cssCompiledStyles": strs(n.CSSCompiledStyles), "cssStyles": strs(n.CSSStyles), "isGroup": n.IsGroup}
		v.opt("shape", n.Shape)
		v.opt("type", n.Type)
		if n.HasParent {
			v["parentId"] = n.Parent
		}
		v.opt("dir", n.Dir)
		v.opt("position", n.Position)
		v.opt("noteFor", n.NoteFor)
		if n.ColorIndex >= 0 {
			v["colorIndex"] = n.ColorIndex
		}
		nodes = append(nodes, v)
	}
	edges := []any{}
	for _, e := range db.Graph.Edges {
		v := object{"id": e.ID, "start": e.Start, "end": e.End, "label": e.Label, "classes": e.Classes}
		v.opt("pattern", e.Pattern)
		edges = append(edges, v)
	}
	o["graph"] = object{"nodes": nodes, "edges": edges}
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
	t.Errorf("%s of %q\n got %s\nPele %s", what, src, gj, wj)
	return false
}

func loadCases(t *testing.T) []peleCase {
	t.Helper()
	data, err := fixture.Read("testdata/pele/state.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		States []peleCase `json:"states"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return p.States
}

// check parses the source as the parser of the type reads it, and compares the tokens, the DB
// and the graph, or the error, with what Pele made of it.
func check(t *testing.T, label, src, parsed string, title string, c *peleCase) bool {
	t.Helper()
	toks, lexError := dumpTokens(parsed)
	if !compare(t, label+"tokens", src, toks, c.Tokens) || !compare(t, label+"lexical error", src, lexError, c.LexError) {
		return false
	}
	db, err := Parse(parsed, title)
	switch {
	case err != nil && c.Error == nil:
		t.Errorf("%sparse of %q: %v, Pele had none", label, src, err)
		return false
	case err != nil:
		return compare(t, label+"error", src, dumpError(err), c.Error)
	case c.Error != nil:
		t.Errorf("%sparse of %q: no error, Pele %v", label, src, c.Error)
		return false
	}
	return compare(t, label+"db", src, dumpDB(db), c.DB)
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
		case typ != diagram.State:
			ok = check(t, "direct ", c.Src, c.Src, "", c.Direct)
		default:
			ok = check(t, "", c.Src, p.Source(false), p.Title, &c)
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}

// Deep nesting is read without recursion, as Pele reads it.
func TestDeep(t *testing.T) {
	src := "stateDiagram-v2\n" + strings.Repeat("state a {\n", 5000) + "x\n" + strings.Repeat("}\n", 5000)
	db, err := Parse(src, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Graph.Nodes) != 2 {
		t.Errorf("%d nodes", len(db.Graph.Nodes))
	}
}
