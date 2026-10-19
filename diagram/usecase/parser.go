package usecase

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// Mermaid parses use case diagrams with a Chevrotain parser, which decides each choice by
// looking at most three tokens ahead and reports the first token that fits nowhere. This
// parser, Pele's, makes the same decisions from the same lookahead, so it accepts the same
// text and names the same token in its errors. It builds drafts of the model directly, where
// Mermaid builds a syntax tree and then visits it.

const (
	fLine = 1 << iota
	fRelation
	fStylePart
	fFollowsName
	fBoundaryLine
	fClosesLabel
	fMetadataFirst
)

const (
	arrowNames     = "DEPENDENCY_ARROW|GENERALIZATION|FORWARD_SOLID|BACKWARD_SOLID|MARKERLESS_SOLID|FORWARD_CIRCLE|BACKWARD_CIRCLE|FORWARD_CROSS|BACKWARD_CROSS"
	nameNames      = "IDENTIFIER|PLAIN_STRING|MARKDOWN_STRING"
	labelNames     = "PLAIN_STRING|MARKDOWN_STRING|LABEL_TEXT"
	statementNames = "ACC_TITLE_LINE|ACC_DESCR_LINE|ACC_DESCR_BLOCK|DIRECTION|ACTOR|SYSTEM_BOUNDARY|NOTE|JSON_DECLARATION_START|CLASS_DEF|CLASS|STYLE"
	stylePartNames = "WORD|PLAIN_STRING|NUMBER|HASH_COLOR|CSS_IDENTIFIER|CSS_ESCAPED_COMMA|DASH|DOT|PERCENT|CSS_PUNCTUATION|COLON|LPAREN|RPAREN|LBRACKET|RBRACKET|LBRACE|RBRACE|AT|MARKERLESS_SOLID"
)

// flags are what each type of token may do, as Pele's FLAGS.
var flags = func() (f [tRPAREN + 1]uint8) {
	flag := func(names string, bits uint8) {
		for _, name := range strings.Split(names, "|") {
			// WORD is a category: every keyword and IDENTIFIER.
			if name == "WORD" {
				for t := tUSECASE; t <= tIDENTIFIER; t++ {
					f[t] |= bits
				}
				continue
			}
			for t, n := range tokenNames {
				if n == name {
					f[t] |= bits
				}
			}
		}
	}
	flag("NEWLINE|COMMENT|"+statementNames+"|"+nameNames, fLine)
	flag("IDENTIFIER|"+arrowNames, fRelation)
	flag(stylePartNames, fStylePart)
	flag("STEREOTYPE_START|CLASS_SEPARATOR|IDENTIFIER|"+arrowNames+"|NEWLINE|EOF", fFollowsName)
	flag("NEWLINE|COMMENT|ACTOR|"+nameNames, fBoundaryLine)
	flag("FORWARD_SOLID|MARKERLESS_SOLID|FORWARD_CIRCLE|FORWARD_CROSS", fClosesLabel)
	flag("NEWLINE|IDENTIFIER|PLAIN_STRING|RBRACE", fMetadataFirst)
	return
}()

// forbidden are the PlantUML statements people are most likely to try, which Mermaid turns
// away.
var forbidden = map[string]bool{"allowmixing": true, "newpage": true, "package": true, "rectangle": true, "skinparam": true}

// item is an element as one statement mentions it, Pele's Item.
type item struct {
	id             string
	label          draftLabel
	span           Span
	generated      bool
	shape          string
	metadata       []draftProperty
	stereotype     string
	stereotypeSpan *Span
	classes        []string
	classSpans     []Span
	// Whether this mention says something about the element, and so declares it.
	explicit bool
	nodeSpan Span
}

// tail is what follows the source of a relationship, Pele's Tail.
type tail struct {
	explicitID     string
	explicitIDSpan *Span
	typ            string
	arrowType      int
	label          *draftLabel
	minlen         int
	target         *item
	span           Span
}

// parser is Pele's parser state.
type parser struct {
	src                  string
	types, starts, ends  []int
	p                    int
	lastEnd              int
	drafts               *drafts
	parent               *draftBoundary
	pendingJSON          []pendingJSON
	edgeCount, noteCount int
}

type pendingJSON struct {
	json    *draftJSON
	literal int
}

// parseFailure carries an error out of the parser, which Pele throws.
type parseFailure struct{ err *diagram.Error }

func (ps *parser) image(k int) string { return ps.src[ps.starts[k]:ps.ends[k]] }

func (ps *parser) span(k int) Span { return Span{ps.starts[k], ps.ends[k]} }

func (ps *parser) fail(detail string) {
	line, column := lineColumn(ps.src, ps.starts[ps.p])
	panic(parseFailure{&diagram.Error{
		Code: diagram.Syntax,
		Message: "Error parsing usecase diagram: " + detail + " at line " + strconv.Itoa(line) + ", column " +
			strconv.Itoa(column) + " [" + strconv.Itoa(unit(ps.src, ps.starts[ps.p])) + "," +
			strconv.Itoa(unit(ps.src, ps.ends[ps.p])) + ")",
		Type: "usecase", Line: line, Column: column,
	}})
}

func (ps *parser) mismatch(name string) {
	ps.fail("Expecting token of type --> " + name + " <-- but found --> '" + ps.image(ps.p) + "' <--")
}

func (ps *parser) expect(typ int) int {
	if ps.types[ps.p] != typ {
		ps.mismatch(tokenNames[typ])
	}
	ps.p++
	return ps.p - 1
}

// noAlternative fails as Chevrotain does when no alternative fits: with every token sequence
// that would have been accepted.
func (ps *parser) noAlternative(expected string) {
	var b strings.Builder
	for i, sequence := range strings.Split(expected, "|") {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString("  " + strconv.Itoa(i+1) + ". [" + sequence + "]")
	}
	ps.fail("Expecting: one of these possible Token sequences:\n" + b.String() + "\nbut found: '" + ps.image(ps.p) + "'")
}

// statementSequences are the token sequences a statement may start with, as Pele's
// statementSequences lists them.
func statementSequences() string {
	names := strings.Split(nameNames, "|")
	var metadata []string
	for _, name := range names {
		for _, next := range []string{"NEWLINE", "IDENTIFIER", "PLAIN_STRING", "RBRACE"} {
			metadata = append(metadata, name+", METADATA_START, "+next)
		}
	}
	follows := strings.Split("STEREOTYPE_START|CLASS_SEPARATOR|IDENTIFIER|"+arrowNames+"|NEWLINE|EOF", "|")
	var entities []string
	for i, name := range names {
		nexts := follows
		if i == 0 {
			nexts = append([]string{"LPAREN", "LBRACKET"}, follows...)
		}
		for _, next := range nexts {
			entities = append(entities, name+", "+next)
		}
	}
	all := append([]string{statementNames}, metadata...)
	all = append(all, entities...)
	all = append(all, metadata...)
	return strings.Join(all, "|")
}

func isWordToken(t int) bool { return t >= tUSECASE && t <= tIDENTIFIER }

func isLabelText(t int) bool { return t >= tUSECASE && t <= tLABEL_SYMBOL }

func (ps *parser) lineEnd() {
	ps.lastEnd = ps.ends[ps.p-1]
	if ps.types[ps.p] == tNEWLINE {
		ps.p++
	} else if ps.types[ps.p] != tEOF {
		ps.noAlternative("NEWLINE|EOF")
	}
}

// content is the text of a name or string token, without its quotes.
func (ps *parser) content(k int) Span {
	trim := 0
	switch ps.types[k] {
	case tMARKDOWN_STRING:
		trim = 2
	case tPLAIN_STRING:
		trim = 1
	}
	return Span{ps.starts[k] + trim, ps.ends[k] - trim}
}

func (ps *parser) tokenLabel(k int) draftLabel {
	at := ps.content(k)
	typ := LabelText
	if ps.types[k] == tMARKDOWN_STRING {
		typ = LabelMarkdown
	}
	return draftLabel{text: ps.src[at[0]:at[1]], typ: typ, span: at}
}

// generatedID is the id made from a label: each UTF-16 code unit that is not a word
// character becomes an underscore, as label.replace(/\W/g, '_') does.
func generatedID(label string) string {
	var b strings.Builder
	for i := 0; i < len(label); {
		r, size := utf8.DecodeRuneInString(label[i:])
		i += size
		switch {
		case r < utf8.RuneSelf && chars.IsWord(int(r)):
			b.WriteRune(r)
		case r > 0xffff:
			b.WriteString("__")
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

// label reads a label: a string, or unquoted text that runs to the next delimiter and is
// taken from the source as written.
func (ps *parser) label() draftLabel {
	typ := ps.types[ps.p]
	if typ == tPLAIN_STRING || typ == tMARKDOWN_STRING {
		ps.p++
		return ps.tokenLabel(ps.p - 1)
	}
	if !isLabelText(typ) {
		ps.noAlternative(labelNames)
	}
	first := ps.p
	for isLabelText(ps.types[ps.p]) {
		ps.p++
	}
	at := Span{ps.starts[first], ps.ends[ps.p-1]}
	return draftLabel{text: ps.src[at[0]:at[1]], typ: LabelText, span: at}
}

func isKey(t int) bool { return t == tIDENTIFIER || t == tPLAIN_STRING }

func (ps *parser) property() draftProperty {
	key := ps.p
	if !isKey(ps.types[ps.p]) {
		ps.noAlternative("IDENTIFIER|PLAIN_STRING")
	}
	ps.p++
	ps.expect(tCOLON)
	value := ps.p
	typ := ps.types[ps.p]
	if !isKey(typ) && typ != tTRUE && typ != tFALSE {
		ps.noAlternative("IDENTIFIER|PLAIN_STRING|TRUE|FALSE")
	}
	ps.p++
	keySpan := ps.content(key)
	valueSpan := ps.content(value)
	var v any
	switch typ {
	case tTRUE:
		v = true
	case tFALSE:
		v = false
	default:
		v = ps.src[valueSpan[0]:valueSpan[1]]
	}
	return draftProperty{
		MetadataOccurrence: MetadataOccurrence{
			Key:  ps.src[keySpan[0]:keySpan[1]],
			Span: Span{ps.starts[key], ps.ends[value]}, KeySpan: keySpan, ValueSpan: valueSpan,
		},
		value: v,
	}
}

// separatorAhead reports whether another property follows. A comma or line breaks before the
// closing brace do not count, but Chevrotain looks only three tokens ahead, so a comma and two
// line breaks, or three line breaks, are taken for a separator and the brace after them is
// then an error.
func (ps *parser) separatorAhead() bool {
	a, b, c := ps.types[ps.p], ps.types[ps.p+1], ps.types[ps.p+2]
	if a == tCOMMA {
		return isKey(b) || b == tNEWLINE && (c == tNEWLINE || isKey(c))
	}
	if a != tNEWLINE {
		return false
	}
	return b == tCOMMA || isKey(b) || b == tNEWLINE && (c == tNEWLINE || c == tCOMMA || isKey(c))
}

func (ps *parser) metadata() []draftProperty {
	ps.expect(tMETADATA_START)
	properties := []draftProperty{}
	for ps.types[ps.p] == tNEWLINE {
		ps.p++
	}
	if isKey(ps.types[ps.p]) {
		properties = append(properties, ps.property())
		for ps.separatorAhead() {
			if ps.types[ps.p] == tCOMMA {
				ps.p++
			} else {
				for ps.types[ps.p] == tNEWLINE {
					ps.p++
				}
				if ps.types[ps.p] == tCOMMA {
					ps.p++
				}
			}
			for ps.types[ps.p] == tNEWLINE {
				ps.p++
			}
			properties = append(properties, ps.property())
		}
		if ps.types[ps.p] == tCOMMA {
			ps.p++
		}
		for ps.types[ps.p] == tNEWLINE {
			ps.p++
		}
	}
	ps.expect(tRBRACE)
	return properties
}

func occurrences(properties []draftProperty) []MetadataOccurrence {
	out := make([]MetadataOccurrence, len(properties))
	for i, p := range properties {
		out[i] = p.MetadataOccurrence
	}
	return out
}

func (ps *parser) classSuffix(classes *[]string, spans *[]Span) {
	for {
		ps.p++
		id := ps.expect(tIDENTIFIER)
		*classes = append(*classes, ps.image(id))
		*spans = append(*spans, ps.span(id))
		if ps.types[ps.p] != tCOMMA {
			return
		}
	}
}

// name reads an id with an optional label in parentheses or brackets, or a string that stands
// for both.
func (ps *parser) name(brackets bool) *item {
	k := ps.p
	typ := ps.types[ps.p]
	it := &item{span: ps.content(k), generated: typ != tIDENTIFIER, classes: []string{}, classSpans: []Span{}}
	switch typ {
	case tIDENTIFIER:
		ps.p++
		it.id = ps.image(k)
		open := ps.types[ps.p]
		if open == tLPAREN || brackets && open == tLBRACKET {
			ps.p++
			it.label = ps.label()
			if open == tLPAREN {
				ps.expect(tRPAREN)
				it.shape = "ellipse"
			} else {
				ps.expect(tRBRACKET)
				it.shape = "rect"
			}
		} else {
			it.label = ps.tokenLabel(k)
		}
	case tPLAIN_STRING, tMARKDOWN_STRING:
		ps.p++
		it.label = ps.tokenLabel(k)
		it.id = generatedID(it.label.text)
	default:
		ps.noAlternative(nameNames)
	}
	return it
}

func (ps *parser) element(actor bool) *item {
	first := ps.p
	it := ps.name(!actor)
	if actor {
		it.shape = ""
	}
	if ps.types[ps.p] == tMETADATA_START {
		it.metadata = ps.metadata()
	}
	if ps.types[ps.p] == tSTEREOTYPE_START {
		ps.p++
		text := ps.expect(tSTEREOTYPE_TEXT)
		ps.expect(tSTEREOTYPE_END)
		it.stereotype = chars.Trim(ps.image(text))
		s := ps.span(text)
		it.stereotypeSpan = &s
	}
	if ps.types[ps.p] == tCLASS_SEPARATOR {
		ps.classSuffix(&it.classes, &it.classSpans)
	}
	it.explicit = it.shape != "" || it.metadata != nil || it.stereotype != ""
	it.nodeSpan = Span{ps.starts[first], ps.ends[ps.p-1]}
	return it
}

func (ps *parser) draft(it *item, kind string) *draftElement {
	return &draftElement{
		named:      named{id: it.id, label: it.label, span: it.span, generated: it.generated},
		kind:       kind,
		parent:     ps.parent,
		shape:      it.shape,
		metadata:   it.metadata,
		stereotype: it.stereotype,
		classes:    it.classes,
	}
}

func endpointOf(it *item, declares bool) draftEndpoint {
	return draftEndpoint{
		named:    named{id: it.id, label: it.label, span: it.span, generated: it.generated},
		declares: declares,
		classed:  len(it.classes) > 0,
	}
}

func occurrence(it *item, defines bool) NodeOccurrence {
	labelSpan := it.label.span
	out := NodeOccurrence{ID: it.id, Span: it.nodeSpan, IDSpan: it.span, LabelSpan: &labelSpan, Defines: defines}
	out.StereotypeSpan = it.stereotypeSpan
	if it.metadata != nil {
		out.Metadata = occurrences(it.metadata)
	}
	if len(it.classSpans) > 0 {
		out.ClassSpans = it.classSpans
	}
	return out
}

// labeledRight reports whether a label and then one of the operators that can close it come
// next.
func (ps *parser) labeledRight(closers uint8) bool {
	if t := ps.types[ps.p]; t < tUSECASE || t > tMARKDOWN_STRING {
		return false
	}
	for k := ps.p + 1; ; k++ {
		t := ps.types[k]
		if t == tMARKERLESS_SOLID || flags[t]&closers != 0 {
			return true
		}
		if t < tUSECASE || t > tMARKDOWN_STRING {
			return false
		}
	}
}

// minimumLength is the ranks a relationship's operator asks for: each dash beyond the second
// asks for one more between the two ends.
func (ps *parser) minimumLength(k int) int {
	dashes := ps.ends[k] - ps.starts[k]
	if ps.types[k] != tMARKERLESS_SOLID {
		dashes--
	}
	return max(1, dashes-1)
}

func (ps *parser) relationTail() *tail {
	first := ps.p
	tl := &tail{typ: "association", arrowType: SolidArrow, minlen: 1}
	if ps.types[ps.p] == tIDENTIFIER && ps.types[ps.p+1] == tAT {
		tl.explicitID = ps.image(ps.p)
		s := ps.span(ps.p)
		tl.explicitIDSpan = &s
		ps.p += 2
	}
	k := ps.p
	typ := ps.types[ps.p]
	ps.p++
	switch typ {
	case tDEPENDENCY_ARROW:
		ps.expect(tCOLON)
		if ps.types[ps.p] != tINCLUDE && ps.types[ps.p] != tEXTEND {
			ps.noAlternative("INCLUDE|EXTEND")
		}
		if ps.types[ps.p] == tINCLUDE {
			tl.typ = "include"
		} else {
			tl.typ = "extend"
		}
		tl.label = &draftLabel{text: tl.typ, typ: LabelText, span: ps.span(ps.p)}
		ps.p++
	case tGENERALIZATION:
		tl.typ = "generalization"
	case tFORWARD_SOLID:
		tl.minlen = ps.minimumLength(k)
	case tBACKWARD_SOLID:
		tl.arrowType = BackArrow
		// Only the shortest operator can open a label: `A <-- label -- B`.
		if ps.ends[k]-ps.starts[k] == 3 && ps.labeledRight(0) {
			l := ps.label()
			tl.label = &l
			tl.minlen = ps.minimumLength(ps.expect(tMARKERLESS_SOLID))
		} else {
			tl.minlen = ps.minimumLength(k)
		}
	case tMARKERLESS_SOLID:
		tl.arrowType = LineSolid
		if ps.ends[k]-ps.starts[k] == 2 && ps.labeledRight(fClosesLabel) {
			l := ps.label()
			tl.label = &l
			switch ps.types[ps.p] {
			case tFORWARD_SOLID:
				tl.arrowType = SolidArrow
			case tFORWARD_CIRCLE:
				tl.arrowType = CircleArrow
			case tFORWARD_CROSS:
				tl.arrowType = CrossArrow
			case tMARKERLESS_SOLID:
			default:
				ps.noAlternative("FORWARD_SOLID|MARKERLESS_SOLID|FORWARD_CIRCLE|FORWARD_CROSS")
			}
			if tl.arrowType == SolidArrow || tl.arrowType == LineSolid {
				tl.minlen = ps.minimumLength(ps.p)
			}
			ps.p++
		} else {
			tl.minlen = ps.minimumLength(k)
		}
	case tFORWARD_CIRCLE:
		tl.arrowType = CircleArrow
	case tFORWARD_CROSS:
		tl.arrowType = CrossArrow
	case tBACKWARD_CIRCLE, tBACKWARD_CROSS:
		if typ == tBACKWARD_CIRCLE {
			tl.arrowType = CircleArrowReversed
		} else {
			tl.arrowType = CrossArrowReversed
		}
		if ps.labeledRight(0) {
			l := ps.label()
			tl.label = &l
			ps.expect(tMARKERLESS_SOLID)
		}
	default:
		ps.p--
		ps.noAlternative(arrowNames)
	}
	tl.target = ps.element(false)
	tl.span = Span{ps.starts[first], ps.ends[ps.p-1]}
	return tl
}

func (ps *parser) relationship(source *item, declaration bool, tl *tail, nodes []NodeOccurrence) *Statement {
	target := tl.target
	if target.explicit {
		ps.drafts.elements = append(ps.drafts.elements, ps.draft(target, "usecase"))
	}
	ps.drafts.relationships = append(ps.drafts.relationships, &draftRelationship{
		source:    endpointOf(source, declaration),
		target:    endpointOf(target, target.explicit),
		span:      tl.span,
		id:        tl.explicitID,
		idSpan:    tl.explicitIDSpan,
		typ:       tl.typ,
		arrowType: tl.arrowType,
		label:     tl.label,
		minlen:    tl.minlen,
	})
	nodes = append(nodes, occurrence(target, target.explicit))
	out := &Statement{Kind: "edge", Nodes: nodes}
	id := tl.explicitID
	if tl.explicitIDSpan == nil {
		id = "edge-" + strconv.Itoa(ps.edgeCount)
		ps.edgeCount++
	}
	edge := EdgeOccurrence{ID: id, IDSpan: tl.explicitIDSpan}
	if tl.label != nil {
		s := tl.label.span
		edge.LabelSpan = &s
	}
	out.Edges = []EdgeOccurrence{edge}
	return out
}

// actors reads `actor A, B`, which declares several actors; at the top level `actor A --> B`
// declares one and relates it.
func (ps *parser) actors(top bool) *Statement {
	ps.p++
	items := []*item{ps.element(true)}
	var tl *tail
	if ps.types[ps.p] == tCOMMA {
		for ps.types[ps.p] == tCOMMA {
			ps.p++
			items = append(items, ps.element(true))
		}
	} else if top && flags[ps.types[ps.p]]&fRelation != 0 {
		tl = ps.relationTail()
	}
	ps.lineEnd()
	for _, it := range items {
		ps.drafts.elements = append(ps.drafts.elements, ps.draft(it, "actor"))
	}
	nodes := make([]NodeOccurrence, len(items))
	for i, it := range items {
		nodes[i] = occurrence(it, true)
	}
	if tl != nil {
		return ps.relationship(items[0], true, tl, nodes)
	}
	return &Statement{Kind: "node", Nodes: nodes}
}

func (ps *parser) entity(top bool) *Statement {
	source := ps.element(false)
	var tl *tail
	if top && flags[ps.types[ps.p]]&fRelation != 0 {
		tl = ps.relationTail()
	}
	ps.lineEnd()
	if tl == nil {
		source.explicit = true
		if source.shape == "" {
			source.shape = "ellipse"
		}
	}
	if source.explicit {
		ps.drafts.elements = append(ps.drafts.elements, ps.draft(source, "usecase"))
	}
	nodes := []NodeOccurrence{occurrence(source, source.explicit)}
	if tl != nil {
		return ps.relationship(source, source.explicit, tl, nodes)
	}
	return &Statement{Kind: "node", Nodes: nodes}
}

func (ps *parser) boundary() *Statement {
	ps.p++
	named := ps.name(true)
	var inline []draftProperty
	if ps.types[ps.p] == tMETADATA_START {
		inline = ps.metadata()
	}
	if ps.types[ps.p] == tCLASS_SEPARATOR {
		ps.classSuffix(&named.classes, &named.classSpans)
	}
	ps.lineEnd()
	group := &draftBoundary{named: namedOf(named), classes: named.classes}
	ps.drafts.boundaries = append(ps.drafts.boundaries, group)
	ps.parent = group
	var children []*Statement
	for flags[ps.types[ps.p]]&fBoundaryLine != 0 {
		children = append(children, ps.line(false))
	}
	ps.parent = nil
	end := ps.expect(tEND)
	ps.lineEnd()
	titleSpan := group.label.span
	idSpan := group.span
	endSpan := ps.span(end)
	out := &Statement{
		Kind: "group", Group: group.id, hasGroup: true, IDSpan: &idSpan, TitleSpan: &titleSpan, EndSpan: &endSpan,
		ClassSpans: named.classSpans, Children: children,
	}
	if inline != nil {
		out.Metadata = occurrences(inline)
		ps.drafts.metadata = append(ps.drafts.metadata, &draftMetadata{target: group.id, span: group.span, properties: inline, statement: out})
	}
	return out
}

func namedOf(it *item) named {
	return named{id: it.id, label: it.label, span: it.span, generated: it.generated}
}

func (ps *parser) style() string {
	first := ps.p
	typ := ps.types[ps.p]
	switch {
	case isWordToken(typ) || typ == tCSS_IDENTIFIER:
		ps.p++
	case typ == tMARKERLESS_SOLID:
		ps.p++
		if !isWordToken(ps.types[ps.p]) {
			ps.mismatch("WORD")
		}
		ps.p++
	default:
		ps.noAlternative("WORD|CSS_IDENTIFIER|MARKERLESS_SOLID")
	}
	ps.expect(tCOLON)
	if flags[ps.types[ps.p]]&fStylePart == 0 {
		ps.fail("Expecting: expecting at least one iteration which starts with one of these possible Token sequences::\n  <[" +
			strings.ReplaceAll(stylePartNames, "|", "] ,[") + "]>\nbut found: '" + ps.image(ps.p) + "'")
	}
	for flags[ps.types[ps.p]]&fStylePart != 0 {
		ps.p++
	}
	// The value is taken from the source, so the spaces inside `1px solid red` survive.
	return strings.ReplaceAll(ps.src[ps.starts[first]:ps.ends[ps.p-1]], "\\,", ",")
}

func (ps *parser) styles() []string {
	out := []string{ps.style()}
	for ps.types[ps.p] == tCOMMA {
		ps.p++
		out = append(out, ps.style())
	}
	return out
}

func (ps *parser) identifiers() []int {
	out := []int{ps.expect(tIDENTIFIER)}
	for ps.types[ps.p] == tCOMMA {
		ps.p++
		out = append(out, ps.expect(tIDENTIFIER))
	}
	return out
}

func (ps *parser) reference(k int) NodeOccurrence {
	at := ps.content(k)
	return NodeOccurrence{ID: ps.image(k), Span: at, IDSpan: at}
}

func (ps *parser) topStatement() *Statement {
	k := ps.p
	typ := ps.types[ps.p]
	var out *Statement
	switch typ {
	case tACC_TITLE_LINE, tACC_DESCR_LINE, tACC_DESCR_BLOCK:
		ps.p++
		text := ps.image(k)
		switch typ {
		case tACC_DESCR_BLOCK:
			ps.drafts.accDescription = chars.Trim(text[strings.IndexByte(text, '{')+1 : strings.LastIndexByte(text, '}')])
		case tACC_DESCR_LINE:
			ps.drafts.accDescription = chars.Trim(text[strings.IndexByte(text, ':')+1:])
		default:
			ps.drafts.accTitle = chars.Trim(text[strings.IndexByte(text, ':')+1:])
		}
		if typ == tACC_TITLE_LINE {
			out = &Statement{Kind: "accTitle"}
		} else {
			out = &Statement{Kind: "accDescr"}
		}
	case tDIRECTION:
		ps.p++
		value := ps.types[ps.p]
		if value < tTD || value > tRL {
			ps.noAlternative("TD|TB|BT|LR|RL")
		}
		if value == tTD {
			ps.drafts.direction = "TB"
		} else {
			ps.drafts.direction = ps.image(ps.p)
		}
		ps.p++
		out = &Statement{Kind: "direction"}
	case tACTOR:
		return ps.actors(true)
	case tSYSTEM_BOUNDARY:
		return ps.boundary()
	case tNOTE:
		ps.p++
		ps.expect(tFOR)
		target := ps.expect(tIDENTIFIER)
		text := ps.label()
		ps.drafts.notes = append(ps.drafts.notes, draftNote{target: ps.image(target), span: ps.span(target), label: text})
		refSpan := text.span
		out = &Statement{Kind: "note", Ref: "note-" + strconv.Itoa(ps.noteCount), RefSpan: &refSpan,
			Nodes: []NodeOccurrence{ps.reference(target)}}
		ps.noteCount++
	case tJSON_DECLARATION_START:
		ps.p++
		literal := ps.expect(tJSON_OBJECT_LITERAL)
		classes := []string{}
		classSpans := []Span{}
		if ps.types[ps.p] == tCLASS_SEPARATOR {
			ps.classSuffix(&classes, &classSpans)
		}
		from := ps.starts[k] + 4
		for at(ps.src, from) == ' ' || at(ps.src, from) == '\t' {
			from++
		}
		to := from
		for chars.IsWord(at(ps.src, to)) {
			to++
		}
		js := &draftJSON{id: ps.src[from:to], span: Span{from, to}, classes: classes}
		ps.drafts.jsons = append(ps.drafts.jsons, js)
		ps.pendingJSON = append(ps.pendingJSON, pendingJSON{js, literal})
		out = &Statement{Kind: "json",
			Nodes:      []NodeOccurrence{{ID: js.id, Span: js.span, IDSpan: js.span, Defines: true, ClassSpans: classSpans}},
			ClassSpans: classSpans}
	case tCLASS_DEF:
		ps.p++
		ids := ps.identifiers()
		names := make([]string, len(ids))
		for i, id := range ids {
			names[i] = ps.image(id)
		}
		ps.drafts.classDefs = append(ps.drafts.classDefs, draftClassDef{ids: names, styles: ps.styles()})
		refSpan := ps.span(ids[0])
		out = &Statement{Kind: "classDef", Ref: ps.image(ids[0]), RefSpan: &refSpan}
	case tCLASS:
		ps.p++
		targets := ps.identifiers()
		classes := ps.identifiers()
		assignment := draftClasses{}
		for _, id := range targets {
			assignment.targets = append(assignment.targets, draftTarget{id: ps.image(id), span: ps.span(id)})
		}
		for _, id := range classes {
			assignment.classes = append(assignment.classes, ps.image(id))
		}
		ps.drafts.classes = append(ps.drafts.classes, assignment)
		refSpan := ps.span(classes[0])
		out = &Statement{Kind: "classAssign", Ref: ps.image(classes[0]), RefSpan: &refSpan}
		for _, id := range targets {
			out.Nodes = append(out.Nodes, ps.reference(id))
		}
	case tSTYLE:
		ps.p++
		target := ps.expect(tIDENTIFIER)
		ps.drafts.styles = append(ps.drafts.styles, draftStyle{target: ps.image(target), span: ps.span(target), styles: ps.styles()})
		out = &Statement{Kind: "style", Nodes: []NodeOccurrence{ps.reference(target)}}
	default:
		next := ps.types[ps.p+1]
		if next == tMETADATA_START {
			if flags[ps.types[ps.p+2]]&fMetadataFirst == 0 {
				ps.noAlternative(statementSequences())
			}
			ps.p++
			target := ps.reference(k)
			if typ != tIDENTIFIER {
				target.ID = generatedID(ps.src[target.Span[0]:target.Span[1]])
			}
			assigned := ps.metadata()
			out = &Statement{Kind: "metadata", Nodes: []NodeOccurrence{target}, Metadata: occurrences(assigned)}
			ps.drafts.metadata = append(ps.drafts.metadata, &draftMetadata{target: target.ID, span: target.Span, properties: assigned, statement: out})
		} else {
			follows := flags[next]&fFollowsName != 0 || typ == tIDENTIFIER && (next == tLPAREN || next == tLBRACKET)
			if !follows || typ == tIDENTIFIER && forbidden[strings.ToLower(ps.image(k))] {
				ps.noAlternative(statementSequences())
			}
			return ps.entity(true)
		}
	}
	ps.lineEnd()
	return out
}

func (ps *parser) line(top bool) *Statement {
	k := ps.p
	typ := ps.types[ps.p]
	if typ == tNEWLINE {
		ps.p++
		return &Statement{Kind: "blank", Span: ps.span(k)}
	}
	if typ == tCOMMENT {
		ps.p++
		ps.lineEnd()
		return &Statement{Kind: "comment", Span: ps.span(k)}
	}
	var out *Statement
	switch {
	case top:
		out = ps.topStatement()
	case typ == tACTOR:
		out = ps.actors(false)
	default:
		out = ps.entity(false)
	}
	out.Span = Span{ps.starts[k], ps.lastEnd}
	for i := range out.Edges {
		out.Edges[i].Span = out.Span
	}
	return out
}

// parse parses a use case diagram, as Pele's parseUsecase does.
func parse(source, title string) (model *Model, err error) {
	t, lexErr := tokenize(source)
	if lexErr != nil {
		return nil, lexErr
	}
	ps := &parser{src: source, types: t.types, starts: t.starts, ends: t.ends, drafts: &drafts{}}
	defer func() {
		if r := recover(); r != nil {
			f, ok := r.(parseFailure)
			if !ok {
				panic(r)
			}
			model, err = nil, f.err
		}
	}()
	header := ps.expect(tUSECASE)
	ps.lineEnd()
	var statements []*Statement
	for flags[ps.types[ps.p]]&fLine != 0 {
		statements = append(statements, ps.line(true))
	}
	if ps.types[ps.p] != tEOF {
		ps.fail("Redundant input, expecting EOF but found: " + ps.image(ps.p))
	}
	// The whole text has to be well formed before the JSON in it is looked at, as in Mermaid.
	for _, pending := range ps.pendingJSON {
		literal := pending.literal
		value, order, jsonErr := parseOrderedJSON(ps.image(literal), func() (int, int) {
			return lineColumn(source, ps.starts[literal])
		})
		if jsonErr != nil {
			return nil, jsonErr
		}
		pending.json.value = value
		pending.json.propertyOrder = order
	}
	return resolve(source, ps.drafts, ps.span(header), statements, title)
}
