package sequence

import (
	"strconv"
	"unicode/utf16"

	"github.com/firede/limu/internal/jsjson"
)

// A value of JSON as JavaScript's JSON.parse gives it: nil for null, a bool, a float64, a
// string, a []any, or a *jsjson.Object.
//
// The menu statements of participants (links and properties) read their text as JSON, and
// keep the entries of what it holds; only whether the text is JSON, what its keys are and
// in which order, and the value of a property named class matter to what is drawn.

// parseJSON is JSON.parse: the value, and false where the text is not JSON.
func parseJSON(text string) (any, bool) {
	v, message := jsjson.Parse(utf16.Encode([]rune(text)))
	return v, message == ""
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
func (e *Entries) JSON(key string) string { return jsjson.Stringify(e.Values[key]) }

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
	case *jsjson.Object:
		for _, k := range v.Keys {
			fn(k, v.Values[k])
		}
	}
}
