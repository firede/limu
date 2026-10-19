package yaml

import (
	"math"
	"slices"
)

// Encode is v in the form limu's tests compare with what Pele makes, as JSON carries it: a
// map as {"$o": [[key, value], …]} in JavaScript's order, or sorted when sorted is set, and
// the numbers JSON has no words for as {"$n": "NaN"} and the like. Records may be *Map or
// map[string]any; the second, having no order, is always sorted.
func Encode(v any, sorted bool) any {
	switch x := v.(type) {
	case float64:
		switch {
		case math.IsNaN(x):
			return map[string]any{"$n": "NaN"}
		case math.IsInf(x, 1):
			return map[string]any{"$n": "Infinity"}
		case math.IsInf(x, -1):
			return map[string]any{"$n": "-Infinity"}
		case x == 0 && math.Signbit(x):
			return map[string]any{"$n": "-0"}
		}
		return x
	case int:
		return Encode(float64(x), sorted)
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = Encode(item, sorted)
		}
		return out
	case []string:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = item
		}
		return out
	case *Map:
		keys := x.Keys()
		if sorted {
			slices.Sort(keys)
		}
		entries := make([]any, len(keys))
		for i, k := range keys {
			entries[i] = []any{k, Encode(x.values[k], sorted)}
		}
		return map[string]any{"$o": entries}
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		entries := make([]any, len(keys))
		for i, k := range keys {
			entries[i] = []any{k, Encode(x[k], sorted)}
		}
		return map[string]any{"$o": entries}
	}
	return v
}
