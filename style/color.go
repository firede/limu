package style

import (
	"math"
	"strconv"
	"strings"
)

// A Color is a color in sRGB with straight alpha.
type Color struct{ R, G, B, A uint8 }

// RGB is an opaque color.
func RGB(r, g, b uint8) Color { return Color{r, g, b, 255} }

// Hex is the color #rgb, #rgba, #rrggbb or #rrggbbaa writes; it panics on anything else, so
// it is for colors written in code. ParseColor reads them from text.
func Hex(s string) Color {
	c, ok := parseHex(strings.TrimPrefix(s, "#"))
	if !ok {
		panic("style: bad hex color " + strconv.Quote(s))
	}
	return c
}

// String writes the color as #rrggbb, or #rrggbbaa when it is not opaque.
func (c Color) String() string {
	const digits = "0123456789abcdef"
	b := []byte{'#', digits[c.R>>4], digits[c.R&15], digits[c.G>>4], digits[c.G&15], digits[c.B>>4], digits[c.B&15]}
	if c.A != 255 {
		b = append(b, digits[c.A>>4], digits[c.A&15])
	}
	return string(b)
}

// ParseColor reads a CSS color: a hex color, rgb() or rgba(), hsl() or hsla(), a named
// color or transparent, with the white space of CSS about it, its names in either case of
// their ASCII letters. It reports false for anything else, such as currentColor or var().
func ParseColor(s string) (Color, bool) {
	s = lowerASCII(cssTrim(s))
	if strings.HasPrefix(s, "#") {
		return parseHex(s[1:])
	}
	if name, args, ok := function(s); ok {
		switch name {
		case "rgb", "rgba":
			return parseRGB(args)
		case "hsl", "hsla":
			return parseHSL(args)
		}
		return Color{}, false
	}
	if s == "transparent" {
		return Color{}, true
	}
	c, ok := namedColors[s]
	return c, ok
}

func parseHex(h string) (Color, bool) {
	var v [8]uint8
	for i := 0; i < len(h); i++ {
		switch b := h[i]; {
		case b >= '0' && b <= '9':
			v[i%8] = b - '0'
		case b >= 'a' && b <= 'f':
			v[i%8] = b - 'a' + 10
		case b >= 'A' && b <= 'F':
			v[i%8] = b - 'A' + 10
		default:
			return Color{}, false
		}
	}
	switch len(h) {
	case 3, 4:
		c := Color{v[0] * 17, v[1] * 17, v[2] * 17, 255}
		if len(h) == 4 {
			c.A = v[3] * 17
		}
		return c, true
	case 6, 8:
		c := Color{v[0]<<4 | v[1], v[2]<<4 | v[3], v[4]<<4 | v[5], 255}
		if len(h) == 8 {
			c.A = v[6]<<4 | v[7]
		}
		return c, true
	}
	return Color{}, false
}

// function splits name(args) into the name and its arguments, by commas, a slash or the
// white space of CSS. CSS has nothing between a function's name and its parenthesis.
func function(s string) (string, []string, bool) {
	open := strings.IndexByte(s, '(')
	if open <= 0 || !strings.HasSuffix(s, ")") {
		return "", nil, false
	}
	args := strings.FieldsFunc(s[open+1:len(s)-1], func(r rune) bool {
		return r == ',' || r == '/' || strings.ContainsRune(" \t\n\r\f", r)
	})
	return s[:open], args, true
}

// number reads a number, or a percentage of full; it reports whether it was a percentage.
func number(s string, full float64) (float64, bool, bool) {
	pct := strings.HasSuffix(s, "%")
	f, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 64)
	if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false, false
	}
	if pct {
		f = f * full / 100
	}
	return f, pct, true
}

func channel(f float64) uint8 { return uint8(math.Round(max(0, min(255, f)))) }

func alpha(args []string) (uint8, bool) {
	if len(args) < 4 {
		return 255, true
	}
	a, _, ok := number(args[3], 1)
	return channel(a * 255), ok
}

func parseRGB(args []string) (Color, bool) {
	if len(args) != 3 && len(args) != 4 {
		return Color{}, false
	}
	var c [3]uint8
	for i := range 3 {
		f, _, ok := number(args[i], 255)
		if !ok {
			return Color{}, false
		}
		c[i] = channel(f)
	}
	a, ok := alpha(args)
	return Color{c[0], c[1], c[2], a}, ok
}

func parseHSL(args []string) (Color, bool) {
	if len(args) != 3 && len(args) != 4 {
		return Color{}, false
	}
	h, _, ok1 := number(strings.TrimSuffix(args[0], "deg"), 360)
	s, _, ok2 := number(args[1], 1)
	l, _, ok3 := number(args[2], 1)
	if !ok1 || !ok2 || !ok3 {
		return Color{}, false
	}
	if !strings.HasSuffix(args[1], "%") {
		s /= 100
	}
	if !strings.HasSuffix(args[2], "%") {
		l /= 100
	}
	s, l = max(0, min(1, s)), max(0, min(1, l))
	h = math.Mod(math.Mod(h, 360)+360, 360) / 360
	f := func(n float64) float64 {
		k := math.Mod(n+h*12, 12)
		return l - s*min(l, 1-l)*max(-1, min(k-3, 9-k, 1))
	}
	a, ok := alpha(args)
	return Color{channel(f(0) * 255), channel(f(8) * 255), channel(f(4) * 255), a}, ok
}

// The named colors of CSS.
var namedColors = map[string]Color{
	"aliceblue": {240, 248, 255, 255}, "antiquewhite": {250, 235, 215, 255}, "aqua": {0, 255, 255, 255},
	"aquamarine": {127, 255, 212, 255}, "azure": {240, 255, 255, 255}, "beige": {245, 245, 220, 255},
	"bisque": {255, 228, 196, 255}, "black": {0, 0, 0, 255}, "blanchedalmond": {255, 235, 205, 255},
	"blue": {0, 0, 255, 255}, "blueviolet": {138, 43, 226, 255}, "brown": {165, 42, 42, 255},
	"burlywood": {222, 184, 135, 255}, "cadetblue": {95, 158, 160, 255}, "chartreuse": {127, 255, 0, 255},
	"chocolate": {210, 105, 30, 255}, "coral": {255, 127, 80, 255}, "cornflowerblue": {100, 149, 237, 255},
	"cornsilk": {255, 248, 220, 255}, "crimson": {220, 20, 60, 255}, "cyan": {0, 255, 255, 255},
	"darkblue": {0, 0, 139, 255}, "darkcyan": {0, 139, 139, 255}, "darkgoldenrod": {184, 134, 11, 255},
	"darkgray": {169, 169, 169, 255}, "darkgreen": {0, 100, 0, 255}, "darkgrey": {169, 169, 169, 255},
	"darkkhaki": {189, 183, 107, 255}, "darkmagenta": {139, 0, 139, 255}, "darkolivegreen": {85, 107, 47, 255},
	"darkorange": {255, 140, 0, 255}, "darkorchid": {153, 50, 204, 255}, "darkred": {139, 0, 0, 255},
	"darksalmon": {233, 150, 122, 255}, "darkseagreen": {143, 188, 143, 255}, "darkslateblue": {72, 61, 139, 255},
	"darkslategray": {47, 79, 79, 255}, "darkslategrey": {47, 79, 79, 255}, "darkturquoise": {0, 206, 209, 255},
	"darkviolet": {148, 0, 211, 255}, "deeppink": {255, 20, 147, 255}, "deepskyblue": {0, 191, 255, 255},
	"dimgray": {105, 105, 105, 255}, "dimgrey": {105, 105, 105, 255}, "dodgerblue": {30, 144, 255, 255},
	"firebrick": {178, 34, 34, 255}, "floralwhite": {255, 250, 240, 255}, "forestgreen": {34, 139, 34, 255},
	"fuchsia": {255, 0, 255, 255}, "gainsboro": {220, 220, 220, 255}, "ghostwhite": {248, 248, 255, 255},
	"gold": {255, 215, 0, 255}, "goldenrod": {218, 165, 32, 255}, "gray": {128, 128, 128, 255},
	"green": {0, 128, 0, 255}, "greenyellow": {173, 255, 47, 255}, "grey": {128, 128, 128, 255},
	"honeydew": {240, 255, 240, 255}, "hotpink": {255, 105, 180, 255}, "indianred": {205, 92, 92, 255},
	"indigo": {75, 0, 130, 255}, "ivory": {255, 255, 240, 255}, "khaki": {240, 230, 140, 255},
	"lavender": {230, 230, 250, 255}, "lavenderblush": {255, 240, 245, 255}, "lawngreen": {124, 252, 0, 255},
	"lemonchiffon": {255, 250, 205, 255}, "lightblue": {173, 216, 230, 255}, "lightcoral": {240, 128, 128, 255},
	"lightcyan": {224, 255, 255, 255}, "lightgoldenrodyellow": {250, 250, 210, 255}, "lightgray": {211, 211, 211, 255},
	"lightgreen": {144, 238, 144, 255}, "lightgrey": {211, 211, 211, 255}, "lightpink": {255, 182, 193, 255},
	"lightsalmon": {255, 160, 122, 255}, "lightseagreen": {32, 178, 170, 255}, "lightskyblue": {135, 206, 250, 255},
	"lightslategray": {119, 136, 153, 255}, "lightslategrey": {119, 136, 153, 255}, "lightsteelblue": {176, 196, 222, 255},
	"lightyellow": {255, 255, 224, 255}, "lime": {0, 255, 0, 255}, "limegreen": {50, 205, 50, 255},
	"linen": {250, 240, 230, 255}, "magenta": {255, 0, 255, 255}, "maroon": {128, 0, 0, 255},
	"mediumaquamarine": {102, 205, 170, 255}, "mediumblue": {0, 0, 205, 255}, "mediumorchid": {186, 85, 211, 255},
	"mediumpurple": {147, 112, 219, 255}, "mediumseagreen": {60, 179, 113, 255}, "mediumslateblue": {123, 104, 238, 255},
	"mediumspringgreen": {0, 250, 154, 255}, "mediumturquoise": {72, 209, 204, 255}, "mediumvioletred": {199, 21, 133, 255},
	"midnightblue": {25, 25, 112, 255}, "mintcream": {245, 255, 250, 255}, "mistyrose": {255, 228, 225, 255},
	"moccasin": {255, 228, 181, 255}, "navajowhite": {255, 222, 173, 255}, "navy": {0, 0, 128, 255},
	"oldlace": {253, 245, 230, 255}, "olive": {128, 128, 0, 255}, "olivedrab": {107, 142, 35, 255},
	"orange": {255, 165, 0, 255}, "orangered": {255, 69, 0, 255}, "orchid": {218, 112, 214, 255},
	"palegoldenrod": {238, 232, 170, 255}, "palegreen": {152, 251, 152, 255}, "paleturquoise": {175, 238, 238, 255},
	"palevioletred": {219, 112, 147, 255}, "papayawhip": {255, 239, 213, 255}, "peachpuff": {255, 218, 185, 255},
	"peru": {205, 133, 63, 255}, "pink": {255, 192, 203, 255}, "plum": {221, 160, 221, 255},
	"powderblue": {176, 224, 230, 255}, "purple": {128, 0, 128, 255}, "rebeccapurple": {102, 51, 153, 255},
	"red": {255, 0, 0, 255}, "rosybrown": {188, 143, 143, 255}, "royalblue": {65, 105, 225, 255},
	"saddlebrown": {139, 69, 19, 255}, "salmon": {250, 128, 114, 255}, "sandybrown": {244, 164, 96, 255},
	"seagreen": {46, 139, 87, 255}, "seashell": {255, 245, 238, 255}, "sienna": {160, 82, 45, 255},
	"silver": {192, 192, 192, 255}, "skyblue": {135, 206, 235, 255}, "slateblue": {106, 90, 205, 255},
	"slategray": {112, 128, 144, 255}, "slategrey": {112, 128, 144, 255}, "snow": {255, 250, 250, 255},
	"springgreen": {0, 255, 127, 255}, "steelblue": {70, 130, 180, 255}, "tan": {210, 180, 140, 255},
	"teal": {0, 128, 128, 255}, "thistle": {216, 191, 216, 255}, "tomato": {255, 99, 71, 255},
	"turquoise": {64, 224, 208, 255}, "violet": {238, 130, 238, 255}, "wheat": {245, 222, 179, 255},
	"white": {255, 255, 255, 255}, "whitesmoke": {245, 245, 245, 255}, "yellow": {255, 255, 0, 255},
	"yellowgreen": {154, 205, 50, 255},
}
