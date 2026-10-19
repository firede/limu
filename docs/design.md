# Design

## Stages

```
text ──parse──▶ model ──layout──▶ scene.Scene ──backend──▶ SVG, a GPU, a grid of pixels
                         ▲                         ▲
                 layout.Options             style.Look
         (measurer, font size, snap,   (theme, overrides, and the
          routing: where things are)    backend's own drawing: how they look)
```

What changes where things are goes to layout, and a change of it lays the diagram out
again: the measurer and its fonts, the size of text, a grid to snap to, how edges are
routed. Everything else is style, settled only when a scene is drawn, and a change of it
only redraws.

## The scene

A `scene.Scene` is a list of `scene.Item`s in the order they are drawn. An item says what
it is — its `Kind`, its `Role` (`"flowchart.node"`) and, if it has an outline, its `Shape`
(`"diam"`) — and where it is: its `Box`, and for an edge its `Route`, the corners of the
line it takes.

It also carries what a plain backend needs to draw it:

- `Parts`: its outline, as paths and boxes, filled and stroked, or stroked only for the
  details of a shape (`Line`). A box whose corners take the theme's radius is a box with
  `Rounded` set, not a path, so the radius stays a matter of style.
- `Ends`: the markers at the ends of an edge, with where their tips are and the way they
  point. `Marker.Outline` draws the ones Pele does.
- `Label`: text laid out in lines and spans, measured, with the point it is drawn about,
  its alignment and its rotation.
- `Style`: the diagram's own colors for it, as tokens of the theme, and `Author`: what the
  diagram's author set with `classDef` and `style` lines.

Paths have four kinds of step — move, line, quadratic and cubic curves — and close. Arcs
become cubic curves as they are added, since MyGo's paths have no arcs.

An item has at most one label. A thing of a diagram with more — a sequence diagram's
frame, with its kind and its condition — is several items, the shape first.

A backend with a look of its own draws an item from its role, shape, box and route, and
ignores its parts.

### What the other diagrams need of it

Before the scene was settled, the drawing of two diagrams unlike flowcharts was checked
against it:

| Pele draws | As items |
|---|---|
| sequence: an actor's box, or its figure | `Node`, `sequence.actor`, shape `rect` or `actor`; the figure's head and limbs are parts, the limbs `Line` |
| sequence: a lifeline | `Mark`, `sequence.lifeline`, one `Line` part, dashed by its style |
| sequence: a message | `Edge`, `sequence.message`, a straight or looped path, markers of Mermaid's arrow kinds, a label above it |
| sequence: an activation | `Mark`, `sequence.activation`, a box |
| sequence: a note | `Node`, `sequence.note` |
| sequence: a loop, alt or opt frame | `Group`, `sequence.frame`, its box, its tab and its dashed dividers as parts; its kind and conditions as `Text` items |
| sequence: an autonumber | `Node`, a circle, its label in the background's color |
| pie: a slice | `Mark`, `pie.slice`, a path with an arc, filled with `style.Series(i)` |
| pie: a slice's percentage, the legend's text | `Text`, the legend's aligned at its start |
| pie: a legend's swatch | `Mark`, a box filled with the slice's series |

The scene needed two things for them that flowcharts do not: labels aligned at their start
or end (`Align`, which journey, kanban, packet, railroad, requirement and treemap diagrams
use too) and labels turned (`Rotate`, for the axes of charts). Markers of other kinds —
sequence diagrams' open and asynchronous arrows, class diagrams' diamonds, entity
relationship diagrams' crow's feet — are added to `Marker.Outline` with the diagrams that
use them.

## Style

Three levels, from the least to the most a host does:

1. **Theme** (`style.Theme`): the colors and measures every diagram takes its look from,
   by token: `Bg`, `Fg`, `Muted`, `Line`, `Surface`, `SurfaceAlt`, `Border`, `Accent`,
   `Critical`, eight series, a corner radius and a stroke width. The svg package writes
   them as CSS variables, `--limu-surface` and the rest, with the theme's colors as
   fallbacks.
2. **Overrides** (`style.Overrides`): a `style.Style` for a role, laid over the diagram's
   own style for its parts. What a style leaves unset comes from under it.
3. **Drawing of one's own**: each backend takes a function that draws an item in place of
   its default, and may call the default. It belongs to the backend, as each draws with
   tools of its own: the svg package's `Options.Draw` writes markup.

A part's style is cascaded, from under to over: the theme's measures, the diagram's own
style for the part, the overrides for its role, and what its author set in the diagram's
text.

## Porting from Pele

Parsing, layout and the geometry of shapes are ported from Pele, function for function, so
that limu lays out what Pele does to the last bit when both measure text with the same
table of Arial's advances (`text.Arial`, Pele's `metricsMeasurer`). Where JavaScript's
meaning differs from Go's, the port keeps JavaScript's: strings counted in UTF-16 code
units, `\s` and `trim` with Unicode's spaces, `Math.round`'s halves.

`scripts/oracle.sh` runs Pele's TypeScript sources with Node, with no install, over the
cases in `internal/oracle/cases`, and writes what Pele makes of them as fixtures in the
packages' `testdata/pele`. The tests compare with those, so they run without Node.

The port follows Pele 1.6.0, commit `44da152`. The fixtures record the version and the
commit they were written from.

## Roadmap

| | | |
|---|---|---|
| M0 | the stages and their types, text measured and laid out, styles, the svg backend, the oracle | done |
| M1 | flowcharts from text to pixels: the parser, the layered and compound layouts, the shapes; a backend for MyGo's native UI; a look in the Platinum style in mygo-platinum | |
| M2 | the diagrams on the layered layout: state, class, entity relationship, requirement, use case, agent flow, swimlane | |
| M3 | sequence, gantt, timeline, journey, git graph, packet, kanban | |
| M4 | the charts: pie, xy, quadrant, radar, sankey, treemap, venn | |
| M5 | the rest: mindmap, architecture, C4, block, wardley, railroad, ishikawa, cynefin, event modeling, tree view, info | |
