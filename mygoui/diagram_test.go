package mygoui

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/firede/limu"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/scene"
)

// diagram lays out a diagram's text with limu, measured with the Go fonts. The host has
// the icons of Font Awesome's names, which DrawIcon draws; Pele draws the rest itself.
func diagram(t *testing.T, src string) *scene.Scene {
	t.Helper()
	sc, err := limu.Layout(src, &layout.Options{Measurer: goFonts, Icons: func(name string) bool {
		return strings.HasPrefix(name, "fa:") || strings.HasPrefix(name, "fa ")
	}})
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

// drawIcon draws every icon alike: a ring with a dot in it, in the color given.
func drawIcon(p *ui.Painter, name string, r ui.Rect, c ui.Color) {
	cx, cy, rr := r.X+r.W/2, r.Y+r.H/2, min(r.W, r.H)/2
	const k = 0.5522847498
	circle := func(r float32) *ui.Path {
		return new(ui.Path).MoveTo(cx+r, cy).
			CubeTo(cx+r, cy+r*k, cx+r*k, cy+r, cx, cy+r).CubeTo(cx-r*k, cy+r, cx-r, cy+r*k, cx-r, cy).
			CubeTo(cx-r, cy-r*k, cx-r*k, cy-r, cx, cy-r).CubeTo(cx+r*k, cy-r, cx+r, cy-r*k, cx+r, cy).Close()
	}
	p.StrokePath(circle(rr*0.8), rr*0.2, c)
	p.FillPath(circle(rr*0.3), c)
}

// An architecture diagram, of M5's: groups, services with the icons Pele draws, a junction,
// and edges between their sides.
func TestGoldenArchitecture(t *testing.T) {
	sc := diagram(t, `architecture-beta
    group api(cloud)[API]
    service db(database)[Database] in api
    service disk(disk)[Storage] in api
    service server(server)[Server] in api
    junction j in api
    db:L -- R:server
    disk:T -- B:j
    j:L -- R:db`)
	snap(t, "architecture", sc, &Options{Fonts: goFonts}, 1)
}

// A flowchart's markers and lines: arrows, circles and crosses, at one end and both, on
// solid, dotted and thick edges, and a dashed node of the author's.
func TestGoldenFlowchartEdges(t *testing.T) {
	sc := diagram(t, `flowchart LR
    A[Start] --> B{Ready?}
    B -.->|no| C(Wait)
    B ==>|yes| D[[Go]]
    C --o A
    D --x E((End))
    C <--> D
    classDef dashed stroke-dasharray:4 3,stroke-width:2px
    class E dashed`)
	snap(t, "flowchart-edges", sc, &Options{Fonts: goFonts}, 1)
}

// Icons the host draws: in a mindmap's nodes, as their pictures, and in a label's text.
func TestGoldenMindmapIcons(t *testing.T) {
	sc := diagram(t, `mindmap
  root((limu))
    Layout
      ::icon(fa fa-book)
      Pele
    Backends
      svg
        ::icon(fa fa-code)
      fa:fa-desktop mygoui`)
	snap(t, "mindmap-icons", sc, &Options{Fonts: goFonts, DrawIcon: drawIcon}, 1)
}
