package architecture

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs built to trigger worst cases, as Pele's tests/architecture/stress.test.ts builds
// them: large layouts, where the grid has to move blocks aside, patterns that could rescan,
// and labels of 50,000 characters. Each must finish quickly, with a scene or a *diagram.Error.
func TestWorstCases(t *testing.T) {
	const n = 50000
	const head = "architecture-beta\n"
	repeat := func(count int, line func(i int) string) string {
		var b strings.Builder
		for i := range count {
			b.WriteString(line(i))
		}
		return b.String()
	}
	itoa := strconv.Itoa
	services := func(count int) string {
		return repeat(count, func(i int) string { return "service n" + itoa(i) + "\n" })
	}
	complete := head + services(70)
	for i := range 70 {
		for j := i + 1; j < 70; j++ {
			complete += "n" + itoa(i) + ":R -- L:n" + itoa(j) + "\n"
		}
	}
	diagonalGroup := func(g int, side string) string {
		s := "group g" + itoa(g) + "\n" + repeat(12, func(i int) string { return "service m" + itoa(g) + "x" + itoa(i) + " in g" + itoa(g) + "\n" })
		return s + repeat(11, func(i int) string {
			return "m" + itoa(g) + "x" + itoa(i) + ":" + side + ":m" + itoa(g) + "x" + itoa(i+1) + "\n"
		})
	}
	cases := map[string]string{
		"4,500 unconnected services":           head + services(4500),
		"a chain of 2,000":                     head + services(2000) + repeat(1999, func(i int) string { return "n" + itoa(i) + ":R -- L:n" + itoa(i+1) + "\n" }),
		"a diagonal of 2,000":                  head + services(2000) + repeat(1999, func(i int) string { return "n" + itoa(i) + ":R -- T:n" + itoa(i+1) + "\n" }),
		"a star of 2,000 all wanting one cell": head + services(2000) + repeat(1999, func(i int) string { return "n0:R -- L:n" + itoa(i+1) + "\n" }),
		"a star of 2,000 bends":                head + services(2000) + repeat(1999, func(i int) string { return "n0:B -- L:n" + itoa(i+1) + "\n" }),
		"a star of 2,000 like sides":           head + services(2000) + repeat(1999, func(i int) string { return "n0:R -- R:n" + itoa(i+1) + "\n" }),
		"a column of hubs with two leaves each": head + services(1800) + repeat(599, func(i int) string {
			return "n" + itoa(i*3) + ":B -- T:n" + itoa(i*3+3) + "\nn" + itoa(i*3) + ":R -- L:n" + itoa(i*3+1) + "\nn" + itoa(i*3) + ":R -- L:n" + itoa(i*3+2) + "\n"
		}),
		"3,000 edges between two services": head + "service a\nservice b\n" + strings.Repeat("a:R <-[x]-> L:b\n", 3000),
		"2,500 self edges":                 head + "service a\n" + strings.Repeat("a:R --> L:a\n", 2500),
		"a complete graph of 70":           complete,
		"2,000 nested groups":              head + "group g0\n" + repeat(1999, func(i int) string { return "group g" + itoa(i+1) + " in g" + itoa(i) + "\n" }) + "service a in g1999\n",
		"1,000 nested groups with titles and edges to the outside": head + "group g0[T]\n" + repeat(999, func(i int) string { return "group g" + itoa(i+1) + "[T] in g" + itoa(i) + "\n" }) +
			"service a in g999\nservice b\n" + strings.Repeat("a:R -- L:b\n", 500),
		"1,200 sibling groups in a chain": head + repeat(1200, func(i int) string { return "group g" + itoa(i) + "\nservice n" + itoa(i) + " in g" + itoa(i) + "\n" }) +
			repeat(1199, func(i int) string { return "n" + itoa(i) + ":R -- L:n" + itoa(i+1) + "\n" }),
		"1,000 groups each wanting the same place": head + "service hub\n" + repeat(1000, func(i int) string {
			return "group g" + itoa(i) + "\nservice n" + itoa(i) + " in g" + itoa(i) + "\nhub:R -- L:n" + itoa(i) + "\n"
		}),
		"large groups all wanting the same place": head + "service hub\n" + repeat(40, func(g int) string { return diagonalGroup(g, "R -- T") + "hub:R -- L:m" + itoa(g) + "x0\n" }),
		"large groups beside a column of 1,500": head + services(1500) + repeat(1499, func(i int) string { return "n" + itoa(i) + ":B -- T:n" + itoa(i+1) + "\n" }) +
			repeat(20, func(g int) string { return diagonalGroup(g, "B -- R") + "n0:R -- L:m" + itoa(g) + "x0\n" }),
		"many groups crossed by many vertical edges": head + repeat(600, func(i int) string {
			return "group g" + itoa(i) + "[Title]\nservice n" + itoa(i) + " in g" + itoa(i) + "\n"
		}) +
			repeat(599, func(i int) string { return "n" + itoa(i) + ":B --> T:n" + itoa(i+1) + "\n" }) + strings.Repeat("n0:B -- T:n599\n", 800),
		"align directives over 2,000 members": head + services(2000) + "align row " + repeat(1000, func(i int) string { return "n" + itoa(i) + " " }) + "\n" +
			repeat(1000, func(i int) string { return "align column n" + itoa(i) + " n" + itoa(i+1000) + "\n" }),
		"one 50,000 character service title":      head + "service a[" + strings.Repeat("word ", n/5) + "]",
		"one 50,000 character unbroken title":     head + "service a[" + strings.Repeat("w", n) + "]",
		"one 50,000 character icon text":          head + "service a \"" + strings.Repeat("word ", n/5) + "\"",
		"one 50,000 character unbroken icon text": head + "service a \"" + strings.Repeat("😀", n/2) + "\"",
		"a 25,000 character group title and edge label": head + "group g[" + strings.Repeat("word ", n/10) + "]\nservice a in g\nservice b\na:R -[" +
			strings.Repeat("*a _b ", n/12) + "]- L:b",
		"quoted titles that never close":    head + repeat(n/20, func(i int) string { return "service n" + itoa(i) + "[\"a]\n" }),
		"one identifier of 50,000":          head + "service " + strings.Repeat("a-", n/2) + "a",
		"unclosed brackets":                 head + "service a" + strings.Repeat("[", n),
		"unclosed parentheses":              head + "service a(" + strings.Repeat("a:-", n/3),
		"unclosed string":                   head + "service a \"" + strings.Repeat("a\\\"", n/3),
		"runs of dashes":                    head + "service a\nservice b\na:R " + strings.Repeat("-", n) + " L:b",
		"fence openers with no closer":      head + "service a\nservice b\n" + strings.Repeat("a:R ---\n", n/8),
		"unclosed directives on every line": head + strings.Repeat("%%{\n", n/4),
		"direction letters":                 head + strings.Repeat("L", n),
		"group modifiers":                   head + "service a\na" + strings.Repeat("{group}", n/7),
	}
	for name, src := range cases {
		started := time.Now()
		pre, err := diagram.Prepare(src, diagram.Options{Limit: -1})
		if err == nil {
			var m *Model
			if m, err = ParsePrepared(pre); err == nil {
				Layout(m, pre.Config, &layout.Options{})
			}
		}
		if _, ok := err.(*diagram.Error); err != nil && !ok {
			t.Errorf("%s: %v", name, err)
		}
		if d := time.Since(started); d > 2*time.Second {
			t.Errorf("%s: took %v", name, d)
		}
	}
}
