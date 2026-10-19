package treeview

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs of about 50,000 characters built to hit the tree view's worst cases, as Pele's
// tests/treeview/stress.test.ts builds them. Each must finish quickly, laid out loosely and
// in a narrow host, with a scene or a *diagram.Error.
func TestWorstCases(t *testing.T) {
	const n = 50000
	const head = "treeView-beta\n"
	repeat := func(count int, line func(i int) string) string {
		var b strings.Builder
		for i := range count {
			b.WriteString(line(i))
		}
		return b.String()
	}
	cases := map[string]string{
		"25,000 flat entries":                      head + strings.Repeat("a\n", n/2),
		"300 levels of nesting":                    head + repeat(300, func(i int) string { return strings.Repeat(" ", i) + "d/\n" }),
		"310 levels indented with tabs and spaces": head + repeat(310, func(i int) string { return strings.Repeat("\t", i%2) + strings.Repeat(" ", i) + "x\n" }),
		"one 50,000 character name":                head + strings.Repeat("a", n),
		"a name with one long run of blanks":       head + "a" + strings.Repeat(" ", n) + "b",
		"a name of many runs of blanks":            head + strings.Repeat("a"+strings.Repeat(" ", 99), n/100) + "b",
		"runs of blanks that almost start a class": head + strings.Repeat("a :::"+strings.Repeat(" ", 95), n/100) + "1",
		"one line of almost-annotations":           head + "a" + strings.Repeat(" ::: 1 icon[ #", n/14),
		"blanks then a colon, many lines":          head + strings.Repeat(strings.Repeat(" ", 98)+":\n", n/100),
		"unclosed accDescr on every line":          head + strings.Repeat("accDescr {\n", n/11),
		"unclosed accDescr after names":            head + "a\n" + strings.Repeat(" accDescr {\n", n/12),
		"accDescr followed by blank lines":         head + strings.Repeat("accDescr"+strings.Repeat("\n", 40), n/48),
		"unclosed quotes":                          head + strings.Repeat("\"a\n", n/3),
		"10,000 annotations on one entry":          head + "a" + strings.Repeat(" :::b icon(c) ## d\n x"[:13], n/13),
		"10,000 classes on one entry":              head + "a" + strings.Repeat(" :::highlight", n/13),
		"every row highlighted with a description": head + repeat(2500, func(i int) string { return fmt.Sprintf(" f%d :::highlight ## d%d\n", i, i) }),
		"comments":                             head + strings.Repeat("%% c\n", n/5),
		"one 50,000 character description":     head + "a ## " + strings.Repeat("word ", n/5),
		"entities":                             head + " " + strings.Repeat("#35;&amp;", n/9),
		"box drawing, 10,000 rows":             head + strings.Repeat("├── a\n", n/6),
		"box drawing, deep columns":            head + repeat(300, func(i int) string { return strings.Repeat("│", i) + "├─ x\n" }),
		"box drawing, one far branch per line": head + "├ a\n" + strings.Repeat(strings.Repeat(" ", 490)+"└ b\n", 100),
		"box decoration only":                  head + strings.Repeat("│   │   │\n", n/10),
		"box drawing, tabs":                    head + strings.Repeat(strings.Repeat("\t", 40)+"├── a\n", n/46),
		"box characters inside names":          head + strings.Repeat("a ─ b\n", n/6),
		"icons on every row": "---\nconfig:\n  treeView:\n    showIcons: true\n---\n" + head +
			repeat(5000, func(i int) string { return fmt.Sprintf(" f%d.ts icon(p:i%d)\n", i, i) }),
	}
	for name, src := range cases {
		started := time.Now()
		err := func() error {
			pre, err := diagram.Prepare(src, diagram.Options{Limit: -1})
			if err != nil {
				return err
			}
			m, err := ParsePrepared(pre)
			if err != nil {
				return err
			}
			Layout(m, pre.Config, &layout.Options{})
			Layout(m, pre.Config, &layout.Options{MaxWidth: 320})
			return nil
		}()
		if _, ok := err.(*diagram.Error); err != nil && !ok {
			t.Errorf("%s: %v", name, err)
		}
		if d := time.Since(started); d > 2*time.Second {
			t.Errorf("%s: took %v", name, d)
		}
	}

	_, err := diagram.Prepare(head+strings.Repeat("a\n", 30000), diagram.Options{})
	if e, ok := err.(*diagram.Error); !ok || !strings.Contains(e.Message, "limit of 50000") {
		t.Errorf("a source over the limit: %v", err)
	}
}
