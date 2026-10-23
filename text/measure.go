package text

// Face is how a run of text is set: bold, italic, monospaced, lowered or raised a little
// and smaller, underlined, or a mix. Its bits are those of Pele's Style.
type Face uint8

const (
	Bold Face = 1 << iota
	Italic
	Mono
	Sub // set lower and smaller, as <sub> is
	Sup // set higher and smaller, as <sup> is
	Underline
)

// shifted is the faces set smaller and off the baseline.
const shifted = Sub | Sup

// smaller is how much smaller a Sub or Sup span is set, as Pele's measurers take
// font-size="smaller" to be.
const smaller = 1.2

// A Measurer gives the advance width of text set in a face at a size, in the units the
// diagram is laid out in. Layout depends on it, so a diagram is drawn in the faces it was
// measured in. Layout asks it only of Bold, Italic and Mono: it measures a Sub or Sup span at
// the smaller size it is set at, and an underline takes no room.
type Measurer interface {
	Width(s string, size float64, f Face) float64
}

// Width is how wide m measures s in the face f at size, as a label's span is measured: a Sub
// or Sup span at size / 1.2, as Pele's measurers set it, and m asked only of Bold, Italic and
// Mono.
func Width(m Measurer, s string, size float64, f Face) float64 {
	if f&shifted != 0 {
		size = size / smaller
	}
	return m.Width(s, size, f&(Bold|Italic|Mono))
}

// Arial estimates widths from a table of Arial's advances, with no font at hand. Wide East
// Asian characters take an em, other characters past ASCII 0.58 of one, and monospaced text
// 0.6 of an em a character. Its widths are those of Pele's metricsMeasurer, to the last bit.
var Arial Measurer = arial{}

// Advance widths of printable ASCII in thousandths of an em, two base-36 digits each, from
// Pele's src/text/metrics.ts.
const (
	arialRegular = "7q7q9vfgfgopij5b9999atg87q997q7qfgfgfgfgfgfgfgfgfgfg7q7qg8g8g8fgs7ijijk2k2ijgzlmk27qdwijfgn5k2lmijlmk2ijgzk2ijq8ijijgz7q7q7qd1fg99fgfgdwfgfg7qfgfg6666dw66n5fgfgfgfg99dw7qfgdwk2dwdwdw9a789ag8"
	arialBold    = "7q99d6fgfgopk26m9999atg87q997q7qfgfgfgfgfgfgfgfgfgfg9999g8g8g8gzr3k2k2k2k2ijgzlmk27qfgk2gzn5k2lmijlmk2ijgzk2ijq8ijijgz997q99g8fg99fggzfggzfg99gzgz7q7qfg7qopgzgzgzgzatfg99gzfglmfgfgdwat7satg8"
)

var arialTables = [2][95]uint16{unpack(arialRegular), unpack(arialBold)}

func unpack(packed string) [95]uint16 {
	var out [95]uint16
	digit := func(c byte) uint16 {
		if c <= '9' {
			return uint16(c - '0')
		}
		return uint16(c-'a') + 10
	}
	for i := range out {
		out[i] = digit(packed[2*i])*36 + digit(packed[2*i+1])
	}
	return out
}

type arial struct{}

// Width counts as JavaScript does, by UTF-16 code unit: a character past the Basic
// Multilingual Plane is a high surrogate, wide, and a low one, of no width.
func (arial) Width(s string, size float64, f Face) float64 {
	mono := f&Mono != 0
	table := &arialTables[0]
	if f&Bold != 0 {
		table = &arialTables[1]
	}
	units := 0
	for _, c := range s {
		switch {
		case c < 32:
		case c < 127:
			if mono {
				units += 600
			} else {
				units += int(table[c-32])
			}
		case c > 0xffff || isWide(c):
			if mono {
				units += 1200
			} else {
				units += 1000
			}
		case isZeroWidth(c):
		default:
			if mono {
				units += 600
			} else {
				units += 580
			}
		}
	}
	return float64(units) * size / 1000
}

func isWide(c rune) bool {
	return c >= 0x1100 && c <= 0x115f ||
		c >= 0x2e80 && c <= 0xa4cf ||
		c >= 0xac00 && c <= 0xd7a3 ||
		c >= 0xf900 && c <= 0xfaff ||
		c >= 0xfe30 && c <= 0xfe4f ||
		c >= 0xff00 && c <= 0xff60 ||
		c >= 0xffe0 && c <= 0xffe6
}

func isZeroWidth(c rune) bool {
	return c == 0xad ||
		c >= 0x300 && c <= 0x36f ||
		c >= 0x200b && c <= 0x200f ||
		c >= 0xfe00 && c <= 0xfe0f ||
		c == 0xfeff
}
