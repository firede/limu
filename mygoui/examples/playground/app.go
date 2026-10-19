package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/mygoui"
	"github.com/firede/limu/mygoui/mermaid"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/svg"
)

// pad is the room about the diagram in its pane, in DIPs.
const pad = 16

// The font families the SVG names, Primer's stacks of the system's fonts: the diagram is
// measured in the system's font, as mygoui draws it.
const (
	fontStack = `-apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif`
	monoStack = `ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace`
)

// The views of the output pane.
const (
	viewDiagram = iota
	viewSVG
)

// app is the playground's state, kept across frames and read and changed on the main
// thread.
type app struct {
	// win is the window, nil without one, as with -png and in tests.
	win *mygo.Window
	// post runs a function on the main thread and builds the window again, as win.Update
	// does. When it is nil, text is laid out in the frame that asks for it, as without a
	// window, where no other goroutine can hand a result back.
	post func(func())

	router *ui.Router
	buf    *editor.Buffer
	code   *editor.State

	plan       planner
	split      float32 // the percentage of the window the editor takes
	shows      int     // the output pane's view: viewDiagram or viewSVG
	appearance string  // the name of the appearance chosen
	path       string  // the file opened or saved last, "" for none

	// What the newest result showed: the scene of the last text that read, kept while the
	// text is in error, and the error.
	scene   *scene.Scene
	typ     diagram.Type
	took    time.Duration
	blank   bool
	problem *problem

	// The SVG of the scene in an appearance, written when the SVG view shows.
	svgBuf  *editor.Buffer
	svgCode *editor.State
	svgOf   struct {
		sc *scene.Scene
		ap *appearance
	}
	svgDoc string

	hovered     string // the item under the pointer, as the status bar says it
	notice      string // a message for a toast in the next frame
	focusEditor bool   // give the editor the keyboard focus in the next frame

	ex examples // the page of examples

	saver *settingsWriter
}

// newApp is the playground with a text and an appearance, which saves its settings with
// saver, nil for not at all.
func newApp(src, appearance string, saver *settingsWriter) *app {
	a := &app{
		router:     ui.NewRouter("/playground"),
		buf:        editor.NewBuffer(src),
		plan:       planner{delay: editDelay},
		split:      42,
		appearance: appearance,
		saver:      saver,
	}
	a.router.Transition = ui.TransitionNone
	a.buf.SetLanguage(mermaid.Language)
	a.code = &editor.State{Buffer: a.buf, Label: "Mermaid text", Placeholder: "Type a Mermaid diagram, or choose one of the Samples."}
	a.svgBuf = editor.NewBuffer("")
	a.svgBuf.SetLanguage(editor.LanguageFor("diagram.svg"))
	a.svgCode = &editor.State{Buffer: a.svgBuf, Label: "SVG", ReadOnly: true, SoftWrap: true}
	if !knownAppearance(a.appearance) {
		a.appearance = appearanceSystem
	}
	return a
}

// look is the appearance chosen, System following the window's theme as the system sets
// it, which the frame has until SetTheme replaces it.
func (a *app) look(c *ui.Context) *appearance {
	return appearanceNamed(a.appearance, c.Theme().Dark)
}

func (a *app) view(c *ui.Context) {
	ap := a.look(c)
	c.SetTheme(ap.UI)
	a.code.Theme, a.svgCode.Theme = ap.Code, ap.Code
	if a.notice != "" {
		c.Toast(a.notice)
		a.notice = ""
	}
	root := ui.Column(c).Fill().Background(ap.UI.Background)
	// A file dropped on the window, where nothing else takes it, is opened.
	if files := root.DroppedFiles(); len(files) > 0 {
		a.openFile(files[0])
	}
	root.Children(func() {
		a.toolbar(c, ap)
		a.router.View(c, func(r *ui.Route) {
			switch {
			case r.Match("/playground"):
				r.Title("Playground")
				a.playground(c, ap)
			case r.Match("/examples"):
				r.Title("Examples")
				a.examplesPage(c, ap, featured)
			case r.Match("/examples/{type}"):
				r.Title("Examples")
				a.examplesPage(c, ap, r.Param("type"))
			default:
				ui.Text(c, "There is no page at "+r.Path()+".")
			}
		}).Grow(1).FillWidth().MinHeight(0)
	})
}

func (a *app) toolbar(c *ui.Context, ap *appearance) {
	t := ap.UI
	ui.Toolbar(c, func() {
		page := 0
		if strings.HasPrefix(a.router.Path(), "/examples") {
			page = 1
		}
		if segmented(c, ap, &page, "Playground", "Examples").Label("Page").Changed() {
			if page == 0 {
				a.router.Push("/playground")
			} else {
				a.router.Push(examplesPath(a.ex.chosen))
			}
		}
		ui.MenuButton(c, "Samples", func(m *ui.Menu) {
			samplesMenu(m, func(src string) {
				a.load(src, "")
				a.router.Push("/playground")
			})
		})
		if ui.Button(c, "Open…").Clicked() {
			a.openDialog()
		}
		if ui.Button(c, "Save…").Clicked() {
			a.saveDialog()
		}
		ui.MenuButton(c, "Export", func(m *ui.Menu) {
			if m.Item("SVG…").Disabled(a.scene == nil).Chosen() {
				a.exportSVG(ap)
			}
			if m.Item("PNG…").Disabled(a.scene == nil).Chosen() {
				a.exportPNG(ap)
			}
		})
		ui.Spacer(c)
		ui.MenuButton(c, "Appearance", func(m *ui.Menu) {
			for i, name := range appearanceNames {
				if i == 1 {
					m.Separator()
				}
				if m.Item(name).Checked(name == a.appearance).Chosen() {
					a.setAppearance(name)
				}
			}
		})
	}).Label("Playground").FillWidth().Padding(4, 8).Background(ap.Chrome).BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
}

// playground is the page of the playground: the text on the left, what limu makes of it on
// the right, and the status bar under them.
func (a *app) playground(c *ui.Context, ap *appearance) {
	a.hovered = ""
	ui.Column(c).Fill().Children(func() {
		ui.SplitPercent(c, &a.split, func() {
			ui.Column(c).Fill().Children(func() {
				ed := editor.View(c, a.code).Grow(1).FillWidth()
				if a.focusEditor {
					ed.Focus()
					a.focusEditor = false
				}
			})
		}, func() {
			a.output(c, ap)
		}).Grow(1).FillWidth().MinHeight(0)
		a.statusBar(c, ap)
	})
}

// output is the pane of what limu makes of the text: the diagram, or its SVG.
func (a *app) output(c *ui.Context, ap *appearance) {
	t := ap.UI
	pane := ui.Column(c).Fill()
	// The width the diagram has is the pane's, from the frame before, less the room about
	// it. When it changes, the diagram is laid out again to fit it.
	a.layout(c, math.Floor(float64(pane.LayoutBounds().W))-2*pad)
	pane.Children(func() {
		ui.Row(c).FillWidth().Padding(6, 12).Gap(8).Background(ap.Chrome).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			segmented(c, ap, &a.shows, "Diagram", "SVG").Label("Output")
			ui.Box(c).Grow(1)
			if a.shows == viewSVG {
				if ui.Button(c, "Copy").Disabled(a.scene == nil).Clicked() {
					a.ensureSVG(ap)
					c.WriteClipboard(a.svgDoc)
					c.Toast("Copied the SVG")
				}
			}
		})
		switch {
		case a.scene == nil:
			ui.Column(c).Grow(1).FillWidth().Center().Padding(24).Gap(6).Children(func() {
				switch {
				case a.blank:
					ui.Text(c, "Nothing to draw yet").FontSize(t.Rem(1.15)).Bold()
					ui.Text(c, "Type Mermaid text on the left, or choose one of the Samples.").TextColor(t.TextMuted)
				case a.problem != nil:
					ui.Text(c, "No diagram").FontSize(t.Rem(1.15)).Bold()
					ui.Text(c, "The diagram shows here once its text reads.").TextColor(t.TextMuted)
				default:
					ui.Text(c, "Laying out…").TextColor(t.TextMuted)
				}
			})
		case a.shows == viewDiagram:
			a.diagram(c, ap, float32(math.Floor(float64(pane.LayoutBounds().W))))
		default:
			a.ensureSVG(ap)
			editor.View(c, a.svgCode).Grow(1).FillWidth()
		}
	})
}

// diagram shows the scene in a scroll view, centered in the pane while it is narrower, and
// dimmed while the text is in error, as it is then the drawing of an older text.
func (a *app) diagram(c *ui.Context, ap *appearance, paneWidth float32) {
	ui.ScrollBoth(c).Grow(1).FillWidth().Children(func() {
		ui.Row(c).Padding(pad).MinWidth(paneWidth).Justify(ui.Center).AlignItems(ui.Start).Children(func() {
			w := mygoui.View(c.Key("diagram"), a.scene, &mygoui.Options{Fonts: mygoui.DefaultFonts, Look: ap.Diagram})
			w.Label(typeTitle(a.typ) + " diagram")
			if a.problem != nil {
				w.Element().Opacity(0.35)
			}
			if it := w.Hovered(); it != nil {
				a.hovered = describe(it)
			}
		})
	})
}

func (a *app) statusBar(c *ui.Context, ap *appearance) {
	t := ap.UI
	ui.Row(c).FillWidth().Padding(5, 12).Gap(16).Background(ap.Chrome).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Role(ui.RoleStatus).Children(func() {
		switch p := a.problem; {
		case p != nil && p.Located:
			b := ui.Box(c).Role(ui.RoleButton).Label(p.Status).Focusable().Cursor(ui.CursorPointer).Tooltip(p.Detail).Shrink(1)
			b.Children(func() { ui.Text(c, p.Status).FontSize(12).TextColor(t.Danger).MaxLines(1) })
			if b.Clicked() {
				a.code.GoTo(a.buf.Pos(p.Start))
				a.code.Select(p.Start, p.End)
				a.focusEditor = true
			}
		case p != nil:
			ui.Text(c, p.Status).FontSize(12).TextColor(t.Danger).MaxLines(1).Tooltip(p.Detail).Shrink(1)
		case a.blank:
			ui.Text(c, "No diagram").FontSize(12).TextColor(t.TextMuted)
		case a.scene != nil:
			ui.Text(c, typeTitle(a.typ)+" · laid out in "+milliseconds(a.took)).FontSize(12).TextColor(t.TextMuted)
		default:
			ui.Text(c, "Laying out…").FontSize(12).TextColor(t.TextMuted)
		}
		ui.Box(c).Grow(1)
		if a.hovered != "" {
			ui.Text(c, a.hovered).FontSize(12).TextColor(t.TextMuted).MaxLines(1)
		}
	})
}

// milliseconds says how long something took, to a tenth of a millisecond under ten.
func milliseconds(d time.Duration) string {
	ms := float64(d) / float64(time.Millisecond)
	if ms < 10 {
		return fmt.Sprintf("%.1f ms", ms)
	}
	return fmt.Sprintf("%.0f ms", ms)
}

// describe says what an item is, for the status bar: its role and its id.
func describe(it *scene.Item) string {
	if it.ID == "" {
		return string(it.Role)
	}
	return string(it.Role) + " " + it.ID
}

// layout starts laying the text out when the planner says it is time: on another
// goroutine, its result handed back to the main thread through post, or in the frame when
// post is nil. Only the newest job's result is shown.
func (a *app) layout(c *ui.Context, width float64) {
	start, wait := a.plan.next(a.buf.Version(), width, c.Now())
	if wait > 0 {
		c.After(wait)
	}
	if width <= 0 {
		// The pane is not laid out yet: the next frame has its width.
		c.Invalidate()
	}
	if !start {
		return
	}
	j := job{stamp: a.plan.stamp, version: a.buf.Version(), src: a.buf.String(), width: width}
	if a.post == nil {
		a.apply(j.run())
		return
	}
	go func() {
		r := j.run()
		a.post(func() { a.apply(r) })
	}()
}

// apply shows a result, unless a newer job started since its own did.
func (a *app) apply(r result) {
	if !a.plan.current(r.stamp) {
		return
	}
	a.blank, a.problem = r.blank, nil
	var diags []editor.Diagnostic
	switch {
	case r.blank:
		a.scene = nil
	case r.err != nil:
		p := problemOf(r.src, r.err)
		a.problem = &p
		if p.Located {
			diags = []editor.Diagnostic{{Start: p.Start, End: p.End, Severity: editor.SeverityError, Message: p.Detail, Source: "limu"}}
		}
	default:
		a.scene, a.typ, a.took = r.scene, r.typ, r.took
	}
	a.code.SetDiagnostics(r.version, diags)
	a.saveSettings(r.src)
}

// ensureSVG writes the SVG of the scene in the appearance, if it has not yet, and shows it
// in the SVG view, indented.
func (a *app) ensureSVG(ap *appearance) {
	if a.scene == nil || a.svgOf.sc == a.scene && a.svgOf.ap == ap {
		return
	}
	a.svgOf.sc, a.svgOf.ap = a.scene, ap
	a.svgDoc = svg.Render(a.scene, svgOptions(ap))
	a.svgBuf.Reset(formatSVG(a.svgDoc))
}

// svgOptions write a scene as SVG in an appearance, as the diagram is drawn in the window.
func svgOptions(ap *appearance) *svg.Options {
	return &svg.Options{Look: ap.Diagram, FontFamily: fontStack, MonoFamily: monoStack}
}

// load shows a new text, laid out at once: a sample, or a file's.
func (a *app) load(src, path string) {
	a.buf.Reset(src)
	a.plan.urgent = true
	a.path = path
	a.code.GoTo(editor.Pos{})
	if a.win != nil {
		title := "limu playground"
		if path != "" {
			title = filepath.Base(path) + " — " + title
		}
		a.win.SetTitle(title)
	}
}

// setAppearance changes the appearance. It only draws the diagram again: the scene stays
// as it was laid out, as nothing of the look moves anything.
func (a *app) setAppearance(name string) {
	a.appearance = name
	a.saveSettings(a.buf.String())
}

func (a *app) saveSettings(src string) {
	a.saver.save(settings{Source: src, Appearance: a.appearance})
}

// run does the work of a dialog, which waits for the user, on a goroutine, and then
// done with what it found on the main thread. Without a window there are no dialogs.
func (a *app) run(work func() func()) {
	if a.post == nil {
		return
	}
	go func() {
		if done := work(); done != nil {
			a.post(done)
		}
	}()
}

func (a *app) openDialog() {
	a.run(func() func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "Open a diagram", Filters: sourceFilters})
		if err != nil || len(paths) == 0 {
			return nil
		}
		src, err := readSource(paths[0])
		return func() { a.opened(paths[0], src, err) }
	})
}

// openFile opens a file dropped on the window.
func (a *app) openFile(path string) {
	if a.post == nil {
		src, err := readSource(path)
		a.opened(path, src, err)
		return
	}
	a.run(func() func() {
		src, err := readSource(path)
		return func() { a.opened(path, src, err) }
	})
}

func (a *app) opened(path, src string, err error) {
	if err != nil {
		a.notice = "Could not open " + filepath.Base(path) + ": " + err.Error()
		return
	}
	a.load(src, path)
}

// defaultName is the name a dialog that saves offers, with an extension.
func (a *app) defaultName(ext string) string {
	name := "diagram"
	if a.path != "" {
		name = strings.TrimSuffix(filepath.Base(a.path), filepath.Ext(a.path))
	}
	return name + ext
}

func (a *app) saveDialog() {
	src := a.buf.String()
	name := a.defaultName(".mmd")
	if a.path != "" {
		name = filepath.Base(a.path)
	}
	a.run(func() func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.win, Title: "Save the diagram", DefaultPath: name, Filters: sourceFilters})
		if err != nil || path == "" {
			return nil
		}
		err = writeSource(path, src)
		return func() {
			if err != nil {
				a.notice = "Could not save: " + err.Error()
				return
			}
			a.path = path
			a.buf.MarkSaved()
			a.notice = "Saved " + filepath.Base(path)
		}
	})
}

func (a *app) exportSVG(ap *appearance) {
	doc := svg.Render(a.scene, svgOptions(ap))
	a.export(".svg", "SVG", func(path string) error { return os.WriteFile(path, []byte(doc), 0o644) })
}

// exportPNG draws the diagram as mygoui draws it in the window, at twice its size, on its
// background.
func (a *app) exportPNG(ap *appearance) {
	img := scenePNG(a.scene, ap, 2)
	a.export(".png", "PNG", func(path string) error {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	})
}

// export asks where to save a file of a kind, and writes it there with write.
func (a *app) export(ext, kind string, write func(path string) error) {
	name := a.defaultName(ext)
	a.run(func() func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent: a.win, Title: "Export " + kind, DefaultPath: name,
			Filters: []mygo.FileFilter{{Name: kind, Extensions: []string{ext[1:]}}},
		})
		if err != nil || path == "" {
			return nil
		}
		err = write(path)
		return func() {
			if err != nil {
				a.notice = "Could not export: " + err.Error()
				return
			}
			a.notice = "Exported " + filepath.Base(path)
		}
	})
}

// scenePNG draws a scene in an appearance, on its background, at scale device pixels to a
// DIP, as ui.Render draws a view sized to the scene.
func scenePNG(sc *scene.Scene, ap *appearance, scale float32) *image.RGBA {
	w, h := int(math.Ceil(sc.Width)), int(math.Ceil(sc.Height))
	o := &mygoui.Options{Fonts: mygoui.DefaultFonts, Look: ap.Diagram}
	return ui.Render(func(c *ui.Context) {
		c.SetTheme(ap.UI)
		ui.Box(c).Size(float32(w), float32(h)).Background(ap.UI.Background).Children(func() {
			mygoui.View(c, sc, o)
		})
	}, w, h, scale)
}
