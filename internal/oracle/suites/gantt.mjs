// Gantt charts parsed as Pele parses them, from text to GanttDb, and the tasks its getTasks
// schedules from them in several time zones, against a fixed time: Mermaid's documentation and
// specs, the sources of Pele's tests, cases written here, those sources mutated at random, and
// charts built from random statements. Each case is parsed as Pele's parse does it, after the
// preprocessing every diagram goes through, and as the parser alone reads the raw text. Node
// reads process.env.TZ again when it is set, so one run goes through every zone.
//
// It also writes what the date code gives on its own, in each zone: strict parsing and
// formatting with dayjs's tokens, d3's axis formats, and the ticks of an axis.
import { enc, peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

export const ZONES = ['UTC', 'America/New_York', 'Europe/London', 'Asia/Kolkata'];

// The times charts are scheduled against: an ordinary day, the days New York and London
// change their clocks on, and Pele's test's own.
export const NOWS = [
  Date.UTC(2024, 0, 15, 12, 0, 0),
  Date.UTC(2024, 2, 10, 7, 30, 0),
  Date.UTC(2024, 9, 27, 0, 30, 0),
  Date.UTC(2031, 4, 6, 9, 30, 15, 250),
  Date.UTC(2024, 10, 3, 5, 59, 59, 999),
];

// Pele's fragments, tests/gantt/oracle.test.ts, and some for dates.
export const FRAGMENTS = [
  ' ', '\n', '\n\n', '\t', '\r\n', ' \n', ':', ';', '#', ',', '%', '%%', '%%{', '}', '{', '"', '(', ')', '()', '( )',
  'gantt', 'gantt\n', 'GANTT', 'dateFormat ', 'dateFormat YYYY-MM-DD', 'dateformat X', 'inclusiveEndDates',
  'topAxis', 'axisFormat ', 'axisFormat %Y-%m-%d', 'tickInterval ', 'tickInterval 1day', 'includes ', 'excludes ',
  'excludes weekends', 'todayMarker ', 'todayMarker off', 'todayMarker stroke:#0f0,opacity:0.5', 'weekday ',
  'weekday monday', 'weekday  tuesday', 'weekday\nwednesday', 'weekday thursday', 'weekday friday',
  'weekday saturday', 'weekday sunday', 'weekday sundays', 'Weekday Monday', 'weekend ', 'weekend friday',
  'weekend saturday', 'weekend sunday', 'title ', 'title A', 'accTitle: ', 'accTitle : x', 'accDescr: ',
  'accDescr { ', 'accDescr {\n x\n}', 'accDescription ', 'accDescription x', 'section ', 'section A',
  'click ', 'click a ', 'click a,b ', 'href ', 'href "', 'href "https://example.com"', 'call ', 'call f()',
  'call f(a, "b")', 'call f( )', '2014-01-01', '2014-01-01x', ' 2014-01-01', 'a1', 'des1, ', 'after a1',
  'until a1', '3d', '1w', 'crit, ', 'done, ', 'active, ', 'milestone, ', 'vert, ', 'A task :a1, 2014-01-01, 30d',
  'Task :', ': 5d', 'x%y', '% z', 'é', 'ö', ' ', ' ', 'ſection x', 'TITLE x', 'Section b',
  '2024-03-10', '2024-11-03', '2024-03-31', '2024-10-27', '1.5d', '36h', '90m', '1M', '1y', '500ms', '2w', 'after a1 b2',
  'until a1 b2', '2024/03/10 01:30', 'Mar 10 2024', '10:30', 'excludes 2024-03-11', 'includes 2024-03-09', 'dateFormat HH:mm',
  'dateFormat x', 'dateFormat DD.MM.YYYY', 'dateFormat MMM D, YYYY', 'tickInterval 1week', 'tickInterval 6hour',
  'axisFormat %H:%M', 'axisFormat %a %d', 'weekend friday\n', '😀', '　',
];

// Charts written here, for the corners of the scheduler and its dates.
export const WRITTEN = [
  'gantt',
  'gantt\n',
  'gantt x',
  'gantt\ndateFormat YYYY-MM-DD\nA :a1, 2014-01-01, 30d\nB :after a1, 20d\nC :2014-01-12, 12d\nD :24d',
  'gantt\ndateFormat YYYY-MM-DD\nA :2024-03-09, 3d\nB :2024-03-10, 36h\nC :2024-03-09 , 1d\nD :2024-03-10, 1w',
  'gantt\ndateFormat YYYY-MM-DD HH:mm\nA :2024-03-10 01:30, 2h\nB :2024-03-10 02:30, 1h\nC :2024-03-10 03:00, 30m\nD :2024-11-03 01:30, 1h\nE :2024-11-03 00:30, 3h',
  'gantt\ndateFormat YYYY-MM-DD HH:mm\nA :2024-03-31 00:30, 2h\nB :2024-03-31 01:30, 1h\nC :2024-10-27 01:30, 1h\nD :2024-10-27 00:00, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nexcludes weekends\nA :2024-03-08, 5d\nB :after A, 3d\nC :2024-10-25, 4d\nD :2024-11-01, 10d',
  'gantt\ndateFormat YYYY-MM-DD\nexcludes weekends 2019-02-06,friday\ninclusiveEndDates\nT1 :id1,2019-02-01,1d\nT2 :id2,after id1,2d\nT3 :id3,after id2,7d\nT4 :id4,2019-02-01,2019-02-20\nT5 :id5,after id4,1d\nT6 :id6,2019-02-13,2d\nT7 :id7,after id6,1d',
  'gantt\ndateFormat YYYY-MM-DD\nexcludes weekends\nweekend friday\nA :2024-02-28, 3d\nB :2024-02-28, 10d',
  'gantt\ndateFormat YYYY-MM-DD\nexcludes weekends,monday,tuesday,wednesday,thursday,friday\nA :2019-02-01,7d',
  'gantt\ndateFormat YYYY-MM-DD\nexcludes weekends,monday,tuesday,wednesday,thursday\nA :2019-02-01,7d',
  'gantt\ndateFormat YYYY-MM-DD\nexcludes weekends\nincludes 2024-03-09\nexcludes 2024-03-12\nA :2024-03-08, 6d',
  'gantt\ndateFormat x\nA :id1,0,20ms\nB :id2,after id1,5ms\nC :id3,20,10ms\nD :id4,after id3,0.005s',
  'gantt\ndateFormat X\nA :1410715640, 1h\nB :1410715640579, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :id1,2013-01-01,2w\nB :id2,after id3,1d\nC :id3,after id1,2d',
  'gantt\ndateFormat YYYY-MM-DD\nA :id1,2013-01-01,until id3\nB :id2,2013-01-10,until id3\nC :id3,2013-02-01,2d',
  'gantt\ndateFormat YYYY-MM-DD\nA :id1,after id2 id3 id4,1d\nB :id2,2013-01-01,1d\nC :id3,2013-02-01,3d\nD :id4,2013-02-01,2d',
  'gantt\ndateFormat YYYY-MM-DD\nA :id1,2013-01-01,until id2 id3 id4\nB :id2,2013-01-11,1d\nC :id3,2013-02-10,1d\nD :id4,2013-02-12,1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :a, after b, 1d\nB :b, after a, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :a, after zz, 1d\nB :b, 2024-01-01, until yy',
  'gantt\ndateFormat YYYY-MM-DD\nA :a, 2024-01-31, 1M\nB :b, 2024-01-31, 13M\nC :c, 2024-02-29, 1y\nD :d, 2024-01-15, 1.5M\nE :e, 2024-01-01, 0.4d\nF :f, 2024-01-01, 0.5w',
  'gantt\ndateFormat YYYY-MM-DD\nA :a, 2024-01-01, 99999999y',
  'gantt\ndateFormat YYYY-MM-DD\nA :a, 2024-01-01, 3dX\nB :b, 2024-01-01, -1d\nC :c, 2024-01-01, 2024-01-05',
  'gantt\ndateFormat YYYY-MM-DD\ninclusiveEndDates\nA :a, 2024-01-01, 2024-01-05\nB :b, 2024-03-09, 2024-03-10',
  'gantt\ndateFormat DD-MM-YYYY\nA :06-01-2014, 3d\nB :2014-01-06, 1d',
  'gantt\ndateFormat YY/M/D\nA :14/1/6, 2d\nB :70/12/31, 1d',
  'gantt\ndateFormat HH:mm\naxisFormat %H:%M\nA :17:49, 1h\nB :02:30, 30m',
  'gantt\ndateFormat h:mm A\nA :9:30 PM, 1h\nB :12:05 AM, 1h',
  'gantt\ndateFormat MMM D, YYYY\nA :Jan 5, 2024, 2d',
  'gantt\ndateFormat Do MMMM YYYY\nA :5th January 2024, 2d\nB :22nd December 2024, 1d\nC :5 January 2024, 1d',
  'gantt\ndateFormat [on] YYYY-MM-DD\nA :on 2024-03-01, 1d',
  'gantt\ndateFormat YYYY\nA :2024, 1y\nB :MM, 1d',
  'gantt\ndateFormat MM\nA :03, 1d',
  'gantt\ndateFormat D\nA :7, 1d\nB :0, 1d',
  'gantt\ndateFormat ss\nA :20, 1s',
  'gantt\ndateFormat Q YYYY\nA :2 2024, 1M',
  'gantt\ndateFormat dddd YYYY\nA :Monday 2024, 1d',
  'gantt\ndateFormat YYYY-MM-DDTHH:mm:ssZ\nA :2024-01-05T10:00:00+00:00, 1d\nB :2024-01-05T10:00:00-05:00, 1d\nC :2024-01-05T10:00:00+05:30, 1d\nD :2024-07-05T10:00:00+01:00, 1d',
  'gantt\ndateFormat YYYY-MM-DDTHH:mm:ssZZ\nA :2024-01-05T10:00:00+0530, 1d\nB :2024-01-05T10:00:00-0500, 1d',
  'gantt\ndateFormat L\nA :12/25/2020, 1d',
  'gantt\ndateFormat YYYY-ww\nA :2020-01, 1d',
  'gantt\ndateFormat\nA :2024-01-01, 1d',
  'gantt\ndateFormat GGGG-[W]WW\nA :2020-W53, 1d\nB :2024-01-01, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :2024/01/05, 2d\nB :Jan 5 2024, 1d\nC :2024-03-10T02:30, 1h\nD :5, 1d\nE :2024-01-05T10:00:00Z, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :garbage, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :20000-01-01, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :Jan 1 2000 10:00 GMT+5, 1d\nB :1999-12-31 23:59:59.999, 1ms',
  'gantt\ndateFormat YYYY-MM-DD\nA :2024-01-01',
  'gantt\ndateFormat YYYY-MM-DD\nA :',
  'gantt\ndateFormat YYYY-MM-DD\nA :crit, done',
  'gantt\ndateFormat YYYY-MM-DD\nA :a, b, c, d',
  'gantt\ndateFormat YYYY-MM-DD\nA :crit, done, active, milestone, vert, a, 2024-01-01, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :milestone, m1, 2024-01-05, 0d\nB :vert, v1, 2024-01-03, 0d\nC :2024-01-01, 10d',
  'gantt\ndateFormat YYYY-MM-DD\nA :a1, 2024-01-01, 1d\nclick a1 href "https://example.com"\nclick a1 call f()\nclick zz href "x"\nclick a1,b1 call g(1, "x") href "y"\nB :b1, 2024-01-02, 1d\nclick b1 href "javascript:alert(1)"',
  'gantt\ntitle T\naccTitle: The title \naccDescr: The description\naccDescr {\n  first\n    second\n}\nA :2024-01-01, 1d',
  'gantt\nweekday monday\ntickInterval 1week\nA :2024-01-01, 30d',
  'gantt\nsection S1\nA :2024-01-01, 1d\nsection S2\nB :2024-01-01, 1d\nsection S1\nC :2024-01-01, 1d',
  'gantt\ntodayMarker off\ntopAxis\naxisFormat %d\nincludes 2024-01-06 sunday\nA :2024-01-01, 1d',
  'gantt\nA :1d\nB :2d',
  'gantt\ndateFormat YYYY-MM-DD\nexcludes 2024-01-0\nA :2024-01-01, 3d',
  'gantt\ndateFormat DD.MM.YYYY\nexcludes 02.01.2024, 2024-01-03\nA :01.01.2024, 5d',
  'gantt\ndateFormat MMM D YYYY\nexcludes Jan 2 2024\nA :Jan 1 2024, 5d',
  'gantt\ndateFormat YYYY-MM-DD\nexcludes weekends\nA :2024-01-01, 10950d\nB :2024-01-01, 10950d',
  'gantt\n%%{init: {}}%%\nA :2024-01-01, 1d',
  'gantt\nA %% c\n: 2024-01-01, 1d',
  'gantt\ndateFormat YYYY-MM-DD\nA :a, 1850-06-01, 1d\nB :b, 1900-01-01, 1d\nC :c, 0001-01-01, 1d\nD :d, 0050-01-01, 1d',
  '---\ntitle: Front matter\ndisplayMode: compact\n---\ngantt\nA :2024-01-01, 1d',
  // Past the limit on the work of scheduling, by days and by formats.
  'gantt\ndateFormat YYYY-MM-DD\nexcludes weekends\n' + 'T :2024-01-01, 10950d\n'.repeat(400),
  'gantt\ndateFormat DD.MM.YYYY [' + 'x'.repeat(2000) + ']\nexcludes 01.01.2024\n' + 'T :2024-01-01, 20000d\n'.repeat(60),
  // Within the limit to schedule, past it to mark the excluded days of.
  'gantt\ndateFormat [' + 'x'.repeat(30000) + ']\nincludes 2030-01-01\nA :2024-01-01, 1800d\n',
];

// The sources in Mermaid's specs as Pele ports them, as Pele's loadCorpus collects them.
const specSources = (pele) => testStrings(pele, 'tests/compat/gantt', /\.spec\.[jt]s$/, /gantt/);

export function corpus(pele) {
  return [...new Set([...docs(pele, 'gantt'), ...specSources(pele), ...WRITTEN])].filter((s) => !lone(s));
}

// Charts of whole statements, which mostly get through the grammar and then schedule in every
// way the scheduler resolves.
export function statementDiagrams(rnd) {
  const pick = picker(rnd);
  const formats = ['YYYY-MM-DD', 'YYYY-MM-DD', 'YYYY-MM-DD HH:mm', 'x', 'DD.MM.YYYY', 'MMM D YYYY', 'HH:mm', 'YYYY-MM-DDTHH:mm', 'D', 'X'];
  return () => {
    const format = pick(formats);
    const ids = ['a', 'b', 'c', 'd', 'e'];
    const date = () => {
      const y = pick([2024, 2024, 2024, 2023, 1999, 2038]);
      const m = pick([0, 2, 2, 9, 10, 11]);
      const d = 1 + Math.floor(rnd() * 31);
      const h = Math.floor(rnd() * 24);
      const mi = pick([0, 30, 59]);
      const p = (n) => String(n).padStart(2, '0');
      const M = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'][m];
      switch (format) {
        case 'YYYY-MM-DD HH:mm':
          return `${y}-${p(m + 1)}-${p(d)} ${p(h)}:${p(mi)}`;
        case 'x':
          return String(Date.UTC(y, m, d, h, mi));
        case 'X':
          return String(Date.UTC(y, m, d, h, mi) / 1000);
        case 'DD.MM.YYYY':
          return `${p(d)}.${p(m + 1)}.${y}`;
        case 'MMM D YYYY':
          return `${M} ${d} ${y}`;
        case 'HH:mm':
          return `${p(h)}:${p(mi)}`;
        case 'YYYY-MM-DDTHH:mm':
          return `${y}-${p(m + 1)}-${p(d)}T${p(h)}:${p(mi)}`;
        case 'D':
          return String(d);
      }
      return rnd() < 0.1 ? pick(['2024/03/10', 'Mar 10 2024 02:30', 'junk', '2024-03-10T01:30']) : `${y}-${p(m + 1)}-${p(d)}`;
    };
    const duration = () => `${pick([1, 1, 2, 3, 5, 7, 10, 30, 0, 1.5])}${pick(['d', 'd', 'd', 'h', 'w', 'm', 'M', 'y', 's', 'ms'])}`;
    const end = () => (rnd() < 0.15 ? date() : rnd() < 0.1 ? `until ${pick(ids)}` : duration());
    const start = () => (rnd() < 0.25 ? `after ${pick(ids)}${rnd() < 0.2 ? ' ' + pick(ids) : ''}` : date());
    const tags = () => {
      let t = '';
      for (const tag of ['crit', 'done', 'active', 'milestone', 'vert']) if (rnd() < 0.08) t += tag + ', ';
      return t;
    };
    const task = () => {
      const r = rnd();
      const name = pick(['Task', 'A longer task name', 'x', 'Ünïcödé', 'Design #35;', '<b>bold</b>', 'Fix it']);
      if (r < 0.5) return `${name} :${tags()}${pick(ids)}, ${start()}, ${end()}`;
      if (r < 0.8) return `${name} :${tags()}${start()}, ${end()}`;
      return `${name} :${tags()}${end()}`;
    };
    const statements = [
      task, task, task, task, task,
      () => `section ${pick(['A', 'B', 'Section two', ''])}`,
      () => `excludes ${pick(['weekends', 'sunday', 'monday,tuesday', '2024-03-11', 'weekends 2024-03-12', date()])}`,
      () => `includes ${pick(['2024-03-09', 'saturday', date()])}`,
      () => pick(['weekend friday', 'weekend saturday', 'inclusiveEndDates', 'topAxis', 'todayMarker off', 'weekday monday']),
      () => `tickInterval ${pick(['1day', '1week', '6hour', '1month', '15minute', '2week', 'bad'])}`,
      () => `axisFormat ${pick(['%Y-%m-%d', '%H:%M', '%a %d %b', '%U %W %V', '%e', '%j'])}`,
      () => `click ${pick(ids)} ${pick(['href "https://e.com"', 'call f()', 'call f(1) href "x"'])}`,
      () => `title ${pick(['T', 'A title'])}`,
    ];
    let src = `gantt\ndateFormat ${format}\n`;
    for (let k = 1 + Math.floor(rnd() * 8); k > 0; k--) src += pick(statements)() + '\n';
    return src;
  };
}

const dumpDate = (d) => (d === undefined ? enc(undefined) : d === null ? null : enc(d.getTime()));

function dumpModel(m) {
  return {
    title: enc(m.title),
    accTitle: enc(m.accTitle),
    accDescr: enc(m.accDescr),
    dateFormat: m.dateFormat,
    axisFormat: m.axisFormat,
    tickInterval: enc(m.tickInterval),
    todayMarker: m.todayMarker,
    includes: m.includes,
    excludes: m.excludes,
    inclusiveEndDates: m.inclusiveEndDates,
    topAxis: m.topAxis,
    displayMode: m.displayMode,
    weekday: enc(m.weekday),
    weekend: m.weekend,
    sections: m.sections,
    links: [...m.links.entries()],
    tasks: m.tasks.map((t) => ({
      id: t.id,
      task: t.task,
      section: t.section,
      flags: [t.active, t.done, t.crit, t.milestone, t.vert],
      order: t.order,
      classes: t.classes,
      prevTaskId: enc(t.prevTaskId),
      raw: [t.raw.data, enc(t.raw.start), t.raw.end],
    })),
  };
}

// What getTasks makes of a model in each zone: per task its start, end, end of bar, and
// whether it was processed and has a manual end; or the error.
function schedules(m, now, caught) {
  const out = [];
  for (const zone of ZONES) {
    process.env.TZ = zone;
    try {
      out.push({ tasks: m.getTasks(now).map((t) => [dumpDate(t.startTime), dumpDate(t.endTime), dumpDate(t.renderEndTime), t.processed, t.manualEndTime]) });
    } catch (e) {
      out.push(caught(e));
    }
  }
  return out;
}

// The date code on its own, in each zone.
function dates(rnd, mod) {
  const { parseStrict, format } = mod.dates;
  const { timeFormat, autoTicks, intervalTicks } = mod.axis;
  const pick = picker(rnd);
  const tokens = ['YYYY', 'YY', 'Y', 'M', 'MM', 'MMM', 'MMMM', 'D', 'DD', 'Do', 'd', 'dd', 'ddd', 'dddd', 'H', 'HH', 'h', 'hh', 'a', 'A',
    'm', 'mm', 's', 'ss', 'S', 'SS', 'SSS', 'Z', 'ZZ', 'X', 'x', 'Q', 'k', 'kk', 'W', 'WW', 'GGGG', 'w', 'z', '[at]', '[W]', '-', '/', ':', ' ',
    '.', ',', 'T', 'L', 'é', '[', ']'];
  const specs = ['%a', '%A', '%b', '%B', '%c', '%d', '%e', '%f', '%g', '%G', '%H', '%I', '%j', '%L', '%m', '%M', '%p', '%q', '%Q', '%s', '%S',
    '%u', '%U', '%V', '%w', '%W', '%x', '%X', '%y', '%Y', '%Z', '%%', '%-d', '%_d', '%0e', '%-', '%k', ' ', '-', '/', 'x', '%😀', '%'];
  const instants = [];
  for (let i = 0; i < 400; i++) {
    const r = rnd();
    instants.push(r < 0.3 ? Date.UTC(2024, pick([0, 2, 2, 9, 10, 11]), 1 + Math.floor(rnd() * 31), Math.floor(rnd() * 24), pick([0, 30]))
      : r < 0.9 ? Date.UTC(1800 + Math.floor(rnd() * 400), Math.floor(rnd() * 12), 1 + Math.floor(rnd() * 28), Math.floor(rnd() * 24), Math.floor(rnd() * 60), Math.floor(rnd() * 60), Math.floor(rnd() * 1000))
        : Math.floor((rnd() * 2 - 1) * 8.64e15));
  }
  const templates = ['YYYY-MM-DD', '', 'YYYY-MM-DDTHH:mm:ssZ', 'DD.MM.YYYY HH:mm', 'Do MMMM YYYY', 'h:mm A', 'X', 'x', 'GGGG-[W]WW'];
  for (let i = 0; i < 300; i++) {
    let t = '';
    for (let k = 1 + Math.floor(rnd() * 5); k > 0; k--) t += pick(tokens);
    templates.push(t);
  }
  const formats = [];
  for (let i = 0; i < 1500; i++) formats.push([pick(instants), pick(templates)]);
  const spans = [];
  for (let i = 0; i < 1500; i++) {
    let s = '';
    for (let k = 1 + Math.floor(rnd() * 4); k > 0; k--) s += pick(specs);
    spans.push([pick(instants), s]);
  }
  const tickCases = [];
  const units = [1, 1000, 60000, 3600000, 86400000, 7 * 86400000, 30 * 86400000, 365 * 86400000];
  for (let i = 0; i < 600; i++) {
    const start = pick(instants.slice(0, 200));
    const stop = start + Math.floor(rnd() * pick(units) * (1 + rnd() * 40));
    const interval = rnd() < 0.5 ? '' : `${pick([1, 1, 2, 3, 6, 10, 15])}${pick(['millisecond', 'second', 'minute', 'hour', 'day', 'week', 'month', 'year'])}`;
    tickCases.push([start, stop, 2 + Math.floor(rnd() * 9), interval, pick(['sunday', 'monday', 'thursday', 'bad'])]);
  }
  return (now) => {
    const nowDate = new Date(now);
    const parses = [];
    for (const [t, template] of formats.slice(0, 600)) {
      const text = format(new Date(t), template);
      if (text === undefined) continue;
      const r = parseStrict(text, template, nowDate);
      parses.push([text, template, r === undefined ? null : r.getTime()]);
    }
    return {
      formats: formats.map(([t, template]) => [t, template, enc(format(new Date(t), template))]),
      parses,
      timeFormats: spans.map(([t, s]) => [t, s, timeFormat(s, new Date(t))]),
      ticks: tickCases.map(([start, stop, count, interval, weekday]) => [
        start, stop, count, interval, weekday,
        interval ? enc(intervalTicks(interval, weekday, start, stop, 200)) : autoTicks(start, stop, count),
      ]),
    };
  };
}

export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { gantt } = await src('diagrams/gantt/index.ts');
  const { parseGantt } = await src('diagrams/gantt/parser.ts');
  const { GanttDb } = await src('diagrams/gantt/db.ts');
  const mod = { dates: await src('diagrams/gantt/dates.ts'), axis: await src('diagrams/gantt/axis.ts') };
  const saved = process.env.TZ;

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const parsed = (parseIt, now) => {
    let m;
    try {
      m = parseIt();
    } catch (e) {
      return caught(e);
    }
    return { model: dumpModel(m), schedules: schedules(m, now, caught) };
  };
  let n = 0;
  const run = (source) => {
    const now = NOWS[n++ % NOWS.length];
    const out = {
      src: source,
      now,
      direct: parsed(() => {
        const db = new GanttDb();
        parseGantt(source, db);
        return db;
      }, now),
    };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'gantt') return out;
    const piped = parsed(() => located(source, false, () => gantt.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title)), now);
    if (JSON.stringify(piped) === JSON.stringify(out.direct)) return { ...out, same: true };
    return { ...out, ...piped };
  };

  const base = corpus(pele);
  const all = [...base];
  const seen = new Set(all);
  const add = (s) => {
    if (s.length <= 4000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  const rnd = random(41);
  // The charts past the work limit are long and slow to schedule; mutated, they would mostly
  // be more of the same.
  const next = mutator(base.filter((s) => s.length <= 1000), FRAGMENTS, rnd);
  for (let i = 0; i < 2500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const diagrams = statementDiagrams(random(42));
  for (let i = 0; i < 2500; i++) add(diagrams());
  const trnd = random(43);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  for (let i = 0; i < 600; i++) {
    let s = ['gantt\n', 'gantt\nA :', 'gantt\nclick ', 'gantt\naccDescr {', ''][Math.floor(trnd() * 5)];
    for (let k = 1 + Math.floor(trnd() * 10); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  const cases = all.map(run);

  // The date code on its own.
  const make = dates(random(44), mod);
  const zones = [];
  for (const zone of ZONES) {
    process.env.TZ = zone;
    zones.push({ zone, now: NOWS[3], ...make(NOWS[3]) });
  }
  process.env.TZ = saved;
  write('diagram/gantt/testdata/pele/gantt.json', { zones: ZONES, cases });
  write('diagram/gantt/testdata/pele/dates.json', { zones });
}
