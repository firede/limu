---
title: "Compatibility"
description: "Supported diagram types, and how limu differs from Pele and Mermaid."
---

limu parses Mermaid text the way [Pele](https://pele.run) 1.9.0 does and produces the same layout, number for number. Where limu differs from Mermaid, it's almost always because Pele does.

## Diagram types

limu supports every diagram type Pele draws, which covers every type built into Mermaid 12.1.0. [Examples](/examples) shows each one.

| Type | Keywords |
| --- | --- |
| Flowchart | `graph`, `flowchart` |
| Swimlane | `swimlane-beta` |
| Sequence | `sequenceDiagram` |
| Class | `classDiagram` |
| State | `stateDiagram`, `stateDiagram-v2` |
| Entity relationship | `erDiagram` |
| Gantt | `gantt` |
| Git graph | `gitGraph` |
| Pie | `pie` |
| Mindmap | `mindmap` |
| Kanban | `kanban` |
| Timeline | `timeline` |
| User journey | `journey` |
| Quadrant chart | `quadrantChart` |
| XY chart | `xychart`, `xychart-beta` |
| Requirement | `requirementDiagram` |
| Sankey | `sankey`, `sankey-beta` |
| Architecture | `architecture-beta` |
| C4 | `C4Context`, `C4Container`, `C4Component`, `C4Dynamic`, `C4Deployment` |
| Block | `block`, `block-beta` |
| Agentflow | `agentflow-beta` |
| Railroad | `railroad-beta`, `railroad-ebnf-beta`, `railroad-abnf-beta`, `railroad-peg-beta` |
| Tree view | `treeView-beta` |
| Cynefin | `cynefin-beta` |
| Wardley map | `wardley-beta` |
| Event modeling | `eventmodeling` |
| Use case | `usecase-beta` |
| Venn | `venn-beta` |
| Ishikawa | `ishikawa`, `ishikawa-beta` |
| Packet | `packet`, `packet-beta` |
| Radar | `radar-beta` |
| Treemap | `treemap`, `treemap-beta` |
| Info | `info` |

ZenUML and other external Mermaid plugins aren't supported. Use `limu.Supports` to check whether limu supports a text's type.

Every type supports front matter, `%%{init}%%` directives, comments, and the accessibility title and description (`accTitle`, `accDescr`). Which configuration settings each type honors is the same as in Pele.

## Differences from Pele

### Pele's API in limu

Pele renders SVG in a browser, while limu lays out a scene for a backend to draw. Pele's options are therefore spread across limu's stages.

| Pele | limu |
| --- | --- |
| `render(source, options)` | `limu.Layout`, then a backend such as `svg.Render` |
| `parse(source, options)` | `limu.Parse` |
| `detectType`, `supports` | `diagram.DetectType`, `limu.Supports` |
| `type`, `mermaidType` | `Diagram.Type()`, `Diagram.MermaidType()` |
| `PeleError` and its codes | `diagram.Error` and its codes |
| `limit`, `maxEdges`, `config` | `diagram.Options` |
| `measurer`, `fontSize`, `maxWidth`, `autoDirection`, `directionBreakpoint`, `padding`, `now` | `layout.Options`, where `KeepDirection` is the opposite of `autoDirection` |
| `links`, `images`, `linkSchemes`, `imageSchemes` | `layout.Options` and the svg package's `Options` |
| `fontFamily`, `fontFamilyMono` | The svg package's `FontFamily` and `MonoFamily` |
| `icons` | `layout.Options.Icons`, plus the svg package's `IconMarkup` or limumygo's `DrawIcon` |
| `--pele-*` variables, `pele-*` classes | `--limu-*` variables, `limu-*` classes |

Features that only make sense in a browser aren't ported. `mount`, its zoom and `refresh` are left to the app, which can do the same with the scene, for example through limumygo's `Options.Scale`. The SVG options `responsive`, `idPrefix`, `linkRel`, `outputLimit` and `sanitize` aren't ported either. Neither are the entry points that load diagram types on demand, since a Go program only links what it imports.

Pele's `math` option isn't ported. With it, Pele typesets `$$…$$` in a label through a function the app gives. Without it, Pele keeps the label as text, and limu always does.

### Drawing

limu draws a few things differently from Pele.

An info diagram shows "limu" and the build's version instead of Pele's.

Pele passes the author's `font-family`, `letter-spacing`, `word-spacing` and `text-decoration` on to the text, and in treemaps and venn diagrams it also applies a weight or size it didn't measure the text with. limu reads only the font's weight, style and size, and only where layout measures text with them, because text drawn differently from how it was measured won't fit the space layout gave it.

Pele writes a sankey label and its value as one piece of text with the value in another color. limu makes them two items and places them by the width of the whole. With a measurer that kerns, the gap between them can differ slightly from Pele's.

A link whose host name isn't ASCII is converted to Punycode without the full UTS #46 mapping, so a few such host names come out differently from Node's.

limumygo draws the same picture as the svg package, apart from the few differences listed under [MyGo](/mygo#painting).

limu produces the same layout on every platform Go supports, and its tests compare against Pele on both amd64 and arm64.

### Text

In a browser, Pele measures text with the fonts it draws in. Without fonts, it uses a table of Arial's character widths. limu measures with the measurer you give it. `text.Arial` is the same table, so with it, limu's layout matches Pele's to the last bit. Chinese, Japanese and Korean text wraps the way it does in Pele 1.6. See [Measuring text](/api#chinese-japanese-and-korean).

## Differences from Mermaid

### Layout and appearance

limu lays out diagrams the way Pele does, which isn't how Mermaid does it. Node positions, spacing and edge paths all differ. Where Mermaid hands a diagram to an outside library, Pele often uses a layout of its own. Pele puts architecture diagrams on a grid where Mermaid uses a force-directed solver, and C4 and block diagrams have their own placement and routing.

Colors come from limu's [themes](/theming). Mermaid's themes and `themeVariables` don't apply.

### Sizing

Diagrams that can get narrower do so to fit `layout.Options.MaxWidth`. A wide flowchart, for example, may run down instead of across. Set `KeepDirection` to prevent that.

### Subgraphs

A flowchart subgraph keeps its own `direction`, even when it links to nodes outside it.

### Limits

limu limits the length of a diagram's text and the number of edges in flowcharts and agent flows. See [Limits](/api#limits).

### Labels

Labels support line breaks, bold and italic, written with `<br>`, `<b>` and `<i>` or as Markdown strings. They also read `<code>`, `<sub>`, `<sup>` and `<u>`, and their kin such as `<strong>`, `<em>`, `<kbd>` and `<ins>`. Other HTML tags are dropped and their text kept. Text shaped like a tag that HTML doesn't have, such as the type in `List<T>`, is shown as written.

### Interaction

A `click` that opens a link sets the item's `Link`, and your app decides whether to follow it. A `click` that calls a function is ignored.

### Icons

Your app supplies icons such as `fa:fa-car`. See [MyGo](/mygo#icons-and-images) and [SVG](/svg#icons). Architecture and tree view diagrams also have built-in symbols.

## Reporting differences

If limu parses or lays out a diagram differently from Pele, that's a bug in limu. Open the diagram in limu's [playground](/playground) and in [Pele's](https://pele.run/playground), then [report the difference](https://github.com/firede/limu/issues). If Pele draws it the same way and both differ from Mermaid, report it to [Pele](https://github.com/obsidianmd/pele/issues) instead.
