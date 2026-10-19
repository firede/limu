//go:build js

package main

import (
	"log"
	"syscall/js"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// appearanceMenu reports whether the toolbar offers the appearances. In a browser it does
// not: the page has the website's switch between light and dark, which sets
// data-mygo-theme on its root, and the playground's appearance is System, which follows it.
const appearanceMenu = false

// pageURL is the address of the playground a link opens: in a browser, the page's own,
// without its fragment.
func pageURL() string {
	loc := js.Global().Get("location")
	return loc.Get("origin").String() + loc.Get("pathname").String()
}

// main shows the playground in the page that loads it, with the text of the URL's
// fragment, or the first example. A link followed to the page while it is open, which
// changes only the fragment, shows its text. Once the first frame shows what the text
// made, it tells the page, which dispatches limu:playground-shown on the document and
// takes away what it showed while loading.
func main() {
	src, ok := linkedSource(js.Global().Get("location").Get("hash").String())
	if !ok {
		src = firstSample()
	}
	a := newApp(src, appearanceSystem)
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "Playground · limu", Content: ui.View(a.view)})
		a.post = a.win.Update
		mygo.Theme.OnUpdated(func() { a.win.Update(func() {}) })
		a.shown = func() {
			doc := js.Global().Get("document")
			doc.Call("dispatchEvent", js.Global().Get("Event").New("limu:playground-shown"))
		}
		js.Global().Call("addEventListener", "hashchange", js.FuncOf(func(js.Value, []js.Value) any {
			if src, ok := linkedSource(js.Global().Get("location").Get("hash").String()); ok {
				go a.post(func() { a.load(src) })
			}
			return nil
		}))
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// download has the browser download a file of data, as a link to it with a download
// attribute does when clicked.
func (a *app) download(name, mediaType string, data []byte) {
	g := js.Global()
	bytes := g.Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(bytes, data)
	blob := g.Get("Blob").New([]any{bytes}, map[string]any{"type": mediaType})
	url := g.Get("URL").Call("createObjectURL", blob)
	link := g.Get("document").Call("createElement", "a")
	link.Set("href", url)
	link.Set("download", name)
	link.Call("click")
	// The browser has the file once the click is handled; the URL is let go after it.
	var revoke js.Func
	revoke = js.FuncOf(func(js.Value, []js.Value) any {
		g.Get("URL").Call("revokeObjectURL", url)
		revoke.Release()
		return nil
	})
	g.Call("setTimeout", revoke, 1000)
}
