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
	Type Type
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
}

// Source is the text a diagram's parser reads, as Pele's parseSource hands it over: Text, or
// WithComments for a diagram that keeps its comments, with its entity codes hidden and a line
// feed after it.
func (p *Prepared) Source(keepComments bool) string {
	text := p.Text
	if keepComments {
		text = p.WithComments
	}
	return encodeEntities(text) + "\n"
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
	pre, err := preprocess(source)
	if err != nil {
		return nil, err
	}
	typ := detect(pre.Text)
	if typ == "" {
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
		Type:         typ,
		Text:         pre.Text,
		WithComments: pre.WithComments,
		Config:       config,
		Title:        pre.Title,
		HasTitle:     pre.HasTitle,
		LineOffset:   pre.LineOffset,
		Limits:       Limits{MaxEdges: maxEdges},
	}, nil
}

// Unsupported is the error of a diagram of a type that has no parser, as Pele's parseSource
// gives it.
func Unsupported(typ Type) *Error {
	return newError(`Diagram type "`+string(typ)+`" is not supported yet.`, UnsupportedDiagram, typ)
}
