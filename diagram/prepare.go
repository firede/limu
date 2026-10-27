// Package diagram reads what all Mermaid diagrams share, as Pele's src/preprocess.ts,
// src/detect.ts and the parse path of src/core.ts do: a diagram's front matter, its
// directives and comments, and its type. Its Error is the error of a diagram's text, as
// Pele's PeleError in src/errors.ts is; the diagrams make theirs with the helpers of
// diagram/internal/common.
//
// Prepare does what Pele's parseSource does before it hands a diagram's text to the parser
// of its type; the diagram packages under it, as flowchart, parse what it gives.
package diagram

import (
	"strconv"

	"github.com/firede/limu/diagram/internal/edit"
	"github.com/firede/limu/internal/chars"
)

// The limits a source is held to when Options leaves them at zero: Mermaid's maxTextSize, and
// the edges Pele lets a flowchart have. Mermaid stops at 500 edges; Pele draws 5,000 in tens
// of milliseconds, and `A & B --> C & D` multiplies, so that without a limit a few kilobytes
// of source ask for millions of edges.
const (
	DefaultLimit    = 50000
	DefaultMaxEdges = 5000
)

// Options are what Pele's ParseOptions hold: the limits and the configuration a source is
// read with.
type Options struct {
	// Limit is the most UTF-16 code units a source may have: DefaultLimit when it is 0, and no
	// limit when it is negative.
	Limit int
	// MaxEdges is the most edges a flowchart or an agent flow may have: DefaultMaxEdges when
	// it is 0, and no limit when it is negative.
	MaxEdges int
	// Config is configuration under the diagram's own, which takes its place key by key.
	Config Config
}

// Limits are the limits a diagram's parser holds its text to.
type Limits struct {
	// MaxEdges is the most edges a flowchart or an agent flow may have, or no limit when it is
	// negative.
	MaxEdges int
}

// Prepared is a diagram's text made ready for the parser of its type.
type Prepared struct {
	// Detected is the diagram's type, as Pele and as Mermaid name it.
	Detected
	// Text is the text after its front matter, with directives and comment lines taken out:
	// what detect read.
	Text string
	// WithComments is the text with its comment lines, for a diagram that keeps them.
	WithComments string
	// Config is the diagram's configuration, Options.Config under its own.
	Config Config
	// Title is the title its front matter gives, if HasTitle.
	Title    string
	HasTitle bool
	// LineOffset is the number of lines of front matter before the text.
	LineOffset int
	Limits     Limits

	// source is the text Prepare was given, with what is not UTF-8 replaced.
	source string
}

// Source is the text a diagram's parser reads, as Pele's parseSource hands it over: Text, or
// WithComments for a diagram that keeps its comments, with its entity codes hidden and a line
// feed after it.
func (p *Prepared) Source(keepComments bool) string {
	text := p.Text
	if keepComments {
		text = p.WithComments
	}
	return encodeEntities(text, nil) + "\n"
}

// InSource gives a function that finds a byte of the text Source gives in the diagram's text
// as its author wrote it, as the function Pele's inSource makes does: the text with its line
// breaks normalized and the line feed Source adds, and the byte in it. It reads the text
// again, recording how it was rewritten, only when the function is first called, since that
// is only to report an error. Where reading the text again does not give what Source gives, as
// for a Prepared that Prepare did not make or whose fields were changed since, it finds each
// byte where it is in the text Source gives.
func (p *Prepared) InSource(keepComments bool) func(at int) (source string, offset int) {
	var steps [][]edit.Edit
	var source string
	read, same := false, false
	return func(at int) (string, int) {
		if !read {
			read = true
			pre, err := preprocess(p.source, &steps)
			if err == nil {
				text := pre.WithComments
				if !keepComments {
					text = cleanupComments(text, &steps)
				}
				given := p.Source(keepComments)
				same = p.source != "" && encodeEntities(text, &steps)+"\n" == given
				// Match the line feed Source adds.
				source = normalizeLineBreaks(p.source) + "\n"
				if !same {
					source = given
				}
			} else {
				source = p.Source(keepComments)
			}
		}
		if !same {
			return source, at
		}
		return source, edit.InputOffset(steps, at)
	}
}

// Prepare reads a diagram's front matter, directives and type, as Pele's parseSource does
// before it calls the parser of the type. It does not know which types are supported: for one
// that is not, Unsupported is Pele's error.
//
// Pele is handed a JavaScript string, which the browser decoded from UTF-8; a Go string may
// hold bytes that are not UTF-8. Prepare first replaces those as the browser's decoder does,
// each maximal subpart of an ill-formed sequence with U+FFFD, so that what is measured and
// drawn is what Pele would measure and draw, and the scene holds only valid UTF-8.
func Prepare(source string, opt Options) (*Prepared, error) {
	source = chars.Decode(source)
	limit := opt.Limit
	if limit == 0 {
		limit = DefaultLimit
	}
	if limit > 0 && chars.Len16(source) > limit {
		return nil, newError("Diagram source is longer than the limit of "+strconv.Itoa(limit)+" characters.", Limit, "")
	}
	pre, err := preprocess(source, nil)
	if err != nil {
		return nil, err
	}
	found, ok := detectDiagram(pre.Text)
	if !ok {
		return nil, newError("No diagram type detected for the given text.", UnsupportedDiagram, "")
	}
	config := pre.Config
	if opt.Config != nil {
		config = Config{}
		for k, v := range opt.Config {
			config[k] = v
		}
		for k, v := range pre.Config {
			config[k] = v
		}
	}
	maxEdges := opt.MaxEdges
	if maxEdges == 0 {
		maxEdges = DefaultMaxEdges
	}
	return &Prepared{
		Detected:     found,
		Text:         pre.Text,
		WithComments: pre.WithComments,
		Config:       config,
		Title:        pre.Title,
		HasTitle:     pre.HasTitle,
		LineOffset:   pre.LineOffset,
		Limits:       Limits{MaxEdges: maxEdges},
		source:       source,
	}, nil
}

// Unsupported is the error of a diagram of a type that has no parser, as Pele's parseSource
// gives it.
func Unsupported(found Detected) *Error {
	e := newError(`Diagram type "`+string(found.Type)+`" is not supported yet.`, UnsupportedDiagram, found.Type)
	e.MermaidType = found.MermaidType
	return e
}

// TagError gives err, if it is an *Error, the diagram's type where it has none, as Pele's
// tagErrors does for the errors of a diagram's parser and of its layout, and writes the
// entity codes in its message and snippet as the text has them, not as the parser read them.
// It is err.
func TagError(found Detected, err error) error {
	if e, ok := err.(*Error); ok {
		if e.Type == "" {
			e.Type = found.Type
		}
		if e.MermaidType == "" {
			e.MermaidType = found.MermaidType
		}
		e.Message = edit.WrittenCodes(e.Message)
		e.Snippet = edit.WrittenCodes(e.Snippet)
	}
	return err
}
