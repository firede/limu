package url

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/firede/limu/internal/chars"
)

// parseHTTP is new URL(input).toString() for an input whose scheme is http or https, as the
// WHATWG URL Standard parses it and Node's URL (Ada) does, or false where the constructor
// throws.
//
// SanitizeURL comes here only with a URL whose first line has no colon but its scheme's, so
// neither a port nor an IPv6 address can be read; an address in brackets is refused. A host
// of ASCII is lowered, as Ada does, without a check of Punycode in it. Host names outside
// ASCII are lowered and written in Punycode without the whole of UTS #46: a host with white
// space, controls, format characters or unassigned code points is refused, as UTS #46
// refuses most of them, and full-width letters and stops are made ASCII, but its other
// mappings (as of ß, ligatures and combining marks) are not made.
func parseHTTP(input string) (string, bool) {
	input = strings.TrimFunc(input, func(r rune) bool { return r <= 0x20 })
	input = strings.Map(func(r rune) rune {
		if r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, input)

	i := 0
	if i >= len(input) || !isAlpha(input[0]) {
		return "", false
	}
	for i < len(input) && (isAlpha(input[i]) || input[i] >= '0' && input[i] <= '9' || input[i] == '+' || input[i] == '-' || input[i] == '.') {
		i++
	}
	if i >= len(input) || input[i] != ':' {
		return "", false
	}
	scheme := strings.ToLower(input[:i])
	if scheme != "http" && scheme != "https" {
		return "", false
	}
	rest := input[i+1:]

	p := 0
	for p < len(rest) && (rest[p] == '/' || rest[p] == '\\') {
		p++
	}
	end := p
	for end < len(rest) && !strings.ContainsRune("/?#\\", rune(rest[end])) {
		end++
	}
	auth := rest[p:end]

	var user, pass strings.Builder
	hostPort := auth
	if at := strings.LastIndexByte(auth, '@'); at >= 0 {
		seen := false
		for _, r := range auth[:at] {
			if r == ':' && !seen {
				seen = true
				continue
			}
			if seen {
				encodeRune(&pass, r, userinfoSet)
			} else {
				encodeRune(&user, r, userinfoSet)
			}
		}
		hostPort = auth[at+1:]
		if hostPort == "" {
			return "", false
		}
	}

	hostText, portText, hasPort := hostPort, "", false
	inside := false
	for k := 0; k < len(hostPort); k++ {
		switch hostPort[k] {
		case '[':
			inside = true
		case ']':
			inside = false
		case ':':
			if !inside {
				hostText, portText, hasPort = hostPort[:k], hostPort[k+1:], true
			}
		}
		if hasPort {
			break
		}
	}
	if hostText == "" {
		return "", false
	}
	host, ok := parseHost(hostText)
	if !ok {
		return "", false
	}
	port := ""
	if hasPort && portText != "" {
		n := 0
		for k := 0; k < len(portText); k++ {
			c := portText[k]
			if c < '0' || c > '9' {
				return "", false
			}
			n = n*10 + int(c-'0')
			if n > 65535 {
				return "", false
			}
		}
		if !(scheme == "http" && n == 80 || scheme == "https" && n == 443) {
			port = ":" + strconv.Itoa(n)
		}
	} else if hasPort {
		for k := 0; k < len(portText); k++ {
			if portText[k] < '0' || portText[k] > '9' {
				return "", false
			}
		}
	}

	tail := rest[end:]
	q := 0
	if q < len(tail) && (tail[q] == '/' || tail[q] == '\\') {
		q++
	}
	var segs []string
	var buf strings.Builder
	query, fragment := "", ""
	hasQuery, hasFragment := false, false
	for {
		c := byte(0)
		eof := q >= len(tail)
		if !eof {
			c = tail[q]
		}
		if eof || c == '/' || c == '\\' || c == '?' || c == '#' {
			seg := buf.String()
			slash := c == '/' || c == '\\'
			switch {
			case isDoubleDot(seg):
				if len(segs) > 0 {
					segs = segs[:len(segs)-1]
				}
				if !slash {
					segs = append(segs, "")
				}
			case isSingleDot(seg):
				if !slash {
					segs = append(segs, "")
				}
			default:
				segs = append(segs, seg)
			}
			buf.Reset()
			if eof {
				break
			}
			if c == '?' || c == '#' {
				after := tail[q+1:]
				if c == '?' {
					hasQuery = true
					text, frag, found := strings.Cut(after, "#")
					query = encodeString(text, querySet)
					if found {
						hasFragment = true
						fragment = encodeString(frag, fragmentSet)
					}
				} else {
					hasFragment = true
					fragment = encodeString(after, fragmentSet)
				}
				break
			}
			q++
			continue
		}
		r, size := utf8.DecodeRuneInString(tail[q:])
		encodeRune(&buf, r, pathSet)
		q += size
	}

	var out strings.Builder
	out.WriteString(scheme + "://")
	if user.Len() > 0 || pass.Len() > 0 {
		out.WriteString(user.String())
		if pass.Len() > 0 {
			out.WriteString(":" + pass.String())
		}
		out.WriteByte('@')
	}
	out.WriteString(host + port)
	for _, s := range segs {
		out.WriteString("/" + s)
	}
	if hasQuery {
		out.WriteString("?" + query)
	}
	if hasFragment {
		out.WriteString("#" + fragment)
	}
	return out.String(), true
}

func isAlpha(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

func isSingleDot(s string) bool { return s == "." || chars.EqualFoldASCII(s, "%2e") }

func isDoubleDot(s string) bool {
	for _, d := range []string{"..", ".%2e", "%2e.", "%2e%2e"} {
		if chars.EqualFoldASCII(s, d) {
			return true
		}
	}
	return false
}

// The ASCII characters each part of a URL percent-encodes, besides the C0 controls and all
// above U+007E.
const (
	pathSet     = " \"#<>?^`{}"
	querySet    = " \"#'<>"
	fragmentSet = " \"<>`"
	userinfoSet = " \"#<>?^`{}/:;=@[\\]|"
)

func encodeByte(b *strings.Builder, c byte, set string) {
	if c < 0x20 || c > 0x7e || strings.IndexByte(set, c) >= 0 {
		b.WriteString("%" + strings.ToUpper(strconv.FormatUint(uint64(c)>>4, 16)+strconv.FormatUint(uint64(c)&15, 16)))
		return
	}
	b.WriteByte(c)
}

func encodeRune(b *strings.Builder, r rune, set string) {
	var tmp [4]byte
	n := utf8.EncodeRune(tmp[:], r)
	for _, c := range tmp[:n] {
		encodeByte(b, c, set)
	}
}

func encodeString(s string, set string) string {
	var b strings.Builder
	for _, r := range s {
		encodeRune(&b, r, set)
	}
	return b.String()
}

func percentDecode(s string) string {
	if !strings.Contains(s, "%") {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			if c, ok := hexByte(s, i); ok {
				b.WriteByte(c)
				i += 2
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func parseHost(input string) (string, bool) {
	if strings.HasPrefix(input, "[") {
		return "", false
	}
	domain := percentDecode(input)
	if !utf8.ValidString(domain) {
		return "", false
	}
	ascii, ok := domainToASCII(domain)
	if !ok {
		return "", false
	}
	for i := 0; i < len(ascii); i++ {
		c := ascii[i]
		if c <= 0x20 || c == 0x7f || strings.IndexByte("#%/:<>?@[\\]^|", c) >= 0 {
			return "", false
		}
	}
	if endsInNumber(ascii) {
		return parseIPv4(ascii)
	}
	return ascii, true
}

func domainToASCII(domain string) (string, bool) {
	ascii := true
	for i := 0; i < len(domain); i++ {
		if domain[i] >= utf8.RuneSelf {
			ascii = false
			break
		}
	}
	// Node's URL, as Ada has it, lowers a host of ASCII and checks no Punycode in it.
	if ascii {
		return strings.ToLower(domain), true
	}
	mapped := strings.Map(func(r rune) rune {
		switch {
		case r == 0x3002 || r == 0xff0e || r == 0xff61:
			return '.'
		case r >= 0xff01 && r <= 0xff5e:
			return unicode.ToLower(r - 0xff01 + 0x21)
		case r < utf8.RuneSelf:
			return unicode.ToLower(r)
		case r == utf8.RuneError || unicode.IsSpace(r) || unicode.In(r, unicode.Cc, unicode.Cf, unicode.Co, unicode.Cs, unicode.Zs, unicode.Zl, unicode.Zp) || !unicode.In(r, unicode.L, unicode.M, unicode.N, unicode.P, unicode.S):
			return utf8.RuneError
		}
		return unicode.ToLower(r)
	}, domain)
	if strings.ContainsRune(mapped, utf8.RuneError) {
		return "", false
	}
	labels := strings.Split(mapped, ".")
	for i, l := range labels {
		if len(l) >= 4 && l[:4] == "xn--" {
			if _, ok := punyDecode(l[4:]); !ok {
				return "", false
			}
			continue
		}
		if strings.IndexFunc(l, func(r rune) bool { return r >= utf8.RuneSelf }) >= 0 {
			labels[i] = "xn--" + punyEncode([]rune(l))
		}
	}
	out := strings.Join(labels, ".")
	return out, out != ""
}

func endsInNumber(s string) bool {
	parts := strings.Split(s, ".")
	if parts[len(parts)-1] == "" {
		if len(parts) == 1 {
			return false
		}
		parts = parts[:len(parts)-1]
	}
	last := parts[len(parts)-1]
	if last != "" && strings.Trim(last, "0123456789") == "" {
		return true
	}
	_, ok := ipv4Number(last)
	return ok
}

// ipv4Number reads a part of an IPv4 address: decimal, octal after a 0, or hexadecimal after
// 0x. Values past 2^40 are kept at 2^40, which is past every limit they are compared with.
func ipv4Number(s string) (uint64, bool) {
	if s == "" {
		return 0, false
	}
	base := uint64(10)
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		s, base = s[2:], 16
	} else if len(s) >= 2 && s[0] == '0' {
		s, base = s[1:], 8
	}
	var n uint64
	for i := 0; i < len(s); i++ {
		d := uint64(99)
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			d = uint64(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint64(c-'A') + 10
		}
		if d >= base {
			return 0, false
		}
		n = min(n*base+d, 1<<40)
	}
	return n, true
}

func parseIPv4(s string) (string, bool) {
	parts := strings.Split(s, ".")
	if parts[len(parts)-1] == "" && len(parts) > 1 {
		parts = parts[:len(parts)-1]
	}
	if len(parts) > 4 {
		return "", false
	}
	nums := make([]uint64, len(parts))
	for i, p := range parts {
		n, ok := ipv4Number(p)
		if !ok {
			return "", false
		}
		nums[i] = n
	}
	for _, n := range nums[:len(nums)-1] {
		if n > 255 {
			return "", false
		}
	}
	last := nums[len(nums)-1]
	if last >= uint64(1)<<(8*(5-len(nums))) {
		return "", false
	}
	addr := last
	for i, n := range nums[:len(nums)-1] {
		addr += n << (8 * (3 - i))
	}
	return strconv.FormatUint(addr>>24, 10) + "." + strconv.FormatUint(addr>>16&255, 10) + "." +
		strconv.FormatUint(addr>>8&255, 10) + "." + strconv.FormatUint(addr&255, 10), true
}

// Punycode, as RFC 3492 has it.
const (
	punyBase = 36
	tMin     = 1
	tMax     = 26
	skew     = 38
	damp     = 700
	initBias = 72
	initN    = 128
)

func adapt(delta, points int, first bool) int {
	if first {
		delta /= damp
	} else {
		delta /= 2
	}
	delta += delta / points
	k := 0
	for delta > (punyBase-tMin)*tMax/2 {
		delta /= punyBase - tMin
		k += punyBase
	}
	return k + (punyBase-tMin+1)*delta/(delta+skew)
}

func digit(d int) byte {
	if d < 26 {
		return byte('a' + d)
	}
	return byte('0' + d - 26)
}

func punyEncode(input []rune) string {
	var out []byte
	for _, r := range input {
		if r < 0x80 {
			out = append(out, byte(r))
		}
	}
	b := len(out)
	h := b
	if b > 0 {
		out = append(out, '-')
	}
	n, delta, bias := initN, 0, initBias
	for h < len(input) {
		m := int(^uint(0) >> 1)
		for _, r := range input {
			if int(r) >= n && int(r) < m {
				m = int(r)
			}
		}
		delta += (m - n) * (h + 1)
		n = m
		for _, r := range input {
			if int(r) < n {
				delta++
			}
			if int(r) == n {
				q := delta
				for k := punyBase; ; k += punyBase {
					t := k - bias
					if t < tMin {
						t = tMin
					} else if t > tMax {
						t = tMax
					}
					if q < t {
						break
					}
					out = append(out, digit(t+(q-t)%(punyBase-t)))
					q = (q - t) / (punyBase - t)
				}
				out = append(out, digit(q))
				bias = adapt(delta, h+1, h == b)
				delta = 0
				h++
			}
		}
		delta++
		n++
	}
	return string(out)
}

func punyDecode(s string) ([]rune, bool) {
	var out []rune
	b := strings.LastIndexByte(s, '-')
	if b > 0 {
		for _, c := range []byte(s[:b]) {
			if c >= 0x80 {
				return nil, false
			}
			out = append(out, rune(c))
		}
		s = s[b+1:]
	} else if b == 0 {
		s = s[1:]
	}
	n, i, bias := initN, 0, initBias
	for p := 0; p < len(s); {
		old, w := i, 1
		for k := punyBase; ; k += punyBase {
			if p >= len(s) {
				return nil, false
			}
			c := s[p]
			p++
			var d int
			switch {
			case c >= '0' && c <= '9':
				d = int(c-'0') + 26
			case c >= 'a' && c <= 'z':
				d = int(c - 'a')
			case c >= 'A' && c <= 'Z':
				d = int(c - 'A')
			default:
				return nil, false
			}
			i += d * w
			if i > 1<<30 {
				return nil, false
			}
			t := k - bias
			if t < tMin {
				t = tMin
			} else if t > tMax {
				t = tMax
			}
			if d < t {
				break
			}
			w *= punyBase - t
		}
		bias = adapt(i-old, len(out)+1, old == 0)
		n += i / (len(out) + 1)
		i %= len(out) + 1
		if n > unicode.MaxRune {
			return nil, false
		}
		out = append(out[:i], append([]rune{rune(n)}, out[i:]...)...)
		i++
	}
	return out, true
}
