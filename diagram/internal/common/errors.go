package common

import (
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
	"unsafe"

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

// An origin is a text a diagram's parser is reading and the function that finds a byte of it
// in the text its author wrote, Pele's Origin.
type origin struct {
	find func(at int) (source string, offset int)
}

// textID is a text by where it is held. The parser of a diagram reads a text that Prepared's
// Source has just made, so no other text is held at the same place while it is read.
type textID struct {
	data *byte
	n    int
}

func idOf(s string) textID { return textID{unsafe.StringData(s), len(s)} }

// origins are the origins of the texts being parsed. Pele keeps one origin in a variable while
// its parser runs; limu's parsers may run in many goroutines at once, so it keeps one for each
// text, which the parser of that text alone reads.
var origins sync.Map // textID → *origin

// ParsePrepared parses a prepared diagram with parse, given the text Source gives, as Pele's
// parseSource does: the errors parse locates in that text report where they are in the text
// its author wrote (locatingIn), and its errors are tagged with diagram.TagError (tagErrors).
func ParsePrepared[T any](p *diagram.Prepared, keepComments bool, parse func(src string) (T, error)) (T, error) {
	// A copy of its own, since a text is known by where it is held, and Source may give one
	// that is held elsewhere too, as "\n" for an empty text is.
	src := strings.Clone(p.Source(keepComments))
	id := idOf(src)
	origins.Store(id, &origin{find: p.InSource(keepComments)})
	defer origins.Delete(id)
	m, err := parse(src)
	if err != nil {
		return m, diagram.TagError(p.Detected, err)
	}
	return m, nil
}

// Written finds byte offset of src in the text its author wrote, as Pele's written does: when
// src is the text a diagram's parser is reading, the source with its line breaks normalized
// and a line feed after it, and the byte in it; otherwise src and offset as they are.
func Written(src string, offset int) (string, int) {
	if o, ok := origins.Load(idOf(src)); ok {
		return o.(*origin).find(offset)
	}
	return src, offset
}

// Locate finds the place of byte offset in src, as Pele's locate finds that of a code unit:
// the column is counted in UTF-16 code units. An offset at or past the end of what src holds
// but white space is put after its last character, and a place in a text a diagram's parser
// is reading is found in the text its author wrote (Written).
func Locate(src string, offset int) Location {
	// Keep EOF errors on the last content line.
	if end := len(chars.TrimEnd(src)); end > 0 && offset >= end {
		// Pele steps back one code unit, to the last character or the low half of it, and on
		// past it; limu steps back to the start of the character.
		_, size := utf8.DecodeLastRuneInString(src[:end])
		src, offset = Written(src, end-size)
		_, size = utf8.DecodeRuneInString(src[min(offset, len(src)):])
		offset += size
	} else {
		src, offset = Written(src, offset)
	}
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
