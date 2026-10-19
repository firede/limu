package common

import (
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// The errors of diagrams' text, as Pele's errors.ts makes them.

// NewError is an error with no place.
func NewError(message string, code diagram.Code, typ diagram.Type) *diagram.Error {
	return &diagram.Error{Code: code, Message: message, Type: typ}
}

// Location is a place in a text: its line and column, counted from 1, and its line with a
// caret under the column.
type Location struct {
	Line    int
	Column  int
	Snippet string
}

// Locate finds the place of byte offset in src, as Pele's locate finds that of a code unit:
// the column is counted in UTF-16 code units.
func Locate(src string, offset int) Location {
	offset = min(max(offset, 0), len(src))
	line := 1
	lineStart := 0
	for i := strings.IndexByte(src, '\n'); i != -1 && i < offset; {
		line++
		lineStart = i + 1
		next := strings.IndexByte(src[i+1:], '\n')
		if next == -1 {
			break
		}
		i += 1 + next
	}
	lineEnd := strings.IndexByte(src[offset:], '\n')
	if lineEnd == -1 {
		lineEnd = len(src)
	} else {
		lineEnd += offset
	}
	column := chars.Len16(src[lineStart:offset]) + 1
	text := src[lineStart:lineEnd]
	return Location{Line: line, Column: column, Snippet: text + "\n" + strings.Repeat("-", column-1) + "^"}
}

// SyntaxError is the error of text that does not parse at byte offset, worded as Jison's
// parsers word it, as Pele's syntaxError does. A lexical error is text no token starts with.
func SyntaxError(typ diagram.Type, src string, offset int, detail string, lexical bool) *diagram.Error {
	loc := Locate(src, offset)
	var message string
	if lexical {
		message = "Lexical error on line " + strconv.Itoa(loc.Line) + ". Unrecognized text.\n" + loc.Snippet
	} else {
		message = "Parse error on line " + strconv.Itoa(loc.Line) + ":\n" + loc.Snippet + "\n" + detail
	}
	return &diagram.Error{Code: diagram.Syntax, Message: message, Type: typ, Line: loc.Line, Column: loc.Column, Snippet: loc.Snippet}
}
