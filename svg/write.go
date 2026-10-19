package svg

import (
	"math"
	"strconv"
	"strings"
)

// num writes a number rounded to hundredths, as Pele's num does: 12, 12.5, 12.05.
func num(b *strings.Builder, f float64) {
	n := math.Floor(f*100 + 0.5)
	if n == 0 || math.IsNaN(n) {
		b.WriteByte('0')
		return
	}
	if math.Abs(n) >= 1e11 {
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

// esc writes text escaped for XML, without the control characters XML does not allow.
func esc(b *strings.Builder, s string) {
	last := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		var rep string
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
		default:
			continue
		}
		b.WriteString(s[last:i])
		b.WriteString(rep)
		last = i + 1
	}
	b.WriteString(s[last:])
}
