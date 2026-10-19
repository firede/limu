package gitgraph

import (
	"strings"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/text"
)

// plainLabel lays out text as plain text in which only <br> is markup, as Pele's plainLabel
// does: Mermaid writes branch names, tags and commit ids as plain text, so other tags and
// icon names show as written.
func plainLabel(raw string, measure text.Measurer, size, maxWidth float64) text.Label {
	s := raw
	if strings.Contains(s, "<") {
		s = escapeTags(s)
	}
	if strings.Contains(s, ":fa-") {
		s = strings.ReplaceAll(s, ":fa-", "&colon;fa-")
	}
	return text.Layout(s, text.Options{Measurer: measure, Size: size, MaxWidth: maxWidth})
}

// escapeTags is s.replace(/<(?!br\s*\/?>)/gi, '&lt;'): every < that does not start a <br>.
func escapeTags(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '<' && !isBreak(s, i+1) {
			b.WriteString("&lt;")
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// isBreak reports whether /br\s*\/?>/i matches at byte i of s.
func isBreak(s string, i int) bool {
	if i+2 > len(s) || !chars.EqualFoldASCII(s[i:i+2], "br") {
		return false
	}
	j := chars.SpaceRun(s, i+2)
	if j < len(s) && s[j] == '/' {
		j++
	}
	return j < len(s) && s[j] == '>'
}
