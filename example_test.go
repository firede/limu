package limu_test

import (
	"fmt"

	"github.com/firede/limu"
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
	"github.com/firede/limu/svg"
)

func Example() {
	sc, err := limu.Layout("flowchart LR\n  A[Text] --> B[Scene] --> C{Backend}\n  C -->|svg| D[SVG]\n  C -->|mygoui| E[Screen]", nil)
	if err != nil {
		panic(err)
	}
	out := svg.Render(sc, nil)
	fmt.Println(sc.Type, len(sc.Items), len(out) > 0)
	// Output: flowchart 11 true
}

// A host that lays a diagram out again when the width it has changes parses it once.
func ExampleParse() {
	d, err := limu.Parse("pie title Pets\n  \"Dogs\": 386\n  \"Cats\": 85", diagram.Options{})
	if err != nil {
		panic(err)
	}
	for _, width := range []float64{0, 240} {
		sc, err := d.Layout(&layout.Options{MaxWidth: width, Padding: new(0.0)})
		if err != nil {
			panic(err)
		}
		fmt.Println(d.Type(), sc.Width, sc.Height)
	}
	// Output:
	// pie 325 280
	// pie 240 352
}
