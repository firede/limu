package common

import (
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/internal/chars"
)

// ClassNames is the names of a class attribute, as Pele's classNames (svg/theme.ts) cleans
// them: what is not an ASCII letter, a digit, an underscore or a hyphen, as /[^\w-]+/ takes
// it, separates them.
func ClassNames(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return !(r == '_' || r == '-' || r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
	})
}

// TrimDefault takes /^default\s?/ off the classes of a node, as Pele does before it writes
// them: the class Mermaid gives every node, and the space after it.
func TrimDefault(classes string) string {
	rest, ok := strings.CutPrefix(classes, "default")
	if !ok {
		return classes
	}
	if r, n := utf8.DecodeRuneInString(rest); n > 0 && chars.IsSpace(int(r)) {
		rest = rest[n:]
	}
	return rest
}
