// Package class parses and lays out Mermaid's class diagrams, as Pele's src/diagrams/class
// does: a lexer and a parser that agree with Mermaid's classDiagram.jison, the DB they fill,
// BuildGraph, which resolves the DB into the nodes and edges that are drawn, and Layout, which
// lays them out as a scene.
package class

import "github.com/firede/limu/diagram"

// Parse parses a class diagram's text, as the parse of Pele's class diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. An error is a *diagram.Error.
func Parse(source, title string) (*DB, error) {
	db := NewDB()
	if title != "" {
		db.Title = title
	}
	if err := ParseInto(source, db); err != nil {
		return nil, err
	}
	return db, nil
}

// ParsePrepared parses a prepared class diagram.
func ParsePrepared(p *diagram.Prepared) (*DB, error) {
	return Parse(p.Source(false), p.Title)
}

// Hierarchical reports whether a class diagram's config nests namespaces, as Pele's render
// reads it for BuildGraph: unless class.hierarchicalNamespaces is false.
func Hierarchical(config diagram.Config) bool {
	conf, _ := config["class"].(diagram.Config)
	v, ok := conf["hierarchicalNamespaces"].(bool)
	return !ok || v
}
