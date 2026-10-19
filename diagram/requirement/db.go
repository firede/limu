package requirement

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The kinds of requirement, as Pele's RequirementType names them.
const (
	KindRequirement            = "Requirement"
	KindFunctionalRequirement  = "Functional Requirement"
	KindInterfaceRequirement   = "Interface Requirement"
	KindPerformanceRequirement = "Performance Requirement"
	KindPhysicalRequirement    = "Physical Requirement"
	KindDesignConstraint       = "Design Constraint"
)

// The levels of risk, as Pele's RiskLevel names them.
const (
	RiskLow    = "Low"
	RiskMedium = "Medium"
	RiskHigh   = "High"
)

// The methods of verification, as Pele's VerifyType names them.
const (
	VerifyAnalysis      = "Analysis"
	VerifyDemonstration = "Demonstration"
	VerifyInspection    = "Inspection"
	VerifyTest          = "Test"
)

// The kinds of relationship, as Pele's Relationships names them.
const (
	Contains  = "contains"
	Copies    = "copies"
	Derives   = "derives"
	Satisfies = "satisfies"
	Verifies  = "verifies"
	Refines   = "refines"
	Traces    = "traces"
)

// maxStyleCopies is the most style declarations a diagram may copy onto its nodes. Applying
// a class copies its declarations onto each node, so a short text can ask for hundreds of
// millions of copies; past this many the diagram is refused, as Pele refuses it.
const maxStyleCopies = 1_000_000

// A Requirement is a requirement of the diagram, as Pele's Requirement. Its fields are ""
// where its body does not give them.
type Requirement struct {
	Name          string
	Type          string // one of the Kind constants
	RequirementID string
	Text          string
	Risk          string // one of the Risk constants
	VerifyMethod  string // one of the Verify constants
	CSSStyles     []string
	Classes       []string
}

// An Element is an element of the diagram, as Pele's RequirementElement.
type Element struct {
	Name      string
	Type      string
	DocRef    string
	CSSStyles []string
	Classes   []string
}

// A Relation is a relationship from Src to Dst, as Pele's Relation. Its ends are names that
// may name no requirement or element.
type Relation struct {
	Type     string // one of the relationship constants
	Src, Dst string
}

// A Class is a class defined with classDef, as Pele's RequirementClass.
type Class struct {
	ID         string
	Styles     []string
	TextStyles []string
}

// DB is a parsed requirement diagram, as Pele's RequirementDb. Requirements, Elements and
// Classes keep the order they were first named in, as a Map does.
type DB struct {
	Direction    string
	Requirements []*Requirement
	Elements     []*Element
	Relations    []Relation
	Classes      []*Class
	Title        string
	AccTitle     string
	AccDescr     string

	requirements map[string]*Requirement
	elements     map[string]*Element
	classes      map[string]*Class

	// The grammar reports a body's fields before the requirement or element they belong to.
	newID, newText, newRisk, newVerifyMethod string
	newElementType, newDocRef                string
	styleCopies                              int
}

// NewDB is an empty DB.
func NewDB() *DB {
	return &DB{
		Direction:    "TB",
		requirements: map[string]*Requirement{},
		elements:     map[string]*Element{},
		classes:      map[string]*Class{},
	}
}

// Requirement is the requirement of the name, or nil.
func (db *DB) Requirement(name string) *Requirement { return db.requirements[name] }

// Element is the element of the name, or nil.
func (db *DB) Element(name string) *Element { return db.elements[name] }

// copyStyles appends styles to target, counting them against maxStyleCopies, as Pele's copy
// does.
func (db *DB) copyStyles(target *[]string, styles []string) {
	db.styleCopies += len(styles)
	if db.styleCopies > maxStyleCopies {
		panic(parseError{common.NewError("Requirement diagram applies too many style declarations.", diagram.Limit, "requirement")})
	}
	*target = append(*target, styles...)
}

func (db *DB) setDirection(dir string) { db.Direction = dir }

// addRequirement adds a requirement with the fields its body gave, unless one of the name is
// there already, as Pele's addRequirement does.
func (db *DB) addRequirement(name, typ string) *Requirement {
	r, ok := db.requirements[name]
	if !ok {
		r = &Requirement{
			Name:          name,
			Type:          typ,
			RequirementID: db.newID,
			Text:          db.newText,
			Risk:          db.newRisk,
			VerifyMethod:  db.newVerifyMethod,
			CSSStyles:     []string{},
			Classes:       []string{"default"},
		}
		db.requirements[name] = r
		db.Requirements = append(db.Requirements, r)
	}
	db.newID, db.newText, db.newRisk, db.newVerifyMethod = "", "", "", ""
	return r
}

// addElement adds an element with the fields its body gave, unless one of the name is there
// already, as Pele's addElement does.
func (db *DB) addElement(name string) *Element {
	e, ok := db.elements[name]
	if !ok {
		e = &Element{Name: name, Type: db.newElementType, DocRef: db.newDocRef, CSSStyles: []string{}, Classes: []string{"default"}}
		db.elements[name] = e
		db.Elements = append(db.Elements, e)
	}
	db.newElementType, db.newDocRef = "", ""
	return e
}

func (db *DB) addRelationship(typ, src, dst string) {
	db.Relations = append(db.Relations, Relation{Type: typ, Src: src, Dst: dst})
}

// node is the styles and classes of the requirement of the id, or else of the element of
// it, as Pele's node finds them.
func (db *DB) node(id string) (styles, classes *[]string, ok bool) {
	if r, found := db.requirements[id]; found {
		return &r.CSSStyles, &r.Classes, true
	}
	if e, found := db.elements[id]; found {
		return &e.CSSStyles, &e.Classes, true
	}
	return nil, nil, false
}

// setCSSStyle gives the nodes of ids the styles, as Pele's setCssStyle does: Mermaid stops at
// the first unknown id, leaving the rest of the list unstyled.
func (db *DB) setCSSStyle(ids, styles []string) {
	for _, id := range ids {
		target, _, ok := db.node(id)
		if styles == nil || !ok {
			return
		}
		for _, s := range styles {
			if strings.Contains(s, ",") {
				db.copyStyles(target, strings.Split(s, ","))
			} else {
				db.copyStyles(target, []string{s})
			}
		}
	}
}

// setClass gives the nodes of ids the classes, and the styles the classes have, as Pele's
// setClass does.
func (db *DB) setClass(ids, classNames []string) {
	for _, id := range ids {
		styles, classes, ok := db.node(id)
		if !ok {
			continue
		}
		for _, name := range classNames {
			*classes = append(*classes, name)
			if c, ok := db.classes[name]; ok {
				db.copyStyles(styles, c.Styles)
			}
		}
	}
}

// defineClass defines the classes of ids, or adds to them, and gives the styles to the nodes
// that have them already, as Pele's defineClass does.
func (db *DB) defineClass(ids, style []string) {
	for _, id := range ids {
		c, ok := db.classes[id]
		if !ok {
			c = &Class{ID: id, Styles: []string{}, TextStyles: []string{}}
			db.classes[id] = c
			db.Classes = append(db.Classes, c)
		}
		for _, s := range style {
			if strings.Contains(s, "color") {
				c.TextStyles = append(c.TextStyles, strings.Replace(s, "fill", "bgFill", 1))
			}
			c.Styles = append(c.Styles, s)
		}
		var split []string
		for _, s := range style {
			split = append(split, strings.Split(s, ",")...)
		}
		for _, r := range db.Requirements {
			if has(r.Classes, id) {
				db.copyStyles(&r.CSSStyles, split)
			}
		}
		for _, e := range db.Elements {
			if has(e.Classes, id) {
				db.copyStyles(&e.CSSStyles, split)
			}
		}
	}
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// setAccTitle is Pele's setAccTitle: the text with the white space at its start taken off.
func (db *DB) setAccTitle(text string) { db.AccTitle = chars.TrimStart(text) }

// setAccDescription is Pele's setAccDescription: the text with /\n\s+/g replaced by a line
// feed.
func (db *DB) setAccDescription(text string) {
	db.AccDescr = chars.DropIndents(text)
}
