package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/limumygo"
	"github.com/firede/limu/limumygo/mermaid"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/svg"
)

// pad is the room about the diagram in its pane, in DIPs.
const pad = 16

// narrowWidth is the width of a window, in DIPs, under which the editor is above the
// diagram rather than beside it, as on a phone.
const narrowWidth = 640

// The font families the SVG names: the system's, as limumygo draws the diagram in it, on
// a desktop and in a browser alike, and measures it in it.
const (
	fontStack = `system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif`
	monoStack = `ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace`
)

// The views of the output pane.
const (
	viewDiagram = iota
	viewSVG
)

// app is the playground's state, kept across frames and read and changed on the main
// thread. A desktop and a browser show the same app; what differs between them, how a
// file is downloaded, the address a link goes to and whether the appearance is chosen in
// the window, is in desktop.go and web.go.
type app struct {
	// win is the window, nil without one, as with -png and in tests.
	win *mygo.Window
	// post runs a function on the main thread and builds the window again, as win.Update
	// does. When it is nil, text is laid out in the frame that asks for it, as without a
	// window, where no other goroutine can hand a result back.
	post func(func())

	buf  *editor.Buffer
	code *editor.State

	plan       planner
	split      float32 // the percentage of the window the editor takes
	shows      int     // the output pane's view: viewDiagram or viewSVG
	appearance string  // the name of the appearance chosen

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

	// shown, when set, is called once, in the first frame that shows what the text made:
	// a diagram, an error, or nothing for blank text. The page that loads the playground in
	// a browser waits for it.
	shown func()
	// remember, when set, keeps the text and the appearance for the next run, as the
	// desktop's settings do.
	remember func(src, appearance string)
}

// newApp is the playground with a text and an appearance.
func newApp(src, appearance string) *app {
	a := &app{
		buf:        editor.NewBuffer(src),
		plan:       planner{delay: editDelay},
		split:      42,
		appearance: appearance,
	}
	a.buf.SetLanguage(mermaid.Language)
	a.code = &editor.State{Buffer: a.buf, Label: "Mermaid text", Placeholder: "Type a Mermaid diagram, or choose one of the Examples."}
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

// view is the playground: the toolbar; the text and what limu makes of it, side by side,
// or the text above in a narrow window; and the status bar under them.
func (a *app) view(c *ui.Context) {
	ap := a.look(c)
	c.SetTheme(ap.UI)
	a.code.Theme, a.svgCode.Theme = ap.Code, ap.Code
	if a.notice != "" {
		c.Toast(a.notice)
		a.notice = ""
	}
	a.hovered = ""
	root := ui.Column(c).Fill().Background(ap.UI.Background)
	w := root.LayoutBounds().W
	root.Children(func() {
		a.toolbar(c, ap)
		text := func() {
			ui.Column(c).Fill().Children(func() {
				ed := editor.View(c, a.code).Grow(1).FillWidth()
				if a.focusEditor {
					ed.Focus()
					a.focusEditor = false
				}
			})
		}
		output := func() { a.output(c, ap) }
		var split ui.Element
		if w > 0 && w < narrowWidth {
			split = ui.SplitVerticalPercent(c, &a.split, text, output)
		} else {
			split = ui.SplitPercent(c, &a.split, text, output)
		}
		split.Grow(1).FillWidth().MinHeight(0)
		a.statusBar(c, ap)
	})
	if a.shown != nil && (a.scene != nil || a.blank || a.problem != nil) {
		shown := a.shown
		a.shown = nil
		shown()
	}
}

func (a *app) toolbar(c *ui.Context, ap *appearance) {
	t := ap.UI
	ui.Toolbar(c, func() {
		ui.MenuButton(c, "Examples", func(m *ui.Menu) { examplesMenu(m, a.load) })
		if ui.Button(c, "Copy link").Tooltip("Copy a link that opens this diagram in the playground").Clicked() {
			a.share(c)
		}
		ui.MenuButton(c, "Download", func(m *ui.Menu) {
			if m.Item("SVG").Disabled(a.scene == nil).Chosen() {
				a.downloadSVG(ap)
			}
			if m.Item("PNG").Disabled(a.scene == nil).Chosen() {
				a.downloadPNG(ap)
			}
		})
		if appearanceMenu {
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
		}
	}).Label("Playground").FillWidth().Padding(4, 8).Background(t.Background).BorderWidth(0, 0, 1, 0).BorderColor(t.Border)
}

// output is the pane of what limu makes of the text: the diagram, or its SVG.
func (a *app) output(c *ui.Context, ap *appearance) {
	t := ap.UI
	pane := ui.Column(c).Fill()
	// The width the diagram has is the pane's, from the frame before, less the room about
	// it. When it changes, the diagram is laid out again to fit it.
	a.layout(c, math.Floor(float64(pane.LayoutBounds().W))-2*pad)
	pane.Children(func() {
		ui.Row(c).FillWidth().Padding(6, 12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
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
					ui.Text(c, "Type a diagram's text, or choose one of the Examples.").TextColor(t.TextMuted)
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
			w := limumygo.View(c.Key("diagram"), a.scene, &limumygo.Options{Fonts: limumygo.DefaultFonts, Look: ap.Diagram})
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
	ui.Row(c).FillWidth().Padding(5, 12).Gap(16).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).Role(ui.RoleStatus).Children(func() {
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
	a.save(r.src)
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

// load shows a new text, laid out at once: an example's, or a link's.
func (a *app) load(src string) {
	a.buf.Reset(src)
	a.plan.urgent = true
	a.code.GoTo(editor.Pos{})
}

// setAppearance changes the appearance. It only draws the diagram again: the scene stays
// as it was laid out, as nothing of the look moves anything.
func (a *app) setAppearance(name string) {
	a.appearance = name
	a.save(a.buf.String())
}

func (a *app) save(src string) {
	if a.remember != nil {
		a.remember(src, a.appearance)
	}
}

// share copies a link that opens the text in the playground on the website, or in the
// browser's playground, where it runs, or says that the text is too long for one.
func (a *app) share(c *ui.Context) {
	link, ok := shareLink(pageURL(), a.buf.String())
	if !ok {
		c.Toast("This diagram's text is too long for a link")
		return
	}
	c.WriteClipboard(link)
	c.Toast("Copied a link to this diagram")
}

// fileName is the name a download of the diagram offers, after its type.
func (a *app) fileName(ext string) string {
	if a.typ == "" {
		return "diagram" + ext
	}
	return string(a.typ) + ext
}

func (a *app) downloadSVG(ap *appearance) {
	a.download(a.fileName(".svg"), "image/svg+xml", []byte(svg.Render(a.scene, svgOptions(ap))))
}

// downloadPNG draws the diagram as limumygo draws it in the window, at twice its size, on
// its background.
func (a *app) downloadPNG(ap *appearance) {
	var b bytes.Buffer
	if err := png.Encode(&b, scenePNG(a.scene, ap, 2)); err != nil {
		a.notice = "Could not draw the PNG: " + err.Error()
		return
	}
	a.download(a.fileName(".png"), "image/png", b.Bytes())
}

// scenePNG draws a scene in an appearance, on its background, at scale device pixels to a
// DIP, as ui.Render draws a view sized to the scene.
func scenePNG(sc *scene.Scene, ap *appearance, scale float32) *image.RGBA {
	w, h := int(math.Ceil(sc.Width)), int(math.Ceil(sc.Height))
	o := &limumygo.Options{Fonts: limumygo.DefaultFonts, Look: ap.Diagram}
	return ui.Render(func(c *ui.Context) {
		c.SetTheme(ap.UI)
		ui.Box(c).Size(float32(w), float32(h)).Background(ap.UI.Background).Children(func() {
			limumygo.View(c, sc, o)
		})
	}, w, h, scale)
}
