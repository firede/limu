package gitgraph

import "testing"

// TestTokenKinds checks that the kinds of token, numbered by hand, are the indexes of
// their types in gitTokens.
func TestTokenKinds(t *testing.T) {
	table := gitTokens
	if len(table) != 32 {
		t.Fatalf("gitTokens has %d types, want 32", len(table))
	}
	for _, c := range []struct {
		kind int
		name string
	}{
		{tCherryPick, "cherry-pick"},
		{tGitGraphColon, "gitGraph:"},
		{tHighlight, "HIGHLIGHT"},
		{tGitGraph, "gitGraph"},
		{tCheckout, "checkout"},
		{tReverse, "REVERSE"},
		{tParent, "parent:"},
		{tCommit, "commit"},
		{tNormal, "NORMAL"},
		{tBranch, "branch"},
		{tOrder, "order:"},
		{tSwitch, "switch"},
		{tType, "type:"},
		{tMerge, "merge"},
		{tMsg, "msg:"},
		{tTag, "tag:"},
		{tID, "id:"},
		{tLR, "LR"},
		{tTB, "TB"},
		{tBT, "BT"},
		{tColon, ":"},
		{tAccDescr, "ACC_DESCR"},
		{tAccTitle, "ACC_TITLE"},
		{tTitle, "TITLE"},
		{tInt, "INT"},
		{tString, "STRING"},
		{tNewline, "NEWLINE"},
		{tWhitespace, "WHITESPACE"},
		{tYAML, "YAML"},
		{tDirective, "DIRECTIVE"},
		{tComment, "SINGLE_LINE_COMMENT"},
		{tReference, "REFERENCE"},
	} {
		if got := table[c.kind].Name; got != c.name {
			t.Errorf("kind %d is %q, want %q", c.kind, got, c.name)
		}
	}
}
