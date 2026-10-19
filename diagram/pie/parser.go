package pie

import (
	"strconv"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// The kinds of token, by index in pieTokens, as Pele's T, and those of the hidden types after
// them.
const (
	tShowData = iota
	tPie
	tColon
	tNumber
	tAccDescr
	tAccTitle
	tTitle
	tString
	tNewline
	tWhitespace
	tYAML
	tDirective
	tComment
)

// digits is the end of the run of ASCII digits from byte from, as Pele's digits gives it.
func digits(src string, from int) int {
	i := from
	for i < len(src) && src[i] >= '0' && src[i] <= '9' {
		i++
	}
	return i
}

// numberScanner makes a matcher that answers as NUMBER_PIE's pattern,
// /(?:-?[0-9]+\.[0-9]+(?!\.))|(?:-?(0|[1-9][0-9]*)(?!\.))/y, does, as Pele's numberScanner
// does. The pattern reads a whole run of digits before it finds there is no fraction, and
// would do so again from every digit of a long run of zeros, so the matcher remembers the
// last run it read.
func numberScanner() func(src string, at int) int {
	from, end, fraction := 0, 0, -1
	return func(src string, p int) int {
		at := p
		if src[p] == '-' {
			at = p + 1
		}
		if at < from || at >= end {
			from = at
			end = digits(src, at)
			fraction = -1
			if end > at && end < len(src) && src[end] == '.' {
				stop := digits(src, end+1)
				length := stop - end - 1
				if length > 0 && (stop == len(src) || src[stop] != '.') {
					fraction = stop
				} else if length > 1 {
					fraction = stop - 1
				}
			}
		}
		if end == at {
			return -1
		}
		if fraction >= 0 {
			return fraction
		}
		if src[at] == '0' {
			if at+1 < len(src) && src[at+1] == '.' {
				return -1
			}
			return at + 1
		}
		if end == len(src) || src[end] != '.' {
			return end
		}
		if end-1 > at {
			return end - 1
		}
		return -1
	}
}

// pieTokens is Pele's PIE_TOKENS.
var pieTokens = []common.TokenType{
	common.Keyword("showData"),
	common.Keyword("pie"),
	{Name: ":", Literal: ":"},
	{Name: "NUMBER_PIE", First: "-0123456789", Scanner: numberScanner},
	common.AccDescrToken,
	common.AccTitleToken,
	common.TitleToken,
	common.StringToken,
	common.NewlineToken,
	common.WhitespaceToken,
	common.YAMLToken,
	common.DirectiveToken,
	common.CommentToken,
}

var pieTable = common.Prepare(pieTokens)

// section is a section of the tree, Pele's PieSection.
type section struct {
	label string
	value float64
}

// ast is Pele's PieAst; a string Pele leaves undefined has its Has field unset.
type ast struct {
	showData                           bool
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	sections                           []section
}

// number is JavaScript's Number() of a NUMBER_PIE token, which may have too many digits for a
// float64 to hold, and is then infinite.
func number(text string) float64 {
	f, _ := strconv.ParseFloat(text, 64)
	return f
}

// parsePie parses a pie chart's text into its tree, as Pele's parsePie does.
func parsePie(src string) (tree *ast, err error) {
	toks, err := common.Tokenize(src, pieTable, "pie")
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
	r := &common.Reader{Tokens: toks, Types: pieTokens, Diagram: "pie"}
	tree = &ast{}

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

	for r.Kind() == tNewline {
		r.I++
	}
	r.Expect(tPie)
	tree.showData = r.Accept(tShowData)
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
		case tString:
			label := common.StringValue(r.Take())
			r.Expect(tColon)
			value := number(r.Expect(tNumber))
			tree.sections = append(tree.sections, section{label, value})
			endOfLine()
		default:
			r.Fail("a section, a title, or a line break")
		}
	}
	return tree, nil
}
