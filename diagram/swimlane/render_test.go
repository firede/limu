package swimlane

import (
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
)

// The fixture is what Pele's render draws of swimlane diagrams, read back out of its SVG as
// one line per thing drawn, in the order drawn, lanes with their title bands and the turn of
// their titles; ./scripts/oracle.sh writes it with the swimlane suite. The test writes the
// same lines from the scene with pelesvg.FlowLines, numbers as Pele's SVG writes them.

type peleRender struct {
	Cases []struct {
		Src        string  `json:"src"`
		MaxWidth   float64 `json:"maxWidth"`
		Keep       bool    `json:"keep"`
		Breakpoint float64 `json:"breakpoint"`
		Unlimited  bool    `json:"unlimited"`
		fixture.Drawing
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 3000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for k, c := range p.Cases {
		opt := diagram.Options{}
		if c.Unlimited {
			opt.Limit = -1
		}
		pre, err := diagram.Prepare(c.Src, opt)
		if err != nil {
			t.Errorf("Prepare(%q): %v", c.Src, err)
			continue
		}
		if pre.Type != diagram.Swimlane {
			t.Errorf("Prepare(%q) is a %s", c.Src, pre.Type)
			continue
		}
		db, err := ParsePrepared(pre)
		if err != nil {
			t.Errorf("ParsePrepared(%q): %v", c.Src, err)
			continue
		}
		o := &layout.Options{MaxWidth: c.MaxWidth, KeepDirection: c.Keep, DirectionBreakpoint: c.Breakpoint}
		sc := Layout(db, pre.Config, o)
		if sc.Type != "swimlane" {
			t.Errorf("Layout(%q) is a scene of a %s", c.Src, sc.Type)
		}
		got := pelesvg.FlowLines(sc)
		if msg := f.Diff(k, &c.Drawing, got, pelesvg.TagsLast("clickable")); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v), %s", c.Src, c.MaxWidth, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}
