package railroad

import "testing"

// TestTokenKindsIR checks that the kinds of token, numbered by hand, are the indexes of
// their types in irTokens.
func TestTokenKindsIR(t *testing.T) {
	table := irTokens
	if len(table) != 24 {
		t.Fatalf("irTokens has %d types, want 24", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{irKeyword, "railroad-beta"},
		{irNonterminal, "nonterminal"},
		{irZeroOrMore, "zeroOrMore"},
		{irOneOrMore, "oneOrMore"},
		{irSequence, "sequence"},
		{irOptional, "optional"},
		{irTerminal, "terminal"},
		{irSpecial, "special"},
		{irChoice, "choice"},
		{irEquals, "="},
		{irSemicolon, ";"},
		{irOpen, "("},
		{irComma, ","},
		{irClose, ")"},
		{irTitle, "TITLE"},
		{irAccTitle, "ACC_TITLE"},
		{irAccDescr, "ACC_DESCR"},
		{irID, "RR_ID"},
		{irString, "RR_STRING"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
