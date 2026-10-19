package wardley

import "github.com/firede/limu/diagram"

// Node is a thing placed on the map, Pele's WardleyNode: an anchor, a component, or a
// component of a pipeline. X is its evolution and Y its visibility, as percentages.
type Node struct {
	ID, Label string
	X, Y      float64
	// Class is "anchor", "component" or "pipeline-component".
	Class string
	// LabelOffsetX and LabelOffsetY move its label, when the text gives them.
	LabelOffsetX, LabelOffsetY   *float64
	InPipeline, IsPipelineParent bool
	// Inertia is whether the component resists change; HasInertia whether anything said so
	// either way, as an anchor does not.
	Inertia, HasInertia bool
	// SourceStrategy is "build", "buy", "outsource" or "market", or "" for none.
	SourceStrategy string
}

// Link is a line from one node to another, Pele's WardleyLink. Flow is "forward",
// "backward" or "bidirectional", or "" for none.
type Link struct {
	Source, Target string
	Dashed         bool
	Label          string
	HasLabel       bool
	Flow           string
}

// Trend is where a component is evolving to, Pele's WardleyTrend.
type Trend struct {
	NodeID           string
	TargetX, TargetY float64
}

// Pipeline is a component and the components that make it up, Pele's WardleyPipeline.
type Pipeline struct {
	NodeID       string
	ComponentIDs []string
}

// Point is a place on the map, as percentages of evolution and visibility.
type Point struct{ X, Y float64 }

// Annotation is a numbered mark at places on the map, Pele's WardleyAnnotation.
type Annotation struct {
	Number      float64
	Coordinates []Point
	Text        string
}

// Note is text placed on the map, Pele's WardleyNote.
type Note struct {
	Text string
	X, Y float64
}

// Force is an accelerator or a deaccelerator, Pele's WardleyForce.
type Force struct {
	Name string
	X, Y float64
}

// Axes are the labels and stages of the axes, Pele's WardleyAxesConfig. Stages and
// StageBoundaries are nil when the text gives none; each boundary is the right edge of a
// stage, from 0 to 1.
type Axes struct {
	XLabel, YLabel  *string
	Stages          []string
	StageBoundaries []float64
}

// Size is the size of the map the text asks for.
type Size struct{ Width, Height float64 }

// buildResult is Pele's WardleyBuildResult.
type buildResult struct {
	Nodes          []*Node
	Links          []Link
	Trends         []Trend
	Pipelines      []Pipeline
	Annotations    []Annotation
	Notes          []Note
	Accelerators   []Force
	Deaccelerators []Force
	AnnotationsBox *Point
	Axes           Axes
	Size           *Size
}

// nodeSet is what a node added to the map sets, as the object Pele hands to addNode holds
// it: an anchor's says nothing of inertia nor strategy, and leaves them as they were.
type nodeSet struct {
	Node
	setsInertia bool
}

// builder is Pele's WardleyBuilder. Its maps keep the order keys were first added in, as
// JavaScript's do.
type builder struct {
	nodes          map[string]*Node
	order          []string
	labels         map[string]string
	links          []Link
	trends         map[string]*Trend
	trendOrder     []string
	pipelines      map[string]*Pipeline
	pipelineOrder  []string
	annotations    []Annotation
	notes          []Note
	accelerators   []Force
	deaccelerators []Force
	annotationsBox *Point
	axes           Axes
	size           *Size
}

func newBuilder() *builder {
	return &builder{nodes: map[string]*Node{}, trends: map[string]*Trend{}, pipelines: map[string]*Pipeline{}}
}

// addNode adds a node, or sets one again, as Pele's addNode does: a node added again keeps
// its class and label offsets unless the new one sets them, and whether it is in a
// pipeline.
func (b *builder) addNode(set nodeSet) {
	node := set.Node
	existing, ok := b.nodes[node.ID]
	if !ok {
		existing = &Node{ID: node.ID, Label: node.Label}
		b.order = append(b.order, node.ID)
	}
	next := *existing
	next.ID, next.Label, next.X, next.Y = node.ID, node.Label, node.X, node.Y
	if node.Class != "" {
		next.Class = node.Class
	}
	if node.LabelOffsetX != nil {
		next.LabelOffsetX = node.LabelOffsetX
	}
	if node.LabelOffsetY != nil {
		next.LabelOffsetY = node.LabelOffsetY
	}
	if set.setsInertia {
		next.Inertia, next.HasInertia, next.SourceStrategy = node.Inertia, node.HasInertia, node.SourceStrategy
	}
	b.nodes[node.ID] = &next
	b.labels = nil
}

func (b *builder) addLink(link Link) { b.links = append(b.links, link) }

func (b *builder) addTrend(trend Trend) {
	if _, ok := b.trends[trend.NodeID]; !ok {
		b.trendOrder = append(b.trendOrder, trend.NodeID)
	}
	b.trends[trend.NodeID] = &trend
}

func (b *builder) startPipeline(nodeID string) {
	if _, ok := b.pipelines[nodeID]; !ok {
		b.pipelineOrder = append(b.pipelineOrder, nodeID)
	}
	b.pipelines[nodeID] = &Pipeline{NodeID: nodeID, ComponentIDs: []string{}}
	if node := b.nodes[nodeID]; node != nil {
		node.IsPipelineParent = true
	}
}

func (b *builder) addPipelineComponent(pipelineNodeID, componentID string) {
	if p := b.pipelines[pipelineNodeID]; p != nil {
		p.ComponentIDs = append(p.ComponentIDs, componentID)
	}
	if node := b.nodes[componentID]; node != nil {
		node.InPipeline = true
	}
}

func (b *builder) addAnnotation(a Annotation) { b.annotations = append(b.annotations, a) }
func (b *builder) addNote(n Note)             { b.notes = append(b.notes, n) }
func (b *builder) addAccelerator(f Force)     { b.accelerators = append(b.accelerators, f) }
func (b *builder) addDeaccelerator(f Force)   { b.deaccelerators = append(b.deaccelerators, f) }
func (b *builder) setAnnotationsBox(x, y float64) {
	b.annotationsBox = &Point{x, y}
}

// setAxes sets the stages of the axes, as Pele's setAxes does with the only part populate
// gives it.
func (b *builder) setAxes(stages []string, boundaries []float64) {
	b.axes.Stages, b.axes.StageBoundaries = stages, boundaries
}

func (b *builder) setSize(width, height float64) { b.size = &Size{width, height} }

func (b *builder) getNode(id string) *Node { return b.nodes[id] }

// resolveNodeID is a name as a node's id, as Pele's resolveNodeId finds it: the name, if a
// node has it as its id, or else the id of the first node with it as its label. Pipeline
// components have ids like "Parent_Child" and are linked by label.
func (b *builder) resolveNodeID(name string) string {
	if _, ok := b.nodes[name]; ok {
		return name
	}
	if b.labels == nil {
		b.labels = map[string]string{}
		for _, id := range b.order {
			label := b.nodes[id].Label
			if _, ok := b.labels[label]; !ok {
				b.labels[label] = id
			}
		}
	}
	if id, ok := b.labels[name]; ok {
		return id
	}
	return name
}

// build is the map the builder holds, as Pele's build gives it.
func (b *builder) build() *buildResult {
	out := &buildResult{
		Links:          append([]Link{}, b.links...),
		Annotations:    append([]Annotation{}, b.annotations...),
		Notes:          append([]Note{}, b.notes...),
		Accelerators:   append([]Force{}, b.accelerators...),
		Deaccelerators: append([]Force{}, b.deaccelerators...),
		AnnotationsBox: b.annotationsBox,
		Axes:           b.axes,
		Size:           b.size,
	}
	for _, id := range b.order {
		// Every node is added with a place, so none is without one, as Pele's build checks.
		out.Nodes = append(out.Nodes, b.nodes[id])
	}
	for _, id := range b.trendOrder {
		out.Trends = append(out.Trends, *b.trends[id])
	}
	for _, id := range b.pipelineOrder {
		out.Pipelines = append(out.Pipelines, *b.pipelines[id])
	}
	return out
}

func semantic(message string) *diagram.Error {
	return diagram.NewError(message, diagram.Semantic, "wardley")
}
