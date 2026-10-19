package radar

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in radarTokens.
func TestTokenKinds(t *testing.T) {
	table := radarTokens(&lexer{})
	if len(table) != 28 {
		t.Fatalf("radarTokens has %d types, want 28", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tRadarColon, "radar-beta:"},
		{tRadar, "radar-beta"},
		{tShowLegend, "showLegend"},
		{tGraticule, "graticule"},
		{tCurve, "curve"},
		{tTicks, "ticks"},
		{tAxis, "axis"},
		{tMax, "max"},
		{tMin, "min"},
		{tColon, ":"},
		{tComma, ","},
		{tOpen, "["},
		{tClose, "]"},
		{tOpenBrace, "{"},
		{tCloseBrace, "}"},
		{tGraticuleValue, "GRATICULE"},
		{tBoolean, "BOOLEAN"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tNumber, "NUMBER"},
		{tString, "STRING"},
		{tID, "ID"},
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
