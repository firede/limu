package agentflow

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The DB, its
// mappings, diagnostics and semantic model, and the graph are written as the oracle writes
// Pele's, field by field, so that any field that differs shows.

type peleCase struct {
	Src      string `json:"src"`
	MaxEdges int    `json:"maxEdges"`
	// Forced cases go to the parser as they are, without being prepared.
	Forced bool            `json:"forced"`
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

func (o object) val(key string, value any) {
	if value != nil {
		o[key] = yaml.Encode(value, false)
	}
}

func (o object) meta(key string, m *yaml.Map) {
	if m != nil {
		o[key] = yaml.Encode(m, false)
	}
}

func strs(s []string) []any {
	out := make([]any, len(s))
	for i, x := range s {
		out[i] = x
	}
	return out
}

func position(p *ElementPosition) []any {
	return []any{p.StartLine, p.StartColumn, p.EndLine, p.EndColumn, p.StartIndex, p.EndIndex}
}

func dumpDiagnostics(list []*Diagnostic) []any {
	out := []any{}
	for _, d := range list {
		v := object{"id": d.ID, "severity": d.Severity, "message": d.Message}
		v.opt("nodeId", d.NodeID)
		v.opt("edgeId", d.EdgeID)
		if d.Position != nil {
			v["position"] = position(d.Position)
		}
		out = append(out, v)
	}
	return out
}

func dumpSemantic(s *SemanticModel) object {
	vertices := []any{}
	for _, v := range s.Vertices {
		o := object{"id": v.ID, "label": v.Label, "vertexKind": v.VertexKind}
		o.opt("shape", v.Shape)
		o.meta("metadata", v.Metadata)
		vertices = append(vertices, o)
	}
	edges := []any{}
	for _, e := range s.Edges {
		o := object{"start": e.Start, "end": e.End, "id": e.ID}
		o.opt("label", e.Label)
		o.opt("type", e.Type)
		o.opt("stroke", e.Stroke)
		o.opt("edgeSemantic", e.EdgeSemantic)
		if e.Length != 0 {
			o["length"] = e.Length
		}
		o.meta("metadata", e.Metadata)
		edges = append(edges, o)
	}
	subgraphs := []any{}
	for _, g := range s.Subgraphs {
		o := object{"id": g.ID, "type": g.Type, "title": g.Title, "nodes": strs(g.Nodes)}
		o.opt("direction", g.Direction)
		o.meta("metadata", g.Metadata)
		subgraphs = append(subgraphs, o)
	}
	connectors := []any{}
	for _, c := range s.Connectors {
		o := object{"id": c.ID}
		o.opt("title", c.Title)
		o.meta("metadata", c.Metadata)
		connectors = append(connectors, o)
	}
	return object{"direction": s.Direction, "vertices": vertices, "edges": edges, "subGraphs": subgraphs,
		"connectors": connectors, "diagnostics": dumpDiagnostics(s.Diagnostics)}
}

func dumpDB(db *DB) object {
	o := object{"direction": db.Direction}
	o.opt("title", db.Title)
	o.opt("accTitle", db.AccTitle)
	o.opt("accDescr", db.AccDescr)
	if db.DefaultEdgeStyle != nil {
		o["defaultEdgeStyle"] = strs(db.DefaultEdgeStyle)
	}
	o.opt("defaultInterpolate", db.DefaultInterpolate)
	nodes := []any{}
	for _, n := range db.Nodes {
		props := []any{}
		for _, p := range n.Props {
			props = append(props, []any{p.Key, p.Value})
		}
		v := object{"id": n.ID, "text": n.Text, "labelType": string(n.LabelType), "styles": strs(n.Styles),
			"classes": strs(n.Classes), "props": props, "isConnector": n.IsConnector}
		v.opt("type", n.Type)
		v.opt("dir", n.Dir)
		v.opt("link", n.Link)
		v.opt("linkTarget", n.LinkTarget)
		v.meta("metadata", n.Metadata)
		nodes = append(nodes, v)
	}
	o["nodes"] = nodes
	edges := []any{}
	for _, e := range db.Edges {
		v := object{"start": e.Start, "end": e.End, "text": e.Text, "labelType": string(e.LabelType),
			"classes": strs(e.Classes), "id": e.ID, "isUserDefinedId": e.IsUserDefinedID}
		v.opt("type", e.Type)
		v.opt("stroke", e.Stroke)
		v.opt("edgeSemantic", e.EdgeSemantic)
		if e.Length != 0 {
			v["length"] = e.Length
		}
		if e.Style != nil {
			v["style"] = strs(e.Style)
		}
		v.val("interpolate", e.Interpolate)
		v.val("animate", e.Animate)
		v.val("animation", e.Animation)
		v.meta("metadata", e.Metadata)
		edges = append(edges, v)
	}
	o["edges"] = edges
	subgraphs := []any{}
	for _, s := range db.Subgraphs {
		v := object{"id": s.ID, "nodes": strs(s.Nodes), "title": s.Title, "classes": strs(s.Classes),
			"labelType": string(s.LabelType), "type": s.Type}
		v.opt("dir", s.Dir)
		v.meta("metadata", s.Metadata)
		subgraphs = append(subgraphs, v)
	}
	o["subgraphs"] = subgraphs
	connectors := []any{}
	for _, c := range db.Connectors {
		connectors = append(connectors, c.ID)
	}
	o["connectors"] = connectors
	classes := []any{}
	for _, c := range db.Classes {
		classes = append(classes, object{"id": c.ID, "styles": strs(c.Styles), "textStyles": strs(c.TextStyles)})
	}
	o["classes"] = classes
	tooltips := []any{}
	for _, t := range db.Tooltips {
		tooltips = append(tooltips, []any{t.ID, t.Text})
	}
	o["tooltips"] = tooltips
	mappings := []any{}
	for _, m := range db.Mappings {
		mappings = append(mappings, append([]any{m.ID, m.Type}, position(m.Position)...))
	}
	o["mappings"] = mappings
	o["diagnostics"] = dumpDiagnostics(db.Diagnostics)
	o["semantic"] = dumpSemantic(db.SemanticModel())
	return o
}

func dumpGraph(g *Graph) object {
	nodes := []any{}
	for _, n := range g.Nodes {
		v := object{"id": n.ID, "label": n.Label, "labelType": string(n.LabelType), "shape": n.Shape,
			"isGroup": n.IsGroup, "cssStyles": strs(n.CSSStyles), "cssCompiledStyles": strs(n.CSSCompiledStyles),
			"cssClasses": n.CSSClasses}
		if n.HasParent {
			v["parentId"] = n.Parent
		}
		v.opt("dir", n.Dir)
		v.opt("link", n.Link)
		v.opt("linkTarget", n.LinkTarget)
		v.opt("tooltip", n.Tooltip)
		if n.ColorIndex >= 0 {
			v["colorIndex"] = n.ColorIndex
		}
		v.opt("kind", n.Kind)
		v.meta("metadata", n.Metadata)
		nodes = append(nodes, v)
	}
	edges := []any{}
	for _, e := range g.Edges {
		v := object{"id": e.ID, "isUserDefinedId": e.IsUserDefinedID, "start": e.Start, "end": e.End, "type": e.Type,
			"label": e.Label, "labelType": string(e.LabelType), "arrowTypeStart": e.ArrowTypeStart,
			"arrowTypeEnd": e.ArrowTypeEnd, "cssCompiledStyles": strs(e.CSSCompiledStyles), "style": strs(e.Style),
			"classes": e.Classes}
		v.opt("thickness", e.Thickness)
		v.opt("pattern", e.Pattern)
		if e.MinLen != 0 {
			v["minlen"] = e.MinLen
		}
		v.val("animate", e.Animate)
		v.val("animation", e.Animation)
		v.val("curve", e.Curve)
		v.meta("metadata", e.Metadata)
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
	if e.Message == "Diagram is nested too deeply." {
		out = map[string]any{"code": string(e.Code), "message": e.Message}
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
	var p struct {
		Agentflows []peleCase `json:"agentflows"`
	}
	if err := fixture.Load("testdata/pele/agentflow.json", &p); err != nil {
		t.Fatal(err)
	}
	return p.Agentflows
}

func TestParseMatchesPele(t *testing.T) {
	cases := loadCases(t)
	if len(cases) < 3000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed, parsed, errors := 0, 0, 0
	for _, c := range cases {
		ok := true
		var p *diagram.Prepared
		var err error
		if !c.Forced {
			p, err = diagram.Prepare(c.Src, diagram.Options{MaxEdges: c.MaxEdges})
		}
		var typ diagram.Type
		if c.Forced {
			typ = ""
		} else if p != nil {
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
		case typ != diagram.AgentFlow:
			db := NewDB(Options{MaxEdges: -1})
			if err := ParseInto(c.Src, db); err != nil {
				errors++
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
				parsed++
				// Built twice, as the oracle builds it: the diagnostics of the second are not
				// reported again.
				BuildGraph(db, Palette, "")
				g := BuildGraph(db, Palette, "")
				ok = compare(t, "direct db", c.Src, dumpDB(db), c.Direct.DB) &&
					compare(t, "direct graph", c.Src, dumpGraph(g), c.Direct.Graph)
			}
		default:
			db, err := ParsePrepared(p)
			if err != nil {
				errors++
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
				parsed++
				ok = compare(t, "db", c.Src, dumpDB(db), c.DB) &&
					compare(t, "graph", c.Src, dumpGraph(db.Graph(DefaultCurve(p.Config))), c.Graph)
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	t.Logf("%d cases: %d parsed, %d errors", len(cases), parsed, errors)
}
