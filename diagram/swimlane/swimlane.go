// Package swimlane lays out Mermaid's swimlane diagrams, as Pele's src/diagrams/swimlane does.
// A swimlane diagram is a flowchart whose top-level subgraphs are lanes: it has the
// flowchart's syntax and model, and is laid out with layout.Lanes, each lane drawn as a band
// the length of the diagram with its title in a band at its start.
//
// A lane has the role swimlane.lane (RoleLane). Everything else is drawn by the flowchart's
// layout and keeps the flowchart's roles: flowchart.node, flowchart.edge,
// flowchart.edge-label, flowchart.cluster and flowchart.cluster-title for a subgraph inside
// a lane, and flowchart.title. A host that styles flowcharts styles a swimlane diagram's
// nodes and edges the same way.
package swimlane

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/flowchart"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
	"github.com/firede/limu/text"
)

// defaultLane is the id of the lane that holds the nodes outside every lane.
const defaultLane = "__swimlane_default__"

// Parse parses a swimlane diagram's text, as the flowchart's Parse does: Pele's swimlane
// parses with the flowchart's parser.
func Parse(source string, config diagram.Config, title string, lineOffset int, limits diagram.Limits) (*flowchart.DB, error) {
	return flowchart.Parse(source, config, title, lineOffset, limits)
}

// ParsePrepared parses a prepared swimlane diagram.
func ParsePrepared(p *diagram.Prepared) (*flowchart.DB, error) {
	return flowchart.ParsePrepared(p)
}

// Layout lays out a parsed swimlane diagram as a scene, as Pele's swimlane renders it: with
// the flowchart's renderFlowchart and the variant that makes its top-level groups lanes. It
// reads the "swimlane" section of the configuration, or the "flowchart" section when there is
// no "swimlane" section.
func Layout(db *flowchart.DB, config diagram.Config, o *layout.Options) *scene.Scene {
	return flowchart.LayoutVariant(db, config, o, &variant)
}

var variant = flowchart.Variant{
	Type:    "swimlane",
	Prepare: prepare,
	Layout: func(nodes []layout.CNode, edges []layout.CEdge, dir layout.Dir, opt layout.LayeredOptions, tight bool) layout.CompoundResult {
		pad := float64(layout.DefaultLanePad)
		if tight {
			pad = 12
		}
		return layout.Lanes(nodes, edges, dir, opt, pad)
	},
	DrawGroup: drawLane,
}

// prepare puts the nodes outside every lane in one unnamed lane, as Mermaid does, and drops
// the direction of groups, as Pele's prepare does: lanes all run the same way, so a
// direction inside a subgraph has no effect.
func prepare(g *flowchart.Graph) {
	loose := false
	for _, node := range g.Nodes {
		if node.IsGroup {
			node.Dir = ""
		} else if !node.HasParent {
			loose = true
		}
	}
	if !loose {
		return
	}
	for _, node := range g.Nodes {
		if !node.IsGroup && !node.HasParent {
			node.Parent, node.HasParent = defaultLane, true
		}
	}
	g.Nodes = append(g.Nodes, &flowchart.GraphNode{
		ID:                defaultLane,
		LabelType:         flowchart.LabelText,
		Shape:             "rect",
		IsGroup:           true,
		CSSStyles:         []string{},
		CSSCompiledStyles: []string{},
		ColorIndex:        -1,
	})
}

// drawLane draws a lane, as Pele's drawLane does: an outline the length of the diagram, with
// its title in a band at the start. Where the lanes run across, the band is at their left
// and the title is turned to read upwards.
func drawLane(c *layout.CNode, label *text.Label, author style.Author, classes []string, id string, dir layout.Dir) scene.Item {
	left := c.X - c.W/2
	top := c.Y - c.H/2
	sideways := dir == layout.LR || dir == layout.RL
	outline := scene.Rect{X: left, Y: top, W: c.W, H: c.H}
	band := scene.Rect{X: left, Y: top, W: c.W, H: c.PadTop}
	at := scene.Point{X: c.X, Y: top + c.PadTop/2}
	if sideways {
		band.W, band.H = c.PadTop, c.H
		at = scene.Point{X: left + c.PadTop/2, Y: c.Y}
	}
	it := scene.Item{
		Kind: scene.Group, Role: RoleLane, ID: id, Class: classes, Box: outline,
		Parts:  []scene.Part{{Box: outline, Paint: scene.StrokeOnly}, {Box: band}},
		Style:  style.Style{Fill: style.SurfaceAlt, Stroke: style.Border, Text: style.Muted},
		Author: author.Style,
	}
	if !label.Empty() {
		it.Label, it.At = label, at
		if sideways {
			it.Rotate = -90
		}
	}
	return it
}
