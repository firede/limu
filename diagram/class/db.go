package class

import (
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/text"
)

// An End is the kind of marker at an end of a relation, by Mermaid's relationType numbers,
// or NoEnd.
type End int

// NoEnd is the end of a relation with no marker.
const NoEnd End = -1

// The ends of a relation with a marker, as Pele's RELATION_TYPE numbers them.
const (
	Aggregation End = iota
	Extension
	Composition
	Dependency
	Lollipop
)

// The lines of a relation, as Pele's LINE_TYPE numbers them.
const (
	LineSolid  = 0
	LineDotted = 1
)

// A Class is a class of a diagram, as Pele's ClassNode is. A string Pele may leave undefined
// is empty here where it is: no id, name or text the parser reads is empty.
type Class struct {
	ID          string
	Type        string // the generic type its name carries, as `T` of `List~T~`
	Label       string
	CSSClasses  string
	Methods     []*Member
	Members     []*Member
	Annotations []string
	Styles      []string
	Parent      string // the namespace it is declared in
	Link        string
	LinkTarget  string
	// HaveCallback is set by a click with a callback, which is never run; the class is only
	// marked clickable, as in Mermaid's strict mode.
	HaveCallback bool
	Tooltip      string
}

// A Relation is a line between two classes, as Pele's ClassRelation is.
type Relation struct {
	ID1, ID2                       string
	RelationTitle1, RelationTitle2 string // "none" where there is none
	Title                          string
	HasTitle                       bool
	Type1, Type2                   End
	LineType                       int
}

// A Note is a note, on its own or for a class, as Pele's ClassNote is.
type Note struct {
	ID     string
	Class  string // the class it is for, or ""
	Text   string
	Index  int
	Parent string // the namespace it is declared in
}

// An Interface is the interface at the lollipop end of a relation, as Pele's ClassInterface is.
type Interface struct {
	ID, Label, ClassID string
}

// A Namespace is a namespace, as Pele's NamespaceNode is. Its classes, notes and children are
// in the order they were added.
type Namespace struct {
	ID       string
	Label    string
	Classes  []*Class
	Notes    []*Note
	Children []*Namespace
	Parent   string
	// Explicit is false for the ancestors created on the way to a dotted name such as `A.B.C`.
	Explicit bool

	has map[any]bool // the classes, notes and the ids of children it has
}

// A StyleClass is a class of styles, from classDef, as Pele's StyleClass is.
type StyleClass struct {
	ID         string
	Styles     []string
	TextStyles []string
}

// DB is a class diagram parsed, as Pele's ClassDb is. Its maps are slices in the order of
// their keys, as JavaScript's Maps iterate.
type DB struct {
	Direction    string
	Classes      []*Class
	Relations    []*Relation
	Notes        []*Note
	Namespaces   []*Namespace
	Interfaces   []*Interface
	StyleClasses []*StyleClass
	Title        string
	AccTitle     string
	AccDescr     string

	classes        map[string]*Class
	notes          map[string]*Note
	namespaces     map[string]*Namespace
	styleClasses   map[string]*StyleClass
	namespaceStack []string
	withCSSClass   map[string]*classSet
}

// classSet is a set of classes in the order they were added.
type classSet struct {
	list []*Class
	has  map[*Class]bool
}

// NewDB is an empty diagram.
func NewDB() *DB {
	return &DB{
		Direction:    "TB",
		classes:      map[string]*Class{},
		notes:        map[string]*Note{},
		namespaces:   map[string]*Namespace{},
		styleClasses: map[string]*StyleClass{},
		withCSSClass: map[string]*classSet{},
	}
}

// Class is the class of an id, or nil.
func (db *DB) Class(id string) *Class { return db.classes[id] }

// Namespace is the namespace of an id, or nil.
func (db *DB) Namespace(id string) *Namespace { return db.namespaces[id] }

// StyleClass is the classDef of a name, or nil.
func (db *DB) StyleClass(id string) *StyleClass { return db.styleClasses[id] }

// domIDPrefix is the prefix Mermaid looks ids that start with a digit up under, so they never
// match a class.
const domIDPrefix = "classId-"

// splitName splits a name such as `List~T~` into the class's name and its generic type.
func splitName(id string) (className, typ string) {
	at := strings.IndexByte(id, '~')
	if at <= 0 {
		return id, ""
	}
	end := strings.IndexByte(id[at+1:], '~')
	if end == -1 {
		return id[:at], id[at+1:]
	}
	return id[:at], id[at+1 : at+1+end]
}

func className(id string) string {
	name, _ := splitName(id)
	return name
}

func lookupID(id string) string {
	if id != "" && chars.IsDigit(int(id[0])) {
		return className(domIDPrefix + id)
	}
	return className(id)
}

// AddClass adds a class of a name that may carry a generic type, unless there is one.
func (db *DB) AddClass(id string) {
	name, typ := splitName(id)
	if db.classes[name] != nil {
		return
	}
	node := &Class{ID: name, Type: typ, Label: name, CSSClasses: "default"}
	db.classes[name] = node
	db.Classes = append(db.Classes, node)
	db.tag(node, "default")
}

func (db *DB) tag(node *Class, cssClass string) {
	set := db.withCSSClass[cssClass]
	if set == nil {
		set = &classSet{has: map[*Class]bool{}}
		db.withCSSClass[cssClass] = set
	}
	if !set.has[node] {
		set.has[node] = true
		set.list = append(set.list, node)
	}
}

// ensure is the class for a name that may carry a generic type, created if needed. Mermaid
// throws a TypeError where this creates, as for an annotation on a class not yet declared.
func (db *DB) ensure(id string) *Class {
	db.AddClass(id)
	return db.classes[className(id)]
}

// SetClassLabel sets the label a class is shown with.
func (db *DB) SetClassLabel(id, label string) { db.ensure(id).Label = label }

// AddRelation adds a relation, and the classes at its ends. A lollipop end with no marker at
// the other is an interface of its own.
func (db *DB) AddRelation(r *Relation) {
	switch {
	case r.Type1 == Lollipop && r.Type2 == NoEnd:
		db.AddClass(r.ID2)
		r.ID1 = db.addInterface(r.ID1, r.ID2)
	case r.Type2 == Lollipop && r.Type1 == NoEnd:
		db.AddClass(r.ID1)
		r.ID2 = db.addInterface(r.ID2, r.ID1)
	default:
		db.AddClass(r.ID1)
		db.AddClass(r.ID2)
	}
	r.ID1 = className(r.ID1)
	r.ID2 = className(r.ID2)
	r.RelationTitle1 = chars.Trim(r.RelationTitle1)
	r.RelationTitle2 = chars.Trim(r.RelationTitle2)
	db.Relations = append(db.Relations, r)
}

func (db *DB) addInterface(label, classID string) string {
	id := "interface" + itoa(len(db.Interfaces))
	db.Interfaces = append(db.Interfaces, &Interface{ID: id, Label: label, ClassID: classID})
	return id
}

// AddAnnotation adds an annotation, as `interface` of `<<interface>>`, to a class.
func (db *DB) AddAnnotation(class, annotation string) {
	node := db.ensure(class)
	node.Annotations = append(node.Annotations, annotation)
}

// AddMember adds a member to a class: an annotation, a method or an attribute, as Pele's
// addMember reads it.
func (db *DB) AddMember(class, member string) {
	node := db.ensure(class)
	t := chars.Trim(member)
	switch {
	case strings.HasPrefix(t, "<<") && strings.HasSuffix(t, ">>"):
		node.Annotations = append(node.Annotations, t[2:len(t)-2])
	case strings.IndexByte(t, ')') > 0:
		m := NewMember(t, "method")
		if m.MemberType == "method" {
			node.Methods = append(node.Methods, m)
		} else {
			node.Members = append(node.Members, m)
		}
	case t != "":
		node.Members = append(node.Members, NewMember(t, "attribute"))
	}
}

// AddMembers adds the members of a class's body, in order.
func (db *DB) AddMembers(class string, members []string) {
	for _, m := range members {
		db.AddMember(class, m)
	}
}

// AddNote adds a note, for a class or "" for none, and gives its id.
func (db *DB) AddNote(text, class string) string {
	index := len(db.Notes)
	id := "note" + itoa(index)
	note := &Note{ID: id, Class: class, Text: text, Index: index}
	db.notes[id] = note
	db.Notes = append(db.Notes, note)
	return id
}

// CleanupLabel is a label after a colon, as Pele's cleanupLabel takes it.
func CleanupLabel(label string) string {
	return chars.Trim(strings.TrimPrefix(label, ":"))
}

// SetCSSClass adds a class name to each class of a list of ids.
func (db *DB) SetCSSClass(ids, cssClass string) {
	for _, id := range strings.Split(ids, ",") {
		node := db.classes[lookupID(id)]
		if node == nil {
			continue
		}
		node.CSSClasses += " " + cssClass
		for _, word := range strings.Split(cssClass, " ") {
			db.tag(node, word)
		}
	}
}

// DefineClass defines a class of styles for each of ids, and styles the classes that already
// have its name. Mermaid tests `cssClasses.includes(id)`, which also matches part of a longer
// name (a classDef named `def` styles every class, through `default`); Pele compares whole
// names, and so does this.
func (db *DB) DefineClass(ids, style []string) {
	for _, id := range ids {
		sc := db.styleClasses[id]
		if sc == nil {
			sc = &StyleClass{ID: id, Styles: []string{}, TextStyles: []string{}}
			db.styleClasses[id] = sc
			db.StyleClasses = append(db.StyleClasses, sc)
		}
		for _, s := range style {
			if strings.Contains(s, "color") {
				sc.TextStyles = append(sc.TextStyles, strings.Replace(s, "fill", "bgFill", 1))
			}
			sc.Styles = append(sc.Styles, s)
		}
		tagged := db.withCSSClass[id]
		if tagged == nil {
			continue
		}
		var parts []string
		for _, s := range style {
			parts = append(parts, strings.Split(s, ",")...)
		}
		for _, node := range tagged.list {
			node.Styles = append(node.Styles, parts...)
		}
	}
}

// SetTooltip sets the tooltip of each class of a list of ids.
func (db *DB) SetTooltip(ids, tooltip string) {
	for _, id := range strings.Split(ids, ",") {
		if node := db.classes[className(id)]; node != nil {
			node.Tooltip = tooltip
		}
	}
}

// SetLink makes each class of a list of ids a link. Entity codes are resolved first, so that
// what is checked is what a browser would follow. A target not given is _blank.
func (db *DB) SetLink(ids, link, target string, hasTarget bool) {
	u := chars.Trim(text.DecodeEntities(link))
	for _, id := range strings.Split(ids, ",") {
		node := db.classes[lookupID(id)]
		if node == nil {
			continue
		}
		node.Link = ""
		if u != "" {
			node.Link = url.SanitizeURL(u)
		}
		node.LinkTarget = "_blank"
		if hasTarget {
			node.LinkTarget = target
		}
	}
	db.SetCSSClass(ids, "clickable")
}

// SetClickEvent marks each class of a list of ids as having a callback. Callbacks are never
// run; the class is only marked clickable, as in Mermaid's strict mode.
func (db *DB) SetClickEvent(ids string) {
	for _, id := range strings.Split(ids, ",") {
		if node := db.classes[className(id)]; node != nil {
			node.HaveCallback = true
		}
	}
	db.SetCSSClass(ids, "clickable")
}

// SetAccTitle sets the title for assistive technology.
func (db *DB) SetAccTitle(s string) { db.AccTitle = chars.TrimStart(s) }

// SetAccDescription sets the description for assistive technology, with the white space at
// the start of each of its lines taken out, as /\n\s+/g replaced with a line feed does.
func (db *DB) SetAccDescription(s string) {
	db.AccDescr = chars.DropIndents(s)
}

// AddNamespace opens a namespace in the one open, and its ancestors where its name is
// dotted, as Pele's addNamespace does, and gives its full name. A label of "" is none.
func (db *DB) AddNamespace(id, label string) string {
	prefix := ""
	if k := len(db.namespaceStack); k > 0 {
		prefix = db.namespaceStack[k-1]
	}
	qualified := id
	if prefix != "" {
		qualified = prefix + "." + id
	}
	db.namespaceStack = append(db.namespaceStack, qualified)

	if existing := db.namespaces[qualified]; existing != nil {
		existing.Explicit = true
		if label != "" {
			existing.Label = label
		}
		return qualified
	}

	// The enclosing namespace and its ancestors exist already, so only the parts of id are
	// walked.
	parentID := prefix
	from := 0
	if prefix != "" {
		from = len(prefix) + 1
	}
	for {
		dot := strings.IndexByte(qualified[from:], '.')
		leaf := dot == -1
		current, part := qualified, qualified[from:]
		if !leaf {
			dot += from
			current, part = qualified[:dot], qualified[from:dot]
		}
		node := db.namespaces[current]
		if node == nil {
			l := part
			if leaf && label != "" {
				l = label
			}
			node = &Namespace{ID: current, Label: l, Parent: parentID, Explicit: leaf, has: map[any]bool{}}
			db.namespaces[current] = node
			db.Namespaces = append(db.Namespaces, node)
		} else if leaf {
			node.Explicit = true
		}
		if parent := db.namespaces[parentID]; parentID != "" && parent != nil {
			if !parent.has[current] {
				parent.has[current] = true
				parent.Children = append(parent.Children, node)
			}
			if node.Parent == "" {
				node.Parent = parentID
			}
		}
		if leaf {
			return qualified
		}
		parentID = current
		from = dot + 1
	}
}

// PopNamespace closes the namespace open.
func (db *DB) PopNamespace() {
	if k := len(db.namespaceStack); k > 0 {
		db.namespaceStack = db.namespaceStack[:k-1]
	}
}

// AddClassesToNamespace puts classes and notes, by name, in a namespace.
func (db *DB) AddClassesToNamespace(id string, classNames, noteNames []string) {
	ns := db.namespaces[id]
	if ns == nil {
		return
	}
	for _, name := range classNames {
		node := db.classes[className(name)]
		if node == nil {
			continue
		}
		node.Parent = id
		if !ns.has[node] {
			ns.has[node] = true
			ns.Classes = append(ns.Classes, node)
		}
	}
	for _, name := range noteNames {
		note := db.notes[name]
		if note == nil {
			continue
		}
		note.Parent = id
		if !ns.has[note] {
			ns.has[note] = true
			ns.Notes = append(ns.Notes, note)
		}
	}
}

// SetCSSStyle adds style declarations to a class, by its id as written.
func (db *DB) SetCSSStyle(id string, styles []string) {
	node := db.classes[id]
	if node == nil {
		return
	}
	for _, s := range styles {
		node.Styles = append(node.Styles, strings.Split(s, ",")...)
	}
}

func itoa(i int) string { return strconv.Itoa(i) }
