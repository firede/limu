// Package flowchart parses Mermaid's flowcharts, as Pele's src/diagrams/flowchart does: a
// lexer and a recursive-descent parser that agree with Mermaid's flow.jison, the DB they
// fill, and buildGraph, which resolves the DB into the nodes and edges that are drawn.
package flowchart

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// Parse parses a flowchart's text, as the parse of Pele's flowchart diagram does: source is
// what diagram.Prepared.Source gives, config the diagram's configuration, and title the title
// its front matter gives, "" for none. An error is a *diagram.Error.
func Parse(source string, config diagram.Config, title string, lineOffset int, limits diagram.Limits) (*DB, error) {
	flow, _ := config["flowchart"].(diagram.Config)
	inherit, _ := flow["inheritDir"].(bool)
	db := newDB(options{InheritDir: inherit, MaxEdges: limits.MaxEdges})
	if title != "" {
		db.Title = title
	}
	if err := parseInto(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared flowchart.
// It is limu's own: it hands Parse what Pele's caller hands the parse of its flowchart
// diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return common.ParsePrepared(p, false, func(src string) (*DB, error) {
		return Parse(src, p.Config, p.Title, p.LineOffset, p.Limits)
	})
}

// defaultCurve is the curve the flowchart's config gives its edges, as Pele's render reads
// it for buildGraph: "" where it gives none.
func defaultCurve(config diagram.Config) string {
	flow, _ := config["flowchart"].(diagram.Config)
	curve, _ := flow["curve"].(string)
	return curve
}
