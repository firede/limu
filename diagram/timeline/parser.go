package timeline

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// statement is what a statement may start with, as Mermaid's errors list it.
const statement = "'EOF', 'SPACE', 'NEWLINE', 'title', 'acc_title', 'acc_descr', 'acc_descr_multiline_value', 'section', 'period', 'event'"

// parseTimeline parses a timeline diagram's text into m, as Pele's parseTimeline does.
func parseTimeline(src string, m *Model) error {
	t := tokenize(src)
	fail := func(i int, expected string) error {
		got := "1"
		if t.Types[i] != tEND {
			got = tokenNames[t.Types[i]]
		}
		return common.SyntaxError("timeline", src, t.Starts[i], "Expecting "+expected+", got '"+got+"'", false)
	}

	head := t.Types[0]
	if head != tTimeline && head != tTimelineLR && head != tTimelineTD {
		return fail(0, "'timeline', 'timeline_lr', 'timeline_td'")
	}
	if head == tTimelineLR {
		m.setDirection("LR")
	} else if head == tTimelineTD {
		m.setDirection("TD")
	}

	for i := 1; ; i++ {
		text := t.Texts[i]
		switch t.Types[i] {
		case tNEWLINE:
		case tTitle:
			m.setDiagramTitle(chars.DropUnits(text, 6))
		case tAccTitle:
			i++
			m.setAccTitle(chars.Trim(t.Texts[i]))
		case tAccDescr:
			i++
			m.setAccDescription(chars.Trim(t.Texts[i]))
		case tAccDescrMultilineValue:
			m.setAccDescription(chars.Trim(text))
		case tSection:
			m.addSection(chars.DropUnits(text, 8))
		case tPeriod:
			m.addTask(text, "")
		case tEvent:
			if err := m.addEvent(chars.DropUnits(text, 2)); err != nil {
				return err
			}
		case tEOF:
			return nil
		default:
			return fail(i, statement)
		}
	}
}

// setDiagramTitle is the TimelineDb's setDiagramTitle.
func (m *Model) setDiagramTitle(text string) { m.Title = text }

// setAccTitle sets the accessible title, without the white space it starts with.
func (m *Model) setAccTitle(text string) { m.AccTitle = chars.TrimStart(text) }

// setAccDescription sets the accessible description, each line break with the white space
// after it, line breaks included, made one line break.
func (m *Model) setAccDescription(text string) { m.AccDescr = chars.DropIndents(text) }

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
		return common.NewError("An event needs a time period before it.", diagram.Semantic, "timeline")
	}
	p := m.Periods[len(m.Periods)-1]
	p.Events = append(p.Events, event)
	return nil
}
