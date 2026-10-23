package text

import (
	"math"
	"slices"
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

// WrappedLineHeight is the height of a line of a label of more than one line, in ems: its
// lines sit closer, so that it reads as one block.
const WrappedLineHeight = 1.3

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
// <strong>, <i>, <em>, <code>, <sub>, <sup>, <u> and their kin set its faces (tagFaces), as
// do ** and _ in markdown, other HTML tags are dropped, fa:fa-name is an icon, entities are
// decoded, and lines wider than MaxWidth wrap between words, between any two Chinese or
// Japanese characters, and at a zero-width space or a soft hyphen. A label of more than one
// line has lines WrappedLineHeight high.
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
		width := Width(o.Measurer, text, o.Size, o.Face)
		if width <= o.MaxWidth || !(strings.Contains(text, " ") || hasCJK(text) || hasHint(text)) {
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
		snug := snugLineHeight(len(lines), o.Size, lineHeight)
		return Label{Lines: lines, Widths: widths, Width: widest, Height: float64(len(lines)) * snug, Size: o.Size, LineHeight: snug}
	}

	source := chars.Trim(raw)
	if o.Markdown {
		source = chars.Trim(dedent(strings.ReplaceAll(strings.ReplaceAll(raw, "\r\n", "\n"), "\r", "\n")))
	}
	var lines [][]Span
	var widths []float64
	width := 0.0
	// HTML tags can span the breaks, while markdown's emphasis stays within its line.
	open := openTags{face: o.Face}
	for _, part := range splitBreaks(source) {
		var line []Span
		inline(&line, chars.Trim(part), o.Markdown, o.Face, &open, &o)
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
	snug := snugLineHeight(len(lines), o.Size, lineHeight)
	return Label{Lines: lines, Widths: widths, Width: width, Height: float64(len(lines)) * snug, Size: o.Size, LineHeight: snug}
}

// snugLineHeight is the height of the lines of a label of n lines: lineHeight for one, and
// closer for more, as Pele's layoutLabel and layoutLines set it.
func snugLineHeight(n int, size, lineHeight float64) float64 {
	if n > 1 {
		return jsmath.Round(size * WrappedLineHeight)
	}
	return lineHeight
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

// softHyphen shows as a hyphen where a line breaks at it, and not otherwise.
const softHyphen = "\u00ad"

// isHint reports whether r is a break hint, a zero-width space or a soft hyphen, where a
// line may break.
func isHint(r rune) bool { return r == 0x200b || r == 0xad }

func hasHint(s string) bool { return strings.ContainsFunc(s, isHint) }

// pieces splits text where a line may break, as Pele's RE_PIECE does:
//
//	\s*(?:[OPEN]*[CJK][CLOSE]*|[^\sCJKHINT]+[HINT]*[CLOSE]*|[HINT]+)\s*|\s+
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
		// [^\sCJKHINT]+[HINT]*[CLOSE]*
		k = runAt(j, func(r rune) bool { return !chars.IsSpaceRune(r) && !isCJK(r) && !isHint(r) })
		if k > j {
			return runAt(runAt(k, isHint), isClosing), true
		}
		// [HINT]+
		if k = runAt(j, isHint); k > j {
			return k, true
		}
		return 0, false
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
			// Every character is a space, a CJK character, a hint or another, so this is not
			// reached.
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

// tagFaces are the faces HTML tags set, as Pele's TAG_STYLES has them.
var tagFaces = map[string]Face{
	"b": Bold, "strong": Bold,
	"i": Italic, "em": Italic, "cite": Italic, "dfn": Italic, "var": Italic,
	"code": Mono, "kbd": Mono, "samp": Mono, "tt": Mono,
	"sub": Sub, "sup": Sup,
	"u": Underline, "ins": Underline,
}

// htmlTags are the tags of HTML, as Pele's HTML_TAGS has them. A label drops them and keeps
// their text, and anything else shaped like a tag, such as the type in List<T>, is text.
var htmlTags = func() map[string]bool {
	m := map[string]bool{}
	for _, t := range strings.Fields("a abbr address article aside audio b bdi bdo big blockquote body br button caption center cite code col " +
		"colgroup data dd del details dfn dialog div dl dt em embed fieldset figcaption figure font footer form h1 h2 " +
		"h3 h4 h5 h6 head header hr html i iframe img input ins kbd label legend li link main map mark meta meter nav " +
		"noscript object ol optgroup option output p picture pre progress q rp rt ruby s samp script section select " +
		"small source span strike strong style sub summary sup table tbody td template textarea tfoot th thead time " +
		"title tr track tt u ul var video wbr") {
		m[t] = true
	}
	return m
}()

// openTags is what the HTML tags of a label leave open from one line to the next, as Pele's
// Open holds it: the faces they set, and the <sub> and <sup> tags open, innermost last, of
// which only the innermost shifts text.
type openTags struct {
	face   Face
	shifts []Face
}

// inline splits inline HTML and, for markdown, emphasis markers into spans, as Pele's inline
// does. A line starts with what HTML tags left open on the lines before it, in open, and
// leaves there what they leave open at its end.
func inline(line *[]Span, source string, markdown bool, base Face, open *openTags, o *Options) {
	f := open.face
	var marked Face
	var text strings.Builder
	flush := func() {
		shift := Face(0)
		if n := len(open.shifts); n > 0 {
			shift = open.shifts[n-1]
		}
		pushIcons(line, text.String(), f|shift, o)
		text.Reset()
	}
	n := len(source)
	for i := 0; i < n; {
		ch := source[i]
		if ch == '<' {
			if name, end := tagAt(source, i); end > 0 {
				tag := strings.ToLower(name)
				closingTag := source[i+1] == '/'
				bit := tagFaces[tag]
				if bit&shifted != 0 {
					flush()
					at := -1
					for k, sh := range open.shifts {
						if sh == bit {
							at = k
						}
					}
					if !closingTag {
						open.shifts = append(open.shifts, bit)
					} else if at != -1 {
						open.shifts = slices.Delete(open.shifts, at, at+1)
					}
				} else if bit != 0 {
					flush()
					if closingTag {
						open.face = open.face&^bit | base&bit
						f = f&^bit | base&bit
					} else {
						open.face |= bit
						f |= bit
					}
				}
				if bit != 0 || htmlTags[tag] {
					i = end
					continue
				}
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
			s.Width = Width(o.Measurer, s.Text, o.Size, s.Face)
		}
		total += s.Width
	}
	return total
}

// wrap breaks a line into lines no wider than o.MaxWidth where it can, as Pele's wrap does.
// A line breaks after a piece (pieces), and the spaces it ends with are dropped. A piece
// ending in a soft hyphen fits if it does with a hyphen in its place, or if the rest of its
// word does, which hides the hyphen; where the line breaks at it, the hyphen shows.
func wrap(line []Span, o *Options, out *[][]Span, widths *[]float64) {
	var current []Span
	width := 0.0
	measure := func(s string, f Face) float64 { return Width(o.Measurer, s, o.Size, f) }
	commit := func(hyphen bool) {
		if n := len(current); n > 0 && current[n-1].Icon == "" {
			last := &current[n-1]
			trimmed := chars.TrimEnd(last.Text)
			if hyphen && strings.HasSuffix(last.Text, softHyphen) {
				trimmed = strings.TrimSuffix(trimmed, softHyphen) + "-"
			}
			if trimmed != last.Text {
				last.Text = trimmed
				w := measure(trimmed, last.Face)
				width += w - last.Width
				last.Width = w
			}
		}
		*out = append(*out, current)
		*widths = append(*widths, width)
		current = nil
		width = 0
	}
	for at, span := range line {
		if span.Icon != "" {
			w := o.Size * 1.25
			if width > 0 && width+w > o.MaxWidth {
				commit(false)
			}
			span.Width = w
			current = append(current, span)
			width += w
			continue
		}
		parts := pieces(span.Text)
		// own is the width of each piece without its spaces or soft hyphen, and rest that of
		// what is left of its word from it on, which is not known past the end of the span.
		soft := strings.Contains(span.Text, softHyphen)
		var own, rest []float64
		if soft {
			n := len(parts)
			own = make([]float64, n)
			rest = make([]float64, n+1)
			for q := n - 1; q >= 0; q-- {
				// A soft hyphen before a space is no break of its own: the space is.
				shy := strings.HasSuffix(parts[q], softHyphen)
				end := chars.TrimEnd(parts[q])
				if shy {
					end = strings.TrimSuffix(end, softHyphen)
				}
				own[q] = measure(end, span.Face)
				switch {
				case !shy:
					rest[q] = own[q]
				case q == n-1 && at == len(line)-1:
					rest[q] = own[q]
				case q == n-1:
					rest[q] = math.Inf(1)
				default:
					rest[q] = own[q] + rest[q+1]
				}
			}
		}
		for p, part := range parts {
			var partWidth float64
			if soft {
				partWidth = own[p]
			} else {
				partWidth = measure(chars.TrimEnd(part), span.Face)
			}
			var fits bool
			if !soft || !strings.HasSuffix(part, softHyphen) {
				fits = width+partWidth <= o.MaxWidth
			} else {
				fits = width+measure(strings.TrimSuffix(part, softHyphen)+"-", span.Face) <= o.MaxWidth || width+rest[p] <= o.MaxWidth
			}
			if width > 0 && !fits {
				// A break at a space leaves a soft hyphen before it hidden.
				commit(chars.TrimStart(part) == part)
				part = chars.TrimStart(part)
			}
			if n := len(current); n > 0 && current[n-1].Face == span.Face && current[n-1].Icon == "" {
				last := &current[n-1]
				last.Text += part
				// Measuring again for each piece of no width would take quadratic time.
				if partWidth > 0 || chars.TrimEnd(part) != part {
					before := last.Width
					last.Width = measure(last.Text, last.Face)
					width += last.Width - before
				}
			} else {
				w := measure(part, span.Face)
				current = append(current, Span{Text: part, Face: span.Face, Width: w})
				width += w
			}
		}
	}
	if len(current) > 0 || len(*out) == 0 {
		commit(false)
	}
}
