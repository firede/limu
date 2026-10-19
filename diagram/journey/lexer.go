package journey

import (
	"github.com/firede/limu/diagram/internal/common"
	"github.com/firede/limu/internal/chars"
)

// The types of token, in the order of Pele's T.
const (
	tEnd = iota
	tEOF
	tInvalid
	tNewline
	tJourney
	tTitle
	tAccTitle
	tAccTitleValue
	tAccDescr
	tAccDescrValue
	tAccDescrMultilineValue
	tSection
	tTaskName
	tTaskData
	tColon
)

// tokenNames are the names Pele's TOKEN_NAMES gives the types, as its errors write them.
var tokenNames = []string{
	"$end",
	"EOF",
	"INVALID",
	"NEWLINE",
	"journey",
	"title",
	"acc_title",
	"acc_title_value",
	"acc_descr",
	"acc_descr_value",
	"acc_descr_multiline_value",
	"section",
	"taskName",
	"taskData",
	":",
}

// textEnd is where text from from ends, as Pele's textEnd finds it: `#`, a line break and
// `;` end every kind of text, and a colon also ends a task name or a section.
func textEnd(src string, from int, colon bool) int {
	e := from
	for ; e < len(src); e++ {
		c := src[e]
		if c == '#' || c == '\n' || c == ';' || colon && c == ':' {
			break
		}
	}
	return e
}

// tokenize lexes a journey diagram, as Pele's tokenize does.
func tokenize(src string) *common.JisonTokens {
	n := len(src)
	out := &common.JisonTokens{}
	p := 0
	emit := func(typ, e int) {
		out.Emit(typ, src, p, e)
		p = e
	}
	for {
		p = common.SkipTrivia(src, p)
		if p >= n {
			break
		}
		c := src[p]
		if c == '\n' {
			e := p + 1
			for e < n && src[e] == '\n' {
				e++
			}
			emit(tNewline, e)
			continue
		}
		if common.KeywordAt(src, p, "journey") && !chars.IsWord(chars.Unit(src, p+7)) {
			emit(tJourney, p+7)
			continue
		}
		// \s after a keyword also takes a line break, so `title` alone on a line takes the
		// next line as its text.
		if common.KeywordAt(src, p, "title") {
			if space, size := chars.UnitAt(src, p+5); chars.IsSpace(space) {
				from := p + 5 + size
				if e := textEnd(src, from, false); e > from {
					emit(tTitle, e)
					continue
				}
			}
		}
		if e := common.Accessibility(src, p, out, tAccTitle); e != p {
			p = e
			continue
		}
		if common.KeywordAt(src, p, "section") {
			if space, size := chars.UnitAt(src, p+7); chars.IsSpace(space) {
				from := p + 7 + size
				if e := textEnd(src, from, true); e > from {
					emit(tSection, e)
					continue
				}
			}
		}
		switch c {
		case ':':
			if e := textEnd(src, p+1, false); e > p+1 {
				emit(tTaskData, e)
			} else {
				emit(tColon, p+1)
			}
		case ';':
			emit(tInvalid, p+1)
		default:
			_, size := chars.UnitAt(src, p)
			emit(tTaskName, textEnd(src, p+size, true))
		}
	}

	// Jison marks its lexer done when a rule runs on empty input, so an accessibility value
	// that matches nothing at the very end of the text is followed by no EOF token.
	if len(out.Starts) == 0 || out.Starts[len(out.Starts)-1] < n {
		emit(tEOF, n)
	}
	emit(tEnd, n)
	return out
}
