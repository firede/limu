package style

// A Paint is what a part is filled, stroked or written with: a color of the theme, a color
// of its own, or nothing. The zero Paint is unset, and takes what is under it.
type Paint struct {
	token Token
	color Color   // when token is custom
	alpha float32 // a factor on the color's alpha; 0 is 1
	base  Token   // the color it is laid over, to be opaque, when set
}

// The colors of the theme, and None, which draws nothing.
var (
	None       = Paint{token: none}
	Bg         = Paint{token: tokenBg}
	Fg         = Paint{token: tokenFg}
	Muted      = Paint{token: tokenMuted}
	Line       = Paint{token: tokenLine}
	Surface    = Paint{token: tokenSurface}
	SurfaceAlt = Paint{token: tokenSurfaceAlt}
	Border     = Paint{token: tokenBorder}
	Accent     = Paint{token: tokenAccent}
	Critical   = Paint{token: tokenCritical}
)

// Series is the color of the nth series of a chart, of eight that repeat.
func Series(n int) Paint { return Paint{token: tokenSeries + Token((n%8+8)%8)} }

// Of paints with a color of its own.
func Of(c Color) Paint { return Paint{token: custom, color: c} }

// Alpha is the paint at a part of its opacity.
func (p Paint) Alpha(a float64) Paint {
	if a <= 0 {
		return None
	}
	if a < 1 {
		p.alpha = p.Factor() * float32(a)
	}
	return p
}

// Over is the paint laid over a color of the theme: an opaque color, the paint's color at
// its opacity mixed into base's, as CSS's color-mix does in sRGB. It is for what must hide
// what is under it in the color it would seem to have there: a title's halo over a group.
func (p Paint) Over(base Paint) Paint {
	if p.Visible() && base.token > custom {
		p.base = base.token
	}
	return p
}

// Base is the token of the color the paint is laid over, if it is.
func (p Paint) Base() Token { return p.base }

// Set reports whether the paint was set: an unset one takes what is under it.
func (p Paint) Set() bool { return p.token != unset }

// Visible reports whether the paint draws: it is set, and not None.
func (p Paint) Visible() bool { return p.token > none }

// Token is the theme's token the paint is, if it is one; Token.Name tells.
func (p Paint) Token() Token { return p.token }

// Factor is the part of its opacity the paint is drawn at.
func (p Paint) Factor() float32 {
	if p.alpha == 0 {
		return 1
	}
	return p.alpha
}

// Own is the paint's own color, when it is not one of the theme's.
func (p Paint) Own() Color { return p.color }

// Color is the color the paint draws with in a theme; its alpha is 0 when it draws nothing.
func (p Paint) Color(th *Theme) Color {
	var c Color
	switch {
	case p.token == custom:
		c = p.color
	case p.token > custom:
		c = th.Color(p.token)
	default:
		return Color{}
	}
	if p.alpha != 0 {
		c.A = uint8(float32(c.A)*p.alpha + 0.5)
	}
	if p.base != unset {
		b := th.Color(p.base)
		t := float64(c.A) / 255
		mix := func(x, y uint8) uint8 { return uint8(float64(x)*t + float64(y)*(1-t) + 0.5) }
		c = Color{mix(c.R, b.R), mix(c.G, b.G), mix(c.B, b.B), 255}
	}
	return c
}

// A Num is a measure that may be unset, and then takes what is under it.
type Num struct {
	v   float64
	set bool
}

// N is the measure v.
func N(v float64) Num { return Num{v, true} }

// Get is the measure, and whether it was set.
func (n Num) Get() (float64, bool) { return n.v, n.set }

// Or is the measure, or d when it was not set.
func (n Num) Or(d float64) float64 {
	if n.set {
		return n.v
	}
	return d
}

// A Dash is the pattern of a line's dashes and gaps, or a solid line, or unset.
type Dash struct {
	v   []float64
	set bool
}

// Dashes is a line of dashes and gaps of the lengths given in turn.
func Dashes(lengths ...float64) Dash { return Dash{lengths, true} }

// Solid is a line with no gaps.
var Solid = Dash{nil, true}

// Pattern is the lengths of the dashes and gaps; none for a solid line.
func (d Dash) Pattern() []float64 { return d.v }

// Set reports whether the dash was set.
func (d Dash) Set() bool { return d.set }

// A Cap is how the ends of a stroked line are drawn, as SVG's stroke-linecap: the ends of a
// line and of each of its dashes. The zero Cap is unset, and takes what is under it; a line
// nothing sets one for ends flat, as SVG's do.
type Cap uint8

const (
	CapButt   Cap = iota + 1 // flat, at the end
	CapRound                 // a half circle about the end
	CapSquare                // flat, half the line's width past the end
)

// Set reports whether the cap was set.
func (c Cap) Set() bool { return c != 0 }

// String is the cap's name in SVG and CSS, "butt", "round" or "square", or "" when unset.
func (c Cap) String() string {
	switch c {
	case CapButt:
		return "butt"
	case CapRound:
		return "round"
	case CapSquare:
		return "square"
	}
	return ""
}

// A Join is how a stroked line turns its corners, as SVG's stroke-linejoin. The zero Join is
// unset, and takes what is under it; a line nothing sets one for turns its corners mitred,
// as SVG's do.
type Join uint8

const (
	// JoinMiter turns a corner in a point where the line's sides meet, or bevelled where
	// SVG's miter limit of 4 would: where the point is more than four times half the line's
	// width from the corner.
	JoinMiter Join = iota + 1
	JoinRound      // an arc about the corner
	JoinBevel      // straight across, from one side's end to the other's
)

// Set reports whether the join was set.
func (j Join) Set() bool { return j != 0 }

// String is the join's name in SVG and CSS, "miter", "round" or "bevel", or "" when unset.
func (j Join) String() string {
	switch j {
	case JoinMiter:
		return "miter"
	case JoinRound:
		return "round"
	case JoinBevel:
		return "bevel"
	}
	return ""
}

// A Style is how a part is painted. What it leaves unset comes from under it.
type Style struct {
	Fill        Paint
	Stroke      Paint // outlines and lines, and the markers at their ends
	Text        Paint
	StrokeWidth Num
	Radius      Num // of the corners of a box that takes the theme's
	Dash        Dash
	Cap         Cap  // of the ends of lines and of their dashes
	Join        Join // of the corners of lines and outlines
	Opacity     Num  // of the whole part
	// Halo is drawn about the letters of the part's text, HaloWidth wide, so that lines
	// under the text stop short of it: the title of a group an edge runs through.
	Halo      Paint
	HaloWidth Num
}

// Over is s laid over base: what s sets, and what base sets where s does not.
func (s Style) Over(base Style) Style {
	if !s.Fill.Set() {
		s.Fill = base.Fill
	}
	if !s.Stroke.Set() {
		s.Stroke = base.Stroke
	}
	if !s.Text.Set() {
		s.Text = base.Text
	}
	if !s.StrokeWidth.set {
		s.StrokeWidth = base.StrokeWidth
	}
	if !s.Radius.set {
		s.Radius = base.Radius
	}
	if !s.Dash.set {
		s.Dash = base.Dash
	}
	if !s.Cap.Set() {
		s.Cap = base.Cap
	}
	if !s.Join.Set() {
		s.Join = base.Join
	}
	if !s.Opacity.set {
		s.Opacity = base.Opacity
	}
	if !s.Halo.Set() {
		s.Halo = base.Halo
	}
	if !s.HaloWidth.set {
		s.HaloWidth = base.HaloWidth
	}
	return s
}

// Overrides give styles to the roles of diagrams' parts, as "flowchart.node", by the
// names of scene.Role.
type Overrides map[string]Style

// A Look is a theme and the overrides on it: all of style a backend is given.
type Look struct {
	Theme     *Theme // nil is Default
	Overrides Overrides
}

var defaultTheme = Default()

// Cascade is the style of a part of a role: the theme's measures under the diagram's own
// style for the part, under the overrides for its role, under the style its author gave it
// in the diagram's text. Text is written in Fg unless something sets another paint.
func (l *Look) Cascade(role string, own, author Style) Style {
	th := l.ThemeOrDefault()
	var over Style
	if l != nil {
		over = l.Overrides[role]
	}
	s := author.Over(over.Over(own))
	if !s.StrokeWidth.set {
		s.StrokeWidth = N(th.StrokeWidth)
	}
	if !s.Radius.set {
		s.Radius = N(th.Radius)
	}
	if !s.Text.Set() {
		s.Text = Fg
	}
	return s
}

// ThemeOrDefault is the look's theme, or Default when it has none. The default is shared:
// to change it, change a copy.
func (l *Look) ThemeOrDefault() *Theme {
	if l == nil || l.Theme == nil {
		return defaultTheme
	}
	return l.Theme
}

// Resolved is a style with its colors and measures settled, ready to draw.
type Resolved struct {
	Fill, Stroke, Text Color // alpha 0 draws nothing
	Line               Color // the theme's color of lines, for parts painted solid
	Bg                 Color // the theme's background, for parts painted hollow
	StrokeWidth        float64
	Radius             float64
	Dash               []float64 // none for a solid line
	Cap                Cap       // CapButt when nothing set it, as SVG's initial value
	Join               Join      // JoinMiter when nothing set it, as SVG's initial value
	Opacity            float64
	Halo               Color // alpha 0 draws none
	HaloWidth          float64
}

// Resolve settles a part's style in the look.
func (l *Look) Resolve(role string, own, author Style) Resolved {
	s := l.Cascade(role, own, author)
	th := l.ThemeOrDefault()
	if !s.Cap.Set() {
		s.Cap = CapButt
	}
	if !s.Join.Set() {
		s.Join = JoinMiter
	}
	return Resolved{
		Fill:        s.Fill.Color(th),
		Stroke:      s.Stroke.Color(th),
		Text:        s.Text.Color(th),
		Line:        th.Line,
		Bg:          th.Bg,
		StrokeWidth: s.StrokeWidth.v,
		Radius:      s.Radius.v,
		Dash:        s.Dash.v,
		Cap:         s.Cap,
		Join:        s.Join,
		Opacity:     s.Opacity.Or(1),
		Halo:        s.Halo.Color(th),
		HaloWidth:   s.HaloWidth.Or(0),
	}
}
