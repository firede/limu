package sankey

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/diagram/internal/edit"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/text"
)

// isBlank reports whether code unit c is white space to a regular expression's \s that does
// not break a line, as Pele's parser's isBlank does.
func isBlank(c int) bool {
	return c != 10 && c != 13 && chars.IsSpace(c)
}

// trimRows takes the blanks off the start and end of each row, outside quoted fields, as
// Pele's trimRows does, and adds an edit for each run it takes to edits. Blanks before a comma
// at the start of a row are kept, as Mermaid reads them as an empty name.
func trimRows(text string, edits *[]edit.Edit) string {
	end := len(text)
	for end > 0 && isBlank(chars.UnitBefore(text, end)) {
		_, size := utf8.DecodeLastRuneInString(text[:end])
		end -= size
	}
	var out strings.Builder
	kept := 0
	quoted := false
	for i := 0; i < end; {
		c, size := chars.UnitAt(text, i)
		switch {
		case quoted:
			if c == '"' {
				if chars.Unit(text, i+1) == '"' {
					i++
				} else {
					quoted = false
				}
			}
			i += size
		case c == '"':
			quoted = true
			i++
		case isBlank(c):
			j := i + size
			for {
				d, n := chars.UnitAt(text, j)
				if !isBlank(d) {
					break
				}
				j += n
			}
			p, q := chars.UnitBefore(text, i), chars.Unit(text, j)
			if i == 0 || q == 10 || q == 13 || (p == 10 || p == 13) && q != ',' {
				out.WriteString(text[kept:i])
				*edits = append(*edits, edit.Edit{From: i, To: j})
				kept = j
			}
			i = j
		default:
			i += size
		}
	}
	if end < len(text) {
		*edits = append(*edits, edit.Edit{From: end, To: len(text)})
	}
	out.WriteString(text[kept:end])
	return out.String()
}

// prepareTextForParsing prepares a text as Pele's prepareTextForParsing does, which is as
// Mermaid's sankeyUtils does but that blanks are also taken off the ends of each row: runs of
// line breaks then become one, so that no row is empty, and what is left is trimmed. Unless
// steps is nil, the edits of each of these passes are added to it, so that an error can be
// reported where it is in the text as written.
func prepareTextForParsing(text string, steps *[][]edit.Edit) string {
	var edits []edit.Edit
	rows := trimRows(text, &edits)
	// .replace(/[\n\r]+/g, '\n')
	var breaks []edit.Edit
	var b strings.Builder
	for i := 0; i < len(rows); {
		if rows[i] == '\n' || rows[i] == '\r' {
			j := i
			for j < len(rows) && (rows[j] == '\n' || rows[j] == '\r') {
				j++
			}
			breaks = append(breaks, edit.Edit{From: i, To: j, Text: "\n"})
			b.WriteByte('\n')
			i = j
			continue
		}
		b.WriteByte(rows[i])
		i++
	}
	collapsed := b.String()
	trimmed := chars.Trim(collapsed)
	lead := len(collapsed) - len(chars.TrimStart(collapsed))
	edit.Push(steps, edits)
	edit.Push(steps, breaks)
	edit.Push(steps, []edit.Edit{{From: 0, To: lead}, {From: lead + len(trimmed), To: len(collapsed)}})
	return trimmed
}

// parseFloat is JavaScript's parseFloat: white space at the start is skipped, and the longest
// start of what follows that is a decimal number, or Infinity, with a sign or none, is read;
// NaN where none is. An exponent counts only with a digit in it. More than a float64 holds is
// an infinity, as in JavaScript.
func parseFloat(s string) float64 {
	s = chars.TrimStart(s)
	i := 0
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	if strings.HasPrefix(s[i:], "Infinity") {
		if s[0] == '-' {
			return math.Inf(-1)
		}
		return math.Inf(1)
	}
	digits := false
	for i < len(s) && chars.IsDigit(int(s[i])) {
		i++
		digits = true
	}
	if i < len(s) && s[i] == '.' {
		j := i + 1
		for j < len(s) && chars.IsDigit(int(s[j])) {
			j++
		}
		if j > i+1 || digits {
			i, digits = j, true
		}
	}
	if !digits {
		return math.NaN()
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			j++
		}
		k := j
		for k < len(s) && chars.IsDigit(int(s[k])) {
			k++
		}
		if k > j {
			i = k
		}
	}
	f, _ := strconv.ParseFloat(s[:i], 64)
	return f
}

// parseFailure carries a syntax error out of the parser.
type parseFailure struct{ err *diagram.Error }

// parseSankey parses a prepared text into m, as Pele's parseSankey does: the keyword, a line
// break, and rows of a source, a target and an amount, separated by commas, each field plain
// or in double quotes, with a doubled quote for a quote in it. An error is reported in source,
// the text as it was before prepareTextForParsing, which is src where src was not prepared.
func parseSankey(src string, m *Model, source string) (err error) {
	tk := tokenize(src)
	types, texts := tk.Types, tk.Texts
	i := 0
	defer func() {
		if r := recover(); r != nil {
			f, ok := r.(parseFailure)
			if !ok {
				panic(r)
			}
			err = f.err
		}
	}()

	// failAt reports an error at byte at of src where it is in source, as Pele's error does.
	failAt := func(at int, detail string) {
		var steps [][]edit.Edit
		if source != src {
			prepareTextForParsing(source, &steps)
		}
		offset := edit.InputOffset(steps, at)
		if at > 0 && at >= len(src) {
			// The end of src is mapped to just after its last character, before the blanks
			// that were trimmed after it.
			_, size := utf8.DecodeLastRuneInString(src)
			offset = edit.InputOffset(steps, len(src)-size) + size
		}
		panic(parseFailure{common.SyntaxError("sankey", source, offset, detail, false)})
	}
	fail := func(expected string) {
		t := types[i]
		got := tokenNames[t]
		if t == tSTUCK {
			got = tokenNames[types[i-1]]
		}
		failAt(tk.Starts[i], "Expecting "+expected+", got '"+got+"'")
	}
	expect := func(typ int, also string) {
		if types[i] != typ {
			fail("'" + tokenNames[typ] + "'" + also)
		}
		i++
	}
	field := func() string {
		if types[i] == tNON_ESCAPED_TEXT {
			i++
			return texts[i-1]
		}
		if types[i] != tDQUOTE {
			fail("'DQUOTE', 'NON_ESCAPED_TEXT'")
		}
		i++
		if types[i] != tESCAPED_TEXT {
			fail("'ESCAPED_TEXT'")
		}
		text := texts[i]
		i++
		expect(tDQUOTE, "")
		return text
	}
	name := func() string {
		start := tk.Starts[i]
		raw := strings.ReplaceAll(chars.Trim(field()), `""`, `"`)
		// Entity codes can stand for characters the lexer leaves out.
		if invalidXML(text.DecodeEntities(raw)) {
			failAt(start, "Invalid XML character in node name")
		}
		return raw
	}

	expect(tSANKEY, "")
	expect(tNEWLINE, "")
	for {
		from := name()
		expect(tCOMMA, ", 'NEWLINE', 'EOF'")
		to := name()
		expect(tCOMMA, ", 'NEWLINE', 'EOF'")
		start := tk.Starts[i]
		// What parseFloat reads at the start of the field is the value, as in `50%`, as
		// Mermaid reads it, but a value that is not finite is an error.
		value := parseFloat(field())
		if math.IsNaN(value) || math.IsInf(value, 0) {
			failAt(start, "Expecting a number")
		}
		m.addLink(m.findOrCreateNode(from), m.findOrCreateNode(to), value)
		if types[i] != tNEWLINE {
			break
		}
		i++
	}
	if types[i] == tEOF {
		i++
	}
	if types[i] != tEND {
		fail("'NEWLINE', 'EOF'")
	}
	return nil
}

// invalidXML reports whether s holds a character XML does not allow, as Pele's
// RE_INVALID_XML finds one: a control but for the tab, the line feed and the carriage return,
// or U+FFFE or U+FFFF. A lone surrogate, which the regular expression finds too, is never
// decoded from an entity, and a Go string holds none.
func invalidXML(s string) bool {
	for _, c := range s {
		if c <= 8 || c == 0xb || c == 0xc || c >= 0xe && c <= 0x1f || c == 0xfffe || c == 0xffff {
			return true
		}
	}
	return false
}
