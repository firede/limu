---
title: "Getting started"
description: "Install limu, lay out a diagram, and render it as SVG."
---

limu turns Mermaid text into a scene, which records what each part of a diagram is and where it goes. A backend then draws the scene. This page uses the svg package, which needs only Go's standard library, as limu itself does. To draw diagrams natively instead, see [MyGo](/mygo).

## Installing

limu needs Go 1.27 or later.

```shell
go get github.com/firede/limu
```

## Drawing a diagram

`limu.Layout` reads a diagram and lays it out, and `svg.Render` writes the resulting scene as an SVG document.

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

By default, layout measures text with `text.Arial`, a table of Arial's character widths, so the SVG should set its text in Arial or a font of the same width. If you draw text in other fonts, measure it with those fonts instead. See [Measuring text](/api#measuring-text).

The theme sets the diagram's colors. The svg package writes them as CSS variables with the theme's colors as fallbacks, so a page can change them without rendering the diagram again. See [Theming](/theming).

## Errors

When limu can't read a diagram, `limu.Layout` returns a `*diagram.Error`. Its `Code` says whether the type is unsupported, the text doesn't parse, or the text is over a limit. `Line` and `Column` give the position when there is one.

```go
sc, err := limu.Layout(src, nil)
var derr *diagram.Error
if errors.As(err, &derr) {
	fmt.Printf("%s at line %d: %s\n", derr.Code, derr.Line, derr.Message)
}
```

Error messages can quote the diagram's text, so display them as text rather than markup.

## Fitting a width

Set `layout.Options.MaxWidth` to the width you have. Diagrams that can get narrower will, so a wide flowchart may switch from running across to running down. Diagrams that can't get narrower keep the width they need.

```go
sc, err := limu.Layout(src, &layout.Options{MaxWidth: 480})
```

The SVG is the size of the scene. To let a page scale it down further, give it `max-width: 100%` and `height: auto` in CSS.

## Laying out again

If you lay out the same diagram more than once, for example when the available width changes, parse it once with `limu.Parse` and call `Layout` on the result as often as you need. Layout doesn't change the parsed diagram, so it's safe to call from several goroutines at once.

```go
d, err := limu.Parse(src, diagram.Options{})
if err != nil {
	return err
}
wide, err := d.Layout(&layout.Options{MaxWidth: 640})
narrow, err := d.Layout(&layout.Options{MaxWidth: 320})
```

Changing the theme doesn't need a new layout at all. Draw the same scene in the new theme.

## Diagram types

limu supports every diagram type that Pele draws, which covers every type built into Mermaid. `limu.Types` lists them. `limu.Supports` checks a text's declared type without parsing the rest.

```go
if !limu.Supports(src) {
	// Show the text as code instead.
}
```

[Examples](/examples) shows each type, and [Compatibility](/compatibility) lists the keywords that declare them.

## Next steps

- [API](/api) covers parsing and layout options, limits and the scene.
- [Theming](/theming) covers themes, per-role overrides and custom drawing.
- [MyGo](/mygo) draws diagrams natively and finds the part under the pointer.
- [SVG](/svg) covers CSS variables, classes and untrusted diagrams.
