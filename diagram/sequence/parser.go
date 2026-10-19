package sequence

import (
	"unicode/utf16"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// An item of the statement list the parser hands to apply, as Mermaid's grammar builds it:
// a *stmt, a string, or a []item. Pele's SeqItem.
type item = any

// stmt is a statement, Pele's SeqStatement. Of its texts, text holds whichever the
// statement's type has: a message's, a note's, a menu's, or a block's.
type stmt struct {
	typ         string
	actor       string
	pair        []string // the two participants of a note over them
	draw        string
	description *description
	config      *string
	signalType  int
	from, to    string
	text        *seqText
	activate    bool
	central     int
	placement   int
	box         boxData
	index, step float64
	hasIndex    bool
	visible     bool
}

// signalOf is the kind of message of each arrow token, Pele's SIGNAL.
var signalOf = func() (out [tokenCount]int) {
	for i := range out {
		out[i] = -1
	}
	out[tSOLID_OPEN_ARROW] = SolidOpen
	out[tDOTTED_OPEN_ARROW] = DottedOpen
	out[tSOLID_ARROW] = Solid
	out[tDOTTED_ARROW] = Dotted
	out[tBIDIRECTIONAL_SOLID_ARROW] = BidirectionalSolid
	out[tBIDIRECTIONAL_DOTTED_ARROW] = BidirectionalDotted
	out[tSOLID_CROSS] = SolidCross
	out[tDOTTED_CROSS] = DottedCross
	out[tSOLID_POINT] = SolidPoint
	out[tDOTTED_POINT] = DottedPoint
	out[tSOLID_ARROW_TOP] = SolidTop
	out[tSOLID_ARROW_BOTTOM] = SolidBottom
	out[tSTICK_ARROW_TOP] = StickTop
	out[tSTICK_ARROW_BOTTOM] = StickBottom
	out[tSOLID_ARROW_TOP_DOTTED] = SolidTopDotted
	out[tSOLID_ARROW_BOTTOM_DOTTED] = SolidBottomDotted
	out[tSTICK_ARROW_TOP_DOTTED] = StickTopDotted
	out[tSTICK_ARROW_BOTTOM_DOTTED] = StickBottomDotted
	out[tSOLID_ARROW_TOP_REVERSE] = SolidArrowTopReverse
	out[tSOLID_ARROW_BOTTOM_REVERSE] = SolidArrowBottomReverse
	out[tSTICK_ARROW_TOP_REVERSE] = StickArrowTopReverse
	out[tSTICK_ARROW_BOTTOM_REVERSE] = StickArrowBottomReverse
	out[tSOLID_ARROW_TOP_REVERSE_DOTTED] = SolidArrowTopReverseDotted
	out[tSOLID_ARROW_BOTTOM_REVERSE_DOTTED] = SolidArrowBottomReverseDotted
	out[tSTICK_ARROW_TOP_REVERSE_DOTTED] = StickArrowTopReverseDotted
	out[tSTICK_ARROW_BOTTOM_REVERSE_DOTTED] = StickArrowBottomReverseDotted
	return
}()

// The kinds of block, Pele's K.
const (
	kTop = iota
	kBox
	kLoop
	kRect
	kOpt
	kBreak
	kAlt
	kPar
	kCritical
)

// blockOf is the kind of block each token opens, or -1.
var blockOf = func() (out [tokenCount]int) {
	for i := range out {
		out[i] = -1
	}
	out[tLoop] = kLoop
	out[tRect] = kRect
	out[tOpt] = kOpt
	out[tBreak] = kBreak
	out[tAlt] = kAlt
	out[tPar] = kPar
	out[tParOver] = kPar
	out[tCritical] = kCritical
	out[tBox] = kBox
	return
}()

// frame is a block being read, Pele's Frame.
type frame struct {
	kind      int
	text      string
	startType int
	// The current section's statements. Slot 0 of a block's first section is kept for its
	// start entry.
	list []item
	// Finished sections, each with the divider text that ended it.
	sections []section
}

type section struct {
	list []item
	text string
}

const signalTokens = "'SOLID_OPEN_ARROW', 'DOTTED_OPEN_ARROW', 'SOLID_ARROW', 'DOTTED_ARROW', 'SOLID_CROSS', 'DOTTED_CROSS', 'SOLID_POINT', 'DOTTED_POINT'"

// syntaxFailure carries a syntax error up out of the parser.
type syntaxFailure struct{ err *diagram.Error }

// parse reads a sequence diagram into db, as Pele's parseSequence does.
func parse(src string, db *DB) (err error) {
	defer func() {
		if e := recover(); e != nil {
			switch e := e.(type) {
			case syntaxFailure:
				err = e.err
			case failure:
				err = e.err
			default:
				panic(e)
			}
		}
	}()
	parseSequence(src, db)
	return nil
}

func parseSequence(src string, db *DB) {
	toks := tokenize(src)
	i := 0

	failAt := func(expected string) {
		t := toks[i]
		if t.typ == tERROR {
			panic(syntaxFailure{diagram.SyntaxError("sequence", src, t.start, "", true)})
		}
		panic(syntaxFailure{diagram.SyntaxError("sequence", src, t.start, "Expecting "+expected+", got '"+tokenNames[t.typ]+"'", false)})
	}
	expect := func(typ int) string {
		if toks[i].typ != typ {
			failAt("'" + tokenNames[typ] + "'")
		}
		i++
		return toks[i-1].text
	}
	actor := func() *stmt {
		if t := toks[i].typ; t != tACTOR && t != tLink {
			failAt("'ACTOR'")
		}
		i++
		return &stmt{typ: "addParticipant", actor: toks[i-1].text}
	}
	text2 := func() *seqText {
		t := db.parseMessage(chars.DropUnits(chars.Trim(expect(tTXT)), 1))
		return &t
	}

	participant := func() *stmt {
		t := toks[i].typ
		i++
		if t == tDestroy {
			node := actor()
			expect(tNEWLINE)
			node.typ = "destroyParticipant"
			return node
		}
		node := actor()
		if toks[i].typ == tCONFIG_START && toks[i-1].typ == tACTOR {
			i++
			content := expect(tCONFIG_CONTENT)
			expect(tCONFIG_END)
			c := chars.Trim(content)
			node.config = &c
		}
		node.draw = "actor"
		if t == tParticipant {
			node.draw = "participant"
		}
		switch toks[i].typ {
		case tAS:
			i++
			alias := expect(tRestOfLine)
			expect(tNEWLINE)
			m := db.parseMessage(alias)
			node.description = &description{m.text, m.wrap}
		case tNEWLINE:
			i++
		default:
			failAt("'AS', 'NEWLINE'")
		}
		return node
	}

	signal := func(from *stmt) item {
		centralFrom := false
		if toks[i].typ == tCENTRAL {
			centralFrom = true
			i++
		}
		signalType := signalOf[toks[i].typ]
		if signalType < 0 {
			failAt(signalTokens)
		}
		i++
		mark := toks[i].typ
		if mark == tCENTRAL || !centralFrom && (mark == tPLUS || mark == tMINUS) {
			i++
		}
		to := actor()
		msg := text2()
		message := &stmt{typ: "addMessage", from: from.actor, to: to.actor, signalType: signalType, text: msg}
		out := []item{from, to, message}
		switch {
		case centralFrom:
			dual := mark == tCENTRAL
			message.activate = dual
			message.central = CentralConnectionReverse
			if dual {
				message.central = CentralConnectionDual
				out = append(out, &stmt{typ: "centralConnection", signalType: CentralConnection, actor: to.actor})
			}
			out = append(out, &stmt{typ: "centralConnectionReverse", signalType: CentralConnectionReverse, actor: from.actor})
		case mark == tPLUS:
			message.activate = true
			out = append(out, &stmt{typ: "activeStart", signalType: ActiveStart, actor: to.actor})
		case mark == tMINUS:
			out = append(out, &stmt{typ: "activeEnd", signalType: ActiveEnd, actor: from.actor})
		case mark == tCENTRAL:
			message.activate = true
			message.central = CentralConnection
			out = append(out, &stmt{typ: "centralConnection", signalType: CentralConnection, actor: to.actor})
		}
		return out
	}

	menu := func(typ string) item {
		i++
		who := actor()
		return []item{who, &stmt{typ: typ, actor: who.actor, text: text2()}}
	}

	note := func() item {
		i++
		t := toks[i].typ
		if t == tOver {
			i++
			first := actor()
			var who item = first
			ids := []string{first.actor, first.actor}
			if toks[i].typ == tCOMMA {
				i++
				second := actor()
				who = []item{first, second}
				ids = []string{first.actor, second.actor}
			}
			return []item{who, &stmt{typ: "addNote", placement: Over, pair: ids, text: text2()}}
		}
		if t != tLeftOf && t != tRightOf {
			failAt("'over', 'left_of', 'right_of'")
		}
		i++
		who := actor()
		placement := RightOf
		if t == tLeftOf {
			placement = LeftOf
		}
		return []item{who, &stmt{typ: "addNote", placement: placement, actor: who.actor, text: text2()}}
	}

	autonumber := func() *stmt {
		i++
		node := &stmt{typ: "sequenceIndex", visible: true, signalType: Autonumber}
		if toks[i].typ == tNUM {
			node.index = chars.Number(toks[i].text)
			node.hasIndex = true
			i++
			node.step = 1
			if toks[i].typ == tNUM {
				node.step = chars.Number(toks[i].text)
				i++
			}
		} else if toks[i].typ == tOff {
			i++
			node.visible = false
		}
		if toks[i].typ != tNEWLINE {
			failAt("'NEWLINE', 'NUM', 'off'")
		}
		i++
		return node
	}

	// fold folds the sections of an alt, par or critical block from the last one back, as the
	// grammar's right recursion does, so that the texts reach the DB in the same order.
	fold := func(f *frame, divider string, signalType int) []item {
		list := f.list
		for k := len(f.sections) - 1; k >= 0; k-- {
			t := db.parseMessage(f.sections[k].text)
			head := &stmt{typ: divider, signalType: signalType, text: &t}
			list = append(append(append([]item(nil), f.sections[k].list...), head), list)
		}
		return list
	}

	msgOf := func(s string) *seqText {
		t := db.parseMessage(s)
		return &t
	}
	closeFrame := func(f *frame) []item {
		text := f.text
		list := f.list
		switch f.kind {
		case kBox:
			list[0] = &stmt{typ: "boxStart", box: db.parseBoxData(text)}
			list = append(list, &stmt{typ: "boxEnd"})
		case kLoop:
			list[0] = &stmt{typ: "loopStart", text: msgOf(text), signalType: LoopStart}
			list = append(list, &stmt{typ: "loopEnd", signalType: LoopEnd})
		case kRect:
			list[0] = &stmt{typ: "rectStart", text: msgOf(text), signalType: RectStart}
			list = append(list, &stmt{typ: "rectEnd", signalType: RectEnd})
		case kOpt:
			list[0] = &stmt{typ: "optStart", text: msgOf(text), signalType: OptStart}
			list = append(list, &stmt{typ: "optEnd", signalType: OptEnd})
		case kBreak:
			list[0] = &stmt{typ: "breakStart", text: msgOf(text), signalType: BreakStart}
			list = append(list, &stmt{typ: "breakEnd", signalType: BreakEnd})
		case kAlt:
			list = fold(f, "else", AltElse)
			list[0] = &stmt{typ: "altStart", text: msgOf(text), signalType: AltStart}
			list = append(list, &stmt{typ: "altEnd", signalType: AltEnd})
		case kPar:
			list = fold(f, "and", ParAnd)
			list[0] = &stmt{typ: "parStart", text: msgOf(text), signalType: f.startType}
			list = append(list, &stmt{typ: "parEnd", signalType: ParEnd})
		default:
			list = fold(f, "option", CriticalOption)
			list[0] = &stmt{typ: "criticalStart", text: msgOf(text), signalType: CriticalStart}
			list = append(list, &stmt{typ: "criticalEnd", signalType: CriticalEnd})
		}
		return list
	}

	for toks[i].typ == tNEWLINE {
		i++
	}
	expect(tSD)

	var stack []*frame
	f := &frame{kind: kTop}
	pop := func() {
		done := closeFrame(f)
		f = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		f.list = append(f.list, done)
	}
	for {
		t := toks[i].typ
		if f.kind == kBox {
			switch t {
			case tParticipant, tParticipantActor, tDestroy:
				f.list = append(f.list, participant())
			case tNEWLINE:
				f.list = append(f.list, []item{})
				i++
			case tEnd:
				i++
				pop()
			default:
				failAt("'end', 'NEWLINE', 'participant', 'participant_actor', 'destroy'")
			}
			continue
		}
		switch t {
		case tNEWLINE, tINVALID:
			f.list = append(f.list, []item{})
			i++
		case tParticipant, tParticipantActor, tDestroy:
			f.list = append(f.list, participant())
		case tCreate:
			i++
			if next := toks[i].typ; next != tParticipant && next != tParticipantActor && next != tDestroy {
				failAt("'participant', 'participant_actor', 'destroy'")
			}
			node := participant()
			node.typ = "createParticipant"
			f.list = append(f.list, node)
		case tACTOR:
			f.list = append(f.list, signal(actor()))
			expect(tNEWLINE)
		case tLink:
			if next := toks[i+1].typ; next == tACTOR || next == tLink {
				f.list = append(f.list, menu("addALink"))
			} else {
				f.list = append(f.list, signal(actor()))
			}
			expect(tNEWLINE)
		case tLinks:
			f.list = append(f.list, menu("addLinks"))
			expect(tNEWLINE)
		case tProperties:
			f.list = append(f.list, menu("addProperties"))
			expect(tNEWLINE)
		case tDetails:
			f.list = append(f.list, menu("addDetails"))
			expect(tNEWLINE)
		case tNote:
			f.list = append(f.list, note())
			expect(tNEWLINE)
		case tAutonumber:
			f.list = append(f.list, autonumber())
		case tActivate, tDeactivate:
			i++
			who := actor()
			expect(tNEWLINE)
			if t == tActivate {
				f.list = append(f.list, &stmt{typ: "activeStart", signalType: ActiveStart, actor: who.actor})
			} else {
				f.list = append(f.list, &stmt{typ: "activeEnd", signalType: ActiveEnd, actor: who.actor})
			}
		case tTitle, tLegacyTitle:
			n := 6
			if t == tLegacyTitle {
				n = 7
			}
			title := chars.DropUnits(toks[i].text, n)
			i++
			db.setDiagramTitle(title)
			f.list = append(f.list, title)
		case tAccTitle:
			i++
			value := chars.Trim(expect(tAccTitleValue))
			db.setAccTitle(value)
			f.list = append(f.list, value)
		case tAccDescr:
			i++
			value := chars.Trim(expect(tAccDescrValue))
			db.setAccDescription(value)
			f.list = append(f.list, value)
		case tAccDescrMultilineValue:
			value := chars.Trim(toks[i].text)
			i++
			db.setAccDescription(value)
			f.list = append(f.list, value)
		case tLoop, tRect, tOpt, tBreak, tAlt, tPar, tParOver, tCritical, tBox:
			i++
			text := expect(tRestOfLine)
			stack = append(stack, f)
			start := ParStart
			if t == tParOver {
				start = ParOverStart
			}
			f = &frame{kind: blockOf[t], text: text, startType: start, list: []item{[]item{}}}
		case tElse, tAnd, tOption:
			want := kCritical
			if t == tElse {
				want = kAlt
			} else if t == tAnd {
				want = kPar
			}
			if f.kind != want {
				failAt("'end'")
			}
			i++
			text := expect(tRestOfLine)
			f.sections = append(f.sections, section{f.list, text})
			f.list = []item{}
		case tEnd:
			if f.kind == kTop {
				failAt("'NEWLINE', 'EOF'")
			}
			i++
			pop()
		case tEND:
			if f.kind != kTop {
				failAt("'end'")
			}
			db.apply(f.list)
			return
		default:
			failAt("'NEWLINE', 'ACTOR', 'participant', 'note', 'end'")
		}
	}
}

// units16 is a string's UTF-16 code units.
func units16(s string) []uint16 { return utf16.Encode([]rune(s)) }

// decode16 is code units as a string; a lone surrogate becomes U+FFFD.
func decode16(u []uint16) string { return string(utf16.Decode(u)) }
