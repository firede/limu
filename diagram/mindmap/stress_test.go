package mindmap

import (
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs built to trigger worst cases, as Pele's tests/mindmap/stress.test.ts builds them:
// long runs the lexer could rescan, maps of many nodes or many levels, and labels of 50,000
// characters. Each must finish quickly, with a scene or a *diagram.Error, laid out across and
// in a narrow host, running down.
func TestWorstCases(t *testing.T) {
	const n = 50000
	repeat := func(count int, line func(i int) string) string {
		var b strings.Builder
		for i := range count {
			b.WriteString(line(i))
		}
		return b.String()
	}
	cases := map[string]string{
		"one line of shaped nodes":                       "mindmap\n" + strings.Repeat("(a) ", n/4),
		"one line of spaces between ids":                 "mindmap\na(b)" + strings.Repeat(" ", n) + "x",
		"blank lines of spaces":                          "mindmap\nroot\n" + strings.Repeat(" \n", n/2) + " a",
		"a newline then a long run of spaces, repeated":  "mindmap\nroot" + strings.Repeat("\n"+strings.Repeat(" ", 99)+"a", n/100),
		"whitespace that never reaches a comment":        "mindmap\nroot\n" + strings.Repeat(" \t\n", n/3) + "%",
		"comment markers":                                "mindmap\nroot\n" + strings.Repeat("%%", n/2),
		"lone closing braces in a node":                  "mindmap\nroot[" + strings.Repeat("}\n", n/2) + "]",
		"unclosed markdown text":                         "mindmap\nroot[\"`" + strings.Repeat("a\n", n/2),
		"unclosed icon":                                  "mindmap\nroot\n::icon(" + strings.Repeat("x\n", n/2),
		"dashes and brackets":                            "mindmap\n" + strings.Repeat("-)(-", n/4),
		"16,000 nodes on one level":                      "mindmap\nroot\n" + strings.Repeat(" a\n", 16000),
		"3,000 levels of nesting":                        "mindmap\n" + repeat(3000, func(i int) string { return strings.Repeat(" ", i) + "n\n" }),
		"indentation that rises and falls":               "mindmap\nroot\n" + repeat(2400, func(i int) string { return strings.Repeat(" ", 1+i%40) + "n\n" }),
		"5,000 branches with two leaves each":            "mindmap\nroot\n" + strings.Repeat(" b\n  l\n  l\n", 5000),
		"1,400 shaped nodes with icons and classes":      "mindmap\nroot\n" + strings.Repeat(" a((b))\n ::icon(fa fa-x)\n :::c d\n", 1400),
		"one 50,000 character label":                     "mindmap\nroot[" + strings.Repeat("word ", n/5) + "]",
		"one 50,000 character word":                      "mindmap\nroot[" + strings.Repeat("w", n) + "]",
		"label of emphasis markers":                      "mindmap\nroot[\"`" + strings.Repeat("*a _b ", n/6) + "`\"]",
		"label of line breaks and entities":              "mindmap\nroot[" + strings.Repeat("#35;&amp;<br>", n/13) + "]",
		"a class list of 25,000 names":                   "mindmap\nroot\n:::" + strings.Repeat("a ", n/2),
		"a second root after 16,000 nodes":               "mindmap\n root\n" + strings.Repeat("  a\n", 16000) + " b\n",
		"deep branches each followed by a shallow one":   "mindmap\nroot\n" + repeat(160, func(int) string { return repeat(24, func(i int) string { return strings.Repeat(" ", 1+i) + "n\n" }) }),
		"an icon name of 50,000 characters, with a host": "mindmap\nroot\n::icon(" + strings.Repeat("fa ", n/3) + ")",
	}
	for name, src := range cases {
		started := time.Now()
		pre, err := diagram.Prepare(src, diagram.Options{Limit: -1})
		if err == nil {
			var m *Model
			if m, err = ParsePrepared(pre); err == nil {
				Layout(m, pre.Config, &layout.Options{Icons: func(string) bool { return true }})
				Layout(m, pre.Config, &layout.Options{MaxWidth: 300})
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
