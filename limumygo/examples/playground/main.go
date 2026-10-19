// Command playground tries limu on Mermaid's text: the text on the left, in an editor that
// highlights it, and on the right the diagram limu lays out, drawn by the limumygo package,
// or the SVG the svg package writes of it. The diagram is laid out again a moment after
// each edit, and when the pane's width changes; changing the appearance only draws it
// again. It remembers the text and the appearance between runs.
//
//	go run ./examples/playground
//
// With -png, it renders the window to a PNG without showing it, and exits, from the text of
// a file and in an appearance:
//
//	go run ./examples/playground -png playground.png -src diagram.mmd -appearance "GitHub Dark"
//
// It is an example, not an API: it may change in any way.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// The size of the window, in DIPs.
const width, height = 1200, 760

func main() {
	out := flag.String("png", "", "render the window to this PNG file and exit")
	srcFile := flag.String("src", "", "with -png, the file whose text the window shows; none for the first sample")
	look := flag.String("appearance", appearanceLight, "with -png, the appearance: "+strings.Join(appearanceNames, ", "))
	dark := flag.Bool("dark", false, "with -png, render as the system does in a dark appearance")
	view := flag.String("view", "diagram", "with -png, the output pane's view: diagram or svg")
	route := flag.String("route", "/playground", "with -png, the page shown: /playground, /examples or /examples/<type>")
	w := flag.Int("width", width, "with -png, the width of the window in DIPs")
	h := flag.Int("height", height, "with -png, the height of the window in DIPs")
	scale := flag.Float64("scale", 2, "with -png, device pixels to a DIP")
	flag.Parse()
	if *out != "" {
		if err := renderPNG(*out, *srcFile, *look, *view, *route, *dark, *w, *h, float32(*scale)); err != nil {
			log.Fatal(err)
		}
		return
	}
	runWindow()
}

// renderPNG renders the window, without one, to a PNG file.
func renderPNG(out, srcFile, look, view, route string, dark bool, w, h int, scale float32) error {
	src := firstSample()
	if srcFile != "" {
		var err error
		if src, err = readSource(srcFile); err != nil {
			return err
		}
	}
	if !knownAppearance(look) {
		return fmt.Errorf("no appearance %q: there are %s", look, strings.Join(appearanceNames, ", "))
	}
	a := newApp(src, look, nil)
	switch view {
	case "diagram":
	case "svg":
		a.shows = viewSVG
	default:
		return fmt.Errorf("no view %q: there are diagram and svg", view)
	}
	a.router.Replace(route)
	t := ui.NewTester(a.view, w, h)
	t.SetScale(scale)
	t.SetDark(dark)
	t.Frame()
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	if err := png.Encode(f, t.Image()); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// runWindow shows the playground in a window, from the settings of the last run.
func runWindow() {
	saver := &settingsWriter{path: settingsFile()}
	s, ok := loadSettings(saver.path)
	if !ok {
		s.Source = firstSample()
	}
	a := newApp(s.Source, s.Appearance, saver)
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleAppMenu}, {Role: mygo.RoleEditMenu}, {Role: mygo.RoleWindowMenu}}))
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "limu playground", Width: width, Height: height, MinWidth: 640, MinHeight: 400, Content: ui.View(a.view)})
		a.post = a.win.Update
		// The window follows the system's appearance as it changes, for System.
		mygo.Theme.OnUpdated(func() { a.win.Update(func() {}) })
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
	saver.wait()
}
