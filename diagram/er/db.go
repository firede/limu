package er

import (
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
)

// The cardinalities, as Mermaid names them: Pele's Cardinality.
const (
	ZeroOrOne  = "ZERO_OR_ONE"
	ZeroOrMore = "ZERO_OR_MORE"
	OneOrMore  = "ONE_OR_MORE"
	OnlyOne    = "ONLY_ONE"
	MDParent   = "MD_PARENT"
)

// The kinds of relationship: Pele's Identification.
const (
	NonIdentifying = "NON_IDENTIFYING"
	Identifying    = "IDENTIFYING"
)

// Attribute is an attribute of an entity, Pele's ErAttribute: its type, its name, its keys
// (PK, FK, UK) and its comment, the last two empty where none is written.
type Attribute struct {
	Type    string
	Name    string
	Keys    []string
	Comment string
}

// Entity is an entity, Pele's ErEntity.
type Entity struct {
	// ID is "entity-<name>-<n>", n counting the entities before it.
	ID    string
	Label string
	// Alias is the name it is shown by, from `name[alias]`, or "".
	Alias      string
	Attributes []Attribute
	// CSSClasses are its classes, "default" first, separated by spaces.
	CSSClasses string
	CSSStyles  []string
}

// RelSpec is a relationship's ends and kind, Pele's RelSpec. CardB sits at entity A's end of
// the line and CardA at entity B's, as in Mermaid.
type RelSpec struct {
	CardA   string
	RelType string
	CardB   string
}

// Relationship is a relationship, Pele's ErRelationship. EntityA and EntityB hold an entity's
// id, or a subgraph's id when the name was a subgraph at the time.
type Relationship struct {
	EntityA string
	RoleA   string
	EntityB string
	RelSpec RelSpec
}

// ClassDef is a class of styles, from classDef: Pele's ErClassDef.
type ClassDef struct {
	ID         string
	Styles     []string
	TextStyles []string
}

// Subgraph is a subgraph, Pele's ErSubgraph. Dir is "" where it sets no direction.
type Subgraph struct {
	ID        string
	Nodes     []string
	Title     string
	Classes   []string
	CSSStyles []string
	Dir       string
	LabelType string
}

// docItem is what a statement inside a subgraph hands to AddSubGraph: a name, or, when dir
// is set, a direction. Pele's DocItem.
type docItem struct {
	name string
	dir  string
}

// DB is what an ER diagram's text says, as Pele's ErDb holds it: its entities in the order
// they were first named, its relationships, classes and subgraphs.
type DB struct {
	Direction     string
	Entities      []*Entity
	Relationships []Relationship
	// Classes are the classes in the order they were first defined; Class finds one.
	Classes   []*ClassDef
	Subgraphs []*Subgraph
	Title     string
	AccTitle  string
	AccDescr  string

	entities       map[string]*Entity
	classes        map[string]*ClassDef
	subgraphLookup map[string]*Subgraph
	claimed        map[string]bool
}

// NewDB is an empty ER diagram.
func NewDB() *DB {
	return &DB{
		Direction:      "TB",
		entities:       map[string]*Entity{},
		classes:        map[string]*ClassDef{},
		subgraphLookup: map[string]*Subgraph{},
		claimed:        map[string]bool{},
	}
}

// Entity is the entity of a name, or nil.
func (d *DB) Entity(name string) *Entity { return d.entities[name] }

// Class is the class of id, or nil.
func (d *DB) Class(id string) *ClassDef { return d.classes[id] }

// Subgraph is the subgraph of id, or nil.
func (d *DB) Subgraph(id string) *Subgraph { return d.subgraphLookup[id] }

// addEntity is the entity of a name, added if it is new, as Pele's addEntity does; an alias
// is taken if the entity has none yet.
func (d *DB) addEntity(name, alias string) *Entity {
	e := d.entities[name]
	if e == nil {
		e = &Entity{
			ID:         "entity-" + name + "-" + strconv.Itoa(len(d.Entities)),
			Label:      name,
			Alias:      alias,
			CSSClasses: "default",
			CSSStyles:  []string{},
		}
		d.entities[name] = e
		d.Entities = append(d.Entities, e)
	} else if e.Alias == "" && alias != "" {
		e.Alias = alias
	}
	return e
}

// addAttributes adds attributes to an entity, as Pele's addAttributes does. The grammar
// builds the list back to front, so the last attribute written comes first.
func (d *DB) addAttributes(name string, attributes []Attribute) {
	e := d.addEntity(name, "")
	for i := len(attributes) - 1; i >= 0; i-- {
		a := attributes[i]
		if a.Keys == nil {
			a.Keys = []string{}
		}
		e.Attributes = append(e.Attributes, a)
	}
}

// addRelationship adds a relationship, as Pele's addRelationship does: a name that is a
// subgraph's id stands for the subgraph.
func (d *DB) addRelationship(entA, roleA, entB string, spec RelSpec) {
	a, b := entA, entB
	if d.subgraphLookup[entA] == nil {
		a = d.addEntity(entA, "").ID
	}
	if d.subgraphLookup[entB] == nil {
		b = d.addEntity(entB, "").ID
	}
	d.Relationships = append(d.Relationships, Relationship{EntityA: a, RoleA: roleA, EntityB: b, RelSpec: spec})
}

// setDirection sets the direction the diagram runs in.
func (d *DB) setDirection(dir string) { d.Direction = dir }

// addCSSStyles adds styles to entities and subgraphs, as Pele's addCssStyles does.
func (d *DB) addCSSStyles(ids, styles []string) {
	if styles == nil {
		return
	}
	for _, id := range ids {
		if e := d.entities[id]; e != nil {
			e.CSSStyles = append(e.CSSStyles, styles...)
		}
		if s := d.subgraphLookup[id]; s != nil {
			s.CSSStyles = append(s.CSSStyles, styles...)
		}
	}
}

// addClass defines classes, or adds styles to them, as Pele's addClass does.
func (d *DB) addClass(ids, styles []string) {
	for _, id := range ids {
		def := d.classes[id]
		if def == nil {
			def = &ClassDef{ID: id, Styles: []string{}, TextStyles: []string{}}
			d.classes[id] = def
			d.Classes = append(d.Classes, def)
		}
		for _, style := range styles {
			if strings.Contains(style, "color") {
				def.TextStyles = append(def.TextStyles, strings.Replace(style, "fill", "bgFill", 1))
			}
			def.Styles = append(def.Styles, style)
		}
	}
}

// setClass gives entities and subgraphs classes, as Pele's setClass does.
func (d *DB) setClass(ids, classNames []string) {
	for _, id := range ids {
		if e := d.entities[id]; e != nil {
			for _, name := range classNames {
				e.CSSClasses += " " + name
			}
		}
		if s := d.subgraphLookup[id]; s != nil {
			s.Classes = append(s.Classes, classNames...)
		}
	}
}

// addSubGraph adds a subgraph of what the statements inside it list, as Pele's addSubGraph
// does, and gives its id. A node belongs to the first subgraph that lists it.
func (d *DB) addSubGraph(idText string, list []docItem, title string) string {
	id := chars.Trim(idText)
	seen := map[string]bool{}
	nodes := []string{}
	dir := ""
	for _, item := range list {
		if item.dir != "" {
			dir = item.dir
			continue
		}
		trimmed := chars.Trim(item.name)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		if !d.claimed[item.name] {
			nodes = append(nodes, item.name)
		}
	}
	for _, node := range nodes {
		d.claimed[node] = true
	}
	s := &Subgraph{
		ID:        id,
		Nodes:     nodes,
		Title:     chars.Trim(title),
		Classes:   []string{},
		CSSStyles: []string{},
		Dir:       dir,
		LabelType: "markdown",
	}
	d.Subgraphs = append(d.Subgraphs, s)
	d.subgraphLookup[id] = s
	return id
}

// setAccTitle sets the title for assistive technology, its leading white space taken off.
func (d *DB) setAccTitle(text string) { d.AccTitle = chars.TrimStart(text) }

// setAccDescription sets the description for assistive technology, as Pele's
// setAccDescription does: the white space that starts each line after the first is taken
// off, /\n\s+/g replaced by a line feed.
func (d *DB) setAccDescription(text string) {
	d.AccDescr = chars.DropIndents(text)
}
