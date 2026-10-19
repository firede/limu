package er

import (
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/style"
)

// An entity's divider takes only the stroke color the author set, as Pele draws it, and not
// the width, the dash or the fill.
func TestDividerTakesOnlyTheStroke(t *testing.T) {
	src := "erDiagram\nA {\nint a\n}\nstyle A fill:#f00,stroke:#00f,stroke-width:4px,stroke-dasharray:5\n"
	pre, err := diagram.Prepare(src, diagram.Options{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := ParsePrepared(pre)
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(db, pre.Config, &layout.Options{})
	var stroke style.Paint
	found := false
	for _, it := range sc.Items {
		switch it.Role {
		case RoleEntity:
			stroke = it.Author.Stroke
		case RoleDivider:
			found = true
			_, width := it.Author.StrokeWidth.Get()
			if !stroke.Set() || it.Author.Stroke != stroke || it.Author.Fill.Set() || width || it.Author.Dash.Set() {
				t.Errorf("divider's author style %+v, want only the stroke %+v", it.Author, stroke)
			}
		}
	}
	if !found {
		t.Error("no divider")
	}
}
