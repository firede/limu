package treeview

import (
	"math"
	"slices"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The scene's roles: "treeview.title" for the title above the tree; "treeview.guide" for the
// line that runs down from a directory to its entries, with a tick to each, its id the
// directory's name; "treeview.directory" and "treeview.file" for an entry, its name its
// label, its classes those of its :::class, its id its name, and for an entry the author
// highlights a box behind its row; "treeview.icon" for an icon Pele draws itself, right after
// its entry, its shape the glyph's name and its id the entry's; and
// "treeview.description" for the text after an entry's ##.
const (
	RoleTitle       scene.Role = "treeview.title"
	RoleGuide       scene.Role = "treeview.guide"
	RoleDirectory   scene.Role = "treeview.directory"
	RoleFile        scene.Role = "treeview.file"
	RoleIcon        scene.Role = "treeview.icon"
	RoleDescription scene.Role = "treeview.description"
)

// The measures of Pele's render.ts: the room between an icon and its name, between the
// widest name and the column of descriptions, how far in from an entry its guide runs, the
// most the guide's corner is rounded by, and how far a highlight reaches past its row.
const (
	iconGap           = 6
	descGap           = 16
	guideInset        = 7
	elbow             = 4
	highlightOverhang = 4
)

// highlight is the opacity of the accent over a highlighted row.
const highlight = 0.14

// row is an entry laid out, Pele's Row: where it starts across, the middle of its icon
// down, its icon, its name decoded, where its name starts, and, laid out tightly, its name
// and description as labels; its top and its height.
type row struct {
	node        *Node
	x, cy       float64
	icon        string
	text        string
	labelX      float64
	highlighted bool
	tight       bool
	label       text.Label
	description *text.Label
	top, height float64
}

// Layout lays a tree view out as a scene, as Pele's renderTreeView does: a row to an entry,
// each indented under its directory, with a guide down from the directory to its entries. A
// tree wider than the width the host has is laid out tightly, its indents narrower and its
// names and descriptions wrapped, when that is narrower.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.Tighten(o, func(tight bool) *scene.Scene { return draw(m, config, o, tight) })
}

// maxWidth is the width the host has, or 320 for none, as Pele takes options.maxWidth.
func maxWidth(o *layout.Options) float64 {
	if o.MaxWidth != 0 {
		return o.MaxWidth
	}
	return 320
}

// draw is Pele's draw.
func draw(m *Model, config diagram.Config, o *layout.Options, tight bool) *scene.Scene {
	size := o.Size()
	measure := o.Measure()
	pad := o.Pad()
	icons := iconConfig(config)
	// The names of the entries are set in the monospace face where the options list files, as
	// Pele's draw sets them where its monospace option does.
	var files text.Face
	if o.Monospace&layout.Files != 0 {
		files = text.Mono
	}

	s := settings(config)
	paddingX, paddingY := s.paddingX, s.paddingY
	maxDepth := 0
	if tight {
		type entry struct {
			node  *Node
			depth int
		}
		var depths []entry
		for _, node := range m.Root.Children {
			depths = append(depths, entry{node, 0})
		}
		for i := 0; i < len(depths); i++ {
			e := depths[i]
			maxDepth = max(maxDepth, e.depth)
			for _, child := range e.node.Children {
				depths = append(depths, entry{child, e.depth + 1})
			}
		}
	}
	step := s.rowIndent + paddingX
	if tight {
		step = min(step, max(2, (maxWidth(o)-120)/float64(max(1, maxDepth))))
	}
	rowHeight := jsmath.Round(size*1.5) + float64(2*paddingY)
	iconSize := size

	titleRoom := 4000.0
	if tight {
		titleRoom = common.TitleRoom(o)
	}
	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: titleRoom, Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}

	// Siblings line up: when one entry of a folder has an icon, the others leave room for one.
	hasIcon := func(list []*Node) bool {
		for _, child := range list {
			if getNodeIcon(child, icons) != "" {
				return true
			}
		}
		return false
	}
	type pend struct {
		node  *Node
		depth int
		slot  bool
	}
	var rows []*row
	rowOf := map[*Node]*row{}
	var pending []pend
	push := func(list []*Node, depth int) {
		slot := hasIcon(list)
		for i := len(list) - 1; i >= 0; i-- {
			pending = append(pending, pend{list[i], depth, slot})
		}
	}
	push(m.Root.Children, 0)
	labelRight := 0.0
	anyHighlight := false
	rowsHeight := 0.0
	for len(pending) > 0 {
		p := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		node := p.node
		x := float64(float64(p.depth) * step)
		name := text.DecodeEntities(node.Name)
		labelX := x
		if p.slot {
			labelX = x + (iconSize + iconGap)
		}
		room := max(40, maxWidth(o)-2*pad-2*highlightOverhang-labelX)
		face := text.Face(0)
		if node.Type == Directory {
			face = text.Bold
		}
		face |= files
		r := &row{node: node, x: x, icon: getNodeIcon(node, icons), text: name, labelX: labelX, tight: tight}
		var width float64
		if tight {
			r.label = common.LiteralLabel(name, measure, size, room, face)
			if node.Description != "" {
				l := common.LiteralLabel(text.DecodeEntities(node.Description), measure, size, room, text.Italic)
				r.description = &l
			}
			width = r.label.Width
		} else {
			width = measure.Width(name, size, face)
		}
		if labelX+width > labelRight {
			labelRight = labelX + width
		}
		r.highlighted = node.CSSClass != "" && slices.Contains(strings.FieldsFunc(node.CSSClass, chars.IsSpaceRune), "highlight")
		if r.highlighted {
			anyHighlight = true
		}
		height := rowHeight
		if tight {
			descHeight := 0.0
			if r.description != nil {
				descHeight = r.description.Height
			}
			height = max(rowHeight, r.label.Height+descHeight+float64(2*paddingY))
			r.cy = rowsHeight + (paddingY + max(iconSize, r.label.Height)/2)
		} else {
			r.cy = rowsHeight + height/2
		}
		r.top, r.height = rowsHeight, height
		rowsHeight += height
		rows = append(rows, r)
		rowOf[node] = r
		push(node.Children, p.depth+1)
	}

	descX := labelRight + descGap
	width := labelRight
	for _, r := range rows {
		if r.node.Description == "" {
			continue
		}
		if r.description != nil {
			width = max(width, r.labelX+r.description.Width)
		} else {
			width = max(width, descX+measure.Width(text.DecodeEntities(r.node.Description), size, text.Italic))
		}
	}
	width = max(width, title.Width)
	inset := float64(pad)
	if anyHighlight {
		inset = pad + highlightOverhang
	}
	top := pad + titleHeight
	totalWidth := math.Ceil(width + float64(2*inset))
	totalHeight := math.Ceil(top + rowsHeight + pad)

	sc := &scene.Scene{
		Type:     "treeView",
		Width:    totalWidth,
		Height:   totalHeight,
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: totalWidth / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}

	guideStyle := style.Style{Stroke: style.Border, Fill: style.None, StrokeWidth: style.N(s.lineThickness)}
	for _, r := range rows {
		children := r.node.Children
		if len(children) == 0 {
			continue
		}
		gx := inset + r.x + guideInset
		tick := max(gx, inset+r.x+step-paddingX)
		last := top + rowOf[children[len(children)-1]].cy
		radius := min(elbow, tick-gx)
		p := new(scene.Path).MoveTo(gx, top+r.top+r.height-2).LineTo(gx, last-radius)
		if radius > 0 {
			p.QuadTo(gx, last, gx+radius, last).LineTo(tick, last)
		}
		for i := 0; i < len(children)-1 && tick > gx; i++ {
			y := top + rowOf[children[i]].cy
			p.MoveTo(gx, y).LineTo(tick, y)
		}
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Mark, Role: RoleGuide, ID: r.node.Name,
			Box:   p.Bounds(),
			Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
			Style: guideStyle,
		})
	}

	for _, r := range rows {
		node := r.node
		cy := top + r.cy
		directory := node.Type == Directory
		role, face := RoleFile, text.Face(0)
		if directory {
			role, face = RoleDirectory, text.Bold
		}
		face |= files
		it := scene.Item{
			Kind: scene.Node, Role: role, ID: node.Name, Class: common.ClassNames(node.CSSClass),
			Box:   scene.Rect{X: inset + r.x, Y: top + r.top, W: totalWidth - inset - (inset + r.x), H: r.height},
			Style: style.Style{Text: style.Fg},
		}
		if r.highlighted {
			x := inset + r.x - highlightOverhang
			box := scene.Rect{X: x, Y: top + r.top + 1, W: totalWidth - inset + highlightOverhang - x, H: r.height - 2}
			it.Parts = []scene.Part{{Box: box, Rounded: true, Paint: scene.FillOnly}}
			it.Style.Fill, it.Style.FillOpacity = style.Accent, style.N(highlight)
		}
		var glyph *scene.Item
		if r.icon != "" {
			box := scene.Rect{X: inset + r.x, Y: cy - iconSize/2, W: iconSize, H: iconSize}
			if common.HostIcon(o, r.icon) {
				it.Picture = &scene.Picture{Icon: text.DecodeEntities(r.icon), Box: box}
			} else {
				name, ok := strings.CutPrefix(r.icon, builtinPack+":")
				if _, known := glyphs[name]; !ok || !known {
					name = "file"
					if directory {
						name = "folder"
					}
				}
				glyph = common.Glyph(RoleIcon, name, node.Name, glyphPaths[name], box, style.Muted)
			}
		}
		if r.tight {
			if !r.label.Empty() {
				l := r.label
				it.Label = &l
				it.At = scene.Point{X: inset + r.labelX, Y: top + r.top + paddingY + l.Height/2}
				it.Align = scene.Start
			}
		} else if r.text != "" {
			l := common.Line(r.text, measure.Width(r.text, size, face), size, face)
			it.Label = l
			it.At = scene.Point{X: inset + r.labelX, Y: cy}
			it.Align = scene.Start
		}
		sc.Items = append(sc.Items, it)
		if glyph != nil {
			sc.Items = append(sc.Items, *glyph)
		}
		description := scene.Item{Kind: scene.Text, Role: RoleDescription, ID: node.Name, Align: scene.Start, Style: style.Style{Text: style.Muted}}
		if r.description != nil {
			labelHeight := r.label.Height
			l := *r.description
			description.Label = &l
			description.At = scene.Point{X: inset + r.labelX, Y: top + r.top + paddingY + labelHeight + l.Height/2}
			sc.Items = append(sc.Items, description)
		} else if node.Description != "" {
			l := common.Line(text.DecodeEntities(node.Description), measure.Width(text.DecodeEntities(node.Description), size, text.Italic), size, text.Italic)
			description.Label = l
			description.At = scene.Point{X: inset + descX, Y: cy}
			sc.Items = append(sc.Items, description)
		}
	}
	return sc
}
