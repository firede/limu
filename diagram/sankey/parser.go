package sankey

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/text"
)

// isBlank reports whether code unit c is white space to a regular expression's \s that does
// not break a line, as Pele's parser's isBlank does.
func isBlank(c int) bool {
	return c != 10 && c != 13 && chars.IsSpace(c)
}

// prepareTextForParsing prepares a text as Mermaid's sankeyUtils does, as Pele's
// prepareTextForParsing does: blanks are taken off both ends of the text, runs of line breaks
// become one, so that no row is empty, and what is left is trimmed.
func prepareTextForParsing(text string) string {
	start, end := 0, len(text)
	for start < end {
		c, size := chars.UnitAt(text, start)
		if !isBlank(c) {
			break
		}
		start += size
	}
	for end > start {
		c := chars.UnitBefore(text, end)
		if !isBlank(c) {
			break
		}
		_, size := utf8.DecodeLastRuneInString(text[:end])
		end -= size
	}
	s := text[start:end]
	// .replace(/[\n\r]+/g, '\n')
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\n' || s[i] == '\r' {
			for i < len(s) && (s[i] == '\n' || s[i] == '\r') {
				i++
			}
			b.WriteByte('\n')
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return chars.Trim(b.String())
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
// or in double quotes, with a doubled quote for a quote in it.
func parseSankey(src string, m *Model) (err error) {
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

	fail := func(expected string) {
		t := types[i]
		got := tokenNames[t]
		if t == tSTUCK {
			got = tokenNames[types[i-1]]
		}
		panic(parseFailure{common.SyntaxError("sankey", src, tk.Starts[i], "Expecting "+expected+", got '"+got+"'", false)})
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
			panic(parseFailure{common.SyntaxError("sankey", src, start, "Invalid XML character in node name", false)})
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
		amount := field()
		source := m.findOrCreateNode(from)
		target := m.findOrCreateNode(to)
		m.addLink(source, target, parseFloat(chars.Trim(amount)))
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
