package gitgraph

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs of about 50,000 characters built to trigger worst cases, as Pele's
// tests/git/stress.test.ts builds them: patterns that could rescan, lookups that could go
// quadratic, and large layouts. Each must finish quickly, with a model or a *diagram.Error.
func TestWorstCases(t *testing.T) {
	const n = 50000
	repeat := func(count int, line func(i int) string) string {
		var b strings.Builder
		for i := range count {
			b.WriteString(line(i))
		}
		return b.String()
	}
	zigzag := repeat(1200, func(i int) string {
		a, b := "b", "main"
		if i%2 == 1 {
			a, b = "main", "b"
		}
		return "checkout " + a + "\ncommit\nmerge " + b + "\n"
	})
	picks := "gitGraph\ncommit id:\"s\"\nbranch b\ncommit\n" + repeat(1500, func(i int) string {
		return "checkout main\ncommit id:\"m" + strconv.Itoa(i) + "\"\ncheckout b\ncommit\ncherry-pick id:\"s\"\n"
	})
	parallel := "---\nconfig:\n  gitGraph:\n    parallelCommits: true\n---\n"
	cases := map[string]string{
		"7,000 commits":                "gitGraph\n" + strings.Repeat("commit\n", 7000),
		"3,000 branches of one commit": "gitGraph\ncommit\n" + repeat(3000, func(i int) string { return "branch b" + strconv.Itoa(i) + "\ncommit\n" }),
		"1,500 branches merged back": "gitGraph\ncommit\n" + repeat(1500, func(i int) string {
			return "branch b" + strconv.Itoa(i) + "\ncommit\ncheckout main\nmerge b" + strconv.Itoa(i) + "\n"
		}),
		"merging into each other":           "gitGraph\ncommit\nbranch b\ncommit\n" + zigzag,
		"merging into each other, parallel": parallel + "gitGraph\ncommit\nbranch b\ncommit\n" + zigzag,
		"1,500 cherry-picks":                picks,
		"1,500 cherry-picks, parallel down": parallel + strings.Replace(picks, "gitGraph", "gitGraph TB:", 1),
		"4,000 tags on one commit":          "gitGraph\ncommit" + strings.Repeat(` tag:"t"`, 4000) + "\nbranch b\ncommit" + strings.Repeat(` tag:"u"`, 2000),
		"3,000 commits with one id":         "gitGraph\n" + strings.Repeat("commit id:\"same\"\n", 3000),
		"one 50,000 character id":           "gitGraph\ncommit id:\"" + strings.Repeat("word ", n/5) + "\"",
		"a name of dots with no end":        "gitGraph\nbranch a" + strings.Repeat(".", n),
		"digits before a dot":               "gitGraph\nbranch b order: " + strings.Repeat("9", n) + ".",
		"unclosed strings":                  "gitGraph\ncommit id:" + strings.Repeat(`"a`, n/2),
		"unclosed accDescr blocks":          "gitGraph\n" + strings.Repeat("accDescr {\n", n/11),
		"unclosed directives":               "gitGraph\n" + strings.Repeat("%%{\n", n/4),
		"fences that never close":           "gitGraph\n" + strings.Repeat("---\n", n/4),
		"title lines with comment marks":    "gitGraph\n" + strings.Repeat("title "+strings.Repeat("a%", 40)+"\n", 550),
		"keywords run together":             "gitGraph\n" + strings.Repeat("commit", n/6),
		"a header repeated":                 strings.Repeat("gitGraph:", n/9),
		"an accDescr of spaces":             "gitGraph\naccDescr {" + strings.Repeat(" ", n) + "x}",
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
