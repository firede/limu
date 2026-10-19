// Package gallery holds the examples the playground shows: for each type of diagram limu
// reads but info, which only shows a version number, its title, a description, and its
// examples. One or two of a type's examples are featured, as Pele's examples page shows one
// or two of each type, and the playground shows them on its overview.
//
// The examples are the diagrams of Mermaid's documentation (https://mermaid.js.org, MIT
// License, Copyright (c) 2014 - 2022 Knut Sveidqvist), as Pele keeps them in its
// tests/corpus, and, for the featured examples of five types, the examples Pele's website
// shows in their place (https://github.com/obsidianmd/pele, MIT License, Copyright (c) 2026
// kepano). gen.mjs writes them to examples.json from a checkout of Pele, choosing and titling
// them as Pele's website does; examples.json says which Pele in its source field.
package gallery

import (
	_ "embed"
	"encoding/json"
	"sync"

	"github.com/firede/limu/diagram"
)

//go:embed examples.json
var data []byte

// A Type is a type of diagram and its examples.
type Type struct {
	// Type is the type, as limu names it.
	Type diagram.Type `json:"type"`
	// Title is what the type is called, as "Entity relationship".
	Title string `json:"title"`
	// Description says what a diagram of the type shows, in a sentence.
	Description string `json:"description"`
	// Beta reports whether Mermaid marks the type with -beta, as one it may still change:
	// none of its examples is written without it.
	Beta bool `json:"beta"`
	// Examples are the featured examples first, then the others.
	Examples []Example `json:"examples"`
}

// An Example is the text of a diagram, with a short title taken from it.
type Example struct {
	Title    string `json:"title"`
	Source   string `json:"source"`
	Featured bool   `json:"featured,omitempty"`
}

var all = sync.OnceValue(func() []Type {
	var file struct {
		Types []Type `json:"types"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		panic("gallery: examples.json: " + err.Error())
	}
	return file.Types
})

// All returns the types, those Mermaid does not mark as beta first, then by title. It reads
// examples.json once; the slice it returns is shared, and must not be changed.
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
