package pie

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in pieTokens.
func TestTokenKinds(t *testing.T) {
	table := pieTokens
	if len(table) != 13 {
		t.Fatalf("pieTokens has %d types, want 13", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tShowData, "showData"},
		{tPie, "pie"},
		{tColon, ":"},
		{tNumber, "NUMBER_PIE"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tString, "STRING"},
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
