// Package samples holds scenes laid out by hand, for the tests and the example: no diagram
// is read yet that would lay them out.
package samples

import (
	"math"

	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

func label(m text.Measurer, s string, size float64) *text.Label {
	l := text.Layout(s, text.Options{Measurer: m, Size: size, MaxWidth: 200})
	return &l
}

// Flowchart is the svg package's sample, measured with m: a group holding a box, a diamond,
// an edge between them with its label, and a title.
func Flowchart(m text.Measurer) *scene.Scene {
	start := label(m, "Start", 16)
	is := label(m, "Is it <b>ready</b>?", 16)
	yes := label(m, "Yes", 14)
	group := label(m, "Group & co", 16)
	edge := new(scene.Path).MoveTo(100, 80).CubeTo(100, 100, 100, 100, 100, 113)
	return &scene.Scene{
		Type: "flowchart", Width: 240, Height: 220, FontSize: 16, Title: "A <sample>",
		Items: []scene.Item{
			{
				Kind: scene.Group, Role: "flowchart.cluster", ID: "g1",
				Box:   scene.Rect{X: 8, Y: 8, W: 224, H: 204},
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 8, Y: 8, W: 224, H: 204}}},
				Label: group, At: scene.Point{X: 120, Y: 24},
				Style: style.Style{Fill: style.SurfaceAlt.Alpha(0.5), Stroke: style.Border, Text: style.Muted},
			},
			{
				Kind: scene.Edge, Role: "flowchart.edge", ID: "L_A_B_0",
				Parts: []scene.Part{{Path: edge, Paint: scene.StrokeOnly}},
				Ends:  [2]scene.Marker{{}, {Type: "arrow_point", At: scene.Point{X: 100, Y: 120}, DY: 1}},
				Style: style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound},
			},
			{
				Kind: scene.Text, Role: "flowchart.edge-label",
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 100 - yes.Width/2 - 4, Y: 100 - yes.Height/2, W: yes.Width + 8, H: yes.Height}}},
				Label: yes, At: scene.Point{X: 100, Y: 100},
				Style: style.Style{Fill: style.Bg, Stroke: style.None, Radius: style.N(3)},
			},
			{
				Kind: scene.Node, Role: "flowchart.node", Shape: "rect", ID: "A", Class: []string{"hot"},
				Box:   scene.Rect{X: 60, Y: 40, W: 80, H: 40},
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 60, Y: 40, W: 80, H: 40}}},
				Label: start, At: scene.Point{X: 100, Y: 60},
				Style:  style.Style{Fill: style.Surface, Stroke: style.Border},
				Author: style.Declarations([]string{"fill:#f9f", "stroke-width:2px"}).Style,
				Link:   "https://example.com/?a=1&b=2", Tooltip: "Where it starts",
			},
			{
				Kind: scene.Node, Role: "flowchart.node", Shape: "diam", ID: "B",
				Box:   scene.Rect{X: 30, Y: 120, W: 140, H: 70},
				Parts: []scene.Part{{Path: new(scene.Path).Polygon(100, 120, 170, 155, 100, 190, 30, 155)}},
				Label: is, At: scene.Point{X: 100, Y: 155},
				Style: style.Style{Fill: style.Surface, Stroke: style.Border},
			},
		},
	}
}

// toward is the unit vector from a to b.
func toward(ax, ay, bx, by float64) (float64, float64) {
	l := math.Hypot(bx-ax, by-ay)
	return (bx - ax) / l, (by - ay) / l
}

// Features is a scene of what a backend draws besides the flowchart's plain parts: dashed
// lines and boxes, opacity, colors of their own at an alpha, the three markers, labels
// aligned at their start and end, turned, and in each face, and a diamond for a drawing of
// one's own to take the place of.
func Features(m text.Measurer) *scene.Scene {
	faces := func(size float64, spans ...text.Span) *text.Label {
		l := text.Label{Size: size, LineHeight: math.Floor(size*text.LineHeight + 0.5)}
		w := 0.0
		for i := range spans {
			spans[i].Width = m.Width(spans[i].Text, size, spans[i].Face)
			w += spans[i].Width
		}
		l.Lines, l.Widths, l.Width, l.Height = [][]text.Span{spans}, []float64{w}, w, l.LineHeight
		return &l
	}
	sdx, sdy := toward(120, 30, 70, 50)
	edx, edy := toward(200, 80, 250, 50)
	half := label(m, "Half", 14)
	return &scene.Scene{
		Type: "features", Width: 440, Height: 320, FontSize: 14, Title: "Features",
		Desc: "Dashes, opacity, markers, alignment, rotation and faces",
		Items: []scene.Item{
			{
				Kind: scene.Group, Role: "test.frame", ID: "frame",
				Box:   scene.Rect{X: 8, Y: 8, W: 424, H: 304},
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 8, Y: 8, W: 424, H: 304}}},
				Label: label(m, "Frame", 12), At: scene.Point{X: 20, Y: 22}, Align: scene.Start,
				Style: style.Style{Fill: style.SurfaceAlt.Alpha(0.4), Stroke: style.Border, Text: style.Muted, Dash: style.Dashes(6, 3), Radius: style.N(8)},
			},
			{
				Kind: scene.Mark, Role: "sequence.lifeline", ID: "life",
				Parts: []scene.Part{{Path: new(scene.Path).MoveTo(40, 40).LineTo(40, 296), Paint: scene.StrokeOnly}},
				Style: style.Style{Stroke: style.Line, Dash: style.Dashes(3, 3)},
			},
			{
				Kind: scene.Edge, Role: "test.edge", ID: "dashed",
				Parts: []scene.Part{{Path: new(scene.Path).MoveTo(70, 50).CubeTo(120, 30, 200, 80, 250, 50), Paint: scene.StrokeOnly}},
				Route: []scene.Point{{X: 70, Y: 50}, {X: 250, Y: 50}},
				Ends: [2]scene.Marker{
					{Type: "arrow_cross", At: scene.Point{X: 70, Y: 50}, DX: sdx, DY: sdy},
					{Type: "arrow_circle", At: scene.Point{X: 250, Y: 50}, DX: edx, DY: edy},
				},
				Style: style.Style{Stroke: style.Accent, StrokeWidth: style.N(2), Dash: style.Dashes(5, 3), Fill: style.None},
			},
			{
				Kind: scene.Edge, Role: "test.edge", ID: "arrow",
				Parts: []scene.Part{{Path: new(scene.Path).MoveTo(90, 100).CubeTo(90, 150, 180, 120, 180, 163), Paint: scene.StrokeOnly}},
				Route: []scene.Point{{X: 90, Y: 100}, {X: 180, Y: 163}},
				Ends:  [2]scene.Marker{{}, {Type: "arrow_point", At: scene.Point{X: 180, Y: 170}, DY: 1}},
				Style: style.Style{Stroke: style.Line, Fill: style.None, Cap: style.CapRound},
			},
			{
				Kind: scene.Node, Role: "test.node", ID: "opaque",
				Box:   scene.Rect{X: 270, Y: 40, W: 100, H: 44},
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 270, Y: 40, W: 100, H: 44}}},
				Label: label(m, "Opaque", 14), At: scene.Point{X: 320, Y: 62},
				Style: style.Style{Fill: style.Series(0), Stroke: style.Border, Text: style.Bg},
			},
			{
				Kind: scene.Node, Role: "test.node", ID: "half",
				Box:   scene.Rect{X: 310, Y: 66, W: 100, H: 44},
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 310, Y: 66, W: 100, H: 44}}},
				Label: half, At: scene.Point{X: 360, Y: 88},
				Style: style.Style{Fill: style.Series(1), Stroke: style.Border, Opacity: style.N(0.5)},
			},
			{
				Kind: scene.Node, Role: "test.node", ID: "alpha",
				Box:   scene.Rect{X: 200, Y: 176, W: 100, H: 44},
				Parts: []scene.Part{{Rounded: true, Box: scene.Rect{X: 200, Y: 176, W: 100, H: 44}}},
				Label: label(m, "Alpha fill", 14), At: scene.Point{X: 250, Y: 198},
				Style: style.Style{Fill: style.Of(style.Hex("#ff000080")), Stroke: style.Critical, StrokeWidth: style.N(3), Radius: style.N(12)},
			},
			{
				Kind: scene.Node, Role: "test.node", Shape: "diam", ID: "mine",
				Box:   scene.Rect{X: 320, Y: 130, W: 100, H: 70},
				Parts: []scene.Part{{Path: new(scene.Path).Polygon(370, 130, 420, 165, 370, 200, 320, 165)}},
				Label: label(m, "Mine", 14), At: scene.Point{X: 370, Y: 165},
				Style: style.Style{Fill: style.Surface, Stroke: style.Border},
			},
			{
				Kind: scene.Mark, Role: "test.joins", ID: "joins",
				Parts: []scene.Part{
					{Path: new(scene.Path).MoveTo(320, 236).LineTo(340, 216).LineTo(360, 236), Paint: scene.StrokeOnly},
					{Path: new(scene.Path).MoveTo(380, 236).LineTo(400, 216).LineTo(420, 236).MoveTo(410, 210).LineTo(410, 210), Paint: scene.StrokeOnly},
				},
				Style: style.Style{Stroke: style.Accent, StrokeWidth: style.N(4), Cap: style.CapSquare, Join: style.JoinBevel},
			},
			{
				Kind: scene.Mark, Role: "test.joins", ID: "round",
				Parts: []scene.Part{{Path: new(scene.Path).MoveTo(170, 296).LineTo(190, 268).LineTo(240, 292), Paint: scene.StrokeOnly}},
				Style: style.Style{Stroke: style.Accent, StrokeWidth: style.N(4), Cap: style.CapRound, Join: style.JoinMiter},
			},
			{
				Kind: scene.Mark, Role: "test.guide",
				Parts: []scene.Part{{Path: new(scene.Path).MoveTo(120, 196).LineTo(120, 300), Paint: scene.StrokeOnly}},
				Style: style.Style{Stroke: style.Critical},
			},
			{
				Kind: scene.Text, Role: "test.text", ID: "start",
				Label: label(m, "Start<br>two lines", 12), At: scene.Point{X: 120, Y: 215}, Align: scene.Start,
			},
			{
				Kind: scene.Text, Role: "test.text", ID: "middle",
				Label: label(m, "Middle", 14), At: scene.Point{X: 120, Y: 250},
			},
			{
				Kind: scene.Text, Role: "test.text", ID: "end",
				Label: label(m, "End<br>two lines", 12), At: scene.Point{X: 120, Y: 280}, Align: scene.End,
			},
			{
				Kind: scene.Text, Role: "test.text", ID: "faces",
				Label: faces(14, text.Span{Text: "Bold ", Face: text.Bold}, text.Span{Text: "italic ", Face: text.Italic}, text.Span{Text: "mono", Face: text.Mono}),
				At:    scene.Point{X: 250, Y: 245},
			},
			{
				Kind: scene.Text, Role: "test.text", ID: "up",
				Label: label(m, "Turned −90°", 12), At: scene.Point{X: 60, Y: 170}, Rotate: -90,
				Style: style.Style{Text: style.Muted},
			},
			{
				Kind: scene.Text, Role: "test.text", ID: "turned",
				Label: label(m, "Turned 30°", 14), At: scene.Point{X: 360, Y: 270}, Rotate: 30,
				Style: style.Style{Text: style.Accent},
			},
		},
	}
}
