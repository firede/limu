package quadrant

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The tokens
// and the model are written as the oracle writes Pele's, with offsets in UTF-16 code units,
// so that any field that differs shows.

type peleResult struct {
	Tokens any            `json:"tokens"`
	Model  any            `json:"model"`
	Error  map[string]any `json:"error"`
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
		Tokens any            `json:"tokens"`
		Model  any            `json:"model"`
		Error  map[string]any `json:"error"`
		// Same says the pipeline's result is the direct one.
		Same bool `json:"same"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Src, c.Type, c.Direct = raw.Src, raw.Type, raw.Direct
	c.peleResult = peleResult{raw.Tokens, raw.Model, raw.Error}
	if raw.Same {
		c.peleResult = raw.Direct
	}
	return nil
}

func dumpTokens(src string) []any {
	t := tokenize(src)
	out := []any{}
	for i, typ := range t.Types {
		out = append(out, []any{tokenNames[typ], t.Texts[i], chars.Len16(src[:t.Starts[i]])})
	}
	return out
}

func dumpText(t *Text) any {
	if t == nil {
		return nil
	}
	typ := "text"
	if t.Markdown {
		typ = "markdown"
	}
	return map[string]any{"text": t.Text, "type": typ}
}

func dumpStyle(s PointStyle) map[string]any {
	out := map[string]any{}
	if s.Radius != nil {
		out["radius"] = fixture.Number(*s.Radius)
	}
	if s.Color != nil {
		out["color"] = *s.Color
	}
	if s.StrokeColor != nil {
		out["strokeColor"] = *s.StrokeColor
	}
	if s.StrokeWidth != nil {
		out["strokeWidth"] = *s.StrokeWidth
	}
	return out
}

func dumpModel(m *Model) map[string]any {
	quadrants := []any{}
	for _, q := range m.Quadrants {
		quadrants = append(quadrants, dumpText(q))
	}
	points := []any{}
	for _, p := range m.Points {
		points = append(points, map[string]any{"x": fixture.Number(p.X), "y": fixture.Number(p.Y), "text": dumpText(&p.Text), "className": p.Class,
			"style": dumpStyle(p.PointStyle)})
	}
	classes := []any{}
	for _, name := range m.ClassOrder {
		classes = append(classes, []any{name, dumpStyle(m.Classes[name])})
	}
	o := map[string]any{"quadrants": quadrants, "xAxisLeft": dumpText(m.XAxisLeft), "xAxisRight": dumpText(m.XAxisRight),
		"yAxisBottom": dumpText(m.YAxisBottom), "yAxisTop": dumpText(m.YAxisTop), "points": points, "classes": classes}
	opt := func(key, value string) {
		if value != "" {
			o[key] = value
		}
	}
	opt("title", m.Title)
	opt("accTitle", m.AccTitle)
	opt("accDescr", m.AccDescr)
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

// check compares the tokens of text given to the parser, and its parse, with Pele's.
func check(t *testing.T, what, text string, m *Model, err error, want peleResult) bool {
	t.Helper()
	if !compare(t, what+" tokens", text, dumpTokens(text), want.Tokens) {
		return false
	}
	switch {
	case err != nil && want.Error == nil:
		t.Errorf("%s(%q): %v, Pele had none", what, text, err)
		return false
	case err != nil:
		return compare(t, what+" error", text, dumpError(err), want.Error)
	case want.Error != nil:
		t.Errorf("%s(%q): no error, Pele %v", what, text, want.Error)
		return false
	}
	return compare(t, what+" model", text, dumpModel(m), want.Model)
}

func TestParseMatchesPele(t *testing.T) {
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := fixture.Load("testdata/pele/quadrant.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 5000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed, piped, parsed := 0, 0, 0
	for _, c := range p.Cases {
		m, err := Parse(c.Src, "")
		ok := check(t, "Parse", c.Src, m, err, c.Direct)
		if ok {
			pre, err := diagram.Prepare(c.Src, diagram.Options{})
			var typ diagram.Type
			if pre != nil {
				typ = pre.Type
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
			case typ == diagram.QuadrantChart:
				m, err := ParsePrepared(pre)
				ok = check(t, "ParsePrepared", pre.Source(false), m, err, c.peleResult)
				piped++
				if err == nil {
					parsed++
				}
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	if piped < 4000 || parsed < 1500 {
		t.Errorf("only %d cases were quadrant charts, %d of them parsed", piped, parsed)
	}
}
