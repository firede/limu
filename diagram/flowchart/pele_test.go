package flowchart

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
)

// The fixtures are what Pele makes of the same cases; ./scripts/oracle.sh writes them. The
// DB and the graph are written as the oracle writes Pele's, field by field, so that any field
// that differs shows.

type peleCase struct {
	Src      string          `json:"src"`
	MaxEdges int             `json:"maxEdges"`
	Type     *string         `json:"type"`
	DB       any             `json:"db"`
	Graph    any             `json:"graph"`
	Error    map[string]any  `json:"error"`
	Direct   *peleCaseResult `json:"direct"`
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

func (o object) num(key string, value *float64) {
	if value != nil {
		o[key] = yaml.Encode(*value, false)
	}
}

func (o object) val(key string, value any) {
	if value != nil {
		o[key] = yaml.Encode(value, false)
	}
}

func strs(s []string) []any {
	out := make([]any, len(s))
	for i, x := range s {
		out[i] = x
	}
	return out
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
			"classes": strs(n.Classes), "props": props}
		v.opt("type", n.Type)
		v.opt("dir", n.Dir)
		v.opt("link", n.Link)
		v.opt("linkTarget", n.LinkTarget)
		v.opt("icon", n.Icon)
		v.opt("form", n.Form)
		v.opt("pos", n.Pos)
		v.opt("img", n.Img)
		v.opt("constraint", n.Constraint)
		v.num("assetWidth", n.AssetWidth)
		v.num("assetHeight", n.AssetHeight)
		nodes = append(nodes, v)
	}
	o["nodes"] = nodes
	edges := []any{}
	for _, e := range db.Edges {
		v := object{"start": e.Start, "end": e.End, "type": e.Type, "text": e.Text, "labelType": string(e.LabelType),
			"stroke": e.Stroke, "classes": strs(e.Classes), "id": e.ID, "isUserDefinedId": e.IsUserDefinedID}
		if e.Length != 0 {
			v["length"] = e.Length
		}
		if e.Style != nil {
			v["style"] = strs(e.Style)
		}
		v.val("interpolate", e.Interpolate)
		v.val("animate", e.Animate)
		v.val("animation", e.Animation)
		edges = append(edges, v)
	}
	o["edges"] = edges
	subgraphs := []any{}
	for _, s := range db.Subgraphs {
		v := object{"id": s.ID, "nodes": strs(s.Nodes), "title": s.Title, "classes": strs(s.Classes),
			"labelType": string(s.LabelType)}
		v.opt("dir", s.Dir)
		if s.Metadata != nil {
			v["metadata"] = yaml.Encode(s.Metadata, false)
		}
		subgraphs = append(subgraphs, v)
	}
	o["subgraphs"] = subgraphs
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
		v.opt("icon", n.Icon)
		v.opt("pos", n.Pos)
		v.opt("img", n.Img)
		v.num("assetWidth", n.AssetWidth)
		v.num("assetHeight", n.AssetHeight)
		v.opt("constraint", n.Constraint)
		if n.ColorIndex >= 0 {
			v["colorIndex"] = n.ColorIndex
		}
		if n.Metadata != nil {
			v["metadata"] = yaml.Encode(n.Metadata, false)
		}
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
	data, err := fixture.Read("testdata/pele/flowchart.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Flowcharts []peleCase `json:"flowcharts"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return p.Flowcharts
}

func TestParseMatchesPele(t *testing.T) {
	cases := loadCases(t)
	if len(cases) < 2000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed := 0
	for _, c := range cases {
		ok := true
		p, err := diagram.Prepare(c.Src, diagram.Options{MaxEdges: c.MaxEdges})
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
		case typ != diagram.Flowchart:
			db := NewDB(Options{MaxEdges: -1})
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
				ok = compare(t, "direct db", c.Src, dumpDB(db), c.Direct.DB) &&
					compare(t, "direct graph", c.Src, dumpGraph(BuildGraph(db, "")), c.Direct.Graph)
			}
		default:
			db, err := ParsePrepared(p)
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
				ok = compare(t, "db", c.Src, dumpDB(db), c.DB) &&
					compare(t, "graph", c.Src, dumpGraph(BuildGraph(db, DefaultCurve(p.Config))), c.Graph)
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}

// The huge diagram from Mermaid's specs, which Mermaid skips because its parser needs
// minutes for it, as Pele's parser test has it.
func TestHuge(t *testing.T) {
	body := strings.Repeat(strings.Repeat("A-->B;B-->A;", 415)+"A-->B;", 57) + strings.Repeat("A-->B;B-->A;", 275)
	db := NewDB(Options{MaxEdges: -1})
	if err := ParseInto("graph LR;"+body, db); err != nil {
		t.Fatal(err)
	}
	if db.Edges[0].Type != "arrow_point" || len(db.Edges) != 47917 || len(db.Nodes) != 2 {
		t.Errorf("%s, %d edges, %d nodes", db.Edges[0].Type, len(db.Edges), len(db.Nodes))
	}
}
