package agentflow

import (
	"strings"

	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// tokenSet is the set of the tokens ts, as Pele fills its tables ID, ALPHANUM, NO_TAGS,
// STYLE_PART and SEPARATOR, one flag for each token.
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
		t_MULT, t_BRKT, t_STYLE, t_LINKSTYLE, t_CLASSDEF, t_CLASS, t_CLICK, t_GRAPH, t_DIR, t_flow,
		t_connector, t_global, t_end, t_DOWN, t_UP, t_START_LINK)
	stylePartTokens = tokenSet(t_NUM, t_NODE_STRING, t_COLON, t_SPACE, t_BRKT, t_STYLE)
	separatorTokens = tokenSet(t_NEWLINE, t_SEMI, t_EOF, t_COMMENT)
)

const (
	textTokens     = "'UNICODE_TEXT', 'TEXT', 'TAGSTART'"
	separators     = "'COMMENT', 'SEMI', 'NEWLINE', 'EOF'"
	statementStart = "'COMMENT', 'SEMI', 'NEWLINE', 'SPACE', 'EOF', 'flow', 'global', 'connector', 'acc_title', " +
		"'acc_descr', 'acc_descr_multiline_value', 'STYLE', 'LINKSTYLE', 'CLASSDEF', 'CLASS', 'CLICK', 'NODE_STRING'"
)

var directions = [...]string{"TB", "BT", "RL", "LR", "TD"}

// frame is an open block, flow or global: Pele's Frame. at is the token index of its
// keyword; list and collect are its parent's.
type frame struct {
	global    bool
	at        int
	id, title *flowText
	data      string
	hasData   bool
	dataFrom  int
	dataTo    int
	list      []docItem
	collect   bool
}

// parseError carries an error out of the parser's depths.
// Pele throws the error instead.
type parseError struct{ err error }

// parser is the state of Pele's parseAgentflow: the tokens, the index of the one being read,
// and the metadata block a node statement read last, which Pele keeps in the closure's
// variables pendingData, dataFrom and dataTo.
type parser struct {
	src         string
	toks        []tok
	i           int
	db          *DB
	pendingData string
	hasPending  bool
	dataFrom    int
	dataTo      int
}

// braceTrail is source.replace(/}[^\S\n]*\n/g, '}\n'): Mermaid trims the white space between
// a closing brace and the end of its line before parsing.
// It is the first line of Pele's parseAgentflow, with RE_BRACE_TRAIL.
func braceTrail(s string) string {
	if !strings.Contains(s, "}") {
		return s
	}
	var b strings.Builder
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '}' {
			continue
		}
		j := i + 1
		for j < len(s) {
			c, size := chars.UnitAt(s, j)
			if c == '\n' || !chars.IsSpace(c) {
				break
			}
			j += size
		}
		if j >= len(s) || s[j] != '\n' {
			continue
		}
		b.WriteString(s[last : i+1])
		b.WriteByte('\n')
		last = j + 1
		i = j
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

// parseInto parses an agent flow's text into db, as Pele's parseAgentflow does. An error is
// a *diagram.Error.
func parseInto(source string, db *DB) (err error) {
	src := braceTrail(source)
	p := &parser{src: src, toks: tokenize(src), db: db}
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

// typ and text are the type and the text of the token being read, Pele's types[i] and
// texts[i].
func (p *parser) typ() token   { return p.toks[p.i].typ }
func (p *parser) text() string { return p.toks[p.i].text }

// next gives the text of the token being read and moves past it, Pele's texts[i++].
func (p *parser) next() string {
	s := p.toks[p.i].text
	p.i++
	return s
}

// check stops the parse with err when there is one. Pele's database throws its errors, where
// limu's returns them; check carries them out of the parser as a throw would.
func (p *parser) check(err error) {
	if err != nil {
		panic(parseError{err})
	}
}

// fail stops the parse with a syntax error at the token being read, as Pele's fail does: an
// error token gives the lexer's error, any other the tokens that were expected.
func (p *parser) fail(expected string) {
	t := p.toks[p.i]
	if t.typ == t_ERROR {
		panic(parseError{common.SyntaxError("agentflow", p.src, t.start, "", true)})
	}
	panic(parseError{common.SyntaxError("agentflow", p.src, t.start, "Expecting "+expected+", got '"+tokenNames[t.typ]+"'", false)})
}

// span is the span from the start of token a to the end of token b.
// It is Pele's span.
func (p *parser) span(a, b int) *loc {
	return &loc{p.toks[a].loc[0], p.toks[a].loc[1], p.toks[b].loc[2], p.toks[b].loc[3]}
}

// expect reads a token of type typ and gives its text, or fails naming typ and also, as Pele's
// expect does.
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

// separator reads the separator that ends a statement, as Pele's separator does.
func (p *parser) separator() {
	if !separatorTokens[p.typ()] {
		p.fail(separators)
	}
	p.i++
}

// run reads one token or more of set and gives their texts joined, or fails naming expected:
// the loop Pele writes out in its idString, alphaNum and style.
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

// idString reads an id of one token or more, as Pele's idString does.
func (p *parser) idString() string { return p.run(&idTokens, "'NODE_STRING'") }

// alphaNum reads a run of the tokens a callback name or an interpolation is made of, as Pele's
// alphaNum does.
func (p *parser) alphaNum() string { return p.run(&alphaNumTokens, "'NODE_STRING'") }

// isTextPart reports whether t is a token a label goes on with, as Pele's text tests it.
func isTextPart(t token) bool {
	return t == t_TEXT || t == t_TAGSTART || t == t_TAGEND || t == t_UNICODE_TEXT
}

// label reads a label, a string, a Markdown string or a run of text tokens, as Pele's text
// does.
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

// closed reads the token typ that closes a label, as Pele's closed does.
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

// textNoTags reads the id of a flow or connector, as Pele's textNoTags does: a string is taken
// as text.
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

// shapeData reads a metadata block, its tokens joined, and remembers the tokens it spans, as
// Pele's shapeData does.
func (p *parser) shapeData() string {
	p.dataFrom = p.i
	s := p.next()
	for p.typ() == t_SHAPE_DATA {
		s += p.next()
	}
	p.dataTo = p.i - 1
	return s
}

// attach gives the node of id the metadata block of tokens from to to, and widens the place
// of its statement over it.
// It is Pele's attach.
func (p *parser) attach(id, data string, from, to int) {
	loc := p.span(from, to)
	p.check(p.db.addVertex(id, nil, "", nil, nil, nil, data, true, loc))
	p.db.extendVertexMapping(id, loc)
}

// attachData gives a flow or connector its header's metadata block.
// It is Pele's attachData.
func (p *parser) attachData(id string, f *frame) {
	if !f.hasData {
		return
	}
	p.check(p.db.addVertex(id, nil, "", nil, nil, nil, f.data, true, p.span(f.dataFrom, f.dataTo)))
}

// vertex reads a node's id and the shape and label it may have, adds the node to the database
// and gives its id, as Pele's vertex does.
func (p *parser) vertex() string {
	from := p.i
	id := p.idString()
	var label *flowText
	typ := ""
	var props []Prop
	switch p.typ() {
	case t_SQS:
		p.i++
		label = p.label()
		p.closed(t_SQE)
		typ = "square"
	case t_DOUBLECIRCLESTART:
		p.i++
		label = p.label()
		p.closed(t_DOUBLECIRCLEEND)
		typ = "doublecircle"
	case t_PS:
		p.i++
		if p.typ() == t_PS {
			p.i++
			label = p.label()
			p.closed(t_PE)
			p.expect(t_PE, "")
			typ = "circle"
		} else {
			label = p.label()
			p.closed(t_PE)
			typ = "round"
		}
	case t_ELLIPSE_START:
		p.i++
		label = p.label()
		p.closed(t_ELLIPSE_END)
		typ = "ellipse"
	case t_STADIUMSTART:
		p.i++
		label = p.label()
		p.closed(t_STADIUMEND)
		typ = "stadium"
	case t_SUBROUTINESTART:
		p.i++
		label = p.label()
		p.closed(t_SUBROUTINEEND)
		typ = "subroutine"
	case t_VERTEX_WITH_PROPS_START:
		p.i++
		field := p.expect(t_NODE_STRING, "")
		p.expect(t_COLON, "")
		value := p.expect(t_NODE_STRING, "")
		p.expect(t_PIPE, "")
		label = p.label()
		p.closed(t_SQE)
		typ = "rect"
		props = []Prop{{field, value}}
	case t_CYLINDERSTART:
		p.i++
		label = p.label()
		p.closed(t_CYLINDEREND)
		typ = "cylinder"
	case t_DIAMOND_START:
		p.i++
		if p.typ() == t_DIAMOND_START {
			p.i++
			label = p.label()
			p.closed(t_DIAMOND_STOP)
			p.expect(t_DIAMOND_STOP, "")
			typ = "hexagon"
		} else {
			label = p.label()
			p.closed(t_DIAMOND_STOP)
			typ = "diamond"
		}
	case t_TAGEND:
		p.i++
		label = p.label()
		p.closed(t_SQE)
		typ = "odd"
	case t_TRAPSTART:
		p.i++
		label = p.label()
		if p.typ() == t_INVTRAPEND {
			p.i++
			typ = "lean_right"
		} else {
			p.closed(t_TRAPEND)
			typ = "trapezoid"
		}
	case t_INVTRAPSTART:
		p.i++
		label = p.label()
		if p.typ() == t_TRAPEND {
			p.i++
			typ = "lean_left"
		} else {
			p.closed(t_INVTRAPEND)
			typ = "inv_trapezoid"
		}
	}
	p.check(p.db.addVertex(id, label, typ, nil, nil, props, "", false, nil))
	p.db.addVertexMapping(id, typ == "", p.span(from, p.i-1))
	return id
}

// styledVertex reads a vertex and the `:::` class that may follow it, as Pele's styledVertex
// does.
func (p *parser) styledVertex() string {
	id := p.vertex()
	if p.typ() == t_STYLE_SEPARATOR {
		p.i++
		p.db.setClass(id, p.idString())
	}
	return id
}

// node reads nodes joined with `&` and the metadata blocks after them, as Pele's node does: a
// block before an `&` goes to the node before it, and the last is left pending for the
// statement.
func (p *parser) node() []string {
	ids := []string{p.styledVertex()}
	p.hasPending = false
	for {
		data, hasData := "", false
		if p.typ() == t_SHAPE_DATA {
			data, hasData = p.shapeData(), true
		}
		from, to := p.dataFrom, p.dataTo
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
			p.attach(ids[len(ids)-1], data, from, to)
		}
		ids = append(ids, next)
	}
}

// link reads a link, with its id and its label if it has them, as Pele's link does.
func (p *parser) link() *linkInfo {
	id, hasID := "", false
	if p.typ() == t_LINK_ID {
		id, hasID = p.next(), true
	}
	if p.typ() == t_LINK {
		info := p.db.destructLink(p.next(), "")
		info.ID, info.HasID = id, hasID
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
		info.ID, info.HasID = id, hasID
		return &info
	}
	p.fail("'LINK', 'START_LINK'")
	return nil
}

// vertexStatement reads nodes and the links between them, adding the links to the database,
// and gives the nodes from the last group to the first, as Pele's vertexStatement does.
func (p *parser) vertexStatement() []string {
	from := p.i
	stmt := p.node()
	if p.hasPending {
		p.attach(stmt[len(stmt)-1], p.pendingData, p.dataFrom, p.dataTo)
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
			p.attach(next[len(next)-1], p.pendingData, p.dataFrom, p.dataTo)
		}
		p.check(p.db.addLink(stmt, next, info))
		p.db.addEdgeMapping(next, p.span(from, p.i-1))
		nodes = append(append([]string{}, next...), nodes...)
		stmt = next
	}
}

// style reads one style of a list, as Pele's style does.
func (p *parser) style() string {
	return p.run(&stylePartTokens, "'NODE_STRING', 'NUM', 'COLON', 'BRKT'")
}

// stylesOpt reads a list of styles separated by commas, as Pele's stylesOpt does.
func (p *parser) stylesOpt() []string {
	out := []string{p.style()}
	for p.typ() == t_COMMA {
		p.i++
		out = append(out, p.style())
	}
	return out
}

// linkStyleStatement reads a `linkStyle` statement: the links it names, then an interpolation,
// styles or both, as Pele's linkStyleStatement does.
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

// linkTail reads what may follow the url of a click: a target, or a tooltip and then a target,
// as Pele's linkTail does.
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

// clickStatement reads a `click` statement, a callback or a link with its tooltip, as Pele's
// clickStatement does.
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

// header reads `flow` or `connector` up to the separator: an id, an optional bracketed title,
// optional metadata.
// It is Pele's header.
func (p *parser) header(at int) *frame {
	p.expect(t_SPACE, separators)
	id := p.textNoTags()
	title := &flowText{"", LabelText}
	if p.typ() == t_SQS {
		p.i++
		title = p.label()
		p.closed(t_SQE)
	}
	f := &frame{at: at, id: id, title: title}
	if p.typ() == t_SHAPE_DATA {
		f.data, f.hasData = p.shapeData(), true
	}
	f.dataFrom, f.dataTo = p.dataFrom, p.dataTo
	if !separatorTokens[p.typ()] {
		p.fail(separators + ", 'SHAPE_DATA'")
	}
	return f
}

// statement parses a statement, and gives what it adds to a flow's body, if anything.
// It is Pele's statement, which gives an empty list where this gives false.
func (p *parser) statement() (docItem, bool) {
	t := p.typ()
	switch t {
	case t_STYLE:
		p.i++
		p.expect(t_SPACE, "")
		id := p.idString()
		p.expect(t_SPACE, "")
		p.check(p.db.addVertex(id, nil, "", p.stylesOpt(), nil, nil, "", false, nil))
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
	case t_connector:
		at := p.i
		p.i++
		f := p.header(at)
		sep := p.i
		p.i++
		id := p.db.addConnector(f.id, f.title)
		p.attachData(id, f)
		p.db.addConnectorMapping(f.id, p.span(f.at, f.at), p.span(sep, sep))
		return docItem{kind: itemString, text: id}, true
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

// document parses the statements. Blocks nest without recursion: each open one keeps the
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
		switch {
		case t == t_end:
			if len(open) == 0 {
				p.finish(open)
				return
			}
			f := open[len(open)-1]
			open = open[:len(open)-1]
			at := p.i
			p.i++
			if f.global {
				p.db.addGlobal(list)
				list, collect = f.list, f.collect
				continue
			}
			id := p.db.addSubGraph(f.id, list, f.title)
			p.attachData(id, f)
			p.db.addSubgraphMapping(f.id, p.span(f.at, f.at), p.span(at, at))
			list, collect = f.list, f.collect
			if collect {
				list = append(list, docItem{kind: itemString, text: id})
			}
		case t == t_SEMI || t == t_NEWLINE || t == t_SPACE || t == t_EOF:
			if collect {
				list = append(list, docItem{kind: itemString, text: p.text()})
			}
			p.i++
		case t == t_COMMENT:
			p.i++
		case t == t_flow || t == t_global:
			at := p.i
			p.i++
			var f *frame
			if t == t_global || separatorTokens[p.typ()] {
				f = &frame{global: t == t_global, at: at, list: list, collect: collect}
				p.separator()
			} else {
				f = p.header(at)
				f.list, f.collect = list, collect
				p.i++
			}
			open = append(open, f)
			list = nil
			collect = true
		default:
			item, ok := p.statement()
			if collect && ok {
				list = append(list, item)
			}
		}
	}
	p.finish(open)
}

// finish fails when a block is still open where the statements end, as the last line of
// Pele's document does.
func (p *parser) finish(open []*frame) {
	if len(open) > 0 {
		p.fail("'end'")
	}
}

// parse reads the header of the chart, its direction and its statements, as the end of Pele's
// parseAgentflow does after its closures.
func (p *parser) parse() {
	for p.typ() == t_SPACE || p.typ() == t_NEWLINE || p.typ() == t_COMMENT {
		p.i++
	}
	p.expect(t_GRAPH, "'SPACE', 'NEWLINE', 'COMMENT'")
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
		} else if t != t_SEMI && t != t_NEWLINE && t != t_COMMENT {
			p.fail("'COMMENT', 'SEMI', 'NEWLINE', 'SPACE'")
		}
		p.i++
		p.db.setDirection(dir)
	}
	p.document()
	if p.typ() != t_END {
		p.fail(statementStart)
	}
}
