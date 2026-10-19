package eventmodeling

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/layout"
)

// Inputs of about 50,000 characters built to hit the event modeling diagram's worst cases,
// as Pele's tests/eventmodeling/stress.test.ts builds them. Each must finish quickly, laid
// out across and in a narrow host, with a scene or a *diagram.Error.
func TestWorstCases(t *testing.T) {
	const n = 50000
	const head = "eventmodeling\n"
	repeat := func(count int, line func(i int) string) string {
		var b strings.Builder
		for i := range count {
			b.WriteString(line(i))
		}
		return b.String()
	}
	id := func(i int) string { return fmt.Sprintf("%03d", i%1000) }
	kinds := []string{"ui", "cmd", "evt", "rmo", "pcr"}
	cases := map[string]string{
		"1,000 frames across the lanes":    head + repeat(1000, func(i int) string { return fmt.Sprintf("tf %s %s E%d\n", id(i), kinds[i%5], i) }),
		"1,000 frames in one lane":         head + repeat(1000, func(i int) string { return fmt.Sprintf("tf %s evt E%d\n", id(i), i) }),
		"1,000 frames in 1,000 namespaces": head + repeat(1000, func(i int) string { return fmt.Sprintf("tf %s evt N%d.E\n", id(i), i) }),
		"1,000 frames that each name every earlier frame": head + repeat(1000, func(i int) string {
			return fmt.Sprintf("tf %s evt E%d%s\n", id(i), i, repeat(min(i, 9), func(k int) string { return " ->> " + id(i-k-1) }))
		}),
		"one frame with 9,000 sources":      head + "tf 001 cmd A\ntf 002 evt B" + strings.Repeat(" ->> 001", 9000) + "\n",
		"a frame number reused 5,000 times": head + strings.Repeat("tf 01 evt A\n", 5000),
		"500 data blocks, each referenced": head + repeat(500, func(i int) string { return fmt.Sprintf("tf %s evt E%d [[D%d]]\n", id(i), i, i) }) +
			repeat(500, func(i int) string { return fmt.Sprintf("data D%d {\n  a: %d\n  b: true\n}\n", i, i) }),
		"one data block of 10,000 lines":                                head + "tf 01 evt A [[D]]\ndata D {\n" + strings.Repeat("a: 1\n", 10000) + "}\n",
		"one data line of 50,000 characters":                            head + "tf 01 evt A { " + strings.Repeat("x", n) + " }\n",
		"inline data of 25,000 braces":                                  head + "tf 01 evt A " + strings.Repeat("{", n/2) + strings.Repeat("}", n/2) + "\n",
		"2,000 notes and specifications":                                head + "tf 01 evt A\n" + strings.Repeat("note 01 {\n n\n}\ngwt 01 given evt A then evt B\n", 1000),
		"a specification of 10,000 statements":                          head + "tf 01 evt A\ngwt 01 given" + strings.Repeat(" evt A", 5000) + " then" + strings.Repeat(" evt B", 5000) + "\n",
		"a qualified name of 10,000 parts":                              head + "tf 01 evt " + strings.Repeat("a.", 10000) + "z\n",
		"one 50,000 character name":                                     head + "tf 01 evt " + strings.Repeat("a", n) + "\n",
		"unclosed braces, one per line":                                 head + strings.Repeat("tf 01 evt A {\n", n/14),
		"an open brace and many closing braces that do not end a block": head + "data D {\n" + strings.Repeat("}x\n", n/3),
		"lines of opening braces":                                       head + strings.Repeat(strings.Repeat("{", 99)+"\n", n/100),
		"one line of quotes":                                            head + "tf 01 evt A " + strings.Repeat(`"`, n),
		"unclosed quotes, one per line":                                 head + strings.Repeat("tf 01 evt A '\n", n/14),
		"unclosed block comment":                                        head + "/*" + strings.Repeat(" x\n", n/3),
		"many block comment openers":                                    head + strings.Repeat("/* ", n/3),
		"line comments":                                                 head + strings.Repeat("// c\n", n/5),
		"percent comments":                                              head + strings.Repeat("%% c\n", n/5),
		"unclosed accDescr after every frame":                           head + strings.Repeat("tf 01 evt A accDescr {\n", n/23),
		"accDescr after long blanks":                                    head + strings.Repeat("x"+strings.Repeat(" ", 90)+"accDescr ", n/100),
		"title after every identifier":                                  head + strings.Repeat("a title b\n", n/10),
		"unclosed directive openers":                                    head + strings.Repeat("x %%{\n", n/6),
		"unclosed front matter fences":                                  head + strings.Repeat("x\n---\n", n/6),
		"blank lines and blanks":                                        head + strings.Repeat(" \n\t", n/3),
		"digits":                                                        head + "tf " + strings.Repeat("1", n),
		"keywords run together":                                         head + strings.Repeat("tfevtuidatanote", n/15),
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

	_, err := diagram.Prepare(head+strings.Repeat("tf 01 evt A\n", 5000), diagram.Options{})
	if e, ok := err.(*diagram.Error); !ok || !strings.Contains(e.Message, "limit of 50000") {
		t.Errorf("a source over the limit: %v", err)
	}
}
