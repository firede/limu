package diagram

import (
	"strings"

	"github.com/firede/limu/diagram/internal/edit"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/yaml"
)

// Config is a diagram's configuration, from its front matter and its init directives, as
// Mermaid's config is: JSON's values, with objects as Config and arrays as []any, numbers as
// float64.
type Config = map[string]any

// preprocessed is a diagram's text read for its front matter and directives, as Pele's
// preprocess gives it.
type preprocessed struct {
	// Text is the text after its front matter, with directives and comment lines taken out
	// and the white space it starts with trimmed.
	Text string
	// Title is the title the front matter gives, if HasTitle.
	Title    string
	HasTitle bool
	Config   Config
	// WithComments is Text before its comment lines are taken out and it is trimmed.
	WithComments string
	// LineOffset is the number of lines of front matter before the text.
	LineOffset int
}

var unsafeKeys = map[string]bool{"__proto__": true, "constructor": true, "prototype": true}

// cleanupComments takes out the lines that are comments, `%%` and not `%%{`, and the white
// space the text starts with, as Pele's cleanupComments does. It adds what it takes out to
// steps, unless steps is nil.
func cleanupComments(text string, steps *[][]edit.Edit) string {
	if strings.Contains(text, "%%") {
		text = dropComments(text, steps)
	}
	trimmed := chars.TrimStart(text)
	edit.Push(steps, []edit.Edit{{From: 0, To: len(text) - len(trimmed)}})
	return trimmed
}

// normalizeLineBreaks writes each CRLF and each lone CR as a line feed, as Pele's
// normalizeLineBreaks does.
func normalizeLineBreaks(text string) string {
	if strings.Contains(text, "\r") {
		text = strings.ReplaceAll(text, "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
	}
	return text
}

// dropComments is text.replace(/^\s*%%(?!{)[^\n]+\n?/gm, ""): at the start of a line, white
// space over any number of lines, %% not followed by {, at least one character up to the line
// feed, and the line feed. It adds what it takes out to steps, unless steps is nil.
func dropComments(text string, steps *[][]edit.Edit) string {
	var b strings.Builder
	var edits []edit.Edit
	last := 0
	for i := 0; i < len(text); {
		if i == 0 || chars.IsLineEnd(chars.UnitBefore(text, i)) {
			if end := commentAt(text, i); end > 0 {
				b.WriteString(text[last:i])
				edits = append(edits, edit.Edit{From: i, To: end})
				last = end
				i = end
				continue
			}
		}
		_, size := chars.UnitAt(text, i)
		i += size
	}
	edit.Push(steps, edits)
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}

func commentAt(text string, i int) int {
	j := chars.SpaceRun(text, i)
	if !strings.HasPrefix(text[j:], "%%") {
		return 0
	}
	j += 2
	if j >= len(text) || text[j] == '{' || text[j] == '\n' {
		return 0
	}
	nl := strings.IndexByte(text[j:], '\n')
	if nl == -1 {
		return len(text)
	}
	return j + nl + 1
}

func isLineBreak(b byte) bool { return b == '\n' || b == '\r' }

type frontMatter struct {
	indent string
	body   string
	length int
}

// matchFrontMatter finds a `---` fenced YAML block at the start of the text the way Mermaid's
// front matter regex does, as Pele's matchFrontMatter does.
func matchFrontMatter(text string) (frontMatter, bool) {
	space := func(i int) (bool, int) {
		c, size := chars.UnitAt(text, i)
		return chars.IsSpace(c), size
	}
	cursor := 0
	for cursor < len(text) {
		ok, size := space(cursor)
		if !ok || isLineBreak(text[cursor]) {
			break
		}
		cursor += size
	}
	indent := text[:cursor]
	if !strings.HasPrefix(text[cursor:], "---") {
		return frontMatter{}, false
	}

	var openingEnds []int
	for i := cursor + 3; i < len(text); {
		ok, size := space(i)
		if !ok {
			break
		}
		if isLineBreak(text[i]) {
			openingEnds = append(openingEnds, i)
		}
		i += size
	}
	if len(openingEnds) == 0 {
		return frontMatter{}, false
	}

	fence := indent + "---"
	closingEnd := func(i int) int {
		if i >= len(text) || !isLineBreak(text[i]) || !strings.HasPrefix(text[i+1:], fence) {
			return -1
		}
		last := -1
		for j := i + 1 + len(fence); j < len(text); {
			ok, size := space(j)
			if !ok {
				break
			}
			if isLineBreak(text[j]) {
				last = j
			}
			j += size
		}
		return last
	}

	needle := "\n" + fence
	lastFence := -1
	for i := strings.LastIndex(text, needle); i > openingEnds[0]; i = strings.LastIndex(text[:i-1+len(needle)], needle) {
		if closingEnd(i) != -1 {
			lastFence = i
			break
		}
	}
	if lastFence == -1 {
		return frontMatter{}, false
	}

	openingEnd := openingEnds[0]
	for _, candidate := range openingEnds {
		if candidate < lastFence {
			openingEnd = candidate
		}
	}

	bodyStart := openingEnd + 1
	firstAt := max(bodyStart-1, 0)
	for i := index(text, needle, firstAt); i != -1 && i <= lastFence; i = index(text, needle, i+1) {
		if i < bodyStart {
			continue
		}
		if end := closingEnd(i); end != -1 {
			return frontMatter{indent: indent, body: text[bodyStart:i], length: end + 1}, true
		}
	}
	return frontMatter{}, false
}

// index is s.indexOf(sub, from).
func index(s, sub string, from int) int {
	if from > len(s) {
		return -1
	}
	i := strings.Index(s[from:], sub)
	if i == -1 {
		return -1
	}
	return from + i
}

func isRecord(v any) bool {
	switch v.(type) {
	case map[string]any, *yaml.Map:
		return true
	}
	return false
}

// entries are the keys and values of a record, in JavaScript's order where it has one.
func entries(v any) func(yield func(string, any) bool) {
	return func(yield func(string, any) bool) {
		switch m := v.(type) {
		case map[string]any:
			for k, x := range m {
				if !yield(k, x) {
					return
				}
			}
		case *yaml.Map:
			for k, x := range m.All() {
				if !yield(k, x) {
					return
				}
			}
		}
	}
}

// plain is a value with its records as Config, as a copy of it; JavaScript would share it.
func plain(v any) any {
	switch x := v.(type) {
	case *yaml.Map:
		out := Config{}
		for k, item := range x.All() {
			out[k] = plain(item)
		}
		return out
	case map[string]any:
		out := Config{}
		for k, item := range x {
			out[k] = plain(item)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = plain(item)
		}
		return out
	}
	return v
}

// merge lays source over target, records into records, as Pele's merge does, leaving out
// the keys that would reach an object's prototype.
func merge(target Config, source any) Config {
	for key, value := range entries(source) {
		if unsafeKeys[key] {
			continue
		}
		current, _ := target[key].(Config)
		switch {
		case isRecord(value) && current != nil:
			merge(current, value)
		case isRecord(value):
			target[key] = merge(Config{}, value)
		default:
			target[key] = plain(value)
		}
	}
	return target
}

// directiveAt matches Pele's RE_DIRECTIVE at s[i],
// /%{2}{\s*(?:(\w+)\s*:|(\w+))\s*(?:(\w+)|((?:(?!}%{2}).|\r?\n)*))?\s*(?:}%{2})?/: the end of the
// match, its type, and the body it gives an init directive.
func directiveAt(s string, i int) (end int, typ, body string, ok bool) {
	if !strings.HasPrefix(s[i:], "%%{") {
		return 0, "", "", false
	}
	j := chars.SpaceRun(s, i+3)
	k := j
	for k < len(s) && chars.IsWord(int(s[k])) {
		k++
	}
	if k == j {
		return 0, "", "", false
	}
	typ = s[j:k]
	j = k
	if c := chars.SpaceRun(s, k); c < len(s) && s[c] == ':' {
		j = c + 1
	}
	j = chars.SpaceRun(s, j)
	if j < len(s) && chars.IsWord(int(s[j])) {
		for j < len(s) && chars.IsWord(int(s[j])) {
			j++
		}
	} else {
		from := j
		for j < len(s) {
			if strings.HasPrefix(s[j:], "}%%") {
				break
			}
			c, size := chars.UnitAt(s, j)
			if !chars.IsLineEnd(c) {
				j += size
			} else if s[j] == '\n' {
				j++
			} else if strings.HasPrefix(s[j:], "\r\n") {
				j += 2
			} else {
				break
			}
		}
		body = s[from:j]
	}
	j = chars.SpaceRun(s, j)
	if strings.HasPrefix(s[j:], "}%%") {
		j += 3
	}
	return j, typ, body, true
}

// directives reads the directives of text into config, as Pele's directives does, and gives
// text without them. It adds what it takes out to steps, unless steps is nil.
func directives(text string, config Config, steps *[][]edit.Edit) string {
	if !strings.Contains(text, "%%{") {
		return text
	}
	normalized := strings.ReplaceAll(chars.Trim(text), "'", "\"")
	for i := 0; i < len(normalized); i++ {
		end, typ, body, ok := directiveAt(normalized, i)
		if !ok {
			continue
		}
		if typ == "wrap" {
			config["wrap"] = true
		} else if (typ == "init" || typ == "initialize") && body != "" {
			// Mermaid ignores an init directive whose body is not valid JSON.
			if args, err := parseJSON(chars.Trim(body)); err == nil && isRecord(args) {
				merge(config, args)
			}
		}
		i = end - 1
	}
	var b strings.Builder
	var edits []edit.Edit
	last := 0
	for i := 0; i < len(text); i++ {
		end, _, _, ok := directiveAt(text, i)
		if !ok {
			continue
		}
		b.WriteString(text[last:i])
		edits = append(edits, edit.Edit{From: i, To: end})
		last = end
		i = end - 1
	}
	edit.Push(steps, edits)
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}

// htmlTags is text.replace(/<(\w+)([^>]*)>/g, …), which writes the double-quoted values of a
// tag's attributes in single quotes, for a text whose last character is the last `>`.
func htmlTags(text string) string {
	var b strings.Builder
	last := 0
	for i := 0; i < len(text); i++ {
		if text[i] != '<' {
			continue
		}
		j := i + 1
		for j < len(text) && chars.IsWord(int(text[j])) {
			j++
		}
		if j == i+1 {
			continue
		}
		close := strings.IndexByte(text[j:], '>')
		if close == -1 {
			continue
		}
		close += j
		b.WriteString(text[last:i])
		b.WriteString(text[i:j])
		b.WriteString(singleQuoted(text[j:close]))
		b.WriteByte('>')
		last = close + 1
		i = close
	}
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}

// singleQuoted is attrs.replace(/="([^"]*)"/g, "='$1'").
func singleQuoted(attrs string) string {
	if !strings.Contains(attrs, "=\"") {
		return attrs
	}
	var b strings.Builder
	last := 0
	for i := 0; i+1 < len(attrs); i++ {
		if attrs[i] != '=' || attrs[i+1] != '"' {
			continue
		}
		end := strings.IndexByte(attrs[i+2:], '"')
		if end == -1 {
			continue
		}
		end += i + 2
		b.WriteString(attrs[last:i])
		b.WriteString("='" + attrs[i+2:end] + "'")
		last = end + 1
		i = end
	}
	b.WriteString(attrs[last:])
	return b.String()
}

// preprocess reads a diagram's front matter and directives and takes out its comment lines,
// as Pele's preprocess does. It adds to steps, unless steps is nil, what it takes out of the
// source on the way to WithComments, once its line breaks are normalized.
func preprocess(source string, steps *[][]edit.Edit) (preprocessed, error) {
	text := normalizeLineBreaks(source)
	// A tag needs a closing bracket, so nothing after the last one can match.
	if strings.Contains(text, "<") {
		if lastClose := strings.LastIndexByte(text, '>'); lastClose > 0 {
			text = htmlTags(text[:lastClose+1]) + text[lastClose+1:]
		}
	}

	pre := preprocessed{Config: Config{}}
	if strings.Contains(text, "---") {
		if front, ok := matchFrontMatter(text); ok {
			body := front.body
			if front.indent != "" {
				lines := strings.Split(body, "\n")
				for i, line := range lines {
					lines[i] = strings.TrimPrefix(line, front.indent)
				}
				body = strings.Join(lines, "\n")
			}
			parsed, err := yaml.Parse(body)
			if err != nil {
				return preprocessed{}, newError("Front matter is not valid YAML. "+err.Error(), Syntax, "")
			}
			if m, ok := parsed.(*yaml.Map); ok {
				if title := m.Value("title"); yaml.Truthy(title) {
					pre.Title = yaml.String(title)
					pre.HasTitle = true
				}
				if config := m.Value("config"); isRecord(config) {
					merge(pre.Config, config)
				}
				if mode := m.Value("displayMode"); yaml.Truthy(mode) {
					gantt, ok := pre.Config["gantt"].(Config)
					if !ok {
						gantt = Config{}
						pre.Config["gantt"] = gantt
					}
					gantt["displayMode"] = yaml.String(mode)
				}
			}
			pre.LineOffset = strings.Count(text[:min(front.length, len(text))], "\n")
			text = text[front.length:]
			edit.Push(steps, []edit.Edit{{From: 0, To: front.length}})
		}
	}

	text = directives(text, pre.Config, steps)
	pre.Text = cleanupComments(text, nil)
	pre.WithComments = text
	return pre, nil
}

// dropColorSemicolons does what Mermaid's `text.replace(/keyword.*:\S*#.*;/g, drop the last
// character)` does, as Pele's does: on a line where the keyword is followed by a colon, a
// color-like `#`, and later a semicolon, the last semicolon goes. It adds what it takes out
// to steps, unless steps is nil.
func dropColorSemicolons(text, keyword string, steps *[][]edit.Edit) string {
	var edits []edit.Edit
	defer func() { edit.Push(steps, edits) }()
	from := strings.Index(text, keyword)
	if from == -1 {
		return text
	}
	var out strings.Builder
	last := 0
	for from != -1 {
		lineEnd, size := len(text), 0
		for j := from; j < len(text); {
			c, n := chars.UnitAt(text, j)
			if chars.IsLineEnd(c) {
				lineEnd, size = j, n
				break
			}
			j += n
		}
		afterColon := false
		hash := -1
		for j := from + len(keyword); j < lineEnd; {
			c, n := chars.UnitAt(text, j)
			if c == ':' {
				afterColon = true
			} else if c == '#' {
				if afterColon {
					hash = j
					break
				}
			} else if chars.IsSpace(c) {
				afterColon = false
			}
			j += n
		}
		if hash != -1 {
			if semi := strings.LastIndexByte(text[:lineEnd], ';'); semi > hash {
				edits = append(edits, edit.Edit{From: semi, To: semi + 1})
				out.WriteString(text[last:semi])
				last = semi + 1
			}
		}
		if lineEnd >= len(text) {
			break
		}
		from = index(text, keyword, lineEnd+size)
	}
	if last == 0 {
		return text
	}
	out.WriteString(text[last:])
	return out.String()
}

// encodeEntities hides `#name;` entity codes behind placeholder characters so `#` and `;`
// survive parsing, as Mermaid does and Pele's encodeEntities does. It adds what it replaces
// to steps, unless steps is nil.
func encodeEntities(text string, steps *[][]edit.Edit) string {
	if !strings.Contains(text, "#") {
		return text
	}
	text = dropColorSemicolons(dropColorSemicolons(text, "style", steps), "classDef", steps)
	var edits []edit.Edit
	defer func() { edit.Push(steps, edits) }()
	var b strings.Builder
	last := 0
	for i := 0; i < len(text); i++ {
		if text[i] != '#' {
			continue
		}
		j := i + 1
		for j < len(text) && chars.IsWord(int(text[j])) {
			j++
		}
		if j == i+1 || j >= len(text) || text[j] != ';' {
			continue
		}
		inner := text[i+1 : j]
		b.WriteString(text[last:i])
		code := "ﬂ°" + inner + "¶ß"
		if strings.Trim(inner, "0123456789") == "" {
			code = "ﬂ°°" + inner + "¶ß"
		}
		b.WriteString(code)
		edits = append(edits, edit.Edit{From: i, To: j + 1, Text: code})
		last = j + 1
		i = j
	}
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}
