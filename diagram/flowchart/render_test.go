package flowchart

import (
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
)

// The fixture is what Pele's render draws of flowcharts, read back out of its SVG as one line
// per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. The test writes the same
// lines from the scene with pelesvg.FlowLines, numbers as Pele's SVG writes them.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		MaxWidth float64  `json:"maxWidth"`
		Padding  *float64 `json:"padding"`
		Lines    []string `json:"lines"`
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	if err := fixture.Load("testdata/pele/render.json", &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 1000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
		pre, err := diagram.Prepare(c.Src, diagram.Options{})
		if err != nil {
			t.Errorf("Prepare(%q): %v", c.Src, err)
			continue
		}
		db, err := ParsePrepared(pre)
		if err != nil {
			t.Errorf("ParsePrepared(%q): %v", c.Src, err)
			continue
		}
		got := pelesvg.FlowLines(Layout(db, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, Padding: c.Padding}))
		want := pelesvg.TagsLast(c.Lines, "clickable")
		if i := pelesvg.FirstDiff(got, want); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, padding %v), line %d\n got %s\nPele %s", c.Src, c.MaxWidth, (&layout.Options{Padding: c.Padding}).Pad(), i, pelesvg.At(got, i), pelesvg.At(want, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}
