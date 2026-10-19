package info

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in infoTokens.
func TestTokenKinds(t *testing.T) {
	table := infoTokens
	if len(table) != 10 {
		t.Fatalf("infoTokens has %d types, want 10", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tShowInfo, "showInfo"},
		{tInfo, "info"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
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
