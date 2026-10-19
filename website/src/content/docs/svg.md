---
title: "SVG"
description: "Render diagrams as SVG, style them with CSS variables, and handle untrusted input."
---

The svg package renders a scene as an SVG document that looks the way Pele draws it. Like the rest of limu, it uses only Go's standard library.

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
| `Look` | Sets the theme and overrides. See [Theming](/theming). |
| `Literal` | Writes theme colors as plain colors instead of CSS variables. |
| `FontFamily`, `MonoFamily` | Name the fonts the scene was measured with. They default to `sans-serif` and `monospace`. |
| `LinkSchemes`, `ImageSchemes` | Set the URL schemes that links and images may use besides relative URLs. See [Untrusted diagrams](#untrusted-diagrams). |
| `IconMarkup` | Returns the markup for an icon by name. See [Icons](#icons). |
| `Draw` | Writes each item in place of the default. See [Custom drawing](#custom-drawing). |

### Fonts

Layout measures text with `text.Arial` unless you give it another measurer. The SVG has to use the same fonts the text was measured with, or labels won't fit their shapes. With `text.Arial`, use Arial or a font of the same width.

```go
svg.Render(sc, &svg.Options{FontFamily: "Arial, Helvetica, sans-serif"})
```

If a page sets `--limu-font` or `--limu-font-mono`, those take precedence. A page that measures text in its own font can use them to set the SVG in that font too.

## Theme variables

The SVG writes each theme color as a CSS variable with the theme's color as the fallback, as in `var(--limu-surface,#e5eeec)`. When the SVG is inline in a page, any variables the page sets win. That lets the page change the colors, for example between light and dark themes, without rendering the diagram again.

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
| `--limu-font`, `--limu-font-mono` | The fonts described above |

[Theming](/theming#themes) lists the values of limu's themes. Colors that don't come from the theme, such as those a diagram's author sets, are written as plain colors and ignore the variables.

An SVG shown through `<img>` or saved to a file can't see the page's variables, so it uses the fallbacks. Set `Literal` for programs that read SVG but don't understand CSS variables.

## Classes

Every item's group carries classes that describe it, so a page's CSS can target them.

| Class | Element |
| --- | --- |
| `limu`, `limu-<type>` | The root `<svg>`, such as `limu-flowchart` |
| `limu-node`, `limu-group`, `limu-edge`, `limu-text`, `limu-mark` | An item's group, by kind |
| `limu-<role>` | An item's group, by role with the dot replaced by a dash, such as `limu-flowchart-node` |
| `limu-shape-<shape>` | An item's group, by shape, such as `limu-shape-diam` |
| the item's tags and classes | Extra facts the diagram records, such as a gantt task's `done`, and the classes the author assigned |
| `limu-marker` | A marker at the end of an edge |
| `limu-icon` | An icon |

Items with an id also get a `data-id` attribute.

```css
.limu-flowchart-node[data-id="scene"] rect {
  stroke: var(--limu-accent);
  stroke-width: 2;
}
```

The root's `width`, `height` and `viewBox` match the scene's size. To let it shrink on a narrow page, give it `max-width: 100%` and `height: auto`.

For assistive technology, the diagram's `accTitle` and `accDescr` become its `<title>` and `<desc>`, and the root has the role `graphics-document`.

## Icons

Labels can contain icons such as `fa:fa-car`, and flowchart nodes can show one. `IconMarkup` returns the markup that goes inside an icon's `<svg>` on a 24 by 24 grid. The markup should draw in `currentColor`, which is set to the item's text color. Layout leaves room in labels only for icons you report through `layout.Options.Icons`, so give both the same set.

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

Without `IconMarkup`, icons are left out. Architecture and tree view diagrams also have built-in icons, which need no markup.

## Custom drawing

`Draw` writes each item in place of the default, and `next` writes it the default way. The `svg.Writer` it receives can write the default pieces with `Item`, `Parts`, `Ends`, `Picture` and `Label`, and your own markup with `Raw`, escaped text with `Text`, and numbers with `Num`. The style it receives is cascaded from the look but not yet resolved to colors, so `Paint` and `PaintAttr` can still write theme colors as CSS variables.

```go
svg.Render(sc, &svg.Options{
	Look: look,
	Draw: func(w *svg.Writer, it *scene.Item, s style.Style, next func()) {
		if it.Kind != scene.Node {
			next()
			return
		}
		// Draw the node as a box of our own, and its label the default way.
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

## Untrusted diagrams

Diagram text sometimes comes from people you don't trust, for example in notes shared with your app. limu escapes all the text it writes, and it never writes scripts, event handlers or HTML. A `click` that calls a function is dropped. The rest is up to your app.

### Links and images

Diagrams can link parts to URLs and show images from URLs. Layout keeps a link or an image only if its URL is relative or uses a scheme you allow.

| Used for | Schemes allowed by default |
| --- | --- |
| Links | `http`, `https`, `mailto`, `tel` |
| Images | `http`, `https` |

`javascript:`, `data:` and `vbscript:` are always blocked, and so are network paths such as `//host/share`. Images make a network request when the SVG is displayed. To drop all links or all images, set `NoLinks` or `NoImages` in `layout.Options`.

Schemes are checked twice, once by layout using `layout.Options.LinkSchemes` and `ImageSchemes`, and again by the svg package using its own options as a final check. To allow another scheme, add it to both.

```go
schemes := []string{"http", "https", "mailto", "obsidian"}
sc, err := limu.Layout(src, &layout.Options{LinkSchemes: schemes})
out := svg.Render(sc, &svg.Options{LinkSchemes: schemes})
```

Links are written as `<a href>` without a target. When the SVG is inline, the page decides how links open, just as it does for its own links.

### Limits

By default, diagram text is limited to 50,000 UTF-16 code units, and flowcharts and agent flows to 5,000 edges. See [Limits](/api#limits). limu can't stop a layout once it has started, so keep these limits for untrusted text.

### What your app is responsible for

The svg package inserts the markup from `IconMarkup` as is. Only return markup from a set you trust, return nothing for unknown names, and never put the name itself into the markup.

Error messages can quote the diagram, so display them as text, not markup.

limu reads an author's styles property by property and writes them as attributes, and it drops values that could reach outside the diagram, such as `url()`. Don't let diagram authors set your options, such as `diagram.Options.Config` or the allowed schemes.
