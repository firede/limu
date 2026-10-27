package flowchart

import (
	"strings"

	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/diagram/internal/edit"
	"github.com/firede/limu/internal/chars"
)

// tokenSet is the set of the token types ts, as the loops that fill Pele's ID, ALPHANUM,
// NO_TAGS and STYLE_PART tables make one.
func tokenSet(ts ...token) (set [t_EOF + 1]bool) {
	for _, t := range ts {
		set[t] = true
	}
	return
}

var (
	idTokens = tokenSet(t_NUM, t_NODE_STRING, t_DOWN, t_MINUS, t_DEFAULT, t_COMMA, t_COLON, t_AMP,
		t_BRKT, t_MULT, t_UNICODE_TEXT)
	alphaNumTokens = tokenSet(t_NUM, t_UNICODE_TEXT, t_NODE_STRING, t_DIR, t_DOWN, t_MINUS, t_COMMA,
		t_COLON, t_AMP, t_BRKT, t_MULT)
	noTagsTokens = tokenSet(t_NUM, t_NODE_STRING, t_SPACE, t_MINUS, t_AMP, t_UNICODE_TEXT, t_COLON,
		t_MULT, t_BRKT, t_STYLE, t_LINKSTYLE, t_CLASSDEF, t_CLASS, t_CLICK, t_GRAPH, t_DIR, t_subgraph,
		t_end, t_DOWN, t_UP, t_START_LINK)
	stylePartTokens = tokenSet(t_NUM, t_NODE_STRING, t_COLON, t_SPACE, t_BRKT, t_STYLE)
)

const (
	textTokens     = "'UNICODE_TEXT', 'TEXT', 'TAGSTART'"
	statementStart = "'SEMI', 'NEWLINE', 'SPACE', 'EOF', 'subgraph', 'acc_title', 'acc_descr', " +
		"'acc_descr_multiline_value', 'STYLE', 'LINKSTYLE', 'CLASSDEF', 'CLASS', 'CLICK', 'NODE_STRING'"
)

var directions = [...]string{"TB", "BT", "RL", "LR", "TD"}

// frame is an open subgraph: its id and title, and the list and collect flag of its parent.
type frame struct {
	id, title *flowText
	list      []docItem
	collect   bool
}

// parseError carries an error out of the parser's depths.
type parseError struct{ err error }

type parser struct {
	src         string      // the text the lexer reads
	source      string      // the text the parser was given, where errors are located
	trims       []edit.Edit // what braceTrail replaced in source to make src
	toks        []tok
	i           int
	db          *DB
	pendingData string
	hasPending  bool
}

// braceTrail is source.replace(/}\s*\n/g, '}\n'): Mermaid trims the white space between a
// closing brace and the end of its line before parsing. It also gives each replacement it
// makes, as replacements(source, RE_BRACE_TRAIL, 2) records them.
// Pele's parseFlowchart does this with RE_BRACE_TRAIL.
func braceTrail(s string) (string, []edit.Edit) {
	if !strings.Contains(s, "}") {
		return s, nil
	}
	var b strings.Builder
	var edits []edit.Edit
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '}' {
			continue
		}
		end := chars.SpaceRun(s, i+1)
		nl := strings.LastIndexByte(s[i+1:end], '\n')
		if nl == -1 {
			continue
		}
		b.WriteString(s[last : i+1])
		b.WriteByte('\n')
		last = i + 1 + nl + 1
		edits = append(edits, edit.Edit{From: i, To: last, Text: "}\n"})
		i = last - 1
	}
	if last == 0 {
		return s, nil
	}
	b.WriteString(s[last:])
	return b.String(), edits
}

// parseInto parses a flowchart's text into db, as Pele's parseFlowchart does. An error is a
// *diagram.Error.
func parseInto(source string, db *DB) (err error) {
	src, trims := braceTrail(source)
	p := &parser{src: src, source: source, trims: trims, toks: tokenize(src), db: db}
	defer func() {
		if r := recover(); r != nil {
			pe, ok := r.(parseError)
			if !ok {
				panic(r)
			}
			err = pe.err
		}
	}()
	p.parse()
	return nil
}

// typ and text are the type and the text of the token the parser is at, Pele's types[i] and
// texts[i].
func (p *parser) typ() token   { return p.toks[p.i].typ }
func (p *parser) text() string { return p.toks[p.i].text }

// next gives the text of the token the parser is at and moves past it, Pele's texts[i++].
func (p *parser) next() string {
	s := p.toks[p.i].text
	p.i++
	return s
}

// check stops the parse with err, if it is not nil. It is limu's own: Pele's db throws its
// errors, where limu's DB returns them.
func (p *parser) check(err error) {
	if err != nil {
		panic(parseError{err})
	}
}

// fail stops the parse with a syntax error at the token the parser is at, saying what was
// expected there, as Pele's fail does. The error is located in the text the parser was given,
// before the blanks after a brace were trimmed.
func (p *parser) fail(expected string) {
	t := p.toks[p.i]
	at := t.start
	if p.src != p.source {
		at = edit.InputOffset([][]edit.Edit{p.trims}, at)
	}
	if t.typ == t_ERROR {
		panic(parseError{common.SyntaxError("flowchart", p.source, at, "", true)})
	}
	panic(parseError{common.SyntaxError("flowchart", p.source, at, "Expecting "+expected+", got '"+tokenNames[t.typ]+"'", false)})
}

// expect gives the text of the token the parser is at and moves past it, or fails if the
// token is not of type typ, naming also among what was expected, as Pele's expect does.
func (p *parser) expect(typ token, also string) string {
	if p.typ() != typ {
		exp := "'" + tokenNames[typ] + "'"
		if also != "" {
			exp += ", " + also
		}
		p.fail(exp)
	}
	return p.next()
}

// separator moves past the newline, semicolon or end of text that ends a statement, as
// Pele's separator does.
func (p *parser) separator() {
	t := p.typ()
	if t != t_NEWLINE && t != t_SEMI && t != t_EOF {
		p.fail("'SEMI', 'NEWLINE', 'EOF'")
	}
	p.i++
}

// run joins the texts of the run of tokens whose types are in set, which must have at least
// one, failing with expected where it has none. It is the loop Pele's idString, alphaNum and
// style each write out.
func (p *parser) run(set *[t_EOF + 1]bool, expected string) string {
	if !set[p.typ()] {
		p.fail(expected)
	}
	s := p.next()
	for set[p.typ()] {
		s += p.next()
	}
	return s
}

// idString joins the run of tokens that make an id, as Pele's idString does.
func (p *parser) idString() string { return p.run(&idTokens, "'NODE_STRING'") }

// alphaNum joins the run of tokens that make a name, as Pele's alphaNum does.
func (p *parser) alphaNum() string { return p.run(&alphaNumTokens, "'NODE_STRING'") }

// isTextPart reports whether a token of type t goes on a node's label, the test Pele's text
// makes of each token.
func isTextPart(t token) bool {
	return t == t_TEXT || t == t_TAGSTART || t == t_TAGEND || t == t_UNICODE_TEXT
}

// label reads a node's label, a string, a Markdown string or a run of text tokens, as Pele's
// text does.
func (p *parser) label() *flowText {
	var lt LabelType
	switch t := p.typ(); {
	case t == t_STR:
		lt = LabelString
	case t == t_MD_STR:
		lt = LabelMarkdown
	case isTextPart(t):
		lt = LabelText
	default:
		p.fail(textTokens + ", 'TAGEND', 'STR', 'MD_STR'")
	}
	s := p.next()
	for isTextPart(p.typ()) {
		s += p.next()
	}
	return &flowText{s, lt}
}

// closed expects the token of type typ that closes a label, as Pele's closed does.
func (p *parser) closed(typ token) { p.expect(typ, textTokens) }

// edgeText reads the label of a link written inside it, as Pele's edgeText does.
func (p *parser) edgeText() *flowText {
	var lt LabelType
	switch t := p.typ(); t {
	case t_STR:
		lt = LabelString
	case t_MD_STR:
		lt = LabelMarkdown
	case t_EDGE_TEXT, t_UNICODE_TEXT:
		lt = LabelText
	default:
		p.fail("'EDGE_TEXT', 'STR', 'MD_STR'")
	}
	s := p.next()
	for p.typ() == t_EDGE_TEXT || p.typ() == t_UNICODE_TEXT {
		s += p.next()
	}
	return &flowText{s, lt}
}

// textNoTags reads a subgraph's id, which takes no tags, as Pele's textNoTags does.
func (p *parser) textNoTags() *flowText {
	var lt LabelType
	switch t := p.typ(); {
	case t == t_STR:
		lt = LabelText
	case t == t_MD_STR:
		lt = LabelMarkdown
	case noTagsTokens[t]:
		lt = LabelText
	default:
		p.fail("'NODE_STRING', 'STR', 'MD_STR'")
	}
	s := p.next()
	for noTagsTokens[p.typ()] {
		s += p.next()
	}
	return &flowText{s, lt}
}

// shapeData joins the run of a node's shape data, as Pele's shapeData does.
func (p *parser) shapeData() string {
	s := p.next()
	for p.typ() == t_SHAPE_DATA {
		s += p.next()
	}
	return s
}

// add adds a vertex with a label and a shape, the call to db.addVertex that each case of
// Pele's vertex makes.
func (p *parser) add(id string, label *flowText, typ string) {
	p.check(p.db.addVertex(id, label, typ, nil, nil, "", nil, "", false))
}

// addData gives a vertex its shape data, the call to db.addVertex that Pele's node and
// vertexStatement make.
func (p *parser) addData(id, data string) {
	p.check(p.db.addVertex(id, nil, "", nil, nil, "", nil, data, true))
}

// vertex reads a node's id and its label in the brackets that give its shape, and adds the
// vertex, as Pele's vertex does. It gives the id.
func (p *parser) vertex() string {
	id := p.idString()
	var label *flowText
	switch p.typ() {
	case t_SQS:
		p.i++
		label = p.label()
		p.closed(t_SQE)
		p.add(id, label, "square")
	case t_DOUBLECIRCLESTART:
		p.i++
		label = p.label()
		p.closed(t_DOUBLECIRCLEEND)
		p.add(id, label, "doublecircle")
	case t_PS:
		p.i++
		if p.typ() == t_PS {
			p.i++
			label = p.label()
			p.closed(t_PE)
			p.expect(t_PE, "")
			p.add(id, label, "circle")
		} else {
			label = p.label()
			p.closed(t_PE)
			p.add(id, label, "round")
		}
	case t_ELLIPSE_START:
		p.i++
		label = p.label()
		p.closed(t_ELLIPSE_END)
		p.add(id, label, "ellipse")
	case t_STADIUMSTART:
		p.i++
		label = p.label()
		p.closed(t_STADIUMEND)
		p.add(id, label, "stadium")
	case t_SUBROUTINESTART:
		p.i++
		label = p.label()
		p.closed(t_SUBROUTINEEND)
		p.add(id, label, "subroutine")
	case t_VERTEX_WITH_PROPS_START:
		p.i++
		field := p.expect(t_NODE_STRING, "")
		p.expect(t_COLON, "")
		value := p.expect(t_NODE_STRING, "")
		p.expect(t_PIPE, "")
		label = p.label()
		p.closed(t_SQE)
		p.check(p.db.addVertex(id, label, "rect", nil, nil, "", []Prop{{field, value}}, "", false))
	case t_CYLINDERSTART:
		p.i++
		label = p.label()
		p.closed(t_CYLINDEREND)
		p.add(id, label, "cylinder")
	case t_DIAMOND_START:
		p.i++
		if p.typ() == t_DIAMOND_START {
			p.i++
			label = p.label()
			p.closed(t_DIAMOND_STOP)
			p.expect(t_DIAMOND_STOP, "")
			p.add(id, label, "hexagon")
		} else {
			label = p.label()
			p.closed(t_DIAMOND_STOP)
			p.add(id, label, "diamond")
		}
	case t_TAGEND:
		p.i++
		label = p.label()
		p.closed(t_SQE)
		p.add(id, label, "odd")
	case t_TRAPSTART:
		p.i++
		label = p.label()
		if p.typ() == t_INVTRAPEND {
			p.i++
			p.add(id, label, "lean_right")
		} else {
			p.closed(t_TRAPEND)
			p.add(id, label, "trapezoid")
		}
	case t_INVTRAPSTART:
		p.i++
		label = p.label()
		if p.typ() == t_TRAPEND {
			p.i++
			p.add(id, label, "lean_left")
		} else {
			p.closed(t_INVTRAPEND)
			p.add(id, label, "inv_trapezoid")
		}
	default:
		p.add(id, nil, "")
	}
	return id
}

// styledVertex reads a vertex and the class after `:::`, if any, as Pele's styledVertex does.
func (p *parser) styledVertex() string {
	id := p.vertex()
	if p.typ() == t_STYLE_SEPARATOR {
		p.i++
		p.db.setClass(id, p.idString())
	}
	return id
}

// node reads a node or a group of them joined by `&`, as Pele's node does: shape data before
// an `&` goes to the vertex it follows, and that after the last vertex is left in
// p.pendingData for the caller.
func (p *parser) node() []string {
	ids := []string{p.styledVertex()}
	p.hasPending = false
	for {
		data, hasData := "", false
		if p.typ() == t_SHAPE_DATA {
			data, hasData = p.shapeData(), true
		}
		if p.typ() != t_SPACE {
			p.pendingData, p.hasPending = data, hasData
			return ids
		}
		for p.typ() == t_SPACE {
			p.i++
		}
		if p.typ() != t_AMP {
			if hasData {
				p.fail("'AMP'")
			}
			return ids
		}
		p.i++
		if p.typ() != t_SPACE {
			p.fail("'SPACE'")
		}
		for p.typ() == t_SPACE {
			p.i++
		}
		next := p.styledVertex()
		if hasData {
			p.addData(ids[len(ids)-1], data)
		}
		ids = append(ids, next)
	}
}

// link reads a link, with its id and its label, as Pele's link does.
func (p *parser) link() *linkInfo {
	id := ""
	if p.typ() == t_LINK_ID {
		id = p.next()
	}
	if p.typ() == t_LINK {
		info := p.db.destructLink(p.next(), "")
		info.ID = id
		if p.typ() == t_PIPE {
			p.i++
			info.Text = p.label()
			p.closed(t_PIPE)
			if p.typ() == t_SPACE {
				p.i++
			}
		}
		return &info
	}
	if p.typ() == t_START_LINK {
		start := p.next()
		label := p.edgeText()
		end := p.expect(t_LINK, "'EDGE_TEXT'")
		info := p.db.destructLink(end, start)
		info.Text = label
		info.ID = id
		return &info
	}
	p.fail("'LINK', 'START_LINK'")
	return nil
}

// vertexStatement reads a chain of nodes and the links between them, adding the links, as
// Pele's vertexStatement does. It gives the ids of the nodes, the last group first.
func (p *parser) vertexStatement() []string {
	stmt := p.node()
	if p.hasPending {
		p.addData(stmt[len(stmt)-1], p.pendingData)
	}
	nodes := stmt
	for {
		t := p.typ()
		if t != t_LINK && t != t_START_LINK && t != t_LINK_ID {
			return nodes
		}
		info := p.link()
		next := p.node()
		if p.hasPending {
			p.addData(next[len(next)-1], p.pendingData)
		}
		p.check(p.db.addLink(stmt, next, info))
		nodes = append(append([]string{}, next...), nodes...)
		stmt = next
	}
}

// style joins the run of tokens of one style, as Pele's style does.
func (p *parser) style() string {
	return p.run(&stylePartTokens, "'NODE_STRING', 'NUM', 'COLON', 'BRKT'")
}

// stylesOpt reads a list of styles split by commas, as Pele's stylesOpt does.
func (p *parser) stylesOpt() []string {
	out := []string{p.style()}
	for p.typ() == t_COMMA {
		p.i++
		out = append(out, p.style())
	}
	return out
}

// linkStyleStatement reads a linkStyle statement, setting the styles and the curve of the
// links it names, as Pele's linkStyleStatement does.
func (p *parser) linkStyleStatement() {
	p.i++
	p.expect(t_SPACE, "")
	var positions []string
	if p.typ() == t_DEFAULT {
		positions = []string{p.next()}
	} else {
		positions = []string{p.expect(t_NUM, "'DEFAULT'")}
		for p.typ() == t_COMMA {
			p.i++
			positions = append(positions, p.expect(t_NUM, ""))
		}
	}
	p.expect(t_SPACE, "")
	if p.typ() != t_INTERPOLATE {
		p.check(p.db.updateLink(positions, p.stylesOpt()))
		return
	}
	p.i++
	p.expect(t_SPACE, "")
	interpolate := p.alphaNum()
	if p.typ() == t_SPACE {
		p.i++
		styles := p.stylesOpt()
		p.check(p.db.updateLinkInterpolate(positions, interpolate))
		p.check(p.db.updateLink(positions, styles))
	} else {
		p.check(p.db.updateLinkInterpolate(positions, interpolate))
	}
}

// linkTail reads what follows the url of a click statement, a tooltip and a target, and sets
// the node's link, as Pele's linkTail does.
func (p *parser) linkTail(id, url string) {
	if p.typ() != t_SPACE {
		p.db.setLink(id, url, "")
		return
	}
	p.i++
	if p.typ() == t_LINK_TARGET {
		p.db.setLink(id, url, p.next())
		return
	}
	tooltip := p.expect(t_STR, "'LINK_TARGET'")
	if p.typ() == t_SPACE {
		p.i++
		p.db.setLink(id, url, p.expect(t_LINK_TARGET, ""))
	} else {
		p.db.setLink(id, url, "")
	}
	p.db.setTooltip(id, tooltip)
}

// clickStatement reads a click statement, as Pele's clickStatement does. The callback's name
// and arguments are read and left, as limu calls no callbacks.
func (p *parser) clickStatement() {
	id := p.next()
	switch p.typ() {
	case t_CALLBACKNAME:
		p.i++
		if p.typ() == t_CALLBACKARGS {
			p.i++
		}
		tooltip, hasTooltip := "", false
		if p.typ() == t_SPACE {
			p.i++
			tooltip, hasTooltip = p.expect(t_STR, ""), true
		}
		p.db.setClickEvent(id)
		if hasTooltip {
			p.db.setTooltip(id, tooltip)
		}
	case t_HREF:
		p.i++
		p.linkTail(id, p.expect(t_STR, ""))
	case t_STR:
		p.linkTail(id, p.next())
	default:
		p.alphaNum()
		if p.typ() == t_SPACE {
			p.i++
			tooltip := p.expect(t_STR, "")
			p.db.setClickEvent(id)
			p.db.setTooltip(id, tooltip)
		} else {
			p.db.setClickEvent(id)
		}
	}
}

// subgraphHeader reads the line that opens a subgraph, its id and its title, as Pele's
// subgraphHeader does.
func (p *parser) subgraphHeader() *frame {
	p.i++
	t := p.typ()
	if t == t_NEWLINE || t == t_SEMI || t == t_EOF {
		p.i++
		return &frame{}
	}
	p.expect(t_SPACE, "'SEMI', 'NEWLINE', 'EOF'")
	id := p.textNoTags()
	title := id
	if p.typ() == t_SQS {
		p.i++
		title = p.label()
		p.closed(t_SQE)
	}
	t = p.typ()
	if t != t_NEWLINE && t != t_SEMI && t != t_EOF {
		p.fail("'SEMI', 'NEWLINE', 'EOF', 'SQS'")
	}
	p.i++
	return &frame{id: id, title: title}
}

// statement parses a statement, and gives what it adds to a subgraph's body, if anything.
// It is Pele's statement, whose empty list of items is the false here.
func (p *parser) statement() (docItem, bool) {
	t := p.typ()
	switch t {
	case t_STYLE:
		p.i++
		p.expect(t_SPACE, "")
		id := p.idString()
		p.expect(t_SPACE, "")
		p.check(p.db.addVertex(id, nil, "", p.stylesOpt(), nil, "", nil, "", false))
		p.separator()
		return docItem{}, false
	case t_LINKSTYLE:
		p.linkStyleStatement()
		p.separator()
		return docItem{}, false
	case t_CLASSDEF:
		p.i++
		p.expect(t_SPACE, "")
		id := p.idString()
		p.expect(t_SPACE, "")
		p.db.addClass(id, p.stylesOpt())
		p.separator()
		return docItem{}, false
	case t_CLASS:
		p.i++
		p.expect(t_SPACE, "")
		ids := p.idString()
		p.expect(t_SPACE, "")
		p.db.setClass(ids, p.idString())
		p.separator()
		return docItem{}, false
	case t_CLICK:
		p.clickStatement()
		p.separator()
		return docItem{}, false
	case t_direction_tb, t_direction_bt, t_direction_rl, t_direction_lr, t_direction_td:
		p.i++
		return docItem{kind: itemDir, text: directions[t-t_direction_tb]}, true
	case t_acc_title:
		p.i++
		value := chars.Trim(p.expect(t_acc_title_value, ""))
		p.db.setAccTitle(value)
		return docItem{kind: itemString, text: value}, true
	case t_acc_descr:
		p.i++
		value := chars.Trim(p.expect(t_acc_descr_value, ""))
		p.db.setAccDescription(value)
		return docItem{kind: itemString, text: value}, true
	case t_acc_descr_multiline_value:
		value := chars.Trim(p.next())
		p.db.setAccDescription(value)
		return docItem{kind: itemString, text: value}, true
	}
	if !idTokens[t] {
		p.fail(statementStart)
	}
	nodes := p.vertexStatement()
	p.separator()
	return docItem{kind: itemList, list: nodes}, len(nodes) > 0
}

// document parses the statements. Subgraphs nest without recursion: each open one keeps the
// list and collect flag of its parent.
// It is Pele's document.
func (p *parser) document() {
	var open []*frame
	var list []docItem
	collect := false
	for {
		t := p.typ()
		if t == t_END {
			break
		}
		if t == t_end {
			if len(open) == 0 {
				break
			}
			f := open[len(open)-1]
			open = open[:len(open)-1]
			p.i++
			id := p.db.addSubGraph(f.id, list, f.title)
			list = f.list
			collect = f.collect
			if collect {
				list = append(list, docItem{kind: itemString, text: id})
			}
		} else if t == t_SEMI || t == t_NEWLINE || t == t_SPACE || t == t_EOF {
			if collect {
				list = append(list, docItem{kind: itemString, text: p.text()})
			}
			p.i++
		} else if t == t_subgraph {
			f := p.subgraphHeader()
			f.list = list
			f.collect = collect
			open = append(open, f)
			list = nil
			collect = true
		} else {
			item, ok := p.statement()
			if collect && ok {
				list = append(list, item)
			}
		}
	}
	if len(open) > 0 {
		p.fail("'end'")
	}
}

// parse reads the header and its direction, then the statements, as the end of Pele's
// parseFlowchart does after it defines its readers.
func (p *parser) parse() {
	for p.typ() == t_SPACE || p.typ() == t_NEWLINE {
		p.i++
	}
	p.expect(t_GRAPH, "'SPACE', 'NEWLINE'")
	if p.typ() == t_NODIR {
		p.i++
		p.db.setDirection("TB")
	} else {
		dir := p.expect(t_DIR, "'NODIR'")
		t := p.typ()
		if t == t_SPACE {
			for p.typ() == t_SPACE {
				p.i++
			}
			if p.typ() != t_NEWLINE {
				p.fail("'NEWLINE', 'SPACE'")
			}
		} else if t != t_SEMI && t != t_NEWLINE {
			p.fail("'SEMI', 'NEWLINE', 'SPACE'")
		}
		p.i++
		p.db.setDirection(dir)
	}
	p.document()
	if p.typ() != t_END {
		p.fail(statementStart)
	}
}
