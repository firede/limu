// Package cynefin parses Mermaid's Cynefin diagrams and lays them out, as Pele's
// src/diagrams/cynefin does: a tokenizer and a parser that read what Mermaid's Langium
// grammar reads, buildCynefin, which turns the tree into the diagram's model, and Layout,
// which lays the five domains out as a scene, with the wavy boundaries between them and the
// transitions from one to another.
package cynefin

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The names of the five domains.
const (
	Complex     = "complex"
	Complicated = "complicated"
	Clear       = "clear"
	Chaotic     = "chaotic"
	Confusion   = "confusion"
)

// Domain is a domain the text declares and the items it lists in it, Pele's CynefinDomain.
type Domain struct {
	Name  string
	Items []string
}

// Transition is a move from one domain to another, Pele's CynefinTransition. A Label Pele
// leaves undefined is empty.
type Transition struct {
	From, To, Label string
}

// Model is a parsed Cynefin diagram, Pele's CynefinModel. A string Pele leaves undefined is
// empty.
type Model struct {
	Title, AccTitle, AccDescr string
	// Domains are the domains the text declares, in the order Pele's map of them holds:
	// where each was first declared, with the items of the last block for it. All five are
	// drawn either way.
	Domains     []Domain
	Transitions []Transition
}

// Domain is the domain of a name the text declares, or nil.
func (m *Model) Domain(name string) *Domain {
	for i := range m.Domains {
		if m.Domains[i].Name == name {
			return &m.Domains[i]
		}
	}
	return nil
}

// setDomains is Pele's CynefinDb.setDomains: a later block for a domain replaces the items
// of an earlier one, which keeps its place, as a Map's set does.
func (m *Model) setDomains(blocks []blockAst) {
	for _, block := range blocks {
		items := append([]string{}, block.items...)
		if d := m.Domain(block.domain); d != nil {
			d.Items = items
			continue
		}
		m.Domains = append(m.Domains, Domain{Name: block.domain, Items: items})
	}
}

// setTransitions is Pele's CynefinDb.setTransitions: a transition from a domain to itself is
// dropped, and an empty label is none.
func (m *Model) setTransitions(transitions []transitionAst) {
	for _, t := range transitions {
		if t.from != t.to {
			m.Transitions = append(m.Transitions, Transition{From: t.from, To: t.to, Label: t.label})
		}
	}
}

// buildCynefin makes the model of a diagram's tree, as Pele's buildCynefin does.
func buildCynefin(tree *ast, title string) *Model {
	m := &Model{}
	m.setDomains(tree.domains)
	m.setTransitions(tree.transitions)
	m.Title = tree.title
	if m.Title == "" {
		m.Title = title
	}
	if tree.accTitle != "" {
		m.AccTitle = chars.TrimStart(tree.accTitle)
	}
	if tree.accDescr != "" {
		m.AccDescr = chars.DropIndents(tree.accDescr)
	}
	return m
}

// Parse parses a Cynefin diagram's text, as the parse of Pele's cynefin diagram does:
// source is what diagram.Prepared.Source gives, and title the title its front matter gives,
// "" for none. An error is a *diagram.Error.
func Parse(source string, title string) (*Model, error) {
	tree, err := parseCynefin(source)
	if err != nil {
		return nil, err
	}
	return buildCynefin(tree, title), nil
}

// ParsePrepared parses a prepared Cynefin diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return common.ParsePrepared(p, false, func(src string) (*Model, error) {
		return Parse(src, p.Title)
	})
}
