// Package jsjson is JavaScript's JSON.parse and JSON.stringify, as V8 has them, for the
// diagrams whose text holds JSON: the values JSON.parse gives, with an object's keys in the
// order JavaScript iterates them, and the messages of the errors V8 throws.
package jsjson

import (
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/firede/limu/internal/chars"
)

// Object is a JSON object as JSON.parse gives it. Its values are nil for null, a bool, a
// float64, a string, a []any or an *Object; its keys are in the order JavaScript gives an
// object's keys: those that are array indices first, in increasing order, then the others in
// the order they were first written. A key written twice keeps its first place and its last
// value.
type Object struct {
	Keys   []string
	Values map[string]any
}

// IsArrayIndex reports whether key is a canonical array index, which JavaScript orders before
// an object's other keys.
func IsArrayIndex(key string) bool {
	if key == "" || len(key) > 10 || key[0] == '0' && len(key) > 1 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if !chars.IsDigit(int(key[i])) {
			return false
		}
	}
	n, _ := strconv.ParseUint(key, 10, 64)
	return n < 1<<32-1
}

// newObject is an object of keys and values in the order they were written.
func newObject(keys []string, values []any) *Object {
	o := &Object{Values: make(map[string]any, len(keys))}
	var indices []string
	for i, k := range keys {
		if _, ok := o.Values[k]; !ok {
			if IsArrayIndex(k) {
				indices = append(indices, k)
			} else {
				o.Keys = append(o.Keys, k)
			}
		}
		o.Values[k] = values[i]
	}
	if len(indices) > 0 {
		slices.SortStableFunc(indices, func(a, b string) int {
			if len(a) != len(b) {
				return len(a) - len(b)
			}
			return strings.Compare(a, b)
		})
		o.Keys = append(indices, o.Keys...)
	}
	return o
}

// Decode is a string of code units, as a Go string holds it: a surrogate with no partner
// becomes U+FFFD.
func Decode(u []uint16) string { return string(utf16.Decode(u)) }

// jsonParser is V8's JSON.parse, as far as the diagrams' use of it shows: the value it gives, and
// the message of the SyntaxError it throws. It reads code units.
type jsonParser struct {
	text []uint16
	at   int
	err  string
}

// jsonFailed stops a parse once its error is set.
type jsonFailed struct{}

// The tokens V8's JSON parser reads a character as.
const (
	jIllegal = iota
	jEOS
	jString
	jNumber
	jTrue
	jFalse
	jNull
	jWhitespace
	jColon
	jComma
	jLBrace
	jRBrace
	jLBrack
	jRBrack
)

func jsonToken(c int) int {
	switch {
	case c < 0:
		return jEOS
	case c == '"':
		return jString
	case c == '-' || c >= '0' && c <= '9':
		return jNumber
	case c == 't':
		return jTrue
	case c == 'f':
		return jFalse
	case c == 'n':
		return jNull
	case c == ' ' || c == '\t' || c == '\n' || c == '\r':
		return jWhitespace
	case c == ':':
		return jColon
	case c == ',':
		return jComma
	case c == '{':
		return jLBrace
	case c == '}':
		return jRBrace
	case c == '[':
		return jLBrack
	case c == ']':
		return jRBrack
	}
	return jIllegal
}

func (j *jsonParser) cur() int {
	if j.at >= len(j.text) {
		return -1
	}
	return int(j.text[j.at])
}

func (j *jsonParser) next() int {
	j.at++
	return j.cur()
}

func (j *jsonParser) skip() {
	for {
		c := j.cur()
		if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
			return
		}
		j.at++
	}
}

// fileLocation is the line and column of the cursor, as V8's CalculateFileLocation counts
// them: a carriage return and a line feed after it end one line.
func (j *jsonParser) fileLocation() (int, int) {
	line := 1
	lastBreak := 0
	k := 0
	for ; k < j.at; k++ {
		if j.text[k] == '\r' && k < j.at-1 && j.text[k+1] == '\n' {
			k++
		}
		if j.text[k] == '\r' || j.text[k] == '\n' {
			line++
			lastBreak = k + 1
		}
	}
	return line, 1 + k - lastBreak
}

// fail sets the error of a message that gives the cursor's place, and stops the parse.
func (j *jsonParser) fail(message string) {
	line, column := j.fileLocation()
	j.err = message + " in JSON at position " + strconv.Itoa(j.at) + " (line " + strconv.Itoa(line) +
		" column " + strconv.Itoa(column) + ")"
	panic(jsonFailed{})
}

// unexpected is V8's ReportUnexpectedToken with no message of its own: what it says depends on
// the token, and for most, on the character at the cursor.
func (j *jsonParser) unexpected(token int) {
	switch token {
	case jEOS:
		j.err = "Unexpected end of JSON input"
		panic(jsonFailed{})
	case jNumber:
		j.fail("Unexpected number")
	case jString:
		j.fail("Unexpected string")
	}
	const context = 10
	char := Decode(j.text[j.at : j.at+1])
	n := len(j.text)
	pos := j.at
	switch {
	case n < 2*context+1:
		j.err = "Unexpected token '" + char + "', \"" + Decode(j.text) + "\" is not valid JSON"
	case pos < context:
		j.err = "Unexpected token '" + char + "', \"" + Decode(j.text[:min(pos+context, n)]) + "\"... is not valid JSON"
	case pos < n-context:
		j.err = "Unexpected token '" + char + "', ...\"" + Decode(j.text[pos-context:pos+context]) + "\"... is not valid JSON"
	default:
		j.err = "Unexpected token '" + char + "', ...\"" + Decode(j.text[pos-context:]) + "\" is not valid JSON"
	}
	panic(jsonFailed{})
}

// unexpectedCharacter is V8's ReportUnexpectedCharacter.
func (j *jsonParser) unexpectedCharacter(c int) {
	token := jIllegal
	if c < 0 {
		token = jEOS
	} else if c <= 0xff {
		token = jsonToken(c)
	}
	j.unexpected(token)
}

// expect takes the token at the cursor if it is the one given, and fails with message if not.
func (j *jsonParser) expect(token int, message string) {
	if jsonToken(j.cur()) != token {
		j.fail(message)
	}
	j.at++
}

func (j *jsonParser) check(token int) bool {
	j.skip()
	if jsonToken(j.cur()) != token {
		return false
	}
	j.at++
	return true
}

// literal reads true, false or null, whose first letter is at the cursor, as V8's
// ScanLiteral does.
func (j *jsonParser) literal(word string) {
	remaining := len(j.text) - j.at
	if remaining >= len(word) {
		same := true
		for k := 1; k < len(word); k++ {
			if int(j.text[j.at+k]) != int(word[k]) {
				same = false
				break
			}
		}
		if same {
			j.at += len(word)
			return
		}
	}
	j.at++
	for k := 0; k < min(len(word)-1, remaining-1); k++ {
		if int(word[1+k]) != j.cur() {
			j.unexpectedCharacter(j.cur())
		}
		j.at++
	}
	j.unexpected(jEOS)
}

// HexValue is the value of a hexadecimal digit, or -1 for another code unit.
func HexValue(c int) int {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return -1
}

// str reads a string whose opening quote the cursor is past, as V8's ScanJsonString does.
func (j *jsonParser) str() string {
	var out []uint16
	for {
		c := j.cur()
		switch {
		case c < 0:
			j.fail("Unterminated string")
		case c == '"':
			j.at++
			return Decode(out)
		case c == '\\':
			e := j.next()
			if e < 0 || e > 0xff {
				j.unexpectedCharacter(e)
			}
			switch e {
			case '"', '\\', '/':
				out = append(out, uint16(e))
			case 'b':
				out = append(out, '\b')
			case 'f':
				out = append(out, '\f')
			case 'n':
				out = append(out, '\n')
			case 'r':
				out = append(out, '\r')
			case 't':
				out = append(out, '\t')
			case 'u':
				value := 0
				for k := 0; k < 4; k++ {
					digit := HexValue(j.next())
					if digit < 0 {
						j.fail("Bad Unicode escape")
					}
					value = value*16 + digit
				}
				out = append(out, uint16(value))
			default:
				j.fail("Bad escaped character")
			}
			j.at++
		case c < 0x20:
			j.fail("Bad control character in string literal")
		default:
			out = append(out, uint16(c))
			j.at++
		}
	}
}

// number reads a number at the cursor, as V8's ParseJsonNumber does.
func (j *jsonParser) number() float64 {
	start := j.at
	c := j.cur()
	if c == '-' {
		c = j.next()
	}
	if c == '0' {
		// A leading zero is only allowed as the only digit before a decimal point or
		// exponent.
		if c = j.next(); chars.IsDigit(c) {
			j.unexpected(jNumber)
		}
	} else {
		from := j.at
		for chars.IsDigit(j.cur()) {
			j.at++
		}
		if j.at == from {
			j.fail("No number after minus sign")
		}
	}
	if j.cur() == '.' {
		if !chars.IsDigit(j.next()) {
			j.fail("Unterminated fractional number")
		}
		for chars.IsDigit(j.cur()) {
			j.at++
		}
	}
	if c := j.cur(); c >= 0 && c|0x20 == 'e' {
		c = j.next()
		if c == '-' || c == '+' {
			c = j.next()
		}
		if !chars.IsDigit(c) {
			j.fail("Exponent part is missing a number")
		}
		for chars.IsDigit(j.cur()) {
			j.at++
		}
	}
	digits := make([]byte, j.at-start)
	for k := range digits {
		digits[k] = byte(j.text[start+k])
	}
	f, _ := strconv.ParseFloat(string(digits), 64)
	return f
}

// value reads a value, as V8's ParseJsonValue does, here by recursion.
func (j *jsonParser) value() any {
	j.skip()
	switch jsonToken(j.cur()) {
	case jString:
		j.at++
		return j.str()
	case jNumber:
		return j.number()
	case jLBrace:
		j.at++
		if j.check(jRBrace) {
			return newObject(nil, nil)
		}
		var keys []string
		var values []any
		j.skip()
		j.expect(jString, "Expected property name or '}'")
		for {
			keys = append(keys, j.str())
			j.skip()
			j.expect(jColon, "Expected ':' after property name")
			values = append(values, j.value())
			if !j.check(jComma) {
				break
			}
			j.skip()
			j.expect(jString, "Expected double-quoted property name")
		}
		j.expect(jRBrace, "Expected ',' or '}' after property value")
		return newObject(keys, values)
	case jLBrack:
		j.at++
		out := []any{}
		if j.check(jRBrack) {
			return out
		}
		for {
			out = append(out, j.value())
			if !j.check(jComma) {
				break
			}
		}
		j.expect(jRBrack, "Expected ',' or ']' after array element")
		return out
	case jTrue:
		j.literal("true")
		return true
	case jFalse:
		j.literal("false")
		return false
	case jNull:
		j.literal("null")
		return nil
	}
	j.unexpectedCharacter(j.cur())
	return nil
}

// Parse is JSON.parse of text, read by UTF-16 code units, or the message of the SyntaxError V8
// throws.
func Parse(text []uint16) (v any, message string) {
	j := &jsonParser{text: text}
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(jsonFailed); !ok {
				panic(r)
			}
			v, message = nil, j.err
		}
	}()
	v = j.value()
	if !j.check(jEOS) {
		j.fail("Unexpected non-whitespace character after JSON")
	}
	return v, ""
}

// Stringify is JSON.stringify of a value Parse gave.
func Stringify(v any) string {
	var b strings.Builder
	var write func(v any)
	write = func(v any) {
		switch v := v.(type) {
		case nil:
			b.WriteString("null")
		case bool:
			b.WriteString(strconv.FormatBool(v))
		case float64:
			if v-v != 0 {
				b.WriteString("null")
			} else {
				b.WriteString(chars.NumberString(v))
			}
		case string:
			quote(&b, v)
		case []any:
			b.WriteByte('[')
			for i, x := range v {
				if i > 0 {
					b.WriteByte(',')
				}
				write(x)
			}
			b.WriteByte(']')
		case *Object:
			b.WriteByte('{')
			for i, k := range v.Keys {
				if i > 0 {
					b.WriteByte(',')
				}
				quote(&b, k)
				b.WriteByte(':')
				write(v.Values[k])
			}
			b.WriteByte('}')
		}
	}
	write(v)
	return b.String()
}

func quote(b *strings.Builder, s string) {
	const hex = "0123456789abcdef"
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"':
			b.WriteString(`\"`)
		case r == '\\':
			b.WriteString(`\\`)
		case r == '\b':
			b.WriteString(`\b`)
		case r == '\f':
			b.WriteString(`\f`)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\r':
			b.WriteString(`\r`)
		case r == '\t':
			b.WriteString(`\t`)
		case r < 0x20:
			b.WriteString(`\u00`)
			b.WriteByte(hex[r>>4])
			b.WriteByte(hex[r&15])
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
}
