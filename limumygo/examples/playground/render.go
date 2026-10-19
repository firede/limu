package main

import (
	"strings"
	"time"

	"github.com/firede/limu"
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/limumygo"
	"github.com/firede/limu/scene"
)

// editDelay is how long after the last edit the text is laid out again, so that a diagram
// is not laid out at each key typed.
const editDelay = 150 * time.Millisecond

// layoutOptions are what the playground lays a diagram out with: its text measured in the
// fonts limumygo draws it in, and the width of the pane it is shown in, which a diagram
// that can draw itself narrower, as one that turns from running across to running down,
// fits.
// The look is not among them: it changes how a diagram is drawn, not where anything is.
func layoutOptions(width float64) *layout.Options {
	return &layout.Options{Measurer: limumygo.DefaultFonts, MaxWidth: width}
}

// A job is a laying out of the text: the text at a version of the editor's buffer, and the
// width it is laid out in. Its stamp tells the newest job from those before it.
type job struct {
	stamp   uint64
	version uint64
	src     string
	width   float64
}

// A result is what a job made: the diagram's scene, or the error its text is in, or
// neither when the text is blank.
type result struct {
	job
	typ   diagram.Type
	scene *scene.Scene
	err   error
	blank bool
	took  time.Duration
}

// run parses the job's text and lays it out. It may run on any goroutine, as a
// limu.Diagram may be laid out on any.
func (j job) run() result {
	r := result{job: j}
	if strings.TrimSpace(j.src) == "" {
		r.blank = true
		return r
	}
	start := time.Now()
	d, err := limu.Parse(j.src, diagram.Options{})
	if err == nil {
		r.typ = d.Type()
		r.scene, err = d.Layout(layoutOptions(j.width))
	}
	r.err = err
	r.took = time.Since(start)
	return r
}

// A planner decides when the text is laid out again. An edit is laid out editDelay after
// the last of a run of edits, and a new text, as a file opened, at once; a change of the
// pane's width is laid out at once too, as it changes what fits. A change of appearance is
// not among what it is told, as it only draws the scene again.
//
// Each job it starts has a stamp one past the last, and only the result of the newest is
// shown (current): a result that comes back after a newer job started is stale, of a text
// or a width the window no longer has, and is dropped.
type planner struct {
	delay time.Duration

	seen     uint64    // the version of the text the last frame had
	editedAt time.Time // when the text came to that version
	urgent   bool      // lay the next change of the text out at once

	stamp   uint64 // of the newest job, 0 before the first
	version uint64 // what the newest job lays out
	width   float64
}

// next is called in each frame with the version of the text, the width of the pane, and
// the time of the frame. It reports whether to start a job now, with the stamp next took
// for it; and if not, how long until one is due, or 0 when none is. A width of 0 is a pane
// not yet laid out, which waits.
func (p *planner) next(version uint64, width float64, now time.Time) (start bool, wait time.Duration) {
	if version != p.seen {
		p.seen, p.editedAt = version, now
	}
	if width <= 0 {
		return false, 0
	}
	if p.stamp > 0 {
		if version == p.version && width == p.width {
			return false, 0
		}
		if version != p.version && !p.urgent {
			if d := p.delay - now.Sub(p.editedAt); d > 0 {
				return false, d
			}
		}
	}
	p.urgent = false
	p.stamp++
	p.version, p.width = version, width
	return true, 0
}

// current reports whether a result's stamp is the newest job's.
func (p *planner) current(stamp uint64) bool { return stamp == p.stamp }
