package class

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

func tokenSet(ts ...token) (set [t_PUNCTUATION + 1]bool) {
	for _, t := range ts {
		set[t] = true
	}
	return
}

var (
	alphaNumTokens  = tokenSet(t_UNICODE_TEXT, t_ALPHA, t_MINUS)
	stylePartTokens = tokenSet(t_ALPHA, t_COLON, t_BRKT, t_STYLE, t_PCT, t_LABEL)
)

// endType is Mermaid's relationType of a token that ends a relation, plus one so that zero
// means "not an end marker": Pele's END_TYPE.
var endType = func() (t [t_PUNCTUATION + 1]int) {
	t[t_AGGREGATION] = 1
	t[t_EXTENSION] = 2
	t[t_COMPOSITION] = 3
	t[t_DEPENDENCY] = 4
	t[t_LOLLIPOP] = 5
	return
}()

const nameTokens = "'UNICODE_TEXT', 'NUM', 'ALPHA', 'MINUS', 'BQUOTE_STR'"

// frame is an open namespace: its id, and the classes and notes declared in it. Pele's Frame.
type frame struct {
	id      string
	classes []string
	notes   []string
}

// parseError carries an error out of the parser's depths.
type parseError struct{ err error }

// parser is Pele's parseClassDiagram, its closures as methods.
type parser struct {
	src  string
	toks []tok
	i    int
	db   *DB
}

// ParseInto parses a class diagram's text into db, as Pele's parseClassDiagram does. An
// error is a *diagram.Error.
func ParseInto(src string, db *DB) (err error) {
	p := &parser{src: src, toks: tokenize(src), db: db}
	defer func() {
		if r := recover(); r != nil {
			e, ok := r.(parseError)
			if !ok {
				panic(r)
			}
			err = e.err
		}
	}()
	p.diagram()
	return nil
}

func (p *parser) typ() token { return p.toks[p.i].typ }

func (p *parser) typAt(i int) token { return p.toks[i].typ }

func (p *parser) text() string { return p.toks[p.i].text }

// next is the text of the token at i, and steps past it.
func (p *parser) next() string {
	s := p.toks[p.i].text
	p.i++
	return s
}

func (p *parser) fail(expected string) {
	t := p.toks[p.i]
	if t.typ == t_ERROR {
		panic(parseError{diagram.SyntaxError("class", p.src, t.start, "", true)})
	}
	panic(parseError{diagram.SyntaxError("class", p.src, t.start, "Expecting "+expected+", got '"+tokenNames[t.typ]+"'", false)})
}

func (p *parser) expect(typ token) string {
	if p.typ() != typ {
		p.fail("'" + tokenNames[typ] + "'")
	}
	return p.next()
}

func (p *parser) alphaNum() string {
	if !alphaNumTokens[p.typ()] {
		p.fail("'UNICODE_TEXT', 'NUM', 'ALPHA', 'MINUS'")
	}
	return p.next()
}

// name reads a class's name: words joined by dots, or a name in backticks, and a generic
// type after it if generic.
func (p *parser) name(generic bool) string {
	out := ""
	for {
		if p.typ() == t_BQUOTE_STR {
			out += p.next()
			break
		}
		if !alphaNumTokens[p.typ()] {
			p.fail(nameTokens)
		}
		out += p.next()
		t := p.typ()
		if t == t_DOT {
			out += "."
			p.i++
		} else if !alphaNumTokens[t] && t != t_BQUOTE_STR {
			break
		}
	}
	if generic && p.typ() == t_GENERICTYPE {
		out += "~" + p.next() + "~"
	}
	return out
}

func (p *parser) classLabel() string {
	p.i++
	label := p.expect(t_STR)
	p.expect(t_SQE)
	return label
}

// members reads the members of a class's body. The grammar collects them last to first and
// addMembers turns them back, so here they are in the order written.
func (p *parser) members() []string {
	if p.typ() != t_MEMBER {
		p.fail("'STRUCT_STOP', 'MEMBER'")
	}
	var list []string
	for p.typ() == t_MEMBER {
		list = append(list, p.next())
	}
	p.expect(t_STRUCT_STOP)
	return list
}

func (p *parser) classStatement() string {
	p.i++
	id := p.name(true)
	p.db.AddClass(id)
	if p.typ() == t_SQS {
		p.db.SetClassLabel(id, p.classLabel())
	}
	switch p.typ() {
	case t_STYLE_SEPARATOR:
		p.i++
		css := p.alphaNum()
		p.db.SetCSSClass(id, css)
		if p.typ() == t_STRUCT_START {
			p.i++
			p.db.AddMembers(id, p.members())
		}
	case t_STRUCT_START:
		p.i++
		if p.typ() == t_STRUCT_STOP {
			p.i++
		} else {
			p.db.AddMembers(id, p.members())
		}
	case t_ANNOTATION_START:
		p.i++
		annotation := p.alphaNum()
		p.expect(t_ANNOTATION_END)
		p.db.AddAnnotation(id, annotation)
		if p.typ() == t_STRUCT_START {
			p.i++
			if p.typ() == t_STRUCT_STOP {
				p.i++
			} else {
				p.db.AddMembers(id, p.members())
			}
		}
	}
	return id
}

func (p *parser) noteStatement() string {
	if p.typ() == t_NOTE {
		p.i++
		return p.db.AddNote(p.expect(t_STR), "")
	}
	p.i++
	id := p.name(true)
	return p.db.AddNote(p.expect(t_STR), id)
}

// namespaceStatement reads namespaces, which nest, with a stack of open ones.
func (p *parser) namespaceStatement() {
	var open []*frame
	for {
		p.i++
		ns := p.name(false)
		var id string
		if p.typ() == t_SQS {
			id = p.db.AddNamespace(ns, p.classLabel())
		} else {
			id = p.db.AddNamespace(ns, "")
		}
		p.expect(t_STRUCT_START)
		if p.typ() == t_NEWLINE {
			p.i++
		}
		open = append(open, &frame{id: id})
		item := true
		for item {
			f := open[len(open)-1]
			t := p.typ()
			if t == t_NAMESPACE {
				break
			}
			switch t {
			case t_CLASS:
				f.classes = append(f.classes, p.classStatement())
			case t_NOTE, t_NOTE_FOR:
				f.notes = append(f.notes, p.noteStatement())
			default:
				p.fail("'NAMESPACE', 'CLASS', 'NOTE_FOR', 'NOTE'")
			}
			for {
				if p.typ() == t_NEWLINE {
					next := p.typAt(p.i + 1)
					if next == t_NAMESPACE || next == t_CLASS || next == t_NOTE || next == t_NOTE_FOR {
						p.i++
						break
					}
					p.i++
				}
				p.expect(t_STRUCT_STOP)
				done := open[len(open)-1]
				open = open[:len(open)-1]
				p.db.AddClassesToNamespace(done.id, done.classes, done.notes)
				p.db.PopNamespace()
				if len(open) == 0 {
					item = false
					break
				}
			}
		}
		if len(open) == 0 {
			return
		}
	}
}

func (p *parser) relationEnd() End {
	t := endType[p.typ()]
	if t == 0 {
		return NoEnd
	}
	p.i++
	return End(t - 1)
}

func (p *parser) classOrRelation() {
	id1 := p.name(true)
	t := p.typ()
	if t == t_LABEL {
		p.db.AddMember(id1, CleanupLabel(p.next()))
		return
	}
	title1 := "none"
	if t == t_STR {
		title1 = p.next()
		t = p.typ()
	} else if endType[t] == 0 && t != t_LINE && t != t_DOTTED_LINE {
		return
	}
	type1 := p.relationEnd()
	t = p.typ()
	if t != t_LINE && t != t_DOTTED_LINE {
		p.fail("'AGGREGATION', 'EXTENSION', 'COMPOSITION', 'DEPENDENCY', 'LOLLIPOP', 'LINE', 'DOTTED_LINE'")
	}
	p.i++
	type2 := p.relationEnd()
	title2 := "none"
	if p.typ() == t_STR {
		title2 = p.next()
	}
	id2 := p.name(true)
	lineType := LineSolid
	if t == t_DOTTED_LINE {
		lineType = LineDotted
	}
	r := &Relation{ID1: id1, ID2: id2, Type1: type1, Type2: type2, LineType: lineType,
		RelationTitle1: title1, RelationTitle2: title2}
	if p.typ() == t_LABEL {
		r.Title = CleanupLabel(p.next())
		r.HasTitle = true
	}
	p.db.AddRelation(r)
}

func (p *parser) styles() []string {
	var out []string
	for {
		if !stylePartTokens[p.typ()] {
			p.fail("'ALPHA', 'NUM', 'COLON', 'UNIT', 'SPACE', 'BRKT', 'STYLE', 'PCT', 'LABEL'")
		}
		s := p.next()
		for stylePartTokens[p.typ()] {
			s += p.next()
		}
		out = append(out, s)
		if p.typ() != t_COMMA {
			return out
		}
		p.i++
	}
}

func (p *parser) tooltip(id string) {
	if p.typ() == t_STR {
		p.db.SetTooltip(id, p.next())
	}
}

func (p *parser) linkStatement(id string) {
	href := p.expect(t_STR)
	tip, hasTip := "", false
	if p.typ() == t_STR {
		tip, hasTip = p.next(), true
	}
	if p.typ() == t_LINK_TARGET {
		p.db.SetLink(id, href, p.next(), true)
	} else {
		p.db.SetLink(id, href, "", false)
	}
	if hasTip {
		p.db.SetTooltip(id, tip)
	}
}

func (p *parser) statement() {
	t := p.typ()
	switch t {
	case t_UNICODE_TEXT, t_ALPHA, t_MINUS, t_BQUOTE_STR:
		p.classOrRelation()
		return
	case t_NAMESPACE:
		p.namespaceStatement()
		return
	case t_CLASS:
		p.classStatement()
		return
	case t_MEMBER:
		p.i++
		return
	case t_ANNOTATION_START:
		p.i++
		annotation := p.alphaNum()
		p.expect(t_ANNOTATION_END)
		p.db.AddAnnotation(p.name(true), annotation)
		return
	case t_CALLBACK:
		p.i++
		id := p.name(true)
		p.db.SetClickEvent(id)
		p.expect(t_STR)
		p.tooltip(id)
		return
	case t_LINK:
		p.i++
		p.linkStatement(p.name(true))
		return
	case t_CLICK:
		p.i++
		id := p.name(true)
		if p.typ() == t_HREF {
			p.i++
			p.linkStatement(id)
			return
		}
		p.expect(t_CALLBACK_NAME)
		if p.typ() == t_CALLBACK_ARGS {
			p.i++
		}
		p.db.SetClickEvent(id)
		p.tooltip(id)
		return
	case t_STYLE:
		p.i++
		id := p.expect(t_ALPHA)
		p.db.SetCSSStyle(id, p.styles())
		return
	case t_CSSCLASS:
		p.i++
		ids := p.expect(t_STR)
		p.db.SetCSSClass(ids, p.expect(t_ALPHA))
		return
	case t_NOTE, t_NOTE_FOR:
		p.noteStatement()
		return
	case t_CLASSDEF:
		p.i++
		ids := []string{p.expect(t_ALPHA)}
		for p.typ() == t_COMMA {
			p.i++
			ids = append(ids, p.expect(t_ALPHA))
		}
		p.db.DefineClass(ids, p.styles())
		return
	case t_direction_tb, t_direction_bt, t_direction_rl, t_direction_lr:
		p.i++
		p.db.Direction = directionNames[t-t_direction_tb]
		return
	case t_acc_title:
		p.i++
		p.db.SetAccTitle(chars.Trim(p.expect(t_acc_title_value)))
		return
	case t_acc_descr:
		p.i++
		p.db.SetAccDescription(chars.Trim(p.expect(t_acc_descr_value)))
		return
	case t_acc_descr_multiline_value:
		p.db.SetAccDescription(chars.Trim(p.next()))
		return
	}
	p.fail("'acc_title', 'acc_descr', 'acc_descr_multiline_value', 'NAMESPACE', 'ANNOTATION_START', 'CLASS', " +
		"'MEMBER', 'NOTE_FOR', 'NOTE', 'CLASSDEF', 'direction_tb', 'CALLBACK', 'LINK', 'CLICK', 'STYLE', " +
		"'CSSCLASS', 'ALPHA', 'MINUS', 'BQUOTE_STR'")
}

func (p *parser) diagram() {
	// Mermaid's grammar also accepts statements with no `classDiagram` line before them.
	header := p.typ() == t_CLASS_DIAGRAM
	if header {
		p.i++
		p.expect(t_NEWLINE)
	}
	for {
		p.statement()
		if p.typ() != t_NEWLINE {
			break
		}
		p.i++
		if p.typ() == t_EOF || p.typ() == t_END {
			break
		}
	}
	if header {
		p.expect(t_EOF)
	}
	p.expect(t_END)
}
