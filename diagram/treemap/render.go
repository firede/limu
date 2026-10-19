package treemap

import (
	"math"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The scene's roles: "treemap.section" for a section's box, with its name in its header,
// aligned at its start; "treemap.leaf" for a leaf's box, with its name centered in it;
// "treemap.value" for the value of either, aligned at the end of a section's header, or
// centered under a leaf's name; and "treemap.title" for the title above the treemap.
const (
	RoleSection scene.Role = "treemap.section"
	RoleLeaf    scene.Role = "treemap.leaf"
	RoleValue   scene.Role = "treemap.value"
	RoleTitle   scene.Role = "treemap.title"
)

// The room a section keeps for its header and about its children, and the height of a line
// of a leaf's text, in ems, as Pele's HEADER, SIDE and LINE.
const (
	header      = 28
	sideInset   = 6
	lineSpacing = 1.25
)

// option is a number of the treemap's config if it is at least lo, at most hi, or fallback,
// as Pele's option reads it.
func option(config diagram.Config, key string, fallback, lo, hi float64) float64 {
	section, _ := config["treemap"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v >= lo {
		return min(v, hi)
	}
	return fallback
}

// line is a label of one line of text, measured, in a size and face of type.
func line(s string, width, size float64, face text.Face) *text.Label {
	lh := jsmath.Round(size * text.LineHeight)
	return &text.Label{
		Lines:  [][]text.Span{{{Text: s, Face: face, Width: width}}},
		Widths: []float64{width}, Width: width, Height: lh, Size: size, LineHeight: lh,
	}
}

// classes is the class a node's text gives it, as Pele's classNames writes it: a name of a
// class is an ID2 token, which holds no character it would take out.
func classes(class string) []string {
	if class == "" {
		return nil
	}
	return []string{class}
}

// Layout lays a treemap out as a scene, as Pele's renderTreemap does: the tree squarified into
// a rectangle ten times the nodes' width by ten times their height, or the host's width when
// that is less, sections tinted in the color of the top-level branch they are in, with their
// names in their headers, and leaves filled in it, with their names and values in them in the
// largest type that fits.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	tiny := jsmath.Round(size * 0.75)
	least := jsmath.Round(size * 0.5625)
	measure := o.Measure()
	section, _ := config["treemap"].(diagram.Config)
	// The host's padding, or the diagram's own.
	pad := option(config, "diagramPadding", 8, 0, 2000)
	if o.Padding != nil {
		pad = *o.Padding
	}

	// Mermaid sizes the diagram as ten times nodeWidth by ten times nodeHeight.
	width := jsmath.Round(common.CapWidth(o, option(config, "nodeWidth", 64, 1, 400)*10, 200, float64(2*pad)))
	height := jsmath.Round(option(config, "nodeHeight", 40, 1, 400) * 10)
	showValues := section["showValues"] != false
	spec := ","
	if s, ok := section["valueFormat"].(string); ok {
		spec = s
	}
	format := formatter(spec)
	boxes := layoutTreemap(m.Roots, width, height, insets{gap: option(config, "padding", 4, 0, 200), header: header, side: sideInset})

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: common.TitleRoom(o), Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 8
	}
	left := pad + max(0, (title.Width-width)/2)
	top := pad + titleHeight

	var sections, leaves []scene.Item
	for _, b := range boxes {
		node := b.node
		author := style.Declarations(node.CSS)
		// The value's text takes what of the author's style a text does.
		valueAuthor := style.Style{Text: author.Style.Text, Opacity: author.Style.Opacity}
		color := style.Series(b.series)
		x := left + b.x
		y := top + b.y
		name := chars.Trim(text.DecodeEntities(node.Name))
		value := ""
		if showValues {
			value = format(b.value)
		}
		rect := scene.Rect{X: x, Y: y, W: b.w, H: b.h}
		it := scene.Item{
			Shape: "rect", ID: node.Name, Class: classes(node.Class),
			Box:    rect,
			Parts:  []scene.Part{{Box: rect, Rounded: true}},
			Author: author.Style,
		}

		if node.Section {
			// A section is a tint of its color, unless a class gives it a fill of its own.
			it.Kind, it.Role = scene.Group, RoleSection
			it.Style = style.Style{Fill: color, Stroke: color, Text: style.Fg}
			if !style.FillDeclared(node.CSS) {
				it.Style.FillOpacity = style.N(0.12)
			}
			var shown *scene.Item
			if b.h >= header {
				room := b.w - 2*sideInset - 4
				valueWidth := 0.0
				if value != "" {
					valueWidth = measure.Width(value, tiny, 0)
				}
				withValue := valueWidth > 0 && room-valueWidth-12 >= 3*small
				fit := room
				if withValue {
					fit = room - valueWidth - 12
				}
				baseline := y + header/2 + 1
				if f, ok := common.FitText(name, fit, small, small, measure, text.Bold); ok {
					it.Label = line(f.Text, f.Width, f.Size, text.Bold)
					it.At = scene.Point{X: x + sideInset + 2, Y: baseline}
					it.Align = scene.Start
				}
				if withValue {
					shown = &scene.Item{
						Kind: scene.Text, Role: RoleValue, ID: node.Name,
						Label: line(value, valueWidth, tiny, 0), Align: scene.End,
						At:     scene.Point{X: x + b.w - sideInset - 2, Y: baseline},
						Style:  style.Style{Text: style.Muted},
						Author: valueAuthor,
					}
				}
			}
			sections = append(sections, it)
			if shown != nil {
				sections = append(sections, *shown)
			}
			continue
		}

		it.Kind, it.Role = scene.Node, RoleLeaf
		it.Style = style.Style{Fill: color, Stroke: style.None, Text: style.Bg}
		room := b.w - 8
		tallest := min(size, math.Floor((b.h-4)/lineSpacing))
		var shown *scene.Item
		if tallest >= least {
			if f, ok := common.FitText(name, room, tallest, least, measure, 0); ok {
				cx := x + b.w/2
				cy := y + b.h/2
				valueSize := max(least, jsmath.Round(f.Size*0.8))
				lines := (f.Size + valueSize) * lineSpacing
				it.Label = line(f.Text, f.Width, f.Size, 0)
				it.At = scene.Point{X: cx, Y: cy}
				valueWidth := 0.0
				if value != "" && lines <= b.h-4 {
					valueWidth = measure.Width(value, valueSize, 0)
				}
				if value != "" && lines <= b.h-4 && valueWidth <= room {
					it.At.Y = cy - lines/2 + (f.Size*lineSpacing)/2
					shown = &scene.Item{
						Kind: scene.Text, Role: RoleValue, ID: node.Name,
						Label:  line(value, valueWidth, valueSize, 0),
						At:     scene.Point{X: cx, Y: cy + lines/2 - (valueSize*lineSpacing)/2},
						Style:  style.Style{Text: style.Bg},
						Author: valueAuthor,
					}
				}
			}
		}
		leaves = append(leaves, it)
		if shown != nil {
			leaves = append(leaves, *shown)
		}
	}

	sc := &scene.Scene{
		Type:     "treemap",
		Width:    math.Ceil(max(width, title.Width) + float64(2*pad)),
		Height:   math.Ceil(height + titleHeight + float64(2*pad)),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: sc.Width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	sc.Items = append(sc.Items, sections...)
	sc.Items = append(sc.Items, leaves...)
	return sc
}
