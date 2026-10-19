// Package journey parses Mermaid's user journey diagrams and lays them out, as Pele's
// src/diagrams/journey does: a lexer and a parser that make the decisions Mermaid's Jison
// grammar makes, the Model they fill, and Layout, which lays a model out as a scene.
//
// Offsets into the source are bytes of the Go string; the offsets and columns in errors are
// UTF-16 code units, as Pele's are.
package journey

import (
	"slices"

	"github.com/firede/limu/diagram"
)

// Task is a step of a journey, Pele's JourneyTask.
type Task struct {
	Task string
	// Score is the score as written: any number, or NaN when it is not one.
	Score  float64
	People []string
	// Section is the name of the section in force when the task was declared, "" before
	// the first one.
	Section string
}

// Model is a parsed user journey diagram, Pele's JourneyModel. A string Pele may leave
// undefined is empty here where it is.
type Model struct {
	Title    string
	AccTitle string
	AccDescr string
	Sections []string
	Tasks    []Task
}

// journeyActors is everyone named on a task, once each, in code unit order, as Pele's journeyActors
// gives them.
func journeyActors(tasks []Task) []string {
	seen := map[string]bool{}
	var names []string
	for _, task := range tasks {
		for _, name := range task.People {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	// JavaScript's sort compares strings by UTF-16 code units, which orders characters
	// above the Basic Multilingual Plane before those from U+E000 up, where UTF-8 orders
	// them after.
	slices.SortStableFunc(names, compare16)
	return names
}

// compare16 compares two strings by their UTF-16 code units, as JavaScript does.
func compare16(a, b string) int {
	ua, ub := units16(a), units16(b)
	for i := 0; i < len(ua) && i < len(ub); i++ {
		if ua[i] != ub[i] {
			if ua[i] < ub[i] {
				return -1
			}
			return 1
		}
	}
	return len(ua) - len(ub)
}

func units16(s string) []uint16 {
	out := make([]uint16, 0, len(s))
	for _, r := range s {
		if r > 0xffff {
			r -= 0x10000
			out = append(out, uint16(0xd800+(r>>10)), uint16(0xdc00+(r&0x3ff)))
		} else {
			out = append(out, uint16(r))
		}
	}
	return out
}

// Parse parses a user journey diagram's text, as the parse of Pele's journey diagram does:
// source is what diagram.Prepared.Source gives, and title the title its front matter gives,
// "" for none. An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	db := &db{Model: Model{Title: title}}
	if err := parse(source, db); err != nil {
		return nil, err
	}
	return &db.Model, nil
}

// ParsePrepared parses a prepared user journey diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}
