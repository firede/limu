// Package agentflow parses Mermaid's agent flow diagrams, as Pele's src/diagrams/agentflow
// does: a lexer and a recursive-descent parser that agree with Mermaid's agentflow.jison, the
// DB they fill, with its metadata, element positions and diagnostics, and the graph that is
// drawn, with its flows, shapes and palette slots. Layout lays the graph out as a scene.
package agentflow

import "github.com/firede/limu/diagram"

// Parse parses an agent flow's text, as the parse of Pele's agentflow diagram does, and
// builds the graph it draws, which records the diagnostics of its shapes and flows: source is
// what diagram.Prepared.Source gives with comments kept, config the diagram's configuration,
// and title the title its front matter gives, "" for none. An error is a *diagram.Error.
func Parse(source string, config diagram.Config, title string, lineOffset int, limits diagram.Limits) (*DB, error) {
	// Mermaid reads this setting from the flowchart section for agent flows too.
	flow, _ := config["flowchart"].(diagram.Config)
	inherit, _ := flow["inheritDir"].(bool)
	dir, _ := config["direction"].(string)
	db := NewDB(Options{InheritDir: inherit, Direction: dir, LineOffset: lineOffset, MaxEdges: limits.MaxEdges})
	if title != "" {
		db.Title = title
	}
	if err := ParseInto(source, db); err != nil {
		return nil, err
	}
	db.Graph(DefaultCurve(config))
	return db, nil
}

// ParsePrepared parses a prepared agent flow.
// It reads the source with its comments, as Pele's agentflow diagram has keepComments.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return Parse(p.Source(true), p.Config, p.Title, p.LineOffset, p.Limits)
}

// DefaultCurve is the curve the flowchart's config gives edges, as Pele's agentGraph reads it:
// "" where it gives none.
func DefaultCurve(config diagram.Config) string {
	flow, _ := config["flowchart"].(diagram.Config)
	curve, _ := flow["curve"].(string)
	return curve
}
