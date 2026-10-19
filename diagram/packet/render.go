package packet

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

// The scene's roles: "packet.field" for a field's box, with its label, in the order of the
// rows; "packet.bit" for the number of a field's first or last bit above it, aligned at its
// start or end, or centered over a field of one bit; and "packet.title" for the title, which
// is under the fields, as Mermaid puts it.
const (
	RoleField scene.Role = "packet.field"
	RoleBit   scene.Role = "packet.bit"
	RoleTitle scene.Role = "packet.title"
)

// Layout lays a packet diagram out as a scene, as Pele's renderPacket does: across in rows of
// its bits, or, when the host is narrow, down in rows of fewer bits, with labels that wrap.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	return common.TurnToFit(o, func(down bool) *scene.Scene { return draw(m, config, o, down) })
}

// line is a label of one line of text, measured, in a size of type.
func line(s string, width, size float64) text.Label {
	lh := jsmath.Round(size * text.LineHeight)
	return text.Label{
		Lines:  [][]text.Span{{{Text: s, Width: width}}},
		Widths: []float64{width}, Width: width, Height: lh, Size: size, LineHeight: lh,
	}
}

// draw lays the packet out across, or down, as Pele's draw does.
func draw(m *Model, config diagram.Config, o *layout.Options, down bool) *scene.Scene {
	size := o.Size()
	labelSize := jsmath.Round(size * 0.875)
	bitSize := jsmath.Round(size * 0.625)
	measure := o.Measure()
	pad := o.Pad()

	section, _ := config["packet"].(diagram.Config)
	rowHeight := packetOption(config, "rowHeight", 32, 1, 400)
	bitWidth := packetOption(config, "bitWidth", 32, 1, 400)
	paddingX := packetOption(config, "paddingX", 5, 0, 400)
	paddingY := packetOption(config, "paddingY", 5, 0, 400)
	showBits := section["showBits"] != false
	descending := section["bitOrder"] == "descending"
	bitsPerRow := m.BitsPerRow
	rows := m.Rows
	if down {
		count := max(1, math.Floor((o.MaxWidth-2*pad)/min(bitWidth, 18)))
		// A byte is the shortest row. Below that every label is one character to a line, and
		// a long packet is a row to a bit.
		bitsPerRow = min(bitsPerRow, max(8, math.Pow(2, math.Floor(math.Log2(count)))))
		last := 0.0
		if len(m.Rows) > 0 {
			if lastRow := m.Rows[len(m.Rows)-1]; len(lastRow) > 0 {
				last = lastRow[len(lastRow)-1].End
			}
		}
		if math.Ceil((last+1)/bitsPerRow) > 10000 {
			bitsPerRow = m.BitsPerRow
		}
		bitWidth = min(bitWidth, max(18, (o.MaxWidth-2*pad)/bitsPerRow))
		// The model has split each field at its own rows. Its pieces are joined again first,
		// so that a field is only split where one of the new rows ends. A piece counts one
		// bit short.
		piece := func(b *Block) bool { return b.Bits == b.End-b.Start }
		var whole []Block
		for _, row := range m.Rows {
			for _, b := range row {
				if n := len(whole); n > 0 {
					prev := &whole[n-1]
					if piece(prev) && piece(&b) && prev.Label == b.Label && b.Start == prev.End+1 && math.Mod(b.Start, m.BitsPerRow) == 0 {
						prev.End = b.End
						prev.Bits = prev.End - prev.Start
						continue
					}
				}
				whole = append(whole, b)
			}
		}
		rows = nil
		for _, b := range whole {
			for start := b.Start; start <= b.End; {
				end := min(b.End, float64((math.Floor(start/bitsPerRow)+1)*bitsPerRow)-1)
				index := int(math.Floor(start / bitsPerRow))
				for len(rows) <= index {
					rows = append(rows, nil)
				}
				rows[index] = append(rows[index], Block{start, end, b.Bits, b.Label})
				start = end + 1
			}
		}
	}
	var labels [][]text.Label
	if down {
		labels = make([][]text.Label, len(rows))
		for i, row := range rows {
			for _, b := range row {
				room := max(1, float64((b.End-b.Start+1)*bitWidth)-min(paddingX, bitWidth-1)-6)
				labels[i] = append(labels[i], common.LiteralLabel(chars.Trim(text.DecodeEntities(b.Label)), measure, labelSize, room, 0))
			}
		}
	}
	// Bit numbers sit above each row and need room of their own.
	numbers := 0.0
	if showBits {
		numbers = bitSize + 4
	}
	rowHeights := make([]float64, len(rows))
	for i := range rows {
		height := rowHeight
		if down {
			for _, l := range labels[i] {
				height = max(height, l.Height+10)
			}
		}
		rowHeights[i] = height
	}
	tops := make([]float64, len(rows))
	rowsHeight := 0.0
	for i, height := range rowHeights {
		tops[i] = rowsHeight
		rowsHeight += numbers + height + paddingY
	}
	if len(rows) > 0 {
		rowsHeight -= paddingY
	}

	gap := min(paddingX, bitWidth-1)
	width := 0.0
	if len(rows) > 0 {
		width = float64(bitsPerRow*bitWidth) - gap
	}
	titleRoom := 4000.0
	if down {
		titleRoom = common.TitleRoom(o)
	}
	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: titleRoom, Face: text.Bold, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height
		if rowsHeight > 0 {
			titleHeight += 12
		}
	}

	// The pieces of a field split across rows share its label, so each width is fitted once.
	type fitKey struct {
		label string
		room  float64
	}
	type fitResult struct {
		fitted common.Fitted
		ok     bool
	}
	fits := map[fitKey]fitResult{}
	fit := func(label string, room float64) (common.Fitted, bool) {
		k := fitKey{label, room}
		r, seen := fits[k]
		if !seen {
			r.fitted, r.ok = common.FitText(chars.Trim(text.DecodeEntities(label)), room, labelSize, jsmath.Round(size*0.5), measure, 0)
			fits[k] = r
		}
		return r.fitted, r.ok
	}

	left := pad + max(0, (title.Width-width)/2)
	var fields, bits []scene.Item
	for index, row := range rows {
		y := pad + tops[index] + numbers
		rowHeight := rowHeights[index]
		for k, b := range row {
			count := b.End - b.Start + 1
			first := math.Mod(b.Start, bitsPerRow)
			cells := first
			if descending {
				cells = bitsPerRow - first - count
			}
			x := left + float64(cells*bitWidth)
			w := float64(count*bitWidth) - gap
			box := scene.Rect{X: x, Y: y, W: w, H: rowHeight}
			field := scene.Item{
				Kind: scene.Node, Role: RoleField, Shape: "rect",
				ID:    chars.NumberString(b.Start) + "-" + chars.NumberString(b.End),
				Box:   box,
				Parts: []scene.Part{{Box: box, Rounded: true}},
				At:    scene.Point{X: x + w/2, Y: y + rowHeight/2},
				Style: style.Style{Fill: style.Surface, Stroke: style.Border, Text: style.Fg},
			}
			if down {
				l := labels[index][k]
				field.Label = &l
			} else if f, ok := fit(b.Label, w-6); ok && f.Size <= rowHeight {
				l := line(f.Text, f.Width, f.Size)
				field.Label = &l
			}
			fields = append(fields, field)
			if !showBits {
				continue
			}
			leading, trailing := b.Start, b.End
			if descending {
				leading, trailing = b.End, b.Start
			}
			number := func(n, at float64, align scene.Align) {
				s := chars.NumberString(n)
				l := line(s, measure.Width(s, bitSize, 0), bitSize)
				bits = append(bits, scene.Item{
					Kind: scene.Text, Role: RoleBit, Label: &l, Align: align,
					// The number's baseline is 4 above the row.
					At:    scene.Point{X: at, Y: y - 4 - float64(bitSize*0.35)},
					Style: style.Style{Text: style.Muted},
				})
			}
			if count == 1 {
				number(leading, x+w/2, scene.Middle)
			} else {
				number(leading, x, scene.Start)
				number(trailing, x+w, scene.End)
			}
		}
	}

	sc := &scene.Scene{
		Type:     "packet",
		Width:    math.Ceil(max(width, title.Width) + 2*pad),
		Height:   math.Ceil(rowsHeight + titleHeight + 2*pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	sc.Items = append(fields, bits...)
	if !title.Empty() {
		// Mermaid puts a packet's title under it.
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At:    scene.Point{X: sc.Width / 2, Y: sc.Height - pad - title.Height/2},
			Style: style.Style{Text: style.Fg},
		})
	}
	return sc
}
