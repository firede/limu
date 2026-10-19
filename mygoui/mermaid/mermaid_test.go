package mermaid

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/editor"
	"github.com/firede/limu"
	"github.com/firede/limu/diagram"
)

// spans writes a line's tokens as text|kind pairs, leaving out plain text and spaces.
func spans(line string, toks []editor.Token) []string {
	names := map[editor.TokenKind]string{
		editor.TokenKeyword: "kw", editor.TokenString: "str", editor.TokenNumber: "num",
		editor.TokenComment: "com", editor.TokenOperator: "op", editor.TokenConstant: "const",
		editor.TokenMeta: "meta", editor.TokenPunctuation: "p",
	}
	var out []string
	start := 0
	for _, tk := range toks {
		if s := strings.TrimSpace(line[start:tk.End]); s != "" && tk.Kind != editor.TokenText {
			out = append(out, s+"|"+names[tk.Kind])
		}
		start = tk.End
	}
	return out
}

// check tokenizes text a line at a time and compares each line's spans with want.
func check(t *testing.T, text []string, want [][]string) uint64 {
	t.Helper()
	var state uint64
	for i, line := range text {
		var toks []editor.Token
		toks, state = Tokenize(line, state, nil)
		if n := len(toks); n == 0 || toks[n-1].End != len(line) {
			t.Errorf("line %d: tokens %v do not reach its end", i, toks)
		}
		if got := spans(line, toks); strings.Join(got, " ") != strings.Join(want[i], " ") {
			t.Errorf("line %d %q:\n got %q\nwant %q", i, line, got, want[i])
		}
	}
	return state
}

func TestFlowchart(t *testing.T) {
	state := check(t, []string{
		"---",
		"title: Order",
		"---",
		"%%{init: {\"theme\": \"dark\"}}%%",
		"%% a comment",
		"flowchart LR",
		"  A[The end] -->|yes| B(\"quoted\")",
		"  subgraph S",
		"    C ==> D -.-> E",
		"  end",
		"  style A fill:#f9f,stroke-width:2px",
		"  A --> commit; click A",
	}, [][]string{
		{"---|meta"},
		{"title: Order|meta"},
		{"---|meta"},
		{`%%{init: {"theme": "dark"}}%%|meta`},
		{"%% a comment|com"},
		{"flowchart|kw", "LR|const"},
		{"[|p", "]|p", "-->|op", "||p", "||p", "(|p", `"quoted"|str`, ")|p"},
		{"subgraph|kw"},
		{"==>|op", "-.->|op"},
		{"end|kw"},
		{"style|kw", ":|p", "#f9f|const", ",|p", ":|p", "2|num"},
		// commit is a keyword of git graphs, not of flowcharts; a semicolon starts a statement.
		{"-->|op", ";|p", "click|kw"},
	})
	if state&inFront != 0 || state&declared == 0 || typeOf(state) != diagram.Flowchart {
		t.Errorf("ends in state %b, type %q", state, typeOf(state))
	}
}

// Each type's keywords are its own: a word is a keyword where it starts a statement of a
// type that has it, and the words of a sequence's notes are keywords anywhere in them.
func TestKeywordsOfTheType(t *testing.T) {
	check(t, []string{
		"sequenceDiagram",
		"  participant A as Alice",
		"  Note right of A: we end at 3",
		"  loop Every minute",
		"  end",
	}, [][]string{
		{"sequenceDiagram|kw"},
		{"participant|kw", "as|kw"},
		{"Note|kw", "right|kw", "of|kw", ":|p", "3|num"},
		{"loop|kw"},
		{"end|kw"},
	})
	check(t, []string{
		"gitGraph",
		"  commit id: \"a\"",
		"  branch develop",
		"  loop",
	}, [][]string{
		{"gitGraph|kw"},
		{"commit|kw", ":|p", `"a"|str`},
		{"branch|kw"},
		{},
	})
	check(t, []string{
		"xychart",
		"  x-axis [jan, feb] ",
		"  y-axis 0 --> 100",
	}, [][]string{
		{"xychart|kw"},
		{"x-axis|kw", "[|p", ",|p", "]|p"},
		{"y-axis|kw", "0|num", "-->|op", "100|num"},
	})
}

// Every type limu reads has its keywords, and its declaration is read as that type.
func TestEveryType(t *testing.T) {
	declarations := map[diagram.Type]string{
		diagram.Flowchart: "graph TD", diagram.Sequence: "sequenceDiagram", diagram.Class: "classDiagram",
		diagram.State: "stateDiagram-v2", diagram.ER: "erDiagram", diagram.Pie: "pie", diagram.Gantt: "gantt",
		diagram.GitGraph: "gitGraph", diagram.Mindmap: "mindmap", diagram.Timeline: "timeline",
		diagram.Journey: "journey", diagram.XYChart: "xychart-beta", diagram.QuadrantChart: "quadrantChart",
		diagram.Requirement: "requirementDiagram", diagram.Sankey: "sankey-beta", diagram.Block: "block-beta",
		diagram.Packet: "packet-beta", diagram.Kanban: "kanban", diagram.Architecture: "architecture-beta",
		diagram.Radar: "radar-beta", diagram.Treemap: "treemap-beta", diagram.Info: "info",
		diagram.AgentFlow: "agentflow-beta", diagram.UseCase: "usecase-beta", diagram.Railroad: "railroad-beta",
		diagram.TreeView: "treeView-beta", diagram.Venn: "venn-beta", diagram.Cynefin: "cynefin-beta",
		diagram.Wardley: "wardley-beta", diagram.Swimlane: "swimlane-beta", diagram.Ishikawa: "ishikawa-beta",
		diagram.EventModeling: "eventmodeling", diagram.C4: "C4Context",
	}
	for _, typ := range limu.Types() {
		d, ok := declarations[typ]
		if !ok {
			t.Errorf("no declaration of %s in the test", typ)
			continue
		}
		if _, s := Tokenize(d, begun, nil); typeOf(s) != typ {
			t.Errorf("%q is read as %q, not %q", d, typeOf(s), typ)
		}
		if k := keywordsOf(typ); !k.starts["accTitle"] {
			t.Errorf("%s has no keywords", typ)
		}
	}
}

// Front matter opens only on the first line, and a directive may go on over lines.
func TestStates(t *testing.T) {
	_, s := Tokenize("pie", 0, nil)
	if toks, _ := Tokenize("---", s, nil); len(toks) != 1 || toks[0].Kind == editor.TokenMeta {
		t.Errorf("--- after the declaration: %v", toks)
	}
	_, s = Tokenize("%%{init: {", 0, nil)
	toks, s := Tokenize(`  "theme": "dark" }}%%`, s, nil)
	if len(toks) != 1 || toks[0].Kind != editor.TokenMeta || s&inDirective != 0 {
		t.Errorf("directive's second line: %v, state %b", toks, s)
	}
	toks, s = Tokenize("sequenceDiagram", s, nil)
	if len(toks) != 1 || toks[0].Kind != editor.TokenKeyword || typeOf(s) != diagram.Sequence {
		t.Errorf("declaration after a directive: %v, type %q", toks, typeOf(s))
	}
}
