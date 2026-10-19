package text

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

// The fixtures are what Pele makes of the same cases; ./scripts/oracle.sh writes them.

type peleSpan struct {
	Text  string  `json:"text"`
	Style Face    `json:"style"`
	Icon  string  `json:"icon"`
	Width float64 `json:"width"`
}

type peleLabel struct {
	Lines      [][]peleSpan `json:"lines"`
	Widths     []float64    `json:"widths"`
	Width      float64      `json:"width"`
	Height     float64      `json:"height"`
	Size       float64      `json:"size"`
	LineHeight float64      `json:"lineHeight"`
}

type peleText struct {
	Measure []struct {
		Text  string  `json:"text"`
		Size  float64 `json:"size"`
		Face  Face    `json:"face"`
		Width float64 `json:"width"`
	} `json:"measure"`
	Label []struct {
		Text     string    `json:"text"`
		Markdown bool      `json:"markdown"`
		Size     float64   `json:"size"`
		MaxWidth float64   `json:"maxWidth"`
		Face     Face      `json:"face"`
		Icons    []string  `json:"icons"`
		Label    peleLabel `json:"label"`
	} `json:"label"`
}

func loadPele(t *testing.T) peleText {
	t.Helper()
	data, err := os.ReadFile("testdata/pele/text.json")
	if err != nil {
		t.Fatal(err)
	}
	var p peleText
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestArialMatchesPele(t *testing.T) {
	for _, c := range loadPele(t).Measure {
		if got := Arial.Width(c.Text, c.Size, c.Face); got != c.Width {
			t.Errorf("Width(%q, %v, %v) = %v, Pele %v", c.Text, c.Size, c.Face, got, c.Width)
		}
	}
}

func TestLayoutMatchesPele(t *testing.T) {
	cases := loadPele(t).Label
	if len(cases) < 100 {
		t.Fatalf("only %d cases", len(cases))
	}
	for _, c := range cases {
		got := Layout(c.Text, Options{
			Measurer: Arial,
			Size:     c.Size,
			MaxWidth: c.MaxWidth,
			Face:     c.Face,
			Markdown: c.Markdown,
			Icons:    func(name string) bool { return slices.Contains(c.Icons, name) },
		})
		if !sameLabel(got, c.Label) {
			g, _ := json.Marshal(got)
			w, _ := json.Marshal(c.Label)
			t.Errorf("Layout(%q, markdown %v, size %v, max %v, face %v)\n got %s\nPele %s", c.Text, c.Markdown, c.Size, c.MaxWidth, c.Face, g, w)
		}
	}
}

func sameLabel(got Label, want peleLabel) bool {
	if got.Width != want.Width || got.Height != want.Height || got.Size != want.Size || got.LineHeight != want.LineHeight {
		return false
	}
	if !slices.Equal(got.Widths, want.Widths) || len(got.Lines) != len(want.Lines) {
		return false
	}
	for i, line := range got.Lines {
		if len(line) != len(want.Lines[i]) {
			return false
		}
		for j, s := range line {
			w := want.Lines[i][j]
			if s.Text != w.Text || s.Face != w.Style || s.Icon != w.Icon || s.Width != w.Width {
				return false
			}
		}
	}
	return true
}
