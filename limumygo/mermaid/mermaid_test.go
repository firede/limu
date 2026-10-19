package mermaid

import (
	"slices"
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
		editor.TokenMeta: "meta", editor.TokenPunctuation: "p", editor.TokenVariable: "var",
		editor.TokenProperty: "prop", editor.TokenFunction: "fn",
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
		{"A|var", "[|p", "The|str", "end|str", "]|p", "-->|op", "||p", "yes|str", "||p", "B|var", "(|p", `"quoted"|str`, ")|p"},
		{"subgraph|kw", "S|var"},
		{"C|var", "==>|op", "D|var", "-.->|op", "E|var"},
		{"end|kw"},
		{"style|kw", "A|var", "fill|prop", ":|p", "#f9f|const", ",|p", "stroke-width|prop", ":|p", "2px|num"},
		// commit is a keyword of git graphs, not of flowcharts; a semicolon starts a statement.
		{"A|var", "-->|op", "commit|var", ";|p", "click|kw", "A|var"},
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
		{"participant|kw", "A|var", "as|kw", "Alice|var"},
		{"Note|kw", "right|kw", "of|kw", "A|var", ":|p", "we|str", "end|str", "at|str", "3|num"},
		{"loop|kw", "Every|var", "minute|var"},
		{"end|kw"},
	})
	check(t, []string{
		"gitGraph",
		"  commit id: \"a\"",
		"  branch develop",
		"  loop",
	}, [][]string{
		{"gitGraph|kw"},
		{"commit|kw", "id|var", ":|p", `"a"|str`},
		{"branch|kw", "develop|var"},
		{"loop|var"},
	})
	check(t, []string{
		"xychart",
		"  x-axis [jan, feb] ",
		"  y-axis 0 --> 100",
	}, [][]string{
		{"xychart|kw"},
		{"x-axis|kw", "[|p", "jan|str", ",|p", "feb|str", "]|p"},
		{"y-axis|kw", "0|num", "-->|op", "100|num"},
	})
}

// Names are variables and labels strings, as the website highlights them: a class given
// with :::, the settings of a node, the declarations of a style, and the links of class and
// entity relationship diagrams.
func TestAsTheWebsite(t *testing.T) {
	check(t, []string{
		"flowchart LR",
		"  draft[Draft] --> review[Review]:::warn --o done@{ shape: rounded, label: \"Done\" }",
		"  classDef warn fill:#e3a040,stroke:#b0611a",
		"  linkStyle 0 stroke-dasharray:4 4,stroke:rgb(1 2 3)",
	}, [][]string{
		{"flowchart|kw", "LR|const"},
		{"draft|var", "[|p", "Draft|str", "]|p", "-->|op", "review|var", "[|p", "Review|str", "]:::|p", "warn|kw",
			"--o|op", "done|var", "@{|p", "shape|prop", ":|p", "rounded|str", ",|p", "label|prop", ":|p", `"Done"|str`, "}|p"},
		{"classDef|kw", "warn|var", "fill|prop", ":|p", "#e3a040|const", ",|p", "stroke|prop", ":|p", "#b0611a|const"},
		{"linkStyle|kw", "0|num", "stroke-dasharray|prop", ":|p", "4|num", "4|num", ",|p", "stroke|prop", ":|p", "rgb|fn", "(|p", "1|num", "2|num", "3|num", ")|p"},
	})
	check(t, []string{
		"classDiagram",
		"  Animal <|-- Duck",
	}, [][]string{
		{"classDiagram|kw"},
		{"Animal|var", "<|--|op", "Duck|var"},
	})
	check(t, []string{
		"erDiagram",
		"  CUSTOMER ||--o{ ORDER : places",
	}, [][]string{
		{"erDiagram|kw"},
		{"CUSTOMER|var", "||--o{|op", "ORDER|var", ":|p", "places|str"},
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

// keywords.json names only types limu reads, so that a type's words are not lost to a
// misspelling.
func TestKeywordsFile(t *testing.T) {
	f := file()
	for _, m := range []map[string]string{f.Starts, f.Inline} {
		for typ := range m {
			if !slices.Contains(limu.Types(), diagram.Type(typ)) {
				t.Errorf("keywords.json has words of %q, which is no type limu reads", typ)
			}
		}
	}
	if !directions()["LR"] || !keywordsOf(diagram.Sequence).inline["over"] {
		t.Error("keywords.json was not read")
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
