package railroad

import "testing"

// TestTokenKindsPEG checks that the kinds of token, numbered by hand, are the indexes of
// their types in pegTokens.
func TestTokenKindsPEG(t *testing.T) {
	table := pegTokens
	if len(table) != 22 {
		t.Fatalf("pegTokens has %d types, want 22", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{pegKeyword, "railroad-peg-beta"},
		{pegArrow, "<-"},
		{pegSemicolon, ";"},
		{pegSlash, "/"},
		{pegAnd, "&"},
		{pegNot, "!"},
		{pegQuestion, "?"},
		{pegStar, "*"},
		{pegPlus, "+"},
		{pegGroup, "("},
		{pegGroupEnd, ")"},
		{pegDot, "."},
		{pegTitle, "TITLE"},
		{pegAccTitle, "ACC_TITLE"},
		{pegAccDescr, "ACC_DESCR"},
		{pegID, "PEG_ID"},
		{pegString, "PEG_STRING"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
