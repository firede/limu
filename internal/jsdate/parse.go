package jsdate

import (
	"math"
	"unicode/utf16"
)

// Parse is new Date(s) for a string: V8's DateParser, which reads the date time strings of
// the spec first, and what it leaves goes to a legacy parser compatible with Safari's that
// takes words, numbers and signs in many orders. It gives NaN for what neither reads. A
// date with no offset is in local time, but one of the spec's that has no time is in UTC.
func (z Zone) Parse(s string) float64 {
	var out [8]float64
	if !parseDate(utf16.Encode([]rune(s)), &out) {
		return math.NaN()
	}
	day := MakeDay(out[outYear], out[outMonth], out[outDay])
	t := MakeTime(out[outHour], out[outMinute], out[outSecond], out[outMillisecond])
	date := MakeDate(day, t)
	if math.IsNaN(out[outUTCOffset]) {
		if !(date >= -maxTimeBeforeUTC && date <= maxTimeBeforeUTC) {
			return math.NaN()
		}
		date = float64(z.toUTC(int64(date)))
	} else {
		date -= float64(out[outUTCOffset] * 1000)
		if date < -MaxTime || date > MaxTime {
			return math.NaN()
		}
	}
	return TimeClip(date)
}

// The slots of the parser's output.
const (
	outYear = iota
	outMonth
	outDay
	outHour
	outMinute
	outSecond
	outMillisecond
	outUTCOffset
)

// none is V8's kNone, a slot not yet given.
const none = math.MaxInt32

// smiMax is the largest small integer of a V8 without pointer compression, as Node is built.
const smiMax = math.MaxInt32

// maxSignificantDigits is how many digits of a number the parser keeps.
const maxSignificantDigits = 9

func between(x, lo, hi int) bool { return lo <= x && x <= hi }

// reader is V8's InputReader: it reads code units, and a NUL reads as the end.
type reader struct {
	s     []uint16
	index int
	ch    int
}

func (r *reader) next() {
	if r.index < len(r.s) {
		r.ch = int(r.s[r.index])
	} else {
		r.ch = 0
	}
	r.index++
}

func (r *reader) position() int { return r.index }

func (r *reader) readUnsignedNumeral() int {
	n, i := 0, 0
	for r.ch == '0' {
		r.next()
	}
	for r.isDigit() {
		if i < maxSignificantDigits {
			n = n*10 + r.ch - '0'
		}
		i++
		r.next()
	}
	return n
}

func (r *reader) readWord(prefix *[3]int) int {
	n := 0
	for ; r.isAlphaOrAbove() && !isWhiteSpace(r.ch); r.next() {
		if n < 3 {
			prefix[n] = r.ch | 0x20
		}
		n++
	}
	for i := n; i < 3; i++ {
		prefix[i] = 0
	}
	return n
}

func (r *reader) skip(c int) bool {
	if r.ch == c {
		r.next()
		return true
	}
	return false
}

func (r *reader) skipWhiteSpace() bool {
	if isWhiteSpace(r.ch) || isLineTerminator(r.ch) {
		r.next()
		return true
	}
	return false
}

func (r *reader) skipParentheses() bool {
	if r.ch != '(' {
		return false
	}
	balance := 0
	for {
		if r.ch == ')' {
			balance--
		} else if r.ch == '(' {
			balance++
		}
		r.next()
		if !(balance > 0 && r.ch != 0) {
			break
		}
	}
	return true
}

func (r *reader) isEnd() bool          { return r.ch == 0 }
func (r *reader) isDigit() bool        { return r.ch >= '0' && r.ch <= '9' }
func (r *reader) isAlphaOrAbove() bool { return r.ch >= 'A' }

// isWhiteSpace is the spec's WhiteSpace, as V8 knows it.
func isWhiteSpace(c int) bool {
	switch c {
	case 9, 11, 12, 32, 0xa0, 0x1680, 0x202f, 0x205f, 0x3000, 0xfeff:
		return true
	}
	return c >= 0x2000 && c <= 0x200a
}

func isLineTerminator(c int) bool { return c == 10 || c == 13 || c == 0x2028 || c == 0x2029 }

// The kinds of keyword, V8's KeywordType.
const (
	kwInvalid = iota
	kwMonthName
	kwTimeZoneName
	kwTimeSeparator
	kwAMPM
)

// The kinds of token, V8's tags; a keyword's tag is its kind, from 0.
const (
	tagInvalid    = -6
	tagUnknown    = -5
	tagWhiteSpace = -4
	tagNumber     = -3
	tagSymbol     = -2
	tagEnd        = -1
)

type token struct {
	tag    int
	length int
	value  int
}

func (t token) isInvalid() bool                { return t.tag == tagInvalid }
func (t token) isNumber() bool                 { return t.tag == tagNumber }
func (t token) isSymbol(c int) bool            { return t.tag == tagSymbol && t.value == c }
func (t token) isWhiteSpace() bool             { return t.tag == tagWhiteSpace }
func (t token) isEnd() bool                    { return t.tag == tagEnd }
func (t token) isKeyword() bool                { return t.tag >= 0 }
func (t token) isFixedLengthNumber(n int) bool { return t.isNumber() && t.length == n }
func (t token) isAsciiSign() bool              { return t.tag == tagSymbol && (t.value == '-' || t.value == '+') }
func (t token) asciiSign() int                 { return 44 - t.value }
func (t token) isKeywordType(kind int) bool    { return t.tag == kind }
func (t token) isKeywordZ() bool               { return t.tag == kwTimeZoneName && t.length == 1 && t.value == 0 }
func (t token) keywordType() int               { return t.tag }

// keywords is V8's KeywordTable: a word matches by its first three letters, and only a month's
// name may be longer.
var keywords = [...]struct {
	prefix [3]int
	kind   int
	value  int
}{
	{[3]int{'j', 'a', 'n'}, kwMonthName, 1},
	{[3]int{'f', 'e', 'b'}, kwMonthName, 2},
	{[3]int{'m', 'a', 'r'}, kwMonthName, 3},
	{[3]int{'a', 'p', 'r'}, kwMonthName, 4},
	{[3]int{'m', 'a', 'y'}, kwMonthName, 5},
	{[3]int{'j', 'u', 'n'}, kwMonthName, 6},
	{[3]int{'j', 'u', 'l'}, kwMonthName, 7},
	{[3]int{'a', 'u', 'g'}, kwMonthName, 8},
	{[3]int{'s', 'e', 'p'}, kwMonthName, 9},
	{[3]int{'o', 'c', 't'}, kwMonthName, 10},
	{[3]int{'n', 'o', 'v'}, kwMonthName, 11},
	{[3]int{'d', 'e', 'c'}, kwMonthName, 12},
	{[3]int{'a', 'm', 0}, kwAMPM, 0},
	{[3]int{'p', 'm', 0}, kwAMPM, 12},
	{[3]int{'u', 't', 0}, kwTimeZoneName, 0},
	{[3]int{'u', 't', 'c'}, kwTimeZoneName, 0},
	{[3]int{'z', 0, 0}, kwTimeZoneName, 0},
	{[3]int{'g', 'm', 't'}, kwTimeZoneName, 0},
	{[3]int{'c', 'd', 't'}, kwTimeZoneName, -5},
	{[3]int{'c', 's', 't'}, kwTimeZoneName, -6},
	{[3]int{'e', 'd', 't'}, kwTimeZoneName, -4},
	{[3]int{'e', 's', 't'}, kwTimeZoneName, -5},
	{[3]int{'m', 'd', 't'}, kwTimeZoneName, -6},
	{[3]int{'m', 's', 't'}, kwTimeZoneName, -7},
	{[3]int{'p', 'd', 't'}, kwTimeZoneName, -7},
	{[3]int{'p', 's', 't'}, kwTimeZoneName, -8},
	{[3]int{'t', 0, 0}, kwTimeSeparator, 0},
}

func lookupKeyword(prefix [3]int, length int) (kind, value int) {
	for _, k := range keywords {
		if k.prefix == prefix && (length <= 3 || k.kind == kwMonthName) {
			return k.kind, k.value
		}
	}
	return kwInvalid, 0
}

// tokenizer is V8's DateStringTokenizer.
type tokenizer struct {
	in    *reader
	ahead token
}

func newTokenizer(in *reader) *tokenizer {
	t := &tokenizer{in: in}
	t.ahead = t.scan()
	return t
}

func (t *tokenizer) scan() token {
	in := t.in
	pre := in.position()
	if in.isEnd() {
		return token{tag: tagEnd}
	}
	if in.isDigit() {
		n := in.readUnsignedNumeral()
		return token{tag: tagNumber, length: in.position() - pre, value: n}
	}
	for _, c := range [...]int{':', '-', '+', '.', ')'} {
		if in.skip(c) {
			return token{tag: tagSymbol, length: 1, value: c}
		}
	}
	if in.isAlphaOrAbove() && !isWhiteSpace(in.ch) {
		var prefix [3]int
		n := in.readWord(&prefix)
		kind, value := lookupKeyword(prefix, n)
		return token{tag: kind, length: n, value: value}
	}
	if in.skipWhiteSpace() {
		return token{tag: tagWhiteSpace, length: in.position() - pre}
	}
	if in.skipParentheses() {
		return token{tag: tagUnknown}
	}
	in.next()
	return token{tag: tagUnknown}
}

func (t *tokenizer) next() token {
	result := t.ahead
	t.ahead = t.scan()
	return result
}

func (t *tokenizer) peek() token { return t.ahead }

func (t *tokenizer) skipSymbol(c int) bool {
	if t.ahead.isSymbol(c) {
		t.ahead = t.scan()
		return true
	}
	return false
}

// timeZone is V8's TimeZoneComposer.
type timeZone struct{ sign, hour, minute int }

func (tz *timeZone) set(hours int) {
	tz.sign = 1
	if hours < 0 {
		tz.sign = -1
	}
	tz.hour = hours * tz.sign
	tz.minute = 0
}

func (tz *timeZone) setSign(sign int) {
	tz.sign = 1
	if sign < 0 {
		tz.sign = -1
	}
}

func (tz *timeZone) isExpecting(n int) bool {
	return tz.hour != none && tz.minute == none && isMinute(n)
}

func (tz *timeZone) isUTC() bool   { return tz.hour == 0 && tz.minute == 0 }
func (tz *timeZone) isEmpty() bool { return tz.hour == none }

func (tz *timeZone) write(out *[8]float64) bool {
	if tz.sign == none {
		out[outUTCOffset] = math.NaN()
		return true
	}
	if tz.hour == none {
		tz.hour = 0
	}
	if tz.minute == none {
		tz.minute = 0
	}
	total := uint32(tz.hour)*3600 + uint32(tz.minute)*60
	if total > smiMax {
		return false
	}
	seconds := int(total)
	if tz.sign < 0 {
		seconds = -seconds
	}
	out[outUTCOffset] = float64(seconds)
	return true
}

func isMinute(x int) bool      { return between(x, 0, 59) }
func isHour(x int) bool        { return between(x, 0, 23) }
func isSecond(x int) bool      { return between(x, 0, 59) }
func isHour12(x int) bool      { return between(x, 0, 12) }
func isMillisecond(x int) bool { return between(x, 0, 999) }
func isMonth(x int) bool       { return between(x, 1, 12) }
func isDay(x int) bool         { return between(x, 1, 31) }

// clock is V8's TimeComposer.
type clock struct {
	comp       [4]int
	index      int
	hourOffset int
}

func (c *clock) isEmpty() bool { return c.index == 0 }

func (c *clock) isExpecting(n int) bool {
	return c.index == 1 && isMinute(n) || c.index == 2 && isSecond(n) || c.index == 3 && isMillisecond(n)
}

func (c *clock) add(n int) bool {
	if c.index < len(c.comp) {
		c.comp[c.index] = n
		c.index++
		return true
	}
	return false
}

func (c *clock) addFinal(n int) bool {
	if !c.add(n) {
		return false
	}
	for c.index < len(c.comp) {
		c.comp[c.index] = 0
		c.index++
	}
	return true
}

func (c *clock) write(out *[8]float64) bool {
	for c.index < len(c.comp) {
		c.comp[c.index] = 0
		c.index++
	}
	hour, minute, second, ms := c.comp[0], c.comp[1], c.comp[2], c.comp[3]
	if c.hourOffset != none {
		if !isHour12(hour) {
			return false
		}
		hour %= 12
		hour += c.hourOffset
	}
	if !isHour(hour) || !isMinute(minute) || !isSecond(second) || !isMillisecond(ms) {
		if hour != 24 || minute != 0 || second != 0 || ms != 0 {
			return false
		}
	}
	out[outHour] = float64(hour)
	out[outMinute] = float64(minute)
	out[outSecond] = float64(second)
	out[outMillisecond] = float64(ms)
	return true
}

// calendar is V8's DayComposer.
type calendar struct {
	comp       [3]int
	index      int
	namedMonth int
	isoDate    bool
}

func (d *calendar) isEmpty() bool { return d.index == 0 }

func (d *calendar) add(n int) bool {
	if d.index < len(d.comp) {
		d.comp[d.index] = n
		d.index++
		return true
	}
	return false
}

func (d *calendar) write(out *[8]float64) bool {
	if d.index < 1 {
		return false
	}
	for d.index < len(d.comp) {
		d.comp[d.index] = 1
		d.index++
	}
	year, month, day := 0, none, none
	if d.namedMonth == none {
		if d.isoDate || d.index == 3 && !isDay(d.comp[0]) {
			year, month, day = d.comp[0], d.comp[1], d.comp[2]
		} else {
			month, day = d.comp[0], d.comp[1]
			if d.index == 3 {
				year = d.comp[2]
			}
		}
	} else {
		month = d.namedMonth
		if d.index == 1 {
			day = d.comp[0]
		} else if !isDay(d.comp[0]) {
			year, day = d.comp[0], d.comp[1]
		} else {
			day, year = d.comp[0], d.comp[1]
		}
	}
	if !d.isoDate {
		if between(year, 0, 49) {
			year += 2000
		} else if between(year, 50, 99) {
			year += 1900
		}
	}
	if year < math.MinInt32 || year > smiMax || !isMonth(month) || !isDay(day) {
		return false
	}
	out[outYear] = float64(year)
	out[outMonth] = float64(month - 1)
	out[outDay] = float64(day)
	return true
}

// readMilliseconds is V8's ReadMilliseconds: the first three significant digits of a
// number, as its length shows where they are.
func readMilliseconds(t token) int {
	number, length := t.value, t.length
	if length < 3 {
		if length == 1 {
			number *= 100
		} else if length == 2 {
			number *= 10
		}
	} else if length > 3 {
		if length > maxSignificantDigits {
			length = maxSignificantDigits
		}
		factor := 1
		for {
			factor *= 10
			length--
			if length <= 3 {
				break
			}
		}
		number /= factor
	}
	return number
}

// parseES5 is V8's ParseES5DateTime: the date time string format of the spec,
// [('-'|'+')yy]yyyy[-MM[-DD]][THH:mm[:ss[.sss]][Z|(+|-)hh:mm]]. It gives the first token it
// did not read, for the legacy parser to go on with, or an invalid one.
func parseES5(scanner *tokenizer, day *calendar, clk *clock, tz *timeZone) token {
	if scanner.peek().isAsciiSign() {
		sign := scanner.next()
		if !scanner.peek().isFixedLengthNumber(6) {
			return sign
		}
		year := scanner.next().value
		if sign.asciiSign() < 0 && year == 0 {
			return sign
		}
		day.add(sign.asciiSign() * year)
	} else if scanner.peek().isFixedLengthNumber(4) {
		day.add(scanner.next().value)
	} else {
		return scanner.next()
	}
	if scanner.skipSymbol('-') {
		if !scanner.peek().isFixedLengthNumber(2) || !isMonth(scanner.peek().value) {
			return scanner.next()
		}
		day.add(scanner.next().value)
		if scanner.skipSymbol('-') {
			if !scanner.peek().isFixedLengthNumber(2) || !isDay(scanner.peek().value) {
				return scanner.next()
			}
			day.add(scanner.next().value)
		}
	}
	if !scanner.peek().isKeywordType(kwTimeSeparator) {
		if !scanner.peek().isEnd() {
			return scanner.next()
		}
	} else {
		scanner.next()
		if !scanner.peek().isFixedLengthNumber(2) || !between(scanner.peek().value, 0, 24) {
			return token{tag: tagInvalid}
		}
		hourIs24 := scanner.peek().value == 24
		clk.add(scanner.next().value)
		if !scanner.skipSymbol(':') {
			return token{tag: tagInvalid}
		}
		if !scanner.peek().isFixedLengthNumber(2) || !isMinute(scanner.peek().value) || hourIs24 && scanner.peek().value > 0 {
			return token{tag: tagInvalid}
		}
		clk.add(scanner.next().value)
		if scanner.skipSymbol(':') {
			if !scanner.peek().isFixedLengthNumber(2) || !isSecond(scanner.peek().value) || hourIs24 && scanner.peek().value > 0 {
				return token{tag: tagInvalid}
			}
			clk.add(scanner.next().value)
			if scanner.skipSymbol('.') {
				if !scanner.peek().isNumber() || hourIs24 && scanner.peek().value > 0 {
					return token{tag: tagInvalid}
				}
				clk.add(readMilliseconds(scanner.next()))
			}
		}
		if scanner.peek().isKeywordZ() {
			scanner.next()
			tz.set(0)
		} else if scanner.peek().isSymbol('+') || scanner.peek().isSymbol('-') {
			if scanner.next().value == '+' {
				tz.setSign(1)
			} else {
				tz.setSign(-1)
			}
			if scanner.peek().isFixedLengthNumber(4) {
				hourmin := scanner.next().value
				hour, minute := hourmin/100, hourmin%100
				if !isHour(hour) || !isMinute(minute) {
					return token{tag: tagInvalid}
				}
				tz.hour, tz.minute = hour, minute
			} else {
				if !scanner.peek().isFixedLengthNumber(2) || !isHour(scanner.peek().value) {
					return token{tag: tagInvalid}
				}
				tz.hour = scanner.next().value
				if !scanner.skipSymbol(':') {
					return token{tag: tagInvalid}
				}
				if !scanner.peek().isFixedLengthNumber(2) || !isMinute(scanner.peek().value) {
					return token{tag: tagInvalid}
				}
				tz.minute = scanner.next().value
			}
		}
		if !scanner.peek().isEnd() {
			return token{tag: tagInvalid}
		}
	}
	// A date with no time and no offset is in UTC; a date and time with no offset is local.
	if tz.isEmpty() && clk.isEmpty() {
		tz.set(0)
	}
	day.isoDate = true
	return token{tag: tagEnd}
}

// parseDate is V8's DateParser::Parse.
func parseDate(s []uint16, out *[8]float64) bool {
	in := &reader{s: s}
	in.next()
	scanner := newTokenizer(in)
	tz := &timeZone{sign: none, hour: none, minute: none}
	clk := &clock{hourOffset: none}
	day := &calendar{namedMonth: none}

	unhandled := parseES5(scanner, day, clk, tz)
	if unhandled.isInvalid() {
		return false
	}
	hasReadNumber := !day.isEmpty()
	for tok := unhandled; !tok.isEnd(); tok = scanner.next() {
		switch {
		case tok.isNumber():
			hasReadNumber = true
			n := tok.value
			if scanner.skipSymbol(':') {
				if scanner.skipSymbol(':') {
					if !clk.isEmpty() {
						return false
					}
					clk.add(n)
					clk.add(0)
				} else {
					if !clk.add(n) {
						return false
					}
					if scanner.peek().isSymbol('.') {
						scanner.next()
					}
				}
			} else if scanner.skipSymbol('.') && clk.isExpecting(n) {
				clk.add(n)
				if !scanner.peek().isNumber() {
					return false
				}
				ms := readMilliseconds(scanner.next())
				if ms < 0 {
					return false
				}
				clk.addFinal(ms)
			} else if tz.isExpecting(n) {
				tz.minute = n
			} else if clk.isExpecting(n) {
				clk.addFinal(n)
				p := scanner.peek()
				if !p.isEnd() && !p.isWhiteSpace() && !p.isKeywordZ() && !p.isAsciiSign() {
					return false
				}
			} else {
				if !day.add(n) {
					return false
				}
				scanner.skipSymbol('-')
			}
		case tok.isKeyword():
			if tok.keywordType() == kwAMPM && !clk.isEmpty() {
				clk.hourOffset = tok.value
			} else if tok.keywordType() == kwMonthName {
				day.namedMonth = tok.value
				scanner.skipSymbol('-')
			} else if tok.keywordType() == kwTimeZoneName && hasReadNumber {
				tz.set(tok.value)
			} else {
				// Words are garbage, allowed only before a number and apart from it.
				if hasReadNumber {
					return false
				}
				if scanner.peek().isNumber() {
					return false
				}
			}
		case tok.isAsciiSign() && (tz.isUTC() || !clk.isEmpty()):
			// An offset from UTC, only after UTC or a time.
			tz.setSign(tok.asciiSign())
			n, length := 0, 0
			if scanner.peek().isNumber() {
				t := scanner.next()
				length, n = t.length, t.value
			}
			hasReadNumber = true
			if scanner.peek().isSymbol(':') {
				tz.hour = n
				tz.minute = none
			} else if length == 2 || length == 1 {
				tz.hour, tz.minute = n, 0
			} else if length == 4 || length == 3 {
				tz.hour, tz.minute = n/100, n%100
			} else {
				return false
			}
		case (tok.isAsciiSign() || tok.isSymbol(')')) && hasReadNumber:
			return false
		}
	}
	return day.write(out) && clk.write(out) && tz.write(out)
}
