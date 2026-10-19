package gitgraph

import (
	"strings"

	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The token types of Pele's GIT_TOKENS, in its order, which is Mermaid's: where two match at
// the same place, the first one listed wins.
const (
	tCherryPick = iota
	tGitGraphColon
	tHighlight
	tGitGraph
	tCheckout
	tReverse
	tParent
	tCommit
	tNormal
	tBranch
	tOrder
	tSwitch
	tType
	tMerge
	tMsg
	tTag
	tID
	tLR
	tTB
	tBT
	tColon
	tAccDescr
	tAccTitle
	tTitle
	tInt
	tString
	tNewline
	tWhitespace
	tYAML
	tDirective
	tComment
	tReference

	// tEnd is the kind of the token after the last, at the end of the text.
	tEnd = -1
)

// referenceToken is Pele's REFERENCE, /\w([-\./\w]*[-\w])?/y: a name, which may hold dots
// and slashes but does not end in one.
var referenceToken = common.TokenType{Name: "REFERENCE", Match: func(s string, p int) int {
	if !chars.IsWord(int(s[p])) {
		return -1
	}
	e := p + 1
	for e < len(s) && (chars.IsWord(int(s[e])) || s[e] == '-' || s[e] == '.' || s[e] == '/') {
		e++
	}
	for e > p+1 && (s[e-1] == '.' || s[e-1] == '/') {
		e--
	}
	return e
}}

// word is Pele's word: a keyword that is the start of a longer name is part of that name, so
// `commits` is a branch.
func word(name string) common.TokenType {
	return common.TokenType{Name: name, Literal: name, Longer: []int{tReference}}
}

// gitGraphToken is Pele's keyword('gitGraph'), which also gives way to a longer name.
var gitGraphToken = func() common.TokenType {
	t := common.Keyword("gitGraph")
	t.Longer = []int{tReference}
	return t
}()

// accDescrToken is Pele's ACC_DESCR with an opener, /[\t ]*accDescr\s*{/y: without a closing
// brace the block form scans to the end of the text, and a brace is not a token on its own,
// so the first failure is final.
var accDescrToken = func() common.TokenType {
	t := common.AccDescrToken
	t.Opener = func(s string, at int) bool {
		i := at
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if !strings.HasPrefix(s[i:], "accDescr") {
			return false
		}
		i = chars.SpaceRun(s, i+len("accDescr"))
		return i < len(s) && s[i] == '{'
	}
	return t
}()

// gitTokens is Pele's GIT_TOKENS.
var gitTokens = []common.TokenType{
	word("cherry-pick"),
	{Name: "gitGraph:", Literal: "gitGraph:"},
	word("HIGHLIGHT"),
	gitGraphToken,
	word("checkout"),
	word("REVERSE"),
	{Name: "parent:", Literal: "parent:"},
	word("commit"),
	word("NORMAL"),
	word("branch"),
	{Name: "order:", Literal: "order:"},
	word("switch"),
	{Name: "type:", Literal: "type:"},
	word("merge"),
	{Name: "msg:", Literal: "msg:"},
	{Name: "tag:", Literal: "tag:"},
	{Name: "id:", Literal: "id:"},
	word("LR"),
	word("TB"),
	word("BT"),
	{Name: ":", Literal: ":"},
	accDescrToken,
	common.AccTitleToken,
	common.TitleToken,
	common.IntToken,
	common.StringToken,
	common.NewlineToken,
	common.WhitespaceToken,
	common.YAMLToken,
	common.DirectiveToken,
	common.CommentToken,
	referenceToken,
}

var gitTable = common.Prepare(gitTokens)
