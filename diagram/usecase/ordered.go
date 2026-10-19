package usecase

// ordered is a map that iterates in the order its keys were added, as JavaScript's Map does:
// setting a key it has keeps the key's place, and a key deleted and set again goes last. It
// stands for the Maps of Pele's model and of its resolveUsecase, and its methods for those of
// Map that they call.
type ordered[V any] struct {
	keys  []string
	vals  []V
	live  []bool
	index map[string]int
}

// newOrdered is an empty map, as new Map() is.
func newOrdered[V any]() *ordered[V] { return &ordered[V]{index: map[string]int{}} }

// get is the value of key, and whether the map has it, as Map's get and has give them.
func (m *ordered[V]) get(key string) (V, bool) {
	if i, ok := m.index[key]; ok {
		return m.vals[i], true
	}
	var zero V
	return zero, false
}

// set sets the value of key, which keeps its place if the map has it and goes last if not,
// as Map's set does.
func (m *ordered[V]) set(key string, v V) {
	if i, ok := m.index[key]; ok {
		m.vals[i] = v
		return
	}
	m.index[key] = len(m.keys)
	m.keys = append(m.keys, key)
	m.vals = append(m.vals, v)
	m.live = append(m.live, true)
}

// delete removes key, as Map's delete does.
func (m *ordered[V]) delete(key string) {
	if i, ok := m.index[key]; ok {
		delete(m.index, key)
		m.live[i] = false
		var zero V
		m.vals[i] = zero
	}
}

// clear removes every key, as Map's clear does.
func (m *ordered[V]) clear() {
	m.keys, m.vals, m.live = nil, nil, nil
	m.index = map[string]int{}
}

// each calls fn with each key and value, in order, as long as it returns true, as iterating a
// Map does.
func (m *ordered[V]) each(fn func(key string, v V) bool) {
	for i, k := range m.keys {
		if m.live[i] && !fn(k, m.vals[i]) {
			return
		}
	}
}

// values are the values in order, as [...map.values()] gives them.
func (m *ordered[V]) values() []V {
	out := make([]V, 0, len(m.index))
	m.each(func(_ string, v V) bool {
		out = append(out, v)
		return true
	})
	return out
}

// copy is a new map of the same keys and values, in the same order, as new Map(map) is.
func (m *ordered[V]) copy() *ordered[V] {
	out := newOrdered[V]()
	m.each(func(k string, v V) bool {
		out.set(k, v)
		return true
	})
	return out
}
