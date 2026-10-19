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
	// Chrome is the background of the toolbar and the status bar, about the panes.
	Chrome ui.Color
	// Track, Knob and KnobBorder are the colors of a segmented control, as Primer's: the
	// track the segments are in, and the segment chosen, which stands out from it in a
	// dark theme too.
	Track, Knob, KnobBorder ui.Color
}

// The names of the appearances, as the Appearance menu lists them and the settings keep
// them. System is GitHub Light or GitHub Dark, as the system's appearance is.
const (
	appearanceSystem     = "System"
	appearanceLight      = "GitHub Light"
	appearanceDark       = "GitHub Dark"
	appearanceDarkDimmed = "GitHub Dark Dimmed"
)

var appearanceNames = []string{appearanceSystem, appearanceLight, appearanceDark, appearanceDarkDimmed}

// A primer is the colors of one of GitHub's themes, by the names of Primer's design tokens.
// The values are those of @primer/primitives 10.3.4, from its CSS files
// dist/css/functional/themes/light.css, dark.css and dark-dimmed.css
// (https://primer.style/foundations/primitives/color, MIT License, by GitHub). Primer
// names the canvas colors bgColor now: canvas default, subtle and inset are bgColor-default,
// bgColor-muted and bgColor-inset.
type primer struct {
	dark bool

	bgDefault, bgMuted, bgInset, bgEmphasis string
	bgNeutralMuted                          string // translucent, laid over bgDefault
	bgAccentEmphasis                        string

	fgDefault, fgMuted, fgOnEmphasis           string
	fgAccent, fgDanger, fgAttention, fgSuccess string

	borderDefault, borderEmphasis string

	controlRest, controlHover, controlActive string
	selection                                string // selection-bgColor, translucent

	// data are Primer's colors for data visualization, data-*-color-emphasis, in the order
	// of limu's own series (blue, orange, green, red, teal, yellow, purple and a brown),
	// so that a diagram's first series is blue in either.
	data [8]string

	// The colors of code, color-prettylights-syntax-*.
	syntaxComment, syntaxConstant, syntaxEntity, syntaxEntityTag string
	syntaxKeyword, syntaxString, syntaxVariable                  string
}

var githubLight = primer{
	bgDefault: "#ffffff", bgMuted: "#f6f8fa", bgInset: "#f6f8fa", bgEmphasis: "#25292e",
	bgNeutralMuted: "#818b981f", bgAccentEmphasis: "#0969da",
	fgDefault: "#1f2328", fgMuted: "#59636e", fgOnEmphasis: "#ffffff",
	fgAccent: "#0969da", fgDanger: "#d1242f", fgAttention: "#9a6700", fgSuccess: "#1a7f37",
	borderDefault: "#d1d9e0", borderEmphasis: "#818b98",
	controlRest: "#f6f8fa", controlHover: "#eff2f5", controlActive: "#e6eaef",
	selection: "#0969da33",
	data: [8]string{
		"#006edb", // blue
		"#eb670f", // orange
		"#30a147", // green
		"#df0c24", // red
		"#179b9b", // teal
		"#b88700", // yellow
		"#894ceb", // purple
		"#9d615c", // auburn
	},
	syntaxComment: "#59636e", syntaxConstant: "#0550ae", syntaxEntity: "#6639ba",
	syntaxEntityTag: "#0550ae", syntaxKeyword: "#cf222e", syntaxString: "#0a3069",
	syntaxVariable: "#953800",
}

// githubDarkData are the colors for data visualization of both dark themes, which Primer
// gives the same.
var githubDarkData = [8]string{
	"#0576ff", // blue
	"#984b10", // orange
	"#2f6f37", // green
	"#eb3342", // red
	"#106c70", // teal
	"#895906", // yellow
	"#975bf1", // purple
	"#a86f6b", // auburn
}

var githubDark = primer{
	dark:      true,
	bgDefault: "#0d1117", bgMuted: "#151b23", bgInset: "#010409", bgEmphasis: "#3d444d",
	bgNeutralMuted: "#656c7633", bgAccentEmphasis: "#1f6feb",
	fgDefault: "#f0f6fc", fgMuted: "#9198a1", fgOnEmphasis: "#ffffff",
	fgAccent: "#4493f8", fgDanger: "#f85149", fgAttention: "#d29922", fgSuccess: "#3fb950",
	borderDefault: "#3d444d", borderEmphasis: "#656c76",
	controlRest: "#212830", controlHover: "#262c36", controlActive: "#2a313c",
	selection:     "#1f6febb3",
	data:          githubDarkData,
	syntaxComment: "#9198a1", syntaxConstant: "#79c0ff", syntaxEntity: "#d2a8ff",
	syntaxEntityTag: "#7ee787", syntaxKeyword: "#ff7b72", syntaxString: "#a5d6ff",
	syntaxVariable: "#ffa657",
}

var githubDarkDimmed = primer{
	dark:      true,
	bgDefault: "#212830", bgMuted: "#262c36", bgInset: "#151b23", bgEmphasis: "#3d444d",
	bgNeutralMuted: "#656c7633", bgAccentEmphasis: "#316dca",
	fgDefault: "#d1d7e0", fgMuted: "#9198a1", fgOnEmphasis: "#f0f6fc",
	fgAccent: "#478be6", fgDanger: "#e5534b", fgAttention: "#c69026", fgSuccess: "#57ab5a",
	borderDefault: "#3d444d", borderEmphasis: "#656c76",
	controlRest: "#2a313c", controlHover: "#2f3742", controlActive: "#3d444d",
	selection:     "#316dcab3",
	data:          githubDarkData,
	syntaxComment: "#9198a1", syntaxConstant: "#6cb6ff", syntaxEntity: "#dcbdfb",
	syntaxEntityTag: "#8ddb8c", syntaxKeyword: "#f47067", syntaxString: "#96d0ff",
	syntaxVariable: "#f69d50",
}

// appearances are the appearances of GitHub's themes, made once, so that an appearance
// stays the same from frame to frame, as what was drawn in it is kept by it.
var appearances = map[string]*appearance{
	appearanceLight:      githubLight.appearance(appearanceLight),
	appearanceDark:       githubDark.appearance(appearanceDark),
	appearanceDarkDimmed: githubDarkDimmed.appearance(appearanceDarkDimmed),
}

// appearanceNamed is the appearance of a name: System is GitHub Dark when dark, the
// system's appearance, is set, and GitHub Light when not; a name it does not know is
// System.
func appearanceNamed(name string, dark bool) *appearance {
	if ap, ok := appearances[name]; ok {
		return ap
	}
	if dark {
		return appearances[appearanceDark]
	}
	return appearances[appearanceLight]
}

func (p primer) appearance(name string) *appearance {
	look := style.Look{Theme: p.diagramTheme()}
	if p.dark {
		// A pie's percentages on its slices are in the background's color, which is too
		// dark on the dark colors Primer gives data in a dark theme: they take the text's.
		look.Overrides = style.Overrides{"pie.slice-label": {Text: style.Fg}}
	}
	return &appearance{Name: name, Dark: p.dark, Diagram: look, UI: p.uiTheme(), Code: p.codeTheme(), Chrome: ui.Hex(p.bgInset),
		Track: ui.Hex(p.controlHover), Knob: ui.Hex(p.knob()), KnobBorder: ui.Hex(p.borderEmphasis)}
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

// diagramTheme is limu's theme in the colors of p: the canvas behind, Primer's text colors,
// shapes on the muted canvas with the emphasized border, so that they stand out from it,
// and groups on the neutral muted color.
func (p primer) diagramTheme() *style.Theme {
	c := style.Hex
	th := &style.Theme{
		Bg:          c(p.bgDefault),
		Fg:          c(p.fgDefault),
		Muted:       c(p.fgMuted),
		Line:        c(p.fgMuted),
		Surface:     c(p.bgMuted),
		SurfaceAlt:  over(c(p.bgNeutralMuted), c(p.bgDefault)),
		Border:      c(p.borderEmphasis),
		Accent:      c(p.fgAccent),
		Critical:    c(p.fgDanger),
		Radius:      6, // Primer's borderRadius-medium
		StrokeWidth: 1,
	}
	for i, s := range p.data {
		th.Series[i] = c(s)
	}
	return th
}

// over is a translucent color laid over an opaque one: a theme's colors are opaque, as
// limu lays its own opacities on them.
func over(fg, bg style.Color) style.Color {
	a := float64(fg.A) / 255
	mix := func(f, b uint8) uint8 { return uint8(float64(f)*a + float64(b)*(1-a) + 0.5) }
	return style.RGB(mix(fg.R, bg.R), mix(fg.G, bg.G), mix(fg.B, bg.B))
}

// uiTheme is MyGo's theme in the colors of p, over MyGo's own for the measures and the
// colors Primer has no token for. Primer's accent has no colors of its own for a hovered
// or pressed control; they are the accent mixed with the text's color.
func (p primer) uiTheme() *ui.Theme {
	c := ui.Hex
	t := ui.LightTheme()
	if p.dark {
		t = ui.DarkTheme()
	}
	t.Dark = p.dark
	t.Background = c(p.bgDefault)
	t.Surface = c(p.controlRest)
	t.SurfaceHover = c(p.controlHover)
	t.SurfacePressed = c(p.controlActive)
	t.Border = c(p.borderDefault)
	t.Text = c(p.fgDefault)
	t.TextMuted = c(p.fgMuted)
	t.Accent = c(p.bgAccentEmphasis)
	t.AccentHover = mixUI(c(p.bgAccentEmphasis), c(p.fgDefault), 0.1)
	t.AccentPressed = mixUI(c(p.bgAccentEmphasis), c(p.fgDefault), 0.2)
	t.AccentText = c(p.fgOnEmphasis)
	t.Danger = c(p.fgDanger)
	t.Warning = c(p.fgAttention)
	t.Success = c(p.fgSuccess)
	t.Selection = c(p.selection)
	t.Focus = c(p.bgAccentEmphasis) // focus-outlineColor is borderColor-accent-emphasis
	t.Inverse = c(p.bgEmphasis)     // tooltip-bgColor
	t.InverseText = c(p.fgOnEmphasis)
	t.Radius = 6
	return t
}

func mixUI(a, b ui.Color, f float64) ui.Color {
	m := func(x, y uint8) uint8 { return uint8(float64(x)*(1-f) + float64(y)*f + 0.5) }
	return ui.RGB(m(a.R, b.R), m(a.G, b.G), m(a.B, b.B))
}

// codeTheme is the editor's theme in the colors of p: GitHub's colors of code, over the
// editor's own theme for the window's theme.
func (p primer) codeTheme() *editor.Theme {
	c := ui.Hex
	th := editor.DefaultTheme(p.uiTheme())
	th.Background = c(p.bgDefault)
	th.Text = c(p.fgDefault)
	th.LineNumber = c(p.fgMuted)
	th.ActiveLineNumber = c(p.fgDefault)
	th.ActiveLine = c(p.bgNeutralMuted) // codeMirror-activeline-bgColor
	th.Cursor = c(p.fgDefault)
	th.Error, th.Warning, th.Info, th.Hint = c(p.fgDanger), c(p.fgAttention), c(p.fgAccent), c(p.fgMuted)
	tok := func(k editor.TokenKind, hex string) { th.Tokens[k].Color = c(hex) }
	tok(editor.TokenKeyword, p.syntaxKeyword)
	tok(editor.TokenOperator, p.syntaxConstant)
	tok(editor.TokenType, p.syntaxVariable)
	tok(editor.TokenFunction, p.syntaxEntity)
	tok(editor.TokenMeta, p.syntaxEntity)
	tok(editor.TokenConstant, p.syntaxConstant)
	tok(editor.TokenNumber, p.syntaxConstant)
	tok(editor.TokenProperty, p.syntaxConstant)
	tok(editor.TokenString, p.syntaxString)
	tok(editor.TokenComment, p.syntaxComment)
	tok(editor.TokenPunctuation, p.fgMuted)
	tok(editor.TokenTag, p.syntaxEntityTag)
	tok(editor.TokenAttribute, p.syntaxConstant)
	return th
}

// knob is the background of a segmented control's chosen segment: the page's in a light
// theme, and the darkest in a dark one, as Primer's controlKnob-bgColor-rest is.
func (p primer) knob() string {
	if p.dark {
		return p.bgInset
	}
	return p.bgDefault
}
