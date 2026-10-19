package flowchart

import (
	"strconv"
	"strings"
	"testing"

	"github.com/firede/limu/diagram"
)

// bigFlowchart is a flowchart of n nodes with labels and shapes, each linked to the one
// before it and every tenth to one further back, with a subgraph, classes and styles.
func bigFlowchart(n int) string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	shapes := []string{"[%s]", "(%s)", "{%s}", "((%s))", "[/%s/]", ">%s]"}
	for i := 0; i < n; i++ {
		id := "N" + strconv.Itoa(i)
		label := strings.Replace(shapes[i%len(shapes)], "%s", "Node "+strconv.Itoa(i), 1)
		b.WriteString("  " + id + label + "\n")
		if i > 0 {
			b.WriteString("  N" + strconv.Itoa(i-1) + " -->|step " + strconv.Itoa(i) + "| " + id + "\n")
		}
		if i >= 10 && i%10 == 0 {
			b.WriteString("  " + id + " -.-> N" + strconv.Itoa(i-10) + "\n")
		}
	}
	b.WriteString("  subgraph group [Group]\n    N1 --> N2\n  end\n")
	b.WriteString("  classDef hot fill:#f96,stroke:#333\n  class N3,N4,N5 hot\n")
	b.WriteString("  style N6 fill:#bbf\n  linkStyle default stroke:#999\n")
	return b.String()
}

func BenchmarkParse500(b *testing.B) {
	src := bigFlowchart(500)
	b.SetBytes(int64(len(src)))
	for b.Loop() {
		p, err := diagram.Prepare(src, diagram.Options{})
		if err != nil {
			b.Fatal(err)
		}
		db, err := ParsePrepared(p)
		if err != nil {
			b.Fatal(err)
		}
		BuildGraph(db, DefaultCurve(p.Config))
	}
}
