# Design

## Stages

```
text ──parse──▶ model ──layout──▶ scene.Scene ──backend──▶ SVG, a GPU, a grid of pixels
                         ▲                         ▲
                 layout.Options             style.Look
         (measurer, font size, width,  (theme, overrides, and the
          padding, snap, routing:       backend's own drawing: how they look)
          where things are)
```

What changes where things are goes to layout, and a change of it lays the diagram out
again: the measurer and its fonts, the size of text, the width the host has and whether a
diagram may turn to fit it, the padding about a diagram (`Padding`, nil for Pele's 8, which
may be set to 0; `Options.Pad` reads it), a grid to snap to, how edges are routed, and for
a gantt chart the present and the time zone its dates are read in. Everything else is
style, settled only when a scene is drawn, and a change of it only redraws.

`limu.Parse` reads a diagram's text with what Pele's `parse` takes, the limits the text is
held to and configuration under the diagram's own, and gives a `limu.Diagram`, whose
`Layout` method takes a `layout.Options`. Laying a diagram out changes neither its model
nor its configuration (a gantt chart is scheduled in a copy), so one `Diagram` may be laid
out many times, and from many goroutines at once. `limu.Layout` does the two in one call,
and `limu.Types` and `limu.Supports` say which diagrams limu reads.

The scene `limu.Layout` and `Diagram.Layout` give holds only finite numbers, and is snapped
to `Options.Snap`. Where Pele would draw NaN, as for `sankey\na,b,5e-324`, they give a
`diagram.Error` of code `diagram.Limit` instead. A diagram package's own `Layout` neither
snaps its scene nor checks it.

## The scene

A `scene.Scene` is a list of `scene.Item`s in the order they are drawn. An item says what
it is — its `Kind`, its `Role` (`"flowchart.node"`) and, if it has an outline, its `Shape`
(`"diam"`) — and where it is: its `Box`, and for an edge its `Route`, the corners of the
line it takes. What `Box` holds depends on the kind: for a node or a group, the box of its
shape; for an edge, the box its path lies in, the control points of its curves included; for
a mark, the box of what it draws; for a `Text` item, the box the diagram sets its label in,
as an edge label's patch, or nothing where it gives the label none. `Item.LabelBox` is where
the label is drawn, laid about its point as `Align` and `Rotate` say.

Every edge has a route. A self-loop, which no layout routes, has the corners its curve is
drawn about (`Path.Corners`), so that a backend with a look of its own has a line to draw.

An item's `Class` holds the classes the diagram's author gave it, and its `Tags` what the
diagram says it is beyond its role, which Pele writes among the classes of what it draws: a
C4 element's, relation's or deployment node's kind and `external`, a gantt task's state,
the branch a gantt section or a mindmap's node or edge is in (`section-0`…), a wardley
component's sourcing, `reset` on an event modeling frame, and `clickable` on what has a
click. The svg backend writes the tags and then the classes as an element's classes.

It also carries what a plain backend needs to draw it:

- `Parts`: its outline, as paths and boxes, each with how it is painted in the item's style
  (below). A part is a path when its `Path` is set, and otherwise the box `Box`; `Rounded`
  applies only to a box. A box whose corners take the theme's radius is a box with
  `Rounded` set, not a path, so the radius stays a matter of style.
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
| `Shade` | the theme's background; where the author set a fill, the background at 60% opacity, laid over the parts drawn before it | none | a stripe of an entity's table |

Paths have four kinds of step — move, line, quadratic and cubic curves — and close. Arcs
become cubic curves as they are added, one for each quarter turn or less, since MyGo's paths
have no arcs; an arc whose radii or ends place no ellipse becomes a line to its end.

`Scene.Snap` moves everything onto a grid: each side of a box, and each point of a path of
straight lines. A path with curves is moved as a whole, by what puts its first point on the
grid, so that its curves keep their shape.

An item has at most one label. A thing of a diagram with more — a sequence diagram's
frame, with its kind and its condition — is several items, the shape first.

A backend with a look of its own draws an item from its role, shape, box and route, and
ignores its parts.

### What the other diagrams need of it

Before the scene was settled, the drawing of two diagrams unlike flowcharts was checked
against it; the charts and the rest of Pele's diagrams were drawn in it since.

A role is the name of the diagram's package, a dot, and the part: `gitgraph.commit`,
`quadrant.point`, `treeview.file`. Each diagram package exports its roles as constants. A
swimlane diagram's items keep the roles of a flowchart's, so that an override for
`flowchart.node` reaches them too, with `swimlane.lane` for its lanes. Flowcharts and
state, class, use case and agent flow diagrams name an edge's label `*.edge-label`, and
class and entity relationship diagrams draw the dividers of a box as items of their own,
`class.divider` and `er.divider`, which take only the stroke color of what the author set.

The other diagrams take these items:

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
| mindmap: a node, a branch | `Node`, `mindmap.node` or `mindmap.root`, its shape from `diagram/internal/draw`, one without a shape a stadium filled `Series(i)` at 0.22, or no parts for a leaf; `Edge`, `mindmap.edge`, in `Series(i)`; a branch has no group, and each of its items the tag `section-N` |
| architecture: a group, a service, a junction | `Group`, `architecture.group`, tinted `SurfaceAlt` at 0.5; `Node`, `architecture.service`, a rounded square with its title below; `Node`, `architecture.junction`, a dot |
| an icon Pele draws itself (architecture, tree view) | `Mark`, `architecture.icon` or `treeview.icon`, after the item it is of, read from Pele's path data with `scene.ParsePath` and stroked round, since a part has no stroke of its own; an icon the host draws is a `Picture` of the item |
| C4: a boundary, an element, a relation | `Group`, `c4.boundary` (dashed) or `c4.deployment`, its heading start-aligned `Text` items; `Node`, `c4.element`, a person, a box or a cylinder, external ones dashed; `Edge`, `c4.rel`, its label and technology `Text` items on a patch; the kind of each in `Tags` |
| block: a composite, a block | `Group`, `block.composite`; `Node`, `block.block`, its shape Pele's or `block-arrow` |
| wardley: a component, its inertia, a pipeline, a trend | `Node`, `wardley.component`, its caption a `Text` item, `wardley.label`; `Mark`, `wardley.inertia`; `Group`, `wardley.pipeline`, and its dashed `Mark`, `wardley.pipeline-line`; `Edge`, `wardley.trend`, and `Mark`, `wardley.trend-target` |
| railroad: a rule's track, a box | `Mark`, `railroad.track`, of lines and arcs; `Node`, `railroad.terminal` (rounded to half its height), `railroad.nonterminal` or `railroad.special`, the marks about a special one `Text` items, `railroad.special-mark` |
| ishikawa: the spine, a category | `Edge`, `ishikawa.spine`; `Edge`, `ishikawa.bones`, a category's bone and twigs as one path; `Node`, `ishikawa.category`; each cause a `Text` item, `ishikawa.cause`; drawn down in a narrow host, an outline joined by `ishikawa.link` lines |
| cynefin: a domain, the boundaries | `Mark`, `cynefin.region`, filled `Series(i)` at 0.14, Confusion's an ellipse, dashed, over a `cynefin.backdrop` of the background; `Mark`, `cynefin.boundary`, dashed wavy curves, and `cynefin.cliff`; `Group`, `cynefin.domain`, its name its label, its items `Node`s, `cynefin.item`; `Edge`, `cynefin.transition`, one quadratic curve |
| event modeling: a lane, a frame, a relation | `Group`, `eventmodeling.lane`; a `Node` for each frame, its role its kind (`eventmodeling.command`, `eventmodeling.event`…), tinted `Series(i)` at 0.22 over a `Shade` box; `Edge`, `eventmodeling.relation` |
| tree view: an entry, a guide | `Node`, `treeview.directory` or `treeview.file`, its label starting at its point, a highlight a rounded part in `Accent` at 0.14; `Mark`, `treeview.guide`, one path a directory |
| info | `Text` items, `info.title` and `info.version`; it draws "limu" and the version the build records (`info.Version`) where Pele draws its own |

The scene needed two things for them that flowcharts do not: labels aligned at their start
or end (`Align`, which C4, class, entity relationship, event modeling, journey, kanban,
mindmap, packet, railroad, requirement, sankey, tree view, treemap, wardley and XY diagrams
use) and labels turned (`Rotate`: the axis labels of XY and quadrant charts and of wardley
diagrams at −90°, a git graph's commit labels at ±45°, and a swimlane diagram's lanes when their
names run sideways). Markers of other kinds — sequence diagrams' open and asynchronous
arrows, class diagrams' diamonds, entity relationship diagrams' crow's feet — are added with
the diagrams that use them.

### Markers

The scene keeps a table of the types of marker it knows (`scene/marker.go`): for each, how
far an edge stops short of its end to leave room for it, and its outline, as parts drawn
from its tip and its direction. Pele's flowchart markers are in the table itself; a diagram
with markers of its own adds them from an init function, in a file of its own
(`scene/markers_er.go`), under names with the diagram's prefix (`"er.zero_or_more"`).
An end with no marker has the empty type, whatever name the diagram's model gives it:
`"none"`, and `"arrow_open"` for an open line, are not types of marker.

A marker's parts are painted in the color of the edge's line, whatever its style's fill,
filled without the stroke's opacity and stroked with it (`style.Resolved.Marker`), as Pele
fills them in the color of the line alone: `FillOnly` fills a part in it, `FillAndStroke`
strokes it too, `StrokeOnly` strokes it only, and `Hollow` fills it with the background and
strokes it, so that the line beneath does not show through. A marker is drawn solid on a
dashed edge, as Pele draws it, with the edge's caps, and with its corners turned as the
edge's are unless its type has a join of its own (`MarkerType.Join`), as Pele gives class
diagrams' triangles and diamonds round ones.

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
   tools of its own. The svg package's `Options.Draw` writes markup, and may write the
   default's pieces with the `Writer`'s `Item`, `Parts`, `Ends`, `Picture` and `Label`, and
   colors at their opacities with `PaintAttr` and `Opacity`; it is given a `style.Style`,
   its paints not settled, so that the theme's colors are written as CSS variables. The
   limumygo package's `Options.Draw` has the same pieces on its `Painter`, and is given a
   `style.Resolved`, as MyGo draws with colors. Icons and images are hooks of the
   backends: the svg package's `IconMarkup`, and limumygo's `DrawIcon` and `DrawImage`.

A part's style is cascaded, from under to over: the theme's measures, the diagram's own
style for the part, the overrides for its role, and what its author set in the diagram's
text. The opacities of the fill and the stroke (`FillOpacity`, `StrokeOpacity`) cascade
apart from the paints, as CSS cascades fill-opacity apart from fill, so that a fill the
author sets on a part the diagram draws at an opacity is drawn at that opacity.

A style also says how a stroked line ends and turns its corners: its `Cap` (butt, round or
square) and its `Join` (miter, round or bevel), as SVG's stroke-linecap and stroke-linejoin
do. They change how a line looks and not where anything is, so they are style, not layout.
Left unset all the way up, a line ends flat and its corners are mitred, as in SVG. Each
diagram sets them in its own style where Pele does: the edges of most diagrams end round,
as Pele draws them in a group with round caps, and a few shapes turn their corners round,
as a gantt chart's milestone. The edges of mindmaps, journeys, timelines, wardley,
ishikawa and event modeling diagrams, and a cynefin diagram's transitions, end flat, as
Pele draws them with no cap. An author's `stroke-linecap` and `stroke-linejoin` in `style`
and `classDef` lines are read, as Pele passes them on to the browser. The svg package
writes them where they are set; the limumygo package strokes with MyGo's round ends and
corners when both are round, and otherwise fills the outline SVG would stroke.

The limumygo package draws what the svg package writes, but not in every way as a browser
draws SVG. An item's opacity is laid on each color it is drawn in, as MyGo has no group to
draw at an opacity, so that where an item's parts overlap, the one below shows through. A
scene is zoomed and panned with `Options.Scale` and the point `Paint` draws it at, not under
a transform of the painter. A radius too large for a box is the smaller of half its width
and half its height, both ways, where SVG makes a narrow box's corners elliptical; on macOS,
MyGo curves a box's corners as Apple does, not in arcs. And MyGo rounds a box's edges to
whole device pixels, where a path is drawn where it falls.

## Porting from Pele

Parsing, layout and the geometry of shapes are ported from Pele, function for function, so
that limu lays out what Pele does to the last bit when both measure text with the same
table of Arial's advances (`text.Arial`, Pele's `metricsMeasurer`). Where JavaScript's
meaning differs from Go's, the port keeps JavaScript's: strings counted in UTF-16 code
units, `\s`, `trim` and `toLowerCase` as JavaScript has them (`internal/chars`; a capital
sigma that ends a word lowers to a final sigma, by Unicode's Final_Sigma condition), `Math.round`'s halves
and V8's `Math.hypot` (`internal/jsmath`), JSON read as V8's `JSON.parse` reads it, with an
object's keys in JavaScript's order (`internal/jsjson`, for sequence and use case
diagrams), and products kept apart from the sums they are in. Go fuses a product and the
sum it is in into one multiply-add, rounded once, on arm64 and on amd64 at `GOAMD64=v3`,
where JavaScript never does; a product that must stay apart is written `float64(x*y)`, as
in the sankey, gantt, state and use case diagrams and the cloud shape of
`diagram/internal/draw`. The arithmetic of `scene` that has no counterpart in Pele, as the
cubic curves of an arc (`scene/path.go`) and `Path.Scaled`, is left free to fuse.

Pele is handed a JavaScript string, which the browser decoded from UTF-8; a Go string may
hold bytes that are not UTF-8. Before anything reads a diagram's text, those bytes are
replaced as a browser's decoder replaces them, each maximal subpart of an ill-formed
sequence with one U+FFFD (`chars.Decode`), so that limu lays out the U+FFFD Pele would be
given. A lone surrogate, which a JavaScript string can hold, is written in a Go string as
the three bytes of its WTF-8 form, and so is read as three U+FFFD. The tests of the `limu`
package check this for every type of diagram, and the text package's fixture checks that
U+FFFD is measured as Pele measures it. The oracle's suites leave out the inputs that hold
a lone surrogate, as a Go string cannot hold one to compare with.

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

### What the diagrams share

What more than one diagram uses is under `diagram/internal`, where hosts cannot reach it, as
Pele keeps it under `src/layout`, `src/svg` and `src/diagrams/common`:

- `graphs`: the layered, compound and swimlane layouts, Pele's `src/layout`. The package is
  `graphs` and not `graph`, as the diagrams on the layered layout keep Pele's local
  variable `graph`, which would hide it.
- `draw`: the geometry of shapes and edges, Pele's `svg/shapes.ts`, `svg/edges.ts` and
  `svg/cluster.ts`.
- `css`: what the author sets with `classDef` and `style` lines, read as Pele's
  `resolveStyle` reads it. It belongs to the diagrams and not to `style`, as the weight, the
  slant and the size of text the author sets are inputs of layout.
- `common`, as Pele's `diagrams/common`: fitting a diagram to the width the host has, and
  the curve its routing asks for (`RoutingCurve`); labels, plain, fitted, cut to fit and
  taken literally, and a diagram's title; generic types and class lists; the icons a host
  can draw and the glyphs drawn for them; the lexer and reader of the Langium grammars,
  with their shared tokens and `AccDescr`; the scanning helpers of the hand-written Jison
  lexers, `WordAt` among them; the indented outlines of kanban boards and mindmaps; and the
  errors of a diagram's text. The padding about a diagram is `layout.Options.Padding`.

`layout.Options.Routing` reaches the diagrams on the layered layout: flowcharts, swimlane,
agent flow, state, class, entity relationship, requirement and use case diagrams. The others
route their edges as Pele does.

### Where limu differs

A few differences from Pele are kept on purpose, or known and left:

- Pele writes on text the author's `font-family`, `letter-spacing`, `word-spacing` and
  `text-decoration`, and a weight or a size it did not measure the text in: both in
  treemaps, the size in venn diagrams. limu reads only the weight, the slant and the size, and only where layout
  measures with them, as text drawn otherwise than it was measured would not fit the room
  layout gave it.
- A sankey diagram's label and its value, which Pele writes as one text with the value in
  a span of another color, are two items, since a label is of one color: the label starts
  where the whole starts and the value ends where it ends, placed by the measure of the
  whole. With a measurer that kerns, the gap between them is not quite Pele's.
- A link's host outside ASCII is lowered and written in Punycode without the whole of
  UTS #46 (`internal/url`): its other mappings, as of ligatures and combining marks, are not
  made, so a few such hosts are written otherwise than Node writes them.

### The oracle

`scripts/oracle.sh` runs Pele's TypeScript sources with Node, with no install, and writes
what Pele makes of the cases of each suite in `internal/oracle/suites` as fixtures in the
packages' `testdata/pele`. The tests compare with those, so they run without Node. Most
suites hold or build their cases themselves, from Pele's tests and documents
(`lib/corpus.mjs`) and mutations of them (`lib/mutate.mjs`); `internal/oracle/cases` holds
the cases of the first few. What the suites share is in `internal/oracle/lib`, as how
values and errors are written (`value.mjs`) and how Pele's SVG is read back, with the caps
and joins a browser cascades (`svg.mjs`). A suite named `x` writes `x.json` in the package
`x`, and `x-y` writes `y.json` there; `run.mjs` lists the suites named otherwise. The oracle
runs only on arm64, as the fixtures of the trigonometry hold the last bits V8 gives there.
It exits with code 2 for a suite it does not know or a path that is not a checkout of Pele,
and leaves a fixture whose content has not changed as it is.

Most diagrams' tests compare what Pele's SVG draws, read back as one line per thing drawn,
with lines written from limu's scene. A fixture of drawings, `render.json`, holds the lines
of its first cases (`fixture.FullCases`, the same as `FULL` in `run.mjs`), and for each case
after them a digest of its lines; a test compares the digest of the lines it writes. A test
that reads some of Pele's lines through limu's own code first, as the author's style or the
arcs of a path, gives that reading to `Drawings.Diff`; its suite keeps in full, beside the
digest, each line the reading changes, and the reading must change no other line.
`./scripts/oracle.sh --full` also writes the lines of every case beside the fixture, in a
file git ignores, to find which line differs from a digest; the fixture is still what the
tests compare with. The render suites of pie charts, treemaps and flowcharts hold cases
with a padding of 0, 2.5 and 20.

Pele writes the insides of a node or a group about the group's origin, where the scene
holds every place on the page, and a number moved back from one to the other can round the
other way at two decimals. A suite that needs it sets `limuExactNumbers` while Pele draws,
and the oracle's hooks then have Pele write its numbers in full, for the suite to move as
the scene's are and round as Pele does. What the oracle patches into Pele's sources is
listed in a table at the head of `internal/oracle/hooks.mjs`, and each patch must find what
it replaces exactly as many times as it says, or the oracle fails. The patches write a
label Pele draws as `<limu-label x y align attrs data n/>` and a marker as
`<limu-marker type x y dx dy color n/>`, right before what Pele writes of it, `n` being how
many UTF-16 code units that writing, which follows, takes; `internal/oracle/lib/limu.mjs` writes and
reads them.

The outlines of `diagram/internal/draw` are compared with Pele's number for number where
Pele draws with lines and curves; a circle, an ellipse, a box with corners of its own and a
path with an arc, which a `scene.Path` draws in cubic curves that only come near, are
compared as figures, within a tolerance.

The port follows Pele 1.6.0, commit `44da152`. The fixtures record the version and the
commit they were written from, and the architecture and the version of Node that wrote
them. The tests read them with `fixture.Load`, which fails on a fixture written from
another version or commit of Pele, or on one with an empty list at its top level, as a
suite that found no cases would write. The numbers JSON has no words for, as NaN and
negative zero, are written as the oracle writes them with `fixture.Number`, and read with
`fixture.Float`.

## Roadmap

| | | |
|---|---|---|
| M0 | the stages and their types, text measured and laid out, styles, the svg backend, the oracle | done |
| M1 | flowcharts from text to pixels: the parser, the layered and compound layouts, the shapes; a backend for MyGo's native UI; a look in the Platinum style in mygo-platinum | done |
| M2 | the diagrams on the layered layout: state, class, entity relationship, requirement, use case, agent flow, swimlane | done |
| M3 | sequence, gantt, timeline, journey, git graph, packet, kanban | done |
| M4 | the charts: pie, xy, quadrant, radar, sankey, treemap, venn | done |
| M5 | the rest: mindmap, architecture, C4, block, wardley, railroad, ishikawa, cynefin, event modeling, tree view, info | done |
