package state

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

var directions = [...]string{"TB", "BT", "RL", "LR"}

const statementTokens = "'NL', 'ID', 'HIDE_EMPTY', 'scale', 'COMPOSIT_STATE', 'STATE_DESCR', 'FORK', 'JOIN', 'CHOICE', " +
	"'CONCURRENT', 'note', 'acc_title', 'acc_descr', 'acc_descr_multiline_value', 'CLICK', 'classDef', 'style', " +
	"'class', 'direction_tb', 'direction_bt', 'direction_rl', 'direction_lr', 'EDGE_STATE'"

// parseError carries an error out of the parser's depths.
type parseError struct{ err error }

type parser struct {
	src string
	lex tokens
	i   int
	db  *DB
}

// parseInto parses a state diagram's text into db, as Pele's parseState does: it reads the
// statements in a loop, with a stack of the composites open, and accepts what Mermaid's
// stateDiagram.jison accepts. It then sets the root document, which resolves the DB's states
// and its graph. An error is a *diagram.Error.
func parseInto(source string, db *DB) (err error) {
	p := &parser{src: source, lex: tokenize(source), db: db}
	defer func() {
		if r := recover(); r != nil {
			pe, ok := r.(parseError)
			if !ok {
				panic(r)
			}
			err = pe.err
		}
	}()
	doc := p.parse()
	return db.setRootDoc(doc)
}

func (p *parser) typ() token { return p.lex.toks[p.i].typ }

func (p *parser) next() string {
	s := p.lex.toks[p.i].text
	p.i++
	return s
}

func (p *parser) fail(expected string) {
	t := p.lex.toks[p.i]
	if t.typ == t_ERROR {
		if p.lex.hasError {
			loc := common.Locate(p.src, t.start)
			panic(parseError{&diagram.Error{Code: diagram.Syntax, Message: p.lex.error, Type: "state",
				Line: loc.Line, Column: loc.Column, Snippet: loc.Snippet}})
		}
		panic(parseError{common.SyntaxError("state", p.src, t.start, "", true)})
	}
	panic(parseError{common.SyntaxError("state", p.src, t.start, "Expecting "+expected+", got '"+tokenNames[t.typ]+"'", false)})
}

func (p *parser) expect(typ token) string {
	if p.typ() != typ {
		p.fail("'" + tokenNames[typ] + "'")
	}
	return p.next()
}

// idStatement is a state named by an id or `[*]`, with a class after `:::`.
func (p *parser) idStatement() *StateStmt {
	if t := p.typ(); t != t_ID && t != t_EDGE_STATE {
		p.fail("'ID', 'EDGE_STATE'")
	}
	id := chars.Trim(p.next())
	if p.typ() != t_STYLE_SEPARATOR {
		return &StateStmt{ID: id, Type: "default", Description: []string{""}}
	}
	p.i++
	return &StateStmt{ID: id, Classes: []string{chars.Trim(p.expect(t_ID))}, Type: "default", Description: []string{""}}
}

func (p *parser) parse() []Stmt {
	for p.typ() == t_NL {
		p.i++
	}
	if p.typ() != t_SD {
		p.fail("'NL', 'SD'")
	}
	p.i++

	doc := []Stmt{}
	var outer [][]Stmt
	// The documents of the open composites, which are set on them when they close.
	var composites []*StateStmt
	// add drops a statement whose value is the string the grammar uses for a blank line.
	add := func(value Stmt) {
		if value != TextStmt("nl") {
			doc = append(doc, value)
		}
	}
	open := func(id, description string) {
		p.i++
		state := &StateStmt{ID: id, Type: "default", Description: []string{description}, Doc: []Stmt{}}
		doc = append(doc, state)
		outer = append(outer, doc)
		composites = append(composites, state)
		doc = []Stmt{}
	}

	for {
		t := p.typ()
		switch t {
		case t_NL:
			p.i++
		case t_ID, t_EDGE_STATE:
			first := p.idStatement()
			switch p.typ() {
			case t_DESCR:
				first.Description = []string{trimColon(p.next())}
				add(first)
			case t_ARROW:
				p.i++
				second := p.idStatement()
				if p.typ() == t_DESCR {
					add(&RelationStmt{State1: first, State2: second, Description: trimColon(p.next()), HasDescription: true})
				} else {
					add(&RelationStmt{State1: first, State2: second})
				}
			default:
				add(first)
			}
		case t_HIDE_EMPTY:
			add(TextStmt(p.next()))
		case t_scale:
			p.i++
			p.expect(t_WIDTH)
			add(TextStmt(p.lex.toks[p.i-2].text))
		case t_COMPOSIT_STATE:
			if p.lex.toks[p.i+1].typ == t_STRUCT_START {
				open(p.next(), "")
			} else {
				add(TextStmt(p.next()))
			}
		case t_STATE_DESCR:
			description := p.next()
			p.expect(t_AS)
			id := p.expect(t_ID)
			switch {
			case p.typ() == t_STRUCT_START:
				open(id, description)
			case strings.Contains(id, ":"):
				parts := strings.Split(id, ":")
				add(&StateStmt{ID: parts[0], Type: "default", Description: []string{chars.Trim(description), parts[1]},
					DescriptionList: true})
			default:
				add(&StateStmt{ID: id, Type: "default", Description: []string{chars.Trim(description)}})
			}
		case t_FORK:
			add(&StateStmt{ID: p.next(), Type: "fork"})
		case t_JOIN:
			add(&StateStmt{ID: p.next(), Type: "join"})
		case t_CHOICE:
			add(&StateStmt{ID: p.next(), Type: "choice"})
		case t_CONCURRENT:
			p.i++
			add(&StateStmt{ID: p.db.dividerID(), Type: "divider"})
		case t_note:
			keyword := p.next()
			if p.typ() == t_left_of || p.typ() == t_right_of {
				position := chars.Trim(p.next())
				id := chars.Trim(p.expect(t_ID))
				add(&StateStmt{ID: id, Note: &Note{Position: position, Text: chars.Trim(p.expect(t_NOTE_TEXT))}})
			} else {
				if p.typ() != t_NOTE_TEXT {
					p.fail("'NOTE_TEXT', 'left_of', 'right_of'")
				}
				p.i++
				p.expect(t_AS)
				p.expect(t_ID)
				add(TextStmt(keyword))
			}
		case t_direction_tb, t_direction_bt, t_direction_rl, t_direction_lr:
			value := directions[t-t_direction_tb]
			p.i++
			p.db.setDirection(value)
			add(&DirectionStmt{value})
		case t_acc_title:
			p.i++
			value := chars.Trim(p.expect(t_acc_title_value))
			p.db.setAccTitle(value)
			add(TextStmt(value))
		case t_acc_descr:
			p.i++
			value := chars.Trim(p.expect(t_acc_descr_value))
			p.db.setAccDescription(value)
			add(TextStmt(value))
		case t_acc_descr_multiline_value:
			value := chars.Trim(p.next())
			p.db.setAccDescription(value)
			add(TextStmt(value))
		case t_CLICK:
			p.i++
			id := p.idStatement()
			var url, tooltip string
			if p.typ() == t_HREF {
				p.i++
				url = p.expect(t_STRING)
			} else {
				if p.typ() != t_STRING {
					p.fail("'STRING', 'HREF'")
				}
				url = p.next()
				tooltip = p.expect(t_STRING)
			}
			p.expect(t_NL)
			add(&ClickStmt{ID: id, URL: url, Tooltip: tooltip})
		case t_classDef:
			p.i++
			id := chars.Trim(p.expect(t_CLASSDEF_ID))
			add(&ClassDefStmt{ID: id, Classes: chars.Trim(p.expect(t_CLASSDEF_STYLEOPTS))})
		case t_style:
			p.i++
			id := chars.Trim(p.expect(t_STYLE_IDS))
			add(&StyleStmt{ID: id, StyleClass: chars.Trim(p.expect(t_STYLEDEF_STYLEOPTS))})
		case t_class:
			p.i++
			id := chars.Trim(p.expect(t_CLASSENTITY_IDS))
			add(&ApplyClassStmt{ID: id, StyleClass: chars.Trim(p.expect(t_STYLECLASS))})
		case t_STRUCT_STOP:
			if len(outer) == 0 {
				p.fail("'$end', " + statementTokens)
			}
			p.i++
			composites[len(composites)-1].Doc = doc
			composites = composites[:len(composites)-1]
			doc = outer[len(outer)-1]
			outer = outer[:len(outer)-1]
		case t_END:
			if len(outer) > 0 {
				p.fail("'STRUCT_STOP', " + statementTokens)
			}
			return doc
		default:
			if len(outer) > 0 {
				p.fail("'STRUCT_STOP', " + statementTokens)
			}
			p.fail("'$end', " + statementTokens)
		}
	}
}
