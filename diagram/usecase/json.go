package usecase

import (
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/firede/limu/diagram"
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

// KeyOrder is the keys of each object of a JSON value in the order they were written, by the
// object's JSON Pointer: Pele's propertyOrder, a Map, in its order.
type KeyOrder struct {
	m *ordered[*[]string]
}

// Pointers are the pointers of the objects, in order.
func (k *KeyOrder) Pointers() []string {
	var out []string
	k.m.each(func(p string, _ *[]string) bool {
		out = append(out, p)
		return true
	})
	return out
}

// Keys are the keys of the object at pointer, in the order they were written.
func (k *KeyOrder) Keys(pointer string) ([]string, bool) {
	keys, ok := k.m.get(pointer)
	if !ok {
		return nil, false
	}
	return *keys, true
}

// isArrayIndex reports whether key is a canonical array index, which JavaScript orders before
// an object's other keys.
func isArrayIndex(key string) bool {
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
			if isArrayIndex(k) {
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

// decode16 is a string of code units, as a Go string holds it: a surrogate with no partner
// becomes U+FFFD.
func decode16(u []uint16) string { return string(utf16.Decode(u)) }

// jsonParser is V8's JSON.parse, as far as Pele's use of it shows: the value it gives, and
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
	char := decode16(j.text[j.at : j.at+1])
	n := len(j.text)
	pos := j.at
	switch {
	case n < 2*context+1:
		j.err = "Unexpected token '" + char + "', \"" + decode16(j.text) + "\" is not valid JSON"
	case pos < context:
		j.err = "Unexpected token '" + char + "', \"" + decode16(j.text[:min(pos+context, n)]) + "\"... is not valid JSON"
	case pos < n-context:
		j.err = "Unexpected token '" + char + "', ...\"" + decode16(j.text[pos-context:pos+context]) + "\"... is not valid JSON"
	default:
		j.err = "Unexpected token '" + char + "', ...\"" + decode16(j.text[pos-context:]) + "\" is not valid JSON"
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

func hexValue(c int) int {
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
			return decode16(out)
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
					digit := hexValue(j.next())
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

// jsonParse is JSON.parse of text, or the message of the error it throws.
func jsonParse(text []uint16) (v any, message string) {
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

// walkError is the offset of the first code unit of text that is not valid JSON.
type walkError struct{ offset int }

// walkFrame is Pele's Frame: an object or an array the walk is in.
type walkFrame struct {
	object  bool
	pointer string
	keys    *[]string
	// For each key read so far, the stretch of created its value added.
	ranges map[string][2]int
	key    string
	from   int
	index  int
}

func escapePointer(key string) string {
	return strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
}

// propertyOrder walks JSON text that JSON.parse has accepted, to recover the order its keys
// were written in, which JSON.parse loses for keys that are array indices, as Pele's
// propertyOrder does. On text JSON.parse rejected it finds where, as a walkError. Nesting is
// kept on a stack of its own.
func propertyOrder(text []uint16) (order *ordered[*[]string], failed *walkError) {
	defer func() {
		if r := recover(); r != nil {
			w, ok := r.(walkError)
			if !ok {
				panic(r)
			}
			order, failed = nil, &w
		}
	}()
	order = newOrdered[*[]string]()
	var created []string
	var stack []*walkFrame
	n := len(text)
	at := 0
	pointer := ""
	const (
		stateValue = iota
		stateKey
		stateAfter
	)
	state := stateValue
	code := func(i int) int {
		if i < n {
			return int(text[i])
		}
		return -1
	}
	skip := func() {
		for at < n {
			c := text[at]
			if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
				return
			}
			at++
		}
	}
	readString := func(decode bool) string {
		at++
		var value []uint16
		for at < n {
			start := at
			ch := text[at]
			at++
			if ch == '"' {
				return decode16(value)
			}
			if ch < 0x20 {
				panic(walkError{start})
			}
			if ch != '\\' {
				if decode {
					value = append(value, ch)
				}
				continue
			}
			escapeAt := at
			escape := code(at)
			at++
			var decoded uint16
			switch escape {
			case '"', '\\', '/':
				decoded = uint16(escape)
			case 'b':
				decoded = '\b'
			case 'f':
				decoded = '\f'
			case 'n':
				decoded = '\n'
			case 'r':
				decoded = '\r'
			case 't':
				decoded = '\t'
			case 'u':
				v := 0
				for k := 0; k < 4; k++ {
					d := hexValue(code(at + k))
					if d < 0 {
						panic(walkError{at})
					}
					v = v*16 + d
				}
				decoded = uint16(v)
				at += 4
			default:
				panic(walkError{escapeAt})
			}
			if decode {
				value = append(value, decoded)
			}
		}
		panic(walkError{at})
	}
	literal := func(word string) {
		for k := 0; k < len(word); k++ {
			if code(at+k) != int(word[k]) {
				panic(walkError{at + k})
			}
		}
		at += len(word)
	}
	number := func() {
		if code(at) == '-' {
			at++
		}
		if code(at) == '0' {
			at++
		} else if chars.IsDigit(code(at)) {
			for chars.IsDigit(code(at)) {
				at++
			}
		} else {
			panic(walkError{at})
		}
		if code(at) == '.' {
			at++
			if !chars.IsDigit(code(at)) {
				panic(walkError{at})
			}
			for chars.IsDigit(code(at)) {
				at++
			}
		}
		if code(at)|32 == 'e' {
			at++
			if code(at) == '+' || code(at) == '-' {
				at++
			}
			if !chars.IsDigit(code(at)) {
				panic(walkError{at})
			}
			for chars.IsDigit(code(at)) {
				at++
			}
		}
	}

	skip()
	for {
		if state == stateValue {
			skip()
			c := code(at)
			state = stateAfter
			switch {
			case c == '{':
				at++
				keys := &[]string{}
				order.set(pointer, keys)
				created = append(created, pointer)
				skip()
				if code(at) == '}' {
					at++
				} else {
					stack = append(stack, &walkFrame{object: true, pointer: pointer, keys: keys, ranges: map[string][2]int{}})
					state = stateKey
				}
			case c == '[':
				at++
				skip()
				if code(at) == ']' {
					at++
				} else {
					stack = append(stack, &walkFrame{pointer: pointer, keys: &[]string{}, ranges: map[string][2]int{}})
					pointer += "/0"
					state = stateValue
				}
			case c == '"':
				readString(false)
			case c == 't':
				literal("true")
			case c == 'f':
				literal("false")
			case c == 'n':
				literal("null")
			case c == '-' || chars.IsDigit(c):
				number()
			default:
				panic(walkError{at})
			}
		} else if state == stateKey {
			frame := stack[len(stack)-1]
			if code(at) != '"' {
				panic(walkError{at})
			}
			key := readString(true)
			// A repeated key keeps its first place in the order and takes the nested order of
			// its last value.
			if r, ok := frame.ranges[key]; !ok {
				*frame.keys = append(*frame.keys, key)
			} else {
				for k := r[0]; k < r[1]; k++ {
					order.delete(created[k])
				}
			}
			frame.key = key
			frame.from = len(created)
			skip()
			if code(at) != ':' {
				panic(walkError{at})
			}
			at++
			pointer = frame.pointer + "/" + escapePointer(key)
			state = stateValue
		} else {
			if len(stack) == 0 {
				break
			}
			frame := stack[len(stack)-1]
			if frame.object {
				frame.ranges[frame.key] = [2]int{frame.from, len(created)}
			} else {
				frame.index++
			}
			skip()
			c := code(at)
			closer := int(']')
			if frame.object {
				closer = '}'
			}
			if c == closer {
				at++
				stack = stack[:len(stack)-1]
			} else if c != ',' {
				panic(walkError{at})
			} else {
				at++
				skip()
				pointer = frame.pointer + "/" + strconv.Itoa(frame.index)
				if frame.object {
					state = stateKey
				} else {
					state = stateValue
				}
			}
		}
	}
	skip()
	if at != n {
		panic(walkError{at})
	}
	return order, nil
}

// locationAt is the line and column of offset in text, which starts at line and column, as
// Pele's locationAt counts them.
func locationAt(text []uint16, offset float64, line, column int) (int, int) {
	end := int(min(max(offset, 0), float64(len(text))))
	for i := 0; i < end; i++ {
		switch c := text[i]; {
		case c == '\r':
			line++
			column = 1
		case c == '\n':
			if i == 0 || text[i-1] != '\r' {
				line++
				column = 1
			}
		default:
			column++
		}
	}
	return line, column
}

func isWordByte(b byte) bool { return chars.IsWord(int(b)) }

// boundedDigits is the run of digits that starts at byte i of s, with its end, if a word
// boundary follows it: the (\d+)\b of a regular expression.
func boundedDigits(s string, i int) (string, int) {
	j := i
	for j < len(s) && chars.IsDigit(int(s[j])) {
		j++
	}
	if j == i || j < len(s) && isWordByte(s[j]) {
		return "", -1
	}
	return s[i:j], j
}

// findWord is the first place in s where word starts after a word boundary.
func findWord(s, word string, from int, fn func(at int) bool) bool {
	for k := from; k <= len(s)-len(word); k++ {
		if strings.HasPrefix(s[k:], word) && (k == 0 || !isWordByte(s[k-1])) && fn(k+len(word)) {
			return true
		}
	}
	return false
}

// parseNumber is parseInt of a run of decimal digits.
func parseNumber(digits string) float64 {
	f, err := strconv.ParseFloat(digits, 64)
	if err != nil && !math.IsInf(f, 0) {
		return 0
	}
	return f
}

// locationFromMessage is the line and column of the place a message of JSON.parse names, as
// Pele's locationFromMessage reads it: engines word their errors differently, some with an
// offset, some with a line and column.
func locationFromMessage(message string, text []uint16, line, column int) (int, int, bool) {
	var outLine, outColumn int
	found := findWord(message, "position ", 0, func(k int) bool {
		digits, _ := boundedDigits(message, k)
		if digits == "" {
			return false
		}
		outLine, outColumn = locationAt(text, parseNumber(digits), line, column)
		return true
	})
	if found {
		return outLine, outColumn, true
	}
	found = findWord(message, "line ", 0, func(k int) bool {
		lineDigits, end := boundedDigitsLoose(message, k)
		if lineDigits == "" || !strings.HasPrefix(message[end:], " column ") {
			return false
		}
		columnDigits, _ := boundedDigits(message, end+len(" column "))
		if columnDigits == "" {
			return false
		}
		localLine := parseNumber(lineDigits)
		localColumn := parseNumber(columnDigits)
		outLine = int(float64(line) + localLine - 1)
		if localLine == 1 {
			outColumn = int(float64(column) + localColumn - 1)
		} else {
			outColumn = int(localColumn)
		}
		return true
	})
	if found {
		return outLine, outColumn, true
	}
	// The pattern's i flag folds the case of ASCII letters alone here.
	lower := []byte(message)
	for k, b := range lower {
		if b >= 'A' && b <= 'Z' {
			lower[k] = b + 32
		}
	}
	if strings.Contains(string(lower), "unexpected end") || strings.Contains(string(lower), "end of json input") {
		l, c := locationAt(text, float64(len(text)), line, column)
		return l, c, true
	}
	return 0, 0, false
}

// boundedDigitsLoose is a run of digits from i, which a space follows in the pattern.
func boundedDigitsLoose(s string, i int) (string, int) {
	j := i
	for j < len(s) && chars.IsDigit(int(s[j])) {
		j++
	}
	if j == i {
		return "", -1
	}
	return s[i:j], j
}

// parseOrderedJSON parses the body of a `json` node, as Pele's parseOrderedJson does. origin
// gives the line and column the text starts at in the diagram.
func parseOrderedJSON(text string, origin func() (int, int)) (*Object, *KeyOrder, *diagram.Error) {
	units := utf16.Encode([]rune(text))
	parsed, message := jsonParse(units)
	if message != "" {
		line, column := origin()
		l, c, ok := locationFromMessage(message, units, line, column)
		if !ok {
			offset := 0
			if _, w := propertyOrder(units); w != nil {
				offset = w.offset
			}
			l, c = locationAt(units, float64(offset), line, column)
		}
		return nil, nil, jsonError("Invalid JSON: "+message, l, c)
	}
	object, ok := parsed.(*Object)
	if !ok {
		line, column := origin()
		return nil, nil, jsonError("JSON value must have an object root", line, column)
	}
	order, _ := propertyOrder(units)
	return object, &KeyOrder{order}, nil
}

func jsonError(message string, line, column int) *diagram.Error {
	return &diagram.Error{
		Code:    diagram.Syntax,
		Message: message + " (line " + strconv.Itoa(line) + ", column " + strconv.Itoa(column) + ")",
		Type:    "usecase", Line: line, Column: column,
	}
}
