package quadrant

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs of about 50,000 characters built to hit the quadrant chart's worst cases, as Pele's
// tests/quadrant/stress.test.ts builds them. Each must finish quickly, with a scene or a
// *diagram.Error.
func TestWorstCases(t *testing.T) {
	const n = 50000
	repeat := func(count int, line func(i int) string) string {
		var b strings.Builder
		for i := range count {
			b.WriteString(line(i))
		}
		return b.String()
	}
	lex := func(src string) func() error {
		return func() error {
			tokenize(src)
			return nil
		}
	}
	styles := func(style string) func() error {
		return func() error {
			_, err := parseStyles([]string{style})
			return err
		}
	}
	draw := func(src string) func() error {
		return func() error {
			pre, err := diagram.Prepare(src, diagram.Options{Limit: -1})
			if err != nil {
				return err
			}
			m, err := ParsePrepared(pre)
			if err != nil {
				return err
			}
			Layout(m, pre.Config, &layout.Options{})
			return nil
		}
	}
	two := func(i int) string { return strconv.Itoa(i % 100) }
	cases := map[string]func() error{
		"a run of spaces with no keyword after it":    lex("quadrantChart\n" + strings.Repeat(" ", n) + "!"),
		"a run of tabs before a colon and no bracket": lex("quadrantChart\nA" + strings.Repeat("\t", n) + ":x"),
		"runs of tabs and colons, never a bracket":    lex("quadrantChart\n" + strings.Repeat(strings.Repeat("\t", 500)+":", 100)),
		"colons and spaces in turn":                   lex("quadrantChart\n" + strings.Repeat(": ", n/2)),
		"a run of dashes with no arrowhead":           lex("quadrantChart\nx-axis a " + strings.Repeat("-", n)),
		"spaces before dashes with no arrowhead":      lex("quadrantChart\nx-axis a" + strings.Repeat("  "+strings.Repeat("-", 50), 1000)),
		"line breaks each followed by a comment":      lex("quadrantChart" + strings.Repeat("\n%% c", n/5)),
		"comment markers with no line break":          lex("quadrantChart\n" + strings.Repeat("a%%", n/3)),
		"accTitle with no colon, many times":          lex("quadrantChart\n" + strings.Repeat("accTitle"+strings.Repeat(" ", 40), 1000)),
		"an unclosed accDescr block":                  draw("quadrantChart\naccDescr {" + strings.Repeat(" a\n", n/3)),
		"an unclosed string":                          draw("quadrantChart\nquadrant-1 \"" + strings.Repeat("a", n)),
		"an unclosed markdown string":                 draw("quadrantChart\nquadrant-1 \"`" + strings.Repeat("a ", n/2)),
		"a point number followed by digits":           lex("quadrantChart\nA: [0." + strings.Repeat("5", n) + ", 1]"),
		"space before the comma of a point":           lex("quadrantChart\nA: [0" + strings.Repeat(" ", n) + "1]"),
		"a style of spaces around one colon":          styles("radius" + strings.Repeat(" ", n) + ":" + strings.Repeat(" ", n) + "5"),
		"a style of spaces and no colon":              styles("a" + strings.Repeat(" ", n) + "b"),
		"a class of 8,000 styles":                     draw("quadrantChart\nA:::c: [0.5, 0.5]\nclassDef c " + strings.Repeat("radius: 5,", 8000) + "radius: 6"),
		"a point with 6,000 styles":                   draw("quadrantChart\nA: [0.5, 0.5] " + strings.Repeat("color: #abc ,", 6000) + "radius: 6"),
		"3,500 points": draw("quadrantChart\n" + repeat(3500, func(i int) string {
			return "P" + strconv.Itoa(i) + ": [0." + two(i) + ", 0." + two(i*7) + "]\n"
		})),
		"300 points at one place": draw("quadrantChart\n" + repeat(300, func(i int) string { return "Point " + strconv.Itoa(i) + ": [0.5, 0.5]\n" })),
		"3,000 classes": draw("quadrantChart\n" + repeat(3000, func(i int) string {
			return "classDef c" + strconv.Itoa(i) + " radius: " + strconv.Itoa(i) + "\n"
		}) + "A:::c7: [0.5, 0.5]"),
		"one point with a 50,000 character label": draw("quadrantChart\n" + strings.Repeat("word ", n/5) + ": [0.5, 0.5]"),
		"a quadrant label of emphasis markers":    draw("quadrantChart\nquadrant-1 \"`" + strings.Repeat("*a _b ", n/6) + "`\""),
		"labels of entities and line breaks":      draw("quadrantChart\nx-axis \"" + strings.Repeat("#35;&amp;<br>", n/13) + "\""),
		"a title of 50,000 characters":            draw("quadrantChart\ntitle " + strings.Repeat("long title ", n/11)),
		"a class name of 50,000 characters":       draw("quadrantChart\nA:::" + strings.Repeat("c", n) + ": [0.5, 0.5]"),
		"10,000 statements on one line":           draw("quadrantChart;" + strings.Repeat("quadrant-1 a;", 10000)),
		"leading spaces before every statement":   draw("quadrantChart\n" + strings.Repeat(strings.Repeat(" ", 100)+"quadrant-2 b\n", 450)),
		"a radius of 400 digits":                  draw("quadrantChart\nA: [0.5, 0.5] radius: " + strings.Repeat("9", 400)),
		"a stroke of 400 digits":                  draw("quadrantChart\nA: [0.5, 0.5] stroke-width: " + strings.Repeat("9", 400) + "px"),
	}
	for name, run := range cases {
		started := time.Now()
		if err := run(); err != nil {
			if _, ok := err.(*diagram.Error); !ok {
				t.Errorf("%s: %v", name, err)
			}
		}
		if d := time.Since(started); d > 2*time.Second {
			t.Errorf("%s: took %v", name, d)
		}
	}
}
