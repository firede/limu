package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/gallery"
)

// firstSample is the text the playground starts with when it has none of its own: the
// first featured example of flowcharts.
func firstSample() string {
	if t, ok := gallery.Find(string(diagram.Flowchart)); ok && len(t.Examples) > 0 {
		return t.Examples[0].Source
	}
	return "flowchart LR\n  A[Text] --> B(Scene) --> C{Backend}\n"
}

// examplesMenu fills the Examples menu with limu's gallery, the website's examples: a
// submenu for each type of diagram, which lists its examples, and calls load with the text
// of the one chosen.
func examplesMenu(m *ui.Menu, load func(src string)) {
	for _, t := range gallery.All() {
		title := t.Title
		if t.Beta {
			title += " (beta)"
		}
		m.Submenu(title, func(sm *ui.Menu) {
			for _, ex := range t.Examples {
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
