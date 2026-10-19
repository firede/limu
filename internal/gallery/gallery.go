// Package gallery holds limu's examples, which the website and the playground show: for each
// type of diagram limu reads but info, which only shows a version number, its title, a
// description, and its examples. One or two of a type's examples are featured, and come
// first; the website's page of examples and the playground's overview show them.
//
// The examples are limu's own, one file for each type in examples/: front matter with the
// type's title, its description, whether it is in beta, and how many of its first examples
// are featured, then each example as a heading with its title over a block of Mermaid. They
// are written to be read: each draws something a reader could meet, and the ones after the
// featured show the type's other features, each in a diagram of its own.
package gallery

import (
	"bufio"
	"embed"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/firede/limu/diagram"
)

//go:embed examples/*.md
var files embed.FS

// A Type is a type of diagram and its examples.
type Type struct {
	// Type is the type, as limu names it.
	Type diagram.Type `json:"type"`
	// Title is what the type is called, as "Entity relationship".
	Title string `json:"title"`
	// Description says what a diagram of the type shows, in a sentence.
	Description string `json:"description"`
	// Beta reports whether Mermaid marks the type with -beta, as one it may still change.
	Beta bool `json:"beta"`
	// Examples are the featured examples first, then the others.
	Examples []Example `json:"examples"`
}

// An Example is the text of a diagram, with its title.
type Example struct {
	Title    string `json:"title"`
	Source   string `json:"source"`
	Featured bool   `json:"featured,omitempty"`
}

var all = sync.OnceValue(func() []Type {
	entries, err := files.ReadDir("examples")
	if err != nil {
		panic("gallery: " + err.Error())
	}
	var types []Type
	for _, e := range entries {
		b, err := files.ReadFile(path.Join("examples", e.Name()))
		if err != nil {
			panic("gallery: " + err.Error())
		}
		t, err := Parse(strings.TrimSuffix(e.Name(), ".md"), string(b))
		if err != nil {
			panic("gallery: " + e.Name() + ": " + err.Error())
		}
		types = append(types, t)
	}
	slices.SortStableFunc(types, func(a, b Type) int {
		if a.Beta != b.Beta {
			if a.Beta {
				return 1
			}
			return -1
		}
		return strings.Compare(a.Title, b.Title)
	})
	return types
})

// All returns the types, those not in beta first, then by title. It reads the examples
// once; the slice it returns is shared, and must not be changed.
func All() []Type { return all() }

// Find returns the type t names, as limu's diagram.Type does, and whether there is one.
func Find(t string) (Type, bool) {
	for _, typ := range all() {
		if string(typ.Type) == t {
			return typ, true
		}
	}
	return Type{}, false
}

// Parse reads the file of the type name: its front matter, then each example, a heading of
// the second level with the example's title, and a block of Mermaid under it. All reads the
// files this way; the preview reads one as it is being written.
func Parse(name, text string) (Type, error) {
	t := Type{Type: diagram.Type(name)}
	featured := 0
	sc := bufio.NewScanner(strings.NewReader(text))
	line, n := "", 0
	next := func() bool {
		if !sc.Scan() {
			return false
		}
		line, n = sc.Text(), n+1
		return true
	}
	if !next() || line != "---" {
		return t, fmt.Errorf("no front matter")
	}
	for {
		if !next() {
			return t, fmt.Errorf("the front matter does not end")
		}
		if line == "---" {
			break
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			return t, fmt.Errorf("line %d: %q is not a key and a value", n, line)
		}
		switch key {
		case "title":
			t.Title = value
		case "description":
			t.Description = value
		case "beta":
			t.Beta = value == "true"
		case "featured":
			v, err := strconv.Atoi(value)
			if err != nil {
				return t, fmt.Errorf("line %d: featured: %v", n, err)
			}
			featured = v
		default:
			return t, fmt.Errorf("line %d: no key %q", n, key)
		}
	}
	for next() {
		switch {
		case strings.TrimSpace(line) == "":
		case strings.HasPrefix(line, "## "):
			t.Examples = append(t.Examples, Example{Title: strings.TrimSpace(line[3:])})
		case line == "```mermaid":
			if len(t.Examples) == 0 || t.Examples[len(t.Examples)-1].Source != "" {
				return t, fmt.Errorf("line %d: a block with no heading of its own", n)
			}
			var src strings.Builder
			for {
				if !next() {
					return t, fmt.Errorf("a block does not end")
				}
				if line == "```" {
					break
				}
				src.WriteString(line + "\n")
			}
			t.Examples[len(t.Examples)-1].Source = src.String()
		default:
			return t, fmt.Errorf("line %d: %q is neither a heading nor a block", n, line)
		}
	}
	for i := range t.Examples {
		if t.Examples[i].Source == "" {
			return t, fmt.Errorf("%q has no block", t.Examples[i].Title)
		}
		t.Examples[i].Featured = i < featured
	}
	return t, nil
}
