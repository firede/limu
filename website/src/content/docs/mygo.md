---
title: "MyGo"
description: "Draw diagrams natively with limumygo, and find the part under the pointer."
---

The limumygo package draws limu's scenes on the GPU with [MyGo](https://mygo.egoist.dev)'s native UI, and tells you which part of a diagram is under the pointer. This is what limu was built for. The diagram lives in your app's own window, uses your app's fonts, and can answer questions about itself.

## Installing

limumygo is a separate module, which keeps limu's other packages free of anything beyond Go's standard library.

```shell
go get github.com/firede/limu/limumygo
```

You also need whatever MyGo needs to build an app. See [MyGo's documentation](https://mygo.egoist.dev).

## Showing a diagram

`limumygo.View` builds an element the size of a scene and draws the scene in it. Lay out the diagram with the same fonts you draw it in, as explained under [Fonts](#fonts).

```go
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/firede/limu"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/limumygo"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

type app struct {
	sc      *scene.Scene
	clicked string
}

func (a *app) view(c *ui.Context) {
	theme := style.Kohu()
	if c.Theme().Dark {
		theme = style.KohuDark()
	}
	ui.Column(c).Padding(16).Gap(12).Children(func() {
		w := limumygo.View(c, a.sc, &limumygo.Options{
			Look:  style.Look{Theme: theme},
			Fonts: limumygo.DefaultFonts,
		})
		if it := w.Clicked(); it != nil {
			a.clicked = string(it.Role) + " " + it.ID
		}
		ui.Text(c, "Clicked: "+a.clicked)
	})
}

func main() {
	sc, err := limu.Layout(`flowchart LR
  text[Mermaid text] --> layout[Layout] --> scene[Scene]`,
		&layout.Options{Measurer: limumygo.DefaultFonts})
	if err != nil {
		log.Fatal(err)
	}
	a := &app{sc: sc}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Diagram", Width: 640, Height: 360, Content: ui.View(a.view)})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

When the theme changes, for example when the system switches to dark mode, the scene is only redrawn.

The widget shows the tooltip of the item under the pointer and turns the pointer into a hand over items with a link. `Hovered` returns the item under the pointer, and `Clicked` returns the item clicked since the last frame. Both return nil when there's no item. `Element` gives you the widget's element so you can place it or give it a background. Assistive technology reads the diagram as an image named by the scene's title, and `Label` sets a different name.

To fit a diagram to its element, read the element's width from `LayoutBounds`, which holds the size from the previous frame. When the width changes, lay the diagram out again with a new `MaxWidth`. Parse the text once with `limu.Parse`, and lay out large diagrams on another goroutine, as the [playground](/playground) does.

### Finding the item under the pointer

`limumygo.ItemAt` finds the item at a point in scene coordinates, without needing a widget. When several items overlap, it returns the one drawn last, which is the one on top. A point hits an item when it's inside the item's box or its label's box. Edges and items made only of lines are hit when the point is within a tolerance of a line or inside one of their markers. The widget uses `limumygo.Slop`, which is 4 DIPs.

```go
if it := limumygo.ItemAt(sc, scene.Point{X: x, Y: y}, limumygo.Slop); it != nil {
	fmt.Println(it.Role, it.ID)
}
```

## Painting

`limumygo.Paint` draws a scene on a `ui.Painter` with its top left corner at a given point, in DIPs. Use it to draw a diagram inside an element of your own.

```go
o := &limumygo.Options{Look: look, Fonts: fonts, Scale: 2}
ui.Box(c).Size(float32(sc.Width*2), float32(sc.Height*2)).Draw(func(p *ui.Painter, r ui.Rect) {
	limumygo.Paint(p, sc, r.X, r.Y, o)
})
```

`Options.Scale` sets how many DIPs one scene unit takes. To zoom or pan, change the scale and the point you draw at, rather than draw under `ui.Painter.Transform` or inside a transformed element. A transformed diagram is still drawn whole. But MyGo draws each path from a mask made at the painter's scale, and a transform stretches it, so a diagram zoomed in this way looks blurred. A very large path can also show faint lines where its tiles meet.

limumygo draws the same picture as the svg package, with a few differences.

- MyGo has no way to draw a group at an opacity, so limumygo applies an item's opacity to each of its colors. Where an item's parts overlap, the lower part shows through.
- When a corner radius is too large for a box, limumygo uses the smaller of half the width and half the height. SVG uses elliptical corners instead.
- MyGo strokes paths only with round ends and corners. Lines with other ends or corners are drawn by filling their outline.

## Fonts

A diagram is laid out with a measurer and drawn with fonts. If the two don't match, text won't fit the space layout gave it. `limumygo.Fonts` does both jobs. It's a `text.Measurer` that measures with the same fonts the painter draws with, shaped by the system's text engine just as MyGo shapes the rest of its text.

```go
fonts := limumygo.NewFonts("Inter, system-ui", "JetBrains Mono, monospace")
sc, err := limu.Layout(src, &layout.Options{Measurer: fonts})
w := limumygo.View(c, sc, &limumygo.Options{Look: look, Fonts: fonts})
```

`limumygo.DefaultFonts` uses the system font and its monospaced font. A family is a list of fonts to try in order, as `ui.Font` takes it. Bold and italic text use the family's bold weight and italics, and monospaced text uses the mono family. Fonts cache the text they shaped most recently, so measuring and then drawing the same label costs nothing extra. They're safe to use from several goroutines.

Chinese, Japanese and Korean text is measured in the fonts it's drawn in, falling back to the system's fonts for scripts the family doesn't cover. The system's text engine adds a little space where Han characters meet Latin letters or digits, and `Fonts` measures that space too.

## Icons and images

limumygo never loads images or follows links on its own. Your app knows where its images come from and which links are safe to open, so it makes those decisions.

`Options.DrawIcon` draws an icon by name, such as `fa:fa-car`, in the color of the item's text. Layout leaves room in labels only for icons you report through `layout.Options.Icons`, so give both the same set.

```go
icons := map[string]*ui.SVG{"fa:fa-car": car}
sc, err := limu.Layout(src, &layout.Options{
	Measurer: fonts,
	Icons:    func(name string) bool { return icons[name] != nil },
})
o := &limumygo.Options{
	Look:  look,
	Fonts: fonts,
	DrawIcon: func(p *ui.Painter, name string, r ui.Rect, c ui.Color) {
		if s := icons[name]; s != nil {
			p.Icon(s, r, c)
		}
	},
}
```

`Options.DrawImage` draws an image from its URL into a box. When `fit` is set, the image keeps its proportions and is centered. Otherwise it's stretched to fill the box. Layout only passes through URLs whose scheme `layout.Options.ImageSchemes` allows, and loading and caching the image is up to you. Without `DrawIcon` or `DrawImage`, icons and images are left out.

When an item with a link is clicked, its `Link` field holds the target. Open it the same way your app opens any other link, or don't.

## Custom drawing

`Options.Draw` draws each item in place of the default, and `next` draws it the default way. The `limumygo.Painter` it receives embeds the `ui.Painter` and converts scene coordinates to DIPs with `Point`, `Rect`, `Len` and `Path`. It can also draw the default pieces with `Item`, `Parts`, `Ends`, `Picture` and `Label`. The style arrives as a `style.Resolved`, with every color already worked out, because MyGo draws with plain colors.

```go
o.Draw = func(p *limumygo.Painter, it *scene.Item, s style.Resolved, next func()) {
	if it.Kind != scene.Node {
		next()
		return
	}
	// Draw the node as a box of our own, and its label the default way.
	p.Fill(p.Rect(it.Box), limumygo.Color(s.Fill, s.Opacity), 8)
	p.Label(it, s)
}
```

[mygo-platinum](https://github.com/firede/mygo-platinum) uses this to draw diagrams in the style of the classic Mac OS.

## Editor highlighting

The `limumygo/mermaid` package makes Mermaid a language in MyGo's code editor. It knows the keywords of each diagram type and picks the type from the line that declares it.

```go
buf := editor.NewBuffer(src)
buf.SetLanguage(mermaid.Language)
```

It reads one line at a time and doesn't parse the diagram. Use limu's errors to find out whether the text is valid.

## In a browser

MyGo apps can also run in a browser as WebAssembly, and limumygo comes along. The [playground](/playground) works this way. See MyGo's documentation for how to build and load such an app. In a browser, the browser shapes text, and `Fonts` measures it the way the browser sets it. limu and limumygo add about 7 MB to the uncompressed WebAssembly.
