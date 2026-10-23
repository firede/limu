package svg

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"

	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/text"
)

// The fixture is what Pele's labelSvg and leftLabelSvg write of labels, read back as what a
// browser is given to place: a line for each run of text placed on its own, with its anchor,
// and a line for each icon. ./scripts/oracle.sh writes it, and the head of
// internal/oracle/suites/svg.mjs says how the lines are read. The test writes each label as a
// Middle and a Start label about the same point and reads its own SVG back the same way.

type peleSpan struct {
	Text  string    `json:"text"`
	Style text.Face `json:"style"`
	Icon  string    `json:"icon"`
	Width float64   `json:"width"`
}

type peleLabels struct {
	Cases []struct {
		Text  string `json:"text"`
		Label struct {
			Lines      [][]peleSpan `json:"lines"`
			Widths     []float64    `json:"widths"`
			Width      float64      `json:"width"`
			Height     float64      `json:"height"`
			Size       float64      `json:"size"`
			LineHeight float64      `json:"lineHeight"`
		} `json:"label"`
		Middle []string `json:"middle"`
		Start  []string `json:"start"`
	} `json:"cases"`
}

func TestLabelMatchesPele(t *testing.T) {
	var p peleLabels
	if err := fixture.Load("testdata/pele/svg.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 300 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	o := &Options{IconMarkup: func(string) string { return "<path/>" }}
	for _, c := range p.Cases {
		pl := c.Label
		l := text.Label{Widths: pl.Widths, Width: pl.Width, Height: pl.Height, Size: pl.Size, LineHeight: pl.LineHeight}
		for _, line := range pl.Lines {
			spans := []text.Span{}
			for _, s := range line {
				spans = append(spans, text.Span{Text: s.Text, Face: s.Style, Icon: s.Icon, Width: s.Width})
			}
			l.Lines = append(l.Lines, spans)
		}
		for _, a := range []struct {
			align scene.Align
			want  []string
		}{{scene.Middle, c.Middle}, {scene.Start, c.Start}} {
			sc := &scene.Scene{Type: "t", Width: 200, Height: 100, FontSize: l.Size, Items: []scene.Item{
				{Kind: scene.Text, Label: &l, At: scene.Point{X: 100, Y: 50}, Align: a.align},
			}}
			svg := Render(sc, o)
			got, err := readLabel(svg)
			if err != nil {
				t.Fatalf("reading %s: %v", svg, err)
			}
			if strings.Join(got, "\n") != strings.Join(a.want, "\n") {
				t.Errorf("label %q, align %d:\n got %q\nPele %q", c.Text, a.align, got, a.want)
			}
		}
	}
}

// readLabel reads the SVG of a label as internal/oracle/suites/svg.mjs reads Pele's: a line
// for each run of text placed on its own, "t anchor x,y spans", and then a line for each icon,
// "icon name x,y,width,height".
func readLabel(svg string) ([]string, error) {
	type el struct {
		bold, italic, mono, under bool
		shift, anchor             string
	}
	type piece struct{ face, text string }
	type run struct {
		anchor, x, y string
		spans        []piece
	}
	var runs []*run
	var icons []string
	var open []el
	var cur *run
	d := xml.NewDecoder(strings.NewReader(svg))
	skip := 0
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			at := func(name string) (string, bool) {
				for _, a := range tok.Attr {
					if a.Name.Local == name {
						return a.Value, true
					}
				}
				return "", false
			}
			if skip > 0 {
				skip++
				continue
			}
			name := tok.Name.Local
			if name == "svg" {
				if icon, ok := at("data-icon"); ok {
					x, _ := at("x")
					y, _ := at("y")
					w, _ := at("width")
					h, _ := at("height")
					icons = append(icons, "icon "+icon+" "+x+","+y+","+w+","+h)
					skip = 1
				}
				continue
			}
			if name != "text" && name != "tspan" {
				continue
			}
			up := el{anchor: "start"}
			if len(open) > 0 {
				up = open[len(open)-1]
			}
			e := up
			if _, ok := at("font-weight"); ok {
				e.bold = true
			}
			if s, _ := at("font-style"); s == "italic" {
				e.italic = true
			}
			if _, ok := at("font-family"); ok {
				e.mono = true
			}
			if s, _ := at("text-decoration"); s == "underline" {
				e.under = true
			}
			switch s, _ := at("baseline-shift"); s {
			case "sub":
				e.shift = "_"
			case "super":
				e.shift = "^"
			}
			if name == "text" {
				e.anchor = "start"
				if s, ok := at("text-anchor"); ok {
					e.anchor = s
				}
			}
			open = append(open, e)
			if x, ok := at("x"); ok {
				y, _ := at("y")
				anchor := e.anchor
				if s, ok := at("text-anchor"); ok {
					anchor = s
				}
				cur = &run{anchor: anchor, x: x, y: y}
				runs = append(runs, cur)
			}
		case xml.EndElement:
			if skip > 0 {
				skip--
				continue
			}
			if n := tok.Name.Local; (n == "text" || n == "tspan") && len(open) > 0 {
				open = open[:len(open)-1]
			}
		case xml.CharData:
			if skip > 0 || cur == nil || len(open) == 0 {
				continue
			}
			top := open[len(open)-1]
			face := ""
			if top.bold {
				face += "b"
			}
			if top.italic {
				face += "i"
			}
			if top.mono {
				face += "m"
			}
			if top.under {
				face += "u"
			}
			face += top.shift
			if n := len(cur.spans); n > 0 && cur.spans[n-1].face == face {
				cur.spans[n-1].text += string(tok)
			} else {
				cur.spans = append(cur.spans, piece{face, string(tok)})
			}
		}
	}
	var lines []string
	for _, r := range runs {
		var all, spans strings.Builder
		for _, s := range r.spans {
			all.WriteString(s.text)
			if s.face != "" {
				spans.WriteString("[" + s.face + ":" + s.text + "]")
			} else {
				spans.WriteString(s.text)
			}
		}
		// A run of nothing but spaces draws nothing.
		if strings.Trim(all.String(), " ") == "" {
			continue
		}
		lines = append(lines, "t "+r.anchor+" "+r.x+","+r.y+" "+spans.String())
	}
	return append(lines, icons...), nil
}
