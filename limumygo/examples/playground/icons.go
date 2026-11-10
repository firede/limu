package main

import "github.com/egoist/mygo/ui"

// The icons of the bar over the output, from Lucide (ISC license), drawn in the color of
// the text.
var (
	linkIcon = icon(`<path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/>` +
		`<path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/>`)
	downloadIcon = icon(`<path d="M12 15V3"/><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>` +
		`<path d="m7 10 5 5 5-5"/>`)
)

// icon is an icon of paths on a 24 box, stroked 1.75 wide with round ends, as the website's
// icons are.
func icon(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" ` +
		`fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

// iconButton is a button that shows an icon, 16 DIPs high, in place of a label, which
// names it to assistive technology and to a toolbar's menu of what does not fit.
func iconButton(c *ui.Context, s *ui.SVG, label string) ui.Element {
	b := ui.Button(c, "").Label(label)
	b.Children(func() { ui.Icon(c, s).Size(16, 16) })
	return b
}
