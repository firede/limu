---
title: "Architecture"
description: "How limu is built, and how it is ported from Pele and checked against it."
---

This page summarizes [limu's design](https://github.com/firede/limu/blob/main/docs/design.md) for people who use limu. The full design document is written for people who work on it.

## Where the pipeline branches

Mermaid and Pele lay out a diagram and write it as SVG for a browser to draw.

```mermaid draw title="A renderer of SVG"
flowchart LR
  text[Mermaid text] --> layout[Layout] --> svg[SVG] --> browser[Browser]
```

limu stops one step earlier, at the scene, and the pipeline branches there to different backends.

```mermaid draw title="limu" mark="scene"
flowchart LR
  text[Mermaid text] --> layout[Layout] --> scene[Scene]
  scene --> mygo[MyGo, on the GPU]
  scene --> svg[SVG]
  scene --> own[A backend of your own]
```

With an SVG renderer, the SVG is all an app gets, and an app that doesn't draw SVG, such as a native app, has nothing to work with. A scene is the laid-out diagram as data, before anything is drawn. An app can draw it with whatever tools it has, and can ask it what is where.

## The stages

A diagram passes through three stages, each in its own package.

1. **Parsing** reads the text into a model. It handles front matter and directives, detects the type, and hands the body to that type's parser.
2. **Layout** places the model's parts and produces a scene, measuring text with the measurer you provide. The scene records what each part is and where it goes, but not how it looks.
3. **A backend** draws the scene and picks its colors from a look. The svg package writes SVG, and limumygo draws with MyGo.

| Package | What it does |
| --- | --- |
| `limu` | Parses a diagram and lays it out. |
| `diagram` | Reads the front matter, directives and type that every diagram shares. Each type has a package under it that parses and lays out that type. |
| `text` | Measures text with the measurer you provide, and lays out labels with line breaks, wrapping, emphasis and icons. |
| `layout` | Holds the layout options. |
| `scene` | Describes a laid-out diagram. |
| `style` | Holds themes and per-role overrides. |
| `svg` | Renders a scene as SVG. |
| `limumygo` | Draws a scene with MyGo. It's a separate module. |

Every package except limumygo uses only Go's standard library.

## Layout and style

Anything that changes where things go or how big they are belongs to layout, and changing it means laying the diagram out again. That covers the measurer and its fonts, text size, available width, padding, snapping to a grid, edge routing, and the current date for gantt charts. Everything else is style, which is applied only when the scene is drawn, so changing it only redraws.

That's why a scene holds no colors. Each part refers to colors by token, such as `Surface` or `Border`, and says what it is by its role, such as `flowchart.node`. The backend looks up the actual colors in the theme and the overrides for each role. The same scene can be drawn in a light theme, a dark theme, or an app's own look without being laid out again. See [Theming](/theming).

![Boxes, paths and labels, with no style.](/pictures/scene.svg "The scene")
![Drawn in limu’s light theme.](/pictures/kohu.png "Kohu")
![Only the colors change.](/pictures/kohu-dark.png "KohuDark")
![Each part drawn its own way, with MyGo’s glass.](/pictures/glass.png "Liquid Glass")

The last picture keeps the scene's geometry and replaces only the drawing. Its nodes and edge labels are panes of MyGo's Liquid Glass, which bend what lies under them at the rim. That's hard to do with SVG in a browser. The program that draws these pictures is [limumygo/examples/pictures](https://github.com/firede/limu/tree/main/limumygo/examples/pictures).

A backend can go further and lay the diagram out again for a look of its own. [mygo-platinum](https://github.com/firede/mygo-platinum) draws Mermaid in the Platinum look of late-1990s desktops. It measures text in its own bitmap faces, so the boxes take other sizes, and it draws every line and arrowhead a pixel at a time. Its [examples/diagrams](https://github.com/firede/mygo-platinum/tree/main/examples/diagrams) drew this picture.

![Laid out again in bitmap faces, and drawn a pixel at a time.](/pictures/platinum.png "Platinum")

## Writing a backend

A backend walks the scene's items in order and draws each one. It fills and strokes the item's `Parts` as their `Paint` says, using the colors the look resolves for the item (`style.Look.Resolve`). It then draws the markers at an edge's ends (`Marker.Outline`), the item's picture, and its label, setting each span of each line in its face around `At`. A backend with its own look can instead draw items from their `Role`, `Shape`, `Box` and `Route`. The svg package is a complete backend in about 700 lines, and the [API](/api#the-scene) page describes what an item holds.

## Ported from Pele

Most of limu is ported function by function from [Pele](https://pele.run), the Mermaid renderer made by the Obsidian team. The port keeps Pele's names where Go allows, so the two codebases can be read side by side. limu follows Pele 1.9.0.

limu's tests check its output against Pele's, number for number. A tool called the oracle runs Pele's TypeScript source with Node over more than a hundred thousand diagrams, taken from Pele's tests and documentation and from variations of them, and saves Pele's output as fixtures. limu's tests compare against those fixtures, and even a difference in the last bit of a number counts as a bug in the port. The tests run on both amd64 and arm64.

To match Pele to the last bit, the port keeps JavaScript's behavior wherever it affects measuring or placement. Strings are counted in UTF-16 code units, white space and letter case follow JavaScript, `Math.round` rounds halves the same way, trigonometry and JSON parsing follow V8, and products are kept separate from the sums they appear in, where Go would otherwise fuse them into a single rounding. Text is measured the way Pele measures it without fonts, from a table of Arial's character widths.

Matching Pele is how the port is verified, not the goal in itself. Where Mermaid hands a diagram to an outside library, Pele often uses a layout of its own, and limu follows Pele. See [Compatibility](/compatibility).
