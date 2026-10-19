package journey

import (
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/text"
)

// Laying out text as Mermaid's journey and timeline diagrams draw it, as Pele's
// diagrams/common/fit.ts does.

// brAt is the end of the <br> tag at byte i of s, matching /<br\s*\/?>/i, or -1.
func brAt(s string, i int) int {
	if i+3 > len(s) || s[i] != '<' || s[i+1]|32 != 'b' || s[i+2]|32 != 'r' {
		return -1
	}
	j := chars.SpaceRun(s, i+3)
	if j < len(s) && s[j] == '/' {
		j++
	}
	if j < len(s) && s[j] == '>' {
		return j + 1
	}
	return -1
}

// plainLabel lays out text as plain text in which only <br> is markup, so that other tags and
// icon names show as written instead of being read as formatting, as Pele's plainLabel does.
func plainLabel(raw string, measure text.Measurer, size, max float64, face text.Face) text.Label {
	s := raw
	if strings.Contains(s, "<") {
		// /<(?!br\s*\/?>)/gi, replaced with &lt;.
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			if s[i] == '<' && brAt(s, i) < 0 {
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
	return text.Layout(s, text.Options{Measurer: measure, Size: size, MaxWidth: max, Face: face})
}

// unitEnd is the end of the piece of text at byte i that fitLabel may cut a word between,
// matching /<br\s*\/?>|ﬂ°°?\w+¶ß|&#?\w+;|[\ud800-\udbff][\udc00-\udfff]|[\s\S]/i: a line
// break, an encoded entity, an entity, or one character.
func unitEnd(s string, i int) int {
	if e := brAt(s, i); e >= 0 {
		return e
	}
	words := func(j int) int {
		for j < len(s) && chars.IsWord(int(s[j])) {
			j++
		}
		return j
	}
	const open, degree, close = "ﬂ°", "°", "¶ß"
	if strings.HasPrefix(s[i:], open) {
		j := i + len(open)
		if strings.HasPrefix(s[j:], degree) {
			j += len(degree)
		}
		if k := words(j); k > j && strings.HasPrefix(s[k:], close) {
			return k + len(close)
		}
	}
	if s[i] == '&' {
		j := i + 1
		if j < len(s) && s[j] == '#' {
			j++
		}
		if k := words(j); k > j && k < len(s) && s[k] == ';' {
			return k + 1
		}
	}
	_, size := chars.UnitAt(s, i)
	return i + size
}

// fitLabel lays out text as plainLabel does, and also cuts any word too long to fit max, for
// a box of a fixed width, as Pele's fitLabel does: plainLabel wraps at spaces only.
func fitLabel(raw string, measure text.Measurer, size, max float64, face text.Face) text.Label {
	label := plainLabel(raw, measure, size, max, face)
	if raw == "" || label.Width <= max {
		return label
	}
	var out strings.Builder
	width := 0.0
	for i := 0; i < len(raw); {
		end := unitEnd(raw, i)
		unit := raw[i:end]
		i = end
		n := chars.Len16(unit)
		if chars.Trim(unit) == "" || n > 2 && unit[0] == '<' {
			width = 0
		} else {
			measured := unit
			if n > 2 {
				measured = "x"
			}
			w := measure.Width(measured, size, face)
			if width > 0 && width+w > max {
				out.WriteByte('\n')
				width = 0
			}
			width += w
		}
		out.WriteString(unit)
	}
	return plainLabel(out.String(), measure, size, max, face)
}
