package kanban

import (
	"math"
	"strconv"
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/internal/url"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The spacing of a board, as Pele's render.ts has it.
const (
	columnGap = 12
	columnPad = 8
	cardPad   = 10
	cardGap   = 8
	metaGap   = 4
	iconGap   = 6
)

// priorities are the priorities Mermaid names, but Medium, which it gives no color, and the
// style of the stripe each marks a card with, as Pele's PRIORITY has them: the high ones in
// the critical color and the low ones in the muted one, the nearer to Medium the fainter.
// They are not series colors, as a theme may put those in any order. Each stripe ends
// round, as Pele's does.
var priorities = map[string]style.Style{
	"very high": {Stroke: style.Critical, StrokeWidth: style.N(3), Cap: style.CapRound},
	"high":      {Stroke: style.Critical, StrokeWidth: style.N(3), Opacity: style.N(0.5), Cap: style.CapRound},
	"low":       {Stroke: style.Muted, StrokeWidth: style.N(3), Opacity: style.N(0.5), Cap: style.CapRound},
	"very low":  {Stroke: style.Muted, StrokeWidth: style.N(3), Cap: style.CapRound},
}

// fitLabel lays out text for a fixed width, as Pele's fitLabel does. A word too long for a
// line is broken between characters, because a card cannot grow sideways to hold it.
func fitLabel(raw string, markdown bool, o *layout.Options, size, max float64, base text.Face) text.Label {
	measure := o.Measure()
	label := text.Layout(raw, text.Options{Measurer: measure, Size: size, MaxWidth: max, Bounded: true, Face: base, Markdown: markdown, Icons: o.Icons})
	if label.Width <= max {
		return label
	}
	var lines [][]text.Span
	var widths []float64
	widest := 0.0
	var line []text.Span
	width := 0.0
	commit := func() {
		lines = append(lines, line)
		widths = append(widths, width)
		widest = math.Max(widest, width)
		line = nil
		width = 0
	}
	for i := range label.Lines {
		if label.Widths[i] <= max {
			line = label.Lines[i]
			width = label.Widths[i]
			commit()
			continue
		}
		for _, span := range label.Lines[i] {
			if span.Icon != "" {
				if width > 0 && width+span.Width > max {
					commit()
				}
				line = append(line, span)
				width += span.Width
				continue
			}
			var b strings.Builder
			from := width
			for _, ch := range span.Text {
				w := measure.Width(string(ch), size, span.Face)
				if width > 0 && width+w > max {
					if b.Len() > 0 {
						line = append(line, text.Span{Text: b.String(), Face: span.Face, Width: width - from})
					}
					commit()
					b.Reset()
					from = 0
				}
				b.WriteRune(ch)
				width += w
			}
			if b.Len() > 0 {
				line = append(line, text.Span{Text: b.String(), Face: span.Face, Width: width - from})
			}
		}
		commit()
	}
	label.Lines = lines
	label.Widths = widths
	label.Width = widest
	label.Height = float64(len(lines)) * label.LineHeight
	return label
}

// card is a card laid out, Pele's Card.
type card struct {
	item     *Node
	label    text.Label
	ticket   text.Label
	assigned text.Label
	// meta is the height of the rows under the text that hold the ticket and the assignee.
	meta   float64
	height float64
}

// labelOf is a label to put on an item: nil for one with nothing to draw.
func labelOf(l *text.Label) *text.Label {
	if l.Empty() {
		return nil
	}
	return l
}

// Layout lays a board out as a scene, as Pele's renderKanban draws it: a column for each
// section, side by side, each with its title, the count of its cards and its cards in a list.
// A board wider than the width the host has is folded into rows of columns, which then share
// the row between them.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	size := o.Size()
	small := jsmath.Round(size * 0.875)
	line := jsmath.Round(size * 1.5)
	smallLine := jsmath.Round(small * 1.5)
	measure := o.Measure()
	const pad = common.Padding
	conf, _ := config["kanban"].(diagram.Config)
	ownWidth := 200.0
	if w, ok := conf["sectionWidth"].(float64); ok && w > 0 {
		ownWidth = math.Min(math.Max(w, 80), 2000)
	}
	room := math.Inf(1)
	if o.MaxWidth > 0 {
		room = o.MaxWidth - 2*pad
	}
	sections := m.Sections
	count := float64(len(sections))
	fits := float64(count*(ownWidth+columnGap))-columnGap <= room
	var perRow, columnWidth float64
	if fits {
		perRow = math.Max(count, 1)
		columnWidth = ownWidth
	} else {
		perRow = math.Max(1, math.Floor((room+columnGap)/(ownWidth+columnGap)))
		columnWidth = math.Max(math.Min(ownWidth, room), math.Min(math.Floor((room+columnGap)/perRow)-columnGap, ownWidth*2))
	}
	baseURL, _ := conf["ticketBaseUrl"].(string)
	cardWidth := columnWidth - 2*columnPad
	textWidth := cardWidth - 2*cardPad
	glyph := small * 1.25
	has := func(name string) bool { return name != "" && common.HostIcon(o, name) }
	iconRoom := func(name string) float64 {
		if has(name) {
			return glyph + iconGap
		}
		return 0
	}

	titles := make([]text.Label, len(sections))
	countWidths := make([]float64, len(sections))
	cards := make([][]card, len(sections))
	headHeight := line
	bodyHeights := make([]float64, len(sections))
	for i, section := range sections {
		countWidth := measure.Width(strconv.Itoa(len(section.Items)), small, 0)
		reserved := countWidth + 12 + iconRoom(section.Icon)
		titles[i] = fitLabel(section.Label, true, o, size, cardWidth-8-reserved, text.Bold)
		headHeight = math.Max(headHeight, titles[i].Height)
		countWidths[i] = countWidth

		list := make([]card, 0, len(section.Items))
		height := 0.0
		for _, item := range section.Items {
			c := card{
				item:     item,
				label:    fitLabel(item.Label, true, o, small, textWidth-iconRoom(item.Icon), 0),
				ticket:   fitLabel(item.Ticket, false, o, small, textWidth, 0),
				assigned: fitLabel(item.Assigned, false, o, small, textWidth, 0),
			}
			if c.ticket.Width+c.assigned.Width+8 > textWidth {
				c.meta = c.ticket.Height + c.assigned.Height
			} else {
				c.meta = math.Max(c.ticket.Height, c.assigned.Height)
			}
			c.height = 2*cardPad + math.Max(c.label.Height, smallLine)
			if c.meta != 0 {
				c.height += metaGap + c.meta
			}
			list = append(list, c)
			height += c.height + cardGap
		}
		cards[i] = list
		bodyHeights[i] = height
	}
	headHeight += 2 * columnPad
	per := int(perRow)
	var rowHeights, rowTops []float64
	for first, y := 0, 0.0; first < len(sections); first += per {
		tallest := float64(cardGap)
		for index := first; index < min(first+per, len(sections)); index++ {
			tallest = math.Max(tallest, bodyHeights[index])
		}
		rowTops = append(rowTops, y)
		rowHeights = append(rowHeights, headHeight+tallest)
		y += headHeight + tallest + columnGap
	}
	boardHeight := 0.0
	if n := len(rowHeights); n > 0 {
		boardHeight = rowTops[n-1] + rowHeights[n-1]
	}

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: common.TitleRoom(o), Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 16
	}
	boardWidth := math.Max(float64(math.Min(count, perRow)*(columnWidth+columnGap))-columnGap, 0)
	width := math.Max(boardWidth, title.Width)
	left := pad + (width-boardWidth)/2
	edge := columnWidth - columnPad - 4

	sc := &scene.Scene{
		Type:     "kanban",
		Width:    math.Ceil(width + 2*pad),
		Height:   math.Ceil(boardHeight + titleHeight + 2*pad),
		FontSize: size,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: "kanban.title", Label: &title,
			At:    scene.Point{X: sc.Width / 2, Y: pad + title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	for index, section := range sections {
		row := index / per
		gx := left + float64(float64(index%per)*(columnWidth+columnGap))
		gy := pad + titleHeight + rowTops[row]
		box := scene.Rect{X: gx, Y: gy, W: columnWidth, H: rowHeights[row]}
		column := scene.Item{
			Kind: scene.Group, Role: "kanban.column", Shape: "rect", ID: section.ID, Class: common.ClassNames(section.CSSClasses),
			Box:   box,
			Parts: []scene.Part{{Box: box, Rounded: true, Paint: scene.FillOnly}},
			Label: labelOf(&titles[index]),
			At:    scene.Point{X: gx + (columnPad + 4), Y: gy + (columnPad + titles[index].Height/2)},
			Align: scene.Start,
			Style: style.Style{Fill: style.Surface, Text: style.Fg},
		}
		if has(section.Icon) {
			column.Picture = &scene.Picture{
				Icon: text.DecodeEntities(section.Icon),
				Box:  scene.Rect{X: gx + (edge - countWidths[index] - iconGap - glyph), Y: gy + (columnPad + (line-glyph)/2), W: glyph, H: glyph},
			}
		}
		number := text.Layout(strconv.Itoa(len(section.Items)), text.Options{Measurer: measure, Size: small})
		sc.Items = append(sc.Items, column, scene.Item{
			Kind: scene.Text, Role: "kanban.count", ID: section.ID, Label: &number,
			At:    scene.Point{X: gx + edge, Y: gy + (columnPad + line/2)},
			Align: scene.End,
			Style: style.Style{Text: style.Muted},
		})

		y := headHeight
		for k := range cards[index] {
			c := &cards[index][k]
			item := c.item
			cx, cy := gx+columnPad, gy+y
			box := scene.Rect{X: cx, Y: cy, W: cardWidth, H: c.height}
			it := scene.Item{
				Kind: scene.Node, Role: "kanban.card", Shape: "rect", ID: item.ID, Class: common.ClassNames(item.CSSClasses),
				Box:   box,
				Parts: []scene.Part{{Box: box, Rounded: true}},
				Label: labelOf(&c.label),
				At:    scene.Point{X: cx + cardPad, Y: cy + (cardPad + c.label.Height/2)},
				Align: scene.Start,
				Style: style.Style{Fill: style.Bg, Stroke: style.Border, Text: style.Fg},
			}
			if has(item.Icon) {
				it.Picture = &scene.Picture{
					Icon: text.DecodeEntities(item.Icon),
					Box:  scene.Rect{X: cx + (cardWidth - cardPad - glyph), Y: cy + (cardPad + (smallLine-glyph)/2), W: glyph, H: glyph},
				}
			}
			sc.Items = append(sc.Items, it)
			priority := chars.ToLower(chars.Trim(item.Priority))
			if own, ok := priorities[priority]; ok {
				stripe := new(scene.Path).MoveTo(cx+2.5, cy+5).LineTo(cx+2.5, cy+(c.height-5))
				sc.Items = append(sc.Items, scene.Item{
					Kind: scene.Mark, Role: "kanban.priority", Shape: scene.Shape(strings.Replace(priority, " ", "-", 1)), ID: item.ID,
					Box:   stripe.Bounds(),
					Parts: []scene.Part{{Path: stripe, Paint: scene.StrokeOnly}},
					Style: own,
				})
			}
			bottom := c.height - cardPad
			if item.Ticket != "" {
				top := bottom - c.meta
				ticket := scene.Item{
					Kind: scene.Text, Role: "kanban.ticket", ID: item.ID, Label: labelOf(&c.ticket),
					At:    scene.Point{X: cx + cardPad, Y: cy + (top + c.ticket.Height/2)},
					Align: scene.Start,
					Style: style.Style{Text: style.Muted},
				}
				if baseURL != "" {
					address := url.SafeURL(strings.Replace(baseURL, "#TICKET#", item.Ticket, 1))
					if href, ok := url.LinkURL(address, !o.NoLinks, o.LinkSchemes); ok {
						ticket.Link = href
						ticket.Style.Text = style.Accent
					}
				}
				sc.Items = append(sc.Items, ticket)
			}
			if item.Assigned != "" {
				sc.Items = append(sc.Items, scene.Item{
					Kind: scene.Text, Role: "kanban.assigned", ID: item.ID, Label: labelOf(&c.assigned),
					At:    scene.Point{X: cx + (cardWidth - cardPad), Y: cy + (bottom - c.assigned.Height + c.assigned.Height/2)},
					Align: scene.End,
					Style: style.Style{Text: style.Muted},
				})
			}
			y += c.height + cardGap
		}
	}
	return sc
}
