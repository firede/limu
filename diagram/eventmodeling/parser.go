package eventmodeling

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// words are the keywords of Pele's WORDS, each of which an identifier may also start with.
var words = []string{
	"resetframe", "readmodel", "processor", "timeframe", "command", "entity", "event", "jsobj", "figma", "given", "json", "salt", "html",
	"text", "data", "note", "when", "then", "rmo", "cmd", "evt", "pcr", "uri",
}

// tokenNames are the names of the types of Pele's EVENTMODELING_TOKENS, in order, so that
// each type's index is known before the table is made.
var tokenNames = func() []string {
	names := []string{"eventmodeling"}
	names = append(names, words...)
	return append(names, "->>", "gwt", "ui", "md", "tf", "[[", "]]", "rf", "`", ".", "EM_ID", "EM_FID", "EM_DATA_INLINE",
		"EM_DATA_BLOCK", "EM_ACC_DESCR", "EM_ACC_TITLE", "EM_TITLE", "EM_WS", "EM_YAML", "EM_DIRECTIVE", "EM_SINGLE_LINE_COMMENT",
		"EM_ML_COMMENT", "EM_SL_COMMENT")
}()

// index is the index of a type in the table, as Pele's index finds it.
func index(name string) int {
	for k, n := range tokenNames {
		if n == name {
			return k
		}
	}
	panic("no token type " + name)
}

// The kinds of token the parser reads, as Pele's T.
var (
	tKeyword  = 0
	tArrow    = index("->>")
	tOpen     = index("[[")
	tClose    = index("]]")
	tTick     = index("`")
	tDot      = index(".")
	tID       = index("EM_ID")
	tFrameID  = index("EM_FID")
	tInline   = index("EM_DATA_INLINE")
	tBlock    = index("EM_DATA_BLOCK")
	tAccDescr = index("EM_ACC_DESCR")
	tAccTitle = index("EM_ACC_TITLE")
	tTitle    = index("EM_TITLE")
	tEntity   = index("entity")
	tData     = index("data")
	tNote     = index("note")
	tGwt      = index("gwt")
	tGiven    = index("given")
	tWhen     = index("when")
	tThen     = index("then")
)

// kinds is the set of the kinds of token named, as Pele's kinds.
func kinds(names ...string) map[int]bool {
	set := map[int]bool{}
	for _, name := range names {
		set[index(name)] = true
	}
	return set
}

var (
	timeFrame  = kinds("tf", "timeframe")
	resetFrame = kinds("rf", "resetframe")
	entityType = kinds("rmo", "readmodel", "ui", "cmd", "command", "evt", "event", "pcr", "processor")
	dataType   = kinds("json", "jsobj", "figma", "salt", "uri", "md", "html", "text")
)

// isLineEnd reports whether a line terminator, which JavaScript's . does not take, starts at
// byte i of s.
func isLineEnd(s string, i int) bool {
	switch s[i] {
	case '\n', '\r':
		return true
	case 0xe2:
		return strings.HasPrefix(s[i:], " ") || strings.HasPrefix(s[i:], " ")
	}
	return false
}

// lineBreak matches \r?\n at byte i, or gives -1.
func lineBreak(s string, i int) int {
	switch {
	case strings.HasPrefix(s[i:], "\r\n"):
		return i + 2
	case strings.HasPrefix(s[i:], "\n"):
		return i + 1
	}
	return -1
}

// closed matches (?:\r?\n|(?!\S)) at byte i, or gives -1.
func closed(s string, i int) int {
	if end := lineBreak(s, i); end >= 0 {
		return end
	}
	if i == len(s) || chars.IsSpace(chars.Unit(s, i)) {
		return i
	}
	return -1
}

// matchID is EM_ID, /[_a-zA-Z][\w_]*/y.
func matchID(s string, at int) int {
	if c := int(s[at]); c != '_' && !chars.IsLetter(c) {
		return -1
	}
	end := at + 1
	for end < len(s) && chars.IsWord(int(s[end])) {
		end++
	}
	return end
}

// matchFrameID is EM_FID, /\d{1,3}/y.
func matchFrameID(s string, at int) int {
	end := at
	for end < len(s) && end-at < 3 && chars.IsDigit(int(s[end])) {
		end++
	}
	if end == at {
		return -1
	}
	return end
}

// matchInline is EM_DATA_INLINE, /\{(.*)\}|"(.*)"|'(.*)'/y: from an opening brace or quote,
// the greedy .* gives back up to the last closing one on the line.
func matchInline(s string, at int) int {
	var closer byte
	switch s[at] {
	case '{':
		closer = '}'
	case '"', '\'':
		closer = s[at]
	default:
		return -1
	}
	end := at + 1
	for end < len(s) && !isLineEnd(s, end) {
		end++
	}
	if last := strings.LastIndexByte(s[at+1:end], closer); last >= 0 {
		return at + 1 + last + 1
	}
	return -1
}

// blockOpener matches /\{[\t ]*\r?\n/y at byte at, or gives -1.
func blockOpener(s string, at int) int {
	if s[at] != '{' {
		return -1
	}
	i := at + 1
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	return lineBreak(s, i)
}

// matchBlock is EM_DATA_BLOCK, /\{[\t ]*\r?\n(?:[\S\s]*?\r?\n)?\}(?:\r?\n|(?!\S))/y. The
// group is tried first, so the closing brace is the first after a line break, past one that
// follows the opener at once.
func matchBlock(s string, at int) int {
	q := blockOpener(s, at)
	if q < 0 {
		return -1
	}
	for k := q; k < len(s); k++ {
		j := lineBreak(s, k)
		if j < 0 || j == len(s) || s[j] != '}' {
			continue
		}
		if end := closed(s, j+1); end >= 0 {
			return end
		}
	}
	if q < len(s) && s[q] == '}' {
		return closed(s, q+1)
	}
	return -1
}

// matchSpace is EM_WS, /\s+/y.
func matchSpace(s string, at int) int {
	if end := chars.SpaceRun(s, at); end > at {
		return end
	}
	return -1
}

// matchBlockComment is EM_ML_COMMENT, /\/\*[\s\S]*?\*\//y.
func matchBlockComment(s string, at int) int {
	if !strings.HasPrefix(s[at:], "/*") {
		return -1
	}
	if end := strings.Index(s[at+2:], "*/"); end >= 0 {
		return at + 2 + end + 2
	}
	return -1
}

// matchLineComment is EM_SL_COMMENT, /\/\/[^\n\r]*/y.
func matchLineComment(s string, at int) int {
	if !strings.HasPrefix(s[at:], "//") {
		return -1
	}
	end := at + 2
	for end < len(s) && s[end] != '\n' && s[end] != '\r' {
		end++
	}
	return end
}

// renamed is a type of the shared table under the name Pele gives it here.
func renamed(t common.TokenType, name string) common.TokenType {
	t.Name = name
	return t
}

// eventModelingTokens is Pele's EVENTMODELING_TOKENS. A data block and a block comment that
// find no closer are not tried again, as none is further on: a later one could close only
// where the first could.
var eventModelingTokens = func() []common.TokenType {
	keyword := common.Keyword("eventmodeling")
	keyword.Longer = []int{tID}
	word := func(text string) common.TokenType {
		return common.TokenType{Name: text, Literal: text, Longer: []int{tID}}
	}
	sign := func(text string) common.TokenType { return common.TokenType{Name: text, Literal: text} }
	types := []common.TokenType{keyword}
	for _, w := range words {
		types = append(types, word(w))
	}
	types = append(types,
		sign("->>"), word("gwt"), word("ui"), word("md"), word("tf"), sign("[["), sign("]]"), word("rf"), sign("`"), sign("."),
		common.TokenType{Name: "EM_ID", Match: matchID},
		common.TokenType{Name: "EM_FID", Match: matchFrameID},
		common.TokenType{Name: "EM_DATA_INLINE", Match: matchInline},
		common.TokenType{Name: "EM_DATA_BLOCK", Match: matchBlock, Opener: func(s string, at int) bool { return blockOpener(s, at) >= 0 }},
		common.AccDescr("EM_ACC_DESCR"),
		renamed(common.AccTitleToken, "EM_ACC_TITLE"),
		renamed(common.TitleToken, "EM_TITLE"),
		common.TokenType{Name: "EM_WS", Match: matchSpace, Hidden: true},
		renamed(common.YAMLToken, "EM_YAML"),
		renamed(common.DirectiveToken, "EM_DIRECTIVE"),
		renamed(common.CommentToken, "EM_SINGLE_LINE_COMMENT"),
		common.TokenType{Name: "EM_ML_COMMENT", Match: matchBlockComment, Hidden: true,
			Opener: func(s string, at int) bool { return strings.HasPrefix(s[at:], "/*") }},
		common.TokenType{Name: "EM_SL_COMMENT", Match: matchLineComment, Hidden: true},
	)
	for k, t := range types {
		if t.Name != tokenNames[k] {
			panic("token type " + t.Name + " out of place")
		}
	}
	return types
}()

var eventModelingTable = common.Prepare(eventModelingTokens)

// frameAst is Pele's EmFrame.
type frameAst struct {
	reset            bool // an EmResetFrame, not an EmTimeFrame
	name             string
	modelEntityType  string
	entityIdentifier string
	sourceFrames     []string
	dataReference    string
	hasDataReference bool
	dataType         string
	dataInlineValue  string
	hasInline        bool
	// offset is the byte the frame starts at, for error messages.
	offset int
}

// dataAst is Pele's EmDataEntity.
type dataAst struct {
	name, dataType, dataBlockValue string
}

// noteAst is Pele's EmNoteEntity.
type noteAst struct {
	sourceFrame, dataType, dataBlockValue string
}

// gwtAst is Pele's EmGwt, with the identifiers of its statements.
type gwtAst struct {
	sourceFrame       string
	given, when, then []string
}

// ast is Pele's EventModelAst. The title, accTitle and accDescr are whole tokens, keyword
// included, as the grammar's own terminals get no conversion.
type ast struct {
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	modelEntities                      []string
	frames                             []frameAst
	dataEntities                       []dataAst
	noteEntities                       []noteAst
	gwtEntities                        []gwtAst
	source                             string
}

// parseEventModel parses an event modeling diagram's text into its tree, as Pele's
// parseEventModel does.
func parseEventModel(src string) (tree *ast, err error) {
	toks, err := common.Tokenize(src, eventModelingTable, "eventmodeling")
	if err != nil {
		return nil, err
	}
	defer func() {
		if e := recover(); e != nil {
			de, ok := e.(*diagram.Error)
			if !ok {
				panic(e)
			}
			tree, err = nil, de
		}
	}()
	r := &common.Reader{Tokens: toks, Types: eventModelingTokens, Diagram: "eventmodeling"}
	tree = &ast{source: src}

	at := func(kind int) bool { return r.Kind() == kind }
	one := func(set map[int]bool, expected string) string {
		if !set[r.Kind()] {
			r.Fail(expected)
		}
		return r.Take()
	}
	qualifiedName := func() string {
		name := r.Expect(tID)
		for r.Accept(tDot) {
			name += "." + r.Expect(tID)
		}
		return name
	}
	readDataType := func() string {
		if !r.Accept(tTick) {
			return ""
		}
		typ := one(dataType, "a data type")
		r.Expect(tTick)
		return typ
	}
	statements := func() []string {
		var out []string
		for {
			one(entityType, "an entity type")
			out = append(out, r.Expect(tID))
			if !entityType[r.Kind()] {
				return out
			}
		}
	}

	r.Expect(tKeyword)
	for !at(-1) {
		kind := r.Kind()
		switch {
		case kind == tAccDescr:
			tree.accDescr, tree.hasAccDescr = r.Take(), true
		case kind == tAccTitle:
			tree.accTitle, tree.hasAccTitle = r.Take(), true
		case kind == tTitle:
			tree.title, tree.hasTitle = r.Take(), true
		case kind == tEntity:
			r.I++
			tree.modelEntities = append(tree.modelEntities, qualifiedName())
		case timeFrame[kind] || resetFrame[kind]:
			frame := frameAst{reset: !timeFrame[kind], offset: r.Tokens.Starts[r.I]}
			r.I++
			frame.name = r.Expect(tFrameID)
			frame.modelEntityType = one(entityType, "an entity type")
			frame.entityIdentifier = qualifiedName()
			for r.Accept(tArrow) {
				frame.sourceFrames = append(frame.sourceFrames, r.Expect(tFrameID))
			}
			if r.Accept(tOpen) {
				frame.dataReference, frame.hasDataReference = r.Expect(tID), true
				r.Expect(tClose)
			}
			if at(tTick) || at(tInline) {
				frame.dataType = readDataType()
				frame.dataInlineValue, frame.hasInline = r.Expect(tInline), true
			}
			tree.frames = append(tree.frames, frame)
		case kind == tData:
			r.I++
			entity := dataAst{name: r.Expect(tID)}
			entity.dataType = readDataType()
			entity.dataBlockValue = r.Expect(tBlock)
			tree.dataEntities = append(tree.dataEntities, entity)
		case kind == tNote:
			r.I++
			note := noteAst{sourceFrame: r.Expect(tFrameID)}
			note.dataType = readDataType()
			note.dataBlockValue = r.Expect(tBlock)
			tree.noteEntities = append(tree.noteEntities, note)
		case kind == tGwt:
			r.I++
			gwt := gwtAst{sourceFrame: r.Expect(tFrameID)}
			r.Expect(tGiven)
			gwt.given = statements()
			if r.Accept(tWhen) {
				gwt.when = statements()
			}
			r.Expect(tThen)
			gwt.then = statements()
			tree.gwtEntities = append(tree.gwtEntities, gwt)
		default:
			r.Fail("a frame, an entity, a data block, a note, or a specification")
		}
	}
	return tree, nil
}
