// Package yaml reads the YAML Mermaid diagrams carry, in their front matter and in the
// metadata of flowchart nodes, as Pele's src/util/yaml.ts reads it: a small reader that
// agrees with js-yaml and its JSON schema on what Mermaid feeds it, with block and flow
// collections, quoted and plain scalars, and block scalars.
//
// A value read is nil (null), a bool, a float64, a string, a []Value or a *Map.
package yaml

import (
	"errors"
	"math"
	"strings"
	"unicode/utf16"

	"github.com/firede/limu/internal/chars"
)

// Value is a value read from YAML: nil, bool, float64, string, []Value or *Map.
type Value = any

// incomplete is the error of a quoted scalar or a flow collection that runs past the text read
// so far, as Pele's Incomplete is.
type incomplete struct{ message string }

func (e *incomplete) Error() string { return e.message }

const maxDepth = 64

func fail(message string) error { return errors.New(message) }

type reader struct {
	lines []string
	i     int
}

func newReader(src string) *reader {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	return &reader{lines: strings.Split(src, "\n")}
}

func (r *reader) skipBlank() {
	for r.i < len(r.lines) {
		t := chars.Trim(r.lines[r.i])
		if t != "" && t[0] != '#' && t != "---" {
			break
		}
		r.i++
	}
}

func (r *reader) indent() (int, error) {
	line := r.lines[r.i]
	k := 0
	for k < len(line) && line[k] == ' ' {
		k++
	}
	if k < len(line) && line[k] == '\t' {
		return 0, fail("YAML: tabs are not allowed for indentation")
	}
	return k, nil
}

// at is the byte of s at i, or 0 past its end, where JavaScript reads undefined: a byte no
// test below looks for.
func at(s string, i int) byte {
	if i < 0 || i >= len(s) {
		return 0
	}
	return s[i]
}

func isBlank(b byte) bool { return b == ' ' || b == '\t' || b == '\n' || b == '\r' }

// isKeyStart is /[\s#:,[\]{}&*!|>'"%@`]/ of a code unit.
func isKeyStart(c int) bool {
	if chars.IsSpace(c) {
		return true
	}
	switch c {
	case '#', ':', ',', '[', ']', '{', '}', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
		return true
	}
	return false
}

// splitKey splits `key: rest` at the start of a line, as Pele's splitKey does: the raw key and
// the offset where the value starts.
func splitKey(line string) (key string, length int, ok bool) {
	if line == "" {
		return "", 0, false
	}
	first := line[0]
	var end, colon int
	if first == '"' || first == '\'' {
		i := 1
		for {
			if i >= len(line) {
				return "", 0, false
			}
			c := line[i]
			if first == '"' && c == '\\' {
				i += 2
			} else if c == first {
				if first == '"' || at(line, i+1) != '\'' {
					break
				}
				i += 2
			} else {
				i++
			}
		}
		end = i + 1
		colon = end
		for isBlank(at(line, colon)) {
			colon++
		}
		if at(line, colon) != ':' {
			return "", 0, false
		}
	} else {
		if isKeyStart(chars.Unit(line, 0)) {
			return "", 0, false
		}
		if first == '-' && (len(line) < 2 || isBlank(line[1]) || line[1] == ':' || line[1] == '#') {
			return "", 0, false
		}
		colon = 1
		for colon < len(line) && line[colon] != ':' && line[colon] != '#' {
			colon++
		}
		if at(line, colon) != ':' {
			return "", 0, false
		}
		end = colon
		for isBlank(at(line, end-1)) {
			end--
		}
	}
	length = colon + 1
	if length < len(line) && !isBlank(line[length]) {
		return "", 0, false
	}
	for isBlank(at(line, length)) {
		length++
	}
	return line[:end], length, true
}

// isFloat is /^[-+]?(?:[0-9][0-9]*(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][-+]?[0-9]+)?$/, which takes in
// every integer /^[-+]?(?:0|[1-9][0-9]*)$/ does.
func isFloat(s string) bool {
	i := 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}
	digits := func() int {
		from := i
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		return i - from
	}
	if digits() > 0 {
		if i < len(s) && s[i] == '.' {
			i++
			digits()
		}
	} else {
		if i >= len(s) || s[i] != '.' {
			return false
		}
		i++
		if digits() == 0 {
			return false
		}
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '-' || s[i] == '+') {
			i++
		}
		if digits() == 0 {
			return false
		}
	}
	return i == len(s)
}

// isPrefixed is /^[-+]?0<letter>[digits]+$/ for hexadecimal, octal and binary numbers.
func isPrefixed(s string, letter byte, base int) bool {
	i := 0
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		i++
	}
	if i+2 >= len(s) || s[i] != '0' || s[i+1] != letter {
		return false
	}
	for _, c := range []byte(s[i+2:]) {
		d := 99
		switch {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c >= 'a' && c <= 'f':
			d = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int(c-'A') + 10
		}
		if d >= base {
			return false
		}
	}
	return true
}

func scalar(raw string) Value {
	s := chars.Trim(raw)
	switch s {
	case "", "~", "null", "Null", "NULL":
		return nil
	case "true", "True", "TRUE":
		return true
	case "false", "False", "FALSE":
		return false
	}
	if isFloat(s) {
		return chars.Number(s)
	}
	if isPrefixed(s, 'x', 16) || isPrefixed(s, 'o', 8) || isPrefixed(s, 'b', 2) {
		neg := s[0] == '-'
		body := s
		if neg || s[0] == '+' {
			body = s[1:]
		}
		v := chars.Number(body)
		if neg {
			return -v
		}
		return v
	}
	if body, ok := strings.CutPrefix(s, "-"); ok && isInf(body) {
		return math.Inf(-1)
	}
	if body, ok := strings.CutPrefix(s, "+"); ok && isInf(body) {
		return math.Inf(1)
	}
	if isInf(s) {
		return math.Inf(1)
	}
	if s == ".nan" || s == ".NaN" || s == ".NAN" {
		return math.NaN()
	}
	return s
}

func isInf(s string) bool { return s == ".inf" || s == ".Inf" || s == ".INF" }

var escapes = map[uint16]string{
	'0': "\x00", 'a': "\x07", 'b': "\b", 't': "\t", 'n': "\n", 'v': "\v", 'f': "\f", 'r': "\r",
	'e': "\x1b", ' ': " ", '"': "\"", '/': "/", '\\': "\\", 'N': "\u0085", '_': "\u00a0",
	'L': "\u2028", 'P': "\u2029",
}

// fold folds the line breaks of a scalar, as Pele's fold does: blanks around a break go, one
// break becomes a space, and each further break stays.
func fold(text string) string {
	if !strings.Contains(text, "\n") {
		return text
	}
	lines := strings.Split(text, "\n")
	last := len(lines) - 1
	for i, line := range lines {
		from, to := 0, len(line)
		if i > 0 {
			for from < to && (line[from] == ' ' || line[from] == '\t') {
				from++
			}
		}
		if i < last {
			for to > from && (line[to-1] == ' ' || line[to-1] == '\t') {
				to--
			}
		}
		lines[i] = line[from:to]
	}
	joined := strings.Join(lines, "\n")
	var b strings.Builder
	for i := 0; i < len(joined); {
		if joined[i] != '\n' {
			b.WriteByte(joined[i])
			i++
			continue
		}
		j := i
		for j < len(joined) && joined[j] == '\n' {
			j++
		}
		if j-i == 1 {
			b.WriteByte(' ')
		} else {
			b.WriteString(strings.Repeat("\n", j-i-1))
		}
		i = j
	}
	return b.String()
}

// dropEscapedBreaks is body.replace(/\\\n[ \t]*/g, ""): an escaped line break goes, with the blanks after it.
func dropEscapedBreaks(body string) string {
	if !strings.Contains(body, "\\\n") {
		return body
	}
	var b strings.Builder
	for i := 0; i < len(body); {
		if body[i] == '\\' && i+1 < len(body) && body[i+1] == '\n' {
			i += 2
			for i < len(body) && (body[i] == ' ' || body[i] == '\t') {
				i++
			}
			continue
		}
		b.WriteByte(body[i])
		i++
	}
	return b.String()
}

// unquoteDouble reads the escapes of a double-quoted scalar. It counts in UTF-16 code units,
// as the lengths of \x, \u and \U are counted.
func unquoteDouble(body string) (string, error) {
	s := utf16.Encode([]rune(fold(dropEscapedBreaks(body))))
	out := make([]uint16, 0, len(s))
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch != '\\' {
			out = append(out, ch)
			continue
		}
		i++
		if i >= len(s) {
			return "", fail("YAML: unknown escape sequence")
		}
		e := s[i]
		if e == 'x' || e == 'u' || e == 'U' {
			n := 2
			if e == 'u' {
				n = 4
			} else if e == 'U' {
				n = 8
			}
			digits := s[min(i+1, len(s)):min(i+1+n, len(s))]
			v := chars.ParseInt(string(utf16.Decode(digits)), 16)
			if math.IsNaN(v) || v < 0 || v > 0x10ffff {
				return "", fail("Invalid code point " + chars.NumberString(v))
			}
			if v > 0xffff {
				hi, lo := utf16.EncodeRune(rune(v))
				out = append(out, uint16(hi), uint16(lo))
			} else {
				out = append(out, uint16(v))
			}
			i += n
		} else if rep, ok := escapes[e]; ok {
			out = append(out, utf16.Encode([]rune(rep))...)
		} else {
			return "", fail("YAML: unknown escape sequence")
		}
	}
	return string(utf16.Decode(out)), nil
}

func skip(s string, pos int, commas bool) int {
	for pos < len(s) {
		c := s[pos]
		if c == ' ' || c == '\n' || c == '\t' || c == '\r' || commas && c == ',' {
			pos++
		} else if c == '#' && (pos == 0 || chars.IsSpace(chars.UnitBefore(s, pos))) {
			nl := strings.IndexByte(s[pos:], '\n')
			if nl == -1 {
				pos = len(s)
			} else {
				pos += nl
			}
		} else {
			break
		}
	}
	return pos
}

// entryStart checks that entries of a flow collection are separated by exactly one comma, as
// js-yaml requires.
func entryStart(ch byte, separated bool) error {
	if !separated {
		return fail("YAML: missed comma between flow collection entries")
	}
	if ch == ',' {
		return fail("YAML: expected the node content, but found ','")
	}
	return nil
}

// flow reads a flow value starting at s[pos], as Pele's flow does; it gives the value and the
// index after it.
func flow(s string, pos int, stops string, depth int) (Value, int, error) {
	if depth > maxDepth {
		return nil, 0, fail("YAML: nesting is too deep")
	}
	pos = skip(s, pos, false)
	ch := at(s, pos)
	switch ch {
	case '{':
		out := NewMap()
		pos++
		separated := true
		for {
			pos = skip(s, pos, false)
			if pos >= len(s) {
				return nil, 0, &incomplete{"YAML: unexpected end of the stream within a flow collection"}
			}
			if s[pos] == '}' {
				return out, pos + 1, nil
			}
			if err := entryStart(s[pos], separated); err != nil {
				return nil, 0, err
			}
			key, next, err := flow(s, pos, ":,}", depth+1)
			if err != nil {
				return nil, 0, err
			}
			pos = skip(s, next, false)
			var value Value
			if at(s, pos) == ':' {
				if value, pos, err = flow(s, pos+1, ",}", depth+1); err != nil {
					return nil, 0, err
				}
			}
			out.Set(String(key), value)
			pos = skip(s, pos, false)
			separated = at(s, pos) == ','
			if separated {
				pos++
			}
		}
	case '[':
		out := []Value{}
		pos++
		separated := true
		for {
			pos = skip(s, pos, false)
			if pos >= len(s) {
				return nil, 0, &incomplete{"YAML: unexpected end of the stream within a flow collection"}
			}
			if s[pos] == ']' {
				return out, pos + 1, nil
			}
			if err := entryStart(s[pos], separated); err != nil {
				return nil, 0, err
			}
			value, next, err := flow(s, pos, ",]", depth+1)
			if err != nil {
				return nil, 0, err
			}
			out = append(out, value)
			pos = skip(s, next, false)
			separated = at(s, pos) == ','
			if separated {
				pos++
			}
		}
	case '"':
		end := pos + 1
		for end < len(s) && s[end] != '"' {
			if s[end] == '\\' {
				end += 2
			} else {
				end++
			}
		}
		if end >= len(s) {
			return nil, 0, &incomplete{"YAML: unexpected end of the stream within a double quoted scalar"}
		}
		text, err := unquoteDouble(s[pos+1 : end])
		if err != nil {
			return nil, 0, err
		}
		return text, end + 1, nil
	case '\'':
		end := pos + 1
		for end < len(s) {
			if s[end] == '\'' {
				if at(s, end+1) != '\'' {
					break
				}
				end++
			}
			end++
		}
		if end >= len(s) {
			return nil, 0, &incomplete{"YAML: unexpected end of the stream within a single quoted scalar"}
		}
		return strings.ReplaceAll(fold(s[pos+1:end]), "''", "'"), end + 1, nil
	}
	end := pos
	for end < len(s) {
		c := s[end]
		if strings.IndexByte(stops, c) >= 0 && (c != ':' || end+1 >= len(s) || endsPlain(chars.Unit(s, end+1))) {
			break
		}
		if c == '#' && end > pos && chars.IsSpace(chars.UnitBefore(s, end)) {
			nl := strings.IndexByte(s[end:], '\n')
			next := len(s)
			if nl != -1 {
				next = end + nl
			}
			return scalar(fold(chars.Trim(s[pos:end]))), next, nil
		}
		end++
	}
	return scalar(fold(chars.Trim(s[pos:end]))), end, nil
}

// endsPlain is /[\s,[\]{}]/ of a code unit.
func endsPlain(c int) bool {
	return chars.IsSpace(c) || c == ',' || c == '[' || c == ']' || c == '{' || c == '}'
}

func stripComment(s string) string {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote != 0 {
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		} else if c == '"' || c == '\'' {
			quote = c
		} else if c == '#' && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
			return s[:i]
		}
	}
	return s
}

func startsBlank(s string) bool { return s != "" && (s[0] == ' ' || s[0] == '\t') }

func blockScalar(r *reader, header string, parentIndent int) string {
	folded := header[0] == '>'
	chomp := byte(0)
	if strings.Contains(header, "-") {
		chomp = '-'
	} else if strings.Contains(header, "+") {
		chomp = '+'
	}
	indent := -1
	if i := strings.IndexAny(header, "123456789"); i >= 0 {
		indent = parentIndent + int(header[i]-'0')
	}
	var out []string
	for r.i < len(r.lines) {
		line := r.lines[r.i]
		if chars.Trim(line) == "" {
			out = append(out, "")
			r.i++
			continue
		}
		k := chars.LeadingSpace(line)
		if indent == -1 {
			if k <= parentIndent {
				break
			}
			indent = k
		}
		if k < indent {
			break
		}
		out = append(out, chars.DropUnits(line, indent))
		r.i++
	}
	trailing := 0
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
		trailing++
	}
	var text string
	if folded {
		var b strings.Builder
		for i, cur := range out {
			if i > 0 {
				prev := out[i-1]
				if cur == "" {
					b.WriteByte('\n')
				} else if prev != "" {
					if startsBlank(prev) || startsBlank(cur) {
						b.WriteByte('\n')
					} else {
						b.WriteByte(' ')
					}
				}
			}
			b.WriteString(cur)
		}
		text = b.String()
	} else {
		text = strings.Join(out, "\n")
	}
	if len(out) == 0 {
		if chomp == '+' {
			return strings.Repeat("\n", trailing)
		}
		return ""
	}
	switch chomp {
	case '-':
		return text
	case '+':
		return text + strings.Repeat("\n", trailing+1)
	}
	return text + "\n"
}

// isBlockHeader is /^[|>][-+0-9]*\s*(?:#.*)?$/.
func isBlockHeader(rest string) bool {
	i := 1
	for i < len(rest) && (rest[i] == '-' || rest[i] == '+' || rest[i] >= '0' && rest[i] <= '9') {
		i++
	}
	i = chars.SpaceRun(rest, i)
	if i == len(rest) {
		return true
	}
	if rest[i] != '#' {
		return false
	}
	for i < len(rest) {
		c, size := chars.UnitAt(rest, i)
		if chars.IsLineEnd(c) {
			return false
		}
		i += size
	}
	return true
}

func inlineValue(r *reader, rest string, indent int) (Value, error) {
	first := at(rest, 0)
	if (first == '|' || first == '>') && isBlockHeader(rest) {
		return blockScalar(r, rest, indent), nil
	}
	if first == '{' || first == '[' || first == '"' || first == '\'' {
		// The value may continue on later lines. Read them in growing batches until it closes.
		start := r.i
		text := rest
		taken := 0
		batch := 1
		for {
			value, end, err := flow(text, 0, "", 0)
			if err == nil {
				used := strings.Count(text[:end], "\n")
				lineEnd := strings.IndexByte(text[end:], '\n')
				tail := text[end:]
				if lineEnd != -1 {
					tail = tail[:lineEnd]
				}
				if chars.Trim(stripComment(tail)) != "" {
					return nil, fail("YAML: unexpected content after a value")
				}
				r.i = start + used
				return value, nil
			}
			inc, ok := err.(*incomplete)
			if !ok {
				return nil, err
			}
			if start+taken >= len(r.lines) {
				return nil, fail(inc.message)
			}
			more := r.lines[start+taken : min(start+taken+batch, len(r.lines))]
			text += "\n" + strings.Join(more, "\n")
			taken += len(more)
			batch *= 2
		}
	}
	text := chars.Trim(stripComment(rest))
	for r.i < len(r.lines) {
		line := r.lines[r.i]
		if chars.Trim(line) == "" {
			j := r.i
			for j < len(r.lines) && chars.Trim(r.lines[j]) == "" {
				j++
			}
			if j >= len(r.lines) || chars.LeadingSpace(r.lines[j]) <= indent {
				break
			}
			text += strings.Repeat("\n", j-r.i)
			r.i = j
			continue
		}
		if chars.LeadingSpace(line) <= indent {
			break
		}
		more := chars.Trim(stripComment(line))
		if !strings.HasSuffix(text, "\n") {
			text += " "
		}
		text += more
		r.i++
	}
	return scalar(text), nil
}

func isItem(line string) bool { return line == "-" || strings.HasPrefix(line, "- ") }

func block(r *reader, minIndent, depth int) (Value, error) {
	if depth > maxDepth {
		return nil, fail("YAML: nesting is too deep")
	}
	r.skipBlank()
	if r.i >= len(r.lines) {
		return nil, nil
	}
	indent, err := r.indent()
	if err != nil {
		return nil, err
	}
	if indent < minIndent {
		return nil, nil
	}
	line := r.lines[r.i][indent:]

	if isItem(line) {
		out := []Value{}
		for r.i < len(r.lines) {
			r.skipBlank()
			if r.i >= len(r.lines) {
				break
			}
			k, err := r.indent()
			if err != nil {
				return nil, err
			}
			if k != indent {
				break
			}
			cur := r.lines[r.i][indent:]
			if !isItem(cur) {
				break
			}
			rest := cur[1:]
			pad := chars.LeadingSpace(rest) + 1
			body := chars.Trim(stripComment(rest))
			var v Value
			if body == "" {
				r.i++
				v, err = block(r, indent+1, depth+1)
			} else if _, _, ok := splitKey(body); ok {
				r.lines[r.i] = strings.Repeat(" ", indent+pad) + chars.TrimStart(rest)
				v, err = block(r, indent+pad, depth+1)
			} else {
				r.i++
				v, err = inlineValue(r, chars.Trim(rest), indent)
			}
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}

	if _, _, ok := splitKey(line); !ok {
		r.i++
		return inlineValue(r, chars.Trim(line), indent-1)
	}

	out := NewMap()
	for r.i < len(r.lines) {
		r.skipBlank()
		if r.i >= len(r.lines) {
			break
		}
		k, err := r.indent()
		if err != nil {
			return nil, err
		}
		if k < indent {
			break
		}
		if k > indent {
			return nil, fail("YAML: bad indentation of a mapping entry")
		}
		cur := r.lines[r.i][indent:]
		raw, length, ok := splitKey(cur)
		if !ok {
			return nil, fail("YAML: can not read a block mapping entry")
		}
		keyValue, _, err := flow(raw, 0, "", 0)
		if err != nil {
			return nil, err
		}
		key := String(keyValue)
		rest := cur[length:]
		r.i++
		var v Value
		if chars.Trim(stripComment(rest)) == "" {
			r.skipBlank()
			if r.i < len(r.lines) {
				next, err := r.indent()
				if err != nil {
					return nil, err
				}
				seq := isItem(r.lines[r.i][next:])
				if next > indent || seq && next == indent {
					child := indent + 1
					if seq {
						child = indent
					}
					if v, err = block(r, child, depth+1); err != nil {
						return nil, err
					}
				}
			}
		} else if v, err = inlineValue(r, chars.Trim(rest), indent); err != nil {
			return nil, err
		}
		out.Set(key, v)
	}
	return out, nil
}

// Parse reads src, as Pele's parseYaml does. An error's message is the one Pele's error has.
func Parse(src string) (Value, error) {
	trimmed := chars.Trim(src)
	if f := at(trimmed, 0); f == '{' || f == '[' {
		value, end, err := flow(trimmed, 0, "", 0)
		if err != nil {
			return nil, fail(err.Error())
		}
		if skip(trimmed, end, false) == len(trimmed) {
			return value, nil
		}
	}
	r := newReader(src)
	value, err := block(r, 0, 0)
	if err != nil {
		return nil, err
	}
	r.skipBlank()
	if r.i < len(r.lines) && chars.Trim(r.lines[r.i]) != "..." {
		return nil, fail("YAML: end of the stream or a document separator is expected")
	}
	return value, nil
}

// String is JavaScript's String() of a value: an array is its items' strings joined by commas,
// with null as nothing, and a map is "[object Object]".
func String(v Value) string {
	switch v := v.(type) {
	case nil:
		return "null"
	case bool:
		if v {
			return "true"
		}
		return "false"
	case float64:
		return chars.NumberString(v)
	case string:
		return v
	case []Value:
		var b strings.Builder
		for i, item := range v {
			if i > 0 {
				b.WriteByte(',')
			}
			if item != nil {
				b.WriteString(String(item))
			}
		}
		return b.String()
	}
	return "[object Object]"
}

// Truthy reports whether JavaScript takes v as true: all but null, false, 0, NaN and "".
func Truthy(v Value) bool {
	switch v := v.(type) {
	case nil:
		return false
	case bool:
		return v
	case float64:
		return v != 0 && !math.IsNaN(v)
	case string:
		return v != ""
	}
	return true
}

// Number is JavaScript's Number() of a value.
func Number(v Value) float64 {
	switch v := v.(type) {
	case nil:
		return 0
	case bool:
		if v {
			return 1
		}
		return 0
	case float64:
		return v
	case string:
		return chars.Number(v)
	case []Value:
		return chars.Number(String(v))
	}
	return math.NaN()
}
