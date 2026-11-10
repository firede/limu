//go:build js

package main

import (
	"log"
	"strconv"
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
	if split, ok := storedSplit(); ok {
		a.split, a.keptSplit = split, split
	}
	a.rememberSplit = storeSplit
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "Playground · limu", Content: ui.View(a.view)})
		a.post = a.win.Update
		mygo.Theme.OnUpdated(func() { a.win.Update(func() {}) })
		// The page's address is kept at the text, as a link to it, so that the link to the
		// Rust playground in the page's bar, which carries the fragment, opens the same
		// text. A text too long for a link leaves the address without one.
		a.remember = func(src, _ string) {
			link, ok := shareLink(pageURL(), src)
			if !ok {
				link = pageURL()
			}
			js.Global().Get("history").Call("replaceState", js.Null(), "", link)
		}
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

// splitKey is where the browser keeps the split, a percentage written as a number. The
// website's Rust playground keeps its split there too, so that the panes are as wide in one
// as they were left in the other.
const splitKey = "limu-playground-split"

// storedSplit is the split the browser keeps, when it keeps one that leaves both panes.
// A browser that refuses its storage throws, and then keeps none.
func storedSplit() (split float32, ok bool) {
	defer func() {
		if recover() != nil {
			split, ok = 0, false
		}
	}()
	v := js.Global().Get("localStorage").Call("getItem", splitKey)
	if v.Type() != js.TypeString {
		return 0, false
	}
	f, err := strconv.ParseFloat(v.String(), 32)
	if err != nil || !(f >= minSplit && f <= maxSplit) {
		return 0, false
	}
	return float32(f), true
}

// storeSplit keeps the split in the browser, which may refuse.
func storeSplit(split float32) {
	defer func() { _ = recover() }()
	js.Global().Get("localStorage").Call("setItem", splitKey, strconv.FormatFloat(float64(split), 'f', 1, 32))
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
