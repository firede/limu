// The code the home page shows, in the order it shows it: MyGo first, as drawing natively
// is what limu is for, then SVG.
export const installs = [
  {
    id: 'mygo',
    title: 'MyGo',
    command: 'go get github.com/firede/limu/limumygo',
    note: 'limumygo draws a scene with MyGo’s native UI, on the GPU, and tells which part of it is under the pointer. Lay the diagram out with the fonts it is drawn in.',
    code: `fonts := limumygo.DefaultFonts
sc, err := limu.Layout(src, &layout.Options{Measurer: fonts})
if err != nil {
	return err
}

// In a view: draw the diagram, and see which part was clicked.
w := limumygo.View(c, sc, &limumygo.Options{
	Look:  style.Look{Theme: style.KohuDark()},
	Fonts: fonts,
})
if it := w.Clicked(); it != nil {
	fmt.Println("clicked", it.Role, it.ID)
}`,
  },
  {
    id: 'svg',
    title: 'SVG',
    command: 'go get github.com/firede/limu',
    note: 'The svg package writes a scene as SVG, its colors as CSS variables that a page can set. limu and its svg package need nothing but Go’s standard library.',
    code: `sc, err := limu.Layout(src, nil)
if err != nil {
	return err
}
out := svg.Render(sc, &svg.Options{Look: style.Look{Theme: style.Kohu()}})`,
  },
];
