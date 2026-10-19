package wardley

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The kinds of token, by index in wardleyTokens, as Pele's T, and those of the hidden types
// after them.
const (
	tLBracket = iota
	tComma
	tRBracket
	tAt
	tSlash
	tLParen
	tRParen
	tMinus
	tLBrace
	tRBrace
	tNumber
	tArrow
	tLinkPort
	tLinkArrow
	tLinkLabel
	tStrategy
	tWardley
	tSize
	tEvolution
	tAnchor
	tComponent
	tLabel
	tInertia
	tEvolve
	tPipeline
	tNote
	tAnnotations
	tAnnotation
	tAccelerator
	tDeaccelerator
	tName
	tWS
	tAccDescr
	tAccTitle
	tTitle
	tInt
	tString
	tID
	tNewline
	tWhitespace
	tYAML
	tDirective
	tComment

	// tEnd is the kind of the token after the last, at the end of the text.
	tEnd = -1
)

// digits is the end of the run of ASCII digits from byte i, as Pele's digits finds it.
func digits(s string, i int) int {
	for i < len(s) && chars.IsDigit(int(s[i])) {
		i++
	}
	return i
}

// numberScanner answers as WARDLEY_NUMBER's pattern, /[0-9]+\.[0-9]+/y, does, as Pele's
// numberScanner does. The pattern reads a whole run of digits before it finds there is no
// fraction, and would do so again from every digit of a long run, so the run is remembered.
func numberScanner() func(string, int) int {
	from, end := 0, 0
	return func(s string, p int) int {
		if p < from || p >= end {
			from = p
			end = digits(s, p)
		}
		if end == p || end >= len(s) || s[end] != '.' {
			return -1
		}
		if stop := digits(s, end+1); stop > end+1 {
			return stop
		}
		return -1
	}
}

// arrowToken is Pele's ARROW, /->/y.
var arrowToken = common.TokenType{Name: "ARROW", First: "-", Match: func(s string, at int) int {
	if strings.HasPrefix(s[at:], "->") {
		return at + 2
	}
	return -1
}}

// linkArrowToken is Pele's LINK_ARROW,
// /-->|-\.->|>|\+'[^']*'<>|\+'[^']*'<|\+'[^']*'>/y. [^']* can only end at the first quote
// after the opening one, so each of the last three alternatives looks at what follows that
// quote; a label may run over lines.
var linkArrowToken = common.TokenType{Name: "LINK_ARROW", First: "->+", Match: func(s string, at int) int {
	rest := s[at:]
	switch {
	case strings.HasPrefix(rest, "-->"):
		return at + 3
	case strings.HasPrefix(rest, "-.->"):
		return at + 4
	case strings.HasPrefix(rest, ">"):
		return at + 1
	case !strings.HasPrefix(rest, "+'"):
		return -1
	}
	q := strings.IndexByte(s[at+2:], '\'')
	if q < 0 {
		return -1
	}
	end := at + 2 + q + 1
	switch {
	case strings.HasPrefix(s[end:], "<>"):
		return end + 2
	case strings.HasPrefix(s[end:], "<"), strings.HasPrefix(s[end:], ">"):
		return end + 1
	}
	return -1
}}

// linkPortToken is Pele's LINK_PORT, /\+<>|\+>|\+</y.
var linkPortToken = common.TokenType{Name: "LINK_PORT", First: "+", Match: func(s string, at int) int {
	rest := s[at:]
	switch {
	case strings.HasPrefix(rest, "+<>"):
		return at + 3
	case strings.HasPrefix(rest, "+>"), strings.HasPrefix(rest, "+<"):
		return at + 2
	}
	return -1
}}

// linkLabelToken is Pele's LINK_LABEL, /;[^\n\r]+/y: a semicolon and the rest of its line,
// which must not be empty.
var linkLabelToken = common.TokenType{Name: "LINK_LABEL", First: ";", Match: func(s string, at int) int {
	if s[at] != ';' {
		return -1
	}
	end := at + 1
	for end < len(s) && s[end] != '\n' && s[end] != '\r' {
		end++
	}
	if end == at+1 {
		return -1
	}
	return end
}}

// strategyToken is Pele's STRATEGY, /build|buy|outsource|market/y, which a letter after it
// does not stop: `builder` starts with one.
var strategyToken = common.TokenType{Name: "STRATEGY", First: "bom", Match: func(s string, at int) int {
	for _, w := range [...]string{"build", "buy", "outsource", "market"} {
		if strings.HasPrefix(s[at:], w) {
			return at + len(w)
		}
	}
	return -1
}}

// word is Pele's word: a keyword matched by the regular expression of its text, which holds
// no character special to one, so that it matches the text itself, followed by anything.
func word(name, text string) common.TokenType {
	return common.TokenType{Name: name, Literal: text}
}

// nameChar reports whether the character at byte i continues a name, as
// (?:[A-Za-z0-9_()&]|-(?!>)) takes it.
func nameChar(s string, i int) bool {
	c := s[i]
	switch {
	case chars.IsLetter(int(c)) || chars.IsDigit(int(c)) || c == '_' || c == '(' || c == ')' || c == '&':
		return true
	case c == '-':
		return i+1 >= len(s) || s[i+1] != '>'
	}
	return false
}

// nameRun is the end of the run of characters that continue a name from byte i.
func nameRun(s string, i int) int {
	for i < len(s) && nameChar(s, i) {
		i++
	}
	return i
}

// nameToken is Pele's NAME_WITH_SPACES,
// /(?!title\s|accTitle|accDescr)[A-Za-z](?:[A-Za-z0-9_()&]|-(?!>))*(?:[ \t]+[A-Za-z(](?:[A-Za-z0-9_()&]|-(?!>))*)*/y:
// words set apart by tabs and spaces, each word after the first starting with a letter or
// a parenthesis. A run of blanks that no such word follows is given back, whole, as each
// word's run can only end where the next character cannot continue it.
var nameToken = common.TokenType{Name: "NAME_WITH_SPACES", First: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
	Match: func(s string, at int) int {
		rest := s[at:]
		if strings.HasPrefix(rest, "title") && len(rest) > 5 && chars.IsSpace(chars.Unit(s, at+5)) ||
			strings.HasPrefix(rest, "accTitle") || strings.HasPrefix(rest, "accDescr") {
			return -1
		}
		if !chars.IsLetter(int(s[at])) {
			return -1
		}
		end := nameRun(s, at+1)
		for {
			j := end
			for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
				j++
			}
			if j == end || j == len(s) || !(chars.IsLetter(int(s[j])) || s[j] == '(') {
				return end
			}
			end = nameRun(s, j+1)
		}
	}}

// wsToken is Pele's WS, /[ \t]+/y, which is hidden.
var wsToken = common.TokenType{Name: "WS", First: "\t ", Hidden: true, Match: common.WhitespaceToken.Match}

// accDescrToken is Pele's accDescr(): ACC_DESCR, matched by a scanner that, once a braced
// description finds no closing brace, fails every later braced one without scanning the rest
// of the text again, as there is no brace further on. It matches as ACC_DESCR does.
var accDescrToken = func() common.TokenType {
	t := common.AccDescrToken
	match := t.Match
	t.Match = nil
	t.Scanner = func() func(string, int) int {
		unclosed := false
		return func(s string, at int) int {
			if unclosed && bracedAt(s, at) {
				return -1
			}
			end := match(s, at)
			if end < 0 && bracedAt(s, at) {
				unclosed = true
			}
			return end
		}
	}
	return t
}()

// bracedAt reports whether the head of a braced description, /[\t ]*accDescr\s*{/y, is at
// byte at, with no colon after the word to make it a description of one line.
func bracedAt(s string, at int) bool {
	i := at
	for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	if !strings.HasPrefix(s[i:], "accDescr") {
		return false
	}
	i += len("accDescr")
	j := i
	for j < len(s) && (s[j] == ' ' || s[j] == '\t') {
		j++
	}
	if j < len(s) && s[j] == ':' {
		return false
	}
	i = chars.SpaceRun(s, i)
	return i < len(s) && s[i] == '{'
}

// intToken is Pele's INT, /0|[1-9][0-9]*(?!\.)/y.
var intToken = func() common.TokenType {
	t := common.IntToken
	t.First = "0123456789"
	return t
}()

// idToken is Pele's ID, /[\w]([-\w]*\w)?/y: word characters and hyphens, from a word
// character to the last word character of the run.
var idToken = common.TokenType{Name: "ID", First: "_0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
	Match: func(s string, at int) int {
		if !chars.IsWord(int(s[at])) {
			return -1
		}
		end := at + 1
		for k := at + 1; k < len(s) && (chars.IsWord(int(s[k])) || s[k] == '-'); k++ {
			if s[k] != '-' {
				end = k + 1
			}
		}
		return end
	}}

// wardleyTokens is Pele's WARDLEY_TOKENS.
var wardleyTokens = []common.TokenType{
	{Name: "[", Literal: "["},
	{Name: ",", Literal: ","},
	{Name: "]", Literal: "]"},
	{Name: "@", Literal: "@"},
	{Name: "/", Literal: "/"},
	{Name: "(", Literal: "("},
	{Name: ")", Literal: ")"},
	// An arrow and front matter also start with a dash, and Langium lets the longer win.
	{Name: "-", Literal: "-", Longer: []int{tArrow, tLinkArrow, tYAML}},
	{Name: "{", Literal: "{"},
	{Name: "}", Literal: "}"},
	{Name: "WARDLEY_NUMBER", First: "0123456789", Scanner: numberScanner},
	arrowToken,
	linkPortToken,
	linkArrowToken,
	linkLabelToken,
	strategyToken,
	word("KW_WARDLEY", "wardley-beta"),
	word("KW_SIZE", "size"),
	word("KW_EVOLUTION", "evolution"),
	word("KW_ANCHOR", "anchor"),
	word("KW_COMPONENT", "component"),
	word("KW_LABEL", "label"),
	word("KW_INERTIA", "inertia"),
	word("KW_EVOLVE", "evolve"),
	word("KW_PIPELINE", "pipeline"),
	word("KW_NOTE", "note"),
	word("KW_ANNOTATIONS", "annotations"),
	word("KW_ANNOTATION", "annotation"),
	word("KW_ACCELERATOR", "accelerator"),
	word("KW_DEACCELERATOR", "deaccelerator"),
	nameToken,
	wsToken,
	accDescrToken,
	common.AccTitleToken,
	common.TitleToken,
	intToken,
	common.StringToken,
	idToken,
	common.NewlineToken,
	common.WhitespaceToken,
	common.YAMLToken,
	common.DirectiveToken,
	common.CommentToken,
}

var wardleyTable = common.Prepare(wardleyTokens)

// labelAst is Pele's LabelAst.
type labelAst struct {
	negX, negY       bool
	offsetX, offsetY float64
}

// stageAst is Pele's StageAst; hasBoundary and hasSecond say which of its parts are given.
type stageAst struct {
	name, secondName       string
	boundary               float64
	hasBoundary, hasSecond bool
}

// componentAst is Pele's ComponentAst; strategy is its decorator's, "" for none.
type componentAst struct {
	name                  string
	visibility, evolution float64
	label                 *labelAst
	strategy              string
	hasDecorator          bool
	inertia               bool
}

// pipelineComponentAst is Pele's PipelineComponentAst.
type pipelineComponentAst struct {
	name      string
	evolution float64
	label     *labelAst
}

// linkAst is Pele's LinkAst; a part Pele leaves undefined is empty, with its has field unset.
type linkAst struct {
	from, fromPort, arrow, to, toPort, linkLabel   string
	hasFromPort, hasArrow, hasToPort, hasLinkLabel bool
}

// placedAst is a statement that names a thing and places it: an anchor and a note by
// visibility and evolution, a force by x and y, as Pele's trees name them.
type placedAst struct {
	name string
	a, b float64
}

type pipelineAst struct {
	parent     string
	components []pipelineComponentAst
}

type evolveAst struct {
	component string
	target    float64
}

type annotationAst struct {
	number, x, y float64
	text         string
}

// ast is Pele's WardleyAst. Anchors and notes hold visibility in a and evolution in b;
// annotations, accelerators and deaccelerators hold x in a and y in b.
type ast struct {
	title, accTitle, accDescr          string
	hasTitle, hasAccTitle, hasAccDescr bool
	size                               *[2]float64
	evolution                          []stageAst
	anchors                            []placedAst
	components                         []componentAst
	links                              []linkAst
	evolves                            []evolveAst
	pipelines                          []pipelineAst
	notes                              []placedAst
	annotations                        [][2]float64
	annotation                         []annotationAst
	accelerators, deaccelerators       []placedAst
}

// parseWardley parses a Wardley map's text into its tree, as Pele's parseWardley does.
func parseWardley(src string) (tree *ast, err error) {
	toks, err := common.Tokenize(src, wardleyTable, "wardley")
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
	r := &common.Reader{Tokens: toks, Types: wardleyTokens, Diagram: "wardley"}
	tree = &ast{}

	at := func(kind int) bool { return r.Kind() == kind }
	isName := func() bool { return at(tString) || at(tID) || at(tName) }
	name := func() string {
		switch {
		case at(tString):
			return common.StringValue(r.Take())
		case at(tID) || at(tName):
			return r.Take()
		}
		r.Fail("a name")
		return ""
	}
	number := func() float64 { return chars.Number(r.Expect(tNumber)) }
	integer := func() float64 { return chars.ParseInt(r.Expect(tInt), 10) }
	coordinate := func() float64 {
		if at(tNumber) {
			return chars.Number(r.Take())
		}
		return chars.ParseInt(r.Expect(tInt), 10)
	}
	endOfLine := func() {
		if at(tEnd) {
			return
		}
		if !at(tNewline) {
			r.Fail("a line break or the end of input")
		}
		for at(tNewline) {
			r.I++
		}
	}
	pair := func(read func() float64) (float64, float64) {
		r.Expect(tLBracket)
		first := read()
		r.Expect(tComma)
		second := read()
		r.Expect(tRBracket)
		return first, second
	}
	label := func() *labelAst {
		r.Expect(tLabel)
		r.Expect(tLBracket)
		l := &labelAst{}
		l.negX = r.Accept(tMinus)
		l.offsetX = integer()
		r.Expect(tComma)
		l.negY = r.Accept(tMinus)
		l.offsetY = integer()
		r.Expect(tRBracket)
		return l
	}
	stage := func() stageAst {
		out := stageAst{name: name()}
		if r.Accept(tAt) {
			out.boundary, out.hasBoundary = number(), true
		}
		if r.Accept(tSlash) {
			out.secondName, out.hasSecond = name(), true
		}
		return out
	}

	for at(tNewline) {
		r.I++
	}
	r.Expect(tWardley)
	for !at(tEnd) {
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
		case tSize:
			r.I++
			width, height := pair(integer)
			endOfLine()
			tree.size = &[2]float64{width, height}
		case tEvolution:
			r.I++
			stages := []stageAst{stage()}
			r.Expect(tArrow)
			stages = append(stages, stage())
			for r.Accept(tArrow) {
				stages = append(stages, stage())
			}
			endOfLine()
			tree.evolution = stages
		case tAnchor:
			r.I++
			anchor := name()
			visibility, evolution := pair(number)
			endOfLine()
			tree.anchors = append(tree.anchors, placedAst{anchor, visibility, evolution})
		case tComponent:
			r.I++
			c := componentAst{name: name()}
			c.visibility, c.evolution = pair(number)
			if at(tLabel) {
				c.label = label()
			}
			if at(tLParen) && r.Tokens.Kinds[r.I+1] == tStrategy {
				r.I++
				c.strategy, c.hasDecorator = r.Take(), true
				r.Expect(tRParen)
			}
			if r.Accept(tInertia) {
				c.inertia = true
			} else if r.Accept(tLParen) {
				r.Expect(tInertia)
				r.Expect(tRParen)
				c.inertia = true
			}
			endOfLine()
			tree.components = append(tree.components, c)
		case tEvolve:
			r.I++
			component := name()
			target := number()
			endOfLine()
			tree.evolves = append(tree.evolves, evolveAst{component, target})
		case tPipeline:
			r.I++
			parent := name()
			r.Expect(tLBrace)
			r.Expect(tNewline)
			for at(tNewline) {
				r.I++
			}
			var components []pipelineComponentAst
			for {
				r.Expect(tComponent)
				c := pipelineComponentAst{name: name()}
				r.Expect(tLBracket)
				c.evolution = number()
				r.Expect(tRBracket)
				if at(tLabel) {
					c.label = label()
				}
				endOfLine()
				components = append(components, c)
				if !at(tComponent) {
					break
				}
			}
			r.Expect(tRBrace)
			endOfLine()
			tree.pipelines = append(tree.pipelines, pipelineAst{parent, components})
		case tNote:
			r.I++
			text := common.StringValue(r.Expect(tString))
			visibility, evolution := pair(number)
			endOfLine()
			tree.notes = append(tree.notes, placedAst{text, visibility, evolution})
		case tAnnotations:
			r.I++
			x, y := pair(coordinate)
			endOfLine()
			tree.annotations = append(tree.annotations, [2]float64{x, y})
		case tAnnotation:
			r.I++
			n := integer()
			r.Expect(tComma)
			x, y := pair(coordinate)
			text := common.StringValue(r.Expect(tString))
			endOfLine()
			tree.annotation = append(tree.annotation, annotationAst{n, x, y, text})
		case tAccelerator, tDeaccelerator:
			slow := r.Kind() == tDeaccelerator
			r.I++
			force := name()
			x, y := pair(number)
			endOfLine()
			if slow {
				tree.deaccelerators = append(tree.deaccelerators, placedAst{force, x, y})
			} else {
				tree.accelerators = append(tree.accelerators, placedAst{force, x, y})
			}
		default:
			if !isName() {
				r.Fail("a statement")
			}
			link := linkAst{from: name()}
			if at(tLinkPort) {
				link.fromPort, link.hasFromPort = r.Take(), true
			}
			if at(tLinkArrow) || at(tArrow) {
				link.arrow, link.hasArrow = r.Take(), true
			}
			link.to = name()
			if at(tLinkPort) {
				link.toPort, link.hasToPort = r.Take(), true
			}
			if at(tLinkLabel) {
				link.linkLabel, link.hasLinkLabel = chars.Trim(r.Take()[1:]), true
			}
			endOfLine()
			tree.links = append(tree.links, link)
		}
	}
	return tree, nil
}
