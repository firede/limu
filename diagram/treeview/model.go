// Package treeview parses Mermaid's tree view diagrams and lays them out, as Pele's
// src/diagrams/treeview does: a preprocessor that reads a tree drawn with box-drawing
// characters as one indented, a tokenizer and a parser that read what Mermaid's Langium
// grammar reads, a database that nests the entries by their indentation, and Layout, which
// lays the tree out as a scene, a row to an entry.
package treeview

import (
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/diagram/internal/common"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// NodeType is whether an entry is a file or a directory, Pele's NodeType.
type NodeType string

const (
	File      NodeType = "file"
	Directory NodeType = "directory"
)

// Node is an entry of the tree, Pele's TreeNode. A string Pele leaves undefined is empty.
type Node struct {
	ID    int
	Level int
	// Name is as written, its entity codes hidden, without the / that makes a directory.
	Name string
	Type NodeType
	// Icon is an icon name as written (pack:name, file, folder), or none to hide the icon.
	Icon        string
	CSSClass    string
	Description string
	Children    []*Node
}

// Model is a parsed tree view, Pele's TreeViewModel. A string Pele leaves undefined is
// empty.
type Model struct {
	Title, AccTitle, AccDescr string
	// Root is a directory named / at level -1 that holds the entries at the top.
	Root  *Node
	Count int
}

// db is Mermaid's tree view database, Pele's TreeViewDb: entries arrive in the order of the
// text with their indentation.
type db struct {
	count int
	root  *Node
	stack []*Node
}

func newDB() *db {
	root := &Node{ID: 0, Level: -1, Name: "/", Type: Directory}
	return &db{count: 1, root: root, stack: []*Node{root}}
}

// addNode puts an entry under the last entry less indented than it, as Pele's addNode does.
func (d *db) addNode(level int, name string, nodeType NodeType, cssClass, icon, description string) {
	for level <= d.stack[len(d.stack)-1].Level {
		d.stack = d.stack[:len(d.stack)-1]
	}
	node := &Node{ID: d.count, Level: level, Name: name, Type: nodeType, Icon: icon, CSSClass: cssClass, Description: description}
	d.count++
	parent := d.stack[len(d.stack)-1]
	parent.Children = append(parent.Children, node)
	d.stack = append(d.stack, node)
}

// populate adds the entries of a tree to the database, as Pele's populate does: a name that
// ends in / is a directory's, and an empty icon() hides the icon.
func populate(tree *ast, d *db) {
	for _, node := range tree.nodes {
		level := 0
		if node.hasIndent {
			level = node.indent
		}
		name := node.name
		nodeType := File
		if strings.HasSuffix(name, "/") {
			name = name[:len(name)-1]
			nodeType = Directory
		}
		icon := ""
		if node.hasIcon {
			icon = node.iconAnnotation
			if icon == "" {
				icon = "none"
			}
		}
		d.addNode(level, name, nodeType, node.classAnnotation, icon, node.descAnn)
	}
}

// readTreeView reads the text to a syntax tree, as Pele's readTreeView does, taking a tree
// drawn with box-drawing characters too. Its errors name the places of the text as given, and
// of the text as written where source is the text a diagram's parser reads (common.Locate).
func readTreeView(source string) (*ast, error) {
	text, lineMap, columns, err := preprocessBoxDrawing(source)
	if err != nil {
		return nil, err
	}
	// Pele asks whether the text is the source by its value. limu finds the text as written
	// by where the source is held (common.Written), so the source itself is read when the
	// text is the same.
	if text == source {
		text = source
	}
	tree, err := parseTreeView(text)
	if err == nil {
		return tree, nil
	}
	// The parser already located errors in unchanged text.
	e, ok := err.(*diagram.Error)
	if !ok || text == source || e.Line < 1 || e.Line > len(lineMap) {
		return nil, err
	}
	line := lineMap[e.Line-1]
	start := lineStart(source, line)
	at := 0
	if e.Column > 0 {
		if column := columns[e.Line-1]; column != nil {
			at = column(e.Column - 1)
		} else {
			at = unitOffset(source[start:], e.Column-1)
		}
	}
	written := common.Locate(source, start+at)
	head := strings.IndexByte(e.Message, '\n')
	if head == -1 {
		head = len(e.Message)
	}
	rest := e.Message[head:]
	if e.Snippet != "" {
		rest = strings.Replace(rest, e.Snippet, written.Snippet, 1)
	}
	message := remapErrorLines(e.Message[:head], map[int]int{e.Line: written.Line}) + rest
	column, snippet := 0, ""
	if e.Column > 0 {
		column = written.Column
	}
	if e.Snippet != "" {
		snippet = written.Snippet
	}
	return nil, &diagram.Error{Code: e.Code, Message: message, Type: e.Type, Line: written.Line, Column: column, Snippet: snippet}
}

// unitOffset is the byte of s at which code unit n of it is, or the end of s.
func unitOffset(s string, n int) int {
	i := 0
	for units := 0; i < len(s) && units < n; {
		r, size := utf8.DecodeRuneInString(s[i:])
		units++
		if r > 0xffff {
			units++
		}
		i += size
	}
	return i
}

// buildTreeView makes the model of a tree, as Pele's buildTreeView does.
func buildTreeView(tree *ast, title string) *Model {
	d := newDB()
	populate(tree, d)
	m := &Model{Title: tree.title, Root: d.root, Count: d.count}
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

// Parse parses a tree view's text, as the parse of Pele's tree view does: source is what
// diagram.Prepared.Source gives, and title the title its front matter gives, "" for none. An
// error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	tree, err := readTreeView(source)
	if err != nil {
		return nil, err
	}
	return buildTreeView(tree, title), nil
}

// ParsePrepared parses a prepared tree view.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return common.ParsePrepared(p, false, func(src string) (*Model, error) {
		return Parse(src, p.Title)
	})
}
