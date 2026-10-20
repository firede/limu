# Design

This document is for people who work on limu. The website's
[Architecture](https://limu.firede.com/architecture) page is the short version for people
who use it.

## Stages

```
text ──parse──▶ model ──layout──▶ scene.Scene ──backend──▶ SVG, a GPU, a grid of pixels
                         ▲                         ▲
                 layout.Options             style.Look
         (measurer, font size, width,  (theme, overrides, and the
          padding, snap, routing:       backend's own drawing: how they look)
          where things are)
```

Anything that changes where things are belongs to layout, and changing it lays the diagram
out again. That covers the measurer and its fonts, the text size, the width the host has
and whether a diagram may turn to fit it, the padding around a diagram, a grid to snap to,
how edges are routed, and the present and time zone of a gantt chart. `Padding` is nil for
Pele's 8 and may be set to 0, and `Options.Pad` reads it. Everything else is style, which
is settled only when a scene is drawn, so changing it only redraws.

`limu.Parse` takes what Pele's `parse` takes, the limits for the text and configuration
beneath the diagram's own. It returns a `limu.Diagram`, whose `Layout` method takes a
`layout.Options`. Laying out a diagram changes neither its model nor its configuration (a
gantt chart is scheduled in a copy), so one `Diagram` can be laid out many times, from
many goroutines at once. `limu.Layout` does both steps in one call, and `limu.Types` and
`limu.Supports` say which diagrams limu reads.

The scenes that `limu.Layout` and `Diagram.Layout` return hold only finite numbers and are
snapped to `Options.Snap`. Where Pele would draw NaN, as for `sankey\na,b,5e-324`, they
return a `diagram.Error` with code `diagram.Limit` instead. A diagram package's own
`Layout` neither snaps its scene nor checks it.

| Package | What it does |
|---|---|
| `limu` | Parses a diagram and lays it out. |
| `diagram` | Reads the front matter, directives and type every diagram shares, as Pele's `preprocess.ts`, `detect.ts` and `core.ts` do. Each type has a package under it that parses and lays out that type, and what they share is under `diagram/internal`. |
| `text` | Measures text with a `Measurer` and lays out labels with line breaks, wrapping, emphasis, icons and entities. `text.Arial` measures from a table of Arial's advances, as Pele's `metricsMeasurer` does. |
| `scene` | Describes a laid-out diagram. |
| `style` | Holds themes and overrides for each role. |
| `layout` | Holds the options that change where things go. |
| `svg` | Renders a scene as SVG, with theme colors as CSS variables. |
| `limumygo` | Draws a scene with MyGo and finds the item under the pointer. It's a module of its own, so the rest of limu needs only the standard library. `limumygo/mermaid` adds Mermaid to MyGo's code editor. |

## The scene

A `scene.Scene` is a list of `scene.Item`s in the order they are drawn. An item says what
it is with its `Kind`, its `Role` (`"flowchart.node"`) and, if it has an outline, its
`Shape` (`"diam"`). It says where it is with its `Box` and, for an edge, its `Route`, the
corners of the line the edge takes.

What `Box` holds depends on the kind. For a node or a group, it's the box of the shape. For
an edge, it's the box the path lies in, including the control points of its curves. For a
mark, it's the box of what it draws. For a `Text` item, it's the box the diagram sets the
label in, such as an edge label's patch, or nothing where the diagram gives the label none.
`Item.LabelBox` is where the label is drawn, laid out around its point as `Align` and
`Rotate` say.

Every edge has a route. A self-loop, which no layout routes, gets the corners its curve is
drawn around (`Path.Corners`), so that a backend with a look of its own has a line to draw.

An item's `Class` holds the classes the diagram's author gave it. Its `Tags` hold what the
diagram says about it beyond its role, which Pele writes among the classes of what it
draws. Examples are a C4 element's kind and `external`, a gantt task's state, the branch a
gantt section or mindmap node is in (`section-0`…), a wardley component's sourcing, `reset`
on an event modeling frame, and `clickable` on anything with a click. The svg backend writes
the tags and then the classes as an element's classes.

An item also carries what a plain backend needs to draw it.

- `Parts` is its outline, as paths and boxes, each with how it's painted in the item's
  style. A part is a path when its `Path` is set and otherwise the box `Box`. `Rounded`
  applies only to a box. A box whose corners take the theme's radius is a box with
  `Rounded` set, not a path, so the radius stays a matter of style.
- `Ends` are the markers at the ends of an edge, with where their tips are and which way
  they point. `Marker.Outline` gives the parts Pele draws them with.
- `Label` is text laid out in lines and spans, already measured, with the point it's drawn
  around, its alignment and its rotation.
- `Style` holds the diagram's own colors for the item, as theme tokens, and `Author` holds
  what the diagram's author set with `classDef` and `style` lines.

A part's `Painting` says which of the style's colors it takes, so a part keeps its role
when a theme changes them.

| Painting | Fill | Stroke | For |
|---|---|---|---|
| `FillAndStroke` | the style's fill | the style's stroke | most outlines |
| `StrokeOnly` | none | the style's stroke | an edge, the lid of a cylinder |
| `FillOnly` | the style's fill | none | the body of a datastore |
| `Solid` | the theme's line | the style's stroke | the dot of a final state, the bar of a fork |
| `Hollow` | the theme's background | the style's stroke | the ring of a final state, which hides what is under it |
| `Shade` | the theme's background, or that background at 60% opacity over the parts below when the author set a fill | none | a stripe of an entity's table |

Paths have four kinds of step (move, line, quadratic curve and cubic curve) and can close.
MyGo's paths have no arcs, so an arc becomes cubic curves as it's added, one for each
quarter turn or less. An arc whose radii or ends describe no ellipse becomes a line to its
end.

`Scene.Snap` moves everything onto a grid, which means each side of a box and each point a
path passes through. In the path of an edge, a node, a group or a text item, each control
point of a curve moves with the point it leans on, so a rounded corner still turns from one
line into the next, and the lines meet the boxes and markers they lead to. A path of curves
alone, such as a circle, and the path of a mark, such as an icon, move as a whole, by
whatever puts the first point on the grid, so they keep their shape.

An item has at most one label. Something with more than one, such as a sequence diagram's
frame with its kind and its condition, is several items, with the shape first.

A backend with a look of its own draws an item from its role, shape, box and route, and
ignores its parts.

### Roles and items

A role is the diagram's package name, a dot, and the part, such as `gitgraph.commit`,
`quadrant.point` or `treeview.file`. Each diagram package exports its roles as constants.

A swimlane diagram's items keep a flowchart's roles, so an override for `flowchart.node`
reaches them too, and its lanes are `swimlane.lane`. Flowcharts and state, class, use case
and agent flow diagrams name an edge's label `*.edge-label`. Class and entity relationship
diagrams draw the dividers of a box as items of their own, `class.divider` and
`er.divider`, which take only the stroke color the author set.

The other diagrams draw these items.

| Pele draws | Items |
|---|---|
| sequence: a participant, at the head and again at the foot of its lifeline | `Node`, `sequence.actor`, with its kind as its shape (`rect` for a participant). A figure's head and limbs, a database's lid and a queue's end are parts, the lines `StrokeOnly`. |
| sequence: a lifeline | `Mark`, `sequence.lifeline`, one solid path |
| sequence: a message | `Edge`, `sequence.message`, a straight or looped path with markers of Mermaid's arrow kinds. The circles of a central connection are `FillOnly` parts. Its label is a `Text` item, `sequence.message-label`. |
| sequence: an activation | `Mark`, `sequence.activation`, a box |
| sequence: a note | `Node`, `sequence.note` |
| sequence: a loop, alt or opt frame | `Group`, `sequence.frame`. Each dashed divider is a `Mark`, `sequence.divider`, since a part takes its dash from its item's style. The tab and kind are a `Text` item, `sequence.frame-kind`, with the tab as a part. Each condition is a `Text` item, `sequence.condition`. |
| sequence: a rect block, a box of participants | `Group`, `sequence.rect` or `sequence.box`, filled with the theme's alternate surface at half its opacity unless the author gave a fill |
| sequence: an autonumber | `Node`, `sequence.number`, a rounded box with its label in the background color |
| sequence: a destroyed participant's cross | `Mark`, `sequence.destroy` |
| pie: a slice | `Mark`, `pie.slice`, a path of arcs filled with `style.Series(i)` |
| pie: a slice's percentage | `Text`, `pie.slice-label`, in the background color on the slice and the text color outside it |
| a chart's legend entry (pie, XY, radar) | `Node`, with its swatch as a part (a box with radius 2) and its label |
| XY: a series | `Mark`, `xychart.bars` (a box per bar) or `xychart.line` (its path, with its dots as `FillOnly` parts) |
| a chart's turned axis title (XY, quadrant) | `Text`, with `Rotate` at −90 |
| radar: the graticule, the spokes | `Mark`, `radar.graticule` (a ring per part) and `radar.axes` |
| radar: a curve | `Mark`, `radar.curve`, filled with `Series(i)` at 0.2, with round joins |
| quadrant: a quadrant, a point | `Node`, `quadrant.quadrant`. `Mark`, `quadrant.point`, a circle with its label as a `Text` item. |
| sankey: a node, a band | `Node`, `sankey.node`. `Mark`, `sankey.link`, filled at 0.4. |
| sankey: a label and its value in another color | two `Text` items, `sankey.label` and `sankey.value`, placed by the measure of the whole |
| treemap: a section, a leaf | `Group`, `treemap.section`, tinted at 0.12. `Node`, `treemap.leaf`. A value is a `Text` item. |
| venn: a set, a styled overlap | `Mark`, `venn.set`, a circle at 0.2. `Mark`, `venn.overlap`, an outline of arcs. |
| mindmap: a node, a branch | `Node`, `mindmap.node` or `mindmap.root`, with its shape from `diagram/internal/draw`. A node without a shape is a stadium filled with `Series(i)` at 0.22, or has no parts for a leaf. `Edge`, `mindmap.edge`, in `Series(i)`. A branch has no group, and each of its items has the tag `section-N`. |
| architecture: a group, a service, a junction | `Group`, `architecture.group`, tinted with `SurfaceAlt` at 0.5. `Node`, `architecture.service`, a rounded square with its title below. `Node`, `architecture.junction`, a dot. |
| an icon Pele draws itself (architecture, tree view) | `Mark`, `architecture.icon` or `treeview.icon`, after the item it belongs to. It's read from Pele's path data with `scene.ParsePath` and stroked round, since a part has no stroke of its own. An icon the host draws is the item's `Picture`. |
| C4: a boundary, an element, a relation | `Group`, `c4.boundary` (dashed) or `c4.deployment`, with its heading as start-aligned `Text` items. `Node`, `c4.element`, a person, a box or a cylinder, dashed when external. `Edge`, `c4.rel`, with its label and technology as `Text` items on a patch. Each one's kind is in `Tags`. |
| block: a composite, a block | `Group`, `block.composite`. `Node`, `block.block`, with Pele's shape or `block-arrow`. |
| wardley: a component, its inertia, a pipeline, a trend | `Node`, `wardley.component`, with its caption as a `Text` item, `wardley.label`. `Mark`, `wardley.inertia`. `Group`, `wardley.pipeline`, and its dashed `Mark`, `wardley.pipeline-line`. `Edge`, `wardley.trend`, and `Mark`, `wardley.trend-target`. |
| railroad: a rule's track, a box | `Mark`, `railroad.track`, of lines and arcs. `Node`, `railroad.terminal` (rounded to half its height), `railroad.nonterminal` or `railroad.special`. The marks around a special box are `Text` items, `railroad.special-mark`. |
| ishikawa: the spine, a category | `Edge`, `ishikawa.spine`. `Edge`, `ishikawa.bones`, a category's bone and twigs as one path. `Node`, `ishikawa.category`. Each cause is a `Text` item, `ishikawa.cause`. In a narrow host it's drawn downward as an outline joined by `ishikawa.link` lines. |
| cynefin: a domain, the boundaries | `Mark`, `cynefin.region`, filled with `Series(i)` at 0.14. Confusion's is a dashed ellipse over a `cynefin.backdrop` of the background. `Mark`, `cynefin.boundary`, dashed wavy curves, and `cynefin.cliff`. `Group`, `cynefin.domain`, with its name as its label and its items as `Node`s, `cynefin.item`. `Edge`, `cynefin.transition`, one quadratic curve. |
| event modeling: a lane, a frame, a relation | `Group`, `eventmodeling.lane`. A `Node` for each frame, with its kind as its role (`eventmodeling.command`, `eventmodeling.event`…), tinted with `Series(i)` at 0.22 over a `Shade` box. `Edge`, `eventmodeling.relation`. |
| tree view: an entry, a guide | `Node`, `treeview.directory` or `treeview.file`, with its label starting at its point and a highlight as a rounded part in `Accent` at 0.14. `Mark`, `treeview.guide`, one path per directory. |
| info | `Text` items, `info.title` and `info.version`. Where Pele draws its own name and version, limu draws "limu" and the version the build records (`info.Version`). |

These diagrams needed two things flowcharts don't. Labels can be aligned at their start or
end with `Align`, which C4, class, entity relationship, event modeling, journey, kanban,
mindmap, packet, railroad, requirement, sankey, tree view, treemap, wardley and XY diagrams
use. Labels can be turned with `Rotate`, as are the axis labels of XY and quadrant charts
and wardley diagrams (−90°), a git graph's commit labels (±45°), and a swimlane diagram's
lanes when their names run sideways.

### Markers

The scene keeps a table of the marker types it knows, in `scene/marker.go`. For each type,
it records how far an edge stops short of its end to leave room for the marker, and the
marker's outline as parts drawn from its tip and direction. Pele's flowchart markers are in
the table itself. A diagram with markers of its own adds them from an init function in a
file of its own (`scene/markers_er.go`), under names with the diagram's prefix
(`"er.zero_or_more"`). An end with no marker has the empty type, whatever the model calls
it, so `"none"` and `"arrow_open"` are not marker types.

A marker's parts are painted in the color of the edge's line, whatever the style's fill, as
Pele fills them. They are filled without the stroke's opacity and stroked with it
(`style.Resolved.Marker`). `FillOnly` fills a part in that color, `FillAndStroke` also
strokes it, `StrokeOnly` only strokes it, and `Hollow` fills it with the background and
strokes it, so the line beneath doesn't show through. A marker on a dashed edge is drawn
solid, as Pele draws it. It takes the edge's caps, and the edge's joins unless its type has
a join of its own (`MarkerType.Join`), as Pele gives class diagrams' triangles and diamonds
round ones.

## Style

Style has three levels, from the least a host does to the most.

1. A **theme** (`style.Theme`) holds the colors and measures every diagram takes its look
   from, by token. The tokens are `Bg`, `Fg`, `Muted`, `Line`, `Surface`, `SurfaceAlt`,
   `Border`, `Accent`, `Critical`, eight series, a corner radius and a stroke width. The svg
   package writes them as CSS variables such as `--limu-surface`, with the theme's colors as
   fallbacks. `style.Default()` is Pele's light theme, and `style.Kohu()` and
   `style.KohuDark()` are limu's own light and dark themes.
2. **Overrides** (`style.Overrides`) lay a `style.Style` for a role over the diagram's own
   style for its parts. Whatever a style leaves unset comes from the level beneath.
3. A backend's **own drawing** replaces how it draws an item, and can still call the
   default. It belongs to the backend, since each draws with its own tools. The svg
   package's `Options.Draw` writes markup. It can write the default's pieces with the
   `Writer`'s `Item`, `Parts`, `Ends`, `Picture` and `Label`, and colors at their opacities
   with `PaintAttr` and `Opacity`. It receives a `style.Style` whose paints aren't settled,
   so theme colors can be written as CSS variables. limumygo's `Options.Draw` has the same
   pieces on its `Painter` and receives a `style.Resolved`, since MyGo draws with colors.
   Icons and images are also hooks of the backends, the svg package's `IconMarkup` and
   limumygo's `DrawIcon` and `DrawImage`.

A part's style cascades from the theme's measures, through the diagram's own style for the
part and the overrides for its role, to what the author set in the diagram's text. The fill
and stroke opacities (`FillOpacity`, `StrokeOpacity`) cascade separately from the paints,
as CSS cascades fill-opacity separately from fill. So a fill the author sets on a part the
diagram draws at some opacity is drawn at that opacity.

A style also says how a stroked line ends and turns, with its `Cap` (butt, round or square)
and its `Join` (miter, round or bevel), as SVG's stroke-linecap and stroke-linejoin do. They
change how a line looks but not where anything is, so they're style. Left unset at every
level, a line ends flat and its corners are mitred, as in SVG. Each diagram sets them in its
own style where Pele does. Most diagrams' edges end round, since Pele draws them in a group
with round caps, and a few shapes such as a gantt milestone turn their corners round. The
edges of mindmaps, journeys, timelines, wardley, ishikawa and event modeling diagrams, and
a cynefin diagram's transitions, end flat, since Pele draws them with no cap. limu reads an
author's `stroke-linecap` and `stroke-linejoin` in `style` and `classDef` lines, as Pele
passes them on to the browser. The svg package writes them where they're set. limumygo
strokes with MyGo's round ends and corners when both are round, and otherwise fills the
outline SVG would stroke.

limumygo draws what the svg package writes, but not in every way as a browser draws SVG.

- MyGo has no group to draw at an opacity, so an item's opacity is applied to each color
  it's drawn in. Where an item's parts overlap, the one below shows through.
- A scene is zoomed and panned with `Options.Scale` and the point `Paint` draws it at, not
  with a transform on the painter.
- A radius too large for a box becomes half the box's smaller side, both ways, where SVG
  would make a narrow box's corners elliptical.
- On macOS, MyGo curves a box's corners as Apple does, not as arcs.
- MyGo rounds a box's edges to whole device pixels, while a path is drawn where it falls.

## Porting from Pele

Parsing, layout and the geometry of shapes are ported from Pele function for function.
Every diagram type Pele draws is ported. When both measure text with the same table of
Arial's advances (`text.Arial`, Pele's `metricsMeasurer`), limu lays out what Pele does to
the last bit.

Where JavaScript's meaning differs from Go's, the port keeps JavaScript's.

- Strings are counted in UTF-16 code units.
- `\s`, `trim` and `toLowerCase` work as in JavaScript (`internal/chars`). A capital sigma
  that ends a word lowers to a final sigma, by Unicode's Final_Sigma condition.
- `Math.round` rounds halves up, and `Math.hypot` is V8's (`internal/jsmath`).
- JSON is read as V8's `JSON.parse` reads it, with an object's keys in JavaScript's order
  (`internal/jsjson`, for sequence and use case diagrams).
- Products stay apart from the sums they're in. On arm64, and on amd64 at `GOAMD64=v3`, Go
  fuses a product and the sum it's in into one multiply-add, rounded once, and JavaScript
  never does. A product that must stay apart is written `float64(x*y)`, as in the sankey,
  gantt, state and use case diagrams and the cloud shape of `diagram/internal/draw`. The
  arithmetic in `scene` that has no counterpart in Pele, such as the cubic curves of an arc
  (`scene/path.go`) and `Path.Scaled`, is free to fuse.

Pele receives a JavaScript string, which the browser decoded from UTF-8, but a Go string can
hold bytes that aren't UTF-8. Before anything reads a diagram's text, `chars.Decode`
replaces those bytes as a browser's decoder does, each maximal subpart of an ill-formed
sequence with one U+FFFD, so limu lays out the U+FFFD Pele would receive. A lone surrogate,
which a JavaScript string can hold, appears in a Go string as the three bytes of its WTF-8
form, so it reads as three U+FFFD. The `limu` package's tests check this for every diagram
type, and the text package's fixture checks that U+FFFD is measured as Pele measures it. The
oracle's suites leave out inputs with a lone surrogate, since a Go string can't hold one to
compare with.

The diagrams' trigonometry is V8's too. `jsmath.Sin`, `Cos`, `Atan2`, `Acos`, `Asin` and
`Log10` are ported from the fdlibm functions in V8's `src/base/ieee754.cc`, with products
fused where clang fuses them when it builds Node for arm64. Go's `math` package gives
different last bits for a good share of inputs, and so does a V8 built without fused
multiply-adds, as on x86-64. V8's `Math.pow` calls the C library's `pow`, which differs from
platform to platform, so limu doesn't copy it. It needs only the powers Pele takes. `Pow10`
gives whole powers of ten, `x ** 2` is `x*x` as V8 computes it, and sankey's six pulls of
0.99 to a power are written out as V8 gives them.

Where Mermaid hands a diagram to a library, Pele often lays it out its own way, and limu
ports that. Architecture diagrams go on a grid where Mermaid uses a force-directed solver,
and C4 and block diagrams have their own placing and routing. Cynefin diagrams' wavy
boundaries are seeded from the diagram's `seed` setting, or else from a hash of `pele`, the
id prefix Pele renders with by default, since limu writes no ids.

### What the diagrams share

What more than one diagram uses lives under `diagram/internal`, where hosts can't reach it,
as Pele keeps it under `src/layout`, `src/svg` and `src/diagrams/common`.

- `graphs` has the layered, compound and swimlane layouts, Pele's `src/layout`. It's named
  `graphs`, not `graph`, because the diagrams on the layered layout keep Pele's local
  variable `graph`, which would hide the package.
- `draw` has the geometry of shapes and edges, Pele's `svg/shapes.ts`, `svg/edges.ts` and
  `svg/cluster.ts`.
- `css` reads what the author sets with `classDef` and `style` lines, as Pele's
  `resolveStyle` does. It belongs to the diagrams rather than to `style`, because the
  weight, slant and size of text the author sets are inputs to layout.
- `common` is Pele's `diagrams/common`. It fits a diagram to the host's width, picks the
  curve the routing asks for (`RoutingCurve`), and handles labels (plain, fitted, cut to fit
  and literal) and a diagram's title. It also has generic types and class lists, the icons a
  host can draw and the glyphs drawn for them, the lexer and reader of the Langium grammars
  with their shared tokens and `AccDescr`, the scanning helpers of the hand-written Jison
  lexers such as `WordAt`, the indented outlines of kanban boards and mindmaps, and the
  errors of a diagram's text.

`layout.Options.Routing` reaches the diagrams on the layered layout, which are flowcharts
and swimlane, agent flow, state, class, entity relationship, requirement and use case
diagrams. The others route their edges as Pele does.

### Where limu differs

A few differences from Pele are on purpose, or known and left alone.

- Pele writes the author's `font-family`, `letter-spacing`, `word-spacing` and
  `text-decoration` on text, and in treemaps and venn diagrams a weight or size it didn't
  measure the text in. limu reads only the weight, slant and size, and only where layout
  measures with them, because text drawn differently from how it was measured wouldn't fit
  the room layout gave it.
- Pele writes a sankey diagram's label and value as one text, with the value in a span of
  another color. limu makes them two items, since a label has one color. The label starts
  where the whole starts and the value ends where it ends, placed by the measure of the
  whole. With a measurer that kerns, the gap between them isn't quite Pele's.
- A link's host outside ASCII is lowercased and written in Punycode without the whole of
  UTS #46 (`internal/url`). Its other mappings, such as those of ligatures and combining
  marks, aren't made, so a few such hosts come out differently from Node's.

## Testing

`./scripts/verify.sh` runs `go vet`, `gofmt` and the tests in limu's module and in
limumygo's. Once limumygo's dependencies are downloaded, it works offline. CI runs it on
amd64 with `GOAMD64` at v1 and at v3, and on arm64, because a product that should stay apart
from a sum only fails where Go fuses it. On Linux, the limumygo tests need Pango
(`libpangocairo-1.0-0`), since MyGo sets text with it there.

### The oracle

The tests compare with fixtures written from Pele's own sources.

```sh
./scripts/oracle.sh ../pele [suite…]
```

The script runs Pele's TypeScript with Node 24, with nothing to install, over the cases of
each suite in `internal/oracle/suites`, and writes the fixtures to the packages'
`testdata/pele`. The tests then run without Node. Most suites build their cases from Pele's
tests and documents (`lib/corpus.mjs`) and from mutations of them (`lib/mutate.mjs`), and
`internal/oracle/cases` holds the cases of the first few. What the suites share is in
`internal/oracle/lib`, such as how values and errors are written (`value.mjs`) and how
Pele's SVG is read back with the caps and joins a browser cascades (`svg.mjs`). A suite
named `x` writes `x.json` in the package `x`, and `x-y` writes `y.json` there. `run.mjs`
lists the suites named otherwise. The oracle runs only on arm64, because the trigonometry
fixtures hold the last bits V8 gives there. It exits with code 2 for an unknown suite or a
path that isn't a checkout of Pele, and leaves a fixture whose content hasn't changed alone.

Most diagrams' tests compare what Pele's SVG draws, read back as one line per thing drawn,
with lines written from limu's scene. A fixture of drawings, `render.json`, holds the lines
of its first cases (`fixture.FullCases`, the same as `FULL` in `run.mjs`) and a digest of
the lines of each case after them, and a test compares the digest of the lines it writes.
Some tests read Pele's lines through limu's own code first, as for the author's style or the
arcs of a path, and give that reading to `Drawings.Diff`. Their suite keeps in full, beside
the digest, each line the reading changes, and the reading must change no other line.
`./scripts/oracle.sh --full` also writes every case's lines beside the fixture, in a file
git ignores, to find which line differs from a digest. The fixture is still what the tests
compare with. The render suites of pie charts, treemaps and flowcharts include cases with a
padding of 0, 2.5 and 20.

Pele writes what's inside a node or a group relative to the group's origin, while the scene
holds every position on the page, and a number moved from one to the other can round the
other way at two decimals. A suite that needs it sets `limuExactNumbers` while Pele draws.
The oracle's hooks then have Pele write its numbers in full, for the suite to move as the
scene's are moved and round as Pele does. What the oracle patches into Pele's sources is
listed in a table at the top of `internal/oracle/hooks.mjs`. Each patch must find what it
replaces exactly as many times as it says, or the oracle fails. The patches write a label
Pele draws as `<limu-label x y align attrs data n/>` and a marker as
`<limu-marker type x y dx dy color n/>`, right before Pele's own markup for it, where `n` is
how many UTF-16 code units that markup takes. `internal/oracle/lib/limu.mjs` writes and
reads them.

The outlines of `diagram/internal/draw` are compared with Pele's number for number where
Pele draws with lines and curves. A circle, an ellipse, a box with corners of its own and a
path with an arc are drawn by a `scene.Path` in cubic curves that only come close, so they
are compared as figures, within a tolerance.

The port follows Pele 1.8.0, commit `8fad23b`. Each fixture records the version and commit
it was written from, and the architecture and Node version that wrote it. The tests read
fixtures with `fixture.Load`, which fails on a fixture written from another version or
commit of Pele, or on one with an empty list at its top level, as a suite that found no
cases would write. Numbers JSON can't express, such as NaN and negative zero, are written
with `fixture.Number` and read with `fixture.Float`.
