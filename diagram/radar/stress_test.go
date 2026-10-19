package radar

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs of about 50,000 characters built to hit the radar chart's worst cases, as Pele's
// tests/radar/stress.test.ts builds them. Each must finish quickly, with a scene or a
// *diagram.Error.
func TestWorstCases(t *testing.T) {
	const n = 50000
	list := func(count int, item func(i int) string) string {
		items := make([]string, count)
		for i := range items {
			items[i] = item(i)
		}
		return strings.Join(items, ",")
	}
	parse := func(src string) func() error {
		return func() error {
			_, err := parseRadar(src)
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
	name := func(i int) string { return "a" + strconv.Itoa(i) }
	cases := map[string]func() error{
		"5,000 axes and one curve": draw("radar-beta\naxis " + list(5000, name) + "\ncurve c{" + list(5000, strconv.Itoa) + "}"),
		"5,000 axes with a polygon graticule of 32 ticks": draw("radar-beta\naxis " + list(5000, name) + "\ncurve c{" +
			list(5000, func(int) string { return "1" }) + "}\ngraticule polygon\nticks 32"),
		"2,000 axes and curves of named entries": func() error {
			entries := list(2000, func(i int) string { return "a" + strconv.Itoa(1999-i) + " " + strconv.Itoa(i) })
			return draw("radar-beta\naxis " + list(2000, name) + "\ncurve x{" + entries + "}\ncurve y{" + entries + "}")()
		},
		"4,000 curves":                        draw("radar-beta\naxis a,b,c\n" + strings.Repeat("curve c{1,2,3}\n", 4000)),
		"5,000 options":                       draw("radar-beta\n" + strings.Repeat("ticks 3\nmax 9\n", 2500)),
		"50,000 zeros as curve values":        parse("radar-beta\ncurve c{" + strings.Repeat("0", n) + "}"),
		"50,000 zeros before a decimal point": parse("radar-beta\nmax " + strings.Repeat("0", n) + ".5"),
		"a 50,000 digit number":               draw("radar-beta\naxis a\ncurve c{1}\nmax " + strings.Repeat("9", n)),
		"numbers and dots":                    parse("radar-beta\ncurve c{" + strings.Repeat("1.2", n/3) + "}"),
		"a 50,000 character name":             draw("radar-beta\naxis " + strings.Repeat("a", n)),
		"a name of dashes":                    parse("radar-beta\naxis a" + strings.Repeat("-", n)),
		"keywords run together":               parse("radar-beta\n" + strings.Repeat("axis", n/4)),
		"accDescr opened 5,000 times":         parse("radar-beta\n" + strings.Repeat("accDescr {", n/10)),
		"an unclosed accDescr block":          parse("radar-beta\naccDescr {" + strings.Repeat(" a\n", n/3)),
		"an unclosed string":                  parse("radar-beta\naxis a[\"" + strings.Repeat("x", n)),
		"a 50,000 character label":            draw("radar-beta\naxis a[\"" + strings.Repeat("word ", n/5) + "\"], b\ncurve c{1,2}"),
		"a label of entities and line breaks": draw("radar-beta\naxis a[\"" + strings.Repeat("#35;&amp;<br>", n/13) + "\"], b\ncurve c{1,2}"),
		"25,000 open brackets":                parse("radar-beta\naxis a" + strings.Repeat("[", n/2)),
		"20,000 line breaks inside a curve":   draw("radar-beta\naxis a\ncurve c{" + strings.Repeat("\n", 20000) + "1" + strings.Repeat("\n", 20000) + "}"),
		"unclosed directives":                 parse("radar-beta\n" + strings.Repeat("%%{\n", n/4)),
		"front matter fences":                 parse("radar-beta\n" + strings.Repeat("---\n", n/4)),
		"a long title":                        draw("radar-beta\ntitle " + strings.Repeat("x", n)),
		"a line of spaces":                    parse("radar-beta\n" + strings.Repeat(" ", n) + "axis a"),
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
