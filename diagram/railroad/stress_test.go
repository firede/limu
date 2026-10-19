package railroad

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/layout"
)

// The worst cases of Pele's tests/railroad/stress.test.ts: text that would make a tokenizer
// rescan, or a parser or a layout recurse, as deep as memory allows. Each is tokenized, parsed
// or laid out well within a few seconds, or fails with a diagram's error.

const worstN = 50000

func repeat(count int, part func(i int) string) string {
	var b strings.Builder
	for i := range count {
		b.WriteString(part(i))
	}
	return b.String()
}

func TestWorstCases(t *testing.T) {
	const (
		ir   = "railroad-beta\n"
		ebnf = "railroad-ebnf-beta\n"
		abnf = "railroad-abnf-beta\n"
		peg  = "railroad-peg-beta\n"
	)
	r := strings.Repeat
	N := worstN
	lex := func(table *common.Table, src string) func() error {
		return func() error {
			_, err := common.Tokenize(src, table, "railroad")
			return err
		}
	}
	parse := func(src string) func() error {
		return func() error {
			_, err := Parse(src+"\n", "")
			return err
		}
	}
	render := func(src string) func() error {
		return func() error {
			m, err := Parse(src+"\n", "")
			if err != nil {
				return err
			}
			Layout(m, nil, &layout.Options{})
			return nil
		}
	}
	cases := []struct {
		name string
		run  func() error
	}{
		{"special sequence opener before a long run with no closer", lex(ebnfTable, ebnf+"r = ?"+r(" a", N/2))},
		{"question marks separated by blanks", lex(ebnfTable, ebnf+"r = "+r("?  ", N/3))},
		{"question marks each followed by a long run and a semicolon", lex(ebnfTable, ebnf+r("? "+r("a ", 50)+";", N/103))},
		{"unclosed ISO comment openers", lex(ebnfTable, ebnf+"r = "+r("(* a ", N/5))},
		{"ISO comment openers that never get a full closer", lex(ebnfTable, ebnf+"r = "+r("(*)", N/3))},
		{"unclosed block comment openers", lex(irTable, ir+r("/* a ", N/5))},
		{"block comment open and close, many times", lex(ebnfTable, ebnf+r("/**/(**)", N/8))},
		{"braced descriptions that are never closed", lex(ebnfTable, ebnf+r("accDescr { ", N/11))},
		{"descriptions with long blank runs and no brace", lex(ebnfTable, ebnf+r("accDescr"+r(" ", 92), N/100))},
		{"one-line descriptions with no comment marker", lex(irTable, ir+r("accDescr: "+r("% ", 45)+"\n", N/101))},
		{"accTitle with no colon, many times", lex(pegTable, peg+r("accTitle"+r(" ", 42), N/50))},
		{"one long title with single percent signs", lex(irTable, ir+"title "+r("a % ", N/4))},
		{"title words on one line", lex(abnfTable, abnf+r("title ", N/6))},
		{"titles with no space after them", lex(irTable, ir+r("titlea", N/6))},
		{"unclosed double quote", lex(irTable, ir+`r = terminal("`+r(`a\"`, N/3))},
		{"quotes that open and never close, with backslashes", lex(pegTable, peg+"R <- '"+r(`\\`, N/2))},
		{"unclosed ABNF string", lex(abnfTable, abnf+`r = "`+r("a ", N/2))},
		{"front matter fences that do not close", lex(ebnfTable, ebnf+"r = a "+r("---\n", N/4))},
		{"directive openers that do not close", lex(irTable, ir+r("%%{ a ", N/6))},
		{"dashes", lex(ebnfTable, ebnf+"r = a "+r("-", N))},
		{"digits with no star", lex(abnfTable, abnf+"r = "+r("1", N)+"a ;")},
		{"digit runs each with a star", lex(abnfTable, abnf+"r = "+r("12*34 ", N/6))},
		{"numeric values with long tails", lex(abnfTable, abnf+"r = %x"+r("41.", N/3)+" ;")},
		{"semicolon comments", lex(abnfTable, abnf+r("; a comment\n", N/12))},
		{"hash comments", lex(pegTable, peg+r("# a comment\n", N/12))},
		{"keywords glued into one name", lex(irTable, ir+r("terminal", N/8))},
		{"one 50,000 character name", render(ebnf + r("a", N) + " = " + r("b", N) + " ;")},
		{"one 50,000 character terminal", render(ebnf + `r = "` + r("word ", N/5) + `" ;`)},
		{"terminal of entity codes and markup", render(ebnf + `r = "` + r("#35;&amp;<br>*_", N/15) + `" ;`)},
		{"terminal of escapes", render(peg + `R <- "` + r(`\n\t\\`, N/6) + `" ;`)},
		{"25,000 nested groups", render(ebnf + "r = " + r("(", N/2) + "a" + r(")", N/2) + " ;")},
		{"12,000 nested optionals", render(ebnf + "r = " + r("[", 12000) + "a" + r("]", 12000) + " ;")},
		{"12,000 nested repetitions", render(ebnf + "r = " + r("{", 12000) + "a" + r("}", 12000) + " ;")},
		{"50,000 postfix operators", render(ebnf + "r = a" + r("*+?", N/3) + " ;")},
		{"chain of 12,000 exceptions", render(ebnf + "r = a" + r(" - b", 12000) + " ;")},
		{"exceptions nested 8,000 deep", render(ebnf + "r = a" + r(" - ( b", 8000) + r(" )", 8000) + " ;")},
		{"unclosed brackets", parse(ebnf + "r = " + r("([{", N/3))},
		{"choice of 12,000 alternatives", render(ebnf + "r = " + repeat(12000, func(i int) string { return "a" + strconv.Itoa(i) + " | " }) + "z ;")},
		{"sequence of 12,000 elements", render(ebnf + "r = " + r("a ", 12000) + ";")},
		{"choices nested 5,000 deep", render(ebnf + "r = " + r("( a | ", 5000) + "b" + r(" )", 5000) + " ;")},
		{"5,000 rules", render(ebnf + repeat(5000, func(i int) string { return "r" + strconv.Itoa(i) + " = a ;\n" }))},
		{"one rule name defined 5,000 times", render(ebnf + r("r = a | b ;\n", 5000))},
		{"constructor calls nested 5,000 deep", render(ir + "r = " + r("optional(oneOrMore(", 2500) + `terminal("a")` + r("))", 2500) + " ;")},
		{"sequence constructors nested 6,000 deep", render(ir + "r = " + r("sequence(", 6000) + `terminal("a")` + r(")", 6000) + " ;")},
		{"choice call with 6,000 arguments", render(ir + "r = choice(" + r(`terminal("a"),`, 6000) + `special("b")) ;`)},
		{"unclosed constructor calls", parse(ir + "r = " + r("choice(", N/7))},
		{"ABNF groups nested 10,000 deep with repeats", render(abnf + "r = " + r("2*3( ", 10000) + "a" + r(" )", 10000) + " ;")},
		{"ABNF optional groups nested 12,000 deep", render(abnf + "r = " + r("[", 12000) + "a" + r("]", 12000) + " ;")},
		{"ABNF repeat of 400 digits", render(abnf + "r = " + r("9", 400) + "*" + r("9", 400) + "a " + r("9", 400) + "b ;")},
		{"ABNF alternation of 12,000", render(abnf + "r = " + r("a / ", 12000) + "b ;")},
		{"PEG predicates nested 10,000 deep", render(peg + "R <- " + r("!( &( ", 5000) + "a" + r(" ) )", 5000) + " ;")},
		{"PEG groups with suffixes nested 10,000 deep", render(peg + "R <- " + r("( ", 10000) + "a" + r(" )*", 10000) + " ;")},
		{"PEG run of operators", parse(peg + "R <- " + r("!&", N/2) + "a ;")},
		{"PEG sequence of 12,000 predicates", render(peg + "R <- " + r(`!a &"b" `, 6000) + ";")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			started := time.Now()
			if err := c.run(); err != nil {
				if _, ok := err.(*diagram.Error); !ok {
					t.Errorf("%v, not a diagram's error", err)
				}
			}
			if d := time.Since(started); d > 3*time.Second {
				t.Errorf("took %v", d)
			}
		})
	}
}
