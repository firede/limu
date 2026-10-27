// Package url checks the URLs Mermaid diagrams link to and show, as Pele's src/util/url.ts
// does: SanitizeURL refuses what @braintree/sanitize-url refuses, which Mermaid applies to
// click targets, LinkURL and ImageURL keep a URL only where the host allows its scheme, and
// InternalLink checks the name of a note that is linked to.
//
// Pele's linked and relAttr, which write a link's markup around what a node draws, are left
// to whoever writes the markup: all they decide is in LinkURL.
package url

import (
	"math"
	"slices"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/text"
)

const blank = "about:blank"

// decode is decodeURIComponent, or uri itself where that throws.
func decode(uri string) string {
	if !strings.Contains(uri, "%") {
		return uri
	}
	var b strings.Builder
	for k := 0; k < len(uri); k++ {
		if uri[k] != '%' {
			b.WriteByte(uri[k])
			continue
		}
		first, ok := hexByte(uri, k)
		if !ok {
			return uri
		}
		k += 2
		if first < 0x80 {
			b.WriteByte(first)
			continue
		}
		n := 0
		for mask := byte(0x80); first&mask != 0; mask >>= 1 {
			n++
		}
		if n == 1 || n > 4 {
			return uri
		}
		seq := []byte{first}
		for j := 1; j < n; j++ {
			k++
			if k >= len(uri) || uri[k] != '%' {
				return uri
			}
			c, ok := hexByte(uri, k)
			if !ok || c&0xc0 != 0x80 {
				return uri
			}
			seq = append(seq, c)
			k += 2
		}
		r, size := utf8.DecodeRune(seq)
		if r == utf8.RuneError || size != len(seq) {
			return uri
		}
		b.Write(seq)
	}
	return b.String()
}

// hexByte reads the two hexadecimal digits after the % at uri[k].
func hexByte(uri string, k int) (byte, bool) {
	if k+2 >= len(uri) {
		return 0, false
	}
	h, l := uri[k+1], uri[k+2]
	if !chars.IsHex(int(h)) || !chars.IsHex(int(l)) {
		return 0, false
	}
	return unhex(h)<<4 | unhex(l), true
}

func unhex(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	}
	return c - 'A' + 10
}

// isCtrl is /[\u0000-\u001F\u007F-\u009F\u2000-\u200D\uFEFF\uD800-\uDFFF\uFFFE\uFFFF]/u, but
// for the lone surrogates, which a Go string can't hold. Only an entity code makes one, and
// readEntities takes it out.
func isCtrl(r rune) bool {
	return r <= 0x1f || r >= 0x7f && r <= 0x9f || r >= 0x2000 && r <= 0x200d || r == 0xfeff || r == 0xfffe || r == 0xffff
}

func hasCtrl(s string) bool { return strings.ContainsFunc(s, isCtrl) }

func dropCtrl(s string) string {
	if !hasCtrl(s) {
		return s
	}
	return strings.Map(func(r rune) rune {
		if isCtrl(r) {
			return -1
		}
		return r
	}, s)
}

// entityAt matches /&#(\w+)(^\w|;)?/ at s[i], where `^\w` can never match: the end of the match
// and its \w+.
func entityAt(s string, i int) (end int, digits string, ok bool) {
	if !strings.HasPrefix(s[i:], "&#") {
		return 0, "", false
	}
	j := i + 2
	for j < len(s) && chars.IsWord(int(s[j])) {
		j++
	}
	if j == i+2 {
		return 0, "", false
	}
	digits = s[i+2 : j]
	if j < len(s) && s[j] == ';' {
		j++
	}
	return j, digits, true
}

// decodeEntities replaces each match of RE_HTML_ENTITIES with String.fromCharCode of its
// digits. It gives code units, as two of them may make a pair of surrogates and one may be
// half of a pair.
func decodeEntities(s string) []uint16 {
	var out []uint16
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '&' {
			continue
		}
		end, digits, ok := entityAt(s, i)
		if !ok {
			continue
		}
		out = append(out, utf16.Encode([]rune(s[last:i]))...)
		out = append(out, toUint16(chars.Number(digits)))
		last = end
		i = end - 1
	}
	return append(out, utf16.Encode([]rune(s[last:]))...)
}

// readEntities makes three of a round's replacements: RE_HTML_ENTITIES, RE_HTML_CTRL_ENTITY
// and RE_CTRL. Where there are entity codes it makes them in code units, since a code may
// give half of a surrogate pair. Its other half may follow at once, or once a &newline; or
// &tab; between them is gone, and otherwise RE_CTRL takes the lone half out.
func readEntities(s string) string {
	if !strings.Contains(s, "&#") {
		return dropCtrl(remove(s, ctrlEntityAt))
	}
	u := decodeEntities(s)
	var kept []uint16
	for i := 0; i < len(u); i++ {
		if end := ctrlEntityAt16(u, i); end > 0 {
			i = end - 1
			continue
		}
		kept = append(kept, u[i])
	}
	out := kept[:0]
	for i := 0; i < len(kept); i++ {
		c := kept[i]
		switch {
		case utf16.IsSurrogate(rune(c)):
			if c < 0xdc00 && i+1 < len(kept) && kept[i+1] >= 0xdc00 && kept[i+1] <= 0xdfff {
				out = append(out, c, kept[i+1])
				i++
			}
		case !isCtrl(rune(c)):
			out = append(out, c)
		}
	}
	return string(utf16.Decode(out))
}

// toUint16 is JavaScript's ToUint16.
func toUint16(f float64) uint16 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0
	}
	m := math.Mod(math.Trunc(f), 65536)
	if m < 0 {
		m += 65536
	}
	return uint16(m)
}

func hasEntity(s string) bool {
	for i := strings.Index(s, "&#"); i >= 0; {
		if _, _, ok := entityAt(s, i); ok {
			return true
		}
		next := strings.Index(s[i+1:], "&#")
		if next < 0 {
			break
		}
		i += 1 + next
	}
	return false
}

// ctrlEntityAt matches /&(newline|tab);/i at s[i].
func ctrlEntityAt(s string, i int) int {
	if s[i] != '&' {
		return 0
	}
	for _, name := range []string{"newline", "tab"} {
		end := i + 1 + len(name)
		if end < len(s) && chars.EqualFoldASCII(s[i+1:end], name) && s[end] == ';' {
			return end + 1
		}
	}
	return 0
}

// ctrlEntityAt16 is ctrlEntityAt in code units.
func ctrlEntityAt16(u []uint16, i int) int {
	if u[i] != '&' {
		return 0
	}
next:
	for _, name := range []string{"newline", "tab"} {
		end := i + 1 + len(name)
		if end >= len(u) || u[end] != ';' {
			continue
		}
		for k := 0; k < len(name); k++ {
			c := u[i+1+k]
			if c >= 'A' && c <= 'Z' {
				c += 'a' - 'A'
			}
			if c != uint16(name[k]) {
				continue next
			}
		}
		return end + 1
	}
	return 0
}

// wsEscapeAt matches /(\\|%5[cC])((%(6[eE]|72|74))|[nrt])/ at s[i].
func wsEscapeAt(s string, i int) int {
	j := 0
	switch {
	case s[i] == '\\':
		j = i + 1
	case strings.HasPrefix(s[i:], "%5c") || strings.HasPrefix(s[i:], "%5C"):
		j = i + 3
	default:
		return 0
	}
	if j < len(s) && (s[j] == 'n' || s[j] == 'r' || s[j] == 't') {
		return j + 1
	}
	if j+2 < len(s) && s[j] == '%' {
		switch s[j+1 : j+3] {
		case "6e", "6E", "72", "74":
			return j + 3
		}
	}
	return 0
}

// remove takes out every match of a pattern, as replace with a global pattern does when it
// replaces with nothing.
func remove(s string, match func(string, int) int) string {
	var b strings.Builder
	last := 0
	for i := 0; i < len(s); i++ {
		if end := match(s, i); end > 0 {
			b.WriteString(s[last:i])
			last = end
			i = end - 1
		}
	}
	if last == 0 {
		return s
	}
	b.WriteString(s[last:])
	return b.String()
}

func has(s string, match func(string, int) int) bool {
	for i := 0; i < len(s); i++ {
		if match(s, i) > 0 {
			return true
		}
	}
	return false
}

// schemeOf is the first match of /^.+(:|&colon;)/gim: from the start of a line, as much of it
// as runs up to its last colon.
func schemeOf(s string) (string, bool) {
	for start := 0; start <= len(s); {
		end := start
		for end < len(s) {
			c, size := chars.UnitAt(s, end)
			if chars.IsLineEnd(c) {
				break
			}
			end += size
		}
		for x := end - 1; x > start; x-- {
			if s[x] == ':' {
				return s[start : x+1], true
			}
			if strings.HasPrefix(s[x:end], "&colon;") {
				return s[start : x+7], true
			}
		}
		if end >= len(s) {
			break
		}
		_, size := chars.UnitAt(s, end)
		start = end + size
	}
	return "", false
}

// invalidProtocol is /^([^\w]*)(javascript|data|vbscript)/im of a protocol, which has no line
// breaks.
func invalidProtocol(p string) bool {
	i := 0
	for i < len(p) && !chars.IsWord(int(p[i])) {
		i++
	}
	rest := p[i:]
	for _, name := range []string{"javascript", "data", "vbscript"} {
		if chars.HasPrefixFold(rest, name) {
			return true
		}
	}
	return false
}

// unwrap removes the control characters, entity codes and escapes that could hide a scheme,
// and with percent, percent escapes too, as Pele's unwrap does. It's false for a URL wrapped
// too many times to be a real one.
func unwrap(url string, percent bool) (string, bool) {
	out := chars.Trim(url)
	if percent {
		out = decode(out)
	}
	for rounds := 1; ; rounds++ {
		if rounds > 32 {
			return "", false
		}
		out = dropCtrl(out)
		out = readEntities(out)
		out = remove(out, wsEscapeAt)
		out = chars.Trim(out)
		if percent {
			out = decode(out)
		}
		if !hasCtrl(out) && !hasEntity(out) && !has(out, ctrlEntityAt) && !has(out, wsEscapeAt) {
			return out, true
		}
	}
}

// SanitizeURL refuses what @braintree/sanitize-url refuses, the check Mermaid applies to click
// targets, as Pele's sanitizeUrl does. Unlike it, it keeps percent escapes, so `file=A%26B`
// still names the note `A&B`.
func SanitizeURL(url string) string {
	if url == "" {
		return blank
	}
	decoded, ok := unwrap(url, true)
	kept, ok2 := unwrap(url, false)
	if !ok || !ok2 || decoded == "" || kept == "" {
		return blank
	}
	if decoded[0] == '.' || decoded[0] == '/' {
		return kept
	}

	scheme, ok := schemeOf(chars.TrimStart(decoded))
	if !ok {
		return kept
	}
	protocol := chars.Trim(chars.ToLower(scheme))
	if invalidProtocol(protocol) {
		return blank
	}

	normalized := strings.ReplaceAll(chars.TrimStart(kept), "\\", "/")
	if protocol == "mailto:" || strings.Contains(protocol, "://") {
		return normalized
	}
	if protocol == "http:" || protocol == "https:" {
		href, ok := parseHTTP(normalized)
		if !ok {
			return blank
		}
		return href
	}
	return normalized
}

// SafeURL is a URL as written in a diagram, checked: entity codes are read first, so they
// cannot hide a scheme. Pele's safeUrl.
func SafeURL(url string) string { return SanitizeURL(text.DecodeEntities(url)) }

// The schemes a link and an image may have when the host names none.
var (
	LinkSchemes  = []string{"http", "https", "mailto", "tel"}
	ImageSchemes = []string{"http", "https"}
)

// allowed reports whether a URL is relative or its scheme is one of schemes, as Pele's allowed
// does. Stricter than Mermaid, which lets any scheme but three through: a diagram in a shared
// note could otherwise open a local file, a network share, or another application.
func allowed(url string, schemes []string) bool {
	if len(url) >= 2 && (url[0] == '/' || url[0] == '\\') && (url[1] == '/' || url[1] == '\\') {
		return false
	}
	i := 0
	if i < len(url) && chars.IsLetter(int(url[i])) {
		i++
		for i < len(url) && (chars.IsLetter(int(url[i])) || chars.IsDigit(int(url[i])) || url[i] == '+' || url[i] == '.' || url[i] == '-') {
			i++
		}
		if i < len(url) && url[i] == ':' {
			return slices.Contains(schemes, strings.ToLower(url[:i]))
		}
	}
	return true
}

// allow keeps a URL that allowed passes both as it is and decoded, for an app that decodes the
// href before opening it, as Pele's allow does.
func allow(url string, schemes []string) (string, bool) {
	decoded, ok := unwrap(url, true)
	if ok && allowed(url, schemes) && allowed(decoded, schemes) {
		return url, true
	}
	return "", false
}

// LinkURL keeps a URL that SanitizeURL has passed when it is relative or its scheme is one the
// host allows, and it allows links at all; schemes nil means LinkSchemes. One that is refused
// is left out of the drawing: written as about:blank, a link would still take the page it is
// in somewhere when clicked. Pele's linkUrl, with RenderOptions' links and linkSchemes.
func LinkURL(url string, links bool, schemes []string) (string, bool) {
	if !links {
		return "", false
	}
	if schemes == nil {
		schemes = LinkSchemes
	}
	return allow(url, schemes)
}

// ImageURL is LinkURL for an image, with RenderOptions' images and imageSchemes. Pele's
// imageUrl.
func ImageURL(url string, images bool, schemes []string) (string, bool) {
	if !images {
		return "", false
	}
	if schemes == nil {
		schemes = ImageSchemes
	}
	return allow(url, schemes)
}

// InternalLink reports whether a note's name may be written as the href of an internal link,
// as Pele's internalLink does. The href keeps the name as it is, so it must pass as a link both
// sanitized and as a browser reads it once the SVG is written: without the characters esc
// drops, without tabs and line breaks, and without the control characters and spaces it starts
// with.
func InternalLink(name string, links bool, schemes []string) bool {
	u := SanitizeURL(name)
	if u == blank {
		return false
	}
	if _, ok := LinkURL(u, links, schemes); !ok {
		return false
	}
	// RE_INVALID_XML and then /[\t\n\r]/ take out every control character below the space.
	seen := strings.Map(func(r rune) rune {
		if r <= 0x1f || r == 0x7f || r == 0xfffe || r == 0xffff {
			return -1
		}
		return r
	}, name)
	seen = strings.TrimLeft(seen, " ")
	_, ok := LinkURL(seen, links, schemes)
	return ok
}
