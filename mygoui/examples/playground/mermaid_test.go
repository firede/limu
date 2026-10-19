package main

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui/editor"
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

func TestTokenizeMermaid(t *testing.T) {
	text := []string{
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
		"  x-axis 0 --> 100",
		"  Alice->>Bob: we end at 3",
	}
	want := [][]string{
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
		{"x-axis|kw", "0|num", "-->|op", "100|num"},
		{"->>|op", ":|p", "3|num"},
	}
	var state uint64
	for i, line := range text {
		var toks []editor.Token
		toks, state = tokenizeMermaid(line, state, nil)
		if n := len(toks); n == 0 || toks[n-1].End != len(line) {
			t.Errorf("line %d: tokens %v do not reach its end", i, toks)
		}
		if got := spans(line, toks); strings.Join(got, " ") != strings.Join(want[i], " ") {
			t.Errorf("line %d %q:\n got %q\nwant %q", i, line, got, want[i])
		}
	}
	if state&inFront != 0 || state&declared == 0 {
		t.Errorf("ends in state %b", state)
	}
}

// Front matter opens only on the first line, and a directive may go on over lines.
func TestTokenizeMermaidStates(t *testing.T) {
	_, s := tokenizeMermaid("pie", 0, nil)
	if toks, _ := tokenizeMermaid("---", s, nil); len(toks) != 1 || toks[0].Kind == editor.TokenMeta {
		t.Errorf("--- after the declaration: %v", toks)
	}
	_, s = tokenizeMermaid("%%{init: {", 0, nil)
	toks, s := tokenizeMermaid(`  "theme": "dark" }}%%`, s, nil)
	if len(toks) != 1 || toks[0].Kind != editor.TokenMeta || s&inDirective != 0 {
		t.Errorf("directive's second line: %v, state %b", toks, s)
	}
	toks, _ = tokenizeMermaid("sequenceDiagram", s, nil)
	if len(toks) != 1 || toks[0].Kind != editor.TokenKeyword {
		t.Errorf("declaration after a directive: %v", toks)
	}
}
