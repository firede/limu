package gitgraph

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/fixture"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it. The tree
// and the model are written as the oracle writes Pele's, field by field, so that any field
// that differs shows.

type peleResult struct {
	AST   any            `json:"ast"`
	Model any            `json:"model"`
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
		AST    any            `json:"ast"`
		Model  any            `json:"model"`
		Error  map[string]any `json:"error"`
		// Same says the pipeline's result is the direct one.
		Same bool `json:"same"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	c.Src, c.Type, c.Direct = raw.Src, raw.Type, raw.Direct
	c.peleResult = peleResult{raw.AST, raw.Model, raw.Error}
	if raw.Same {
		c.peleResult = raw.Direct
	}
	return nil
}

// object builds a JSON object, leaving out the fields Pele leaves undefined.
type object map[string]any

func (o object) ptr(key string, v *string) {
	if v != nil {
		o[key] = *v
	}
}

func (o object) opt(key, v string) {
	if v != "" {
		o[key] = v
	}
}

func dumpAST(a *gitGraphAst) object {
	o := object{"$type": a.Type}
	o.opt("dir", a.Dir)
	o.ptr("title", a.Title)
	o.ptr("accTitle", a.AccTitle)
	o.ptr("accDescr", a.AccDescr)
	statements := []any{}
	for _, s := range a.Statements {
		v := object{"$type": s.Kind}
		switch s.Kind {
		case "Commit":
			v.ptr("id", s.ID)
			v.ptr("message", s.Message)
			v["tags"] = s.Tags
			v.ptr("type", s.Type)
		case "Branch":
			v["name"] = s.Name
			if s.Order != nil {
				v["order"] = fixture.Number(*s.Order)
			}
		case "Merge":
			v["branch"] = s.Branch
			v.ptr("id", s.ID)
			v["tags"] = s.Tags
			v.ptr("type", s.Type)
		case "Checkout":
			v["branch"] = s.Branch
		case "CherryPicking":
			v.ptr("id", s.ID)
			v["tags"] = s.Tags
			v.ptr("parent", s.Parent)
		}
		statements = append(statements, v)
	}
	o["statements"] = statements
	return o
}

func dumpModel(m *Model) object {
	o := object{"direction": m.Direction, "currentBranch": m.CurrentBranch, "lanes": m.Lanes}
	o.opt("title", m.Title)
	o.opt("accTitle", m.AccTitle)
	o.opt("accDescr", m.AccDescr)
	commits := []any{}
	for _, c := range m.Commits {
		v := object{"id": c.ID, "message": c.Message, "seq": c.Seq, "type": c.Type, "tags": c.Tags,
			"parents": c.Parents, "branch": c.Branch}
		if c.CustomType != nil {
			v["customType"] = *c.CustomType
		}
		if c.CustomID != nil {
			v["customId"] = *c.CustomID
		}
		commits = append(commits, v)
	}
	o["commits"] = commits
	branches := []any{}
	for _, b := range m.Branches {
		var head any
		if b.HasHead {
			head = b.Head
		}
		branches = append(branches, []any{b.Name, head})
	}
	o["branches"] = branches
	if m.Head != nil {
		o["head"] = m.Head.ID
	} else {
		o["head"] = nil
	}
	warnings := []any{}
	for _, w := range m.Warnings {
		warnings = append(warnings, w)
	}
	o["warnings"] = warnings
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

// check parses text as Pele's parse does, with config and title, and compares with Pele's
// result.
func check(t *testing.T, what, text string, config diagram.Config, title string, want peleResult, pre *diagram.Prepared) bool {
	t.Helper()
	ast, err := parseGit(text)
	if pre != nil {
		// Parse as ParsePrepared does, so that errors are located in the text as written.
		ast, err = common.ParsePrepared(pre, false, parseGit)
	}
	if err != nil {
		if want.Error == nil || want.AST != nil {
			t.Errorf("%s(%q): %v, Pele had none", what, text, err)
			return false
		}
		return compare(t, what+" error", text, dumpError(err), want.Error)
	}
	if want.AST == nil {
		t.Errorf("%s(%q): no error, Pele %v", what, text, want.Error)
		return false
	}
	if !compare(t, what+" tree", text, dumpAST(ast), want.AST) {
		return false
	}
	m, err := buildGit(ast, config, title)
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
	if err := fixture.Load("testdata/pele/gitgraph.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 5000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
		ok := check(t, "Parse", c.Src, diagram.Config{}, "", c.Direct, nil)
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
			case typ == diagram.GitGraph:
				ok = check(t, "ParsePrepared", pre.Source(false), pre.Config, pre.Title, c.peleResult, pre)
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}
