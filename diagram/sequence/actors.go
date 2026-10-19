package sequence

import (
	"github.com/firede/limu/scene"
	"github.com/firede/limu/text"
)

// The measures of a participant, as Pele's actors.ts has them.
const (
	padX     = 16
	padY     = 10
	glyph    = 34
	glyphGap = 4
	stack    = 5
	capW     = 6
)

// actorKind is the kind a participant is drawn as, Pele's actorKind: one of the kinds it
// knows, or participant.
func actorKind(typ string) string {
	switch typ {
	case "participant", "actor", "boundary", "control", "entity", "database", "collections", "queue":
		return typ
	}
	return "participant"
}

// isFigure reports whether a kind is a figure, which stands above its label; the other kinds
// hold the label inside. It is Pele's isFigure.
func isFigure(kind string) bool {
	switch kind {
	case "actor", "boundary", "control", "entity", "database":
		return true
	}
	return false
}

// actorSize is the size of a participant, Pele's actorSize.
func actorSize(kind string, label *text.Label, minW, minH, paddingX float64) (w, h float64) {
	if isFigure(kind) {
		h = glyph
		if label.Height > 0 {
			h = glyph + (glyphGap + label.Height)
		}
		return max(40, label.Width+8), h
	}
	extra := 0.0
	switch kind {
	case "collections":
		extra = stack
	case "queue":
		extra = 2 * capW
	}
	w = max(minW, label.Width+2*paddingX) + extra
	h = max(minH, label.Height+2*padY)
	if kind == "collections" {
		h += stack
	}
	return w, h
}

// circle is a circle as Pele's <circle> draws it. Pele writes the element inline in its
// figure; the function is limu's own.
func circle(cx, cy, r float64, paint scene.Painting) scene.Part {
	return scene.Part{Path: new(scene.Path).Ellipse(cx, cy, r, r), Paint: paint}
}

// figure is the parts of a participant's figure whose top middle is at cx, top, as Pele's
// figure draws them: shapes filled and stroked, and lines stroked.
func figure(kind string, cx, top float64) []scene.Part {
	y := func(dy float64) float64 { return top + dy }
	line := func(p *scene.Path) scene.Part { return scene.Part{Path: p, Paint: scene.StrokeOnly} }
	switch kind {
	case "actor":
		return []scene.Part{
			circle(cx, y(7), 6, scene.FillAndStroke),
			line(new(scene.Path).MoveTo(cx, y(13)).LineTo(cx, y(24)).
				MoveTo(cx-10, y(17)).LineTo(cx+10, y(17)).
				MoveTo(cx-8, y(34)).LineTo(cx, y(24)).LineTo(cx+8, y(34))),
		}
	case "boundary":
		return []scene.Part{
			line(new(scene.Path).MoveTo(cx-19, y(6)).LineTo(cx-19, y(28)).MoveTo(cx-19, y(17)).LineTo(cx-8, y(17))),
			circle(cx+3, y(17), 11, scene.FillAndStroke),
		}
	case "control":
		return []scene.Part{
			circle(cx, y(21), 11, scene.FillAndStroke),
			line(new(scene.Path).MoveTo(cx+5, y(5)).LineTo(cx-1, y(10)).LineTo(cx+5, y(15))),
		}
	case "entity":
		return []scene.Part{
			circle(cx, y(15), 11, scene.FillAndStroke),
			line(new(scene.Path).MoveTo(cx-11, y(30)).LineTo(cx+11, y(30))),
		}
	}
	// A database: a cylinder, and the front of its lid.
	l, r := cx-14, cx+14
	body := new(scene.Path).MoveTo(l, y(7)).LineTo(l, y(27)).ArcTo(14, 5, 0, false, false, r, y(27)).
		LineTo(r, y(7)).ArcTo(14, 5, 0, false, false, l, y(7)).Close()
	lid := new(scene.Path).MoveTo(l, y(7)).ArcTo(14, 5, 0, false, false, r, y(7))
	return []scene.Part{{Path: body}, line(lid)}
}

// drawActor is a participant whose box has its top middle at cx, top, as Pele's drawActor
// draws it: its parts, and the point its label is centered on.
func drawActor(kind string, cx, top, w, h float64, label *text.Label) (parts []scene.Part, at scene.Point) {
	if isFigure(kind) {
		return figure(kind, cx, top), scene.Point{X: cx, Y: top + glyph + glyphGap + label.Height/2}
	}
	left := cx - w/2
	switch kind {
	case "collections":
		bw, bh := w-stack, h-stack
		return []scene.Part{
			{Box: scene.Rect{X: left + stack, Y: top, W: bw, H: bh}, Rounded: true},
			{Box: scene.Rect{X: left, Y: top + stack, W: bw, H: bh}, Rounded: true},
		}, scene.Point{X: left + bw/2, Y: top + stack + bh/2}
	case "queue":
		l, r, b, ry := left+capW, left+w-capW, top+h, h/2
		body := new(scene.Path).MoveTo(l, top).LineTo(r, top).ArcTo(capW, ry, 0, false, true, r, b).
			LineTo(l, b).ArcTo(capW, ry, 0, false, true, l, top).Close()
		end := new(scene.Path).MoveTo(r, top).ArcTo(capW, ry, 0, false, false, r, b)
		return []scene.Part{{Path: body}, {Path: end, Paint: scene.StrokeOnly}},
			scene.Point{X: cx - capW/2, Y: top + h/2}
	}
	return []scene.Part{{Box: scene.Rect{X: left, Y: top, W: w, H: h}, Rounded: true}}, scene.Point{X: cx, Y: top + h/2}
}
