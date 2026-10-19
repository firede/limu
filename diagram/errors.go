package diagram

import (
	"strconv"
	"strings"

	"github.com/firede/limu/internal/chars"
)

// Code says what kind of error an Error is, as Pele's PeleErrorCode does.
type Code string

const (
	// UnsupportedDiagram is text of no diagram type known, or of one not supported.
	UnsupportedDiagram Code = "unsupported-diagram"
	// Syntax is text that does not parse.
	Syntax Code = "syntax"
	// Semantic is text that parses but asks for what cannot be, as a shape of no such name.
	Semantic Code = "semantic"
	// Limit is text past a limit, as of its length or of a flowchart's edges.
	Limit Code = "limit"
)

// Error is an error in a diagram's text, as Pele's PeleError is. Line and Column count from 1,
// the column in UTF-16 code units as JavaScript counts them, and are 0 where the error has no
// place.
type Error struct {
	Code    Code
	Message string
	// Type is the type of the diagram the error is in, where it is known.
	Type    string
	Line    int
	Column  int
	Snippet string
}

func (e *Error) Error() string { return e.Message }

// NewError is an error with no place.
func NewError(message string, code Code, typ string) *Error {
	return &Error{Code: code, Message: message, Type: typ}
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
func SyntaxError(typ, src string, offset int, detail string, lexical bool) *Error {
	loc := Locate(src, offset)
	var message string
	if lexical {
		message = "Lexical error on line " + strconv.Itoa(loc.Line) + ". Unrecognized text.\n" + loc.Snippet
	} else {
		message = "Parse error on line " + strconv.Itoa(loc.Line) + ":\n" + loc.Snippet + "\n" + detail
	}
	return &Error{Code: Syntax, Message: message, Type: typ, Line: loc.Line, Column: loc.Column, Snippet: loc.Snippet}
}
