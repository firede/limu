package railroad

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
// and the tree of the text given to the parser of the notation its keyword names as it is,
// and the model of the text as it comes out of Pele's parseSource.

type peleDirect struct {
	Notation string         `json:"notation"`
	Tokens   [][2]string    `json:"tokens"`
	AST      map[string]any `json:"ast"`
	Error    map[string]any `json:"error"`
}

type peleCase struct {
	Src    string         `json:"src"`
	Type   *string        `json:"type"`
	Direct peleDirect     `json:"direct"`
	Model  map[string]any `json:"model"`
	Error  map[string]any `json:"error"`
}

// enc is a number as the oracle writes it.
func enc(f float64) any { return yaml.Encode(f, false) }

// dumpNode writes a node of the model as the oracle does.
func dumpNode(n *Node) []any {
	kids := func(tag string) []any {
		out := []any{tag}
		for _, k := range n.Kids {
			out = append(out, dumpNode(k))
		}
		return out
	}
	switch n.Type {
	case Terminal:
		return []any{"t", n.Text}
	case Nonterminal:
		return []any{"n", n.Text}
	case Special:
		return []any{"s", n.Text}
	case Sequence:
		return kids("seq")
	case Choice:
		return kids("alt")
	case Optional:
		return []any{"opt", dumpNode(n.Kids[0])}
	}
	return []any{"rep", enc(n.Min), enc(n.Max), dumpNode(n.Kids[0])}
}

func dumpModel(m *Model) map[string]any {
	rules := []any{}
	for _, r := range m.Rules {
		rules = append(rules, []any{r.Name, dumpNode(r.Definition)})
	}
	out := map[string]any{"notation": string(m.Notation), "rules": rules}
	for k, v := range map[string]string{"title": m.Title, "accTitle": m.AccTitle, "accDescr": m.AccDescr} {
		if v != "" {
			out[k] = v
		}
	}
	return out
}

// dumpGrammar writes a syntax tree's statements and rules as the oracle does, each rule's
// definition by def.
func dumpGrammar[P any](g *grammar[rule[P]], def func(P) any) map[string]any {
	rules := []any{}
	for _, r := range g.rules {
		rules = append(rules, map[string]any{"$type": r.typ, "name": r.name, "definition": def(r.definition)})
	}
	out := map[string]any{"$type": g.typ, "rules": rules}
	if g.hasTitle {
		out["title"] = g.title
	}
	if g.hasAccTitle {
		out["accTitle"] = g.accTitle
	}
	if g.hasAccDescr {
		out["accDescr"] = g.accDescr
	}
	return out
}

func dumpIr(e *irExpression) any {
	out := map[string]any{"$type": e.typ}
	switch e.typ {
	case "RailroadTerminalExpr":
		out["value"] = e.value
	case "RailroadNonTerminalExpr":
		out["name"] = e.value
	case "RailroadSpecialExpr":
		out["text"] = e.value
	case "RailroadSequenceExpr", "RailroadChoiceExpr":
		list := []any{}
		for _, k := range e.list {
			list = append(list, dumpIr(k))
		}
		if e.typ == "RailroadSequenceExpr" {
			out["elements"] = list
		} else {
			out["alternatives"] = list
		}
	default:
		out["element"] = dumpIr(e.element)
	}
	return out
}

func dumpEbnf(part ebnfPart) any {
	switch p := part.(type) {
	case *ebnfChoice:
		list := []any{}
		for _, a := range p.alternatives {
			list = append(list, dumpEbnf(a))
		}
		return map[string]any{"$type": "EbnfChoice", "alternatives": list}
	case *ebnfSequence:
		list := []any{}
		for _, e := range p.elements {
			list = append(list, dumpEbnf(e))
		}
		return map[string]any{"$type": "EbnfSequence", "elements": list}
	case *ebnfTerm:
		postfixes := []any{}
		for _, f := range p.postfixes {
			if f.typ == "EbnfExceptionPostfix" {
				postfixes = append(postfixes, map[string]any{"$type": f.typ, "except": dumpEbnf(f.except)})
			} else {
				postfixes = append(postfixes, map[string]any{"$type": f.typ, "operator": f.operator})
			}
		}
		return map[string]any{"$type": "EbnfTerm", "base": dumpEbnf(p.base), "postfixes": postfixes}
	case *ebnfPrimary:
		out := map[string]any{"$type": p.typ}
		switch p.typ {
		case "EbnfTerminal":
			out["value"] = p.value
		case "EbnfNonTerminal":
			out["name"] = p.value
		case "EbnfSpecial":
			out["text"] = p.value
		default:
			out["element"] = dumpEbnf(p.element)
		}
		return out
	}
	panic("no such part")
}

func dumpAbnf(part abnfPart) any {
	switch p := part.(type) {
	case *abnfAlternation:
		list := []any{}
		for _, a := range p.alternatives {
			list = append(list, dumpAbnf(a))
		}
		return map[string]any{"$type": "AbnfAlternation", "alternatives": list}
	case *abnfConcatenation:
		list := []any{}
		for _, e := range p.elements {
			list = append(list, dumpAbnf(e))
		}
		return map[string]any{"$type": "AbnfConcatenation", "elements": list}
	case *abnfElement:
		out := map[string]any{"$type": "AbnfElement", "primary": dumpAbnf(p.primary)}
		if p.hasRepeat {
			out["repeat"] = p.repeat
		}
		return out
	case *abnfPrimary:
		out := map[string]any{"$type": p.typ}
		switch p.typ {
		case "AbnfStringLiteral", "AbnfNumVal":
			out["value"] = p.value
		case "AbnfRuleName":
			out["name"] = p.value
		default:
			out["element"] = dumpAbnf(p.element)
		}
		return out
	}
	panic("no such part")
}

func dumpPeg(part pegPart) any {
	switch p := part.(type) {
	case *pegChoice:
		list := []any{}
		for _, a := range p.alternatives {
			list = append(list, dumpPeg(a))
		}
		return map[string]any{"$type": "PegOrderedChoice", "alternatives": list}
	case *pegSequence:
		list := []any{}
		for _, e := range p.elements {
			list = append(list, dumpPeg(e))
		}
		return map[string]any{"$type": "PegSequence", "elements": list}
	case *pegPrefix:
		out := map[string]any{"$type": "PegPrefix", "suffix": dumpPeg(p.suffix)}
		if p.operator != "" {
			out["operator"] = p.operator
		}
		return out
	case *pegSuffix:
		out := map[string]any{"$type": "PegSuffix", "primary": dumpPeg(p.primary)}
		if p.operator != "" {
			out["operator"] = p.operator
		}
		return out
	case *pegPrimary:
		out := map[string]any{"$type": p.typ}
		switch p.typ {
		case "PegLiteral":
			out["value"] = p.value
		case "PegIdentifier":
			out["name"] = p.value
		case "PegAny":
			out["dot"] = p.value
		default:
			out["element"] = dumpPeg(p.element)
		}
		return out
	}
	panic("no such part")
}

// direct parses a text with the parser of a notation as Pele's does, and gives its table and
// its tree as the oracle writes it.
func direct(src string, notation string) (*common.Table, any, error) {
	switch notation {
	case "ebnf":
		ast, err := parseEbnf(src)
		if err != nil {
			return ebnfTable, nil, err
		}
		return ebnfTable, dumpGrammar(ast, dumpEbnf), nil
	case "abnf":
		ast, err := parseAbnf(src)
		if err != nil {
			return abnfTable, nil, err
		}
		return abnfTable, dumpGrammar(ast, dumpAbnf), nil
	case "peg":
		ast, err := parsePeg(src)
		if err != nil {
			return pegTable, nil, err
		}
		return pegTable, dumpGrammar(ast, dumpPeg), nil
	}
	ast, err := parseIr(src)
	if err != nil {
		return irTable, nil, err
	}
	return irTable, dumpGrammar(ast, func(e *irExpression) any { return dumpIr(e) }), nil
}

func dumpTokens(src string, table *common.Table) [][2]string {
	toks, err := common.Tokenize(src, table, "railroad")
	if err != nil {
		return nil
	}
	out := [][2]string{}
	for i, k := range toks.Kinds[:len(toks.Kinds)-1] {
		out = append(out, [2]string{table.Types[k].Name, src[toks.Starts[i]:toks.Ends[i]]})
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
	data, err := fixture.Read("testdata/pele/railroad.json")
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
	if len(cases) < 8000 {
		t.Fatalf("only %d cases", len(cases))
	}
	failed := 0
	notations := map[string]int{}
	for _, c := range cases {
		table, tree, err := direct(c.Src, c.Direct.Notation)
		ok := compare(t, "tokens", c.Src, dumpTokens(c.Src, table), c.Direct.Tokens)
		switch {
		case err != nil && c.Direct.Error == nil:
			t.Errorf("parse %s (%q): %v, Pele had none", c.Direct.Notation, c.Src, err)
			ok = false
		case err != nil:
			ok = compare(t, "parse error", c.Src, dumpError(err), c.Direct.Error) && ok
		case c.Direct.Error != nil:
			t.Errorf("parse %s (%q): no error, Pele %v", c.Direct.Notation, c.Src, c.Direct.Error)
			ok = false
		case c.Direct.AST != nil:
			ok = compare(t, "tree", c.Src, tree, c.Direct.AST) && ok
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
		case p.Type == diagram.Railroad:
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
				ok = compare(t, "model", c.Src, dumpModel(m), c.Model) && ok
				notations[string(m.Notation)]++
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
	// Each notation's models must be common, or the comparison says little of it.
	for _, n := range []string{"railroad", "ebnf", "abnf", "peg"} {
		if notations[n] < 400 {
			t.Errorf("only %d models in %s", notations[n], n)
		}
	}
}

func TestGetRuleGivesTheLastDefinition(t *testing.T) {
	m, err := Parse("railroad-ebnf-beta\nr = a ;\nr = b ;\n", "")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := m.GetRule("r")
	if !ok || r.Definition.Text != "b" || len(m.Rules) != 2 {
		t.Errorf("GetRule: %v %v, %d rules", r, ok, len(m.Rules))
	}
	if _, ok := m.GetRule("s"); ok {
		t.Error("GetRule found a rule never defined")
	}
}
