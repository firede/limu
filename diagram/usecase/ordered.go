package usecase

// ordered is a map that iterates in the order its keys were added, as JavaScript's Map does:
// setting a key it has keeps the key's place, and a key deleted and set again goes last.
type ordered[V any] struct {
	keys  []string
	vals  []V
	live  []bool
	index map[string]int
}

func newOrdered[V any]() *ordered[V] { return &ordered[V]{index: map[string]int{}} }

func (m *ordered[V]) get(key string) (V, bool) {
	if i, ok := m.index[key]; ok {
		return m.vals[i], true
	}
	var zero V
	return zero, false
}

func (m *ordered[V]) has(key string) bool {
	_, ok := m.index[key]
	return ok
}

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

func (m *ordered[V]) delete(key string) {
	if i, ok := m.index[key]; ok {
		delete(m.index, key)
		m.live[i] = false
		var zero V
		m.vals[i] = zero
	}
}

func (m *ordered[V]) clear() {
	m.keys, m.vals, m.live = nil, nil, nil
	m.index = map[string]int{}
}

func (m *ordered[V]) size() int { return len(m.index) }

// each calls fn with each key and value, in order, as long as it returns true.
func (m *ordered[V]) each(fn func(key string, v V) bool) {
	for i, k := range m.keys {
		if m.live[i] && !fn(k, m.vals[i]) {
			return
		}
	}
}

// values are the values in order.
func (m *ordered[V]) values() []V {
	out := make([]V, 0, len(m.index))
	m.each(func(_ string, v V) bool {
		out = append(out, v)
		return true
	})
	return out
}

// copy is a new map of the same keys and values, in the same order.
func (m *ordered[V]) copy() *ordered[V] {
	out := newOrdered[V]()
	m.each(func(k string, v V) bool {
		out.set(k, v)
		return true
	})
	return out
}
