package treeview

import (
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
)

// Mermaid's box-drawing preprocessor, as Pele's boxDrawing.ts has it: it turns `├──`, `└──`
// and `│`, and their heavy variants, into the indentation the grammar reads. Columns are
// counted in UTF-16 code units, as Pele counts them.

// The box-drawing characters, Pele's ALL_BOX_CHARS, and those that start a branch, its
// BRANCH_CHAR. Its DASH_CHAR is ─ and ━.
const (
	allBoxChars = "─━│┃└┗├┣"
	branchChars = "└┗├┣"
)

// indentUnit is Pele's INDENT_UNIT.
const indentUnit = "    "

// isBoxDrawingFormat reports whether a line holds a box-drawing character, as Pele's
// isBoxDrawingFormat does.
func isBoxDrawingFormat(lines []string) bool {
	for _, line := range lines {
		if strings.ContainsAny(line, allBoxChars) {
			return true
		}
	}
	return false
}

// branchAt is where BRANCH_CHAR first matches in a line: the byte and the code unit, or -1.
func branchAt(line string) (at, unit int) {
	at = strings.IndexAny(line, branchChars)
	if at < 0 {
		return -1, -1
	}
	return at, chars.Len16(line[:at])
}

// inferSegmentWidth is how many columns a level of the tree takes, as Pele's
// inferSegmentWidth finds it: where the first branch that is not at the start of its line is.
func inferSegmentWidth(contentLines []string) int {
	for _, line := range contentLines {
		if _, unit := branchAt(line); unit > 0 {
			return unit
		}
	}
	return 4
}

// spacesOr reports whether every character of s from byte i is white space, as \s matches,
// or one of extra, and there is at least one: /^[\s…]+$/ from i.
func spacesOr(s string, i int, extra string) bool {
	if i >= len(s) {
		return false
	}
	for i < len(s) {
		c, size := chars.UnitAt(s, i)
		if !chars.IsSpace(c) && !(size > 1 && strings.Contains(extra, s[i:i+size])) {
			return false
		}
		i += size
	}
	return true
}

// decorationOnly is Pele's DECORATION_ONLY, /^[\s│┃]+$/.
func decorationOnly(line string) bool { return spacesOr(line, 0, "│┃") }

// boxOnly is Pele's BOX_ONLY, /^[\s─━│┃└┗├┣]+$/.
func boxOnly(line string) bool { return spacesOr(line, 0, allBoxChars) }

// metadataLine is Pele's METADATA_LINE, /^\s*(title[\t ]|accTitle[\t ]*:|accDescr[\t ]*[:{])/.
func metadataLine(line string) bool {
	rest := line[chars.SpaceRun(line, 0):]
	switch {
	case strings.HasPrefix(rest, "title"):
		return len(rest) > 5 && (rest[5] == '\t' || rest[5] == ' ')
	case strings.HasPrefix(rest, "accTitle"):
		i := blanks(rest, len("accTitle"))
		return i < len(rest) && rest[i] == ':'
	case strings.HasPrefix(rest, "accDescr"):
		i := blanks(rest, len("accDescr"))
		return i < len(rest) && (rest[i] == ':' || rest[i] == '{')
	}
	return false
}

// commentLine is Pele's COMMENT_LINE, /^\s*%%/.
func commentLine(line string) bool {
	return strings.HasPrefix(line[chars.SpaceRun(line, 0):], "%%")
}

// remapErrorLines writes the line numbers an error's message names as those of the original
// text, as Pele's remapErrorLines does with /\bline\s+(\d+)\b/gi. lineMap holds, for each
// line of the text read, from 1, the line of the original it came from.
func remapErrorLines(message string, lineMap []int) string {
	word := func(i int) bool { return i >= 0 && i < len(message) && chars.IsWord(int(message[i])) }
	var b strings.Builder
	last := 0
	for i := 0; i+4 <= len(message); i++ {
		if !chars.HasPrefixFold(message[i:], "line") || word(i-1) {
			continue
		}
		j := chars.SpaceRun(message, i+4)
		if j == i+4 {
			continue
		}
		k := j
		for k < len(message) && chars.IsDigit(int(message[k])) {
			k++
		}
		if k == j || word(k) {
			continue
		}
		replaced := message[i:k]
		if original := lookup(lineMap, message[j:k]); original > 0 {
			replaced = "line " + strconv.Itoa(original)
		}
		b.WriteString(message[last:i])
		b.WriteString(replaced)
		last = k
		i = k - 1
	}
	b.WriteString(message[last:])
	return b.String()
}

// lookup is lineMap.get(parseInt(digits, 10)) for a line number written in digits, or 0
// where the map has none.
func lookup(lineMap []int, digits string) int {
	digits = strings.TrimLeft(digits, "0")
	if len(digits) > 15 {
		return 0
	}
	n, _ := strconv.Atoi(digits)
	if n < 1 || n > len(lineMap) {
		return 0
	}
	return lineMap[n-1]
}

// failAt is the error of a line of the original text, Pele's fail.
func failAt(line int, message string) *diagram.Error {
	return &diagram.Error{Code: diagram.Syntax, Message: "Line " + strconv.Itoa(line) + ": " + message, Type: "treeView", Line: line}
}

// preprocessBoxDrawing turns a tree drawn with box-drawing characters into one indented as
// the grammar reads it, as Pele's preprocessBoxDrawing does. The line map holds, for each
// line of the text given back, the line of the input it came from; it is empty when the
// text is the input unchanged.
func preprocessBoxDrawing(input string) (string, []int, error) {
	if !strings.ContainsAny(input, allBoxChars) {
		return input, nil, nil
	}
	lines := strings.Split(input, "\n")

	keywordIdx := -1
	for i, line := range lines {
		if chars.Trim(line) == "treeView-beta" {
			keywordIdx = i
			break
		}
	}
	if keywordIdx == -1 {
		return input, nil, nil
	}

	passes := func(line string) bool {
		return chars.Trim(line) == "" || commentLine(line) || metadataLine(line)
	}

	var contentLines []string
	for _, line := range lines[keywordIdx+1:] {
		if passes(line) || decorationOnly(line) {
			continue
		}
		contentLines = append(contentLines, strings.ReplaceAll(line, "\t", "    "))
	}
	if !isBoxDrawingFormat(contentLines) {
		return input, nil, nil
	}

	segmentWidth := inferSegmentWidth(contentLines)
	var out []string
	var lineMap []int
	keep := func(text string, index int) {
		out = append(out, text)
		lineMap = append(lineMap, index+1)
	}

	for i := 0; i <= keywordIdx; i++ {
		keep(lines[i], i)
	}
	for i := keywordIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if passes(line) {
			keep(line, i)
			continue
		}
		if decorationOnly(line) {
			continue
		}

		normalized := strings.ReplaceAll(line, "\t", "    ")
		if at, unit := branchAt(normalized); at >= 0 {
			depth := int(jsmath.Round(float64(unit)/float64(segmentWidth))) + 1
			pos := at + len("├")
			for strings.HasPrefix(normalized[pos:], "─") || strings.HasPrefix(normalized[pos:], "━") {
				pos += len("─")
			}
			for pos < len(normalized) && normalized[pos] == ' ' {
				pos++
			}
			content := chars.TrimEnd(normalized[pos:])
			if content == "" {
				return "", nil, failAt(i+1, "Empty node — expected a filename or directory name after the box-drawing prefix")
			}
			keep(strings.Repeat(indentUnit, depth)+content, i)
		} else if boxOnly(normalized) {
			continue
		} else if strings.ContainsAny(normalized, allBoxChars) {
			// A name that happens to contain a box character, such as "Section ─ A.txt", is
			// a root item.
			keep(line, i)
		} else if chars.SpaceRun(normalized, 0) > 0 {
			return "", nil, failAt(i+1,
				"Unexpected indentation without box-drawing characters. In box-drawing format, use ├── or └── prefixes for indented nodes.")
		} else {
			keep(line, i)
		}
	}
	return strings.Join(out, "\n"), lineMap, nil
}
