package sankey

import (
	"math"
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

// The measures of Pele's sankey render: the room between a bar and its label, what the nodes
// are kept further apart by where their values are shown, about how many lines of label each
// node of the most crowded column has room for, and the most nodes whose labels are kept
// apart.
const (
	labelGap  = 6
	valueGap  = 15
	labelRoom = 1.8
	careful   = 300
)

// The roles of a sankey diagram's items.
const (
	RoleTitle scene.Role = "sankey.title"
	RoleLink  scene.Role = "sankey.link"
	RoleNode  scene.Role = "sankey.node"
	RoleLabel scene.Role = "sankey.label"
	RoleValue scene.Role = "sankey.value"
)

// linkOpacity is the opacity Pele fills the bands with.
const linkOpacity = 0.4

// record is a section of the config that is an object, and nothing otherwise, as Pele's
// isRecord reads it.
func record(value any) diagram.Config {
	c, _ := value.(diagram.Config)
	return c
}

// numberOption is a number the config gives, at most max, or fallback for anything but a
// number of at least least, as Pele's numberOption reads it.
func numberOption(conf diagram.Config, key string, fallback, least, most float64) float64 {
	if v, ok := conf[key].(float64); ok && v >= least {
		return min(v, most)
	}
	return fallback
}

// textOption is a string the config gives, or a finite number written as JavaScript writes
// it, and "" for anything else, as Pele's textOption reads it.
func textOption(conf diagram.Config, key string) string {
	switch v := conf[key].(type) {
	case string:
		return v
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			return chars.NumberString(v)
		}
	}
	return ""
}

// rounded is a value to two decimals, as Mermaid shows values, and a number too large for
// that as it is, as Pele's rounded gives it. The product is rounded before Round takes it,
// as Go would otherwise fuse it with the subtraction in Round once Round is inlined.
func rounded(value float64) float64 {
	if math.Abs(value) < 1e15 {
		return jsmath.Round(float64(value*100)) / 100
	}
	return value
}

// fillStyle is the style of a color the config gives, which Pele writes as an inline style
// over the series color, and false where Pele writes none: for a value that is not a string,
// and for one its resolveStyle drops.
func fillStyle(color any) (style.Style, bool) {
	s, ok := color.(string)
	if !ok {
		return style.Style{}, false
	}
	if css.ShapeCSS([]string{"fill:" + s}) == "" {
		return style.Style{}, false
	}
	// Pele writes the style without the control characters XML does not allow, and that is
	// what a browser reads.
	s = strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, s)
	return css.Declarations([]string{"fill:" + s}).Style, true
}

// Layout lays a sankey diagram out as a scene, as Pele's renderSankey draws it: the chart at
// the width and height its config gives, narrower where the host is, with the bands of the
// links under the bars of the nodes, a label beside each bar, and the title above. A diagram
// whose links run in a circle is an error, as in Pele.
func Layout(m *Model, config diagram.Config, o *layout.Options) (*scene.Scene, error) {
	if o == nil {
		o = &layout.Options{}
	}
	conf := record(config["sankey"])
	var failed error
	natural := numberOption(conf, "width", 600, 1, 10000)
	sc := common.FitWidth(o, natural, 160, func(chartWidth float64, _ bool) *scene.Scene {
		sc, err := draw(m, conf, o, chartWidth, natural)
		if err != nil {
			failed = err
			return &scene.Scene{}
		}
		return sc
	})
	if failed != nil {
		return nil, failed
	}
	return sc, nil
}

// draw lays the diagram out with its chart chartWidth wide, of the natural width its config
// gives, as Pele's draw does.
func draw(m *Model, conf diagram.Config, o *layout.Options, chartWidth, natural float64) (*scene.Scene, error) {
	pad := o.Pad()
	size := o.Size()
	least := jsmath.Round(float64(size * 0.625))
	// Labels shrink with a chart narrowed to fit, as they would if the whole picture were
	// scaled down.
	small := max(least, jsmath.Round(float64(size*0.875*min(1, chartWidth/natural))))
	measure := o.Measure()

	chartHeight := numberOption(conf, "height", 400, 1, 10000)
	nodeWidth := min(numberOption(conf, "nodeWidth", 10, 1, 200), chartWidth)
	showValues := conf["showValues"] != false
	prefix := textOption(conf, "prefix")
	suffix := textOption(conf, "suffix")
	outlined := conf["labelStyle"] == "outlined"
	align := alignment("justify")
	switch a := conf["nodeAlignment"]; a {
	case "left", "right", "center":
		align = alignment(a.(string))
	}
	linkColor := "gradient"
	if s, ok := conf["linkColor"].(string); ok {
		linkColor = s
	}

	extra := 0.0
	if showValues {
		extra = valueGap
	}
	lay := func(labelSize float64) ([]placedNode, []placedLink, error) {
		return layoutSankey(m, layoutOptions{
			width:       chartWidth,
			height:      chartHeight,
			nodeWidth:   nodeWidth,
			nodePadding: numberOption(conf, "nodePadding", 12, 0, 200) + extra,
			labelHeight: labelSize * 1.15,
			alignment:   align,
		})
	}
	nodes, links, err := lay(small)
	if err != nil {
		return nil, err
	}
	// Labels also shrink with the most crowded column, so that each keeps about labelRoom
	// lines.
	rows := map[int]int{}
	fullest := 1
	for _, node := range nodes {
		rows[node.layer]++
		fullest = max(fullest, rows[node.layer])
	}
	if crowded := max(least, jsmath.Round(chartHeight/float64(fullest)/labelRoom)); crowded < small {
		small = crowded
		if nodes, links, err = lay(small); err != nil {
			return nil, err
		}
	}
	lineHeight := small * 1.5

	custom := map[string]style.Style{}
	for id, color := range record(conf["nodeColors"]) {
		if s, ok := fillStyle(color); ok {
			custom[id] = s
		}
	}

	// Labels sit beside the bars, on the side that faces the middle of the chart unless that
	// side is crowded.
	central := 0
	most := 0.0
	for _, node := range nodes {
		if node.value > most {
			most = node.value
			central = node.layer
		}
	}
	minX, maxX := 0.0, chartWidth
	minY, maxY := 0.0, chartHeight
	// placed is where a node's label goes, the whole of its name and value.
	type placed struct {
		value   string
		width   float64
		after   bool
		left, y float64
	}
	placing := make([]*placed, len(nodes))
	for i, node := range nodes {
		name := text.DecodeEntities(m.Nodes[i].ID)
		value := ""
		if showValues {
			value = prefix + chars.NumberString(rounded(node.value)) + suffix
		}
		whole := name
		if value != "" {
			whole = name + " " + value
		}
		width := measure.Width(whole, small, 0)
		after := node.x < chartWidth/2
		if outlined {
			after = node.layer >= central
		}
		x := node.x - labelGap
		left := x - width
		if after {
			x = node.x + nodeWidth + labelGap
			left = x
		}
		y := node.y + node.height/2
		if after {
			minX = min(minX, x)
			maxX = max(maxX, x+width)
		} else {
			minX = min(minX, x-width)
			maxX = max(maxX, x)
		}
		minY = min(minY, y-float64(small*0.75))
		maxY = max(maxY, y+float64(small*0.75))
		placing[i] = &placed{value: value, width: width, after: after, left: left, y: y}
	}
	// A label moves at most one line, so each band of a line's height holds the labels two
	// bands either side of it.
	nearby := map[float64][]*placed{}
	if len(nodes) <= careful {
		for _, l := range placing {
			band := math.Floor(l.y / lineHeight)
			for _, key := range []float64{band - 2, band - 1, band, band + 1, band + 2} {
				nearby[key] = append(nearby[key], l)
			}
		}
	}
	var labels, bars []scene.Item
	var halo style.Style
	if outlined {
		halo = style.Style{Halo: style.Bg, HaloWidth: style.N(3)}
	}
	for i, node := range nodes {
		id := m.Nodes[i].ID
		name := text.DecodeEntities(id)
		height := max(node.height, 1)
		bars = append(bars, scene.Item{
			Kind: scene.Node, Role: RoleNode, Shape: "rect", ID: id,
			Box:    scene.Rect{X: node.x, Y: node.y, W: nodeWidth, H: height},
			Parts:  []scene.Part{{Box: scene.Rect{X: node.x, Y: node.y, W: nodeWidth, H: height}, Rounded: true, Paint: scene.FillOnly}},
			Style:  style.Style{Fill: style.Series(i), Radius: style.N(min(2, height/2))},
			Author: custom[id],
		})

		l := placing[i]
		overlaps := func(left, y float64) bool {
			for _, other := range nearby[math.Floor(y/lineHeight)] {
				if other != l && math.Abs(other.y-y) < lineHeight && left < other.left+other.width+labelGap && left+l.width > other.left-labelGap {
					return true
				}
			}
			return false
		}
		// A label that overlaps another tries the other side, and then a line above or below
		// where both sides are crowded, as Pele's draw places it.
		if len(nodes) <= careful && overlaps(l.left, l.y) {
		place:
			for _, dy := range []float64{0, -lineHeight, lineHeight} {
				for _, after := range []bool{l.after, !l.after} {
					left := node.x - labelGap - l.width
					if after {
						left = node.x + nodeWidth + labelGap
					}
					y := l.y + dy
					top, bottom := y-float64(small*0.75), y+float64(small*0.75)
					if overlaps(left, y) || left < minX || left+l.width > maxX || top < minY || bottom > maxY {
						continue
					}
					// A label stays nearer its own bar than any other in the column, so that it
					// can't name that one.
					own := math.Max(0, math.Max(node.y-y, y-node.y-node.height))
					crosses, nearer := false, false
					for j := range nodes {
						bar := &nodes[j]
						if j != i && left < bar.x+nodeWidth+labelGap && left+l.width > bar.x-labelGap && top < bar.y+bar.height && bottom > bar.y {
							crosses = true
						}
						if j != i && bar.x == node.x && math.Max(0, math.Max(bar.y-y, y-bar.y-bar.height)) <= own {
							nearer = true
						}
					}
					if crosses || nearer {
						continue
					}
					l.left, l.y, l.after = left, y, after
					break place
				}
			}
		}
		value, after, y, textWidth := l.value, l.after, l.y, l.width
		x := node.x - labelGap
		if after {
			x = node.x + nodeWidth + labelGap
		}

		// Pele writes the value in a span of the muted color after the id, in one text
		// anchored at x. A label is of one color, so the id and the value are two: the id
		// starts where the whole starts, and the value ends where it ends.
		idItem := scene.Item{
			Kind: scene.Text, Role: RoleLabel, ID: id, At: scene.Point{X: x, Y: y}, Align: scene.End,
			Label: common.Line(name, measure.Width(name, small, 0), small, 0),
			Style: style.Style{Text: style.Fg, Halo: halo.Halo, HaloWidth: halo.HaloWidth},
		}
		valueItem := scene.Item{
			Kind: scene.Text, Role: RoleValue, ID: id, At: scene.Point{X: x, Y: y}, Align: scene.End,
			Label: common.Line(value, measure.Width(value, small, 0), small, 0),
			Style: style.Style{Text: style.Muted, Halo: halo.Halo, HaloWidth: halo.HaloWidth},
		}
		switch {
		case value == "":
			if after {
				idItem.Align = scene.Start
			}
		case after:
			idItem.Align = scene.Start
			valueItem.At.X = x + textWidth
		default:
			idItem.At.X, idItem.Align = x-textWidth, scene.Start
		}
		labels = append(labels, idItem)
		if value != "" {
			labels = append(labels, valueItem)
		}
	}

	var fixed style.Style
	hasFixed := false
	if linkColor != "gradient" && linkColor != "source" && linkColor != "target" {
		fixed, hasFixed = fillStyle(linkColor)
	}
	var bands []scene.Item
	for _, link := range links {
		// Without definitions there are no gradients, so a gradient link takes its source's
		// color.
		end := link.source
		if linkColor == "target" {
			end = link.target
		}
		half := max(link.thickness, 1) / 2
		x0 := nodes[link.source].x + nodeWidth
		x1 := nodes[link.target].x
		mid := (x0 + x1) / 2
		from := m.Nodes[link.source].ID
		to := m.Nodes[link.target].ID
		path := new(scene.Path).
			MoveTo(x0, link.sourceY-half).
			CubeTo(mid, link.sourceY-half, mid, link.targetY-half, x1, link.targetY-half).
			LineTo(x1, link.targetY+half).
			CubeTo(mid, link.targetY+half, mid, link.sourceY+half, x0, link.sourceY+half).
			Close()
		author := custom[m.Nodes[end].ID]
		if hasFixed {
			author = fixed
		}
		// The bands are drawn in a group with a fill opacity of 0.4, which an author's fill
		// takes too.
		bands = append(bands, scene.Item{
			Kind: scene.Mark, Role: RoleLink, ID: from + "->" + to,
			Box:    path.Bounds(),
			Parts:  []scene.Part{{Path: path, Paint: scene.FillOnly}},
			Style:  style.Style{Fill: style.Series(end), FillOpacity: style.N(linkOpacity)},
			Author: author,
			Tooltip: text.DecodeEntities(from) + " → " + text.DecodeEntities(to) + ": " +
				text.DecodeEntities(prefix) + chars.NumberString(rounded(link.value)) + text.DecodeEntities(suffix),
		})
	}

	title := text.Layout(m.Title, text.Options{Measurer: measure, Size: size, MaxWidth: common.TitleRoom(o), Face: text.Bold, Bounded: true, Icons: o.Icons})
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	inner := max(maxX-minX, title.Width)
	ox := pad - minX + (inner-(maxX-minX))/2
	oy := pad + titleHeight - minY
	width := math.Ceil(inner + 2*pad)
	height := math.Ceil(maxY - minY + titleHeight + 2*pad)

	sc := &scene.Scene{
		Type:     "sankey",
		Width:    width,
		Height:   height,
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
	}
	if !title.Empty() {
		sc.Items = append(sc.Items, scene.Item{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At: scene.Point{X: width / 2, Y: pad + title.Height/2}, Style: style.Style{Text: style.Fg},
		})
	}
	for _, group := range [][]scene.Item{bands, bars, labels} {
		for k := range group {
			group[k].Move(ox, oy)
		}
		sc.Items = append(sc.Items, group...)
	}
	return sc, nil
}
