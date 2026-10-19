package timeline

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// statement is what a statement may start with, as Mermaid's errors list it.
const statement = "'EOF', 'SPACE', 'NEWLINE', 'title', 'acc_title', 'acc_descr', 'acc_descr_multiline_value', 'section', 'period', 'event'"

// parseTimeline parses a timeline diagram's text into m, as Pele's parseTimeline does.
func parseTimeline(src string, m *Model) error {
	t := tokenize(src)
	fail := func(i int, expected string) error {
		got := "1"
		if t.types[i] != tEND {
			got = tokenNames[t.types[i]]
		}
		return diagram.SyntaxError("timeline", src, t.starts[i], "Expecting "+expected+", got '"+got+"'", false)
	}

	head := t.types[0]
	if head != tTimeline && head != tTimelineLR && head != tTimelineTD {
		return fail(0, "'timeline', 'timeline_lr', 'timeline_td'")
	}
	if head == tTimelineLR {
		m.setDirection("LR")
	} else if head == tTimelineTD {
		m.setDirection("TD")
	}

	for i := 1; ; i++ {
		text := t.texts[i]
		switch t.types[i] {
		case tNEWLINE:
		case tTitle:
			m.setDiagramTitle(afterKeyword(text, 5))
		case tAccTitle:
			i++
			m.setAccTitle(chars.Trim(t.texts[i]))
		case tAccDescr:
			i++
			m.setAccDescription(chars.Trim(t.texts[i]))
		case tAccDescrMultilineValue:
			m.setAccDescription(chars.Trim(text))
		case tSection:
			m.addSection(afterKeyword(text, 7))
		case tPeriod:
			m.addTask(text, "")
		case tEvent:
			if err := m.addEvent(afterKeyword(text, 1)); err != nil {
				return err
			}
		case tEOF:
			return nil
		default:
			return fail(i, statement)
		}
	}
}

// afterKeyword is a token's text after its first n bytes, a keyword in ASCII, and the one
// character of white space that follows it: what Pele's substring(n + 1) leaves, as that
// character is in the Basic Multilingual Plane, one code unit.
func afterKeyword(text string, n int) string {
	_, size := chars.UnitAt(text, n)
	return text[n+size:]
}

// setDiagramTitle is the TimelineDb's setDiagramTitle.
func (m *Model) setDiagramTitle(text string) { m.Title = text }

// setAccTitle sets the accessible title, without the white space it starts with.
func (m *Model) setAccTitle(text string) { m.AccTitle = chars.TrimStart(text) }

// setAccDescription sets the accessible description, each line break with the white space
// after it, line breaks included, made one line break, as /\n\s+/g replaced with "\n" does.
func (m *Model) setAccDescription(text string) {
	var b strings.Builder
	for i := 0; i < len(text); {
		if text[i] != '\n' {
			b.WriteByte(text[i])
			i++
			continue
		}
		b.WriteByte('\n')
		i++
		for i < len(text) {
			c, size := chars.UnitAt(text, i)
			if !chars.IsSpace(c) {
				break
			}
			i += size
		}
	}
	m.AccDescr = b.String()
}

func (m *Model) setDirection(direction string) { m.Direction = direction }

// addSection starts a section, as the TimelineDb's addSection does.
func (m *Model) addSection(text string) {
	m.currentSection = text
	m.Sections = append(m.Sections, text)
}

// addTask adds a time period, with an event if one is given, as the TimelineDb's addTask
// does.
func (m *Model) addTask(period, event string) {
	p := &Period{Text: period, Events: []string{}, Section: m.currentSection, SectionIndex: len(m.Sections) - 1}
	if event != "" {
		p.Events = append(p.Events, event)
	}
	m.Periods = append(m.Periods, p)
}

// addEvent adds an event to the last time period, as the TimelineDb's addEvent does.
func (m *Model) addEvent(event string) error {
	if len(m.Periods) == 0 {
		// Mermaid fails here too, with a TypeError.
		return diagram.NewError("An event needs a time period before it.", diagram.Semantic, "timeline")
	}
	p := m.Periods[len(m.Periods)-1]
	p.Events = append(p.Events, event)
	return nil
}
