package gantt

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/fixture"
	"github.com/firede/limu/internal/jsdate"
)

// The fixtures are what Pele makes of the same cases; ./scripts/oracle.sh writes them. A
// chart's model is written as the oracle writes Pele's, field by field, and so are the tasks
// its getTasks schedules in each zone, so that any field that differs shows. Values JSON has
// no words for are written as the oracle's enc writes them.

// enc is a value as the oracle's enc writes it.
func enc(v any) any {
	switch x := v.(type) {
	case nil:
		return map[string]any{"$": "undefined"}
	case float64:
		switch {
		case math.IsNaN(x):
			return map[string]any{"$n": "NaN"}
		case math.IsInf(x, 1):
			return map[string]any{"$n": "Infinity"}
		case math.IsInf(x, -1):
			return map[string]any{"$n": "-Infinity"}
		case x == 0 && math.Signbit(x):
			return map[string]any{"$n": "-0"}
		}
	case []float64:
		out := make([]any, len(x))
		for i, f := range x {
			out[i] = enc(f)
		}
		return out
	}
	return v
}

// optional is a string Pele may leave undefined, as enc writes it.
func optional(s string, set bool) any {
	if !set {
		return enc(nil)
	}
	return s
}

func strs(s []string) []any {
	out := make([]any, len(s))
	for i, x := range s {
		out[i] = x
	}
	return out
}

func dumpModel(db *DB) map[string]any {
	links := []any{}
	for _, id := range db.LinkIDs {
		links = append(links, []any{id, db.Links[id]})
	}
	tasks := []any{}
	for _, t := range db.Tasks {
		tasks = append(tasks, map[string]any{
			"id": t.ID, "task": t.Task, "section": t.Section,
			"flags": []any{t.Active, t.Done, t.Crit, t.Milestone, t.Vert},
			"order": t.Order, "classes": strs(t.Classes), "prevTaskId": optional(t.PrevTaskID, t.HasPrev),
			"raw": []any{t.Raw.Data, optional(t.Raw.Start, t.Raw.HasStart), t.Raw.End},
		})
	}
	return map[string]any{
		"title": optional(db.Title, db.Title != ""), "accTitle": optional(db.AccTitle, db.HasAccTitle),
		"accDescr": optional(db.AccDescr, db.HasAccDescr), "dateFormat": db.DateFormat, "axisFormat": db.AxisFormat,
		"tickInterval": optional(db.TickInterval, db.TickInterval != ""), "todayMarker": db.TodayMarker,
		"includes": strs(db.Includes), "excludes": strs(db.Excludes), "inclusiveEndDates": db.InclusiveEndDates,
		"topAxis": db.TopAxis, "displayMode": db.DisplayMode, "weekday": optional(db.Weekday, db.Weekday != ""),
		"weekend": db.Weekend, "sections": strs(db.Sections), "links": links, "tasks": tasks,
	}
}

func dumpMoment(m moment) any {
	if !m.defined {
		return enc(nil)
	}
	return enc(m.ms)
}

func dumpSchedule(db *DB, err error) map[string]any {
	if err != nil {
		return map[string]any{"error": dumpError(err)}
	}
	tasks := []any{}
	for _, t := range db.Tasks {
		renderEnd := dumpMoment(t.renderEnd)
		if !t.renderEnd.defined {
			renderEnd = nil
		}
		tasks = append(tasks, []any{dumpMoment(t.start), dumpMoment(t.end), renderEnd, t.Processed, t.ManualEndTime})
	}
	return map[string]any{"tasks": tasks}
}

func dumpError(err error) map[string]any {
	e, ok := err.(*diagram.Error)
	if !ok {
		return map[string]any{"crash": err.Error()}
	}
	out := map[string]any{"code": string(e.Code), "message": e.Message, "type": nil}
	if e.Type != "" {
		out["type"] = e.Type
	}
	out["line"], out["column"], out["snippet"] = e.Line, e.Column, e.Snippet
	return out
}

func roundTrip(v any) any {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out any
	json.Unmarshal(data, &out)
	return out
}

func compare(t *testing.T, what, src string, got, want any) bool {
	t.Helper()
	g, w := roundTrip(got), roundTrip(want)
	if reflect.DeepEqual(g, w) {
		return true
	}
	gj, _ := json.Marshal(g)
	wj, _ := json.Marshal(w)
	t.Errorf("%s of %q\n got %s\nPele %s", what, src, gj, wj)
	return false
}

type peleResult struct {
	Model     any              `json:"model"`
	Schedules []map[string]any `json:"schedules"`
	Error     map[string]any   `json:"error"`
}

type peleCase struct {
	Src    string     `json:"src"`
	Now    float64    `json:"now"`
	Type   *string    `json:"type"`
	Direct peleResult `json:"direct"`
	Same   bool       `json:"same"`
	peleResult
}

func (c *peleCase) UnmarshalJSON(data []byte) error {
	type plain peleCase
	var raw struct {
		plain
		Model     any              `json:"model"`
		Schedules []map[string]any `json:"schedules"`
		Error     map[string]any   `json:"error"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = peleCase(raw.plain)
	c.peleResult = peleResult{raw.Model, raw.Schedules, raw.Error}
	if c.Same {
		c.peleResult = c.Direct
	}
	return nil
}

func loadZones(t *testing.T, names []string) []*time.Location {
	t.Helper()
	var out []*time.Location
	for _, name := range names {
		loc, err := time.LoadLocation(name)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, loc)
	}
	return out
}

// check compares a parse, and the schedules of what it parsed in each zone, with Pele's. The
// schedules are left out when schedule is false: a model that matches one already scheduled
// schedules as it did.
func check(t *testing.T, what string, c *peleCase, zones []*time.Location, db *DB, err error, want peleResult, schedule bool) bool {
	t.Helper()
	switch {
	case err != nil && want.Error == nil:
		t.Errorf("%s(%q): %v, Pele had none", what, c.Src, err)
		return false
	case err != nil:
		return compare(t, what+" error", c.Src, dumpError(err), want.Error)
	case want.Error != nil:
		t.Errorf("%s(%q): no error, Pele %v", what, c.Src, want.Error)
		return false
	}
	if !compare(t, what+" model", c.Src, dumpModel(db), want.Model) {
		return false
	}
	if !schedule {
		return true
	}
	now := time.UnixMilli(int64(c.Now))
	for i, loc := range zones {
		err := db.Schedule(now, loc)
		if !compare(t, what+" schedule in "+loc.String(), c.Src, dumpSchedule(db, err), want.Schedules[i]) {
			return false
		}
	}
	return true
}

func TestParseMatchesPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/gantt.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Zones []string   `json:"zones"`
		Cases []peleCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Cases) < 4000 {
		t.Fatalf("only %d cases", len(p.Cases))
	}
	zones := loadZones(t, p.Zones)
	failed := 0
	for i := range p.Cases {
		c := &p.Cases[i]
		db := newDB()
		err := parseInto(c.Src, db)
		if err != nil {
			db = nil
		}
		ok := check(t, "parse", c, zones, db, err, c.Direct, true)
		if ok {
			pre, err := diagram.Prepare(c.Src, diagram.Options{})
			switch {
			case err != nil:
				if e, isPele := err.(*diagram.Error); isPele && e.Code == diagram.UnsupportedDiagram && c.Error == nil {
					break
				}
				if c.Error == nil {
					t.Errorf("Prepare(%q): %v, Pele had none", c.Src, err)
					ok = false
				} else {
					ok = compare(t, "error", c.Src, dumpError(err), c.Error)
				}
			case c.Type == nil || string(pre.Type) != *c.Type:
				t.Errorf("type of %q: %q, Pele %v", c.Src, pre.Type, c.Type)
				ok = false
			case pre.Type == diagram.Gantt:
				db, err := ParsePrepared(pre)
				ok = check(t, "ParsePrepared", c, zones, db, err, c.peleResult, !c.Same)
			}
		}
		if !ok {
			if failed++; failed >= 10 {
				t.Fatal("too many failures")
			}
		}
	}
}

// The date code on its own, in each zone.

type peleDates struct {
	Zone        string               `json:"zone"`
	Now         float64              `json:"now"`
	Formats     [][3]json.RawMessage `json:"formats"`
	Parses      [][3]json.RawMessage `json:"parses"`
	TimeFormats [][3]json.RawMessage `json:"timeFormats"`
	Ticks       [][6]json.RawMessage `json:"ticks"`
}

func TestDatesMatchPele(t *testing.T) {
	data, err := fixture.Read("testdata/pele/dates.json")
	if err != nil {
		t.Fatal(err)
	}
	var p struct {
		Zones []peleDates `json:"zones"`
	}
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	str := func(raw json.RawMessage) string {
		var s string
		json.Unmarshal(raw, &s)
		return s
	}
	num := func(raw json.RawMessage) float64 {
		var f float64
		json.Unmarshal(raw, &f)
		return f
	}
	for _, zc := range p.Zones {
		loc, err := time.LoadLocation(zc.Zone)
		if err != nil {
			t.Fatal(err)
		}
		z := zone{jsdate.In(loc)}
		failed := 0
		fail := func(format string, args ...any) {
			if failed++; failed <= 20 {
				t.Errorf(zc.Zone+": "+format, args...)
			}
		}
		for _, c := range zc.Formats {
			date, template := num(c[0]), str(c[1])
			var got any = enc(nil)
			if s, ok := z.format(date, template); ok {
				got = s
			}
			var want any
			json.Unmarshal(c[2], &want)
			if !reflect.DeepEqual(roundTrip(got), want) {
				fail("format(%v, %q) = %v, Pele %v", date, template, got, want)
			}
		}
		for _, c := range zc.Parses {
			text, template := str(c[0]), str(c[1])
			var got any
			if d, ok := z.parseStrict(text, template, zc.Now); ok {
				got = d
			}
			var want any
			json.Unmarshal(c[2], &want)
			if !reflect.DeepEqual(roundTrip(got), want) {
				fail("parseStrict(%q, %q) = %v, Pele %v", text, template, got, want)
			}
		}
		for _, c := range zc.TimeFormats {
			date, spec := num(c[0]), str(c[1])
			if got, want := z.timeFormat(spec, date), str(c[2]); got != want {
				fail("timeFormat(%q, %v) = %q, Pele %q", spec, date, got, want)
			}
		}
		for _, c := range zc.Ticks {
			start, stop, count, interval, weekday := num(c[0]), num(c[1]), num(c[2]), str(c[3]), str(c[4])
			var got any
			if interval == "" {
				got = z.autoTicks(start, stop, count)
			} else if ts, ok := z.intervalTicks(interval, weekday, start, stop, 200); ok {
				got = ts
			} else {
				got = enc(nil)
			}
			var want any
			json.Unmarshal(c[5], &want)
			if !reflect.DeepEqual(roundTrip(got), want) {
				fail("ticks(%v, %v, %v, %q, %q) = %v, Pele %v", start, stop, count, interval, weekday, roundTrip(got), want)
			}
		}
		if failed > 0 {
			t.Errorf("%s: %d differ", zc.Zone, failed)
		}
	}
}
