package sequence

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The tokens
// and the DB are written as the oracle writes Pele's, field by field, with offsets in UTF-16
// code units, so that any field that differs shows.

type peleResult struct {
	Tokens []any          `json:"tokens"`
	DB     any            `json:"db"`
	Error  map[string]any `json:"error"`
}

type peleCase struct {
	Src    string
	Type   *string
	Direct peleResult
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	var raw struct {
		Src    string         `json:"src"`
		Type   *string        `json:"type"`
		Tokens []any          `json:"tokens"`
		Direct peleResult     `json:"direct"`
		DB     any            `json:"db"`
		Error  map[string]any `json:"error"`
		// Same says the pipeline's result is the direct one.
		Same bool `json:"same"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Src, c.Type, c.Direct = raw.Src, raw.Type, raw.Direct
	c.peleResult = peleResult{raw.Tokens, raw.DB, raw.Error}
	if raw.Same {
		c.DB, c.Error = c.Direct.DB, c.Direct.Error
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

func dumpTokens(src string) []any {
	toks := tokenize(src)
	out := make([]any, len(toks))
	for i, t := range toks {
		out[i] = []any{tokenNames[t.typ], t.text, chars.Len16(src[:t.start])}
	}
	return out
}

func dumpEntries(e Entries) []any {
	out := []any{}
	for _, k := range e.Keys {
		out = append(out, []any{k, e.JSON(k)})
	}
	return out
}

func dumpDB(db *DB) object {
	boxIndex := map[*Box]int{}
	boxes := []any{}
	for i, b := range db.Boxes {
		boxIndex[b] = i
		v := object{"wrap": b.Wrap, "fill": b.Fill, "actorKeys": strs(b.ActorKeys)}
		v.opt("name", b.Name)
		boxes = append(boxes, v)
	}
	actors := []any{}
	for _, a := range db.Actors {
		box := -1
		if a.Box != nil {
			box = boxIndex[a.Box]
		}
		v := object{"id": a.ID, "box": box, "name": a.Name, "description": a.Description, "wrap": a.Wrap,
			"links": dumpEntries(a.Links), "properties": dumpEntries(a.Properties), "type": a.Type}
		v.opt("prevActor", a.PrevActor)
		v.opt("nextActor", a.NextActor)
		if a.HasDetails {
			v["details"] = a.Details
		}
		actors = append(actors, v)
	}
	positions := func(list []Position) []any {
		out := []any{}
		for _, p := range list {
			out = append(out, []any{p.ID, p.Index})
		}
		return out
	}
	messages := []any{}
	for _, m := range db.Messages {
		v := object{"id": m.ID, "wrap": m.Wrap, "type": m.Type}
		v.opt("from", m.From)
		v.opt("to", m.To)
		switch {
		case m.Numbering != nil:
			n := object{"visible": m.Numbering.Visible}
			if m.Numbering.Given {
				n["start"], n["step"] = m.Numbering.Start, m.Numbering.Step
			}
			v["message"] = n
		case m.Type == NoteType:
			v["message"] = m.Text
			v["placement"] = m.Placement
		default:
			v["message"] = m.Text
			v["activate"] = m.Activate
			v["centralConnection"] = m.CentralConnection
		}
		messages = append(messages, v)
	}
	notes := []any{}
	for _, n := range db.Notes {
		v := object{"placement": n.Placement, "message": n.Message, "wrap": n.Wrap}
		if n.Actors != nil {
			v["actor"] = strs(n.Actors)
		} else {
			v["actor"] = n.Actor
		}
		notes = append(notes, v)
	}
	o := object{"actors": actors, "created": positions(db.Created), "destroyed": positions(db.Destroyed),
		"boxes": boxes, "messages": messages, "notes": notes}
	o.opt("title", db.Title)
	if db.HasAccTitle {
		o["accTitle"] = db.AccTitle
	}
	if db.HasAccDescr {
		o["accDescr"] = db.AccDescr
	}
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

func check(t *testing.T, label, src, text string, db *DB, err error, want peleResult) bool {
	t.Helper()
	if !compare(t, label+" tokens", src, dumpTokens(text), want.Tokens) {
		return false
	}
	switch {
	case err != nil && want.Error == nil:
		t.Errorf("%s of %q: %v, Pele had none", label, src, err)
		return false
	case err != nil:
		return compare(t, label+" error", src, dumpError(err), want.Error)
	case want.Error != nil:
		t.Errorf("%s of %q: no error, Pele %v", label, src, want.Error["message"])
		return false
	}
	return compare(t, label+" db", src, dumpDB(db), want.DB)
}

func TestParseMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/sequence.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Cases []peleCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 5000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
		db, err := Parse(c.Src, nil, "")
		ok := check(t, "Parse", c.Src, c.Src, db, err, c.Direct)
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
			case typ == diagram.Sequence:
				db, err := ParsePrepared(pre)
				ok = check(t, "ParsePrepared", c.Src, pre.Source(false), db, err, c.peleResult)
			}
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
	src := "sequenceDiagram\n" + strings.Repeat("loop a\n", 5000) + "A->>B: x\n" + strings.Repeat("end\n", 5000)
	db, err := Parse(src, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Messages) != 10001 {
		t.Errorf("%d messages", len(db.Messages))
	}
}
