// Package class parses and lays out Mermaid's class diagrams, as Pele's src/diagrams/class
// does: a lexer and a parser that agree with Mermaid's classDiagram.jison, the DB they fill,
// buildGraph, which resolves the DB into the nodes and edges that are drawn, and Layout, which
// lays them out as a scene.
package class

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
)

// Parse parses a class diagram's text, as the parse of Pele's class diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. An error is a *diagram.Error.
func Parse(source, title string) (*DB, error) {
	db := newDB()
	if title != "" {
		db.Title = title
	}
	if err := parseInto(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared class diagram, with the text and the title Pele's core passes
// to the parse of its class diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return common.ParsePrepared(p, false, func(src string) (*DB, error) {
		return Parse(src, p.Title)
	})
}

// hierarchical reports whether a class diagram's config nests namespaces, as Pele's render
// reads it for buildGraph: unless class.hierarchicalNamespaces is false.
func hierarchical(config diagram.Config) bool {
	conf, _ := config["class"].(diagram.Config)
	v, ok := conf["hierarchicalNamespaces"].(bool)
	return !ok || v
}
