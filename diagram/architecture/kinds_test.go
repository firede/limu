package architecture

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in archTokens.
func TestTokenKinds(t *testing.T) {
	table := archTokens
	if len(table) != 26 {
		t.Fatalf("archTokens has %d types, want 26", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tHeader, "architecture-beta"},
		{tJunction, "junction"},
		{tService, "service"},
		{tColumn, "column"},
		{tGroup, "group"},
		{tAlign, "align"},
		{tRow, "row"},
		{tDashes, "--"},
		{tIn, "in"},
		{tColon, ":"},
		{tDash, "-"},
		{tDirection, "ARROW_DIRECTION"},
		{tArrowGroup, "ARROW_GROUP"},
		{tInto, "ARROW_INTO"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tString, "STRING"},
		{tID, "ID"},
		{tNewline, "NEWLINE"},
		{tWhitespace, "WHITESPACE"},
		{tYAML, "YAML"},
		{tDirective, "DIRECTIVE"},
		{tComment, "SINGLE_LINE_COMMENT"},
		{tIcon, "ARCH_ICON"},
		{tLabel, "ARCH_TITLE"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
