//go:build !js

package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// appearanceMenu reports whether the toolbar offers the appearances. A desktop offers
// them; the page around the playground in a browser has its own switch.
const appearanceMenu = true

// pageURL is the address of the playground a link opens: on a desktop, the website's.
func pageURL() string { return siteURL }

func main() {
	out := flag.String("png", "", "render the window to this PNG file and exit")
	srcFile := flag.String("src", "", "with -png, the file whose text the window shows; none for the first example")
	look := flag.String("appearance", appearanceLight, "with -png, the appearance: "+strings.Join(appearanceNames, ", "))
	dark := flag.Bool("dark", false, "with -png, render as the system does in a dark appearance")
	view := flag.String("view", "diagram", "with -png, the output pane's view: diagram or svg")
	w := flag.Int("width", width, "with -png, the width of the window in DIPs")
	h := flag.Int("height", height, "with -png, the height of the window in DIPs")
	scale := flag.Float64("scale", 2, "with -png, device pixels to a DIP")
	flag.Parse()
	if *out != "" {
		if err := renderPNG(*out, *srcFile, *look, *view, *dark, *w, *h, float32(*scale)); err != nil {
			log.Fatal(err)
		}
		return
	}
	runWindow()
}

// renderPNG renders the window, without one, to a PNG file.
func renderPNG(out, srcFile, look, view string, dark bool, w, h int, scale float32) error {
	src := firstSample()
	if srcFile != "" {
		b, err := os.ReadFile(srcFile)
		if err != nil {
			return err
		}
		// Without the byte order mark an editor may have written before the text.
		src = strings.TrimPrefix(string(b), "\uFEFF")
	}
	if !knownAppearance(look) {
		return fmt.Errorf("no appearance %q: there are %s", look, strings.Join(appearanceNames, ", "))
	}
	a := newApp(src, look)
	switch view {
	case "diagram":
	case "svg":
		a.shows = viewSVG
	default:
		return fmt.Errorf("no view %q: there are diagram and svg", view)
	}
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
	a := newApp(s.Source, s.Appearance)
	a.remember = func(src, appearance string) { saver.save(settings{Source: src, Appearance: appearance}) }
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

// download asks where to save a file, and writes data there. The dialog waits for the
// user on a goroutine of its own, and what came of it is said on the main thread. Without
// a window there are no dialogs, and nothing is saved.
func (a *app) download(name, mediaType string, data []byte) {
	if a.post == nil {
		return
	}
	ext := filepath.Ext(name)
	kind := strings.ToUpper(ext[1:])
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent: a.win, Title: "Download " + kind, DefaultPath: name,
			Filters: []mygo.FileFilter{{Name: kind, Extensions: []string{ext[1:]}}},
		})
		if err != nil || path == "" {
			return
		}
		err = os.WriteFile(path, data, 0o644)
		a.post(func() {
			if err != nil {
				a.notice = "Could not save: " + err.Error()
				return
			}
			a.notice = "Saved " + filepath.Base(path)
		})
	}()
}
