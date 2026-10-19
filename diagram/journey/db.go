package journey

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// db is what the parser fills, Pele's JourneyDb.
type db struct {
	Model
	currentSection string
}

func (d *db) setDiagramTitle(text string) { d.Title = text }

// setAccTitle drops the white space the title starts with, as /^\s+/ does.
func (d *db) setAccTitle(text string) { d.AccTitle = chars.TrimStart(text) }

// setAccDescription drops the indentation of each line after the first, as
// replace(/\n\s+/g, '\n') does: \s takes line breaks too, so blank lines go with it.
func (d *db) setAccDescription(text string) {
	if !strings.Contains(text, "\n") {
		d.AccDescr = text
		return
	}
	var b strings.Builder
	for i := 0; i < len(text); {
		if text[i] != '\n' {
			b.WriteByte(text[i])
			i++
			continue
		}
		end := chars.SpaceRun(text, i+1)
		if end == i+1 {
			b.WriteByte('\n')
			i++
			continue
		}
		b.WriteByte('\n')
		i = end
	}
	d.AccDescr = b.String()
}

func (d *db) addSection(text string) {
	d.currentSection = text
	d.Sections = append(d.Sections, text)
}

// addTask adds a task, as Pele's addTask does: data is the rest of the line from its first
// colon, ":score" or ":score:actor, actor".
func (d *db) addTask(task, data string) {
	pieces := strings.Split(data[1:], ":")
	people := []string{}
	if len(pieces) > 1 {
		for _, name := range strings.Split(pieces[1], ",") {
			people = append(people, chars.Trim(name))
		}
	}
	d.Tasks = append(d.Tasks, Task{Task: task, Score: chars.Number(pieces[0]), People: people, Section: d.currentSection})
}
