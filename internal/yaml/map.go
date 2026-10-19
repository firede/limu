package yaml

import (
	"slices"
	"strconv"
)

// Map is a mapping read from YAML, a JavaScript object to the code Pele's readers are ported
// from. Its keys come in the order JavaScript gives an object's keys: those that are array
// indices, "0" to "4294967294" written without leading zeros, in numeric order, and then the
// rest in the order they were first set.
type Map struct {
	keys   []string
	values map[string]Value
}

// NewMap returns an empty map.
func NewMap() *Map { return &Map{values: map[string]Value{}} }

// Len is the number of keys.
func (m *Map) Len() int {
	if m == nil {
		return 0
	}
	return len(m.keys)
}

// Get is the value of key, and whether it is set. A nil map has no keys.
func (m *Map) Get(key string) (Value, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m.values[key]
	return v, ok
}

// Value is the value of key, nil when it is not set.
func (m *Map) Value(key string) Value {
	v, _ := m.Get(key)
	return v
}

// Set sets key to v. A key set again keeps its place.
func (m *Map) Set(key string, v Value) {
	if _, ok := m.values[key]; !ok {
		m.keys = append(m.keys, key)
	}
	m.values[key] = v
}

// Keys are the keys in JavaScript's order.
func (m *Map) Keys() []string {
	if m == nil {
		return nil
	}
	var index []string
	for _, k := range m.keys {
		if isIndex(k) {
			index = append(index, k)
		}
	}
	if len(index) == 0 {
		return slices.Clone(m.keys)
	}
	slices.SortFunc(index, func(a, b string) int {
		x, _ := strconv.ParseUint(a, 10, 64)
		y, _ := strconv.ParseUint(b, 10, 64)
		switch {
		case x < y:
			return -1
		case x > y:
			return 1
		}
		return 0
	})
	out := index
	for _, k := range m.keys {
		if !isIndex(k) {
			out = append(out, k)
		}
	}
	return out
}

// All yields the keys and values in JavaScript's order.
func (m *Map) All() func(yield func(string, Value) bool) {
	return func(yield func(string, Value) bool) {
		for _, k := range m.Keys() {
			if !yield(k, m.values[k]) {
				return
			}
		}
	}
}

// Spread is JavaScript's { ...m, ...other }: a new map with the keys of both, other's values
// over m's. Either may be nil.
func (m *Map) Spread(other *Map) *Map {
	out := NewMap()
	for k, v := range m.All() {
		out.Set(k, v)
	}
	for k, v := range other.All() {
		out.Set(k, v)
	}
	return out
}

// isIndex reports whether k is an array index as JavaScript has them.
func isIndex(k string) bool {
	if k == "" || len(k) > 10 || len(k) > 1 && k[0] == '0' {
		return false
	}
	for i := 0; i < len(k); i++ {
		if k[i] < '0' || k[i] > '9' {
			return false
		}
	}
	n, _ := strconv.ParseUint(k, 10, 64)
	return n < 4294967295
}
