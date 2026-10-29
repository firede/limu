package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/firede/limu/style"
)

// An appearance is how the playground looks: the look its diagrams are drawn in, the
// theme of its own controls, and the colors of its code, so that the whole window matches.
// It is style only. Changing it draws the diagram again in other colors, and never lays it
// out again, as limu keeps what moves things (layout.Options) apart from what colors them
// (style.Look).
type appearance struct {
	Name    string
	Dark    bool
	Diagram style.Look
	UI      *ui.Theme
	Code    *editor.Theme
	// Track and Knob are the colors of a segmented control, as the website's: the track
	// the segments are in, and the pill behind the one chosen.
	Track, Knob ui.Color
	// Strong is the color of text that stands out, as a segment chosen.
	Strong ui.Color
}

// The names of the appearances, as the Appearance menu lists them and the settings keep
// them. System is Light or Dark, as the system's appearance is; in a browser, as the page's
// is.
const (
	appearanceSystem = "System"
	appearanceLight  = "Light"
	appearanceDark   = "Dark"
)

var appearanceNames = []string{appearanceSystem, appearanceLight, appearanceDark}

// A palette is the colors of one of the website's themes, by the names of its CSS
// variables in src/styles/base.scss of firede/limu-website, so that the playground looks as
// the site around it does. The website's TestPalettes holds the palettes of the release it
// builds to that file, reading them from kohu and kohuDark, so their names and fields stay.
type palette struct {
	dark bool

	ink, panel, line, lineHover, faint, muted, body, paper string
	accent, accentInk, red, green                          string
	selection                                              string // translucent

	// The colors of code.
	syntaxPunctuation, syntaxKeyword, syntaxType, syntaxConstant, syntaxNumber string
	syntaxVariable, syntaxFunction, syntaxString, syntaxComment                string

	// diagram is the theme the site draws diagrams in, in this palette.
	diagram func() *style.Theme
}

var kohu = palette{
	ink: "#f1f6f5", panel: "#e3edeb", line: "#cddcda", lineHover: "#bdd0ce",
	faint: "#8aa3a5", muted: "#536f72", body: "#1d3539", paper: "#0f262a",
	accent: "#a92e4a", accentInk: "#f1f6f5", red: "#b5435c", green: "#4f8a3a",
	selection:         "#a92e4a29", // rgb(169 46 74 / 16%)
	syntaxPunctuation: "#536f72", syntaxKeyword: "#a92e4a", syntaxType: "#6a4a92",
	syntaxConstant: "#8a5f10", syntaxNumber: "#8a5f10", syntaxVariable: "#2f5d96",
	syntaxFunction: "#2f5d96", syntaxString: "#1c6e6a", syntaxComment: "#4f686b",
	diagram: style.Kohu,
}

var kohuDark = palette{
	dark: true,
	ink:  "#0c1618", panel: "#132125", line: "#213439", lineHover: "#2a4247",
	faint: "#4f6b70", muted: "#7f989b", body: "#d0dddc", paper: "#e6efee",
	accent: "#ee8494", accentInk: "#0c1618", red: "#e37d90", green: "#8cbf6a",
	selection:         "#ee849438", // rgb(238 132 148 / 22%)
	syntaxPunctuation: "#7f989b", syntaxKeyword: "#ee8494", syntaxType: "#ad92d1",
	syntaxConstant: "#d9a748", syntaxNumber: "#d9a748", syntaxVariable: "#78a3d9",
	syntaxFunction: "#78a3d9", syntaxString: "#5cbcb3", syntaxComment: "#6f8a8e",
	diagram: style.KohuDark,
}

// appearances are the appearances of the site's themes, made once, so that an
// appearance stays the same from frame to frame, as what was drawn in it is kept by it.
var appearances = map[string]*appearance{
	appearanceLight: kohu.appearance(appearanceLight),
	appearanceDark:  kohuDark.appearance(appearanceDark),
}

// appearanceNamed is the appearance of a name: System is Dark when dark, the system's
// appearance, is set, and Light when not; a name it does not know is System.
func appearanceNamed(name string, dark bool) *appearance {
	if ap, ok := appearances[name]; ok {
		return ap
	}
	if dark {
		return appearances[appearanceDark]
	}
	return appearances[appearanceLight]
}

func (p palette) appearance(name string) *appearance {
	return &appearance{Name: name, Dark: p.dark, Diagram: style.Look{Theme: p.diagram()}, UI: p.uiTheme(), Code: p.codeTheme(),
		Track: ui.Hex(p.panel), Knob: ui.Hex(p.line), Strong: ui.Hex(p.paper)}
}

// knownAppearance reports whether name is one of the appearances.
func knownAppearance(name string) bool {
	for _, n := range appearanceNames {
		if n == name {
			return true
		}
	}
	return false
}

// uiTheme is MyGo's theme in the colors of p, over MyGo's own for the measures and the
// colors the site has no variable for. The window is on the page's color, ink, and a
// control on the panel's, as the site's buttons are when hovered. The accent has no colors
// of its own for a hovered or pressed control; they are the accent mixed with the
// strongest text's. What calls for attention is in the amber of the diagram's critical.
func (p palette) uiTheme() *ui.Theme {
	c := ui.Hex
	t := ui.LightTheme()
	if p.dark {
		t = ui.DarkTheme()
	}
	t.Dark = p.dark
	t.Background = c(p.ink)
	t.Surface = c(p.panel)
	t.SurfaceHover = c(p.line)
	t.SurfacePressed = c(p.lineHover)
	t.Border = c(p.line)
	t.Text = c(p.body)
	t.TextMuted = c(p.muted)
	t.Accent = c(p.accent)
	t.AccentHover = mixUI(c(p.accent), c(p.paper), 0.1)
	t.AccentPressed = mixUI(c(p.accent), c(p.paper), 0.2)
	t.AccentText = c(p.accentInk)
	t.Danger = c(p.red)
	critical := p.diagram().Critical
	t.Warning = ui.RGB(critical.R, critical.G, critical.B)
	t.Success = c(p.green)
	t.Selection = c(p.selection)
	t.Focus = c(p.accent)
	t.Inverse = c(p.paper)
	t.InverseText = c(p.ink)
	t.Scrollbar = c(p.lineHover)
	t.Radius = 6 // the site's --radius-small
	return t
}

func mixUI(a, b ui.Color, f float64) ui.Color {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-f) + float64(y)*f + 0.5) }
	return ui.RGB(m(a.R, b.R), m(a.G, b.G), m(a.B, b.B))
}

// codeTheme is the editor's theme in the colors of p, each kind of token in the color the
// site's highlighter gives it (src/lib/highlight.ts of firede/limu-website): a link's arrow as
// punctuation, and front matter and directives, which say how to draw the rest, as the
// keys of front matter.
func (p palette) codeTheme() *editor.Theme {
	c := ui.Hex
	t := p.uiTheme()
	th := editor.DefaultTheme(t)
	th.Background = c(p.ink)
	th.Text = c(p.body)
	th.LineNumber = c(p.faint)
	th.ActiveLineNumber = c(p.muted)
	th.ActiveLine = mixUI(c(p.ink), c(p.panel), 0.6)
	th.Cursor = c(p.paper)
	th.Error, th.Warning, th.Info, th.Hint = t.Danger, t.Warning, c(p.syntaxVariable), t.TextMuted
	tok := func(k editor.TokenKind, hex string) { th.Tokens[k].Color = c(hex) }
	tok(editor.TokenKeyword, p.syntaxKeyword)
	tok(editor.TokenOperator, p.syntaxPunctuation)
	tok(editor.TokenType, p.syntaxType)
	tok(editor.TokenFunction, p.syntaxFunction)
	tok(editor.TokenVariable, p.syntaxVariable)
	tok(editor.TokenMeta, p.syntaxVariable)
	tok(editor.TokenConstant, p.syntaxConstant)
	tok(editor.TokenNumber, p.syntaxNumber)
	tok(editor.TokenProperty, p.syntaxVariable)
	tok(editor.TokenString, p.syntaxString)
	tok(editor.TokenComment, p.syntaxComment)
	tok(editor.TokenPunctuation, p.syntaxPunctuation)
	tok(editor.TokenTag, p.syntaxVariable)
	tok(editor.TokenAttribute, p.syntaxVariable)
	return th
}
