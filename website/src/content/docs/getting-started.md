---
title: "Getting started"
description: "Install limu, lay a diagram out, and draw it as SVG."
---

limu reads a diagram's Mermaid text and lays it out as a scene: each part of the diagram, what it is and where it is. A backend draws the scene. This page lays a diagram out and draws it with the svg package, which, like limu itself, needs only Go's standard library. To draw diagrams natively, see [MyGo](/mygo).

## Installing

limu needs Go 1.27 or later.

```shell
go get github.com/firede/limu
```

## Drawing a diagram

`limu.Layout` reads a diagram's text and lays it out; `svg.Render` writes the scene as an SVG document.

```go
package main

import (
	"fmt"
	"log"

	"github.com/firede/limu"
	"github.com/firede/limu/style"
	"github.com/firede/limu/svg"
)

func main() {
	sc, err := limu.Layout(`flowchart LR
  text[Mermaid text] --> layout[Layout] --> scene[Scene]`, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(svg.Render(sc, &svg.Options{
		Look:       style.Look{Theme: style.Kohu()},
		FontFamily: "Arial, Helvetica, sans-serif",
	}))
}
```

With no options, layout measures text with `text.Arial`, a table of Arial's advances, so the SVG sets its text in Arial or a font as wide. A program that draws in fonts of its own measures with them; see [Measuring text](/api#measuring-text).

The theme gives the diagram its colors. The svg package writes them as CSS variables, with the theme's colors as their fallbacks, so a page can change them without drawing the diagram again. See [Theming](/theming).

## Errors

`limu.Layout` gives a `*diagram.Error` for text it cannot read: text of no type limu reads, text that does not parse, or text past a limit. Its `Code` says which, and `Line` and `Column` say where, when the error has a place.

```go
sc, err := limu.Layout(src, nil)
var derr *diagram.Error
if errors.As(err, &derr) {
	fmt.Printf("%s at line %d: %s\n", derr.Code, derr.Line, derr.Message)
}
```

A message can hold text from the diagram. Show it as text, not as markup.

## Fitting a width

`layout.Options.MaxWidth` is the width the host has for the diagram. A diagram that can draw itself narrower does: a flowchart that runs across, for one, turns to run down when it does not fit. A diagram that cannot is as wide as it needs.

```go
sc, err := limu.Layout(src, &layout.Options{MaxWidth: 480})
```

The SVG's size is the scene's. To scale it down further in a page, give it `max-width: 100%` and `height: auto` in CSS.

## Laying out again

A host that lays a diagram out more than once, as when the width it has changes, parses the text once with `limu.Parse` and lays the `Diagram` out as often as it needs. Laying it out changes nothing in it, so it may be laid out from many goroutines at once.

```go
d, err := limu.Parse(src, diagram.Options{})
if err != nil {
	return err
}
wide, err := d.Layout(&layout.Options{MaxWidth: 640})
narrow, err := d.Layout(&layout.Options{MaxWidth: 320})
```

A change of theme needs no layout at all: the same scene is drawn in any theme.

## Types of diagram

limu reads every type of diagram that Pele draws, which is every type built into Mermaid. `limu.Types` lists them. `limu.Supports` tells whether limu reads the type a text declares, without parsing the rest of it:

```go
if !limu.Supports(src) {
	// Show the text as code instead.
}
```

[Examples](/examples) shows each type, and [Compatibility](/compatibility) lists the keywords that declare them.

## Next

- [API](/api): the options of parsing and layout, the limits, and the scene.
- [Theming](/theming): themes, overrides by role, and drawing of your own.
- [MyGo](/mygo): drawing diagrams natively, and finding the part under the pointer.
- [SVG](/svg): the SVG's variables and classes, and diagrams that are not trusted.
