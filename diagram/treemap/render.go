package treemap

import (
	"math"
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/diagram/internal/css"
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

// The room a section keeps for its header, or for a short one, and about its children at
// most, and the height of a line of a leaf's text, in ems, as Pele's HEADER, SHORT_HEADER,
// SIDE and LINE.
const (
	header      = 28
	shortHeader = 18
	sideInset   = 6
	lineSpacing = 1.25
)

// titlePart is a name as a tooltip writes it, as Pele's titlePart gives it: its entities
// decoded and trimmed, without the characters XML does not allow, and trimmed again.
func titlePart(raw string) string {
	s := strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, chars.Trim(text.DecodeEntities(raw)))
	return chars.Trim(s)
}

// leafCount is how many leaves of positive value the tree holds, as Pele's leafCount counts
// them.
func leafCount(roots []*Node) int {
	count := 0
	stack := slices.Clone(roots)
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if node.Section {
			stack = append(stack, node.Children...)
		} else if !math.IsInf(node.Value, 0) && !math.IsNaN(node.Value) && node.Value > 0 {
			count++
		}
	}
	return count
}

// option is a number of the treemap's config if it is at least lo, at most hi, or fallback,
// as Pele's option reads it.
func option(config diagram.Config, key string, fallback, lo, hi float64) float64 {
	section, _ := config["treemap"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v >= lo {
		return min(v, hi)
	}
	return fallback
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
// largest type that fits, or cut short at the size a class sets. Boxes are spaced less the
// more leaves share the area. A section too small for a header draws no box and gives its
// children all of it, and their tooltips name it.
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
	// Gaps and insets shrink as the average area of a leaf does.
	crowd := min(1, max(0, jsmath.Log10((width*height)/float64(max(1, leafCount(m.Roots)))/800)))
	boxes := layoutTreemap(m.Roots, width, height, insets{
		gap:    option(config, "padding", jsmath.Round(2+float64(2*crowd)), 0, 200),
		header: header,
		short:  shortHeader,
		side:   jsmath.Round(3 + float64(3*crowd)),
	})

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
		author := css.Declarations(node.CSS)
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
		// A tooltip names the sections above with no header, as their children cover them.
		own := titlePart(node.Name)
		var path []string
		if own != "" {
			path = append(path, own)
		}
		for p := b.parent; p != nil && !p.headed; p = p.parent {
			if section := titlePart(p.node.Name); section != "" {
				path = slices.Insert(path, 0, section)
			}
		}
		full := strings.Join(path, " › ")
		tooltip := func(drawn string, ok bool) string {
			if full == "" || ok && drawn == name && full == own {
				return ""
			}
			return full
		}
		// A size the author sets is kept, as Pele's inline style overrides the size it writes,
		// so text is cut short instead of made smaller.
		fixed := author.FontSize
		var face text.Face
		if author.Bold {
			face |= text.Bold
		}
		if author.Italic {
			face |= text.Italic
		}
		rect := scene.Rect{X: x, Y: y, W: b.w, H: b.h}
		it := scene.Item{
			Shape: "rect", ID: node.Name, Class: classes(node.Class),
			Box:    rect,
			Parts:  []scene.Part{{Box: rect, Rounded: true}},
			Author: author.Style,
		}
		// The corners of a small box are rounded by a quarter of its shorter side at most,
		// which is less than the theme's 4.
		if limit := min(b.w, b.h) / 4; limit < 4 {
			it.Style.Radius = style.N(limit)
		}

		if node.Section {
			it.Kind, it.Role = scene.Group, RoleSection
			if !b.headed {
				// A section with no header draws no box, so that its children blend into the
				// grid.
				it.Parts[0].Rounded = false
				it.Style = style.Style{Fill: style.None, Stroke: style.None}
				it.Author = style.Style{}
				it.Tooltip = tooltip("", false)
				sections = append(sections, it)
				continue
			}
			// A section is a tint of its color, unless a class gives it a fill of its own.
			it.Style.Fill, it.Style.Stroke, it.Style.Text = color, color, style.Fg
			if !css.FillDeclared(node.CSS) {
				it.Style.FillOpacity = style.N(0.12)
			}
			var shown *scene.Item
			short := b.header < header
			labelSize, valueSize := small, tiny
			if short {
				labelSize = tiny
			}
			if fixed > 0 {
				labelSize, valueSize = fixed, fixed
			}
			room := b.w - 2*sideInset - 4
			valueWidth := 0.0
			if value != "" {
				valueWidth = measure.Width(value, valueSize, face)
			}
			withValue := !short && valueWidth > 0 && room-valueWidth-12 >= 3*labelSize
			fit := room
			if withValue {
				fit = room - valueWidth - 12
			}
			baseline := y + b.header/2 + 1
			drawn, ok := "", false
			if labelSize*lineSpacing < b.header {
				var f common.Fitted
				if f, ok = common.FitText(name, fit, labelSize, labelSize, measure, face|text.Bold); ok {
					drawn = f.Text
					it.Label = common.Line(f.Text, f.Width, f.Size, face|text.Bold)
					it.At = scene.Point{X: x + sideInset + 2, Y: baseline}
					it.Align = scene.Start
				}
			}
			it.Tooltip = tooltip(drawn, ok)
			if withValue && valueSize*lineSpacing <= header {
				shown = &scene.Item{
					Kind: scene.Text, Role: RoleValue, ID: node.Name,
					Label: common.Line(value, valueWidth, valueSize, face), Align: scene.End,
					At:     scene.Point{X: x + b.w - sideInset - 2, Y: baseline},
					Style:  style.Style{Text: style.Muted},
					Author: valueAuthor,
				}
			}
			sections = append(sections, it)
			if shown != nil {
				sections = append(sections, *shown)
			}
			continue
		}

		it.Kind, it.Role = scene.Node, RoleLeaf
		it.Style.Fill, it.Style.Stroke, it.Style.Text = color, style.None, style.Bg
		room := b.w - 8
		smallest := least
		tallest := min(size, math.Floor((b.h-4)/lineSpacing))
		if fixed > 0 {
			smallest = fixed
			tallest = 0
			if fixed*lineSpacing <= b.h-4 {
				tallest = fixed
			}
		}
		var shown *scene.Item
		drawn, ok := "", false
		if tallest >= smallest {
			var f common.Fitted
			if f, ok = common.FitText(name, room, tallest, smallest, measure, face); ok {
				drawn = f.Text
				cx := x + b.w/2
				cy := y + b.h/2
				valueSize := max(least, jsmath.Round(f.Size*0.8))
				if fixed > 0 {
					valueSize = fixed
				}
				lines := (f.Size + valueSize) * lineSpacing
				it.Label = common.Line(f.Text, f.Width, f.Size, face)
				it.At = scene.Point{X: cx, Y: cy}
				valueWidth := 0.0
				if value != "" && lines <= b.h-4 {
					valueWidth = measure.Width(value, valueSize, face)
				}
				if value != "" && lines <= b.h-4 && valueWidth <= room {
					it.At.Y = cy - lines/2 + (f.Size*lineSpacing)/2
					shown = &scene.Item{
						Kind: scene.Text, Role: RoleValue, ID: node.Name,
						Label:  common.Line(value, valueWidth, valueSize, face),
						At:     scene.Point{X: cx, Y: cy + lines/2 - (valueSize*lineSpacing)/2},
						Style:  style.Style{Text: style.Bg},
						Author: valueAuthor,
					}
				}
			}
		}
		it.Tooltip = tooltip(drawn, ok)
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
