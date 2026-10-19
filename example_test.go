package limu_test

import (
	"fmt"

	"github.com/firede/limu"
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
