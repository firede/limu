package packet

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in packetTokens.
func TestTokenKinds(t *testing.T) {
	table := packetTokens
	if len(table) != 15 {
		t.Fatalf("packetTokens has %d types, want 15", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tPacketBeta, "packet-beta"},
		{tPacket, "packet"},
		{tDash, "-"},
		{tPlus, "+"},
		{tColon, ":"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tInt, "INT"},
		{tString, "STRING"},
		{tNewline, "NEWLINE"},
		{tWhitespace, "WHITESPACE"},
		{tYAML, "YAML"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
