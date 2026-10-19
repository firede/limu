package sequence

import (
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/firede/limu/internal/chars"
)

// A value of JSON as JavaScript's JSON.parse gives it: nil for null, a bool, a float64, a
// string, a []any, or an *jsObject.
//
// The menu statements of participants (links and properties) read their text as JSON, and
// keep the entries of what it holds; only whether the text is JSON, what its keys are and
// in which order, and the value of a property named class matter to what is drawn.

// jsObject is a JavaScript object made by JSON.parse: its keys in the order its properties
// iterate in, the keys that are array indices first, in order of their value, then the others
// in the order they were first written. A key written again keeps its place and takes the
// later value.
type jsObject struct {
	keys   []string
	values map[string]any
}

func (o *jsObject) set(key string, v any) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = v
}

// ordered is the keys in the order JavaScript iterates them.
func (o *jsObject) ordered() []string {
	var index, other []string
	for _, k := range o.keys {
		if isArrayIndex(k) {
			index = append(index, k)
		} else {
			other = append(other, k)
		}
	}
	if len(index) == 0 {
		return other
	}
	// Insertion sort by value: there are few of them.
	for i := 1; i < len(index); i++ {
		for j := i; j > 0 && arrayIndex(index[j]) < arrayIndex(index[j-1]); j-- {
			index[j], index[j-1] = index[j-1], index[j]
		}
	}
	return append(index, other...)
}

func arrayIndex(k string) uint64 {
	v, _ := strconv.ParseUint(k, 10, 64)
	return v
}

// isArrayIndex reports whether a key is an array index: the canonical decimal of an integer
// below 2^32 - 1.
func isArrayIndex(k string) bool {
	if k == "" || len(k) > 10 || len(k) > 1 && k[0] == '0' {
		return false
	}
	for i := 0; i < len(k); i++ {
		if k[i] < '0' || k[i] > '9' {
			return false
		}
	}
	return arrayIndex(k) < 4294967295
}

// jsonReader reads JSON by UTF-16 code units, as JSON.parse does.
type jsonReader struct {
	u []uint16
	i int
}

type jsonFailed struct{}

func (r *jsonReader) fail() { panic(jsonFailed{}) }

func (r *jsonReader) peek() int {
	if r.i < len(r.u) {
		return int(r.u[r.i])
	}
	return -1
}

func (r *jsonReader) space() {
	for {
		switch r.peek() {
		case ' ', '\t', '\n', '\r':
			r.i++
		default:
			return
		}
	}
}

func (r *jsonReader) value() any {
	r.space()
	switch c := r.peek(); {
	case c == '{':
		r.i++
		o := &jsObject{values: map[string]any{}}
		r.space()
		if r.peek() == '}' {
			r.i++
			return o
		}
		for {
			r.space()
			if r.peek() != '"' {
				r.fail()
			}
			k := r.str()
			r.space()
			if r.peek() != ':' {
				r.fail()
			}
			r.i++
			o.set(k, r.value())
			r.space()
			switch r.peek() {
			case ',':
				r.i++
			case '}':
				r.i++
				return o
			default:
				r.fail()
			}
		}
	case c == '[':
		r.i++
		a := []any{}
		r.space()
		if r.peek() == ']' {
			r.i++
			return a
		}
		for {
			a = append(a, r.value())
			r.space()
			switch r.peek() {
			case ',':
				r.i++
			case ']':
				r.i++
				return a
			default:
				r.fail()
			}
		}
	case c == '"':
		return r.str()
	case c == '-' || chars.IsDigit(c):
		return r.number()
	case c == 't':
		r.literal("true")
		return true
	case c == 'f':
		r.literal("false")
		return false
	case c == 'n':
		r.literal("null")
		return nil
	}
	r.fail()
	return nil
}

func (r *jsonReader) literal(word string) {
	for k := 0; k < len(word); k++ {
		if r.peek() != int(word[k]) {
			r.fail()
		}
		r.i++
	}
}

func (r *jsonReader) str() string {
	r.i++
	var out []uint16
	for {
		c := r.peek()
		switch {
		case c < 0x20:
			r.fail()
		case c == '"':
			r.i++
			return string(utf16.Decode(out))
		case c == '\\':
			r.i++
			e := r.peek()
			r.i++
			switch e {
			case '"', '\\', '/':
				out = append(out, uint16(e))
			case 'b':
				out = append(out, 8)
			case 'f':
				out = append(out, 12)
			case 'n':
				out = append(out, 10)
			case 'r':
				out = append(out, 13)
			case 't':
				out = append(out, 9)
			case 'u':
				v := 0
				for k := 0; k < 4; k++ {
					h := r.peek()
					if !chars.IsHex(h) {
						r.fail()
					}
					r.i++
					v = v*16 + hexDigit(h)
				}
				out = append(out, uint16(v))
			default:
				r.fail()
			}
		default:
			out = append(out, uint16(c))
			r.i++
		}
	}
}

func hexDigit(c int) int {
	switch {
	case c <= '9':
		return c - '0'
	case c >= 'a':
		return c - 'a' + 10
	}
	return c - 'A' + 10
}

func (r *jsonReader) number() float64 {
	start := r.i
	if r.peek() == '-' {
		r.i++
	}
	switch {
	case r.peek() == '0':
		r.i++
	case chars.IsDigit(r.peek()):
		for chars.IsDigit(r.peek()) {
			r.i++
		}
	default:
		r.fail()
	}
	if r.peek() == '.' {
		r.i++
		if !chars.IsDigit(r.peek()) {
			r.fail()
		}
		for chars.IsDigit(r.peek()) {
			r.i++
		}
	}
	if c := r.peek(); c == 'e' || c == 'E' {
		r.i++
		if c := r.peek(); c == '+' || c == '-' {
			r.i++
		}
		if !chars.IsDigit(r.peek()) {
			r.fail()
		}
		for chars.IsDigit(r.peek()) {
			r.i++
		}
	}
	digits := make([]byte, r.i-start)
	for k := range digits {
		digits[k] = byte(r.u[start+k])
	}
	// ParseFloat rounds as JavaScript does, and gives an infinity past the range.
	f, _ := strconv.ParseFloat(string(digits), 64)
	return f
}

// parseJSON is JSON.parse: the value, and false where the text is not JSON.
func parseJSON(text string) (v any, ok bool) {
	r := &jsonReader{u: utf16.Encode([]rune(text))}
	defer func() {
		if e := recover(); e != nil {
			if _, failed := e.(jsonFailed); !failed {
				panic(e)
			}
			v, ok = nil, false
		}
	}()
	v = r.value()
	r.space()
	if r.i != len(r.u) {
		return nil, false
	}
	return v, true
}

// stringify is JSON.stringify of a value JSON.parse gave.
func stringify(v any) string {
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
		case *jsObject:
			b.WriteByte('{')
			for i, k := range v.ordered() {
				if i > 0 {
					b.WriteByte(',')
				}
				quote(&b, k)
				b.WriteByte(':')
				write(v.values[k])
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

// Entries are a JavaScript Map's keys and values, in the order the keys were first set.
type Entries struct {
	Keys   []string
	Values map[string]any
}

// Set sets key to v; a key set again keeps its place.
func (e *Entries) Set(key string, v any) {
	if e.Values == nil {
		e.Values = map[string]any{}
	}
	if _, ok := e.Values[key]; !ok {
		e.Keys = append(e.Keys, key)
	}
	e.Values[key] = v
}

// Get is the value of key, and whether it is set.
func (e *Entries) Get(key string) (any, bool) {
	v, ok := e.Values[key]
	return v, ok
}

// JSON is the value of key as JSON.stringify writes it.
func (e *Entries) JSON(key string) string { return stringify(e.Values[key]) }

// entriesOf is Pele's entries: a string's characters by index, or an object's or an array's
// entries in JavaScript's order, or nothing.
func entriesOf(v any, fn func(key string, value any)) {
	switch v := v.(type) {
	case string:
		i := 0
		for _, r := range v {
			fn(strconv.Itoa(i), string(r))
			i++
		}
	case []any:
		for i, x := range v {
			fn(strconv.Itoa(i), x)
		}
	case *jsObject:
		for _, k := range v.ordered() {
			fn(k, v.values[k])
		}
	}
}
