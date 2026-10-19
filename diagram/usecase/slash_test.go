package usecase

import (
	"math"
	"testing"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// The slash of a business use case starts where it meets the oval, the middle's y and the
// oval's half height times a root, each rounded apart, as JavaScript works it out. The
// render suite writes the slash to two decimals, as Pele does, and so cannot see the last
// bit a fused multiply-add would change; this checks it to the bit, for labels where fusing
// would change it.
func TestBusinessSlashKeepsTheProductApart(t *testing.T) {
	reached := 0
	for _, name := range []string{"Use", "Pay the bill", "Return goods now", "Order", "Check out the cart", "W"} {
		src := "usecase-beta\nU(" + name + ")@{ business: true }\n"
		pre, err := diagram.Prepare(src, diagram.Options{})
		if err != nil {
			t.Fatal(err)
		}
		m, err := ParsePrepared(pre)
		if err != nil {
			t.Fatal(err)
		}
		sc := Layout(m, pre.Config, &layout.Options{})
		for _, it := range sc.Items {
			if it.Shape != "ellipse-business" {
				continue
			}
			w, h := it.Box.W, it.Box.H
			y := it.Parts[0].Path.Args[1] // the oval starts at its right end, level with its middle
			q := (w/2 - 15) / (w / 2)
			root := math.Sqrt(1 - float64(q*q))
			want := y + float64((h/2)*root)
			if got := it.Parts[1].Path.Args[1]; got != want {
				t.Errorf("%q: slash starts at y %v, want %v", name, got, want)
			}
			if math.FMA(h/2, root, y) != want {
				reached++
			}
		}
	}
	if reached == 0 {
		t.Error("no label where a fused multiply-add would differ")
	}
}
