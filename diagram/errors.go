package diagram

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
	Type Type
	// MermaidType is that type as Mermaid names it, where it is known.
	MermaidType MermaidType
	Line        int
	Column      int
	Snippet     string
}

func (e *Error) Error() string { return e.Message }

// newError is an error with no place, as common.NewError is for the diagrams.
func newError(message string, code Code, typ Type) *Error {
	return &Error{Code: code, Message: message, Type: typ}
}
