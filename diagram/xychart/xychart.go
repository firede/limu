// Package xychart parses Mermaid's XY charts and lays them out, as Pele's src/diagrams/xychart
// does: a lexer and a parser that make the decisions Mermaid's Jison parser makes, the model
// they fill, Pele's XyChartDb, and Layout, which lays out the axes, the bars and lines of each
// series, and a legend, as a scene.
//
// Offsets into the source are bytes of the Go string; the columns in errors are UTF-16 code
// units, as Pele's are.
package xychart

import (
	"math"

	"github.com/firede/limu/diagram"
)

// Axis is an axis of a chart, Pele's BandAxis or LinearAxis: Type is "band", an axis of
// categories, or "linear", an axis of numbers from Min to Max.
type Axis struct {
	Type       string
	Title      string
	Categories []string
	Min, Max   float64
}

// Datum is a value of a series with its place on the x axis: a category, or a number written
// as text, as JavaScript writes it. Value is NaN for a category with no value, which Pele
// leaves undefined; a value read from the text is never NaN.
type Datum struct {
	X     string
	Value float64
}

// Plot is a series, Pele's Plot: Type is "line" or "bar".
type Plot struct {
	Type  string
	Title string
	Data  []Datum
	// PointLabels are the labels of the points of a line, nil where Pele leaves them
	// undefined: for a bar series, and for a line none of whose points has a label.
	PointLabels []string
}

// Model is a parsed XY chart, Pele's XyChartModel. A string Pele may leave undefined is empty
// here where it is.
type Model struct {
	Title    string
	AccTitle string
	AccDescr string
	// Orientation is "vertical" or "horizontal" where the text says, and "" where it does
	// not, for the chart's config to decide.
	Orientation string
	XAxis       Axis
	YAxis       Axis
	HasSetXAxis bool
	// HasSetYAxis is false while the y range comes from the data and not from a y-axis
	// statement.
	HasSetYAxis bool
	Plots       []Plot
}

// newModel is a model with nothing in it, as Pele's XyChartDb starts.
func newModel() *Model {
	return &Model{
		XAxis: Axis{Type: "band", Categories: []string{}},
		YAxis: Axis{Type: "linear", Min: math.Inf(1), Max: math.Inf(-1)},
	}
}

// Parse parses an XY chart's text, as the parse of Pele's xychart diagram does: source is
// what diagram.Prepared.Source gives, and title the title its front matter gives, "" for
// none. An error is a *diagram.Error.
func Parse(source, title string) (*Model, error) {
	m := newModel()
	m.Title = title
	if err := parseXyChart(source, m); err != nil {
		return nil, err
	}
	return m, nil
}

// ParsePrepared parses a prepared XY chart.
func ParsePrepared(p *diagram.Prepared) (*Model, error) {
	return Parse(p.Source(false), p.Title)
}

// nan is the value of a category that has none.
var nan = math.NaN()
