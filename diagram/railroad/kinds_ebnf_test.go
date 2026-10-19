package railroad

import "testing"

// TestTokenKindsEBNF checks that the kinds of token, numbered by hand, are the indexes of
// their types in ebnfTokens.
func TestTokenKindsEBNF(t *testing.T) {
	table := ebnfTokens
	if len(table) != 28 {
		t.Fatalf("ebnfTokens has %d types, want 28", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{ebnfKeyword, "railroad-ebnf-beta"},
		{ebnfDefines, "::="},
		{ebnfEquals, "="},
		{ebnfSemicolon, ";"},
		{ebnfBar, "|"},
		{ebnfComma, ","},
		{ebnfGroup, "("},
		{ebnfGroupEnd, ")"},
		{ebnfOption, "["},
		{ebnfOptionEnd, "]"},
		{ebnfRepeat, "{"},
		{ebnfRepeatEnd, "}"},
		{ebnfQuestion, "?"},
		{ebnfStar, "*"},
		{ebnfPlus, "+"},
		{ebnfMinus, "-"},
		{ebnfTitle, "TITLE"},
		{ebnfAccTitle, "ACC_TITLE"},
		{ebnfAccDescr, "ACC_DESCR"},
		{ebnfID, "EBNF_ID"},
		{ebnfString, "EBNF_STRING"},
		{ebnfSpecial, "EBNF_SPECIAL_SEQUENCE"},
		{ebnfYAML, "EBNF_YAML"},
		{ebnfISOComment, "EBNF_ISO_COMMENT"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
