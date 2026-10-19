// Package railroad parses Mermaid's railroad diagrams and lays them out, as Pele's
// src/diagrams/railroad does. Mermaid reads rules in four notations, each with a keyword of
// its own: its own notation of constructor calls (railroad-beta, ir.go), EBNF
// (railroad-ebnf-beta, ebnf.go), ABNF (railroad-abnf-beta, abnf.go) and PEG
// (railroad-peg-beta, peg.go). Each is read into a syntax tree of its own by a tokenizer and a
// parser that read what Mermaid's Langium grammar reads, and the trees are turned into one
// model of rules, which Layout lays out as a scene: a diagram for each rule, with its track
// running from left to right through the terminals and non-terminals of its definition.
package railroad

import (
	"math"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// NodeType is the kind of a node of a rule's definition.
type NodeType uint8

const (
	Terminal    NodeType = iota + 1 // text written as it is
	Nonterminal                     // the name of a rule
	Special                         // text that says what it stands for
	Sequence                        // its kids, one after another
	Choice                          // one of its kids
	Optional                        // its one kid, or nothing
	Repetition                      // its one kid, from Min to Max times
)

// Node is a node of a rule's definition, Pele's RailroadNode.
type Node struct {
	Type NodeType
	// Text is a terminal's value, a non-terminal's name, or a special's text.
	Text string
	// Kids are a sequence's elements, a choice's alternatives, or the one element of an
	// optional or a repetition.
	Kids []*Node
	// Min and Max are how often a repetition repeats; Max is +Inf when the count has no upper
	// bound.
	Min, Max float64
}

// parts are the node's kids, as Pele's parts gives them for the rule model.
func (n *Node) parts() []*Node { return n.Kids }

// Rule is a rule: its name and its definition, Pele's RailroadRule.
type Rule struct {
	Name       string
	Definition *Node
}

// Notation is which of Mermaid's four grammars the text was written in.
type Notation string

const (
	NotationRailroad Notation = "railroad"
	NotationEBNF     Notation = "ebnf"
	NotationABNF     Notation = "abnf"
	NotationPEG      Notation = "peg"
)

// Model is a parsed railroad diagram, Pele's RailroadDb. A string Pele leaves undefined is
// empty.
type Model struct {
	Notation                  Notation
	Title, AccTitle, AccDescr string
	// Rules are every rule in source order. A name defined twice appears twice.
	Rules  []Rule
	byName map[string]int
}

func (m *Model) setTitle(text string) { m.Title = text }

// setAccTitle sets the accessible title, as Pele's setAccTitle does, with
// text.replace(/^\s+/g, ”).
func (m *Model) setAccTitle(text string) { m.AccTitle = chars.TrimStart(text) }

// setAccDescription sets the accessible description, as Pele's setAccDescription does, with
// text.replace(/\n\s+/g, '\n').
func (m *Model) setAccDescription(text string) { m.AccDescr = chars.DropIndents(text) }

// addRule adds a rule, as Pele's addRule does. A name defined again is drawn again; looking
// the name up gives the later definition.
func (m *Model) addRule(rule Rule) {
	m.Rules = append(m.Rules, rule)
	if m.byName == nil {
		m.byName = map[string]int{}
	}
	m.byName[rule.Name] = len(m.Rules) - 1
}

// GetRule is the last rule of a name, as Pele's getRule gives it.
func (m *Model) GetRule(name string) (Rule, bool) {
	i, ok := m.byName[name]
	if !ok {
		return Rule{}, false
	}
	return m.Rules[i], true
}

// notationOf is the notation of a text by its keyword, as Pele's notationOf reads it with
// /^\s*railroad-(ebnf-|abnf-|peg-)?beta/i: Mermaid has a detector per keyword, each matching
// without regard to case. It is "" for none.
func notationOf(source string) Notation {
	rest := source[chars.SpaceRun(source, 0):]
	if !chars.HasPrefixFold(rest, "railroad-") {
		return ""
	}
	rest = rest[len("railroad-"):]
	for _, n := range []struct {
		infix    string
		notation Notation
	}{{"ebnf-", NotationEBNF}, {"abnf-", NotationABNF}, {"peg-", NotationPEG}} {
		if chars.HasPrefixFold(rest, n.infix) && chars.HasPrefixFold(rest[len(n.infix):], "beta") {
			return n.notation
		}
	}
	if chars.HasPrefixFold(rest, "beta") {
		return NotationRailroad
	}
	return ""
}

// parsed is what Pele's Parsed holds of a syntax tree: its title and accessibility
// statements.
type parsed struct {
	title, accTitle, accDescr string
}

// grammarOf parses the text in one notation, as Pele's grammar does: its tree's statements,
// and its rules.
func grammarOf(source string, notation Notation) (parsed, []Rule, error) {
	switch notation {
	case NotationEBNF:
		ast, err := parseEbnf(source)
		if err != nil {
			return parsed{}, nil, err
		}
		return ast.parsed(), ebnfRules(ast), nil
	case NotationABNF:
		ast, err := parseAbnf(source)
		if err != nil {
			return parsed{}, nil, err
		}
		return ast.parsed(), abnfRules(ast), nil
	case NotationPEG:
		ast, err := parsePeg(source)
		if err != nil {
			return parsed{}, nil, err
		}
		return ast.parsed(), pegRules(ast), nil
	}
	ast, err := parseIr(source)
	if err != nil {
		return parsed{}, nil, err
	}
	return ast.parsed(), irRules(ast), nil
}

// populate parses the text in one notation and loads the result into the model, as Pele's
// populate does, as Mermaid's four parsers do.
func populate(m *Model, source string, notation Notation) error {
	ast, rules, err := grammarOf(source, notation)
	if err != nil {
		return err
	}
	m.Notation = notation
	if ast.accDescr != "" {
		m.setAccDescription(ast.accDescr)
	}
	if ast.accTitle != "" {
		m.setAccTitle(ast.accTitle)
	}
	if ast.title != "" {
		m.setTitle(ast.title)
	}
	for _, rule := range rules {
		m.addRule(rule)
	}
	return nil
}

// Parse parses a railroad diagram's text, as Pele's parseRailroad does: source is what
// diagram.Prepared.Source gives, and title the title its front matter gives, "" for none. An
// error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	notation := notationOf(source)
	if notation == "" {
		return nil, diagram.NewError("No railroad diagram keyword found.", diagram.Syntax, "railroad")
	}
	m := &Model{Notation: NotationRailroad}
	if title != "" {
		m.Title = title
	}
	if err := populate(m, source, notation); err != nil {
		return nil, err
	}
	return m, nil
}

// ParsePrepared parses a prepared railroad diagram.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}

// infinite is the bound of a repetition with none.
var infinite = math.Inf(1)
