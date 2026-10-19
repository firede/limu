package info

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// The fixture is what Pele's render draws of info diagrams, read back out of its SVG as one
// line per thing drawn, with every number in full; ./scripts/oracle.sh writes it. Pele draws
// in each case, in place of its own name and version, the text limu draws that the case
// gives. Each label is written as the point it is centered on and what is in it. The test
// writes the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src      string   `json:"src"`
		Text     string   `json:"text"`
		FontSize float64  `json:"fontSize"`
		Icons    bool     `json:"icons"`
		Lines    []string `json:"lines"`
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/render.json")
	if err != nil {
		t.Fatal(err)
	}
	var p peleRender
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 600 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for _, c := range p.Cases {
		pre, err := diagram.Prepare(c.Src, diagram.Options{})
		if err != nil {
			t.Errorf("Prepare(%q): %v", c.Src, err)
			continue
		}
		m, err := ParsePrepared(pre)
		if err != nil {
			t.Errorf("ParsePrepared(%q): %v", c.Src, err)
			continue
		}
		o := &layout.Options{FontSize: c.FontSize}
		if c.Icons {
			o.Icons = func(string) bool { return true }
		}
		got := sceneLines(lay(m, o, c.Text))
		if i := pelesvg.FirstDiff(got, c.Lines); i >= 0 {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, %q, size %v, icons %v), line %d\n got %s\nPele %s", c.Src, c.Text, c.FontSize, c.Icons, i, pelesvg.At(got, i), pelesvg.At(c.Lines, i))
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// TestLayoutDrawsVersion checks that Layout draws limu's name and the version the build
// records.
func TestLayoutDrawsVersion(t *testing.T) {
	sc := Layout(&Model{}, nil, nil)
	if len(sc.Items) != 1 || sc.Items[0].Role != RoleVersion {
		t.Fatalf("items %+v", sc.Items)
	}
	l := sc.Items[0].Label
	if got, want := l.Lines[0][0].Text, "limu "+Version(); got != want {
		t.Errorf("drew %q, not %q", got, want)
	}
	if v := Version(); v == "" || v != "(devel)" && !strings.HasPrefix(v, "v") {
		t.Errorf("version %q", v)
	}
}

// classes are the classes Pele gives the text of an item of each role.
var classes = map[scene.Role]string{RoleTitle: "pele-title", RoleVersion: "pele-version"}

// sceneLines writes a scene as the oracle reads Pele's SVG, every number as JavaScript's
// String writes it.
func sceneLines(sc *scene.Scene) []string {
	n := chars.NumberString
	pt := func(x, y float64) string { return n(x) + "," + n(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	if sc.Title != "" {
		lines = append(lines, "acc-title "+pelesvg.ID(sc.Title))
	}
	if sc.Desc != "" {
		lines = append(lines, "acc-desc "+pelesvg.ID(sc.Desc))
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		l := it.Label
		fill := "?"
		switch it.Style.Text {
		case style.Fg:
			fill = "-"
		case style.Muted:
			fill = "var(--_m)"
		}
		var widths, spans []string
		for _, w := range l.Widths {
			widths = append(widths, n(w))
		}
		for _, line := range l.Lines {
			var s []string
			for _, span := range line {
				if span.Icon != "" {
					s = append(s, "i:"+span.Icon+":"+n(span.Width))
				} else {
					s = append(s, "t:"+strconv.Itoa(int(span.Face))+":"+span.Text+":"+n(span.Width))
				}
			}
			spans = append(spans, strings.Join(s, "|"))
		}
		lines = append(lines, "label "+classes[it.Role]+" "+fill+" "+pt(it.At.X, it.At.Y)+" "+
			n(l.Size)+","+n(l.LineHeight)+","+n(l.Width)+","+n(l.Height)+" | "+strings.Join(widths, ",")+" | "+strings.Join(spans, " / "))
	}
	return lines
}
