// Command playground tries limu on Mermaid's text: the text on the left, in an editor that
// highlights it, and on the right the diagram limu lays out, drawn by the limumygo package,
// or the SVG the svg package writes of it. The diagram is laid out again a moment after
// each edit, and when the pane's width changes; changing between light and dark only draws
// it again. A link copies the text into a URL, and the diagram downloads as SVG or PNG.
//
// It is one program for a desktop and a browser, which differ in desktop.go and web.go. On
// a desktop it remembers the text and the appearance between runs:
//
//	go run ./examples/playground
//
// With -png, it renders the window to a PNG without showing it, and exits, from the text of
// a file and in an appearance:
//
//	go run ./examples/playground -png playground.png -src diagram.mmd -appearance Dark
//
// In a browser it is the website's playground, compiled to WebAssembly by
// website/scripts/playground.mjs; it takes the text from the URL's fragment, as the
// website's links have it, and is light or dark as the page is.
//
// It is an example, not an API: it may change in any way.
package main

// The size of the window, in DIPs.
const width, height = 1200, 760
