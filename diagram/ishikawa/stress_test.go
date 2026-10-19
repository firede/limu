package ishikawa

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs built to hit the Ishikawa diagram's worst cases, as Pele's
// tests/ishikawa/stress.test.ts builds them. Each must finish quickly, laid out across and in
// a narrow host, with a scene or a *diagram.Error.
func TestWorstCases(t *testing.T) {
	const n = 50000
	const head = "ishikawa-beta\n"
	repeat := func(count int, line func(i int) string) string {
		var b strings.Builder
		for i := range count {
			b.WriteString(line(i))
		}
		return b.String()
	}
	deep := func(count, from int) string {
		return repeat(count, func(i int) string { return strings.Repeat(" ", i+from) + "n\n" })
	}
	tokens := map[string]string{
		"50,000 spaces":                                    head + strings.Repeat(" ", n),
		"50,000 line breaks":                               head + strings.Repeat("\n", n),
		"spaces and line breaks by turns":                  head + strings.Repeat(" \n", n/2),
		"long runs of spaces, each ending in a line break": head + strings.Repeat(strings.Repeat(" ", 999)+"\n", n/1000),
		"long runs of spaces, each ending in text":         head + strings.Repeat(strings.Repeat(" ", 999)+"x\n", n/1000),
		"a line break and then 50,000 spaces":              head + "x\n" + strings.Repeat(" ", n),
		"white space before a comment that never comes":    head + strings.Repeat(" \t\r  \n", n/6) + "%",
		"white space before a comment":                     head + strings.Repeat(" \t\r  \n", n/6) + "%% c",
		"comment lines":                                    head + strings.Repeat("  %% c\n", n/7),
		"percent signs":                                    head + strings.Repeat("% ", n/2),
		"carriage returns":                                 head + strings.Repeat("x\r", n/2),
		"the keyword over and over":                        strings.Repeat("ishikawa-beta ", n/14),
		"the keyword without its ending, over and over":    strings.Repeat("ishikawa-bet\n", n/13),
	}
	for name, src := range tokens {
		started := time.Now()
		tokenize(src)
		if d := time.Since(started); d > 2*time.Second {
			t.Errorf("%s: took %v", name, d)
		}
	}
	cases := map[string]string{
		"one line of 50,000 characters":                 head + "  " + strings.Repeat("a", n),
		"one line of 10,000 words":                      head + "  Effect\n    " + strings.Repeat("word ", 10000),
		"a cause of entities and markup":                head + "  Effect\n    Category\n      " + strings.Repeat("#35;&amp;<br>", n/13),
		"12,000 categories":                             head + "  E\n" + strings.Repeat("    c\n", 12000),
		"12,000 causes in one category":                 head + "  E\n    C\n" + strings.Repeat("      c\n", 12000),
		"12,000 sub-causes of one cause":                head + "  E\n    C\n      c\n" + strings.Repeat("        s\n", 12000),
		"an outline 300 levels deep":                    head + deep(300, 1),
		"an outline 3,000 levels deep":                  head + deep(3000, 1),
		"an outline 3,000 levels deep, twice":           head + " E\n" + deep(3000, 2) + deep(3000, 2),
		"indentation that goes up and down":             head + repeat(8000, func(i int) string { return strings.Repeat(" ", 1+(i*7)%13) + "n\n" }),
		"a sawtooth of indentation":                     head + strings.Repeat(deep(40, 1), 200),
		"indentation by 2,000 spaces":                   head + " E\n" + repeat(20, func(i int) string { return strings.Repeat(" ", 1+(i%2)*2000) + "n\n" }),
		"4,000 categories with a cause and a sub-cause": head + "  E\n" + strings.Repeat("    c\n      d\n        e\n", 4000),
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
			for _, o := range []*layout.Options{{}, {MaxWidth: 320}} {
				sc := Layout(m, pre.Config, o)
				if math.IsNaN(sc.Width) || math.IsInf(sc.Width, 0) || math.IsNaN(sc.Height) || math.IsInf(sc.Height, 0) {
					t.Errorf("%s: a scene of %v by %v", name, sc.Width, sc.Height)
				}
			}
			return nil
		}()
		if _, ok := err.(*diagram.Error); err != nil && !ok {
			t.Errorf("%s: %v", name, err)
		}
		if d := time.Since(started); d > 3*time.Second {
			t.Errorf("%s: took %v", name, d)
		}
	}

	// A deep outline keeps its causes in order, to the last.
	m, err := Parse(head+repeat(400, func(i int) string { return strings.Repeat(" ", i+1) + fmt.Sprintf("n%d\n", i) }), "", false)
	if err != nil {
		t.Fatal(err)
	}
	sc := Layout(m, nil, &layout.Options{})
	if last := sc.Items[len(sc.Items)-2]; last.Label == nil || last.Label.Lines[0][0].Text != "n399" {
		t.Errorf("the last cause of a deep outline is not n399")
	}

	_, err = diagram.Prepare(head+strings.Repeat("  cause\n", 8000), diagram.Options{})
	if e, ok := err.(*diagram.Error); !ok || !strings.Contains(e.Message, "limit of 50000") {
		t.Errorf("a source over the limit: %v", err)
	}
}
