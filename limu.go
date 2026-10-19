package limu

import (
	"slices"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/agentflow"
	"github.com/firede/limu/diagram/architecture"
	"github.com/firede/limu/diagram/block"
	"github.com/firede/limu/diagram/c4"
	"github.com/firede/limu/diagram/class"
	"github.com/firede/limu/diagram/cynefin"
	"github.com/firede/limu/diagram/er"
	"github.com/firede/limu/diagram/eventmodeling"
	"github.com/firede/limu/diagram/flowchart"
	"github.com/firede/limu/diagram/gantt"
	"github.com/firede/limu/diagram/gitgraph"
	"github.com/firede/limu/diagram/info"
	"github.com/firede/limu/diagram/ishikawa"
	"github.com/firede/limu/diagram/journey"
	"github.com/firede/limu/diagram/kanban"
	"github.com/firede/limu/diagram/mindmap"
	"github.com/firede/limu/diagram/packet"
	"github.com/firede/limu/diagram/pie"
	"github.com/firede/limu/diagram/quadrant"
	"github.com/firede/limu/diagram/radar"
	"github.com/firede/limu/diagram/railroad"
	"github.com/firede/limu/diagram/requirement"
	"github.com/firede/limu/diagram/sankey"
	"github.com/firede/limu/diagram/sequence"
	"github.com/firede/limu/diagram/state"
	"github.com/firede/limu/diagram/swimlane"
	"github.com/firede/limu/diagram/timeline"
	"github.com/firede/limu/diagram/treemap"
	"github.com/firede/limu/diagram/treeview"
	"github.com/firede/limu/diagram/usecase"
	"github.com/firede/limu/diagram/venn"
	"github.com/firede/limu/diagram/wardley"
	"github.com/firede/limu/diagram/xychart"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
)

// A kind is what limu does with a type of diagram, as Pele's Diagram is: parse, the parser
// of the type, and layout, its layout. They take and give its model as any, so that one
// table holds every type; kindOf and kindOfErr adapt a package's ParsePrepared and Layout to
// them, the second for a Layout that can fail.
type kind struct {
	parse  func(*diagram.Prepared) (any, error)
	layout func(model any, config diagram.Config, o *layout.Options) (*scene.Scene, error)
}

func kindOf[M any](parse func(*diagram.Prepared) (M, error), lay func(M, diagram.Config, *layout.Options) *scene.Scene) kind {
	return kindOfErr(parse, func(m M, config diagram.Config, o *layout.Options) (*scene.Scene, error) {
		return lay(m, config, o), nil
	})
}

func kindOfErr[M any](parse func(*diagram.Prepared) (M, error), lay func(M, diagram.Config, *layout.Options) (*scene.Scene, error)) kind {
	return kind{
		parse: func(p *diagram.Prepared) (any, error) {
			m, err := parse(p)
			if err != nil {
				return nil, err
			}
			return m, nil
		},
		layout: func(m any, config diagram.Config, o *layout.Options) (*scene.Scene, error) {
			return lay(m.(M), config, o)
		},
	}
}

// kinds are the types of diagram limu reads, each with its package's parser and layout.
var kinds = map[diagram.Type]kind{
	diagram.Flowchart:     kindOf(flowchart.ParsePrepared, flowchart.Layout),
	diagram.Pie:           kindOf(pie.ParsePrepared, pie.Layout),
	diagram.ER:            kindOf(er.ParsePrepared, er.Layout),
	diagram.Mindmap:       kindOf(mindmap.ParsePrepared, mindmap.Layout),
	diagram.Kanban:        kindOf(kanban.ParsePrepared, kanban.Layout),
	diagram.Timeline:      kindOf(timeline.ParsePrepared, timeline.Layout),
	diagram.Journey:       kindOf(journey.ParsePrepared, journey.Layout),
	diagram.Class:         kindOf(class.ParsePrepared, class.Layout),
	diagram.QuadrantChart: kindOf(quadrant.ParsePrepared, quadrant.Layout),
	diagram.XYChart:       kindOfErr(xychart.ParsePrepared, xychart.Layout),
	diagram.State:         kindOf(state.ParsePrepared, state.Layout),
	diagram.Gantt:         kindOfErr(gantt.ParsePrepared, gantt.Layout),
	diagram.Architecture:  kindOf(architecture.ParsePrepared, architecture.Layout),
	diagram.Requirement:   kindOf(requirement.ParsePrepared, requirement.Layout),
	diagram.Sankey:        kindOfErr(sankey.ParsePrepared, sankey.Layout),
	diagram.Packet:        kindOf(packet.ParsePrepared, packet.Layout),
	diagram.Radar:         kindOf(radar.ParsePrepared, radar.Layout),
	diagram.Treemap:       kindOf(treemap.ParsePrepared, treemap.Layout),
	diagram.Info:          kindOf(info.ParsePrepared, info.Layout),
	diagram.Sequence:      kindOf(sequence.ParsePrepared, sequence.Layout),
	diagram.GitGraph:      kindOf(gitgraph.ParsePrepared, gitgraph.Layout),
	diagram.C4:            kindOfErr(c4.ParsePrepared, c4.Layout),
	diagram.Block:         kindOfErr(block.ParsePrepared, block.Layout),
	diagram.Railroad:      kindOf(railroad.ParsePrepared, railroad.Layout),
	diagram.AgentFlow:     kindOf(agentflow.ParsePrepared, agentflow.Layout),
	diagram.TreeView:      kindOf(treeview.ParsePrepared, treeview.Layout),
	diagram.Cynefin:       kindOf(cynefin.ParsePrepared, cynefin.Layout),
	diagram.Wardley:       kindOf(wardley.ParsePrepared, wardley.Layout),
	diagram.EventModeling: kindOf(eventmodeling.ParsePrepared, eventmodeling.Layout),
	diagram.UseCase:       kindOf(usecase.ParsePrepared, usecase.Layout),
	diagram.Venn:          kindOf(venn.ParsePrepared, venn.Layout),
	diagram.Ishikawa:      kindOf(ishikawa.ParsePrepared, ishikawa.Layout),
	diagram.Swimlane:      kindOf(swimlane.ParsePrepared, swimlane.Layout),
}

// types are the keys of kinds, in the order Pele's registry lists them.
var types = []diagram.Type{
	diagram.Flowchart,
	diagram.Pie,
	diagram.ER,
	diagram.Mindmap,
	diagram.Kanban,
	diagram.Timeline,
	diagram.Journey,
	diagram.Class,
	diagram.QuadrantChart,
	diagram.XYChart,
	diagram.State,
	diagram.Gantt,
	diagram.Architecture,
	diagram.Requirement,
	diagram.Sankey,
	diagram.Packet,
	diagram.Radar,
	diagram.Treemap,
	diagram.Info,
	diagram.Sequence,
	diagram.GitGraph,
	diagram.C4,
	diagram.Block,
	diagram.Railroad,
	diagram.AgentFlow,
	diagram.TreeView,
	diagram.Cynefin,
	diagram.Wardley,
	diagram.EventModeling,
	diagram.UseCase,
	diagram.Venn,
	diagram.Ishikawa,
	diagram.Swimlane,
}

// Types are the types of diagram limu reads, in the order Pele's registry lists them: every
// type Pele draws.
func Types() []diagram.Type { return slices.Clone(types) }

// Supports reports whether limu reads the type of diagram src is, as Pele's supports does:
// its type is detected, and it is one limu reads. Text whose front matter does not read is
// not supported. It does not parse the diagram, which may still be in error.
func Supports(src string) bool {
	typ, err := diagram.DetectType(src)
	if err != nil {
		return false
	}
	_, ok := kinds[typ]
	return ok
}

// A Diagram is a diagram's text read into a model, ready to be laid out. Laying it out
// changes neither its model nor its configuration, so one Diagram may be laid out many
// times, with other options, and from many goroutines at once: a host that lays a diagram
// out again when its width changes parses it once.
type Diagram struct {
	typ    diagram.Type
	config diagram.Config
	model  any
	kind   kind
}

// Parse reads a diagram's text, as Pele's parse does: its front matter and directives, its
// type, and then its body, with the parser of its type. opt holds what Pele's ParseOptions
// do, the limits the text is held to and the configuration under its own; what changes where
// things are is given to Layout. Its errors are *diagram.Error: text of no type limu reads,
// text it cannot parse, or text past a limit.
func Parse(src string, opt diagram.Options) (*Diagram, error) {
	p, err := diagram.Prepare(src, opt)
	if err != nil {
		return nil, err
	}
	k, ok := kinds[p.Type]
	if !ok {
		return nil, diagram.Unsupported(p.Type)
	}
	m, err := k.parse(p)
	if err != nil {
		return nil, err
	}
	return &Diagram{typ: p.Type, config: p.Config, model: m, kind: k}, nil
}

// Type is the diagram's type.
func (d *Diagram) Type() diagram.Type { return d.typ }

// Config is the diagram's configuration: its own, over the Options.Config it was parsed
// with. It is not a copy; a host that changes it must not lay the diagram out meanwhile.
func (d *Diagram) Config() diagram.Config { return d.config }

// Model is the model the diagram's package parsed it into, as *flowchart.DB or *pie.Model:
// what that package's Layout takes. It is not a copy, as Config is not.
func (d *Diagram) Model() any { return d.model }

// Layout lays the diagram out as a scene, for a backend to draw, with the layout of its
// type; nil options are the zero Options. It snaps the scene to o.Snap, which the diagram
// packages' own Layout functions leave to their caller. Its errors are *diagram.Error: one a
// layout gives, as a sankey diagram's circular link, or one of Code diagram.Limit for a
// scene that would hold a number that is not finite, as numbers at the edge of what a
// float64 holds can give, where Pele draws NaN.
func (d *Diagram) Layout(o *layout.Options) (*scene.Scene, error) {
	if o == nil {
		o = &layout.Options{}
	}
	sc, err := d.kind.layout(d.model, d.config, o)
	if err != nil {
		return nil, err
	}
	if o.Snap > 0 {
		sc.Snap(o.Snap)
	}
	if !sc.Finite() {
		return nil, diagram.NewError("Diagram cannot be laid out in finite numbers.", diagram.Limit, d.typ)
	}
	return sc, nil
}

// Layout reads a diagram's text and lays it out as a scene, for a backend to draw: Parse,
// with the default limits and no configuration of the host's, and then Diagram.Layout. Its
// errors are theirs.
func Layout(src string, o *layout.Options) (*scene.Scene, error) {
	d, err := Parse(src, diagram.Options{})
	if err != nil {
		return nil, err
	}
	return d.Layout(o)
}
