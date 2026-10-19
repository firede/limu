package gantt

// token is a kind of token, Pele's T, in the order of Mermaid's gantt.jison.
type token uint8

const (
	tEND token = iota
	tERROR
	tOpenDirective
	tAccTitle
	tAccTitleValue
	tAccDescr
	tAccDescrValue
	tAccDescrMultilineValue
	tNL
	tHref
	tCallbackname
	tCallbackargs
	tClick
	tGantt
	tDateFormat
	tInclusiveEndDates
	tTopAxis
	tAxisFormat
	tTickInterval
	tIncludes
	tExcludes
	tTodayMarker
	tWeekdayMonday
	tWeekdayTuesday
	tWeekdayWednesday
	tWeekdayThursday
	tWeekdayFriday
	tWeekdaySaturday
	tWeekdaySunday
	tWeekendFriday
	tWeekendSaturday
	tDate
	tTitle
	tAccDescription
	tSection
	tTaskTxt
	tTaskData
	tCOLON
	tEOF
)

// tokenNames are the names Mermaid's parser gives the tokens in its errors.
var tokenNames = [...]string{
	"$end", "INVALID", "open_directive", "acc_title", "acc_title_value", "acc_descr", "acc_descr_value",
	"acc_descr_multiline_value", "NL", "href", "callbackname", "callbackargs", "click", "gantt", "dateFormat",
	"inclusiveEndDates", "topAxis", "axisFormat", "tickInterval", "includes", "excludes", "todayMarker",
	"weekday_monday", "weekday_tuesday", "weekday_wednesday", "weekday_thursday", "weekday_friday",
	"weekday_saturday", "weekday_sunday", "weekend_friday", "weekend_saturday", "date", "title", "accDescription",
	"section", "taskTxt", "taskData", ":", "EOF",
}
