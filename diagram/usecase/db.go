package usecase

import (
	"slices"
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// draftLabel is a label as written, Pele's DraftLabel.
type draftLabel struct {
	text string
	typ  LabelType
	span Span
}

// draftProperty is a property of a metadata block: a string or a bool. Its KeySpan is also
// where an error about it points.
type draftProperty struct {
	MetadataOccurrence
	value any
}

// named is what every mention of an id carries: where it is, and whether the id was made from
// a label. Pele's Named.
type named struct {
	id        string
	label     draftLabel
	span      Span
	generated bool
}

type draftElement struct {
	named
	kind       string
	parent     *draftBoundary
	shape      string
	metadata   []draftProperty
	stereotype string
	classes    []string
}

type draftBoundary struct {
	named
	classes []string
}

type draftJSON struct {
	id            string
	value         *Object
	propertyOrder *KeyOrder
	span          Span
	classes       []string
}

type draftEndpoint struct {
	named
	// Whether this mention declares the element, and whether it carries a `:::` suffix.
	declares bool
	classed  bool
}

type draftRelationship struct {
	source, target draftEndpoint
	span           Span
	id             string
	idSpan         *Span
	typ            string
	arrowType      int
	label          *draftLabel
	minlen         int
}

type draftNote struct {
	target string
	span   Span
	label  draftLabel
}

type draftMetadata struct {
	target     string
	span       Span
	properties []draftProperty
	statement  *Statement
}

type draftTarget struct {
	id   string
	span Span
}

type draftClasses struct {
	targets []draftTarget
	classes []string
}

type draftStyle struct {
	target string
	span   Span
	styles []string
}

type draftClassDef struct {
	ids    []string
	styles []string
}

// drafts are everything the parser read, in source order, before any of it is resolved:
// Pele's Drafts.
type drafts struct {
	elements       []*draftElement
	boundaries     []*draftBoundary
	jsons          []*draftJSON
	relationships  []*draftRelationship
	notes          []draftNote
	metadata       []*draftMetadata
	classes        []draftClasses
	styles         []draftStyle
	classDefs      []draftClassDef
	direction      string
	accTitle       string
	accDescription string
}

// origin is what an id was first declared as, Pele's Origin.
type origin struct {
	kind      string
	span      Span
	generated bool
	// The label a generated id was derived from.
	label string
}

// elementState is an actor or a use case as the declarations so far make it, Pele's
// ElementState.
type elementState struct {
	named
	kind           string
	parent         *draftBoundary
	shape          string
	stereotype     string
	stereotypeSpan *Span
	classes        []string
	actorType      string
	icon           string
	hasIcon        bool
	business       *bool
}

type boundaryState struct {
	draftBoundary
	typ     string
	members []string
}

// stylable is what classes and styles may be given to.
type stylable struct {
	classes *[]string
	styles  *[]string
}

func pushUnique(target *[]string, values []string) {
	if len(values) == 0 {
		return
	}
	seen := make(map[string]bool, len(*target))
	for _, v := range *target {
		seen[v] = true
	}
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		*target = append(*target, v)
	}
}

// inferKind is the kind of element the keys of a metadata block could only belong to, for the
// error message, as Pele's inferKind finds it.
func inferKind(properties []draftProperty) string {
	possible := map[string]bool{"actor": true, "usecase": true, "boundary": true, "edge": true}
	only := func(kind string) {
		possible = map[string]bool{}
		if kind != "" {
			possible[kind] = true
		}
	}
	for _, p := range properties {
		switch p.Key {
		case "icon":
			only("actor")
		case "animate", "animation":
			only("edge")
		case "type":
			switch p.value {
			case "rect", "package":
				only("boundary")
			case "normal", "hollow", "awesome":
				only("actor")
			default:
				only("")
			}
		case "business":
			delete(possible, "boundary")
			delete(possible, "edge")
		default:
			return ""
		}
	}
	if len(possible) == 1 {
		for k := range possible {
			return k
		}
	}
	return ""
}

// resolver is the state of a resolution.
type resolver struct {
	src     string
	model   *Model
	actors  *ordered[*Actor]
	useCase *ordered[*UseCase]
	bounds  *ordered[*SystemBoundary]
	jsons   *ordered[*JSONNode]
	symbols map[string]*origin
	// The offset at which each id is first mentioned, which decides the order of the model.
	first map[string]int
}

func (r *resolver) error(message string, span Span) *diagram.Error {
	line, column := lineColumn(r.src, span[0])
	return &diagram.Error{Code: diagram.Semantic, Message: message, Type: "usecase", Line: line, Column: column}
}

func (r *resolver) at(span Span) string { return locationText(r.src, span[0], span[1]) }

func (r *resolver) fail(message string, span Span) {
	panic(parseFailure{r.error(message+" at "+r.at(span), span)})
}

func (r *resolver) conflict(message string, current, previous Span, currentLabel, previousLabel *string) {
	suffix := func(label *string) string {
		if label == nil {
			return ""
		}
		return " (label \"" + *label + "\")"
	}
	panic(parseFailure{r.error(message+" at "+r.at(current)+suffix(currentLabel)+"; previous declaration at "+
		r.at(previous)+suffix(previousLabel), current)})
}

func (r *resolver) invalid(id, kind string, p draftProperty) {
	r.fail("Metadata property '"+p.Key+"' is invalid for "+kind+" '"+id+"'", p.KeySpan)
}

func (r *resolver) mention(id string, offset int) {
	if previous, ok := r.first[id]; !ok || offset < previous {
		r.first[id] = offset
	}
}

func (r *resolver) firstOf(id string) int { return r.first[id] }

func (r *resolver) register(n named, kind string) {
	r.symbols[n.id] = &origin{kind: kind, span: n.span, generated: n.generated, label: n.label.text}
}

// claim checks that an id means one thing in the whole diagram, and that an id made from a
// label is declared once.
func (r *resolver) claim(d named, kind string) {
	o, ok := r.symbols[d.id]
	if !ok {
		return
	}
	var label, originLabel *string
	if d.generated {
		label = &d.label.text
	}
	if o.generated {
		originLabel = &o.label
	}
	if o.kind != kind {
		r.conflict("ID '"+d.id+"' is declared as both "+o.kind+" and "+kind, d.span, o.span, label, originLabel)
	}
	if o.generated || d.generated {
		r.conflict("Generated ID '"+d.id+"' collides with another declaration", d.span, o.span, label, originLabel)
	}
}

func (r *resolver) unique(id, kind string, span Span) {
	if previous, ok := r.symbols[id]; ok {
		var previousLabel *string
		if previous.generated {
			previousLabel = &previous.label
		}
		r.conflict("ID '"+id+"' is declared more than once ("+previous.kind+" and "+kind+")", span, previous.span, nil, previousLabel)
	}
	r.symbols[id] = &origin{kind: kind, span: span}
}

// actorProperty sets a property of an actor. A metadata statement replaces what a declaration
// said; two declarations have to agree.
func (r *resolver) actorProperty(state *elementState, p draftProperty, replace bool) {
	clash := func(set bool, previous any) {
		if !replace && set && previous != p.value {
			r.conflict("Actor '"+state.id+"' has conflicting "+p.Key+" metadata", p.KeySpan, state.span, nil, nil)
		}
	}
	s, isString := p.value.(string)
	b, isBool := p.value.(bool)
	switch {
	case p.Key == "type" && (s == "normal" || s == "hollow" || s == "awesome"):
		clash(state.actorType != "", state.actorType)
		state.actorType = s
	case p.Key == "icon" && isString:
		clash(state.hasIcon, state.icon)
		state.icon, state.hasIcon = s, true
	case p.Key == "business" && isBool:
		if state.business != nil {
			clash(true, *state.business)
		}
		state.business = &b
	default:
		r.invalid(state.id, "actor", p)
	}
}

func (r *resolver) elementMetadata(state *elementState, properties []draftProperty, replace bool) {
	for _, p := range properties {
		if state.kind == "actor" {
			r.actorProperty(state, p, replace)
			continue
		}
		if b, ok := p.value.(bool); ok && p.Key == "business" {
			if !replace && state.business != nil && *state.business != b {
				r.conflict("Use case '"+state.id+"' has conflicting business metadata", p.KeySpan, state.span, nil, nil)
			}
			state.business = &b
		} else {
			r.invalid(state.id, "usecase", p)
		}
	}
}

func (r *resolver) publish(state *elementState) {
	business := state.business != nil && *state.business
	parentID := ""
	if state.parent != nil {
		parentID = state.parent.id
	}
	if state.kind == "actor" {
		r.useCase.delete(state.id)
		var styles []string
		if previous, ok := r.actors.get(state.id); ok {
			styles = previous.Styles
		}
		a := &Actor{
			ID: state.id, Label: state.label.text, LabelType: state.label.typ, Business: business,
			Classes: slices.Clone(state.classes), Styles: orEmpty(styles), Stereotype: state.stereotype, ParentID: parentID,
		}
		switch {
		case state.icon != "":
			a.Type, a.Icon = "icon", state.icon
		case state.actorType != "":
			a.Type = state.actorType
		default:
			a.Type = "normal"
		}
		r.actors.set(state.id, a)
		return
	}
	r.actors.delete(state.id)
	var styles []string
	if previous, ok := r.useCase.get(state.id); ok {
		styles = previous.Styles
	}
	shape := state.shape
	if shape == "" {
		shape = "ellipse"
	}
	r.useCase.set(state.id, &UseCase{
		ID: state.id, Label: state.label.text, LabelType: state.label.typ, Business: business,
		Classes: slices.Clone(state.classes), Shape: shape, Styles: orEmpty(styles), Stereotype: state.stereotype,
		ParentID: parentID,
	})
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// resolve resolves what the parser read into a model, once the whole diagram is read: which ids
// are actors, use cases, boundaries, JSON nodes or edges, in what order they appear, and whether
// the statements about them agree. Ports Pele's resolveUsecase, which ports Mermaid's
// UsecaseModelBuilder, with its messages.
func resolve(src string, d *drafts, headerSpan Span, statements []*Statement, title string) (model *Model, err error) {
	r := &resolver{
		src: src, model: &Model{Direction: "LR"},
		actors: newOrdered[*Actor](), useCase: newOrdered[*UseCase](), bounds: newOrdered[*SystemBoundary](),
		jsons: newOrdered[*JSONNode](), symbols: map[string]*origin{}, first: map[string]int{},
	}
	defer func() {
		if x := recover(); x != nil {
			f, ok := x.(parseFailure)
			if !ok {
				panic(x)
			}
			model, err = nil, f.err
		}
	}()
	elements := newOrdered[*elementState]()
	boundaries := newOrdered[*boundaryState]()
	edges := map[string]*Relationship{}

	inOrder := func(ids []string) []string {
		slices.SortStableFunc(ids, func(a, b string) int { return r.firstOf(a) - r.firstOf(b) })
		return ids
	}
	keys := func(each func(fn func(string) bool)) []string {
		var out []string
		each(func(k string) bool {
			out = append(out, k)
			return true
		})
		return out
	}
	elementIDs := func() []string {
		return inOrder(keys(func(fn func(string) bool) {
			elements.each(func(k string, _ *elementState) bool { return fn(k) })
		}))
	}

	declareElement := func(de *draftElement) {
		r.claim(de.named, de.kind)
		var parent *draftBoundary
		if de.parent != nil && de.parent.id != "" {
			parent = de.parent
		}
		existing, ok := elements.get(de.id)
		if !ok {
			state := &elementState{
				named: de.named, kind: de.kind, classes: slices.Clone(de.classes), parent: parent,
				shape: de.shape, stereotype: de.stereotype,
			}
			if de.stereotype != "" {
				s := de.span
				state.stereotypeSpan = &s
			}
			r.elementMetadata(state, de.metadata, false)
			elements.set(de.id, state)
			r.register(de.named, de.kind)
			return
		}
		if existing.label.text != de.label.text || existing.label.typ != de.label.typ {
			r.conflict("ID '"+de.id+"' has conflicting labels", de.span, existing.span, nil, nil)
		}
		if de.shape != "" && existing.shape != "" && de.shape != existing.shape {
			r.conflict("Use case '"+de.id+"' has conflicting shapes", de.span, existing.span, nil, nil)
		}
		// A boundary whose title makes an empty id is no parent, but it still takes the place
		// of one.
		if parent != nil && existing.parent != nil && existing.parent.id != "" && parent.id != existing.parent.id {
			r.conflict("Element '"+de.id+"' belongs to more than one system boundary", parent.span, existing.parent.span, nil, nil)
		}
		if de.stereotype != "" && existing.stereotype != "" && de.stereotype != existing.stereotype {
			previous := existing.span
			if existing.stereotypeSpan != nil {
				previous = *existing.stereotypeSpan
			}
			r.conflict("Element '"+de.id+"' has conflicting stereotypes", de.span, previous, nil, nil)
		}
		if existing.shape == "" {
			existing.shape = de.shape
		}
		if existing.parent == nil {
			existing.parent = de.parent
		}
		if existing.stereotype == "" {
			existing.stereotype = de.stereotype
		}
		if existing.stereotypeSpan == nil && de.stereotype != "" {
			s := de.span
			existing.stereotypeSpan = &s
		}
		pushUnique(&existing.classes, de.classes)
		r.elementMetadata(existing, de.metadata, false)
	}

	declareBoundary := func(db *draftBoundary) {
		r.claim(db.named, "boundary")
		if existing, ok := boundaries.get(db.id); ok {
			if existing.label.text != db.label.text || existing.label.typ != db.label.typ {
				r.conflict("Boundary '"+db.id+"' has conflicting titles", db.span, existing.span, nil, nil)
			}
			pushUnique(&existing.classes, db.classes)
			return
		}
		boundaries.set(db.id, &boundaryState{draftBoundary: draftBoundary{named: db.named, classes: slices.Clone(db.classes)}, members: []string{}})
		r.register(db.named, "boundary")
	}

	// An end of a relationship that nothing declares becomes a use case.
	endpoint := func(end draftEndpoint) string {
		if end.classed && !end.declares {
			r.fail("Relationship endpoint '"+end.id+"' uses ::: without declaring the node", end.span)
		}
		if o, ok := r.symbols[end.id]; ok {
			return o.kind
		}
		elements.set(end.id, &elementState{named: end.named, kind: "usecase", shape: "ellipse", classes: []string{}})
		r.register(end.named, "usecase")
		r.mention(end.id, end.span[0])
		return "usecase"
	}

	stylableOf := func(id string, span Span) stylable {
		var kind string
		if o, ok := r.symbols[id]; ok {
			kind = o.kind
		}
		switch kind {
		case "actor":
			if a, ok := r.actors.get(id); ok {
				return stylable{&a.Classes, &a.Styles}
			}
		case "usecase":
			if u, ok := r.useCase.get(id); ok {
				return stylable{&u.Classes, &u.Styles}
			}
		case "boundary":
			if b, ok := r.bounds.get(id); ok {
				return stylable{&b.Classes, &b.Styles}
			}
		case "json":
			if j, ok := r.jsons.get(id); ok {
				return stylable{&j.Classes, &j.Styles}
			}
		default:
			if e, ok := edges[id]; ok {
				return stylable{&e.Classes, &e.Styles}
			}
		}
		r.fail("Class/style target '"+id+"' is unresolved or anonymous", span)
		return stylable{}
	}

	type declaration struct {
		offset int
		run    func()
	}
	var declarations []declaration
	for _, rel := range d.relationships {
		r.mention(rel.source.id, rel.source.span[0])
		r.mention(rel.target.id, rel.target.span[0])
	}
	for _, de := range d.elements {
		r.mention(de.id, de.span[0])
		declarations = append(declarations, declaration{de.span[0], func() { declareElement(de) }})
	}
	for _, db := range d.boundaries {
		r.mention(db.id, db.span[0])
		declarations = append(declarations, declaration{db.span[0], func() { declareBoundary(db) }})
	}
	for _, dj := range d.jsons {
		r.mention(dj.id, dj.span[0])
		declarations = append(declarations, declaration{dj.span[0], func() { r.unique(dj.id, "json", dj.span) }})
	}
	for _, rel := range d.relationships {
		if rel.idSpan != nil {
			declarations = append(declarations, declaration{rel.idSpan[0], func() { r.unique(rel.id, "edge", *rel.idSpan) }})
		}
	}
	slices.SortStableFunc(declarations, func(a, b declaration) int { return a.offset - b.offset })
	for _, dec := range declarations {
		dec.run()
	}

	for _, id := range elementIDs() {
		state, _ := elements.get(id)
		r.publish(state)
	}
	boundaryIDs := inOrder(keys(func(fn func(string) bool) {
		boundaries.each(func(k string, _ *boundaryState) bool { return fn(k) })
	}))
	for _, id := range boundaryIDs {
		state, _ := boundaries.get(id)
		r.bounds.set(id, &SystemBoundary{
			ID: id, Label: state.label.text, LabelType: state.label.typ, Type: "rect",
			Members: []string{}, Classes: []string{}, Styles: []string{},
		})
	}
	jsons := slices.Clone(d.jsons)
	slices.SortStableFunc(jsons, func(a, b *draftJSON) int { return r.firstOf(a.id) - r.firstOf(b.id) })
	for _, dj := range jsons {
		r.jsons.set(dj.id, &JSONNode{
			ID: dj.id, Value: dj.value, PropertyOrder: dj.propertyOrder,
			Classes: slices.Clone(dj.classes), Styles: []string{},
		})
	}

	anonymous := 0
	for _, rel := range d.relationships {
		source, target, typ := rel.source, rel.target, rel.typ
		sourceKind := endpoint(source)
		targetKind := endpoint(target)
		related := func(kind string) bool { return kind == "actor" || kind == "usecase" || kind == "json" }
		if !related(sourceKind) {
			r.conflict("Relationship source '"+source.id+"' cannot be "+sourceKind, source.span, r.symbols[source.id].span, nil, nil)
		}
		if !related(targetKind) {
			r.conflict("Relationship target '"+target.id+"' cannot be "+targetKind, target.span, r.symbols[target.id].span, nil, nil)
		}
		if (typ == "include" || typ == "extend") && (sourceKind != "usecase" || targetKind != "usecase") {
			r.fail(typ+" relationship requires use-case endpoints", rel.span)
		}
		if typ == "generalization" && (sourceKind == "json" || sourceKind != targetKind) {
			r.fail("Generalization requires actor-to-actor or use-case-to-use-case endpoints", rel.span)
		}
		if typ == "association" && (sourceKind == "json" || targetKind == "json") && rel.arrowType > 2 {
			r.fail("JSON relationship '"+source.id+"' to '"+target.id+"' permits only point, reversed-point, or markerless solid association", rel.span)
		}
		id := rel.id
		if rel.idSpan == nil {
			id = "edge-" + itoa(anonymous)
			anonymous++
		}
		relationship := &Relationship{
			ID: id, ExplicitID: rel.idSpan != nil, Source: source.id, Target: target.id, Type: typ,
			ArrowType: rel.arrowType, MinLen: rel.minlen, Classes: []string{}, Styles: []string{},
		}
		if rel.label != nil {
			relationship.Label = rel.label.text
			relationship.LabelType = rel.label.typ
		}
		r.model.Relationships = append(r.model.Relationships, relationship)
		edges[id] = relationship
	}

	// The relationships may have added use cases; put all elements back in the order of first
	// mention.
	declaredActors := r.actors.copy()
	declaredUseCases := r.useCase.copy()
	r.actors.clear()
	r.useCase.clear()
	for _, id := range elementIDs() {
		if a, ok := declaredActors.get(id); ok {
			r.actors.set(id, a)
		} else if u, ok := declaredUseCases.get(id); ok {
			r.useCase.set(id, u)
		} else {
			state, _ := elements.get(id)
			r.publish(state)
		}
	}

	for _, dm := range d.metadata {
		o, ok := r.symbols[dm.target]
		if !ok {
			message := "Metadata target '" + dm.target + "' is unresolved"
			if inferred := inferKind(dm.properties); inferred != "" {
				message += " (metadata implies " + inferred + ")"
			}
			r.fail(message, dm.span)
		}
		kind := o.kind
		if kind == "actor" || kind == "usecase" {
			state, _ := elements.get(dm.target)
			r.elementMetadata(state, dm.properties, true)
			continue
		}
		if kind == "edge" {
			s := dm.statement
			s.Kind = "edgeMetadata"
			idSpan := dm.span
			s.Edges = []EdgeOccurrence{{ID: dm.target, Span: s.Span, IDSpan: &idSpan, Metadata: s.Metadata}}
			s.Nodes = nil
		}
		boundary, _ := boundaries.get(dm.target)
		edge := edges[dm.target]
		for _, p := range dm.properties {
			s, isString := p.value.(string)
			b, isBool := p.value.(bool)
			switch {
			case kind == "boundary" && boundary != nil && p.Key == "type" && (s == "rect" || s == "package"):
				boundary.typ = s
			case kind == "edge" && edge != nil && p.Key == "animate" && isBool:
				edge.Animate = b
			case kind == "edge" && edge != nil && p.Key == "animation" && isString && (s == "fast" || s == "slow"):
				edge.Animation = s
				edge.Animate = true
			default:
				r.invalid(dm.target, kind, p)
			}
		}
	}
	for _, rel := range r.model.Relationships {
		if rel.Animation != "" {
			rel.Animate = true
		}
	}

	elements.each(func(_ string, state *elementState) bool {
		if state.kind == "actor" {
			if state.icon != "" && state.actorType != "" && state.actorType != "normal" {
				r.fail("Actor '"+state.id+"' cannot combine icon with type '"+state.actorType+"'", state.span)
			}
			if state.business != nil && *state.business && (state.icon != "" || state.actorType == "awesome") {
				r.fail("Business actor '"+state.id+"' must use normal or hollow geometry", state.span)
			}
		} else if state.shape == "rect" && state.business != nil && *state.business {
			r.fail("Rectangular use case '"+state.id+"' cannot be a business use case", state.span)
		}
		r.publish(state)
		return true
	})

	sorted := slices.Clone(d.elements)
	slices.SortStableFunc(sorted, func(a, b *draftElement) int { return a.span[0] - b.span[0] })
	for _, de := range sorted {
		if de.parent == nil || de.parent.id == "" {
			continue
		}
		b, _ := boundaries.get(de.parent.id)
		if !slices.Contains(b.members, de.id) {
			b.members = append(b.members, de.id)
		}
	}
	boundaries.each(func(id string, state *boundaryState) bool {
		b, _ := r.bounds.get(id)
		b.Type = state.typ
		if b.Type == "" {
			b.Type = "rect"
		}
		b.Members = state.members
		b.Classes = slices.Clone(state.classes)
		return true
	})

	notes := newOrdered[*Note]()
	for i, dn := range d.notes {
		o, ok := r.symbols[dn.target]
		if !ok {
			r.fail("Note target '"+dn.target+"' is unresolved", dn.span)
		}
		if o.kind != "actor" && o.kind != "usecase" {
			r.conflict("Note target '"+dn.target+"' must be an actor or use case, not "+o.kind, dn.span, o.span, nil, nil)
		}
		id := "note-" + itoa(i)
		notes.set(id, &Note{ID: id, Target: dn.target, Label: dn.label.text, LabelType: dn.label.typ})
	}

	classDefs := newOrdered[*ClassDef]()
	for _, def := range d.classDefs {
		for _, id := range def.ids {
			classDefs.set(id, &ClassDef{ID: id, Styles: slices.Clone(def.styles)})
		}
	}
	for _, assignment := range d.classes {
		for _, target := range assignment.targets {
			pushUnique(stylableOf(target.id, target.span).classes, assignment.classes)
		}
	}
	for _, assignment := range d.styles {
		s := stylableOf(assignment.target, assignment.span)
		*s.styles = append(*s.styles, assignment.styles...)
	}

	m := r.model
	m.Actors = r.actors.values()
	m.UseCases = r.useCase.values()
	m.SystemBoundaries = r.bounds.values()
	m.Notes = notes.values()
	m.JSONNodes = r.jsons.values()
	m.ClassDefs = classDefs.values()
	if d.direction != "" {
		m.Direction = d.direction
	}
	m.Title = title
	m.AccTitle = d.accTitle
	m.AccDescription = d.accDescription
	m.AccDescr = dropIndents(d.accDescription)
	m.Source = src
	m.HeaderSpan = headerSpan
	m.Statements = statements
	return m, nil
}

// dropIndents is s.replace(/\n\s+/g, '\n'): a line feed and the white space after it, line
// breaks among it, become one line feed.
func dropIndents(s string) string {
	if !strings.Contains(s, "\n") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '\n' {
			b.WriteByte(s[i])
			i++
			continue
		}
		j := chars.SpaceRun(s, i+1)
		b.WriteByte('\n')
		i = j
	}
	return b.String()
}

func itoa(n int) string { return strconv.Itoa(n) }
