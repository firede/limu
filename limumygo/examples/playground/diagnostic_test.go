package main

import (
	"errors"
	"testing"

	"github.com/firede/limu"
	"github.com/firede/limu/diagram"
)

func TestProblemOf(t *testing.T) {
	cases := []struct {
		src, status, under string
	}{
		{
			"sequenceDiagram\n  Alice->>: Hi",
			"Line 2, column 11: Parse error: Expecting 'ACTOR', got 'TXT'", ":",
		},
		// Front matter, comments and directives are not in the text the parser reads, and
		// the line is found in the text the editor has.
		{
			"---\ntitle: x\n---\n%%{init: {}}%%\nflowchart LR\n%% a comment\n  A{{ --> }",
			"Line 7, column 11: Parse error: Expecting 'DIAMOND_STOP', got 'TEXT'", "}",
		},
		// An error at the end of the text is at the end of its last line.
		{
			"flowchart LR\n  A -->\n\n",
			"Line 2, column 7: Parse error: Expecting 'NODE_STRING', got 'EOF'", ">",
		},
		// A column counts UTF-16 code units, and the underline whole characters.
		{
			"pie\n  \"😀é\": x",
			"Line 2, column 10: Lexical error. Unrecognized text.", "x",
		},
	}
	for _, tc := range cases {
		_, err := limu.Parse(tc.src, diagram.Options{})
		if err == nil {
			t.Fatalf("%q parses", tc.src)
		}
		p := problemOf(tc.src, err)
		if p.Status != tc.status || !p.Located || tc.src[p.Start:p.End] != tc.under {
			t.Errorf("%q:\n got %q, %q (located %v)\nwant %q, %q", tc.src, p.Status, tc.src[p.Start:p.End], p.Located, tc.status, tc.under)
		}
		if p.Detail != err.Error() {
			t.Errorf("%q: detail %q", tc.src, p.Detail)
		}
	}
	// An error with no place is told, and underlines nothing.
	_, err := limu.Parse("not a diagram", diagram.Options{})
	if p := problemOf("not a diagram", err); p.Located || p.Status != "No diagram type detected for the given text." {
		t.Errorf("unsupported: %+v", p)
	}
	if p := problemOf("x", errors.New("plain")); p.Located || p.Status != "plain" {
		t.Errorf("plain error: %+v", p)
	}
}

func TestColumns(t *testing.T) {
	s := "a😀b"
	for col, at := range map[int]int{1: 0, 2: 1, 4: 5, 5: 6, 9: 6} {
		if got := byteOfColumn(s, col); got != at {
			t.Errorf("byteOfColumn(%d) = %d, want %d", col, got, at)
		}
	}
	if got := columnOfByte(s, 5); got != 4 {
		t.Errorf("columnOfByte(5) = %d", got)
	}
}
