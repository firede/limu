package text

import (
	"strconv"
	"strings"
	"sync"

	"github.com/firede/limu/internal/chars"
)

// The entities Mermaid text may use by name, from Pele's src/text/entities.ts.
const namedEntities = "amp:38,lt:60,gt:62,quot:34,apos:39,nbsp:160,iexcl:161,cent:162,pound:163,curren:164,yen:165,sect:167," +
	"copy:169,laquo:171,not:172,reg:174,deg:176,plusmn:177,sup2:178,sup3:179,micro:181,para:182,middot:183," +
	"sup1:185,raquo:187,frac14:188,frac12:189,frac34:190,iquest:191,times:215,divide:247,ndash:8211,mdash:8212," +
	"lsquo:8216,rsquo:8217,sbquo:8218,ldquo:8220,rdquo:8221,bdquo:8222,dagger:8224,Dagger:8225,bull:8226," +
	"hellip:8230,permil:8240,prime:8242,Prime:8243,lsaquo:8249,rsaquo:8250,euro:8364,trade:8482,larr:8592," +
	"uarr:8593,rarr:8594,darr:8595,harr:8596,lArr:8656,uArr:8657,rArr:8658,dArr:8659,hArr:8660,forall:8704," +
	"part:8706,exist:8707,empty:8709,nabla:8711,isin:8712,notin:8713,ni:8715,prod:8719,sum:8721,minus:8722," +
	"radic:8730,infin:8734,and:8743,or:8744,cap:8745,cup:8746,int:8747,there4:8756,sim:8764,cong:8773," +
	"asymp:8776,ne:8800,equiv:8801,le:8804,ge:8805,sub:8834,sup:8835,sube:8838,supe:8839,oplus:8853,otimes:8855," +
	"perp:8869,sdot:8901,loz:9674,spades:9824,clubs:9827,hearts:9829,diams:9830,check:10003,cross:10007," +
	"equals:61,colon:58,semi:59,num:35,percnt:37,lpar:40,rpar:41,lbrack:91,rbrack:93,lbrace:123,rbrace:125," +
	"vert:124,sol:47,bsol:92,quest:63,excl:33,ast:42,plus:43,comma:44,period:46,dollar:36,commat:64,grave:96," +
	"hat:94,lowbar:95,tilde:126,Alpha:913,Beta:914,Gamma:915,Delta:916,Epsilon:917,Zeta:918,Eta:919,Theta:920," +
	"Iota:921,Kappa:922,Lambda:923,Mu:924,Nu:925,Xi:926,Omicron:927,Pi:928,Rho:929,Sigma:931,Tau:932," +
	"Upsilon:933,Phi:934,Chi:935,Psi:936,Omega:937,alpha:945,beta:946,gamma:947,delta:948,epsilon:949,zeta:950," +
	"eta:951,theta:952,iota:953,kappa:954,lambda:955,mu:956,nu:957,xi:958,omicron:959,pi:960,rho:961,sigmaf:962," +
	"sigma:963,tau:964,upsilon:965,phi:966,chi:967,psi:968,omega:969"

var named = sync.OnceValue(func() map[string]string {
	m := make(map[string]string, 160)
	for pair := range strings.SplitSeq(namedEntities, ",") {
		name, code, _ := strings.Cut(pair, ":")
		n, _ := strconv.Atoi(code)
		m[name] = string(rune(n))
	}
	return m
})

// codePoint is the character n names, or fallback when it names none.
func codePoint(n int64, fallback string) string {
	if n > 0 && n <= 0x10ffff && (n < 0xd800 || n > 0xdfff) {
		return string(rune(n))
	}
	return fallback
}

const (
	placeholderOpen  = "ﬂ°"
	placeholderClose = "¶ß"
)

// decodePlaceholders turns Mermaid's placeholders, from `ﬂ°` to `¶ß` with a second `°`
// marking a numeric code, into characters.
func decodePlaceholders(s string) string {
	var out strings.Builder
	last := 0
	from := strings.Index(s, placeholderOpen)
	for from != -1 {
		body := from + len(placeholderOpen)
		rel := strings.Index(s[body:], "¶")
		if rel == -1 {
			break
		}
		stop := body + rel
		after := stop + len("¶")
		if strings.HasPrefix(s[after:], "ß") {
			end := after + len("ß")
			numeric := strings.HasPrefix(s[body:], "°")
			if numeric {
				body += len("°")
			}
			raw := s[from:end]
			out.WriteString(s[last:from])
			if numeric {
				out.WriteString(codePoint(jsNumber(s[body:stop]), raw))
			} else if c, ok := named()[s[body:stop]]; ok {
				out.WriteString(c)
			} else {
				out.WriteString("&" + s[body:stop] + ";")
			}
			last = end
		}
		next := strings.Index(s[after:], placeholderOpen)
		if next == -1 {
			break
		}
		from = after + next
	}
	if last == 0 {
		return s
	}
	out.WriteString(s[last:])
	return out.String()
}

// jsNumber is JavaScript's Number() of a code's digits, as far as a code point can come of
// it: anything else is -1, which names no character.
func jsNumber(s string) int64 {
	s = chars.Trim(s)
	if s == "" {
		return 0
	}
	base := 10
	if len(s) > 2 && s[0] == '0' {
		switch s[1] {
		case 'x', 'X':
			base = 16
		case 'o', 'O':
			base = 8
		case 'b', 'B':
			base = 2
		}
		if base != 10 {
			s = s[2:]
		}
	}
	if base == 10 {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil || f != float64(int64(f)) || strings.ContainsAny(s, "_xXpP") {
			return -1
		}
		return int64(f)
	}
	n, err := strconv.ParseInt(s, base, 64)
	if err != nil || strings.Contains(s, "_") {
		return -1
	}
	return n
}

// DecodeEntities turns Mermaid's `#35;` and `#quot;` codes, as Mermaid hides them, and HTML
// entities into characters.
func DecodeEntities(s string) string {
	if strings.Contains(s, placeholderClose) {
		s = decodePlaceholders(s)
	}
	if !strings.Contains(s, "&") {
		return s
	}
	var out strings.Builder
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '&' {
			continue
		}
		end, rep, ok := entityAt(s, i)
		if !ok {
			continue
		}
		out.WriteString(s[last:i])
		out.WriteString(rep)
		last = end
		i = end - 1
	}
	if last == 0 {
		return s
	}
	out.WriteString(s[last:])
	return out.String()
}

// entityAt matches /&(?:#(\d+)|#[xX]([0-9a-fA-F]+)|(\w+));/ at s[i], and gives where the
// match ends and what it stands for.
func entityAt(s string, i int) (end int, rep string, ok bool) {
	j := i + 1
	run := func(from int, in func(byte) bool) int {
		k := from
		for k < len(s) && in(s[k]) {
			k++
		}
		return k
	}
	digit := func(b byte) bool { return b >= '0' && b <= '9' }
	hex := func(b byte) bool { return digit(b) || b >= 'a' && b <= 'f' || b >= 'A' && b <= 'F' }
	if j < len(s) && s[j] == '#' {
		if k := run(j+1, digit); k > j+1 && k < len(s) && s[k] == ';' {
			n, err := strconv.ParseInt(s[j+1:k], 10, 64)
			if err != nil {
				n = -1
			}
			return k + 1, codePoint(n, s[i:k+1]), true
		}
		if j+1 < len(s) && (s[j+1] == 'x' || s[j+1] == 'X') {
			if k := run(j+2, hex); k > j+2 && k < len(s) && s[k] == ';' {
				n, err := strconv.ParseInt(s[j+2:k], 16, 64)
				if err != nil {
					n = -1
				}
				return k + 1, codePoint(n, s[i:k+1]), true
			}
		}
		return 0, "", false
	}
	k := run(j, func(b byte) bool { return chars.IsWord(int(b)) })
	if k == j || k >= len(s) || s[k] != ';' {
		return 0, "", false
	}
	if c, found := named()[s[j:k]]; found {
		return k + 1, c, true
	}
	return k + 1, s[i : k+1], true
}
