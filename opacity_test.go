package limu_test

import (
	"testing"

	"github.com/firede/limu"
	"github.com/firede/limu/scene"
	"github.com/firede/limu/style"
)

// An author's fill and fill opacity cascade each on their own, as a browser takes the style
// Pele writes over the attributes it writes beside it. Pele 1.6.0 draws these diagrams so:
//
//	style A fill-opacity:0.5,stroke-opacity:0.25 on a flowchart's node:
//	  <rect fill="var(--_s)" stroke="var(--_b)" style="rx:var(--_r);fill-opacity:0.5;stroke-opacity:0.25;"/>
//	classDef red fill:red on a subgraph:
//	  <rect fill="var(--_a)" fill-opacity="0.5" stroke="var(--_b)" style="rx:var(--_r);fill:red;"/>
//	classDef red fill:red on a state composite:
//	  <rect rx="10" fill="var(--_a)" fill-opacity="0.5" stroke="var(--_b)" style="fill:red;"/>
//	linkStyle 0 stroke:red,stroke-opacity:0.25 on an edge, its arrowhead in the group:
//	  <g class="pele-edge" style="stroke:red;stroke-opacity:0.25;">… <path class="pele-marker" fill="red" stroke="none"/>
func TestAuthorOpacitiesAsPele(t *testing.T) {
	th := style.Default()
	at := func(c style.Color, a uint8) style.Color { c.A = a; return c }
	red := style.Color{R: 255, A: 255}
	for _, c := range []struct {
		src    string
		role   scene.Role
		id     string
		fill   style.Color
		stroke style.Color
		marker style.Color
	}{
		{"flowchart LR\n  A --> B\n  style A fill-opacity:0.5,stroke-opacity:0.25", "flowchart.node", "A",
			at(th.Surface, 128), at(th.Border, 64), th.Border},
		{"flowchart LR\n  subgraph S\n    A\n  end\n  classDef red fill:red\n  class S red", "flowchart.cluster", "S",
			at(red, 128), th.Border, th.Border},
		{"stateDiagram-v2\n  state C {\n    a --> b\n  }\n  classDef red fill:red\n  class C red", "state.composite", "C",
			at(red, 128), th.Border, th.Border},
		{"flowchart LR\n  A --> B\n  linkStyle 0 stroke:red,stroke-opacity:0.25", "flowchart.edge", "L_A_B_0",
			style.Color{}, at(red, 64), red},
	} {
		sc, err := limu.Layout(c.src, nil)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for i := range sc.Items {
			it := &sc.Items[i]
			if it.Role != c.role || it.ID != c.id {
				continue
			}
			found = true
			var look *style.Look
			r := look.Resolve(string(it.Role), it.Style, it.Author)
			if r.Fill != c.fill || r.Stroke != c.stroke || r.Marker != c.marker {
				t.Errorf("%q: %s fill %v stroke %v marker %v, want %v %v %v", c.src, c.id, r.Fill, r.Stroke, r.Marker, c.fill, c.stroke, c.marker)
			}
		}
		if !found {
			t.Errorf("%q: no %s %s", c.src, c.role, c.id)
		}
	}
}
