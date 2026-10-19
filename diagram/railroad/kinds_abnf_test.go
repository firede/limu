package railroad

import "testing"

// TestTokenKindsABNF checks that the kinds of token, numbered by hand, are the indexes of
// their types in abnfTokens.
func TestTokenKindsABNF(t *testing.T) {
	table := abnfTokens
	if len(table) != 21 {
		t.Fatalf("abnfTokens has %d types, want 21", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{abnfKeyword, "railroad-abnf-beta"},
		{abnfEquals, "="},
		{abnfSemicolon, ";"},
		{abnfSlash, "/"},
		{abnfGroup, "("},
		{abnfGroupEnd, ")"},
		{abnfOption, "["},
		{abnfOptionEnd, "]"},
		{abnfTitle, "TITLE"},
		{abnfAccTitle, "ACC_TITLE"},
		{abnfAccDescr, "ACC_DESCR"},
		{abnfName, "ABNF_RULENAME"},
		{abnfString, "ABNF_STRING"},
		{abnfNumber, "ABNF_NUMVAL"},
		{abnfRepeat, "ABNF_REPEAT"},
		{abnfExact, "ABNF_EXACT_REPEAT"},
		{abnfComment, "ABNF_COMMENT"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
