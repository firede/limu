package sequence

import (
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/yaml"
)

// A flag of three values, for JavaScript's true, false and undefined.
const (
	unset = 0
	yes   = 1
	no    = -1
)

// seqText is a text and whether it is to wrap, if it says: Pele's SeqText.
type seqText struct {
	text string
	wrap int
}

// boxData is what a box statement's line says, Pele's SeqBoxData. text is "" for none.
type boxData struct {
	text  string
	color string
	wrap  int
}

func newDB(config diagram.Config) *DB {
	db := &DB{actorIndex: map[string]*Actor{}, active: map[string]int{}}
	if w, ok := config["wrap"].(bool); ok {
		db.wrap = no
		if w {
			db.wrap = yes
		}
	}
	if seq, ok := config["sequence"].(diagram.Config); ok {
		db.sequenceWrap = seq["wrap"] == true
	}
	return db
}

func semantic(message string) *diagram.Error {
	return diagram.NewError(message, diagram.Semantic, "sequence")
}

// failure carries an error of the DB up through the parser, as Pele's throw does.
type failure struct{ err *diagram.Error }

func fail(message string) { panic(failure{semantic(message)}) }

func (db *DB) autoWrap() bool {
	if db.wrap != unset {
		return db.wrap == yes
	}
	return db.sequenceWrap
}

func (db *DB) wrapOr(w int) bool {
	if w != unset {
		return w == yes
	}
	return db.autoWrap()
}

func (db *DB) setAccTitle(text string) {
	db.AccTitle = chars.TrimStart(text)
	db.HasAccTitle = true
}

// setAccDescription replaces each line break and the white space after it with a line
// break, as text.replace(/\n\s+/g, '\n') does.
func (db *DB) setAccDescription(text string) {
	var b strings.Builder
	for i := 0; i < len(text); {
		if text[i] == '\n' {
			if j := chars.SpaceRun(text, i+1); j > i+1 {
				b.WriteByte('\n')
				i = j
				continue
			}
		}
		b.WriteByte(text[i])
		i++
	}
	db.AccDescr = b.String()
	db.HasAccDescr = true
}

func (db *DB) setDiagramTitle(text string) { db.Title = text }

// wrapPrefix is the length of a leading wrap: or nowrap:, either after an optional colon,
// and whether it says to wrap.
func wrapPrefix(s string) (int, int) {
	i := 0
	if strings.HasPrefix(s, ":") {
		i = 1
	}
	switch {
	case strings.HasPrefix(s[i:], "wrap:"):
		return i + 5, yes
	case strings.HasPrefix(s[i:], "nowrap:"):
		return i + 7, no
	}
	return 0, unset
}

// extractWrap is Pele's extractWrap.
func extractWrap(raw string) seqText {
	text := chars.Trim(raw)
	n, wrap := wrapPrefix(text)
	if wrap != unset {
		text = text[n:]
	}
	return seqText{chars.Trim(text), wrap}
}

func (db *DB) parseMessage(s string) seqText { return extractWrap(s) }

// splitBox splits a box's line into a leading color and the rest, as Mermaid's
// /^((?:rgba?|hsla?)\s*\(.*\)|\w*)(.*)$/ does, as Pele's splitBox does. `.` stops at a line
// terminator, so a line that holds one outside the gap before the parenthesis does not match.
func splitBox(s string) (string, string, bool) {
	if fn := boxFunction(s); fn > 0 {
		if end := strings.LastIndexByte(s, ')'); end >= fn {
			if hasLineTerminator(s[fn:]) {
				return "", "", false
			}
			return s[:end+1], s[end+1:], true
		}
	}
	if hasLineTerminator(s) {
		return "", "", false
	}
	w := 0
	for w < len(s) && chars.IsWord(int(s[w])) {
		w++
	}
	return s[:w], s[w:], true
}

// boxFunction is the length of what /^(?:rgba?|hsla?)\s*\(/ matches at the start of s, or 0.
func boxFunction(s string) int {
	i := 0
	switch {
	case strings.HasPrefix(s, "rgb"), strings.HasPrefix(s, "hsl"):
		i = 3
	default:
		return 0
	}
	if i < len(s) && s[i] == 'a' {
		i++
	}
	i = chars.SpaceRun(s, i)
	if i < len(s) && s[i] == '(' {
		return i + 1
	}
	return 0
}

// hasLineTerminator tests /[\r  ]/.
func hasLineTerminator(s string) bool {
	return strings.ContainsAny(s, "\r  ")
}

func (db *DB) parseBoxData(s string) boxData {
	first, rest, ok := splitBox(s)
	color := "transparent"
	if ok && first != "" {
		color = chars.Trim(first)
	}
	title, hasTitle := "", false
	if ok && rest != "" {
		title, hasTitle = chars.Trim(rest), true
	}
	if !isCSSColor(color) {
		color = "transparent"
		title, hasTitle = chars.Trim(s), true
	}
	if !hasTitle {
		return boxData{color: color}
	}
	t := extractWrap(title)
	return boxData{t.text, color, t.wrap}
}

func (db *DB) addBox(data boxData) {
	db.currentBox = &Box{Name: data.text, Wrap: db.wrapOr(data.wrap), Fill: data.color}
	db.Boxes = append(db.Boxes, db.currentBox)
}

// description is the label a participant statement gives, Pele's { text, wrap?, type? }.
type description struct {
	text string
	wrap int
}

func (db *DB) addActor(id, name string, desc *description, typ string, hasType bool, metadata *string) {
	assignedBox := db.currentBox
	var doc *yaml.Map
	if metadata != nil {
		src := "{\n" + *metadata + "\n}"
		if strings.Contains(*metadata, "\n") {
			src = *metadata + "\n"
		}
		parsed, err := yaml.Parse(src)
		if err != nil {
			panic(failure{diagram.NewError(`Config of participant "`+id+`" is not valid YAML. `+err.Error(), diagram.Syntax, "sequence")})
		}
		if m, ok := parsed.(*yaml.Map); ok {
			doc = m
		}
	}
	if docType, ok := doc.Get("type"); ok && docType != nil {
		typ, hasType = yaml.String(docType), true
	}
	if docAlias, ok := doc.Get("alias"); ok && yaml.Truthy(docAlias) && (desc == nil || desc.text == name) {
		w := unset
		if desc != nil {
			w = desc.wrap
		}
		desc = &description{yaml.String(docAlias), w}
	}

	if old := db.actorIndex[id]; old != nil {
		if db.currentBox != nil && old.Box != nil && db.currentBox != old.Box {
			fail("A same participant should only be defined in one Box: " + old.Name + " can't be in '" +
				boxName(old.Box) + "' and in '" + boxName(db.currentBox) + "' at the same time.")
		}
		if old.Box != nil {
			assignedBox = old.Box
		} else {
			assignedBox = db.currentBox
		}
		old.Box = assignedBox
		if name == old.Name && desc == nil {
			return
		}
	}
	// A participant that is only mentioned in a message has no kind and takes its name as
	// its label.
	if desc == nil || !hasType {
		desc = &description{text: name}
	}
	if !hasType {
		typ = "participant"
	}
	a := &Actor{ID: id, Box: assignedBox, Name: name, Description: desc.text, Wrap: db.wrapOr(desc.wrap),
		PrevActor: db.prevActor, Type: typ}
	if old := db.actorIndex[id]; old != nil {
		// Map.set keeps the key where it was.
		for i, x := range db.Actors {
			if x == old {
				db.Actors[i] = a
			}
		}
	} else {
		db.Actors = append(db.Actors, a)
	}
	db.actorIndex[id] = a
	if db.prevActor != "" {
		if prev := db.actorIndex[db.prevActor]; prev != nil {
			prev.NextActor = id
		}
	}
	if db.currentBox != nil {
		db.currentBox.ActorKeys = append(db.currentBox.ActorKeys, id)
	}
	db.prevActor = id
}

// boxName is a box's name as a template string writes it: "undefined" for none.
func boxName(b *Box) string {
	if b.Name == "" {
		return "undefined"
	}
	return b.Name
}

func (db *DB) addSignal(from, to string, message *seqText, typ int, activate bool, central int) {
	// Mermaid recounts every earlier message here; a running count per participant says the
	// same.
	if typ == ActiveEnd {
		count := 0
		if from != "" {
			count = db.active[from]
		}
		if count < 1 {
			who := from
			if from == "" {
				who = "undefined"
			}
			fail("Trying to inactivate an inactive participant (" + who + ")")
		}
		db.active[from] = count - 1
	} else if typ == ActiveStart && from != "" {
		db.active[from]++
	}
	m := &Message{ID: strconv.Itoa(len(db.Messages)), From: from, To: to, Type: typ, Activate: activate,
		CentralConnection: central}
	w := unset
	if message != nil {
		m.Text = message.text
		w = message.wrap
	}
	m.Wrap = db.wrapOr(w)
	db.Messages = append(db.Messages, m)
}

func (db *DB) addNote(actor string, pair []string, placement int, message seqText) {
	wrap := db.wrapOr(message.wrap)
	db.Notes = append(db.Notes, &Note{Actor: actor, Actors: pair, Placement: placement, Message: message.text, Wrap: wrap})
	from, to := actor, actor
	if pair != nil {
		from, to = pair[0], pair[1]
	}
	db.Messages = append(db.Messages, &Message{ID: strconv.Itoa(len(db.Messages)), From: from, To: to,
		Text: message.text, Wrap: wrap, Type: NoteType, Placement: placement})
}

// addLinks reads a links statement's JSON. Mermaid logs and ignores text that is not JSON.
// It also undoes the &amp; and &equals; its sanitizer writes; Pele has no sanitizer, and a
// `;` cannot occur in the text.
func (db *DB) addLinks(id string, text seqText) {
	a := db.actorIndex[id]
	if a == nil {
		return
	}
	if v, ok := parseJSON(text.text); ok {
		entriesOf(v, a.Links.Set)
	}
}

// addALink reads a link statement, as Pele's addALink does: the label before an @ and the
// address after it, cut by code units as JavaScript's slice cuts them.
func (db *DB) addALink(id string, text seqText) {
	a := db.actorIndex[id]
	if a == nil {
		return
	}
	s := text.text
	sep := strings.IndexByte(s, '@')
	sep16 := -1
	if sep >= 0 {
		sep16 = chars.Len16(s[:sep])
	}
	key := chars.Trim(slice16(s, 0, sep16-1))
	value := chars.Trim(slice16(s, sep16+1, chars.Len16(s)))
	a.Links.Set(key, value)
}

// slice16 is s.slice(from, to) in code units, with JavaScript's negative ends counted from
// the end.
func slice16(s string, from, to int) string {
	n := chars.Len16(s)
	if from < 0 {
		from = max(0, n+from)
	}
	if to < 0 {
		to = max(0, n+to)
	}
	from, to = min(from, n), min(to, n)
	if from >= to {
		return ""
	}
	u := units16(s)
	return decode16(u[from:to])
}

func (db *DB) addProperties(id string, text seqText) {
	a := db.actorIndex[id]
	if a == nil {
		return
	}
	if v, ok := parseJSON(text.text); ok {
		entriesOf(v, a.Properties.Set)
	}
}

// addDetails keeps the id of the page element Mermaid would read a participant's properties
// and links from.
func (db *DB) addDetails(id string, text seqText) {
	if a := db.actorIndex[id]; a != nil {
		a.Details, a.HasDetails = text.text, true
	}
}

// apply walks the nested statement lists, as Pele's apply does: Mermaid recurses, an
// explicit stack takes any depth.
func (db *DB) apply(root item) {
	var lists [][]item
	var at []int
	var it item = root
	have := true
	for {
		if have {
			switch v := it.(type) {
			case []item:
				lists = append(lists, v)
				at = append(at, 0)
			case *stmt:
				db.applyOne(v)
			}
		}
		depth := len(lists) - 1
		if depth < 0 {
			return
		}
		if list := lists[depth]; at[depth] < len(list) {
			it = list[at[depth]]
			at[depth]++
			have = true
		} else {
			lists = lists[:depth]
			at = at[:depth]
			have = false
		}
	}
}

func (db *DB) applyOne(p *stmt) {
	actor := p.actor
	switch p.typ {
	case "sequenceIndex":
		n := &Numbering{Visible: p.visible}
		if p.hasIndex {
			n.Start, n.Step, n.Given = p.index, p.step, true
		}
		db.Messages = append(db.Messages, &Message{ID: strconv.Itoa(len(db.Messages)), Numbering: n, Type: p.signalType})
	case "addParticipant":
		db.addActor(actor, actor, p.description, p.draw, p.draw != "", p.config)
	case "createParticipant":
		if db.actorIndex[actor] != nil {
			fail("It is not possible to have actors with the same id, even if one is destroyed before the next is created. Use 'AS' aliases to simulate the behavior")
		}
		db.lastCreated = actor
		db.addActor(actor, actor, p.description, p.draw, p.draw != "", p.config)
		db.Created = setPosition(db.Created, actor, len(db.Messages))
	case "destroyParticipant":
		db.lastDestroyed = actor
		db.Destroyed = setPosition(db.Destroyed, actor, len(db.Messages))
	case "activeStart", "activeEnd", "centralConnection", "centralConnectionReverse":
		db.addSignal(actor, "", nil, p.signalType, false, 0)
	case "addNote":
		db.addNote(actor, p.pair, p.placement, *p.text)
	case "addLinks":
		db.addLinks(actor, *p.text)
	case "addALink":
		db.addALink(actor, *p.text)
	case "addProperties":
		db.addProperties(actor, *p.text)
	case "addDetails":
		db.addDetails(actor, *p.text)
	case "addMessage":
		if db.lastCreated != "" {
			if p.to != db.lastCreated {
				fail("The created participant " + db.lastCreated + " does not have an associated creating message after its declaration. Please check the sequence diagram.")
			}
			db.lastCreated = ""
		} else if db.lastDestroyed != "" {
			if p.to != db.lastDestroyed && p.from != db.lastDestroyed {
				fail("The destroyed participant " + db.lastDestroyed + " does not have an associated destroying message after its declaration. Please check the sequence diagram.")
			}
			db.lastDestroyed = ""
		}
		db.addSignal(p.from, p.to, p.text, p.signalType, p.activate, p.central)
	case "boxStart":
		db.addBox(p.box)
	case "boxEnd":
		db.currentBox = nil
	case "loopStart", "rectStart", "optStart", "altStart", "else", "parStart", "and", "criticalStart", "option", "breakStart":
		db.addSignal("", "", p.text, p.signalType, false, 0)
	case "loopEnd", "rectEnd", "optEnd", "altEnd", "parEnd", "criticalEnd", "breakEnd":
		db.addSignal("", "", nil, p.signalType, false, 0)
	}
}

// setPosition is Map.set on a list of positions.
func setPosition(list []Position, id string, index int) []Position {
	for i := range list {
		if list[i].ID == id {
			list[i].Index = index
			return list
		}
	}
	return append(list, Position{id, index})
}
