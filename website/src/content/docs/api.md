---
title: "API"
description: "What limu's packages give a program: laying diagrams out, their limits and errors, the options of layout, and the scene."
---

limu is a few packages, each a stage of drawing a diagram. This page says what each gives a program and when to reach for it; [pkg.go.dev](https://pkg.go.dev/github.com/firede/limu) has every name in full.

| Package | What it does |
| --- | --- |
| [`limu`](https://pkg.go.dev/github.com/firede/limu) | Reads a diagram's text and lays it out. |
| [`diagram`](https://pkg.go.dev/github.com/firede/limu/diagram) | The options of parsing, the types of diagram, and the error of a diagram's text. |
| [`layout`](https://pkg.go.dev/github.com/firede/limu/layout) | The options of layout: what changes where things are. |
| [`scene`](https://pkg.go.dev/github.com/firede/limu/scene) | A diagram laid out: its items, where each is and what it is. |
| [`text`](https://pkg.go.dev/github.com/firede/limu/text) | Measuring text, and laying out labels. |
| [`style`](https://pkg.go.dev/github.com/firede/limu/style) | How a scene looks: themes and overrides. See [Theming](/theming). |
| [`svg`](https://pkg.go.dev/github.com/firede/limu/svg) | Draws a scene as SVG. See [SVG](/svg). |
| [`limumygo`](https://pkg.go.dev/github.com/firede/limu/limumygo) | Draws a scene with MyGo's native UI, in a module of its own. See [MyGo](/mygo). |

All but `limumygo` import only Go's standard library.

## The limu package

| Name | What it does |
| --- | --- |
| `Layout(src, o)` | Reads a diagram's text and lays it out as a scene. |
| `Parse(src, opt)` | Reads a diagram's text into a `Diagram`, to lay out later. |
| `Diagram.Layout(o)` | Lays the diagram out. It may be called many times, with other options, from many goroutines at once. |
| `Diagram.Type()` | The diagram's type, as `diagram.Flowchart`. |
| `Diagram.Config()` | The diagram's configuration: its own, over the configuration it was parsed with. |
| `Types()` | The types of diagram limu reads. |
| `Supports(src)` | Whether limu reads the type of diagram a text declares. It does not parse the rest, which may still be in error. |

`Layout` is `Parse` with the default options, then `Diagram.Layout`. A host that lays the same diagram out more than once, as when its width changes, parses it once.

## Parsing

`diagram.Options` holds what `Parse` reads a text with: the limits it holds the text to, and configuration under the diagram's own.

```go
d, err := limu.Parse(src, diagram.Options{
	Config: diagram.Config{"flowchart": diagram.Config{"curve": "linear"}},
})
```

`Config` takes the place of Mermaid's configuration, which a diagram also sets in its front matter or in an `%%{init: …}%%` directive; what the diagram sets wins, key by key. Its values are JSON's: objects as `diagram.Config`, arrays as `[]any`, numbers as `float64`.

### Limits

| Limit | Default | Option |
| --- | --- | --- |
| Length of the text | 50,000 UTF-16 code units, as Mermaid's `maxTextSize` | `Limit` |
| Edges in a flowchart or an agent flow | 5,000 | `MaxEdges` |

A text past a limit is an error of code `diagram.Limit`. A negative limit is no limit. For text that is not trusted, keep the defaults or lower them: the edges are limited because `A & B --> C & D` multiplies, so that a few kilobytes of text can ask for millions of edges. limu cannot stop a layout once it has started; the limits are what keeps its work in bounds.

### Errors

`Parse`, `Layout` and `Diagram.Layout` give a `*diagram.Error`, with these fields:

| Field | What it holds |
| --- | --- |
| `Code` | The kind of error. |
| `Message` | What is wrong, which can hold text from the diagram. |
| `Type` | The diagram's type, where it is known. |
| `Line`, `Column` | Where the error is, from 1, the column in UTF-16 code units; 0 where it has no place. |
| `Snippet` | The line the error is on, with a caret under where it is, when the error has a place. |

| Code | When |
| --- | --- |
| `diagram.UnsupportedDiagram` | The text is of no type limu reads. |
| `diagram.Syntax` | The text does not parse. |
| `diagram.Semantic` | The text parses but asks for what cannot be, as a shape of no such name. |
| `diagram.Limit` | The text is past a limit, or the scene would hold a number that is not finite, where Pele would draw NaN. |

## Layout

`layout.Options` holds what changes where things are. A change of them lays a diagram out again; a change of style does not. Nil options are the zero `Options`, and each zero value is a default.

| Field | Default | What it does |
| --- | --- | --- |
| `Measurer` | `text.Arial` | Measures text. See [Measuring text](#measuring-text). |
| `FontSize` | 16 | The size of text with no size of its own. |
| `Icons` | none | Whether the host can draw an icon by name, as `fa:fa-car`; labels leave room for those it can. |
| `MaxWidth` | no limit | The width the host has. A diagram that can draw itself narrower does. |
| `KeepDirection` | false | Keeps a diagram that runs across from turning to run down to fit `MaxWidth`. |
| `DirectionBreakpoint` | 640 | Under this `MaxWidth`, a diagram may turn. |
| `NoLinks`, `NoImages` | false | Leave out the links and the images a diagram asks for. |
| `LinkSchemes`, `ImageSchemes` | see [SVG](/svg#diagrams-that-are-not-trusted) | The schemes links and images may use besides a relative address. |
| `Padding` | 8 | The room about the diagram. It is a pointer, so that 0 can be set: `new(0.0)`. |
| `Snap` | none | Rounds where things are to whole multiples of it, for a backend that draws on whole pixels. |
| `Routing` | `layout.Auto` | How edges run in the diagrams laid out in ranks: as the text asks, `layout.Orthogonal` or `layout.Straight`. |
| `Now` | `time.Now()` | The present, for a gantt chart: where today is drawn, and where a task with no start begins. |
| `Location` | `time.Local` | The time zone a gantt chart reads and writes dates in. |

`Routing` reaches flowcharts and swimlane, agent flow, state, class, entity relationship, requirement and use case diagrams. The others route their edges as Pele does.

A build that must draw the same diagram whenever it runs sets `Now` and `Location`, as this site does for its gantt charts.

## The scene

A `scene.Scene` is a diagram laid out: its size, and its items, in the order they are drawn. An item says what it is and where it is, but not how it looks; its colors are tokens of the theme, settled when it is drawn.

| Field | What it holds |
| --- | --- |
| `Kind` | `scene.Node`, `Group`, `Edge`, `Text` (a label on its own) or `Mark` (a shape with no label). |
| `Role` | Which part of which diagram it is, as `flowchart.node`. Styles are given by role. |
| `Shape` | The name of its outline, as `rect` or `diam`, where it has one. |
| `ID` | The id the diagram's text gave it. |
| `Class`, `Tags` | The classes its author gave it, and what the diagram says it is beyond its role, as a gantt task's `done`. |
| `Box` | Where it is. `LabelBox()` is where its label is drawn. |
| `Parts` | Its outline: paths and boxes, each with how it is painted. |
| `Route`, `Ends` | For an edge: the corners of the line it takes, and the markers at its ends. |
| `Label`, `At`, `Align`, `Rotate` | Its label, laid out and measured, and where and how it is drawn. |
| `Picture` | An image or an icon it shows. |
| `Style`, `Author` | The diagram's own style for it, and what its author set with `classDef` and `style`. |
| `Link`, `Tooltip` | Where it leads, and its tooltip. |

A program reads a scene as data. This one prints where each node of a diagram is:

```go
for _, it := range sc.Items {
	if it.Kind == scene.Node {
		fmt.Printf("%s %q at %v\n", it.Role, it.ID, it.Box)
	}
}
```

A backend draws each item's parts, the markers at an edge's ends (`Marker.Outline`) and its label; a backend with a look of its own draws an item from its role, shape, box and route instead. Paths have four kinds of step, moves, lines, and quadratic and cubic curves, as arcs are made of cubic curves. `Path.Flatten` gives a path as polylines, and `scene.Dash` cuts polylines into dashes, for a backend that draws neither curves nor dashes.

## Measuring text

Layout measures every label, and a diagram is drawn in the fonts it was measured in, or its text does not fit the room layout gave it. A `text.Measurer` gives the width of text in a face at a size:

```go
type Measurer interface {
	Width(s string, size float64, f text.Face) float64
}
```

`text.Arial`, the default, estimates widths from a table of Arial's advances, with no font at hand; its widths are Pele's, to the last bit, and the SVG it is drawn in sets its text in Arial or a font as wide. A host that draws in fonts of its own gives a measurer of those fonts, as `limumygo.Fonts` is for MyGo.

### Chinese, Japanese and Korean

`text.Arial` only estimates text in Chinese, Japanese and Korean, at an em a character; the fonts that draw it may be narrower or wider. A label wider than its room wraps between words, and, as Pele 1.6 does, between any two Chinese or Japanese characters, but not before a closing mark or after an opening one; Korean wraps between words. A host that draws such text measures it with the fonts it draws it in.
