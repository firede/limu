package common

import "strings"

// ParseGenericTypes turns `List~int~` into `List<int>`, with the quirks of Mermaid's
// parseGenericTypes around commas, as Pele's parseGenericTypes does. Tildes are ASCII, so
// the work is done on bytes.
func ParseGenericTypes(input string) string {
	if !strings.Contains(input, "~") {
		return input
	}
	sets := splitCommas(input)
	var out []string
	for i := 0; i < len(sets); i++ {
		set := sets[i]
		if set == "," && i > 0 && i+1 < len(sets) && strings.Count(sets[i-1], "~") == 1 && strings.Count(sets[i+1], "~") == 1 {
			set = sets[i-1] + "," + sets[i+1]
			i++
			out = out[:len(out)-1]
		}
		out = append(out, pairTildes(set))
	}
	return strings.Join(out, "")
}

// splitCommas is JavaScript's split(/(,)/): the pieces between commas, and each comma.
func splitCommas(s string) []string {
	var out []string
	for {
		i := strings.IndexByte(s, ',')
		if i < 0 {
			return append(out, s)
		}
		out = append(out, s[:i], ",")
		s = s[i+1:]
	}
}

// pairTildes pairs tildes from the outside in: the first with the last, the second with the
// one before it, as Pele's pairTildes does.
func pairTildes(text string) string {
	count := strings.Count(text, "~")
	if count <= 1 {
		return text
	}
	lead := count%2 != 0 && strings.HasPrefix(text, "~")
	b := []byte(text)
	var at []int
	start := 0
	if lead {
		start = 1
	}
	for i := start; i < len(b); i++ {
		if b[i] == '~' {
			at = append(at, i)
		}
	}
	for lo, hi := 0, len(at)-1; lo < hi; lo, hi = lo+1, hi-1 {
		b[at[lo]] = '<'
		b[at[hi]] = '>'
	}
	return string(b)
}
