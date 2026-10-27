// Package sequence parses Mermaid's sequence diagrams and lays them out, as Pele's
// src/diagrams/sequence does: a lexer and a parser that agree with Mermaid's
// sequenceDiagram.jison, the DB the statements they read are applied to, and Layout, which
// lays the participants, messages, notes and blocks out as a scene.
//
// The scene's items have the roles sequence.title, sequence.box (a group of participants),
// sequence.rect (a shaded block), sequence.lifeline, sequence.activation, sequence.frame (a
// loop, alt, opt, par, critical or break block, its kind its shape), sequence.divider (the
// dashed line between the sections of a frame), sequence.frame-kind (a frame's tab, with
// its kind), sequence.condition (the text of a frame's section), sequence.message,
// sequence.number (a message's number), sequence.destroy (the cross a lifeline ends in),
// sequence.message-label, sequence.note and sequence.actor (a participant, at the head of
// its lifeline and, mirrored, at its foot; its kind its shape).
package sequence

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// The kinds of message, as Mermaid numbers them: Pele's LINETYPE.
const (
	Solid                         = 0
	Dotted                        = 1
	NoteType                      = 2
	SolidCross                    = 3
	DottedCross                   = 4
	SolidOpen                     = 5
	DottedOpen                    = 6
	LoopStart                     = 10
	LoopEnd                       = 11
	AltStart                      = 12
	AltElse                       = 13
	AltEnd                        = 14
	OptStart                      = 15
	OptEnd                        = 16
	ActiveStart                   = 17
	ActiveEnd                     = 18
	ParStart                      = 19
	ParAnd                        = 20
	ParEnd                        = 21
	RectStart                     = 22
	RectEnd                       = 23
	SolidPoint                    = 24
	DottedPoint                   = 25
	Autonumber                    = 26
	CriticalStart                 = 27
	CriticalOption                = 28
	CriticalEnd                   = 29
	BreakStart                    = 30
	BreakEnd                      = 31
	ParOverStart                  = 32
	BidirectionalSolid            = 33
	BidirectionalDotted           = 34
	SolidTop                      = 41
	SolidBottom                   = 42
	StickTop                      = 43
	StickBottom                   = 44
	SolidArrowTopReverse          = 45
	SolidArrowBottomReverse       = 46
	StickArrowTopReverse          = 47
	StickArrowBottomReverse       = 48
	SolidTopDotted                = 51
	SolidBottomDotted             = 52
	StickTopDotted                = 53
	StickBottomDotted             = 54
	SolidArrowTopReverseDotted    = 55
	SolidArrowBottomReverseDotted = 56
	StickArrowTopReverseDotted    = 57
	StickArrowBottomReverseDotted = 58
	CentralConnection             = 59
	CentralConnectionReverse      = 60
	CentralConnectionDual         = 61
)

// Where a note is, Pele's PLACEMENT.
const (
	LeftOf  = 0
	RightOf = 1
	Over    = 2
)

// Box is a box drawn about participants, Pele's SeqBox. Name is "" for a box with none.
type Box struct {
	Name      string
	Wrap      bool
	Fill      string
	ActorKeys []string
}

// Actor is a participant, Pele's SeqActor.
type Actor struct {
	ID          string
	Box         *Box
	Name        string
	Description string
	Wrap        bool
	// PrevActor and NextActor are the ids of the participants declared before and after it,
	// "" for none.
	PrevActor, NextActor string
	// Links and Properties are what its menu statements set, in the order a JavaScript Map
	// keeps them.
	Links, Properties Entries
	// Details is the id of the page element a details statement names, if HasDetails.
	Details    string
	HasDetails bool
	Type       string
}

// Numbering is what an autonumber statement says: where the numbers start and the step
// between them, if Given, and whether they are shown.
type Numbering struct {
	Start, Step float64
	Given       bool
	Visible     bool
}

// Message is an entry of the diagram's list of messages, Pele's SeqMessage: a message
// between participants, a note, the start or end of an activation or a block, or an
// autonumber statement, by Type.
type Message struct {
	ID string
	// From and To are participants' ids, "" where there is none.
	From, To string
	// Text is the message's text; an autonumber statement has its Numbering instead.
	Text      string
	Numbering *Numbering
	Wrap      bool
	Type      int
	Activate  bool
	// Placement is where a note is.
	Placement int
	// CentralConnection is the kind of central connection a message has, 0 for none.
	CentralConnection int
}

// Note is a note, Pele's SeqNote. A note over two participants, or over one, has the pair
// in Actors; one left or right of a participant has it in Actor.
type Note struct {
	Actor     string
	Actors    []string
	Placement int
	Message   string
	Wrap      bool
}

// Position is a participant's id with the index of the message it is created or destroyed
// at.
type Position struct {
	ID    string
	Index int
}

// DB is a parsed sequence diagram, Pele's SeqDb.
type DB struct {
	Actors      []*Actor
	actorIndex  map[string]*Actor
	Created     []Position
	Destroyed   []Position
	Boxes       []*Box
	Messages    []*Message
	Notes       []*Note
	Title       string
	AccTitle    string
	HasAccTitle bool
	AccDescr    string
	HasAccDescr bool

	prevActor     string
	currentBox    *Box
	lastCreated   string
	lastDestroyed string
	// wrap is the top-level wrap setting: 0 unset, 1 true, -1 false.
	wrap         int
	sequenceWrap bool
	active       map[string]int
}

// Actor is the participant of an id, or nil. It stands for Map.get on the actors of Pele's
// SeqDb.
func (db *DB) Actor(id string) *Actor { return db.actorIndex[id] }

// Parse parses a sequence diagram's text, as the parse of Pele's sequence diagram does:
// source is what diagram.Prepared.Source gives, config the diagram's configuration, whose
// wrap and sequence.wrap say whether text wraps, and title the title its front matter gives,
// "" for none. An error is a *diagram.Error.
func Parse(source string, config diagram.Config, title string) (*DB, error) {
	db := newDB(config)
	if title != "" {
		db.Title = title
	}
	if err := parse(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared sequence diagram. It is limu's own: Pele's core hands its
// parse the source, the config and the title itself.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return common.ParsePrepared(p, false, func(src string) (*DB, error) {
		return Parse(src, p.Config, p.Title)
	})
}
