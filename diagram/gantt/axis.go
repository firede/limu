package gantt

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/firede/limu/internal/jsdate"
	"github.com/firede/limu/internal/jsmath"
)

// Time axis support, as Pele's axis.ts has it, following d3-time and d3-time-format (English
// locale, local time), which Mermaid uses: the axisFormat specifiers, tick intervals with
// d3's every rule, and its choice of automatic ticks.

const (
	secondMs = 1000
	minuteMs = 60 * secondMs
	hourMs   = 60 * minuteMs
	monthMs  = 30 * dayMs
	yearMs   = 365 * dayMs
)

// A tickCalendar is axis.ts's Calendar: an interval of time, as a floor and an offset of
// dates, which give the date they leave.
type tickCalendar struct {
	floor  func(d float64) float64
	offset func(d, step float64) float64
	// field is the value d3 tests against a step: a tick falls where it is a multiple of the
	// step. nil where there is none.
	field func(d float64) float64
	// cycle is how many values field cycles through, which bounds how far apart filtered
	// ticks can be.
	cycle float64
	ms    float64
}

func (z zone) startOfDay(d float64) float64 { return z.SetHours(d, 0, 0, 0, 0) }

func (z zone) weekCalendar(weekStart float64) tickCalendar {
	return tickCalendar{
		floor: func(d float64) float64 {
			d = z.SetDate(d, z.Date(d)-math.Mod(z.Day(d)+7-weekStart, 7))
			return z.startOfDay(d)
		},
		offset: func(d, step float64) float64 { return z.SetDate(d, z.Date(d)+float64(step*7)) },
		cycle:  math.Inf(1),
		ms:     weekMs,
	}
}

// calendarOf is axis.ts's CALENDARS, for every unit but the week.
func (z zone) calendarOf(unit string) tickCalendar {
	switch unit {
	case "millisecond":
		return tickCalendar{
			floor:  func(d float64) float64 { return d },
			offset: func(d, step float64) float64 { return jsdate.SetTime(d + step) },
			cycle:  math.Inf(1), ms: 1,
		}
	case "second":
		return tickCalendar{
			floor:  func(d float64) float64 { return jsdate.SetTime(d - z.Milliseconds(d)) },
			offset: func(d, step float64) float64 { return jsdate.SetTime(d + float64(step*secondMs)) },
			field:  jsdate.UTCSeconds, cycle: 60, ms: secondMs,
		}
	case "minute":
		return tickCalendar{
			floor: func(d float64) float64 {
				return jsdate.SetTime(d - z.Milliseconds(d) - float64(z.Seconds(d)*secondMs))
			},
			offset: func(d, step float64) float64 { return jsdate.SetTime(d + float64(step*minuteMs)) },
			field:  z.Minutes, cycle: 60, ms: minuteMs,
		}
	case "hour":
		return tickCalendar{
			floor: func(d float64) float64 {
				return jsdate.SetTime(d - z.Milliseconds(d) - float64(z.Seconds(d)*secondMs) - float64(z.Minutes(d)*minuteMs))
			},
			offset: func(d, step float64) float64 { return jsdate.SetTime(d + float64(step*hourMs)) },
			field:  z.Hours, cycle: 24, ms: hourMs,
		}
	case "day":
		return tickCalendar{
			floor:  z.startOfDay,
			offset: func(d, step float64) float64 { return z.SetDate(d, z.Date(d)+step) },
			field:  func(d float64) float64 { return z.Date(d) - 1 },
			cycle:  31, ms: dayMs,
		}
	case "month":
		return tickCalendar{
			floor:  func(d float64) float64 { return z.startOfDay(z.SetDate(d, 1)) },
			offset: func(d, step float64) float64 { return z.SetMonth(d, z.Month(d)+step) },
			field:  z.Month, cycle: 12, ms: monthMs,
		}
	}
	return tickCalendar{
		floor:  func(d float64) float64 { return z.startOfDay(z.SetMonthDate(d, 0, 1)) },
		offset: func(d, step float64) float64 { return z.SetFullYear(d, z.FullYear(d)+step) },
		cycle:  math.Inf(1), ms: yearMs,
	}
}

// span is axis.ts's span: whole units between two floored dates, discounting a change of UTC
// offset between them.
func (z zone) span(start, end, unit float64) float64 {
	return math.Floor((end - start - float64((z.TimezoneOffset(end)-z.TimezoneOffset(start))*minuteMs)) / unit)
}

func mod(a, b float64) float64 { return math.Mod(math.Mod(a, b)+b, b) }

// every is the interval d3's unit.every(step) gives, axis.ts's every.
func (z zone) every(unit string, step, weekStart float64) tickCalendar {
	var base tickCalendar
	if unit == "week" {
		base = z.weekCalendar(weekStart)
	} else {
		base = z.calendarOf(unit)
	}
	if step <= 1 {
		return base
	}
	c := base
	switch unit {
	case "millisecond":
		c.floor = func(d float64) float64 { return jsdate.SetTime(float64(math.Floor(d/step) * step)) }
		c.offset = func(d, n float64) float64 { return jsdate.SetTime(d + float64(n*step)) }
		return c
	case "year":
		c.floor = func(d float64) float64 {
			d = z.SetFullYear(d, float64(math.Floor(z.FullYear(d)/step)*step))
			return base.floor(d)
		}
		c.offset = func(d, n float64) float64 { return z.SetFullYear(d, z.FullYear(d)+float64(n*step)) }
		return c
	case "week":
		epoch := base.floor(0)
		c.floor = func(d float64) float64 {
			d = base.floor(d)
			return base.offset(d, -mod(z.span(epoch, d, weekMs), step))
		}
		c.offset = func(d, n float64) float64 { return base.offset(d, float64(n*step)) }
		return c
	}
	test := func(d float64) bool { return math.Mod(base.field(d), step) == 0 }
	finite := func(d float64) bool { return !math.IsNaN(d) && !math.IsInf(d, 0) }
	c.floor = func(d float64) float64 {
		// Date overflow makes field(d) NaN, which can never match the interval.
		for d = base.floor(d); finite(d) && !test(d); d = base.floor(d) {
			d = jsdate.SetTime(d - 1)
		}
		return d
	}
	c.offset = func(d, n float64) float64 {
		for i := 0.0; i < n && finite(d); i++ {
			for {
				d = base.offset(d, 1)
				if !(finite(d) && !test(d)) {
					break
				}
			}
		}
		return d
	}
	return c
}

// ticks is axis.ts's range: ticks from start to stop inclusive, or false when there would be
// more than limit.
func ticks(c tickCalendar, start, stop float64, limit int) ([]float64, bool) {
	d := jsdate.TimeClip(start - 1)
	d = c.floor(d)
	d = c.offset(d, 1)
	d = c.floor(d)
	out := []float64{}
	previous := math.Inf(-1)
	for d <= stop && d > previous {
		if len(out) >= limit {
			return nil, false
		}
		previous = d
		out = append(out, previous)
		d = c.offset(d, 1)
		d = c.floor(d)
	}
	return out, true
}

// tickStep is axis.ts's tickStep.
func tickStep(start, stop, count float64) float64 {
	step := (stop - start) / max(0, count)
	power := math.Floor(jsmath.Log10(step))
	err := step / jsmath.Pow10(power)
	factor := 1.0
	switch {
	case err >= math.Sqrt(50):
		factor = 10
	case err >= math.Sqrt(10):
		factor = 5
	case err >= math.Sqrt2:
		factor = 2
	}
	return jsmath.Pow10(power) * factor
}

var autoSteps = [...]struct {
	unit string
	step float64
	ms   float64
}{
	{"second", 1, secondMs},
	{"second", 5, 5 * secondMs},
	{"second", 15, 15 * secondMs},
	{"second", 30, 30 * secondMs},
	{"minute", 1, minuteMs},
	{"minute", 5, 5 * minuteMs},
	{"minute", 15, 15 * minuteMs},
	{"minute", 30, 30 * minuteMs},
	{"hour", 1, hourMs},
	{"hour", 3, 3 * hourMs},
	{"hour", 6, 6 * hourMs},
	{"hour", 12, 12 * hourMs},
	{"day", 1, dayMs},
	{"day", 2, 2 * dayMs},
	{"week", 1, weekMs},
	{"month", 1, monthMs},
	{"month", 3, 3 * monthMs},
	{"year", 1, yearMs},
}

// autoTicks is axis.ts's autoTicks: about count ticks at round times, chosen the way d3's time
// scale chooses them.
func (z zone) autoTicks(start, stop, count float64) []float64 {
	target := (stop - start) / count
	i := 0
	for i < len(autoSteps) && autoSteps[i].ms <= target {
		i++
	}
	var c tickCalendar
	switch {
	case i == len(autoSteps):
		c = z.every("year", max(1, tickStep(start/yearMs, stop/yearMs, count)), 0)
	case i == 0:
		c = z.every("millisecond", max(tickStep(start, stop, count), 1), 0)
	default:
		k := i
		if target/autoSteps[i-1].ms < autoSteps[i].ms/target {
			k = i - 1
		}
		c = z.every(autoSteps[k].unit, autoSteps[k].step, 0)
	}
	out, ok := ticks(c, start, stop, int(4*count+8))
	if !ok {
		return []float64{}
	}
	return out
}

var weekStarts = [...]string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

// readInterval reads a tick interval, /^([1-9]\d*)(millisecond|second|minute|hour|day|week|month)$/.
func readInterval(s string) (step float64, unit string, ok bool) {
	i := 0
	if i < len(s) && s[i] >= '1' && s[i] <= '9' {
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
	} else {
		return 0, "", false
	}
	for _, u := range [...]string{"millisecond", "second", "minute", "hour", "day", "week", "month"} {
		if s[i:] == u {
			return parseIntDigits(s[:i]), u, true
		}
	}
	return 0, "", false
}

// parseIntDigits is parseInt(s, 10) of a string of digits.
func parseIntDigits(s string) float64 {
	return number(s)
}

// intervalTicks is axis.ts's intervalTicks: ticks for a tickInterval such as 1week, or false
// when it is not one or asks for more than limit.
func (z zone) intervalTicks(interval, weekday string, start, stop float64, limit int) ([]float64, bool) {
	step, unit, ok := readInterval(interval)
	if !ok {
		return nil, false
	}
	weekStart := 0.0
	for i, w := range weekStarts {
		if w == weekday {
			weekStart = float64(i)
		}
	}
	c := z.every(unit, step, weekStart)
	if (stop-start)/float64(min(step, c.cycle)*c.ms) > float64(limit) {
		return nil, false
	}
	return ticks(c, start, stop, limit)
}

// padTime is axis.ts's pad.
func padTime(value float64, fill string, width int) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	digits := numStr(value)
	if len(digits) < width {
		digits = strings.Repeat(fill, width-len(digits)) + digits
	}
	return sign + digits
}

func (z zone) floored(date float64, c tickCalendar) float64 { return c.floor(date) }

// weekOfYear is axis.ts's weekOfYear: weeks since the last weekStart day before the year
// began, which is d3's %U and %W.
func (z zone) weekOfYear(date, weekStart float64) float64 {
	week := z.weekCalendar(weekStart)
	yearStart := z.floored(date, z.calendarOf("year"))
	return z.span(z.floored(jsdate.TimeClip(yearStart-1), week), z.floored(date, week), weekMs)
}

// isoThursday is axis.ts's isoThursday: the Thursday that decides which ISO week and ISO year
// a date belongs to.
func (z zone) isoThursday(date float64) float64 {
	thursday := z.weekCalendar(4)
	day := z.Day(date)
	late := day >= 4 || day == 0
	from := date
	if !late {
		from = jsdate.TimeClip(date - 1)
	}
	d := z.floored(from, thursday)
	if !late {
		d = thursday.offset(d, 1)
		d = thursday.floor(d)
	}
	return d
}

// timeFormat is axis.ts's timeFormat: a date written with a d3 time format specifier such as
// %Y-%m-%d.
func (z zone) timeFormat(specifier string, date float64) string {
	var b strings.Builder
	from := 0
	n := len(specifier)
	for i := 0; i < n; i++ {
		if specifier[i] != '%' {
			continue
		}
		b.WriteString(specifier[from:i])
		i++
		c, size := charAt(specifier, i)
		fill, isPad := "", false
		switch c {
		case "-":
			fill, isPad = "", true
		case "_":
			fill, isPad = " ", true
		case "0":
			fill, isPad = "0", true
		}
		if isPad {
			i += size
			c, size = charAt(specifier, i)
		} else if c == "e" {
			fill = " "
		} else {
			fill = "0"
		}
		b.WriteString(z.directive(c, date, fill))
		if size > 0 {
			i += size - 1
		}
		from = i + 1
		if from > n {
			from = n
		}
	}
	b.WriteString(specifier[min(from, n):])
	return b.String()
}

// charAt is the character at byte i of s, and its length in bytes; "" and 0 past the end.
func charAt(s string, i int) (string, int) {
	if i >= len(s) {
		return "", 0
	}
	_, size := utf8.DecodeRuneInString(s[i:])
	return s[i : i+size], size
}

func (z zone) directive(c string, d float64, p string) string {
	switch c {
	case "a":
		return weekdays[int(z.Day(d))][:3]
	case "A":
		return weekdays[int(z.Day(d))]
	case "b":
		return months[int(z.Month(d))][:3]
	case "B":
		return months[int(z.Month(d))]
	case "c":
		return z.timeFormat("%x, %X", d)
	case "d", "e":
		return padTime(z.Date(d), p, 2)
	case "f":
		return padTime(z.Milliseconds(d), p, 3) + "000"
	case "g":
		return padTime(math.Mod(z.FullYear(z.isoThursday(d)), 100), p, 2)
	case "G":
		return padTime(math.Mod(z.FullYear(z.isoThursday(d)), 10000), p, 4)
	case "H":
		return padTime(z.Hours(d), p, 2)
	case "I":
		return padTime(orZero(math.Mod(z.Hours(d), 12), 12), p, 2)
	case "j":
		return padTime(1+z.span(z.floored(d, z.calendarOf("year")), z.floored(d, z.calendarOf("day")), dayMs), p, 3)
	case "L":
		return padTime(z.Milliseconds(d), p, 3)
	case "m":
		return padTime(z.Month(d)+1, p, 2)
	case "M":
		return padTime(z.Minutes(d), p, 2)
	case "p":
		if z.Hours(d) >= 12 {
			return "PM"
		}
		return "AM"
	case "q":
		return numStr(1 + math.Floor(z.Month(d)/3))
	case "Q":
		return numStr(d)
	case "s":
		return numStr(math.Floor(d / 1000))
	case "S":
		return padTime(z.Seconds(d), p, 2)
	case "u":
		return numStr(orZero(z.Day(d), 7))
	case "U":
		return padTime(z.weekOfYear(d, 0), p, 2)
	case "V":
		thursday := z.isoThursday(d)
		yearStart := z.floored(thursday, z.calendarOf("year"))
		first := z.floored(yearStart, z.weekCalendar(4))
		extra := 0.0
		if z.Day(yearStart) == 4 {
			extra = 1
		}
		return padTime(z.span(first, thursday, weekMs)+extra, p, 2)
	case "w":
		return numStr(z.Day(d))
	case "W":
		return padTime(z.weekOfYear(d, 1), p, 2)
	case "x":
		return z.timeFormat("%-m/%-d/%Y", d)
	case "X":
		return z.timeFormat("%-I:%M:%S %p", d)
	case "y":
		return padTime(math.Mod(z.FullYear(d), 100), p, 2)
	case "Y":
		return padTime(math.Mod(z.FullYear(d), 10000), p, 4)
	case "Z":
		off := z.TimezoneOffset(d)
		abs := math.Abs(off)
		sign := "+"
		if off > 0 {
			sign = "-"
		}
		return sign + padTime(math.Floor(abs/60), "0", 2) + padTime(math.Mod(abs, 60), "0", 2)
	case "%":
		return "%"
	}
	return c
}
