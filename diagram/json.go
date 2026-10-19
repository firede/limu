package diagram

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf16"
)

var errJSON = errors.New("not JSON")

// parseJSON is JSON.parse: strict JSON, with objects as Config and arrays as []any. A number
// too large for a float64 is an infinity, as JavaScript reads it, where Go's encoding/json
// would refuse it.
func parseJSON(s string) (any, error) {
	p := jsonParser{s: s}
	p.space()
	v, err := p.value()
	if err != nil {
		return nil, err
	}
	p.space()
	if p.i != len(s) {
		return nil, errJSON
	}
	return v, nil
}

type jsonParser struct {
	s string
	i int
}

func (p *jsonParser) space() {
	for p.i < len(p.s) {
		switch p.s[p.i] {
		case ' ', '\t', '\n', '\r':
			p.i++
		default:
			return
		}
	}
}

func (p *jsonParser) value() (any, error) {
	if p.i >= len(p.s) {
		return nil, errJSON
	}
	switch c := p.s[p.i]; {
	case c == '{':
		p.i++
		out := Config{}
		p.space()
		if p.i < len(p.s) && p.s[p.i] == '}' {
			p.i++
			return out, nil
		}
		for {
			p.space()
			if p.i >= len(p.s) || p.s[p.i] != '"' {
				return nil, errJSON
			}
			key, err := p.str()
			if err != nil {
				return nil, err
			}
			p.space()
			if p.i >= len(p.s) || p.s[p.i] != ':' {
				return nil, errJSON
			}
			p.i++
			p.space()
			v, err := p.value()
			if err != nil {
				return nil, err
			}
			out[key] = v
			p.space()
			if p.i < len(p.s) && p.s[p.i] == ',' {
				p.i++
				continue
			}
			if p.i < len(p.s) && p.s[p.i] == '}' {
				p.i++
				return out, nil
			}
			return nil, errJSON
		}
	case c == '[':
		p.i++
		out := []any{}
		p.space()
		if p.i < len(p.s) && p.s[p.i] == ']' {
			p.i++
			return out, nil
		}
		for {
			p.space()
			v, err := p.value()
			if err != nil {
				return nil, err
			}
			out = append(out, v)
			p.space()
			if p.i < len(p.s) && p.s[p.i] == ',' {
				p.i++
				continue
			}
			if p.i < len(p.s) && p.s[p.i] == ']' {
				p.i++
				return out, nil
			}
			return nil, errJSON
		}
	case c == '"':
		return p.str()
	case c == '-' || c >= '0' && c <= '9':
		return p.number()
	}
	for _, lit := range []struct {
		word  string
		value any
	}{{"true", true}, {"false", false}, {"null", nil}} {
		if strings.HasPrefix(p.s[p.i:], lit.word) {
			p.i += len(lit.word)
			return lit.value, nil
		}
	}
	return nil, errJSON
}

func (p *jsonParser) number() (any, error) {
	start := p.i
	if p.s[p.i] == '-' {
		p.i++
	}
	digits := func() int {
		from := p.i
		for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
		}
		return p.i - from
	}
	if p.i < len(p.s) && p.s[p.i] == '0' {
		p.i++
	} else if digits() == 0 {
		return nil, errJSON
	}
	if p.i < len(p.s) && p.s[p.i] == '.' {
		p.i++
		if digits() == 0 {
			return nil, errJSON
		}
	}
	if p.i < len(p.s) && (p.s[p.i] == 'e' || p.s[p.i] == 'E') {
		p.i++
		if p.i < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
			p.i++
		}
		if digits() == 0 {
			return nil, errJSON
		}
	}
	f, _ := strconv.ParseFloat(p.s[start:p.i], 64)
	return f, nil
}

func (p *jsonParser) str() (string, error) {
	p.i++
	var units []uint16
	var b strings.Builder
	flush := func() {
		if len(units) > 0 {
			b.WriteString(string(utf16.Decode(units)))
			units = units[:0]
		}
	}
	for {
		if p.i >= len(p.s) {
			return "", errJSON
		}
		c := p.s[p.i]
		switch {
		case c == '"':
			p.i++
			flush()
			return b.String(), nil
		case c < 0x20:
			return "", errJSON
		case c == '\\':
			if p.i+1 >= len(p.s) {
				return "", errJSON
			}
			e := p.s[p.i+1]
			p.i += 2
			if e == 'u' {
				if p.i+4 > len(p.s) {
					return "", errJSON
				}
				n, err := strconv.ParseUint(p.s[p.i:p.i+4], 16, 16)
				if err != nil || strings.ContainsAny(p.s[p.i:p.i+4], "+-_") {
					return "", errJSON
				}
				units = append(units, uint16(n))
				p.i += 4
				continue
			}
			flush()
			switch e {
			case '"', '\\', '/':
				b.WriteByte(e)
			case 'b':
				b.WriteByte('\b')
			case 'f':
				b.WriteByte('\f')
			case 'n':
				b.WriteByte('\n')
			case 'r':
				b.WriteByte('\r')
			case 't':
				b.WriteByte('\t')
			default:
				return "", errJSON
			}
		default:
			flush()
			b.WriteByte(c)
			p.i++
		}
	}
}
