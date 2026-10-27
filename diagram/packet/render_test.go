package packet

import (
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/pelesvg"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// The fixture is what Pele's render draws of packet diagrams, read back out of its SVG as one
// line per thing drawn, in the order drawn; ./scripts/oracle.sh writes it. The test writes
// the same lines from the scene.

type peleRender struct {
	Cases []struct {
		Src      string  `json:"src"`
		MaxWidth float64 `json:"maxWidth"`
		FontSize float64 `json:"fontSize"`
		fixture.Drawing
	} `json:"cases"`
}

func TestLayoutMatchesPele(t *testing.T) {
	var p peleRender
	f, err := fixture.LoadDrawings("testdata/pele/render.json", &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	failed := 0
	for k, c := range p.Cases {
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
		got := sceneLines(Layout(m, pre.Config, &layout.Options{MaxWidth: c.MaxWidth, FontSize: c.FontSize}))
		if msg := f.Diff(k, &c.Drawing, got, nil); msg != "" {
			if failed++; failed <= 20 {
				t.Errorf("Layout(%q, max %v, size %v), %s", c.Src, c.MaxWidth, c.FontSize, msg)
			}
		}
	}
	if failed > 0 {
		t.Errorf("%d of %d differ", failed, len(p.Cases))
	}
}

// paintNames are the theme's paints as Pele's SVG names them.
var paintNames = map[style.Paint]string{
	style.Fg:      "var(--_fg)",
	style.Muted:   "var(--_m)",
	style.Surface: "var(--_s)",
	style.Border:  "var(--_b)",
}

func paintName(p style.Paint) string {
	if s, ok := paintNames[p]; ok {
		return s
	}
	return "?"
}

// written is text as Pele's esc writes it, without the control characters XML does not
// allow.
func written(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, s)
}

var anchors = [...]string{scene.Middle: "middle", scene.Start: "start", scene.End: "end"}

// sceneLines writes a scene as the oracle reads Pele's SVG.
func sceneLines(sc *scene.Scene) []string {
	num := pelesvg.Num
	pt := func(x, y float64) string { return num(x) + "," + num(y) }
	lines := []string{"size " + pt(sc.Width, sc.Height)}
	// The lines of a label drawn about At, as Pele's text and labelSvg place them, after
	// prefix, and with what the oracle writes of the size and paint of its type.
	label := func(prefix, kind string, it *scene.Item) {
		l := it.Label
		if l == nil || l.Empty() {
			return
		}
		top := it.At.Y - l.Height/2
		for _, line := range l.Lines {
			y := top + l.LineHeight/2 + float64(l.Size*0.35)
			at := prefix + "t " + pt(it.At.X, y) + " " + kind + " " + anchors[it.Align] + " "
			if len(line) == 0 {
				lines = append(lines, at+" ")
			}
			for k, span := range line {
				if k == 0 {
					lines = append(lines, at+written(span.Text))
				} else {
					lines = append(lines, prefix+"+ "+written(span.Text))
				}
			}
			top += l.LineHeight
		}
	}
	for i := range sc.Items {
		it := &sc.Items[i]
		switch it.Role {
		case RoleField:
			r := it.Parts[0].Box
			rounded := "-"
			if it.Parts[0].Rounded {
				rounded = "r"
			}
			lines = append(lines, "field "+it.ID+" "+pt(r.X, r.Y)+","+pt(r.W, r.H)+" "+rounded+" "+
				paintName(it.Style.Fill)+" "+paintName(it.Style.Stroke))
			if it.Label != nil {
				label("", num(it.Label.Size), it)
			}
		case RoleBit:
			label("bit ", num(it.Label.Size)+" "+paintName(it.Style.Text), it)
		case RoleTitle:
			label("title ", "-", it)
		default:
			lines = append(lines, "unknown role "+string(it.Role))
		}
	}
	return lines
}
