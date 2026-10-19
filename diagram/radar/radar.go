// Package radar parses Mermaid's radar charts and lays them out, as Pele's
// src/diagrams/radar does: a tokenizer and a parser that read what Mermaid's Langium grammar
// reads, buildRadar, which puts each curve's values in the order of the axes, and Layout,
// which lays the chart out as a scene: a graticule, the axes and their labels, a curve for
// each series, and a legend.
//
// Values are numbers as JavaScript's are, float64, as a value may be written with more digits
// than any number holds.
package radar

import (
	"math"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

// Axis is an axis of the chart, Pele's RadarAxis: its name, and the label it is drawn with,
// which is its name when the text gives none.
type Axis struct {
	Name, Label string
}

// Curve is a series, Pele's RadarCurve: its name, its label, and its values, in the order of
// the axes when the text names them.
type Curve struct {
	Name, Label string
	Entries     []float64
}

// Options are the chart's own options, Pele's RadarOptions. Max is NaN where Pele leaves it
// null, for the largest value to give it; a value read from the text is never NaN.
// Graticule is "circle" or "polygon".
type Options struct {
	ShowLegend bool
	Ticks      float64
	Max, Min   float64
	Graticule  string
}

// Model is a parsed radar chart, Pele's RadarModel. A string Pele leaves undefined is empty.
type Model struct {
	Title, AccTitle, AccDescr string
	Axes                      []Axis
	Curves                    []Curve
	Options                   Options
}

// maxTicks is the most rings a chart's graticule has, as Pele's MAX_TICKS.
const maxTicks = 32

func invalid(message string) *diagram.Error {
	return diagram.NewError(message, diagram.Semantic, "radar")
}

// buildRadar makes the model of a radar chart's tree, as Pele's buildRadar does: a curve
// whose values name their axes has them put in the order of the axes, the first value for
// each axis counting, and an option given more than once takes its last value.
func buildRadar(tree *ast, title string) (*Model, error) {
	m := &Model{
		Axes:    []Axis{},
		Curves:  []Curve{},
		Options: Options{ShowLegend: true, Ticks: 5, Max: math.NaN(), Min: 0, Graticule: "circle"},
	}
	for _, a := range tree.axes {
		axis := Axis{Name: a.name, Label: a.name}
		if a.hasLabel {
			axis.Label = a.label
		}
		m.Axes = append(m.Axes, axis)
	}

	for _, c := range tree.curves {
		var entries []float64
		if !c.entries[0].hasAxis {
			for _, e := range c.entries {
				entries = append(entries, e.value)
			}
		} else {
			if len(m.Axes) == 0 {
				return nil, invalid("Axes must be populated before curves for reference entries")
			}
			byAxis := map[string]float64{}
			for _, e := range c.entries {
				if _, ok := byAxis[e.axis]; e.hasAxis && !ok {
					byAxis[e.axis] = e.value
				}
			}
			for _, axis := range m.Axes {
				value, ok := byAxis[axis.Name]
				if !ok {
					return nil, invalid("Missing entry for axis " + axis.Label)
				}
				entries = append(entries, value)
			}
		}
		curve := Curve{Name: c.name, Label: c.name, Entries: entries}
		if c.hasLabel {
			curve.Label = c.label
		}
		m.Curves = append(m.Curves, curve)
	}

	o := &m.Options
	for _, option := range tree.options {
		switch option.name {
		case "showLegend":
			o.ShowLegend = option.value.(bool)
		case "graticule":
			o.Graticule = option.value.(string)
		case "ticks":
			o.Ticks = option.value.(float64)
		case "max":
			o.Max = option.value.(float64)
		case "min":
			o.Min = option.value.(float64)
		}
	}
	if o.Ticks > maxTicks {
		o.Ticks = maxTicks
	}

	m.Title = tree.title
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

// Parse parses a radar chart's text, as the parse of Pele's radar diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for none.
// An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	tree, err := parseRadar(source)
	if err != nil {
		return nil, err
	}
	return buildRadar(tree, title)
}

// ParsePrepared parses a prepared radar chart.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}
