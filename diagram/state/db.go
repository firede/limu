package state

import (
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
)

// Stmt is a statement of a document, Pele's Stmt: a *StateStmt, *RelationStmt, *ClassDefStmt,
// *StyleStmt, *ApplyClassStmt, *DirectionStmt, *ClickStmt, or a TextStmt.
type Stmt interface{ isStmt() }

// TextStmt is the text Mermaid's grammar leaves of a statement that has no effect in a
// document: `hide empty description`, `scale`, a bare `state name`, a floating note, and the
// values of accTitle and accDescr.
type TextStmt string

// Note is a note beside a state, Pele's StateNote.
type Note struct {
	Position string // "left of" or "right of"
	Text     string
}

// StateStmt is a state, as a statement gives it and as the DB keeps it: Pele's StateStmt. A
// string or list Pele may leave undefined is nil here where it is.
type StateStmt struct {
	ID string
	// Type is "default", "fork", "join", "choice" or "divider"; "" where the statement gives
	// none, as a note's does.
	Type string
	// Description is the statement's description: one string, or, where DescriptionList is
	// set, a list. Nil where it gives none.
	Description     []string
	DescriptionList bool
	// Descriptions are the descriptions the DB has gathered for a state; nil in a statement.
	Descriptions []string
	// Doc is the document of a composite state: nil where the state is none, and empty, not
	// nil, for a composite with nothing in it.
	Doc  []Stmt
	Note *Note
	// Start is set on a `[*]` once it is named: true where it starts, false where it ends.
	Start      *bool
	Classes    []string
	Styles     []string
	TextStyles []string
}

// RelationStmt is a transition, Pele's RelationStmt. HasDescription tells a description that
// is empty from none.
type RelationStmt struct {
	State1, State2 *StateStmt
	Description    string
	HasDescription bool
}

// ClassDefStmt is a classDef statement.
type ClassDefStmt struct{ ID, Classes string }

// StyleStmt is a style statement.
type StyleStmt struct{ ID, StyleClass string }

// ApplyClassStmt is a class statement.
type ApplyClassStmt struct{ ID, StyleClass string }

// DirectionStmt is a direction statement.
type DirectionStmt struct{ Value string }

// ClickStmt is a click statement: a link from a state, with a tooltip.
type ClickStmt struct {
	ID           *StateStmt
	URL, Tooltip string
}

func (TextStmt) isStmt()        {}
func (*StateStmt) isStmt()      {}
func (*RelationStmt) isStmt()   {}
func (*ClassDefStmt) isStmt()   {}
func (*StyleStmt) isStmt()      {}
func (*ApplyClassStmt) isStmt() {}
func (*DirectionStmt) isStmt()  {}
func (*ClickStmt) isStmt()      {}

// Relation is a transition as the DB keeps it, Pele's StateRelation.
type Relation struct {
	ID1, ID2      string
	RelationTitle string
}

// ClassDef is a class of styles, from classDef: Pele's StateClassDef.
type ClassDef struct {
	ID         string
	Styles     []string
	TextStyles []string
}

// Link is where a click statement makes a state lead, Pele's StateLink. Its URL and tooltip
// keep the quotes they were written with.
type Link struct {
	ID           string
	URL, Tooltip string
}

// DB is what a state diagram's text says, as Pele's StateDb holds it: Mermaid's StateDB in its
// version 2 form, which Mermaid uses for both `stateDiagram` and `stateDiagram-v2`.
type DB struct {
	RootDoc []Stmt
	// States are the states of the root document, in the order they were first seen; State
	// finds one by its id.
	States    []*StateStmt
	Relations []Relation
	// Classes are the classes in the order they were first defined.
	Classes []*ClassDef
	// Links are the links in the order their states were first given one.
	Links    []Link
	Graph    *Graph
	Title    string
	AccTitle string
	AccDescr string

	states       map[string]*StateStmt
	classes      map[string]*ClassDef
	links        map[string]int
	dividerCount int
}

// NewDB is an empty DB.
func NewDB() *DB {
	return &DB{Graph: &Graph{}, states: map[string]*StateStmt{}, classes: map[string]*ClassDef{}, links: map[string]int{}}
}

// State is the state of the id, or nil.
func (d *DB) State(id string) *StateStmt { return d.states[id] }

// Class is the class of the id, or nil.
func (d *DB) Class(id string) *ClassDef { return d.classes[id] }

// Link is the link of the state of the id, and whether it has one.
func (d *DB) Link(id string) (Link, bool) {
	i, ok := d.links[id]
	if !ok {
		return Link{}, false
	}
	return d.Links[i], true
}

// Direction is the direction the root document is laid out in: its last `direction`
// statement, or TB, as Pele's direction getter gives it.
func (d *DB) Direction() string {
	dir := "TB"
	for _, item := range d.RootDoc {
		if s, ok := item.(*DirectionStmt); ok {
			dir = s.Value
		}
	}
	return dir
}

// setRootDoc takes the parsed document, as Pele's setRootDoc does.
func (d *DB) setRootDoc(doc []Stmt) error {
	d.RootDoc = doc
	d.translate()
	return d.extract(doc)
}

// extract reads the root document's statements into states, relations and classes, as Pele's
// extract does; the statements inside composites are read when the graph is built.
func (d *DB) extract(doc []Stmt) error {
	d.States, d.states = nil, map[string]*StateStmt{}
	d.Relations = nil
	d.Classes, d.classes = nil, map[string]*ClassDef{}
	for _, item := range doc {
		switch s := item.(type) {
		case *StateStmt:
			d.addState(chars.Trim(s.ID), s.Type, s.Doc, s.Description, s.DescriptionList, s.Note, nil)
		case *RelationStmt:
			d.addRelation(s.State1, s.State2, s.Description)
		case *ClassDefStmt:
			d.addStyleClass(chars.Trim(s.ID), s.Classes)
		case *StyleStmt:
			parts := strings.Split(s.StyleClass, ",")
			styles := make([]string, len(parts))
			for i, part := range parts {
				styles[i] = chars.Trim(strings.ReplaceAll(part, ";", ""))
			}
			for _, state := range d.named(chars.Trim(s.ID)) {
				state.Styles = append([]string{}, styles...)
			}
		case *ApplyClassStmt:
			d.setCSSClass(chars.Trim(s.ID), s.StyleClass)
		case *ClickStmt:
			d.addLink(s.ID.ID, s.URL, s.Tooltip)
		}
	}
	g, err := buildGraph(doc, d.states, d.classes)
	if err != nil {
		return err
	}
	d.Graph = g
	return nil
}

// translateItem is an item of translate's work: a statement, the id of its parent, and whether
// a `[*]` there starts its document.
type translateItem struct {
	parentID string
	node     *StateStmt
	relation *RelationStmt
	first    bool
}

// translate is Mermaid's docTranslator, as Pele's translate does it: it gives each `[*]` an id
// scoped to its parent, and turns a document split by `--` into one region per part.
func (d *DB) translate() {
	root := &StateStmt{ID: "root", Doc: d.RootDoc}
	pending := []translateItem{{"root", root, nil, true}}
	for len(pending) > 0 {
		it := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if it.relation != nil {
			pending = append(pending, translateItem{it.parentID, it.relation.State2, nil, false},
				translateItem{it.parentID, it.relation.State1, nil, true})
			continue
		}
		node := it.node
		if node != root {
			if node.ID == "[*]" {
				suffix := "_end"
				if it.first {
					suffix = "_start"
				}
				node.ID = it.parentID + suffix
				first := it.first
				node.Start = &first
			} else {
				node.ID = chars.Trim(node.ID)
			}
		}
		if node.Doc == nil {
			continue
		}

		var regions []Stmt
		current := []Stmt{}
		for _, item := range node.Doc {
			if s, ok := item.(*StateStmt); ok && s.Type == "divider" && s.Doc == nil {
				region := *s
				region.Doc = current
				regions = append(regions, &region)
				current = []Stmt{}
			} else {
				current = append(current, item)
			}
		}
		// Mermaid names the region after the last divider with a random id; here it continues
		// the divider numbering. Mermaid also leaves a document that ends with a divider
		// ungrouped and then fails to draw it; here the regions before that divider are kept.
		if len(regions) > 0 {
			if len(current) > 0 {
				regions = append(regions, &StateStmt{ID: d.dividerID(), Type: "divider", Doc: current})
			}
			node.Doc = regions
		}
		for i := len(node.Doc) - 1; i >= 0; i-- {
			switch s := node.Doc[i].(type) {
			case *StateStmt:
				pending = append(pending, translateItem{node.ID, s, nil, true})
			case *RelationStmt:
				pending = append(pending, translateItem{node.ID, nil, s, true})
			}
		}
	}
}

// addState adds a state, or adds to the one of the same id, as Pele's addState does. An empty
// typ is the default.
func (d *DB) addState(id, typ string, doc []Stmt, descr []string, descrList bool, note *Note, classes []string) {
	if typ == "" {
		typ = "default"
	}
	trimmed := chars.Trim(id)
	state := d.states[trimmed]
	if state == nil {
		state = &StateStmt{ID: trimmed, Descriptions: []string{}, Type: typ, Doc: doc, Note: note,
			Classes: []string{}, Styles: []string{}, TextStyles: []string{}}
		d.states[trimmed] = state
		d.States = append(d.States, state)
	} else {
		if state.Doc == nil {
			state.Doc = doc
		}
		if state.Type == "" {
			state.Type = typ
		}
	}
	// A description is there unless it is none or an empty string.
	if descr != nil && (descrList || descr[0] != "") {
		for _, text := range descr {
			d.addDescription(trimmed, chars.Trim(text))
		}
	}
	if note != nil {
		state.Note = note
	}
	for _, name := range classes {
		d.setCSSClass(trimmed, chars.Trim(name))
	}
}

// addLink makes a state lead to a URL, as Pele's addLink does.
func (d *DB) addLink(stateID, url, tooltip string) {
	if i, ok := d.links[stateID]; ok {
		d.Links[i] = Link{stateID, url, tooltip}
		return
	}
	d.links[stateID] = len(d.Links)
	d.Links = append(d.Links, Link{stateID, url, tooltip})
}

// addRelation adds a transition and the states at its ends, as Pele's addRelation does.
func (d *DB) addRelation(item1, item2 *StateStmt, title string) {
	for _, item := range [2]*StateStmt{item1, item2} {
		d.addState(item.ID, item.Type, item.Doc, item.Description, item.DescriptionList, item.Note, item.Classes)
	}
	d.Relations = append(d.Relations, Relation{chars.Trim(item1.ID), chars.Trim(item2.ID), title})
}

// addDescription adds a description to a state, as Pele's addDescription does.
func (d *DB) addDescription(id, descr string) {
	state := d.states[id]
	if state == nil {
		return
	}
	if strings.HasPrefix(descr, ":") {
		descr = chars.Trim(strings.Replace(descr, ":", "", 1))
	}
	state.Descriptions = append(state.Descriptions, descr)
}

// dividerID is the id of the next region, as Pele's getDividerId gives it.
func (d *DB) dividerID() string {
	d.dividerCount++
	return "divider-id-" + strconv.Itoa(d.dividerCount)
}

// addStyleClass adds a class, or adds to the one of the same id, as Pele's addStyleClass does.
func (d *DB) addStyleClass(id, styleAttributes string) {
	found := d.classes[id]
	if found == nil {
		found = &ClassDef{ID: id, Styles: []string{}, TextStyles: []string{}}
		d.classes[id] = found
		d.Classes = append(d.Classes, found)
	}
	if styleAttributes == "" {
		return
	}
	for _, attrib := range strings.Split(styleAttributes, ",") {
		fixed := chars.Trim(strings.Replace(attrib, ";", "", 1))
		if strings.Contains(attrib, "color") {
			found.TextStyles = append(found.TextStyles,
				strings.Replace(strings.Replace(fixed, "fill", "bgFill", 1), "color", "fill", 1))
		}
		found.Styles = append(found.Styles, fixed)
	}
}

// setCSSClass gives the states of a comma-separated list of ids a class, as Pele's
// setCssClass does.
func (d *DB) setCSSClass(itemIDs, cssClassName string) {
	for _, state := range d.named(itemIDs) {
		state.Classes = append(state.Classes, cssClassName)
	}
}

// setDirection is what the grammar calls on a direction statement. It runs while the document
// is still being built, so the statement lands in the previous root document, and the
// direction is read from the parsed `dir` statements instead, as in Pele.
func (d *DB) setDirection(dir string) {
	for _, item := range d.RootDoc {
		if s, ok := item.(*DirectionStmt); ok {
			s.Value = dir
			return
		}
	}
	d.RootDoc = append([]Stmt{&DirectionStmt{dir}}, d.RootDoc...)
}

// trimColon is a description without the colon it starts with, as Pele's trimColon gives it.
func trimColon(text string) string {
	if strings.HasPrefix(text, ":") {
		return chars.Trim(text[1:])
	}
	return chars.Trim(text)
}

// setAccTitle sets the accessible title, without the white space it starts with.
func (d *DB) setAccTitle(text string) { d.AccTitle = chars.TrimStart(text) }

// setAccDescription sets the accessible description, without the white space each of its
// lines but the first starts with: text.replace(/\n\s+/g, '\n').
func (d *DB) setAccDescription(text string) {
	var b strings.Builder
	for i := 0; i < len(text); {
		if text[i] == '\n' {
			b.WriteByte('\n')
			i = chars.SpaceRun(text, i+1)
			continue
		}
		b.WriteByte(text[i])
		i++
	}
	d.AccDescr = b.String()
}

// named is the states a comma-separated list of ids names, created where they do not exist
// yet, as Pele's named gives them. As in Mermaid, an id is looked up as written and trimmed
// only when it has to be created.
func (d *DB) named(ids string) []*StateStmt {
	var out []*StateStmt
	for _, id := range strings.Split(ids, ",") {
		state := d.states[id]
		if state == nil {
			d.addState(id, "", nil, nil, false, nil, nil)
			state = d.states[chars.Trim(id)]
		}
		if state != nil {
			out = append(out, state)
		}
	}
	return out
}
