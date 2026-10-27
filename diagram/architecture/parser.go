package architecture

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The kinds of token, by index in archTokens, as Pele's T.
const (
	tHeader = iota
	tJunction
	tService
	tColumn
	tGroup
	tAlign
	tRow
	tDashes
	tIn
	tColon
	tDash
	tDirection
	tArrowGroup
	tInto
	tAccDescr
	tAccTitle
	tTitle
	tString
	tID
	tNewline
	tWhitespace
	tYAML
	tDirective
	tComment
	tIcon
	tLabel
)

// isWord reports whether \w matches the byte.
func isWord(b byte) bool { return chars.IsWord(int(b)) }

// idMatch is Pele's ID, /[\w]([-\w]*\w)?/y: a run of word characters and dashes that starts
// with a word character, given back to its last word character.
func idMatch(s string, at int) int {
	if !isWord(s[at]) {
		return -1
	}
	end := at + 1
	for i := at + 1; i < len(s) && (s[i] == '-' || isWord(s[i])); i++ {
		if s[i] != '-' {
			end = i + 1
		}
	}
	return end
}

// idRest reports whether /[-\w]*\w/ matches at byte at, which is where the run of word
// characters and dashes there holds a word character. It is the rest of an ID that
// titleBefore gives way to in Pele's ARCHITECTURE_TOKENS, so that `titles` is an ID.
func idRest(s string, at int) bool {
	for i := at; i < len(s) && (s[i] == '-' || isWord(s[i])); i++ {
		if s[i] != '-' {
			return true
		}
	}
	return false
}

// word is Pele's word: the text, which an identifier that goes on longer takes over.
func word(text string) common.TokenType {
	return common.TokenType{Name: text, Literal: text, Longer: []int{tID}}
}

// iconMatch is Pele's ARCH_ICON, /\([\w-:]+\)/y, in which the class holds word characters,
// dashes and colons.
func iconMatch(s string, at int) int {
	if s[at] != '(' {
		return -1
	}
	i := at + 1
	for i < len(s) && (isWord(s[i]) || s[i] == '-' || s[i] == ':') {
		i++
	}
	if i == at+1 || i == len(s) || s[i] != ')' {
		return -1
	}
	return i + 1
}

// titleMatch is Pele's ARCH_TITLE,
// /\[(?:"([^"\\]|\\.)*"|'([^'\\]|\\.)*'|[^\[\]\r\n]+)\]/y. A quoted string ends at its first
// quote that no backslash takes, and must be followed by the bracket; when it is not, nor
// can any shorter run of its characters be, the text is tried as the third alternative, a
// run of characters other than brackets and line breaks. A backslash before a line
// terminator, which . does not take, ends no string.
func titleMatch(s string, at int) int {
	if s[at] != '[' {
		return -1
	}
	i := at + 1
	if i < len(s) && (s[i] == '"' || s[i] == '\'') {
		q := s[i]
	quoted:
		for j := i + 1; j < len(s); {
			switch s[j] {
			case q:
				if j+1 < len(s) && s[j+1] == ']' {
					return j + 2
				}
				break quoted
			case '\\':
				c, size := chars.UnitAt(s, j+1)
				if c < 0 || chars.IsLineEnd(c) {
					break quoted
				}
				j += 1 + size
			default:
				j++
			}
		}
	}
	j := i
	for j < len(s) && s[j] != '[' && s[j] != ']' && s[j] != '\r' && s[j] != '\n' {
		j++
	}
	if j > i && j < len(s) && s[j] == ']' {
		return j + 1
	}
	return -1
}

// archTokens is Pele's ARCHITECTURE_TOKENS. The direction letters come before identifiers
// and have no longer alternative, so Mermaid reads an id that starts with L, R, T or B as a
// direction followed by the rest.
var archTokens = []common.TokenType{
	word("architecture-beta"),
	word("junction"),
	word("service"),
	word("column"),
	word("group"),
	word("align"),
	word("row"),
	{Name: "--", Literal: "--", Longer: []int{tYAML}},
	word("in"),
	{Name: ":", Literal: ":"},
	{Name: "-", Literal: "-", Longer: []int{tYAML}},
	{Name: "ARROW_DIRECTION", Match: func(s string, at int) int {
		switch s[at] {
		case 'L', 'R', 'T', 'B':
			return at + 1
		}
		return -1
	}},
	{Name: "ARROW_GROUP", Match: func(s string, at int) int {
		if strings.HasPrefix(s[at:], "{group}") {
			return at + len("{group}")
		}
		return -1
	}},
	{Name: "ARROW_INTO", Match: func(s string, at int) int {
		if s[at] == '<' || s[at] == '>' {
			return at + 1
		}
		return -1
	}},
	common.AccDescrToken,
	common.AccTitleToken,
	common.TitleBefore(idRest),
	common.StringToken,
	{Name: "ID", Match: idMatch},
	common.NewlineToken,
	common.WhitespaceToken,
	common.YAMLToken,
	common.DirectiveToken,
	common.CommentToken,
	{Name: "ARCH_ICON", Match: iconMatch},
	{Name: "ARCH_TITLE", Match: titleMatch},
}

var archTable = common.Prepare(archTokens)

// Side is the side of a node an edge meets: 'L', 'R', 'T' or 'B'.
type Side byte

// groupAst is Pele's GroupAst; a string Pele leaves undefined is empty.
type groupAst struct {
	id, icon, title, in string
}

// serviceAst is Pele's ServiceAst.
type serviceAst struct {
	id, icon, iconText, title, in string
}

// junctionAst is Pele's JunctionAst.
type junctionAst struct {
	id, in string
}

// edgeAst is Pele's EdgeAst.
type edgeAst struct {
	lhsID    string
	lhsGroup bool
	lhsDir   Side
	lhsInto  bool
	title    string
	rhsInto  bool
	rhsDir   Side
	rhsID    string
	rhsGroup bool
}

// alignmentAst is Pele's AlignmentAst.
type alignmentAst struct {
	direction string
	members   []string
}

// ast is Pele's ArchitectureAst.
type ast struct {
	title, accTitle, accDescr string
	groups                    []groupAst
	services                  []serviceAst
	junctions                 []junctionAst
	edges                     []edgeAst
	alignments                []alignmentAst
}

// labelValue is the text of an ARCH_TITLE token, as Mermaid's ArchitectureValueConverter
// and Pele's labelValue read it: the brackets taken off and the rest trimmed, and a string
// in quotes unquoted, with its escaped quotes made quotes.
func labelValue(text string) string {
	result := chars.Trim(text[1 : len(text)-1])
	if strings.HasPrefix(result, `"`) && strings.HasSuffix(result, `"`) ||
		strings.HasPrefix(result, "'") && strings.HasSuffix(result, "'") {
		if len(result) < 2 {
			result = ""
		} else {
			result = result[1 : len(result)-1]
		}
		result = strings.ReplaceAll(result, `\"`, `"`)
		result = strings.ReplaceAll(result, `\'`, "'")
	}
	return chars.Trim(result)
}

// parseArchitecture parses an architecture diagram's text into its tree, as Pele's
// parseArchitecture does.
func parseArchitecture(src string) (tree *ast, err error) {
	toks, err := common.Tokenize(src, archTable, "architecture")
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
	r := &common.Reader{Tokens: toks, Types: archTokens, Diagram: "architecture"}
	tree = &ast{}

	is := func(kind int) bool { return r.Kind() == kind }
	endOfLine := func() {
		if r.Kind() == -1 {
			return
		}
		if r.Kind() != tNewline {
			r.Fail("a line break or the end of input")
		}
		for r.Kind() == tNewline {
			r.I++
		}
	}
	icon := func() string {
		s := r.Take()
		return s[1 : len(s)-1]
	}

	for r.Kind() == tNewline {
		r.I++
	}
	r.Expect(tHeader)
	for r.Kind() != -1 {
		switch r.Kind() {
		case tNewline:
			r.I++
			continue
		case tAccDescr:
			tree.accDescr = common.AccDescrValue(r.Take())
		case tAccTitle:
			tree.accTitle = common.AccTitleValue(r.Take())
		case tTitle:
			tree.title = common.TitleValue(r.Take())
		case tGroup:
			r.I++
			group := groupAst{id: r.Expect(tID)}
			if is(tIcon) {
				group.icon = icon()
			}
			if is(tLabel) {
				group.title = labelValue(r.Take())
			}
			if r.Accept(tIn) {
				group.in = r.Expect(tID)
			}
			tree.groups = append(tree.groups, group)
		case tService:
			r.I++
			service := serviceAst{id: r.Expect(tID)}
			if is(tString) {
				service.iconText = common.StringValue(r.Take())
			} else if is(tIcon) {
				service.icon = icon()
			}
			if is(tLabel) {
				service.title = labelValue(r.Take())
			}
			if r.Accept(tIn) {
				service.in = r.Expect(tID)
			}
			tree.services = append(tree.services, service)
		case tJunction:
			r.I++
			junction := junctionAst{id: r.Expect(tID)}
			if r.Accept(tIn) {
				junction.in = r.Expect(tID)
			}
			tree.junctions = append(tree.junctions, junction)
		case tID:
			var edge edgeAst
			edge.lhsID = r.Take()
			edge.lhsGroup = r.Accept(tArrowGroup)
			r.Expect(tColon)
			edge.lhsDir = Side(r.Expect(tDirection)[0])
			edge.lhsInto = r.Accept(tInto)
			if !r.Accept(tDashes) {
				if !is(tDash) {
					r.Fail("'--' or '-'")
				}
				r.I++
				edge.title = labelValue(r.Expect(tLabel))
				r.Expect(tDash)
			}
			edge.rhsInto = r.Accept(tInto)
			edge.rhsDir = Side(r.Expect(tDirection)[0])
			r.Expect(tColon)
			edge.rhsID = r.Expect(tID)
			edge.rhsGroup = r.Accept(tArrowGroup)
			tree.edges = append(tree.edges, edge)
		case tAlign:
			r.I++
			if !is(tRow) && !is(tColumn) {
				r.Fail("'row' or 'column'")
			}
			a := alignmentAst{direction: r.Take()}
			a.members = []string{r.Expect(tID), r.Expect(tID)}
			for is(tID) {
				a.members = append(a.members, r.Take())
			}
			tree.alignments = append(tree.alignments, a)
		default:
			r.Fail("a group, a service, a junction, an edge, an alignment, a title, or a line break")
		}
		endOfLine()
	}
	return tree, nil
}
