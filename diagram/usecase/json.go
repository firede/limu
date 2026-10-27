package usecase

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/edit"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsjson"
)

// Object is a JSON object as JSON.parse gives it, its keys in JavaScript's order.
type Object = jsjson.Object

// KeyOrder is the keys of each object of a JSON value in the order they were written, by the
// object's JSON Pointer: Pele's propertyOrder, a Map, in its order.
type KeyOrder struct {
	m *ordered[*[]string]
}

// Pointers are the pointers of the objects, in order, as iterating Pele's propertyOrder Map
// gives them.
func (k *KeyOrder) Pointers() []string {
	var out []string
	k.m.each(func(p string, _ *[]string) bool {
		out = append(out, p)
		return true
	})
	return out
}

// Keys are the keys of the object at pointer, in the order they were written, as
// propertyOrder.get(pointer) gives them in Pele's flattenJsonRows.
func (k *KeyOrder) Keys(pointer string) ([]string, bool) {
	keys, ok := k.m.get(pointer)
	if !ok {
		return nil, false
	}
	return *keys, true
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

// escapePointer escapes a key for a JSON Pointer, ~ as ~0 and / as ~1, as Pele's propertyOrder
// and flattenJsonRows do in line.
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
	// readString reads a string at the cursor, giving its value if decode is set, as the
	// readString closure of Pele's propertyOrder does.
	readString := func(decode bool) string {
		at++
		var value []uint16
		for at < n {
			start := at
			ch := text[at]
			at++
			if ch == '"' {
				return jsjson.Decode(value)
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
					d := jsjson.HexValue(code(at + k))
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
	// literal reads true, false or null, as the literal closure of Pele's propertyOrder does.
	literal := func(word string) {
		for k := 0; k < len(word); k++ {
			if code(at+k) != int(word[k]) {
				panic(walkError{at + k})
			}
		}
		at += len(word)
	}
	// number reads a number, as the number closure of Pele's propertyOrder does.
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

// isWordByte reports whether \w matches byte b, for the \b of the patterns of Pele's
// locationFromMessage.
func isWordByte(b byte) bool { return chars.IsWord(int(b)) }

// boundedDigits is the run of digits that starts at byte i of s, with its end, if a word
// boundary follows it: the (\d+)\b of the patterns of Pele's locationFromMessage.
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

// findWord calls fn with the end of each place in s, from from on, where word starts after a
// word boundary, until fn reports a match: the search for the \bposition and \bline of the
// patterns of Pele's locationFromMessage.
func findWord(s, word string, from int, fn func(at int) bool) bool {
	for k := from; k <= len(s)-len(word); k++ {
		if strings.HasPrefix(s[k:], word) && (k == 0 || !isWordByte(s[k-1])) && fn(k+len(word)) {
			return true
		}
	}
	return false
}

// parseNumber is parseInt of a run of decimal digits, as Pele's locationFromMessage reads one.
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

// boundedDigitsLoose is a run of digits from i, which a space follows in the pattern: the
// (\d+) after `line ` in Pele's locationFromMessage.
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
	parsed, message := jsjson.Parse(units)
	if message != "" {
		// Parse again with the entity codes as written, so that the message and the place it
		// names are those of the source.
		if shown := edit.WrittenCodes(text); shown != text {
			units = utf16.Encode([]rune(shown))
			if _, again := jsjson.Parse(units); again != "" {
				message = again
			}
		}
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

// jsonError is a syntax error in the body of a `json` node at a line and column, as Pele's
// jsonError makes it.
func jsonError(message string, line, column int) *diagram.Error {
	return &diagram.Error{
		Code:    diagram.Syntax,
		Message: message + " (line " + strconv.Itoa(line) + ", column " + strconv.Itoa(column) + ")",
		Type:    "usecase", Line: line, Column: column,
	}
}
