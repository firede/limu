package wardley

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in wardleyTokens.
func TestTokenKinds(t *testing.T) {
	table := wardleyTokens
	if len(table) != 43 {
		t.Fatalf("wardleyTokens has %d types, want 43", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tLBracket, "["},
		{tComma, ","},
		{tRBracket, "]"},
		{tAt, "@"},
		{tSlash, "/"},
		{tLParen, "("},
		{tRParen, ")"},
		{tMinus, "-"},
		{tLBrace, "{"},
		{tRBrace, "}"},
		{tNumber, "WARDLEY_NUMBER"},
		{tArrow, "ARROW"},
		{tLinkPort, "LINK_PORT"},
		{tLinkArrow, "LINK_ARROW"},
		{tLinkLabel, "LINK_LABEL"},
		{tStrategy, "STRATEGY"},
		{tWardley, "KW_WARDLEY"},
		{tSize, "KW_SIZE"},
		{tEvolution, "KW_EVOLUTION"},
		{tAnchor, "KW_ANCHOR"},
		{tComponent, "KW_COMPONENT"},
		{tLabel, "KW_LABEL"},
		{tInertia, "KW_INERTIA"},
		{tEvolve, "KW_EVOLVE"},
		{tPipeline, "KW_PIPELINE"},
		{tNote, "KW_NOTE"},
		{tAnnotations, "KW_ANNOTATIONS"},
		{tAnnotation, "KW_ANNOTATION"},
		{tAccelerator, "KW_ACCELERATOR"},
		{tDeaccelerator, "KW_DEACCELERATOR"},
		{tName, "NAME_WITH_SPACES"},
		{tWS, "WS"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tInt, "INT"},
		{tString, "STRING"},
		{tID, "ID"},
		{tNewline, "NEWLINE"},
		{tWhitespace, "WHITESPACE"},
		{tYAML, "YAML"},
		{tDirective, "DIRECTIVE"},
		{tComment, "SINGLE_LINE_COMMENT"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
