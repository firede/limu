package common

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/text"
)

// The fixture is what Pele makes of the same cases; ./scripts/oracle.sh writes it.

type peleSpan struct {
	Text  string    `json:"text"`
	Style text.Face `json:"style"`
	Icon  string    `json:"icon"`
	Width float64   `json:"width"`
}

type peleLabel struct {
	Lines      [][]peleSpan `json:"lines"`
	Widths     []float64    `json:"widths"`
	Width      float64      `json:"width"`
	Height     float64      `json:"height"`
	Size       float64      `json:"size"`
	LineHeight float64      `json:"lineHeight"`
}

type peleCommon struct {
	Generics []struct{ Input, Output string } `json:"generics"`
	Labels   []struct {
		Text    string    `json:"text"`
		Size    float64   `json:"size"`
		Max     float64   `json:"max"`
		Face    text.Face `json:"face"`
		Plain   peleLabel `json:"plain"`
		Fit     peleLabel `json:"fit"`
		Literal peleLabel `json:"literal"`
	} `json:"labels"`
}

func loadPele(t *testing.T) peleCommon {
	t.Helper()
	var p peleCommon
	if err := fixture.Load("testdata/pele/common.json", &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestGenericsMatchPele(t *testing.T) {
	p := loadPele(t)
	if len(p.Generics) < 1000 {
		t.Fatalf("only %d cases", len(p.Generics))
	}
	for _, c := range p.Generics {
		if got := ParseGenericTypes(c.Input); got != c.Output {
			t.Errorf("ParseGenericTypes(%q) = %q, Pele %q", c.Input, got, c.Output)
		}
	}
}

func TestLabelsMatchPele(t *testing.T) {
	p := loadPele(t)
	if len(p.Labels) < 500 {
		t.Fatalf("only %d cases", len(p.Labels))
	}
	for _, c := range p.Labels {
		for _, f := range []struct {
			name string
			got  text.Label
			want peleLabel
		}{
			{"PlainLabel", PlainLabel(c.Text, text.Arial, c.Size, c.Max, c.Face), c.Plain},
			{"FitLabel", FitLabel(c.Text, text.Arial, c.Size, c.Max, c.Face), c.Fit},
			{"LiteralLabel", LiteralLabel(c.Text, text.Arial, c.Size, c.Max, c.Face), c.Literal},
		} {
			if !sameLabel(f.got, f.want) {
				g, _ := json.Marshal(f.got)
				w, _ := json.Marshal(f.want)
				t.Errorf("%s(%q, size %v, max %v, face %v)\n got %s\nPele %s", f.name, c.Text, c.Size, c.Max, c.Face, g, w)
			}
		}
	}
}

func sameLabel(got text.Label, want peleLabel) bool {
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
