// Package wardley parses Mermaid's Wardley maps and lays them out, as Pele's
// src/diagrams/wardley does: a tokenizer and a parser that read what Mermaid's Langium
// grammar reads, populate and the builder, which turn the tree into the map's model, and
// Layout, which lays the map out as a scene.
//
// Places are percentages, as Pele keeps them: a node's X is its evolution and its Y its
// visibility, each from 0 to 100.
package wardley

import (
	"strings"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// Model is a parsed Wardley map, Pele's WardleyModel. A string Pele leaves undefined is
// empty.
type Model struct {
	Title, AccTitle, AccDescr string
	Nodes                     []*Node
	Links                     []Link
	Trends                    []Trend
	Pipelines                 []Pipeline
	Annotations               []Annotation
	Notes                     []Note
	Accelerators              []Force
	Deaccelerators            []Force
	// AnnotationsBox is where the list of annotations goes, or nil for under the map.
	AnnotationsBox *Point
	Axes           Axes
	// Size is the size the text asks for, or nil.
	Size *Size
}

// toPercent is a place as a percentage, as Pele's toPercent reads it: values up to 1 are
// fractions, and larger ones are percentages already.
func toPercent(value float64, context string) (float64, error) {
	normalized := value
	if value <= 1 {
		normalized = value * 100
	}
	if normalized < 0 || normalized > 100 {
		return 0, semantic(context + " must be between 0-1 (decimal) or 0-100 (percentage). Received: " + chars.NumberString(value))
	}
	return normalized, nil
}

// toCoordinates is a place given by visibility and evolution, as Pele's toCoordinates reads
// it: x is the evolution, read first, and y the visibility.
func toCoordinates(visibility, evolution float64, context string) (x, y float64, err error) {
	if x, err = toPercent(evolution, context+" evolution"); err != nil {
		return 0, 0, err
	}
	if y, err = toPercent(visibility, context+" visibility"); err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

// flowFromPort is the flow a port on a link gives, as Pele's flowFromPort reads it.
func flowFromPort(port string) string {
	switch port {
	case "+<>":
		return "bidirectional"
	case "+<":
		return "backward"
	case "+>":
		return "forward"
	}
	return ""
}

// flowFromArrow is the flow and the label an arrow gives, as Pele's flowFromArrow reads
// them: only an arrow that starts with a plus has either, and its label is what is between
// the quotes after the plus. The flow is read from the whole arrow, its label too.
func flowFromArrow(arrow string) (flow, label string, hasLabel bool) {
	if !strings.HasPrefix(arrow, "+") {
		return "", "", false
	}
	if end := strings.IndexByte(arrow[min(2, len(arrow)):], '\''); len(arrow) > 1 && arrow[1] == '\'' && end >= 0 {
		label, hasLabel = arrow[2:2+end], true
	}
	switch {
	case strings.Contains(arrow, "<>"):
		flow = "bidirectional"
	case strings.Contains(arrow, "<"):
		flow = "backward"
	case strings.Contains(arrow, ">"):
		flow = "forward"
	}
	return flow, label, hasLabel
}

// offsets are a label's offsets, as Pele's offsets gives them; a minus sign before a zero
// makes it -0, as in JavaScript.
func offsets(label *labelAst) (x, y *float64) {
	if label == nil {
		return nil, nil
	}
	sign := func(neg bool) float64 {
		if neg {
			return -1
		}
		return 1
	}
	ox, oy := sign(label.negX)*label.offsetX, sign(label.negY)*label.offsetY
	return &ox, &oy
}

// populate fills the builder from the tree, as Pele's populate does, which follows Mermaid's
// populateDb: each kind of statement in turn, in Mermaid's order.
func populate(tree *ast, db *builder) error {
	if tree.size != nil {
		db.setSize(tree.size[0], tree.size[1])
	}

	if tree.evolution != nil {
		stages := make([]string, 0, len(tree.evolution))
		boundaries := []float64{}
		for _, stage := range tree.evolution {
			if stage.hasSecond {
				stages = append(stages, chars.Trim(stage.name)+" / "+chars.Trim(stage.secondName))
			} else {
				stages = append(stages, chars.Trim(stage.name))
			}
		}
		for _, stage := range tree.evolution {
			if stage.hasBoundary {
				boundaries = append(boundaries, stage.boundary)
			}
		}
		db.setAxes(stages, boundaries)
	}

	for _, anchor := range tree.anchors {
		x, y, err := toCoordinates(anchor.a, anchor.b, `Anchor "`+anchor.name+`"`)
		if err != nil {
			return err
		}
		db.addNode(nodeSet{Node: Node{ID: anchor.name, Label: anchor.name, X: x, Y: y, Class: "anchor"}})
	}

	for _, c := range tree.components {
		x, y, err := toCoordinates(c.visibility, c.evolution, `Component "`+c.name+`"`)
		if err != nil {
			return err
		}
		ox, oy := offsets(c.label)
		db.addNode(nodeSet{Node: Node{
			ID: c.name, Label: c.name, X: x, Y: y, Class: "component",
			LabelOffsetX: ox, LabelOffsetY: oy,
			Inertia: c.inertia, HasInertia: true, SourceStrategy: c.strategy,
		}, setsInertia: true})
	}

	for _, note := range tree.notes {
		x, y, err := toCoordinates(note.a, note.b, `Note "`+note.name+`"`)
		if err != nil {
			return err
		}
		db.addNote(Note{note.name, x, y})
	}

	for _, pipeline := range tree.pipelines {
		parent := db.getNode(pipeline.parent)
		if parent == nil {
			return semantic(`Pipeline "` + pipeline.parent + `" must reference an existing component with coordinates.`)
		}
		y := parent.Y
		db.startPipeline(pipeline.parent)
		for _, c := range pipeline.components {
			id := pipeline.parent + "_" + c.name
			ox, oy := offsets(c.label)
			x, err := toPercent(c.evolution, `Pipeline component "`+c.name+`" evolution`)
			if err != nil {
				return err
			}
			db.addNode(nodeSet{Node: Node{
				ID: id, Label: c.name, X: x, Y: y, Class: "pipeline-component",
				LabelOffsetX: ox, LabelOffsetY: oy,
			}, setsInertia: true})
			db.addPipelineComponent(pipeline.parent, id)
		}
	}

	for _, link := range tree.links {
		dashed := link.hasArrow && (strings.Contains(link.arrow, "-.->") || strings.Contains(link.arrow, ".-."))
		arrowFlow, label, hasLabel := flowFromArrow(link.arrow)
		flow := flowFromPort(link.fromPort)
		if flow == "" {
			flow = flowFromPort(link.toPort)
		}
		if flow == "" {
			flow = arrowFlow
		}
		if !hasLabel {
			label, hasLabel = link.linkLabel, link.hasLinkLabel
		}
		db.addLink(Link{
			Source: db.resolveNodeID(link.from),
			Target: db.resolveNodeID(link.to),
			Dashed: dashed, Label: label, HasLabel: hasLabel, Flow: flow,
		})
	}

	for _, evolve := range tree.evolves {
		if node := db.getNode(evolve.component); node != nil {
			x, err := toPercent(evolve.target, `Evolve target for "`+evolve.component+`"`)
			if err != nil {
				return err
			}
			db.addTrend(Trend{NodeID: evolve.component, TargetX: x, TargetY: node.Y})
		}
	}

	if len(tree.annotations) > 0 {
		x, y, err := toCoordinates(tree.annotations[0][0], tree.annotations[0][1], "Annotations box")
		if err != nil {
			return err
		}
		db.setAnnotationsBox(x, y)
	}

	for _, a := range tree.annotation {
		x, y, err := toCoordinates(a.x, a.y, "Annotation "+chars.NumberString(a.number))
		if err != nil {
			return err
		}
		db.addAnnotation(Annotation{Number: a.number, Coordinates: []Point{{x, y}}, Text: a.text})
	}

	for _, f := range tree.accelerators {
		x, y, err := toCoordinates(f.a, f.b, `Accelerator "`+f.name+`"`)
		if err != nil {
			return err
		}
		db.addAccelerator(Force{f.name, x, y})
	}

	for _, f := range tree.deaccelerators {
		x, y, err := toCoordinates(f.a, f.b, `Deaccelerator "`+f.name+`"`)
		if err != nil {
			return err
		}
		db.addDeaccelerator(Force{f.name, x, y})
	}
	return nil
}

// buildWardley makes the model of a map's tree, as Pele's buildWardley does; title is the
// one its front matter gives, which the text's own takes the place of.
func buildWardley(tree *ast, title string) (*Model, error) {
	db := newBuilder()
	if err := populate(tree, db); err != nil {
		return nil, err
	}
	b := db.build()
	m := &Model{
		Title:          tree.title,
		Nodes:          b.Nodes,
		Links:          b.Links,
		Trends:         b.Trends,
		Pipelines:      b.Pipelines,
		Annotations:    b.Annotations,
		Notes:          b.Notes,
		Accelerators:   b.Accelerators,
		Deaccelerators: b.Deaccelerators,
		AnnotationsBox: b.AnnotationsBox,
		Axes:           b.Axes,
		Size:           b.Size,
	}
	if m.Title == "" {
		m.Title = title
	}
	if tree.accTitle != "" {
		m.AccTitle = chars.TrimStart(tree.accTitle)
	}
	if tree.accDescr != "" {
		m.AccDescr = chars.DropIndents(tree.accDescr)
	}
	return m, nil
}

// Parse parses a Wardley map's text, as the parse of Pele's wardley diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. An error is a *diagram.Error.
func Parse(source string, title string) (*Model, error) {
	tree, err := parseWardley(source)
	if err != nil {
		return nil, err
	}
	return buildWardley(tree, title)
}

// ParsePrepared parses a prepared Wardley map.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}
