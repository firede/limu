package timeline

import "github.com/firede/limu/internal/chars"

// The types of token, in the order of Pele's T.
const (
	tEND = iota
	tEOF
	tINVALID
	tNEWLINE
	tTimeline
	tTimelineLR
	tTimelineTD
	tTitle
	tAccTitle
	tAccTitleValue
	tAccDescr
	tAccDescrValue
	tAccDescrMultilineValue
	tSection
	tPeriod
	tEvent
)

// tokenNames are the names of the types of token, as Mermaid's errors give them: Pele's
// TOKEN_NAMES.
var tokenNames = [...]string{
	"$end", "EOF", "INVALID", "NEWLINE", "timeline", "timeline_lr", "timeline_td", "title", "acc_title",
	"acc_title_value", "acc_descr", "acc_descr_value", "acc_descr_multiline_value", "section", "period", "event",
}

// tokenize splits a timeline diagram's text into tokens, as Pele's tokenize does.
func tokenize(src string) *tokens {
	n := len(src)
	out := &tokens{}
	p := 0
	emit := func(typ, e int) {
		out.types = append(out.types, typ)
		out.texts = append(out.texts, src[p:e])
		out.starts = append(out.starts, p)
		p = e
	}

	for {
		p = skipTrivia(src, p)
		if p >= n {
			break
		}
		c, size := chars.UnitAt(src, p)
		e := p + 1
		if c == 10 {
			for e < n && src[e] == 10 {
				e++
			}
			emit(tNEWLINE, e)
			continue
		}
		if keywordAt(src, p, "timeline") {
			e = p + 8
			for e < n && (src[e] == 32 || src[e] == 9) {
				e++
			}
			lr := keywordAt(src, e, "lr")
			if e > p+8 && (lr || keywordAt(src, e, "td")) && !chars.IsWord(chars.Unit(src, e+2)) {
				if lr {
					emit(tTimelineLR, e+2)
				} else {
					emit(tTimelineTD, e+2)
				}
				continue
			}
			if !chars.IsWord(chars.Unit(src, p+8)) {
				emit(tTimeline, p+8)
				continue
			}
		}
		// `\s` after a keyword also takes a line break, so `title` alone on a line takes the
		// next line as its text.
		if keywordAt(src, p, "title") {
			if s, size := chars.UnitAt(src, p+5); chars.IsSpace(s) && p+5+size < n && src[p+5+size] != 10 {
				emit(tTitle, lineEnd(src, p+5+size))
				continue
			}
		}
		if e = accessibility(src, p, out, tAccTitle); e != p {
			p = e
			continue
		}
		if keywordAt(src, p, "section") {
			if s, size := chars.UnitAt(src, p+7); chars.IsSpace(s) {
				from := p + 7 + size
				e = from
				for e < n && src[e] != 58 && src[e] != 10 {
					e++
				}
				if e > from {
					emit(tSection, e)
					continue
				}
			}
		}
		if c != 58 {
			e = p + size
			for e < n && src[e] != 35 && src[e] != 58 && src[e] != 10 {
				e++
			}
			emit(tPeriod, e)
			continue
		}
		// An event runs to the end of the line or to a colon that is followed by white space.
		if s, size := chars.UnitAt(src, p+1); chars.IsSpace(s) {
			from := p + 1 + size
			for e = from; e < n; e++ {
				k := src[e]
				if k == 10 || k == 58 && chars.IsSpace(chars.Unit(src, e+1)) {
					break
				}
			}
			if e > from {
				emit(tEvent, e)
				continue
			}
		}
		emit(tINVALID, p+1)
	}

	// Jison marks its lexer done when a rule runs on empty input, so an accessibility value
	// that matches nothing at the very end of the text is followed by no EOF token.
	if len(out.starts) == 0 || out.starts[len(out.starts)-1] < n {
		emit(tEOF, n)
	}
	emit(tEND, n)
	return out
}
