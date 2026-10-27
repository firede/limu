---
title: "API"
description: "Parse and lay out diagrams, set limits, handle errors, and read the scene."
---

limu is split into a few packages, one for each stage of drawing a diagram. This page explains what each one is for and when you need it. [pkg.go.dev](https://pkg.go.dev/github.com/firede/limu) documents every name in full.

| Package | What it does |
| --- | --- |
| [`limu`](https://pkg.go.dev/github.com/firede/limu) | Reads a diagram and lays it out. |
| [`diagram`](https://pkg.go.dev/github.com/firede/limu/diagram) | Holds the parsing options, the diagram types and the error type. |
| [`layout`](https://pkg.go.dev/github.com/firede/limu/layout) | Holds the options that change where things go. |
| [`scene`](https://pkg.go.dev/github.com/firede/limu/scene) | Describes a laid-out diagram as a list of items. |
| [`text`](https://pkg.go.dev/github.com/firede/limu/text) | Measures text and lays out labels. |
| [`style`](https://pkg.go.dev/github.com/firede/limu/style) | Holds themes and overrides. See [Theming](/theming). |
| [`svg`](https://pkg.go.dev/github.com/firede/limu/svg) | Renders a scene as SVG. See [SVG](/svg). |
| [`limumygo`](https://pkg.go.dev/github.com/firede/limu/limumygo) | Draws a scene with MyGo's native UI. It's a separate module. See [MyGo](/mygo). |

Every package except `limumygo` uses only Go's standard library.

## The limu package

| Name | What it does |
| --- | --- |
| `Layout(src, o)` | Reads a diagram and lays it out as a scene. |
| `Parse(src, opt)` | Reads a diagram into a `Diagram` to lay out later. |
| `Diagram.Layout(o)` | Lays out the diagram. You can call it many times, with different options, from several goroutines at once. |
| `Diagram.Type()` | Returns the diagram's type, such as `diagram.Flowchart`. |
| `Diagram.MermaidType()` | Returns the type as Mermaid names it, such as `flowchart-v2` for `graph TD`. |
| `Diagram.Config()` | Returns the diagram's configuration, with its own settings over the ones it was parsed with. |
| `Types()` | Lists the diagram types limu supports. |
| `Supports(src)` | Reports whether limu supports the type a text declares. It doesn't parse the rest, so the text may still have errors. |

`Layout` is the same as calling `Parse` with default options and then `Diagram.Layout`. If you lay out the same diagram more than once, for example when its width changes, parse it only once.

## Parsing

`diagram.Options` sets the limits for the text and any configuration to apply beneath the diagram's own.

```go
d, err := limu.Parse(src, diagram.Options{
	Config: diagram.Config{"flowchart": diagram.Config{"curve": "linear"}},
})
```

`Config` plays the role of Mermaid's configuration. A diagram can also set configuration in its front matter or in an `%%{init: …}%%` directive, and its own settings win key by key. Values follow JSON, so objects are `diagram.Config`, arrays are `[]any` and numbers are `float64`.

### Limits

| Limit | Default | Option |
| --- | --- | --- |
| Length of the text | 50,000 UTF-16 code units, Mermaid's `maxTextSize` | `Limit` |
| Edges in a flowchart or agent flow | 5,000 | `MaxEdges` |

Text over a limit gives an error with code `diagram.Limit`. A negative limit turns it off. For untrusted text, keep the defaults or lower them. The edge limit matters because `A & B --> C & D` multiplies, so a few kilobytes of text can ask for millions of edges. limu can't stop a layout once it has started, and the limits are what keep its work bounded.

### Errors

`Parse`, `Layout` and `Diagram.Layout` return a `*diagram.Error` with these fields.

| Field | What it holds |
| --- | --- |
| `Code` | The kind of error. |
| `Message` | A description of the problem, which may quote the diagram. |
| `Type` | The diagram's type, if known. |
| `MermaidType` | The type as Mermaid names it, if known. |
| `Line`, `Column` | The position of the error, counted from 1, with the column in UTF-16 code units. Both are 0 when the error has no position. |
| `Snippet` | The line with the error and a caret under the position, when there is one. |

| Code | Meaning |
| --- | --- |
| `diagram.UnsupportedDiagram` | The text isn't a type limu supports. |
| `diagram.Syntax` | The text doesn't parse. |
| `diagram.Semantic` | The text parses but asks for something impossible, such as a shape that doesn't exist. |
| `diagram.Limit` | The text is over a limit, or the scene would contain a number that isn't finite where Pele would draw NaN. |

## Layout

`layout.Options` holds everything that affects where things go. Changing these options means laying the diagram out again, while changing the style doesn't. Nil options are the same as the zero value, and every zero field means the default.

| Field | Default | What it does |
| --- | --- | --- |
| `Measurer` | `text.Arial` | Measures text. See [Measuring text](#measuring-text). |
| `FontSize` | 16 | Sets the size of text that has no size of its own. |
| `Icons` | none | Reports whether you can draw an icon such as `fa:fa-car`. Labels leave room only for icons you can draw. |
| `MaxWidth` | no limit | Sets the width you have. Diagrams that can get narrower will. |
| `KeepDirection` | false | Stops a diagram that runs across from switching to run down to fit `MaxWidth`. |
| `DirectionBreakpoint` | 640 | Sets the `MaxWidth` below which a diagram may switch direction. |
| `NoLinks`, `NoImages` | false | Drop the links and images a diagram asks for. |
| `LinkSchemes`, `ImageSchemes` | see [SVG](/svg#untrusted-diagrams) | Set the URL schemes that links and images may use besides relative URLs. |
| `Padding` | 8 | Sets the space around the diagram. It's a pointer so you can set 0 with `new(0.0)`. |
| `Snap` | none | Rounds positions to multiples of this value, for backends that draw on whole pixels. |
| `Routing` | `layout.Auto` | Sets how edges run in ranked layouts. `Auto` follows the text, and `layout.Orthogonal` and `layout.Straight` override it. |
| `Monospace` | none | Sets which parts are set in the monospace face. `layout.Members` covers the members of class diagrams, `layout.Attributes` the attributes of entity relationship diagrams, `layout.Files` the entries of tree views, and `layout.Messages` the messages of sequence diagrams. A part whose author gives it a font keeps it. |
| `Now` | `time.Now()` | Sets the current time for gantt charts, which marks today and starts tasks that have no start date. |
| `Location` | `time.Local` | Sets the time zone gantt charts read and write dates in. |

`Routing` applies to flowcharts and to swimlane, agent flow, state, class, entity relationship, requirement and use case diagrams. Other diagrams route their edges as Pele does.

To render the same diagram on every build, set `Now` and `Location`. This site does that for its gantt charts.

## The scene

A `scene.Scene` is a laid-out diagram. It has a size and a list of items in drawing order. Each item records what it is and where it goes, but not how it looks. Its colors are theme tokens that are resolved only when it's drawn.

| Field | What it holds |
| --- | --- |
| `Kind` | `scene.Node`, `Group`, `Edge`, `Text` for a standalone label, or `Mark` for a shape without a label. |
| `Role` | Which part of which diagram the item is, such as `flowchart.node`. Styles are assigned by role. |
| `Shape` | The name of the item's outline, such as `rect` or `diam`. |
| `ID` | The id the diagram gave the item. |
| `Class`, `Tags` | The classes the author assigned, and extra facts the diagram records, such as a gantt task being `done`. |
| `Box` | Where the item is. `LabelBox()` returns where its label is drawn. |
| `Parts` | The item's outline as paths and boxes, each with how it's painted. |
| `Route`, `Ends` | For an edge, the corners of its line and the markers at its ends. |
| `Label`, `At`, `Align`, `Rotate` | The measured label, and where and how to draw it. |
| `Picture` | An image or icon the item shows. |
| `Style`, `Author` | The diagram's own style for the item, and what the author set with `classDef` and `style`. |
| `Link`, `Tooltip` | Where the item links to, and its tooltip. |

Because a scene is plain data, a program can read it directly. This loop prints where each node is.

```go
for _, it := range sc.Items {
	if it.Kind == scene.Node {
		fmt.Printf("%s %q at %v\n", it.Role, it.ID, it.Box)
	}
}
```

A backend draws each item's parts, the markers at an edge's ends (`Marker.Outline`) and the label. A backend with its own look can instead draw items from their role, shape, box and route. Paths use only four kinds of step, which are moves, lines, and quadratic and cubic curves. Arcs are converted to cubic curves. For backends that can't draw curves or dashes, `Path.Flatten` turns a path into polylines and `scene.Dash` cuts polylines into dashes.

## Measuring text

Layout measures every label, so a diagram must be drawn in the fonts it was measured with. Otherwise the text won't fit the space layout gave it. A `text.Measurer` returns the width of text in a given face and size.

```go
type Measurer interface {
	Width(s string, size float64, f text.Face) float64
}
```

The default, `text.Arial`, estimates widths from a table of Arial's character widths without loading any font. It matches Pele's widths exactly, and SVG drawn with it should use Arial or a font of the same width. If you draw text in other fonts, pass a measurer for those fonts. `limumygo.Fonts` is one for MyGo.

A label's span can also be `text.Sub`, `text.Sup` or `text.Underline`. Layout asks the measurer only about `Bold`, `Italic` and `Mono`. It measures subscript and superscript at the font size divided by 1.2, as Pele does, so a backend should draw them that much smaller and lowered or raised. An underline takes no room.

### Chinese, Japanese and Korean

`text.Arial` treats every Chinese, Japanese and Korean character as one em wide, which is only an estimate. Real fonts may be narrower or wider. When a label is too wide, it wraps between words. Like Pele 1.6, it also wraps between any two Chinese or Japanese characters, but never before a closing punctuation mark or after an opening one. Korean wraps only between words. If you draw this kind of text, measure it with the fonts you draw it in.
