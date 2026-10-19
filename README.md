# limu

English | [简体中文](README.zh.md)

[![verify](https://github.com/firede/limu/actions/workflows/verify.yml/badge.svg?branch=main)](https://github.com/firede/limu/actions/workflows/verify.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/firede/limu.svg)](https://pkg.go.dev/github.com/firede/limu)

limu is a diagram layout engine in Go. It reads Mermaid and leaves the drawing to
your program, which can draw natively with [MyGo](https://mygo.egoist.dev), as SVG,
or in a style of its own. [limu.firede.com](https://limu.firede.com) has the documentation, examples and a
playground.

Most of limu is ported from [Pele](https://github.com/obsidianmd/pele), the small,
fast Mermaid renderer made by the Obsidian team. Pele writes SVG, and limu stops one
step earlier, at the scene. A scene is the laid-out diagram as data. It says what
each part is and where it goes, but not how it looks, and a backend draws it. limu's
tests compare its output with Pele's, number for number.

> [!WARNING]
> **limu is an early, personal project**, and its API may change in any 0.x
> release. It supports every diagram type Pele draws, from flowcharts and sequence
> diagrams to sankey, C4 and wardley diagrams.
> [Compatibility](https://limu.firede.com/compatibility) lists them all.

```go
sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}", nil)
if err != nil {
	return err
}
out := svg.Render(sc, nil)
```

To lay out a diagram more than once, for example when its width changes, parse it
once and lay out the result as often as you need, from any goroutine. `Parse` takes
the limits and configuration that Pele's `parse` takes.

```go
d, err := limu.Parse(src, diagram.Options{Config: diagram.Config{"seed": 7.0}})
if err != nil {
	return err
}
sc, err := d.Layout(&layout.Options{MaxWidth: 480, Padding: new(0.0)})
```

## How it's built

```
text ──parse──▶ model ──layout──▶ scene ──backend──▶ SVG, a GPU, a grid of pixels
```

| Package | What it does |
|---|---|
| `limu` | Parses a diagram and lays it out. `Parse` lets you lay out one diagram many times, and `Types` and `Supports` tell which types limu supports. |
| `diagram` | Reads the front matter, directives and type that every diagram shares. Each type has a package under it that parses and lays out that type. Code the types share lives under `diagram/internal`, with the layered, compound and swimlane layouts in `graphs`, the geometry of shapes and edges in `draw`, the styles authors set with `classDef` and `style` in `css`, and the rest in `common`. |
| `text` | Measures text with the `Measurer` you provide, and lays out labels with line breaks, wrapping, emphasis, icons and entities. `text.Arial` measures from a table of Arial's character widths without loading a font. |
| `scene` | Describes a laid-out diagram as items with roles, shapes, boxes, paths, markers and labels. |
| `style` | Holds themes, which give colors by token, and overrides for each role. |
| `layout` | Holds the options that change where things go, such as the measurer, text size, available width, padding, grid snapping, edge routing, and the current time and time zone for gantt charts. |
| `svg` | Renders a scene as SVG, with theme colors as CSS variables. |
| `limumygo` | Draws a scene with MyGo's native UI and finds the item under the pointer. It's a separate module, so the rest of limu needs only the standard library. Its `limumygo/mermaid` package adds Mermaid to MyGo's code editor and highlights each diagram type's keywords. |

Anything that changes where things go belongs to layout. Everything else is style,
and changing it only redraws. [docs/design.md](docs/design.md) has the details and
the roadmap.

To try limu on your own diagram, open the
[playground](https://limu.firede.com/playground). It shows the text next to the
diagram drawn by limumygo and its SVG, and its Examples menu opens any of limu's
examples. The same program runs on the desktop with
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

limu has three themes. `style.Default()` is Pele's light theme, and `style.Kohu()`
and `style.KohuDark()` are limu's own light and dark themes. A part's style is
layered from the theme up through the diagram's own style and your overrides for its
role, with what the diagram's author set with `classDef` and `style` on top. To draw a part
another way, give a backend a function to use in place of its default drawing. The
function can still draw the default pieces.

```go
svg.Render(sc, &svg.Options{Draw: func(w *svg.Writer, it *scene.Item, s style.Style, next func()) {
	if it.Shape != "diam" {
		next()
		return
	}
	// Write the diamond your own way, then its label the default way.
	w.Label(it, s)
}})
```

## Testing

```sh
./scripts/verify.sh
```

This runs `go vet`, `gofmt` and the tests in limu's module and in the limumygo
module. Once limumygo's dependencies are downloaded, it works offline. CI runs it on
amd64, with `GOAMD64` at v1 and v3, and on arm64. On Linux, the limumygo tests need
Pango (`libpangocairo-1.0-0`).

The tests compare against fixtures generated from Pele's own source.

```sh
./scripts/oracle.sh ../pele
```

This runs Pele's TypeScript with Node 24, with nothing to install, over the cases of
the suites in `internal/oracle/suites`, and writes the fixtures to each package's
`testdata/pele`. It runs only on arm64. Run it after adding a case or to follow a new
version of Pele.

## License

MIT. See [LICENSE](LICENSE), which also includes Pele's license.
