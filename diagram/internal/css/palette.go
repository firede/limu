package css

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/style"
)

// A palette is a diagram type's palette theme variables, as Pele's PALETTES gives them: first
// is the number of the variable for the first series. Mindmap branches start at cScale1, as
// cScale0 is the root's in Mermaid's styles. A venn palette repeats the colors it sets, as in
// Mermaid.
type palette struct {
	prefix string
	first  int
	repeat bool
}

// palettes are Mermaid's palette theme variables, which set a diagram's own series colors, by
// Pele's name of the diagram type, as Pele's PALETTES.
var palettes = map[string]palette{
	"pie":      {prefix: "pie", first: 1},
	"gitGraph": {prefix: "git", first: 0},
	"timeline": {prefix: "cScale", first: 0},
	"radar":    {prefix: "cScale", first: 0},
	"treemap":  {prefix: "cScale", first: 0},
	"mindmap":  {prefix: "cScale", first: 1},
	"venn":     {prefix: "venn", first: 1, repeat: true},
}

// slots is how many series a palette sets, Pele's SLOTS.
const slots = 8

// paletteFunctions are the functions a palette's color may use, Pele's FUNCTIONS.
var paletteFunctions = map[string]bool{
	"rgb": true, "rgba": true, "hsl": true, "hsla": true, "hwb": true, "lab": true, "lch": true,
	"oklab": true, "oklch": true, "color": true, "color-mix": true, "light-dark": true, "var": true,
}

// A Palette is the series colors a diagram's theme variables set, as Pele's palette reads
// them.
type Palette struct {
	// Series is the color each series takes, where a theme variable sets one limu can read:
	// a color, None for transparent, or Fg for currentColor, as Pele's root sets the color of
	// the text. A color Pele passes on that limu can't work out, such as var(--pele-accent),
	// color-mix() or oklch(), leaves its series unset, in the theme's color.
	Series [slots]style.Paint
	// inks says on which series Pele's ink picks a color for a label, as its palette's inks.
	inks [slots]bool
}

// Ground is the ground a label on series i lies on, the series' paint, where the palette sets
// that series and Pele's ink picks a color for a label on it, and unset otherwise, as Pele's
// seriesInkFill gives a label a fill. A diagram sets it as the label's style.Style.TextGround,
// and the style picks the color when the scene is drawn.
func (p *Palette) Ground(i int) style.Paint {
	k := (i%slots + slots) % slots
	if !p.inks[k] || !p.Series[k].Set() {
		return style.Paint{}
	}
	return style.Series(k)
}

// ReadPalette is the palette a diagram's theme variables set, for Pele's name of its type
// ("pie", "gitGraph", "xychart"…), as Pele's palette reads config.themeVariables. An entry
// that isn't a usable color leaves its series in the theme's color.
func ReadPalette(typ string, config map[string]any) Palette {
	var out Palette
	vars, _ := config["themeVariables"].(map[string]any)
	list, repeat, ok := paletteEntries(typ, vars)
	if !ok || len(list) == 0 {
		return out
	}
	for i := range slots {
		if !repeat && i >= len(list) {
			break
		}
		color, ok := safeColor(list[i%len(list)])
		if !ok {
			continue
		}
		out.Series[i] = PaletteColor(color)
		alpha, ok := colorAlpha(color)
		out.inks[i] = ok && alpha >= 0.5
	}
	return out
}

// paletteEntries is the list of a type's palette theme variables, and whether it repeats, as
// Pele's entries gives them: xychart's plotColorPalette, a list split at its commas, and the
// numbered variables of the rest.
func paletteEntries(typ string, vars map[string]any) ([]any, bool, bool) {
	if typ == "xychart" {
		xy, _ := vars["xyChart"].(map[string]any)
		list, ok := xy["plotColorPalette"].(string)
		if !ok {
			return nil, false, false
		}
		var out []any
		for _, part := range splitList(list) {
			out = append(out, part)
		}
		return out, true, true
	}
	p, ok := palettes[typ]
	if !ok {
		return nil, false, false
	}
	var list []any
	for i := range slots {
		v := vars[p.prefix+strconv.Itoa(p.first+i)]
		// Pele's filter drops undefined, null and the empty string.
		if p.repeat && (v == nil || v == "") {
			continue
		}
		list = append(list, v)
	}
	return list, p.repeat, true
}

// splitList splits a list of colors at the commas between them, not those inside rgb() and
// other functions, and trims each, as Pele's splitList does.
func splitList(value string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, value[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, value[start:])
	for i, part := range parts {
		parts[i] = chars.Trim(part)
	}
	return parts
}

// safeColor is a palette's value, trimmed, where it is a color that can't load a resource,
// end its declaration or read the page's own variables, as Pele's safeColor takes it. Values
// outside functions are checked as colors, so a typo keeps the theme color.
func safeColor(value any) (string, bool) {
	s, ok := value.(string)
	if !ok {
		return "", false
	}
	color := chars.Trim(s)
	// /^[\w\s#%.,+\-/()]{1,200}$/, counted in UTF-16 code units, which each of these is one.
	n := 0
	for _, r := range color {
		n++
		if !(r < utf8.RuneSelf && (isWord(byte(r)) || strings.IndexByte("#%.,+-/()", byte(r)) >= 0) || chars.IsSpaceRune(r)) {
			return "", false
		}
	}
	if n < 1 || n > 200 {
		return "", false
	}
	if !strings.Contains(color, "(") {
		if plainColor(color) {
			return color, true
		}
		return "", false
	}
	depth := 0
	for i := 0; i < len(color); i++ {
		switch color[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return "", false
			}
		}
	}
	if depth != 0 {
		return "", false
	}
	// The name before each parenthesis, as /([\w-]*)\s*\(/g finds it: the word characters and
	// dashes before the white space before it, back to the parenthesis before.
	from := 0
	for i := 0; i < len(color); i++ {
		if color[i] != '(' {
			continue
		}
		j := i
		for j > from {
			r, size := utf8.DecodeLastRuneInString(color[from:j])
			if !chars.IsSpaceRune(r) {
				break
			}
			j -= size
		}
		k := j
		for k > from && (isWord(color[k-1]) || color[k-1] == '-') {
			k--
		}
		if !paletteFunctions[lowerASCII(color[k:j])] {
			return "", false
		}
		from = i + 1
	}
	// Each var() reads only Pele's own variables, as /var\(\s*([\w-]*)/gi finds them.
	lower := lowerASCII(color)
	for at := 0; ; {
		i := strings.Index(lower[at:], "var(")
		if i < 0 {
			break
		}
		j := at + i + len("var(")
		for j < len(color) {
			r, size := utf8.DecodeRuneInString(color[j:])
			if !chars.IsSpaceRune(r) {
				break
			}
			j += size
		}
		k := j
		for k < len(color) && (isWord(color[k]) || color[k] == '-') {
			k++
		}
		if !strings.HasPrefix(color[j:k], "--pele-") {
			return "", false
		}
		at = k
	}
	return color, true
}

// isWord says whether b is one of the word characters of a regular expression's \w.
func isWord(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b == '_'
}

// plainColor reports whether a value is a hex color, one of CSS's named colors, transparent
// or currentColor, as Pele's plainColor does.
func plainColor(raw string) bool {
	value := chars.ToLower(chars.Trim(raw))
	switch value {
	case "transparent", "currentcolor":
		return true
	}
	if h, ok := strings.CutPrefix(value, "#"); ok {
		switch len(h) {
		case 3, 4, 6, 8:
		default:
			return false
		}
		for i := range len(h) {
			if c := h[i]; !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
				return false
			}
		}
		return true
	}
	if strings.Contains(value, "(") {
		return false
	}
	// The named colors are those style reads, with gray for the first grey.
	_, ok := style.ParseColor(strings.Replace(value, "grey", "gray", 1))
	return ok
}

// PaletteColor is the paint limu reads a color Pele's palette passes on as: a color style
// reads, None where it draws nothing, as transparent, or Fg for currentColor. A color limu
// can't work out, such as var(), color-mix() or oklch(), is unset.
func PaletteColor(color string) style.Paint {
	if lowerASCII(cssTrim(color)) == "currentcolor" {
		return style.Fg
	}
	c, ok := style.ParseColor(color)
	switch {
	case !ok:
		return style.Paint{}
	case c.A == 0:
		return style.None
	}
	return style.Of(c)
}
