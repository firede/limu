package mygoui

import (
	"github.com/egoist/mygo/ui"
	"github.com/firede/limu/scene"
)

// Slop is how far from a line, in DIPs, the pointer is still over it.
const Slop = 4

// A Widget is a scene shown in an element, as View builds it.
type Widget struct {
	el               ui.Element
	sc               *scene.Scene
	k                float64
	hovered, clicked *scene.Item
}

type viewState struct {
	press   scene.Point // where the primary button last went down, in the scene's units
	pressed bool
}

// View builds an element the size of a scene, at the options' Scale, that draws it from its
// top left corner. The item under the pointer shows its Tooltip, and the pointer is a hand
// over one with a Link; Hovered and Clicked tell which item that is, and following a link is
// the host's to do. Assistive technology reads the diagram as an image, named by the
// scene's Title and described by its Desc.
func View(c *ui.Context, sc *scene.Scene, o *Options) *Widget {
	k := o.scale()
	w := &Widget{sc: sc, k: k}
	el := ui.Box(c).Size(float32(sc.Width*k), float32(sc.Height*k)).Role(ui.RoleImage)
	w.el = el
	if sc.Title != "" {
		el.Label(sc.Title)
	}
	if sc.Desc != "" {
		el.Description(sc.Desc)
	}
	st := ui.Local(el, "limu", func() viewState { return viewState{} })
	el.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputPointerDown && ev.Button == 0 {
			st.press, st.pressed = w.at(ev.X, ev.Y), true
		}
		return false
	})
	if x, y, ok := el.PointerPosition(); ok {
		w.hovered = ItemAt(sc, w.at(x, y), Slop/k)
	}
	if el.Clicked() && st.pressed {
		w.clicked = ItemAt(sc, st.press, Slop/k)
	}
	if it := w.hovered; it != nil {
		if it.Link != "" {
			el.Cursor(ui.CursorPointer)
		}
		if it.Tooltip != "" {
			el.Tooltip(it.Tooltip)
		}
	}
	el.Draw(func(p *ui.Painter, r ui.Rect) { Paint(p, sc, r.X, r.Y, o) })
	return w
}

// at is a point of the element, in DIPs, in the scene's units.
func (w *Widget) at(x, y float32) scene.Point {
	return scene.Point{X: float64(x) / w.k, Y: float64(y) / w.k}
}

// Label names the diagram for assistive technology, in place of the scene's Title.
func (w *Widget) Label(s string) *Widget {
	w.el.Label(s)
	return w
}

// Element returns the widget's element, to place it or give it a background.
func (w *Widget) Element() ui.Element { return w.el }

// Hovered returns the item under the pointer, as ItemAt finds it, or nil.
func (w *Widget) Hovered() *scene.Item { return w.hovered }

// Clicked returns the item the primary button clicked since the last frame, or nil when
// there was no click, or none on an item.
func (w *Widget) Clicked() *scene.Item { return w.clicked }
