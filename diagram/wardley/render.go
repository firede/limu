package wardley

import (
	"math"
	"slices"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// The scene's roles, in the order Pele draws them: "wardley.title" for the title;
// "wardley.stage" for the dashed line between two stages of evolution, "wardley.grid" for
// each pair of grid lines, and "wardley.axis" for the axes; "wardley.stage-label" for the
// name of each stage and "wardley.axis-label" for the titles of the axes, the one of the
// visibility axis turned; "wardley.pipeline" for the box of a pipeline, its id its parent's,
// and "wardley.pipeline-line" for the dashed line through its components; "wardley.link"
// for a link, its id its ends' ids joined by a hyphen, and "wardley.link-label" for its
// label; "wardley.trend" for the dashed arrow to where a component is evolving and
// "wardley.trend-target" for the circle there, their ids the component's; for each node, a
// Node of its class — "wardley.anchor", "wardley.component", "wardley.pipeline-parent" or
// "wardley.pipeline-component" — with its shape and its sourcing strategy as its tag, a
// "wardley.inertia" bar when it resists change, and "wardley.label" for its label, all with
// the node's id; "wardley.annotation" for each numbered circle, "wardley.annotations" for the
// box that lists their texts and "wardley.annotation-text" for each line of it;
// "wardley.note" for each note; and "wardley.accelerator" and "wardley.deaccelerator" for
// the arrows of forces, their ids their names.
const (
	RoleTitle             scene.Role = "wardley.title"
	RoleStage             scene.Role = "wardley.stage"
	RoleGrid              scene.Role = "wardley.grid"
	RoleAxis              scene.Role = "wardley.axis"
	RoleStageLabel        scene.Role = "wardley.stage-label"
	RoleAxisLabel         scene.Role = "wardley.axis-label"
	RolePipeline          scene.Role = "wardley.pipeline"
	RolePipelineLine      scene.Role = "wardley.pipeline-line"
	RoleLink              scene.Role = "wardley.link"
	RoleLinkLabel         scene.Role = "wardley.link-label"
	RoleTrend             scene.Role = "wardley.trend"
	RoleTrendTarget       scene.Role = "wardley.trend-target"
	RoleAnchor            scene.Role = "wardley.anchor"
	RoleComponent         scene.Role = "wardley.component"
	RolePipelineParent    scene.Role = "wardley.pipeline-parent"
	RolePipelineComponent scene.Role = "wardley.pipeline-component"
	RoleInertia           scene.Role = "wardley.inertia"
	RoleLabel             scene.Role = "wardley.label"
	RoleAnnotation        scene.Role = "wardley.annotation"
	RoleAnnotations       scene.Role = "wardley.annotations"
	RoleAnnotationText    scene.Role = "wardley.annotation-text"
	RoleNote              scene.Role = "wardley.note"
	RoleAccelerator       scene.Role = "wardley.accelerator"
	RoleDeaccelerator     scene.Role = "wardley.deaccelerator"
)

// The measures of Pele's wardley render. Mermaid's canvas has canvasMargin of room around
// the plot on every side.
const (
	canvasMargin = 48
	radius       = 6
	square       = 10
	labelOffset  = 8
	pipelinePad  = 15
	forceWidth   = 60
	forceHeight  = 30
	forceHead    = 20
)

// Measures Pele works out with floating point, which Go would work out exactly as constants:
// the triangle and the dots of a market, the reach of a pipeline's parent, and the angle of
// the triangle's lower corners.
var (
	tri         = float64(1.2) * radius
	dot         = float64(0.7) * radius
	squareReach = square / float64(math.Sqrt2)
	sixthPi     = float64(math.Pi) / 6
)

var defaultStages = []string{"Genesis", "Custom Built", "Product", "Commodity"}

// arrowPoint is the marker of links and trends, and arrowTrim how far a line stops short of
// it, as Pele's markerTrim gives it.
const arrowPoint scene.MarkerType = "arrow_point"

var arrowTrim = arrowPoint.Trim()

// placed is a node and where it is drawn, Pele's Placed.
type placed struct {
	x, y float64
	node *Node
}

// setting is a number of the map's own section of the config, between lo and hi, or
// fallback, as Pele's setting reads it.
func setting(config diagram.Config, key string, fallback, lo, hi float64) float64 {
	section, _ := config["wardley-beta"].(diagram.Config)
	if v, ok := section[key].(float64); ok && v >= lo && v <= hi {
		return v
	}
	return fallback
}

func clamp(value, lo, hi float64) float64 {
	switch {
	case value < lo:
		return lo
	case value > hi:
		return hi
	}
	return value
}

// dimension is the width or the height of the plot, as Pele's dimension works it out: the
// text's size, or the config's, or the default, held between 200 and 10,000, less the
// canvas's margins.
func dimension(config diagram.Config, given *float64, key string, fallback float64) float64 {
	value := 0.0
	if given != nil && !math.IsInf(*given, 0) && !math.IsNaN(*given) {
		value = *given
	} else {
		value = setting(config, key, fallback, 1, 1e6)
	}
	return clamp(value, 200, 10000) - 2*canvasMargin
}

// Layout lays a Wardley map out as a scene, as Pele's renderWardley draws it: the plot as
// wide as the map asks, or narrower to fit the host, with its stages under it and its axes'
// titles, and on it the pipelines, links, trends, nodes, annotations, notes and forces.
func Layout(m *Model, config diagram.Config, o *layout.Options) *scene.Scene {
	if o == nil {
		o = &layout.Options{}
	}
	var width, height *float64
	if m.Size != nil {
		width, height = &m.Size.Width, &m.Size.Height
	}
	natural := dimension(config, width, "width", 900)
	return common.FitWidth(o, natural, 160, func(plotW float64, tight bool) *scene.Scene {
		return draw(m, config, o, plotW, natural, tight, height)
	})
}

// draw lays the map out with a plot plotW wide, as Pele's draw does.
func draw(m *Model, config diagram.Config, o *layout.Options, plotW, naturalW float64, tight bool, givenHeight *float64) *scene.Scene {
	size := o.Size()
	small := jsmath.Round(size * 0.8125)
	tiny := jsmath.Round(size * 0.6875)
	measure := o.Measure()
	pad := o.Pad()

	// A narrower map is not as much shorter, so that components keep some room between them.
	plotH := float64(dimension(config, givenHeight, "height", 600) * max(0.75, plotW/naturalW))
	section, _ := config["wardley-beta"].(diagram.Config)
	showGrid := section["showGrid"] == true

	minX, minY, maxX, maxY := 0.0, 0.0, plotW, plotH
	grow := func(x0, y0, x1, y1 float64) {
		if x0 < minX {
			minX = x0
		}
		if y0 < minY {
			minY = y0
		}
		if x1 > maxX {
			maxX = x1
		}
		if y1 > maxY {
			maxY = y1
		}
	}
	// The items of the plot, about its corner; they are moved onto the page at the end.
	var items []scene.Item
	labelItem := func(kind scene.Kind, role scene.Role, id string, l text.Label, at scene.Point, align scene.Align, st style.Style) {
		if l.Empty() {
			return
		}
		items = append(items, scene.Item{Kind: kind, Role: role, ID: id, Label: &l, At: at, Align: align, Style: st})
	}
	// centered places a label centered on cx, cy, as Pele's centered does.
	centered := func(role scene.Role, id string, l text.Label, cx, cy float64, st style.Style) {
		grow(cx-l.Width/2, cy-l.Height/2, cx+l.Width/2, cy+l.Height/2)
		labelItem(scene.Text, role, id, l, scene.Point{X: cx, Y: cy}, scene.Middle, st)
	}
	// starting places a label with its lines starting at x and its first baseline at y, as
	// Pele's starting does. The scene centers it down on its point, which is then where Pele
	// puts the middle of its lines.
	starting := func(role scene.Role, id string, l text.Label, x, y float64, st style.Style) {
		grow(x, y-l.Size, x+l.Width, y-l.Size+l.Height+4)
		at := y - float64(l.Size*0.35) + l.Height/2 - l.LineHeight/2
		labelItem(scene.Text, role, id, l, scene.Point{X: x, Y: at}, scene.Start, st)
	}
	layoutText := func(raw string, fontSize, maxWidth float64, face text.Face) text.Label {
		return text.Layout(raw, text.Options{Measurer: measure, Size: fontSize, MaxWidth: maxWidth, Face: face, Icons: o.Icons})
	}
	// The width Pele's text gives a label when it is not told one.
	plain := 4000.0
	if tight {
		plain = max(64, min(120, plotW/2))
	}
	projectX := func(value float64) float64 { return float64((value / 100) * plotW) }
	projectY := func(value float64) float64 { return plotH - float64((value/100)*plotH) }
	line := func(role scene.Role, p *scene.Path, st style.Style) scene.Item {
		return scene.Item{Kind: scene.Mark, Role: role, Box: p.Bounds(), Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}}, Style: st}
	}
	circle := func(cx, cy, r float64, paint scene.Painting) scene.Part {
		return scene.Part{Path: new(scene.Path).Ellipse(cx, cy, r, r), Paint: paint}
	}
	circleBox := func(cx, cy, r float64) scene.Rect { return scene.Rect{X: cx - r, Y: cy - r, W: 2 * r, H: 2 * r} }

	stages := m.Axes.Stages
	if len(stages) == 0 {
		stages = defaultStages
	}
	boundaries := m.Axes.StageBoundaries
	custom := boundaries != nil && len(boundaries) == len(stages)
	var stageLabels []scene.Item
	stageBottom := plotH
	previous := 0.0
	for index, stage := range stages {
		start := float64(index) / float64(len(stages))
		end := float64(index+1) / float64(len(stages))
		if custom {
			start, end = previous, clamp(boundaries[index], 0, 1)
		}
		previous = end
		if index > 0 {
			p := new(scene.Path).MoveTo(start*plotW, 0).LineTo(start*plotW, plotH)
			items = append(items, line(RoleStage, p, style.Style{Stroke: style.Border, Dash: style.Dashes(4, 4)}))
		}
		l := layoutText(stage, small, max(48, float64(math.Abs(end-start)*plotW)-12), 0)
		cx, cy := float64(((start+end)/2)*plotW), plotH+8+l.Height/2
		grow(cx-l.Width/2, cy-l.Height/2, cx+l.Width/2, cy+l.Height/2)
		if !l.Empty() {
			stageLabels = append(stageLabels, scene.Item{Kind: scene.Text, Role: RoleStageLabel, Label: &l, At: scene.Point{X: cx, Y: cy},
				Style: style.Style{Text: style.Muted}})
		}
		stageBottom = max(stageBottom, plotH+8+l.Height)
	}
	if showGrid {
		for i := 1.0; i < 4; i++ {
			x, y := (plotW*i)/4, (plotH*i)/4
			p := new(scene.Path).MoveTo(x, 0).LineTo(x, plotH).MoveTo(0, y).LineTo(plotW, y)
			items = append(items, line(RoleGrid, p, style.Style{Stroke: style.Border, Dash: style.Dashes(2, 6)}))
		}
	}
	items = append(items, line(RoleAxis, new(scene.Path).MoveTo(0, 0).LineTo(0, plotH).LineTo(plotW, plotH), style.Style{Stroke: style.Line}))
	items = append(items, stageLabels...)

	xLabel, yLabel := "Evolution", "Visibility"
	if m.Axes.XLabel != nil {
		xLabel = *m.Axes.XLabel
	}
	if m.Axes.YLabel != nil {
		yLabel = *m.Axes.YLabel
	}
	// The titles of the axes are measured in their own weight, and drawn in the weight of
	// headings.
	xTitle := heading(layoutText(xLabel, small, plain, 0))
	yTitle := heading(layoutText(yLabel, small, plain, 0))
	centered(RoleAxisLabel, "", xTitle, plotW/2, stageBottom+6+xTitle.Height/2, style.Style{})
	if !yTitle.Empty() {
		items = append(items, scene.Item{Kind: scene.Text, Role: RoleAxisLabel, Label: &yTitle,
			At: scene.Point{X: -16, Y: plotH / 2}, Rotate: -90})
	}
	grow(-16-yTitle.Height/2, 0, 0, 0)
	legendTop := stageBottom + 6 + xTitle.Height + 10

	positions := make([]*placed, len(m.Nodes))
	byID := make(map[string]*placed, len(m.Nodes))
	for i, node := range m.Nodes {
		positions[i] = &placed{projectX(node.X), projectY(node.Y), node}
		byID[node.ID] = positions[i]
	}

	// Pipelines: a box around the components, with the parent as a square on its top edge.
	members := map[string]map[string]bool{}
	for _, pipeline := range m.Pipelines {
		set := map[string]bool{}
		for _, id := range pipeline.ComponentIDs {
			set[id] = true
		}
		members[pipeline.NodeID] = set
		var parts []*placed
		for _, id := range pipeline.ComponentIDs {
			if part := byID[id]; part != nil {
				parts = append(parts, part)
			}
		}
		if len(parts) == 0 {
			continue
		}
		slices.SortStableFunc(parts, func(a, b *placed) int { return order(a.x - b.x) })
		left, right := parts[0].x, parts[len(parts)-1].x
		y := parts[0].y
		top := y - 2*radius
		if parent := byID[pipeline.NodeID]; parent != nil {
			parent.x = (left + right) / 2
			parent.y = top - square/6.0
		}
		grow(left-pipelinePad, top, right+pipelinePad, top+4*radius)
		box := scene.Rect{X: left - pipelinePad, Y: top, W: right - left + 2*pipelinePad, H: 4 * radius}
		items = append(items, scene.Item{
			Kind: scene.Group, Role: RolePipeline, ID: pipeline.NodeID, Box: box,
			Parts: []scene.Part{{Box: box, Rounded: true}},
			Style: style.Style{Fill: style.Surface, Stroke: style.Border},
		})
		if len(parts) > 1 {
			p := new(scene.Path).MoveTo(left, y).LineTo(right, y)
			it := line(RolePipelineLine, p, style.Style{Stroke: style.Line, Dash: style.Dashes(4, 4)})
			it.ID = pipeline.NodeID
			items = append(items, it)
		}
	}

	// How far from a node's center a line to it stops.
	anchorLabels := map[*Node]text.Label{}
	anchorWidth := 4000.0
	if tight {
		anchorWidth = max(64, plotW/2)
	}
	anchorLabel := func(node *Node) text.Label {
		l, ok := anchorLabels[node]
		if !ok {
			l = layoutText(node.Label, small, anchorWidth, text.Bold)
			anchorLabels[node] = l
		}
		return l
	}
	reach := func(at *placed, ux, uy float64) float64 {
		node := at.node
		switch {
		case node.Class == "anchor":
			l := anchorLabel(node)
			return 1 / jsmath.Hypot(ux/(l.Width/2+5), uy/float64(small*0.8))
		case node.IsPipelineParent:
			return squareReach
		case node.SourceStrategy != "":
			return 2 * radius
		}
		return radius
	}

	// Quadrants around a node that a link runs through: 1 up and right, 2 down and right, 4
	// up and left, 8 down and left.
	taken := map[*placed]int{}
	occupy := func(at *placed, ux, uy float64) {
		if math.Abs(ux) < 0.25 || math.Abs(uy) < 0.25 {
			return
		}
		quadrant := 8
		switch {
		case ux > 0 && uy < 0:
			quadrant = 1
		case ux > 0:
			quadrant = 2
		case uy < 0:
			quadrant = 4
		}
		taken[at] |= quadrant
	}

	edge := func(role scene.Role, id string, x1, y1, x2, y2 float64, st style.Style) scene.Item {
		p := new(scene.Path).MoveTo(x1, y1).LineTo(x2, y2)
		return scene.Item{
			Kind: scene.Edge, Role: role, ID: id, Box: p.Bounds(),
			Parts: []scene.Part{{Path: p, Paint: scene.StrokeOnly}},
			Route: []scene.Point{{X: x1, Y: y1}, {X: x2, Y: y2}},
			Style: st,
		}
	}

	for _, link := range m.Links {
		from, to := byID[link.Source], byID[link.Target]
		if from == nil || to == nil || from == to {
			continue
		}
		// A pipeline component belongs to its parent already.
		if members[link.Target][link.Source] {
			continue
		}
		dx, dy := to.x-from.x, to.y-from.y
		distance := jsmath.Hypot(dx, dy)
		if distance < 0.01 {
			continue
		}
		ux, uy := dx/distance, dy/distance
		start, end := reach(from, ux, uy), reach(to, ux, uy)
		if start+end >= distance {
			continue
		}
		occupy(from, ux, uy)
		occupy(to, -ux, -uy)
		x1, y1 := from.x+float64(ux*start), from.y+float64(uy*start)
		x2, y2 := to.x-float64(ux*end), to.y-float64(uy*end)
		var ends [2]scene.Marker
		if link.Flow == "forward" || link.Flow == "bidirectional" {
			ends[1] = scene.Marker{Type: arrowPoint, At: scene.Point{X: x2, Y: y2}, DX: ux, DY: uy}
			x2 -= float64(ux * arrowTrim)
			y2 -= float64(uy * arrowTrim)
		}
		if link.Flow == "backward" || link.Flow == "bidirectional" {
			ends[0] = scene.Marker{Type: arrowPoint, At: scene.Point{X: x1, Y: y1}, DX: -ux, DY: -uy}
			x1 += float64(ux * arrowTrim)
			y1 += float64(uy * arrowTrim)
		}
		st := style.Style{Stroke: style.Line}
		if link.Dashed {
			st.Dash = style.Dashes(6, 6)
		}
		it := edge(RoleLink, link.Source+"-"+link.Target, x1, y1, x2, y2, st)
		it.Ends = ends
		items = append(items, it)
		if link.Label != "" {
			l := layoutText(link.Label, tiny+1, 200, 0)
			// Beside the middle of the line, on its upper side.
			side := 1.0
			if ux < 0 {
				side = -1
			}
			px, py := uy*side, -ux*side
			away := 5 + float64(math.Abs(px)*l.Width)/2 + float64(math.Abs(py)*l.Height)/2
			centered(RoleLinkLabel, it.ID, l, (from.x+to.x)/2+float64(px*away), (from.y+to.y)/2+float64(py*away),
				style.Style{Text: style.Muted})
		}
	}

	// Evolution: a dashed arrow to where the component is heading, drawn as an open circle.
	for _, trend := range m.Trends {
		origin := byID[trend.NodeID]
		if origin == nil {
			continue
		}
		tx, ty := projectX(trend.TargetX), projectY(trend.TargetY)
		dx, dy := tx-origin.x, ty-origin.y
		distance := jsmath.Hypot(dx, dy)
		start := 0.0
		if distance > 0 {
			start = reach(origin, dx/distance, dy/distance)
		}
		if distance > start+radius+4 {
			ux, uy := dx/distance, dy/distance
			ex, ey := tx-float64(ux*(radius+2)), ty-float64(uy*(radius+2))
			it := edge(RoleTrend, trend.NodeID, origin.x+float64(ux*start), origin.y+float64(uy*start),
				ex-float64(ux*arrowTrim), ey-float64(uy*arrowTrim), style.Style{Stroke: style.Accent, Dash: style.Dashes(4, 4)})
			it.Ends[1] = scene.Marker{Type: arrowPoint, At: scene.Point{X: ex, Y: ey}, DX: ux, DY: uy}
			items = append(items, it)
		}
		grow(tx-radius, ty-radius, tx+radius, ty+radius)
		items = append(items, scene.Item{
			Kind: scene.Mark, Role: RoleTrendTarget, Shape: "circle", ID: trend.NodeID, Box: circleBox(tx, ty, radius),
			Parts: []scene.Part{circle(tx, ty, radius, scene.FillAndStroke)},
			Style: style.Style{Fill: style.Bg, Stroke: style.Accent},
		})
	}

	for _, at := range positions {
		node := at.node
		x, y := at.x, at.y
		anchor := node.Class == "anchor"
		strategy := node.SourceStrategy
		role := RoleComponent
		switch {
		case anchor:
			role = RoleAnchor
		case node.IsPipelineParent:
			role = RolePipelineParent
		case node.InPipeline:
			role = RolePipelineComponent
		}
		var ox, oy *float64
		if node.LabelOffsetX != nil {
			v := clamp(*node.LabelOffsetX, -5000, 5000)
			ox = &v
		}
		if node.LabelOffsetY != nil {
			v := clamp(*node.LabelOffsetY, -5000, 5000)
			oy = &v
		}
		or := func(v *float64, d float64) float64 {
			if v != nil {
				return *v
			}
			return d
		}
		// An anchor is its label, unless it is a pipeline's parent too.
		it := scene.Item{Kind: scene.Node, Role: role, ID: node.ID, Box: scene.Rect{X: x, Y: y}, Style: style.Style{Fill: style.Bg, Stroke: style.Line}}
		if anchor {
			l := anchorLabel(node)
			cx, cy := x+or(ox, 0), y+or(oy, -3)
			it.Box = scene.Rect{X: cx - l.Width/2, Y: cy - l.Height/2, W: l.Width, H: l.Height}
		}
		// The shape on the node's own fill; a strategy's circle takes the fill, and what is
		// on it is hollow.
		inner := scene.FillAndStroke
		if strategy != "" {
			it.Tags = []string{strategy}
			it.Shape = "circle"
			it.Box = circleBox(x, y, 2*radius)
			switch strategy {
			case "outsource":
				it.Style.Fill = style.Muted
			case "buy":
				it.Style.Fill = style.SurfaceAlt
			case "build":
				it.Style.Fill = style.Surface
			}
			it.Parts = append(it.Parts, circle(x, y, 2*radius, scene.FillAndStroke))
			inner = scene.Hollow
		}
		switch {
		case strategy == "market":
			bx := float64(tri * jsmath.Cos(sixthPi))
			by := y + float64(tri*jsmath.Sin(sixthPi))
			it.Shape = "market"
			it.Parts = append(it.Parts, scene.Part{Path: new(scene.Path).Polygon(x, y-tri, x-bx, by, x+bx, by), Paint: scene.StrokeOnly})
			for _, c := range [][2]float64{{x, y - tri}, {x - bx, by}, {x + bx, by}} {
				it.Parts = append(it.Parts, circle(c[0], c[1], dot, scene.Hollow))
			}
		case node.IsPipelineParent:
			box := scene.Rect{X: x - square/2, Y: y - square/2, W: square, H: square}
			if strategy == "" {
				it.Shape, it.Box = "square", box
			}
			it.Parts = append(it.Parts, scene.Part{Box: box, Paint: inner})
		case !anchor:
			if strategy == "" {
				it.Shape, it.Box = "circle", circleBox(x, y, radius)
			}
			it.Parts = append(it.Parts, circle(x, y, radius, inner))
		}
		items = append(items, it)
		if node.HasInertia && node.Inertia {
			offset := 0.0
			if node.IsPipelineParent {
				offset = square / 2
			} else {
				offset = radius
			}
			offset += 10
			if strategy != "" {
				offset += radius + 4
			}
			box := scene.Rect{X: x + offset, Y: y - radius, W: 4, H: 2 * radius}
			items = append(items, scene.Item{
				Kind: scene.Mark, Role: RoleInertia, ID: node.ID, Box: box,
				Parts: []scene.Part{{Box: box, Paint: scene.FillOnly}},
				Style: style.Style{Fill: style.Line},
			})
		}
		if anchor {
			centered(RoleLabel, node.ID, anchorLabel(node), x+or(ox, 0), y+or(oy, -3), style.Style{})
		} else {
			l := layoutText(node.Label, small, plain, 0)
			lift := float64(labelOffset)
			if strategy != "" {
				lift += 10
			}
			switch {
			case ox != nil || oy != nil:
				starting(RoleLabel, node.ID, l, x+or(ox, lift), y+or(oy, -lift), style.Style{})
			case node.InPipeline:
				centered(RoleLabel, node.ID, l, x, y+2*radius+4+l.Height/2, style.Style{})
			default:
				// The label goes up and to the right unless a link leaves that way; then the
				// next free corner.
				busy := taken[at]
				corner := 1
				switch {
				case busy&1 == 0:
					corner = 1
				case busy&2 == 0:
					corner = 2
				case busy&4 == 0:
					corner = 4
				case busy&8 == 0:
					corner = 8
				}
				right := corner == 1 || corner == 2
				up := corner == 1 || corner == 4
				lx := x - lift - l.Width
				if right {
					lx = x + lift
				}
				ly := y + lift + float64(small*0.75)
				if up {
					ly = y - lift
				}
				starting(RoleLabel, node.ID, l, lx, ly, style.Style{})
			}
		}
		grow(x-2*radius, y-2*radius, x+2*radius+20, y+2*radius)
	}

	for _, a := range m.Annotations {
		number := chars.NumberString(a.Number)
		l := layoutText(number, tiny, 4000, text.Bold)
		for _, c := range a.Coordinates {
			x, y := projectX(c.X), projectY(c.Y)
			r := max(10, l.Width/2+4)
			grow(x-r, y-r, x+r, y+r)
			it := scene.Item{
				Kind: scene.Node, Role: RoleAnnotation, Shape: "circle", ID: number, Box: circleBox(x, y, r),
				Parts: []scene.Part{circle(x, y, r, scene.FillAndStroke)},
				At:    scene.Point{X: x, Y: y},
				Style: style.Style{Fill: style.Bg, Stroke: style.Line},
			}
			if !l.Empty() {
				label := l
				it.Label = &label
			}
			items = append(items, it)
		}
	}
	var listed []Annotation
	for _, a := range m.Annotations {
		if a.Text != "" {
			listed = append(listed, a)
		}
	}
	slices.SortStableFunc(listed, func(a, b Annotation) int { return order(a.Number - b.Number) })
	if len(listed) > 0 {
		room := 320.0
		if tight {
			room = max(80, plotW-20)
		}
		lines := make([]text.Label, len(listed))
		width, height := 0.0, 0.0
		for i, a := range listed {
			lines[i] = layoutText(chars.NumberString(a.Number)+". "+a.Text, small, room, 0)
			width = max(width, lines[i].Width)
			height += lines[i].Height
		}
		width += 20
		height += 14
		// Without a position the list goes under the map, where it covers nothing.
		bx, by := 0.0, legendTop
		if box := m.AnnotationsBox; box != nil {
			bx = clamp(projectX(box.X), 0, max(0, plotW-width))
			by = clamp(projectY(box.Y), 0, max(0, plotH-height))
		}
		grow(bx, by, bx+width, by+height)
		box := scene.Rect{X: bx, Y: by, W: width, H: height}
		items = append(items, scene.Item{
			Kind: scene.Group, Role: RoleAnnotations, Box: box,
			Parts: []scene.Part{{Box: box, Rounded: true}},
			Style: style.Style{Fill: style.Bg, Stroke: style.Border},
		})
		y := by + 7
		for _, l := range lines {
			baseline := y + l.LineHeight/2 + float64(small*0.35)
			at := baseline - float64(l.Size*0.35) + l.Height/2 - l.LineHeight/2
			labelItem(scene.Text, RoleAnnotationText, "", l, scene.Point{X: bx + 10, Y: at}, scene.Start, style.Style{})
			y += l.Height
		}
	}

	for _, note := range m.Notes {
		l := layoutText(note.Text, small, min(360, float64(plotW*0.6)), text.Bold)
		starting(RoleNote, "", l, projectX(note.X), projectY(note.Y), style.Style{})
	}

	// Accelerators point right, deaccelerators left.
	force := func(name string, px, py float64, forward bool) {
		x, y := projectX(px), projectY(py)
		const half = forceHeight / 2
		tip, tail, neck, role := x, x+forceWidth, x+forceHead, RoleDeaccelerator
		if forward {
			tip, tail, neck, role = x+forceWidth, x, x+forceWidth-forceHead, RoleAccelerator
		}
		p := new(scene.Path).Polygon(tail, y-half, neck, y-half, neck, y-half-8, tip, y, neck, y+half+8, neck, y+half, tail, y+half)
		grow(x, y-half-8, x+forceWidth, y+half+8)
		it := scene.Item{
			Kind: scene.Node, Role: role, Shape: "arrow", ID: name, Box: p.Bounds(),
			Parts: []scene.Part{{Path: p}},
			At:    scene.Point{X: x + forceWidth/2, Y: y + half + 8 + 12},
			Style: style.Style{Fill: style.Surface, Stroke: style.Line},
		}
		l := layoutText(name, tiny+1, 160, text.Bold)
		grow(it.At.X-l.Width/2, it.At.Y-l.Height/2, it.At.X+l.Width/2, it.At.Y+l.Height/2)
		if !l.Empty() {
			it.Label = &l
		}
		items = append(items, it)
	}
	for _, a := range m.Accelerators {
		force(a.Name, a.X, a.Y, true)
	}
	for _, d := range m.Deaccelerators {
		force(d.Name, d.X, d.Y, false)
	}

	titleRoom := 4000.0
	if tight {
		titleRoom = max(80, plotW)
	}
	title := layoutText(m.Title, size, titleRoom, text.Bold)
	titleHeight := 0.0
	if title.Height > 0 {
		titleHeight = title.Height + 12
	}
	left := min(minX, plotW/2-title.Width/2)
	right := max(maxX, plotW/2+title.Width/2)
	tx := pad - left
	ty := pad + titleHeight - minY
	for i := range items {
		items[i].Move(tx, ty)
	}
	if !title.Empty() {
		items = append([]scene.Item{{
			Kind: scene.Text, Role: RoleTitle, Label: &title,
			At: scene.Point{X: tx + plotW/2, Y: pad + title.Height/2},
		}}, items...)
	}
	return &scene.Scene{
		Type:     "wardley",
		Width:    math.Ceil(right - left + 2*pad),
		Height:   math.Ceil(maxY - minY + titleHeight + 2*pad),
		FontSize: size,
		Title:    m.AccTitle,
		Desc:     m.AccDescr,
		Items:    items,
	}
}

// order is the order JavaScript's sort takes from a comparator's difference: less below
// zero, and the same at zero or NaN.
func order(d float64) int {
	switch {
	case d < 0:
		return -1
	case d > 0:
		return 1
	}
	return 0
}

// heading is a label drawn in the weight of headings, measured in the weight it was laid out
// in, as Pele's font-weight on the text draws it.
func heading(l text.Label) text.Label {
	lines := make([][]text.Span, len(l.Lines))
	for i, line := range l.Lines {
		lines[i] = make([]text.Span, len(line))
		for j, s := range line {
			s.Face |= text.Bold
			lines[i][j] = s
		}
	}
	l.Lines = lines
	return l
}
