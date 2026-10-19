package treemap

import (
	"math"
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
)

// numberSpec is what Pele's formatter reads of a specifier of d3-format, the parts of
// RE_SPEC's match it uses: a sign of +, a $, a comma for thousands separators, a precision,
// ~ to trim trailing zeros, and the type, 0 for none.
type numberSpec struct {
	plus, dollar, comma, trim bool
	precision                 int
	hasPrecision              bool
	typ                       byte
}

// isAlign reports whether c is one of d3-format's alignments, [<>=^].
func isAlign(c byte) bool { return c == '<' || c == '>' || c == '=' || c == '^' }

// readSpec matches Pele's RE_SPEC,
// /^(?:.?[<>=^])?([-+( ])?(\$)?0?\d*(,)?(?:\.(\d+))?(~)?([a-zA-Z%])?$/, against s. The fill
// and alignment are tried as a regular expression tries them: a character and an alignment,
// an alignment alone, and neither. A character is a UTF-16 code unit other than a line
// terminator, so one outside the Basic Multilingual Plane, two units, is no fill. What
// follows has one way to match at most, as each part takes characters none after it can.
func readSpec(s string) (numberSpec, bool) {
	var starts []int
	if r, size := chars.UnitAt(s, 0); r >= 0 && !chars.IsLineEnd(r) && size < len(s) && isAlign(s[size]) {
		if !(r >= 0xd800 && r <= 0xdbff) {
			starts = append(starts, size+1)
		}
	}
	if len(s) > 0 && isAlign(s[0]) {
		starts = append(starts, 1)
	}
	starts = append(starts, 0)
	for _, i := range starts {
		if spec, ok := readSpecRest(s, i); ok {
			return spec, true
		}
	}
	return numberSpec{}, false
}

// readSpecRest matches what follows the fill and alignment of RE_SPEC, from byte i to the
// end of s.
func readSpecRest(s string, i int) (numberSpec, bool) {
	var spec numberSpec
	at := func(c byte) bool { return i < len(s) && s[i] == c }
	digit := func(k int) bool { return k < len(s) && chars.IsDigit(int(s[k])) }
	if i < len(s) && strings.IndexByte("-+( ", s[i]) >= 0 {
		spec.plus = s[i] == '+'
		i++
	}
	if at('$') {
		spec.dollar = true
		i++
	}
	for digit(i) {
		i++
	}
	if at(',') {
		spec.comma = true
		i++
	}
	if at('.') && digit(i+1) {
		i++
		// Number(m[4]), at most 20, which a precision of more digits than two always is.
		p := 0
		for ; digit(i); i++ {
			p = min(21, p*10+int(s[i]-'0'))
		}
		spec.precision, spec.hasPrecision = min(p, 20), true
	}
	if at('~') {
		spec.trim = true
		i++
	}
	if i < len(s) && (chars.IsLetter(int(s[i])) || s[i] == '%') {
		spec.typ = s[i]
		i++
	}
	return spec, i == len(s)
}

// group puts a comma between each three digits of digits from its end, as Pele's group does,
// whatever the characters are.
func group(digits string) string {
	out := ""
	for i := len(digits); i > 0; i -= 3 {
		piece := digits[max(0, i-3):i]
		if out != "" {
			out = piece + "," + out
		} else {
			out = piece
		}
	}
	return out
}

// formatter is a function that writes values as a specifier of d3-format says, as Pele's
// formatter makes it: the parts of d3-format that treemap values use, an optional $, a comma
// for thousands separators, a precision, and a type of f, %, d, e or none. Fill, alignment
// and width are read and ignored. Anything else falls back to the default, ",".
func formatter(s string) func(float64) string {
	// Mermaid documents $0,0 for a dollar amount with thousands separators.
	if s == "$0,0" {
		s = "$,"
	}
	spec, ok := readSpec(s)
	if !ok {
		spec, _ = readSpec(",")
	}
	precision := func(fallback int) int {
		if spec.hasPrecision {
			return spec.precision
		}
		return fallback
	}
	return func(value float64) string {
		abs := math.Abs(value)
		var body, suffix string
		switch spec.typ {
		case 'f':
			body = jsmath.ToFixed(abs, precision(6))
		case '%':
			body = jsmath.ToFixed(abs*100, precision(6))
			suffix = "%"
		case 'd':
			body = jsmath.ToFixed(jsmath.Round(abs), 0)
		case 'e':
			body = jsmath.ToExponential(abs, precision(6))
		default:
			body = chars.NumberString(chars.Number(jsmath.ToPrecision(abs, max(1, precision(12)))))
		}
		if spec.trim && strings.Contains(body, ".") && !strings.Contains(body, "e") {
			// body.replace(/\.?0+$/, ""): the zeros at the end, and the point before them.
			body = strings.TrimRight(body, "0")
			body = strings.TrimSuffix(body, ".")
		}
		if spec.comma && !strings.Contains(body, "e") {
			if dot := strings.IndexByte(body, '.'); dot >= 0 {
				body = group(body[:dot]) + body[dot:]
			} else {
				body = group(body)
			}
		}
		sign := ""
		switch {
		case value < 0:
			sign = "-"
		case spec.plus:
			sign = "+"
		}
		if spec.dollar {
			sign += "$"
		}
		return sign + body + suffix
	}
}
