package wardley

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/yaml"
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

// enc is a number as the oracle writes it.
func enc(f float64) any { return yaml.Encode(f, false) }

// encOpt is a number Pele may leave undefined.
func encOpt(f float64, has bool) any {
	if !has {
		return undefined
	}
	return enc(f)
}

func encPtr(f *float64) any {
	if f == nil {
		return undefined
	}
	return enc(*f)
}

func encStr(s string, has bool) any {
	if !has {
		return undefined
	}
	return s
}

// encTrue is a flag Pele sets to true or leaves undefined.
func encTrue(b bool) any {
	if !b {
		return undefined
	}
	return true
}

func dumpLabel(l *labelAst) any {
	if l == nil {
		return undefined
	}
	return []any{l.negX, enc(l.offsetX), l.negY, enc(l.offsetY)}
}

func dumpPlaced(list []placedAst) []any {
	out := []any{}
	for _, p := range list {
		out = append(out, []any{p.name, enc(p.a), enc(p.b)})
	}
	return out
}

func dumpAST(t *ast) map[string]any {
	out := map[string]any{
		"title":     encStr(t.title, t.hasTitle),
		"accTitle":  encStr(t.accTitle, t.hasAccTitle),
		"accDescr":  encStr(t.accDescr, t.hasAccDescr),
		"size":      undefined,
		"evolution": undefined,
	}
	if t.size != nil {
		out["size"] = []any{enc(t.size[0]), enc(t.size[1])}
	}
	if t.evolution != nil {
		stages := []any{}
		for _, s := range t.evolution {
			stages = append(stages, []any{s.name, encOpt(s.boundary, s.hasBoundary), encStr(s.secondName, s.hasSecond)})
		}
		out["evolution"] = stages
	}
	out["anchors"] = dumpPlaced(t.anchors)
	components := []any{}
	for _, c := range t.components {
		components = append(components, []any{c.name, enc(c.visibility), enc(c.evolution), dumpLabel(c.label),
			encStr(c.strategy, c.hasDecorator), c.inertia})
	}
	out["components"] = components
	links := []any{}
	for _, l := range t.links {
		links = append(links, []any{l.from, encStr(l.fromPort, l.hasFromPort), encStr(l.arrow, l.hasArrow), l.to,
			encStr(l.toPort, l.hasToPort), encStr(l.linkLabel, l.hasLinkLabel)})
	}
	out["links"] = links
	evolves := []any{}
	for _, e := range t.evolves {
		evolves = append(evolves, []any{e.component, enc(e.target)})
	}
	out["evolves"] = evolves
	pipelines := []any{}
	for _, p := range t.pipelines {
		components := []any{}
		for _, c := range p.components {
			components = append(components, []any{c.name, enc(c.evolution), dumpLabel(c.label)})
		}
		pipelines = append(pipelines, []any{p.parent, components})
	}
	out["pipelines"] = pipelines
	out["notes"] = dumpPlaced(t.notes)
	boxes := []any{}
	for _, a := range t.annotations {
		boxes = append(boxes, []any{enc(a[0]), enc(a[1])})
	}
	out["annotations"] = boxes
	annotation := []any{}
	for _, a := range t.annotation {
		annotation = append(annotation, []any{enc(a.number), enc(a.x), enc(a.y), a.text})
	}
	out["annotation"] = annotation
	out["accelerators"] = dumpPlaced(t.accelerators)
	out["deaccelerators"] = dumpPlaced(t.deaccelerators)
	return out
}

func forces(list []Force) []any {
	out := []any{}
	for _, f := range list {
		out = append(out, []any{f.Name, enc(f.X), enc(f.Y)})
	}
	return out
}

func dumpModel(m *Model) map[string]any {
	opt := func(s string) any {
		if s == "" {
			return undefined
		}
		return s
	}
	out := map[string]any{"title": opt(m.Title), "accTitle": opt(m.AccTitle), "accDescr": opt(m.AccDescr)}
	nodes := []any{}
	for _, n := range m.Nodes {
		nodes = append(nodes, []any{n.ID, n.Label, enc(n.X), enc(n.Y), encStr(n.Class, n.Class != ""), encPtr(n.LabelOffsetX),
			encPtr(n.LabelOffsetY), encTrue(n.InPipeline), encTrue(n.IsPipelineParent),
			map[bool]any{true: n.Inertia, false: undefined}[n.HasInertia], encStr(n.SourceStrategy, n.SourceStrategy != "")})
	}
	out["nodes"] = nodes
	links := []any{}
	for _, l := range m.Links {
		links = append(links, []any{l.Source, l.Target, l.Dashed, encStr(l.Label, l.HasLabel), encStr(l.Flow, l.Flow != "")})
	}
	out["links"] = links
	trends := []any{}
	for _, t := range m.Trends {
		trends = append(trends, []any{t.NodeID, enc(t.TargetX), enc(t.TargetY)})
	}
	out["trends"] = trends
	pipelines := []any{}
	for _, p := range m.Pipelines {
		pipelines = append(pipelines, []any{p.NodeID, p.ComponentIDs})
	}
	out["pipelines"] = pipelines
	annotations := []any{}
	for _, a := range m.Annotations {
		coords := []any{}
		for _, c := range a.Coordinates {
			coords = append(coords, []any{enc(c.X), enc(c.Y)})
		}
		annotations = append(annotations, []any{enc(a.Number), coords, a.Text})
	}
	out["annotations"] = annotations
	notes := []any{}
	for _, n := range m.Notes {
		notes = append(notes, []any{n.Text, enc(n.X), enc(n.Y)})
	}
	out["notes"] = notes
	out["accelerators"] = forces(m.Accelerators)
	out["deaccelerators"] = forces(m.Deaccelerators)
	out["annotationsBox"] = undefined
	if b := m.AnnotationsBox; b != nil {
		out["annotationsBox"] = []any{enc(b.X), enc(b.Y)}
	}
	axes := []any{undefined, undefined, undefined, undefined}
	if m.Axes.XLabel != nil {
		axes[0] = *m.Axes.XLabel
	}
	if m.Axes.YLabel != nil {
		axes[1] = *m.Axes.YLabel
	}
	if m.Axes.Stages != nil {
		axes[2] = m.Axes.Stages
	}
	if m.Axes.StageBoundaries != nil {
		bounds := []any{}
		for _, b := range m.Axes.StageBoundaries {
			bounds = append(bounds, enc(b))
		}
		axes[3] = bounds
	}
	out["axes"] = axes
	out["size"] = undefined
	if s := m.Size; s != nil {
		out["size"] = []any{enc(s.Width), enc(s.Height)}
	}
	return out
}

func dumpTokens(src string) [][2]string {
	toks, err := common.Tokenize(src, wardleyTable, "wardley")
	if err != nil {
		return nil
	}
	out := [][2]string{}
	for i, k := range toks.Kinds[:len(toks.Kinds)-1] {
		out = append(out, [2]string{wardleyTokens[k].Name, src[toks.Starts[i]:toks.Ends[i]]})
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
	if err := fixture.Load("testdata/pele/wardley.json", &p); err != nil {
		t.Fatal(err)
	}
	return p.Cases
}

func TestParseMatchesPele(t *testing.T) {
	cases := loadCases(t)
	if len(cases) < 6000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed, models := 0, 0
	for _, c := range cases {
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src), c.Direct.Tokens)
		tree, err := parseWardley(c.Src)
		switch {
		case err != nil && c.Direct.Error == nil:
			t.Errorf("parseWardley(%q): %v, Pele had none", c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "parseWardley error", c.Src, dumpError(err), c.Direct.Error) && ok
		case c.Direct.Error != nil:
			t.Errorf("parseWardley(%q): no error, Pele %v", c.Src, c.Direct.Error)
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
		case p.Type == diagram.Wardley:
			m, err := ParsePrepared(p)
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
	if models < 1500 {
		t.Errorf("only %d models", models)
	}
}
