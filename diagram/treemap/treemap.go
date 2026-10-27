// Package treemap parses Mermaid's treemaps and lays them out, as Pele's src/diagrams/treemap
// does: a tokenizer and a parser that read what Mermaid's Langium grammar reads, buildTreemap,
// which nests the rows into a tree by their indentation, and Layout, which lays the tree out
// as a squarified treemap in a scene.
package treemap

import (
	"github.com/firede/limu/diagram/internal/common"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// Node is a section or a leaf of a treemap, Pele's TreemapNode.
type Node struct {
	Name string
	// Section is set for a section, which has a list of children, perhaps empty; a leaf has
	// none, and a value.
	Section  bool
	Children []*Node
	// Value is a leaf's value as its text gives it, NaN where that is no number.
	Value float64
	// Class is the class the node's text names after :::, "" for none.
	Class string
	// CSS is the declarations of the class, Pele's cssCompiledStyles, nil when the class is
	// not defined.
	CSS []string
}

// Class is a class definition: its name and its declarations, in the order of the classDef
// lines that give them.
type Class struct {
	Name   string
	Styles []string
}

// Model is a parsed treemap, Pele's TreemapModel. A string Pele leaves undefined is empty.
type Model struct {
	Title, AccTitle, AccDescr string
	Roots                     []*Node
	Classes                   []Class
}

// item is a row of the treemap, Pele's TreemapItem.
type item struct {
	level int
	name  string
	leaf  bool
	value float64
	class string
	css   []string
}

// buildHierarchy nests the rows, as Pele's buildHierarchy does: a row belongs to the nearest
// section above it that is indented less.
func buildHierarchy(items []item) []*Node {
	type open struct {
		node  *Node
		level int
	}
	var roots []*Node
	var stack []open
	for _, it := range items {
		node := &Node{Name: it.name, Section: !it.leaf, Class: it.class, CSS: it.css}
		if !it.leaf {
			node.Children = []*Node{}
		}
		if it.leaf {
			node.Value = it.value
		}
		for len(stack) > 0 && stack[len(stack)-1].level >= it.level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, node)
		} else {
			parent := stack[len(stack)-1].node
			parent.Children = append(parent.Children, node)
		}
		if !it.leaf {
			stack = append(stack, open{node, it.level})
		}
	}
	return roots
}

// splitStyles splits the text of a classDef into declarations, as Pele's buildTreemap does: an
// escaped comma stays in its value, and other commas and semicolons separate them. What
// stands for the escaped comma meanwhile, §§§, is a comma after, wherever it was.
func splitStyles(text string) []string {
	text = strings.ReplaceAll(text, `\,`, "§§§")
	text = strings.ReplaceAll(text, ",", ";")
	text = strings.ReplaceAll(text, "§§§", ",")
	return strings.Split(text, ";")
}

// buildTreemap makes the model of a treemap's tree, as Pele's buildTreemap does: the classes
// first, from every classDef wherever it is, then the rows nested into a tree.
func buildTreemap(tree *ast, title string) *Model {
	m := &Model{}
	index := map[string]int{}
	for _, row := range tree.rows {
		if !row.classDef {
			continue
		}
		k, ok := index[row.className]
		if !ok {
			k = len(m.Classes)
			index[row.className] = k
			m.Classes = append(m.Classes, Class{Name: row.className, Styles: []string{}})
		}
		m.Classes[k].Styles = append(m.Classes[k].Styles, splitStyles(row.styleText)...)
	}

	var items []item
	for _, row := range tree.rows {
		if row.classDef {
			continue
		}
		it := row.item
		var css []string
		if it.hasClass {
			if k, ok := index[it.selector]; ok && len(m.Classes[k].Styles) > 0 {
				css = m.Classes[k].Styles
			}
		}
		items = append(items, item{
			level: row.indent, name: it.name, leaf: it.leaf, value: it.value, class: it.selector, css: css,
		})
	}

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
	m.Roots = buildHierarchy(items)
	return m
}

// Parse parses a treemap's text, as the parse of Pele's treemap does: source is what
// diagram.Prepared.Source gives, and title the title its front matter gives, "" for none. An
// error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	tree, err := parseTreemap(source)
	if err != nil {
		return nil, err
	}
	return buildTreemap(tree, title), nil
}

// ParsePrepared parses a prepared treemap.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return common.ParsePrepared(p, false, func(src string) (*Model, error) {
		return Parse(src, p.Title)
	})
}
