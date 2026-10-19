// Package layout holds the options that change where things are: the measurer and the size
// of text, the width the host has, a grid to snap to, how edges are routed, and the like. A
// change of them lays a diagram out again; a change of style does not.
package layout

import (
	"time"

	"github.com/firede/limu/text"
)

// Routing is how edges run between the things they join. It is a setting of the diagrams on
// the layered layout: flowcharts, swimlane, agent flow, state, class, ER, requirement and use
// case diagrams. Auto leaves them to the text's curve, or to the curve Pele draws where the
// text has none, and Orthogonal and Straight take the place of the step and linear curves.
// The other diagrams route their edges as Pele does and pay it no heed.
type Routing uint8

const (
	// Auto routes edges as the diagram's text asks, with its curve setting.
	Auto Routing = iota
	// Orthogonal routes edges in straight runs across and down, turning at right angles.
	Orthogonal
	// Straight routes edges in straight lines from corner to corner.
	Straight
)

// Options are what a layout takes besides the diagram.
type Options struct {
	Measurer text.Measurer // nil is text.Arial
	FontSize float64       // of text with no size of its own; 0 is 16
	// Icons reports whether the host can draw an icon by name; labels leave room for those
	// it can.
	Icons func(name string) bool
	// MaxWidth is the width the host has for the diagram; a diagram that can draw itself
	// narrower does. 0 is no limit.
	MaxWidth float64
	// KeepDirection keeps a diagram that runs across from being laid out running down when it
	// does not fit MaxWidth; it is, by default, when MaxWidth is under DirectionBreakpoint
	// (0 is 640) and running down is narrower.
	KeepDirection       bool
	DirectionBreakpoint float64

	// NoLinks and NoImages leave out the links and the images a diagram's text asks for.
	// LinkSchemes and ImageSchemes are the schemes they may use besides a relative address:
	// nil is http, https, mailto and tel for links, and http and https for images.
	NoLinks, NoImages         bool
	LinkSchemes, ImageSchemes []string
	// Padding is the room about a diagram, Pele's padding option: nil is Pele's 8, and a
	// pointer to 0 is none. Set it with new(12.0). A host that wants no room about some
	// diagram and the default about others tells them apart this way, since 0 is a padding.
	Padding *float64
	// Snap rounds where things are to whole multiples of it, for a backend that draws on a
	// grid of pixels. 0 leaves them as they fall. limu.Layout and Diagram.Layout snap the
	// scene; a diagram package's own Layout leaves it to its caller, who snaps the scene
	// with Scene.Snap.
	Snap float64
	// Routing is how edges run, in the diagrams laid out in ranks.
	Routing Routing

	// Now is the time a diagram that places things in time takes as the present: where a
	// gantt chart draws today, and where a task with no start begins. The zero time is
	// time.Now().
	Now time.Time
	// Location is the time zone such a diagram reads and writes dates in, as JavaScript
	// does in the machine's: the days that tasks start on, the ticks of its axis. nil is
	// time.Local.
	Location *time.Location
}

// Pad is the options' Padding, or 8, as Pele takes options.padding ?? 8.
func (o *Options) Pad() float64 {
	if o == nil || o.Padding == nil {
		return 8
	}
	return *o.Padding
}

// Clock is the options' Now, or time.Now() when it is the zero time.
func (o *Options) Clock() time.Time {
	if o == nil || o.Now.IsZero() {
		return time.Now()
	}
	return o.Now
}

// Zone is the options' Location, or time.Local.
func (o *Options) Zone() *time.Location {
	if o == nil || o.Location == nil {
		return time.Local
	}
	return o.Location
}

// Measure is the options' measurer, or text.Arial.
func (o *Options) Measure() text.Measurer {
	if o == nil || o.Measurer == nil {
		return text.Arial
	}
	return o.Measurer
}

// Size is the options' font size, or 16.
func (o *Options) Size() float64 {
	if o == nil || o.FontSize <= 0 {
		return 16
	}
	return o.FontSize
}
