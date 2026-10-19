package xychart

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// xyText is a piece of text of the chart, Pele's XyText; Mermaid's markdown strings are
// never read, so all of it is plain.
type xyText struct{ text string }

// dataPoint is a value of a series as written, with its label, Pele's DataPoint.
type dataPoint struct {
	value float64
	label string
}

// word is the tokens that run together into a piece of text, Pele's WORD, and eol those that
// end a statement, its EOL.
var word, eol [tEOF + 1]bool

func init() {
	for _, t := range []int{tAMP, tNUM, tALPHA, tPLUS, tEQUALS, tMULT, tDOT, tBRKT, tMINUS, tUNDERSCORE} {
		word[t] = true
	}
	for _, t := range []int{tNEWLINE, tSEMI, tEOF} {
		eol[t] = true
	}
}

// parseFailure carries a syntax error out of the parser.
type parseFailure struct{ err *diagram.Error }

// parseXyChart parses an XY chart's text into m, as Pele's parseXyChart does.
func parseXyChart(src string, m *Model) (err error) {
	tk := tokenize(src)
	types, texts := tk.Types, tk.Texts
	i := 0
	defer func() {
		if r := recover(); r != nil {
			f, ok := r.(parseFailure)
			if !ok {
				panic(r)
			}
			err = f.err
		}
	}()

	fail := func(expected string) {
		t := types[i]
		if t == tERROR {
			panic(parseFailure{common.SyntaxError("xychart", src, tk.Starts[i], "", true)})
		}
		panic(parseFailure{common.SyntaxError("xychart", src, tk.Starts[i], "Expecting "+expected+", got '"+tokenNames[t]+"'", false)})
	}
	expect := func(typ int) string {
		if types[i] != typ {
			fail("'" + tokenNames[typ] + "'")
		}
		i++
		return texts[i-1]
	}
	isText := func(t int) bool { return t == tSTR || word[t] }
	text := func() xyText {
		if !isText(types[i]) {
			fail("'STR', 'ALPHA', 'NUM'")
		}
		s := texts[i]
		i++
		if types[i-1] != tSTR {
			for word[types[i]] {
				s += texts[i]
				i++
			}
		}
		return xyText{s}
	}
	axisRange := func() (float64, float64) {
		lo := chars.Number(expect(tNUMBER_WITH_DECIMAL))
		expect(tARROW_DELIMITER)
		return lo, chars.Number(expect(tNUMBER_WITH_DECIMAL))
	}
	points := func() []dataPoint {
		var out []dataPoint
		expect(tSQUARE_BRACES_START)
		for {
			value := chars.Number(expect(tNUMBER_WITH_DECIMAL))
			label := ""
			if types[i] == tSTR {
				label = texts[i]
				i++
			}
			out = append(out, dataPoint{value, label})
			if types[i] != tCOMMA {
				break
			}
			i++
		}
		expect(tSQUARE_BRACES_END)
		return out
	}

	for {
		if eol[types[i]] {
			i++
		} else if types[i] != tXYCHART {
			break
		} else if i++; types[i] == tCHART_ORIENTATION {
			m.setOrientation(texts[i])
			i++
		}
	}

	for types[i] != tEND {
		t := types[i]
		i++
		switch t {
		case tTitle:
			m.setDiagramTitle(chars.Trim(text().text))
		case tX_AXIS, tY_AXIS:
			var title *xyText
			if isText(types[i]) {
				tt := text()
				title = &tt
			}
			if t == tX_AXIS && types[i] == tSQUARE_BRACES_START {
				var categories []xyText
				for {
					i++
					categories = append(categories, text())
					if types[i] != tCOMMA {
						break
					}
				}
				expect(tSQUARE_BRACES_END)
				m.setXAxisBand(categories)
			} else if types[i] == tNUMBER_WITH_DECIMAL || title == nil {
				lo, hi := axisRange()
				if t == tX_AXIS {
					m.setXAxisRangeData(lo, hi)
				} else {
					m.setYAxisRangeData(lo, hi)
				}
			}
			if title == nil {
				title = &xyText{}
			}
			if t == tX_AXIS {
				m.setXAxisTitle(*title)
			} else {
				m.setYAxisTitle(*title)
			}
		case tLINE, tBAR:
			title := xyText{}
			if types[i] != tSQUARE_BRACES_START {
				title = text()
			}
			data := points()
			if t == tLINE {
				m.setLineData(title, data)
			} else {
				m.setBarData(title, data)
			}
		case tAccTitle:
			m.setAccTitle(chars.Trim(expect(tAccTitleValue)))
		case tAccDescr:
			m.setAccDescription(chars.Trim(expect(tAccDescrValue)))
		case tAccDescrMultilineValue:
			m.setAccDescription(chars.Trim(texts[i-1]))
		default:
			i--
			fail("'title', 'X_AXIS', 'Y_AXIS', 'LINE', 'BAR', 'acc_title', 'acc_descr', 'acc_descr_multiline_value'")
		}
		for eol[types[i]] {
			i++
		}
	}
	return nil
}

// The methods of Pele's XyChartDb.

func (m *Model) setOrientation(orientation string) {
	if orientation == "horizontal" {
		m.Orientation = "horizontal"
	} else {
		m.Orientation = "vertical"
	}
}

func (m *Model) setXAxisTitle(title xyText) { m.XAxis.Title = chars.Trim(title.text) }

func (m *Model) setXAxisRangeData(lo, hi float64) {
	m.XAxis = Axis{Type: "linear", Title: m.XAxis.Title, Min: lo, Max: hi}
	m.HasSetXAxis = true
}

func (m *Model) setXAxisBand(categories []xyText) {
	names := make([]string, len(categories))
	for k, c := range categories {
		names[k] = chars.Trim(c.text)
	}
	m.XAxis = Axis{Type: "band", Title: m.XAxis.Title, Categories: names}
	m.HasSetXAxis = true
}

func (m *Model) setYAxisTitle(title xyText) { m.YAxis.Title = chars.Trim(title.text) }

func (m *Model) setYAxisRangeData(lo, hi float64) {
	m.YAxis = Axis{Type: "linear", Title: m.YAxis.Title, Min: lo, Max: hi}
	m.HasSetYAxis = true
}

// plotData places the values of a series on the x axis, as XyChartDb's plotData does: the
// first series fixes the x range when no x-axis statement did, values past the categories
// are left out, and the y range takes in the values until a y-axis statement sets it.
func (m *Model) plotData(values []float64) []Datum {
	if len(values) == 0 {
		return []Datum{}
	}
	if !m.HasSetXAxis {
		m.setXAxisRangeData(1, float64(len(values)))
	}
	x := m.XAxis
	if x.Type == "band" && len(values) > len(x.Categories) {
		values = values[:len(x.Categories)]
	}
	if !m.HasSetYAxis {
		lo, hi := m.YAxis.Min, m.YAxis.Max
		for _, value := range values {
			if value < lo {
				lo = value
			}
			if value > hi {
				hi = value
			}
		}
		m.YAxis = Axis{Type: "linear", Title: m.YAxis.Title, Min: lo, Max: hi}
	}
	if x.Type == "band" {
		out := make([]Datum, len(x.Categories))
		for k, category := range x.Categories {
			out[k] = Datum{category, nan}
			if k < len(values) {
				out[k].Value = values[k]
			}
		}
		return out
	}
	if len(values) == 1 {
		return []Datum{{chars.NumberString(x.Min), values[0]}}
	}
	step := (x.Max - x.Min) / float64(len(values)-1)
	out := make([]Datum, len(values))
	for k, value := range values {
		out[k] = Datum{chars.NumberString(x.Min + float64(float64(k)*step)), value}
	}
	return out
}

// addPlot adds a series, as XyChartDb's addPlot does. Mermaid accepts labels on the points of
// a bar series and ignores them.
func (m *Model) addPlot(typ string, title xyText, data []dataPoint) {
	values := make([]float64, len(data))
	for k, d := range data {
		values[k] = d.value
	}
	plot := Plot{Type: typ, Title: chars.Trim(title.text), Data: m.plotData(values)}
	if typ == "line" {
		labelled := false
		for _, d := range data {
			if chars.Trim(d.label) != "" {
				labelled = true
			}
		}
		if labelled {
			plot.PointLabels = make([]string, len(data))
			for k, d := range data {
				plot.PointLabels[k] = chars.Trim(d.label)
			}
		}
	}
	m.Plots = append(m.Plots, plot)
}

func (m *Model) setLineData(title xyText, data []dataPoint) { m.addPlot("line", title, data) }

func (m *Model) setBarData(title xyText, data []dataPoint) { m.addPlot("bar", title, data) }

func (m *Model) setDiagramTitle(text string) { m.Title = text }

// setAccTitle sets the accessible title, without the white space it starts with.
func (m *Model) setAccTitle(text string) { m.AccTitle = chars.TrimStart(text) }

// setAccDescription sets the accessible description, each line break with the white space
// after it, line breaks included, made one line break.
func (m *Model) setAccDescription(text string) { m.AccDescr = chars.DropIndents(text) }
