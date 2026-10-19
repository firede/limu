// Package eventmodeling parses Mermaid's event modeling diagrams and lays them out, as Pele's
// src/diagrams/eventmodeling does: a tokenizer and a parser that read what Mermaid's Langium
// grammar reads, buildEventModel, which sorts the frames into lanes and finds what feeds
// each, and Layout, which lays the lanes out across, with time running left to right, or the
// frames down as cards in a narrow host.
package eventmodeling

import (
	"slices"
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// Kind is the kind of entity a frame shows, Pele's EntityKind: "ui", "processor",
// "command", "readmodel" or "event".
type Kind string

// Group is the group of lanes a kind of entity is drawn in, Pele's LaneGroup: "trigger"
// for screens and processors, "state" for commands and read models, "stream" for events.
type Group string

// Lane is a swimlane, Pele's EmLane.
type Lane struct {
	// Index orders the lanes, as Mermaid's key does: 0 for UI/Automation, 100 for
	// Command/Read Model, 200 for Events, and the next free number after those for each
	// namespace.
	Index     int
	Label     string
	Group     Group
	Namespace string // "" for none
}

// Box is a frame, Pele's EmBox.
type Box struct {
	ID        string // the frame number as written, as "01"
	Kind      Kind
	Entity    string
	Name      string
	Namespace string // "" for none
	Reset     bool
	Lane      *Lane
	// Data is the example data of the frame or of the data block it refers to, if HasData.
	Data     string
	HasData  bool
	DataType string // "" for none
}

// Relation is a frame that feeds another, Pele's EmRelation.
type Relation struct {
	Source, Target *Box
}

// Note is a note on a frame, Pele's EmNote.
type Note struct {
	Frame, Text string
	DataType    string // "" for none
}

// Specification is a given, when, then specification of a frame, Pele's EmSpecification.
type Specification struct {
	Frame             string
	Given, When, Then []string
}

// Model is a parsed event modeling diagram, Pele's EventModelingModel. A string Pele leaves
// undefined is empty.
type Model struct {
	Title, AccTitle, AccDescr string
	Boxes                     []*Box
	// Lanes are sorted from top to bottom.
	Lanes          []*Lane
	Relations      []Relation
	Notes          []Note
	Specifications []Specification
	Entities       []string
}

// kindOf is Pele's KINDS: the kind of entity each keyword names.
var kindOf = map[string]Kind{
	"ui": "ui", "pcr": "processor", "processor": "processor", "rmo": "readmodel", "readmodel": "readmodel",
	"cmd": "command", "command": "command", "evt": "event", "event": "event",
}

// groupInfo is an entry of Pele's GROUPS: the first index of a group, its label, and the
// prefix of the label of a lane of a namespace in it.
type groupInfo struct {
	first         int
	label, prefix string
}

var groups = map[Group]groupInfo{
	"trigger": {0, "UI/Automation", "UI/A: "},
	"state":   {100, "Command/Read Model", "C/RM: "},
	"stream":  {200, "Events", "Stream: "},
}

// groupOf is the group of lanes of a kind of entity, as Pele's groupOf gives it.
func groupOf(kind Kind) Group {
	switch kind {
	case "ui", "processor":
		return "trigger"
	case "event":
		return "stream"
	}
	return "state"
}

// blockText is the text between the outer braces of raw, or inside its quotes, without the
// blank lines about it and the indentation its lines share, as Pele's blockText gives it.
func blockText(raw string) string {
	open := strings.IndexByte(raw, '{')
	end := strings.LastIndexByte(raw, '}')
	var inner string
	switch {
	case open != -1 && end > open:
		inner = raw[open+1 : end]
	case open == -1 && chars.Len16(raw) >= 2:
		// Inline data without braces is quoted, and its quotes are one code unit each.
		inner = raw[1 : len(raw)-1]
	}
	lines := strings.Split(strings.ReplaceAll(inner, "\r", ""), "\n")
	for len(lines) > 0 && chars.Trim(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && chars.Trim(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	indent := -1
	for _, line := range lines {
		if chars.Trim(line) == "" {
			continue
		}
		if n := chars.LeadingSpace(line); indent < 0 || n < indent {
			indent = n
		}
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		// A line that is all white space keeps none of it; any other starts with at least
		// indent characters of it, each one code unit.
		out[i] = chars.TrimEnd(chars.DropUnits(line, indent))
	}
	return strings.Join(out, "\n")
}

// semantic is an error of text that parses but asks for what cannot be, on a line.
func semantic(message string, line int) *diagram.Error {
	return &diagram.Error{Code: diagram.Semantic, Message: message, Type: "eventmodeling", Line: line}
}

// buildEventModel makes the model of a diagram's tree, as Pele's buildEventModel does: a
// lane for each group of kinds and each namespace in one, a box for each frame, and the
// relations between them, named or found.
func buildEventModel(tree *ast, title string) (*Model, error) {
	ids := map[string]bool{}
	for _, frame := range tree.frames {
		if ids[frame.name] {
			line := common.Locate(tree.source, frame.offset).Line
			return nil, semantic(`Duplicate event modeling frame ID "`+frame.name+`" on line `+strconv.Itoa(line), line)
		}
		ids[frame.name] = true
	}

	data := map[string]string{}
	for _, entity := range tree.dataEntities {
		if _, ok := data[entity.name]; !ok {
			data[entity.name] = entity.dataBlockValue
		}
	}

	// One lane per group, and one more per namespace within a group, in the order they are
	// first needed.
	laneOf := map[string]*Lane{}
	var laneOrder []*Lane
	next := map[Group]int{}
	lane := func(kind Kind, namespace string, hasNamespace bool) *Lane {
		group := groupOf(kind)
		key := string(group)
		if hasNamespace {
			key += " " + namespace
		}
		found := laneOf[key]
		if found == nil {
			info := groups[group]
			index := info.first
			label := info.label
			if hasNamespace {
				n, ok := next[group]
				if !ok {
					n = info.first + 1
				}
				index = n
				next[group] = index + 1
				label = info.prefix + namespace
			}
			found = &Lane{Index: index, Label: label, Group: group, Namespace: namespace}
			laneOf[key] = found
			laneOrder = append(laneOrder, found)
		}
		return found
	}

	var boxes []*Box
	var relations []Relation
	for index, frame := range tree.frames {
		// Only a two-part name has a namespace.
		parts := strings.Split(frame.entityIdentifier, ".")
		namespace, name := "", frame.entityIdentifier
		if len(parts) == 2 {
			namespace, name = parts[0], parts[1]
		}
		kind, ok := kindOf[frame.modelEntityType]
		if !ok {
			kind = "event"
		}
		box := &Box{
			ID:        frame.name,
			Kind:      kind,
			Entity:    frame.entityIdentifier,
			Name:      name,
			Namespace: namespace,
			Reset:     frame.reset,
			Lane:      lane(kind, namespace, len(parts) == 2),
			DataType:  frame.dataType,
		}
		if frame.hasInline {
			box.Data, box.HasData = blockText(frame.dataInlineValue), true
		}
		if frame.hasDataReference {
			if block, ok := data[frame.dataReference]; ok {
				box.Data, box.HasData = blockText(block), true
			}
		}
		boxes = append(boxes, box)

		if len(frame.sourceFrames) > 0 {
			// Sources connect in frame order, and only frames that already have a box.
			wanted := map[string]bool{}
			for _, ref := range frame.sourceFrames {
				wanted[ref] = true
			}
			for _, source := range boxes {
				if wanted[source.ID] {
					relations = append(relations, Relation{source, box})
				}
			}
		} else if index > 0 && !box.Reset {
			// Otherwise the nearest earlier frame in another lane feeds this one.
			for i := index - 1; i >= 0; i-- {
				if boxes[i].Lane != box.Lane {
					relations = append(relations, Relation{boxes[i], box})
					break
				}
			}
		}
	}

	m := &Model{Boxes: boxes, Relations: relations}
	if tree.hasTitle {
		m.Title = common.TitleValue(tree.title)
	}
	if m.Title == "" {
		m.Title = title
	}
	if tree.hasAccTitle {
		m.AccTitle = chars.TrimStart(common.AccTitleValue(tree.accTitle))
	}
	if tree.hasAccDescr {
		m.AccDescr = chars.DropIndents(common.AccDescrValue(tree.accDescr))
	}
	m.Lanes = slices.Clone(laneOrder)
	slices.SortStableFunc(m.Lanes, func(a, b *Lane) int { return a.Index - b.Index })
	for _, note := range tree.noteEntities {
		m.Notes = append(m.Notes, Note{Frame: note.sourceFrame, Text: blockText(note.dataBlockValue), DataType: note.dataType})
	}
	for _, gwt := range tree.gwtEntities {
		m.Specifications = append(m.Specifications, Specification{Frame: gwt.sourceFrame, Given: gwt.given, When: gwt.when, Then: gwt.then})
	}
	m.Entities = tree.modelEntities
	return m, nil
}

// Parse parses an event modeling diagram's text, as the parse of Pele's eventmodeling diagram
// does: source is what diagram.Prepared.Source gives with its comments kept, and title the
// title its front matter gives, "" for none. An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	tree, err := parseEventModel(source)
	if err != nil {
		return nil, err
	}
	return buildEventModel(tree, title)
}

// ParsePrepared parses a prepared event modeling diagram. Its comment lines are kept, as
// Pele keeps them, for a data block to show them.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(true), p.Title)
}
