---
title: "limu"
description: "limu reads Mermaid diagrams and lays them out in Go, for programs that draw them themselves."
---

limu reads [Mermaid](https://mermaid.js.org) diagrams and lays them out in Go, for programs that draw them themselves: natively, on a GPU with [MyGo](https://mygo.egoist.dev)'s UI, as SVG, or in a look of their own.

Most of it is ported from [Pele](https://pele.run), the small and fast Mermaid renderer of the Obsidian team. Where Pele writes SVG, limu stops one step short: it lays a diagram out as a scene, each part, what it is and where it is but not how it looks, and leaves the drawing to a backend.

```go
sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}", nil)
if err != nil {
	return err
}
w := limumygo.View(c, sc, &limumygo.Options{Look: style.Look{Theme: style.KohuDark()}})
```

This site is being written; the documents and the examples are on their way.
