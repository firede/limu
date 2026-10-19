package usecase

import (
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
)

// jsonRow is a row of a JSON table, Pele's JsonRow.
type jsonRow struct {
	// Key is blank on the second and later values of an array.
	Key string
	// AccessibleKey is the key again, for every row.
	AccessibleKey string
	Value         string
}

// graphNode is an actor, a use case, a note, a JSON table or a boundary as it is drawn, Pele's
// UsecaseGraphNode. A string Pele may leave undefined is empty where it is.
type graphNode struct {
	ID        string
	Label     string
	LabelType LabelType
	// Shape is Mermaid's name of the shape: usecaseActor, usecaseActorHollow,
	// usecaseActorAwesome, usecaseActorIcon, usecaseEllipse, usecaseBusiness, rect, note,
	// usecaseJsonTable or usecaseSystemBoundary.
	Shape             string
	IsGroup           bool
	Padding           float64
	CSSClasses        string
	CSSStyles         []string
	CSSCompiledStyles []string
	ActorType         string
	// Business is whether an actor or a use case is a business one; HasBusiness says whether
	// the node is either.
	Business     bool
	HasBusiness  bool
	Icon         string
	Stereotype   string
	ParentID     string
	NoteTarget   string
	JSONRows     []jsonRow
	BoundaryType string
}

// graphEdge is a relationship, or the line from a note to what it is about, as it is drawn:
// Pele's UsecaseGraphEdge.
type graphEdge struct {
	ID    string
	Start string
	End   string
	// RelationshipType is the relationship's type, or "note".
	RelationshipType string
	// Pattern is "solid" or "dotted".
	Pattern        string
	ArrowTypeStart string
	ArrowTypeEnd   string
	// Label and LabelType are each "" where Pele leaves them undefined.
	Label             string
	LabelType         LabelType
	Classes           string
	Style             []string
	CSSCompiledStyles []string
	Animate           bool
	Animation         string
	MinLen            int
	ExplicitID        bool
	// Internal is set on the dotted line from a note to what it is about.
	Internal bool
}

// graph is what is drawn of a use case diagram, Pele's UsecaseGraph.
type graph struct {
	Nodes []*graphNode
	Edges []*graphEdge
}

var actorShapes = map[string]string{
	"normal":  "usecaseActor",
	"hollow":  "usecaseActorHollow",
	"awesome": "usecaseActorAwesome",
	"icon":    "usecaseActorIcon",
}

// markerTypes are the start and end markers of each arrow type, in the order of the arrow
// types: Pele's MARKERS.
var markerTypes = [...][2]string{
	{"none", "arrow_point"},
	{"arrow_point", "none"},
	{"none", "none"},
	{"none", "arrow_circle"},
	{"none", "arrow_cross"},
	{"arrow_circle", "none"},
	{"arrow_cross", "none"},
}

// scalar is how a JSON table writes a value that is not an array or an object, as String()
// does in Pele's scalar.
func scalar(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return "null"
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		return chars.NumberString(x)
	case undefinedValue:
		return "undefined"
	}
	return ""
}

// undefinedValue is the value of a key a JSON object does not have.
type undefinedValue struct{}

// isObjectValue reports whether a JSON value is an array or an object, which Pele's
// flattenJsonRows tests as item !== null && typeof item === 'object'.
func isObjectValue(v any) bool {
	switch v.(type) {
	case *Object, []any:
		return true
	}
	return false
}

// flattenJSONRows is one row for each leaf of a JSON object, depth first in the order the keys
// were written, as Pele's flattenJsonRows makes them. Nesting is kept on a stack of its own.
func flattenJSONRows(value *Object, order *KeyOrder) []jsonRow {
	type entry struct {
		value         any
		path, pointer string
	}
	var rows []jsonRow
	stack := []entry{{value, "", ""}}
	var children []entry
	for len(stack) > 0 {
		e := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		children = children[:0]
		switch current := e.value.(type) {
		case []any:
			scalars := true
			for _, item := range current {
				if isObjectValue(item) {
					scalars = false
					break
				}
			}
			switch {
			case len(current) == 0:
				rows = append(rows, jsonRow{e.path, e.path, "[]"})
			case scalars:
				for i, item := range current {
					key := ""
					if i == 0 {
						key = e.path
					}
					rows = append(rows, jsonRow{key, e.path, scalar(item)})
				}
			default:
				for i, item := range current {
					children = append(children, entry{item, e.path + "[" + strconv.Itoa(i) + "]", e.pointer + "/" + strconv.Itoa(i)})
				}
			}
		case *Object:
			var keys []string
			if order != nil {
				if k, ok := order.Keys(e.pointer); ok {
					keys = k
				} else {
					keys = current.Keys
				}
			} else {
				keys = current.Keys
			}
			if len(keys) == 0 {
				rows = append(rows, jsonRow{e.path, e.path, "{}"})
			}
			for _, key := range keys {
				v, ok := current.Values[key]
				if !ok {
					v = undefinedValue{}
				}
				path := key
				if e.path != "" {
					path = e.path + "." + key
				}
				children = append(children, entry{v, path, e.pointer + "/" + escapePointer(key)})
			}
		default:
			rows = append(rows, jsonRow{e.path, e.path, scalar(current)})
		}
		for i := len(children) - 1; i >= 0; i-- {
			stack = append(stack, children[i])
		}
	}
	return rows
}

// classList joins the names that are not empty with spaces, as Pele's classList does.
func classList(names ...string) string {
	var kept []string
	for _, n := range names {
		if n != "" {
			kept = append(kept, n)
		}
	}
	return strings.Join(kept, " ")
}

// markersOf are the start and end markers of a relationship, as Pele's markers gives them.
func markersOf(r *Relationship) [2]string {
	if r.Type == "generalization" {
		return [2]string{"none", "extension"}
	}
	if r.Type != "association" {
		return markerTypes[SolidArrow]
	}
	if r.ArrowType >= 0 && r.ArrowType < len(markerTypes) {
		return markerTypes[r.ArrowType]
	}
	return markerTypes[LineSolid]
}

// buildGraph resolves a parsed use case diagram into what is drawn: every actor, use case,
// note, JSON table and boundary as a node with its class styles, and every relationship and
// note connector as an edge with its markers. It ports Pele's buildUsecaseGraph, which ports
// getData from Mermaid's use case DB.
func buildGraph(m *Model) *graph {
	g := &graph{}
	classDefs := map[string]*ClassDef{}
	for _, c := range m.ClassDefs {
		classDefs[c.ID] = c
	}
	// A later class wins a property but keeps the place the property first took, as the
	// compiled closure of Pele's buildUsecaseGraph has it.
	compiled := func(classNames []string) []string {
		out := newOrdered[string]()
		for _, name := range append([]string{"default"}, classNames...) {
			def, ok := classDefs[name]
			if !ok {
				continue
			}
			for _, raw := range def.Styles {
				s := chars.Trim(raw)
				property := s
				if colon := strings.IndexByte(s, ':'); colon != -1 {
					property = s[:colon]
				}
				if property = chars.Trim(property); property != "" {
					out.set(property, s)
				}
			}
		}
		return orEmpty(out.values())
	}
	business := func(b bool, name string) string {
		if b {
			return name
		}
		return ""
	}

	for _, a := range m.Actors {
		g.Nodes = append(g.Nodes, &graphNode{
			ID: a.ID, Label: a.Label, LabelType: a.LabelType, Shape: actorShapes[a.Type], Padding: 10,
			CSSClasses: classList(append([]string{"default", "usecase-actor", "usecase-actor-" + a.Type,
				business(a.Business, "usecase-business")}, a.Classes...)...),
			CSSStyles: orEmpty(append([]string(nil), a.Styles...)), CSSCompiledStyles: compiled(a.Classes),
			ActorType: a.Type, Business: a.Business, HasBusiness: true, Icon: a.Icon, Stereotype: a.Stereotype,
			ParentID: a.ParentID,
		})
	}
	for _, u := range m.UseCases {
		ellipse := u.Shape == "ellipse"
		shape, padding := "rect", 10.0
		if ellipse {
			padding = 20
			if u.Business {
				shape = "usecaseBusiness"
			} else {
				shape = "usecaseEllipse"
			}
		}
		g.Nodes = append(g.Nodes, &graphNode{
			ID: u.ID, Label: u.Label, LabelType: u.LabelType, Shape: shape, Padding: padding,
			CSSClasses: classList(append([]string{"default", "usecase-element", "usecase-" + u.Shape,
				business(u.Business, "usecase-business")}, u.Classes...)...),
			CSSStyles: orEmpty(append([]string(nil), u.Styles...)), CSSCompiledStyles: compiled(u.Classes),
			Business: u.Business, HasBusiness: true, Stereotype: u.Stereotype, ParentID: u.ParentID,
		})
	}
	for _, n := range m.Notes {
		g.Nodes = append(g.Nodes, &graphNode{
			ID: n.ID, Label: n.Label, LabelType: n.LabelType, Shape: "note", Padding: 10,
			CSSClasses: "default usecase-note", CSSStyles: []string{}, CSSCompiledStyles: compiled(nil),
			NoteTarget: n.Target,
		})
	}
	for _, j := range m.JSONNodes {
		g.Nodes = append(g.Nodes, &graphNode{
			ID: j.ID, Label: j.ID, LabelType: LabelText, Shape: "usecaseJsonTable", Padding: 10,
			CSSClasses: classList(append([]string{"default", "usecase-json-table"}, j.Classes...)...),
			CSSStyles:  orEmpty(append([]string(nil), j.Styles...)), CSSCompiledStyles: compiled(j.Classes),
			JSONRows: flattenJSONRows(j.Value, j.PropertyOrder),
		})
	}
	for _, b := range m.SystemBoundaries {
		g.Nodes = append(g.Nodes, &graphNode{
			ID: b.ID, Label: b.Label, LabelType: b.LabelType, Shape: "usecaseSystemBoundary", IsGroup: true,
			Padding:    20,
			CSSClasses: classList(append([]string{"default", "system-boundary", "system-boundary-" + b.Type}, b.Classes...)...),
			CSSStyles:  orEmpty(append([]string(nil), b.Styles...)), CSSCompiledStyles: compiled(b.Classes),
			BoundaryType: b.Type,
		})
	}

	for _, r := range m.Relationships {
		semantic := r.Type == "include" || r.Type == "extend"
		ends := markersOf(r)
		animated := r.Animate || r.Animation != ""
		pattern := "solid"
		if semantic {
			pattern = "dotted"
		}
		animation := ""
		if animated {
			animation = "edge-animation-fast"
			if r.Animation != "" {
				animation = "edge-animation-" + r.Animation
			}
		}
		e := &graphEdge{
			ID: r.ID, Start: r.Source, End: r.Target, RelationshipType: r.Type, Pattern: pattern,
			ArrowTypeStart: ends[0], ArrowTypeEnd: ends[1],
			Classes: classList(append(append([]string{"default", "relationship", "relationship-" + r.Type}, r.Classes...), animation)...),
			Style:   orEmpty(append([]string(nil), r.Styles...)), CSSCompiledStyles: compiled(r.Classes),
			Animate: r.Animate, Animation: r.Animation, MinLen: r.MinLen, ExplicitID: r.ExplicitID,
		}
		// Include and extend are labelled with their kind, whatever label the relationship
		// carries.
		if semantic {
			e.Label, e.LabelType = r.Type, LabelText
		} else if r.Type == "association" {
			e.Label, e.LabelType = r.Label, r.LabelType
		}
		g.Edges = append(g.Edges, e)
	}
	for _, n := range m.Notes {
		g.Edges = append(g.Edges, &graphEdge{
			ID: n.ID + "-edge", Start: n.ID, End: n.Target, RelationshipType: "note", Pattern: "dotted",
			ArrowTypeStart: "none", ArrowTypeEnd: "none", Classes: "default relationship relationship-note",
			Style: []string{}, CSSCompiledStyles: compiled(nil), MinLen: 1, Internal: true,
		})
	}
	return g
}
