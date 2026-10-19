---
title: "SVG"
description: "Writing diagrams as SVG, themed with CSS variables, and drawing diagrams that are not trusted."
---

The svg package writes a scene as an SVG document, as Pele draws it. Like limu, it needs only Go's standard library.

```go
sc, err := limu.Layout(src, nil)
if err != nil {
	return err
}
out := svg.Render(sc, &svg.Options{
	Look:       style.Look{Theme: style.Kohu()},
	FontFamily: "Arial, Helvetica, sans-serif",
})
```

## Options

| Field | What it does |
| --- | --- |
| `Look` | The theme and the overrides. See [Theming](/theming). |
| `Literal` | Writes the theme's colors as colors, not as CSS variables. |
| `FontFamily`, `MonoFamily` | The fonts the scene was measured in. Without them, `sans-serif` and `monospace`. |
| `LinkSchemes`, `ImageSchemes` | The schemes links and images may use besides a relative address. See [Diagrams that are not trusted](#diagrams-that-are-not-trusted). |
| `IconMarkup` | The markup of an icon, by its name. See [Icons](#icons). |
| `Draw` | Writes each item in place of the default. See [Drawing of one's own](#drawing-of-ones-own). |

### Fonts

Layout measures text with a measurer, `text.Arial` unless the host gives another, and the SVG must set its text in the fonts it was measured in, or labels do not fit their shapes. With `text.Arial`, set it in Arial, or a font as wide:

```go
svg.Render(sc, &svg.Options{FontFamily: "Arial, Helvetica, sans-serif"})
```

A page's `--limu-font` and `--limu-font-mono` come before these, so a page that measures text in its own font can set the SVG in it too.

## Theme variables

The SVG writes each color of the theme as a CSS variable, with the theme's color as its fallback: `var(--limu-surface,#e5eeec)`. A diagram set in a page takes the page's variables where it sets them, so the page changes its colors, as for a light and a dark theme, without drawing it again:

```css
.diagram {
  --limu-bg: #f1f6f5;
  --limu-fg: #0f262a;
  --limu-line: #23393d;
  --limu-surface: #e5eeec;
  --limu-border: #76939a;
  --limu-accent: #a92e4a;
}

@media (prefers-color-scheme: dark) {
  .diagram {
    --limu-bg: #0c1618;
    --limu-fg: #d0dddc;
    --limu-line: #b4c7c7;
    --limu-surface: #142428;
    --limu-border: #557176;
    --limu-accent: #ee8494;
  }
}
```

| Variable | Token |
| --- | --- |
| `--limu-bg` | `Bg` |
| `--limu-fg` | `Fg` |
| `--limu-muted` | `Muted` |
| `--limu-line` | `Line` |
| `--limu-surface` | `Surface` |
| `--limu-surface-alt` | `SurfaceAlt` |
| `--limu-border` | `Border` |
| `--limu-accent` | `Accent` |
| `--limu-critical` | `Critical` |
| `--limu-series-1` … `--limu-series-8` | `Series(0)` … `Series(7)` |
| `--limu-font`, `--limu-font-mono` | The fonts, as above. |

[Theming](/theming#themes) has the values of limu's themes. A color of a part's own, as one the diagram's author set, is written as a color, and does not change with the variables.

An SVG shown with `<img>`, or saved as a file, sees no page's variables, and is drawn in the fallbacks. `Literal` writes the colors themselves, for programs that read SVG without CSS's variables.

## Classes

The SVG writes what each item is as classes, for a page's CSS:

| Class | Element |
| --- | --- |
| `limu`, `limu-<type>` | The root `<svg>`, as `limu-flowchart` |
| `limu-node`, `limu-group`, `limu-edge`, `limu-text`, `limu-mark` | The group of an item, by its kind |
| `limu-<role>` | The group of an item, by its role, its dot a dash: `limu-flowchart-node` |
| `limu-shape-<shape>` | The group of an item with an outline, by its shape: `limu-shape-diam` |
| the item's tags and classes | What the diagram says it is beyond its role, as a gantt task's `done`, and the classes its author gave it |
| `limu-marker` | A marker at the end of an edge |
| `limu-icon` | An icon |

An item with an id has it as `data-id`:

```css
.limu-flowchart-node[data-id="scene"] rect {
  stroke: var(--limu-accent);
  stroke-width: 2;
}
```

The root has the scene's size as its `width` and `height`, and as its `viewBox`. To scale it down in a narrower page, give it `max-width: 100%` and `height: auto`.

A diagram's title and description, from `accTitle` and `accDescr`, are its `<title>` and `<desc>`, and its root has the role `graphics-document`, for assistive technology.

## Icons

A label can hold an icon, as `fa:fa-car`, and a flowchart's node can show one. `IconMarkup` gives the markup inside an icon's `<svg>`, on a box of 24 by 24, by the icon's name; the markup draws in `currentColor`, which is the color of the item's text. Layout leaves room in a label only for the icons the host says it can draw, through `layout.Options.Icons`, so give both the same set:

```go
icons := map[string]string{
	"fa:fa-car": `<path d="M5 17h14v-5l-2-5H7l-2 5z" fill="currentColor"/>`,
}
sc, err := limu.Layout(src, &layout.Options{
	Icons: func(name string) bool { _, ok := icons[name]; return ok },
})
out := svg.Render(sc, &svg.Options{
	IconMarkup: func(name string) string { return icons[name] },
})
```

Without `IconMarkup`, icons are left out. Architecture and tree view diagrams also draw icons of their own, which need no markup.

## Drawing of one's own

`Draw` writes each item in place of the default, which `next` writes. The `svg.Writer` it is given writes the default's pieces, `Item`, `Parts`, `Ends`, `Picture` and `Label`, and markup of one's own, with `Raw` for markup, `Text` for escaped text and `Num` for numbers. It is given the item's style cascaded from the look, its paints not settled into colors, so that `Paint` and `PaintAttr` write the theme's colors as CSS variables.

```go
svg.Render(sc, &svg.Options{
	Look: look,
	Draw: func(w *svg.Writer, it *scene.Item, s style.Style, next func()) {
		if it.Kind != scene.Node {
			next()
			return
		}
		// A node as a box of its own, with its label as the default writes it.
		b := it.Box
		w.Raw(`<rect rx="8" x="`)
		w.Num(b.X)
		w.Raw(`" y="`)
		w.Num(b.Y)
		w.Raw(`" width="`)
		w.Num(b.W)
		w.Raw(`" height="`)
		w.Num(b.H)
		w.Raw(`"`)
		w.PaintAttr("fill", s.Fill)
		w.Raw(`/>`)
		w.Label(it, s)
	},
})
```

## Diagrams that are not trusted

A diagram's text may come from someone a program does not trust, as a note shared with it. limu escapes the text it writes, and writes no script, no event handler and no HTML; a `click` that calls a function is left out. What else such a diagram can do is in the host's hands.

### Links and images

A diagram may link its parts to addresses, and show images from them. Layout keeps a link or an image only when its address is relative, or of a scheme the host allows:

| Used for | Schemes allowed by default |
| --- | --- |
| Links | `http`, `https`, `mailto`, `tel` |
| Images | `http`, `https` |

`javascript:`, `data:` and `vbscript:` are never allowed, nor are network paths, as `//host/share`. An image shown makes a request to its address when the SVG is drawn. To leave out every link or image, set `NoLinks` or `NoImages` in `layout.Options`.

The schemes are checked twice: by layout, with `layout.Options.LinkSchemes` and `ImageSchemes`, and by the svg package, with its own, as a last check. To allow another, give it to both:

```go
schemes := []string{"http", "https", "mailto", "obsidian"}
sc, err := limu.Layout(src, &layout.Options{LinkSchemes: schemes})
out := svg.Render(sc, &svg.Options{LinkSchemes: schemes})
```

A link is written as `<a href>`, with no target. A page that shows the SVG inline decides how its links open, as it does for its own.

### Limits

The length of a diagram's text and the edges of a flowchart or an agent flow are limited, by default to 50,000 UTF-16 code units and 5,000 edges; see [Limits](/api#limits). limu cannot stop a layout once it has started, so for text that is not trusted, keep the limits.

### What the host gives

- **Icons.** The svg package writes the markup `IconMarkup` gives as it is. Give markup from a set you trust, and nothing for a name you do not know; do not write the name into the markup.
- **Errors.** An error's message can hold text from the diagram. Show it as text, not as markup.
- **Styles.** An author's styles are read into limu's own, property by property, and written as attributes; a value that could reach outside the diagram, as `url()`, is left out.
- **Options.** Do not let a diagram's author set the options a host gives, as `diagram.Options.Config` or the schemes.
