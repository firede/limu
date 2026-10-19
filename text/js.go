package text

import "strings"

// The ports read text the way the JavaScript they come from does, where it matters to what
// is measured: what counts as white space, and what trim takes off.

// isSpace reports whether r is white space to a JavaScript regular expression's \s and to
// String.prototype.trim.
func isSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return r >= 0x2000 && r <= 0x200a
}

// trim is String.prototype.trim.
func trim(s string) string { return strings.TrimFunc(s, isSpace) }

func trimStart(s string) string { return strings.TrimLeftFunc(s, isSpace) }

func trimEnd(s string) string { return strings.TrimRightFunc(s, isSpace) }

// isWord reports whether b is in a JavaScript regular expression's \w.
func isWord(b byte) bool {
	return b == '_' || b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}
