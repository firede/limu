package er

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// cardinality is the cardinality of a token, or "" for a token that is none: Pele's
// CARDINALITY, a map there.
func cardinality(t token) string {
	switch t {
	case t_ZERO_OR_ONE:
		return ZeroOrOne
	case t_ZERO_OR_MORE:
		return ZeroOrMore
	case t_ONE_OR_MORE:
		return OneOrMore
	case t_ONLY_ONE:
		return OnlyOne
	case t_MD_PARENT:
		return MDParent
	}
	return ""
}

// directions are the directions of the direction tokens, in their order, as Pele's DIRECTIONS.
var directions = [4]string{"TB", "BT", "RL", "LR"}

// The tokens a name, a cardinality and a statement may start with, as an error lists them:
// Pele's NAME, CARDS and LINE.
const (
	nameTokens = "'UNICODE_TEXT', 'NUM', 'ENTITY_NAME', 'DECIMAL_NUM', 'ENTITY_ONE'"
	cardTokens = "'ZERO_OR_ONE', 'ZERO_OR_MORE', 'ONE_OR_MORE', 'ONLY_ONE', 'MD_PARENT'"
	lineTokens = "'EOF', 'NEWLINE', 'title', 'acc_title', 'acc_descr', 'acc_descr_multiline_value', 'SUBGRAPH', 'direction_tb', " +
		"'direction_bt', 'direction_rl', 'direction_lr', 'CLASSDEF', 'CLASS', 'STYLE', " + nameTokens
)

// frame is an open subgraph: Pele's Frame. list is what Mermaid's grammar hands to
// addSubGraph: the value of every statement inside, which besides entity names holds
// accessibility text and the keywords style, class and classDef.
type frame struct {
	id, title string
	list      []docItem
}

// parseError carries an error out of the parser's depths.
type parseError struct{ err error }

// parser is the state of Pele's parseEr: its tokens, the index i of the next, and the
// database it fills.
type parser struct {
	src  string
	toks []tok
	i    int
	db   *DB
}

// ParseInto parses an ER diagram's text into db, as Pele's parseEr does. An error is a
// *diagram.Error.
func ParseInto(src string, db *DB) (err error) {
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

// typ is the type of the next token, Pele's types[i].
func (p *parser) typ() token { return p.toks[p.i].typ }

// next is the text of the next token, which it steps past: Pele's texts[i++].
func (p *parser) next() string {
	s := p.toks[p.i].text
	p.i++
	return s
}

// fail stops the parse with a syntax error at the next token, saying what was expected, as
// the fail of Pele's parseEr does.
func (p *parser) fail(expected string) {
	t := p.toks[p.i]
	if t.typ == t_ERROR {
		panic(parseError{diagram.SyntaxError("er", p.src, t.start, "", true)})
	}
	got := tokenNames[t.typ]
	if t.typ == t_CHAR {
		got = t.text
	}
	panic(parseError{diagram.SyntaxError("er", p.src, t.start, "Expecting "+expected+", got '"+got+"'", false)})
}

// expect steps past a token of type typ and gives its text, or fails, as the expect of Pele's
// parseEr does.
func (p *parser) expect(typ token) string {
	if p.typ() != typ {
		p.fail("'" + tokenNames[typ] + "'")
	}
	return p.next()
}

// separator steps past the newline, semicolon or end of input that ends a statement, or
// fails, as the separator of Pele's parseEr does.
func (p *parser) separator() {
	if t := p.typ(); t != t_NEWLINE && t != t_SEMI && t != t_EOF {
		p.fail("'EOF', 'NEWLINE', 'SEMI'")
	}
	p.i++
}

// isName reports whether a token of type t may be an entity's name, as the isName of Pele's
// parseEr does.
func isName(t token) bool {
	return t == t_UNICODE_TEXT || t == t_ENTITY_NAME || t == t_NUM || t == t_DECIMAL_NUM || t == t_ENTITY_ONE
}

// unquote is a quoted token's text without its quotes, Pele's text.slice(1, -1).
func unquote(s string) string { return s[1 : len(s)-1] }

// entityName reads an entity's name, without the quotes of a quoted one, or fails, as the
// entityName of Pele's parseEr does.
func (p *parser) entityName() string {
	t := p.typ()
	if !isName(t) {
		p.fail(nameTokens)
	}
	text := p.next()
	if t == t_ENTITY_NAME {
		return unquote(text)
	}
	return text
}

// idList reads a list of ids that commas separate, as the idList of Pele's parseEr does.
func (p *parser) idList() []string {
	var ids []string
	for {
		if t := p.typ(); t != t_UNICODE_TEXT && t != t_STYLE_TEXT {
			p.fail("'UNICODE_TEXT', 'STYLE_TEXT'")
		}
		ids = append(ids, p.next())
		if p.typ() != t_COMMA {
			return ids
		}
		p.i++
	}
}

// isStyleComponent reports whether a token of type t may be part of a style, as the
// isStyleComponent of Pele's parseEr does.
func isStyleComponent(t token) bool {
	return t == t_STYLE_TEXT || t == t_NUM || t == t_COLON || t == t_BRKT
}

// stylesOpt reads a list of styles that commas separate, each the text of the run of style
// components it is made of, as the stylesOpt of Pele's parseEr does.
func (p *parser) stylesOpt() []string {
	var styles []string
	for {
		if !isStyleComponent(p.typ()) {
			p.fail("'STYLE_TEXT', 'COLON', 'NUM', 'BRKT'")
		}
		style := p.next()
		for isStyleComponent(p.typ()) {
			style += p.next()
		}
		styles = append(styles, style)
		if p.typ() != t_COMMA {
			return styles
		}
		p.i++
	}
}

// relSpec reads a relationship's two cardinalities and its line, the next token being a
// cardinality, as the relSpec of Pele's parseEr does.
func (p *parser) relSpec() RelSpec {
	cardB := cardinality(p.typ())
	p.i++
	r := p.typ()
	if r != t_IDENTIFYING && r != t_NON_IDENTIFYING {
		p.fail("'NON_IDENTIFYING', 'IDENTIFYING'")
	}
	relType := NonIdentifying
	if r == t_IDENTIFYING {
		relType = Identifying
	}
	p.i++
	cardA := cardinality(p.typ())
	if cardA == "" {
		p.fail(cardTokens)
	}
	p.i++
	return RelSpec{CardA: cardA, RelType: relType, CardB: cardB}
}

// role reads a relationship's label, without the quotes of a quoted one, as the role of
// Pele's parseEr does.
func (p *parser) role() string {
	t := p.typ()
	if t != t_WORD && t != t_ENTITY_NAME && t != t_UNICODE_TEXT {
		p.fail("'UNICODE_TEXT', 'ENTITY_NAME', 'WORD'")
	}
	text := p.next()
	if t == t_UNICODE_TEXT {
		return text
	}
	return unquote(text)
}

// isChar reports whether the next token is the CHAR ch, as the isChar of Pele's parseEr does.
func (p *parser) isChar(ch string) bool {
	return p.typ() == t_CHAR && p.toks[p.i].text == ch
}

// attributes parses the attributes of a block and gives them last to first, as Mermaid's
// right-recursive rule builds them, as the attributes of Pele's parseEr does.
func (p *parser) attributes() []Attribute {
	var list []Attribute
	for {
		typ := p.expect(t_ATTRIBUTE_WORD)
		if p.isChar("?") {
			typ += p.next()
		}
		if p.typ() != t_ATTRIBUTE_WORD {
			if strings.HasSuffix(typ, "?") {
				p.fail("'ATTRIBUTE_WORD'")
			}
			p.fail("'ATTRIBUTE_WORD', '?'")
		}
		a := Attribute{Type: typ, Name: p.next()}
		if p.typ() == t_ATTRIBUTE_KEY {
			keys := []string{p.next()}
			for p.isChar(",") {
				p.i++
				keys = append(keys, p.expect(t_ATTRIBUTE_KEY))
			}
			a.Keys = keys
		}
		if p.typ() == t_COMMENT {
			a.Comment = unquote(p.next())
		}
		list = append(list, a)
		if p.typ() == t_BLOCK_STOP {
			break
		}
	}
	for i, j := 0, len(list)-1; i < j; i, j = i+1, j-1 {
		list[i], list[j] = list[j], list[i]
	}
	return list
}

// block parses an optional `{ ... }` block; nil when there is none or it is empty. It is the
// block of Pele's parseEr.
func (p *parser) block() []Attribute {
	if p.typ() != t_BLOCK_START {
		return nil
	}
	p.i++
	if p.typ() == t_BLOCK_STOP {
		p.i++
		return nil
	}
	list := p.attributes()
	p.i++
	return list
}

// entityStatement parses a statement that starts with an entity's name: an entity with an
// alias, a relationship, or an entity alone, each with its classes and attributes, and gives
// the names of the entities it names, as the entityStatement of Pele's parseEr does.
func (p *parser) entityStatement() []string {
	db := p.db
	a := p.entityName()
	t := p.typ()
	if t == t_SQS {
		p.i++
		alias := p.entityName()
		p.expect(t_SQE)
		var classes []string
		if p.typ() == t_STYLE_SEPARATOR {
			p.i++
			classes = p.idList()
		}
		attrs := p.block()
		db.AddEntity(a, alias)
		if attrs != nil {
			db.AddAttributes(a, attrs)
		}
		if classes != nil {
			db.SetClass([]string{a}, classes)
		}
		return []string{a}
	}
	var classes []string
	if t == t_STYLE_SEPARATOR {
		p.i++
		classes = p.idList()
		t = p.typ()
	}
	if cardinality(t) != "" {
		spec := p.relSpec()
		b := p.entityName()
		var classesB []string
		if p.typ() == t_STYLE_SEPARATOR {
			p.i++
			classesB = p.idList()
		}
		p.expect(t_COLON)
		label := p.role()
		db.AddEntity(a, "")
		db.AddEntity(b, "")
		db.AddRelationship(a, label, b, spec)
		if classes != nil {
			db.SetClass([]string{a}, classes)
		}
		if classesB != nil {
			db.SetClass([]string{b}, classesB)
		}
		return []string{a, b}
	}
	attrs := p.block()
	db.AddEntity(a, "")
	if attrs != nil {
		db.AddAttributes(a, attrs)
	}
	if classes != nil {
		db.SetClass([]string{a}, classes)
	}
	return []string{a}
}

// parse parses the diagram's statements into the database, as the body of Pele's parseEr
// does.
func (p *parser) parse() {
	db := p.db
	p.expect(t_ER_DIAGRAM)
	// Subgraphs nest on this stack rather than on the call stack; the first frame is the
	// diagram itself.
	frames := []*frame{{}}
	f := frames[0]
	nested := false
	// push adds a statement's value to the open subgraph's list, as Pele's
	// `if (nested) frame.list.push(...)` does after each statement.
	push := func(name string) {
		if nested {
			f.list = append(f.list, docItem{name: name})
		}
	}
	for {
		t := p.typ()
		switch t {
		case t_NEWLINE:
			p.i++
		case t_EOF:
			p.i++
			if nested {
				p.fail("'END', " + lineTokens)
			}
			return
		case t_END_KW:
			if !nested {
				p.fail(lineTokens)
			}
			p.i++
			id := db.addSubGraph(f.id, f.list, f.title)
			frames = frames[:len(frames)-1]
			f = frames[len(frames)-1]
			nested = len(frames) > 1
			push(id)
		case t_SUBGRAPH:
			p.i++
			id := p.entityName()
			title := id
			if p.typ() == t_SQS {
				p.i++
				title = p.entityName()
				for isName(p.typ()) {
					title += " " + p.entityName()
				}
				p.expect(t_SQE)
			}
			p.separator()
			f = &frame{id: id, title: title}
			frames = append(frames, f)
			nested = true
		case t_acc_title:
			p.i++
			value := chars.Trim(p.expect(t_acc_title_value))
			db.SetAccTitle(value)
			push(value)
		case t_acc_descr:
			p.i++
			value := chars.Trim(p.expect(t_acc_descr_value))
			db.SetAccDescription(value)
			push(value)
		case t_acc_descr_multiline_value:
			value := chars.Trim(p.next())
			db.SetAccDescription(value)
			push(value)
		case t_direction_tb, t_direction_bt, t_direction_rl, t_direction_lr:
			p.i++
			value := directions[t-t_direction_tb]
			if nested {
				f.list = append(f.list, docItem{dir: value})
			} else {
				db.SetDirection(value)
			}
		case t_CLASSDEF:
			keyword := p.next()
			ids := p.idList()
			styles := p.stylesOpt()
			p.separator()
			db.AddClass(ids, styles)
			push(keyword)
		case t_CLASS:
			keyword := p.next()
			ids := p.idList()
			db.SetClass(ids, p.idList())
			push(keyword)
		case t_STYLE:
			keyword := p.next()
			ids := p.idList()
			styles := p.stylesOpt()
			p.separator()
			db.AddCSSStyles(ids, styles)
			push(keyword)
		default:
			if !isName(t) {
				if nested {
					p.fail("'END', " + lineTokens)
				}
				p.fail(lineTokens)
			}
			for _, name := range p.entityStatement() {
				push(name)
			}
		}
	}
}
