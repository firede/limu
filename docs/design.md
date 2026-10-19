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
routed, and for a gantt chart the present and the time zone its dates are read in. Everything else is style, settled only when a scene is drawn, and a change of it
only redraws.

## The scene

A `scene.Scene` is a list of `scene.Item`s in the order they are drawn. An item says what
it is — its `Kind`, its `Role` (`"flowchart.node"`) and, if it has an outline, its `Shape`
(`"diam"`) — and where it is: its `Box`, and for an edge its `Route`, the corners of the
line it takes.

It also carries what a plain backend needs to draw it:

- `Parts`: its outline, as paths and boxes, each with how it is painted in the item's style
  (below). A box whose corners take the theme's radius is a box with `Rounded` set, not a
  path, so the radius stays a matter of style.
- `Ends`: the markers at the ends of an edge, with where their tips are and the way they
  point. `Marker.Outline` gives the parts Pele draws them with.
- `Label`: text laid out in lines and spans, measured, with the point it is drawn about,
  its alignment and its rotation.
- `Style`: the diagram's own colors for it, as tokens of the theme, and `Author`: what the
  diagram's author set with `classDef` and `style` lines.

A part's `Painting` says which of the style's colors it takes, so that a part keeps its
role in a theme that changes them:

| Painting | Fill | Stroke | For |
|---|---|---|---|
| `FillAndStroke` | the style's fill | the style's stroke | most outlines |
| `StrokeOnly` | none | the style's stroke | an edge, the lid of a cylinder |
| `FillOnly` | the style's fill | none | the body of a datastore |
| `Solid` | the theme's line | the style's stroke | the dot of a final state, the bar of a fork |
| `Hollow` | the theme's background | the style's stroke | the ring of a final state, which hides what is under it |
| `Shade` | the theme's background, or over a fill the author set, the background at 60% over it | none | a stripe of an entity's table |

Paths have four kinds of step — move, line, quadratic and cubic curves — and close. Arcs
become cubic curves as they are added, since MyGo's paths have no arcs.

An item has at most one label. A thing of a diagram with more — a sequence diagram's
frame, with its kind and its condition — is several items, the shape first.

A backend with a look of its own draws an item from its role, shape, box and route, and
ignores its parts.

### What the other diagrams need of it

Before the scene was settled, the drawing of two diagrams unlike flowcharts was checked
against it; the charts and the rest of Pele's diagrams were drawn in it since. They take
these items:

| Pele draws | As items |
|---|---|
| sequence: a participant, at the head of its lifeline and again at its foot | `Node`, `sequence.actor`, its kind its shape (`rect` for a participant); a figure's head and limbs, a database's lid and a queue's end are parts, the lines `StrokeOnly` |
| sequence: a lifeline | `Mark`, `sequence.lifeline`, one solid path, as Pele 1.6 draws it |
| sequence: a message | `Edge`, `sequence.message`, a straight or looped path, markers of Mermaid's arrow kinds, the circles of a central connection as `FillOnly` parts; its label a `Text` item, `sequence.message-label` |
| sequence: an activation | `Mark`, `sequence.activation`, a box |
| sequence: a note | `Node`, `sequence.note` |
| sequence: a loop, alt or opt frame | `Group`, `sequence.frame`, its box; each dashed divider a `Mark`, `sequence.divider`, since a part takes its dash from its item's style; its tab and kind a `Text` item, `sequence.frame-kind`, with the tab a part; each condition a `Text` item, `sequence.condition` |
| sequence: a rect block, a box of participants | `Group`, `sequence.rect` or `sequence.box`, a box filled in the theme's alternate surface, at half its opacity unless the author gave a fill |
| sequence: an autonumber | `Node`, `sequence.number`, a rounded box, its label in the background's color |
| sequence: a destroyed participant's cross | `Mark`, `sequence.destroy` |
| pie: a slice | `Mark`, `pie.slice`, a path of arcs, filled with `style.Series(i)` |
| pie: a slice's percentage | `Text`, `pie.slice-label`, in the background's color on the slice, the text's outside it |
| a chart's legend entry (pie, XY, radar) | `Node`, its swatch a part, a box of radius 2, and its label |
| XY: a series | `Mark`, `xychart.bars` (a box per bar) or `xychart.line` (its path, its dots `FillOnly` parts) |
| a chart's turned axis title (XY, quadrant) | `Text`, `Rotate` −90 |
| radar: the graticule, the spokes | `Mark`, `radar.graticule` (a ring per part) and `radar.axes` |
| radar: a curve | `Mark`, `radar.curve`, filled `Series(i)` at 0.2, with round joins |
| quadrant: a quadrant, a point | `Node`, `quadrant.quadrant`; `Mark`, `quadrant.point`, a circle, its label a `Text` item |
| sankey: a node, a band | `Node`, `sankey.node`; `Mark`, `sankey.link`, filled at 0.4 |
| sankey: a label, its value in another color | two `Text` items, `sankey.label` and `sankey.value`, placed by the measure of the whole |
| treemap: a section, a leaf | `Group`, `treemap.section`, tinted at 0.12; `Node`, `treemap.leaf`; a value a `Text` item |
| venn: a set, an overlap with a style | `Mark`, `venn.set`, a circle at 0.2; `Mark`, `venn.overlap`, an outline of arcs |
| mindmap: a node, a branch | `Node`, `mindmap.node` or `mindmap.root`, its shape from `draw`, one without a shape a stadium filled `Series(i)` at 0.22, or no parts for a leaf; `Edge`, `mindmap.edge`, in `Series(i)`; a branch has no group, and each of its items the class `section-N` |
| architecture: a group, a service, a junction | `Group`, `architecture.group`, tinted `SurfaceAlt` at 0.5; `Node`, `architecture.service`, a rounded square with its title below; `Node`, `architecture.junction`, a dot |
| an icon Pele draws itself (architecture, tree view) | `Mark`, `architecture.icon` or `treeview.icon`, after the item it is of, read from Pele's path data with `scene.ParsePath` and stroked round, since a part has no stroke of its own; an icon the host draws is a `Picture` of the item |
| C4: a boundary, an element, a relation | `Group`, `c4.boundary` (dashed) or `c4.deployment`, its heading start-aligned `Text` items; `Node`, `c4.element`, a person, a box or a cylinder, external ones dashed; `Edge`, `c4.rel`, its label and technology `Text` items on a patch; the kind of each in `Class` |
| block: a composite, a block | `Group`, `block.composite`; `Node`, `block.block`, its shape Pele's or `block-arrow` |
| wardley: a component, its inertia, a pipeline, a trend | `Node`, `wardley.component`, its caption a `Text` item, `wardley.label`; `Mark`, `wardley.inertia`; `Mark`, `wardley.pipeline` and its dashed `wardley.pipeline-line`; `Edge`, `wardley.trend`, and `Mark`, `wardley.trend-target` |
| railroad: a rule's track, a box | `Mark`, `railroad.track`, of lines and arcs; `Node`, `railroad.terminal` (rounded to half its height), `railroad.nonterminal` or `railroad.special`, the marks about a special one `Text` items, `railroad.special-mark` |
| ishikawa: the spine, a category | `Edge`, `ishikawa.spine`; `Edge`, `ishikawa.bones`, a category's bone and twigs as one path; `Node`, `ishikawa.category`; each cause a `Text` item, `ishikawa.cause`; drawn down in a narrow host, an outline joined by `ishikawa.link` lines |
| cynefin: a domain, the boundaries | `Mark`, `cynefin.region`, filled `Series(i)` at 0.14, Confusion's an ellipse, dashed, over a `cynefin.backdrop` of the background; `Mark`, `cynefin.boundary`, dashed wavy curves, and `cynefin.cliff`; `Group`, `cynefin.domain`, its name its label, its items `Node`s, `cynefin.item`; `Edge`, `cynefin.transition`, one quadratic curve |
| event modeling: a lane, a frame, a relation | `Group`, `eventmodeling.lane`; a `Node` for each frame, its role its kind (`eventmodeling.command`, `eventmodeling.event`…), tinted `Series(i)` at 0.22 over a `Shade` box; `Edge`, `eventmodeling.relation` |
| tree view: an entry, a guide | `Node`, `treeview.directory` or `treeview.file`, its label starting at its point, a highlight a rounded part in `Accent` at 0.14; `Mark`, `treeview.guide`, one path a directory |
| info | `Text` items, `info.title` and `info.version`; it draws "limu" and the version the build records (`info.Version`) where Pele draws its own |

The scene needed two things for them that flowcharts do not: labels aligned at their start
or end (`Align`, which C4, journey, kanban, mindmap, packet, railroad, requirement, tree
view, treemap and wardley diagrams use too) and labels turned (`Rotate`, for the axes of
charts). Markers of other kinds — sequence diagrams' open and asynchronous arrows, class
diagrams' diamonds, entity relationship diagrams' crow's feet — are added with the diagrams
that use them.

### Markers

The scene keeps a table of the types of marker it knows (`scene/marker.go`): for each, how
far an edge stops short of its end to leave room for it, and its outline, as parts drawn
from its tip and its direction. Pele's flowchart markers are in the table itself; a diagram
with markers of its own adds them from an init function, in a file of its own
(`scene/markers_er.go`), under names with the diagram's prefix (`"er.zero_or_more"`).

A marker's parts are painted in the color of the edge's line, whatever its style's fill:
`FillOnly` fills a part in it, `FillAndStroke` strokes it too, `StrokeOnly` strokes it only,
and `Hollow` fills it with the background and strokes it, so that the line beneath does not
show through. A marker is drawn solid on a dashed edge, as Pele draws it, with the edge's
caps, and with its corners turned as the edge's are unless its type has a join of its own
(`MarkerType.Join`), as Pele gives class diagrams' triangles and diamonds round ones.

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

A style also says how a stroked line ends and turns its corners: its `Cap` (butt, round or
square) and its `Join` (miter, round or bevel), as SVG's stroke-linecap and stroke-linejoin
do. They change how a line looks and not where anything is, so they are style, not layout.
Left unset all the way up, a line ends flat and its corners are mitred, as in SVG. Each
diagram sets them in its own style where Pele does: the edges of every diagram end round,
as Pele draws them in a group with round caps, and a few shapes turn their corners round,
as a gantt chart's milestone. An author's `stroke-linecap` and `stroke-linejoin` in `style`
and `classDef` lines are read, as Pele passes them on to the browser. The svg package
writes them where they are set; the mygoui package strokes with MyGo's round ends and
corners when both are round, and otherwise fills the outline SVG would stroke.

## Porting from Pele

Parsing, layout and the geometry of shapes are ported from Pele, function for function, so
that limu lays out what Pele does to the last bit when both measure text with the same
table of Arial's advances (`text.Arial`, Pele's `metricsMeasurer`). Where JavaScript's
meaning differs from Go's, the port keeps JavaScript's: strings counted in UTF-16 code
units, `\s` and `trim` with Unicode's spaces, `Math.round`'s halves and V8's `Math.hypot`
(`internal/jsmath`), and products kept apart from the sums they are in, as Go on arm64
would fuse them into one multiply-add that JavaScript never does.

The diagrams' trigonometry is V8's too. `jsmath.Sin`, `Cos`, `Atan2`, `Acos`, `Asin` and
`Log10` are ported from the fdlibm functions in V8's `src/base/ieee754.cc`, with their
products fused where clang fuses them when it builds Node for arm64; Go's `math` package
gives other last bits for a good share of inputs, and so does a V8 built without fused
multiply-adds, as on x86-64. V8's `Math.pow` calls the C library's `pow`, which is not the
same on every platform, so limu does not copy it. It needs only the powers Pele takes:
`Pow10` gives whole powers of ten, `x ** 2` is `x*x`, as V8 works it out, and Sankey's six
pulls of 0.99 to a power are written out as V8 gives them, as Go's `math.Pow` multiplies
and gives other last bits.

Where Mermaid hands a diagram to a library, Pele often lays it out in a way of its own, and
limu ports that: architecture diagrams on a grid, where Mermaid uses a force-directed
solver, and C4 and block diagrams with their own placing and routing. Cynefin diagrams'
wavy boundaries are seeded: from the diagram's `seed` setting, or else from a hash of
`pele`, the id prefix Pele renders with by default, since limu writes no ids.

What the diagrams share beyond layout and drawing is in `diagram/internal/common`, as Pele
keeps it in `diagrams/common`: the padding about a diagram, fitting it to the host's width,
reading generic types, and the title above it.

`scripts/oracle.sh` runs Pele's TypeScript sources with Node, with no install, over the
cases in `internal/oracle/cases`, and writes what Pele makes of them as fixtures in the
packages' `testdata/pele`. The tests compare with those, so they run without Node.

Most diagrams' tests compare what Pele's SVG draws, read back as one line per thing drawn,
with lines written from limu's scene. Pele writes the insides of a node or a group about
the group's origin, where the scene holds every place on the page, and a number moved
back from one to the other can round the other way at two decimals. A suite that needs it
sets `limuExactNumbers` while Pele draws, and the oracle's hooks then have Pele write its
numbers in full, for the suite to move as the scene's are and round as Pele does.

The port follows Pele 1.6.0, commit `44da152`. The fixtures record the version and the
commit they were written from.

## Roadmap

| | | |
|---|---|---|
| M0 | the stages and their types, text measured and laid out, styles, the svg backend, the oracle | done |
| M1 | flowcharts from text to pixels: the parser, the layered and compound layouts, the shapes; a backend for MyGo's native UI; a look in the Platinum style in mygo-platinum | done |
| M2 | the diagrams on the layered layout: state, class, entity relationship, requirement, use case, agent flow, swimlane | done |
| M3 | sequence, gantt, timeline, journey, git graph, packet, kanban | done |
| M4 | the charts: pie, xy, quadrant, radar, sankey, treemap, venn | done |
| M5 | the rest: mindmap, architecture, C4, block, wardley, railroad, ishikawa, cynefin, event modeling, tree view, info | done |
