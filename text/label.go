package text

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsmath"
)

// A Span is a run of a line in one face: text, or an icon in its place.
type Span struct {
	Text  string
	Face  Face
	Icon  string // the icon's name, as "fa:fa-car"; Text is then empty
	Width float64
}

// A Label is text laid out in lines to a width, measured.
type Label struct {
	Lines      [][]Span
	Widths     []float64 // of each line
	Width      float64   // of the widest line
	Height     float64
	Size       float64
	LineHeight float64
}

// Empty reports whether the label has nothing to draw.
func (l *Label) Empty() bool { return len(l.Lines) == 0 }

// LineHeight is the height of a line, in ems.
const LineHeight = 1.5

// Options says how to lay out a label.
type Options struct {
	Measurer Measurer
	Size     float64
	MaxWidth float64 // lines longer than this are wrapped; 0 is no limit
	Face     Face    // of text with no emphasis of its own
	Markdown bool    // read **bold** and _italic_
	// Icons reports whether the host can draw an icon by name. A label leaves room for an
	// icon only where it can; with nil, it never does.
	Icons func(name string) bool
	// Bounded takes MaxWidth as it is even at 0 or below, as Pele's layoutLabel does: a width
	// worked out to fit text in a box of a set size can come out at nothing or less.
	Bounded bool
}

// Layout lays out a label as Pele's layoutLabel does: <br> and new lines break it, <b>,
// <strong>, <i> and <em> set emphasis, as do ** and _ in markdown, fa:fa-name is an icon,
// entities are decoded, and lines wider than MaxWidth wrap between words, and between any
// two Chinese or Japanese characters.
func Layout(raw string, o Options) Label {
	if raw == "" {
		return Label{}
	}
	if o.MaxWidth <= 0 && !o.Bounded {
		o.MaxWidth = math.Inf(1)
	}
	lineHeight := jsmath.Round(o.Size * LineHeight)

	if !hasMarkup(raw) && !strings.Contains(raw, "&") && !strings.Contains(raw, "ﬂ") {
		text := chars.Trim(raw)
		width := o.Measurer.Width(text, o.Size, o.Face)
		if width <= o.MaxWidth || !(strings.Contains(text, " ") || hasCJK(text)) {
			return Label{
				Lines:      [][]Span{{{Text: text, Face: o.Face, Width: width}}},
				Widths:     []float64{width},
				Width:      width,
				Height:     lineHeight,
				Size:       o.Size,
				LineHeight: lineHeight,
			}
		}
		var lines [][]Span
		var widths []float64
		wrap([]Span{{Text: text, Face: o.Face, Width: width}}, &o, &lines, &widths)
		widest := 0.0
		for _, w := range widths {
			if w > widest {
				widest = w
			}
		}
		return Label{Lines: lines, Widths: widths, Width: widest, Height: float64(len(lines)) * lineHeight, Size: o.Size, LineHeight: lineHeight}
	}

	source := chars.Trim(raw)
	if o.Markdown {
		source = chars.Trim(dedent(strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")))
	}
	var lines [][]Span
	var widths []float64
	width := 0.0
	for _, part := range splitBreaks(source) {
		var line []Span
		inline(&line, chars.Trim(part), o.Markdown, o.Face, &o)
		w := measureSpans(line, &o)
		if w > o.MaxWidth {
			wrap(line, &o, &lines, &widths)
		} else {
			lines = append(lines, line)
			widths = append(widths, w)
		}
	}
	for _, w := range widths {
		if w > width {
			width = w
		}
	}
	return Label{Lines: lines, Widths: widths, Width: width, Height: float64(len(lines)) * lineHeight, Size: o.Size, LineHeight: lineHeight}
}

// hasMarkup matches /[<\\\n*_]|fa[bklrs]?:fa-/.
func hasMarkup(s string) bool {
	if strings.ContainsAny(s, "<\\\n*_") {
		return true
	}
	for i := strings.Index(s, "fa"); i != -1; {
		rest := s[i+2:]
		if len(rest) > 0 && strings.IndexByte("bklrs", rest[0]) >= 0 {
			rest = rest[1:]
		}
		if strings.HasPrefix(rest, ":fa-") {
			return true
		}
		next := strings.Index(s[i+1:], "fa")
		if next == -1 {
			break
		}
		i += 1 + next
	}
	return false
}

// isCJK reports whether r is one of the Chinese and Japanese characters a line may break
// between. Hangul is left out: Korean puts spaces between words.
func isCJK(r rune) bool {
	return r >= 0x2e80 && r <= 0x9fff || r >= 0xf900 && r <= 0xfaff || r >= 0xff00 && r <= 0xff60
}

func hasCJK(s string) bool {
	for _, r := range s {
		if isCJK(r) {
			return true
		}
	}
	return false
}

// Characters that may not begin a line, and those that may not end one.
const (
	closing = "、。，．：；！？）］｝」』】〕〉》〙〗ー…‥・ゝゞぁぃぅぇぉっゃゅょゎァィゥェォッャュョヮ,.:;!?)]}"
	opening = "（［｛「『【〔〈《〘〖([{"
)

func isClosing(r rune) bool { return strings.ContainsRune(closing, r) }
func isOpening(r rune) bool { return strings.ContainsRune(opening, r) }

// pieces splits text where a line may break, as Pele's RE_PIECE does:
//
//	\s*(?:[OPEN]*[CJK][CLOSE]*|[^\sCJK]+[CLOSE]*)\s*|\s+
//
// Each piece keeps the space around it.
func pieces(s string) []string {
	var out []string
	runAt := func(i int, in func(rune) bool) int {
		for i < len(s) {
			r, n := utf8.DecodeRuneInString(s[i:])
			if !in(r) {
				break
			}
			i += n
		}
		return i
	}
	// word matches the group between the spaces at s[j], and gives where it ends.
	word := func(j int) (int, bool) {
		// [OPEN]*[CJK][CLOSE]*: the opening run is taken greedily, then given back a
		// character at a time until a CJK character follows it.
		var opens []int
		k := j
		for k < len(s) {
			r, n := utf8.DecodeRuneInString(s[k:])
			if !isOpening(r) {
				break
			}
			opens = append(opens, k)
			k += n
		}
		for t := len(opens); t >= 0; t-- {
			at := j
			if t > 0 {
				_, n := utf8.DecodeRuneInString(s[opens[t-1]:])
				at = opens[t-1] + n
			}
			if at < len(s) {
				if r, n := utf8.DecodeRuneInString(s[at:]); isCJK(r) {
					return runAt(at+n, isClosing), true
				}
			}
		}
		// [^\sCJK]+[CLOSE]*
		k = runAt(j, func(r rune) bool { return !chars.IsSpaceRune(r) && !isCJK(r) })
		if k == j {
			return 0, false
		}
		return runAt(k, isClosing), true
	}
	for i := 0; i < len(s); {
		lead := runAt(i, chars.IsSpaceRune)
		end, ok := word(lead)
		if !ok {
			// Only white space is left. The regular expression would give the spaces back one
			// at a time, and might take U+3000, a CJK character too, as a word, but the
			// trailing spaces would take the rest: the piece is the same.
			end = lead
		} else {
			end = runAt(end, chars.IsSpaceRune)
		}
		if end == i {
			// Every character is a space, a CJK character or another, so this is not reached.
			_, n := utf8.DecodeRuneInString(s[i:])
			i += n
			continue
		}
		out = append(out, s[i:end])
		i = end
	}
	return out
}

// dedent takes off the indent every line has. Indents are counted in characters, as
// JavaScript counts them in code units: white space is all in the Basic Multilingual Plane.
func dedent(text string) string {
	lines := strings.Split(text, "\n")
	least := -1
	for _, line := range lines {
		if chars.Trim(line) == "" {
			continue
		}
		if indent := indentOf(line); least == -1 || indent < least {
			least = indent
		}
	}
	if least <= 0 {
		return text
	}
	for i, line := range lines {
		skip := min(least, indentOf(line))
		for skip > 0 {
			_, n := utf8.DecodeRuneInString(line)
			line = line[n:]
			skip--
		}
		lines[i] = line
	}
	return strings.Join(lines, "\n")
}

func indentOf(line string) int {
	return utf8.RuneCountInString(line) - utf8.RuneCountInString(chars.TrimStart(line))
}

// splitBreaks splits text at /<br\s*\/?>|\\n|\n/gi: a tag, a backslash and an n, of either
// case, or a new line.
func splitBreaks(s string) []string {
	var out []string
	last := 0
	for i := 0; i < len(s); {
		switch {
		case s[i] == '\n':
			out = append(out, s[last:i])
			i++
			last = i
			continue
		case s[i] == '\\' && i+1 < len(s) && (s[i+1] == 'n' || s[i+1] == 'N'):
			out = append(out, s[last:i])
			i += 2
			last = i
			continue
		case s[i] == '<' && i+3 <= len(s) && strings.EqualFold(s[i+1:i+3], "br"):
			j := i + 3
			for j < len(s) {
				r, n := utf8.DecodeRuneInString(s[j:])
				if !chars.IsSpaceRune(r) {
					break
				}
				j += n
			}
			if j < len(s) && s[j] == '/' {
				j++
			}
			if j < len(s) && s[j] == '>' {
				out = append(out, s[last:i])
				i = j + 1
				last = i
				continue
			}
		}
		i++
	}
	return append(out, s[last:])
}

func pushText(line *[]Span, text string, f Face) {
	if text == "" {
		return
	}
	decoded := DecodeEntities(text)
	if n := len(*line); n > 0 {
		last := &(*line)[n-1]
		if last.Face == f && last.Icon == "" {
			last.Text += decoded
			return
		}
	}
	*line = append(*line, Span{Text: decoded, Face: f})
}

// iconAt matches /(fa[bklrs]?):fa-([\w-]+)/ at s[i], and gives where it ends.
func iconAt(s string, i int) int {
	if !strings.HasPrefix(s[i:], "fa") {
		return -1
	}
	j := i + 2
	if j < len(s) && strings.IndexByte("bklrs", s[j]) >= 0 && strings.HasPrefix(s[j+1:], ":fa-") {
		j++
	}
	if !strings.HasPrefix(s[j:], ":fa-") {
		return -1
	}
	j += 4
	k := j
	for k < len(s) && (chars.IsWord(int(s[k])) || s[k] == '-') {
		k++
	}
	if k == j {
		return -1
	}
	return k
}

func pushIcons(line *[]Span, text string, f Face, o *Options) {
	if !strings.Contains(text, ":fa-") {
		pushText(line, text, f)
		return
	}
	last := 0
	for i := 0; i < len(text); i++ {
		end := iconAt(text, i)
		if end < 0 {
			continue
		}
		icon := text[i:end]
		pushText(line, text[last:i], f)
		last = end
		switch {
		case o.Icons != nil && o.Icons(icon):
			*line = append(*line, Span{Face: f, Icon: icon})
		case last < len(text) && text[last] == ' ':
			// An icon the host does not have leaves no gap, nor the space that set it off.
			last++
		case last == len(text):
			if n := len(*line); n > 0 && (*line)[n-1].Icon == "" {
				(*line)[n-1].Text = chars.TrimEnd((*line)[n-1].Text)
			}
		}
		i = end - 1
	}
	pushText(line, text[last:], f)
}

// tagAt matches /<\/?([a-zA-Z][\w-]*)(?:\s[^<>]*)?\/?>/ at s[i], and gives the tag's name
// and where the match ends.
func tagAt(s string, i int) (name string, end int) {
	j := i + 1
	if j < len(s) && s[j] == '/' {
		j++
	}
	if j >= len(s) || !(s[j] >= 'a' && s[j] <= 'z' || s[j] >= 'A' && s[j] <= 'Z') {
		return "", -1
	}
	k := j + 1
	for k < len(s) && (chars.IsWord(int(s[k])) || s[k] == '-') {
		k++
	}
	name = s[j:k]
	if k < len(s) {
		if r, n := utf8.DecodeRuneInString(s[k:]); chars.IsSpaceRune(r) {
			m := k + n
			for m < len(s) && s[m] != '<' && s[m] != '>' {
				m++
			}
			if m < len(s) && s[m] == '>' {
				return name, m + 1
			}
			return "", -1
		}
	}
	if k < len(s) && s[k] == '/' {
		k++
	}
	if k < len(s) && s[k] == '>' {
		return name, k + 1
	}
	return "", -1
}

// inline splits inline HTML and, for markdown, emphasis markers into spans.
func inline(line *[]Span, source string, markdown bool, base Face, o *Options) {
	f := base
	var marked Face
	var text strings.Builder
	flush := func() {
		pushIcons(line, text.String(), f, o)
		text.Reset()
	}
	n := len(source)
	for i := 0; i < n; {
		ch := source[i]
		if ch == '<' {
			if name, end := tagAt(source, i); end > 0 {
				tag := strings.ToLower(name)
				closingTag := source[i+1] == '/'
				var bit Face
				switch tag {
				case "b", "strong":
					bit = Bold
				case "i", "em":
					bit = Italic
				}
				if bit != 0 {
					flush()
					if closingTag {
						f = f&^bit | base&bit
					} else {
						f |= bit
					}
				}
				i = end
				continue
			}
		} else if markdown && (ch == '*' || ch == '_') {
			double := i+1 < n && source[i+1] == ch
			bit, size := Italic, 1
			marker := string(ch)
			if double {
				bit, size = Bold, 2
				marker += marker
			}
			open := marked&bit == 0
			prev, next := " ", " "
			if i > 0 {
				_, w := utf8.DecodeLastRuneInString(source[:i])
				prev = source[i-w : i]
			}
			if i+size < n {
				_, w := utf8.DecodeRuneInString(source[i+size:])
				next = source[i+size : i+size+w]
			}
			canOpen := open && chars.Trim(next) != "" && strings.Contains(source[i+size:], marker)
			canClose := !open && chars.Trim(prev) != ""
			wordy := ch == '_' && len(prev) == 1 && chars.IsWord(int(prev[0])) && len(next) == 1 && chars.IsWord(int(next[0]))
			if (canOpen || canClose) && !wordy {
				flush()
				if canOpen {
					marked |= bit
					f |= bit
				} else {
					marked &^= bit
					f = f&^bit | base&bit
				}
				i += size
				continue
			}
		}
		text.WriteByte(ch)
		i++
	}
	flush()
}

func measureSpans(line []Span, o *Options) float64 {
	total := 0.0
	for i := range line {
		s := &line[i]
		if s.Icon != "" {
			s.Width = o.Size * 1.25
		} else {
			s.Width = o.Measurer.Width(s.Text, o.Size, s.Face)
		}
		total += s.Width
	}
	return total
}

func wrap(line []Span, o *Options, out *[][]Span, widths *[]float64) {
	var current []Span
	width := 0.0
	commit := func() {
		if n := len(current); n > 0 && current[n-1].Icon == "" {
			last := &current[n-1]
			if trimmed := chars.TrimEnd(last.Text); trimmed != last.Text {
				last.Text = trimmed
				w := o.Measurer.Width(trimmed, o.Size, last.Face)
				width += w - last.Width
				last.Width = w
			}
		}
		*out = append(*out, current)
		*widths = append(*widths, width)
		current = nil
		width = 0
	}
	for _, span := range line {
		if span.Icon != "" {
			w := o.Size * 1.25
			if width > 0 && width+w > o.MaxWidth {
				commit()
			}
			span.Width = w
			current = append(current, span)
			width += w
			continue
		}
		for _, part := range pieces(span.Text) {
			if width > 0 && width+o.Measurer.Width(chars.TrimEnd(part), o.Size, span.Face) > o.MaxWidth {
				commit()
				part = chars.TrimStart(part)
			}
			if n := len(current); n > 0 && current[n-1].Face == span.Face && current[n-1].Icon == "" {
				last := &current[n-1]
				last.Text += part
				last.Width = o.Measurer.Width(last.Text, o.Size, last.Face)
				width = 0
				for _, s := range current {
					width += s.Width
				}
			} else {
				w := o.Measurer.Width(part, o.Size, span.Face)
				current = append(current, Span{Text: part, Face: span.Face, Width: w})
				width += w
			}
		}
	}
	if len(current) > 0 || len(*out) == 0 {
		commit()
	}
}
