package svg

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/internal/jsmath"
)

// num writes a number rounded to hundredths, as Pele's num does: 12, 12.5, 12.05. The
// product is rounded on its own, as JavaScript rounds it, before Math.round takes it. NaN
// and the infinities, which Pele writes as words SVG does not read, are written as 0.
func num(b *strings.Builder, f float64) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		b.WriteByte('0')
		return
	}
	n := jsmath.Round(float64(f * 100))
	if n == 0 {
		b.WriteByte('0')
		return
	}
	if math.Abs(n) >= 1e11 {
		if math.IsInf(n, 0) {
			// Too large to count in hundredths; it has no fraction to round.
			b.WriteString(strconv.FormatFloat(f, 'f', -1, 64))
			return
		}
		b.WriteString(strconv.FormatFloat(n/100, 'f', -1, 64))
		return
	}
	i := int64(n)
	if i < 0 {
		b.WriteByte('-')
		i = -i
	}
	var buf [24]byte
	b.Write(strconv.AppendInt(buf[:0], i/100, 10))
	switch frac := i % 100; {
	case frac == 0:
	case frac%10 == 0:
		b.WriteByte('.')
		b.WriteByte(byte('0' + frac/10))
	default:
		b.WriteByte('.')
		b.WriteByte(byte('0' + frac/10))
		b.WriteByte(byte('0' + frac%10))
	}
}

// esc writes text escaped for XML, with what XML 1.0 does not allow left out: the control
// characters but tab, line feed and carriage return, DEL, and U+FFFE and U+FFFF. Bytes that
// are not UTF-8 are written as U+FFFD, as a browser decodes them; a surrogate's encoding is
// not UTF-8, and is three of them.
func esc(b *strings.Builder, s string) {
	last := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		var rep string
		n := 1
		switch {
		case c == '&':
			rep = "&amp;"
		case c == '<':
			rep = "&lt;"
		case c == '>':
			rep = "&gt;"
		case c == '"':
			rep = "&quot;"
		case c < 0x20 && c != '\t' && c != '\n' && c != '\r', c == 0x7f:
			rep = ""
		case c < utf8.RuneSelf:
			continue
		default:
			r, size := utf8.DecodeRuneInString(s[i:])
			switch {
			case r == utf8.RuneError && size == 1:
				rep = "�"
			case r == 0xfffe || r == 0xffff:
				rep, n = "", size
			default:
				i += size - 1
				continue
			}
		}
		b.WriteString(s[last:i])
		b.WriteString(rep)
		i += n - 1
		last = i + 1
	}
	b.WriteString(s[last:])
}
