package main

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/diagram"
)

// A problem is an error in the text, as the playground shows it: a message for the status
// bar, the whole of the error for the editor's hover, and where it is in the text, as byte
// offsets, when the error says.
type problem struct {
	Status     string // "Line 2, column 11: Parse error: Expecting 'ACTOR', got 'TXT'"
	Detail     string // the error's message in full
	Located    bool
	Start, End int // the bytes of the text the error underlines
}

// problemOf is the problem err is in src. limu's errors count lines and columns from 1, the
// columns in UTF-16 code units, in the text its parser reads: the text after the front
// matter, with its comment lines and directives taken out. problemOf finds that line in
// src again (sourceLine), and underlines from the column to the end of the word there; the
// status gives the line and the column in src.
func problemOf(src string, err error) problem {
	var e *diagram.Error
	if !errors.As(err, &e) {
		return problem{Status: err.Error(), Detail: err.Error()}
	}
	title := summary(e)
	p := problem{Status: title, Detail: e.Message}
	if e.Line <= 0 {
		return p
	}
	lines := strings.Split(src, "\n")
	snippet, _, _ := strings.Cut(e.Snippet, "\n")
	line, past := sourceLine(src, lines, e.Line, snippet)
	start := 0
	for i := range line {
		start += len(lines[i]) + 1
	}
	text := lines[line]
	at := byteOfColumn(text, e.Column)
	// A line past the end, where an error expects more of the text, and a blank line are
	// shown at the end of the last line before them that holds anything.
	if past || strings.TrimSpace(text) == "" {
		for line > 0 && strings.TrimSpace(lines[line]) == "" {
			line--
			start -= len(lines[line]) + 1
		}
		text = lines[line]
		at = len(strings.TrimRight(text, " \t\r"))
	}
	end := at
	for end < len(text) && !strings.ContainsRune(" \t\r", rune(text[end])) {
		_, n := utf8.DecodeRuneInString(text[end:])
		end += n
	}
	if end == at && at > 0 {
		// At the end of the line: the character before it.
		_, n := utf8.DecodeLastRuneInString(text[:at])
		at -= n
	}
	p.Located, p.Start, p.End = true, start+at, start+end
	p.Status = fmt.Sprintf("Line %d, column %d: %s", line+1, columnOfByte(text, at), title)
	return p
}

// summary is the first line of an error's message without the line it gives, which the
// status says, followed by what the message says after the error's snippet, as "Expecting
// 'ACTOR', got 'TXT'".
func summary(e *diagram.Error) string {
	msg := strings.Split(e.Message, "\n")
	title := strings.TrimSuffix(cutOnLine(msg[0]), ":")
	var rest []string
	snip := strings.Split(e.Snippet, "\n")
	for _, l := range msg[1:] {
		if strings.TrimSpace(l) == "" || l == "^" || containsLine(snip, l) {
			continue
		}
		rest = append(rest, strings.TrimSpace(l))
	}
	if len(rest) > 0 {
		title += ": " + strings.Join(rest, " ")
	}
	return title
}

// cutOnLine takes " on line N" out of a message, as in "Parse error on line 2:".
func cutOnLine(s string) string {
	i := strings.Index(s, " on line ")
	if i < 0 {
		return s
	}
	j := i + len(" on line ")
	k := j
	for k < len(s) && '0' <= s[k] && s[k] <= '9' {
		k++
	}
	if k == j {
		return s
	}
	return s[:i] + s[k:]
}

func containsLine(lines []string, l string) bool {
	for _, s := range lines {
		if s == l {
			return true
		}
	}
	return false
}

// sourceLine is the line of src, from 0, that line n of the text the parser read is: the
// lines of src after its front matter, less the lines of comments and directives the
// parser does not see. Where that line is not the error's snippet, as for a diagram whose
// parser keeps its comments, it is the line nearest it that is. A line past the end of the
// text is its last line, and sourceLine reports that it was past.
func sourceLine(src string, lines []string, n int, snippet string) (int, bool) {
	from := 0
	if pre, err := diagram.Prepare(src, diagram.Options{Limit: -1}); err == nil {
		from = pre.LineOffset
	}
	// The lines the parser reads, in order: those that are neither blank before the first
	// that is not, nor a comment, nor a directive.
	var read []int
	begun := false
	inDirective := false
	for i := from; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		switch {
		case inDirective:
			inDirective = !strings.Contains(t, "}%%")
			continue
		case strings.HasPrefix(t, "%%{"):
			inDirective = !strings.Contains(t[3:], "}%%")
			continue
		case strings.HasPrefix(t, "%%") && len(t) > 2:
			continue
		case !begun && t == "":
			continue
		}
		begun = true
		read = append(read, i)
	}
	if n-1 >= len(read) {
		return len(lines) - 1, true
	}
	guess := read[n-1]
	if snippet == "" || lines[guess] == snippet {
		return guess, false
	}
	for d := 1; d < len(lines); d++ {
		for _, i := range []int{guess - d, guess + d} {
			if i >= 0 && i < len(lines) && lines[i] == snippet {
				return i, false
			}
		}
	}
	return guess, false
}

// byteOfColumn is the byte of s that a column, from 1 in UTF-16 code units, is at.
func byteOfColumn(s string, col int) int {
	units := 0
	for i, r := range s {
		if units >= col-1 {
			return i
		}
		units++
		if r >= 0x10000 {
			units++
		}
	}
	return len(s)
}

// columnOfByte is the column, from 1 in UTF-16 code units, that byte i of s is at.
func columnOfByte(s string, i int) int {
	col := 1
	for _, r := range s[:i] {
		col++
		if r >= 0x10000 {
			col++
		}
	}
	return col
}
