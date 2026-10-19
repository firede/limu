package gantt

import (
	"math"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsdate"
	"github.com/firede/limu/internal/jsmath"
)

// The parts of dayjs 1.11 that Mermaid's gantt code relies on, with the plugins it loads
// (custom parse format, advanced format, ISO week) and the English locale, as Pele's dates.ts
// has them: strict parsing against a format, formatting, and calendar arithmetic, all in
// local time. Quirks are kept, because a date is accepted only when formatting the parsed
// result gives back the input.
//
// A date is a JavaScript time value (see jsdate), and local time is the zone's.

var months = [...]string{"January", "February", "March", "April", "May", "June", "July", "August", "September",
	"October", "November", "December"}

var weekdays = [...]string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}

const (
	dayMs  = 86400000
	weekMs = 7 * dayMs
)

// A unit of a duration, as Pele's Unit.
type unit string

// zone does the date arithmetic of a chart in its time zone.
type zone struct{ jsdate.Zone }

// numStr is JavaScript's String() of a number, which Pele calls throughout dates.ts and axis.ts.
func numStr(v float64) string { return chars.NumberString(v) }

// pad is Pele's pad: String(value), with zeros before it to length.
func pad(value float64, length int) string {
	s := numStr(value)
	if n := chars.Len16(s); n < length {
		return strings.Repeat("0", length-n) + s
	}
	return s
}

// ordinal is Pele's ordinal: n with its English ordinal suffix, as Do writes a day.
func ordinal(n float64) string {
	s := [...]string{"th", "st", "nd", "rd"}
	v := math.Mod(n, 100)
	suffix := s[0]
	if k := math.Mod(v-20, 10); k >= 0 && k < 4 && k == math.Trunc(k) {
		suffix = s[int(k)]
	} else if v >= 0 && v < 4 && v == math.Trunc(v) {
		suffix = s[int(v)]
	}
	return numStr(n) + suffix
}

// utcOffset is Pele's utcOffset: minutes ahead of UTC, rounded to a quarter of an hour.
func (z zone) utcOffset(date float64) float64 {
	offset := -jsmath.Round(z.TimezoneOffset(date)/15) * 15
	if offset == 0 {
		return 0
	}
	return offset
}

// offsetString is Pele's zone: the UTC offset as +hh:mm.
func (z zone) offsetString(date float64) string {
	offset := z.utcOffset(date)
	minutes := math.Abs(offset)
	sign := "-"
	if offset >= 0 {
		sign = "+"
	}
	return sign + pad(math.Floor(minutes/60), 2) + ":" + pad(math.Mod(minutes, 60), 2)
}

// isoWeekday is Pele's isoWeekday: getDay() || 7.
func (z zone) isoWeekday(date float64) float64 {
	d := z.Day(date)
	if d == 0 || math.IsNaN(d) {
		return 7
	}
	return d
}

// addDays is Pele's addDays: the date days later on the calendar, at the same time of day.
func (z zone) addDays(date, days float64) float64 {
	return z.SetDate(date, z.Date(date)+days)
}

// daysInMonth is Pele's daysInMonth: the number of days in the month of date.
func (z zone) daysInMonth(date float64) float64 {
	last := z.New(z.FullYear(date), z.Month(date)+1, 0, 0, 0, 0, 0)
	last = z.SetHours(last, 23, 59, 59, 999)
	return z.Date(last)
}

// setCalendar is Pele's setCalendar: it sets the month or the year with apply, keeping
// the day of the month unless the new month is shorter.
func (z zone) setCalendar(date float64, apply func(d float64) float64) float64 {
	next := z.SetDate(date, 1)
	next = apply(next)
	return z.SetDate(next, min(z.Date(date), z.daysInMonth(next)))
}

// add is Pele's add: the date amount units later, months and years by the calendar, days
// and weeks rounded to whole days, and smaller units by the clock.
func (z zone) add(date, amount float64, u unit) float64 {
	switch u {
	case "M":
		return z.setCalendar(date, func(d float64) float64 { return z.SetMonth(d, z.Month(date)+amount) })
	case "y":
		return z.setCalendar(date, func(d float64) float64 { return z.SetFullYear(d, z.FullYear(date)+amount) })
	case "d":
		return z.addDays(date, jsmath.Round(amount))
	case "w":
		return z.addDays(date, jsmath.Round(float64(7*amount)))
	case "h":
		return jsdate.TimeClip(date + float64(amount*3600000))
	case "m":
		return jsdate.TimeClip(date + float64(amount*60000))
	case "s":
		return jsdate.TimeClip(date + float64(amount*1000))
	}
	return jsdate.TimeClip(date + amount)
}

// weekThursday is Pele's weekThursday: the Thursday of the ISO week date is in.
func (z zone) weekThursday(date float64) float64 {
	return z.addDays(date, 4-z.isoWeekday(date))
}

// isoWeek is Pele's isoWeek: the number of the ISO week date is in.
func (z zone) isoWeek(date float64) float64 {
	thursday := z.weekThursday(date)
	jan1 := z.New(z.FullYear(thursday), 0, 1, 0, 0, 0, 0)
	weekday := z.isoWeekday(jan1)
	extra := 0.0
	if weekday > 4 {
		extra = 7
	}
	first := z.addDays(jan1, 4-weekday+extra)
	zoneDelta := float64((z.utcOffset(first) - z.utcOffset(thursday)) * 60000)
	weeks := (thursday - first - zoneDelta) / weekMs
	if weeks < 0 {
		w := math.Ceil(weeks)
		if w == 0 || math.IsNaN(w) {
			w = 0
		}
		return w + 1
	}
	return math.Floor(weeks) + 1
}

const defaultFormat = "YYYY-MM-DDTHH:mm:ssZ"

// bracketAt is the length of `\[([^\]]+)]` at byte i of s, or 0: the first alternative of
// Pele's RE_ADVANCED and RE_FORMAT, and its RE_LITERAL.
func bracketAt(s string, i int) int {
	if s[i] != '[' {
		return 0
	}
	j := strings.IndexByte(s[i+1:], ']')
	if j <= 0 {
		return 0
	}
	return j + 2
}

// run is the number of bytes of c at i of s, up to most. Pele has no such function; it is a
// helper of the regular expressions written out here.
func run(s string, i int, c byte, most int) int {
	n := 0
	for n < most && i+n < len(s) && s[i+n] == c {
		n++
	}
	return n
}

// advancedAt is the length of a match of
// /\[([^\]]+)]|Q|wo|ww|w|WW|W|zzz|z|gggg|GGGG|Do|X|x|k{1,2}|S/ at i, or 0.
// It is Pele's RE_ADVANCED, written out by hand.
func advancedAt(s string, i int) int {
	if n := bracketAt(s, i); n > 0 {
		return n
	}
	rest := s[i:]
	for _, alt := range [...]string{"Q", "wo", "ww", "w", "WW", "W", "zzz", "z", "gggg", "GGGG", "Do", "X", "x"} {
		if strings.HasPrefix(rest, alt) {
			return len(alt)
		}
	}
	if rest[0] == 'k' {
		return run(s, i, 'k', 2)
	}
	if rest[0] == 'S' {
		return 1
	}
	return 0
}

// formatAt is the length of a match of
// /\[([^\]]+)]|YYYY|YY|M{1,4}|D{1,2}|d{1,4}|H{1,2}|h{1,2}|a|A|m{1,2}|s{1,2}|Z{1,2}|SSS/ at
// i, or 0. It is Pele's RE_FORMAT, written out by hand.
func formatAt(s string, i int) int {
	if n := bracketAt(s, i); n > 0 {
		return n
	}
	switch c := s[i]; c {
	case 'Y':
		if n := run(s, i, 'Y', 4); n == 4 {
			return 4
		} else if n >= 2 {
			return 2
		}
	case 'M', 'd':
		return run(s, i, c, 4)
	case 'D', 'H', 'h', 'm', 's', 'Z':
		return run(s, i, c, 2)
	case 'a', 'A':
		return 1
	case 'S':
		if run(s, i, 'S', 3) == 3 {
			return 3
		}
	}
	return 0
}

// replaceAll is a global replace with a regular expression whose matches at is: each match,
// leftmost first, is replaced with what repl gives for it. It is String.prototype.replace
// with a global regular expression and a function, as Pele's format and compile call it.
func replaceAll(s string, at func(s string, i int) int, repl func(match string) string) string {
	var b strings.Builder
	b.Grow(len(s))
	from := 0
	// A replacement depends on the match alone, so a run of one token, as a long format
	// may hold, is replaced once.
	var last, lastOut string
	for i := 0; i < len(s); {
		n := at(s, i)
		if n == 0 {
			i++
			continue
		}
		b.WriteString(s[from:i])
		if match := s[i : i+n]; match != last || last == "" {
			last, lastOut = match, repl(match)
		}
		b.WriteString(lastOut)
		i += n
		from = i
	}
	b.WriteString(s[from:])
	return b.String()
}

// format is Pele's format. It reports false for the week-of-year and time zone name
// tokens, which need dayjs plugins Mermaid does not load and so fail there.
func (z zone) format(date float64, template string) (string, bool) {
	if !jsdate.Valid(date) {
		return "Invalid Date", true
	}
	str := template
	if str == "" {
		str = defaultFormat
	}
	if str == "YYYY-MM-DD" {
		return z.isoDate(date), true
	}
	unsupported := false
	advanced := replaceAll(str, advancedAt, func(match string) string {
		switch match {
		case "Q":
			return numStr(math.Ceil((z.Month(date) + 1) / 3))
		case "Do":
			return "[" + ordinal(z.Date(date)) + "]"
		case "GGGG":
			return numStr(z.FullYear(z.weekThursday(date)))
		case "W":
			return numStr(z.isoWeek(date))
		case "WW":
			return pad(z.isoWeek(date), 2)
		case "k":
			return numStr(orZero(z.Hours(date), 24))
		case "kk":
			return pad(orZero(z.Hours(date), 24), 2)
		case "X":
			return numStr(math.Floor(date / 1000))
		case "x":
			return numStr(date)
		case "gggg", "wo", "w", "ww", "z", "zzz":
			unsupported = true
		}
		return match
	})
	if unsupported {
		return "", false
	}
	hours := z.Hours(date)
	return replaceAll(advanced, formatAt, func(match string) string {
		if match[0] == '[' {
			return match[1 : len(match)-1]
		}
		switch match {
		case "YY":
			s := numStr(z.FullYear(date))
			return sliceLast16(s, 2)
		case "YYYY":
			return pad(z.FullYear(date), 4)
		case "M":
			return numStr(z.Month(date) + 1)
		case "MM":
			return pad(z.Month(date)+1, 2)
		case "MMM":
			return months[int(z.Month(date))][:3]
		case "MMMM":
			return months[int(z.Month(date))]
		case "D":
			return numStr(z.Date(date))
		case "DD":
			return pad(z.Date(date), 2)
		case "d":
			return numStr(z.Day(date))
		case "dd":
			return weekdays[int(z.Day(date))][:2]
		case "ddd":
			return weekdays[int(z.Day(date))][:3]
		case "dddd":
			return weekdays[int(z.Day(date))]
		case "H":
			return numStr(hours)
		case "HH":
			return pad(hours, 2)
		case "h":
			return numStr(orZero(math.Mod(hours, 12), 12))
		case "hh":
			return pad(orZero(math.Mod(hours, 12), 12), 2)
		case "a":
			if hours < 12 {
				return "am"
			}
			return "pm"
		case "A":
			if hours < 12 {
				return "AM"
			}
			return "PM"
		case "m":
			return numStr(z.Minutes(date))
		case "mm":
			return pad(z.Minutes(date), 2)
		case "s":
			return numStr(z.Seconds(date))
		case "ss":
			return pad(z.Seconds(date), 2)
		case "SSS":
			return pad(z.Milliseconds(date), 3)
		case "Z":
			return z.offsetString(date)
		}
		return strings.Replace(z.offsetString(date), ":", "", 1)
	}), true
}

// orZero is x || or: or in place of 0 or NaN, as Pele writes `|| 12`, `|| 7` and `|| 24`.
func orZero(x, or float64) float64 {
	if x == 0 || math.IsNaN(x) {
		return or
	}
	return x
}

// sliceLast16 is s.slice(-n) for a string of ASCII, as Pele's format takes the last two
// digits of a year for YY.
func sliceLast16(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// isoDate is Pele's isoDate: the date as YYYY-MM-DD.
func (z zone) isoDate(date float64) string {
	year, month, day := z.FullYear(date), z.Month(date)+1, z.Date(date)
	var b strings.Builder
	if year >= 1000 && year <= 9999 {
		b.WriteString(numStr(year))
	} else {
		b.WriteString(pad(year, 4))
	}
	if month < 10 {
		b.WriteString("-0")
	} else {
		b.WriteString("-")
	}
	b.WriteString(numStr(month))
	if day < 10 {
		b.WriteString("-0")
	} else {
		b.WriteString("-")
	}
	b.WriteString(numStr(day))
	return b.String()
}

// fields are what a format reads of a date, Pele's Fields; a field not read is unset.
type fields struct {
	year, month, day, hours, minutes, seconds, milliseconds, week, offset opt
	dayGiven                                                              bool
	afternoon                                                             *bool
}

// opt is a number that may be undefined.
type opt struct {
	v   float64
	set bool
}

// some is a number that is set, as Pele's readers assign a field of Fields.
func some(v float64) opt { return opt{v, true} }

// truthy is the field's truth in JavaScript: set, and neither 0 nor NaN, as Pele's loose tests
// a field.
func (o opt) truthy() bool { return o.set && o.v != 0 && !math.IsNaN(o.v) }

// or is the field || d, as Pele's loose defaults a field.
func (o opt) or(d float64) float64 {
	if o.truthy() {
		return o.v
	}
	return d
}

// A matcher finds a match of one of Pele's MATCH_ patterns in s, as exec does: the
// first place it matches at, and the match.
type matcher func(s []uint16) (match []uint16, ok bool)

// isDigit16 reports whether code unit c is matched by \d, as Pele's MATCH_ patterns use it.
func isDigit16(c uint16) bool { return c >= '0' && c <= '9' }

// search is an unanchored search for a pattern that matches at i with the length at gives,
// or -1. It is RegExp.prototype.exec of a pattern without the y or g flag, as Pele's loose
// executes its MATCH_ patterns.
func search(s []uint16, at func(s []uint16, i int) int) ([]uint16, bool) {
	for i := 0; i < len(s); i++ {
		if n := at(s, i); n >= 0 {
			return s[i : i+n], true
		}
	}
	return nil, false
}

// digits matches \d{n}: Pele's MATCH_1, MATCH_2, MATCH_3 and MATCH_4.
func digits(n int) matcher {
	return func(s []uint16) ([]uint16, bool) {
		return search(s, func(s []uint16, i int) int {
			for k := 0; k < n; k++ {
				if i+k >= len(s) || !isDigit16(s[i+k]) {
					return -1
				}
			}
			return n
		})
	}
}

// oneOrTwo matches \d\d?, Pele's MATCH_1_TO_2.
func oneOrTwo(s []uint16) ([]uint16, bool) {
	return search(s, func(s []uint16, i int) int {
		if !isDigit16(s[i]) {
			return -1
		}
		if i+1 < len(s) && isDigit16(s[i+1]) {
			return 2
		}
		return 1
	})
}

// signed matches [+-]?\d+, Pele's MATCH_SIGNED.
func signed(s []uint16) ([]uint16, bool) {
	return search(s, func(s []uint16, i int) int {
		j := i
		if s[j] == '+' || s[j] == '-' {
			j++
		}
		k := j
		for k < len(s) && isDigit16(s[k]) {
			k++
		}
		if k == j {
			return -1
		}
		return k - i
	})
}

// offsetMatch matches [+-]\d\d:?(\d\d)?|Z, Pele's MATCH_OFFSET.
func offsetMatch(s []uint16) ([]uint16, bool) {
	return search(s, func(s []uint16, i int) int {
		if (s[i] == '+' || s[i] == '-') && i+2 < len(s) && isDigit16(s[i+1]) && isDigit16(s[i+2]) {
			j := i + 3
			if j < len(s) && s[j] == ':' {
				j++
			}
			if j+1 < len(s) && isDigit16(s[j]) && isDigit16(s[j+1]) {
				j += 2
			}
			return j - i
		}
		if s[i] == 'Z' {
			return 1
		}
		return -1
	})
}

// isWordUnit is [^-_:/,()\s\d], the class of Pele's MATCH_WORD.
func isWordUnit(c uint16) bool {
	switch c {
	case '-', '_', ':', '/', ',', '(', ')':
		return false
	}
	return !isDigit16(c) && !chars.IsSpace(int(c))
}

// word matches \d*[^-_:/,()\s\d]+, Pele's MATCH_WORD.
func word(s []uint16) ([]uint16, bool) {
	return search(s, func(s []uint16, i int) int {
		j := i
		for j < len(s) && isDigit16(s[j]) {
			j++
		}
		k := j
		for k < len(s) && isWordUnit(s[k]) {
			k++
		}
		if k == j {
			return -1
		}
		return k - i
	})
}

// A reader stores what a token matched in the fields, or reports that it cannot.
type reader func(f *fields, input string) bool

type step struct {
	literal string
	match   matcher
	read    reader
}

// number is JavaScript's Number() of a string, as Pele's readers take +input.
func number(s string) float64 { return chars.Number(s) }

// field is Pele's field: a reader that stores the number its input is in one field, which
// set names.
func field(set func(f *fields, v float64)) reader {
	return func(f *fields, input string) bool {
		set(f, number(input))
		return true
	}
}

// readOffset is Pele's readOffset: it reads a UTC offset, Z or [+-]hh[:mm], as the minutes
// to add to the time to reach UTC.
func readOffset(f *fields, input string) bool {
	if input == "Z" {
		f.offset = some(0)
		return true
	}
	// input.match(/([+-]|\d\d)/g)
	var parts []string
	for i := 0; i < len(input); {
		if input[i] == '+' || input[i] == '-' {
			parts = append(parts, input[i:i+1])
			i++
		} else if i+1 < len(input) && chars.IsDigit(int(input[i])) && chars.IsDigit(int(input[i+1])) {
			parts = append(parts, input[i:i+2])
			i += 2
		} else {
			i++
		}
	}
	second := 0.0
	if len(parts) > 2 {
		second = orZero(number(parts[2]), 0)
	}
	minutes := float64(number(parts[1])*60) + second
	switch {
	case minutes == 0:
		f.offset = some(0)
	case parts[0] == "+":
		f.offset = some(-minutes)
	default:
		f.offset = some(minutes)
	}
	return true
}

// monthName is Pele's monthName: a reader of a month by its name in names, which fails for
// a name not among them.
func monthName(names []string) reader {
	return func(f *fields, input string) bool {
		index := -1
		for i, n := range names {
			if n == input {
				index = i
				break
			}
		}
		index++
		if index < 1 {
			return false
		}
		f.month = some(float64(index))
		return true
	}
}

var shortMonths = func() []string {
	out := make([]string, len(months))
	for i, m := range months {
		out[i] = m[:3]
	}
	return out
}()

var readers = map[string]step{
	"A":    {match: word, read: func(f *fields, input string) bool { pm := input == "PM"; f.afternoon = &pm; return true }},
	"a":    {match: word, read: func(f *fields, input string) bool { pm := input == "pm"; f.afternoon = &pm; return true }},
	"Q":    {match: digits(1), read: func(f *fields, input string) bool { f.month = some(float64((number(input)-1)*3) + 1); return true }},
	"S":    {match: digits(1), read: func(f *fields, input string) bool { f.milliseconds = some(number(input) * 100); return true }},
	"SS":   {match: digits(2), read: func(f *fields, input string) bool { f.milliseconds = some(number(input) * 10); return true }},
	"SSS":  {match: digits(3), read: func(f *fields, input string) bool { f.milliseconds = some(number(input)); return true }},
	"s":    {match: oneOrTwo, read: field(func(f *fields, v float64) { f.seconds = some(v) })},
	"ss":   {match: oneOrTwo, read: field(func(f *fields, v float64) { f.seconds = some(v) })},
	"m":    {match: oneOrTwo, read: field(func(f *fields, v float64) { f.minutes = some(v) })},
	"mm":   {match: oneOrTwo, read: field(func(f *fields, v float64) { f.minutes = some(v) })},
	"H":    {match: oneOrTwo, read: field(func(f *fields, v float64) { f.hours = some(v) })},
	"h":    {match: oneOrTwo, read: field(func(f *fields, v float64) { f.hours = some(v) })},
	"HH":   {match: oneOrTwo, read: field(func(f *fields, v float64) { f.hours = some(v) })},
	"hh":   {match: oneOrTwo, read: field(func(f *fields, v float64) { f.hours = some(v) })},
	"D":    {match: oneOrTwo, read: field(func(f *fields, v float64) { f.day = some(v) })},
	"DD":   {match: digits(2), read: field(func(f *fields, v float64) { f.day = some(v) })},
	"Do":   {match: word, read: readOrdinal},
	"w":    {match: oneOrTwo, read: field(func(f *fields, v float64) { f.week = some(v) })},
	"ww":   {match: digits(2), read: field(func(f *fields, v float64) { f.week = some(v) })},
	"M":    {match: oneOrTwo, read: field(func(f *fields, v float64) { f.month = some(v) })},
	"MM":   {match: digits(2), read: field(func(f *fields, v float64) { f.month = some(v) })},
	"MMM":  {match: word, read: monthName(shortMonths)},
	"MMMM": {match: word, read: monthName(months[:])},
	"Y":    {match: signed, read: field(func(f *fields, v float64) { f.year = some(v) })},
	"YY": {match: digits(2), read: func(f *fields, input string) bool {
		v := number(input)
		if v > 68 {
			f.year = some(v + 1900)
		} else {
			f.year = some(v + 2000)
		}
		return true
	}},
	"YYYY": {match: digits(4), read: field(func(f *fields, v float64) { f.year = some(v) })},
	"Z":    {match: offsetMatch, read: readOffset},
	"ZZ":   {match: offsetMatch, read: readOffset},
}

// readOrdinal reads Do, as the reader of Pele's READERS for Do does. dayjs keeps the digits
// as a string here, so even "0" counts as a day having been given.
func readOrdinal(f *fields, input string) bool {
	i := strings.IndexFunc(input, func(r rune) bool { return r >= '0' && r <= '9' })
	if i < 0 {
		return false
	}
	j := i
	for j < len(input) && chars.IsDigit(int(input[j])) {
		j++
	}
	f.day = some(number(input[i:j]))
	f.dayGiven = true
	for k := 1; k <= 31; k++ {
		if ordinal(float64(k)) == input {
			f.day = some(float64(k))
		}
	}
	return true
}

// tokenAt is the length of a match of Pele's RE_TOKENS,
// /(\[[^[]*\])|([-_:/.,()\s]+)|(A|a|Q|YYYY|YY?|ww?|MM?M?M?|Do|DD?|hh?|HH?|mm?|ss?|S{1,3}|z|ZZ?)/,
// at byte i of s, or 0.
func tokenAt(s string, i int) int {
	if s[i] == '[' {
		end := strings.IndexByte(s[i+1:], '[')
		if end < 0 {
			end = len(s)
		} else {
			end += i + 1
		}
		if k := strings.LastIndexByte(s[i+1:end], ']'); k >= 0 {
			return k + 2
		}
	}
	j := i
	for j < len(s) {
		r, size := utf8.DecodeRuneInString(s[j:])
		if !strings.ContainsRune("-_:/.,()", r) && !chars.IsSpaceRune(r) {
			break
		}
		j += size
	}
	if j > i {
		return j - i
	}
	switch c := s[i]; c {
	case 'A', 'a', 'Q', 'z':
		return 1
	case 'Y':
		if run(s, i, 'Y', 4) == 4 {
			return 4
		}
		return run(s, i, 'Y', 2)
	case 'w', 'h', 'H', 'm', 's', 'Z':
		return run(s, i, c, 2)
	case 'M':
		return run(s, i, 'M', 4)
	case 'D':
		if i+1 < len(s) && s[i+1] == 'o' {
			return 2
		}
		return run(s, i, 'D', 2)
	case 'S':
		return run(s, i, 'S', 3)
	}
	return 0
}

// compile is Pele's compile: the steps of a format, or false when dayjs cannot parse with
// it.
func compile(template string) ([]step, bool) {
	// dayjs expands localized tokens through a locale table the English locale does not have,
	// and fails.
	stripped := replaceAll(template, func(s string, i int) int { return literalAt(s, i) }, func(string) string { return "" })
	if strings.ContainsAny(stripped, "Ll") {
		return nil, false
	}
	var steps []step
	for i := 0; i < len(template); {
		n := tokenAt(template, i)
		if n == 0 {
			i++
			continue
		}
		tok := template[i : i+n]
		i += n
		if s, ok := readers[tok]; ok {
			steps = append(steps, s)
			continue
		}
		tok = strings.TrimPrefix(tok, "[")
		tok = strings.TrimSuffix(tok, "]")
		steps = append(steps, step{literal: tok})
	}
	if steps == nil {
		return nil, false
	}
	return steps, true
}

// literalAt is the length of /\[[^\]]+]/ at i, or 0: Pele's RE_LITERAL.
func literalAt(s string, i int) int { return bracketAt(s, i) }

// indexUnits is the first index of sub in s, by code units, or -1, as String.prototype.replace
// with a string finds what Pele's loose drops.
func indexUnits(s, sub []uint16) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		match := true
		for k := range sub {
			if s[i+k] != sub[k] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// loose is Pele's loose: the date input reads as in template, unchecked.
func (z zone) loose(input, template string, now float64) (float64, bool) {
	if template == "x" || template == "X" {
		scale := 1.0
		if template == "X" {
			scale = 1000
		}
		return jsdate.TimeClip(scale * number(input)), true
	}
	steps, ok := compile(template)
	if !ok {
		return 0, false
	}
	var f fields
	rest := utf16.Encode([]rune(input))
	start := 0
	for _, st := range steps {
		if st.match == nil {
			start += chars.Len16(st.literal)
			continue
		}
		from := min(start, len(rest))
		m, ok := st.match(rest[from:])
		if !ok || !st.read(&f, string(utf16.Decode(m))) {
			return 0, false
		}
		// dayjs drops the first occurrence of the matched text, wherever it is, and does not
		// move start.
		if k := indexUnits(rest, m); k >= 0 {
			m = append([]uint16(nil), m...)
			rest = append(rest[:k:k], rest[k+len(m):]...)
		}
	}
	if f.afternoon != nil && f.hours.set {
		if *f.afternoon {
			if f.hours.v < 12 {
				f.hours.v += 12
			}
		} else if f.hours.v == 12 {
			f.hours.v = 0
		}
	}
	var day float64
	switch {
	case f.dayGiven || f.day.truthy():
		day = f.day.v
	case !f.year.truthy() && !f.month.truthy():
		day = z.Date(now)
	default:
		day = 1
	}
	year := f.year.or(z.FullYear(now))
	month := 0.0
	if !(f.year.truthy() && !f.month.truthy()) {
		if f.month.set && f.month.v > 0 {
			month = f.month.v - 1
		} else {
			month = z.Month(now)
		}
	}
	hours := f.hours.or(0)
	minutes := f.minutes.or(0)
	seconds := f.seconds.or(0)
	ms := f.milliseconds.or(0)
	if f.offset.set {
		return jsdate.TimeClip(jsdate.UTC(year, month, day, hours, minutes, seconds, ms+float64(f.offset.v*60000))), true
	}
	// Applying a week number needs a dayjs plugin Mermaid does not load.
	if f.week.truthy() {
		return 0, false
	}
	return z.New(year, month, day, hours, minutes, seconds, ms), true
}

// parseStrict parses the way dayjs(input, format, true) does, as Pele's parseStrict does. now
// supplies the fields the format leaves out.
func (z zone) parseStrict(input, template string, now float64) (float64, bool) {
	date, ok := z.loose(input, template, now)
	if !ok || !jsdate.Valid(date) {
		return 0, false
	}
	if s, ok := z.format(date, template); !ok || s != input {
		return 0, false
	}
	return date, true
}
