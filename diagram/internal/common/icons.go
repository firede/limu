package common

import (
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// HostIcon reports whether the host draws the icon of a name, as Pele asks its options'
// icons for one: by the name with its entities decoded.
func HostIcon(o *layout.Options, name string) bool {
	return o.Icons != nil && o.Icons(text.DecodeEntities(name))
}

// GlyphStroke is how wide Pele strokes a glyph it draws itself, in units of its grid.
const GlyphStroke = 1.5

// Glyphs reads the outlines of the glyphs a diagram draws itself, as SVG path data on a 24
// by 24 grid, into paths.
func Glyphs(outlines map[string]string) map[string]*scene.Path {
	out := map[string]*scene.Path{}
	for name, d := range outlines {
		p, err := scene.ParsePath(d)
		if err != nil {
			panic(err)
		}
		out[name] = p
	}
	return out
}

// Glyph is the item of a glyph a diagram draws itself, in the square box: its outline
// scaled from its grid to the box and stroked round in the paint given, as Pele strokes the
// path data it writes in place of an icon the host does not draw. id is that of the item
// the glyph is the icon of.
func Glyph(role scene.Role, name, id string, outline *scene.Path, box scene.Rect, stroke style.Paint) *scene.Item {
	return &scene.Item{
		Kind: scene.Mark, Role: role, Shape: scene.Shape(name), ID: id,
		Box:   box,
		Parts: []scene.Part{{Path: outline.Scaled(box.W/24, box.X, box.Y), Paint: scene.StrokeOnly}},
		Style: style.Style{Stroke: stroke, Fill: style.None, StrokeWidth: style.N(float64(GlyphStroke*box.W) / 24),
			Cap: style.CapRound, Join: style.JoinRound},
	}
}
