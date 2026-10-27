package treemap

import (
	"math"
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The kinds of token, by index in treemapTokens, as Pele's T.
const (
	tAccDescr = iota
	tAccTitle
	tTitle
	tKeyword
	tClassDef
	tStyleSeparator
	tSeparator
	tComma
	tIndentation
	tWhitespace
	tComment
	tNewline
	tID
	tNumber
	tString
)

// treemapTokens is Pele's TREEMAP_TOKENS, Mermaid's treemap grammar's own terminals. Every
// run of spaces or tabs is an INDENTATION token, since that comes before WS, and line breaks
// are hidden, so rows are told apart by their tokens alone and the indentation of a row is
// whatever white space comes before it. Pele gives most of these types no first characters;
// those given here are only the characters their patterns can start with, which changes
// nothing but how many are tried.
var treemapTokens = []common.TokenType{
	common.AccDescrToken,
	common.AccTitleToken,
	// Pele's titleBefore(String.raw`\w`): the word alone is no title where a word character
	// follows, so that a class name such as title_2 is read whole.
	common.TitleBefore(func(s string, at int) bool { return at < len(s) && chars.IsWord(int(s[at])) }),
	{Name: "TREEMAP_KEYWORD", First: "t", Match: matchKeyword},
	{Name: "CLASS_DEF", First: "c", Match: func(s string, at int) int {
		end, _, _, _ := matchClassDef(s, at)
		return end
	}},
	{Name: "STYLE_SEPARATOR", Literal: ":::"},
	{Name: "SEPARATOR", Literal: ":"},
	{Name: "COMMA", Literal: ","},
	{Name: "INDENTATION", First: " \t", Match: blankRun},
	{Name: "WS", First: " \t", Hidden: true, Match: blankRun},
	{Name: "ML_COMMENT", First: "%", Hidden: true, Match: func(s string, at int) int {
		if !strings.HasPrefix(s[at:], "%%") {
			return -1
		}
		if end := strings.IndexByte(s[at+2:], '\n'); end >= 0 {
			return at + 2 + end
		}
		return len(s)
	}},
	{Name: "NL", First: "\r\n", Hidden: true, Match: func(s string, at int) int {
		switch {
		case strings.HasPrefix(s[at:], "\r\n"):
			return at + 2
		case s[at] == '\n':
			return at + 1
		}
		return -1
	}},
	{Name: "ID2", First: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_", Match: func(s string, at int) int {
		if !idStart(s[at]) {
			return -1
		}
		return idRest(s, at+1)
	}},
	{Name: "NUMBER2", First: "0123456789_.,", Match: func(s string, at int) int {
		end := at
		for end < len(s) && (chars.IsDigit(int(s[end])) || s[end] == '_' || s[end] == '.' || s[end] == ',') {
			end++
		}
		if end == at {
			return -1
		}
		return end
	}},
	{Name: "STRING2", First: `"'`, Match: func(s string, at int) int {
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
}

var treemapTable = common.Prepare(treemapTokens)

// matchKeyword matches /treemap-beta|treemap/y, which needs nothing after the word.
func matchKeyword(s string, at int) int {
	switch {
	case strings.HasPrefix(s[at:], "treemap-beta"):
		return at + len("treemap-beta")
	case strings.HasPrefix(s[at:], "treemap"):
		return at + len("treemap")
	}
	return -1
}

// blankRun matches /[ \t]+/y.
func blankRun(s string, at int) int {
	end := at
	for end < len(s) && (s[end] == ' ' || s[end] == '\t') {
		end++
	}
	if end == at {
		return -1
	}
	return end
}

func idStart(c byte) bool { return chars.IsLetter(int(c)) || c == '_' }

// idRest is the end of the run of [a-zA-Z0-9_] from byte i.
func idRest(s string, i int) int {
	for i < len(s) && (idStart(s[i]) || chars.IsDigit(int(s[i]))) {
		i++
	}
	return i
}

// matchClassDef matches CLASS_DEF,
// /classDef\s+([a-zA-Z_][a-zA-Z0-9_]+)(?:\s+([^;\r\n]*))?(?:;)?/y, at byte at, and gives the
// end of the match, or -1, its class's name, and the text of its styles when that group
// took part. The white space before the styles is greedy and may cross lines; the styles
// then run to the end of a line or a semicolon. No path that gives back characters matches
// where the first does not, so the first is the match.
func matchClassDef(s string, at int) (end int, name, styles string, hasStyles bool) {
	if !strings.HasPrefix(s[at:], "classDef") {
		return -1, "", "", false
	}
	i := at + len("classDef")
	j := chars.SpaceRun(s, i)
	if j == i || j == len(s) || !idStart(s[j]) {
		return -1, "", "", false
	}
	k := idRest(s, j+1)
	if k == j+1 {
		return -1, "", "", false
	}
	name, end = s[j:k], k
	if m := chars.SpaceRun(s, k); m > k {
		e := m
		for e < len(s) && s[e] != ';' && s[e] != '\r' && s[e] != '\n' {
			e++
		}
		styles, hasStyles, end = s[m:e], true, e
	}
	if end < len(s) && s[end] == ';' {
		end++
	}
	return end, name, styles, hasStyles
}

// itemAst is Pele's SectionAst or LeafAst.
type itemAst struct {
	leaf     bool
	name     string
	value    float64
	selector string
	hasClass bool
}

// rowAst is Pele's TreemapRowAst or ClassDefAst; an indent Pele leaves undefined has
// hasIndent unset.
type rowAst struct {
	classDef  bool
	indent    int
	hasIndent bool
	item      itemAst
	className string
	styleText string
	hasStyle  bool
}

// ast is Pele's TreemapAst.
type ast struct {
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	rows                               []rowAst
}

// parseFloat is JavaScript's parseFloat of what a NUMBER2 token holds once its commas are
// taken out: the longest start of it that is a decimal number, digits with a point among or
// after them, and NaN when it starts with no number. An underscore ends the number. More
// digits than a float64 holds give an infinity, as in JavaScript.
func parseFloat(s string) float64 {
	i := 0
	for i < len(s) && chars.IsDigit(int(s[i])) {
		i++
	}
	end, digits := i, i > 0
	if i < len(s) && s[i] == '.' {
		j := i + 1
		for j < len(s) && chars.IsDigit(int(s[j])) {
			j++
		}
		if j > i+1 || digits {
			end, digits = j, true
		}
	}
	if !digits {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(s[:end], 64)
	return f
}

// parseTreemap parses a treemap's text into its tree, as Pele's parseTreemap does.
func parseTreemap(src string) (tree *ast, err error) {
	toks, err := common.Tokenize(src, treemapTable, "treemap")
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
	r := &common.Reader{Tokens: toks, Types: treemapTokens, Diagram: "treemap"}
	kinds := toks.Kinds
	tree = &ast{}

	r.Expect(tKeyword)
	for r.Kind() != -1 {
		kind := r.Kind()
		switch kind {
		case tAccDescr:
			tree.accDescr, tree.hasAccDescr = common.AccDescrValue(r.Take()), true
			continue
		case tAccTitle:
			tree.accTitle, tree.hasAccTitle = common.AccTitleValue(r.Take()), true
			continue
		case tTitle:
			tree.title, tree.hasTitle = common.TitleValue(r.Take()), true
			continue
		case tIndentation, tString, tClassDef:
		default:
			r.Fail("a node, a class definition, or a title")
		}

		var row rowAst
		if kind == tIndentation {
			row.indent, row.hasIndent = len(r.Take()), true
		}
		if r.Kind() == tClassDef {
			text := r.Take()
			_, name, styles, has := matchClassDef(text, 0)
			row.classDef, row.className = true, name
			if has && styles != "" {
				row.styleText, row.hasStyle = styles, true
			}
			tree.rows = append(tree.rows, row)
			continue
		}

		quoted := r.Expect(tString)
		item := itemAst{name: quoted[1 : len(quoted)-1]}
		// A leaf is a name followed by a colon or comma, with or without white space before it.
		after := r.I
		if kinds[r.I] == tIndentation {
			after++
		}
		if kinds[after] == tSeparator || kinds[after] == tComma {
			r.I = after + 1
			r.Accept(tIndentation)
			item.leaf = true
			item.value = parseFloat(strings.ReplaceAll(r.Expect(tNumber), ",", ""))
		}
		if r.Accept(tStyleSeparator) {
			item.selector, item.hasClass = r.Expect(tID), true
		}
		row.item = item
		tree.rows = append(tree.rows, row)
	}
	return tree, nil
}
