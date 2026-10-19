// Package chars reads text the way the JavaScript that limu's parsers are ported from reads
// it: by UTF-16 code units, with JavaScript's white space, its trim and toLowerCase, and its
// conversions between numbers and strings.
//
// The parsers keep their text as Go strings and step through it by bytes, which is what
// JavaScript does by code units wherever the characters looked for are ASCII. Where a
// character outside ASCII is read, UnitAt gives the code unit JavaScript's charCodeAt would
// give at the same place: the character itself in the Basic Multilingual Plane, and the
// first of its two surrogates above it. Positions shown to people, such as the column of a
// syntax error, are counted in code units with Len16.
package chars

import (
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

// The tests on code units, as Pele's src/util/chars.ts has them. Past the end of a string a
// lexer reads -1, which is none of these, as charCodeAt's NaN is none of them.

// IsSpace reports whether code unit c is white space to a regular expression's \s.
func IsSpace(c int) bool {
	if c < 128 {
		return c == 32 || c >= 9 && c <= 13
	}
	return c == 0xa0 || c == 0x1680 || c >= 0x2000 && c <= 0x200a || c == 0x2028 || c == 0x2029 ||
		c == 0x202f || c == 0x205f || c == 0x3000 || c == 0xfeff
}

// IsBlank reports whether c is a space or a tab.
func IsBlank(c int) bool { return c == 32 || c == 9 }

// IsLineEnd reports whether c ends a line: what a regular expression's `.` does not match.
func IsLineEnd(c int) bool { return c == 10 || c == 13 || c == 0x2028 || c == 0x2029 }

// IsDigit reports whether c is an ASCII digit.
func IsDigit(c int) bool { return c >= 48 && c <= 57 }

// IsLetter reports whether c is an ASCII letter.
func IsLetter(c int) bool { return c >= 65 && c <= 90 || c >= 97 && c <= 122 }

// IsHex reports whether c is an ASCII hexadecimal digit.
func IsHex(c int) bool { return c >= 48 && c <= 57 || c >= 65 && c <= 70 || c >= 97 && c <= 102 }

// IsWord reports whether c is in a regular expression's \w.
func IsWord(c int) bool {
	return c >= 48 && c <= 57 || c >= 65 && c <= 90 || c >= 97 && c <= 122 || c == 95
}

// IsSpaceRune reports whether r is white space to \s and to trim.
func IsSpaceRune(r rune) bool { return IsSpace(int(r)) }

// UnitAt is the code unit JavaScript reads at byte i of s, with the number of bytes of the
// character it starts: -1 and 0 past the end. A character above the Basic Multilingual Plane
// reads as its high surrogate, and its bytes are all stepped over at once; no lexer stops
// on the low surrogate that JavaScript would read next, as it is no character a lexer looks
// for. A byte that is not UTF-8 reads as U+FFFD.
func UnitAt(s string, i int) (c, size int) {
	if i < 0 || i >= len(s) {
		return -1, 0
	}
	if b := s[i]; b < utf8.RuneSelf {
		return int(b), 1
	}
	r, n := utf8.DecodeRuneInString(s[i:])
	if r > 0xffff {
		hi, _ := utf16.EncodeRune(r)
		return int(hi), n
	}
	return int(r), n
}

// Unit is UnitAt without the size.
func Unit(s string, i int) int {
	c, _ := UnitAt(s, i)
	return c
}

// UnitBefore is the code unit JavaScript reads just before byte i of s: the low surrogate of
// a character above the Basic Multilingual Plane, and -1 at the start.
func UnitBefore(s string, i int) int {
	if i <= 0 || i > len(s) {
		return -1
	}
	if b := s[i-1]; b < utf8.RuneSelf {
		return int(b)
	}
	r, _ := utf8.DecodeLastRuneInString(s[:i])
	if r > 0xffff {
		_, lo := utf16.EncodeRune(r)
		return int(lo)
	}
	return int(r)
}

// Len16 is the length of s in UTF-16 code units, JavaScript's length.
func Len16(s string) int {
	n := 0
	for i := 0; i < len(s); {
		if s[i] < utf8.RuneSelf {
			n++
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if r > 0xffff {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// Trim is String.prototype.trim.
func Trim(s string) string { return strings.TrimFunc(s, IsSpaceRune) }

// TrimStart is String.prototype.trimStart.
func TrimStart(s string) string { return strings.TrimLeftFunc(s, IsSpaceRune) }

// TrimEnd is String.prototype.trimEnd.
func TrimEnd(s string) string { return strings.TrimRightFunc(s, IsSpaceRune) }

// LeadingSpace is the number of code units trimStart takes off s: s.length -
// s.trimStart().length. White space is one code unit a character.
func LeadingSpace(s string) int {
	n := 0
	for _, r := range s {
		if !IsSpaceRune(r) {
			break
		}
		n++
	}
	return n
}

// DropUnits takes the first n characters off s, as s.slice(n) does where they are all of
// one code unit, as the white space LeadingSpace counts is.
func DropUnits(s string, n int) string {
	i := 0
	for ; n > 0 && i < len(s); n-- {
		_, size := utf8.DecodeRuneInString(s[i:])
		i += size
	}
	return s[i:]
}

// SpaceRun is the end of the run of white space that starts at byte i of s.
func SpaceRun(s string, i int) int {
	for i < len(s) {
		c, size := UnitAt(s, i)
		if !IsSpace(c) {
			break
		}
		i += size
	}
	return i
}

// DropIndents is s.replace(/\n\s+/g, "\n"), as the diagrams set an accessible description:
// the white space after each line feed taken off, with the line breaks in it.
func DropIndents(s string) string {
	if !strings.Contains(s, "\n") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		b.WriteByte(s[i])
		if s[i] == '\n' {
			i = SpaceRun(s, i+1) - 1
		}
	}
	return b.String()
}

// ToLower is String.prototype.toLowerCase. Go's simple case mapping agrees with JavaScript's
// full one except for U+0130, which JavaScript lowers to an i and a combining dot above, and
// a capital sigma that ends a word, which it lowers to a final sigma by Unicode's
// Final_Sigma condition.
func ToLower(s string) string {
	if strings.ContainsRune(s, 0x130) {
		s = strings.ReplaceAll(s, "\u0130", "i\u0307")
	}
	if !strings.ContainsRune(s, 0x3a3) {
		return strings.ToLower(s)
	}
	var b strings.Builder
	b.Grow(len(s))
	for i, r := range s {
		if r == 0x3a3 && finalSigma(s, i) {
			b.WriteRune(0x3c2)
		} else {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// finalSigma reports whether the capital sigma at byte i of s ends a word, by Unicode's
// Final_Sigma condition: a cased letter comes before it and none after it, with only
// case-ignorable characters between. A character that is both is skipped as ignorable, as
// ICU's ucase_toFullLower does, which V8 lowers strings with.
func finalSigma(s string, i int) bool {
	before := false
	for j := i; j > 0; {
		r, size := utf8.DecodeLastRuneInString(s[:j])
		j -= size
		if caseIgnorable(r) {
			continue
		}
		before = cased(r)
		break
	}
	if !before {
		return false
	}
	for _, r := range s[i+utf8.RuneLen(0x3a3):] {
		if caseIgnorable(r) {
			continue
		}
		return !cased(r)
	}
	return true
}

// cased is Unicode's Cased property: the upper, lower and title case letters, and the
// others Unicode counts as upper or lower case.
func cased(r rune) bool {
	return unicode.In(r, unicode.Lu, unicode.Ll, unicode.Lt, unicode.Other_Lowercase, unicode.Other_Uppercase)
}

// caseIgnorable is Unicode's Case_Ignorable property: marks, format characters, modifier
// letters and symbols, and the characters that may sit inside a word, as an apostrophe or a
// period does (Word_Break MidLetter, MidNumLet and Single_Quote).
func caseIgnorable(r rune) bool {
	switch r {
	case '\'', '.', ':', 0xb7, 0x387, 0x55f, 0x5f4, 0x2018, 0x2019, 0x2024, 0x2027, 0xfe13, 0xfe52, 0xfe55, 0xff07, 0xff0e, 0xff1a:
		return true
	}
	return unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf, unicode.Lm, unicode.Sk)
}

// EqualFoldASCII reports whether a and b are the same but for the case of ASCII letters, as a
// regular expression's i flag without u compares them, and as the URL Standard's ASCII
// case-insensitive match does.
func EqualFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if x >= 'A' && x <= 'Z' {
			x += 32
		}
		if y >= 'A' && y <= 'Z' {
			y += 32
		}
		if x != y {
			return false
		}
	}
	return true
}

// HasPrefixFold reports whether s starts with prefix but for the case of ASCII letters.
func HasPrefixFold(s, prefix string) bool {
	return len(s) >= len(prefix) && EqualFoldASCII(s[:len(prefix)], prefix)
}
