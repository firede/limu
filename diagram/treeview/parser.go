package treeview

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The kinds of token, by index in treeViewTokens, as Pele's T.
const (
	tKeyword = iota
	tAccDescr
	tAccTitle
	tTitle
	tClassAnnotation
	tIconAnnotation
	tDescAnnotation
	tIndentation
	tQuotedName
	tWS
	tComment
	tNewline
	tBareName
)

// isBlank reports whether byte i of s is a space or a tab, and not past its end, as Pele's
// isBlank of charCodeAt.
func isBlank(s string, i int) bool { return i < len(s) && chars.IsBlank(int(s[i])) }

// blanks is the end of the run of spaces and tabs from byte i.
func blanks(s string, i int) int {
	for isBlank(s, i) {
		i++
	}
	return i
}

// annotationAt reports whether an annotation starts at byte at, as Pele's annotationAt does:
// a description, an icon, or a class whose name starts as one may.
func annotationAt(src string, at int) bool {
	rest := src[min(at, len(src)):]
	if strings.HasPrefix(rest, "##") || strings.HasPrefix(rest, "icon(") {
		return true
	}
	if !strings.HasPrefix(rest, ":::") {
		return false
	}
	i := blanks(src, at+3)
	if i >= len(src) {
		return false
	}
	c := src[i]
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_'
}

// matchBareName is Pele's matchBareName, which matches as its pattern
// /(?!:::|icon\(|##)[^ \t\n\r"'](?:(?![ \t]+:::[ \t]*[A-Za-z_]|[ \t]+icon\(|[ \t]+##)[^\n\r])*/y
// does, looking for an annotation once for each run of blanks: a name runs to the end of its
// line or to the blanks before an annotation.
func matchBareName(src string, at int) int {
	first := src[at]
	if isBlank(src, at) || first == '\n' || first == '\r' || first == '"' || first == '\'' {
		return -1
	}
	rest := src[at:]
	if strings.HasPrefix(rest, ":::") || strings.HasPrefix(rest, "icon(") || strings.HasPrefix(rest, "##") {
		return -1
	}
	i := at + 1
	for i < len(src) {
		c := src[i]
		if c == '\n' || c == '\r' {
			break
		}
		if isBlank(src, i) {
			j := blanks(src, i+1)
			if annotationAt(src, j) {
				break
			}
			i = j
		} else {
			i++
		}
	}
	return i
}

// matchRun matches [ \t]+ followed by what more matches from the end of the blanks, or -1.
func matchRun(s string, at int, more func(i int) int) int {
	i := blanks(s, at)
	if i == at {
		return -1
	}
	return more(i)
}

// wordRun is the end of the run of [\w-] from byte i.
func wordRun(s string, i int) int {
	for i < len(s) && (chars.IsWord(int(s[i])) || s[i] == '-') {
		i++
	}
	return i
}

// keyword is Pele's keyword('treeView-beta'), with a name that runs on longer taking its
// place.
var keyword = func() common.TokenType {
	t := common.Keyword("treeView-beta")
	t.Longer = []int{tBareName}
	return t
}()

// treeViewTokens is Pele's TREEVIEW_TOKENS.
var treeViewTokens = []common.TokenType{
	keyword,
	common.AccDescr("ACC_DESCR"),
	common.AccTitleToken,
	common.TitleToken,
	// /[ \t]+:::[ \t]*[A-Za-z_][\w-]*/y
	{Name: "CLASS_ANNOTATION", First: "\t ", Match: func(s string, at int) int {
		return matchRun(s, at, func(i int) int {
			if !strings.HasPrefix(s[i:], ":::") {
				return -1
			}
			i = blanks(s, i+3)
			if i >= len(s) || !(chars.IsLetter(int(s[i])) || s[i] == '_') {
				return -1
			}
			return wordRun(s, i+1)
		})
	}},
	// /[ \t]+icon\([\w-]*(?::[\w-]+)?\)/y
	{Name: "ICON_ANNOTATION", First: "\t ", Match: func(s string, at int) int {
		return matchRun(s, at, func(i int) int {
			if !strings.HasPrefix(s[i:], "icon(") {
				return -1
			}
			i = wordRun(s, i+len("icon("))
			if i < len(s) && s[i] == ':' {
				if j := wordRun(s, i+1); j > i+1 {
					i = j
				}
			}
			if i < len(s) && s[i] == ')' {
				return i + 1
			}
			return -1
		})
	}},
	// /[ \t]+##[^\n\r]*/y
	{Name: "DESC_ANNOTATION", First: "\t ", Match: func(s string, at int) int {
		return matchRun(s, at, func(i int) int {
			if !strings.HasPrefix(s[i:], "##") {
				return -1
			}
			i += 2
			for i < len(s) && s[i] != '\n' && s[i] != '\r' {
				i++
			}
			return i
		})
	}},
	// /[ \t]{1,}/y
	{Name: "INDENTATION", First: "\t ", Match: func(s string, at int) int {
		return matchRun(s, at, func(i int) int { return i })
	}},
	// /"[^"]*"|'[^']*'/y
	{Name: "QUOTED_NAME", First: `"'`, Match: func(s string, at int) int {
		q := s[at]
		if q != '"' && q != '\'' {
			return -1
		}
		end := strings.IndexByte(s[at+1:], q)
		if end < 0 {
			return -1
		}
		return at + 1 + end + 1
	}},
	// /[ \t]+/y, which INDENTATION always takes first.
	{Name: "WS", First: "\t ", Hidden: true, Match: func(s string, at int) int {
		return matchRun(s, at, func(i int) int { return i })
	}},
	// /\%\%[^\n]*/y
	{Name: "ML_COMMENT", First: "%", Hidden: true, Match: func(s string, at int) int {
		if !strings.HasPrefix(s[at:], "%%") {
			return -1
		}
		if end := strings.IndexByte(s[at:], '\n'); end >= 0 {
			return at + end
		}
		return len(s)
	}},
	{Name: "NL", First: "\r\n", Hidden: true, Match: common.NewlineToken.Match},
	{Name: "BARE_NAME", Match: matchBareName},
}

var treeViewTable = common.Prepare(treeViewTokens)

// tokenize splits a tree view's text into tokens, as Pele's tokenize does with its table.
func tokenize(src string) (*common.Tokens, error) {
	return common.Tokenize(src, treeViewTable, "treeView")
}

// nodeAst is Pele's TreeNodeAst; a field Pele leaves undefined has its Has field unset.
type nodeAst struct {
	indent                                   int
	hasIndent                                bool
	name                                     string
	classAnnotation, iconAnnotation, descAnn string
	hasClass, hasIcon, hasDesc               bool
}

// ast is Pele's TreeViewAst.
type ast struct {
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	nodes                              []nodeAst
}

// trimBlanksEnd is the text with the spaces and tabs at its end taken off, as Pele's
// trimBlanksEnd.
func trimBlanksEnd(text string) string {
	end := len(text)
	for end > 0 && chars.IsBlank(int(text[end-1])) {
		end--
	}
	return text[:end]
}

// convertValue is Mermaid's TreeViewValueConverter for the annotations, by terminal name, as
// Pele's convertValue converts them.
func convertValue(rule, input string) string {
	switch rule {
	case "CLASS_ANNOTATION":
		return chars.Trim(chars.Trim(input)[3:])
	case "ICON_ANNOTATION":
		trimmed := chars.Trim(input)
		return trimmed[5 : len(trimmed)-1]
	case "DESC_ANNOTATION":
		return chars.Trim(chars.Trim(input)[2:])
	}
	return ""
}

// parseTreeView parses a tree view's text into its tree, as Pele's parseTreeView does.
func parseTreeView(src string) (tree *ast, err error) {
	toks, err := tokenize(src)
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
	r := &common.Reader{Tokens: toks, Types: treeViewTokens, Diagram: "treeView"}
	tree = &ast{}
	r.Expect(tKeyword)
	for {
		switch r.Kind() {
		case tAccDescr:
			tree.accDescr, tree.hasAccDescr = common.AccDescrValue(r.Take()), true
			continue
		case tAccTitle:
			tree.accTitle, tree.hasAccTitle = common.AccTitleValue(r.Take()), true
			continue
		case tTitle:
			tree.title, tree.hasTitle = common.TitleValue(r.Take()), true
			continue
		}
		break
	}
	for r.Kind() != -1 {
		var node nodeAst
		if r.Kind() == tIndentation {
			node.indent, node.hasIndent = len(r.Take()), true
		}
		switch r.Kind() {
		case tQuotedName:
			text := r.Take()
			node.name = text[1 : len(text)-1]
		case tBareName:
			node.name = trimBlanksEnd(r.Take())
		default:
			r.Fail("a name")
		}
	annotations:
		for {
			switch r.Kind() {
			case tClassAnnotation:
				node.classAnnotation, node.hasClass = convertValue("CLASS_ANNOTATION", r.Take()), true
			case tIconAnnotation:
				node.iconAnnotation, node.hasIcon = convertValue("ICON_ANNOTATION", r.Take()), true
			case tDescAnnotation:
				node.descAnn, node.hasDesc = convertValue("DESC_ANNOTATION", r.Take()), true
			default:
				break annotations
			}
		}
		tree.nodes = append(tree.nodes, node)
	}
	return tree, nil
}
