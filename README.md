# limu

English | [简体中文](README.zh.md)

[![verify](https://github.com/firede/limu/actions/workflows/verify.yml/badge.svg?branch=main)](https://github.com/firede/limu/actions/workflows/verify.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/firede/limu.svg)](https://pkg.go.dev/github.com/firede/limu)

Mermaid diagrams, read and laid out in Go, for programs that draw them
themselves — on a GPU with [MyGo](https://mygo.egoist.dev)'s native UI, as
SVG, or in a look of their own.

Most of it is ported from [Pele](https://github.com/obsidianmd/pele), the
small and fast Mermaid renderer of the Obsidian team. Where Pele writes SVG,
limu stops one step short: it lays a diagram out as a scene — each part, what
it is and where it is, but not how it looks — and leaves the drawing to a
backend. Its tests compare what it does with what Pele does, number for
number.

> [!WARNING]
> **An early, personal project.** It reads every type of diagram Pele draws:
> flowcharts; state, class, entity relationship, requirement, use case, agent
> flow and swimlane diagrams; sequence, gantt, timeline, user journey, git graph,
> packet and kanban diagrams; pie, XY, quadrant and radar charts, sankey
> diagrams, treemaps and venn diagrams; and mindmaps, architecture, C4, block,
> wardley, railroad, ishikawa, cynefin, event modeling, tree view and info
> diagrams. The API will change in any 0.x release.

```go
sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}", nil)
if err != nil {
	return err
}
out := svg.Render(sc, nil)
```

A host that lays a diagram out again, as when the width it has changes, parses
it once, with the limits and configuration Pele's `parse` takes, and lays it out
as often as it needs, from any goroutine:

```go
d, err := limu.Parse(src, diagram.Options{Config: diagram.Config{"seed": 7.0}})
if err != nil {
	return err
}
sc, err := d.Layout(&layout.Options{MaxWidth: 480, Padding: new(0.0)})
```

## How it is built

```
text ──parse──▶ model ──layout──▶ scene ──backend──▶ SVG, a GPU, a grid of pixels
```

| Package | What it does |
|---|---|
| `limu` | `Layout`: a diagram's text to its scene; `Parse`, to lay one diagram out many times; `Types` and `Supports`, the types it reads. |
| `diagram` | Reads a diagram's front matter and directives, and tells its type; a package for each type parses it and lays it out. What the diagrams share is under `diagram/internal`: `graphs`, the layered, compound and swimlane layouts; `draw`, the geometry of shapes and edges; `css`, what authors set with `classDef` and `style`; and `common`. |
| `text` | Measures text, through a `Measurer` the host gives, and lays out labels: lines, wrapping, emphasis, icons, entities. `text.Arial` measures with a table of Arial's advances, with no font at hand. |
| `scene` | A diagram laid out: items, their roles and shapes, boxes, paths, markers, labels. |
| `style` | How a scene looks, at three levels: a theme of colors by token, overrides by role, and a backend's drawing of its own. |
| `layout` | What changes where things are: the measurer, the size of text, the width the host has, the padding about a diagram, a grid to snap to, how edges are routed, the present and the time zone of a gantt chart. |
| `svg` | Draws a scene as SVG, its theme as CSS variables. |
| `limumygo` | Draws a scene with MyGo's native UI, and finds the item under the pointer. A module of its own, so that the rest needs only the standard library. Its package `limumygo/mermaid` is Mermaid's text as a language of MyGo's code editor, highlighted by the keywords of the diagram's type. |

What changes where things are goes to layout; everything else is style, and a
change of it only redraws. [docs/design.md](docs/design.md) has the details and
the roadmap.

To try limu on a diagram of your own, run the playground, which shows the text
beside its diagram, drawn by limumygo, and its SVG, and has a page of examples of
every type of diagram, each of which opens in the playground:
`cd limumygo && go run ./examples/playground`.

## Style

```go
look := style.Look{
	Theme: style.Default(), // Bg, Fg, Surface, Border, Accent, eight series…
	Overrides: style.Overrides{
		"flowchart.node": {Fill: style.Accent, StrokeWidth: style.N(2)},
		"flowchart.edge": {Dash: style.Dashes(3, 4)},
	},
}
out := svg.Render(sc, &svg.Options{Look: look})
```

The themes limu has are `style.Default()`, Pele's light theme, and its own
`style.Kohu()` and `style.KohuDark()`, a light and a dark one. A part's style is
the theme's, under the diagram's own, under the overrides for its role, under
what the diagram's author set with `classDef` and `style`. To draw a part some
other way, a backend takes a function in place of its default, and gives it the
default's pieces to draw with:

```go
svg.Render(sc, &svg.Options{Draw: func(w *svg.Writer, it *scene.Item, s style.Style, next func()) {
	if it.Shape != "diam" {
		next()
		return
	}
	// write the diamond some other way, then its label as the default does
	w.Label(it, s)
}})
```

## Testing

```sh
./scripts/verify.sh
```

runs `go vet`, `gofmt` and the tests, in limu's module and in the limumygo
module, offline once limumygo's modules are downloaded. CI runs it on amd64, at `GOAMD64` v1 and v3, and on arm64. On
Linux, the limumygo tests need Pango (`libpangocairo-1.0-0`). The tests compare
with fixtures written from Pele's own sources:

```sh
./scripts/oracle.sh ../pele
```

runs Pele's TypeScript with Node 24, with nothing to install, over the cases of
the suites in `internal/oracle/suites`, and writes the fixtures to the packages'
`testdata/pele`. It runs only on arm64. Run it after adding a case, or to follow
a new version of Pele.

## License

MIT — see [LICENSE](LICENSE), which carries Pele's license too.
