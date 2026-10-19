package limu

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/agentflow"
	"github.com/firede/limu/diagram/block"
	"github.com/firede/limu/diagram/class"
	"github.com/firede/limu/diagram/er"
	"github.com/firede/limu/diagram/flowchart"
	"github.com/firede/limu/diagram/gantt"
	"github.com/firede/limu/diagram/gitgraph"
	"github.com/firede/limu/diagram/journey"
	"github.com/firede/limu/diagram/kanban"
	"github.com/firede/limu/diagram/packet"
	"github.com/firede/limu/diagram/pie"
	"github.com/firede/limu/diagram/quadrant"
	"github.com/firede/limu/diagram/radar"
	"github.com/firede/limu/diagram/requirement"
	"github.com/firede/limu/diagram/sankey"
	"github.com/firede/limu/diagram/sequence"
	"github.com/firede/limu/diagram/state"
	"github.com/firede/limu/diagram/swimlane"
	"github.com/firede/limu/diagram/timeline"
	"github.com/firede/limu/diagram/treemap"
	"github.com/firede/limu/diagram/usecase"
	"github.com/firede/limu/diagram/venn"
	"github.com/firede/limu/diagram/xychart"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
)

// Layout reads a diagram's text and lays it out as a scene, for a backend to draw. Its
// errors are *diagram.Error: a diagram of a type limu does not read yet, text it cannot
// parse, or text past a limit.
func Layout(src string, o *layout.Options) (*scene.Scene, error) {
	if o == nil {
		o = &layout.Options{}
	}
	p, err := diagram.Prepare(src, diagram.Options{})
	if err != nil {
		return nil, err
	}
	var sc *scene.Scene
	switch p.Type {
	case diagram.Flowchart:
		db, err := flowchart.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = flowchart.Layout(db, p.Config, o)
	case diagram.Swimlane:
		db, err := swimlane.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = swimlane.Layout(db, p.Config, o)
	case diagram.Requirement:
		db, err := requirement.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = requirement.Layout(db, p.Config, o)
	case diagram.State:
		db, err := state.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = state.Layout(db, p.Config, o)
	case diagram.ER:
		db, err := er.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = er.Layout(db, p.Config, o)
	case diagram.AgentFlow:
		db, err := agentflow.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = agentflow.Layout(db, p.Config, o)
	case diagram.Class:
		db, err := class.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = class.Layout(db, p.Config, o)
	case diagram.UseCase:
		m, err := usecase.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = usecase.Layout(m, p.Config, o)
	case diagram.Timeline:
		m, err := timeline.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = timeline.Layout(m, p.Config, o)
	case diagram.Journey:
		m, err := journey.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = journey.Layout(m, p.Config, o)
	case diagram.Packet:
		m, err := packet.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = packet.Layout(m, p.Config, o)
	case diagram.Pie:
		m, err := pie.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = pie.Layout(m, p.Config, o)
	case diagram.QuadrantChart:
		m, err := quadrant.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = quadrant.Layout(m, p.Config, o)
	case diagram.Radar:
		m, err := radar.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = radar.Layout(m, p.Config, o)
	case diagram.Kanban:
		m, err := kanban.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = kanban.Layout(m, p.Config, o)
	case diagram.GitGraph:
		m, err := gitgraph.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = gitgraph.Layout(m, p.Config, o)
	case diagram.Sequence:
		db, err := sequence.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = sequence.Layout(db, p.Config, o)
	case diagram.Treemap:
		m, err := treemap.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = treemap.Layout(m, p.Config, o)
	case diagram.Gantt:
		db, err := gantt.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		if sc, err = gantt.Layout(db, p.Config, o); err != nil {
			return nil, err
		}
	case diagram.XYChart:
		m, err := xychart.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		if sc, err = xychart.Layout(m, p.Config, o); err != nil {
			return nil, err
		}
	case diagram.Block:
		db, err := block.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		if sc, err = block.Layout(db, p.Config, o); err != nil {
			return nil, err
		}
	case diagram.Sankey:
		m, err := sankey.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		if sc, err = sankey.Layout(m, p.Config, o); err != nil {
			return nil, err
		}
	case diagram.Venn:
		m, err := venn.ParsePrepared(p)
		if err != nil {
			return nil, err
		}
		sc = venn.Layout(m, p.Config, o)
	default:
		return nil, diagram.Unsupported(p.Type)
	}
	if o.Snap > 0 {
		sc.Snap(o.Snap)
	}
	return sc, nil
}
