---
title: "Compatibility"
description: "The diagrams limu reads, and where it differs from Pele and from Mermaid."
---

limu reads Mermaid's text as [Pele](https://pele.run) 1.6.0 does, and lays it out as Pele does, number for number. Where it differs from Mermaid, it mostly differs as Pele does.

## Diagram types

limu reads every type of diagram that Pele draws, which is every type built into Mermaid 12.1.0. [Examples](/examples) shows each.

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

ZenUML and Mermaid's other external plugins are not read. `limu.Supports` tells whether limu reads a text's type.

Front matter, `%%{init}%%` directives, comments, and the titles and descriptions for assistive technology (`accTitle`, `accDescr`) are read for every type. What each type's configuration reaches is as in Pele.

## Differences from Pele

### Pele's API in limu

Pele renders SVG in a browser; limu lays out a scene for a backend. Pele's options are spread over limu's stages:

| Pele | limu |
| --- | --- |
| `render(source, options)` | `limu.Layout`, then a backend, as `svg.Render` |
| `parse(source, options)` | `limu.Parse` |
| `detectType`, `supports` | `diagram.DetectType`, `limu.Supports` |
| `PeleError` and its codes | `diagram.Error` and its codes |
| `limit`, `maxEdges`, `config` | `diagram.Options` |
| `measurer`, `fontSize`, `maxWidth`, `autoDirection`, `directionBreakpoint`, `padding`, `now` | `layout.Options`, where `KeepDirection` is the opposite of `autoDirection` |
| `links`, `images`, `linkSchemes`, `imageSchemes` | `layout.Options`, and the svg package's `Options` |
| `fontFamily`, `fontFamilyMono` | The svg package's `FontFamily` and `MonoFamily` |
| `icons` | `layout.Options.Icons`, and the svg package's `IconMarkup` or limumygo's `DrawIcon` |
| `--pele-*` variables, `pele-*` classes | `--limu-*` variables, `limu-*` classes |

What belongs to Pele's place in a browser is not ported: `mount`, its zoom and `refresh`, which a host does with the scene in its own way, as limumygo's `Options.Scale`; the SVG's `responsive` sizing, `idPrefix`, `linkRel`, `outputLimit` and `sanitize`; and the entry points that load types on demand, as a Go program links what it imports.

### Drawing

- **Info.** An info diagram draws "limu" and the version of the build, where Pele draws its own.
- **The author's text styles.** Pele writes on text the author's `font-family`, `letter-spacing`, `word-spacing` and `text-decoration`, and in treemaps and venn diagrams a weight or a size it did not measure the text in. limu reads only the weight, the slant and the size, and only where layout measures text with them, as text drawn otherwise than it was measured would not fit the room layout gave it.
- **Sankey labels.** A sankey diagram's label and its value, which Pele writes as one text with the value in another color, are two items, placed by the measure of the whole. With a measurer that kerns, the gap between them is not quite Pele's.
- **Hosts outside ASCII.** A link's host outside ASCII is written in Punycode without the whole of UTS #46, so a few such hosts are written otherwise than Node writes them.
- **MyGo.** limumygo draws what the svg package writes, with the few differences [MyGo](/mygo#painting) lists.

limu lays a diagram out the same on every platform Go builds for; its tests compare with Pele on amd64 and arm64.

### Text

Pele measures text in a browser with the fonts it is drawn in, or, with no font at hand, with a table of Arial's advances. limu measures with the measurer the host gives: `text.Arial` is Pele's table, so with it, limu lays out what Pele does to the last bit. Text in Chinese, Japanese and Korean is wrapped as Pele 1.6 wraps it; see [Measuring text](/api#chinese-japanese-and-korean).

## Differences from Mermaid

### Layout and appearance

limu lays diagrams out as Pele does, which is not as Mermaid does: where things are, the room between them, and the paths of edges differ. Where Mermaid hands a diagram to a library, Pele often lays it out in a way of its own: architecture diagrams on a grid, where Mermaid uses a force-directed solver, and C4 and block diagrams with their own placing and routing.

Colors come from limu's [themes](/theming). Mermaid's themes and `themeVariables` do not apply.

### Sizing

A diagram that can draw itself narrower does, to fit `layout.Options.MaxWidth`: a wide flowchart, for one, may run down instead of across. Set `KeepDirection` to keep its direction.

### Subgraphs

A flowchart's subgraph keeps its own `direction`, even when it links to nodes outside it.

### Limits

limu limits the length of a diagram's text and the edges of a flowchart or an agent flow. See [Limits](/api#limits).

### Labels

Labels are text with line breaks, bold and italic, from `<br>`, `<b>` and `<i>` or from Markdown strings. Other HTML in a label is not drawn as HTML.

### Interaction

A `click` that opens a link gives its item a `Link`, which a host follows as it decides. A `click` that calls a function is ignored.

### Icons

A host gives icons such as `fa:fa-car`: see [MyGo](/mygo#icons-and-images) and [SVG](/svg#icons). Architecture and tree view diagrams also draw symbols of their own.

## Reporting differences

If limu reads or lays out a diagram otherwise than Pele does, that is a bug in limu: open the diagram in the [playground](/playground) and in [Pele's](https://pele.run/playground), and [report the difference](https://github.com/firede/limu/issues). If Pele draws it the same way, and it differs from Mermaid, report it to [Pele](https://github.com/obsidianmd/pele/issues).
