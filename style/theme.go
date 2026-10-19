// Package style says how a diagram looks, at three levels.
//
// A Theme is the first: a handful of colors and measures every diagram is drawn from, as
// Pele's --pele-* variables are. A diagram's parts name its colors by Token (Surface,
// Border, Series(2)…) rather than by value, so a change of theme only redraws.
//
// Overrides are the second: a Style for each role of the parts of diagrams, such as
// "flowchart.node" or "sequence.actor", that sets some of what the theme gives.
//
// The third, drawing a part some way of one's own, belongs to the backends, as each draws
// with tools of its own; see the svg package.
//
// What changes where things are — the font, the sizes of text — is not style: it is given to
// layout, and a change of it lays the diagram out again.
package style

// A Token names a color of the theme.
type Token uint8

const (
	unset Token = iota
	none
	custom
	tokenBg
	tokenFg
	tokenMuted
	tokenLine
	tokenSurface
	tokenSurfaceAlt
	tokenBorder
	tokenAccent
	tokenCritical
	tokenSeries // and the seven after it
)

// Name is the token's name, as in the variables the svg package writes: "surface-alt",
// "series-3". It is empty for a color that is not the theme's.
func (t Token) Name() string {
	switch t {
	case tokenBg:
		return "bg"
	case tokenFg:
		return "fg"
	case tokenMuted:
		return "muted"
	case tokenLine:
		return "line"
	case tokenSurface:
		return "surface"
	case tokenSurfaceAlt:
		return "surface-alt"
	case tokenBorder:
		return "border"
	case tokenAccent:
		return "accent"
	case tokenCritical:
		return "critical"
	}
	if t >= tokenSeries && t < tokenSeries+8 {
		return "series-" + string(rune('1'+t-tokenSeries))
	}
	return ""
}

// A Theme is the colors and measures diagrams are drawn with.
type Theme struct {
	Bg         Color // behind the diagram, and behind labels on lines
	Fg         Color // text
	Muted      Color // secondary text, as the titles of groups
	Line       Color // edges and other lines
	Surface    Color // the inside of shapes
	SurfaceAlt Color // the inside of groups, and of shapes set apart
	Border     Color // the outlines of shapes
	Accent     Color // what a diagram marks out
	Critical   Color // what a diagram marks as critical
	Series     [8]Color

	Radius      float64 // of the corners of rounded boxes
	StrokeWidth float64 // of outlines and lines
}

// Default is Pele's light theme.
func Default() *Theme {
	return &Theme{
		Bg:         Hex("#fff"),
		Fg:         Hex("#1f1f1f"),
		Muted:      Hex("#6e6e6e"),
		Line:       Hex("#1f1f1f"),
		Surface:    Hex("#f3f3f3"),
		SurfaceAlt: Hex("#e6e6e6"),
		Border:     Hex("#8a8a8a"),
		Accent:     Hex("#5b7bd5"),
		Critical:   Hex("#e45756"),
		Series: [8]Color{
			Hex("#4c78a8"), Hex("#f58518"), Hex("#54a24b"), Hex("#e45756"),
			Hex("#72b7b2"), Hex("#eeca3b"), Hex("#b279a2"), Hex("#9d755d"),
		},
		Radius:      4,
		StrokeWidth: 1,
	}
}

// Color is the theme's color for a token.
func (th *Theme) Color(t Token) Color {
	switch t {
	case tokenBg:
		return th.Bg
	case tokenFg:
		return th.Fg
	case tokenMuted:
		return th.Muted
	case tokenLine:
		return th.Line
	case tokenSurface:
		return th.Surface
	case tokenSurfaceAlt:
		return th.SurfaceAlt
	case tokenBorder:
		return th.Border
	case tokenAccent:
		return th.Accent
	case tokenCritical:
		return th.Critical
	}
	if t >= tokenSeries && t < tokenSeries+8 {
		return th.Series[t-tokenSeries]
	}
	return Color{}
}
