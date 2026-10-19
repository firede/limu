package flowchart

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

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
	id, title *Text
	list      []docItem
	collect   bool
}

// parseError carries an error out of the parser's depths.
type parseError struct{ err error }

type parser struct {
	src         string
	toks        []tok
	i           int
	db          *DB
	pendingData string
	hasPending  bool
}

// braceTrail is source.replace(/}\s*\n/g, '}\n'): Mermaid trims the white space between a
// closing brace and the end of its line before parsing.
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
		end := chars.SpaceRun(s, i+1)
		nl := strings.LastIndexByte(s[i+1:end], '\n')
		if nl == -1 {
			continue
		}
		b.WriteString(s[last : i+1])
		b.WriteByte('\n')
		last = i + 1 + nl + 1
		i = last - 1
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

// ParseInto parses a flowchart's text into db, as Pele's parseFlowchart does. An error is a
// *diagram.Error.
func ParseInto(source string, db *DB) (err error) {
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

func (p *parser) typ() token   { return p.toks[p.i].typ }
func (p *parser) text() string { return p.toks[p.i].text }

func (p *parser) next() string {
	s := p.toks[p.i].text
	p.i++
	return s
}

func (p *parser) check(err error) {
	if err != nil {
		panic(parseError{err})
	}
}

func (p *parser) fail(expected string) {
	t := p.toks[p.i]
	if t.typ == t_ERROR {
		panic(parseError{diagram.SyntaxError("flowchart", p.src, t.start, "", true)})
	}
	panic(parseError{diagram.SyntaxError("flowchart", p.src, t.start, "Expecting "+expected+", got '"+tokenNames[t.typ]+"'", false)})
}

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

func (p *parser) separator() {
	t := p.typ()
	if t != t_NEWLINE && t != t_SEMI && t != t_EOF {
		p.fail("'SEMI', 'NEWLINE', 'EOF'")
	}
	p.i++
}

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

func (p *parser) idString() string { return p.run(&idTokens, "'NODE_STRING'") }

func (p *parser) alphaNum() string { return p.run(&alphaNumTokens, "'NODE_STRING'") }

func isTextPart(t token) bool {
	return t == t_TEXT || t == t_TAGSTART || t == t_TAGEND || t == t_UNICODE_TEXT
}

func (p *parser) label() *Text {
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
	return &Text{s, lt}
}

func (p *parser) closed(typ token) { p.expect(typ, textTokens) }

func (p *parser) edgeText() *Text {
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
	return &Text{s, lt}
}

func (p *parser) textNoTags() *Text {
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
	return &Text{s, lt}
}

func (p *parser) shapeData() string {
	s := p.next()
	for p.typ() == t_SHAPE_DATA {
		s += p.next()
	}
	return s
}

func (p *parser) add(id string, label *Text, typ string) {
	p.check(p.db.AddVertex(id, label, typ, nil, nil, "", nil, "", false))
}

func (p *parser) addData(id, data string) {
	p.check(p.db.AddVertex(id, nil, "", nil, nil, "", nil, data, true))
}

func (p *parser) vertex() string {
	id := p.idString()
	var label *Text
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
		p.check(p.db.AddVertex(id, label, "rect", nil, nil, "", []Prop{{field, value}}, "", false))
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

func (p *parser) styledVertex() string {
	id := p.vertex()
	if p.typ() == t_STYLE_SEPARATOR {
		p.i++
		p.db.SetClass(id, p.idString())
	}
	return id
}

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

func (p *parser) link() *LinkInfo {
	id := ""
	if p.typ() == t_LINK_ID {
		id = p.next()
	}
	if p.typ() == t_LINK {
		info := p.db.DestructLink(p.next(), "")
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
		info := p.db.DestructLink(end, start)
		info.Text = label
		info.ID = id
		return &info
	}
	p.fail("'LINK', 'START_LINK'")
	return nil
}

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
		p.check(p.db.AddLink(stmt, next, info))
		nodes = append(append([]string{}, next...), nodes...)
		stmt = next
	}
}

func (p *parser) style() string {
	return p.run(&stylePartTokens, "'NODE_STRING', 'NUM', 'COLON', 'BRKT'")
}

func (p *parser) stylesOpt() []string {
	out := []string{p.style()}
	for p.typ() == t_COMMA {
		p.i++
		out = append(out, p.style())
	}
	return out
}

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
		p.check(p.db.UpdateLink(positions, p.stylesOpt()))
		return
	}
	p.i++
	p.expect(t_SPACE, "")
	interpolate := p.alphaNum()
	if p.typ() == t_SPACE {
		p.i++
		styles := p.stylesOpt()
		p.check(p.db.UpdateLinkInterpolate(positions, interpolate))
		p.check(p.db.UpdateLink(positions, styles))
	} else {
		p.check(p.db.UpdateLinkInterpolate(positions, interpolate))
	}
}

func (p *parser) linkTail(id, url string) {
	if p.typ() != t_SPACE {
		p.db.SetLink(id, url, "")
		return
	}
	p.i++
	if p.typ() == t_LINK_TARGET {
		p.db.SetLink(id, url, p.next())
		return
	}
	tooltip := p.expect(t_STR, "'LINK_TARGET'")
	if p.typ() == t_SPACE {
		p.i++
		p.db.SetLink(id, url, p.expect(t_LINK_TARGET, ""))
	} else {
		p.db.SetLink(id, url, "")
	}
	p.db.SetTooltip(id, tooltip)
}

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
		p.db.SetClickEvent(id)
		if hasTooltip {
			p.db.SetTooltip(id, tooltip)
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
			p.db.SetClickEvent(id)
			p.db.SetTooltip(id, tooltip)
		} else {
			p.db.SetClickEvent(id)
		}
	}
}

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
func (p *parser) statement() (docItem, bool) {
	t := p.typ()
	switch t {
	case t_STYLE:
		p.i++
		p.expect(t_SPACE, "")
		id := p.idString()
		p.expect(t_SPACE, "")
		p.check(p.db.AddVertex(id, nil, "", p.stylesOpt(), nil, "", nil, "", false))
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
		p.db.AddClass(id, p.stylesOpt())
		p.separator()
		return docItem{}, false
	case t_CLASS:
		p.i++
		p.expect(t_SPACE, "")
		ids := p.idString()
		p.expect(t_SPACE, "")
		p.db.SetClass(ids, p.idString())
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
		p.db.SetAccTitle(value)
		return docItem{kind: itemString, text: value}, true
	case t_acc_descr:
		p.i++
		value := chars.Trim(p.expect(t_acc_descr_value, ""))
		p.db.SetAccDescription(value)
		return docItem{kind: itemString, text: value}, true
	case t_acc_descr_multiline_value:
		value := chars.Trim(p.next())
		p.db.SetAccDescription(value)
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

func (p *parser) parse() {
	for p.typ() == t_SPACE || p.typ() == t_NEWLINE {
		p.i++
	}
	p.expect(t_GRAPH, "'SPACE', 'NEWLINE'")
	if p.typ() == t_NODIR {
		p.i++
		p.db.SetDirection("TB")
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
		p.db.SetDirection(dir)
	}
	p.document()
	if p.typ() != t_END {
		p.fail(statementStart)
	}
}
