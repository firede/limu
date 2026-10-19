package requirement

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// values are what the keyword tokens stand for, from REQUIREMENT to TRACES in token order,
// as Pele's VALUES.
var values = [...]string{
	KindRequirement,
	KindFunctionalRequirement,
	KindInterfaceRequirement,
	KindPerformanceRequirement,
	KindPhysicalRequirement,
	KindDesignConstraint,
	RiskLow,
	RiskMedium,
	RiskHigh,
	VerifyAnalysis,
	VerifyDemonstration,
	VerifyInspection,
	VerifyTest,
	"",
	Contains,
	Copies,
	Derives,
	Satisfies,
	Verifies,
	Refines,
	Traces,
}

func isStylePart(t token) bool {
	return t == t_ALPHA || t == t_COLON || t == t_BRKT || t == t_MINUS || t == t_SEMICOLON
}

// parseError carries an error out of the parser's depths.
type parseError struct{ err error }

type parser struct {
	src    string
	toks   []tok
	i      int
	db     *DB
	fields []token
	values []string
}

// ParseInto parses a requirement diagram's text into db, as Pele's parseRequirement does: a
// recursive-descent parser that agrees with Mermaid's requirementDiagram.jison. An error is
// a *diagram.Error.
func ParseInto(source string, db *DB) (err error) {
	p := &parser{src: source, toks: tokenize(source), db: db}
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

// typ is the type of the token at i. The stream ends with END or ERROR, which the parser
// never steps past.
func (p *parser) typ() token { return p.toks[p.i].typ }

// fail fails with Jison's wording, naming the token types from to to and any others as
// expected.
func (p *parser) fail(from, to token, others ...token) {
	t := p.toks[p.i]
	if t.typ == t_ERROR {
		panic(parseError{diagram.SyntaxError("requirement", p.src, t.start, "", true)})
	}
	var b strings.Builder
	b.WriteString("Expecting ")
	first := true
	name := func(k token) {
		if !first {
			b.WriteString(", ")
		}
		first = false
		b.WriteString("'" + tokenNames[k] + "'")
	}
	for k := from; k <= to; k++ {
		name(k)
	}
	for _, k := range others {
		name(k)
	}
	b.WriteString(", got '" + tokenNames[t.typ] + "'")
	panic(parseError{diagram.SyntaxError("requirement", p.src, t.start, b.String(), false)})
}

func (p *parser) expect(typ token) string {
	if p.typ() != typ {
		p.fail(typ, typ)
	}
	s := p.toks[p.i].text
	p.i++
	return s
}

// keyword is the value of the next token if it is a keyword in the given range of token
// types.
func (p *parser) keyword(from, to token) string {
	t := p.typ()
	if t < from || t > to {
		p.fail(from, to)
	}
	p.i++
	return values[t-t_REQUIREMENT]
}

func (p *parser) string() string {
	t := p.typ()
	if t != t_unqString && t != t_qString {
		p.fail(t_qString, t_unqString)
	}
	s := p.toks[p.i].text
	p.i++
	return s
}

func (p *parser) directive() bool {
	switch p.typ() {
	case t_acc_title:
		p.i++
		p.db.setAccTitle(chars.Trim(p.expect(t_acc_title_value)))
	case t_acc_descr:
		p.i++
		p.db.setAccDescription(chars.Trim(p.expect(t_acc_descr_value)))
	case t_acc_descr_multiline_value:
		p.db.setAccDescription(chars.Trim(p.toks[p.i].text))
		p.i++
	default:
		return false
	}
	return true
}

func (p *parser) idList() []string {
	var list []string
	for {
		t := p.typ()
		if t != t_ALPHA && t != t_unqString && t != t_qString {
			p.fail(t_qString, t_unqString, t_ALPHA)
		}
		list = append(list, p.toks[p.i].text)
		p.i++
		if p.typ() != t_COMMA {
			return list
		}
		p.i++
	}
}

func (p *parser) stylesOpt() []string {
	var styles []string
	for {
		if !isStylePart(p.typ()) {
			p.fail(t_ALPHA, t_SEMICOLON, t_MINUS, t_BRKT)
		}
		style := p.toks[p.i].text
		p.i++
		for isStylePart(p.typ()) {
			style += p.toks[p.i].text
			p.i++
		}
		styles = append(styles, style)
		if p.typ() != t_COMMA {
			return styles
		}
		p.i++
	}
}

// classList reads what follows a requirement's or an element's name up to its body: the
// classes after :::, if any, and the opening brace.
func (p *parser) classList() []string {
	var classes []string
	switch p.typ() {
	case t_STYLE_SEPARATOR:
		p.i++
		classes = p.idList()
		p.expect(t_STRUCT_START)
	case t_STRUCT_START:
		p.i++
	default:
		p.fail(t_STRUCT_START, t_STRUCT_START, t_STYLE_SEPARATOR)
	}
	p.expect(t_NEWLINE)
	return classes
}

// body reads a body up to its closing brace. The grammar is right recursive, so Mermaid
// applies the fields last to first and the first of two repeated fields wins.
func (p *parser) body(requirement bool) {
	p.fields = p.fields[:0]
	p.values = p.values[:0]
	for {
		t := p.typ()
		if t == t_STRUCT_STOP {
			p.i++
			break
		}
		if t == t_NEWLINE {
			p.i++
			continue
		}
		if requirement && (t < t_ID || t > t_VERIFYMTHD) || !requirement && t != t_TYPE && t != t_DOCREF {
			if requirement {
				p.fail(t_ID, t_VERIFYMTHD, t_NEWLINE, t_STRUCT_STOP)
			}
			p.fail(t_TYPE, t_DOCREF, t_NEWLINE, t_STRUCT_STOP)
		}
		p.i++
		p.expect(t_COLONSEP)
		var v string
		switch t {
		case t_RISK:
			v = p.keyword(t_LOW_RISK, t_HIGH_RISK)
		case t_VERIFYMTHD:
			v = p.keyword(t_VERIFY_ANALYSIS, t_VERIFY_TEST)
		default:
			v = p.string()
		}
		p.values = append(p.values, v)
		p.fields = append(p.fields, t)
		p.expect(t_NEWLINE)
	}
	for k := len(p.fields) - 1; k >= 0; k-- {
		v := p.values[k]
		switch p.fields[k] {
		case t_ID:
			p.db.newID = v
		case t_TEXT:
			p.db.newText = v
		case t_RISK:
			p.db.newRisk = v
		case t_VERIFYMTHD:
			p.db.newVerifyMethod = v
		case t_TYPE:
			p.db.newElementType = v
		default:
			p.db.newDocRef = v
		}
	}
}

func (p *parser) parse() {
	db := p.db
	for p.directive() {
		if p.typ() == t_NEWLINE {
			p.i++
		}
	}
	if p.typ() != t_RD {
		p.fail(t_RD, t_RD, t_acc_title, t_acc_descr, t_acc_descr_multiline_value)
	}
	p.i++
	p.expect(t_NEWLINE)

	for {
		t := p.typ()
		switch {
		case t == t_EOF:
			return
		case t == t_NEWLINE:
			p.i++
		case p.directive():
		case t >= t_direction_tb && t <= t_direction_lr:
			p.i++
			db.setDirection([...]string{"TB", "BT", "RL", "LR"}[t-t_direction_tb])
		case t >= t_REQUIREMENT && t <= t_DESIGN_CONSTRAINT:
			typ := p.keyword(t_REQUIREMENT, t_DESIGN_CONSTRAINT)
			name := p.string()
			classes := p.classList()
			p.body(true)
			db.addRequirement(name, typ)
			if classes != nil {
				db.setClass([]string{name}, classes)
			}
		case t == t_ELEMENT:
			p.i++
			name := p.string()
			classes := p.classList()
			p.body(false)
			db.addElement(name)
			if classes != nil {
				db.setClass([]string{name}, classes)
			}
		case t == t_STYLE:
			p.i++
			ids := p.idList()
			db.setCSSStyle(ids, p.stylesOpt())
		case t == t_CLASSDEF:
			p.i++
			ids := p.idList()
			db.defineClass(ids, p.stylesOpt())
		case t == t_CLASS:
			p.i++
			ids := p.idList()
			db.setClass(ids, p.idList())
		case t == t_unqString || t == t_qString:
			id := p.toks[p.i].text
			p.i++
			switch next := p.typ(); next {
			case t_STYLE_SEPARATOR:
				p.i++
				db.setClass([]string{id}, p.idList())
			case t_END_ARROW_L, t_LINE:
				p.i++
				relationship := p.keyword(t_CONTAINS, t_TRACES)
				if next == t_LINE {
					p.expect(t_END_ARROW_R)
				} else {
					p.expect(t_LINE)
				}
				other := p.string()
				if next == t_LINE {
					db.addRelationship(relationship, id, other)
				} else {
					db.addRelationship(relationship, other, id)
				}
			default:
				p.fail(t_STYLE_SEPARATOR, t_STYLE_SEPARATOR, t_END_ARROW_L, t_LINE)
			}
		default:
			p.fail(t_NEWLINE, t_EOF, t_acc_title, t_acc_descr, t_acc_descr_multiline_value,
				t_direction_tb, t_direction_bt, t_direction_rl, t_direction_lr,
				t_REQUIREMENT, t_FUNCTIONAL_REQUIREMENT, t_INTERFACE_REQUIREMENT, t_PERFORMANCE_REQUIREMENT,
				t_PHYSICAL_REQUIREMENT, t_DESIGN_CONSTRAINT, t_ELEMENT, t_CLASSDEF, t_CLASS, t_STYLE,
				t_unqString, t_qString)
		}
	}
}
