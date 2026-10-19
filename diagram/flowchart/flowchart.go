// Package flowchart parses Mermaid's flowcharts, as Pele's src/diagrams/flowchart does: a
// lexer and a recursive-descent parser that agree with Mermaid's flow.jison, the DB they
// fill, and BuildGraph, which resolves the DB into the nodes and edges that are drawn.
package flowchart

import "github.com/firede/limu/diagram"

// Parse parses a flowchart's text, as the parse of Pele's flowchart diagram does: source is
// what diagram.Prepared.Source gives, config the diagram's configuration, and title the title
// its front matter gives, "" for none. An error is a *diagram.Error.
func Parse(source string, config diagram.Config, title string, lineOffset int, limits diagram.Limits) (*DB, error) {
	flow, _ := config["flowchart"].(diagram.Config)
	inherit, _ := flow["inheritDir"].(bool)
	db := NewDB(Options{InheritDir: inherit, MaxEdges: limits.MaxEdges})
	if title != "" {
		db.Title = title
	}
	if err := ParseInto(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared flowchart.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return Parse(p.Source(false), p.Config, p.Title, p.LineOffset, p.Limits)
}

// DefaultCurve is the curve the flowchart's config gives its edges, as Pele's render reads
// it for BuildGraph: "" where it gives none.
func DefaultCurve(config diagram.Config) string {
	flow, _ := config["flowchart"].(diagram.Config)
	curve, _ := flow["curve"].(string)
	return curve
}
