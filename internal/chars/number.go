package chars

import (
	"math"
	"math/big"
	"strconv"
	"strings"
)

// NumberString is JavaScript's String() of a number: the fewest digits that read back as it,
// written out in full from 1e-7 up to 1e21 and with an exponent outside that.
func NumberString(f float64) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case f == 0:
		return "0"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f < 0:
		return "-" + NumberString(-f)
	}
	e := strconv.FormatFloat(f, 'e', -1, 64)
	mant, exp, _ := strings.Cut(e, "e")
	digits := strings.Replace(mant, ".", "", 1)
	x, _ := strconv.Atoi(exp)
	k := len(digits)
	n := x + 1
	switch {
	case k <= n && n <= 21:
		return digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return "0." + strings.Repeat("0", -n) + digits
	}
	sign, e10 := "+", n-1
	if e10 < 0 {
		sign, e10 = "-", -e10
	}
	exponent := strconv.Itoa(e10)
	if k == 1 {
		return digits + "e" + sign + exponent
	}
	return digits[:1] + "." + digits[1:] + "e" + sign + exponent
}

// Number is JavaScript's Number() of a string: white space around it is ignored, an empty
// string is 0, and anything that is not a number as JavaScript writes one is NaN.
func Number(s string) float64 {
	s = Trim(s)
	if s == "" {
		return 0
	}
	if len(s) > 2 && s[0] == '0' {
		base := 0
		switch s[1] {
		case 'x', 'X':
			base = 16
		case 'o', 'O':
			base = 8
		case 'b', 'B':
			base = 2
		}
		if base != 0 {
			return digitsValue(s[2:], base, false)
		}
	}
	body := s
	if body[0] == '+' || body[0] == '-' {
		body = body[1:]
	}
	if body == "Infinity" {
		if s[0] == '-' {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	if !isDecimal(body) {
		return math.NaN()
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// isDecimal reports whether s is digits with at most one point and some digit, and then an
// optional exponent: JavaScript's StrUnsignedDecimalLiteral.
func isDecimal(s string) bool {
	i, digits := 0, 0
	for i < len(s) && IsDigit(int(s[i])) {
		i++
		digits++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && IsDigit(int(s[i])) {
			i++
			digits++
		}
	}
	if digits == 0 {
		return false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		from := i
		for i < len(s) && IsDigit(int(s[i])) {
			i++
		}
		if i == from {
			return false
		}
	}
	return i == len(s)
}

// digitsValue is the value of s in base, rounded to the nearest number as JavaScript rounds
// it. With prefix set, it reads the digits s starts with, as parseInt does; otherwise all of
// s must be digits. Either way no digits is NaN.
func digitsValue(s string, base int, prefix bool) float64 {
	end := 0
	for end < len(s) && digitOf(s[end]) < base {
		end++
	}
	if end == 0 || !prefix && end != len(s) {
		return math.NaN()
	}
	if end <= 12 {
		n, _ := strconv.ParseInt(s[:end], base, 64)
		return float64(n)
	}
	n, _ := new(big.Int).SetString(s[:end], base)
	f, _ := new(big.Float).SetInt(n).Float64()
	return f
}

func digitOf(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'z':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'Z':
		return int(b-'A') + 10
	}
	return 99
}

// ParseInt is JavaScript's parseInt(s, radix) for a radix from 2 to 36: white space before
// the number is skipped, a sign is read, and so is 0x before a number in base 16; then as
// many digits as there are. With none, it is NaN.
func ParseInt(s string, radix int) float64 {
	s = TrimStart(s)
	neg := false
	if s != "" && (s[0] == '+' || s[0] == '-') {
		neg = s[0] == '-'
		s = s[1:]
	}
	if radix == 16 && len(s) >= 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		s = s[2:]
	}
	var f float64
	if radix == 10 {
		end := 0
		for end < len(s) && IsDigit(int(s[end])) {
			end++
		}
		if end == 0 {
			return math.NaN()
		}
		f, _ = strconv.ParseFloat(s[:end], 64)
	} else {
		f = digitsValue(s, radix, true)
	}
	if neg {
		return -f
	}
	return f
}
