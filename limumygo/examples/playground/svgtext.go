package main

import "strings"

// formatSVG indents an SVG document the svg package writes on one line, an element on a
// line, for reading. It is for display only: what is copied and exported is the document
// as it was written. A text, title or desc element stays on one line with what is in it,
// as white space there is part of the text. A document it cannot follow, as one whose
// tags do not close, it returns as it was.
func formatSVG(doc string) string {
	var b strings.Builder
	depth := 0
	line := func(s string) {
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(strings.Repeat("  ", depth))
		b.WriteString(s)
	}
	for i := 0; i < len(doc); {
		if doc[i] != '<' {
			j := strings.IndexByte(doc[i:], '<')
			if j < 0 {
				j = len(doc) - i
			}
			if t := strings.TrimSpace(doc[i : i+j]); t != "" {
				line(t)
			}
			i += j
			continue
		}
		end := tagEnd(doc, i)
		if end < 0 {
			return doc
		}
		tag := doc[i:end]
		name := tagName(tag)
		switch {
		case strings.HasPrefix(tag, "</"):
			depth--
			if depth < 0 {
				return doc
			}
			line(tag)
		case strings.HasSuffix(tag, "/>") || strings.HasPrefix(tag, "<?") || strings.HasPrefix(tag, "<!"):
			line(tag)
		case name == "text" || name == "title" || name == "desc" || name == "style":
			stop := strings.Index(doc[end:], "</"+name+">")
			if stop < 0 {
				return doc
			}
			stop += end + len("</"+name+">")
			line(doc[i:stop])
			end = stop
		default:
			line(tag)
			depth++
		}
		i = end
	}
	if depth != 0 {
		return doc
	}
	return b.String()
}

// tagEnd is the byte after the tag that starts at i, past the > that is not in a quoted
// value, or -1 when the tag does not end.
func tagEnd(s string, i int) int {
	var quote byte
	for j := i + 1; j < len(s); j++ {
		switch c := s[j]; {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '>':
			return j + 1
		}
	}
	return -1
}

// tagName is the name of the element a tag opens or closes.
func tagName(tag string) string {
	s := strings.TrimLeft(tag, "</")
	end := strings.IndexAny(s, " \t\n/>")
	if end < 0 {
		return s
	}
	return s[:end]
}
