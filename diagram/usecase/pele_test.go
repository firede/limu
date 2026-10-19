package usecase

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The model
// and the graph are written as the oracle writes Pele's, field by field, with offsets in UTF-16
// code units, so that any field that differs shows.

type peleResult struct {
	Model any            `json:"model"`
	Graph any            `json:"graph"`
	Error map[string]any `json:"error"`
}

type peleCase struct {
	Src    string  `json:"src"`
	Type   *string `json:"type"`
	Direct peleResult
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Src    string         `json:"src"`
		Type   *string        `json:"type"`
		Direct peleResult     `json:"direct"`
		Model  any            `json:"model"`
		Graph  any            `json:"graph"`
		Error  map[string]any `json:"error"`
		// Same says the pipeline's result is the direct one.
		Same bool `json:"same"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Src, c.Type, c.Direct = raw.Src, raw.Type, raw.Direct
	c.peleResult = peleResult{raw.Model, raw.Graph, raw.Error}
	if raw.Same {
		c.peleResult = raw.Direct
	}
	return nil
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

// units converts byte offsets of a source to UTF-16 code units.
type units []int

func unitsOf(src string) units {
	u := make(units, len(src)+1)
	for i := 0; i <= len(src); i++ {
		u[i] = chars.Len16(src[:i])
	}
	return u
}

func (u units) span(s Span) []any { return []any{u[s[0]], u[s[1]]} }

func (u units) spanPtr(o object, key string, s *Span) {
	if s != nil {
		o[key] = u.span(*s)
	}
}

func (u units) metadata(ms []MetadataOccurrence) []any {
	out := []any{}
	for _, m := range ms {
		out = append(out, object{"key": m.Key, "span": u.span(m.Span), "keySpan": u.span(m.KeySpan), "valueSpan": u.span(m.ValueSpan)})
	}
	return out
}

func (u units) spans(ss []Span) []any {
	out := []any{}
	for _, s := range ss {
		out = append(out, u.span(s))
	}
	return out
}

func (u units) statement(s *Statement) object {
	o := object{"kind": s.Kind, "span": u.span(s.Span)}
	if s.Nodes != nil {
		nodes := []any{}
		for _, n := range s.Nodes {
			v := object{"id": n.ID, "span": u.span(n.Span), "idSpan": u.span(n.IDSpan)}
			u.spanPtr(v, "labelSpan", n.LabelSpan)
			if n.Defines {
				v["defines"] = true
			}
			u.spanPtr(v, "stereotypeSpan", n.StereotypeSpan)
			if n.Metadata != nil {
				v["metadata"] = u.metadata(n.Metadata)
			}
			if n.ClassSpans != nil {
				v["classSpans"] = u.spans(n.ClassSpans)
			}
			nodes = append(nodes, v)
		}
		o["nodes"] = nodes
	}
	if s.Edges != nil {
		edges := []any{}
		for _, e := range s.Edges {
			v := object{"id": e.ID, "span": u.span(e.Span)}
			u.spanPtr(v, "labelSpan", e.LabelSpan)
			u.spanPtr(v, "idSpan", e.IDSpan)
			if e.Metadata != nil {
				v["metadata"] = u.metadata(e.Metadata)
			}
			edges = append(edges, v)
		}
		o["edges"] = edges
	}
	if s.HasGroup() {
		o["group"] = s.Group
	}
	u.spanPtr(o, "idSpan", s.IDSpan)
	u.spanPtr(o, "titleSpan", s.TitleSpan)
	u.spanPtr(o, "endSpan", s.EndSpan)
	o.opt("ref", s.Ref)
	u.spanPtr(o, "refSpan", s.RefSpan)
	if s.Children != nil {
		children := []any{}
		for _, c := range s.Children {
			children = append(children, u.statement(c))
		}
		o["children"] = children
	}
	if s.Metadata != nil {
		o["metadata"] = u.metadata(s.Metadata)
	}
	if s.ClassSpans != nil {
		o["classSpans"] = u.spans(s.ClassSpans)
	}
	return o
}

// encodeJSON is a JSON value as the oracle's enc writes it.
func encodeJSON(v any) any {
	switch x := v.(type) {
	case *Object:
		entries := []any{}
		for _, k := range x.Keys {
			entries = append(entries, []any{k, encodeJSON(x.Values[k])})
		}
		return map[string]any{"$o": entries}
	case []any:
		out := []any{}
		for _, e := range x {
			out = append(out, encodeJSON(e))
		}
		return out
	case float64:
		return yaml.Encode(x, false)
	}
	return v
}

func dumpModel(m *Model) object {
	u := unitsOf(m.Source)
	actors := []any{}
	for _, a := range m.Actors {
		v := object{"id": a.ID, "label": a.Label, "labelType": string(a.LabelType), "type": a.Type, "business": a.Business,
			"classes": strs(a.Classes), "styles": strs(a.Styles)}
		v.opt("icon", a.Icon)
		v.opt("stereotype", a.Stereotype)
		v.opt("parentId", a.ParentID)
		actors = append(actors, v)
	}
	useCases := []any{}
	for _, uc := range m.UseCases {
		v := object{"id": uc.ID, "label": uc.Label, "labelType": string(uc.LabelType), "shape": uc.Shape,
			"business": uc.Business, "classes": strs(uc.Classes), "styles": strs(uc.Styles)}
		v.opt("stereotype", uc.Stereotype)
		v.opt("parentId", uc.ParentID)
		useCases = append(useCases, v)
	}
	boundaries := []any{}
	for _, b := range m.SystemBoundaries {
		boundaries = append(boundaries, object{"id": b.ID, "label": b.Label, "labelType": string(b.LabelType), "type": b.Type,
			"members": strs(b.Members), "classes": strs(b.Classes), "styles": strs(b.Styles)})
	}
	relationships := []any{}
	for _, r := range m.Relationships {
		v := object{"id": r.ID, "explicitId": r.ExplicitID, "source": r.Source, "target": r.Target, "type": r.Type,
			"arrowType": r.ArrowType, "minlen": r.MinLen, "classes": strs(r.Classes), "styles": strs(r.Styles), "animate": r.Animate}
		if r.LabelType != "" {
			v["label"] = r.Label
			v["labelType"] = string(r.LabelType)
		}
		v.opt("animation", r.Animation)
		relationships = append(relationships, v)
	}
	notes := []any{}
	for _, n := range m.Notes {
		notes = append(notes, object{"id": n.ID, "target": n.Target, "label": n.Label, "labelType": string(n.LabelType)})
	}
	jsons := []any{}
	for _, j := range m.JSONNodes {
		order := []any{}
		for _, p := range j.PropertyOrder.Pointers() {
			keys, _ := j.PropertyOrder.Keys(p)
			order = append(order, []any{p, strs(keys)})
		}
		jsons = append(jsons, object{"id": j.ID, "value": encodeJSON(j.Value), "propertyOrder": order,
			"classes": strs(j.Classes), "styles": strs(j.Styles)})
	}
	classDefs := []any{}
	for _, c := range m.ClassDefs {
		classDefs = append(classDefs, object{"id": c.ID, "styles": strs(c.Styles)})
	}
	statements := []any{}
	for _, s := range m.Statements {
		statements = append(statements, u.statement(s))
	}
	o := object{"actors": actors, "useCases": useCases, "boundaries": boundaries, "relationships": relationships,
		"notes": notes, "jsonNodes": jsons, "classDefs": classDefs, "direction": m.Direction, "accTitle": m.AccTitle,
		"accDescr": m.AccDescr, "accDescription": m.AccDescription, "headerSpan": u.span(m.HeaderSpan),
		"statements": statements}
	o.opt("title", m.Title)
	return o
}

func dumpGraph(g *graph) object {
	nodes := []any{}
	for _, n := range g.Nodes {
		v := object{"id": n.ID, "label": n.Label, "labelType": string(n.LabelType), "shape": n.Shape, "isGroup": n.IsGroup,
			"padding": n.Padding, "cssClasses": n.CSSClasses, "cssStyles": strs(n.CSSStyles),
			"cssCompiledStyles": strs(n.CSSCompiledStyles)}
		v.opt("actorType", n.ActorType)
		if n.HasBusiness {
			v["business"] = n.Business
		}
		v.opt("icon", n.Icon)
		v.opt("stereotype", n.Stereotype)
		v.opt("parentId", n.ParentID)
		v.opt("noteTarget", n.NoteTarget)
		if n.Shape == "usecaseJsonTable" {
			rows := []any{}
			for _, r := range n.JSONRows {
				rows = append(rows, object{"key": r.Key, "accessibleKey": r.AccessibleKey, "value": r.Value})
			}
			v["jsonRows"] = rows
		}
		v.opt("boundaryType", n.BoundaryType)
		nodes = append(nodes, v)
	}
	edges := []any{}
	for _, e := range g.Edges {
		v := object{"id": e.ID, "start": e.Start, "end": e.End, "relationshipType": e.RelationshipType, "pattern": e.Pattern,
			"arrowTypeStart": e.ArrowTypeStart, "arrowTypeEnd": e.ArrowTypeEnd, "classes": e.Classes, "style": strs(e.Style),
			"cssCompiledStyles": strs(e.CSSCompiledStyles), "animate": e.Animate, "minlen": e.MinLen,
			"explicitId": e.ExplicitID, "internal": e.Internal}
		v.opt("label", e.Label)
		v.opt("labelType", string(e.LabelType))
		v.opt("animation", e.Animation)
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

// check compares a parse with Pele's result.
func check(t *testing.T, what, src string, m *Model, err error, want peleResult) bool {
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
	return compare(t, what+" model", src, dumpModel(m), want.Model) &&
		compare(t, what+" graph", src, dumpGraph(buildGraph(m)), want.Graph)
}

func loadCases(t *testing.T) []peleCase {
	t.Helper()
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/usecase.json", &p); err != nil {
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
		m, err := Parse(c.Src, "")
		ok := check(t, "Parse", c.Src, m, err, c.Direct)
		if ok {
			p, err := diagram.Prepare(c.Src, diagram.Options{})
			var typ diagram.Type
			if p != nil {
				typ = p.Type
			}
			switch {
			case err != nil:
				if e, isPele := err.(*diagram.Error); isPele && e.Code == diagram.UnsupportedDiagram && c.Error == nil {
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
			case typ == diagram.UseCase:
				m, err := ParsePrepared(p)
				ok = check(t, "ParsePrepared", c.Src, m, err, c.peleResult)
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
