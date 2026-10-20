# limu

English | [简体中文](README.zh.md)

[![verify](https://github.com/firede/limu/actions/workflows/verify.yml/badge.svg?branch=main)](https://github.com/firede/limu/actions/workflows/verify.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/firede/limu.svg)](https://pkg.go.dev/github.com/firede/limu)

limu is a diagram layout engine in Go. It reads Mermaid and leaves the drawing to
your program, which can draw natively with [MyGo](https://mygo.egoist.dev), as SVG,
or in a style of its own.

Most of limu is ported from [Pele](https://github.com/obsidianmd/pele), the Mermaid
renderer made by the Obsidian team. Pele writes SVG. limu stops one step earlier, at
the scene, which is the laid-out diagram as data. limu's tests compare its output
with Pele's, number for number.

> [!WARNING]
> limu is an early, personal project, and its API may change in any 0.x release.

## Getting started

```sh
go get github.com/firede/limu
```

```go
sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}", nil)
if err != nil {
	return err
}
out := svg.Render(sc, nil)
```

limu reads every diagram type Pele draws, from flowcharts and sequence diagrams to
sankey, C4 and wardley diagrams. To draw with MyGo, use the separate module
`github.com/firede/limu/limumygo`.

## Documentation

[limu.firede.com](https://limu.firede.com) has the documentation, examples of every
diagram type, and a [playground](https://limu.firede.com/playground) to try your
own diagrams.

- [Getting started](https://limu.firede.com/getting-started)
- [MyGo](https://limu.firede.com/mygo), [SVG](https://limu.firede.com/svg) and
  [Theming](https://limu.firede.com/theming)
- [API](https://limu.firede.com/api)
- [Architecture](https://limu.firede.com/architecture) and
  [Compatibility](https://limu.firede.com/compatibility)

## Development

```sh
./scripts/verify.sh
```

This runs `go vet`, `gofmt` and the tests of both modules. On Linux, the limumygo
tests need Pango (`libpangocairo-1.0-0`). [docs/design.md](docs/design.md) explains
how limu is built and how it's checked against Pele, and [AGENTS.md](AGENTS.md) has
the rules for changing it.

## License

MIT. See [LICENSE](LICENSE), which also includes Pele's license.
