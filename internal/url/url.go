// Package url checks the URLs Mermaid diagrams link to and show, as Pele's src/util/url.ts
// does: sanitizeUrl follows @braintree/sanitize-url, which Mermaid applies to click targets,
// and LinkURL and ImageURL keep a URL only where the host allows its scheme.
//
// Pele's linked and relAttr, which write a link's markup around what a node draws, are left
// to whoever writes the markup: all they decide is in LinkURL.
package url

import (
	"math"
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

// isCtrl is /[\u0000-\u001F\u007F-\u009F\u2000-\u200D\uFEFF]/.
func isCtrl(r rune) bool {
	return r <= 0x1f || r >= 0x7f && r <= 0x9f || r >= 0x2000 && r <= 0x200d || r == 0xfeff
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
// digits. It works in UTF-16 code units, as two of them may make a pair of surrogates.
func decodeEntities(s string) string {
	if !strings.Contains(s, "&#") {
		return s
	}
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
	out = append(out, utf16.Encode([]rune(s[last:]))...)
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

// SanitizeURL follows the algorithm of @braintree/sanitize-url, as Pele's sanitizeUrl does.
func SanitizeURL(url string) string {
	if url == "" {
		return blank
	}
	decoded := decode(chars.Trim(url))
	for rounds := 1; ; rounds++ {
		// Each round peels one layer of encoding. A URL wrapped this many times is not a real one.
		if rounds > 32 {
			return blank
		}
		decoded = dropCtrl(decoded)
		decoded = decodeEntities(decoded)
		decoded = remove(decoded, ctrlEntityAt)
		decoded = dropCtrl(decoded)
		decoded = remove(decoded, wsEscapeAt)
		decoded = chars.Trim(decoded)
		decoded = decode(decoded)
		if !hasCtrl(decoded) && !hasEntity(decoded) && !has(decoded, ctrlEntityAt) && !has(decoded, wsEscapeAt) {
			break
		}
	}

	if decoded == "" {
		return blank
	}
	if decoded[0] == '.' || decoded[0] == '/' {
		return decoded
	}

	trimmed := chars.TrimStart(decoded)
	scheme, ok := schemeOf(trimmed)
	if !ok {
		return decoded
	}
	protocol := chars.Trim(chars.ToLower(scheme))
	if invalidProtocol(protocol) {
		return blank
	}

	normalized := strings.ReplaceAll(trimmed, "\\", "/")
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

// allow keeps a URL that is relative or whose scheme is one of schemes. Stricter than Mermaid,
// which lets any scheme but three through: a diagram in a shared note could otherwise open a
// local file, a network share, or another application.
func allow(url string, schemes []string) (string, bool) {
	if len(url) >= 2 && (url[0] == '/' || url[0] == '\\') && (url[1] == '/' || url[1] == '\\') {
		return "", false
	}
	i := 0
	if i < len(url) && chars.IsLetter(int(url[i])) {
		i++
		for i < len(url) && (chars.IsLetter(int(url[i])) || chars.IsDigit(int(url[i])) || url[i] == '+' || url[i] == '.' || url[i] == '-') {
			i++
		}
		if i < len(url) && url[i] == ':' {
			name := strings.ToLower(url[:i])
			for _, s := range schemes {
				if s == name {
					return url, true
				}
			}
			return "", false
		}
	}
	return url, true
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
