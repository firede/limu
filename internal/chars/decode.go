package chars

import "unicode/utf8"

// Decode is a text as a browser's decoder of UTF-8 reads it: valid UTF-8 as it is, and each
// maximal subpart of an ill-formed sequence replaced with U+FFFD, as the WHATWG Encoding
// Standard decodes and as Unicode recommends. A maximal subpart is the longest run of bytes
// that begins a well-formed sequence, or else a single byte: "\xe2\x82" that a third byte
// does not finish is one U+FFFD, and the "\xed\xa0\xbd" a lone surrogate is written as in
// WTF-8 is three, since no well-formed sequence begins "\xed\xa0".
//
// Go's strings.ToValidUTF8 replaces each run of bad bytes with one replacement instead, and
// ranging over a string gives one U+FFFD for each bad byte; neither is what JavaScript
// would be handed.
func Decode(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	out := make([]byte, 0, len(s)+8)
	for i := 0; i < len(s); {
		if r, n := utf8.DecodeRuneInString(s[i:]); r != utf8.RuneError || n > 1 {
			out = append(out, s[i:i+n]...)
			i += n
			continue
		}
		out = append(out, "�"...)
		i += subpart(s[i:])
	}
	return string(out)
}

// subpart is the length of the maximal subpart of the ill-formed sequence s begins with.
func subpart(s string) int {
	b := s[0]
	var n int                        // the bytes the sequence b leads takes
	lo, hi := byte(0x80), byte(0xbf) // the range of its second byte
	switch {
	case b >= 0xc2 && b <= 0xdf:
		n = 2
	case b == 0xe0:
		n, lo = 3, 0xa0
	case b == 0xed:
		n, hi = 3, 0x9f
	case b >= 0xe1 && b <= 0xef:
		n = 3
	case b == 0xf0:
		n, lo = 4, 0x90
	case b == 0xf4:
		n, hi = 4, 0x8f
	case b >= 0xf1 && b <= 0xf3:
		n = 4
	default:
		return 1
	}
	i := 1
	for ; i < n && i < len(s); i++ {
		if c := s[i]; c < lo || c > hi {
			break
		}
		lo, hi = 0x80, 0xbf
	}
	return i
}
