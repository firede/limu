// Package common holds what the diagrams share beyond layout and drawing, as Pele's
// diagrams/common does:
//
//   - the room about a diagram (Padding), and fitting a diagram to the width the host has
//     (Tighten, TurnToFit, FitWidth), with the curve the host's routing asks for
//     (RoutingCurve);
//   - laying out labels: plain and fitted ones (fit.go), text cut to fit a room and
//     one-line labels (fittext.go: FitText, Line, LineOf), labels taken literally
//     (literallabel.go), and a diagram's title (title.go);
//   - reading generic types (generics.go) and class lists (classes.go: ClassNames,
//     TrimDefault);
//   - the icons a host can draw, and the glyphs drawn for them (icons.go);
//   - the lexer and reader of the Langium grammars, with the token types they share
//     (tokens.go: Tokenize, Reader, Keyword, AccDescr and the rest);
//   - the scanning helpers of the hand-written Jison lexers (scan.go: KeywordAt, WordAt,
//     Accessibility and the rest);
//   - the indented outlines of kanban boards and mindmaps (outline.go).
package common

import (
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
)

// Padding is the room about a diagram, as Pele's default padding.
const Padding = 8

// RoutingCurve is the curve, by Mermaid's name, that the host's routing asks edges to be
// drawn with: "step" for layout.Orthogonal, "linear" for layout.Straight, and "" for
// layout.Auto, which leaves the diagram's own curve. Pele has no such setting; it is limu's.
func RoutingCurve(r layout.Routing) string {
	switch r {
	case layout.Orthogonal:
		return "step"
	case layout.Straight:
		return "linear"
	}
	return ""
}

// RunsAcross reports whether a direction runs across, as Pele's runsAcross does.
func RunsAcross(dir string) bool { return dir == "LR" || dir == "RL" }

// Tighten lays a diagram out with room to spare, and tightly instead when that does not fit
// the width the host has and tight is narrower, as Pele's tighten does.
func Tighten(o *layout.Options, lay func(tight bool) *scene.Scene) *scene.Scene {
	loose := lay(false)
	if !(o.MaxWidth > 0) || loose.Width <= o.MaxWidth {
		return loose
	}
	if tight := lay(true); tight.Width < loose.Width {
		return tight
	}
	return loose
}

// TurnToFit lays a diagram out across, and down instead when the host allows it, the width
// it has is under the breakpoint, across does not fit, and down is narrower, as Pele's
// turnToFit does.
func TurnToFit(o *layout.Options, lay func(down bool) *scene.Scene) *scene.Scene {
	across := lay(false)
	if o.KeepDirection || !(o.MaxWidth > 0) || across.Width <= o.MaxWidth {
		return across
	}
	breakpoint := 640.0
	if o.DirectionBreakpoint > 0 {
		breakpoint = o.DirectionBreakpoint
	}
	if o.MaxWidth >= breakpoint {
		return across
	}
	if down := lay(true); down.Width < across.Width {
		return down
	}
	return across
}

// FitWidth lays out a chart whose plot can be any width so that the whole of it fits the
// width the host has, as Pele's fitWidth does: what surrounds the plot is only known once it
// is laid out, so the plot gives up the overflow and is laid out again, at most three times
// more. narrow tells the chart it did not fit at its own width.
func FitWidth(o *layout.Options, natural, least float64, lay func(plot float64, narrow bool) *scene.Scene) *scene.Scene {
	sc := lay(natural, false)
	available := o.MaxWidth
	if !(available > 0) || sc.Width <= available {
		return sc
	}
	floor := min(least, natural)
	plot := natural
	sc = lay(plot, true)
	for round := 0; round < 3 && sc.Width > available && plot > floor; round++ {
		plot = max(floor, plot-(sc.Width-available))
		sc = lay(plot, true)
	}
	return sc
}

// TitleRoom is the width a title may take before it wraps, as Pele's titleRoom gives it.
func TitleRoom(o *layout.Options) float64 {
	if o.MaxWidth > 0 {
		return max(80, o.MaxWidth-2*Padding)
	}
	return 4000
}

// CapWidth is the width of a chart drawn to a set width: its own, or the host's when that is
// smaller, with around what the chart adds outside it, as Pele's capWidth gives it.
func CapWidth(o *layout.Options, natural, least, around float64) float64 {
	if !(o.MaxWidth > 0) {
		return natural
	}
	return max(min(least, natural), min(natural, o.MaxWidth-around))
}
