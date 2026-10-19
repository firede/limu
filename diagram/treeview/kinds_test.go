package treeview

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in treeViewTokens.
func TestTokenKinds(t *testing.T) {
	table := treeViewTokens
	if len(table) != 13 {
		t.Fatalf("treeViewTokens has %d types, want 13", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tKeyword, "treeView-beta"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tClassAnnotation, "CLASS_ANNOTATION"},
		{tIconAnnotation, "ICON_ANNOTATION"},
		{tDescAnnotation, "DESC_ANNOTATION"},
		{tIndentation, "INDENTATION"},
		{tQuotedName, "QUOTED_NAME"},
		{tWS, "WS"},
		{tComment, "ML_COMMENT"},
		{tNewline, "NL"},
		{tBareName, "BARE_NAME"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
