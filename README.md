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
> **An early, personal project.** It reads flowcharts; state, class, entity
> relationship, requirement, use case, agent flow and swimlane diagrams; and
> sequence, gantt, timeline, user journey, git graph, packet and kanban diagrams.
> The charts and the rest follow, one family at a time. The API will change in
> any 0.x release.

```go
sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}", nil)
if err != nil {
	return err
}
out := svg.Render(sc, nil)
```

## How it is built

```
text ──parse──▶ model ──layout──▶ scene ──backend──▶ SVG, a GPU, a grid of pixels
```

| Package | What it does |
|---|---|
| `limu` | `Layout`: a diagram's text to its scene. |
| `diagram` | Reads a diagram's front matter and directives, and tells its type; a package for each type parses it and lays it out. |
| `text` | Measures text, through a `Measurer` the host gives, and lays out labels: lines, wrapping, emphasis, icons, entities. `text.Arial` measures with a table of Arial's advances, with no font at hand. |
| `scene` | A diagram laid out: items, their roles and shapes, boxes, paths, markers, labels. |
| `style` | How a scene looks, at three levels: a theme of colors by token, overrides by role, and a backend's drawing of its own. |
| `layout` | What changes where things are: the measurer, the size of text, a grid to snap to, how edges are routed, the present and the time zone of a gantt chart. The layered and compound layouts. |
| `draw` | The geometry diagrams share: shapes' outlines and edges' paths. |
| `svg` | Draws a scene as SVG, its theme as CSS variables. |
| `mygoui` | Draws a scene with MyGo's native UI, and finds the item under the pointer. A module of its own, so that the rest needs only the standard library. |

What changes where things are goes to layout; everything else is style, and a
change of it only redraws. [docs/design.md](docs/design.md) has the details and
the roadmap.

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

A part's style is the theme's, under the diagram's own, under the overrides for
its role, under what the diagram's author set with `classDef` and `style`. To
draw a part some other way, a backend takes a function in place of its
default:

```go
svg.Render(sc, &svg.Options{Draw: func(w *svg.Writer, it *scene.Item, s style.Style, next func()) {
	if it.Shape != "diam" {
		next()
		return
	}
	// write the diamond some other way
}})
```

## Testing

```sh
./scripts/verify.sh
```

runs `go vet`, `gofmt` and the tests, offline. The tests compare with
fixtures written from Pele's own sources:

```sh
./scripts/oracle.sh ../pele
```

runs Pele's TypeScript with Node 24, with nothing to install, over the cases in
`internal/oracle/cases`, and writes the fixtures to the packages' `testdata/pele`.
Run it after adding a case, or to follow a new version of Pele.

## License

MIT — see [LICENSE](LICENSE), which carries Pele's license too.
