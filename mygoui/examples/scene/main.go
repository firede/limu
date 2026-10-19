// Command scene shows two scenes laid out by hand in a window, drawn by the mygoui
// package, and tells which item the pointer is over and which was clicked; with -png, it
// renders them to a PNG without one:
//
//	go run ./examples/scene
//	go run ./examples/scene -png scene.png -scale 2
package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/mygoui"
	"github.com/firede/limu/mygoui/internal/samples"
	"github.com/firede/limu/scene"
)

// The size of the window, in DIPs.
const width, height = 760, 400

type app struct {
	scenes  []*scene.Scene
	opts    *mygoui.Options
	status  string
	clicked string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	hovered := ""
	ui.Column(c).Size(width, height).Padding(16).Gap(12).Background(t.Background).Children(func() {
		ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
			for i, sc := range a.scenes {
				w := mygoui.View(c.Key(i), sc, a.opts)
				w.Element().Background(ui.RGB(255, 255, 255)).Radius(6).Border(1, t.Border)
				if it := w.Hovered(); it != nil {
					hovered = describe(it)
				}
				if it := w.Clicked(); it != nil {
					a.clicked = describe(it)
					if it.Link != "" {
						a.clicked += " → " + it.Link
					}
				}
			}
		})
		ui.Textf(c, "Over: %s", or(hovered, "nothing")).TextColor(t.TextMuted)
		ui.Textf(c, "Clicked: %s", or(a.clicked, "nothing yet")).TextColor(t.TextMuted)
	})
}

func describe(it *scene.Item) string {
	s := string(it.Role)
	if it.ID != "" {
		s += " " + it.ID
	}
	if it.Label != nil && len(it.Label.Lines) > 0 {
		var text string
		for _, sp := range it.Label.Lines[0] {
			text += sp.Text
		}
		s += fmt.Sprintf(" %q", text)
	}
	return s
}

func or(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

func main() {
	out := flag.String("png", "", "render the scenes to this PNG file and exit")
	scale := flag.Float64("scale", 1, "the scale of the PNG")
	flag.Parse()
	// The scenes are measured with the fonts they are drawn in.
	fonts := mygoui.DefaultFonts
	a := &app{
		scenes: []*scene.Scene{samples.Flowchart(fonts), samples.Features(fonts)},
		opts:   &mygoui.Options{Fonts: fonts},
	}
	if *out != "" {
		img := ui.Render(a.view, width, height, float32(*scale))
		f, err := os.Create(*out)
		if err != nil {
			log.Fatal(err)
		}
		if err := png.Encode(f, img); err != nil {
			log.Fatal(err)
		}
		if err := f.Close(); err != nil {
			log.Fatal(err)
		}
		return
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "limu scenes",
			Width:   width,
			Height:  height,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
