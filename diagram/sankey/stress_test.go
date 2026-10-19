package sankey

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs built to hit the sankey diagram's worst cases, as Pele's tests/sankey/stress.test.ts
// builds them. Each must finish quickly, with a scene or a *diagram.Error.
func TestWorstCases(t *testing.T) {
	const n = 50000
	const head = "sankey\n"
	repeat := func(count int, line func(i int) string) string {
		var b strings.Builder
		for i := range count {
			b.WriteString(line(i))
		}
		return b.String()
	}
	itoa := strconv.Itoa
	prepare := func(src string) func() error {
		return func() error {
			prepareTextForParsing(src)
			return nil
		}
	}
	lex := func(src string) func() error {
		return func() error {
			tokenize(src)
			return nil
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
			_, err = Layout(m, pre.Config, &layout.Options{})
			return err
		}
	}
	mesh := head
	for i := 0; i < 70; i++ {
		for j := 0; j < 70; j++ {
			mesh += "a" + itoa(i) + ",b" + itoa(j) + "," + itoa((i*7+j)%13) + "\n"
		}
	}
	layered := head
	for l := 0; l < 40; l++ {
		for i := 0; i < 12; i++ {
			for j := 0; j < 12; j += 3 {
				layered += itoa(l) + "_" + itoa(i) + "," + itoa(l+1) + "_" + itoa((i+j)%12) + "," + itoa(1+(i+j)%5) + "\n"
			}
		}
	}
	cases := map[string]func() error{
		"preparing text with long runs of blanks":        prepare(head + strings.Repeat("a,b,1"+strings.Repeat(" ", 500)+"\n", n/506)),
		"preparing one run of 50,000 blanks in the text": prepare(head + "a,b,1" + strings.Repeat(" ", n) + "x"),
		"preparing 50,000 line breaks":                   prepare(head + strings.Repeat("\r\n", n/2) + "a,b,1"),
		"one field of 50,000 characters":                 lex(head + strings.Repeat("x", n) + ",b,1"),
		"one unclosed quoted field":                      lex(head + `"` + strings.Repeat("x", n)),
		"a quoted field made of escaped quotes":          lex(head + `"` + strings.Repeat(`""`, n/2) + `",b,1`),
		"quotes and nothing else":                        lex(head + strings.Repeat(`"`, n)),
		"commas and nothing else":                        lex(head + strings.Repeat(",", n)),
		"a character outside the alphabet after a field": lex(head + strings.Repeat("x", n) + "\t"),
		"the keyword repeated":                           lex(strings.Repeat("sankey", n/6)),
		"a quoted field spanning 10,000 lines":           draw(head + `"` + strings.Repeat("line\n", 10000) + `",b,1`),
		"8,000 links between 8,001 nodes in a chain":     draw(head + repeat(8000, func(i int) string { return itoa(i) + "," + itoa(i+1) + ",1\n" })),
		"a star of 6,000 targets":                        draw(head + repeat(6000, func(i int) string { return "c,t" + itoa(i) + "," + itoa(i+1) + "\n" })),
		"6,000 sources into one target":                  draw(head + repeat(6000, func(i int) string { return "s" + itoa(i) + ",c," + itoa(i+1) + "\n" })),
		"8,000 parallel links":                           draw(head + strings.Repeat("a,b,1\n", 8000)),
		"a hub joined to 2,500 columns":                  draw(head + repeat(2500, func(i int) string { return "n" + itoa(i) + ",n" + itoa(i+1) + ",1\nhub,n" + itoa(i+1) + ",1\n" })),
		"a dense two-column mesh":                        draw(mesh),
		"a layered mesh of 40 columns":                   draw(layered),
		"a cycle of 8,000":                               draw(head + repeat(8000, func(i int) string { return itoa(i) + "," + itoa((i+1)%8000) + ",1\n" })),
		"a long tail hanging off a small cycle":          draw(head + "a,b,1\nb,a,1\n" + repeat(7000, func(i int) string { return itoa(i) + "," + itoa(i+1) + ",1\n" }) + "7000,a,1\n"),
		"8,000 self links":                               draw(head + repeat(8000, func(i int) string { return itoa(i) + "," + itoa(i) + ",1\n" })),
		"values at the edge of what a number holds": draw(head + repeat(3000, func(i int) string {
			return "a" + itoa(i%50) + ",b" + itoa(i%60) + "," + [2]string{"5e-324", "1e308"}[i%2] + "\n"
		})),
		"values that are not numbers": draw(head + repeat(5000, func(i int) string { return "a" + itoa(i%50) + ",b" + itoa(i%60) + ",x\n" })),
		"node names of 5,000 characters": draw(head + repeat(5, func(i int) string {
			return strings.Repeat(itoa(i), 5000) + "," + strings.Repeat(itoa(i+1), 5000) + ",1\n"
		})),
	}
	for name, run := range cases {
		started := time.Now()
		if err := run(); err != nil {
			if _, ok := err.(*diagram.Error); !ok {
				t.Errorf("%s: %v", name, err)
			}
		}
		if d := time.Since(started); d > 3*time.Second {
			t.Errorf("%s: took %v", name, d)
		}
	}
}
