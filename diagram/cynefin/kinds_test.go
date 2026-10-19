package cynefin

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in cynefinTokens.
func TestTokenKinds(t *testing.T) {
	table := cynefinTokens
	if len(table) != 14 {
		t.Fatalf("cynefinTokens has %d types, want 14", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tKeywordColon, "cynefin-beta:"},
		{tKeyword, "cynefin-beta"},
		{tArrow, "-->"},
		{tColon, ":"},
		{tDomain, "DOMAIN_NAME"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tString, "STRING"},
		{tNewline, "NEWLINE"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
