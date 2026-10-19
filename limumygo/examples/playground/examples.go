package main

import (
	"math"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu"
	"github.com/firede/limu/limumygo"
	"github.com/firede/limu/limumygo/examples/playground/internal/gallery"
	"github.com/firede/limu/scene"
)

// The size of a card on the page of examples, the room about its drawing, and the width
// its drawing is laid out to fit before it is scaled down to the card, as Pele's page of
// examples draws them at a width wider than its cards.
const (
	cardWidth  = 320
	cardHeight = 300
	cardPad    = 12
	cardLayout = 640
)

// featured is the sidebar's choice of the featured examples of every type.
const featured = "featured"

// examples is the state of the page of examples: the type chosen in the sidebar, the grid
// of cards, and the drawings laid out for the cards so far.
type examples struct {
	chosen string
	grid   ui.GridState
	drawn  map[string]*card // by the example's text
}

// A card is an example's drawing, laid out to fit a card, or the error limu gave for it.
// Pending is true while it is laid out off the main thread.
type card struct {
	sc      *scene.Scene
	err     error
	pending bool
}

// entry is an example on a card, with the title of its type.
type entry struct {
	typ string
	ex  gallery.Example
}

// entries are the examples the sidebar's choice shows: the featured ones of every type, or
// all of one.
func entries(chosen string) []entry {
	var out []entry
	for _, t := range gallery.All() {
		if chosen != featured && string(t.Type) != chosen {
			continue
		}
		for _, ex := range t.Examples {
			if chosen == featured && !ex.Featured {
				continue
			}
			out = append(out, entry{t.Title, ex})
		}
	}
	return out
}

// examplesPage is the page of examples: the types in a sidebar, and the examples of the one
// chosen as cards, each of which opens its text in the playground. chosen is the type the
// route names, or featured.
func (a *app) examplesPage(c *ui.Context, ap *appearance, chosen string) {
	t := ap.UI
	if a.ex.drawn == nil {
		a.ex.drawn = map[string]*card{}
	}
	a.ex.chosen = chosen
	ui.Row(c).Fill().Children(func() {
		side := ui.Sidebar(c, &a.ex.chosen, func() {
			ui.SidebarItem(c, featured, nil, "Featured")
			ui.SidebarSection(c, "Diagrams", nil, func() {
				for _, g := range gallery.All() {
					if g.Beta {
						continue
					}
					ui.SidebarItem(c, string(g.Type), nil, g.Title)
				}
			})
			ui.SidebarSection(c, "Beta", nil, func() {
				for _, g := range gallery.All() {
					if g.Beta {
						ui.SidebarItem(c, string(g.Type), nil, g.Title)
					}
				}
			})
		}).Width(220).FillHeight().Background(ap.Chrome).BorderWidth(0, 1, 0, 0).BorderColor(t.Border)
		if side.Changed() && a.ex.chosen != chosen {
			a.router.Replace(examplesPath(a.ex.chosen))
		}
		list := entries(chosen)
		ui.Column(c).Grow(1).FillHeight().MinWidth(0).Children(func() {
			ui.Column(c).FillWidth().Padding(16, 20, 8, 20).Gap(4).Children(func() {
				if chosen == featured {
					ui.Text(c, "Examples").FontSize(t.Rem(1.4)).Bold()
					ui.Text(c, "A few examples of each type of diagram limu reads. Choose a type for all of its examples.").TextColor(t.TextMuted)
					return
				}
				if g, ok := gallery.Find(chosen); ok {
					title := g.Title
					if g.Beta {
						title += " (beta)"
					}
					ui.Text(c, title).FontSize(t.Rem(1.4)).Bold()
					ui.Text(c, g.Description).TextColor(t.TextMuted)
				}
			})
			ui.GridView(c, &a.ex.grid, len(list), cardWidth, cardHeight, func(i int) {
				a.card(c, ap, list[i], chosen == featured)
			}).Grow(1).FillWidth().Padding(8, 12)
		})
	})
}

// examplesPath is the route of the page of examples showing chosen.
func examplesPath(chosen string) string {
	if chosen == featured || chosen == "" {
		return "/examples"
	}
	return "/examples/" + chosen
}

// card shows an example: its title, its type when the page shows several, its drawing
// scaled down to fit, and a button that opens its text in the playground.
func (a *app) card(c *ui.Context, ap *appearance, e entry, showType bool) {
	t := ap.UI
	cd := a.drawing(e.ex.Source)
	ui.Column(c).Fill().Margin(6).Padding(cardPad).Gap(6).Background(t.Background).Radius(t.Radius).
		Border(1, t.Border).Children(func() {
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Children(func() {
				ui.Text(c, e.ex.Title).Bold().SingleLine()
				if showType {
					ui.Text(c, e.typ).TextColor(t.TextMuted).FontSize(t.Rem(0.85)).SingleLine()
				}
			})
			if ui.Button(c, "Open").Tooltip("Open in the playground").Clicked() {
				a.load(e.ex.Source, "")
				a.router.Push("/playground")
			}
		})
		body := ui.Column(c).Grow(1).FillWidth().MinHeight(0).Center().Clip()
		// The drawing is scaled down to the room the card has for it, as the frame before
		// laid the card out, and not up.
		room := body.LayoutBounds()
		body.Children(func() {
			switch {
			case cd.sc != nil:
				w, h := float64(room.W), float64(room.H)
				if w <= 0 || h <= 0 {
					w, h = cardWidth-2*cardPad, cardHeight-2*cardPad-48
				}
				scale := math.Min(1, math.Min(w/cd.sc.Width, h/cd.sc.Height))
				v := limumygo.View(c.Key(e.ex.Source), cd.sc, &limumygo.Options{Fonts: limumygo.DefaultFonts, Look: ap.Diagram, Scale: scale})
				v.Label(e.ex.Title)
			case cd.err != nil:
				ui.Text(c, cd.err.Error()).TextColor(t.Danger).MaxLines(6)
			default:
				ui.Text(c, "Laying out…").TextColor(t.TextMuted)
			}
		})
	})
}

// drawing is the card of an example's text, laid out once to fit a card. Without a window
// it is laid out at once; with one, off the main thread, and the card shows when it is done.
func (a *app) drawing(src string) *card {
	if cd, ok := a.ex.drawn[src]; ok {
		return cd
	}
	cd := &card{pending: true}
	a.ex.drawn[src] = cd
	lay := func() (*scene.Scene, error) {
		return limu.Layout(src, layoutOptions(cardLayout))
	}
	if a.post == nil {
		cd.sc, cd.err = lay()
		cd.pending = false
		return cd
	}
	go func() {
		sc, err := lay()
		a.post(func() { cd.sc, cd.err, cd.pending = sc, err, false })
	}()
	return cd
}
