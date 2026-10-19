---
title: "MyGo"
description: "Drawing diagrams natively with limumygo: the widget, painting, fonts, and finding the item under the pointer."
---

The limumygo package draws limu's scenes with [MyGo](https://mygo.egoist.dev)'s native UI, on the GPU, and tells which part of a diagram is under the pointer. It is what limu is for: a diagram drawn in an app's own window, in its own fonts, that the app can ask about.

## Installing

limumygo is a module of its own, so that limu's other packages need only Go's standard library:

```shell
go get github.com/firede/limu/limumygo
```

It needs what MyGo needs to build an app; see [MyGo's documentation](https://mygo.egoist.dev).

## Showing a diagram

`limumygo.View` builds an element the size of a scene, which draws it. Lay the diagram out with the fonts it is drawn in, as the [Fonts](#fonts) section says:

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

A change of theme, as when the system turns dark, only draws the scene again.

The widget shows the tooltip of the item under the pointer, and the pointer is a hand over an item with a link. `Hovered` returns the item under the pointer and `Clicked` the item clicked since the last frame, or nil. `Element` returns the widget's element, to place it or give it a background, and `Label` names the diagram for assistive technology, which reads it as an image, in place of the scene's title.

To fit a diagram to the width an element has, read the width from the element's `LayoutBounds`, which holds the frame before's, and lay the diagram out again when it changes, with `MaxWidth`. Parse the text once with `limu.Parse`, and lay out a large diagram on another goroutine, as the [playground](/playground) does.

### Finding the item under the pointer

`limumygo.ItemAt` finds the item of a scene at a point, in the scene's units, without a widget: the one drawn last of those there, and so on top. An item is there when its box holds the point, or its label's box; an edge, and an item drawn only in lines, when one of its lines passes within a slop of the point, or one of its markers holds it. The widget's slop is `limumygo.Slop`, 4 DIPs.

```go
if it := limumygo.ItemAt(sc, scene.Point{X: x, Y: y}, limumygo.Slop); it != nil {
	fmt.Println(it.Role, it.ID)
}
```

## Painting

`limumygo.Paint` draws a scene on a `ui.Painter`, with its top left corner at a point, in DIPs. It draws an element of one's own:

```go
o := &limumygo.Options{Look: look, Fonts: fonts, Scale: 2}
ui.Box(c).Size(float32(sc.Width*2), float32(sc.Height*2)).Draw(func(p *ui.Painter, r ui.Rect) {
	limumygo.Paint(p, sc, r.X, r.Y, o)
})
```

`Options.Scale` is how many DIPs a unit of the scene takes. To zoom or pan a diagram, change the scale and the point it is drawn at; do not draw it under `ui.Painter.Transform`, nor in an element that is transformed. A path too large for one of MyGo's masks is drawn only where the painter shows, which MyGo tells in the window's DIPs and not under a transform, so a transformed diagram can be drawn in part.

MyGo draws the scene as the svg package writes it, with a few differences. An item's opacity is laid on each color it is drawn in, as MyGo's painter has no group to draw at an opacity, so that where an item's parts overlap, the one below shows through. A radius too large for a box is the smaller of half its width and half its height, where SVG makes a narrow box's corners elliptical. MyGo strokes paths with round ends and corners only, so a line with others is drawn by filling its outline.

## Fonts

A diagram is laid out with a measurer and drawn in fonts; when they differ, text does not fit the room layout gave it. `limumygo.Fonts` are both: a `text.Measurer` that measures with the fonts the painter draws in, shaped by the system's text engine as MyGo shapes the text of its elements.

```go
fonts := limumygo.NewFonts("Inter, system-ui", "JetBrains Mono, monospace")
sc, err := limu.Layout(src, &layout.Options{Measurer: fonts})
w := limumygo.View(c, sc, &limumygo.Options{Look: look, Fonts: fonts})
```

`limumygo.DefaultFonts` are the system's font and its monospaced one. A family is a list tried in turn, as `ui.Font` takes it. Bold text is set in the family's bold weight, italic text in its italics, and monospaced text in the mono family. Fonts keep the text they shaped last, so that measuring and drawing a label again costs nothing, and are safe to use from several goroutines.

Text in Chinese, Japanese and Korean is measured in the fonts it is drawn in, falling back to the system's fonts for the scripts a family lacks. The system's text engine puts a little room where Han characters meet Latin letters and digits, and `Fonts` measure that room too.

## Icons and images

limumygo loads no image and follows no link: the host decides, as an app is the one that knows where its images come from and what a link may open.

`Options.DrawIcon` draws an icon by name, as `fa:fa-car`, in the color of the item's text. Layout leaves room in a label only for the icons the host says it can draw, through `layout.Options.Icons`, so give both the same set:

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

`Options.DrawImage` draws an image from its address, in a box: fitted and centered when `fit` is set, and otherwise stretched to fill it. The address is one layout let through, of a scheme `layout.Options.ImageSchemes` allows; loading the image, and keeping it, is the host's. Without `DrawIcon` or `DrawImage`, icons or images are left out.

A clicked item's `Link` is where it leads. Open it, or not, as the app's own links are.

## Drawing of one's own

`Options.Draw` draws each item in place of the default, which `next` draws. The `limumygo.Painter` it is given embeds the `ui.Painter`, places the scene's units in DIPs (`Point`, `Rect`, `Len`, `Path`), and draws the default's pieces: `Item`, `Parts`, `Ends`, `Picture` and `Label`. It is given the item's style settled into colors and measures, a `style.Resolved`, as MyGo draws with colors.

```go
o.Draw = func(p *limumygo.Painter, it *scene.Item, s style.Resolved, next func()) {
	if it.Kind != scene.Node {
		next()
		return
	}
	// A node as a box of its own, with its label as the default draws it.
	p.Fill(p.Rect(it.Box), limumygo.Color(s.Fill, s.Opacity), 8)
	p.Label(it, s)
}
```

[mygo-platinum](https://github.com/firede/mygo-platinum) draws diagrams this way, in the look of the classic Mac OS.

## The editor's language

The package `limumygo/mermaid` is Mermaid's text as a language of MyGo's code editor. It highlights the keywords of the diagram's type, as the line that declares it says:

```go
buf := editor.NewBuffer(src)
buf.SetLanguage(mermaid.Language)
```

It reads a line at a time and does not parse; limu's errors are what say whether the text is right.

## In a browser

A MyGo app runs in a browser, compiled to WebAssembly, and limumygo with it, as the [playground](/playground) does: see MyGo's documentation for building and loading it. There, text is shaped by the browser, and `Fonts` measure it as the browser sets it. limu and limumygo add about 7 MB to the WebAssembly before it is compressed.
