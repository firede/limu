// Package layout holds what is shared by the layouts of diagrams: the options that change
// where things are. A change of them lays a diagram out again; a change of style does not.
package layout

import "github.com/firede/limu/text"

// Routing is how edges run between the things they join.
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
	// Snap rounds where things are to whole multiples of it, for a backend that draws on a
	// grid of pixels. 0 leaves them as they fall.
	Snap    float64
	Routing Routing
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
