// Package jsdate is JavaScript's Date as V8 works it out, for code ported from Pele that
// computes with dates in local time: the arithmetic of its constructor, getters and setters,
// Date.UTC, and the parser of new Date(string). Where JavaScript reads the machine's time zone,
// a Zone holds a *time.Location instead, so that a host decides which zone a diagram is in.
//
// A date is a time value as JavaScript keeps one: a float64 of milliseconds since the epoch,
// a whole number from -8.64e15 to 8.64e15, or NaN for an invalid date.
//
// Local time is converted to UTC as V8 has ICU do it: a local time that a change of offset
// skips is read with the offset before the change, and one that a change repeats is the
// earlier of the two instants it names. The offsets are the location's, which come from the
// same tz database that ICU's do.
package jsdate

import (
	"math"
	"time"
)

const (
	msPerSecond = 1000
	msPerMinute = 60000
	msPerHour   = 3600000
	msPerDay    = 86400000

	// MaxTime is the largest time value a date can hold, and -MaxTime the smallest.
	MaxTime = 8.64e15
	// A local time may be ten days past MaxTime before it is converted, as V8 allows.
	maxTimeBeforeUTC = MaxTime + 864000000

	// The range of years and months V8's MakeDay works with.
	minYear  = -1000000
	maxYear  = 1000000
	minMonth = -10000000
	maxMonth = 10000000
)

// A Zone is a time zone dates are read and written in.
type Zone struct {
	loc *time.Location
}

// In is the zone of a location; nil is time.Local.
func In(loc *time.Location) Zone {
	if loc == nil {
		loc = time.Local
	}
	return Zone{loc}
}

// Valid reports whether t is a date, not NaN: JavaScript's !isNaN(date.getTime()).
func Valid(t float64) bool { return !math.IsNaN(t) }

// toInteger is V8's DoubleToInteger, ToIntegerOrInfinity: NaN is 0, and the rest are cut
// toward zero, with no negative zero.
func toInteger(x float64) float64 {
	if math.IsNaN(x) || x == 0 {
		return 0
	}
	if math.IsInf(x, 0) {
		return x
	}
	return math.Trunc(x) + 0
}

// TimeClip is the spec's TimeClip: a time value in range cut to a whole number, and NaN
// outside it.
func TimeClip(t float64) float64 {
	if -MaxTime <= t && t <= MaxTime {
		return toInteger(t)
	}
	return math.NaN()
}

// MakeTime is the spec's MakeTime, as V8 computes it.
func MakeTime(h, m, s, ms float64) float64 {
	if isFinite(h) && isFinite(m) && isFinite(s) && isFinite(ms) {
		return float64(toInteger(h)*msPerHour) + float64(toInteger(m)*msPerMinute) + float64(toInteger(s)*msPerSecond) + toInteger(ms)
	}
	return math.NaN()
}

var daysBeforeMonth = [2][12]int64{
	{0, 31, 59, 90, 120, 151, 181, 212, 243, 273, 304, 334},
	{0, 31, 60, 91, 121, 152, 182, 213, 244, 274, 305, 335},
}

// MakeDay is the spec's MakeDay, as V8 computes it: a year or a month past the range V8
// works with is NaN.
func MakeDay(year, month, date float64) float64 {
	if !(minYear <= year && year <= maxYear && minMonth <= month && month <= maxMonth && isFinite(date)) {
		return math.NaN()
	}
	y := int64(year)
	m := int64(month)
	y += m / 12
	m %= 12
	if m < 0 {
		m += 12
		y--
	}
	const yearDelta = 399999
	const baseDay = 365*(1970+yearDelta) + (1970+yearDelta)/4 - (1970+yearDelta)/100 + (1970+yearDelta)/400
	day := 365*(y+yearDelta) + (y+yearDelta)/4 - (y+yearDelta)/100 + (y+yearDelta)/400 - baseDay
	leap := 0
	if y%4 == 0 && (y%100 != 0 || y%400 == 0) {
		leap = 1
	}
	day += daysBeforeMonth[leap][m]
	return float64(day-1) + toInteger(date)
}

// MakeDate is the spec's MakeDate, as V8 computes it.
func MakeDate(day, t float64) float64 {
	if !isFinite(day) || !isFinite(t) {
		return math.NaN()
	}
	if t == 0 && day != 0 {
		return float64(day * msPerDay)
	}
	return t + float64(day*msPerDay)
}

func isFinite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

// offset is the zone's offset from UTC, in milliseconds, at the instant t.
func (z Zone) offset(t int64) int64 {
	_, off := time.UnixMilli(t).In(z.loc).Zone()
	return int64(off) * msPerSecond
}

// toLocal is V8's DateCache::ToLocal.
func (z Zone) toLocal(t int64) int64 { return t + z.offset(t) }

// toUTC is V8's DateCache::ToUTC: the instant of a local time, as ICU's getOffsetFromLocal
// gives it with the former offset for a time that is skipped and for one that is repeated.
// It looks at the offsets a day either side, as a change of offset is never closer than that
// to another.
func (z Zone) toUTC(local int64) int64 {
	before := z.offset(local - msPerDay)
	after := z.offset(local + msPerDay)
	if before == after {
		return local - before
	}
	early, late := local-before, local-after
	earlyFits := z.offset(early) == before
	lateFits := z.offset(late) == after
	switch {
	case earlyFits && lateFits:
		return min(early, late)
	case earlyFits:
		return early
	case lateFits:
		return late
	}
	// Skipped: read with the offset before the change.
	return early
}

// setLocal is V8's SetLocalDateValue: a local time converted to a date.
func (z Zone) setLocal(local float64) float64 {
	if local >= -maxTimeBeforeUTC && local <= maxTimeBeforeUTC {
		return TimeClip(float64(z.toUTC(int64(local))))
	}
	return math.NaN()
}

// fields is a date broken down in local time.
type fields struct {
	year, month, date, weekday int64
	days, timeInDay            int64
}

func (z Zone) breakDown(t float64) fields {
	local := z.toLocal(int64(t))
	days := floorDiv(local, msPerDay)
	f := fields{days: days, timeInDay: local - days*msPerDay}
	f.year, f.month, f.date = civil(days)
	f.weekday = (days + 4) % 7
	if f.weekday < 0 {
		f.weekday += 7
	}
	return f
}

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && (a < 0) != (b < 0) {
		q--
	}
	return q
}

// civil is the year, the month from 0 and the day of the month of a number of days since
// 1970-01-01.
func civil(days int64) (year, month, day int64) {
	z := days + 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if m > 12 {
		m -= 12
	}
	if m <= 2 {
		y++
	}
	return y, m - 1, d
}

// The getters, in local time. Each is NaN for an invalid date.

// FullYear is getFullYear.
func (z Zone) FullYear(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	return float64(z.breakDown(t).year)
}

// Month is getMonth, from 0.
func (z Zone) Month(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	return float64(z.breakDown(t).month)
}

// Date is getDate, the day of the month.
func (z Zone) Date(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	return float64(z.breakDown(t).date)
}

// Day is getDay, the day of the week from Sunday, 0.
func (z Zone) Day(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	return float64(z.breakDown(t).weekday)
}

// Hours is getHours.
func (z Zone) Hours(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	return float64(z.breakDown(t).timeInDay / msPerHour)
}

// Minutes is getMinutes.
func (z Zone) Minutes(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	return float64(z.breakDown(t).timeInDay / msPerMinute % 60)
}

// Seconds is getSeconds.
func (z Zone) Seconds(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	return float64(z.breakDown(t).timeInDay / msPerSecond % 60)
}

// Milliseconds is getMilliseconds.
func (z Zone) Milliseconds(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	return float64(z.breakDown(t).timeInDay % msPerSecond)
}

// TimezoneOffset is getTimezoneOffset: minutes from local time to UTC, cut to a whole
// number as V8 cuts them, so that an offset of local mean time in seconds loses them.
func (z Zone) TimezoneOffset(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	ms := int64(t)
	return float64((ms - z.toLocal(ms)) / msPerMinute)
}

// UTCSeconds is getUTCSeconds.
func UTCSeconds(t float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	ms := int64(t)
	return float64((ms - floorDiv(ms, msPerDay)*msPerDay) / msPerSecond % 60)
}

// The setters, in local time: each gives the date that the setter leaves in its object.

// SetTime is setTime.
func SetTime(v float64) float64 { return TimeClip(v) }

// SetDate is setDate.
func (z Zone) SetDate(t, date float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	f := z.breakDown(t)
	return z.setLocal(MakeDate(MakeDay(float64(f.year), float64(f.month), date), float64(f.timeInDay)))
}

// SetMonth is setMonth with one argument.
func (z Zone) SetMonth(t, month float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	f := z.breakDown(t)
	return z.setLocal(MakeDate(MakeDay(float64(f.year), month, float64(f.date)), float64(f.timeInDay)))
}

// SetMonthDate is setMonth with two, the month and the day of the month.
func (z Zone) SetMonthDate(t, month, date float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	f := z.breakDown(t)
	return z.setLocal(MakeDate(MakeDay(float64(f.year), month, date), float64(f.timeInDay)))
}

// SetFullYear is setFullYear with one argument. An invalid date is taken as +0, as the spec
// has it.
func (z Zone) SetFullYear(t, year float64) float64 {
	m, d, tid := 0.0, 1.0, 0.0
	if !math.IsNaN(t) {
		f := z.breakDown(t)
		m, d, tid = float64(f.month), float64(f.date), float64(f.timeInDay)
	}
	return z.setLocal(MakeDate(MakeDay(year, m, d), tid))
}

// SetHours is setHours with all four arguments.
func (z Zone) SetHours(t, h, m, s, ms float64) float64 {
	if math.IsNaN(t) {
		return t
	}
	f := z.breakDown(t)
	return z.setLocal(MakeDate(float64(f.days), MakeTime(h, m, s, ms)))
}

// fullYear is the year the constructor and Date.UTC take: one from 0 to 99 is in the 1900s.
func fullYear(year float64) float64 {
	if !math.IsNaN(year) {
		if y := toInteger(year); 0 <= y && y <= 99 {
			return 1900 + y
		}
	}
	return year
}

// New is new Date(year, month, date, hours, minutes, seconds, ms) in local time. The
// arguments JavaScript leaves out are 1 for date and 0 for the rest.
func (z Zone) New(year, month, date, h, m, s, ms float64) float64 {
	return z.setLocal(MakeDate(MakeDay(fullYear(year), month, date), MakeTime(h, m, s, ms)))
}

// UTC is Date.UTC.
func UTC(year, month, date, h, m, s, ms float64) float64 {
	return TimeClip(MakeDate(MakeDay(fullYear(year), month, date), MakeTime(h, m, s, ms)))
}
