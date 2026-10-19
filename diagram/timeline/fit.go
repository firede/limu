package timeline

import (
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/text"
)

// Laying out text the way Mermaid's journey and timeline diagrams draw it, as Pele's
// diagrams/common/fit.ts does.

// brAt is the length of a <br> tag at i of s, as /<br\s*\/?>/i matches it, or 0.
func brAt(s string, i int) int {
	if i+3 > len(s) || s[i] != '<' || s[i+1]|32 != 'b' || s[i+2]|32 != 'r' {
		return 0
	}
	j := i + 3
	for j < len(s) {
		c, size := chars.UnitAt(s, j)
		if !chars.IsSpace(c) {
			break
		}
		j += size
	}
	if j < len(s) && s[j] == '/' {
		j++
	}
	if j < len(s) && s[j] == '>' {
		return j + 1 - i
	}
	return 0
}

// plainLabel lays out text as plain text in which only <br> is markup, so that other tags
// and icon names show as written instead of being read as formatting, as Pele's plainLabel
// does: a < that starts no <br> is written &lt;, and :fa- is written &colon;fa-.
func plainLabel(raw string, measure text.Measurer, size, max float64, base text.Face) text.Label {
	s := raw
	if strings.Contains(s, "<") {
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			if s[i] == '<' && brAt(s, i) == 0 {
				b.WriteString("&lt;")
			} else {
				b.WriteByte(s[i])
			}
		}
		s = b.String()
	}
	if strings.Contains(s, ":fa-") {
		s = strings.ReplaceAll(s, ":fa-", "&colon;fa-")
	}
	return text.Layout(s, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: base})
}

// isWordByte reports whether b is in a regular expression's \w.
func isWordByte(b byte) bool { return chars.IsWord(int(b)) }

// unitAt is the length in bytes of the piece of s at i that a word can be cut before, as
// Pele's RE_UNIT finds it: a line break, an encoded entity, a placeholder of one, or one
// character, a surrogate pair whole.
func unitAt(s string, i int) int {
	if n := brAt(s, i); n > 0 {
		return n
	}
	// ﬂ°°?\w+¶ß, the placeholder of an entity.
	if strings.HasPrefix(s[i:], "ﬂ°") {
		j := i + len("ﬂ°")
		if strings.HasPrefix(s[j:], "°") {
			// \w+ cannot take ° itself, so the second ° is the optional one whenever it is there.
			j += len("°")
		}
		k := j
		for k < len(s) && isWordByte(s[k]) {
			k++
		}
		if k > j && strings.HasPrefix(s[k:], "¶ß") {
			return k + len("¶ß") - i
		}
	}
	// &#?\w+;, where \w+ cannot take a #, so #? takes one whenever it is there.
	if s[i] == '&' {
		j := i + 1
		if j < len(s) && s[j] == '#' {
			j++
		}
		k := j
		for k < len(s) && isWordByte(s[k]) {
			k++
		}
		if k > j && k < len(s) && s[k] == ';' {
			return k + 1 - i
		}
	}
	_, size := utf8.DecodeRuneInString(s[i:])
	return size
}

// fitLabel lays out text as plainLabel does, and for a box of fixed width also cuts any word
// too long to fit, as Pele's fitLabel does: text.Layout wraps at spaces only.
func fitLabel(raw string, measure text.Measurer, size, max float64, base text.Face) text.Label {
	label := plainLabel(raw, measure, size, max, base)
	if raw == "" || label.Width <= max {
		return label
	}
	var out strings.Builder
	width := 0.0
	for i := 0; i < len(raw); {
		n := unitAt(raw, i)
		unit := raw[i : i+n]
		i += n
		units := chars.Len16(unit)
		if chars.Trim(unit) == "" || units > 2 && unit[0] == '<' {
			width = 0
		} else {
			measured := unit
			if units > 2 {
				measured = "x"
			}
			w := measure.Width(measured, size, base)
			if width > 0 && width+w > max {
				out.WriteByte('\n')
				width = 0
			}
			width += w
		}
		out.WriteString(unit)
	}
	return plainLabel(out.String(), measure, size, max, base)
}
