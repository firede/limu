package gantt

import (
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/firede/limu/diagram"
	"github.com/firede/limu/internal/chars"
	"github.com/firede/limu/internal/jsdate"
	"github.com/firede/limu/internal/url"
)

// Scheduling walks tasks day by day and parses dates with a format the chart supplies. This
// bounds the total, in rough character operations, so a hostile chart fails instead of
// hanging, as Pele's WORK_LIMIT does.
const (
	workLimit  = 4e8
	dayStep    = 400
	datedStep  = 200
	formatStep = 400
)

// DB is a parsed gantt chart, Pele's GanttDb. Strings Pele may leave undefined are empty here
// when they are, but for the accessible title and description, which say whether they were
// given.
type DB struct {
	Title       string
	AccTitle    string
	HasAccTitle bool
	AccDescr    string
	HasAccDescr bool
	DateFormat  string
	AxisFormat  string
	// TickInterval is as written, as "1week", or "" for none.
	TickInterval      string
	TodayMarker       string
	Includes          []string
	Excludes          []string
	InclusiveEndDates bool
	TopAxis           bool
	DisplayMode       string
	// Weekday is the day week-based ticks start on, when the chart names one.
	Weekday  string
	Weekend  string
	Sections []string
	// Links are the tasks' links, by id, in the order Pele's map of them iterates.
	LinkIDs []string
	Links   map[string]string
	Tasks   []*Task

	byID           map[string]*Task
	currentSection string
	taskCount      int
	lastOrder      int
	lastTaskID     string
	hasLastTask    bool

	zone         zone
	loc          *time.Location
	now          float64
	work         float64
	includeSet   map[string]bool
	excludeSet   map[string]bool
	excludedDays [7]bool
	datedTokens  bool
	scheduled    bool

	// Warn, when set, is told of what a chart refers to that does not exist.
	Warn func(message string)
}

func newDB() *DB {
	return &DB{Weekend: "saturday", Links: map[string]string{}, byID: map[string]*Task{}}
}

func (db *DB) setDateFormat(txt string) { db.DateFormat = txt }

func (db *DB) setAccTitle(txt string) {
	db.AccTitle, db.HasAccTitle = chars.TrimStart(txt), true
}

// setAccDescription drops the white space that starts each line after the first, as
// replace(/\n\s+/g, '\n') does.
func (db *DB) setAccDescription(txt string) {
	var b strings.Builder
	for i := 0; i < len(txt); {
		if txt[i] == '\n' {
			if e := spaceEnd(txt, i+1); e > i+1 {
				b.WriteByte('\n')
				i = e
				continue
			}
		}
		b.WriteByte(txt[i])
		i++
	}
	db.AccDescr, db.HasAccDescr = b.String(), true
}

func (db *DB) addSection(txt string) {
	db.currentSection = txt
	db.Sections = append(db.Sections, txt)
}

// mergeTokens is Pele's mergeTokens: the tokens of txt, lowered and split at white space and
// commas, added to those there are, each once.
func mergeTokens(existing []string, txt string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(t string) {
		if !seen[t] {
			seen[t] = true
			out = append(out, t)
		}
	}
	for _, t := range existing {
		add(t)
	}
	lower := chars.ToLower(txt)
	for _, t := range strings.FieldsFunc(lower, func(r rune) bool { return r == ',' || chars.IsSpaceRune(r) }) {
		add(t)
	}
	return out
}

// isTag reports which tag a field is, as /^\s*(active|done|crit|milestone|vert)\s*$/.
func isTag(field string) string {
	switch t := chars.Trim(field); t {
	case "active", "done", "crit", "milestone", "vert":
		// Only the white space \s reads may be about the tag.
		return t
	}
	return ""
}

func (db *DB) addTask(descr, dataStr string) error {
	data := strings.Split(strings.TrimPrefix(dataStr, ":"), ",")
	task := &Task{
		Task:       descr,
		Section:    db.currentSection,
		Order:      -1,
		Classes:    []string{},
		PrevTaskID: db.lastTaskID,
		HasPrev:    db.hasLastTask,
		Raw:        Raw{Data: dataStr},
	}
	for len(data) > 0 {
		tag := isTag(data[0])
		if tag == "" {
			break
		}
		switch tag {
		case "active":
			task.Active = true
		case "done":
			task.Done = true
		case "crit":
			task.Crit = true
		case "milestone":
			task.Milestone = true
		case "vert":
			task.Vert = true
		}
		data = data[1:]
	}
	// Mermaid fails with a TypeError on both of these.
	if len(data) == 0 {
		return diagram.NewError(`Task "`+trim(descr)+`" needs a duration or an end date.`, diagram.Semantic, "gantt")
	}
	if len(data) > 3 {
		return diagram.NewError(`Task "`+trim(descr)+`" has too many fields. A task takes an id, a start, and an end or duration.`,
			diagram.Semantic, "gantt")
	}
	for i := range data {
		data[i] = trim(data[i])
	}
	if len(data) == 3 {
		task.ID = data[0]
	} else {
		db.taskCount++
		task.ID = "task" + strconv.Itoa(db.taskCount)
	}
	if len(data) > 1 {
		task.Raw.Start, task.Raw.HasStart = data[len(data)-2], true
	}
	task.Raw.End = data[len(data)-1]
	if !task.Vert {
		task.Order = db.lastOrder
		db.lastOrder++
	}
	db.Tasks = append(db.Tasks, task)
	db.lastTaskID, db.hasLastTask = task.ID, true
	db.byID[task.ID] = task
	return nil
}

func (db *DB) setLink(ids, link string) {
	href := url.SafeURL(link)
	for _, id := range strings.Split(ids, ",") {
		if _, ok := db.byID[id]; ok {
			if _, had := db.Links[id]; !had {
				db.LinkIDs = append(db.LinkIDs, id)
			}
			db.Links[id] = href
		}
	}
	db.setClass(ids, "clickable")
}

// setClass marks tasks with a class. Callbacks are recognized and never run; the task is only
// marked clickable, as in Mermaid.
func (db *DB) setClass(ids, className string) {
	for _, id := range strings.Split(ids, ",") {
		if t, ok := db.byID[id]; ok {
			t.Classes = append(t.Classes, className)
		}
	}
}

// Schedule resolves every task's start and end against the time now in the zone loc, from
// scratch on each call, as Pele's getTasks does. A task may refer to tasks defined after it,
// so, like Mermaid, it makes up to eleven passes, each of which schedules every task again.
// An error is a *diagram.Error.
func (db *DB) Schedule(now time.Time, loc *time.Location) error {
	if loc == nil {
		loc = time.Local
	}
	db.loc = loc
	db.zone = zone{jsdate.In(loc)}
	db.now = float64(now.UnixMilli())
	db.work = 0
	db.includeSet = map[string]bool{}
	for _, t := range db.Includes {
		db.includeSet[t] = true
	}
	db.excludeSet = map[string]bool{}
	for _, t := range db.Excludes {
		db.excludeSet[t] = true
	}
	weekendStart := math.NaN()
	switch db.Weekend {
	case "friday":
		weekendStart = 5
	case "saturday":
		weekendStart = 6
	}
	weekends := db.excludeSet["weekends"]
	for day, name := range weekdays {
		iso := float64(day)
		if day == 0 {
			iso = 7
		}
		db.excludedDays[day] = weekends && (iso == weekendStart || iso == weekendStart+1) || db.excludeSet[strings.ToLower(name)]
	}
	// A date written with only these tokens and separators can never read as a weekday name,
	// so when nothing else is excluded or included there is no need to format each day.
	db.datedTokens = len(db.Includes) > 0 || !numericFormat(trim(db.DateFormat))
	for _, t := range db.Excludes {
		if t != "weekends" && !isDayName(t) {
			db.datedTokens = true
		}
	}
	for _, t := range db.Tasks {
		t.loc = loc
		t.start, t.end = moment{}, moment{}
		t.renderEnd = moment{}
		t.Processed, t.ManualEndTime = false, false
	}
	db.scheduled = true
	all, err := db.compileTasks()
	for i := 0; err == nil && !all && i < 10; i++ {
		all, err = db.compileTasks()
	}
	return err
}

// copied is a copy of the chart for Layout to schedule, its tasks copied with it, since
// Schedule writes into each task: laying a chart out then leaves the DB as it was, and may be
// done from many goroutines at once.
func (db *DB) copied() *DB {
	c := *db
	c.Tasks = make([]*Task, len(db.Tasks))
	c.byID = make(map[string]*Task, len(db.byID))
	for i, t := range db.Tasks {
		own := *t
		c.Tasks[i] = &own
		if db.byID[t.ID] == t {
			c.byID[t.ID] = &own
		}
	}
	return &c
}

// numericFormat is RE_NUMERIC_FORMAT, /^[YMDHhmsSZXx\-/.:, T]*$/.
func numericFormat(s string) bool {
	for i := 0; i < len(s); i++ {
		if !strings.ContainsRune("YMDHhmsSZXx-/.:, T", rune(s[i])) {
			return false
		}
	}
	return true
}

func isDayName(t string) bool {
	for _, w := range weekdays {
		if strings.ToLower(w) == t {
			return true
		}
	}
	return false
}

// isInvalidDate is Pele's isInvalidDate: whether a day is excluded. A day matches a token by
// its weekday name, by its date as YYYY-MM-DD, or by its date written in the chart's own
// format.
func (db *DB) isInvalidDate(date float64) (bool, error) {
	weekday := false
	if d := db.zone.Day(date); !math.IsNaN(d) {
		weekday = db.excludedDays[int(d)]
	}
	if !db.datedTokens {
		return weekday, nil
	}
	template := trim(db.DateFormat)
	dateOnly := db.zone.isoDate(date)
	formatted, ok := dateOnly, true
	if err := db.spend(datedStep); err != nil {
		return false, err
	}
	if template != "YYYY-MM-DD" {
		if err := db.spend(formatStep + float64(chars.Len16(template)*8)); err != nil {
			return false, err
		}
		formatted, ok = db.zone.format(date, template)
	}
	listed := func(set map[string]bool) bool { return set[dateOnly] || ok && set[formatted] }
	if listed(db.includeSet) {
		return false, nil
	}
	return weekday || listed(db.excludeSet), nil
}

func (db *DB) spend(units float64) error {
	db.work += units
	if db.work > workLimit {
		return diagram.NewError("The gantt chart takes too much work to schedule.", diagram.Limit, "gantt")
	}
	return nil
}

func (db *DB) parse(str, template string) (float64, bool, error) {
	if err := db.spend(float64((chars.Len16(template) + 1) * (chars.Len16(str) + 1))); err != nil {
		return 0, false, err
	}
	d, ok := db.zone.parseStrict(str, template, db.now)
	return d, ok, nil
}

func (db *DB) today() float64 { return db.zone.startOfDay(db.now) }

func (db *DB) warn(message func() string) {
	if db.Warn != nil {
		db.Warn(message())
	}
}

func (db *DB) compileTasks() (bool, error) {
	all := true
	for _, t := range db.Tasks {
		if err := db.compileTask(t); err != nil {
			return false, err
		}
		all = all && t.Processed
	}
	return all, nil
}

func (db *DB) compileTask(task *Task) error {
	if !task.Raw.HasStart {
		// Mermaid fails with a TypeError when the first task has no start. It starts today
		// here.
		var previous *Task
		if task.HasPrev {
			previous = db.byID[task.PrevTaskID]
		}
		if previous != nil {
			task.start = previous.end
		} else {
			task.start = at(db.today())
		}
	} else {
		start, err := db.getStartDate(task.Raw.Start)
		if err != nil {
			return err
		}
		if start.defined {
			task.start = start
		}
	}
	if !task.start.defined {
		return nil
	}
	end, err := db.getEndDate(task.start, task.Raw.End)
	if err != nil {
		return err
	}
	task.end = end
	if !task.end.defined {
		return nil
	}
	task.Processed = true
	_, manual, err := db.parse(task.Raw.End, "YYYY-MM-DD")
	if err != nil {
		return err
	}
	task.ManualEndTime = manual
	return db.checkTaskDates(task)
}

func (db *DB) references(keyword, list string) []*Task {
	var found []*Task
	var unknown []string
	for _, id := range strings.Split(list, " ") {
		if id == "" {
			continue
		}
		if t, ok := db.byID[id]; ok {
			found = append(found, t)
		} else {
			unknown = append(unknown, id)
		}
	}
	if unknown != nil {
		db.warn(func() string {
			return `Gantt: "` + keyword + `" refers to task ids that do not exist: ` + strings.Join(unknown, ", ") + "."
		})
	}
	return found
}

// isRefChar is [\d\w- ].
func isRefChar(c byte) bool { return chars.IsWord(int(c)) || c == '-' || c == ' ' }

// matchRef is /^keyword\s+([\d\w- ]+)/: the ids it refers to, and whether it matched.
func matchRef(s, keyword string) (string, bool) {
	if !strings.HasPrefix(s, keyword) {
		return "", false
	}
	p := len(keyword)
	// The white space \s+ takes, with each character's start, to give back one at a time.
	var starts []int
	q := p
	for q < len(s) {
		c, size := chars.UnitAt(s, q)
		if !chars.IsSpace(c) {
			break
		}
		starts = append(starts, q)
		q += size
	}
	if len(starts) == 0 {
		return "", false
	}
	capture := func(from int) (string, bool) {
		e := from
		for e < len(s) && isRefChar(s[e]) {
			e++
		}
		return s[from:e], e > from
	}
	if m, ok := capture(q); ok {
		return m, true
	}
	for k := len(starts) - 1; k >= 1; k-- {
		if m, ok := capture(starts[k]); ok {
			return m, true
		}
	}
	return "", false
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func invalidDate(str string) error {
	return diagram.NewError("Invalid date:"+str, diagram.Semantic, "gantt")
}

func (db *DB) getStartDate(raw string) (moment, error) {
	str := trim(raw)
	template := trim(db.DateFormat)

	if (template == "x" || template == "X") && isDigits(str) {
		d := jsdate.TimeClip(number(str))
		if !jsdate.Valid(d) {
			return moment{}, invalidDate(str)
		}
		return at(d), nil
	}

	if list, ok := matchRef(str, "after"); ok {
		var latest *Task
		for _, t := range db.references("after", list) {
			if latest == nil || t.end.value() > latest.end.value() {
				latest = t
			}
		}
		if latest != nil {
			return latest.end, nil
		}
		return at(db.today()), nil
	}

	parsed, ok, err := db.parse(str, template)
	if err != nil {
		return moment{}, err
	}
	if ok {
		return at(parsed), nil
	}

	// What does not fit the format gets a second chance with the JavaScript date parser, as
	// in Mermaid.
	date := db.zone.Parse(str)
	if year := db.zone.FullYear(date); !jsdate.Valid(date) || year < -10000 || year > 10000 {
		return moment{}, invalidDate(str)
	}
	return at(date), nil
}

func (db *DB) getEndDate(start moment, raw string) (moment, error) {
	str := trim(raw)

	if list, ok := matchRef(str, "until"); ok {
		var earliest *Task
		for _, t := range db.references("until", list) {
			if earliest == nil || t.start.value() < earliest.start.value() {
				earliest = t
			}
		}
		if earliest != nil {
			return earliest.start, nil
		}
		return at(db.today()), nil
	}

	template := trim(db.DateFormat)
	parsed, ok, err := db.parse(str, template)
	if err != nil {
		return moment{}, err
	}
	if ok {
		if db.InclusiveEndDates {
			return at(db.zone.addDays(parsed, 1)), nil
		}
		return at(parsed), nil
	}

	value, u := parseDuration(str)
	if math.IsNaN(value) {
		db.warn(func() string {
			return `Gantt: "` + str + `" is neither a date in the "` + template +
				`" format nor a duration such as "3d", so the task gets no duration.`
		})
		return at(jsdate.TimeClip(start.ms)), nil
	}
	end := db.zone.add(start.ms, value, u)
	if !jsdate.Valid(end) {
		return at(jsdate.TimeClip(start.ms)), nil
	}
	return at(end), nil
}

// parseDuration is Pele's parseDuration: a duration such as 3d, as
// /^(\d+(?:\.\d+)?)([Mdhmswy]|ms)$/ reads it, or NaN.
func parseDuration(str string) (float64, unit) {
	s := trim(str)
	i := 0
	for i < len(s) && chars.IsDigit(int(s[i])) {
		i++
	}
	if i == 0 {
		return math.NaN(), "ms"
	}
	if i+1 < len(s) && s[i] == '.' && chars.IsDigit(int(s[i+1])) {
		i++
		for i < len(s) && chars.IsDigit(int(s[i])) {
			i++
		}
	}
	switch s[i:] {
	case "M", "d", "h", "m", "s", "w", "y", "ms":
		return number(s[:i]), unit(s[i:])
	}
	return math.NaN(), "ms"
}

// checkTaskDates pushes the end of a task out by one day for every excluded day it covers.
func (db *DB) checkTaskDates(task *Task) error {
	if len(db.Excludes) == 0 || task.ManualEndTime {
		return nil
	}
	z := db.zone
	cursor := z.addDays(task.start.ms, 1)
	endTime := jsdate.TimeClip(task.end.ms)
	maxEndTime := z.addDays(endTime, 10000)
	invalid := false
	renderEnd, hasRenderEnd := 0.0, false
	for cursor <= endTime {
		if err := db.spend(dayStep); err != nil {
			return err
		}
		if !invalid {
			renderEnd, hasRenderEnd = endTime, true
		}
		var err error
		if invalid, err = db.isInvalidDate(cursor); err != nil {
			return err
		}
		if invalid {
			endTime = z.SetDate(endTime, z.Date(endTime)+1)
			if endTime > maxEndTime {
				return diagram.NewError("Failed to find a valid date that was not excluded by `excludes` after 10,000 iterations.",
					diagram.Semantic, "gantt")
			}
		}
		cursor = z.SetDate(cursor, z.Date(cursor)+1)
	}
	task.end = at(endTime)
	if hasRenderEnd {
		task.renderEnd = at(renderEnd)
	} else {
		task.renderEnd = moment{}
	}
	return nil
}
