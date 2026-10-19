package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/gallery"
)

// menuExamples is how many of a type's examples the Samples menu lists, the featured ones
// first; the others are left for a page of examples.
const menuExamples = 8

// firstSample is the text the playground starts with when it has none saved: the first
// featured example of flowcharts.
func firstSample() string {
	if t, ok := gallery.Find(string(diagram.Flowchart)); ok && len(t.Examples) > 0 {
		return t.Examples[0].Source
	}
	return "flowchart LR\n  A[Text] --> B(Scene) --> C{Backend}\n"
}

// samplesMenu fills the Samples menu: a submenu for each type of diagram, which lists its
// first examples, and calls load with the text of the one chosen.
func samplesMenu(m *ui.Menu, load func(src string)) {
	for _, t := range gallery.All() {
		title := t.Title
		if t.Beta {
			title += " (beta)"
		}
		m.Submenu(title, func(sm *ui.Menu) {
			for i, ex := range t.Examples {
				if i == menuExamples {
					break
				}
				if sm.Item(ex.Title).Chosen() {
					load(ex.Source)
				}
			}
		})
	}
}

// typeTitle is what the status bar calls a type of diagram: its title in the gallery, as
// "Entity relationship", or its name.
func typeTitle(t diagram.Type) string {
	if g, ok := gallery.Find(string(t)); ok {
		return g.Title
	}
	return string(t)
}
