package gantt

import (
	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
)

const lineTokens = "'EOF', 'SPACE', 'NL', 'weekday_monday', 'weekday_tuesday', 'weekday_wednesday', 'weekday_thursday', " +
	"'weekday_friday', 'weekday_saturday', 'weekday_sunday', 'weekend_friday', 'weekend_saturday', 'dateFormat', " +
	"'inclusiveEndDates', 'topAxis', 'axisFormat', 'tickInterval', 'excludes', 'includes', 'todayMarker', 'title', " +
	"'acc_title', 'acc_descr', 'acc_descr_multiline_value', 'section', 'taskTxt', 'click'"

// startsLine is Pele's STARTS_LINE: the tokens that can start a line, which are also the only
// ones that can follow a complete statement.
var startsLine = func() [tEOF + 1]bool {
	var s [tEOF + 1]bool
	for t := tWeekdayMonday; t <= tWeekendSaturday; t++ {
		s[t] = true
	}
	for _, t := range []token{tEOF, tNL, tDateFormat, tInclusiveEndDates, tTopAxis, tAxisFormat, tTickInterval, tExcludes,
		tIncludes, tTodayMarker, tTitle, tAccTitle, tAccDescr, tAccDescrMultilineValue, tSection, tTaskTxt, tClick} {
		s[t] = true
	}
	return s
}()

// parseError carries an error out of the parser's depths.
type parseError struct{ err error }

// parseInto is Pele's parseGantt: it reads the statements of src into db, accepting what
// Mermaid's gantt.jison accepts.
func parseInto(src string, db *DB) (err error) {
	toks := tokenize(src)
	i := 0
	defer func() {
		if r := recover(); r != nil {
			pe, ok := r.(parseError)
			if !ok {
				panic(r)
			}
			err = pe.err
		}
	}()

	fail := func(expected string) {
		t := toks[i].typ
		got := "1"
		if t != tEND {
			got = tokenNames[t]
		}
		panic(parseError{diagram.SyntaxError("gantt", src, toks[i].start, "Expecting "+expected+", got '"+got+"'", false)})
	}
	// Mermaid's parser runs a statement's action only after seeing that the next token can
	// follow it.
	end := func(also string) {
		if !startsLine[toks[i].typ] {
			fail(lineTokens + also)
		}
	}

	if toks[i].typ != tGantt {
		fail("'gantt'")
	}
	i++

	for {
		t := toks[i].typ
		text := toks[i].text
		if !startsLine[t] {
			fail(lineTokens)
		}
		i++
		switch t {
		case tEOF:
			if toks[i].typ == tEND {
				return nil
			}
			end("")
		case tNL:
		case tDateFormat:
			end("")
			db.setDateFormat(chars.DropUnits(text, 11))
		case tInclusiveEndDates:
			end("")
			db.InclusiveEndDates = true
		case tTopAxis:
			end("")
			db.TopAxis = true
		case tAxisFormat:
			end("")
			db.AxisFormat = chars.DropUnits(text, 11)
		case tTickInterval:
			end("")
			db.TickInterval = chars.DropUnits(text, 13)
		case tExcludes:
			end("")
			db.Excludes = mergeTokens(db.Excludes, chars.DropUnits(text, 9))
		case tIncludes:
			end("")
			db.Includes = mergeTokens(db.Includes, chars.DropUnits(text, 9))
		case tTodayMarker:
			end("")
			db.TodayMarker = chars.DropUnits(text, 12)
		case tWeekendFriday:
			end("")
			db.Weekend = "friday"
		case tWeekendSaturday:
			end("")
			db.Weekend = "saturday"
		case tTitle:
			end("")
			db.Title = chars.DropUnits(text, 6)
		case tAccTitle:
			if toks[i].typ != tAccTitleValue {
				fail("'acc_title_value'")
			}
			value := toks[i].text
			i++
			end("")
			db.setAccTitle(trim(value))
		case tAccDescr:
			if toks[i].typ != tAccDescrValue {
				fail("'acc_descr_value'")
			}
			value := toks[i].text
			i++
			end("")
			db.setAccDescription(trim(value))
		case tAccDescrMultilineValue:
			end("")
			db.setAccDescription(trim(text))
		case tSection:
			end("")
			db.addSection(chars.DropUnits(text, 8))
		case tTaskTxt:
			if toks[i].typ != tTaskData {
				fail("'taskData'")
			}
			data := toks[i].text
			i++
			end("")
			if err := db.addTask(text, data); err != nil {
				return err
			}
		case tClick:
			var href string
			hasName, hasHref := false, false
			if toks[i].typ == tCallbackname {
				hasName = true
				i++
				hasArgs := false
				if toks[i].typ == tCallbackargs {
					hasArgs = true
					i++
				}
				if toks[i].typ == tHref {
					href, hasHref = toks[i].text, true
					i++
					end("")
				} else if hasArgs {
					end(", 'href'")
				} else {
					end(", 'callbackargs', 'href'")
				}
			} else if toks[i].typ == tHref {
				href, hasHref = toks[i].text, true
				i++
				if toks[i].typ == tCallbackname {
					hasName = true
					i++
					if toks[i].typ == tCallbackargs {
						i++
						end("")
					} else {
						end(", 'callbackargs'")
					}
				} else {
					end(", 'callbackname'")
				}
			} else {
				fail("'callbackname', 'href'")
			}
			if hasName {
				db.setClass(text, "clickable")
			}
			if hasHref {
				db.setLink(text, href)
			}
		default:
			end("")
			db.Weekday = dayNames[t-tWeekdayMonday]
		}
	}
}
