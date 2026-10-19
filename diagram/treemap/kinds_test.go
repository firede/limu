package treemap

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in treemapTokens.
func TestTokenKinds(t *testing.T) {
	table := treemapTokens
	if len(table) != 15 {
		t.Fatalf("treemapTokens has %d types, want 15", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tKeyword, "TREEMAP_KEYWORD"},
		{tClassDef, "CLASS_DEF"},
		{tStyleSeparator, "STYLE_SEPARATOR"},
		{tSeparator, "SEPARATOR"},
		{tComma, "COMMA"},
		{tIndentation, "INDENTATION"},
		{tWhitespace, "WS"},
		{tComment, "ML_COMMENT"},
		{tNewline, "NL"},
		{tID, "ID2"},
		{tNumber, "NUMBER2"},
		{tString, "STRING2"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
