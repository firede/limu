package radar

import (
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The kinds of token, by index in the table radarTokens makes, as Pele's T, and those of the
// hidden types after them.
const (
	tRadarColon = iota
	tRadar
	tShowLegend
	tGraticule
	tCurve
	tTicks
	tAxis
	tMax
	tMin
	tColon
	tComma
	tOpen
	tClose
	tOpenBrace
	tCloseBrace
	tGraticuleValue
	tBoolean
	tAccDescr
	tAccTitle
	tTitle
	tNumber
	tString
	tID
	tNewline
	tWhitespace
	tYAML
	tDirective
	tComment
)

// isWord reports whether \w matches byte c: an ASCII letter, digit or underscore.
func isWord(c byte) bool { return chars.IsWord(int(c)) }

// matchID is Pele's ID, /[\w]([-\w]*\w)?/y: a word character, and the run of word characters
// and dashes after it up to the last word character in it.
func matchID(src string, at int) int {
	if !isWord(src[at]) {
		return -1
	}
	end := at + 1
	for k := at + 1; k < len(src) && (isWord(src[k]) || src[k] == '-'); k++ {
		if src[k] != '-' {
			end = k + 1
		}
	}
	return end
}

// digits is the end of the run of digits from byte at, as Pele's digits gives it.
func digits(src string, at int) int {
	end := at
	for end < len(src) && src[end] >= '0' && src[end] <= '9' {
		end++
	}
	return end
}

// lexer holds what the matchers of one text remember of it, as Pele's runFrom, runEnd and
// lastBrace do.
type lexer struct {
	runFrom, runEnd int
	lastBrace       int
}

// matchNumber is Pele's NUMBER, /(?:[0-9]+\.[0-9]+(?!\.))|(?:0|[1-9][0-9]*(?!\.))/y, as
// Pele's matchNumber reads it without the backtracking: the run of digits a position is in is
// read once, as a regular expression would read to its end from every position in it.
func (l *lexer) matchNumber(src string, at int) int {
	if at < l.runFrom || at >= l.runEnd {
		l.runFrom = at
		l.runEnd = digits(src, at)
	}
	whole := l.runEnd
	if whole == at {
		return -1
	}
	dot := whole < len(src) && src[whole] == '.'
	if dot {
		fraction := digits(src, whole+1)
		length := fraction - whole - 1
		if length > 0 && (fraction == len(src) || src[fraction] != '.') {
			return fraction
		}
		if length > 1 {
			return fraction - 1
		}
	}
	if src[at] == '0' {
		return at + 1
	}
	if !dot {
		return whole
	}
	if whole-1 > at {
		return whole - 1
	}
	return -1
}

// blockOpener matches /[\t ]*accDescr\s*{/y at byte at, giving the end of the brace, or -1.
func blockOpener(src string, at int) int {
	i := at
	for i < len(src) && (src[i] == ' ' || src[i] == '\t') {
		i++
	}
	if !strings.HasPrefix(src[i:], "accDescr") {
		return -1
	}
	i = chars.SpaceRun(src, i+len("accDescr"))
	if i < len(src) && src[i] == '{' {
		return i + 1
	}
	return -1
}

// matchAccDescr is Pele's ACC_DESCR as its matchAccDescr reads it: here `accDescr` is also a
// name and `{` a token, so a block that never closes is known at once by the last closing
// brace of the text, rather than read to the end of the text each time.
func (l *lexer) matchAccDescr(src string, at int) int {
	if end := blockOpener(src, at); end >= 0 {
		if l.lastBrace == -2 {
			l.lastBrace = strings.LastIndexByte(src, '}')
		}
		if end > l.lastBrace {
			return -1
		}
	}
	return common.AccDescrToken.Match(src, at)
}

// word is Pele's word: the text, which an ID that goes on past it takes over.
func word(text string) common.TokenType {
	return common.TokenType{Name: text, Literal: text, Longer: []int{tID}}
}

// mark is Pele's mark: the text alone.
func mark(text string) common.TokenType { return common.TokenType{Name: text, Literal: text} }

// radarTokens is Pele's RADAR_TOKENS, with matchers that remember what l has read of a text.
func radarTokens(l *lexer) []common.TokenType {
	radar := common.Keyword("radar-beta")
	radar.Longer = []int{tID}
	accDescr := common.AccDescrToken
	accDescr.Match = l.matchAccDescr
	return []common.TokenType{
		mark("radar-beta:"),
		radar,
		word("showLegend"),
		word("graticule"),
		word("curve"),
		word("ticks"),
		word("axis"),
		word("max"),
		word("min"),
		mark(":"),
		mark(","),
		mark("["),
		mark("]"),
		mark("{"),
		mark("}"),
		{Name: "GRATICULE", Match: func(src string, at int) int {
			for _, w := range [...]string{"circle", "polygon"} {
				if strings.HasPrefix(src[at:], w) {
					return at + len(w)
				}
			}
			return -1
		}},
		{Name: "BOOLEAN", Match: func(src string, at int) int {
			for _, w := range [...]string{"true", "false"} {
				if strings.HasPrefix(src[at:], w) {
					return at + len(w)
				}
			}
			return -1
		}},
		accDescr,
		common.AccTitleToken,
		common.TitleToken,
		{Name: "NUMBER", Match: l.matchNumber},
		common.StringToken,
		{Name: "ID", Match: matchID},
		common.NewlineToken,
		common.WhitespaceToken,
		common.YAMLToken,
		common.DirectiveToken,
		common.CommentToken,
	}
}

// names are the names of the token types, by kind.
var names = func() []string {
	var out []string
	for _, t := range radarTokens(&lexer{}) {
		out = append(out, t.Name)
	}
	return out
}()

// lexRadar splits a radar chart's text into tokens, as Pele's lexRadar does. The types it
// returns are those it tokenized with.
func lexRadar(src string) (*common.Tokens, []common.TokenType, error) {
	types := radarTokens(&lexer{lastBrace: -2})
	toks, err := common.Tokenize(src, common.Prepare(types), "radar")
	return toks, types, err
}

// axisAst is Pele's AxisAst; a label Pele leaves undefined has hasLabel unset.
type axisAst struct {
	name, label string
	hasLabel    bool
}

// entryAst is Pele's EntryAst: the name of the axis it refers to, if it names one, and its
// value.
type entryAst struct {
	axis    string
	hasAxis bool
	value   float64
}

// curveAst is Pele's CurveAst.
type curveAst struct {
	name, label string
	hasLabel    bool
	entries     []entryAst
}

// optionAst is Pele's OptionAst: showLegend's value is a bool, graticule's a string, and the
// others' a number.
type optionAst struct {
	name  string
	value any
}

// ast is Pele's RadarAst; a string Pele leaves undefined has its Has field unset.
type ast struct {
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	axes                               []axisAst
	curves                             []curveAst
	options                            []optionAst
}

// number is JavaScript's Number() of a NUMBER token, whose digits may be too many for a
// float64 to hold, and are then infinite.
func number(text string) float64 {
	f, _ := strconv.ParseFloat(text, 64)
	return f
}

// parseRadar parses a radar chart's text into its tree, as Pele's parseRadar does.
func parseRadar(src string) (tree *ast, err error) {
	toks, types, err := lexRadar(src)
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
	r := &common.Reader{Tokens: toks, Types: types, Diagram: "radar"}
	tree = &ast{}

	newlines := func() {
		for r.Kind() == tNewline {
			r.I++
		}
	}
	endOfLine := func() {
		if r.Kind() == -1 {
			return
		}
		if r.Kind() != tNewline {
			r.Fail("a line break or the end of input")
		}
		newlines()
	}
	label := func() (string, bool) {
		if !r.Accept(tOpen) {
			return "", false
		}
		text := common.StringValue(r.Expect(tString))
		r.Expect(tClose)
		return text, true
	}
	entry := func(detailed bool) entryAst {
		if !detailed {
			return entryAst{value: number(r.Expect(tNumber))}
		}
		axis := r.Expect(tID)
		r.Accept(tColon)
		return entryAst{axis: axis, hasAxis: true, value: number(r.Expect(tNumber))}
	}
	curve := func() curveAst {
		node := curveAst{name: r.Expect(tID)}
		node.label, node.hasLabel = label()
		r.Expect(tOpenBrace)
		newlines()
		if r.Kind() != tNumber && r.Kind() != tID {
			r.Fail("a number or an axis name")
		}
		detailed := r.Kind() == tID
		node.entries = append(node.entries, entry(detailed))
		for r.Accept(tComma) {
			newlines()
			node.entries = append(node.entries, entry(detailed))
		}
		newlines()
		r.Expect(tCloseBrace)
		return node
	}
	axis := func() axisAst {
		node := axisAst{name: r.Expect(tID)}
		node.label, node.hasLabel = label()
		return node
	}
	option := func() optionAst {
		kind := r.Kind()
		name := r.Take()
		var value any
		switch kind {
		case tShowLegend:
			value = r.Expect(tBoolean) == "true"
		case tGraticule:
			value = r.Expect(tGraticuleValue)
		default:
			value = number(r.Expect(tNumber))
		}
		return optionAst{name, value}
	}
	isOption := func() bool {
		switch r.Kind() {
		case tShowLegend, tTicks, tMax, tMin, tGraticule:
			return true
		}
		return false
	}

	newlines()
	if !r.Accept(tRadarColon) {
		r.Expect(tRadar)
		r.Accept(tColon)
	}
	newlines()
	for r.Kind() != -1 {
		switch r.Kind() {
		case tNewline:
			r.I++
		case tAccDescr:
			tree.accDescr, tree.hasAccDescr = common.AccDescrValue(r.Take()), true
			endOfLine()
		case tAccTitle:
			tree.accTitle, tree.hasAccTitle = common.AccTitleValue(r.Take()), true
			endOfLine()
		case tTitle:
			tree.title, tree.hasTitle = common.TitleValue(r.Take()), true
			endOfLine()
		case tAxis:
			r.I++
			tree.axes = append(tree.axes, axis())
			for r.Accept(tComma) {
				tree.axes = append(tree.axes, axis())
			}
		case tCurve:
			r.I++
			tree.curves = append(tree.curves, curve())
			for r.Accept(tComma) {
				tree.curves = append(tree.curves, curve())
			}
		default:
			if !isOption() {
				r.Fail("an axis, a curve, an option, a title, or a line break")
			}
			tree.options = append(tree.options, option())
			for r.Accept(tComma) {
				if !isOption() {
					r.Fail("an option")
				}
				tree.options = append(tree.options, option())
			}
		}
	}
	return tree, nil
}
