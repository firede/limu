package sequence

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// Mermaid asks the browser whether the first word of a box's title is a color. Pele has no
// browser, so it knows the color keywords and the rgb() and hsl() notations, as its colors.ts
// does; other functions, such as calc() and var(), are not recognized.
const colorKeywords = "aliceblue antiquewhite aqua aquamarine azure beige bisque black blanchedalmond blue blueviolet brown burlywood " +
	"cadetblue chartreuse chocolate coral cornflowerblue cornsilk crimson cyan darkblue darkcyan darkgoldenrod darkgray " +
	"darkgreen darkgrey darkkhaki darkmagenta darkolivegreen darkorange darkorchid darkred darksalmon darkseagreen " +
	"darkslateblue darkslategray darkslategrey darkturquoise darkviolet deeppink deepskyblue dimgray dimgrey dodgerblue " +
	"firebrick floralwhite forestgreen fuchsia gainsboro ghostwhite gold goldenrod gray green greenyellow grey honeydew " +
	"hotpink indianred indigo ivory khaki lavender lavenderblush lawngreen lemonchiffon lightblue lightcoral lightcyan " +
	"lightgoldenrodyellow lightgray lightgreen lightgrey lightpink lightsalmon lightseagreen lightskyblue lightslategray " +
	"lightslategrey lightsteelblue lightyellow lime limegreen linen magenta maroon mediumaquamarine mediumblue " +
	"mediumorchid mediumpurple mediumseagreen mediumslateblue mediumspringgreen mediumturquoise mediumvioletred " +
	"midnightblue mintcream mistyrose moccasin navajowhite navy oldlace olive olivedrab orange orangered orchid " +
	"palegoldenrod palegreen paleturquoise palevioletred papayawhip peachpuff peru pink plum powderblue purple " +
	"rebeccapurple red rosybrown royalblue saddlebrown salmon sandybrown seagreen seashell sienna silver skyblue " +
	"slateblue slategray slategrey snow springgreen steelblue tan teal thistle tomato turquoise violet wheat white " +
	"whitesmoke yellow yellowgreen transparent currentcolor inherit initial unset revert " +
	"accentcolor accentcolortext activetext buttonborder buttonface buttontext canvas canvastext field fieldtext " +
	"graytext highlight highlighttext linktext mark marktext selecteditem selecteditemtext visitedtext " +
	"activeborder activecaption appworkspace background buttonhighlight buttonshadow captiontext inactiveborder " +
	"inactivecaption inactivecaptiontext infobackground infotext menu menutext scrollbar threeddarkshadow threedface " +
	"threedhighlight threedlightshadow threedshadow window windowframe windowtext"

var keywordSet = func() map[string]bool {
	m := map[string]bool{}
	for _, k := range strings.Split(colorKeywords, " ") {
		m[k] = true
	}
	return m
}()

// The kinds of term of a color function, Pele's K.
const (
	termBad = iota
	termNumber
	termPercent
	termAngle
	termNone
)

// termKind reads a term as /^[+-]?(?:\d+(?:\.\d+)?|\.\d+)(?:e[+-]?\d+)?(%|deg|grad|rad|turn)?$/i
// does, as Pele's kind does.
func termKind(term string) int {
	if chars.ToLower(term) == "none" {
		return termNone
	}
	i, n := 0, len(term)
	digits := func() int {
		start := i
		for i < n && term[i] >= '0' && term[i] <= '9' {
			i++
		}
		return i - start
	}
	if i < n && (term[i] == '+' || term[i] == '-') {
		i++
	}
	if digits() > 0 {
		if i < n && term[i] == '.' {
			save := i
			i++
			if digits() == 0 {
				i = save
			}
		}
	} else {
		if i >= n || term[i] != '.' {
			return termBad
		}
		i++
		if digits() == 0 {
			return termBad
		}
	}
	if i < n && (term[i] == 'e' || term[i] == 'E') {
		save := i
		i++
		if i < n && (term[i] == '+' || term[i] == '-') {
			i++
		}
		if digits() == 0 {
			i = save
		}
	}
	unit := strings.ToLower(term[i:])
	switch unit {
	case "":
		return termNumber
	case "%":
		return termPercent
	case "deg", "grad", "rad", "turn":
		return termAngle
	}
	return termBad
}

// isAlpha reports whether a term can be a color function's alpha: a number or a percentage, or
// none in the modern notation, as Pele's isAlpha does.
func isAlpha(term string, modern bool) bool {
	k := termKind(term)
	return k == termNumber || k == termPercent || modern && k == termNone
}

// splitSpaces is s.split(/\s+/), for a string that starts and ends with no space. It stands
// for the split in Pele's isFunction.
func splitSpaces(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); {
		c, size := chars.UnitAt(s, i)
		if !chars.IsSpace(c) {
			i += size
			continue
		}
		out = append(out, s[start:i])
		i = chars.SpaceRun(s, i)
		start = i
	}
	return append(out, s[start:])
}

// isFunction reports whether a color is an rgb(), rgba(), hsl() or hsla() function whose terms
// are of the kinds the notation takes, in the legacy notation with commas or the modern one
// with spaces, as Pele's isFunction does.
func isFunction(color string) bool {
	hsl := strings.HasPrefix(color, "hsl")
	open := strings.IndexByte(color, '(')
	name := color[:open]
	if name != "rgb" && name != "rgba" && name != "hsl" && name != "hsla" {
		return false
	}
	if chars.Len16(color) > 120 || strings.IndexByte(color, ')') != len(color)-1 {
		return false
	}
	inner := chars.Trim(color[open+1 : len(color)-1])
	var parts []string
	alpha, hasAlpha := "", false
	legacy := strings.Contains(inner, ",")
	if legacy {
		for _, s := range strings.Split(inner, ",") {
			parts = append(parts, chars.Trim(s))
		}
		if len(parts) == 4 {
			alpha, hasAlpha = parts[3], true
			parts = parts[:3]
		}
	} else {
		slash := strings.Split(inner, "/")
		if len(slash) > 2 {
			return false
		}
		parts = splitSpaces(chars.Trim(slash[0]))
		if len(slash) == 2 {
			alpha, hasAlpha = chars.Trim(slash[1]), true
		}
	}
	if len(parts) != 3 || hasAlpha && !isAlpha(alpha, !legacy) {
		return false
	}
	var kinds [3]int
	for i, p := range parts {
		kinds[i] = termKind(p)
	}
	if legacy {
		if hsl {
			return (kinds[0] == termNumber || kinds[0] == termAngle) && kinds[1] == termPercent && kinds[2] == termPercent
		}
		return (kinds[0] == termNumber || kinds[0] == termPercent) && kinds[1] == kinds[0] && kinds[2] == kinds[0]
	}
	for i, k := range kinds {
		ok := k == termNumber || k == termNone
		if hsl && i == 0 {
			ok = ok || k == termAngle
		} else {
			ok = ok || k == termPercent
		}
		if !ok {
			return false
		}
	}
	return true
}

// isCSSColor reports whether a box's first word is a color, as Pele's isCssColor does.
func isCSSColor(color string) bool {
	if strings.Contains(color, "(") {
		return isFunction(color)
	}
	return keywordSet[chars.ToLower(color)]
}
