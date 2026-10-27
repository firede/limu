// Package gantt parses Mermaid's gantt charts and lays them out, as Pele's
// src/diagrams/gantt does: a lexer and a parser that accept what Mermaid's gantt.jison
// accepts, a DB that schedules the tasks in local time as Mermaid's ganttDb does with dayjs,
// and Layout, which draws the chart on a time axis as d3 would lay it out.
//
// Dates are computed as JavaScript computes them in local time (see internal/jsdate), in the
// zone layout.Options.Location gives, and against the time layout.Options.Now gives, so that
// a host decides both.
//
// The scene's items carry the classes Pele gives the chart's elements, for a host's CSS. A
// task ("gantt.task") has, in this order, "active" or "done", "crit", "milestone",
// "clickable" when a click line gives it a link or a callback, and "unscheduled" when its
// dates could not be worked out. A section's band ("gantt.section") has "section-N", N the
// section's place among the colors numberSectionStyles sets, counted from 0.
package gantt

import (
	"github.com/firede/limu/diagram/internal/common"
	"math"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// Raw is what a task's data said, before it is scheduled.
type Raw struct {
	Data string
	// Start is the start as written, if HasStart; with none, a task starts where the task
	// before it ends.
	Start    string
	HasStart bool
	End      string
}

// moment is a time of a task as Pele holds one: undefined, or a date, which may be invalid,
// as a JavaScript time value.
type moment struct {
	ms      float64
	defined bool
}

// at is a defined moment at the time value ms, as Pele assigns a Date to a task's time.
func at(ms float64) moment { return moment{ms, true} }

// value is the moment as a number, as +date gives it: NaN when it is undefined. Pele's
// getStartDate and getEndDate compare tasks' times so.
func (m moment) value() float64 {
	if !m.defined {
		return math.NaN()
	}
	return m.ms
}

// Time is the moment as a time.Time, if it is a valid date. Pele has no such function; it
// gives the Date itself.
func (m moment) time(loc *time.Location) (time.Time, bool) {
	if !m.defined || math.IsNaN(m.ms) {
		return time.Time{}, false
	}
	return time.UnixMilli(int64(m.ms)).In(loc), true
}

// Task is a task of a chart, Pele's GanttTask.
type Task struct {
	ID string
	// Task is the task's label, under the name Mermaid gives it.
	Task    string
	Section string
	// The moments scheduling gives the task, and the end of its bar when excluded days pushed
	// its end out.
	start, end, renderEnd moment
	ManualEndTime         bool
	Processed             bool
	Active, Done, Crit    bool
	Milestone, Vert       bool
	// Order is the task's row in source order; -1 for a vertical marker, which takes no row.
	Order   int
	Classes []string
	// PrevTaskID is the id of the task before it, if HasPrev.
	PrevTaskID string
	HasPrev    bool
	Raw        Raw

	loc *time.Location
}

// Start is when the task starts, once scheduled; false when it has no valid start. It is
// Pele's GanttTask.startTime.
func (t *Task) Start() (time.Time, bool) { return t.start.time(t.loc) }

// End is when the task ends, once scheduled, as Pele's GanttTask.endTime.
func (t *Task) End() (time.Time, bool) { return t.end.time(t.loc) }

// RenderEnd is where the task's bar ends, when excluded days pushed its end out, as Pele's
// GanttTask.renderEndTime.
func (t *Task) RenderEnd() (time.Time, bool) { return t.renderEnd.time(t.loc) }

// Parse parses a gantt chart's text, as the parse of Pele's gantt chart does: source is what
// diagram.Prepared.Source gives, config its configuration, and title the title its front
// matter gives, "" for none. An error is a *diagram.Error.
func Parse(source string, config diagram.Config, title string) (*DB, error) {
	db := newDB()
	if title != "" {
		db.Title = title
	}
	if g, ok := config["gantt"].(diagram.Config); ok {
		if mode, ok := g["displayMode"].(string); ok {
			db.DisplayMode = mode
		}
	}
	if err := parseInto(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared gantt chart, giving Parse what Pele's parse is given.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return common.ParsePrepared(p, false, func(src string) (*DB, error) {
		return Parse(src, p.Config, p.Title)
	})
}

// trim is JavaScript's trim, as Pele's String.prototype.trim calls give it.
func trim(s string) string { return chars.Trim(s) }
