// Package edit maps a place in a text that was rewritten back to the place in the text it was
// written from, as the edits of Pele's errors.ts do, so that an error is reported where its
// author wrote it.
package edit

import (
	"strings"

	"github.com/firede/limu/internal/chars"
)

// An Edit is Pele's Edit: the bytes [From, To) of a text replaced by Text. Pele records the
// length of the replacement in code units; Edit keeps the replacement itself, so that a place
// inside it is counted in code units as Pele counts it.
type Edit struct {
	From, To int
	Text     string
}

// stepInput maps byte at of the text edits made back to the text they were made in, as Pele's
// stepInput does. A place inside a replacement maps to as many code units into what it
// replaced, or to the last of those.
func stepInput(edits []Edit, at int) int {
	shift := 0
	for _, e := range edits {
		start := e.From - shift
		if at < start {
			break
		}
		if at < start+len(e.Text) {
			return e.From + min(chars.Len16(e.Text[:at-start]), e.To-e.From-1)
		}
		shift += e.To - e.From - len(e.Text)
	}
	return at + shift
}

// InputOffset maps byte at of the text the last of steps made back to the text the first was
// made in, as Pele's inputOffset does. Each step is the edits of one pass over the text, in
// the order they were made.
func InputOffset(steps [][]Edit, at int) int {
	for s := len(steps) - 1; s >= 0; s-- {
		at = stepInput(steps[s], at)
	}
	return at
}

// Push adds the edits of a pass to steps, unless steps is nil, which records nothing. A pass
// pushes its edits once it is done, where Pele's recording pushes them as it begins; no pass
// pushes while another is under way, so the steps are in the same order.
func Push(steps *[][]Edit, edits []Edit) {
	if steps != nil {
		*steps = append(*steps, edits)
	}
}

// WrittenCodes writes each entity code that Pele's encodeEntities hid behind placeholders, as
// `ﬂ°°35¶ß` or `ﬂ°quot¶ß`, back as its author wrote it, `#35;` or `#quot;`, as Pele's
// writtenCodes does: text.replace(/ﬂ°°?(\w+)¶ß/g, '#$1;').
func WrittenCodes(text string) string {
	const open, close = "ﬂ°", "¶ß"
	if !strings.Contains(text, close) {
		return text
	}
	var b strings.Builder
	last := 0
	for i := 0; ; {
		at := strings.Index(text[i:], open)
		if at < 0 {
			break
		}
		at += i
		j := at + len(open)
		if strings.HasPrefix(text[j:], "°") {
			j += len("°")
		}
		k := j
		for k < len(text) && chars.IsWord(int(text[k])) {
			k++
		}
		if k > j && strings.HasPrefix(text[k:], close) {
			b.WriteString(text[last:at])
			b.WriteString("#" + text[j:k] + ";")
			last = k + len(close)
			i = last
			continue
		}
		// With no word after `ﬂ°°`, the regular expression backtracks to take the second
		// `°` as no part of it, and no word starts there either.
		i = at + len(open)
	}
	if last == 0 {
		return text
	}
	b.WriteString(text[last:])
	return b.String()
}
