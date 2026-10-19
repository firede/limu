package c4

import (
	"math"

	"github.com/firede/limu/internal/chars"
)

// attr is an argument as the parser reads it, Pele's C4Attr: a value written where it is, or,
// when named, a `$key="value"` argument.
type attr struct {
	named      bool
	key, value string
}

// Named is a `$key="value"` argument, Pele's C4Named.
type Named struct{ Key, Value string }

// The fields of an element or a relation that may be left undefined, by Pele's names, as
// bits of a set.
const (
	fType uint32 = 1 << iota
	fDescr
	fTechn
	fSprite
	fTags
	fLink
	fBgColor
	fFontColor
	fBorderColor
	fShadowing
	fShape
	fLegendText
	fLegendSprite
	fTextColor
	fLineColor
	fOffsetX
	fOffsetY
)

// Element is what shapes and boundaries share, Pele's C4Element. A field Pele leaves
// undefined is empty, and Defined tells it from one set to "".
type Element struct {
	Alias, Label string
	// A named argument written where the alias or the label belongs. It is applied to the
	// field it names.
	AliasAttr, LabelAttr *Named
	Parent               *Boundary

	Type, Descr, Techn, Sprite, Tags, Link     string
	BgColor, FontColor, BorderColor            string
	Shadowing, Shape, LegendText, LegendSprite string
	// Extra holds the named arguments that set no known field, in the order they were first
	// given; a later one of the same key replaces the value.
	Extra []Named

	defined uint32
}

// Shape is a person, a system, a container or a component, Pele's C4Shape.
type Shape struct {
	Element
	// Kind is Mermaid's typeC4Shape: person, external_system_db, container_queue and so on.
	Kind string
}

// Boundary is a boundary or a deployment node, Pele's C4Boundary.
type Boundary struct {
	Element
	// NodeType is set on deployment nodes: node, nodeL or nodeR.
	NodeType string
}

// Rel is a relation, Pele's C4Rel.
type Rel struct {
	Type, From, To, Label       string
	FromAttr, ToAttr, LabelAttr *Named

	Techn, Descr, Sprite, Tags, Link, TextColor, LineColor string
	// OffsetX and OffsetY are as parseInt reads them, NaN among them.
	OffsetX, OffsetY float64
	Extra            []Named

	defined uint32
}

// Defined reports whether the field of the name Pele gives it, as "descr", is defined.
func (e *Element) Defined(field string) bool {
	_, bit := e.field(field)
	return bit != 0 && e.defined&bit != 0
}

// Defined reports whether the field of the name Pele gives it, as "techn", is defined.
func (r *Rel) Defined(field string) bool {
	_, bit := r.field(field)
	switch field {
	case "offsetX":
		bit = fOffsetX
	case "offsetY":
		bit = fOffsetY
	}
	return bit != 0 && r.defined&bit != 0
}

// field is the field of an element of the name, and its bit, as ELEMENT_FIELDS lists them; a
// label has no bit, as it is always defined. nil for a name not in the list.
func (e *Element) field(name string) (*string, uint32) {
	switch name {
	case "label":
		return &e.Label, 0
	case "type":
		return &e.Type, fType
	case "descr":
		return &e.Descr, fDescr
	case "techn":
		return &e.Techn, fTechn
	case "sprite":
		return &e.Sprite, fSprite
	case "tags":
		return &e.Tags, fTags
	case "link":
		return &e.Link, fLink
	case "bgColor":
		return &e.BgColor, fBgColor
	case "fontColor":
		return &e.FontColor, fFontColor
	case "borderColor":
		return &e.BorderColor, fBorderColor
	case "shadowing":
		return &e.Shadowing, fShadowing
	case "shape":
		return &e.Shape, fShape
	case "legendText":
		return &e.LegendText, fLegendText
	case "legendSprite":
		return &e.LegendSprite, fLegendSprite
	}
	return nil, 0
}

// field is the field of a relation of the name, and its bit, as REL_FIELDS lists them.
func (r *Rel) field(name string) (*string, uint32) {
	switch name {
	case "label":
		return &r.Label, 0
	case "descr":
		return &r.Descr, fDescr
	case "techn":
		return &r.Techn, fTechn
	case "sprite":
		return &r.Sprite, fSprite
	case "tags":
		return &r.Tags, fTags
	case "link":
		return &r.Link, fLink
	case "textColor":
		return &r.TextColor, fTextColor
	case "lineColor":
		return &r.LineColor, fLineColor
	}
	return nil, 0
}

// target is an element or a relation, what Pele's named and put assign to.
type target interface {
	field(name string) (*string, uint32)
	define(bit uint32)
	isDefined(bit uint32) bool
	extra() *[]Named
}

func (e *Element) define(bit uint32)         { e.defined |= bit }
func (e *Element) isDefined(bit uint32) bool { return e.defined&bit != 0 }
func (e *Element) extra() *[]Named           { return &e.Extra }
func (r *Rel) define(bit uint32)             { r.defined |= bit }
func (r *Rel) isDefined(bit uint32) bool     { return r.defined&bit != 0 }
func (r *Rel) extra() *[]Named               { return &r.Extra }

// set sets the field of the name.
func set(t target, name, value string) {
	f, bit := t.field(name)
	*f = value
	t.define(bit)
}

// setExtra keeps a named argument that sets no known field, as a Map keeps a key: where it
// was first set.
func setExtra(t target, key, value string) {
	list := t.extra()
	for i := range *list {
		if (*list)[i].Key == key {
			(*list)[i].Value = value
			return
		}
	}
	*list = append(*list, Named{key, value})
}

// parseInt is JavaScript's parseInt with no radix: in base 16 after 0x, in base 10 otherwise.
func parseInt(s string) float64 {
	t := chars.TrimStart(s)
	body := t
	if body != "" && (body[0] == '+' || body[0] == '-') {
		body = body[1:]
	}
	if len(body) >= 2 && body[0] == '0' && (body[1] == 'x' || body[1] == 'X') {
		return chars.ParseInt(t, 16)
	}
	return chars.ParseInt(t, 10)
}

// named applies a named argument, as Pele's named does: Mermaid assigns it to whatever
// property it names, structural ones included; here it reaches only the known fields, and any
// other name is kept aside in Extra.
func named(t target, a attr) {
	if r, ok := t.(*Rel); ok && (a.key == "offsetX" || a.key == "offsetY") {
		v := parseInt(a.value)
		if a.key == "offsetX" {
			r.OffsetX = v
			r.defined |= fOffsetX
		} else {
			r.OffsetY = v
			r.defined |= fOffsetY
		}
		return
	}
	if f, _ := t.field(a.key); f != nil {
		set(t, a.key, a.value)
		return
	}
	setExtra(t, a.key, a.value)
}

// put fills a field with an argument, as Pele's put does: a positional argument fills the
// field of its slot, a named one the field it names, whichever slot it is in. has is false
// for an argument not given.
func put(t target, field string, a attr, has bool) {
	if !has {
		return
	}
	if !a.named {
		set(t, field, a.value)
	} else {
		named(t, a)
	}
}

// putText fills the description or the technology, as Pele's text does: one not given is
// defined as "" when it was not before.
func putText(t target, field string, a attr, has bool) {
	if !has {
		if _, bit := t.field(field); !t.isDefined(bit) {
			set(t, field, "")
		}
		return
	}
	put(t, field, a, true)
}

// arg is the argument at index i, and whether there is one.
func arg(attrs []attr, i int) (attr, bool) {
	if i < len(attrs) {
		return attrs[i], true
	}
	return attr{}, false
}

// setLabel sets a label, as Pele's label does: a named argument in its slot leaves it empty
// and fills the field it names.
func setLabel(t target, labelAttr **Named, label *string, a attr, has bool) {
	*labelAttr = nil
	if has && a.named {
		*label = ""
		*labelAttr = &Named{a.key, a.value}
		named(t, a)
		return
	}
	*label = a.value
}

// DB is a parsed C4 diagram, Pele's C4Db, and the builder the parser calls.
type DB struct {
	// C4Type is the header keyword: C4Context, C4Container, C4Component, C4Dynamic or
	// C4Deployment.
	C4Type string
	// Title is the title, when HasTitle.
	Title    string
	HasTitle bool
	// AccTitle is never set: the grammar hands accTitle to the title.
	AccTitle string
	// AccDescr is the accessible description, when HasAccDescr.
	AccDescr    string
	HasAccDescr bool
	Shapes      []*Shape
	// The first boundary is the unnamed one that holds everything at the top level.
	Boundaries []*Boundary
	Rels       []*Rel
	// ShapeInRow and BoundaryInRow are as parseInt reads them, which may be past any int.
	ShapeInRow, BoundaryInRow float64

	shapeIndex map[string]*Shape
	// The top-level boundary is not listed, so a boundary a diagram names `global` is its own
	// box. In Mermaid it replaces the top level and nothing is drawn.
	boundaryIndex map[string]*Boundary
	relIndex      map[string]map[string]*Rel
	current       *Boundary
	stack         []*Boundary
}

func newDB() *DB {
	global := &Boundary{Element: Element{Alias: "global", Label: "global", Type: "global", defined: fType}}
	return &DB{
		Boundaries:    []*Boundary{global},
		ShapeInRow:    4,
		BoundaryInRow: 2,
		shapeIndex:    map[string]*Shape{},
		boundaryIndex: map[string]*Boundary{},
		relIndex:      map[string]map[string]*Rel{},
		current:       global,
	}
}

func (db *DB) setTitle(title string) {
	db.Title = title
	db.HasTitle = true
}

func (db *DB) setAccDescription(descr string) {
	db.AccDescr = chars.DropIndents(descr)
	db.HasAccDescr = true
}

// element finds the element of an alias, or makes one, and sets its label, as Pele's element
// does.
func element[E any](list *[]*E, index map[string]*E, base func(*E) *Element, alias attr, title attr, hasTitle bool) *E {
	var el *E
	if !alias.named {
		el = index[alias.value]
	}
	if el == nil {
		el = new(E)
		*list = append(*list, el)
		e := base(el)
		if !alias.named {
			e.Alias = alias.value
			index[alias.value] = el
		} else {
			e.AliasAttr = &Named{alias.key, alias.value}
			named(e, alias)
		}
	}
	e := base(el)
	setLabel(e, &e.LabelAttr, &e.Label, title, hasTitle)
	return el
}

func shapeBase(s *Shape) *Element       { return &s.Element }
func boundaryBase(b *Boundary) *Element { return &b.Element }

// addPersonOrSystem: alias, label, ?descr, ?sprite, ?tags, $link.
func (db *DB) addPersonOrSystem(kind string, attrs []attr) {
	title, has := arg(attrs, 1)
	shape := element(&db.Shapes, db.shapeIndex, shapeBase, attrs[0], title, has)
	a, has := arg(attrs, 2)
	putText(&shape.Element, "descr", a, has)
	db.finishShape(shape, kind, attrs, 3)
}

// addContainer: alias, label, ?techn, ?descr, ?sprite, ?tags, $link.
func (db *DB) addContainer(kind string, attrs []attr) {
	title, has := arg(attrs, 1)
	shape := element(&db.Shapes, db.shapeIndex, shapeBase, attrs[0], title, has)
	a, has := arg(attrs, 2)
	putText(&shape.Element, "techn", a, has)
	a, has = arg(attrs, 3)
	putText(&shape.Element, "descr", a, has)
	db.finishShape(shape, kind, attrs, 4)
}

func (db *DB) addComponent(kind string, attrs []attr) { db.addContainer(kind, attrs) }

func (db *DB) finishShape(shape *Shape, kind string, attrs []attr, at int) {
	for k, field := range [...]string{"sprite", "tags", "link"} {
		a, has := arg(attrs, at+k)
		put(&shape.Element, field, a, has)
	}
	shape.Kind = kind
	shape.Parent = db.current
}

func (db *DB) boundary(attrs []attr, fallback string) *Boundary {
	title, has := arg(attrs, 1)
	b := element(&db.Boundaries, db.boundaryIndex, boundaryBase, attrs[0], title, has)
	if a, has := arg(attrs, 2); !has {
		set(&b.Element, "type", fallback)
	} else {
		put(&b.Element, "type", a, true)
	}
	return b
}

func (db *DB) open(b *Boundary) {
	b.Parent = db.current
	db.stack = append(db.stack, db.current)
	db.current = b
}

// addPersonOrSystemBoundary: alias, label, ?type, ?tags, $link.
func (db *DB) addPersonOrSystemBoundary(attrs []attr, fallback string) {
	b := db.boundary(attrs, fallback)
	a, has := arg(attrs, 3)
	put(&b.Element, "tags", a, has)
	a, has = arg(attrs, 4)
	put(&b.Element, "link", a, has)
	db.open(b)
}

func (db *DB) addContainerBoundary(attrs []attr) { db.addPersonOrSystemBoundary(attrs, "container") }

// addDeploymentNode: alias, label, ?type, ?descr, ?sprite, ?tags, $link.
func (db *DB) addDeploymentNode(nodeType string, attrs []attr) {
	b := db.boundary(attrs, "node")
	a, has := arg(attrs, 3)
	putText(&b.Element, "descr", a, has)
	for k, field := range [...]string{"sprite", "tags", "link"} {
		a, has := arg(attrs, 4+k)
		put(&b.Element, field, a, has)
	}
	b.NodeType = nodeType
	db.open(b)
}

func (db *DB) popBoundaryParseStack() {
	if n := len(db.stack); n > 0 {
		db.current = db.stack[n-1]
		db.stack = db.stack[:n-1]
	} else {
		db.current = db.Boundaries[0]
	}
}

// addRel: from, to, label, ?techn, ?descr, ?sprite, ?tags, $link.
func (db *DB) addRel(typ string, attrs []attr) {
	if len(attrs) < 3 {
		return
	}
	from, to := attrs[0], attrs[1]
	known := !from.named && !to.named
	var rel *Rel
	if known {
		rel = db.relIndex[from.value][to.value]
	}
	if rel == nil {
		rel = &Rel{Type: typ}
		db.Rels = append(db.Rels, rel)
		if known {
			targets := db.relIndex[from.value]
			if targets == nil {
				targets = map[string]*Rel{}
				db.relIndex[from.value] = targets
			}
			targets[to.value] = rel
		}
		if !from.named {
			rel.From = from.value
		} else {
			rel.FromAttr = &Named{from.key, from.value}
		}
		if !to.named {
			rel.To = to.value
		} else {
			rel.ToAttr = &Named{to.key, to.value}
		}
	}
	rel.Type = typ
	setLabel(rel, &rel.LabelAttr, &rel.Label, attrs[2], true)
	a, has := arg(attrs, 3)
	putText(rel, "techn", a, has)
	a, has = arg(attrs, 4)
	putText(rel, "descr", a, has)
	for k, field := range [...]string{"sprite", "tags", "link"} {
		a, has := arg(attrs, 5+k)
		put(rel, field, a, has)
	}
}

// getC4Shape is the shape or the boundary of an alias, as Pele's getC4Shape finds it: a
// shape first.
func (db *DB) getC4Shape(alias string) *Element {
	if s := db.shapeIndex[alias]; s != nil {
		return &s.Element
	}
	if b := db.boundaryIndex[alias]; b != nil {
		return &b.Element
	}
	return nil
}

// updateElStyle: elementName, ?bgColor, ?fontColor, ?borderColor, ?shadowing, ?shape,
// ?sprite, ?techn, ?legendText, ?legendSprite.
func (db *DB) updateElStyle(attrs []attr) {
	if attrs[0].named {
		return
	}
	el := db.getC4Shape(attrs[0].value)
	if el == nil {
		return
	}
	for k, field := range [...]string{"bgColor", "fontColor", "borderColor", "shadowing", "shape", "sprite", "techn", "legendText", "legendSprite"} {
		a, has := arg(attrs, 1+k)
		put(el, field, a, has)
	}
}

// updateRelStyle: from, to, ?textColor, ?lineColor, ?offsetX, ?offsetY.
func (db *DB) updateRelStyle(attrs []attr) {
	from, _ := arg(attrs, 0)
	to, hasTo := arg(attrs, 1)
	if from.named || !hasTo || to.named {
		return
	}
	rel := db.relIndex[from.value][to.value]
	if rel == nil {
		return
	}
	a, has := arg(attrs, 2)
	put(rel, "textColor", a, has)
	a, has = arg(attrs, 3)
	put(rel, "lineColor", a, has)
	if x, has := arg(attrs, 4); has && !x.named {
		rel.OffsetX = parseInt(x.value)
		rel.defined |= fOffsetX
	} else if has {
		named(rel, x)
	}
	if y, has := arg(attrs, 5); has && !y.named {
		rel.OffsetY = parseInt(y.value)
		rel.defined |= fOffsetY
	} else if has {
		named(rel, y)
	}
}

// updateLayoutConfig: ?c4ShapeInRow, ?c4BoundaryInRow. As in Mermaid, the slot decides which
// one a value sets, not its name.
func (db *DB) updateLayoutConfig(attrs []attr) {
	count := func(i int) float64 {
		a, has := arg(attrs, i)
		if !has {
			return math.NaN()
		}
		return parseInt(a.value)
	}
	shapes, boundaries := count(0), count(1)
	if shapes >= 1 {
		db.ShapeInRow = shapes
	}
	if boundaries >= 1 {
		db.BoundaryInRow = boundaries
	}
}
